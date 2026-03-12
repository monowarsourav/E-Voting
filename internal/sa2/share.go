package sa2

import (
	"crypto/rand"
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
)

// VoteSplitter handles splitting votes into shares for SA²
type VoteSplitter struct {
	PublicKey *crypto.PaillierPublicKey
}

// NewVoteSplitter creates a new vote splitter
func NewVoteSplitter(pk *crypto.PaillierPublicKey) *VoteSplitter {
	return &VoteSplitter{
		PublicKey: pk,
	}
}

// SplitVote splits an encrypted vote into two shares
// share_A = encrypted_vote × E(mask)
// share_B = E(-mask)
// When combined: share_A × share_B = encrypted_vote × E(mask) × E(-mask) = encrypted_vote
func (vs *VoteSplitter) SplitVote(voterID string, encryptedVote *big.Int) (*VoteShare, error) {
	pk := vs.PublicKey

	// Generate random mask (smaller than n to avoid overflow)
	maxMask := new(big.Int).Div(pk.N, big.NewInt(100))
	mask, err := rand.Int(rand.Reader, maxMask)
	if err != nil {
		return nil, err
	}

	// Encrypt the mask: E(mask)
	encMask, err := pk.Encrypt(mask)
	if err != nil {
		return nil, err
	}

	// Share A = encrypted_vote × E(mask) (homomorphic addition)
	shareA := pk.Add(encryptedVote, encMask)

	// Encrypt negative mask: E(-mask)
	// In modular arithmetic: -mask = n - mask
	negMask := new(big.Int).Sub(pk.N, mask)
	shareB, err := pk.Encrypt(negMask)
	if err != nil {
		return nil, err
	}

	return &VoteShare{
		VoterID: voterID,
		ShareA:  shareA,
		ShareB:  shareB,
	}, nil
}

// ReconstructVote reconstructs the encrypted vote from shares (for verification)
// encrypted_vote = share_A × share_B
func (vs *VoteSplitter) ReconstructVote(share *VoteShare) *big.Int {
	pk := vs.PublicKey
	// Homomorphic addition: E(vote + mask) × E(-mask) = E(vote)
	return pk.Add(share.ShareA, share.ShareB)
}
