#!/bin/bash
# GAENDE Radio — bridge liveness watchdog.
#
# Install as Whatbox cron: `* * * * * ~/gaende-radio/scripts/ops/bridge-watchdog.sh`
#
# Probes /health every minute. Restarts the bridge ONLY if 3 consecutive probes
# fail (HTTP != 200) — a transient blip doesn't page a restart loop, but a
# sustained outage does. State lives in ~/files/backups/watchdog-state so the
# counter survives cron re-exec.
#
# Restart strategy: SIGTERM with 5s grace → SIGKILL → respawn with env from
# ~/.gaende.env. Uses the same invocation deploy.sh does, so env stays
# consistent across manual deploys and watchdog-driven restarts.

set -uo pipefail

HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:3001/health}"
STATE_FILE="${STATE_FILE:-$HOME/files/backups/watchdog-state}"
BRIDGE_DIR="${BRIDGE_DIR:-$HOME/gaende-radio}"
FAIL_THRESHOLD="${FAIL_THRESHOLD:-3}"
ENV_FILE="${ENV_FILE:-$HOME/.gaende.env}"

mkdir -p "$(dirname "$STATE_FILE")"
FAILS=0
if [ -f "$STATE_FILE" ]; then
    FAILS=$(cat "$STATE_FILE" 2>/dev/null || echo 0)
    case "$FAILS" in ''|*[!0-9]*) FAILS=0 ;; esac
fi

HTTP=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "$HEALTH_URL" || echo 000)

if [ "$HTTP" = "200" ]; then
    # Reset on success. No-op log to keep watchdog.log from overflowing with
    # "all green" spam; only write when state changes.
    if [ "$FAILS" -ne 0 ]; then
        echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) watchdog: recovered (HTTP 200 after $FAILS failures)"
    fi
    echo 0 > "$STATE_FILE"
    exit 0
fi

FAILS=$((FAILS + 1))
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) watchdog: probe failed (HTTP $HTTP), consecutive=$FAILS"
echo "$FAILS" > "$STATE_FILE"

if [ "$FAILS" -lt "$FAIL_THRESHOLD" ]; then
    exit 0
fi

echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) watchdog: threshold reached ($FAILS >= $FAIL_THRESHOLD), restarting bridge"

# Source env file if present (same contract deploy.sh uses).
if [ -f "$ENV_FILE" ]; then
    set -a
    # shellcheck source=/dev/null
    . "$ENV_FILE"
    set +a
fi

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

# Reset counter so we don't immediately try again on the next tick.
echo 0 > "$STATE_FILE"
