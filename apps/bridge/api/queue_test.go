package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
	mpdpool "github.com/Y0lan/pavoia-webradio-v2/apps/bridge/mpd"
)

func TestHandleQueue_UnknownStage(t *testing.T) {
	cfg := &config.Config{}
	pool := mpdpool.NewPool(nil, nil)
	h := &QueueHandlers{Pool: pool, Config: cfg}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stages/nonexistent/queue", nil)
	r.SetPathValue("id", "nonexistent")
	h.HandleQueue(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleQueue_OfflineStage(t *testing.T) {
	stages := []config.StageConfig{{ID: "s1", Name: "S1", MPDPort: 6600, Visible: true}}
	cfg := &config.Config{Stages: stages}
	pool := mpdpool.NewPool(stages, nil)
	h := &QueueHandlers{Pool: pool, Config: cfg}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stages/s1/queue", nil)
	r.SetPathValue("id", "s1")
	h.HandleQueue(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for offline stage, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "stage offline" {
		t.Errorf("unexpected error: %s", body["error"])
	}
}
