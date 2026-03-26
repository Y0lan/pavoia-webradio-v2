package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSearch_MissingQuery(t *testing.T) {
	h := &SearchHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/search", nil)
	h.HandleSearch(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing q param, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "q parameter required" {
		t.Errorf("unexpected error: %s", body["error"])
	}
}
