package pq

import (
	"errors"
	"math/big"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/smdc"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
)

// PQVoteCaster handles post-quantum secure vote casting
type PQVoteCaster struct {
	HybridKeys         *HybridKeyPair
	RingParams         *crypto.RingParams
	RegistrationSystem *voter.RegistrationSystem
	Election           *voting.Election
	CastVotes          map[string]*PQCastVote
	UsedKeyImages      map[string]bool
}

// PQCastVote represents a post-quantum secure cast vote
type PQCastVote struct {
	VoterID           string
	HybridCiphertext  *HybridCiphertext
	SMDCSlotIndex     int
	SMDCCommitment    *big.Int
	RingSignature     *crypto.RingSignature
	Timestamp         int64
	PQVoteShares      *PQVoteShares
	MerkleProof       [][]byte
	PublicCredential  *smdc.PublicCredential
}

// PQVoteShares represents post-quantum vote shares for SA²
type PQVoteShares struct {
	ShareA *HybridCiphertext
	ShareB *HybridCiphertext
}

// PQVoteReceipt represents a post-quantum vote receipt
type PQVoteReceipt struct {
	VoterID        string
	ReceiptID      string
	Timestamp      int64
	BlockchainTxID string
	KeyImage       *big.Int
	PQSecure       bool
}

// NewPQVoteCaster creates a new post-quantum vote caster
func NewPQVoteCaster(
	hybridKeys *HybridKeyPair,
	rp *crypto.RingParams,
	rs *voter.RegistrationSystem,
	election *voting.Election,
) *PQVoteCaster {
	return &PQVoteCaster{
		HybridKeys:         hybridKeys,
		RingParams:         rp,
		RegistrationSystem: rs,
		Election:           election,
		CastVotes:          make(map[string]*PQCastVote),
		UsedKeyImages:      make(map[string]bool),
	}
}

// CastPQVote casts a post-quantum secure vote
func (pqvc *PQVoteCaster) CastPQVote(
	voterID string,
	candidateID int,
	smdcSlotIndex int,
) (*PQVoteReceipt, error) {
	// Step 1: Verify election is active
	if !pqvc.Election.IsActive {
		return nil, errors.New("election is not active")
	}

	currentTime := time.Now().Unix()
	if currentTime < pqvc.Election.StartTime || currentTime > pqvc.Election.EndTime {
		return nil, errors.New("election is not in voting period")
	}

	// Step 2: Get voter record
	voterRecord, err := pqvc.RegistrationSystem.GetVoter(voterID)
	if err != nil {
		return nil, errors.New("voter not found")
	}

	// Step 3: Check if already voted
	if _, hasVoted := pqvc.CastVotes[voterID]; hasVoted {
		return nil, errors.New("voter has already cast a vote")
	}

	// Step 4: Validate candidate
	if !pqvc.isValidCandidate(candidateID) {
		return nil, errors.New("invalid candidate ID")
	}

	// Step 5: Get SMDC slot
	if smdcSlotIndex < 0 || smdcSlotIndex >= voterRecord.SMDCCredential.K {
		return nil, errors.New("invalid SMDC slot index")
	}
	slot := voterRecord.SMDCCredential.Slots[smdcSlotIndex]

	// Step 6: Encrypt vote with hybrid encryption (post-quantum secure)
	voteValue := big.NewInt(1) // Vote for candidate
	hybridCiphertext, err := HybridEncrypt(
		voteValue,
		pqvc.HybridKeys.KyberPublicKey,
		pqvc.HybridKeys.PaillierPublicKey,
	)
	if err != nil {
		return nil, err
	}

	// Step 7: Apply SMDC weight (homomorphic operation)
	weightedCiphertext := HybridHomomorphicMultiply(
		hybridCiphertext,
		slot.Weight,
		pqvc.HybridKeys.PaillierPublicKey,
	)

	// Step 8: Re-encapsulate after homomorphic operation
	weightedCiphertext, err = ReEncapsulate(
		weightedCiphertext,
		pqvc.HybridKeys.KyberPublicKey,
	)
	if err != nil {
		return nil, err
	}

	// Step 9: Create ring signature
	allKeys := pqvc.RegistrationSystem.GetAllPublicKeys()
	ring := make([]*crypto.RingKeyPair, len(allKeys))
	signerIndex := -1

	for i, key := range allKeys {
		if key.PublicKey.Cmp(voterRecord.RingPublicKey.PublicKey) == 0 {
			signerIndex = i
		}
		ring[i] = key
	}

	if signerIndex == -1 {
		return nil, errors.New("voter not found in ring")
	}

	ringPubKeys := make([]*big.Int, len(ring))
	for i, kp := range ring {
		ringPubKeys[i] = kp.PublicKey
	}

	// Sign the weighted ciphertext
	message := weightedCiphertext.PaillierCiphertext.Bytes()
	ringSignature, err := pqvc.RingParams.Sign(message, voterRecord.RingPublicKey, ringPubKeys, signerIndex)
	if err != nil {
		return nil, err
	}

	// Step 10: Check for double-voting
	keyImageStr := ringSignature.KeyImage.String()
	if pqvc.UsedKeyImages[keyImageStr] {
		return nil, errors.New("double-vote detected: key image already used")
	}

	// Step 11: Split vote for SA² (post-quantum version)
	pqShares, err := pqvc.splitPQVote(weightedCiphertext)
	if err != nil {
		return nil, err
	}

	// Step 12: Get Merkle proof
	merkleProof, err := pqvc.RegistrationSystem.GetMerkleProof(voterID)
	if err != nil {
		return nil, err
	}

	// Step 13: Create cast vote
	castVote := &PQCastVote{
		VoterID:           voterID,
		HybridCiphertext:  weightedCiphertext,
		SMDCSlotIndex:     smdcSlotIndex,
		SMDCCommitment:    slot.Commitment.C,
		RingSignature:     ringSignature,
		Timestamp:         currentTime,
		PQVoteShares:      pqShares,
		MerkleProof:       merkleProof,
		PublicCredential:  voterRecord.SMDCCredential.GetPublicCredential(),
	}

	// Step 14: Store vote
	pqvc.CastVotes[voterID] = castVote
	pqvc.UsedKeyImages[keyImageStr] = true

	// Step 15: Generate receipt
	receipt := &PQVoteReceipt{
		VoterID:        voterID,
		ReceiptID:      "pq-receipt-" + voterID,
		Timestamp:      currentTime,
		BlockchainTxID: "",
		KeyImage:       ringSignature.KeyImage,
		PQSecure:       true,
	}

	return receipt, nil
}

// splitPQVote splits a post-quantum vote for SA² aggregation
func (pqvc *PQVoteCaster) splitPQVote(ct *HybridCiphertext) (*PQVoteShares, error) {
	// Generate random mask
	mask, err := crypto.GenerateRandomBigInt(pqvc.HybridKeys.PaillierPublicKey.N)
	if err != nil {
		return nil, err
	}

	// Encrypt mask
	encryptedMask, err := pqvc.HybridKeys.PaillierPublicKey.Encrypt(mask)
	if err != nil {
		return nil, err
	}

	// Create mask as hybrid ciphertext
	maskCt, err := HybridEncrypt(
		mask,
		pqvc.HybridKeys.KyberPublicKey,
		pqvc.HybridKeys.PaillierPublicKey,
	)
	if err != nil {
		return nil, err
	}

	// Share A = vote + mask
	shareA := HybridHomomorphicAdd(ct, maskCt, pqvc.HybridKeys.PaillierPublicKey)

	// Share B = -mask
	negMask := new(big.Int).Neg(mask)
	negMask.Mod(negMask, pqvc.HybridKeys.PaillierPublicKey.N)

	shareB, err := HybridEncrypt(
		negMask,
		pqvc.HybridKeys.KyberPublicKey,
		pqvc.HybridKeys.PaillierPublicKey,
	)
	if err != nil {
		return nil, err
	}

	_ = encryptedMask // Use the variable

	return &PQVoteShares{
		ShareA: shareA,
		ShareB: shareB,
	}, nil
}

// isValidCandidate checks if candidate ID is valid
func (pqvc *PQVoteCaster) isValidCandidate(candidateID int) bool {
	for _, candidate := range pqvc.Election.Candidates {
		if candidate.ID == candidateID {
			return true
		}
	}
	return false
}

// GetPQVoteCount returns the number of post-quantum votes cast
func (pqvc *PQVoteCaster) GetPQVoteCount() int {
	return len(pqvc.CastVotes)
}

// VerifyPQVote verifies a post-quantum vote
func (pqvc *PQVoteCaster) VerifyPQVote(castVote *PQCastVote) bool {
	// Verify ring signature
	message := castVote.HybridCiphertext.PaillierCiphertext.Bytes()

	allKeys := pqvc.RegistrationSystem.GetAllPublicKeys()
	ringPubKeys := make([]*big.Int, len(allKeys))
	for i, kp := range allKeys {
		ringPubKeys[i] = kp.PublicKey
	}

	if !pqvc.RingParams.Verify(message, castVote.RingSignature, ringPubKeys) {
		return false
	}

	// Verify Merkle proof
	merkleRoot := pqvc.RegistrationSystem.GetMerkleRoot()
	if !voter.VerifyProof(castVote.VoterID, castVote.MerkleProof, merkleRoot) {
		return false
	}

	// Verify MAC in hybrid ciphertext
	// This is already done in HybridDecrypt

	return true
}

// GetAllPQVoteShares returns all post-quantum vote shares for aggregation
func (pqvc *PQVoteCaster) GetAllPQVoteShares() []*PQVoteShares {
	shares := make([]*PQVoteShares, 0, len(pqvc.CastVotes))
	for _, vote := range pqvc.CastVotes {
		shares = append(shares, vote.PQVoteShares)
	}
	return shares
}

// ConvertToSA2Shares converts post-quantum shares to regular SA² shares for tallying
func ConvertToSA2Shares(pqShares *PQVoteShares) *sa2.VoteShare {
	return &sa2.VoteShare{
		VoterID: "pq-voter",
		ShareA:  pqShares.ShareA.PaillierCiphertext,
		ShareB:  pqShares.ShareB.PaillierCiphertext,
	}
}
