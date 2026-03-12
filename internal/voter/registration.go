package voter

import (
	"errors"
	"time"

	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/smdc"
)

// RegistrationSystem handles voter registration
type RegistrationSystem struct {
	FingerprintProcessor *biometric.FingerprintProcessor
	LivenessDetector     *biometric.LivenessDetector
	SMDCGenerator        *smdc.SMDCGenerator
	RingParams           *crypto.RingParams
	MerkleTree           *MerkleTree
	RegisteredVoters     map[string]*Voter // voterID -> Voter
}

// NewRegistrationSystem creates a new registration system
func NewRegistrationSystem(
	pp *crypto.PedersenParams,
	rp *crypto.RingParams,
	k int,
	eligibleVoterIDs []string,
	electionID string,
) *RegistrationSystem {
	return &RegistrationSystem{
		FingerprintProcessor: biometric.NewFingerprintProcessor(),
		LivenessDetector:     biometric.NewLivenessDetector(0.7),
		SMDCGenerator:        smdc.NewSMDCGenerator(pp, k, electionID),
		RingParams:           rp,
		MerkleTree:           NewMerkleTree(eligibleVoterIDs),
		RegisteredVoters:     make(map[string]*Voter),
	}
}

// RegisterVoter registers a new voter with fingerprint hash
// NOTE: Liveness check should be done by the caller before calling this function
func (rs *RegistrationSystem) RegisterVoter(voterID string, fingerprintHash []byte) (*Voter, error) {
	timestamp := time.Now().Unix()

	// Step 4: Check if already registered
	if _, exists := rs.RegisteredVoters[voterID]; exists {
		return nil, errors.New("voter already registered")
	}

	// Step 5: Check eligibility (from Merkle tree)
	isEligible := rs.MerkleTree.IsEligible(voterID)
	if !isEligible {
		return nil, errors.New("voter not eligible")
	}

	// Step 6: Generate ring signature key pair
	ringKeyPair, err := rs.RingParams.GenerateRingKeyPair()
	if err != nil {
		return nil, err
	}

	// Step 7: Generate SMDC credential
	smdcCred, _, err := rs.SMDCGenerator.GenerateCredential(voterID)
	if err != nil {
		return nil, err
	}

	// Step 8: Create voter record
	voter := &Voter{
		VoterID:               voterID,
		NID:                   voterID, // Use voterID as NID for biometric registration
		FingerprintHash:       fingerprintHash,
		RingPublicKey:         ringKeyPair,
		SMDCCredential:        smdcCred,
		RegistrationTimestamp: timestamp,
		IsEligible:            true,
	}

	// Step 9: Store voter
	rs.RegisteredVoters[voterID] = voter

	return voter, nil
}

// RegisterVoterWithPassword registers a new voter using password authentication
func (rs *RegistrationSystem) RegisterVoterWithPassword(voterID string, passwordHash []byte) (*Voter, error) {
	// Step 1: Check if already registered
	if _, exists := rs.RegisteredVoters[voterID]; exists {
		return nil, errors.New("voter already registered")
	}

	// Step 2: Check eligibility (from Merkle tree)
	isEligible := rs.MerkleTree.IsEligible(voterID)
	if !isEligible {
		return nil, errors.New("voter not eligible")
	}

	// Step 3: Generate ring signature key pair
	ringKeyPair, err := rs.RingParams.GenerateRingKeyPair()
	if err != nil {
		return nil, err
	}

	// Step 4: Generate SMDC credential
	smdcCred, _, err := rs.SMDCGenerator.GenerateCredential(voterID)
	if err != nil {
		return nil, err
	}

	// Step 5: Create voter record
	timestamp := time.Now().Unix()
	voter := &Voter{
		VoterID:               voterID,
		NID:                   voterID,
		PasswordHash:          passwordHash,
		RingPublicKey:         ringKeyPair,
		SMDCCredential:        smdcCred,
		RegistrationTimestamp: timestamp,
		IsEligible:            true,
	}

	// Step 6: Store voter
	rs.RegisteredVoters[voterID] = voter

	return voter, nil
}

// GetVoter retrieves a registered voter
func (rs *RegistrationSystem) GetVoter(voterID string) (*Voter, error) {
	voter, exists := rs.RegisteredVoters[voterID]
	if !exists {
		return nil, errors.New("voter not found")
	}
	return voter, nil
}

// VerifyVoter verifies a voter's fingerprint
func (rs *RegistrationSystem) VerifyVoter(voterID string, fingerprintData []byte) bool {
	voter, err := rs.GetVoter(voterID)
	if err != nil {
		return false
	}

	return rs.FingerprintProcessor.VerifyFingerprint(fingerprintData, voter.FingerprintHash)
}

// GetMerkleProof gets the Merkle proof for a voter
func (rs *RegistrationSystem) GetMerkleProof(voterID string) ([][]byte, error) {
	return rs.MerkleTree.GetProof(voterID)
}

// GetMerkleRoot returns the Merkle root
func (rs *RegistrationSystem) GetMerkleRoot() []byte {
	return rs.MerkleTree.Root
}

// GetAllPublicKeys returns all registered voter public keys (for ring)
func (rs *RegistrationSystem) GetAllPublicKeys() []*crypto.RingKeyPair {
	var keys []*crypto.RingKeyPair
	for _, voter := range rs.RegisteredVoters {
		keys = append(keys, voter.RingPublicKey)
	}
	return keys
}
