package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/coreycole/go_webserver/webserver/db/sqlc"
)

//go:embed migrations/*.sql
var migrations embed.FS

const (
	dbMaxConns = 4
	dirPerms   = 0o750
)

// DB wraps a sql.DB connection and embeds sqlc-generated query methods.
type DB struct {
	*sqlc.Queries
	db *sql.DB
}

func New(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, dirPerms); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)",
		dbPath,
	)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB.SetMaxOpenConns(dbMaxConns)
	sqlDB.SetMaxIdleConns(dbMaxConns)

	d := &DB{Queries: sqlc.New(sqlDB), db: sqlDB}
	if err := d.runMigrations(context.Background()); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return d, nil
}

func (d *DB) Close() error { return d.db.Close() }

func (d *DB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, nil)
}

func (d *DB) WithTx(tx *sql.Tx) *sqlc.Queries {
	return d.Queries.WithTx(tx)
}

// CompactOldMetrics aggregates metrics older than the given time offset into hourly buckets.
// offset is a SQLite datetime modifier string, e.g. "-3 days".
// In a single transaction: compute hourly averages, delete raw rows, insert compacted rows.
// Idempotent — re-compacting an already-compacted hour-bucket is a no-op (HAVING COUNT(*) > 1).
func (d *DB) CompactOldMetrics(ctx context.Context, offset string) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Find hourly buckets with more than 1 row (i.e., not already compacted).
	rows, err := tx.QueryContext(ctx, `
		SELECT
			strftime('%Y-%m-%d %H:00:00', created_at) AS bucket,
			AVG(cpu_percent),
			AVG(mem_used_percent),
			CAST(AVG(mem_used_bytes) AS INTEGER),
			MAX(total_visits),
			MAX(unique_visitors),
			COUNT(*) AS cnt
		FROM metrics_snapshots
		WHERE created_at < datetime('now', ?)
		GROUP BY bucket
		HAVING cnt > 1
	`, offset)
	if err != nil {
		return err
	}

	type compacted struct {
		bucket         string
		cpuPercent     float64
		memUsedPercent float64
		memUsedBytes   int64
		totalVisits    int64
		uniqueVisitors int64
	}
	var buckets []compacted
	for rows.Next() {
		var c compacted
		var cnt int
		if scanErr := rows.Scan(
			&c.bucket,
			&c.cpuPercent,
			&c.memUsedPercent,
			&c.memUsedBytes,
			&c.totalVisits,
			&c.uniqueVisitors,
			&cnt,
		); scanErr != nil {
			_ = rows.Close()
			return scanErr
		}
		buckets = append(buckets, c)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, c := range buckets {
		// Delete all raw rows in this hour bucket.
		if _, execErr := tx.ExecContext(ctx, `
			DELETE FROM metrics_snapshots
			WHERE created_at < datetime('now', ?) AND strftime('%Y-%m-%d %H:00:00', created_at) = ?
		`, offset, c.bucket); execErr != nil {
			return execErr
		}
		// Insert single compacted row with the bucket timestamp.
		if _, execErr := tx.ExecContext(ctx, `
			INSERT INTO metrics_snapshots (cpu_percent, mem_used_percent, mem_used_bytes, total_visits, unique_visitors, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, c.cpuPercent, c.memUsedPercent, c.memUsedBytes, c.totalVisits, c.uniqueVisitors, c.bucket); execErr != nil {
			return execErr
		}
	}

	return tx.Commit()
}

func (d *DB) runMigrations(ctx context.Context) error {
	// Create migrations tracking table.
	_, err := d.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create _migrations table: %w", err)
	}

	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Check if already applied.
		var count int
		if scanErr := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM _migrations WHERE name = ?", name).
			Scan(&count); scanErr != nil {
			return fmt.Errorf("check migration %s: %w", name, scanErr)
		}
		if count > 0 {
			continue
		}

		// Read and execute migration.
		content, readErr := migrations.ReadFile("migrations/" + name)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", name, readErr)
		}

		// Split on semicolons and execute each statement.
		stmts := strings.Split(string(content), ";")
		for _, stmt := range stmts {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, execErr := d.db.ExecContext(ctx, stmt); execErr != nil {
				return fmt.Errorf("execute migration %s: %w", name, execErr)
			}
		}

		// Record migration as applied.
		_, err = d.db.ExecContext(ctx, "INSERT INTO _migrations (name) VALUES (?)", name)
		if err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}

	return nil
}
