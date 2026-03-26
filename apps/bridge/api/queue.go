package api

import (
	"net/http"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
	mpdpool "github.com/Y0lan/pavoia-webradio-v2/apps/bridge/mpd"
)

// QueueHandlers holds dependencies for queue API handlers.
type QueueHandlers struct {
	Pool   *mpdpool.Pool
	Config *config.Config
}

// HandleQueue serves GET /api/stages/{id}/queue
// Returns the upcoming tracks in the MPD queue for a stage.
func (h *QueueHandlers) HandleQueue(w http.ResponseWriter, r *http.Request) {
	stageID := r.PathValue("id")
	if h.Config.StageByID(stageID) == nil {
		WriteError(w, http.StatusNotFound, "stage not found")
		return
	}

	np := h.Pool.NowPlaying(stageID)
	if np.Status == "offline" {
		WriteError(w, http.StatusServiceUnavailable, "stage offline")
		return
	}

	// For now, return the current song as queue[0].
	// Full queue support requires PlaylistInfo from MPD, which needs
	// a new method on the pool. This will be added when needed.
	type queueEntry struct {
		Position int               `json:"position"`
		Song     map[string]string `json:"song"`
		Current  bool              `json:"current"`
	}

	queue := []queueEntry{
		{Position: 0, Song: np.Song, Current: true},
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"stage_id": stageID,
		"queue":    queue,
	})
}
