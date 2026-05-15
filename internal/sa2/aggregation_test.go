package sa2

import (
	"math/big"
	"testing"

	"github.com/covertvote/e-voting/internal/crypto"
)

// singletonCiphertextVec wraps a single ciphertext into a one-candidate vector
// so existing tests can exercise the new per-candidate API without rewriting
// the underlying vote semantics.
func singletonCiphertextVec(e *big.Int) []*big.Int {
	return []*big.Int{e}
}

func TestVoteSplitAndReconstruct(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	vote := big.NewInt(42)
	encVote, _ := pk.Encrypt(vote)

	splitter := NewVoteSplitter(pk)
	share, err := splitter.SplitVote("voter1", singletonCiphertextVec(encVote))
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	reconstructed, err := splitter.ReconstructVote(share)
	if err != nil {
		t.Fatalf("Reconstruct failed: %v", err)
	}
	if len(reconstructed) != 1 {
		t.Fatalf("expected one-element reconstruction, got %d", len(reconstructed))
	}

	originalDecrypted, _ := sk.Decrypt(encVote)
	reconstructedDecrypted, _ := sk.Decrypt(reconstructed[0])
	if originalDecrypted.Cmp(reconstructedDecrypted) != 0 {
		t.Errorf("Reconstruction failed: expected %v, got %v", originalDecrypted, reconstructedDecrypted)
	}
}

func TestSA2FullFlow(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	votes := []*big.Int{
		big.NewInt(1),
		big.NewInt(2),
		big.NewInt(1),
		big.NewInt(1),
	}

	splitter := NewVoteSplitter(pk)
	var columnA []*big.Int
	var columnB []*big.Int

	for i, vote := range votes {
		encVote, _ := pk.Encrypt(vote)
		share, _ := splitter.SplitVote("voter"+string(rune(i)), singletonCiphertextVec(encVote))
		columnA = append(columnA, share.SharesA[0])
		columnB = append(columnB, share.SharesB[0])
	}

	aggA := NewAggregator("ServerA", pk)
	aggregatedA := aggA.AggregateShares(0, columnA)
	aggB := NewAggregator("ServerB", pk)
	aggregatedB := aggB.AggregateShares(0, columnB)

	combiner := NewCombiner(pk)
	result := combiner.CombineAggregates(aggregatedA, aggregatedB)

	tally, _ := sk.Decrypt(result.EncryptedTally)
	expected := big.NewInt(5)
	if tally.Cmp(expected) != 0 {
		t.Errorf("Tally mismatch: expected %v, got %v", expected, tally)
	}
	if result.TotalVotes != 4 {
		t.Errorf("Vote count mismatch: expected 4, got %d", result.TotalVotes)
	}
}

func TestSA2Privacy(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	vote := big.NewInt(100)
	encVote, _ := pk.Encrypt(vote)

	splitter := NewVoteSplitter(pk)
	share, _ := splitter.SplitVote("voter1", singletonCiphertextVec(encVote))

	shareADecrypted, _ := sk.Decrypt(share.SharesA[0])
	shareBDecrypted, _ := sk.Decrypt(share.SharesB[0])

	if shareADecrypted.Cmp(vote) == 0 {
		t.Error("ShareA should not reveal the original vote")
	}

	sum := new(big.Int).Add(shareADecrypted, shareBDecrypted)
	sum.Mod(sum, pk.N)
	if sum.Cmp(vote) != 0 {
		t.Errorf("Sum of decrypted shares should equal vote: expected %v, got %v", vote, sum)
	}
}
