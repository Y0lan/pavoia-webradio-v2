package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/api"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/db"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/enrichment"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/hub"
	mpdpool "github.com/Y0lan/pavoia-webradio-v2/apps/bridge/mpd"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/plex"
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
			playWg.Add(1)
			go func() {
				defer playWg.Done()
				for np := range playCh {
					logPlay(playCtx, database, np)
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

	// Plex sync worker
	var plexClient *plex.Client
	if cfg.PlexToken != "" && cfg.PlexURL != "" {
		plexClient = plex.NewClient(cfg.PlexURL, cfg.PlexToken)
		if plexClient.Healthy() {
			slog.Info("plex connected", "url", cfg.PlexURL)

			if database != nil {
				mappings := buildPlexMappings(cfg)
				syncWorker := plex.NewSyncWorker(plexClient, database.Pool, mappings, 5*time.Minute)
				syncWorker.Start(ctx)
				slog.Info("plex sync worker started", "interval", "5m", "playlists", len(mappings))
			}
		} else {
			slog.Warn("plex not reachable", "url", cfg.PlexURL)
		}
	} else {
		slog.Info("plex not configured — skipping sync")
	}

	// Artist enrichment worker (Last.fm + MusicBrainz)
	var enrichWorker *enrichment.Worker
	if cfg.LastFMKey != "" && database != nil {
		enrichWorker = enrichment.NewWorker(database.Pool, cfg.LastFMKey, 30*time.Minute)
		enrichWorker.Start(ctx)
		slog.Info("enrichment worker started", "interval", "30m")
	} else {
		slog.Info("enrichment not configured — skipping (need LASTFM_KEY + database)")
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
		plexStatus := "not_configured"
		if plexClient != nil {
			if plexClient.HealthyWithTimeout(3 * time.Second) {
				plexStatus = "ok"
			} else {
				plexStatus = "down"
			}
		}

		health := map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
			"stages": len(visibleStages),
			"checks": map[string]string{
				"mpd":         mpdStatus,
				"postgres":    dbStatus,
				"redis":       "not_connected",
				"meilisearch": "not_connected",
				"plex":        plexStatus,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(health); err != nil {
			slog.Debug("health response write failed", "error", err)
		}
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

func logPlay(ctx context.Context, database *db.DB, np mpdpool.NowPlaying) {
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO track_plays (stage_id, artist, title, album, file_path, played_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, np.StageID, np.Song["Artist"], np.Song["Title"], np.Song["Album"], np.Song["file"])
	if err != nil {
		slog.Warn("failed to log play", "stage", np.StageID, "error", err)
	}
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

func buildPlexMappings(cfg *config.Config) []plex.StageMapping {
	var mappings []plex.StageMapping
	for _, s := range cfg.VisibleStages() {
		mappings = append(mappings, plex.StageMapping{
			PlaylistName: s.ID,
			StageID:      s.ID,
		})
	}
	return mappings
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
