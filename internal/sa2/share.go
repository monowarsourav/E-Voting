package sa2

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
)

// VoteSplitter handles splitting per-candidate Paillier ciphertexts into SA²
// shares for two non-colluding aggregator servers.
type VoteSplitter struct {
	PublicKey *crypto.PaillierPublicKey
}

// NewVoteSplitter creates a new vote splitter.
func NewVoteSplitter(pk *crypto.PaillierPublicKey) *VoteSplitter {
	return &VoteSplitter{
		PublicKey: pk,
	}
}

// SplitVote splits a vector of per-candidate encrypted votes into two SA² shares.
// For each candidate position j:
//
//	S_{A,j} = E_w_j * Enc(mask_j)   = Enc(v_j + mask_j)
//	S_{B,j} = Enc(-mask_j)
//
// Reconstruction (S_{A,j} * S_{B,j}) recovers E_w_j. The mask is sampled fresh
// per candidate so the two aggregators see independent random noise on each
// candidate column.
func (vs *VoteSplitter) SplitVote(voterID string, encryptedVotes []*big.Int) (*VoteShare, error) {
	if vs.PublicKey == nil || vs.PublicKey.N == nil {
		return nil, errors.New("sa2: nil paillier public key")
	}
	if len(encryptedVotes) == 0 {
		return nil, errors.New("sa2: empty ciphertext vector")
	}
	pk := vs.PublicKey
	maxMask := new(big.Int).Div(pk.N, big.NewInt(100))

	sharesA := make([]*big.Int, len(encryptedVotes))
	sharesB := make([]*big.Int, len(encryptedVotes))

	for j, E := range encryptedVotes {
		if E == nil {
			return nil, errors.New("sa2: nil ciphertext entry")
		}
		mask, err := rand.Int(rand.Reader, maxMask)
		if err != nil {
			return nil, err
		}
		encMask, err := pk.Encrypt(mask)
		if err != nil {
			return nil, err
		}
		sharesA[j] = pk.Add(E, encMask)
		negMask := new(big.Int).Sub(pk.N, mask)
		encNegMask, err := pk.Encrypt(negMask)
		if err != nil {
			return nil, err
		}
		sharesB[j] = encNegMask
	}

	return &VoteShare{
		VoterID: voterID,
		SharesA: sharesA,
		SharesB: sharesB,
	}, nil
}

// ReconstructVote recovers the per-candidate weighted ciphertext vector from a
// single voter's two shares. Used in tests and for verification; the production
// flow keeps the shares with their respective aggregators and reconstructs only
// after homomorphic aggregation across many voters (see Aggregator/Combiner).
func (vs *VoteSplitter) ReconstructVote(share *VoteShare) ([]*big.Int, error) {
	if share == nil || len(share.SharesA) != len(share.SharesB) {
		return nil, errors.New("sa2: malformed VoteShare")
	}
	out := make([]*big.Int, len(share.SharesA))
	for j := range share.SharesA {
		out[j] = vs.PublicKey.Add(share.SharesA[j], share.SharesB[j])
	}
	return out, nil
}
