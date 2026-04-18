-- Phase C6 — move library_tracks.stage_id (scalar) to a track_stages join table.
--
-- Why: the 9 stages aren't 1:1 with Plex playlists. etage-0 aggregates
-- (ETAGE 0, Etage 0 - FAST DARK MINIMAL) and fontanna-laputa aggregates
-- (FONTANNA, MINIMAL). The scalar stage_id silently collapsed multi-playlist
-- memberships via a "last-synced wins" race in plex/sync.go. This join table
-- lets a single file_path belong to N stages correctly.
--
-- Data migration: library_tracks is currently empty (sync has never successfully
-- run to populate it — see docs/ARCHITECTURE.md §5). No backfill needed. If
-- rows existed, we'd INSERT SELECT id, stage_id FROM library_tracks WHERE
-- stage_id IS NOT NULL before dropping the column.

CREATE TABLE IF NOT EXISTS track_stages (
    file_path TEXT NOT NULL REFERENCES library_tracks(file_path) ON DELETE CASCADE,
    stage_id  TEXT NOT NULL,
    PRIMARY KEY (file_path, stage_id)
);

CREATE INDEX IF NOT EXISTS idx_track_stages_stage ON track_stages (stage_id);

DROP INDEX IF EXISTS idx_tracks_stage;
ALTER TABLE library_tracks DROP COLUMN IF EXISTS stage_id;
