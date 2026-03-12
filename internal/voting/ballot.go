package voting

import (
	"errors"
	"math/big"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
)

// BallotCreator handles ballot creation
type BallotCreator struct {
	PublicKey *crypto.PaillierPublicKey
}

// NewBallotCreator creates a new ballot creator
func NewBallotCreator(pk *crypto.PaillierPublicKey) *BallotCreator {
	return &BallotCreator{
		PublicKey: pk,
	}
}

// CreateBallot creates a ballot for a candidate
func (bc *BallotCreator) CreateBallot(voterID string, candidateID int) (*Ballot, error) {
	if candidateID < 0 {
		return nil, errors.New("invalid candidate ID")
	}

	// Encrypt the vote (candidateID)
	vote := big.NewInt(int64(candidateID))
	encryptedVote, err := bc.PublicKey.Encrypt(vote)
	if err != nil {
		return nil, err
	}

	return &Ballot{
		VoterID:       voterID,
		CandidateID:   candidateID,
		EncryptedVote: encryptedVote,
		Weight:        big.NewInt(1), // Will be applied via SMDC
		Timestamp:     time.Now().Unix(),
	}, nil
}

// ApplyWeight applies SMDC weight to encrypted vote
// Returns E(vote × weight) using Paillier scalar multiplication
func (bc *BallotCreator) ApplyWeight(encryptedVote, weight *big.Int) *big.Int {
	// E(vote)^weight = E(vote × weight)
	return bc.PublicKey.Multiply(encryptedVote, weight)
}

// VerifyBallotRange verifies vote is within valid range
func (bc *BallotCreator) VerifyBallotRange(encryptedVote *big.Int) bool {
	// In production, this would use range proofs
	// For now, just check it's not nil and within n²
	if encryptedVote == nil {
		return false
	}

	return encryptedVote.Cmp(bc.PublicKey.N2) < 0
}
