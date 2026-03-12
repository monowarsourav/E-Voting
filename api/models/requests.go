package models

// RegistrationRequest represents voter registration request
// Can register with either biometric OR username/password
type RegistrationRequest struct {
	VoterID           string `json:"voter_id" binding:"required"`
	// Option 1: Biometric registration
	FingerprintData   []byte `json:"fingerprint_data"`
	LivenessData      []byte `json:"liveness_data"`
	// Option 2: Password registration
	Password          string `json:"password"`
	// Additional data
	BiographicData    string `json:"biographic_data"`
	EligibilityProof  string `json:"eligibility_proof"`
}

// RegistrationResponse represents registration response
type RegistrationResponse struct {
	VoterID          string   `json:"voter_id"`
	PublicKey        string   `json:"public_key"`
	SMDCPublicCred   string   `json:"smdc_public_credential"`
	MerkleRoot       string   `json:"merkle_root"`
	RegistrationTime int64    `json:"registration_time"`
	Message          string   `json:"message"`
}

// LoginRequest represents login request with either fingerprint or username/password
type LoginRequest struct {
	VoterID         string `json:"voter_id" binding:"required"`
	// Option 1: Biometric authentication
	FingerprintData []byte `json:"fingerprint_data"`
	// Option 2: Password authentication
	Password        string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	VoterID   string `json:"voter_id"`
	AuthToken string `json:"auth_token"`
	ExpiresIn int64  `json:"expires_in"`
	Message   string `json:"message"`
}

// VoteRequest represents vote casting request
// MUST include fingerprint + liveness for every vote
type VoteRequest struct {
	VoterID         string `json:"voter_id" binding:"required"`
	ElectionID      string `json:"election_id" binding:"required"`
	CandidateID     int    `json:"candidate_id" binding:"required"`
	SMDCSlotIndex   int    `json:"smdc_slot_index"` // 0 is valid, so not using binding:"required"
	AuthToken       string `json:"auth_token" binding:"required"`
	// Biometric verification (required for every vote)
	FingerprintData []byte `json:"fingerprint_data" binding:"required"`
	LivenessData    []byte `json:"liveness_data" binding:"required"`
}

// VoteResponse represents vote response
type VoteResponse struct {
	ReceiptID      string `json:"receipt_id"`
	VoterID        string `json:"voter_id"`
	ElectionID     string `json:"election_id"`
	Timestamp      int64  `json:"timestamp"`
	BlockchainTxID string `json:"blockchain_tx_id"`
	KeyImage       string `json:"key_image"`
	Message        string `json:"message"`
}

// TallyRequest represents tally request (admin only)
type TallyRequest struct {
	ElectionID string `json:"election_id" binding:"required"`
	AdminToken string `json:"admin_token" binding:"required"`
}

// TallyResponse represents tally results
type TallyResponse struct {
	ElectionID       string         `json:"election_id"`
	CandidateTallies map[int]int64  `json:"candidate_tallies"`
	TotalVotes       int64          `json:"total_votes"`
	TallyTime        int64          `json:"tally_time"`
	Verified         bool           `json:"verified"`
}

// ElectionRequest represents election creation request
type ElectionRequest struct {
	Title         string      `json:"title" binding:"required"`
	Description   string      `json:"description"`
	Candidates    []Candidate `json:"candidates" binding:"required,min=2"`
	StartTime     int64       `json:"start_time" binding:"required"`
	EndTime       int64       `json:"end_time" binding:"required"`
	AdminToken    string      `json:"admin_token" binding:"required"`
}

// Candidate represents a candidate
type Candidate struct {
	ID          int    `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Party       string `json:"party"`
}

// ElectionResponse represents election creation response
type ElectionResponse struct {
	ElectionID   string      `json:"election_id"`
	Title        string      `json:"title"`
	Candidates   []Candidate `json:"candidates"`
	StartTime    int64       `json:"start_time"`
	EndTime      int64       `json:"end_time"`
	IsActive     bool        `json:"is_active"`
	Message      string      `json:"message"`
}

// VerifyVoteRequest represents vote verification request
type VerifyVoteRequest struct {
	ReceiptID string `json:"receipt_id" binding:"required"`
	VoterID   string `json:"voter_id" binding:"required"`
}

// VerifyVoteResponse represents vote verification response
type VerifyVoteResponse struct {
	Valid          bool   `json:"valid"`
	ElectionID     string `json:"election_id"`
	Timestamp      int64  `json:"timestamp"`
	BlockchainTxID string `json:"blockchain_tx_id"`
	Message        string `json:"message"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  int64  `json:"uptime"`
}

// ElectionListResponse represents list of elections
type ElectionListResponse struct {
	Elections []ElectionInfo `json:"elections"`
	Total     int            `json:"total"`
}

// ElectionInfo represents election information
type ElectionInfo struct {
	ElectionID    string      `json:"election_id"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Candidates    []Candidate `json:"candidates"`
	StartTime     int64       `json:"start_time"`
	EndTime       int64       `json:"end_time"`
	IsActive      bool        `json:"is_active"`
	TotalVotes    int         `json:"total_votes"`
}

// VoterInfoResponse represents voter information
type VoterInfoResponse struct {
	VoterID          string `json:"voter_id"`
	Registered       bool   `json:"registered"`
	RegistrationTime int64  `json:"registration_time"`
	HasVoted         bool   `json:"has_voted"`
	VoteTime         int64  `json:"vote_time,omitempty"`
}
