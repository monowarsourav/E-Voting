package voter

import (
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/smdc"
)

// Voter represents a registered voter
type Voter struct {
	VoterID         string
	NID             string // National ID
	FingerprintHash []byte // Optional: for biometric authentication
	PasswordHash    []byte // Optional: for password authentication
	RingPublicKey   *crypto.RingKeyPair
	SMDCCredential  *smdc.SMDCCredential
	RegistrationTimestamp int64
	IsEligible      bool
}

// VoterRegistration represents registration data
type VoterRegistration struct {
	NID             string
	FingerprintData *biometric.FingerprintData
	LivenessResult  *biometric.LivenessResult
}

// VoterEligibility represents eligibility verification
type VoterEligibility struct {
	VoterID    string
	MerkleRoot []byte
	MerkleProof [][]byte
	IsEligible bool
}
