-- migrations/005_create_tally_results.sql

CREATE TABLE IF NOT EXISTS tally_results (
    id TEXT PRIMARY KEY,
    election_id TEXT NOT NULL UNIQUE,
    candidate_tallies TEXT NOT NULL, -- JSON object {candidate_id: vote_count}
    total_votes INTEGER NOT NULL,
    encrypted_tally TEXT, -- Encrypted aggregate before decryption
    decryption_proof TEXT, -- JSON proof of correct decryption
    computed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    verified BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (election_id) REFERENCES elections(id) ON DELETE CASCADE
);

CREATE INDEX idx_tally_election ON tally_results(election_id);
CREATE INDEX idx_tally_computed_at ON tally_results(computed_at);
