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
	// DuressDetector stores each voter's behavioral duress signal. Registration
	// writes the signal here, and the reveal-slot-index flow verifies against
	// it before disclosing the real SMDC slot to the voter.
	DuressDetector   biometric.DuressDetector
	RegisteredVoters map[string]*Voter // voterID -> Voter
}

// NewRegistrationSystem creates a new registration system. smdcSecret is the
// server-held HMAC key used to derive real SMDC slot indices. duressDetector
// stores per-voter behavioral signals collected at registration; it may be nil
// in tests that do not exercise the reveal-slot-index flow.
func NewRegistrationSystem(
	pp *crypto.PedersenParams,
	rp *crypto.RingParams,
	k int,
	eligibleVoterIDs []string,
	electionID string,
	smdcSecret []byte,
	duressDetector biometric.DuressDetector,
) *RegistrationSystem {
	return &RegistrationSystem{
		FingerprintProcessor: biometric.NewFingerprintProcessor(),
		LivenessDetector:     biometric.NewLivenessDetector(0.7),
		SMDCGenerator:        smdc.NewSMDCGenerator(pp, k, electionID, smdcSecret),
		RingParams:           rp,
		MerkleTree:           NewMerkleTree(eligibleVoterIDs),
		DuressDetector:       duressDetector,
		RegisteredVoters:     make(map[string]*Voter),
	}
}

// RegisterVoter registers a new voter with fingerprint hash and a behavioral
// duress signal. The duress signal is mandatory: it gates the later
// RevealRealSlotIndex flow, and without it the voter cannot retrieve their
// real SMDC slot. The real slot index is computed at registration but is NOT
// returned in the response — the voter must later submit the registered
// signal to retrieve it.
// NOTE: Liveness check should be done by the caller before calling this function.
func (rs *RegistrationSystem) RegisterVoter(voterID string, fingerprintHash []byte, signalType, signalValue string) (*Voter, error) {
	timestamp := time.Now().Unix()

	if rs.DuressDetector == nil {
		return nil, errors.New("registration requires a duress detector")
	}

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

	// Step 7: Generate SMDC credential. realIndex is computed but intentionally
	// not surfaced — the voter retrieves it later by presenting their duress
	// signal to RevealRealSlotIndex.
	smdcCred, _, err := rs.SMDCGenerator.GenerateCredential(voterID)
	if err != nil {
		return nil, err
	}

	// Step 7a: Verify the freshly generated credential.
	// Validates the per-slot binary ZK proofs (w_i ∈ {0,1}) and the sum-one
	// proof (Σ w_i = 1) using Strong Fiat-Shamir (BPW 2012). Theorem 2
	// (Vote Validity / ZKP Soundness) reduces to this verification step;
	// without it the soundness claim is vacuous in the deployed system.
	if !rs.SMDCGenerator.VerifyCredential(smdcCred.GetPublicCredential()) {
		return nil, errors.New("smdc credential failed ZK verification at registration")
	}

	// Step 7b: Persist the behavioral duress signal. This gates the later
	// reveal-slot-index flow: the voter can only learn their real slot by
	// presenting this exact (signalType, signalValue).
	if _, err := rs.DuressDetector.SetSignal(voterID, signalType, signalValue); err != nil {
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
// and a mandatory behavioral duress signal. See RegisterVoter for the role of
// the signal in the reveal-slot-index flow.
func (rs *RegistrationSystem) RegisterVoterWithPassword(voterID string, passwordHash []byte, signalType, signalValue string) (*Voter, error) {
	if rs.DuressDetector == nil {
		return nil, errors.New("registration requires a duress detector")
	}

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

	// Step 4a: Verify the freshly generated credential (see RegisterVoter
	// for rationale). Strong Fiat-Shamir binary + sum-one proofs are
	// validated before the credential is persisted.
	if !rs.SMDCGenerator.VerifyCredential(smdcCred.GetPublicCredential()) {
		return nil, errors.New("smdc credential failed ZK verification at registration")
	}

	// Step 4b: Persist the behavioral duress signal (see RegisterVoter Step 7b).
	if _, err := rs.DuressDetector.SetSignal(voterID, signalType, signalValue); err != nil {
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

// RevealRealSlotIndex returns the voter's real SMDC slot index after verifying
// the presented behavioral duress signal matches the one stored at registration.
// This is the only path through which a voter learns which of their k slots
// carries weight = 1. A coercer who does not know the voter's signal cannot
// extract the index, and an attacker compromising the bulletin board cannot
// either (the index is derived from a server-held secret, not from public
// inputs).
func (rs *RegistrationSystem) RevealRealSlotIndex(voterID, signalType, signalValue string) (int, error) {
	if rs.DuressDetector == nil {
		return 0, errors.New("duress detector not configured")
	}
	if _, exists := rs.RegisteredVoters[voterID]; !exists {
		return 0, errors.New("voter not registered")
	}
	ok, err := rs.DuressDetector.VerifySignal(voterID, signalType, signalValue)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, errors.New("duress signal mismatch")
	}
	return rs.SMDCGenerator.DeriveRealIndex(voterID), nil
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
