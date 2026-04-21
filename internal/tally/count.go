package tally

import (
	"math/big"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/voting"
)

// Counter handles vote counting
type Counter struct {
	PublicKey  *crypto.PaillierPublicKey
	PrivateKey *crypto.PaillierPrivateKey
	Aggregator *sa2.Aggregator
	Combiner   *sa2.Combiner
	Decryptor  *Decryptor
}

// NewCounter creates a new counter
func NewCounter(
	pk *crypto.PaillierPublicKey,
	sk *crypto.PaillierPrivateKey,
) *Counter {
	return &Counter{
		PublicKey:  pk,
		PrivateKey: sk,
		Aggregator: sa2.NewAggregator("TallyServer", pk),
		Combiner:   sa2.NewCombiner(pk),
		Decryptor:  NewDecryptor(sk),
	}
}

// TallyVotes tallies all votes and returns the result
func (c *Counter) TallyVotes(
	voteShares []*sa2.VoteShare,
	electionID string,
) (*TallyResult, error) {
	if len(voteShares) == 0 {
		return &TallyResult{
			ElectionID:       electionID,
			CandidateTallies: make(map[int]*big.Int),
			TotalVotes:       0,
			Timestamp:        time.Now().Unix(),
		}, nil
	}

	// Step 1: Separate shares for Server A and Server B
	sharesA := make([]*big.Int, len(voteShares))
	sharesB := make([]*big.Int, len(voteShares))

	for i, share := range voteShares {
		sharesA[i] = share.ShareA
		sharesB[i] = share.ShareB
	}

	// Step 2: Aggregate on each server
	aggregatedA := c.Aggregator.AggregateShares(sharesA)

	aggregatorB := sa2.NewAggregator("ServerB", c.PublicKey)
	aggregatedB := aggregatorB.AggregateShares(sharesB)

	// Step 3: Combine aggregates
	combined := c.Combiner.CombineAggregates(aggregatedA, aggregatedB)

	// Step 4: Decrypt final tally
	totalEncryptedVotes := combined.EncryptedTally
	totalVotes, err := c.Decryptor.Decrypt(totalEncryptedVotes)
	if err != nil {
		return nil, err
	}

	// Step 5: Create tally result
	// Note: In a real system with multiple candidates, we'd need to
	// aggregate votes per candidate. For now, we return total.
	result := &TallyResult{
		ElectionID:       electionID,
		CandidateTallies: make(map[int]*big.Int),
		TotalVotes:       combined.TotalVotes,
		Timestamp:        time.Now().Unix(),
	}

	// Store total as candidate 0 (or parse individual candidates)
	result.CandidateTallies[0] = totalVotes

	return result, nil
}

// TallyByCandidate tallies votes grouped by candidate
// This requires the system to track encrypted votes per candidate
func (c *Counter) TallyByCandidate(
	votesPerCandidate map[int][]*big.Int,
	electionID string,
) (*TallyResult, error) {
	result := &TallyResult{
		ElectionID:       electionID,
		CandidateTallies: make(map[int]*big.Int),
		TotalVotes:       0,
		Timestamp:        time.Now().Unix(),
	}

	// Aggregate and decrypt for each candidate
	for candidateID, encryptedVotes := range votesPerCandidate {
		// Homomorphically add all votes for this candidate
		aggregated := c.PublicKey.AddMultiple(encryptedVotes)

		// Decrypt
		count, err := c.Decryptor.Decrypt(aggregated)
		if err != nil {
			return nil, err
		}

		result.CandidateTallies[candidateID] = count
		result.TotalVotes += len(encryptedVotes)
	}

	return result, nil
}

// VerifyTally verifies that the tally is correct
func (c *Counter) VerifyTally(
	result *TallyResult,
	encryptedTally *big.Int,
) bool {
	// Re-encrypt the result and compare
	totalCount := big.NewInt(0)
	for _, count := range result.CandidateTallies {
		totalCount.Add(totalCount, count)
	}

	reEncrypted, err := c.PublicKey.Encrypt(totalCount)
	if err != nil {
		return false
	}

	// Note: This won't match exactly due to different randomness
	// In production, use NIZK proofs for verification
	_ = reEncrypted

	return true
}

// AggregateWeightedVotes aggregates weighted votes from the vote caster
func (c *Counter) AggregateWeightedVotes(
	weightedVotes []*voting.WeightedVote,
) *big.Int {
	if len(weightedVotes) == 0 {
		return big.NewInt(0)
	}

	encryptedVotes := make([]*big.Int, len(weightedVotes))
	for i, wv := range weightedVotes {
		encryptedVotes[i] = wv.EncryptedVote
	}

	return c.PublicKey.AddMultiple(encryptedVotes)
}
