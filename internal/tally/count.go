package tally

import (
	"errors"
	"math/big"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/voting"
)

// Counter handles vote counting under the per-candidate SA² aggregation model.
// Every voter contributes a single VoteShare whose SharesA/SharesB slices are
// indexed by candidate position; the counter walks each candidate column in
// parallel and decrypts the resulting per-candidate homomorphic sum.
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

// TallyVotes performs per-candidate homomorphic aggregation across all voters,
// reconstructs the SA² masks via the two-server combine step, and decrypts the
// resulting per-candidate ciphertexts to integer vote counts. numCandidates must
// equal the length of every voter's per-candidate share vector.
func (c *Counter) TallyVotes(
	voteShares []*sa2.VoteShare,
	electionID string,
	numCandidates int,
) (*TallyResult, error) {
	if numCandidates <= 0 {
		return nil, errors.New("tally: numCandidates must be positive")
	}
	result := &TallyResult{
		ElectionID:       electionID,
		CandidateTallies: make(map[int]*big.Int),
		TotalVotes:       len(voteShares),
		Timestamp:        time.Now().Unix(),
	}
	if len(voteShares) == 0 {
		for j := 0; j < numCandidates; j++ {
			result.CandidateTallies[j] = big.NewInt(0)
		}
		return result, nil
	}

	aggregatorB := sa2.NewAggregator("ServerB", c.PublicKey)

	for j := 0; j < numCandidates; j++ {
		// Gather column j of A and B shares from every voter.
		columnA := make([]*big.Int, 0, len(voteShares))
		columnB := make([]*big.Int, 0, len(voteShares))
		for _, share := range voteShares {
			if share == nil {
				continue
			}
			if j >= len(share.SharesA) || j >= len(share.SharesB) {
				return nil, errors.New("tally: voter share has wrong arity")
			}
			columnA = append(columnA, share.SharesA[j])
			columnB = append(columnB, share.SharesB[j])
		}

		aggregatedA := c.Aggregator.AggregateShares(j, columnA)
		aggregatedB := aggregatorB.AggregateShares(j, columnB)
		combined := c.Combiner.CombineAggregates(aggregatedA, aggregatedB)

		count, err := c.Decryptor.Decrypt(combined.EncryptedTally)
		if err != nil {
			return nil, err
		}
		result.CandidateTallies[j] = count
	}
	return result, nil
}

// TallyByCandidate is a convenience helper that homomorphically aggregates a
// caller-supplied per-candidate ciphertext map and decrypts it. It bypasses
// SA² and is primarily useful for tests or for off-chain analysis.
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
	for candidateID, encryptedVotes := range votesPerCandidate {
		aggregated := c.PublicKey.AddMultiple(encryptedVotes)
		count, err := c.Decryptor.Decrypt(aggregated)
		if err != nil {
			return nil, err
		}
		result.CandidateTallies[candidateID] = count
		result.TotalVotes += len(encryptedVotes)
	}
	return result, nil
}

// VerifyTally is a structural sanity check: every candidate column must have a
// decrypted count and the sum of counts must not exceed the total number of
// voters. NIZK-based result correctness verification is identified as future
// work.
func (c *Counter) VerifyTally(result *TallyResult, totalVoters int) bool {
	if result == nil {
		return false
	}
	totalCount := big.NewInt(0)
	for _, count := range result.CandidateTallies {
		if count == nil {
			return false
		}
		totalCount.Add(totalCount, count)
	}
	return totalCount.Cmp(big.NewInt(int64(totalVoters))) <= 0
}

// AggregateWeightedVotes performs per-candidate homomorphic addition over the
// per-candidate weighted ciphertexts pulled directly from each voter's record
// (i.e. without going through SA²). Returns one aggregated ciphertext per
// candidate. Primarily used in tests and as a sanity check against TallyVotes.
func (c *Counter) AggregateWeightedVotes(
	weightedVotes []*voting.WeightedVote,
	numCandidates int,
) []*big.Int {
	out := make([]*big.Int, numCandidates)
	for j := 0; j < numCandidates; j++ {
		column := make([]*big.Int, 0, len(weightedVotes))
		for _, wv := range weightedVotes {
			if wv == nil || j >= len(wv.EncryptedVotes) {
				continue
			}
			column = append(column, wv.EncryptedVotes[j])
		}
		if len(column) == 0 {
			out[j] = big.NewInt(1) // Paillier identity (encrypts 0)
			continue
		}
		out[j] = c.PublicKey.AddMultiple(column)
	}
	return out
}
