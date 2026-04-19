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

# 0. Pre-flight: validate remote prerequisites BEFORE any destructive action.
#    If these checks fail we exit without touching anything on Whatbox —
#    previously the mise-Node check only ran after the bridge had already been
#    killed + relaunched, leaving the host in a mixed state when the check
#    tripped. Build-host arch is also sanity-checked: Next 16 standalone output
#    includes traced native deps, so local and remote must match.
echo "[0/6] Pre-flight checks..."
LOCAL_ARCH=$(uname -m)
LOCAL_OS=$(uname -s)
if [ "$LOCAL_ARCH" != "x86_64" ] || [ "$LOCAL_OS" != "Linux" ]; then
    echo "  ✗ local host is $LOCAL_OS/$LOCAL_ARCH; Whatbox is Linux/x86_64." >&2
    echo "    bridge cross-compiles fine (GOOS=linux GOARCH=amd64), but the Next" >&2
    echo "    standalone bundle carries traced native node addons compiled for the" >&2
    echo "    builder's OS+arch — a macOS x86_64 build won't boot on Linux." >&2
    exit 1
fi
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" bash -s << 'PREFLIGHT' || exit 1
set -e
MISE_NODE="$HOME/.local/share/mise/installs/node/22.22.2/bin/node"
if [ ! -x "$MISE_NODE" ]; then
    echo "  ✗ $MISE_NODE not found on remote" >&2
    echo "    install via: mise install node@22.22.2" >&2
    exit 1
fi
if ! "$MISE_NODE" -e 'require("node:inspector")' 2>/dev/null; then
    echo "  ✗ mise node at $MISE_NODE cannot load inspector module — Next 16 will refuse to boot" >&2
    exit 1
fi
echo "  ✓ mise node 22.22.2 present and inspector-capable"
PREFLIGHT
echo "  ✓ pre-flight passed"

# 1. Cross-compile bridge
echo "[1/6] Building bridge for Linux..."
cd apps/bridge
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../bridge-linux .
cd ../..
echo "  ✓ bridge-linux built"

# 2. Build frontend locally (output: "standalone" in next.config.ts).
#    We must build on the developer machine rather than on Whatbox — Whatbox's
#    /usr/bin/node is compiled without the inspector module, and Next.js 16
#    requires `node:inspector` at startup. Node 22.22.2 under mise on Whatbox
#    works for RUNTIME; we use that for `node server.js` later.
echo "[2/6] Building frontend..."
cd apps/web
# Bake the public bridge URL into the build. NEXT_PUBLIC_* vars are inlined at
# build time — without this, BRIDGE_URL defaults to "" which resolves to the
# Next.js origin (:13000) and every fetch/WS goes to a dead route. There is no
# reverse proxy in front of the bridge on Whatbox; callers must hit :3001
# directly. Allow override via env for dev/staging.
#
# CORS is handled by the bridge (see main.go corsMiddleware — it echoes the
# request Origin so authenticated cookies would work if we ever add auth,
# and falls back to "*" for listeners).
: "${PUBLIC_BRIDGE_URL:=http://orange.whatbox.ca:${BRIDGE_PORT}}"
NEXT_PUBLIC_BRIDGE_URL="$PUBLIC_BRIDGE_URL" NEXT_TELEMETRY_DISABLED=1 npm run build
cd ../..
echo "  ✓ frontend built (standalone output, bridge=$PUBLIC_BRIDGE_URL)"

# 3. Upload
#
# Stage the new bridge as `bridge.new` instead of overwriting `bridge`
# in place. Linux refuses to overwrite a busy text file (ETXTBSY) so scp
# straight onto a running binary fails with "dest open: Failure". We
# promote bridge.new → bridge in step [5/6] after pkill-ing the running
# process, which keeps redeploys idempotent even when the current bridge
# is healthy.
echo "[3/6] Uploading to Whatbox..."
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" "mkdir -p $REMOTE_DIR"
scp -i "$WHATBOX_KEY" bridge-linux "$WHATBOX_HOST:$REMOTE_DIR/bridge.new"
scp -i "$WHATBOX_KEY" podman-compose.yml "$WHATBOX_HOST:$REMOTE_DIR/"
ssh -i "$WHATBOX_KEY" "$WHATBOX_HOST" "chmod +x $REMOTE_DIR/bridge.new"

# Frontend ships as the Next.js "standalone" bundle — a self-contained
# app/server.js + its trimmed node_modules, plus static + public served
# alongside. We do NOT rsync the big node_modules / dev deps.
rsync -avz --delete -e "ssh -i $WHATBOX_KEY" \
    apps/web/.next/standalone/ "$WHATBOX_HOST:$REMOTE_DIR/web/standalone/"
rsync -avz --delete -e "ssh -i $WHATBOX_KEY" \
    apps/web/.next/static/ "$WHATBOX_HOST:$REMOTE_DIR/web/standalone/.next/static/"
rsync -avz --delete -e "ssh -i $WHATBOX_KEY" \
    apps/web/public/ "$WHATBOX_HOST:$REMOTE_DIR/web/standalone/public/"
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
# The standalone bundle we rsync'd already includes its own pruned
# node_modules — no npm ci needed on Whatbox (which would fail anyway with
# the inspector-missing system node).
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

# Promote staged binary (bridge.new → bridge). Only done AFTER the pkill so
# the old process can't be holding the text-file-busy lock on the target.
# If bridge.new doesn't exist, preserve whatever's currently installed —
# that way an interrupted upload earlier in this run doesn't leave us with
# no binary at all.
if [ -f bridge.new ]; then
    mv -f bridge.new bridge
    chmod +x bridge
    echo "  Promoted bridge.new → bridge"
fi

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

# Start Next.js standalone server. Use the mise-installed Node 22.22.2 —
# Whatbox's /usr/bin/node is compiled without the inspector module and
# Next 16 refuses to start against it. Presence was validated in step [0/6].
#
# Launch with an ABSOLUTE path to server.js so the resulting cmdline
# contains ".../web/standalone/server.js", which pkill's path-based match
# can find reliably — cd'ing in and running "server.js" relative made the
# pkill pattern never hit the newly-launched process.
MISE_NODE="\$HOME/.local/share/mise/installs/node/22.22.2/bin/node"
WEB_SERVER="\$HOME/gaende-radio/web/standalone/server.js"
WEB_PID_FILE="\$HOME/gaende-radio/web.pid"

# --- Steady-state kill path: PID file with validation ---
#
# Previous deploys have been burned by a grab-bag of partial fixes
# (cmdline pkill, ss-based port scan, /proc/cwd pattern match) because
# Next rewrites process.title to "next-server (vX.Y.Z)" a few minutes
# after boot and shared hosting denies ss -tlnp to non-root users. The
# clean design — codex-reviewed 2026-04-19 — is: we own the PID because
# WE launched it, so write it to disk and read it back next time.
# Validate before killing so PID reuse doesn't bite us: the same PID
# must (a) still exist, (b) still belong to a process whose cwd is under
# our install tree. Only then TERM; only KILL if that same PID still
# validates after a short wait.
kill_old_web_by_pidfile() {
    [ -f "\$WEB_PID_FILE" ] || return 0
    local old_pid
    old_pid=\$(cat "\$WEB_PID_FILE" 2>/dev/null | tr -d '[:space:]')
    [ -n "\$old_pid" ] || return 0
    [ -e "/proc/\$old_pid" ] || { echo "  pidfile PID \$old_pid gone; clearing"; rm -f "\$WEB_PID_FILE"; return 0; }
    local cwd
    cwd=\$(readlink "/proc/\$old_pid/cwd" 2>/dev/null || echo "")
    case "\$cwd" in
        */gaende-radio|*/gaende-radio/*) ;;
        *) echo "  pidfile PID \$old_pid has unrelated cwd=\$cwd; refusing to kill"; return 0 ;;
    esac
    echo "  stopping previous web (PID \$old_pid, cwd=\$cwd)"
    kill -TERM "\$old_pid" 2>/dev/null || true
    local i
    for i in 1 2 3 4 5; do
        [ -e "/proc/\$old_pid" ] || break
        sleep 1
    done
    if [ -e "/proc/\$old_pid" ]; then
        # Re-check cwd before KILL in case this PID was reused during the wait.
        local cwd2
        cwd2=\$(readlink "/proc/\$old_pid/cwd" 2>/dev/null || echo "")
        case "\$cwd2" in
            */gaende-radio|*/gaende-radio/*)
                echo "  escalating to KILL on PID \$old_pid"
                kill -KILL "\$old_pid" 2>/dev/null || true
                ;;
            *)
                echo "  PID \$old_pid no longer owned by us (cwd=\$cwd2); leaving alone"
                ;;
        esac
    fi
    rm -f "\$WEB_PID_FILE"
}
kill_old_web_by_pidfile

PORT="${WEB_PORT}" HOSTNAME=0.0.0.0 nohup "\$MISE_NODE" "\$WEB_SERVER" > ~/gaende-radio/web.log 2>&1 &
WEB_PID=\$!
echo "\$WEB_PID" > "\$WEB_PID_FILE"
echo "  Web PID: \$WEB_PID (written to \$WEB_PID_FILE)"
REMOTE

# 6. Health gate — fail the deploy if bridge or web isn't responding.
echo "[6/6] Verifying deploy..."
sleep 5
HEALTH_URL="http://orange.whatbox.ca:${BRIDGE_PORT}/health"
HEALTH_OUT=$(curl -sS -w '\n%{http_code}' "$HEALTH_URL" || true)
HEALTH_CODE=$(printf '%s' "$HEALTH_OUT" | tail -n1)
HEALTH_BODY=$(printf '%s' "$HEALTH_OUT" | sed '$d')
echo "  bridge HTTP $HEALTH_CODE — $HEALTH_BODY"

if [ "$HEALTH_CODE" != "200" ]; then
  echo "  ✗ /health returned $HEALTH_CODE (expected 200). Deploy aborted as failed."
  exit 1
fi
if ! printf '%s' "$HEALTH_BODY" | grep -q '"status":"ok"'; then
  echo "  ✗ /health status != ok. Deploy aborted as failed."
  exit 1
fi
echo "  ✓ bridge healthy"

# Web readiness — nohup backgrounded the server; if it crashed immediately,
# curl fails here. We probe several routes to spread the check surface: a
# dead server returns nothing regardless of route, but a half-broken
# deployment might serve `/` (which is a client component with local UI
# only) while `/artists`, `/digging`, `/stats` all 500. The /api/stages
# endpoint ON THE BRIDGE (not proxied through web) is the canonical
# web→bridge→DB pipeline check; we already validate that in the /health
# probe above, so this block just needs to prove Next is serving shells.
for path in / /artists /digging /stats; do
    url="http://orange.whatbox.ca:${WEB_PORT}${path}"
    code=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "$url" || echo 000)
    echo "  web HTTP $code  $path"
    if [ "$code" != "200" ]; then
        echo "  ✗ web $path returned $code. Deploy aborted as failed." >&2
        exit 1
    fi
done
echo "  ✓ web healthy"

echo ""
echo "=== Deploy complete ==="
echo "Bridge: http://orange.whatbox.ca:${BRIDGE_PORT}/health"
echo "Web:    http://orange.whatbox.ca:${WEB_PORT}"
