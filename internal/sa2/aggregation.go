// Package sa2 implements Samplable Anonymous Aggregation (SA²) for private vote tallying.
//
// THREAT MODEL:
// - Adversary: PPT, Dolev-Yao network model
// - Corruption: At most ONE of {SA²-Leader, SA²-Helper} may be corrupted
// - Non-collusion: Leader and Helper MUST be operated by independent administrative domains
// - Security guarantee: If at least one server is honest, individual vote shares are
//   information-theoretically hidden from the adversary
// - Mask cancellation: share_A = E(vote + mask), share_B = E(-mask)
//   Combined: E(vote + mask) × E(-mask) = E(vote) (Paillier homomorphic property)
//
// SECURITY REQUIREMENT: The Leader and Helper aggregation servers MUST be deployed
// on separate machines or containers managed by independent administrative domains.
// The Prio/SA² security model requires that at most one server is compromised.
// Running both servers in the same process negates this guarantee entirely.
//
// DEPLOYMENT REQUIREMENT: Use docker-compose-sa2.yml for container separation.
// See: Talwar et al., "Samplable Anonymous Aggregation", ACM CCS 2024
package sa2

import (
	"math/big"

	"github.com/covertvote/e-voting/internal/crypto"
)

// Aggregator handles aggregation of vote shares
type Aggregator struct {
	ServerID  string
	PublicKey *crypto.PaillierPublicKey
}

// NewAggregator creates a new aggregator for a server
func NewAggregator(serverID string, pk *crypto.PaillierPublicKey) *Aggregator {
	return &Aggregator{
		ServerID:  serverID,
		PublicKey: pk,
	}
}

// AggregateShares aggregates multiple shares for one server
// Returns the homomorphic sum of all shares
func (agg *Aggregator) AggregateShares(shares []*big.Int) *AggregatedShare {
	pk := agg.PublicKey

	// Start with identity element for multiplication (which is 1 for Paillier)
	aggregated := big.NewInt(1)

	// Homomorphically add all shares
	for _, share := range shares {
		aggregated = pk.Add(aggregated, share)
	}

	return &AggregatedShare{
		ServerID: agg.ServerID,
		Value:    aggregated,
		Count:    len(shares),
	}
}

// Combiner combines shares from multiple servers
type Combiner struct {
	PublicKey *crypto.PaillierPublicKey
}

// NewCombiner creates a new combiner
func NewCombiner(pk *crypto.PaillierPublicKey) *Combiner {
	return &Combiner{
		PublicKey: pk,
	}
}

// CombineAggregates combines aggregated shares from Server A and Server B
// final = agg_A × agg_B
// All masks cancel out, leaving E(Σ votes)
func (c *Combiner) CombineAggregates(aggA, aggB *AggregatedShare) *CombinedResult {
	pk := c.PublicKey

	// Verify counts match
	totalVotes := aggA.Count
	if aggA.Count != aggB.Count {
		// In production, handle this error properly
		if aggA.Count < aggB.Count {
			totalVotes = aggA.Count
		} else {
			totalVotes = aggB.Count
		}
	}

	// Combine: E(sum + Σmask) × E(-Σmask) = E(sum)
	encryptedTally := pk.Add(aggA.Value, aggB.Value)

	return &CombinedResult{
		EncryptedTally: encryptedTally,
		TotalVotes:     totalVotes,
	}
}
