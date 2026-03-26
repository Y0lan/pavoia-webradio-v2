package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HistoryEntry represents a single track play record.
type HistoryEntry struct {
	ID       int64     `json:"id"`
	StageID  string    `json:"stage_id"`
	Artist   string    `json:"artist"`
	Title    string    `json:"title"`
	Album    string    `json:"album"`
	FilePath string    `json:"file_path,omitempty"`
	PlayedAt time.Time `json:"played_at"`
}

// CalendarDay represents a single day in the play calendar heatmap.
type CalendarDay struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	Count int    `json:"count"` // number of plays
}

// HeatmapCell represents one cell in the 7x24 listening heatmap.
type HeatmapCell struct {
	DayOfWeek int `json:"day_of_week"` // 0=Sun, 6=Sat
	Hour      int `json:"hour"`        // 0-23
	Count     int `json:"count"`
}

// HistoryHandlers holds dependencies for history API handlers.
type HistoryHandlers struct {
	DB *pgxpool.Pool
}

// HandleHistory serves GET /api/history with filters and pagination.
func (h *HistoryHandlers) HandleHistory(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	pg := ParsePagination(r)
	tr := ParseTimeRange(r)
	q := r.URL.Query()

	// Build WHERE clause dynamically
	where := "WHERE 1=1"
	args := []any{}
	argN := 0

	nextArg := func() string {
		argN++
		return fmt.Sprintf("$%d", argN)
	}

	if stage := q.Get("stage"); stage != "" {
		where += " AND stage_id = " + nextArg()
		args = append(args, stage)
	}
	if tr.From != nil {
		where += " AND played_at >= " + nextArg()
		args = append(args, *tr.From)
	}
	if tr.To != nil {
		where += " AND played_at <= " + nextArg()
		args = append(args, *tr.To)
	}
	if artist := q.Get("artist"); artist != "" {
		where += " AND lower(artist) LIKE lower(" + nextArg() + ")"
		args = append(args, "%"+artist+"%")
	}
	if genre := q.Get("genre"); genre != "" {
		// Join with library_tracks for genre filter
		where += " AND file_path IN (SELECT file_path FROM library_tracks WHERE genre = " + nextArg() + ")"
		args = append(args, genre)
	}
	if search := q.Get("search"); search != "" {
		where += " AND (title ILIKE " + nextArg() + " OR artist ILIKE " + nextArg() + ")"
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM track_plays " + where
	if err := h.DB.QueryRow(r.Context(), countQuery, args...).Scan(&total); err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	// Sort
	orderBy := "ORDER BY played_at DESC"
	if sort := q.Get("sort"); sort == "oldest" {
		orderBy = "ORDER BY played_at ASC"
	}

	// Fetch page
	dataQuery := fmt.Sprintf(
		"SELECT id, stage_id, artist, title, COALESCE(album, ''), COALESCE(file_path, ''), played_at FROM track_plays %s %s LIMIT %d OFFSET %d",
		where, orderBy, pg.PerPage, pg.Offset,
	)

	rows, err := h.DB.Query(r.Context(), dataQuery, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	entries := make([]HistoryEntry, 0)
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.StageID, &e.Artist, &e.Title, &e.Album, &e.FilePath, &e.PlayedAt); err != nil {
			WriteError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		entries = append(entries, e)
	}

	WritePaged(w, entries, pg, total)
}

// HandleHistoryCalendar serves GET /api/history/calendar?year=
// Returns play counts per day for a GitHub-style heatmap.
func (h *HistoryHandlers) HandleHistoryCalendar(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	year := QueryInt(r, "year", time.Now().Year())

	rows, err := h.DB.Query(r.Context(), `
		SELECT DATE(played_at)::text, COUNT(*)
		FROM track_plays
		WHERE EXTRACT(YEAR FROM played_at) = $1
		GROUP BY DATE(played_at)
		ORDER BY DATE(played_at)
	`, year)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	days := make([]CalendarDay, 0)
	for rows.Next() {
		var d CalendarDay
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			WriteError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		days = append(days, d)
	}

	WriteJSON(w, http.StatusOK, map[string]any{"year": year, "days": days})
}

// HandleHistoryHeatmap serves GET /api/history/heatmap
// Returns a 7x24 grid of play counts (day_of_week x hour).
func (h *HistoryHandlers) HandleHistoryHeatmap(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	query := `
		SELECT EXTRACT(DOW FROM played_at)::int, EXTRACT(HOUR FROM played_at)::int, COUNT(*)
		FROM track_plays
	`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += " WHERE stage_id = $1"
		args = append(args, stage)
	}
	query += " GROUP BY 1, 2 ORDER BY 1, 2"

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	cells := make([]HeatmapCell, 0)
	for rows.Next() {
		var c HeatmapCell
		if err := rows.Scan(&c.DayOfWeek, &c.Hour, &c.Count); err != nil {
			WriteError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		cells = append(cells, c)
	}

	WriteJSON(w, http.StatusOK, map[string]any{"cells": cells})
}

// HandleStageHistory serves GET /api/stages/{id}/history
// Shortcut for history filtered by a single stage.
func (h *HistoryHandlers) HandleStageHistory(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	stageID := r.PathValue("id")
	pg := ParsePagination(r)

	var total int
	if err := h.DB.QueryRow(r.Context(),
		"SELECT COUNT(*) FROM track_plays WHERE stage_id = $1", stageID,
	).Scan(&total); err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	rows, err := h.DB.Query(r.Context(), `
		SELECT id, stage_id, artist, title, COALESCE(album, ''), COALESCE(file_path, ''), played_at
		FROM track_plays WHERE stage_id = $1
		ORDER BY played_at DESC
		LIMIT $2 OFFSET $3
	`, stageID, pg.PerPage, pg.Offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	entries := make([]HistoryEntry, 0)
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.StageID, &e.Artist, &e.Title, &e.Album, &e.FilePath, &e.PlayedAt); err != nil {
			WriteError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		entries = append(entries, e)
	}

	WritePaged(w, entries, pg, total)
}

// HandleHistoryByID serves GET /api/history/{id}
func (h *HistoryHandlers) HandleHistoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var e HistoryEntry
	err = h.DB.QueryRow(r.Context(), `
		SELECT id, stage_id, artist, title, COALESCE(album, ''), COALESCE(file_path, ''), played_at
		FROM track_plays WHERE id = $1
	`, id).Scan(&e.ID, &e.StageID, &e.Artist, &e.Title, &e.Album, &e.FilePath, &e.PlayedAt)
	if err != nil {
		WriteError(w, http.StatusNotFound, "play not found")
		return
	}

	WriteJSON(w, http.StatusOK, e)
}
