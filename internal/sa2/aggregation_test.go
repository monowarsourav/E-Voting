package sa2

import (
	"math/big"
	"testing"

	"github.com/covertvote/e-voting/internal/crypto"
)

func TestVoteSplitAndReconstruct(t *testing.T) {
	// Setup
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	// Encrypt a vote
	vote := big.NewInt(42)
	encVote, _ := pk.Encrypt(vote)

	// Split
	splitter := NewVoteSplitter(pk)
	share, err := splitter.SplitVote("voter1", encVote)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	// Reconstruct
	reconstructed := splitter.ReconstructVote(share)

	// Decrypt both and compare
	originalDecrypted, _ := sk.Decrypt(encVote)
	reconstructedDecrypted, _ := sk.Decrypt(reconstructed)

	if originalDecrypted.Cmp(reconstructedDecrypted) != 0 {
		t.Errorf("Reconstruction failed: expected %v, got %v", originalDecrypted, reconstructedDecrypted)
	}
}

func TestSA2FullFlow(t *testing.T) {
	// Setup
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	// Create multiple votes
	votes := []*big.Int{
		big.NewInt(1), // Vote for candidate 1
		big.NewInt(2), // Vote for candidate 2
		big.NewInt(1), // Vote for candidate 1
		big.NewInt(1), // Vote for candidate 1
	}

	splitter := NewVoteSplitter(pk)
	var sharesA []*big.Int
	var sharesB []*big.Int

	// Split all votes
	for i, vote := range votes {
		encVote, _ := pk.Encrypt(vote)
		share, _ := splitter.SplitVote("voter"+string(rune(i)), encVote)
		sharesA = append(sharesA, share.ShareA)
		sharesB = append(sharesB, share.ShareB)
	}

	// Aggregate at Server A
	aggA := NewAggregator("ServerA", pk)
	aggregatedA := aggA.AggregateShares(sharesA)

	// Aggregate at Server B
	aggB := NewAggregator("ServerB", pk)
	aggregatedB := aggB.AggregateShares(sharesB)

	// Combine
	combiner := NewCombiner(pk)
	result := combiner.CombineAggregates(aggregatedA, aggregatedB)

	// Decrypt final result
	tally, _ := sk.Decrypt(result.EncryptedTally)

	// Expected: 1+2+1+1 = 5
	expected := big.NewInt(5)
	if tally.Cmp(expected) != 0 {
		t.Errorf("Tally mismatch: expected %v, got %v", expected, tally)
	}

	if result.TotalVotes != 4 {
		t.Errorf("Vote count mismatch: expected 4, got %d", result.TotalVotes)
	}
}

func TestSA2Privacy(t *testing.T) {
	// This test demonstrates that individual shares don't reveal votes
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	vote := big.NewInt(100)
	encVote, _ := pk.Encrypt(vote)

	splitter := NewVoteSplitter(pk)
	share, _ := splitter.SplitVote("voter1", encVote)

	// Decrypt individual shares - should NOT equal the original vote
	shareADecrypted, _ := sk.Decrypt(share.ShareA)
	shareBDecrypted, _ := sk.Decrypt(share.ShareB)

	// ShareA should not equal the vote (it's masked)
	if shareADecrypted.Cmp(vote) == 0 {
		t.Error("ShareA should not reveal the original vote")
	}

	// But when we add the decrypted shares mod n, we get the vote
	sum := new(big.Int).Add(shareADecrypted, shareBDecrypted)
	sum.Mod(sum, pk.N)

	if sum.Cmp(vote) != 0 {
		t.Errorf("Sum of decrypted shares should equal vote: expected %v, got %v", vote, sum)
	}
}
