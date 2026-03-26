package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SearchHandlers holds dependencies for search API handlers.
type SearchHandlers struct {
	DB *pgxpool.Pool
	// MeiliURL and MeiliKey will be added when Meilisearch is integrated.
	// For now, search falls back to Postgres ILIKE.
}

// HandleSearch serves GET /api/search?q=&type=&limit=
// Falls back to Postgres ILIKE when Meilisearch is not configured.
func (h *SearchHandlers) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		WriteError(w, http.StatusBadRequest, "q parameter required")
		return
	}

	limit := QueryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	searchType := r.URL.Query().Get("type")
	if searchType == "" {
		searchType = "all"
	}

	pattern := "%" + q + "%"
	result := map[string]any{}

	// Search tracks
	if searchType == "all" || searchType == "tracks" {
		rows, err := h.DB.Query(r.Context(), `
			SELECT id, title, artist, COALESCE(album, ''), COALESCE(stage_id, ''), COALESCE(genre, '')
			FROM library_tracks
			WHERE title ILIKE $1 OR artist ILIKE $1 OR album ILIKE $1
			ORDER BY title
			LIMIT $2
		`, pattern, limit)
		if err == nil {
			defer rows.Close()
			type trackResult struct {
				ID      int64  `json:"id"`
				Title   string `json:"title"`
				Artist  string `json:"artist"`
				Album   string `json:"album"`
				StageID string `json:"stage_id"`
				Genre   string `json:"genre"`
			}
			tracks := make([]trackResult, 0)
			for rows.Next() {
				var t trackResult
				rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.StageID, &t.Genre)
				tracks = append(tracks, t)
			}
			result["tracks"] = tracks
		}
	}

	// Search artists
	if searchType == "all" || searchType == "artists" {
		rows, err := h.DB.Query(r.Context(), `
			SELECT id, name, country, image_url
			FROM artists
			WHERE name ILIKE $1
			ORDER BY name
			LIMIT $2
		`, pattern, limit)
		if err == nil {
			defer rows.Close()
			type artistResult struct {
				ID       int64   `json:"id"`
				Name     string  `json:"name"`
				Country  *string `json:"country"`
				ImageURL *string `json:"image_url"`
			}
			artists := make([]artistResult, 0)
			for rows.Next() {
				var a artistResult
				rows.Scan(&a.ID, &a.Name, &a.Country, &a.ImageURL)
				artists = append(artists, a)
			}
			result["artists"] = artists
		}
	}

	result["query"] = q
	result["fallback"] = true // indicate Postgres fallback, not Meilisearch

	WriteJSON(w, http.StatusOK, result)
}
