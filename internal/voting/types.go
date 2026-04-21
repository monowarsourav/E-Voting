package voting

import (
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/smdc"
)

// Ballot represents a voter's ballot
type Ballot struct {
	VoterID       string
	CandidateID   int
	EncryptedVote *big.Int // Paillier encrypted vote
	Weight        *big.Int // From SMDC real slot (should be 1)
	Timestamp     int64
}

// WeightedVote represents an encrypted vote multiplied by SMDC weight
type WeightedVote struct {
	VoterID        string
	EncryptedVote  *big.Int              // E(vote × weight)
	SMDCSlotIndex  int                   // Which SMDC slot was used
	SMDCCommitment *big.Int              // Public commitment of the slot
	RingSignature  *crypto.RingSignature // Anonymous signature
	RingPublicKeys []*big.Int            // The fixed-size ring used for the signature
	Timestamp      int64
}

// CastVote represents a complete vote submission
type CastVote struct {
	VoterID          string
	WeightedVote     *WeightedVote
	VoteShares       *sa2.VoteShare         // SA² shares
	MerkleProof      [][]byte               // Eligibility proof
	PublicCredential *smdc.PublicCredential // SMDC public credential
}

// VoteReceipt represents a receipt after vote submission
type VoteReceipt struct {
	VoterID        string
	ReceiptID      string
	Timestamp      int64
	BlockchainTxID string   // Hyperledger transaction ID (TODO)
	KeyImage       *big.Int // For double-vote detection
}

// Candidate represents a candidate in the election
type Candidate struct {
	ID          int
	Name        string
	Description string
	Party       string
}

// Election represents an election
type Election struct {
	ElectionID  string
	Title       string
	Description string
	Candidates  []*Candidate
	StartTime   int64
	EndTime     int64
	IsActive    bool
}
