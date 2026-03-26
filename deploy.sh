#!/bin/bash
# GAENDE Radio — Deploy to Whatbox
# Usage: ./deploy.sh

set -euo pipefail

WHATBOX_HOST="yolan@orange.whatbox.ca"
WHATBOX_KEY="$HOME/.ssh/id_ed25519_whatbox"
REMOTE_DIR="~/gaende-radio"

echo "=== GAENDE Radio Deploy ==="
echo ""

# 1. Cross-compile bridge
echo "[1/5] Building bridge for Linux..."
cd apps/bridge
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../bridge-linux .
cd ../..
echo "  ✓ bridge-linux built"

# 2. Build frontend
echo "[2/5] Building frontend..."
cd apps/web
npm run build
cd ../..
echo "  ✓ frontend built"

# 3. Upload to Whatbox
echo "[3/5] Uploading to Whatbox..."
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" "mkdir -p $REMOTE_DIR"
scp -i "$WHATBOX_KEY" bridge-linux "$WHATBOX_HOST:$REMOTE_DIR/bridge"
scp -i "$WHATBOX_KEY" podman-compose.yml "$WHATBOX_HOST:$REMOTE_DIR/"
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" "chmod +x $REMOTE_DIR/bridge"

# Upload frontend build
rsync -avz -e "ssh -i $WHATBOX_KEY" apps/web/.next "$WHATBOX_HOST:$REMOTE_DIR/web/"
rsync -avz -e "ssh -i $WHATBOX_KEY" apps/web/package.json apps/web/package-lock.json "$WHATBOX_HOST:$REMOTE_DIR/web/"
rsync -avz -e "ssh -i $WHATBOX_KEY" apps/web/public "$WHATBOX_HOST:$REMOTE_DIR/web/"
echo "  ✓ files uploaded"

# 4. Start services on Whatbox
echo "[4/5] Starting services..."
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" << 'REMOTE'
cd ~/gaende-radio

# Start Podman containers (if not running)
if ! podman ps | grep -q gaende-postgres; then
  podman-compose up -d
  echo "  Waiting for Postgres..."
  sleep 5
fi

# Install frontend dependencies and start
cd web
npm install --production 2>/dev/null
cd ..

echo "  Containers ready"
REMOTE
echo "  ✓ services started"

# 5. Start bridge + frontend
echo "[5/5] Starting bridge and frontend..."
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" << 'REMOTE'
cd ~/gaende-radio

# Kill existing bridge if running
pkill -f "./bridge" 2>/dev/null || true
sleep 1

# Start bridge in background
nohup env \
  DATABASE_URL="postgres://gaende:${POSTGRES_PASSWORD:-gaende_dev}@127.0.0.1:15432/gaende?sslmode=disable" \
  MPD_HOST=127.0.0.1 \
  PLEX_URL="http://127.0.0.1:31711" \
  PLEX_TOKEN="${PLEX_TOKEN}" \
  LASTFM_KEY="${LASTFM_KEY}" \
  ADMIN_TOKEN="${ADMIN_TOKEN}" \
  ./bridge > bridge.log 2>&1 &

echo "  Bridge PID: $!"

# Start Next.js in background
cd web
pkill -f "next start" 2>/dev/null || true
sleep 1
nohup npx next start -p 3000 > ../web.log 2>&1 &
echo "  Web PID: $!"

sleep 2
echo "  Checking bridge health..."
curl -s http://127.0.0.1:3001/health | head -c 200
echo ""
REMOTE

echo ""
echo "=== Deploy complete ==="
echo "Bridge: http://orange.whatbox.ca:3001/health"
echo "Web:    http://orange.whatbox.ca:3000"
