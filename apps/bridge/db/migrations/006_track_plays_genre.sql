-- P1 — snapshot the track's genre at play time so /api/history?genre= doesn't
-- rewrite the past when a track is retagged.
--
-- Before this, the history genre filter joined against library_tracks.genre,
-- which is mutable (disk importer upserts overwrite it on every Python sync).
-- If "Vertigo" was tagged "melodic techno" when it played at 3am and someone
-- retagged it to "deep house" at 9am, the 3am play would then match genre=
-- deep house — corrupting historical analytics.
--
-- New column is NULLable so existing rows (plays logged before the library
-- was populated) stay filterable by artist/title but aren't accidentally
-- matched under any genre filter.

ALTER TABLE track_plays ADD COLUMN IF NOT EXISTS genre TEXT;

CREATE INDEX IF NOT EXISTS idx_plays_genre ON track_plays (genre) WHERE genre IS NOT NULL;
