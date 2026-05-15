package voting

import (
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/smdc"
)

// Ballot represents a voter's 1-hot ballot. EncryptedVotes is a length-m vector
// of Paillier ciphertexts, one per candidate, with v_j = 1 for the chosen
// candidate and 0 otherwise. Randomness is the per-candidate Paillier encryption
// randomness; it is required by the prover for the sum-to-one proof and MUST
// NOT leave the voter's device.
type Ballot struct {
	VoterID        string
	CandidateID    int
	EncryptedVotes []*big.Int                       // E_j = (1+n)^{v_j} r_j^n mod n^2, length m
	Randomness     []*big.Int                       // r_j per candidate (PRIVATE — prover only)
	BinaryProofs   []*crypto.PaillierBinaryProof    // per-candidate v_j ∈ {0,1}
	SumProof       *crypto.PaillierSumProof         // Σ v_j = 1
	Weight         *big.Int                         // From SMDC real slot (should be 1)
	Timestamp      int64
}

// WeightedVote represents the per-candidate ciphertexts after Paillier scalar
// multiplication by the voter's finalWeight (slot.Weight × behaviorWeight).
// OriginalCiphertexts is retained so verifiers can re-check the ZK proofs (which
// are bound to the unweighted E_j); EncryptedVotes is what feeds the tally.
//
// Note (pre-existing privacy limitation, also present in the legacy scalar
// implementation): when finalWeight = 0, every E_w_j collapses to the literal
// value 1 (since c^0 = 1 mod n^2). An observer can therefore distinguish a
// real-slot-real-duress submission from a fake-slot-real-duress submission by
// inspecting the weighted ciphertexts. Re-randomisation plus a linkage proof is
// required to close this gap and is identified as future work in Chapter~5.
type WeightedVote struct {
	VoterID             string
	EncryptedVotes      []*big.Int                       // E_w_j = E_j^finalWeight (for tally)
	OriginalCiphertexts []*big.Int                       // E_j (kept so proofs remain checkable)
	BinaryProofs        []*crypto.PaillierBinaryProof
	SumProof            *crypto.PaillierSumProof
	SMDCSlotIndex       int                              // Which SMDC slot was used
	SMDCCommitment      *big.Int                         // Public commitment of the slot
	RingSignature       *crypto.RingSignature            // Anonymous signature over ciphertext vector
	RingPublicKeys      []*big.Int                       // Fixed-size ring used for the signature
	Timestamp           int64
}

// CastVote represents a complete vote submission. VoteShares holds one SA²
// share per candidate (paired with Server A and Server B masks); tally
// aggregates each candidate position independently.
type CastVote struct {
	VoterID          string
	WeightedVote     *WeightedVote
	VoteShares       *sa2.VoteShare         // per-candidate shares stored inside
	MerkleProof      [][]byte               // Eligibility proof
	PublicCredential *smdc.PublicCredential // SMDC public credential
}

// VoteReceipt represents a receipt after vote submission
type VoteReceipt struct {
	VoterID        string
	ReceiptID      string
	Timestamp      int64
	BlockchainTxID string   // Hyperledger Fabric transaction ID
	KeyImage       *big.Int // For double-vote detection
}

// Candidate represents a candidate in the election
type Candidate struct {
	ID          int
	Name        string
	Description string
	Party       string
}

// Election represents an election
type Election struct {
	ElectionID  string
	Title       string
	Description string
	Candidates  []*Candidate
	StartTime   int64
	EndTime     int64
	IsActive    bool
}
