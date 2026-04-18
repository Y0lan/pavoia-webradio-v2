# GAENDE Radio — System Architecture

This document describes the live system on `orange.whatbox.ca` as of 2026-04-18.
Section claims are cited to `file:line`. Reviewed adversarially by Codex (`high` reasoning); validated challenges are incorporated below.

---

## 1. Intention

GAENDE Radio is **Yolan's DJ crate, streamed as a 24/7 public radio**, wrapped in a **radically transparent analytics view of the music collection itself**.

The curation process IS the content:
- Each Plex playlist is a "stage" — a genre/mood/context curated manually as a DJ.
- Every track that ever plays on any stage is logged to Postgres; the dashboards surface *what* is being played, *when*, *how often*, by *whom*, and *how the collection grew over time*.
- The `/digging` calendar is the heartbeat — it shows every day a track was added, so visitors see the crate evolve in real time.
- The only private surface is `/admin`. Everything else — every track, every artist, every play-count — is public.

The webapp is therefore **two things at once**:
1. A playable multi-stage radio (9 channels, switchable like SomaFM but curated as a DJ set).
2. A public mirror of Yolan's library: browse artists, dig the catalogue by date, compare stages, see when a track was dropped into the rotation.

---

## 2. Physical Layout (Whatbox seedbox)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  orange.whatbox.ca  (Gentoo, 64-core EPYC, shared hosting, rootless only)   │
│                                                                             │
│  ┌───────────────────────────┐      ┌──────────────────────────────────┐    │
│  │ Plex Media Server         │      │ Music library (flat files)       │    │
│  │   (PID 111865, ~/Library) │      │   ~/files/plex_music_library/    │    │
│  │   library: playlists +    │◀────▶│                                  │    │
│  │   tagged metadata         │      │                                  │    │
│  └───────────────────────────┘      └──────────────────────────────────┘    │
│              │ plexapi (user/pass, SERVERNAME=AEGIR)                        │
│              ▼                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ Python sync  (cron,  0 */6 * * *  @ ~/files/Webradio/scripts/)      │    │
│  │   start.sh → plex_webradio_sync.py                                  │    │
│  │   → creates symlinks in ~/files/Webradio/<PlexPlaylistName>/*.mp3   │    │
│  │   → writes sidecar *.mp3.json with Plex metadata                    │    │
│  │   → rebuilds artists.json, albums.json, playlists.json, tracks_idx  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│              │                                                              │
│              ▼ (symlinks)                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ ~/.local/share/mpd/<stage-id>/music/ → symlink → ~/files/Webradio/  │    │
│  │ 9 MPD instances, all with  auto_update "yes"                        │    │
│  │   ~/bin/mpd ~/.config/mpd/<stage-id>/mpd.conf                       │    │
│  │   control :6600-:6608   |   HTTP httpd stream :14000-:14008 @128k   │    │
│  │   started via @reboot cron loop                                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│              │  (TCP control)                            │  (MP3 HTTP)      │
│              ▼                                           ▼                  │
│  ┌────────────────────────┐            ┌───────────────────────────────┐    │
│  │ Go bridge (bare binary)│            │ Podman rootless               │    │
│  │   ./bridge  :3001      │──DATABASE─▶│  gaende-postgres  :15432      │    │
│  │   reads MPD (control + │            │  (redis + meili defined in    │    │
│  │   proxies streams for  │            │   compose BUT bridge code     │    │
│  │   listener counting)   │            │   does not use them)          │    │
│  │   writes track_plays   │            └───────────────────────────────┘    │
│  │   reads library_tracks │                                                 │
│  │   + WS, SSE, REST API  │                                                 │
│  └────────────────────────┘                                                 │
│              ▲                                                              │
│              │ HTTP / WS / SSE                                              │
│  ┌───────────┴────────────┐                                                 │
│  │ Next.js frontend       │                                                 │
│  │   npx next start       │                                                 │
│  │   running :13000 today;│                                                 │
│  │   deploy.sh says :3000 │                                                 │
│  └────────────────────────┘                                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. The Nine Stages

Stage IDs, names, ports and genres are the defaults from `apps/bridge/config/config.go:92-101`. A `STAGES` env var can override the whole list (`config.go:84`), but no override is currently in the running bridge env.

Source-folder column is from `~/files/Webradio/scripts/readme.md` on the server — it is **not** part of `StageConfig`, which has no playlist-name or folder-name field (`config.go:10-22`).

| Stage ID             | Display              | MPD port | Stream port | Source folder(s) in `~/files/Webradio/`              | Genre                       |
|----------------------|----------------------|----------|-------------|------------------------------------------------------|-----------------------------|
| `gaende-favorites`   | Main Stage           | 6600     | 14000       | `❤️ Tracks`                                           | Progressive Melodic Techno |
| `etage-0`            | Techno Bunker        | 6601     | 14001       | `ETAGE 0` **+** `Etage 0 - FAST DARK MINIMAL`        | Techno                      |
| `ambiance-safe`      | Ambient Horizon      | 6602     | 14002       | `AMBIANCE`                                           | Ambient                     |
| `palac-dance`        | Indie Floor          | 6603     | 14003       | `PALAC - DANCE`                                      | Indie Dance                 |
| `fontanna-laputa`    | Deep Current         | 6604     | 14004       | `FONTANNA` **+** `MINIMAL`                           | Deep House                  |
| `palac-slow-hypno`   | Chill Terrace        | 6605     | 14005       | `PALAC - SLOW AND HYPNOTIC - POETIC`                 | Chillout                    |
| `bermuda-night`      | Bass Cave            | 6606     | 14006       | `BERMUDA - AFTER 6`                                  | DnB                         |
| `bermuda-day`        | World Frequencies    | 6607     | 14007       | `BERMUDA - BEFORE 6`                                 | Afro House                  |
| `closing`            | Live Sets            | 6608     | 14008       | `Outro`                                              | Live                        |

Live verification (2026-04-18):
- **Control ports 6600-6608:** each replies `OK MPD 0.25.0` to `status` with `random: 1, repeat: 1, single: 0, consume: 0` (classic-radio shuffle).
- **HTTP stream ports 14000-14008:** each returns HTTP 200.
- **Process list:** `ps aux | grep mpd` shows 9 running under `~/bin/mpd`.

---

## 4. How a track reaches the radio

There are three pipelines, all independent in the code. The bridge is **not** "Postgres-only" — it reads MPD for now-playing (`main.go:75, 102, 104`), proxies HTTP streams (`api/stream.go:53`), serves MPD-backed queue (`api/queue.go:25`), and only *writes* to Postgres for play logging and library sync.

### 4a. Plex → disk → MPD (the audible pipeline — Python, cron)

```
Add a track to Plex playlist "AMBIANCE"
    │
    ▼
Cron 0 */6 * * *  runs  ~/files/Webradio/scripts/start.sh
    │
    ▼  (plexapi walks every playlist)
plex_webradio_sync.py
  • creates/refreshes symlink:
      ~/files/Webradio/AMBIANCE/<filename>.mp3  →  ~/files/plex_music_library/.../track.mp3
  • writes sidecar <filename>.mp3.json with Plex metadata
  • rebuilds ~/files/Webradio/{artists,albums,playlists,tracks_index}.json
    │
    ▼
MPD (auto_update "yes") picks up new symlink under its music/ subtree
    │
    ▼
Track enters the random-shuffle queue
```

Lag: up to **6 hours** (next cron tick) plus MPD scan (seconds). This is the only pipeline that controls what listeners actually hear.

### 4b. Plex → Postgres (the analytics pipeline — Go bridge)

`plex.SyncWorker` runs every 5 minutes from `main.go:115` when both `PLEX_TOKEN` and `PLEX_URL` are set and Plex is healthy. For each `StageMapping` it looks up a Plex playlist **by case-insensitive title match on the stage ID string** (`plex/sync.go:82`), then upserts tracks into `library_tracks` on `file_path` conflict (`sync.go:137-150`). Plex's `addedAt` is what drives the `/digging` calendar (`sync.go:129`).

```
SyncWorker.SyncAll → Client.Playlists()
    │
    ▼ for each mapping { PlaylistName: stageID, StageID: stageID }   (main.go:339-348)
Client.PlaylistTracks(ratingKey)
    │
    ▼
UPSERT into library_tracks (ON CONFLICT file_path DO UPDATE)
INSERT into plex_sync_log (added, removed, total, error)
```

This pipeline feeds `/api/digging`, `/api/artists`, the `library_tracks` half of `/api/stats`, and the filter joins in `/api/history`. It never writes to MPD.

### 4c. MPD → Postgres (the live-ear pipeline — Go bridge)

```
mpd.Pool watcher idle "player"  (mpd/pool.go:241)
    │
    ▼
NowPlaying callback → main.go:75
    ├─ hub.BroadcastNowPlaying (WS fan-out)
    └─ playCh <- np   (buffered chan cap 64; dropped if full, main.go:94-98)
                │
                ▼ (single-worker drain)
            logPlay → INSERT track_plays (main.go:314)
```

This is what the "now playing" badges and `/api/history` rows come from.

---

## 5. Current operational state (observed 2026-04-18)

**What works:**
- 9 MPDs streaming; all control + HTTP ports healthy.
- Bridge binary up since 2026-03-26 (~23 days).
- Postgres container reachable on `:15432`.
- Frontend serving on `:13000`.

**What is broken or degraded — evidence-linked:**

| Finding | Evidence | Severity |
|---|---|---|
| **Cron path typo** blocks the entire audible sync | `crontab -l` → `~/files/Webradio/script/start.sh` (singular). Real path is `scripts/` (plural). `sync_plex_playlist.log` is 100% "No such file or directory." | Audible pipeline down — new Plex additions can't reach MPD without manual sync. |
| **Go Plex sync disabled** | Bridge env has only `DATABASE_URL` and `MPD_HOST`; log line `"plex not configured — skipping sync"`. `library_tracks = 0, plex_sync_log = 0, artists = 0`. | Dashboards empty. |
| **Even if enabled, the Go sync would fail to find any playlist** | Actual Plex playlist titles are `❤️ Tracks, AMBIANCE, ETAGE 0, Etage 0 - FAST DARK MINIMAL, FONTANNA, MINIMAL, PALAC - DANCE, PALAC - SLOW AND HYPNOTIC - POETIC, BERMUDA - AFTER 6, BERMUDA - BEFORE 6, Outro` (from live `playlists.json`). The Go bridge searches by title equal to the *stage ID* (`ambiance-safe`, `gaende-favorites`, …) — `main.go:343`, `sync.go:82`. Zero matches. | **Architectural bug.** Sync was never going to work against this Plex server without a name-mapping table. Also, `StageMapping` has only one `PlaylistName` per stage, so `etage-0` and `fontanna-laputa` (two Plex playlists each) can't be fully represented. |
| **Env-var name mismatch for Last.fm** | Bridge reads `LASTFM_API_KEY` (`config.go:52`). `deploy.sh:77` exports `LASTFM_KEY`. Enrichment would stay off even with the deploy variable populated. | Enrichment can't be turned on by the current deploy script alone. |
| **`track_plays` stopped growing on 2026-03-26 22:05** | DB: last `played_at = 2026-03-26 22:05:46`, 36 rows total. Bridge has been up since. | Cause is NOT proven by this evidence — could be watcher failure, MPD `file` duplicates producing no distinct events, or queue-full drops (`main.go:94-98`). Needs investigation, not blind restart. |
| **Health says `mpd="down"`** | `/health` reads live `pool.IsAlive(id)` (`main.go:146, 324, mpd/pool.go:137`), not the startup `connected` count. So `alive` flipped false sometime after boot. Not a contradiction with "connected 9/9 at startup"; it is expected that live state drifts. | Real signal that at least one MPD was marked dead; watcher liveness is independent from connection liveness so watchers can be broken silently even when `alive=true`. |
| **Redis + Meilisearch not running** | Only `gaende-postgres` in `podman ps`. | **Impact is smaller than it sounds:** the bridge code never reads from Redis, and Meilisearch is not integrated — `api/search.go` uses Postgres ILIKE regardless. Starting them changes nothing in current behavior. |
| **podman exec is broken** | `podman exec gaende-postgres …` fails with OCI crun-state error. The DB process still serves over `:15432` (psql works). | Management commands need a container restart to clear the corrupted state. Data is fine. |
| **Frontend port drift** | `PORT=13000` in live web process env; `deploy.sh:87` uses `-p 3000`; docs/README say `3000`. | Bookmarks and reverse-proxy rules will be wrong if anyone re-reads the docs. |
| **No respawn for bridge + web** | `deploy.sh:71, 87` start them via `nohup`. No `@reboot` cron (unlike the MPDs, which have one). Whatbox has no `systemctl --user`. | On next reboot, only MPDs come back; bridge + web require manual redeploy. |

---

## 6. `deploy.sh` review

Read end-to-end. It is a working one-shot helper, not a production deploy. The critiques below are the ones that survive checking against the script text.

1. **Heredoc is single-quoted (`deploy.sh:43, 64 << 'REMOTE'`)** → `${PLEX_TOKEN}`, `${LASTFM_KEY}`, `${ADMIN_TOKEN}` and `${POSTGRES_PASSWORD:-gaende_dev}` expand on the **remote** shell, where none of those vars are set. The running bridge env confirms: only `DATABASE_URL` and `MPD_HOST` survived. Fix by sourcing a `.env` file on Whatbox before the `nohup env …` block.

2. **Last.fm var name is wrong.** `deploy.sh:77` sets `LASTFM_KEY`; `config.go:52` reads `LASTFM_API_KEY`. Even if #1 is fixed, enrichment stays off until one side is renamed.

3. **No respawn.** Both bridge and web are `nohup &`. Unlike MPD, no `@reboot` cron entry. Add equivalent entries for bridge/web.

4. **Container health not verified.** `podman ps | grep gaende-postgres` only proves the container *exists*; it does not check health, does not detect the crun-state corruption we are seeing right now.

5. **No health gate.** `curl -s http://127.0.0.1:3001/health | head -c 200` prints output but never sets exit status. A deploy against a broken bridge still ends with "Deploy complete."

6. **`set -euo pipefail` not inherited** into the SSH heredocs. Remote failures do not fail the script.

7. **Redis + Meilisearch start but are not used.** `podman-compose up -d` in `deploy.sh:48` does bring up redis and meili; their absence now is because the container state broke, not because deploy skipped them. Starting them also gains nothing today — the bridge code doesn't talk to Redis at all, and `api/search.go` uses Postgres ILIKE regardless of Meilisearch availability. If those are the intended architecture, that is future work, not a deploy-script fix.

8. **`npm install --production` vs `npm ci`.** `deploy.sh:55` runs `npm install --production 2>/dev/null` on the remote against the uploaded lockfile. This can silently resolve different versions than the ones the build was compiled against, and swallowing stderr hides any real failure. Use `npm ci --omit=dev` instead.

9. **Port drift.** `deploy.sh:87` hard-codes `next start -p 3000`, but the live process is on `:13000`. Pick one and align deploy + docs + reverse-proxy expectations. No evidence in the repo of *why* they diverged.

Skipped (from earlier drafts) because Codex was right they were speculation or framed wrong:
- "Someone edited the env later." Pure guess. The repo only proves the deploy script *tries* 3000.
- Suggesting `node_modules` be shipped instead of reinstalled. Bad idea across machines; `npm ci` is the right answer.

---

## 7. Revised plan (2026-04-18 after dual-voice review)

**Original §7 was reframed after `/autoplan` CEO dual-voice review surfaced a buried third option and 7 missing gaps. See §8 for the findings that drove this revision. Original §7 is preserved in `~/.gstack/projects/Y0lan-pavoia-webradio-v2/main-autoplan-restore-*.md`.**

### Core architectural decision

**Go bridge becomes a consumer of disk artifacts, not a Plex client.** The Python cron (`~/files/Webradio/scripts/plex_webradio_sync.py`) remains the single Plex consumer. The bridge reads `artists.json`, `albums.json`, `playlists.json`, `tracks_index.json`, and per-track `*.mp3.json` sidecars that Python already writes. This eliminates: Plex token in the bridge, name-mapping table, multi-playlist-per-stage gymnastics, polling cadence drift, and duplicated metadata logic.

### Ordered implementation

**Phase A — Emergency hotfix (unblock audible pipeline).**
1. **Fix the cron typo.** `crontab -e` → `/script/` → `/scripts/`. One line. Restores the Python sync so new Plex additions propagate to MPD within 6h.

**Phase B — Stop the bleeding (correctness bugs visible to users).**
2. **Investigate the `track_plays` stall before restarting.** §5 says cause is unproven. Before killing the bridge: inspect `/proc/88202/fd` for open MPD socket state, run `tcpdump` for 30s on ports 6600-6608 to see if watcher idle calls are still in flight, check `bridge.log` growth since Mar 26. Decide whether to restart, reconnect, or patch. Restart destroys evidence.
3. **Fix `track_plays.duration_sec`** (`main.go:315`) — `logPlay` currently inserts 5 fields; duration from `NowPlaying` is dropped. `total_hours` in `/api/stats` is stuck at zero until this is fixed.
4. **Fix `/health` to report real status** (`main.go:164`). Top-level `status:"ok"` is hardcoded. Should be `"ok"` only if all listed checks pass, `"degraded"` if some pass, `"down"` if critical (DB, all MPDs) fail. HTTP 503 for degraded/down.

**Phase C — Wire up the disk-import pipeline (replace Plex sync).**
5. **Build `disk_sync` package.** New `apps/bridge/disk/` replaces `apps/bridge/plex/`. Worker watches `~/files/Webradio/{artists,albums,playlists,tracks_index}.json` for mtime changes; rereads on change. Unmarshals + upserts into Postgres.
6. **Populate `artists` table.** Today nothing writes to it — `/api/artists` and `enrichment/worker.go` are dead. The disk importer must upsert artists from `artists.json`, then link tracks to artists.
7. **Fix `library_tracks.stage_id` cardinality** (`db/migrations/001_initial_schema.sql:55`). One stage per file_path doesn't represent `etage-0` (2 playlists) or `fontanna-laputa` (2 playlists). Either: (a) migrate to a join table `track_stages(file_path, stage_id)` primary-key on both, OR (b) explicitly accept "last-synced wins" and document it. Option (a) is correct; option (b) is defensible if simpler wins for v1.
8. **Decommission the Go Plex client.** Delete `apps/bridge/plex/`, remove `PLEX_URL`/`PLEX_TOKEN` from config, remove Plex wiring in `main.go:106-124`. Update tests. `plex_sync_log` table can stay for historical reference or be renamed `disk_sync_log`.

**Phase D — Observability (prevent the next silent outage).**
9. **Add freshness probe.** `/health.checks.disk_sync`: seconds since last successful import. `/health.checks.plays`: seconds since last `track_plays` row. Both exposed so an external monitor can alert.
10. **Add Postgres backup job.** `pg_dump gaende` to `~/files/backups/` nightly via cron; retain 7 days. Currently zero backup of `track_plays`.
11. **Replace the fake health gate.** `/api/stages returns 9` is useless (it iterates config, not live state). Real gate: `/health.status == "ok"` AND `/api/stages[*].alive == true` for all 9.

**Phase E — Deploy hardening.**
12. **Put secrets in `~/.gaende.env` on Whatbox.** Source before `nohup env …` in `deploy.sh`. Stop depending on heredoc quoting. Drop `LASTFM_KEY`/`LASTFM_API_KEY` ambiguity — pick one name in both places.
13. **Replace `@reboot` with real supervision.** `@reboot` only catches host reboot, not crash. Use a 1-min watchdog cron that restarts the bridge if `curl -sf http://127.0.0.1:3001/health` fails. Same for web.
14. **Fix podman-compose crun state.** `podman system reset` on `gaende-postgres` (data is on a volume, safe). Redis + Meili stay off until code actually uses them.
15. **Align frontend port.** Pick one. Either `deploy.sh:87` → `-p 13000` or the running process → `3000`. Update all docs.

### What stays in scope for this cycle vs deferred

In scope (Phases A–E above): everything listed. **Eng review will validate tests, architecture, and risk for each.**

Deferred (not this cycle):
- Multi-bitrate Icecast, Wrapped, Three.js visualizer (already in v2 per `CLAUDE.md`).
- Artist deduplication / alias handling — out of scope for the remediation cycle.
- Meilisearch integration — current ILIKE is fine until search latency becomes a user complaint.

---

## 8. Autoplan Review — CEO phase (2026-04-18)

Two independent reviewers (Claude subagent + Codex, high reasoning) evaluated §7 without seeing each other's output. Both arrived at the same reframing of the plan. This is not a minor critique — it is a structural challenge to the fix list itself.

### 8.1 CEO consensus table

| Dimension | Claude subagent | Codex | Consensus |
|---|---|---|---|
| 1. §7 frames the right problem? | No (medium) | No (high) | **DISAGREE with §7** — wrong frame. The real problem is duplicated ingestion, not deploy drift. |
| 2. §7.2 option set is complete? | No — buries 3rd option | No — avoids 3rd option | **DISAGREE with §7** — a third, simpler path exists and is omitted. |
| 3. Ordering #1 (cron) before #2 (arch decision) correct? | No — #2 should come first | No — strategic choice must precede hotfix | **DISAGREE with §7.** |
| 4. §7.5 "restart bridge" conclusion valid? | No — §5 says cause unproven | No — restart destroys diagnostic evidence | **DISAGREE with §7.** |
| 5. §7.6 podman/redis/meili priority correct? | N/A | No — cosmetic work | Codex flags as misprioritized. |
| 6. §7 covers critical ops concerns? | No — missing backup + freshness alerts | No — missing monitoring, backup, replay | **Both add missing items.** |

### 8.2 The third option both models surface

§7.2 today offers: (a) drop the Go Plex sync, or (b) extend `StageMapping` to hold multiple titles. Both reviewers independently arrived at **option (c)**: keep the Go bridge but remove its Plex client role entirely. Let the Python cron be the only Plex consumer; have Go read the sidecar `*.mp3.json` files and the `{artists,albums,playlists,tracks_index}.json` indexes that `plex_webradio_sync.py` already writes to `~/files/Webradio/`.

Why this wins:
- Python already resolved the hard problems: name matching, multi-playlist aggregation per stage, addedAt extraction, sidecar metadata. Re-implementing any of it in Go is duplication.
- The schema's cardinality is wrong anyway — `library_tracks` has one `stage_id` per `file_path` (`db/migrations/001_initial_schema.sql:55`), but `etage-0` and `fontanna-laputa` aggregate two Plex playlists each. "Last-synced wins" (`plex/sync.go:135` comment) is a known bug. Fixing this in the sync worker plus extending the mapping is more work than just reading JSON.
- Removes Plex token from the bridge env entirely. Less config, less surface area.
- Freshness lag drops from 5 min poll → filesystem-change trigger (seconds).

### 8.3 New findings Codex surfaced that §7 misses

| # | Finding | Evidence |
|---|---|---|
| A | `artists` table is never populated. Sync only upserts `library_tracks`; `api/artists.go:95, 148` and `enrichment/worker.go:75` all need rows that nothing creates. Even a "fixed" Plex sync leaves `/api/artists` empty. | `plex/sync.go:137-150` inserts into `library_tracks` only. |
| B | `track_plays.duration_sec` is never written. `main.go:315` in `logPlay` does not insert duration. Stats' `total_hours` aggregate stays stuck at zero. | `main.go:315`, `api/stats.go:39`. |
| C | Schema enforces one stage per file — conflicts with multi-playlist stages. | `db/migrations/001_initial_schema.sql:55`, `plex/sync.go:135` ("last-synced wins"). |
| D | `/health` always returns `"status":"ok"` with HTTP 200 even when dependencies are down. | `main.go:164` — top-level status hardcoded. |
| E | §7.8's proposed health gate "assert /api/stages returns 9 entries" is a fake check. That endpoint just iterates `cfg.VisibleStages()` — it returns 9 regardless of whether MPD, Plex, or DB is healthy. | `main.go:183-202`. |
| F | `@reboot` cron is not supervision. It only catches host reboot, not process crash. Real supervision needs a 1-minute restart loop. | `deploy.sh:72, 87` (nohup + &, no monitor). |
| G | Missing: freshness alerts (cron output age, track_plays growth rate), Postgres backup, replay/backfill from disk artifacts. | Nothing in repo addresses these. |

### 8.4 Recommended reordering (both models agree)

1. **Decide §7.2 first.** This is the one load-bearing architectural choice. All other §7 items are downstream.
2. **Fix cron typo** (if Python stays as ingest path) or **plan its decommission** (if Go reads disk).
3. **Add artist materialization** (gap A) — required for `/api/artists` to work at all.
4. **Fix `track_plays.duration_sec`** (gap B) and fix `/health` to report real status (gap D).
5. **Fix schema cardinality** for multi-playlist stages (gap C) or formalize "last-synced wins" if acceptable.
6. **Add freshness + backup monitoring** (gap G) before touching deploy.
7. **Investigate** why track_plays stalled, *then* decide whether restart is the fix.
8. **Then** harden deploy.sh (env file, respawn, health gate), align ports, clean podman state.

---

---

## 9. Autoplan Review — Eng phase (2026-04-18)

Two independent reviewers (Claude subagent + Codex, high reasoning) stress-tested the revised §7. Both found the direction sound but flagged the same set of blocking issues before any line of `apps/bridge/disk/` can be written.

### 9.1 Eng consensus table

| Dimension | Claude subagent | Codex | Consensus |
|---|---|---|---|
| 1. Phase ordering A-B-C-D-E sound? | ≈ (flagged rollout gap) | No — item 7 (schema) must precede item 5 (importer) | **DISAGREE with current §7.** Schema + path normalization precede importer build. |
| 2. Phase C atomicity model sufficient? | No — no manifest / atomic rename | No — mtime watch races sequential Python writes | **Both flag** — manifest file + `tempfile+os.replace()` + sha256 per artifact required. |
| 3. Schema migration has a plan? | No — pick (a), say so; library_tracks is empty so no backfill pain | No — DDL is easy; the hard part is rewriting `digging.go:17-24`, `artists.go:41-49`, `search.go:47-53` that all assume scalar `stage_id` | **Both flag** — pick join table (a), update all read sites. |
| 4. Path normalization addressed? | No — MPD `file` ≠ Python absolute path; `history.go:73-76` join is already broken | No — `MUSIC_BASE_PATH` exists in config but not wired | **Both flag** — canonicalize at *both* play-logging and import. |
| 5. Deletion semantics addressed? | No — orphaned tracks accumulate | No — current sync never deletes; `tracks_removed` always 0 | **Both flag** — explicit membership-vs-added semantics (soft-delete `deleted_at` or tombstone; keep added_at for /digging history). |
| 6. Security: JSON + filesystem trust boundary covered? | No — path traversal, symlink escape, memory DoS | No — local JSON poisoning, TOCTOU, symlink escape | **Both flag** — `io.LimitReader`, `filepath.EvalSymlinks` + root confinement, regex-validated `file_path`. |
| 7. Test plan sufficient for 15 items? | No — 12 of 15 items have no test strategy | No — existence-only / helper-level tests today | **Both flag** — need JSON+Postgres fixtures for items 3, 4, 5, 6, 7, 9; live-server-only for 1, 2, 8, 10, 12-15. |
| 8. Rollout / cutover safe? | No — Phase C #8 deletes plex/ before validation | No — no `SYNC_SOURCE` flag, no shadow mode | **Both flag** — feature-flag guarded shadow mode + row-count diff before deletion. |

### 9.2 Top blocking issues (fix before writing importer code)

1. **Atomicity contract on disk.** The Python sync must emit a `sync_manifest.json` last, containing `{generation_id, artifact_sha256_map, written_at}`. All four indexes + sidecars are written via `tempfile + os.replace()`. Go importer reads the manifest first, verifies sha256 of each artifact, and refuses partial generations. **This requires changing `~/files/Webradio/scripts/plex_webradio_sync.py` — not just Go.** Without this, the disk_sync is a data-corruption generator.

2. **Schema migration before importer.** `library_tracks` has 0 rows — do the cutover to `track_stages(file_path, stage_id)` join table now. Update all read sites (`digging.go`, `artists.go`, `search.go`, `stats.go`, `history.go`) in the same commit. The DDL is trivial; the API surface rewrite is the work.

3. **Path normalization.** Right now `track_plays.file_path` is MPD's `Song["file"]` (relative to MPD `music_directory`, e.g. `00_❤️ Tracks/01 - Cafius - Vertigo.mp3`). `library_tracks.file_path` will be the Python-written path (absolute, resolved through symlink to `~/files/plex_music_library/...`). **These will never join.** Normalize at play-log time: resolve MPD's relative path through the symlink to get the canonical absolute path, same as Python uses. Wire `MUSIC_BASE_PATH` into `main.go:315` logPlay.

4. **Deletion / membership semantics.** Keep `library_tracks.added_at` forever (powers /digging retrospective). Add `library_tracks.deleted_at` (nullable). After each successful import: `UPDATE library_tracks SET deleted_at = now() WHERE file_path NOT IN (<current manifest set>)`. Stage memberships live in `track_stages` and are deleted outright on removal — no historical "this track used to be in etage-0" needed.

5. **Feature-flag rollout.** Add `SYNC_SOURCE=plex|disk|shadow` env var. In `shadow` mode: run both, write disk results to a shadow table (or dry-run log), diff against Plex results. Only `podman-compose down` Plex path after 48h of clean diffs. Phase C item 8 (delete `apps/bridge/plex/`) is the LAST thing to land, not part of Phase C proper.

### 9.3 Revised phase ordering (replaces §7 ordering)

```
Phase A — Hotfix (1-line change)
  1. Fix cron typo

Phase B — Correctness (no new features; fix existing bugs)
  2. Investigate track_plays stall (diagnostic; decide if restart needed)
  3. Fix track_plays.duration_sec logging
  4. Fix /health to report real status

Phase C — Foundation (prerequisites for importer)
  5. Python: add sync_manifest.json + tempfile+os.replace() + sha256
  6. Go: migrate library_tracks stage_id → track_stages join table
     (update digging/artists/search/stats/history read sites in same PR)
  7. Go: normalize file_path in logPlay via MUSIC_BASE_PATH

Phase D — Importer (feature-flagged)
  8. Go: apps/bridge/disk/ package, behind SYNC_SOURCE=shadow by default
  9. Go: artist materialization from artists.json
  10. Go: deletion semantics (deleted_at + track_stages cleanup)

Phase E — Cutover (after 48h clean shadow diffs)
  11. Flip SYNC_SOURCE=disk
  12. Delete apps/bridge/plex/

Phase F — Ops hardening (parallel to C–E, any order)
  13. Freshness probes in /health
  14. pg_dump nightly with heartbeat + alerting
  15. Real watchdog (exponential backoff, not tight restart loop)
  16. Secrets in ~/.gaende.env sourced by deploy.sh
  17. Real end-to-end health gate in deploy.sh
  18. Align frontend port (13000 or 3000, pick one)
  19. podman system reset on gaende-postgres
```

### 9.4 Test plan per item

| # | Type | Fixtures needed | Notes |
|---|---|---|---|
| 1 | Manual | — | SSH + verify `sync_plex_playlist.log` stops showing "No such file" |
| 2 | Diagnostic | — | `/proc/$PID/fd`, `tcpdump`, bridge.log growth |
| 3 | Unit | Fake `NowPlaying{Duration: 180}` | Assert `track_plays.duration_sec = 180` |
| 4 | Table-driven unit | Fake DB, fake pool alive/dead, fake Plex | Assert `/health.status` varies: ok/degraded/down |
| 5 | Integration (Python + Go) | Fixture JSONs + sha256 manifest | Golden-file compare; corrupt-manifest rejection |
| 6 | Migration test | Empty `library_tracks` | Apply 002_track_stages.sql, assert new reads work |
| 7 | Unit | Mock MPD "file" field + `MUSIC_BASE_PATH` | Assert canonical path in `track_plays.file_path` |
| 8 | Integration | Real JSON artifacts (copied from Whatbox) | Shadow-mode dry-run diff vs live Plex |
| 9 | Unit | Mock `artists.json` | Assert `artists` table populated, FKs valid |
| 10 | Integration | Two fixture sets (v1, v2 with removal) | Assert `deleted_at` set, `track_stages` row gone |
| 11 | Manual | — | Live env flip |
| 12 | Grep check | — | CI fails if `apps/bridge/plex/` exists |
| 13 | Unit | Fake `time.Now` | Assert freshness math |
| 14 | Shell integration | — | `pg_dump` runs, heartbeat row inserted, restore verifies |
| 15 | Integration | Fake curl returning 503 | Assert watchdog backs off, doesn't loop-kill |
| 16-19 | Manual + health gate | — | E2E curl in deploy.sh asserts all green |

Total new tests: ~25. Current: 38. Target: 63.

Test plan artifact: this table is also saved to `~/.gstack/projects/Y0lan-pavoia-webradio-v2/main-autoplan-test-plan-20260418.md`.

### 9.5 Failure modes registry (combined)

| ID | Failure | Detection |
|---|---|---|
| F1 | Mixed-generation import (partial Python write) | Manifest sha256 verification before ingest |
| F2 | Sidecar-only updates missed (watching indexes only) | Sidecar mtime audit vs `tracks_index` manifest |
| F3 | Path canonical mismatch → zero joins | `track_plays`↔`library_tracks` match-rate metric |
| F4 | Playlist removals never propagate | Diff current manifest set vs DB per-sync |
| F5 | Multi-stage tracks collapse to one stage | Count files-in-multiple-playlists vs `track_stages` rows |
| F6 | Watchdog loop destroys evidence when DB down | PID churn counter, exponential backoff required |
| F7 | `pg_dump` silently fails | Heartbeat row; alert if stale > 25h |
| F8 | Advisory lock deadlock | `disk_sync_duration_seconds` p99 alert |
| F9 | Unicode NFC/NFD artist dupes (Björk vs Bjork) | CI test with mixed normalization fixtures |
| F10 | First-deploy missing JSON | Explicit wait-for-artifacts backoff, not crash-loop |
| F11 | `file_path` collision between Plex libraries | Log + count `ON CONFLICT` hits per run |
| F12 | `/health 503` + watchdog restart when DB is the actual problem | Backoff + alert on restart_count > 3/hour |

### 9.6 What's NOT in scope for this cycle

- Meilisearch integration (ILIKE is fine until latency becomes a complaint)
- Three.js / Wrapped / multi-bitrate (already v2 per CLAUDE.md)
- Artist alias deduplication beyond NFC/NFD normalization
- Migrating off Whatbox (separate decision; costly; raised in §10)

### 9.7 What already exists (avoid reinventing)

- Python sync already resolves name matching, multi-playlist aggregation, addedAt, sidecar metadata → `~/files/Webradio/scripts/plex_webradio_sync.py`
- `MUSIC_BASE_PATH` config field already present, just unwired → `config.go:37,55`
- `ON CONFLICT file_path DO UPDATE` semantics already in schema, reusable → `db/migrations/001_initial_schema.sql`
- Enrichment worker already stubbed and will wake up once `artists` rows exist → `enrichment/worker.go`
- `plex_playlist_snapshots` table exists and could be repurposed as `disk_sync_snapshots` for audit

