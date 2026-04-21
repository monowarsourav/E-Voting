// Package database provides a typed wrapper around the application's SQLite
// connection and a simple file-based migration runner.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQL database connection with application-specific helpers.
type DB struct {
	*sql.DB
}

// Options configures New. Production callers should enable WAL for
// concurrent reads.
type Options struct {
	// WALMode turns on write-ahead logging (recommended in production).
	WALMode bool
	// MaxOpenConns is the upper bound on open connections. SQLite only
	// supports one writer, but readers under WAL can coexist — keep this
	// modest (e.g. 5–10).
	MaxOpenConns int
	MaxIdleConns int
	// BusyTimeoutMs is the value for PRAGMA busy_timeout.
	BusyTimeoutMs int
}

// DefaultOptions returns conservative production defaults.
func DefaultOptions() Options {
	return Options{
		WALMode:       true,
		MaxOpenConns:  5,
		MaxIdleConns:  5,
		BusyTimeoutMs: 5000,
	}
}

// New opens a SQLite connection at the given path and applies production
// pragmas. The parent directory is created if necessary.
func New(dataSourceName string) (*DB, error) {
	return NewWithOptions(dataSourceName, DefaultOptions())
}

// NewWithOptions is like New but allows overriding defaults.
func NewWithOptions(dataSourceName string, opts Options) (*DB, error) {
	if dir := filepath.Dir(dataSourceName); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dataSourceName+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	pragmas := []string{"PRAGMA foreign_keys = ON"}
	if opts.WALMode {
		pragmas = append(pragmas, "PRAGMA journal_mode = WAL")
		pragmas = append(pragmas, "PRAGMA synchronous = NORMAL")
	}
	if opts.BusyTimeoutMs > 0 {
		pragmas = append(pragmas, fmt.Sprintf("PRAGMA busy_timeout = %d", opts.BusyTimeoutMs))
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("apply pragma %q: %w", p, err)
		}
	}

	if opts.MaxOpenConns > 0 {
		db.SetMaxOpenConns(opts.MaxOpenConns)
	}
	if opts.MaxIdleConns > 0 {
		db.SetMaxIdleConns(opts.MaxIdleConns)
	}

	return &DB{db}, nil
}

// RunMigrations applies every .sql file in migrationsDir whose basename has
// not yet been recorded in schema_migrations. Files are applied in
// lexicographic order so a zero-padded numeric prefix (e.g. 001_, 002_) is
// strongly recommended.
func (db *DB) RunMigrations(migrationsDir string) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var applied int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied > 0 {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}
