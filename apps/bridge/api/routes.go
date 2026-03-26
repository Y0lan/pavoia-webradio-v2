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

	// History
	mux.HandleFunc("GET /api/history", history.HandleHistory)
	mux.HandleFunc("GET /api/history/{id}", history.HandleHistoryByID)
	mux.HandleFunc("GET /api/history/calendar", history.HandleHistoryCalendar)
	mux.HandleFunc("GET /api/history/heatmap", history.HandleHistoryHeatmap)
	mux.HandleFunc("GET /api/stages/{id}/history", history.HandleStageHistory)

	// Digging
	mux.HandleFunc("GET /api/digging/calendar", digging.HandleDiggingCalendar)
	mux.HandleFunc("GET /api/digging/calendar/{date}", digging.HandleDiggingDate)
	mux.HandleFunc("GET /api/digging/streaks", digging.HandleDiggingStreaks)
	mux.HandleFunc("GET /api/digging/patterns", digging.HandleDiggingPatterns)

	// Stats
	mux.HandleFunc("GET /api/stats/overview", stats.HandleStatsOverview)
	mux.HandleFunc("GET /api/stats/top-artists", stats.HandleStatsTopArtists)
	mux.HandleFunc("GET /api/stats/top-tracks", stats.HandleStatsTopTracks)
	mux.HandleFunc("GET /api/stats/stages", stats.HandleStatsStages)
	mux.HandleFunc("GET /api/stats/bpm", stats.HandleStatsBPM)
	mux.HandleFunc("GET /api/stats/keys", stats.HandleStatsKeys)
	mux.HandleFunc("GET /api/stats/decades", stats.HandleStatsDecades)
	mux.HandleFunc("GET /api/stats/genres", stats.HandleStatsGenres)
	mux.HandleFunc("GET /api/stats/discovery-velocity", stats.HandleStatsDiscoveryVelocity)
	mux.HandleFunc("GET /api/stats/listening-heatmap", stats.HandleStatsListeningHeatmap)

	// Artists
	mux.HandleFunc("GET /api/artists", artists.HandleArtistsList)
	mux.HandleFunc("GET /api/artists/{id}", artists.HandleArtistDetail)
	mux.HandleFunc("GET /api/artists/{id}/tracks", artists.HandleArtistTracks)
	mux.HandleFunc("GET /api/artists/{id}/similar", artists.HandleArtistSimilar)

	// Search
	mux.HandleFunc("GET /api/search", search.HandleSearch)

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
