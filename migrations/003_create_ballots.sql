-- migrations/003_create_ballots.sql

CREATE TABLE IF NOT EXISTS ballots (
    id TEXT PRIMARY KEY,
    election_id TEXT NOT NULL,
    encrypted_votes TEXT NOT NULL, -- JSON array of encrypted votes
    ring_signature TEXT NOT NULL,
    key_image TEXT NOT NULL UNIQUE, -- For double-vote detection
    smdc_commitment TEXT,
    merkle_proof TEXT, -- JSON array
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (election_id) REFERENCES elections(id) ON DELETE CASCADE
);

CREATE INDEX idx_ballots_election ON ballots(election_id);
CREATE INDEX idx_ballots_key_image ON ballots(key_image);
CREATE INDEX idx_ballots_timestamp ON ballots(timestamp);
