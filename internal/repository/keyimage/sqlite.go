// Package keyimage provides a SQLite-backed implementation of
// voting.KeyImageStore for durable double-vote prevention.
package keyimage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/covertvote/e-voting/internal/voting"
)

// SQLiteStore persists used key images in SQLite. The unique primary key on
// key_image enforces the atomic check-and-mark semantics required by
// voting.KeyImageStore.MarkUsed.
type SQLiteStore struct {
	db *sql.DB
}

// New returns a key-image store backed by the given database handle.
func New(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// Exists reports whether the key image has already been recorded.
func (s *SQLiteStore) Exists(keyImage string) (bool, error) {
	var dummy int
	err := s.db.QueryRow(
		`SELECT 1 FROM key_images WHERE key_image = ? LIMIT 1`,
		keyImage,
	).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("key image exists: %w", err)
	}
	return true, nil
}

// MarkUsed records a key image. Returns voting.ErrKeyImageAlreadyUsed if the
// key image was already present (UNIQUE constraint violation).
func (s *SQLiteStore) MarkUsed(keyImage string) error {
	_, err := s.db.Exec(
		`INSERT INTO key_images (key_image, used_at) VALUES (?, ?)`,
		keyImage, time.Now().Unix(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return voting.ErrKeyImageAlreadyUsed
		}
		return fmt.Errorf("key image mark used: %w", err)
	}
	return nil
}

// isUniqueViolation detects SQLite UNIQUE constraint failures. The mattn
// driver surfaces these as errors containing "UNIQUE constraint failed".
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
