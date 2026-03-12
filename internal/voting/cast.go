package voting

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/voter"
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
	KeyImageStore      KeyImageStore // persistent key-image storage (may be nil for legacy/test usage)

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

// CastVote handles the complete vote casting process.
//
// Concurrency strategy (double-checked locking):
//  1. Acquire RLock and check in-memory maps (fast path).
//  2. Release the lock and perform all expensive work (crypto, DB I/O)
//     OUTSIDE the mutex.
//  3. Persist the key image via KeyImageStore. The DB UNIQUE constraint
//     is the authoritative guard against races; if two goroutines pass
//     the in-memory check concurrently, only one will succeed at the DB.
//  4. Acquire full Lock and update the in-memory maps.
func (vc *VoteCaster) CastVote(voterID string, candidateID int, smdcSlotIndex int) (*VoteReceipt, error) {
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

	// Step 7: Apply SMDC weight to encrypted vote
	// E(vote)^weight = E(vote × weight)
	weightedEncryptedVote := vc.BallotCreator.ApplyWeight(ballot.EncryptedVote, slot.Weight)

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
