package hub

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
)

const (
	sendBufSize = 32
	sseBufSize  = 16
)

// NowPlayingEvent is the data broadcast to WS clients on track change.
type NowPlayingEvent struct {
	StageID  string `json:"stage_id"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Elapsed  string `json:"elapsed,omitempty"`
	Duration string `json:"duration,omitempty"`
	File     string `json:"file,omitempty"`
}

// Envelope wraps all WS messages with a type discriminator.
type Envelope struct {
	Type    string `json:"type"`
	StageID string `json:"stage_id,omitempty"`
	Data    any    `json:"data"`
}

// ClientMessage is a message received from a WS client.
type ClientMessage struct {
	Type   string   `json:"type"`   // "subscribe", "unsubscribe"
	Stages []string `json:"stages"`
}

// SSEEvent is a server-sent event for the /api/events endpoint.
type SSEEvent struct {
	Event string `json:"event"` // "listeners", "sync_update", "enrichment"
	Data  any    `json:"data"`
	ID    string `json:"id,omitempty"`
}

// Client represents a connected WebSocket client.
type Client struct {
	hub      *Hub
	stages   map[string]bool
	stagesMu sync.RWMutex
	Send     chan []byte
}

// Subscribe adds valid stages to this client's subscription set.
// Unknown stage IDs are silently ignored.
func (c *Client) Subscribe(stages []string) {
	c.stagesMu.Lock()
	defer c.stagesMu.Unlock()
	for _, s := range stages {
		if c.hub.validStages[s] {
			c.stages[s] = true
		}
	}
}

// Unsubscribe removes stages from this client's subscription set.
func (c *Client) Unsubscribe(stages []string) {
	c.stagesMu.Lock()
	defer c.stagesMu.Unlock()
	for _, s := range stages {
		delete(c.stages, s)
	}
}

// IsSubscribed checks if the client is subscribed to a stage.
func (c *Client) IsSubscribed(stageID string) bool {
	c.stagesMu.RLock()
	defer c.stagesMu.RUnlock()
	return c.stages[stageID]
}

// SubscribedStages returns a copy of the subscribed stage IDs.
func (c *Client) SubscribedStages() []string {
	c.stagesMu.RLock()
	defer c.stagesMu.RUnlock()
	out := make([]string, 0, len(c.stages))
	for s := range c.stages {
		out = append(out, s)
	}
	return out
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	Events chan SSEEvent
}

// Hub manages WebSocket and SSE client connections and broadcasting.
type Hub struct {
	mu          sync.RWMutex
	clients     map[*Client]struct{}
	validStages map[string]bool

	// Last known now-playing per stage — sent to new subscribers as snapshot
	snapshotMu sync.RWMutex
	snapshots  map[string][]byte // stage_id → marshaled Envelope

	sseMu      sync.RWMutex
	sseClients map[*SSEClient]struct{}
}

// New creates a new Hub. stageIDs defines the set of valid stages for subscription.
func New(stageIDs ...string) *Hub {
	valid := make(map[string]bool, len(stageIDs))
	for _, id := range stageIDs {
		valid[id] = true
	}
	return &Hub{
		clients:     make(map[*Client]struct{}),
		validStages: valid,
		snapshots:   make(map[string][]byte),
		sseClients:  make(map[*SSEClient]struct{}),
	}
}

// NewClient creates a new WS client bound to this hub.
func (h *Hub) NewClient() *Client {
	return &Client{
		hub:    h,
		stages: make(map[string]bool),
		Send:   make(chan []byte, sendBufSize),
	}
}

// Register adds a WS client to the hub.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

// Unregister removes a WS client from the hub and closes its send channel.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.Send)
	}
}

// ClientCount returns the number of connected WS clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ListenerCounts returns a map of stage_id → number of subscribed WS clients.
func (h *Hub) ListenerCounts() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	counts := make(map[string]int)
	for c := range h.clients {
		for _, s := range c.SubscribedStages() {
			counts[s]++
		}
	}
	return counts
}

// Snapshot returns the cached now-playing data for subscribed stages.
func (h *Hub) Snapshot(stageIDs []string) [][]byte {
	h.snapshotMu.RLock()
	defer h.snapshotMu.RUnlock()

	var result [][]byte
	for _, id := range stageIDs {
		if data, ok := h.snapshots[id]; ok {
			result = append(result, data)
		}
	}
	return result
}

// BroadcastNowPlaying sends a now_playing event to all clients subscribed to the event's stage.
// Uses latest-wins backpressure: drains stale messages before sending.
func (h *Hub) BroadcastNowPlaying(event NowPlayingEvent) {
	env := Envelope{
		Type:    "now_playing",
		StageID: event.StageID,
		Data:    event,
	}
	data, err := json.Marshal(env)
	if err != nil {
		slog.Warn("failed to marshal now_playing", "error", err)
		return
	}

	// Cache for snapshot delivery to new subscribers
	h.snapshotMu.Lock()
	h.snapshots[event.StageID] = data
	h.snapshotMu.Unlock()

	h.broadcastToStage(event.StageID, data)
}

// broadcastToStage sends pre-marshaled data to all clients subscribed to the given stage.
// Uses latest-wins: drains stale messages if the send buffer is full.
func (h *Hub) broadcastToStage(stageID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c.IsSubscribed(stageID) {
			select {
			case c.Send <- data:
			default:
				// Latest-wins: drain one stale message and retry
				select {
				case <-c.Send:
				default:
				}
				select {
				case c.Send <- data:
				default:
					slog.Debug("dropping ws message, client send buffer full")
				}
			}
		}
	}
}

// NewSSEClient creates a new SSE client.
func (h *Hub) NewSSEClient() *SSEClient {
	return &SSEClient{
		Events: make(chan SSEEvent, sseBufSize),
	}
}

// RegisterSSE adds an SSE client to the hub.
func (h *Hub) RegisterSSE(sc *SSEClient) {
	h.sseMu.Lock()
	defer h.sseMu.Unlock()
	h.sseClients[sc] = struct{}{}
}

// UnregisterSSE removes an SSE client from the hub.
func (h *Hub) UnregisterSSE(sc *SSEClient) {
	h.sseMu.Lock()
	defer h.sseMu.Unlock()
	if _, ok := h.sseClients[sc]; ok {
		delete(h.sseClients, sc)
		close(sc.Events)
	}
}

// SSEClientCount returns the number of connected SSE clients.
func (h *Hub) SSEClientCount() int {
	h.sseMu.RLock()
	defer h.sseMu.RUnlock()
	return len(h.sseClients)
}

// BroadcastSSE sends an event to all connected SSE clients.
func (h *Hub) BroadcastSSE(event SSEEvent) {
	// Sanitize event fields to prevent SSE newline injection
	event.Event = strings.ReplaceAll(event.Event, "\n", "")
	event.ID = strings.ReplaceAll(event.ID, "\n", "")

	h.sseMu.RLock()
	defer h.sseMu.RUnlock()

	for sc := range h.sseClients {
		select {
		case sc.Events <- event:
		default:
			slog.Debug("dropping sse event, client buffer full")
		}
	}
}
