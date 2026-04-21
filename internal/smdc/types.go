package smdc

import (
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
)

// SMDCCredential represents a voter's SMDC credential set
type SMDCCredential struct {
	VoterID  string            // Voter identifier
	K        int               // Number of slots
	Slots    []*CredentialSlot // All k slots
	SumProof *crypto.SumProof  // Proof that weights sum to 1
}

// CredentialSlot represents one slot in SMDC
type CredentialSlot struct {
	Index       int                 // Slot index (0 to k-1)
	Weight      *big.Int            // 0 or 1 (SECRET!)
	Randomness  *big.Int            // Pedersen randomness (SECRET!)
	Commitment  *crypto.Commitment  // Public commitment
	BinaryProof *crypto.BinaryProof // Proof that weight ∈ {0,1}
}

// PublicCredential is what gets published (no secrets)
type PublicCredential struct {
	VoterID      string
	K            int
	Commitments  []*big.Int
	BinaryProofs []*crypto.BinaryProof
	SumProof     *crypto.SumProof
}
