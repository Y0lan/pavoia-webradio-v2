# Pavoia Webradio v2

## What is this

A complete rewrite of the Pavoia festival webradio frontend. v1 is live and working — this is v2, a clean slate.

## v1 reference

- **Repo**: https://github.com/Y0lan/pavoia-webradio
- **Live**: https://pavoia.nicemouth.box.ca
- **Local code**: `/home/seven/pavoia-webradio/`
- **What v1 does**: React + Vite frontend serving 9 MPD live streams for a multi-stage festival (Pavoia). Sidebar lists stages, click to browse, "Switch to this stage" button to change audio. Cover art + now-playing metadata from Plex. Bus mystery easter egg.

## v2 deploys here

- **Repo**: https://github.com/Y0lan/pavoia-webradio-v2
- **Live**: https://pavoia2.nicemouth.box.ca (Whatbox app on port 20002)
- **Local code**: `/home/seven/pavoia-webradio-v2/`
- **Whatbox dir**: `~/files/Webradio/scripts/streaming-metadata-webradio-v2/`
- **Auto-deploy**: GitHub Actions on push to master (builds frontend, rsyncs to Whatbox, restarts server on port 20002)

## Backend (shared with v1)

The backend `server.js` needs to be recreated for v2. It's a Node.js + Express server that:
- Reads 9 MPD stream configs (control ports 6600-6608, HTTP streaming ports 14000-14008)
- Serves `/api/streams` — list of streams with names and URLs
- Serves `/api/streams/:id/now` — current track metadata (title, artist, album, cover) via MPD protocol
- Proxies Plex cover art via `/api/cover-proxy`
- Serves the frontend static files from `frontend/dist/`
- The MPD instances and Plex are already running on Whatbox — v2 just needs to talk to them

## Stream infrastructure (already running on Whatbox)

| Stream | Control Port | HTTP Port | Subdomain |
|--------|-------------|-----------|-----------|
| gaende-favorites | 6600 | 14000 | mpd-gaende-favorites.nicemouth.box.ca |
| ambiance-safe | 6601 | 14001 | mpd-ambiance-safe.nicemouth.box.ca |
| etage-0 | 6602 | 14002 | mpd-etage-0.nicemouth.box.ca |
| fontanna-laputa | 6603 | 14003 | mpd-fontanna-laputa.nicemouth.box.ca |
| palac-dance | 6604 | 14004 | mpd-palac-dance.nicemouth.box.ca |
| palac-slow-hypno | 6605 | 14005 | mpd-palac-slow-hypno.nicemouth.box.ca |
| bermuda-night | 6606 | 14006 | mpd-bermuda-night.nicemouth.box.ca |
| bermuda-day | 6607 | 14007 | mpd-bermuda-day.nicemouth.box.ca |
| closing | 6608 | 14008 | mpd-closing.nicemouth.box.ca |

Streams are MP3 128kbps via MPD's LAME httpd output. CORS enabled (`Access-Control-Allow-Origin: *`).

## v1 lessons learned (critical for v2)

1. **`canplay` event never fires on live MP3 streams** — use `play()` directly, the promise resolves when audio flows
2. **Browsers reject concurrent `play()` on two audio elements** — for zero-silence switching, load the standby element and wait for its `playing` event before touching the active one
3. **Zero silence tolerance** — this plays on a festival floor. Never stop the current stream until the new one is confirmed playing
4. **No Web Audio API** — `MediaElementAudioSourceNode` silently mutes cross-origin audio. Use `element.volume` for everything
5. **Deploy restarts should use `pkill` not PID files** — PID files go stale

## Stages / rooms at Pavoia

| # | Stage | Vibe |
|---|-------|------|
| 1 | Gaende's Favorites | Personal selection, all genres |
| 2 | Ambiance / Safe Space | Chill, relaxing beats |
| 3 | Bermuda Day / Oaza | Sunlit grooves |
| 4 | Bermuda Night | Progressive & Indie, sunset lift |
| 5 | Palac Feel | Melodic, hypnotic, emotional |
| 6 | Palac Dance | High energy, darker side |
| 7 | Fontanna / Laputa | After-hour house, minimal, tech house |
| 8 | Etage 0 | Hard techno, euro dance, trance |
| 9 | Closing | Beautiful closing tracks, no categories |
| 10 | Bus | Mystery — physical only easter egg |

## SSH to Whatbox

```bash
ssh -i ~/.ssh/id_ed25519_whatbox yolan@orange.whatbox.ca
```
