package plex

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StageMapping maps a Plex playlist name to a stage ID.
type StageMapping struct {
	PlaylistName string
	StageID      string
}

// SyncWorker periodically syncs Plex playlists to the database.
type SyncWorker struct {
	client   *Client
	db       *pgxpool.Pool
	mappings []StageMapping
	interval time.Duration
}

// SyncResult holds the result of a single sync run.
type SyncResult struct {
	StageID      string
	Playlist     string
	TracksAdded  int
	TracksTotal  int
	Error        error
}

// NewSyncWorker creates a sync worker.
func NewSyncWorker(client *Client, db *pgxpool.Pool, mappings []StageMapping, interval time.Duration) *SyncWorker {
	return &SyncWorker{
		client:   client,
		db:       db,
		mappings: mappings,
		interval: interval,
	}
}

// Start runs the sync loop in a goroutine.
func (w *SyncWorker) Start(ctx context.Context) {
	go func() {
		// Initial sync on startup
		w.SyncAll(ctx)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.SyncAll(ctx)
			}
		}
	}()
}

// SyncAll syncs all mapped playlists.
func (w *SyncWorker) SyncAll(ctx context.Context) []SyncResult {
	slog.Info("plex sync starting")

	playlists, err := w.client.Playlists()
	if err != nil {
		slog.Error("plex sync failed: cannot fetch playlists", "error", err)
		return nil
	}

	var results []SyncResult

	for _, mapping := range w.mappings {
		// Find the playlist by name
		var playlist *Playlist
		for _, p := range playlists {
			if strings.EqualFold(p.Title, mapping.PlaylistName) {
				playlist = &p
				break
			}
		}

		if playlist == nil {
			slog.Warn("plex playlist not found", "name", mapping.PlaylistName, "stage", mapping.StageID)
			results = append(results, SyncResult{
				StageID:  mapping.StageID,
				Playlist: mapping.PlaylistName,
				Error:    fmt.Errorf("playlist %q not found in Plex", mapping.PlaylistName),
			})
			continue
		}

		result := w.syncPlaylist(ctx, playlist, mapping.StageID)
		results = append(results, result)
	}

	slog.Info("plex sync complete", "playlists", len(results))
	return results
}

func (w *SyncWorker) syncPlaylist(ctx context.Context, playlist *Playlist, stageID string) SyncResult {
	result := SyncResult{
		StageID:  stageID,
		Playlist: playlist.Title,
	}

	tracks, err := w.client.PlaylistTracks(playlist.RatingKey)
	if err != nil {
		result.Error = err
		slog.Error("plex sync: cannot fetch tracks", "playlist", playlist.Title, "error", err)
		w.logSync(ctx, stageID, playlist.Title, 0, 0, 0, err.Error())
		return result
	}

	result.TracksTotal = len(tracks)
	added := 0

	for _, track := range tracks {
		if track.FilePath == "" || track.Title == "" {
			continue
		}

		// Use Plex's addedAt as the track's added_at timestamp (critical for digging calendar)
		addedAt := time.Unix(track.AddedAt, 0)
		if track.AddedAt == 0 {
			addedAt = time.Now()
		}

		// Upsert: insert if new, update metadata if exists (catches re-tagged files in Plex).
		// Note: a track can only belong to one stage. If the same file is in multiple
		// Plex playlists, the last-synced playlist wins the stage_id assignment.
		tag, err := w.db.Exec(ctx, `
			INSERT INTO library_tracks (file_path, title, artist, album, genre, year, duration_sec, file_format, stage_id, plex_rating_key, plex_added_at, added_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
			ON CONFLICT (file_path) DO UPDATE SET
				title = EXCLUDED.title,
				artist = EXCLUDED.artist,
				album = EXCLUDED.album,
				genre = EXCLUDED.genre,
				year = EXCLUDED.year,
				duration_sec = EXCLUDED.duration_sec,
				file_format = EXCLUDED.file_format,
				stage_id = EXCLUDED.stage_id,
				plex_rating_key = EXCLUDED.plex_rating_key
		`,
			track.FilePath,
			track.Title,
			track.Artist,
			track.Album,
			track.Genre,
			nullIfZero(track.Year),
			track.Duration/1000, // Plex sends milliseconds
			track.Format,
			stageID,
			track.RatingKey,
			addedAt,
		)

		if err != nil {
			slog.Warn("plex sync: insert track failed", "file", track.FilePath, "error", err)
			continue
		}

		if tag.RowsAffected() > 0 {
			added++
		}
	}

	result.TracksAdded = added

	if added > 0 {
		slog.Info("plex sync: tracks added", "stage", stageID, "playlist", playlist.Title, "added", added, "total", len(tracks))
	}

	w.logSync(ctx, stageID, playlist.Title, added, 0, len(tracks), "")
	return result
}

func (w *SyncWorker) logSync(ctx context.Context, stageID, playlistName string, added, removed, total int, errMsg string) {
	_, err := w.db.Exec(ctx, `
		INSERT INTO plex_sync_log (playlist_name, stage_id, tracks_added, tracks_removed, total_tracks, error)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
	`, playlistName, stageID, added, removed, total, errMsg)
	if err != nil {
		slog.Warn("plex sync: log entry failed", "error", err)
	}
}

func nullIfZero(v int) any {
	if v == 0 {
		return nil
	}
	return v
}
