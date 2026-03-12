-- migrations/004_create_credentials.sql

CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    voter_id TEXT NOT NULL,
    election_id TEXT NOT NULL,
    commitments TEXT NOT NULL, -- JSON array
    binary_proofs TEXT NOT NULL, -- JSON
    sum_proof TEXT NOT NULL, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (voter_id) REFERENCES voters(voter_id) ON DELETE CASCADE,
    FOREIGN KEY (election_id) REFERENCES elections(id) ON DELETE CASCADE
);

CREATE INDEX idx_credentials_voter ON credentials(voter_id);
CREATE INDEX idx_credentials_election ON credentials(election_id);
CREATE UNIQUE INDEX idx_credentials_voter_election ON credentials(voter_id, election_id);
