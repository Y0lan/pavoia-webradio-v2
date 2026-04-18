-- Phase D — soft-delete semantics for library_tracks.
--
-- Before this migration the disk-importer has no way to mark a track that
-- used to be in a Plex playlist but isn't anymore. Without soft-delete,
-- either (a) we hard-delete and /digging loses retrospective history, or
-- (b) we never delete and /api/artists/search/stats accumulate stale rows
-- forever. soft-delete keeps added_at intact for /digging while letting
-- active-only queries filter with `WHERE deleted_at IS NULL`.

ALTER TABLE library_tracks ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Partial index — active-only queries are the hot path.
CREATE INDEX IF NOT EXISTS idx_tracks_active
    ON library_tracks (id) WHERE deleted_at IS NULL;
