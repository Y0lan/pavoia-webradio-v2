-- Phase D follow-up (F-D1.3) — drop the useless partial index that 003 added
-- and replace with one that actually helps the active-only queries.
--
-- The old idx_tracks_active covered `id`, but no query path probes by `id`;
-- library_tracks reads filter by artist_id, file_path, genre, year, etc.
-- The new partial index lands on artist_id since /api/artists/<id>/tracks
-- is the hottest active-only query once the disk importer is populated.

DROP INDEX IF EXISTS idx_tracks_active;

CREATE INDEX IF NOT EXISTS idx_tracks_active_artist
    ON library_tracks (artist_id) WHERE deleted_at IS NULL;
