package smdc

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
)

// SMDCGenerator generates SMDC credentials
type SMDCGenerator struct {
	PedersenParams *crypto.PedersenParams
	K              int    // Number of slots
	ElectionID     string // Election context for proof binding
}

// NewSMDCGenerator creates a new SMDC generator
func NewSMDCGenerator(pp *crypto.PedersenParams, k int, electionID string) *SMDCGenerator {
	return &SMDCGenerator{
		PedersenParams: pp,
		K:              k,
		ElectionID:     electionID,
	}
}

// DeriveRealIndex deterministically derives the real slot index from voterID
// and electionID using HMAC. This allows the legitimate voter to re-derive
// their real index without it being stored in the credential struct.
func (gen *SMDCGenerator) DeriveRealIndex(voterID string) int {
	context := []byte("smdc-real-index:" + voterID + ":" + gen.ElectionID)
	mac := crypto.HMACSHA256([]byte(gen.ElectionID), context)
	// Use first 8 bytes as uint64, then reduce mod k
	idx := binary.BigEndian.Uint64(mac[:8])
	return int(idx % uint64(gen.K))
}

// GenerateCredential generates a new SMDC credential for a voter.
// Returns the credential and the real index separately.
// The real index is derived via HMAC(voterID, electionID) so the legitimate
// voter can re-derive it, but it is never stored in the credential struct.
func (gen *SMDCGenerator) GenerateCredential(voterID string) (*SMDCCredential, int, error) {
	k := gen.K
	pp := gen.PedersenParams

	// Step 1: Derive the real slot index via HMAC (deterministic, re-derivable)
	realIndex := gen.DeriveRealIndex(voterID)

	// Step 2: Create all k slots
	slots := make([]*CredentialSlot, k)
	commitments := make([]*crypto.Commitment, k)

	for i := 0; i < k; i++ {
		var weight *big.Int

		if i == realIndex {
			weight = big.NewInt(1) // Real slot
		} else {
			weight = big.NewInt(0) // Fake slot
		}

		// Create Pedersen commitment
		commitment, err := pp.Commit(weight)
		if err != nil {
			return nil, 0, err
		}

		// Generate a fresh nonce for each binary proof
		nonce, err := crypto.GenerateNonce()
		if err != nil {
			return nil, 0, err
		}

		// Create binary proof with nonce and election context
		binaryProof, err := pp.ProveBinary(weight, commitment.R, commitment.C, nonce, gen.ElectionID)
		if err != nil {
			return nil, 0, err
		}

		slots[i] = &CredentialSlot{
			Index:       i,
			Weight:      weight,
			Randomness:  commitment.R,
			Commitment:  commitment,
			BinaryProof: binaryProof,
		}

		commitments[i] = commitment
	}

	// Step 3: Create sum proof (Σweights = 1) with fresh nonce
	sumNonce, err := crypto.GenerateNonce()
	if err != nil {
		return nil, 0, err
	}
	sumProof, err := pp.ProveSumOne(commitments, sumNonce, gen.ElectionID)
	if err != nil {
		return nil, 0, err
	}

	return &SMDCCredential{
		VoterID:  voterID,
		K:        k,
		Slots:    slots,
		SumProof: sumProof,
	}, realIndex, nil
}

// GetPublicCredential extracts the public part of credential
func (cred *SMDCCredential) GetPublicCredential() *PublicCredential {
	commitments := make([]*big.Int, cred.K)
	binaryProofs := make([]*crypto.BinaryProof, cred.K)

	for i, slot := range cred.Slots {
		commitments[i] = slot.Commitment.C
		binaryProofs[i] = slot.BinaryProof
	}

	return &PublicCredential{
		VoterID:      cred.VoterID,
		K:            cred.K,
		Commitments:  commitments,
		BinaryProofs: binaryProofs,
		SumProof:     cred.SumProof,
	}
}

// GetRealSlot returns the real slot. The caller must supply the realIndex
// (derived via DeriveRealIndex) — it is never stored in the credential.
func (cred *SMDCCredential) GetRealSlot(realIndex int) (*CredentialSlot, error) {
	if realIndex < 0 || realIndex >= cred.K {
		return nil, errors.New("invalid real index")
	}
	return cred.Slots[realIndex], nil
}

// GetFakeSlot returns a fake slot (for coercion scenario).
// The caller must supply the realIndex so the method can reject attempts
// to return the real slot as fake.
func (cred *SMDCCredential) GetFakeSlot(index int, realIndex int) (*CredentialSlot, error) {
	if index == realIndex {
		return nil, errors.New("cannot return real slot as fake")
	}
	if index < 0 || index >= cred.K {
		return nil, errors.New("invalid slot index")
	}
	return cred.Slots[index], nil
}

// VerifyCredential verifies a public credential
func (gen *SMDCGenerator) VerifyCredential(pub *PublicCredential) bool {
	pp := gen.PedersenParams

	// Verify each binary proof
	for i := 0; i < pub.K; i++ {
		if !pp.VerifyBinary(pub.Commitments[i], pub.BinaryProofs[i]) {
			return false
		}
	}

	// Verify sum proof
	if !pp.VerifySumOne(pub.Commitments, pub.SumProof) {
		return false
	}

	return true
}
