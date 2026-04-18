#!/bin/bash
# Plex → Webradio sync launcher. Non-interactive, cron-safe.
# Sources ~/.config/gaende/plex.env for credentials (PLEX_USERNAME, PLEX_PASSWORD,
# optional PLEX_SERVERNAME, WEBRADIO_FOLDER, SERVER_MUSIC).

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
VENV_PATH="$SCRIPT_DIR/plex_sync_env"
SYNC_SCRIPT="$SCRIPT_DIR/plex_webradio_sync.py"

ENV_FILE="$HOME/.config/gaende/plex.env"
if [ -f "$ENV_FILE" ]; then
    set -a
    . "$ENV_FILE"
    set +a
else
    echo "WARN: $ENV_FILE missing — PLEX_USERNAME/PLEX_PASSWORD must be exported by caller." >&2
fi

if [ ! -d "$VENV_PATH" ]; then
    echo "Creating Python virtual environment..."
    python3 -m venv "$VENV_PATH"
    . "$VENV_PATH/bin/activate"
    pip install --upgrade pip
    pip install plexapi tqdm
    echo "Virtual environment created and packages installed!"
else
    . "$VENV_PATH/bin/activate"
fi

echo "Starting Plex to Webradio sync..."
# Pipe "0" = "Download all playlists" for non-interactive (cron) runs.
echo "0" | python "$SYNC_SCRIPT"
echo "Done!"
