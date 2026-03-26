package api

import (
	"testing"
)

// NoDB tests removed — RequireDB middleware now guards all stats handlers.
// Stats overview combined-query test requires a real or mock DB.

func TestStatsHandlersExist(t *testing.T) {
	// Verify all handler methods exist on StatsHandlers
	h := &StatsHandlers{DB: nil}
	_ = h.HandleStatsOverview
	_ = h.HandleStatsTopArtists
	_ = h.HandleStatsTopTracks
	_ = h.HandleStatsStages
	_ = h.HandleStatsBPM
	_ = h.HandleStatsKeys
	_ = h.HandleStatsDecades
	_ = h.HandleStatsGenres
	_ = h.HandleStatsDiscoveryVelocity
	_ = h.HandleStatsListeningHeatmap
}
