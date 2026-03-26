package api

import (
	"fmt"
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
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var totalTracks, totalArtists, totalPlays int
	var totalHours float64

	h.DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM library_tracks").Scan(&totalTracks)
	h.DB.QueryRow(r.Context(), "SELECT COUNT(DISTINCT lower(artist)) FROM library_tracks").Scan(&totalArtists)
	h.DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM track_plays").Scan(&totalPlays)
	h.DB.QueryRow(r.Context(), "SELECT COALESCE(SUM(duration_sec)/3600.0, 0) FROM track_plays").Scan(&totalHours)

	// This week's additions and plays
	weekAgo := time.Now().AddDate(0, 0, -7)
	var weekAdded, weekPlays int
	h.DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM library_tracks WHERE added_at >= $1", weekAgo).Scan(&weekAdded)
	h.DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM track_plays WHERE played_at >= $1", weekAgo).Scan(&weekPlays)

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
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	limit := QueryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	by := r.URL.Query().Get("by")
	stage := r.URL.Query().Get("stage")
	period := r.URL.Query().Get("period")

	var query string
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

	switch by {
	case "tracks":
		query = fmt.Sprintf(`
			SELECT artist, COUNT(*) as count
			FROM library_tracks WHERE 1=1 %s
			GROUP BY artist ORDER BY count DESC LIMIT %d
		`, where, limit)
	default: // plays
		query = fmt.Sprintf(`
			SELECT artist, COUNT(*) as count
			FROM track_plays WHERE 1=1 %s
			GROUP BY artist ORDER BY count DESC LIMIT %d
		`, where, limit)
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
			continue
		}
		results = append(results, e)
	}

	WriteJSON(w, http.StatusOK, results)
}

// HandleStatsTopTracks serves GET /api/stats/top-tracks?limit=&stage=&period=
func (h *StatsHandlers) HandleStatsTopTracks(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	limit := QueryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
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
			continue
		}
		results = append(results, e)
	}

	WriteJSON(w, http.StatusOK, results)
}

// HandleStatsStages serves GET /api/stats/stages
func (h *StatsHandlers) HandleStatsStages(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	rows, err := h.DB.Query(r.Context(), `
		SELECT stage_id, COUNT(*) as plays,
			COUNT(DISTINCT artist) as unique_artists,
			AVG(NULLIF(lt.bpm, 0)) as avg_bpm
		FROM track_plays tp
		LEFT JOIN library_tracks lt ON tp.file_path = lt.file_path
		GROUP BY stage_id
		ORDER BY plays DESC
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type stageStat struct {
		StageID       string   `json:"stage_id"`
		Plays         int      `json:"plays"`
		UniqueArtists int      `json:"unique_artists"`
		AvgBPM        *float64 `json:"avg_bpm"`
	}
	results := make([]stageStat, 0)
	for rows.Next() {
		var s stageStat
		if err := rows.Scan(&s.StageID, &s.Plays, &s.UniqueArtists, &s.AvgBPM); err != nil {
			continue
		}
		results = append(results, s)
	}

	WriteJSON(w, http.StatusOK, results)
}

// HandleStatsBPM serves GET /api/stats/bpm?stage=
func (h *StatsHandlers) HandleStatsBPM(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	query := `SELECT bpm, COUNT(*) FROM library_tracks WHERE bpm IS NOT NULL AND bpm > 0`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += " AND stage_id = $1"
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
			continue
		}
		bins = append(bins, b)
	}

	WriteJSON(w, http.StatusOK, bins)
}

// HandleStatsKeys serves GET /api/stats/keys?stage=
func (h *StatsHandlers) HandleStatsKeys(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	query := `SELECT camelot_key, COUNT(*) FROM library_tracks WHERE camelot_key IS NOT NULL`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += " AND stage_id = $1"
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
			continue
		}
		keys = append(keys, k)
	}

	WriteJSON(w, http.StatusOK, keys)
}

// HandleStatsDecades serves GET /api/stats/decades?stage=
func (h *StatsHandlers) HandleStatsDecades(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	query := `SELECT (year/10)*10 as decade, COUNT(*) FROM library_tracks WHERE year IS NOT NULL AND year > 0`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += " AND stage_id = $1"
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
			continue
		}
		decades = append(decades, d)
	}

	WriteJSON(w, http.StatusOK, decades)
}

// HandleStatsGenres serves GET /api/stats/genres?stage=
func (h *StatsHandlers) HandleStatsGenres(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	query := `SELECT genre, COUNT(*) FROM library_tracks WHERE genre IS NOT NULL AND genre != ''`
	args := []any{}
	if stage := r.URL.Query().Get("stage"); stage != "" {
		query += " AND stage_id = $1"
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
			continue
		}
		genres = append(genres, g)
	}

	WriteJSON(w, http.StatusOK, genres)
}

// HandleStatsDiscoveryVelocity serves GET /api/stats/discovery-velocity
// Returns tracks added per week for the last 52 weeks.
func (h *StatsHandlers) HandleStatsDiscoveryVelocity(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

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
			continue
		}
		weeks = append(weeks, wc)
	}

	WriteJSON(w, http.StatusOK, weeks)
}

// HandleStatsListeningHeatmap serves GET /api/stats/listening-heatmap?stage=
// Same as history heatmap but under /stats/ path.
func (h *StatsHandlers) HandleStatsListeningHeatmap(w http.ResponseWriter, r *http.Request) {
	hh := &HistoryHandlers{DB: h.DB}
	hh.HandleHistoryHeatmap(w, r)
}
