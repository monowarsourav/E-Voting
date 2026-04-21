// Package session provides a SQLite-backed implementation of the
// middleware.PersistentSessionStore interface.
package session

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/covertvote/e-voting/api/middleware"
)

// ErrNotFound is returned by Get when no session exists for the token.
var ErrNotFound = errors.New("session not found")

// SQLiteStore persists voter sessions in SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// New returns a session store backed by the given database handle. The
// caller is expected to have run migration 006_create_sessions.sql.
func New(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// ErrNotFound returns the sentinel so callers can test for it generically.
func (s *SQLiteStore) ErrNotFound() error { return ErrNotFound }

// Get retrieves a session by its token. Returns (nil, nil) if absent so the
// in-memory cache can treat it as a miss without treating it as an error.
func (s *SQLiteStore) Get(token string) (*middleware.Session, error) {
	row := s.db.QueryRow(
		`SELECT token, voter_id, created_at, expires_at FROM sessions WHERE token = ?`,
		token,
	)
	var sess middleware.Session
	if err := row.Scan(&sess.Token, &sess.VoterID, &sess.CreatedAt, &sess.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("session get: %w", err)
	}
	return &sess, nil
}

// Set upserts a session.
func (s *SQLiteStore) Set(sess *middleware.Session) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (token, voter_id, created_at, expires_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(token) DO UPDATE SET
		   voter_id   = excluded.voter_id,
		   created_at = excluded.created_at,
		   expires_at = excluded.expires_at`,
		sess.Token, sess.VoterID, sess.CreatedAt, sess.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("session set: %w", err)
	}
	return nil
}

// Delete removes a session by token.
func (s *SQLiteStore) Delete(token string) error {
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token); err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	return nil
}

// DeleteExpired removes sessions whose ExpiresAt is <= now.
func (s *SQLiteStore) DeleteExpired(now int64) error {
	if _, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now); err != nil {
		return fmt.Errorf("session delete expired: %w", err)
	}
	return nil
}
