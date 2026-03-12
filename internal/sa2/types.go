package sa2

import (
	"math/big"
)

// VoteShare represents a split vote share
type VoteShare struct {
	VoterID string
	ShareA  *big.Int // For Server A
	ShareB  *big.Int // For Server B
}

// AggregatedShare represents aggregated shares from one server
type AggregatedShare struct {
	ServerID string
	Value    *big.Int
	Count    int
}

// CombinedResult represents the final combined tally
type CombinedResult struct {
	EncryptedTally *big.Int
	TotalVotes     int
}
