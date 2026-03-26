package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DiggingDay represents a single day in the digging calendar heatmap.
type DiggingDay struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	Count int    `json:"count"` // tracks added
}

// DiggingTrack represents a track added on a specific date.
type DiggingTrack struct {
	ID      int64     `json:"id"`
	Title   string    `json:"title"`
	Artist  string    `json:"artist"`
	Album   string    `json:"album"`
	StageID string    `json:"stage_id"`
	Genre   string    `json:"genre"`
	AddedAt time.Time `json:"added_at"`
}

// DiggingStreaks holds streak calculations.
type DiggingStreaks struct {
	Current  int    `json:"current"`        // consecutive days ending today
	Longest  int    `json:"longest"`        // all-time longest
	BestWeek int    `json:"best_week"`      // max tracks in any 7-day window
	LastDate string `json:"last_dig_date"`  // most recent day with additions
}

// DiggingHandlers holds dependencies for digging API handlers.
type DiggingHandlers struct {
	DB *pgxpool.Pool
}

// HandleDiggingCalendar serves GET /api/digging/calendar?year=
// Returns addition counts per day from library_tracks.added_at.
func (h *DiggingHandlers) HandleDiggingCalendar(w http.ResponseWriter, r *http.Request) {
	year := QueryInt(r, "year", time.Now().Year())
	colorBy := r.URL.Query().Get("color_by") // "", "stage", "genre"

	var query string
	switch colorBy {
	case "stage":
		query = `
			SELECT DATE(added_at)::text, stage_id, COUNT(*)
			FROM library_tracks
			WHERE EXTRACT(YEAR FROM added_at) = $1
			GROUP BY DATE(added_at), stage_id
			ORDER BY DATE(added_at)
		`
	default:
		query = `
			SELECT DATE(added_at)::text, COUNT(*)
			FROM library_tracks
			WHERE EXTRACT(YEAR FROM added_at) = $1
			GROUP BY DATE(added_at)
			ORDER BY DATE(added_at)
		`
	}

	rows, err := h.DB.Query(r.Context(), query, year)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	if colorBy == "stage" {
		type stageDay struct {
			Date    string `json:"date"`
			StageID string `json:"stage_id"`
			Count   int    `json:"count"`
		}
		days := make([]stageDay, 0)
		for rows.Next() {
			var d stageDay
			if err := rows.Scan(&d.Date, &d.StageID, &d.Count); err != nil {
				WriteError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			days = append(days, d)
		}
		if err := rows.Err(); err != nil {
			WriteError(w, http.StatusInternalServerError, "rows iteration failed")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"year": year, "color_by": "stage", "days": days})
		return
	}

	days := make([]DiggingDay, 0)
	for rows.Next() {
		var d DiggingDay
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			WriteError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		days = append(days, d)
	}
	if err := rows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "rows iteration failed")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"year": year, "days": days})
}

// HandleDiggingDate serves GET /api/digging/calendar/{date}
// Returns tracks added on a specific date.
func (h *DiggingHandlers) HandleDiggingDate(w http.ResponseWriter, r *http.Request) {
	dateStr := r.PathValue("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
		return
	}

	nextDay := date.Add(24 * time.Hour)
	rows, queryErr := h.DB.Query(r.Context(), `
		SELECT id, title, artist, COALESCE(album, ''), COALESCE(stage_id, ''), COALESCE(genre, ''), added_at
		FROM library_tracks
		WHERE added_at >= $1 AND added_at < $2
		ORDER BY added_at
	`, date, nextDay)
	if queryErr != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	tracks := make([]DiggingTrack, 0)
	for rows.Next() {
		var t DiggingTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Album, &t.StageID, &t.Genre, &t.AddedAt); err != nil {
			WriteError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		tracks = append(tracks, t)
	}
	if err := rows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "rows iteration failed")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"date":   dateStr,
		"count":  len(tracks),
		"tracks": tracks,
	})
}

// HandleDiggingStreaks serves GET /api/digging/streaks
// Computes current streak, longest streak, and best week.
func (h *DiggingHandlers) HandleDiggingStreaks(w http.ResponseWriter, r *http.Request) {
	// Get all distinct dates with additions
	rows, err := h.DB.Query(r.Context(), `
		SELECT DISTINCT DATE(added_at)::text
		FROM library_tracks
		ORDER BY 1
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var ds string
		if err := rows.Scan(&ds); err != nil {
			WriteError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		if t, err := time.Parse("2006-01-02", ds); err == nil {
			dates = append(dates, t)
		}
	}
	if err := rows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "rows iteration failed")
		return
	}

	streaks := computeStreaks(dates)

	// Best week: get per-day counts and compute max 7-day window
	countRows, err := h.DB.Query(r.Context(), `
		SELECT DATE(added_at)::text, COUNT(*)
		FROM library_tracks
		GROUP BY DATE(added_at)
		ORDER BY DATE(added_at)
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer countRows.Close()

	var dayCounts []dayCount
	for countRows.Next() {
		var ds string
		var c int
		if err := countRows.Scan(&ds, &c); err != nil {
			continue
		}
		if t, err := time.Parse("2006-01-02", ds); err == nil {
			dayCounts = append(dayCounts, dayCount{date: t, count: c})
		}
	}
	if err := countRows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "rows iteration failed")
		return
	}
	streaks.BestWeek = computeBestWeek(dayCounts)

	WriteJSON(w, http.StatusOK, streaks)
}

// HandleDiggingPatterns serves GET /api/digging/patterns
// Returns day-of-week and hour distributions for additions.
func (h *DiggingHandlers) HandleDiggingPatterns(w http.ResponseWriter, r *http.Request) {
	// Day of week distribution
	dowRows, err := h.DB.Query(r.Context(), `
		SELECT EXTRACT(DOW FROM added_at)::int, COUNT(*)
		FROM library_tracks
		GROUP BY 1 ORDER BY 1
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer dowRows.Close()

	dayOfWeek := make(map[int]int)
	for dowRows.Next() {
		var dow, count int
		if err := dowRows.Scan(&dow, &count); err != nil {
			continue
		}
		dayOfWeek[dow] = count
	}
	if err := dowRows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "rows iteration failed")
		return
	}

	// Hour distribution
	hourRows, err := h.DB.Query(r.Context(), `
		SELECT EXTRACT(HOUR FROM added_at)::int, COUNT(*)
		FROM library_tracks
		GROUP BY 1 ORDER BY 1
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer hourRows.Close()

	hourOfDay := make(map[int]int)
	for hourRows.Next() {
		var hour, count int
		if err := hourRows.Scan(&hour, &count); err != nil {
			continue
		}
		hourOfDay[hour] = count
	}
	if err := hourRows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "rows iteration failed")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"day_of_week": dayOfWeek,
		"hour_of_day": hourOfDay,
	})
}

// computeStreaks calculates current and longest consecutive-day streaks.
func computeStreaks(dates []time.Time) DiggingStreaks {
	if len(dates) == 0 {
		return DiggingStreaks{}
	}

	today := time.Now().Truncate(24 * time.Hour)
	s := DiggingStreaks{
		LastDate: dates[len(dates)-1].Format("2006-01-02"),
	}

	// Find longest streak
	longest, current := 1, 1
	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1]).Hours() / 24
		if diff <= 1.08 { // consecutive days (26 hours for timezone drift)
			current++
		} else {
			current = 1
		}
		if current > longest {
			longest = current
		}
	}
	s.Longest = longest

	// Current streak (counting back from today)
	s.Current = 0
	for i := len(dates) - 1; i >= 0; i-- {
		dayDiff := today.Sub(dates[i]).Hours() / 24
		if dayDiff < 0 {
			dayDiff = 0
		}
		expectedDiff := float64(len(dates) - 1 - i)
		if dayDiff-expectedDiff > 1.08 {
			break
		}
		s.Current++
	}

	return s
}

type dayCount struct {
	date  time.Time
	count int
}

// computeBestWeek finds the maximum sum of tracks in any 7-day sliding window.
func computeBestWeek(dayCounts []dayCount) int {
	if len(dayCounts) == 0 {
		return 0
	}

	best := 0
	for i := range dayCounts {
		sum := 0
		for j := i; j < len(dayCounts); j++ {
			if dayCounts[j].date.Sub(dayCounts[i].date).Hours() > 7*24 {
				break
			}
			sum += dayCounts[j].count
		}
		if sum > best {
			best = sum
		}
	}
	return best
}
