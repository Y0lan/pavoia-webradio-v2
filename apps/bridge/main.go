package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
	mpdpool "github.com/Y0lan/pavoia-webradio-v2/apps/bridge/mpd"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// MPD connection pool
	mpdHost := envOr("MPD_HOST", "localhost")
	pool := mpdpool.NewPool(cfg.VisibleStages(), func(np mpdpool.NowPlaying) {
		title := np.Song["Title"]
		artist := np.Song["Artist"]
		slog.Info("track changed", "stage", np.StageID, "artist", artist, "title", title)
	})

	connected := pool.ConnectAll(mpdHost)
	slog.Info("mpd pool ready", "connected", connected, "total", len(cfg.VisibleStages()))

	// Start idle watchers for track change notifications
	pool.StartWatchers(mpdHost)

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		mpdStatus := "degraded"
		aliveCount := 0
		for _, s := range cfg.VisibleStages() {
			if pool.IsAlive(s.ID) {
				aliveCount++
			}
		}
		if aliveCount == len(cfg.VisibleStages()) {
			mpdStatus = "ok"
		} else if aliveCount > 0 {
			mpdStatus = fmt.Sprintf("partial (%d/%d)", aliveCount, len(cfg.VisibleStages()))
		} else {
			mpdStatus = "down"
		}

		health := map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
			"stages": len(cfg.VisibleStages()),
			"checks": map[string]string{
				"mpd":         mpdStatus,
				"postgres":    "not_connected",
				"redis":       "not_connected",
				"meilisearch": "not_connected",
				"plex":        "not_connected",
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

	slog.Info("bridge starting", "addr", cfg.Addr(), "stages", len(cfg.VisibleStages()), "mpd_connected", connected)
	if err := http.ListenAndServe(cfg.Addr(), handler); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
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
