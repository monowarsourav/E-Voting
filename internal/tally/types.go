package tally

import (
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
)

// TallyResult represents the final tally of an election
type TallyResult struct {
	ElectionID    string
	CandidateTallies map[int]*big.Int // candidateID -> vote count
	TotalVotes    int
	Timestamp     int64
}

// PartialDecryption represents a partial decryption from one server
type PartialDecryption struct {
	ServerID string
	Value    *big.Int
	Proof    []byte // NIZK proof (simplified for now)
}

// ThresholdKey represents a share of the private key
type ThresholdKey struct {
	ServerID  string
	Share     *big.Int
	Index     int
	PublicKey *crypto.PaillierPublicKey
}
