-- migrations/006_create_sessions.sql
-- Persistent session storage so server restarts do not invalidate live voter
-- sessions. Tokens are 256-bit cryptographic random values.

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    voter_id   TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_voter   ON sessions(voter_id);
