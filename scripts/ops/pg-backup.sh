#!/bin/bash
# GAENDE Radio — nightly Postgres backup + rotation + heartbeat.
#
# Install as Whatbox cron: `0 3 * * * ~/gaende-radio/scripts/ops/pg-backup.sh`
#
# Dumps the gaende DB over the port-mapped connection (bypasses `podman exec`,
# which has been flaky with crun-state corruption on this host), gzips to
# ~/files/backups/pg/, rotates to 7 days, and writes a heartbeat row to the
# pg_backup_log table so /health can surface "backups stopped working."
#
# Exit codes:
#   0 — success, heartbeat written
#   1 — dump failed (heartbeat still written with status=failed + error)

set -uo pipefail

BACKUP_DIR="${BACKUP_DIR:-$HOME/files/backups/pg}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
PGPASSWORD="${POSTGRES_PASSWORD:-gaende_prod}"
PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-15432}"
PGUSER="${PGUSER:-gaende}"
PGDB="${PGDB:-gaende}"
export PGPASSWORD

mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
DUMP_PATH="$BACKUP_DIR/gaende-$TIMESTAMP.sql.gz"
LOG_SQL=""

log_heartbeat() {
    # Write a row to pg_backup_log. Run inline so a DB failure during logging
    # doesn't abort the script's exit code (backup itself may have succeeded).
    local status="$1" rows="$2" size="$3" path="$4" err="$5"
    psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDB" -v ON_ERROR_STOP=1 <<SQL || true
INSERT INTO pg_backup_log (status, rows_dumped, size_bytes, backup_path, error)
VALUES ('$status', NULLIF($rows, 0), NULLIF($size, 0), NULLIF('$path', ''), NULLIF(\$\$$err\$\$, ''));
SQL
}

# --- Dump.
# Use an ephemeral pg-16 container so the version matches the server exactly;
# Whatbox ships pg_dump v14 which refuses to dump a v16 server. `podman exec
# gaende-postgres` has been flaky with crun-state corruption on this host, so
# `podman run --rm --network host` is the reliable path.
DUMP_CMD=(
    podman run --rm --network host
    -e PGPASSWORD="$PGPASSWORD"
    docker.io/library/postgres:16-alpine
    pg_dump -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" --no-owner --no-acl "$PGDB"
)
if ! "${DUMP_CMD[@]}" 2>/tmp/pgdump.err | gzip -6 > "$DUMP_PATH.tmp"; then
    err=$(tr '\n' ' ' < /tmp/pgdump.err | head -c 500)
    log_heartbeat "failed" 0 0 "" "$err"
    echo "pg-backup: dump failed — $err" >&2
    rm -f "$DUMP_PATH.tmp"
    exit 1
fi
mv "$DUMP_PATH.tmp" "$DUMP_PATH"

# Quick sanity: `gzip -t` must succeed and the file must be non-empty.
if ! gzip -t "$DUMP_PATH" 2>/dev/null; then
    log_heartbeat "failed" 0 0 "$DUMP_PATH" "corrupt gzip archive"
    echo "pg-backup: produced corrupt archive at $DUMP_PATH" >&2
    exit 1
fi

SIZE=$(stat -c %s "$DUMP_PATH")
# Count rows by grepping COPY blocks (cheaper than a full restore probe).
ROWS=$(zgrep -c '^COPY ' "$DUMP_PATH" 2>/dev/null || echo 0)

log_heartbeat "ok" "$ROWS" "$SIZE" "$DUMP_PATH" ""
echo "pg-backup: $DUMP_PATH ($SIZE bytes, $ROWS COPY blocks)"

# --- Rotate — remove backups older than RETENTION_DAYS.
find "$BACKUP_DIR" -maxdepth 1 -name 'gaende-*.sql.gz' -mtime +"$RETENTION_DAYS" -delete

exit 0
