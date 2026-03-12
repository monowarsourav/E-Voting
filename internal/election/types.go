// internal/election/types.go

package election

import (
	"time"
)

// Election represents an election in the system
type Election struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	Candidates            []string  `json:"candidates"`
	NumCandidates         int       `json:"num_candidates"`
	StartTime             time.Time `json:"start_time"`
	EndTime               time.Time `json:"end_time"`
	RegistrationDeadline  time.Time `json:"registration_deadline"`
	Status                string    `json:"status"` // pending, active, closed, tallied
	MerkleRoot            string    `json:"merkle_root"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// ElectionStatus represents the status of an election
type ElectionStatus string

const (
	StatusPending ElectionStatus = "pending"
	StatusActive  ElectionStatus = "active"
	StatusClosed  ElectionStatus = "closed"
	StatusTallied ElectionStatus = "tallied"
)

// CreateElectionRequest represents a request to create an election
type CreateElectionRequest struct {
	Name                 string    `json:"name" binding:"required"`
	Description          string    `json:"description"`
	Candidates           []string  `json:"candidates" binding:"required,min=2"`
	StartTime            time.Time `json:"start_time" binding:"required"`
	EndTime              time.Time `json:"end_time" binding:"required"`
	RegistrationDeadline time.Time `json:"registration_deadline" binding:"required"`
}

// UpdateElectionRequest represents a request to update an election
type UpdateElectionRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Candidates  []string `json:"candidates,omitempty"`
	Status      string   `json:"status,omitempty"`
}
