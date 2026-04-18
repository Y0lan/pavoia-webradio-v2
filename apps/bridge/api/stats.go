package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsHandlers holds dependencies for stats API handlers.
type StatsHandlers struct {
	DB *pgxpool.Pool
}

// HandleStatsOverview serves GET /api/stats/overview
func (h *StatsHandlers) HandleStatsOverview(w http.ResponseWriter, r *http.Request) {
	weekAgo := time.Now().AddDate(0, 0, -7)

	// Combine library_tracks stats into a single query.
	var totalTracks, totalArtists, weekAdded int
	err := h.DB.QueryRow(r.Context(), `
		SELECT
			COUNT(*),
			COUNT(DISTINCT lower(artist)),
			COUNT(*) FILTER (WHERE added_at >= $1)
		FROM library_tracks
	`, weekAgo).Scan(&totalTracks, &totalArtists, &weekAdded)
	if err != nil {
		slog.Warn("stats overview: library_tracks query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	// Combine track_plays stats into a single query.
	var totalPlays, weekPlays int
	var totalHours float64
	err = h.DB.QueryRow(r.Context(), `
		SELECT
			COUNT(*),
			COALESCE(SUM(duration_sec)/3600.0, 0),
			COUNT(*) FILTER (WHERE played_at >= $1)
		FROM track_plays
	`, weekAgo).Scan(&totalPlays, &totalHours, &weekPlays)
	if err != nil {
		slog.Warn("stats overview: track_plays query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"total_tracks":  totalTracks,
		"total_artists": totalArtists,
		"total_plays":   totalPlays,
		"total_hours":   totalHours,
		"week_added":    weekAdded,
		"week_plays":    weekPlays,
	})
}

// HandleStatsTopArtists serves GET /api/stats/top-artists?by=plays|tracks&limit=&stage=&period=
func (h *StatsHandlers) HandleStatsTopArtists(w http.ResponseWriter, r *http.Request) {
	limit := QueryIntBounded(r, "limit", 20, 1, 100)
	by := r.URL.Query().Get("by")
	stage := r.URL.Query().Get("stage")
	period := r.URL.Query().Get("period")

	var query string
	args := []any{}
	argN := 0
	nextArg := func() string { argN++; return fmt.Sprintf("$%d", argN) }

	// Period filter applies to different tables (library_tracks.added_at vs track_plays.played_at)
	// depending on `by`; build it here so we can splice into each branch's query.
	periodClause := ""
	if period != "" {
		var since time.Time
		switch period {
		case "week":
			since = time.Now().AddDate(0, 0, -7)
		case "month":
			since = time.Now().AddDate(0, -1, 0)
		case "year":
			since = time.Now().AddDate(-1, 0, 0)
		}
		if !since.IsZero() {
			switch by {
			case "tracks":
				periodClause = " AND added_at >= " + nextArg()
			default:
				periodClause = " AND played_at >= " + nextArg()
			}
			args = append(args, since)
		}
	}

	switch by {
	case "tracks":
		// Filter library_tracks by stage via the track_stages join table.
		stageClause := ""
		if stage != "" {
			stageClause = " AND EXISTS (SELECT 1 FROM track_stages ts WHERE ts.file_path = library_tracks.file_path AND ts.stage_id = " + nextArg() + ")"
			args = append(args, stage)
		}
		query = fmt.Sprintf(`
			SELECT artist, COUNT(*) as count
			FROM library_tracks WHERE 1=1 %s %s
			GROUP BY artist ORDER BY count DESC LIMIT %d
		`, stageClause, periodClause, limit)
	default: // plays — track_plays.stage_id is still scalar (a play happens on one stage)
		stageClause := ""
		if stage != "" {
			stageClause = " AND stage_id = " + nextArg()
			args = append(args, stage)
		}
		query = fmt.Sprintf(`
			SELECT artist, COUNT(*) as count
			FROM track_plays WHERE 1=1 %s %s
			GROUP BY artist ORDER BY count DESC LIMIT %d
		`, stageClause, periodClause, limit)
	}

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type entry struct {
		Artist string `json:"artist"`
		Count  int    `json:"count"`
	}
	results := make([]entry, 0)
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Artist, &e.Count); err != nil {
			slog.Warn("stats top artists: scan failed", "error", err)
			continue
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats top artists: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// HandleStatsTopTracks serves GET /api/stats/top-tracks?limit=&stage=&period=
func (h *StatsHandlers) HandleStatsTopTracks(w http.ResponseWriter, r *http.Request) {
	limit := QueryIntBounded(r, "limit", 20, 1, 100)
	stage := r.URL.Query().Get("stage")
	period := r.URL.Query().Get("period")

	args := []any{}
	argN := 0
	nextArg := func() string { argN++; return fmt.Sprintf("$%d", argN) }
	where := ""

	if stage != "" {
		where += " AND stage_id = " + nextArg()
		args = append(args, stage)
	}
	if period != "" {
		var since time.Time
		switch period {
		case "week":
			since = time.Now().AddDate(0, 0, -7)
		case "month":
			since = time.Now().AddDate(0, -1, 0)
		case "year":
			since = time.Now().AddDate(-1, 0, 0)
		}
		if !since.IsZero() {
			where += " AND played_at >= " + nextArg()
			args = append(args, since)
		}
	}

	query := fmt.Sprintf(`
		SELECT title, artist, COUNT(*) as plays
		FROM track_plays WHERE 1=1 %s
		GROUP BY title, artist ORDER BY plays DESC LIMIT %d
	`, where, limit)

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type entry struct {
		Title  string `json:"title"`
		Artist string `json:"artist"`
		Plays  int    `json:"plays"`
	}
	results := make([]entry, 0)
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Title, &e.Artist, &e.Plays); err != nil {
			slog.Warn("stats top tracks: scan failed", "error", err)
			continue
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats top tracks: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// HandleStatsStages serves GET /api/stats/stages
func (h *StatsHandlers) HandleStatsStages(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT stage_id, COUNT(*) as plays,
			COUNT(DISTINCT artist) as unique_artists
		FROM track_plays
		GROUP BY stage_id
		ORDER BY plays DESC
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type stageStat struct {
		StageID       string `json:"stage_id"`
		Plays         int    `json:"plays"`
		UniqueArtists int    `json:"unique_artists"`
	}
	results := make([]stageStat, 0)
	for rows.Next() {
		var s stageStat
		if err := rows.Scan(&s.StageID, &s.Plays, &s.UniqueArtists); err != nil {
			slog.Warn("stats stages: scan failed", "error", err)
			continue
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats stages: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, results)
}

// HandleStatsBPM serves GET /api/stats/bpm?stage=
func (h *StatsHandlers) HandleStatsBPM(w http.ResponseWriter, r *http.Request) {
	query := `SELECT bpm, COUNT(*) FROM library_tracks WHERE bpm IS NOT NULL AND bpm > 0`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += ` AND EXISTS (SELECT 1 FROM track_stages ts WHERE ts.file_path = library_tracks.file_path AND ts.stage_id = $1)`
		args = append(args, stage)
	}
	query += " GROUP BY bpm ORDER BY bpm"

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type bin struct {
		BPM   int `json:"bpm"`
		Count int `json:"count"`
	}
	bins := make([]bin, 0)
	for rows.Next() {
		var b bin
		if err := rows.Scan(&b.BPM, &b.Count); err != nil {
			slog.Warn("stats bpm: scan failed", "error", err)
			continue
		}
		bins = append(bins, b)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats bpm: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, bins)
}

// HandleStatsKeys serves GET /api/stats/keys?stage=
func (h *StatsHandlers) HandleStatsKeys(w http.ResponseWriter, r *http.Request) {
	query := `SELECT camelot_key, COUNT(*) FROM library_tracks WHERE camelot_key IS NOT NULL`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += ` AND EXISTS (SELECT 1 FROM track_stages ts WHERE ts.file_path = library_tracks.file_path AND ts.stage_id = $1)`
		args = append(args, stage)
	}
	query += " GROUP BY camelot_key ORDER BY camelot_key"

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type keyCount struct {
		Key   string `json:"key"`
		Count int    `json:"count"`
	}
	keys := make([]keyCount, 0)
	for rows.Next() {
		var k keyCount
		if err := rows.Scan(&k.Key, &k.Count); err != nil {
			slog.Warn("stats keys: scan failed", "error", err)
			continue
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats keys: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, keys)
}

// HandleStatsDecades serves GET /api/stats/decades?stage=
func (h *StatsHandlers) HandleStatsDecades(w http.ResponseWriter, r *http.Request) {
	query := `SELECT (year/10)*10 as decade, COUNT(*) FROM library_tracks WHERE year IS NOT NULL AND year > 0`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += ` AND EXISTS (SELECT 1 FROM track_stages ts WHERE ts.file_path = library_tracks.file_path AND ts.stage_id = $1)`
		args = append(args, stage)
	}
	query += " GROUP BY decade ORDER BY decade"

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type decadeCount struct {
		Decade int `json:"decade"`
		Count  int `json:"count"`
	}
	decades := make([]decadeCount, 0)
	for rows.Next() {
		var d decadeCount
		if err := rows.Scan(&d.Decade, &d.Count); err != nil {
			slog.Warn("stats decades: scan failed", "error", err)
			continue
		}
		decades = append(decades, d)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats decades: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, decades)
}

// HandleStatsGenres serves GET /api/stats/genres?stage=
func (h *StatsHandlers) HandleStatsGenres(w http.ResponseWriter, r *http.Request) {
	query := `SELECT genre, COUNT(*) FROM library_tracks WHERE genre IS NOT NULL AND genre != ''`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += ` AND EXISTS (SELECT 1 FROM track_stages ts WHERE ts.file_path = library_tracks.file_path AND ts.stage_id = $1)`
		args = append(args, stage)
	}
	query += " GROUP BY genre ORDER BY COUNT(*) DESC"

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type genreCount struct {
		Genre string `json:"genre"`
		Count int    `json:"count"`
	}
	genres := make([]genreCount, 0)
	for rows.Next() {
		var g genreCount
		if err := rows.Scan(&g.Genre, &g.Count); err != nil {
			slog.Warn("stats genres: scan failed", "error", err)
			continue
		}
		genres = append(genres, g)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats genres: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, genres)
}

// HandleStatsDiscoveryVelocity serves GET /api/stats/discovery-velocity
// Returns tracks added per week for the last 52 weeks.
func (h *StatsHandlers) HandleStatsDiscoveryVelocity(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT DATE_TRUNC('week', added_at)::date::text, COUNT(*)
		FROM library_tracks
		WHERE added_at >= NOW() - INTERVAL '52 weeks'
		GROUP BY DATE_TRUNC('week', added_at)
		ORDER BY 1
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type weekCount struct {
		Week  string `json:"week"`
		Count int    `json:"count"`
	}
	weeks := make([]weekCount, 0)
	for rows.Next() {
		var wc weekCount
		if err := rows.Scan(&wc.Week, &wc.Count); err != nil {
			slog.Warn("stats discovery velocity: scan failed", "error", err)
			continue
		}
		weeks = append(weeks, wc)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("stats discovery velocity: rows iteration error", "error", err)
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}

	WriteJSON(w, http.StatusOK, weeks)
}

// HandleStatsListeningHeatmap serves GET /api/stats/listening-heatmap?stage=
// Same as history heatmap but under /stats/ path.
func (h *StatsHandlers) HandleStatsListeningHeatmap(w http.ResponseWriter, r *http.Request) {
	hh := &HistoryHandlers{DB: h.DB}
	hh.HandleHistoryHeatmap(w, r)
}
