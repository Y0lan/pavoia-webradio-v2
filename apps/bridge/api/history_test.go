package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHistory_NoDB(t *testing.T) {
	h := &HistoryHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/history", nil)
	h.HandleHistory(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleHistoryCalendar_NoDB(t *testing.T) {
	h := &HistoryHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/history/calendar", nil)
	h.HandleHistoryCalendar(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleHistoryHeatmap_NoDB(t *testing.T) {
	h := &HistoryHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/history/heatmap", nil)
	h.HandleHistoryHeatmap(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleHistoryByID_NoDB(t *testing.T) {
	h := &HistoryHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/history/123", nil)
	r.SetPathValue("id", "123")
	h.HandleHistoryByID(w, r)

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

	// Invalid ID should return 400 before checking DB
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ID, got %d", w.Code)
	}
}

func TestHandleStageHistory_NoDB(t *testing.T) {
	h := &HistoryHandlers{DB: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stages/gaende-favorites/history", nil)
	r.SetPathValue("id", "gaende-favorites")
	h.HandleStageHistory(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHistoryEntry_JSON(t *testing.T) {
	// Verify the JSON serialization format
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
	// file_path should be omitted when empty
	if _, ok := m["file_path"]; ok && m["file_path"] != "" {
		t.Errorf("expected file_path to be empty or omitted, got %v", m["file_path"])
	}
}
