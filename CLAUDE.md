# CLAUDE.md

## Rules

- **Never mention Claude, AI, or co-authored-by in git commits, PR descriptions, or changelogs.** No `Co-Authored-By` trailers. No "Generated with Claude Code" footers.
- Challenge spec decisions if a better approach exists. The spec is directional, not sacred.

## Project

**GAENDE Radio** — 9-stage 24/7 webradio with cyber-brutalist terminal aesthetic.
The curation process IS the content. Radical transparency — everything is public except `/admin`.

- **Repo:** https://github.com/Y0lan/pavoia-webradio-v2
- **Seedbox:** `yolan@orange.whatbox.ca` (SSH key: `~/.ssh/id_ed25519_whatbox`)
- **Infra validated:** Podman 5.7.1, 128 cores, 503GB shared RAM, all ports free

## Key Docs (read these first in any new session)

- `docs/DESIGN.md` — Approved architecture + eng review + design review findings
- `docs/DESIGN_SYSTEM.md` — Complete cyber-brutalist visual identity (colors, fonts, geometry, animations, components)
- `docs/GAENDE_WEBRADIO_v3.1_COMPLETE.md` — Full 2000-line spec (~260 features)

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js 15 (App Router) + React 19 + Tailwind 4 + Framer Motion 11 |
| Bridge API | Go 1.24+ — **bare binary on host** (NOT containerized) |
| Database | PostgreSQL 16 (Podman container on Whatbox, local for dev) |
| Cache | Redis 7 (Podman container) |
| Search | Meilisearch (Podman container) |
| State | Zustand 5 |
| Data fetching | TanStack Query v5 |
| Audio | Raw Web Audio API (dual-slot crossfade, 5 curves) |
| Fonts | Syne, JetBrains Mono, Space Mono, Instrument Serif |
| Containers | Podman (rootless, no Docker) |

## Architecture

```
Browser (PWA) ──HTTP/WS/SSE──► Go Bridge (:3001) ──TCP──► MPD ×9 (:6600-6608)
                                    │                        │
                                    ├──► PostgreSQL (:15432)  └──► HTTP streams (:14000-14008)
                                    ├──► Redis (:16379)            (proxied through bridge for CORS)
                                    ├──► Meilisearch (:7700)
                                    └──► Plex (:31711)

Next.js SSR (:3000) ──► serves frontend, talks to bridge API
```

### Critical Design Decisions

- **Audio streams proxied through bridge** — solves CORS (MPD httpd has no CORS headers) + enables listener counting
- **Bridge is bare binary on host** — direct localhost access to MPD + Plex, no Podman networking needed
- **Plex `addedAt` timestamps** — sync worker extracts Plex's addedAt, not Postgres now(). Critical for digging calendar.
- **Stage color scope:** contained to player bar + `/stage/[id]` page only. All other pages use default accent (#00ffc8).
- **WebSocket + SSE dual transport** — WS for per-stage now-playing, SSE for global broadcasts. Fallback to WS-only if proxy blocks SSE.
- **Stats computed on-request** — fast enough at ~3000 tracks. Add materialized views after ~1M track_plays rows.

## What's Built (Steps 1-4 complete + Step 5 draft)

### Go Bridge (`apps/bridge/`)
- `main.go` — HTTP server with graceful shutdown, CORS, health, stages API, hub wiring, API routes
- `config/config.go` — env-based config, 9 stage definitions (matches Whatbox MPD instances)
- `mpd/pool.go` — connection pool for 9 MPD instances with:
  - Context-aware watcher goroutines (reuse connections, proper shutdown)
  - Exponential backoff reconnection with context cancellation
  - Transient error retry (Ping before marking dead)
- `mpd/pool_test.go` — 6 tests
- `db/db.go` — pgx pool + embedded migrations with schema_migrations tracking
- `db/migrations/001_initial_schema.sql` — 8 tables (library_tracks, track_plays, artists, artist_relations, plex_sync_log, plex_playlist_snapshots, wrapped_data, user_preferences)
- `plex/client.go` — Plex API client (playlists, tracks, health with timeout)
- `plex/sync.go` — sync worker (5min interval, upsert with metadata updates, addedAt extraction)
- `hub/hub.go` — WebSocket + SSE hub with:
  - Stage subscription validation, latest-wins backpressure
  - Snapshot cache (new subscribers get current now-playing)
  - SSE keepalive (15s), write error detection, newline sanitization
  - WS OriginPatterns, SetReadLimit(4096), dedicated context
  - Writer goroutine joined via sync.WaitGroup
- `hub/handler.go` — WS upgrade + SSE stream handlers
- `hub/hub_test.go` + `hub/handler_test.go` — 22 tests (13 unit + 9 integration)
- `api/` — REST API endpoints (30 routes):
  - `helpers.go` — pagination, time range, admin auth, JSON helpers (10 tests)
  - `history.go` — play history with filters, calendar heatmap, 7x24 heatmap
  - `digging.go` — digging calendar, date drill-down, streaks, patterns
  - `stats.go` — overview, top artists/tracks, stages, BPM, keys, decades, genres, velocity, heatmap
  - `artists.go` — list, detail, tracks, similar graph
  - `search.go` — Postgres ILIKE fallback (Meilisearch ready)
  - `queue.go` — current track from MPD
  - `routes.go` — central route registration

### Tested & Verified
- 9/9 MPD instances connected via SSH tunnel (control ports 6600-6608)
- Play logging to Postgres working (real tracks from live MPD)
- Schema migrations with tracking table (safe to add new migrations)
- 15 eng review findings fixed (watcher lifecycle, graceful shutdown, migration tracking, etc.)
- Hub: eng review + Codex/Claude outside voice + adversarial review, 17 fixes applied
- 38 tests total: 22 hub + 10 API helpers + 6 MPD

### Step 5 Known Issues (needs TDD pass)
- API endpoint handlers lack database integration tests
- History search filter has argN counter bug
- Digging streaks algorithm untested
- Search is Postgres ILIKE only (Meilisearch not wired yet)

## What's Next (in order)

- **Step 5b:** TDD pass on API endpoints (fix bugs, add integration tests)
- **Step 6:** Artist enrichment pipeline (Last.fm + MusicBrainz)
- **Step 7:** Audio stream proxy (bridge proxies MPD HTTP streams for CORS)
- **Step 8:** Next.js frontend — root layout, audio engine, player bar
- **Step 9:** Frontend pages one by one
- **Step 10:** Deploy to Whatbox

## Deferred to v2

- Wrapped (December 2026)
- Sankey flows, crossover analysis, label treemap
- Three.js particle visualizer (Canvas 2D sufficient for v1)
- Track download (seedbox TOS unresolved)
- Multi-bitrate Icecast
- Discogs + Wikidata + TheAudioDB enrichment
- Digging session clustering
- Multi-year calendar comparison
- Artist alias deduplication
- Accessibility (prefers-reduced-motion, contrast fixes)

## Testing

- **Go:** `cd apps/bridge && go test ./... -v`
- **Framework:** Go stdlib testing
- **Strategy:** TDD for bridge, test-after for frontend (Vitest + Playwright)

## Dev Setup

```bash
# Go (installed to ~/go-install/)
export PATH=~/go-install/go/bin:$HOME/go/bin:$PATH

# SSH tunnel to Whatbox MPD (for local dev)
ssh -i ~/.ssh/id_ed25519_whatbox -f -N \
  -L 6600:localhost:6600 -L 6601:localhost:6601 -L 6602:localhost:6602 \
  -L 6603:localhost:6603 -L 6604:localhost:6604 -L 6605:localhost:6605 \
  -L 6606:localhost:6606 -L 6607:localhost:6607 -L 6608:localhost:6608 \
  yolan@orange.whatbox.ca

# Local Postgres (trust auth on 127.0.0.1)
# DATABASE_URL=postgres://seven@127.0.0.1/gaende?sslmode=disable

# Build and run
cd apps/bridge && go build -o ../../bridge . && cd ../..
DATABASE_URL="postgres://seven@127.0.0.1/gaende?sslmode=disable" MPD_HOST=127.0.0.1 ./bridge

# Cross-compile for Whatbox
cd apps/bridge && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../bridge-linux .
```

## MPD Instance Map (Whatbox)

| Stage ID | Display Name | MPD Control | HTTP Stream | Genre |
|---|---|---|---|---|
| gaende-favorites | Main Stage | :6600 | :14000 | Progressive Melodic Techno |
| ambiance-safe | Ambient Horizon | :6601 | :14001 | Ambient |
| etage-0 | Techno Bunker | :6602 | :14002 | Techno |
| fontanna-laputa | Deep Current | :6603 | :14003 | Deep House |
| palac-dance | Indie Floor | :6604 | :14004 | Indie Dance |
| palac-slow-hypno | Chill Terrace | :6605 | :14005 | Chillout |
| bermuda-night | Bass Cave | :6606 | :14006 | DnB |
| bermuda-day | World Frequencies | :6607 | :14007 | Afro House |
| closing | Live Sets | :6608 | :14008 | Live |

> **Port ordering is non-obvious:** stages 2–5 don't run on ports in stage-order. The MPD config files on Whatbox bind `ambiance-safe` to `6601`, `etage-0` to `6602`, `fontanna-laputa` to `6603`, `palac-dance` to `6604` — and each MPD's `music_directory` is named after its stage, so the content semantically follows the bind. The bridge `config.go` mirrors this. Fixing the "out-of-order" look here would require also rebinding the Whatbox MPDs; otherwise stages will stream the wrong content (previously etage-0 streamed ambient tracks labelled as "Techno"). Diagnosed 2026-04-19; drift is now logged at boot via `slog` "stage port mapping" lines.
