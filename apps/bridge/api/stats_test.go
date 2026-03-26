package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleStatsOverview_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/overview", nil)
	h.HandleStatsOverview(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsTopArtists_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/top-artists", nil)
	h.HandleStatsTopArtists(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsTopTracks_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/top-tracks", nil)
	h.HandleStatsTopTracks(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsBPM_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/bpm", nil)
	h.HandleStatsBPM(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsKeys_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/keys", nil)
	h.HandleStatsKeys(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsDecades_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/decades", nil)
	h.HandleStatsDecades(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsGenres_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/genres", nil)
	h.HandleStatsGenres(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsStages_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/stages", nil)
	h.HandleStatsStages(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStatsDiscoveryVelocity_NoDB(t *testing.T) {
	h := &StatsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stats/discovery-velocity", nil)
	h.HandleStatsDiscoveryVelocity(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
