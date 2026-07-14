package voting

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
)

// BallotCreator handles ballot creation under the Helios-style 1-hot Paillier
// encoding: every ballot is a length-m vector of per-candidate Paillier
// ciphertexts (v_j ∈ {0,1}, Σ v_j = 1) accompanied by per-candidate CDS binary
// proofs and a single sum-to-one proof, all generated directly on the
// ciphertexts (no parallel Pedersen system).
type BallotCreator struct {
	PublicKey *crypto.PaillierPublicKey
}

// NewBallotCreator creates a new ballot creator.
func NewBallotCreator(pk *crypto.PaillierPublicKey) *BallotCreator {
	return &BallotCreator{
		PublicKey: pk,
	}
}

// CreateBallot produces a 1-hot Paillier ballot for the chosen candidate
// together with the CDS OR-proof per candidate and a single Schnorr-style
// sum-to-one proof. candidates is the full candidate list (ordering fixes the
// canonical index j); candidateID must match one of the entries.
//
// The returned Ballot retains the encryption randomness in Ballot.Randomness;
// the caller MUST treat it as secret material and never persist or transmit it.
func (bc *BallotCreator) CreateBallot(
	voterID string,
	candidateID int,
	candidates []*Candidate,
	electionID string,
) (*Ballot, error) {
	if bc.PublicKey == nil || bc.PublicKey.N == nil {
		return nil, errors.New("ballot: nil paillier public key")
	}
	if len(candidates) == 0 {
		return nil, errors.New("ballot: empty candidate list")
	}
	if electionID == "" {
		return nil, errors.New("ballot: electionID must not be empty")
	}
	// Find canonical index of the chosen candidate.
	chosenIdx := -1
	for i, c := range candidates {
		if c.ID == candidateID {
			chosenIdx = i
			break
		}
	}
	if chosenIdx < 0 {
		return nil, fmt.Errorf("ballot: candidate %d not in election", candidateID)
	}

	m := len(candidates)
	ciphertexts := make([]*big.Int, m)
	randomness := make([]*big.Int, m)
	binaryProofs := make([]*crypto.PaillierBinaryProof, m)

	for j := 0; j < m; j++ {
		var vj *big.Int
		if j == chosenIdx {
			vj = big.NewInt(1)
		} else {
			vj = big.NewInt(0)
		}
		rj, err := rand.Int(rand.Reader, bc.PublicKey.N)
		if err != nil {
			return nil, fmt.Errorf("ballot: entropy for r[%d]: %w", j, err)
		}
		// Ensure r is in Z_n^* (extremely unlikely to hit p, q for large n).
		if rj.Sign() == 0 {
			rj = big.NewInt(1)
		}
		Ej, err := bc.PublicKey.EncryptWithRandomness(vj, rj)
		if err != nil {
			return nil, fmt.Errorf("ballot: encrypt[%d]: %w", j, err)
		}
		nonce, err := crypto.GenerateNonce()
		if err != nil {
			return nil, fmt.Errorf("ballot: nonce[%d]: %w", j, err)
		}
		proof, err := crypto.ProvePaillierBinary(bc.PublicKey, vj, rj, Ej, j, nonce, electionID)
		if err != nil {
			return nil, fmt.Errorf("ballot: binary proof[%d]: %w", j, err)
		}
		ciphertexts[j] = Ej
		randomness[j] = rj
		binaryProofs[j] = proof
	}

	sumNonce, err := crypto.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("ballot: sum nonce: %w", err)
	}
	sumProof, err := crypto.ProvePaillierSumToOne(bc.PublicKey, randomness, ciphertexts, sumNonce, electionID)
	if err != nil {
		return nil, fmt.Errorf("ballot: sum proof: %w", err)
	}

	return &Ballot{
		VoterID:        voterID,
		CandidateID:    candidateID,
		EncryptedVotes: ciphertexts,
		Randomness:     randomness,
		BinaryProofs:   binaryProofs,
		SumProof:       sumProof,
		Weight:         big.NewInt(1),
		Timestamp:      time.Now().Unix(),
	}, nil
}

// ApplyWeight raises every per-candidate ciphertext to the SMDC × duress weight
// via Paillier scalar multiplication. The returned slice is independent of the
// input (callers can mutate without affecting the source ballot).
//
// Pre-existing limitation: when weight = 0, every output collapses to 1
// (the multiplicative identity in Z_{n^2}^*), making the ballot deterministic
// and observable as a non-counting submission on chain. See WeightedVote doc.
func (bc *BallotCreator) ApplyWeight(encryptedVotes []*big.Int, weight *big.Int) []*big.Int {
	out := make([]*big.Int, len(encryptedVotes))
	for j, Ej := range encryptedVotes {
		out[j] = bc.PublicKey.Multiply(Ej, weight)
	}
	return out
}

// VerifyBallotZK runs every per-candidate binary proof and the sum-to-one
// proof against the ballot's original ciphertexts. Used at cast time for
// defence-in-depth and by chaincode at submission time.
func (bc *BallotCreator) VerifyBallotZK(
	ciphertexts []*big.Int,
	binaryProofs []*crypto.PaillierBinaryProof,
	sumProof *crypto.PaillierSumProof,
) bool {
	if len(ciphertexts) == 0 || len(ciphertexts) != len(binaryProofs) {
		return false
	}
	for j, E := range ciphertexts {
		if !crypto.VerifyPaillierBinary(bc.PublicKey, E, binaryProofs[j]) {
			return false
		}
	}
	return crypto.VerifyPaillierSumToOne(bc.PublicKey, ciphertexts, sumProof)
}
