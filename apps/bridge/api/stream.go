package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
)

// StreamHandlers holds dependencies for the audio stream proxy.
type StreamHandlers struct {
	Config  *config.Config
	MPDHost string
	http    *http.Client

	// Listener counts per stage (atomic for concurrent access)
	listeners sync.Map // stageID → *atomic.Int64

	// For testing: override the stream URL instead of using MPDHost:StreamPort
	streamURLOverride string
}

// NewStreamHandlers creates a stream proxy handler.
func NewStreamHandlers(cfg *config.Config, mpdHost string) *StreamHandlers {
	return &StreamHandlers{
		Config:  cfg,
		MPDHost: mpdHost,
		http: &http.Client{
			// No timeout — streams are long-lived
			Timeout: 0,
		},
	}
}

// HandleStream serves GET /api/stream/{stageId}
// Proxies the MPD HTTP audio stream with CORS headers.
func (h *StreamHandlers) HandleStream(w http.ResponseWriter, r *http.Request) {
	stageID := r.PathValue("stageId")
	stage := h.Config.StageByID(stageID)
	if stage == nil {
		WriteError(w, http.StatusNotFound, "stage not found")
		return
	}

	// Build upstream URL
	streamURL := h.streamURLOverride
	if streamURL == "" {
		streamURL = fmt.Sprintf("http://%s:%d", h.MPDHost, stage.StreamPort)
	}

	// Connect to MPD HTTP stream
	req, err := http.NewRequestWithContext(r.Context(), "GET", streamURL, nil)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	resp, err := h.http.Do(req)
	if err != nil {
		slog.Warn("stream proxy: upstream failed", "stage", stageID, "url", streamURL, "error", err)
		WriteError(w, http.StatusBadGateway, "stream unavailable")
		return
	}
	defer resp.Body.Close()

	// Track listener
	counter := h.getCounter(stageID)
	counter.Add(1)
	defer counter.Add(-1)

	slog.Info("stream: listener connected", "stage", stageID, "listeners", counter.Load())
	defer func() {
		slog.Info("stream: listener disconnected", "stage", stageID, "listeners", counter.Load())
	}()

	// Forward headers
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "audio/mpeg")
	}
	w.Header().Set("X-Accel-Buffering", "no")     // Nginx: don't buffer
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Stream the audio — blocks until client disconnects or upstream closes
	w.WriteHeader(http.StatusOK)
	io.Copy(w, resp.Body)
}

// ListenerCounts returns the current listener count per stage.
func (h *StreamHandlers) ListenerCounts() map[string]int {
	counts := make(map[string]int)
	h.listeners.Range(func(key, value any) bool {
		stageID := key.(string)
		counter := value.(*atomic.Int64)
		if n := int(counter.Load()); n > 0 {
			counts[stageID] = n
		}
		return true
	})
	return counts
}

// TotalListeners returns the total number of active stream listeners.
func (h *StreamHandlers) TotalListeners() int {
	total := 0
	h.listeners.Range(func(_, value any) bool {
		total += int(value.(*atomic.Int64).Load())
		return true
	})
	return total
}

func (h *StreamHandlers) getCounter(stageID string) *atomic.Int64 {
	val, _ := h.listeners.LoadOrStore(stageID, &atomic.Int64{})
	return val.(*atomic.Int64)
}

// StreamKeepAlive returns an http.Client with no timeout for long-lived streams.
func StreamKeepAlive() *http.Client {
	return &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
}
