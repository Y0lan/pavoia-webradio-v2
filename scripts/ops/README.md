# Ops scripts

Phase F (ops hardening) lives here. Each script is self-contained, env-driven,
and safe to run from cron.

## Install on Whatbox

```bash
# One-time: rsync the scripts
rsync -avz scripts/ops/ yolan@orange.whatbox.ca:~/gaende-radio/scripts/ops/
ssh yolan@orange.whatbox.ca 'chmod +x ~/gaende-radio/scripts/ops/*.sh'

# Add to crontab (crontab -e):
0 3 * * * ~/gaende-radio/scripts/ops/pg-backup.sh >> ~/files/backups/pg-backup.log 2>&1
* * * * * ~/gaende-radio/scripts/ops/bridge-watchdog.sh >> ~/files/backups/watchdog.log 2>&1
```

## pg-backup.sh

Nightly pg_dump → `~/files/backups/pg/gaende-<timestamp>.sql.gz`. Rotates at 7 days.
Writes a heartbeat row to `pg_backup_log` so `/health.checks.pg_backup` surfaces
a "stale" signal if backups stop running.

Environment overrides: `BACKUP_DIR`, `RETENTION_DAYS`, `POSTGRES_PASSWORD`,
`PGHOST`, `PGPORT`, `PGUSER`, `PGDB`.

## bridge-watchdog.sh

Every minute, probes `http://127.0.0.1:3001/health`. Restarts the bridge if 3
consecutive probes fail. State in `~/files/backups/watchdog-state` so the
counter survives cron re-exec. Transient single-tick blips are ignored.

Environment overrides: `HEALTH_URL`, `STATE_FILE`, `BRIDGE_DIR`, `FAIL_THRESHOLD`,
`ENV_FILE`.
