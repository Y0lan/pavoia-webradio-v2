package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireDB_RejectsNilDB(t *testing.T) {
	handler := RequireDB(nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleHistoryByID_InvalidID(t *testing.T) {
	h := &HistoryHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/history/abc", nil)
	r.SetPathValue("id", "abc")
	h.HandleHistoryByID(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ID, got %d", w.Code)
	}
}

func TestHistoryEntry_JSON(t *testing.T) {
	e := HistoryEntry{
		ID:       1,
		StageID:  "gaende-favorites",
		Artist:   "ARTBAT",
		Title:    "Meridian",
		Album:    "Meridian EP",
		FilePath: "",
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	if m["stage_id"] != "gaende-favorites" {
		t.Errorf("expected stage_id gaende-favorites, got %v", m["stage_id"])
	}
}

func TestQueryIntBounded(t *testing.T) {
	tests := []struct {
		query    string
		min, max int
		expected int
	}{
		{"limit=25", 1, 100, 25},
		{"limit=-5", 1, 100, 1},   // below min → clamped
		{"limit=999", 1, 100, 100}, // above max → clamped
		{"limit=1", 1, 100, 1},    // at min
		{"limit=100", 1, 100, 100}, // at max
		{"", 1, 100, 20},           // default
	}

	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/?"+tt.query, nil)
		v := QueryIntBounded(r, "limit", 20, tt.min, tt.max)
		if v != tt.expected {
			t.Errorf("query=%q: got %d, want %d", tt.query, v, tt.expected)
		}
	}
}
