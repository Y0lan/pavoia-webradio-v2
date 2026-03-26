package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
)

func TestHandleStream_UnknownStage(t *testing.T) {
	h := &StreamHandlers{Config: &config.Config{}, MPDHost: "127.0.0.1"}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stream/nonexistent", nil)
	r.SetPathValue("stageId", "nonexistent")
	h.HandleStream(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleStream_ProxiesAudio(t *testing.T) {
	// Mock MPD HTTP stream server
	audioData := strings.Repeat("FAKE_AUDIO_DATA", 100)
	mpd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte(audioData))
	}))
	defer mpd.Close()

	cfg := &config.Config{
		Stages: []config.StageConfig{
			{ID: "test-stage", Name: "Test", StreamPort: 0, Visible: true},
		},
	}

	h := NewStreamHandlers(cfg, "127.0.0.1")
	// Override the URL builder for testing
	h.streamURLOverride = mpd.URL

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stream/test-stage", nil)
	r.SetPathValue("stageId", "test-stage")
	h.HandleStream(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "audio/mpeg" {
		t.Errorf("expected Content-Type audio/mpeg, got %s", ct)
	}

	if w.Body.String() != audioData {
		t.Errorf("expected audio data to be proxied, got %d bytes", w.Body.Len())
	}
}

func TestHandleStream_MPDUnreachable(t *testing.T) {
	cfg := &config.Config{
		Stages: []config.StageConfig{
			{ID: "dead-stage", Name: "Dead", StreamPort: 19999, Visible: true},
		},
	}

	h := NewStreamHandlers(cfg, "127.0.0.1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stream/dead-stage", nil)
	r.SetPathValue("stageId", "dead-stage")
	h.HandleStream(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestHandleStream_ListenerCounting(t *testing.T) {
	// Slow stream that stays open
	mpd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(200)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Write slowly to keep connection open
		for i := 0; i < 5; i++ {
			w.Write([]byte("chunk"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer mpd.Close()

	cfg := &config.Config{
		Stages: []config.StageConfig{
			{ID: "s1", Name: "S1", StreamPort: 0, Visible: true},
		},
	}

	h := NewStreamHandlers(cfg, "127.0.0.1")
	h.streamURLOverride = mpd.URL

	// Check initial count
	counts := h.ListenerCounts()
	if counts["s1"] != 0 {
		t.Fatalf("expected 0 listeners initially, got %d", counts["s1"])
	}

	// Start a stream in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/stream/s1", nil)
		r.SetPathValue("stageId", "s1")
		h.HandleStream(w, r)
	}()

	// Wait briefly for the connection to establish
	time.Sleep(30 * time.Millisecond)

	counts = h.ListenerCounts()
	if counts["s1"] != 1 {
		t.Errorf("expected 1 listener during stream, got %d", counts["s1"])
	}

	// Wait for stream to finish
	<-done

	counts = h.ListenerCounts()
	if counts["s1"] != 0 {
		t.Errorf("expected 0 listeners after disconnect, got %d", counts["s1"])
	}
}

func TestHandleStream_Headers(t *testing.T) {
	mpd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/ogg")
		w.Header().Set("icy-name", "GAENDE Radio")
		io.WriteString(w, "ogg-data")
	}))
	defer mpd.Close()

	cfg := &config.Config{
		Stages: []config.StageConfig{
			{ID: "s1", Name: "S1", StreamPort: 0, Visible: true},
		},
	}

	h := NewStreamHandlers(cfg, "127.0.0.1")
	h.streamURLOverride = mpd.URL

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/stream/s1", nil)
	r.SetPathValue("stageId", "s1")
	h.HandleStream(w, r)

	if ct := w.Header().Get("Content-Type"); ct != "audio/ogg" {
		t.Errorf("expected audio/ogg, got %s", ct)
	}
	// Should have no-buffering header
	if nb := w.Header().Get("X-Accel-Buffering"); nb != "no" {
		t.Errorf("expected X-Accel-Buffering: no, got %s", nb)
	}
}
