// internal/blockchain/types.go

package blockchain

import (
	"time"
)

// BlockchainConfig holds blockchain configuration
type BlockchainConfig struct {
	Enabled      bool   `json:"enabled"`
	ConfigPath   string `json:"config_path"`
	ChannelID    string `json:"channel_id"`
	ChaincodeID  string `json:"chaincode_id"`
	OrgID        string `json:"org_id"`
	UserID       string `json:"user_id"`
	PeerEndpoint string `json:"peer_endpoint"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // election, vote, credential, tally
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	BlockNum  uint64    `json:"block_num"`
	TxIndex   uint32    `json:"tx_index"`
}

// ElectionOnChain represents an election stored on blockchain
type ElectionOnChain struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Candidates   []string `json:"candidates"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
	MerkleRoot   string   `json:"merkle_root"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
	CreatorOrgID string   `json:"creator_org_id"`
}

// VoteOnChain represents a vote stored on blockchain
type VoteOnChain struct {
	ID             string `json:"id"`
	ElectionID     string `json:"election_id"`
	EncryptedVote  string `json:"encrypted_vote"` // Base64 encoded
	RingSignature  string `json:"ring_signature"` // JSON encoded
	KeyImage       string `json:"key_image"`      // Hex encoded
	SMDCSlot       int    `json:"smdc_slot"`
	Timestamp      string `json:"timestamp"`
	SubmitterOrgID string `json:"submitter_org_id"`
}

// CredentialOnChain represents a voter credential stored on blockchain
type CredentialOnChain struct {
	ID           string   `json:"id"`
	ElectionID   string   `json:"election_id"`
	VoterHash    string   `json:"voter_hash"` // Hash of voter ID (anonymized)
	Commitments  []string `json:"commitments"`
	BinaryProofs string   `json:"binary_proofs"` // JSON encoded
	SumProof     string   `json:"sum_proof"`     // JSON encoded
	CreatedAt    string   `json:"created_at"`
}

// TallyResultOnChain represents tally results on blockchain
type TallyResultOnChain struct {
	ID               string         `json:"id"`
	ElectionID       string         `json:"election_id"`
	TotalVotes       int            `json:"total_votes"`
	CandidateTallies map[string]int `json:"candidate_tallies"`
	TallyProof       string         `json:"tally_proof"` // JSON encoded
	TalliedAt        string         `json:"tallied_at"`
	TallierOrgID     string         `json:"tallier_org_id"`
	Verified         bool           `json:"verified"`
}

// QueryResult represents a generic query result from blockchain
type QueryResult struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

// BlockchainEvent represents an event emitted by chaincode
type BlockchainEvent struct {
	EventName   string    `json:"event_name"`
	Payload     []byte    `json:"payload"`
	TxID        string    `json:"tx_id"`
	BlockNumber uint64    `json:"block_number"`
	Timestamp   time.Time `json:"timestamp"`
}

// ChaincodeResponse represents a response from chaincode invocation
type ChaincodeResponse struct {
	TxID    string `json:"tx_id"`
	Payload []byte `json:"payload"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
