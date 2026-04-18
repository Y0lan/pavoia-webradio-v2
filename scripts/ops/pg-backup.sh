#!/bin/bash
# GAENDE Radio — nightly Postgres backup + rotation + heartbeat.
#
# Install as Whatbox cron: `0 3 * * * ~/gaende-radio/scripts/ops/pg-backup.sh`
#
# Dumps the gaende DB over the port-mapped connection via an ephemeral
# postgres:16 container (Whatbox host pg_dump is v14 and refuses a v16
# server; `podman exec gaende-postgres` has been flaky with crun-state
# corruption). Gzips to BACKUP_DIR, rotates at RETENTION_DAYS, and writes a
# heartbeat row to pg_backup_log so /health can surface "backups stopped."

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

# mktemp-per-invocation stderr buffer so concurrent cron ticks (two runs on
# different hours, or an admin kick overlapping with cron) don't trample each
# other's error output or fall prey to a symlink at /tmp/pgdump.err.
STDERR_FILE=$(mktemp /tmp/pg-backup-err.XXXXXX)
trap 'rm -f "$STDERR_FILE"' EXIT

# Include the PID in BOTH the tempfile and the final artifact name. Without it,
# two same-second successes (admin kick overlapping with cron, or back-to-back
# manual retries) each mv their tempfile into the same canonical path; the
# second mv clobbers the first and the heartbeat row's `backup_path` now points
# at a file that was replaced by someone else's dump. Rotation still works —
# `gaende-*.sql.gz` matches PID-suffixed files fine.
TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
DUMP_PATH="$BACKUP_DIR/gaende-$TIMESTAMP.$$.sql.gz"
DUMP_TMP="$DUMP_PATH.tmp"

log_heartbeat() {
    # Writes one row to pg_backup_log. Uses psql's `:'var'` substitution
    # (heredoc, not -c — the :'var' macro only expands in script input) to
    # avoid injection hazards from env-derived paths / stderr strings that
    # might contain quotes or $$ dollar-quote sequences.
    local status="$1" rows="$2" size="$3" path="$4" err="$5"
    PGPASSWORD="$PGPASSWORD" psql -v ON_ERROR_STOP=1 -q \
        -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDB" \
        -v "h_status=$status" \
        -v "h_rows=$rows" \
        -v "h_size=$size" \
        -v "h_path=$path" \
        -v "h_err=$err" <<'SQL' 2>&1 || echo "pg-backup: heartbeat write failed (non-fatal)" >&2
INSERT INTO pg_backup_log (status, rows_dumped, size_bytes, backup_path, error)
VALUES (
    :'h_status',
    NULLIF(:'h_rows','')::bigint,
    NULLIF(:'h_size','')::bigint,
    NULLIF(:'h_path',''),
    NULLIF(:'h_err','')
);
SQL
}

# --- Dump via ephemeral pg-16 container.
DUMP_CMD=(
    podman run --rm --network host
    -e PGPASSWORD="$PGPASSWORD"
    docker.io/library/postgres:16-alpine
    pg_dump -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" --no-owner --no-acl "$PGDB"
)
if ! "${DUMP_CMD[@]}" 2>"$STDERR_FILE" | gzip -6 > "$DUMP_TMP"; then
    err=$(tr '\n' ' ' < "$STDERR_FILE" | head -c 500)
    log_heartbeat "failed" "" "" "" "$err"
    echo "pg-backup: dump failed — $err" >&2
    rm -f "$DUMP_TMP"
    exit 1
fi
mv "$DUMP_TMP" "$DUMP_PATH"

# Sanity: gzip -t must pass before we call the backup "ok".
if ! gzip -t "$DUMP_PATH" 2>/dev/null; then
    log_heartbeat "failed" "" "" "$DUMP_PATH" "corrupt gzip archive"
    echo "pg-backup: produced corrupt archive at $DUMP_PATH" >&2
    exit 1
fi

SIZE=$(stat -c %s "$DUMP_PATH")
ROWS=$(zgrep -c '^COPY ' "$DUMP_PATH" 2>/dev/null || echo 0)

log_heartbeat "ok" "$ROWS" "$SIZE" "$DUMP_PATH" ""
echo "pg-backup: $DUMP_PATH ($SIZE bytes, $ROWS COPY blocks)"

# --- Rotate — remove backups older than RETENTION_DAYS.
find "$BACKUP_DIR" -maxdepth 1 -name 'gaende-*.sql.gz' -mtime +"$RETENTION_DAYS" -delete

exit 0
