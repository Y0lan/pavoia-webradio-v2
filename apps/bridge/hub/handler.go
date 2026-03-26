package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// HandleWS upgrades an HTTP connection to WebSocket and manages the client lifecycle.
// Route: GET /ws
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Tighten to actual domain on deploy
	})
	if err != nil {
		slog.Warn("ws accept failed", "error", err, "remote", r.RemoteAddr)
		return
	}
	conn.SetReadLimit(4096) // Subscribe messages are ~50 bytes

	// Dedicated context for WS lifecycle — decoupled from r.Context()
	ctx, cancel := context.WithCancel(context.Background())

	client := h.NewClient()
	h.Register(client)
	slog.Info("ws client connected", "remote", r.RemoteAddr, "clients", h.ClientCount())

	// Writer goroutine: reads from client.Send, writes to WS
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		defer conn.CloseNow()
		for msg := range client.Send {
			writeCtx, writeCancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, msg)
			writeCancel()
			if err != nil {
				slog.Debug("ws write failed", "error", err)
				return
			}
		}
	}()

	// Reader loop: reads client messages (subscribe/unsubscribe)
	defer func() {
		cancel() // Cancel the WS context
		h.Unregister(client)
		conn.CloseNow()
		writerWg.Wait() // Wait for writer goroutine to exit
		slog.Info("ws client disconnected", "remote", r.RemoteAddr, "clients", h.ClientCount())
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			slog.Debug("ws invalid message", "error", err, "data", string(data))
			continue
		}

		switch msg.Type {
		case "subscribe":
			client.Subscribe(msg.Stages)
			slog.Debug("ws subscribe", "stages", msg.Stages, "remote", r.RemoteAddr)

			// Send snapshot for newly subscribed stages
			snapshots := h.Snapshot(msg.Stages)
			for _, snap := range snapshots {
				select {
				case client.Send <- snap:
				default:
				}
			}
		case "unsubscribe":
			client.Unsubscribe(msg.Stages)
			slog.Debug("ws unsubscribe", "stages", msg.Stages, "remote", r.RemoteAddr)
		default:
			slog.Debug("ws unknown message type", "type", msg.Type)
		}
	}
}

// HandleSSE serves the Server-Sent Events endpoint for global broadcasts.
// Route: GET /api/events
func (h *Hub) HandleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Nginx: disable proxy buffering
	flusher.Flush()                            // Send headers immediately

	sc := h.NewSSEClient()
	h.RegisterSSE(sc)
	slog.Info("sse client connected", "remote", r.RemoteAddr, "sse_clients", h.SSEClientCount())

	defer func() {
		h.UnregisterSSE(sc)
		slog.Info("sse client disconnected", "remote", r.RemoteAddr, "sse_clients", h.SSEClientCount())
	}()

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-sc.Events:
			if !ok {
				return
			}
			data, err := json.Marshal(event.Data)
			if err != nil {
				slog.Warn("sse marshal failed", "error", err)
				continue
			}

			if event.ID != "" {
				if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
					return
				}
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, data); err != nil {
				return
			}
			flusher.Flush()

		case <-keepalive.C:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
