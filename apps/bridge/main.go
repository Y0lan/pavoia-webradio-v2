package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/api"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/db"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/disk"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/enrichment"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/hub"
	mpdpool "github.com/Y0lan/pavoia-webradio-v2/apps/bridge/mpd"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Play logging work queue (bounded, prevents goroutine accumulation)
	var playWg sync.WaitGroup
	playCh := make(chan mpdpool.NowPlaying, 64)

	// Database
	var database *db.DB
	if cfg.DatabaseURL != "" {
		var err error
		database, err = db.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Warn("database not available — running without persistence", "error", err)
		} else {
			slog.Info("database connected")
			if err := database.Migrate(ctx); err != nil {
				slog.Error("migration failed", "error", err)
				os.Exit(1)
			}

			// Start play logger worker — uses its own context so shutdown drain works
			playCtx, playCancel := context.WithCancel(context.Background())
			musicBase := cfg.MusicBasePath
			playWg.Add(1)
			go func() {
				defer playWg.Done()
				for np := range playCh {
					logPlay(playCtx, database, np, musicBase)
				}
			}()
			defer playCancel()
		}
	}

	// WebSocket + SSE hub — pass valid stage IDs for subscription validation
	stageIDs := make([]string, len(cfg.VisibleStages()))
	for i, s := range cfg.VisibleStages() {
		stageIDs[i] = s.ID
	}
	wsHub := hub.New(stageIDs...)

	// MPD connection pool
	pool := mpdpool.NewPool(cfg.VisibleStages(), func(np mpdpool.NowPlaying) {
		title := np.Song["Title"]
		artist := np.Song["Artist"]
		slog.Info("track changed", "stage", np.StageID, "artist", artist, "title", title)

		// Broadcast to WebSocket clients
		wsHub.BroadcastNowPlaying(hub.NowPlayingEvent{
			StageID:  np.StageID,
			Status:   np.Status,
			Title:    np.Song["Title"],
			Artist:   np.Song["Artist"],
			Album:    np.Song["Album"],
			Elapsed:  np.Elapsed,
			Duration: np.Duration,
			File:     np.Song["file"],
		})

		// Send to play logger (non-blocking — drop if channel is full)
		if database != nil {
			select {
			case playCh <- np:
			default:
				slog.Warn("play log queue full, dropping event", "stage", np.StageID)
			}
		}
	})

	// Echo the resolved stage → port mapping once at boot so config drift between
	// the bridge's StageConfig and the MPD config files on disk surfaces in logs
	// instead of only through /api/stages returning mysteriously-wrong content.
	// Caught the 2026-04-19 6601↔6602 + 6603↔6604 swap; future drift will be
	// visible the moment the bridge restarts.
	for _, s := range cfg.VisibleStages() {
		slog.Info("stage port mapping",
			"stage", s.ID,
			"genre", s.Genre,
			"mpd_port", s.MPDPort,
			"stream_port", s.StreamPort,
		)
	}

	connected := pool.ConnectAll(cfg.MPDHost)
	slog.Info("mpd pool ready", "connected", connected, "total", len(cfg.VisibleStages()))
	pool.StartWatchers(ctx, cfg.MPDHost)

	// Disk importer — the authoritative library-metadata ingest path as of Phase D.
	// Reads JSON artifacts (sync_manifest.json + artists/playlists/albums/tracks_index
	// + per-track sidecars) that scripts/plex-sync/ writes atomically to MUSIC_BASE_PATH,
	// verifies sha256 against the manifest, upserts library_tracks + track_stages +
	// artists, and soft-deletes rows that are no longer on disk.
	//
	// The bridge is no longer a Plex API client — all Plex auth lives in the Python
	// sync. PLEX_URL / PLEX_TOKEN env vars are accepted for backward compatibility
	// with pre-Phase-E .gaende.env files but have no effect.
	var diskImporter *disk.Importer
	if database != nil && cfg.MusicBasePath != "" {
		diskImporter = disk.NewImporter(database.Pool, cfg, cfg.MusicBasePath)
		diskImporter.Start(ctx, 2*time.Minute)
		slog.Info("disk importer started", "webradio", cfg.MusicBasePath, "interval", "2m")
	} else {
		slog.Info("disk importer not configured — skipping (need DATABASE_URL + MUSIC_BASE_PATH)")
	}

	// Artist enrichment worker (Last.fm + MusicBrainz)
	var enrichWorker *enrichment.Worker
	if cfg.LastFMKey != "" && database != nil {
		enrichWorker = enrichment.NewWorker(database.Pool, cfg.LastFMKey, 30*time.Minute)
		enrichWorker.Start(ctx)
		slog.Info("enrichment worker started", "interval", "30m")
	} else {
		slog.Info("enrichment not configured — skipping (need LASTFM_API_KEY + database)")
	}

	// Cache visible stages (config never changes at runtime)
	visibleStages := cfg.VisibleStages()

	mux := http.NewServeMux()

	// Health endpoint — uses short timeouts for external checks
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		healthCtx, healthCancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer healthCancel()

		mpdStatus := mpdHealthStatus(pool, visibleStages)
		dbStatus := "not_configured"
		if database != nil {
			if database.Healthy(healthCtx) {
				dbStatus = "ok"
			} else {
				dbStatus = "down"
			}
		}
		checks := map[string]string{
			"mpd":         mpdStatus,
			"postgres":    dbStatus,
			"redis":       "not_connected",
			"meilisearch": "not_connected",
			// Plex check removed in Phase E — the bridge no longer authenticates
			// against Plex (Python sync is the only consumer). Kept in the response
			// shape as "not_used" so downstream dashboards see an explicit signal
			// rather than a silently-disappearing key.
			"plex":        "not_used",
			// Disk importer freshness — flags if the library_tracks ingest has
			// stopped converging, which /health previously couldn't see.
			"disk_sync": diskSyncStatus(diskImporter),
			// pg_backup heartbeat — last successful row in pg_backup_log. "stale"
			// if no ok row in the last 30 hours, which an external monitor can
			// alert on when the nightly cron has silently stopped.
			"pg_backup": pgBackupStatus(healthCtx, database),
		}
		writeHealthResponse(w, checks, len(visibleStages))
	})

	// Stages list with now-playing
	mux.HandleFunc("GET /api/stages", func(w http.ResponseWriter, r *http.Request) {
		type stageResponse struct {
			config.StageConfig
			NowPlaying mpdpool.NowPlaying `json:"now_playing"`
			Alive      bool               `json:"alive"`
		}

		result := make([]stageResponse, 0, len(visibleStages))
		for _, s := range visibleStages {
			result = append(result, stageResponse{
				StageConfig: s,
				NowPlaying:  pool.NowPlaying(s.ID),
				Alive:       pool.IsAlive(s.ID),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			slog.Debug("stages response write failed", "error", err)
		}
	})

	// Single stage now-playing
	mux.HandleFunc("GET /api/stages/{id}/now-playing", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if cfg.StageByID(id) == nil {
			http.Error(w, `{"error":"stage not found"}`, http.StatusNotFound)
			return
		}
		np := pool.NowPlaying(id)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(np); err != nil {
			slog.Debug("now-playing response write failed", "error", err)
		}
	})

	// REST API endpoints (history, digging, stats, artists, search, queue)
	var dbPool *pgxpool.Pool
	if database != nil {
		dbPool = database.Pool
	}
	apiDeps := api.Deps{
		DB:         dbPool,
		Pool:       pool,
		Config:     cfg,
		AdminToken: cfg.AdminToken,
		MPDHost:    cfg.MPDHost,
	}
	api.RegisterRoutes(mux, apiDeps)

	// Admin: force artist enrichment
	if enrichWorker != nil {
		mux.HandleFunc("POST /api/artists/{id}/enrich", api.AdminAuth(cfg.AdminToken, func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			var artistID int64
			if _, err := fmt.Sscanf(id, "%d", &artistID); err != nil {
				api.WriteError(w, http.StatusBadRequest, "invalid artist id")
				return
			}
			if err := enrichWorker.EnrichArtist(r.Context(), artistID); err != nil {
				api.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			api.WriteJSON(w, http.StatusOK, map[string]string{"status": "enriched"})
		}))
	}

	// Admin: verify manifest sidecar aggregate.
	//
	// Expensive — walks + hashes every *.mp3.json/.flac.json (~7000 files,
	// ~2s wall-clock on Whatbox). Not called from the hot importer tick,
	// hence wired to an admin endpoint so ops can manually spot-check for
	// filesystem drift between Python-written manifests and actual disk state.
	//
	// Response codes:
	//   200 status=verified            — hashes match
	//   409 status=drift               — count or aggregate mismatch
	//   422 status=manifest_too_old    — manifest predates sidecars field
	//   500 status=probe_error         — load manifest failed / walk errored
	if diskImporter != nil {
		mux.HandleFunc("POST /api/admin/verify-sidecars", api.AdminAuth(cfg.AdminToken, func(w http.ResponseWriter, r *http.Request) {
			manifest, err := disk.LoadManifest(cfg.MusicBasePath)
			if err != nil {
				api.WriteJSON(w, http.StatusInternalServerError, map[string]any{
					"status": "probe_error",
					"error":  fmt.Sprintf("load manifest: %v", err),
				})
				return
			}
			if vErr := disk.VerifySidecars(cfg.MusicBasePath, manifest); vErr != nil {
				switch {
				case errors.Is(vErr, disk.ErrSidecarsFieldMissing):
					api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
						"status":        "manifest_too_old",
						"generation_id": manifest.GenerationID,
						"error":         vErr.Error(),
					})
				case errors.Is(vErr, disk.ErrSidecarDrift):
					api.WriteJSON(w, http.StatusConflict, map[string]any{
						"status":        "drift",
						"generation_id": manifest.GenerationID,
						"error":         vErr.Error(),
					})
				default:
					// Walk / IO / anything else — genuine probe failure.
					api.WriteJSON(w, http.StatusInternalServerError, map[string]any{
						"status":        "probe_error",
						"generation_id": manifest.GenerationID,
						"error":         vErr.Error(),
					})
				}
				return
			}
			api.WriteJSON(w, http.StatusOK, map[string]any{
				"status":        "verified",
				"generation_id": manifest.GenerationID,
				"count":         manifest.Sidecars.Count,
			})
		}))
	}

	// WebSocket endpoint — per-stage now-playing broadcasts
	mux.HandleFunc("GET /ws", wsHub.HandleWS)

	// SSE endpoint — global broadcasts (listener counts, sync updates)
	mux.HandleFunc("GET /api/events", wsHub.HandleSSE)

	// Listener count broadcaster — sends counts to SSE clients every 10s
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				counts := wsHub.ListenerCounts()
				wsHub.BroadcastSSE(hub.SSEEvent{
					Event: "listeners",
					Data:  counts,
				})
			}
		}
	}()

	handler := corsMiddleware(mux)

	server := &http.Server{
		Addr:    cfg.Addr(),
		Handler: handler,
	}

	// Graceful shutdown — blocks main() until drain is complete
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down")

		cancel() // Cancel context — stops watchers, sync worker

		// Shut down HTTP server with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)

		// Wait for play logger to drain
		close(playCh)
		playWg.Wait()

		pool.Close()
	}()

	slog.Info("bridge starting", "addr", cfg.Addr(), "stages", len(visibleStages), "mpd_connected", connected)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown goroutine to finish draining before main() exits
	<-shutdownDone
	slog.Info("bridge stopped")
}

func logPlay(ctx context.Context, database *db.DB, np mpdpool.NowPlaying, musicBasePath string) {
	filePath := canonicalFilePath(np.Song["file"], musicBasePath)

	// Best-effort genre snapshot (migration 006). Semantics:
	//
	//   * The lookup runs in its own 500ms-timeout context so a flapping DB
	//     doesn't stall the single-goroutine play-logger queue (which would
	//     back-pressure into the 64-slot playCh and drop events). The 500ms
	//     bound covers pool-acquire + query together; under pool saturation
	//     (MaxConns=10) a play can still wait the full 500ms before falling
	//     back to genre=NULL. That's the intended ceiling — missing one
	//     snapshot is strictly better than dropping a play event.
	//   * "Not found" (track not yet in library_tracks) and "NULL value"
	//     (column unset) both yield genre=NULL on the play row — correctly
	//     "unknown" rather than a guess.
	//   * A transient DB error beyond ErrNoRows is logged at warn, not swallowed.
	//   * Soft-deleted library_tracks rows still yield a valid snapshot — a
	//     track removed the day after it played should still carry its genre
	//     in history.
	//   * Stale-read window: library_tracks.genre is written by the disk
	//     importer inside a per-playlist transaction that can cover 100s of
	//     tracks (see disk/importer.go:syncPlaylistFolder). If a retag for
	//     this file_path is sitting uncommitted during that walk, our read
	//     sees the pre-retag value. The snapshot is "genre as-of last-
	//     committed generation," not "as-of this exact instant." Acceptable
	//     because the only remaining ambiguity is "did the sync 30 seconds
	//     ago see the retag?" — which is always going to be Python-timing-
	//     dependent regardless.
	var genre *string
	if filePath != "" {
		lookupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		var g *string
		err := database.Pool.QueryRow(lookupCtx,
			`SELECT genre FROM library_tracks WHERE file_path = $1`, filePath,
		).Scan(&g)
		cancel()
		switch {
		case err == nil:
			if g != nil && *g != "" {
				genre = g
			}
		case errors.Is(err, pgx.ErrNoRows):
			// Track not yet ingested; leave genre NULL.
		default:
			slog.Warn("logPlay: genre snapshot lookup failed", "stage", np.StageID, "file", filePath, "error", err)
			// Fall through to INSERT with genre=NULL — losing the snapshot is
			// strictly better than losing the play event.
		}
	}

	_, err := database.Pool.Exec(ctx, `
		INSERT INTO track_plays (stage_id, artist, title, album, file_path, duration_sec, genre, played_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
	`,
		np.StageID,
		np.Song["Artist"],
		np.Song["Title"],
		np.Song["Album"],
		filePath,
		parseDurationSec(np.Duration),
		genre,
	)
	if err != nil {
		slog.Warn("failed to log play", "stage", np.StageID, "error", err)
	}
}

// parseDurationSec converts MPD's status["duration"] (fractional seconds as a string)
// into the nullable integer stored in track_plays.duration_sec. Returns nil on anything
// unparseable or non-positive so Postgres stores NULL, not a fake zero.
func parseDurationSec(s string) any {
	d, err := strconv.ParseFloat(s, 64)
	if err != nil || d < 1 {
		return nil
	}
	// Round-to-nearest instead of truncation so SUM(duration_sec) aggregates
	// aren't systematically biased low on every fractional duration.
	return int(math.Round(d))
}

// canonicalFilePath turns MPD's relative file (e.g. "00_❤️ Tracks/01 - Cafius - Vertigo.mp3")
// into the Webradio-level path ("{musicBasePath}/❤️ Tracks/01 - Cafius - Vertigo.mp3") so
// track_plays.file_path and a future library_tracks importer can join on the same key.
//
// MPD's music_directory contains symlinks named "NN_<PlaylistName>" pointing at
// ~/files/Webradio/<PlaylistName>/. We strip that NN_ prefix and re-root under
// musicBasePath. If musicBasePath is empty (config not set), return mpdFile unchanged —
// the join won't work but logs keep flowing.
func canonicalFilePath(mpdFile, musicBasePath string) string {
	if mpdFile == "" || musicBasePath == "" {
		return mpdFile
	}
	i := strings.IndexByte(mpdFile, '/')
	if i < 0 {
		return mpdFile
	}
	prefix, rest := mpdFile[:i], mpdFile[i+1:]
	playlist := stripNNPrefix(prefix)
	return path.Join(musicBasePath, playlist, rest)
}

// stripNNPrefix removes a leading "NN_" (two digits + underscore) if present.
// Works on the first path component only.
func stripNNPrefix(s string) string {
	if len(s) < 4 || s[2] != '_' {
		return s
	}
	if s[0] < '0' || s[0] > '9' || s[1] < '0' || s[1] > '9' {
		return s
	}
	return s[3:]
}

// writeHealthResponse is the I/O-free half of the /health handler, extracted so
// httptest can exercise status code + body shape without spinning a DB/MPD/Plex.
// Status code rule: 503 only for "down" (critical failure — bridge can't serve).
// 200 for "ok" and "degraded" so a watchdog doesn't restart the bridge when the
// thing that's broken is a non-critical dependency (Plex blip, partial MPD).
func writeHealthResponse(w http.ResponseWriter, checks map[string]string, stageCount int) {
	overall := overallHealth(checks)
	health := map[string]any{
		"status": overall,
		"time":   time.Now().UTC().Format(time.RFC3339),
		"stages": stageCount,
		"checks": checks,
	}
	w.Header().Set("Content-Type", "application/json")
	if overall == "down" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(health); err != nil {
		slog.Debug("health response write failed", "error", err)
	}
}

// pgBackupStatus reports the freshness AND recent failure state of the nightly
// pg-backup cron. We care about both signals: "latest ok is fresh" alone would
// hide three consecutive failed nights until the 30h window expired.
//
//   - "not_configured" if the database isn't connected
//   - "probe_error"    the DB query itself failed — surface separately from
//                      never_ran so ops sees "monitoring is broken" distinct
//                      from "monitoring says nothing ran"
//   - "never_ran"      table exists but has no rows yet
//   - "failing"        the most recent failed row is AFTER the most recent ok
//                      AND within pgBackupFailureWindow — cron tried and
//                      failed recently, surface immediately
//   - "stale"          last ok row older than pgBackupStaleAfter — watchdog
//                      should alert (backups silently stopped)
//   - "ok"             latest row is 'ok' and within the stale window
const (
	pgBackupStaleAfter    = 30 * time.Hour
	pgBackupFailureWindow = 30 * time.Hour
)

func pgBackupStatus(ctx context.Context, database *db.DB) string {
	if database == nil {
		return "not_configured"
	}
	queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Decision table for /health.pg_backup. Uses pgBackupStaleAfter (30h) as
	// both the "ok window" and the "still-a-recent-failure" window so a slow
	// or slightly-delayed nightly cron doesn't flip to stale prematurely.
	//
	//   1. lastOK within 30h                        → "ok"
	//   2. lastFailed AFTER lastOK AND within 30h   → "failing" (recent
	//                                                 convergence failure)
	//   3. any row exists (lastOK or lastFailed)    → "stale"
	//   4. no rows at all                           → "never_ran"
	//
	// Rule 2 requires the failure to be newer than the ok — a failure from T-29h
	// with a success at T-25h is a recovered state, not an active failure.
	var lastOKAt, lastFailedAt *time.Time
	err := database.Pool.QueryRow(queryCtx, `
		SELECT
			(SELECT MAX(written_at) FROM pg_backup_log WHERE status = 'ok'),
			(SELECT MAX(written_at) FROM pg_backup_log WHERE status = 'failed')
	`).Scan(&lastOKAt, &lastFailedAt)
	if err != nil {
		slog.Warn("pg_backup probe failed", "error", err)
		return "probe_error"
	}
	return pgBackupStatusFrom(lastOKAt, lastFailedAt, time.Now())
}

// pgBackupStatusFrom is the pure-function decision table extracted from
// pgBackupStatus so unit tests can exercise every branch without a live DB.
// `now` is passed explicitly so tests can simulate arbitrary timelines.
//
//   - never_ran: both nil (table empty)
//   - ok:        lastOKAt within pgBackupStaleAfter of now
//   - failing:   lastFailedAt AFTER lastOKAt AND within pgBackupFailureWindow
//   - stale:     any row exists but nothing is fresh (silent-stopped backups
//                OR only-failures that have aged past the failure window)
func pgBackupStatusFrom(lastOKAt, lastFailedAt *time.Time, now time.Time) string {
	if lastOKAt == nil && lastFailedAt == nil {
		return "never_ran"
	}

	// 1. Recent successful backup dominates — a manual-admin failure after a
	//    clean nightly run should not poison /health.
	if lastOKAt != nil && now.Sub(*lastOKAt) <= pgBackupStaleAfter {
		return "ok"
	}

	// 2. Recent failure that's strictly NEWER than the last ok (if any).
	if lastFailedAt != nil &&
		now.Sub(*lastFailedAt) <= pgBackupFailureWindow &&
		(lastOKAt == nil || lastFailedAt.After(*lastOKAt)) {
		return "failing"
	}

	// 3. Any row exists but nothing is fresh — backups silently stopped.
	return "stale"
}

// diskSyncStatus returns the /health check value for the disk importer.
//   - "not_configured" if the importer wasn't wired (missing DATABASE_URL or MUSIC_BASE_PATH)
//   - "never_ran"      if wired but no SyncOnce has completed yet (cold start)
//   - "stale"          if last success was more than diskSyncStaleAfter ago
//     (default 6 minutes = 3x the 2-minute poll interval, so a single skipped
//     run doesn't trip the alarm but two in a row does)
//   - "ok"             otherwise
//
// Post-Phase-E, this is the only signal that surfaces "library ingest stopped
// working" — before, /health would happily say status:ok even if the importer
// died at startup and the DB hadn't seen a new track in days.
const diskSyncStaleAfter = 6 * time.Minute

func diskSyncStatus(im *disk.Importer) string {
	if im == nil {
		return "not_configured"
	}
	last := im.LastSuccess()
	if last.IsZero() {
		return "never_ran"
	}
	if time.Since(last) > diskSyncStaleAfter {
		return "stale"
	}
	return "ok"
}

// overallHealth collapses the per-check map into a single verdict.
//   - "down"     if any CRITICAL check is "down" (postgres, or mpd entirely down).
//   - "degraded" if any check is non-ok (partial MPD, plex down, etc.) but nothing critical failed.
//   - "ok"       only when every check reports "ok", "not_connected", or "not_configured".
//
// Non-critical checks: redis + meilisearch (never used today), plex (advisory — Python cron
// is the real ingest path).
func overallHealth(checks map[string]string) string {
	critical := map[string]bool{"postgres": true, "mpd": true}

	degraded := false
	for name, status := range checks {
		// "not_used" is explicit "wired into the response shape for downstream
		// monitoring, but this bridge doesn't depend on the dep" — post-Phase-E
		// Plex is the only one, but future deprecations can reuse the same token.
		// "never_ran" is cold-start — flags as degraded so it surfaces.
		if status == "ok" || status == "not_connected" || status == "not_configured" || status == "not_used" {
			continue
		}
		if critical[name] && status == "down" {
			return "down"
		}
		degraded = true
	}
	if degraded {
		return "degraded"
	}
	return "ok"
}

func mpdHealthStatus(pool *mpdpool.Pool, stages []config.StageConfig) string {
	// Use HasRecentActivity (not IsAlive) so /health counts a stage as up when
	// its watcher socket is producing events, even if the main client briefly
	// looks dead to HTTP probes (MPD 60s server connection_timeout, etc.).
	// /api/stages stays on the stricter IsAlive for per-stage "queryable now".
	aliveCount := 0
	for _, s := range stages {
		if pool.HasRecentActivity(s.ID) {
			aliveCount++
		}
	}
	if aliveCount == len(stages) {
		return "ok"
	} else if aliveCount > 0 {
		return fmt.Sprintf("partial (%d/%d)", aliveCount, len(stages))
	}
	return "down"
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
