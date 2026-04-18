package api

import (
	"log/slog"
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
			SELECT lt.id, lt.title, lt.artist, COALESCE(lt.album, ''),
				COALESCE(
					(SELECT array_agg(ts.stage_id ORDER BY ts.stage_id)
					   FROM track_stages ts WHERE ts.file_path = lt.file_path),
					ARRAY[]::text[]
				) AS stage_ids,
				COALESCE(lt.genre, '')
			FROM library_tracks lt
			WHERE lt.deleted_at IS NULL AND (lt.title ILIKE $1 OR lt.artist ILIKE $1 OR lt.album ILIKE $1)
			ORDER BY lt.title
			LIMIT $2
		`, pattern, limit)
		if err != nil {
			slog.Warn("search tracks: query failed", "error", err)
			WriteError(w, http.StatusInternalServerError, "query failed")
			return
		}
		type trackResult struct {
			ID       int64    `json:"id"`
			Title    string   `json:"title"`
			Artist   string   `json:"artist"`
			Album    string   `json:"album"`
			StageIDs []string `json:"stage_ids"`
			Genre    string   `json:"genre"`
		}
		tracks := make([]trackResult, 0)
		for rows.Next() {
			var t trackResult
			if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.StageIDs, &t.Genre); err != nil {
				rows.Close()
				slog.Warn("search tracks: scan failed", "error", err)
				WriteError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			if t.StageIDs == nil {
				t.StageIDs = []string{}
			}
			tracks = append(tracks, t)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			WriteError(w, http.StatusInternalServerError, "rows iteration failed")
			return
		}
		rows.Close()
		result["tracks"] = tracks
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
		if err != nil {
			slog.Warn("search artists: query failed", "error", err)
			WriteError(w, http.StatusInternalServerError, "query failed")
			return
		}
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
			if err := rows.Scan(&a.ID, &a.Name, &a.Country, &a.ImageURL); err != nil {
				slog.Warn("search artists: scan failed", "error", err)
				WriteError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			artists = append(artists, a)
		}
		if err := rows.Err(); err != nil {
			WriteError(w, http.StatusInternalServerError, "rows iteration failed")
			return
		}
		result["artists"] = artists
	}

	result["query"] = q
	result["fallback"] = true // indicate Postgres fallback, not Meilisearch

	WriteJSON(w, http.StatusOK, result)
}
