package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/db"
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
			defer database.Close()
		}
	}

	// MPD connection pool
	mpdHost := envOr("MPD_HOST", "localhost")
	pool := mpdpool.NewPool(cfg.VisibleStages(), func(np mpdpool.NowPlaying) {
		title := np.Song["Title"]
		artist := np.Song["Artist"]
		slog.Info("track changed", "stage", np.StageID, "artist", artist, "title", title)

		// Log play to database
		if database != nil {
			go logPlay(ctx, database, np)
		}
	})

	connected := pool.ConnectAll(mpdHost)
	slog.Info("mpd pool ready", "connected", connected, "total", len(cfg.VisibleStages()))
	pool.StartWatchers(mpdHost)

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

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		mpdStatus := mpdHealthStatus(pool, cfg)
		dbStatus := "not_configured"
		if database != nil {
			if database.Healthy(r.Context()) {
				dbStatus = "ok"
			} else {
				dbStatus = "down"
			}
		}
		plexStatus := "not_configured"
		if plexClient != nil {
			if plexClient.Healthy() {
				plexStatus = "ok"
			} else {
				plexStatus = "down"
			}
		}

		health := map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
			"stages": len(cfg.VisibleStages()),
			"checks": map[string]string{
				"mpd":         mpdStatus,
				"postgres":    dbStatus,
				"redis":       "not_connected",
				"meilisearch": "not_connected",
				"plex":        plexStatus,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	// Stages list with now-playing
	mux.HandleFunc("GET /api/stages", func(w http.ResponseWriter, r *http.Request) {
		type stageResponse struct {
			config.StageConfig
			NowPlaying mpdpool.NowPlaying `json:"now_playing"`
			Alive      bool               `json:"alive"`
		}

		stages := cfg.VisibleStages()
		result := make([]stageResponse, 0, len(stages))
		for _, s := range stages {
			result = append(result, stageResponse{
				StageConfig: s,
				NowPlaying:  pool.NowPlaying(s.ID),
				Alive:       pool.IsAlive(s.ID),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
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
		json.NewEncoder(w).Encode(np)
	})

	handler := corsMiddleware(mux)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down")
		cancel()
		pool.Close()
		if database != nil {
			database.Close()
		}
		os.Exit(0)
	}()

	slog.Info("bridge starting", "addr", cfg.Addr(), "stages", len(cfg.VisibleStages()), "mpd_connected", connected)
	if err := http.ListenAndServe(cfg.Addr(), handler); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
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

func mpdHealthStatus(pool *mpdpool.Pool, cfg *config.Config) string {
	aliveCount := 0
	stages := cfg.VisibleStages()
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
	// Map Plex playlist names to stage IDs
	// The playlist names in Plex match the stage IDs (which are the MPD instance names)
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
