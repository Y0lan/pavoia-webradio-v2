package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ArtistSummary is the list view of an artist.
type ArtistSummary struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Country    *string  `json:"country"`
	ImageURL   *string  `json:"image_url"`
	Tags       []string `json:"tags"`
	TrackCount int      `json:"track_count"`
}

// ArtistDetail is the full artist profile.
type ArtistDetail struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	MBID             *string    `json:"mbid"`
	Country          *string    `json:"country"`
	Bio              *string    `json:"bio"`
	ImageURL         *string    `json:"image_url"`
	BannerURL        *string    `json:"banner_url"`
	Tags             []string   `json:"tags"`
	ExternalLinks    any        `json:"external_links"`
	EnrichedAt       *time.Time `json:"enriched_at"`
	EnrichmentSource *string    `json:"enrichment_source"`
	TrackCount       int        `json:"track_count"`
	PlayCount        int        `json:"play_count"`
}

// ArtistTrack is a track belonging to an artist.
type ArtistTrack struct {
	ID       int64     `json:"id"`
	Title    string    `json:"title"`
	Album    string    `json:"album"`
	StageID  string    `json:"stage_id"`
	Genre    string    `json:"genre"`
	BPM      *int      `json:"bpm"`
	Year     *int      `json:"year"`
	AddedAt  time.Time `json:"added_at"`
}

// ArtistsHandlers holds dependencies for artist API handlers.
type ArtistsHandlers struct {
	DB         *pgxpool.Pool
	AdminToken string
}

// HandleArtistsList serves GET /api/artists
func (h *ArtistsHandlers) HandleArtistsList(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	pg := ParsePagination(r)
	q := r.URL.Query()

	where := "WHERE 1=1"
	args := []any{}
	argN := 0
	nextArg := func() string { argN++; return fmt.Sprintf("$%d", argN) }

	if country := q.Get("country"); country != "" {
		where += " AND a.country = " + nextArg()
		args = append(args, country)
	}
	if search := q.Get("search"); search != "" {
		where += " AND a.name ILIKE " + nextArg()
		args = append(args, "%"+search+"%")
	}

	// Count
	var total int
	countQ := "SELECT COUNT(*) FROM artists a " + where
	h.DB.QueryRow(r.Context(), countQ, args...).Scan(&total)

	// Sort
	orderBy := "ORDER BY track_count DESC"
	switch q.Get("sort") {
	case "name":
		orderBy = "ORDER BY a.name"
	case "plays":
		orderBy = "ORDER BY play_count DESC"
	}

	// Query with track counts from library_tracks
	dataQ := fmt.Sprintf(`
		SELECT a.id, a.name, a.country, a.image_url, a.tags,
			(SELECT COUNT(*) FROM library_tracks lt WHERE lt.artist_id = a.id) as track_count
		FROM artists a
		%s %s
		LIMIT %d OFFSET %d
	`, where, orderBy, pg.PerPage, pg.Offset)

	rows, err := h.DB.Query(r.Context(), dataQ, args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	artists := make([]ArtistSummary, 0)
	for rows.Next() {
		var a ArtistSummary
		if err := rows.Scan(&a.ID, &a.Name, &a.Country, &a.ImageURL, &a.Tags, &a.TrackCount); err != nil {
			continue
		}
		if a.Tags == nil {
			a.Tags = []string{}
		}
		artists = append(artists, a)
	}

	WritePaged(w, artists, pg, total)
}

// HandleArtistDetail serves GET /api/artists/{id}
func (h *ArtistsHandlers) HandleArtistDetail(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var a ArtistDetail
	err = h.DB.QueryRow(r.Context(), `
		SELECT a.id, a.name, a.mbid, a.country, a.bio, a.image_url, a.banner_url,
			a.tags, a.external_links, a.enriched_at, a.enrichment_source,
			(SELECT COUNT(*) FROM library_tracks lt WHERE lt.artist_id = a.id) as track_count,
			(SELECT COUNT(*) FROM track_plays tp WHERE lower(tp.artist) = lower(a.name)) as play_count
		FROM artists a WHERE a.id = $1
	`, id).Scan(
		&a.ID, &a.Name, &a.MBID, &a.Country, &a.Bio, &a.ImageURL, &a.BannerURL,
		&a.Tags, &a.ExternalLinks, &a.EnrichedAt, &a.EnrichmentSource,
		&a.TrackCount, &a.PlayCount,
	)
	if err != nil {
		WriteError(w, http.StatusNotFound, "artist not found")
		return
	}
	if a.Tags == nil {
		a.Tags = []string{}
	}

	WriteJSON(w, http.StatusOK, a)
}

// HandleArtistTracks serves GET /api/artists/{id}/tracks
func (h *ArtistsHandlers) HandleArtistTracks(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	pg := ParsePagination(r)

	var total int
	h.DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM library_tracks WHERE artist_id = $1", id).Scan(&total)

	rows, queryErr := h.DB.Query(r.Context(), `
		SELECT id, title, COALESCE(album, ''), COALESCE(stage_id, ''),
			COALESCE(genre, ''), bpm, year, added_at
		FROM library_tracks WHERE artist_id = $1
		ORDER BY added_at DESC
		LIMIT $2 OFFSET $3
	`, id, pg.PerPage, pg.Offset)
	if queryErr != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	tracks := make([]ArtistTrack, 0)
	for rows.Next() {
		var t ArtistTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Album, &t.StageID, &t.Genre, &t.BPM, &t.Year, &t.AddedAt); err != nil {
			continue
		}
		tracks = append(tracks, t)
	}

	WritePaged(w, tracks, pg, total)
}

// HandleArtistSimilar serves GET /api/artists/{id}/similar
func (h *ArtistsHandlers) HandleArtistSimilar(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		WriteError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	inLibrary := r.URL.Query().Get("in_library") == "true"

	query := `
		SELECT a.id, a.name, a.country, a.image_url, ar.weight
		FROM artist_relations ar
		JOIN artists a ON (
			CASE WHEN ar.artist_id_a = $1 THEN ar.artist_id_b ELSE ar.artist_id_a END = a.id
		)
		WHERE (ar.artist_id_a = $1 OR ar.artist_id_b = $1)
	`
	if inLibrary {
		query += " AND EXISTS (SELECT 1 FROM library_tracks lt WHERE lt.artist_id = a.id)"
	}
	query += " ORDER BY ar.weight DESC LIMIT 50"

	rows, queryErr := h.DB.Query(r.Context(), query, id)
	if queryErr != nil {
		WriteError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type similar struct {
		ID       int64   `json:"id"`
		Name     string  `json:"name"`
		Country  *string `json:"country"`
		ImageURL *string `json:"image_url"`
		Weight   float64 `json:"weight"`
	}
	results := make([]similar, 0)
	for rows.Next() {
		var s similar
		if err := rows.Scan(&s.ID, &s.Name, &s.Country, &s.ImageURL, &s.Weight); err != nil {
			continue
		}
		results = append(results, s)
	}

	WriteJSON(w, http.StatusOK, results)
}
