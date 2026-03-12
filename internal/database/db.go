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

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports 1 writer
	db.SetMaxIdleConns(5)

	return &DB{db}, nil
}

// RunMigrations runs all migration files in the specified directory
func (db *DB) RunMigrations(migrationsDir string) error {
	// Create migrations table if not exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort migration files
	var migrations []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			migrations = append(migrations, file.Name())
		}
	}
	sort.Strings(migrations)

	// Apply each migration
	for _, migration := range migrations {
		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migration).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if count > 0 {
			fmt.Printf("Migration %s already applied, skipping\n", migration)
			continue
		}

		// Read migration file
		path := filepath.Join(migrationsDir, migration)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migration, err)
		}

		// Execute migration inside a transaction for atomicity
		fmt.Printf("Applying migration %s...\n", migration)
		if err := db.Transaction(func(tx *sql.Tx) error {
			if _, err := tx.Exec(string(content)); err != nil {
				return fmt.Errorf("failed to apply migration SQL: %w", err)
			}
			if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration); err != nil {
				return fmt.Errorf("failed to record migration: %w", err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration, err)
		}

		fmt.Printf("Applied migration %s\n", migration)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// Transaction executes a function within a database transaction
func (db *DB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// ParseJSON helper to parse JSON from database
func ParseJSON(data string) map[string]interface{} {
	result := make(map[string]interface{})
	// Simple JSON parsing - in production use encoding/json
	data = strings.Trim(data, "[]")
	return result
}
