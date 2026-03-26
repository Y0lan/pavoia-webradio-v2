package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleArtistsList_NoDB(t *testing.T) {
	h := &ArtistsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/artists", nil)
	h.HandleArtistsList(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleArtistDetail_NoDB(t *testing.T) {
	h := &ArtistsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/artists/1", nil)
	r.SetPathValue("id", "1")
	h.HandleArtistDetail(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleArtistDetail_InvalidID(t *testing.T) {
	h := &ArtistsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/artists/abc", nil)
	r.SetPathValue("id", "abc")
	h.HandleArtistDetail(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ID, got %d", w.Code)
	}
}

func TestHandleArtistTracks_NoDB(t *testing.T) {
	h := &ArtistsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/artists/1/tracks", nil)
	r.SetPathValue("id", "1")
	h.HandleArtistTracks(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleArtistSimilar_NoDB(t *testing.T) {
	h := &ArtistsHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/artists/1/similar", nil)
	r.SetPathValue("id", "1")
	h.HandleArtistSimilar(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
