-- migrations/001_create_elections.sql

CREATE TABLE IF NOT EXISTS elections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'created',
    candidates TEXT NOT NULL, -- JSON array
    num_candidates INTEGER NOT NULL,
    merkle_root TEXT,
    start_time DATETIME,
    end_time DATETIME,
    registration_deadline DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_elections_status ON elections(status);
CREATE INDEX idx_elections_start_time ON elections(start_time);
CREATE INDEX idx_elections_end_time ON elections(end_time);
