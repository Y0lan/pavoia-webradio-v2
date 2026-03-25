-- GAENDE Radio — Initial Schema
-- Tracks, plays, artists, relations, sync log

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Artists (enriched from Last.fm + MusicBrainz)
CREATE TABLE IF NOT EXISTS artists (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    mbid            TEXT,                    -- MusicBrainz ID
    country         TEXT,
    bio             TEXT,
    image_url       TEXT,
    banner_url      TEXT,
    tags            TEXT[],                  -- genre tags from enrichment
    external_links  JSONB DEFAULT '{}',      -- {spotify, soundcloud, bandcamp, etc.}
    enriched_at     TIMESTAMPTZ,
    enrichment_source TEXT,                  -- "lastfm", "musicbrainz", "lastfm+musicbrainz"
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_artists_name ON artists (lower(name));
CREATE INDEX IF NOT EXISTS idx_artists_trgm ON artists USING gin (name gin_trgm_ops);

-- Artist relations (for the similarity graph)
CREATE TABLE IF NOT EXISTS artist_relations (
    id              BIGSERIAL PRIMARY KEY,
    artist_id_a     BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    artist_id_b     BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    relation_type   TEXT NOT NULL DEFAULT 'similar',  -- similar, same_label, same_genre
    weight          REAL NOT NULL DEFAULT 0.5,        -- 0.0-1.0 similarity score
    source          TEXT NOT NULL DEFAULT 'lastfm',   -- lastfm, musicbrainz
    created_at      TIMESTAMPTZ DEFAULT now(),
    UNIQUE(artist_id_a, artist_id_b, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_artist_relations_a ON artist_relations (artist_id_a);
CREATE INDEX IF NOT EXISTS idx_artist_relations_b ON artist_relations (artist_id_b);

-- Library tracks (synced from Plex)
CREATE TABLE IF NOT EXISTS library_tracks (
    id              BIGSERIAL PRIMARY KEY,
    file_path       TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    artist          TEXT NOT NULL,
    album           TEXT,
    genre           TEXT,
    label           TEXT,
    year            SMALLINT,
    bpm             SMALLINT,
    camelot_key     TEXT,                    -- e.g. "8A", "11B"
    duration_sec    INT,
    file_format     TEXT,                    -- mp3, flac, etc.
    stage_id        TEXT,                    -- which stage/playlist this belongs to
    artist_id       BIGINT REFERENCES artists(id),
    plex_rating_key TEXT,                    -- Plex internal ID
    plex_added_at   TIMESTAMPTZ,            -- when Plex first saw this track
    added_at        TIMESTAMPTZ NOT NULL,    -- when added to the playlist (from Plex addedAt)
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tracks_artist ON library_tracks (lower(artist));
CREATE INDEX IF NOT EXISTS idx_tracks_stage ON library_tracks (stage_id);
CREATE INDEX IF NOT EXISTS idx_tracks_added ON library_tracks (added_at);
CREATE INDEX IF NOT EXISTS idx_tracks_genre ON library_tracks (genre);
CREATE INDEX IF NOT EXISTS idx_tracks_trgm_title ON library_tracks USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_tracks_trgm_artist ON library_tracks USING gin (artist gin_trgm_ops);

-- Track play history (logged by the bridge when MPD plays a track)
CREATE TABLE IF NOT EXISTS track_plays (
    id              BIGSERIAL PRIMARY KEY,
    track_id        BIGINT REFERENCES library_tracks(id),
    stage_id        TEXT NOT NULL,
    artist          TEXT NOT NULL,
    title           TEXT NOT NULL,
    album           TEXT,
    file_path       TEXT,
    played_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_sec    INT
);

CREATE INDEX IF NOT EXISTS idx_plays_stage ON track_plays (stage_id);
CREATE INDEX IF NOT EXISTS idx_plays_played ON track_plays (played_at);
CREATE INDEX IF NOT EXISTS idx_plays_artist ON track_plays (lower(artist));
CREATE INDEX IF NOT EXISTS idx_plays_track ON track_plays (track_id);

-- Plex sync log (tracks playlist diffs)
CREATE TABLE IF NOT EXISTS plex_sync_log (
    id              BIGSERIAL PRIMARY KEY,
    playlist_name   TEXT NOT NULL,
    stage_id        TEXT NOT NULL,
    tracks_added    INT NOT NULL DEFAULT 0,
    tracks_removed  INT NOT NULL DEFAULT 0,
    total_tracks    INT NOT NULL DEFAULT 0,
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    error           TEXT
);

-- Plex playlist snapshots (for diffing)
CREATE TABLE IF NOT EXISTS plex_playlist_snapshots (
    id              BIGSERIAL PRIMARY KEY,
    plex_playlist_id TEXT NOT NULL,
    stage_id        TEXT NOT NULL,
    track_keys      TEXT[] NOT NULL,          -- ordered list of track file paths
    snapshot_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_snapshots_stage ON plex_playlist_snapshots (stage_id);

-- Wrapped data (pre-computed year/month summaries)
CREATE TABLE IF NOT EXISTS wrapped_data (
    id              BIGSERIAL PRIMARY KEY,
    period          TEXT NOT NULL UNIQUE,     -- "2026", "2026-03"
    data            JSONB NOT NULL,
    generated_at    TIMESTAMPTZ DEFAULT now()
);

-- User preferences (key-value store for settings)
CREATE TABLE IF NOT EXISTS user_preferences (
    id    BIGSERIAL PRIMARY KEY,
    key   TEXT NOT NULL UNIQUE,
    value JSONB NOT NULL
);
