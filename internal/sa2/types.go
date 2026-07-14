package sa2

import (
	"math/big"
)

// VoteShare holds a voter's SA² split for every candidate position. SharesA[j]
// and SharesB[j] are sent to Server A and Server B respectively, and their
// reconstruction (homomorphic addition under Paillier) recovers the weighted
// per-candidate ciphertext E_w_j.
type VoteShare struct {
	VoterID string
	SharesA []*big.Int // per-candidate share for Server A; length = number of candidates
	SharesB []*big.Int // per-candidate share for Server B; length = number of candidates
}

// AggregatedShare represents aggregated shares from one server for a single
// candidate position.
type AggregatedShare struct {
	ServerID     string
	CandidateIdx int
	Value        *big.Int
	Count        int
}

// CombinedResult represents the final combined tally for one candidate.
type CombinedResult struct {
	CandidateIdx   int
	EncryptedTally *big.Int
	TotalVotes     int
}
