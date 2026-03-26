package enrichment

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Worker periodically enriches artists from Last.fm and MusicBrainz.
type Worker struct {
	db       *pgxpool.Pool
	lastfm   *LastFMClient
	mb       *MBClient
	interval time.Duration
	batchSize int

	lastfmCB *CircuitBreaker
	mbCB     *CircuitBreaker
}

// NewWorker creates an enrichment worker.
func NewWorker(db *pgxpool.Pool, lastfmKey string, interval time.Duration) *Worker {
	return &Worker{
		db:        db,
		lastfm:    NewLastFMClient(lastfmKey),
		mb:        NewMBClient(),
		interval:  interval,
		batchSize: 10,
		lastfmCB:  NewCircuitBreaker(5, time.Minute, 5*time.Minute),
		mbCB:      NewCircuitBreaker(5, time.Minute, 5*time.Minute),
	}
}

// Start launches the enrichment worker in a background goroutine.
func (w *Worker) Start(ctx context.Context) {
	go func() {
		// Initial run after short delay
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}

		w.enrichBatch(ctx)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.enrichBatch(ctx)
			}
		}
	}()
}

// EnrichArtist enriches a single artist by name. Used for the admin force-enrich endpoint.
func (w *Worker) EnrichArtist(ctx context.Context, artistID int64) error {
	var name string
	err := w.db.QueryRow(ctx, "SELECT name FROM artists WHERE id = $1", artistID).Scan(&name)
	if err != nil {
		return err
	}

	return w.enrichOne(ctx, artistID, name)
}

// enrichBatch finds unenriched artists and enriches them.
func (w *Worker) enrichBatch(ctx context.Context) {
	rows, err := w.db.Query(ctx, `
		SELECT id, name FROM artists
		WHERE enriched_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`, w.batchSize)
	if err != nil {
		slog.Warn("enrichment: failed to query unenriched artists", "error", err)
		return
	}
	defer rows.Close()

	type artist struct {
		id   int64
		name string
	}
	var batch []artist
	for rows.Next() {
		var a artist
		if err := rows.Scan(&a.id, &a.name); err != nil {
			continue
		}
		batch = append(batch, a)
	}

	if len(batch) == 0 {
		return
	}

	slog.Info("enrichment: starting batch", "count", len(batch))

	for _, a := range batch {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.enrichOne(ctx, a.id, a.name); err != nil {
			slog.Warn("enrichment: failed", "artist", a.name, "error", err)
		}

		// Rate limit: 1 request per second (MusicBrainz requirement)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}

	slog.Info("enrichment: batch complete", "count", len(batch))
}

// enrichOne enriches a single artist from both sources.
func (w *Worker) enrichOne(ctx context.Context, artistID int64, name string) error {
	var sources []string
	var bio, imageURL, mbid, country *string
	var tags []string

	// Last.fm enrichment
	if w.lastfmCB.Allow() {
		info, err := w.lastfm.GetArtistInfo(ctx, name)
		if err != nil {
			w.lastfmCB.RecordFailure()
			slog.Debug("enrichment: lastfm failed", "artist", name, "error", err)
		} else {
			w.lastfmCB.RecordSuccess()
			sources = append(sources, "lastfm")

			if info.Bio != "" {
				bio = &info.Bio
			}
			if info.Image != "" {
				imageURL = &info.Image
			}
			if info.MBID != "" {
				mbid = &info.MBID
			}
			tags = info.Tags
		}

		// Get similar artists (only if Last.fm succeeded)
		if err == nil {
			w.enrichSimilar(ctx, artistID, name)
		}
	}

	// MusicBrainz enrichment
	if w.mbCB.Allow() {
		var mbArtist *MBArtist
		var err error

		// Prefer MBID lookup (from Last.fm) over name search
		if mbid != nil && *mbid != "" {
			mbArtist, err = w.mb.LookupArtist(ctx, *mbid)
		} else {
			mbArtist, err = w.mb.SearchArtist(ctx, name)
		}

		if err != nil {
			w.mbCB.RecordFailure()
			slog.Debug("enrichment: musicbrainz failed", "artist", name, "error", err)
		} else {
			w.mbCB.RecordSuccess()
			sources = append(sources, "musicbrainz")

			// MB has priority for country and MBID
			if mbArtist.Country != "" {
				country = &mbArtist.Country
			}
			if mbArtist.MBID != "" {
				mbid = &mbArtist.MBID
			}
		}
	}

	if len(sources) == 0 {
		// Mark as attempted even if both sources failed (prevents infinite retry)
		_, err := w.db.Exec(ctx, `
			UPDATE artists SET enriched_at = now(), enrichment_source = 'failed'
			WHERE id = $1
		`, artistID)
		return err
	}

	// Update the artist record
	source := strings.Join(sources, "+")
	_, err := w.db.Exec(ctx, `
		UPDATE artists SET
			bio = COALESCE($2, bio),
			image_url = COALESCE($3, image_url),
			mbid = COALESCE($4, mbid),
			country = COALESCE($5, country),
			tags = CASE WHEN $6::text[] IS NOT NULL AND array_length($6::text[], 1) > 0 THEN $6 ELSE tags END,
			enriched_at = now(),
			enrichment_source = $7,
			updated_at = now()
		WHERE id = $1
	`, artistID, bio, imageURL, mbid, country, tags, source)

	if err != nil {
		return err
	}

	slog.Info("enrichment: enriched", "artist", name, "sources", source)
	return nil
}

// enrichSimilar fetches and stores similar artist relationships.
func (w *Worker) enrichSimilar(ctx context.Context, artistID int64, name string) {
	if !w.lastfmCB.Allow() {
		return
	}

	similar, err := w.lastfm.GetSimilarArtists(ctx, name, 20)
	if err != nil {
		w.lastfmCB.RecordFailure()
		return
	}
	w.lastfmCB.RecordSuccess()

	for _, s := range similar {
		if s.Match < 0.1 {
			continue // skip very weak matches
		}

		// Find or create the similar artist
		var similarID int64
		err := w.db.QueryRow(ctx, `
			INSERT INTO artists (name) VALUES ($1)
			ON CONFLICT (lower(name)) DO UPDATE SET name = artists.name
			RETURNING id
		`, s.Name).Scan(&similarID)
		if err != nil {
			continue
		}

		// Insert the relation (ignore duplicates)
		aID, bID := artistID, similarID
		if aID > bID {
			aID, bID = bID, aID // normalize ordering
		}
		w.db.Exec(ctx, `
			INSERT INTO artist_relations (artist_id_a, artist_id_b, relation_type, weight, source)
			VALUES ($1, $2, 'similar', $3, 'lastfm')
			ON CONFLICT (artist_id_a, artist_id_b, relation_type) DO UPDATE SET weight = $3
		`, aID, bID, s.Match)
	}
}
