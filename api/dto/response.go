// api/dto/response.go

package dto

import (
	"time"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int         `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

// ElectionResponse represents an election in API responses
type ElectionResponse struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	Candidates           []string  `json:"candidates"`
	NumCandidates        int       `json:"num_candidates"`
	Status               string    `json:"status"`
	StartTime            time.Time `json:"start_time"`
	EndTime              time.Time `json:"end_time"`
	RegistrationDeadline time.Time `json:"registration_deadline"`
	MerkleRoot           string    `json:"merkle_root,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// VoterResponse represents a voter in API responses
type VoterResponse struct {
	ID           string    `json:"id"`
	VoterID      string    `json:"voter_id"`
	IsEligible   bool      `json:"is_eligible"`
	HasVoted     bool      `json:"has_voted"`
	ElectionID   string    `json:"election_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

// RegistrationResponse represents the response from voter registration
type RegistrationResponse struct {
	Success      bool              `json:"success"`
	VoterID      string            `json:"voter_id"`
	Message      string            `json:"message"`
	Credential   *CredentialInfo   `json:"credential,omitempty"`
	RingKey      *RingKeyInfo      `json:"ring_key,omitempty"`
	MerkleProof  []string          `json:"merkle_proof,omitempty"`
}

// CredentialInfo represents SMDC credential information
type CredentialInfo struct {
	NumSlots      int      `json:"num_slots"`
	RealSlotIndex int      `json:"real_slot_index"` // Only sent to voter, not stored
	Commitments   []string `json:"commitments,omitempty"`
}

// RingKeyInfo represents ring signature key information
type RingKeyInfo struct {
	PublicKey string `json:"public_key"`
	Index     int    `json:"index"`
}

// VoteResponse represents the response from casting a vote
type VoteResponse struct {
	Success           bool      `json:"success"`
	Message           string    `json:"message"`
	ReceiptID         string    `json:"receipt_id"`
	KeyImage          string    `json:"key_image"`
	BlockchainTxID    string    `json:"blockchain_tx_id,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
}

// TallyResponse represents the response from tallying votes
type TallyResponse struct {
	Success          bool                   `json:"success"`
	ElectionID       string                 `json:"election_id"`
	TotalVotes       int                    `json:"total_votes"`
	CandidateTallies map[string]int         `json:"candidate_tallies"`
	Winner           string                 `json:"winner,omitempty"`
	TallyTime        time.Time              `json:"tally_time"`
	Verified         bool                   `json:"verified"`
}

// VoteVerificationResponse represents the response from verifying a vote
type VoteVerificationResponse struct {
	Success    bool      `json:"success"`
	Valid      bool      `json:"valid"`
	ReceiptID  string    `json:"receipt_id"`
	KeyImage   string    `json:"key_image"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	Message    string    `json:"message"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Services  map[string]string `json:"services,omitempty"`
}

// EligibilityResponse represents the eligibility check response
type EligibilityResponse struct {
	Eligible bool   `json:"eligible"`
	VoterID  string `json:"voter_id,omitempty"`
	Message  string `json:"message"`
}

// VoteCountResponse represents vote count statistics
type VoteCountResponse struct {
	ElectionID string `json:"election_id"`
	TotalVotes int    `json:"total_votes"`
	Status     string `json:"status"`
}

// AdminStatsResponse represents admin statistics
type AdminStatsResponse struct {
	TotalElections      int            `json:"total_elections"`
	ActiveElections     int            `json:"active_elections"`
	TotalVoters         int            `json:"total_voters"`
	TotalVotesCast      int            `json:"total_votes_cast"`
	ElectionsByStatus   map[string]int `json:"elections_by_status"`
}
