package main

import (
	"context"
	"encoding/json"
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
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO track_plays (stage_id, artist, title, album, file_path, duration_sec, played_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
	`, np.StageID, np.Song["Artist"], np.Song["Title"], np.Song["Album"], filePath, parseDurationSec(np.Duration))
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
//   - "never_ran"      table exists but has no ok rows yet
//   - "failing"        the most recent row overall is status='failed' AND was
//                      written within pgBackupFailureWindow — the cron tried
//                      and failed recently, surface immediately
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

	// Two signals: last ok + last failed. A fresh ok that happened AFTER the
	// last failed clears the flag — a manual admin run that failed shouldn't
	// poison /health if the nightly cron then ran cleanly. "failing" only
	// when the latest failure is more recent than the latest ok AND within
	// the failure window.
	var lastOKAt, lastFailedAt *time.Time
	err := database.Pool.QueryRow(queryCtx, `
		SELECT
			(SELECT MAX(written_at) FROM pg_backup_log WHERE status = 'ok'),
			(SELECT MAX(written_at) FROM pg_backup_log WHERE status = 'failed')
	`).Scan(&lastOKAt, &lastFailedAt)
	if err != nil {
		// Table missing (migration hasn't run) or DB query failed.
		return "never_ran"
	}
	if lastOKAt == nil && lastFailedAt == nil {
		return "never_ran"
	}
	// "failing" if failure is the most-recent event AND recent.
	if lastFailedAt != nil &&
		(lastOKAt == nil || lastFailedAt.After(*lastOKAt)) &&
		time.Since(*lastFailedAt) <= pgBackupFailureWindow {
		return "failing"
	}
	if lastOKAt == nil {
		return "never_ran"
	}
	if time.Since(*lastOKAt) > pgBackupStaleAfter {
		return "stale"
	}
	return "ok"
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
	aliveCount := 0
	for _, s := range stages {
		if pool.IsAlive(s.ID) {
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
