-- migrations/002_create_voters.sql

CREATE TABLE IF NOT EXISTS voters (
    id TEXT PRIMARY KEY,
    voter_id TEXT UNIQUE NOT NULL,
    fingerprint_hash TEXT NOT NULL,
    nid_hash TEXT NOT NULL,
    ring_public_key TEXT NOT NULL,
    merkle_leaf TEXT,
    is_eligible BOOLEAN DEFAULT TRUE,
    has_voted BOOLEAN DEFAULT FALSE,
    election_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (election_id) REFERENCES elections(id) ON DELETE CASCADE
);

CREATE INDEX idx_voters_voter_id ON voters(voter_id);
CREATE INDEX idx_voters_election ON voters(election_id);
CREATE INDEX idx_voters_has_voted ON voters(has_voted);
CREATE INDEX idx_voters_fingerprint ON voters(fingerprint_hash);
