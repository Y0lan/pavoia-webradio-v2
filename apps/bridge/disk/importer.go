package disk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
)

// ErrPlaylistAborted is returned by SyncOnce when a playlist-level failure
// forced the whole run to abort. Callers (Start, admin force-sync) can
// distinguish this from "manifest not found" or DB connection errors.
var ErrPlaylistAborted = errors.New("disk sync aborted: playlist failed")

// Importer reads the Python sync's on-disk artifacts and upserts them into
// Postgres. One instance per bridge process; SyncOnce is safe to call
// concurrently with ongoing reads (all writes happen in a single transaction
// per invocation).
type Importer struct {
	db            *pgxpool.Pool
	webradioDir   string
	cfg           *config.Config

	// Playlist title (lower-case) → stage ID. Built once on construction.
	playlistStage map[string]string

	// Guards lastGeneration so overlapping SyncOnce calls don't re-process
	// the same generation. Takes the place of an advisory lock — in-process
	// single-writer, cross-process exclusion is handled on the Python side
	// via .sync.lock.
	mu             sync.Mutex
	lastGeneration string

	// lastSuccessAtNanos is read from /health under its own 3s deadline, so it
	// MUST NOT share a mutex with SyncOnce (which can run for multiple seconds
	// on a cold cache). atomic.Int64 of UnixNano lets readers bypass the mutex
	// entirely while still getting a coherent view.
	lastSuccessAtNanos atomic.Int64
}

// LastSuccess returns the timestamp of the most recent SyncOnce that either
// (a) processed a new generation cleanly or (b) short-circuited because the
// manifest was unchanged. Both states mean "we verified the disk artifacts
// are readable and consistent with what we already have", which is what
// /health cares about — a manifest that stops changing is a Python-side
// freshness concern, not a Go-side liveness concern.
//
// Returns zero if the importer hasn't completed a SyncOnce yet. Safe to call
// concurrently without blocking on the SyncOnce mutex.
func (im *Importer) LastSuccess() time.Time {
	n := im.lastSuccessAtNanos.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n).UTC()
}

// NewImporter constructs the importer. webradioDir is the root where Python
// writes sync_manifest.json and the per-playlist folders. cfg provides the
// frozen stage ↔ playlist mapping.
func NewImporter(db *pgxpool.Pool, cfg *config.Config, webradioDir string) *Importer {
	return &Importer{
		db:            db,
		webradioDir:   webradioDir,
		cfg:           cfg,
		playlistStage: cfg.PlaylistToStage(),
	}
}

// SyncResult summarizes what a single SyncOnce did.
type SyncResult struct {
	GenerationID    string
	ArtistsUpserted int
	TracksUpserted  int
	TracksDeleted   int
	StagesWritten   int
	StagesRemoved   int
	Errors          []string
	Skipped         bool // true when generation matched the last one we processed
}

// Start launches a goroutine that runs SyncOnce on startup and then every
// `interval`. Exits when ctx is cancelled.
func (im *Importer) Start(ctx context.Context, interval time.Duration) {
	go func() {
		if _, err := im.SyncOnce(ctx); err != nil {
			slog.Warn("disk sync: initial run failed", "error", err)
		}
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, err := im.SyncOnce(ctx); err != nil {
					slog.Warn("disk sync: periodic run failed", "error", err)
				}
			}
		}
	}()
}

// SyncOnce runs a single import cycle: verify manifest, upsert artists, walk
// playlist folders, soft-delete orphans. Idempotent and safe to retry.
func (im *Importer) SyncOnce(ctx context.Context) (*SyncResult, error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	result := &SyncResult{}

	manifest, err := LoadManifest(im.webradioDir)
	if err != nil {
		return result, err
	}
	if err := VerifyManifest(im.webradioDir, manifest); err != nil {
		return result, fmt.Errorf("manifest verify: %w", err)
	}
	result.GenerationID = manifest.GenerationID

	if manifest.GenerationID == im.lastGeneration {
		// Manifest verified + generation matches prior run → not just "skipped"
		// but "re-verified cleanly." Bump lastSuccessAt so /health doesn't flip
		// to "stale" during long idle windows between Python syncs (cron every
		// 6h, importer polls every 2 min → 180 expected skips between real work).
		im.lastSuccessAtNanos.Store(time.Now().UnixNano())
		result.Skipped = true
		return result, nil
	}

	slog.Info("disk sync: starting", "generation", manifest.GenerationID, "webradio", im.webradioDir)

	// 1. Artists — upsert all, then build a name→id lookup for track linking.
	artistsArtifact, err := LoadArtists(filepath.Join(im.webradioDir, manifest.Artifacts["artists"].Path))
	if err != nil {
		return result, err
	}
	artistIDs, n, err := im.upsertArtists(ctx, artistsArtifact.Artists)
	if err != nil {
		return result, fmt.Errorf("upsert artists: %w", err)
	}
	result.ArtistsUpserted = n

	// 2. Walk each mapped playlist folder, upsert library_tracks + track_stages.
	currentPaths := make(map[string]struct{})
	stageCurrentPaths := make(map[string]map[string]struct{}) // stage_id → file_path set
	for _, stage := range im.cfg.VisibleStages() {
		stageCurrentPaths[stage.ID] = make(map[string]struct{})
		for _, pl := range stage.Playlists {
			tracks, err := im.syncPlaylistFolder(ctx, pl, stage.ID, artistIDs)
			if err != nil {
				// Abort the whole sync cycle on any playlist-level failure. If we
				// continued with partial currentPaths, softDeleteOrphans would
				// wrongly mark valid tracks in the failed playlist as deleted.
				// Return a sentinel error so Start()'s failure log fires — silent
				// (result, nil) returns previously swallowed the abort signal.
				result.Errors = append(result.Errors, fmt.Sprintf("playlist %q: %v", pl, err))
				slog.Warn("disk sync: playlist failed, aborting run", "playlist", pl, "error", err)
				return result, fmt.Errorf("%w: %s: %v", ErrPlaylistAborted, pl, err)
			}
			for _, p := range tracks {
				currentPaths[p] = struct{}{}
				stageCurrentPaths[stage.ID][p] = struct{}{}
				result.TracksUpserted++
				result.StagesWritten++
			}
		}
	}

	// 3. Artist stubs — Python's artists.json only contains grandparentRatingKey
	//    entries (Plex's artist-scope records). Tracks whose Plex grandparent is a
	//    "Various Artists" compilation leave library_tracks.artist = "Björk" (say)
	//    without any corresponding artists row, so /api/search?type=artists and
	//    /api/artists track_count both miss them. Create a minimal stub row for
	//    every distinct library_tracks.artist we just wrote, then backfill
	//    artist_id so /api/artists/<id>/tracks works consistently.
	stubs, err := im.materializeArtistStubs(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("artist stubs: %v", err))
	} else {
		result.ArtistsUpserted += stubs
	}

	// 4. Soft-delete library_tracks that are no longer on disk in any mapped
	//    playlist. added_at is preserved so /digging keeps retrospective history.
	deleted, err := im.softDeleteOrphans(ctx, currentPaths)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("soft-delete: %v", err))
	} else {
		result.TracksDeleted = deleted
	}

	// 5. Prune stale stage memberships (track removed from a Plex playlist but
	//    still in another, so library_tracks stays alive but track_stages for
	//    the removed playlist must go).
	for stageID, paths := range stageCurrentPaths {
		removed, err := im.pruneStageMemberships(ctx, stageID, paths)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("prune %s: %v", stageID, err))
			continue
		}
		result.StagesRemoved += removed
	}

	// Advance the "done" marker only on a clean run. If any playlist threw an
	// error, stubs failed, soft-delete failed, or prune failed, we leave
	// lastGeneration unchanged so the next tick retries against this same
	// manifest instead of skipping it. Otherwise a transient DB blip could
	// permanently skip a generation's worth of changes.
	if len(result.Errors) == 0 {
		im.lastGeneration = manifest.GenerationID
		im.lastSuccessAtNanos.Store(time.Now().UnixNano())
	}

	slog.Info("disk sync: complete",
		"generation", manifest.GenerationID,
		"artists", result.ArtistsUpserted,
		"tracks_upserted", result.TracksUpserted,
		"tracks_deleted", result.TracksDeleted,
		"stages_written", result.StagesWritten,
		"stages_removed", result.StagesRemoved,
		"errors", len(result.Errors),
	)
	return result, nil
}

// upsertArtists inserts or updates artists rows keyed on lower(name), returning
// a map from lowercased name → artists.id so the track loop can link artist_id.
func (im *Importer) upsertArtists(ctx context.Context, records []ArtistRecord) (map[string]int64, int, error) {
	ids := make(map[string]int64, len(records))
	upserted := 0

	tx, err := im.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)

	for _, a := range records {
		if a.Name == "" {
			continue
		}
		var id int64
		// Tags policy: the enrichment worker (Last.fm + MusicBrainz) owns the
		// `tags` column once `enriched_at` is set. The disk importer writes
		// Plex-sourced tags only when the row is new OR has never been enriched,
		// so the 2-min disk tick doesn't stomp enrichment data.
		err := tx.QueryRow(ctx, `
			INSERT INTO artists (name, bio, image_url, tags, updated_at)
			VALUES ($1, $2, NULLIF($3, ''), $4, now())
			ON CONFLICT ((lower(name))) DO UPDATE SET
				bio        = COALESCE(EXCLUDED.bio, artists.bio),
				image_url  = COALESCE(EXCLUDED.image_url, artists.image_url),
				tags       = CASE
				               WHEN artists.enriched_at IS NULL THEN EXCLUDED.tags
				               ELSE artists.tags
				             END,
				updated_at = now()
			RETURNING id
		`, a.Name, a.Bio, a.ThumbPath, mergeTagSlices(a.Genres, a.Moods)).Scan(&id)
		if err != nil {
			slog.Warn("disk sync: artist upsert failed", "name", a.Name, "error", err)
			// Return err so caller fails the whole generation instead of silently
			// advancing lastGeneration past a partial upsert.
			return nil, 0, fmt.Errorf("artist %q: %w", a.Name, err)
		}
		ids[strings.ToLower(a.Name)] = id
		upserted++
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return ids, upserted, nil
}

// mergeTagSlices merges Plex genres + moods into a single de-duplicated tag list
// stored in artists.tags. Preserves input order, case-insensitive dedup.
func mergeTagSlices(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, x := range append(append([]string{}, a...), b...) {
		k := strings.ToLower(strings.TrimSpace(x))
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, x)
	}
	return out
}

// syncPlaylistFolder walks $WEBRADIO/<playlist>, reads each *.mp3.json sidecar,
// and upserts the track into library_tracks plus its stage membership into
// track_stages. Returns the canonical file_paths that are now present (for
// soft-delete bookkeeping). One transaction per playlist — a mid-flight crash
// loses at most one playlist's changes from this run, not the whole batch.
func (im *Importer) syncPlaylistFolder(
	ctx context.Context,
	playlistName string,
	stageID string,
	artistIDs map[string]int64,
) ([]string, error) {
	folder := filepath.Join(im.webradioDir, playlistName)
	entries, err := os.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			// Folder missing is expected until Python syncs that playlist for the
			// first time; log at INFO so it's visible but not alarming.
			slog.Info("disk sync: playlist folder absent, skipping", "playlist", playlistName, "folder", folder)
			return nil, nil
		}
		return nil, err
	}

	tx, err := im.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var kept []string
	for _, entry := range entries {
		name := entry.Name()
		if !isAudioFile(name) {
			continue
		}
		audioPath := filepath.Join(folder, name)
		sidecarPath := audioPath + ".json"

		sc, err := LoadSidecar(sidecarPath)
		if err != nil {
			// Missing or malformed sidecar — not a sync-level error; it just
			// means Python hasn't produced one for this file yet. Skipping one
			// sidecar in a playlist with hundreds is not worth a whole-run retry.
			// Logged at DEBUG to keep info-level output clean.
			slog.Debug("disk sync: sidecar load failed", "file", sidecarPath, "error", err)
			continue
		}

		// Canonical file_path: the Webradio-rooted absolute path. Matches the
		// format track_plays.file_path uses (main.go canonicalFilePath), so the
		// history genre-filter join actually joins.
		filePath := audioPath

		var artistID *int64
		if id, ok := artistIDs[strings.ToLower(sc.Track.Artist)]; ok {
			v := id
			artistID = &v
		}

		mtime := time.Time{}
		if fi, err := os.Stat(audioPath); err == nil {
			mtime = fi.ModTime()
		}
		addedAt := sc.AddedAt(mtime)

		_, err = tx.Exec(ctx, `
			INSERT INTO library_tracks (
				file_path, title, artist, album, genre, year, duration_sec,
				file_format, artist_id, plex_rating_key, plex_added_at, added_at,
				bpm, camelot_key, deleted_at
			)
			-- $11 is the one "when was this added" timestamp; we store it as both
			-- plex_added_at (provenance) and added_at (what /digging groups by).
			VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, $9, $10, $11, $11,
			        $12, NULLIF($13, ''), NULL)
			ON CONFLICT (file_path) DO UPDATE SET
				title           = EXCLUDED.title,
				artist          = EXCLUDED.artist,
				album           = EXCLUDED.album,
				genre           = EXCLUDED.genre,
				year            = EXCLUDED.year,
				duration_sec    = EXCLUDED.duration_sec,
				file_format     = EXCLUDED.file_format,
				artist_id       = EXCLUDED.artist_id,
				plex_rating_key = EXCLUDED.plex_rating_key,
				-- added_at stays on the original value so /digging history is stable;
				-- re-appearing a deleted track just clears deleted_at, not added_at.
				bpm             = COALESCE(EXCLUDED.bpm, library_tracks.bpm),
				camelot_key     = COALESCE(EXCLUDED.camelot_key, library_tracks.camelot_key),
				deleted_at      = NULL
		`,
			filePath,
			sc.Track.Title,
			sc.Track.Artist,
			sc.Track.Album,
			sc.PrimaryGenre(),
			sc.Track.Year,
			sc.DurationSeconds(),
			fileExt(name),
			artistID,
			RatingKeyString(sc.Metadata.PlexRatingKey),
			addedAt,
			sc.Track.BPM,
			sc.Track.CamelotKey,
		)
		if err != nil {
			// A library_tracks write failing is a real DB problem; abort the
			// playlist so the outer loop aborts the sync cycle, not a silent
			// per-file warn that advances the generation regardless.
			return nil, fmt.Errorf("library_tracks upsert %q: %w", filePath, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO track_stages (file_path, stage_id)
			VALUES ($1, $2)
			ON CONFLICT (file_path, stage_id) DO NOTHING
		`, filePath, stageID); err != nil {
			return nil, fmt.Errorf("track_stages upsert %q/%s: %w", filePath, stageID, err)
		}
		kept = append(kept, filePath)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return kept, nil
}

// softDeleteOrphans marks library_tracks rows whose file_path is no longer in
// any mapped-playlist folder as deleted. added_at is preserved so /digging
// retrospective history still works; queries already filter with
// `deleted_at IS NULL` via migration 003.
func (im *Importer) softDeleteOrphans(ctx context.Context, current map[string]struct{}) (int, error) {
	if len(current) == 0 {
		// Defensive: if current is empty, something went wrong upstream. Don't
		// delete everything — return 0 and let the next run reconverge.
		slog.Warn("disk sync: current set is empty, refusing to soft-delete all library_tracks")
		return 0, nil
	}
	paths := make([]string, 0, len(current))
	for p := range current {
		paths = append(paths, p)
	}
	tag, err := im.db.Exec(ctx, `
		UPDATE library_tracks
		SET deleted_at = now()
		WHERE deleted_at IS NULL AND file_path <> ALL($1::text[])
	`, paths)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// pruneStageMemberships removes track_stages rows for a stage whose file_path
// is no longer in that stage's current folder set. Does NOT delete rows for
// file_paths present in OTHER stages — a track that migrated from ambiance-safe
// to palac-dance loses its ambiance-safe row but keeps palac-dance.
func (im *Importer) pruneStageMemberships(ctx context.Context, stageID string, current map[string]struct{}) (int, error) {
	paths := make([]string, 0, len(current))
	for p := range current {
		paths = append(paths, p)
	}
	tag, err := im.db.Exec(ctx, `
		DELETE FROM track_stages
		WHERE stage_id = $1 AND file_path <> ALL($2::text[])
	`, stageID, paths)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// materializeArtistStubs creates minimal artist rows for any distinct
// library_tracks.artist that's missing one (typically tracks whose Plex
// grandparent is a "Various Artists" compilation, where Python's artists.json
// entry points at the compilation rather than the real track artist). Then
// backfills library_tracks.artist_id for rows it just created.
//
// Returns the number of new artist rows inserted (idempotent — ON CONFLICT skips).
func (im *Importer) materializeArtistStubs(ctx context.Context) (int, error) {
	// One round-trip instead of distinct+loop: Postgres handles the de-dup.
	tag, err := im.db.Exec(ctx, `
		INSERT INTO artists (name, updated_at)
		SELECT DISTINCT lt.artist, now()
		FROM library_tracks lt
		WHERE lt.deleted_at IS NULL
		  AND lt.artist <> ''
		  AND NOT EXISTS (SELECT 1 FROM artists a WHERE lower(a.name) = lower(lt.artist))
		ON CONFLICT ((lower(name))) DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	inserted := int(tag.RowsAffected())

	// Backfill artist_id for library_tracks that either never had one or whose
	// artist name changed. Uses lower(name) match to stay consistent with the
	// artists unique index.
	if _, err := im.db.Exec(ctx, `
		UPDATE library_tracks lt
		SET artist_id = a.id
		FROM artists a
		WHERE a.id IS DISTINCT FROM lt.artist_id
		  AND lower(a.name) = lower(lt.artist)
		  AND lt.deleted_at IS NULL
	`); err != nil {
		return inserted, err
	}
	return inserted, nil
}

// audioExtensions lists every suffix the Python sync emits into Webradio folders.
// Python may convert FLAC → MP3, but .flac/.opus/.m4a/.aac/.ogg still appear
// directly for formats it passes through.
var audioExtensions = []string{".mp3", ".flac", ".wav", ".opus", ".m4a", ".aac", ".ogg"}

func isAudioFile(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range audioExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func fileExt(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 && i < len(name)-1 {
		return strings.ToLower(name[i+1:])
	}
	return ""
}
