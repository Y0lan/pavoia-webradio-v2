package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	mux := http.NewServeMux()

	// Health endpoint — returns status of all dependencies
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		health := map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
			"stages": len(cfg.VisibleStages()),
			"checks": map[string]string{
				"postgres":    "not_connected",
				"redis":       "not_connected",
				"meilisearch": "not_connected",
				"mpd":         "not_connected",
				"plex":        "not_connected",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	// Stages list — returns visible stage configs
	mux.HandleFunc("GET /api/stages", func(w http.ResponseWriter, r *http.Request) {
		stages := cfg.VisibleStages()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stages)
	})

	// CORS middleware for frontend
	handler := corsMiddleware(mux)

	slog.Info("bridge starting", "addr", cfg.Addr(), "stages", len(cfg.VisibleStages()))
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
