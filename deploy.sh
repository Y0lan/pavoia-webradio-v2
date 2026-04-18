#!/bin/bash
# GAENDE Radio — Deploy to Whatbox.
#
# Usage: ./deploy.sh
#
# Secrets (PLEX_TOKEN, LASTFM_API_KEY, ADMIN_TOKEN, POSTGRES_PASSWORD) live in
# ~/.gaende.env on Whatbox — see .env.example. This script sources that file on
# the remote side before starting the bridge, so they end up in the bridge's real
# environment (vs the earlier design that passed them through a single-quoted SSH
# heredoc, which silently dropped them because expansion happens on the remote
# shell with nothing set there).

set -euo pipefail

WHATBOX_HOST="yolan@orange.whatbox.ca"
WHATBOX_KEY="$HOME/.ssh/id_ed25519_whatbox"
REMOTE_DIR="~/gaende-radio"
BRIDGE_PORT="${BRIDGE_PORT:-3001}"
WEB_PORT="${WEB_PORT:-13000}"

echo "=== GAENDE Radio Deploy ==="
echo ""

# 1. Cross-compile bridge
echo "[1/6] Building bridge for Linux..."
cd apps/bridge
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../bridge-linux .
cd ../..
echo "  ✓ bridge-linux built"

# 2. Build frontend
echo "[2/6] Building frontend..."
cd apps/web
npm run build
cd ../..
echo "  ✓ frontend built"

# 3. Upload
echo "[3/6] Uploading to Whatbox..."
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" "mkdir -p $REMOTE_DIR"
scp -i "$WHATBOX_KEY" bridge-linux "$WHATBOX_HOST:$REMOTE_DIR/bridge"
scp -i "$WHATBOX_KEY" podman-compose.yml "$WHATBOX_HOST:$REMOTE_DIR/"
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" "chmod +x $REMOTE_DIR/bridge"

rsync -avz -e "ssh -i $WHATBOX_KEY" apps/web/.next "$WHATBOX_HOST:$REMOTE_DIR/web/"
rsync -avz -e "ssh -i $WHATBOX_KEY" apps/web/package.json apps/web/package-lock.json "$WHATBOX_HOST:$REMOTE_DIR/web/"
rsync -avz -e "ssh -i $WHATBOX_KEY" apps/web/public "$WHATBOX_HOST:$REMOTE_DIR/web/"
echo "  ✓ files uploaded"

# 4. Start Podman + install web deps
echo "[4/6] Starting Podman services + installing web deps..."
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" bash -s << 'REMOTE'
set -e
cd ~/gaende-radio

if ! podman ps --format '{{.Names}}' | grep -q '^gaende-postgres$'; then
  podman-compose up -d
  echo "  Waiting for Postgres..."
  sleep 5
fi

cd web
# `ci` is deterministic against the uploaded lockfile; `install --production` was
# not, and swallowed stderr hid failures. `--omit=dev` replaces the deprecated flag.
npm ci --omit=dev
REMOTE
echo "  ✓ services started"

# 5. Start bridge + frontend with env from ~/.gaende.env
echo "[5/6] Starting bridge and frontend..."
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" bash -s << REMOTE
set -e
cd ~/gaende-radio

# Source secrets if the file exists; warn otherwise.
ENV_FILE="\$HOME/.gaende.env"
if [ -f "\$ENV_FILE" ]; then
    set -a
    . "\$ENV_FILE"
    set +a
    echo "  Loaded env from \$ENV_FILE"
else
    echo "  WARN: \$ENV_FILE missing — bridge will run without PLEX_TOKEN/LASTFM_API_KEY/ADMIN_TOKEN." >&2
fi

# Kill existing bridge if running.
pkill -TERM -f "^\./bridge\$" 2>/dev/null || true
sleep 2
pkill -KILL -f "^\./bridge\$" 2>/dev/null || true

# Start bridge in background with explicit env (heredoc runs on remote shell, so
# the variables we just sourced are in scope here).
nohup env \\
  DATABASE_URL="postgres://gaende:\${POSTGRES_PASSWORD:-gaende_dev}@127.0.0.1:15432/gaende?sslmode=disable" \\
  MPD_HOST=127.0.0.1 \\
  MUSIC_BASE_PATH=/home/yolan/files/Webradio \\
  PORT="${BRIDGE_PORT}" \\
  PLEX_URL="http://127.0.0.1:31711" \\
  PLEX_TOKEN="\${PLEX_TOKEN:-}" \\
  LASTFM_API_KEY="\${LASTFM_API_KEY:-}" \\
  ADMIN_TOKEN="\${ADMIN_TOKEN:-}" \\
  ./bridge > bridge.log 2>&1 &
echo "  Bridge PID: \$!"

# Start Next.js in background.
cd web
pkill -TERM -f "next start" 2>/dev/null || true
sleep 2
pkill -KILL -f "next start" 2>/dev/null || true
PORT="${WEB_PORT}" nohup npx next start -p "${WEB_PORT}" > ../web.log 2>&1 &
echo "  Web PID: \$!"
REMOTE

# 6. Health gate — fail the deploy if /health isn't returning "ok".
echo "[6/6] Verifying deploy..."
sleep 5
HEALTH_URL="http://orange.whatbox.ca:${BRIDGE_PORT}/health"
HEALTH_OUT=$(curl -sS -w '\n%{http_code}' "$HEALTH_URL" || true)
HEALTH_CODE=$(printf '%s' "$HEALTH_OUT" | tail -n1)
HEALTH_BODY=$(printf '%s' "$HEALTH_OUT" | sed '$d')
echo "  HTTP $HEALTH_CODE — $HEALTH_BODY"

if [ "$HEALTH_CODE" != "200" ]; then
  echo "  ✗ /health returned $HEALTH_CODE (expected 200). Deploy aborted as failed."
  exit 1
fi
if ! printf '%s' "$HEALTH_BODY" | grep -q '"status":"ok"'; then
  echo "  ✗ /health status != ok. Deploy aborted as failed."
  exit 1
fi
echo "  ✓ bridge healthy"

echo ""
echo "=== Deploy complete ==="
echo "Bridge: http://orange.whatbox.ca:${BRIDGE_PORT}/health"
echo "Web:    http://orange.whatbox.ca:${WEB_PORT}"
