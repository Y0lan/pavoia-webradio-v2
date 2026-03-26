package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// NoDB tests removed — RequireDB middleware now guards all handlers.

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
