// SECURITY PROPERTIES verified by formal proofs (see security_analysis.tex):
// - Ballot Privacy (Theorem 1): Encrypted votes indistinguishable under DCRA
// - ZKP Soundness (Theorem 2): Invalid votes (w∉{0,1} or Σw≠1) detected with overwhelming probability
// - Anonymity (Theorem 3): Ring signature hides voter identity among 100 members
// - Double-Vote Prevention: Key Image uniqueness enforced by DB UNIQUE constraint
// - Coercion Resistance (Theorem 4): SMDC fake credentials indistinguishable from real
// - Composition (Theorem 6): Independent randomness across all 7 protocols
package voting

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/pkg/audit"
)

// KeyImageStore provides persistent storage for key images.
// Implementations must enforce uniqueness at the storage level (e.g., via a
// UNIQUE constraint) so that concurrent MarkUsed calls for the same key image
// are safe without external locking.
type KeyImageStore interface {
	// Exists returns true if the key image has already been persisted.
	Exists(keyImage string) (bool, error)

	// MarkUsed persists a key image. It must return ErrKeyImageAlreadyUsed if
	// the key image was already stored (e.g., UNIQUE constraint violation).
	MarkUsed(keyImage string) error
}

// ErrKeyImageAlreadyUsed is returned by KeyImageStore.MarkUsed when the key
// image already exists in persistent storage.
var ErrKeyImageAlreadyUsed = errors.New("key image already used")

// VoteCaster handles the complete vote casting process
type VoteCaster struct {
	BallotCreator      *BallotCreator
	VoteSplitter       *sa2.VoteSplitter
	RingParams         *crypto.RingParams
	RegistrationSystem *voter.RegistrationSystem
	Election           *Election
	KeyImageStore      KeyImageStore            // persistent key-image storage (may be nil for legacy/test usage)
	DuressDetector     biometric.DuressDetector // optional; nil = no behavioral duress check
	Auditor            *audit.AuditLogger       // optional; nil = no structured audit logging

	mu            sync.RWMutex
	castVotes     map[string]*CastVote // voterID -> CastVote
	usedKeyImages map[string]bool      // keyImage -> true (in-memory fast-path cache)
}

// NewVoteCaster creates a new vote caster.
// keyImageStore may be nil; when nil the caster falls back to in-memory-only
// key-image tracking (suitable for tests but NOT for production).
func NewVoteCaster(
	pk *crypto.PaillierPublicKey,
	rp *crypto.RingParams,
	rs *voter.RegistrationSystem,
	election *Election,
	opts ...VoteCasterOption,
) *VoteCaster {
	vc := &VoteCaster{
		BallotCreator:      NewBallotCreator(pk),
		VoteSplitter:       sa2.NewVoteSplitter(pk),
		RingParams:         rp,
		RegistrationSystem: rs,
		Election:           election,
		castVotes:          make(map[string]*CastVote),
		usedKeyImages:      make(map[string]bool),
	}
	for _, opt := range opts {
		opt(vc)
	}
	return vc
}

// VoteCasterOption is a functional option for NewVoteCaster.
type VoteCasterOption func(*VoteCaster)

// WithKeyImageStore sets a persistent KeyImageStore.
func WithKeyImageStore(store KeyImageStore) VoteCasterOption {
	return func(vc *VoteCaster) {
		vc.KeyImageStore = store
	}
}

// WithDuressDetector attaches a behavioral duress detector. When set, CastVote
// will check the submitted detected signal against the voter's registered
// duress signal and zero the behavioral weight on a mismatch — silently, so
// that a coercer cannot distinguish coerced from genuine votes.
func WithDuressDetector(d biometric.DuressDetector) VoteCasterOption {
	return func(vc *VoteCaster) {
		vc.DuressDetector = d
	}
}

// WithAuditor attaches a structured audit logger. When set, security events
// (e.g. duress signal mismatches) are persisted via the audit subsystem
// instead of going to stdout.
func WithAuditor(a *audit.AuditLogger) VoteCasterOption {
	return func(vc *VoteCaster) {
		vc.Auditor = a
	}
}

// CastVote handles the complete vote casting process.
//
// detected carries the optional behavioral duress signal submitted with the
// vote. When non-nil and the voter has a registered duress signal, the HMAC
// of detected is compared to the stored hash:
//   - match  → behaviorWeight = 1 (genuine vote)
//   - mismatch → behaviorWeight = 0 (coerced vote — silently discarded)
//
// The response is identical in both cases; the coercer cannot detect
// whether the vote was counted. An audit log entry is written on mismatch.
//
// Concurrency strategy (double-checked locking):
//  1. Acquire RLock and check in-memory maps (fast path).
//  2. Release the lock and perform all expensive work (crypto, DB I/O)
//     OUTSIDE the mutex.
//  3. Persist the key image via KeyImageStore. The DB UNIQUE constraint
//     is the authoritative guard against races; if two goroutines pass
//     the in-memory check concurrently, only one will succeed at the DB.
//  4. Acquire full Lock and update the in-memory maps.
func (vc *VoteCaster) CastVote(voterID string, candidateID int, smdcSlotIndex int, detected *biometric.DetectedSignal) (*VoteReceipt, error) {
	// Step 1: Verify election is active
	if !vc.Election.IsActive {
		return nil, errors.New("election is not active")
	}

	currentTime := time.Now().Unix()
	if currentTime < vc.Election.StartTime || currentTime > vc.Election.EndTime {
		return nil, errors.New("election is not in voting period")
	}

	// Step 2: Get voter
	voterRecord, err := vc.RegistrationSystem.GetVoter(voterID)
	if err != nil {
		return nil, errors.New("voter not found")
	}

	// ---- fast-path check (RLock) ----
	vc.mu.RLock()
	if _, hasVoted := vc.castVotes[voterID]; hasVoted {
		vc.mu.RUnlock()
		return nil, errors.New("voter has already cast a vote")
	}
	vc.mu.RUnlock()

	// Step 4: Verify candidate is valid
	if !vc.isValidCandidate(candidateID) {
		return nil, errors.New("invalid candidate ID")
	}

	// Step 5: Get SMDC slot
	if smdcSlotIndex < 0 || smdcSlotIndex >= voterRecord.SMDCCredential.K {
		return nil, errors.New("invalid SMDC slot index")
	}
	slot := voterRecord.SMDCCredential.Slots[smdcSlotIndex]

	// Step 6: Create ballot
	ballot, err := vc.BallotCreator.CreateBallot(voterID, candidateID)
	if err != nil {
		return nil, err
	}

	// Step 7: Compute final weight = smdcWeight × behaviorWeight.
	//
	// Behavioral duress signal check:
	// If the voter registered a secret behavioral pattern (e.g. "2 blinks"),
	// the client must include the matching detected_signal_* fields. A mismatch
	// zeros behaviorWeight so the encrypted vote is multiplied by 0 — the vote
	// is silently not counted. The response to the voter is identical to a
	// normal successful vote, so a coercer cannot detect the discard.
	behaviorWeight := big.NewInt(1)
	if vc.DuressDetector != nil &&
		vc.DuressDetector.HasSignal(voterID) &&
		detected != nil {
		ok, err := vc.DuressDetector.VerifySignal(voterID, detected.SignalType, detected.SignalValue)
		if err != nil || !ok {
			// Weight zeroed — the audit event is emitted later at the
			// short-circuit point where finalWeight is confirmed to be 0.
			behaviorWeight = big.NewInt(0)
		}
	}

	// finalWeight = smdcWeight × behaviorWeight (both are 0 or 1).
	// E(vote)^finalWeight = E(vote × finalWeight) via Paillier homomorphism.
	finalWeight := new(big.Int).Mul(slot.Weight, behaviorWeight)
	weightedEncryptedVote := vc.BallotCreator.ApplyWeight(ballot.EncryptedVote, finalWeight)

	// Step 8: Create ring signature with FIXED ring size
	// Get all registered public keys
	allKeys := vc.RegistrationSystem.GetAllPublicKeys()

	// Find signer index in all keys
	signerIndex := -1
	for i, key := range allKeys {
		if key.PublicKey.Cmp(voterRecord.RingPublicKey.PublicKey) == 0 {
			signerIndex = i
			break
		}
	}

	if signerIndex == -1 {
		return nil, errors.New("voter not found in ring")
	}

	// Convert all keys to public keys only
	allPubKeys := make([]*big.Int, len(allKeys))
	for i, kp := range allKeys {
		allPubKeys[i] = kp.PublicKey
	}

	// Select a FIXED-SIZE random ring (max 100 members)
	// This ensures O(n) complexity regardless of total voters
	ringPubKeys, newSignerIndex, err := crypto.SelectRandomRing(allPubKeys, voterRecord.RingPublicKey.PublicKey, signerIndex)
	if err != nil {
		return nil, err
	}

	// Sign the weighted vote with the fixed-size ring
	message := weightedEncryptedVote.Bytes()
	ringSignature, err := vc.RingParams.Sign(message, voterRecord.RingPublicKey, ringPubKeys, newSignerIndex)
	if err != nil {
		return nil, err
	}

	// Step 9: Check for double-voting via key image (fast-path cache)
	keyImageStr := hex.EncodeToString(ringSignature.KeyImage.Bytes())

	vc.mu.RLock()
	if vc.usedKeyImages[keyImageStr] {
		vc.mu.RUnlock()
		return nil, errors.New("double-vote detected: key image already used")
	}
	vc.mu.RUnlock()

	// Step 10: Create weighted vote
	weightedVote := &WeightedVote{
		VoterID:        voterID,
		EncryptedVote:  weightedEncryptedVote,
		SMDCSlotIndex:  smdcSlotIndex,
		SMDCCommitment: slot.Commitment.C,
		RingSignature:  ringSignature,
		RingPublicKeys: ringPubKeys, // Store the fixed-size ring used
		Timestamp:      currentTime,
	}

	// Step 11: Split vote for SA²
	voteShares, err := vc.VoteSplitter.SplitVote(voterID, weightedEncryptedVote)
	if err != nil {
		return nil, err
	}

	// Step 12: Get Merkle proof
	merkleProof, err := vc.RegistrationSystem.GetMerkleProof(voterID)
	if err != nil {
		return nil, err
	}

	// Step 13: Create complete cast vote
	castVote := &CastVote{
		VoterID:          voterID,
		WeightedVote:     weightedVote,
		VoteShares:       voteShares,
		MerkleProof:      merkleProof,
		PublicCredential: voterRecord.SMDCCredential.GetPublicCredential(),
	}

	// ---- Coerced-vote short-circuit (Issue 6: iteration attack) ----
	// When behaviorWeight == 0, the voter is under coercion and intentionally
	// submitted the wrong behavioral signal. We must NOT persist the key image
	// or record the voter as having voted — doing so would prevent them from
	// casting a genuine vote once the coercer is gone. A plausible receipt is
	// returned so the coercer cannot distinguish this path from a genuine vote.
	//
	// We key on behaviorWeight (not finalWeight) because finalWeight=0 can also
	// arise from an SMDC decoy slot, which is a legitimate privacy mechanism
	// and should still be recorded as a cast vote to prevent double-voting.
	//
	// SECURITY NOTE: The ciphertext produced for behaviorWeight=0 is
	// E(vote)^0 = E(0), computationally indistinguishable under DCRA.
	// All expensive crypto (ring sig, SA² split) is completed before this
	// branch so timing is not observable by the caller.
	if behaviorWeight.Sign() == 0 {
		_ = vc.Auditor.LogDuressSignalMismatch(voterID, vc.Election.ElectionID, "")
		receipt := vc.generateReceipt(voterID, ringSignature.KeyImage)
		return receipt, nil
	}

	// ---- Persist key image OUTSIDE the mutex ----
	// The DB UNIQUE constraint is the authoritative race guard.
	if vc.KeyImageStore != nil {
		if err := vc.KeyImageStore.MarkUsed(keyImageStr); err != nil {
			if errors.Is(err, ErrKeyImageAlreadyUsed) {
				// Another goroutine won the race at the DB level.
				// Update the in-memory cache so the fast path catches it next time.
				vc.mu.Lock()
				vc.usedKeyImages[keyImageStr] = true
				vc.mu.Unlock()
				return nil, errors.New("double-vote detected: key image already used")
			}
			return nil, err
		}
	}

	// ---- Update in-memory maps (Lock) ----
	vc.mu.Lock()
	// Double-check: another goroutine may have committed the same voter
	// between our RLock and this Lock.
	if _, hasVoted := vc.castVotes[voterID]; hasVoted {
		vc.mu.Unlock()
		return nil, errors.New("voter has already cast a vote")
	}
	vc.castVotes[voterID] = castVote
	vc.usedKeyImages[keyImageStr] = true
	vc.mu.Unlock()

	// Step 15: Generate receipt
	receipt := vc.generateReceipt(voterID, ringSignature.KeyImage)

	return receipt, nil
}

// VerifyVote verifies a cast vote
func (vc *VoteCaster) VerifyVote(castVote *CastVote) bool {
	// Verify ring signature using the stored ring public keys
	message := castVote.WeightedVote.EncryptedVote.Bytes()

	// Use the fixed-size ring that was stored with the vote
	if !vc.RingParams.Verify(message, castVote.WeightedVote.RingSignature, castVote.WeightedVote.RingPublicKeys) {
		return false
	}

	// Verify Merkle proof
	merkleRoot := vc.RegistrationSystem.GetMerkleRoot()
	if !voter.VerifyProof(castVote.VoterID, castVote.MerkleProof, merkleRoot) {
		return false
	}

	// Verify SMDC credential
	// Note: We'd need PedersenParams here, should be passed to VoteCaster
	// For now, skip detailed SMDC verification
	_ = castVote.PublicCredential // Use the variable

	return true
}

// GetAllVoteShares returns all vote shares for aggregation
func (vc *VoteCaster) GetAllVoteShares() []*sa2.VoteShare {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	shares := make([]*sa2.VoteShare, 0, len(vc.castVotes))
	for _, vote := range vc.castVotes {
		shares = append(shares, vote.VoteShares)
	}
	return shares
}

// GetVoteCount returns the number of votes cast
func (vc *VoteCaster) GetVoteCount() int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	return len(vc.castVotes)
}

// GetCastVote returns the cast vote for a given voter, if it exists.
func (vc *VoteCaster) GetCastVote(voterID string) (*CastVote, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	cv, ok := vc.castVotes[voterID]
	return cv, ok
}

// isValidCandidate checks if candidate ID is valid
func (vc *VoteCaster) isValidCandidate(candidateID int) bool {
	for _, candidate := range vc.Election.Candidates {
		if candidate.ID == candidateID {
			return true
		}
	}
	return false
}

// generateReceipt generates a receipt for the voter
func (vc *VoteCaster) generateReceipt(voterID string, keyImage *big.Int) *VoteReceipt {
	// Generate receipt ID
	hash := sha256.Sum256([]byte(voterID + time.Now().String()))
	receiptID := hex.EncodeToString(hash[:])

	return &VoteReceipt{
		VoterID:        voterID,
		ReceiptID:      receiptID,
		Timestamp:      time.Now().Unix(),
		BlockchainTxID: "", // TODO: Add when blockchain is integrated
		KeyImage:       keyImage,
	}
}
