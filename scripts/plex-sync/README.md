# Plex → Webradio sync

Python script that walks the user's Plex playlists, creates symlinks under
`$WEBRADIO_FOLDER/<PlaylistName>/`, and writes JSON artifacts the Go bridge
will later consume as the single source of truth for library metadata.

## Why this exists

MPD's per-stage music directories symlink into folders under
`$WEBRADIO_FOLDER`. The Python script keeps those folders in sync with Plex
playlist membership. It also produces JSON artifacts with richer metadata than
MPD exposes natively (addedAt, Plex rating keys, genres, moods, cover art URLs).

## Artifacts written (under `WEBRADIO_FOLDER`)

| File | Purpose |
|---|---|
| `<Playlist>/<file>.mp3` | symlink → real file in `plex_music_library` |
| `<Playlist>/<file>.mp3.json` | per-track sidecar metadata |
| `artists.json` | artist database with albums + tracks + playlist associations |
| `playlists.json` | all playlists with artists + tracks |
| `albums.json` | albums with track listings |
| `tracks_index.json` | flat track index keyed by Plex ratingKey |
| `sync_manifest.json` | **written last** — sha256 + size + mtime of each top-level artifact, plus a `generation_id`. Downstream readers must verify these hashes before trusting the generation. A mismatch means a partial write; reject. |

All writes are atomic (tempfile in the same directory → `fsync` → `os.replace()`).
The four top-level JSONs are written first; the manifest is written LAST.

## Credentials

`start.sh` sources `~/.config/gaende/plex.env`. Copy `plex.env.example` there and fill in:

```bash
mkdir -p ~/.config/gaende
cp scripts/plex-sync/plex.env.example ~/.config/gaende/plex.env
chmod 600 ~/.config/gaende/plex.env
$EDITOR ~/.config/gaende/plex.env
```

## Cron

```
0 */6 * * * /home/yolan/files/Webradio/scripts/start.sh >> /home/yolan/files/sync_plex_playlist.log 2>&1
```

## Stage ↔ Playlist mapping (frozen)

The radio has 9 stages; each maps to one or two Plex playlists. **Do not add or
remove stages, or rename playlists, without also updating the Go bridge config
and the MPD music-dir symlinks.**

| Stage ID | Plex playlist(s) |
|---|---|
| gaende-favorites | `❤️ Tracks` |
| etage-0 | `ETAGE 0`, `Etage 0 - FAST DARK MINIMAL` |
| ambiance-safe | `AMBIANCE` |
| palac-dance | `PALAC - DANCE` |
| fontanna-laputa | `FONTANNA`, `MINIMAL` |
| palac-slow-hypno | `PALAC - SLOW AND HYPNOTIC - POETIC` |
| bermuda-night | `BERMUDA - AFTER 6` |
| bermuda-day | `BERMUDA - BEFORE 6` |
| closing | `Outro` |

Other Plex playlists (`BPM`, `MIRROR FÉVRIER`, `MOME`, `MOME 2`, `TRA`, `XO`) are
Yolan's personal drafts — Python still syncs them to disk but the radio ignores
them.

See `MPD_DEPLOYMENT.md` for the historical MPD multi-stream deployment notes.
