package models

// voterIDRule constrains voter IDs to alphanumeric + underscore/dash, 3–64
// chars. Applied via struct tags across request payloads — keeping the rule
// in one place.
const voterIDRule = "required,min=3,max=64,alphanumdash"

// RegistrationRequest represents voter registration request.
// Can register with either biometric OR username/password.
type RegistrationRequest struct {
	VoterID          string `json:"voter_id" binding:"required,min=3,max=64,alphanumdash"`
	FingerprintData  []byte `json:"fingerprint_data"`
	LivenessData     []byte `json:"liveness_data"`
	Password         string `json:"password" binding:"omitempty,min=8,max=128"`
	BiographicData   string `json:"biographic_data" binding:"omitempty,max=1024"`
	EligibilityProof string `json:"eligibility_proof" binding:"omitempty,max=4096"`
}

// RegistrationResponse represents registration response
type RegistrationResponse struct {
	VoterID          string `json:"voter_id"`
	PublicKey        string `json:"public_key"`
	SMDCPublicCred   string `json:"smdc_public_credential"`
	MerkleRoot       string `json:"merkle_root"`
	RegistrationTime int64  `json:"registration_time"`
	Message          string `json:"message"`
}

// LoginRequest represents login request with either fingerprint or username/password.
type LoginRequest struct {
	VoterID         string `json:"voter_id" binding:"required,min=3,max=64,alphanumdash"`
	FingerprintData []byte `json:"fingerprint_data"`
	Password        string `json:"password" binding:"omitempty,min=8,max=128"`
}

// LoginResponse represents login response
type LoginResponse struct {
	VoterID   string `json:"voter_id"`
	AuthToken string `json:"auth_token"`
	ExpiresIn int64  `json:"expires_in"`
	Message   string `json:"message"`
}

// VoteRequest represents vote casting request.
// MUST include fingerprint + liveness for every vote.
// DetectedSignalType and DetectedSignalValue are optional: when both are set
// the server verifies them against the voter's registered duress signal; a
// mismatch silently zeros the vote weight (coercion resistance).
type VoteRequest struct {
	VoterID              string `json:"voter_id" binding:"required,min=3,max=64,alphanumdash"`
	ElectionID           string `json:"election_id" binding:"required,min=3,max=64,alphanumdash"`
	CandidateID          int    `json:"candidate_id" binding:"required,min=1"`
	SMDCSlotIndex        int    `json:"smdc_slot_index" binding:"min=0"`
	AuthToken            string `json:"auth_token" binding:"required"`
	FingerprintData      []byte `json:"fingerprint_data" binding:"required"`
	LivenessData         []byte `json:"liveness_data" binding:"required"`
	DetectedSignalType   string `json:"detected_signal_type,omitempty"`
	DetectedSignalValue  string `json:"detected_signal_value,omitempty"`
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
	ElectionID       string        `json:"election_id"`
	CandidateTallies map[int]int64 `json:"candidate_tallies"`
	TotalVotes       int64         `json:"total_votes"`
	TallyTime        int64         `json:"tally_time"`
	Verified         bool          `json:"verified"`
}

// ElectionRequest represents election creation request
type ElectionRequest struct {
	Title       string      `json:"title" binding:"required"`
	Description string      `json:"description"`
	Candidates  []Candidate `json:"candidates" binding:"required,min=2"`
	StartTime   int64       `json:"start_time" binding:"required"`
	EndTime     int64       `json:"end_time" binding:"required"`
	AdminToken  string      `json:"admin_token" binding:"required"`
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
	ElectionID string      `json:"election_id"`
	Title      string      `json:"title"`
	Candidates []Candidate `json:"candidates"`
	StartTime  int64       `json:"start_time"`
	EndTime    int64       `json:"end_time"`
	IsActive   bool        `json:"is_active"`
	Message    string      `json:"message"`
}

// VerifyVoteRequest represents vote verification request.
type VerifyVoteRequest struct {
	ReceiptID string `json:"receipt_id" binding:"required,hexadecimal,min=16,max=128"`
	VoterID   string `json:"voter_id" binding:"required,min=3,max=64,alphanumdash"`
}

// VerifyVoteResponse represents vote verification response
type VerifyVoteResponse struct {
	Valid          bool   `json:"valid"`
	ElectionID     string `json:"election_id"`
	Timestamp      int64  `json:"timestamp"`
	BlockchainTxID string `json:"blockchain_tx_id"`
	Message        string `json:"message"`
}

// SetDuressSignalRequest is the body for POST /api/v1/voters/:voterID/duress-signal.
type SetDuressSignalRequest struct {
	SignalType  string `json:"signal_type" binding:"required"`
	SignalValue string `json:"signal_value" binding:"required"`
}

// SetDuressSignalResponse is the response after a duress signal is registered.
type SetDuressSignalResponse struct {
	Status   string `json:"status"`
	SignalID string `json:"signal_id"`
	SetAt    int64  `json:"set_at"`
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
	ElectionID  string      `json:"election_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Candidates  []Candidate `json:"candidates"`
	StartTime   int64       `json:"start_time"`
	EndTime     int64       `json:"end_time"`
	IsActive    bool        `json:"is_active"`
	TotalVotes  int         `json:"total_votes"`
}

// VoterInfoResponse represents voter information
type VoterInfoResponse struct {
	VoterID          string `json:"voter_id"`
	Registered       bool   `json:"registered"`
	RegistrationTime int64  `json:"registration_time"`
	HasVoted         bool   `json:"has_voted"`
	VoteTime         int64  `json:"vote_time,omitempty"`
}
