package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	Pool *pgxpool.Pool
}

// Connect creates a connection pool to Postgres.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Migrate runs all SQL migration files in order, tracking which have already been applied.
func (d *DB) Migrate(ctx context.Context) error {
	// Create tracking table if it doesn't exist
	if _, err := d.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if already applied
		var count int
		if err := d.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE filename = $1", entry.Name()).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if count > 0 {
			continue // already applied
		}

		sql, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		slog.Info("running migration", "file", entry.Name())
		if _, err := d.Pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
		}

		// Record as applied
		if _, err := d.Pool.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1)", entry.Name()); err != nil {
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Close shuts down the connection pool.
func (d *DB) Close() {
	d.Pool.Close()
}

// Healthy checks if the database is reachable.
func (d *DB) Healthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return d.Pool.Ping(ctx) == nil
}
