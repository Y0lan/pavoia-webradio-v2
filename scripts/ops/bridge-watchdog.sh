#!/bin/bash
# GAENDE Radio — bridge liveness watchdog.
#
# Install as Whatbox cron: `* * * * * ~/gaende-radio/scripts/ops/bridge-watchdog.sh`
#
# Probes the bridge every minute and ONLY restarts when the bridge itself is
# unreachable (HTTP 000 — connection refused or curl timeout). A 5xx response
# means the bridge is alive but a dependency is degraded; restarting would
# churn the process without fixing the underlying issue and would destroy
# diagnostic state during an outage. Let /health degraded signals be surfaced
# by whatever external monitoring watches them, not by the watchdog.
#
# Restart trigger: N consecutive connection failures (default 3). State in
# ~/files/backups/watchdog-state so the counter survives cron re-exec.
# Restart strategy: SIGTERM with 5s grace → SIGKILL → respawn with env from
# ~/.gaende.env (same invocation deploy.sh uses, so env stays consistent).

set -uo pipefail

STATE_FILE="${STATE_FILE:-$HOME/files/backups/watchdog-state}"
BRIDGE_DIR="${BRIDGE_DIR:-$HOME/gaende-radio}"
FAIL_THRESHOLD="${FAIL_THRESHOLD:-3}"
ENV_FILE="${ENV_FILE:-$HOME/.gaende.env}"

# Source env first so BRIDGE_PORT / MUSIC_BASE_PATH / POSTGRES_PASSWORD / etc
# from ~/.gaende.env are in scope for both the health probe AND a respawn.
if [ -f "$ENV_FILE" ]; then
    set -a
    # shellcheck source=/dev/null
    . "$ENV_FILE"
    set +a
fi

HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:${BRIDGE_PORT:-3001}/health}"

mkdir -p "$(dirname "$STATE_FILE")"
FAILS=0
if [ -f "$STATE_FILE" ]; then
    FAILS=$(cat "$STATE_FILE" 2>/dev/null || echo 0)
    case "$FAILS" in ''|*[!0-9]*) FAILS=0 ;; esac
fi

HTTP=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "$HEALTH_URL" 2>/dev/null || echo 000)

# HTTP 000 only fires when curl itself couldn't connect (ECONNREFUSED, timeout,
# DNS). Anything else — including 503 "degraded" and 500 "internal error" —
# means the process is listening and deliberately reporting its state.
if [ "$HTTP" != "000" ]; then
    if [ "$FAILS" -ne 0 ]; then
        echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) watchdog: recovered (HTTP $HTTP after $FAILS connection failures)"
    fi
    echo 0 > "$STATE_FILE"
    exit 0
fi

FAILS=$((FAILS + 1))
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) watchdog: connection failed (HTTP $HTTP), consecutive=$FAILS"
echo "$FAILS" > "$STATE_FILE"

if [ "$FAILS" -lt "$FAIL_THRESHOLD" ]; then
    exit 0
fi

echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) watchdog: threshold reached ($FAILS >= $FAIL_THRESHOLD), restarting bridge"

# Graceful stop → force if needed.
pkill -TERM -f "^\./bridge\$" 2>/dev/null || true
for _ in 1 2 3 4 5; do
    pgrep -f "^\./bridge\$" >/dev/null 2>&1 || break
    sleep 1
done
pkill -KILL -f "^\./bridge\$" 2>/dev/null || true

cd "$BRIDGE_DIR" || exit 1
nohup env \
    DATABASE_URL="postgres://gaende:${POSTGRES_PASSWORD:-gaende_prod}@127.0.0.1:15432/gaende?sslmode=disable" \
    MPD_HOST=127.0.0.1 \
    MUSIC_BASE_PATH="${MUSIC_BASE_PATH:-$HOME/files/Webradio}" \
    PORT="${BRIDGE_PORT:-3001}" \
    PLEX_URL="${PLEX_URL:-}" \
    PLEX_TOKEN="${PLEX_TOKEN:-}" \
    LASTFM_API_KEY="${LASTFM_API_KEY:-}" \
    ADMIN_TOKEN="${ADMIN_TOKEN:-}" \
    ./bridge > bridge.log 2>&1 &
NEW_PID=$!
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) watchdog: relaunched bridge PID=$NEW_PID"

echo 0 > "$STATE_FILE"
