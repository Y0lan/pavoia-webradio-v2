-- Phase F — heartbeat table for the nightly pg_dump job.
--
-- /health queries this table for its backup freshness probe: if the most
-- recent `ok` row is older than 30h, the check flips to "stale" and the
-- top-level status to "degraded". That's the only way an external monitor
-- can detect that backups have silently stopped running.

CREATE TABLE IF NOT EXISTS pg_backup_log (
    id          BIGSERIAL PRIMARY KEY,
    status      TEXT NOT NULL,    -- 'ok' or 'failed'
    rows_dumped BIGINT,
    size_bytes  BIGINT,
    backup_path TEXT,
    error       TEXT,
    written_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backup_log_written ON pg_backup_log (written_at DESC);
