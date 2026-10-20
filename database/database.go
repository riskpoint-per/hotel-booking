package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql

var migrationFiles embed.FS

// Connect opens a connection pool and waits, with retries, until the database
// is reachable. Retrying matters under docker-compose, where the API container
// can start before Postgres accepts connections.
func Connect(ctx context.Context, dsn string, attempts int) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 1; i <= attempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return db, nil
		}
		slog.Warn("database not ready, retrying", "attempt", i, "error", err)
		select {
		case <-ctx.Done():
			db.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	db.Close()
	return nil, fmt.Errorf("database unreachable after %d attempts: %w", attempts, err)
}

// Migrate applies every embedded migration not yet applied, in filename order,
// each inside its own transaction.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	for _, name := range names {
		var applied bool
		if err := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}

		script, err := migrationFiles.ReadFile(name)
		if err != nil {
			return err
		}
		if err := applyMigration(ctx, db, name, string(script)); err != nil {
			return err
		}
		slog.Info("applied migration", "version", name)
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, name, script string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	return tx.Commit()
}
