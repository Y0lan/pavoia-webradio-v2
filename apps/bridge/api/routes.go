package api

import (
	"net/http"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
	mpdpool "github.com/Y0lan/pavoia-webradio-v2/apps/bridge/mpd"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps holds shared dependencies for all API handlers.
type Deps struct {
	DB         *pgxpool.Pool
	Pool       *mpdpool.Pool
	Config     *config.Config
	AdminToken string
	MPDHost    string
	Stream     *StreamHandlers // exposed so main.go can read listener counts
}

// RegisterRoutes registers all REST API endpoints on the given mux.
func RegisterRoutes(mux *http.ServeMux, d Deps) {
	history := &HistoryHandlers{DB: d.DB}
	digging := &DiggingHandlers{DB: d.DB}
	stats := &StatsHandlers{DB: d.DB}
	artists := &ArtistsHandlers{DB: d.DB, AdminToken: d.AdminToken}
	search := &SearchHandlers{DB: d.DB}
	queue := &QueueHandlers{Pool: d.Pool, Config: d.Config}

	// RequireDB wraps handlers that need a database connection.
	rdb := func(h http.HandlerFunc) http.HandlerFunc {
		return RequireDB(d.DB, h)
	}

	// History
	mux.HandleFunc("GET /api/history", rdb(history.HandleHistory))
	mux.HandleFunc("GET /api/history/{id}", history.HandleHistoryByID) // validates ID first
	mux.HandleFunc("GET /api/history/calendar", rdb(history.HandleHistoryCalendar))
	mux.HandleFunc("GET /api/history/heatmap", rdb(history.HandleHistoryHeatmap))
	mux.HandleFunc("GET /api/stages/{id}/history", rdb(history.HandleStageHistory))

	// Digging
	mux.HandleFunc("GET /api/digging/calendar", rdb(digging.HandleDiggingCalendar))
	mux.HandleFunc("GET /api/digging/calendar/{date}", rdb(digging.HandleDiggingDate))
	mux.HandleFunc("GET /api/digging/streaks", rdb(digging.HandleDiggingStreaks))
	mux.HandleFunc("GET /api/digging/patterns", rdb(digging.HandleDiggingPatterns))

	// Stats
	mux.HandleFunc("GET /api/stats/overview", rdb(stats.HandleStatsOverview))
	mux.HandleFunc("GET /api/stats/top-artists", rdb(stats.HandleStatsTopArtists))
	mux.HandleFunc("GET /api/stats/top-tracks", rdb(stats.HandleStatsTopTracks))
	mux.HandleFunc("GET /api/stats/stages", rdb(stats.HandleStatsStages))
	mux.HandleFunc("GET /api/stats/bpm", rdb(stats.HandleStatsBPM))
	mux.HandleFunc("GET /api/stats/keys", rdb(stats.HandleStatsKeys))
	mux.HandleFunc("GET /api/stats/decades", rdb(stats.HandleStatsDecades))
	mux.HandleFunc("GET /api/stats/genres", rdb(stats.HandleStatsGenres))
	mux.HandleFunc("GET /api/stats/discovery-velocity", rdb(stats.HandleStatsDiscoveryVelocity))
	mux.HandleFunc("GET /api/stats/listening-heatmap", rdb(stats.HandleStatsListeningHeatmap))

	// Artists
	mux.HandleFunc("GET /api/artists", rdb(artists.HandleArtistsList))
	mux.HandleFunc("GET /api/artists/{id}", artists.HandleArtistDetail)   // validates ID first
	mux.HandleFunc("GET /api/artists/{id}/tracks", artists.HandleArtistTracks) // validates ID first
	mux.HandleFunc("GET /api/artists/{id}/similar", artists.HandleArtistSimilar) // validates ID first
	mux.HandleFunc("GET /api/search", search.HandleSearch) // validates q param first

	// Queue
	mux.HandleFunc("GET /api/stages/{id}/queue", queue.HandleQueue)

	// Audio stream proxy
	if d.Stream == nil && d.MPDHost != "" {
		d.Stream = NewStreamHandlers(d.Config, d.MPDHost)
	}
	if d.Stream != nil {
		mux.HandleFunc("GET /api/stream/{stageId}", d.Stream.HandleStream)
	}
}
