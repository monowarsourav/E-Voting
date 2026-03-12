package tally

import (
	"math/big"
	"testing"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
)

func TestDecryption(t *testing.T) {
	// Setup
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	decryptor := NewDecryptor(sk)

	// Encrypt a value
	value := big.NewInt(42)
	encrypted, _ := pk.Encrypt(value)

	// Decrypt
	decrypted, err := decryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted.Cmp(value) != 0 {
		t.Errorf("Decryption mismatch: expected %v, got %v", value, decrypted)
	}
}

func TestDecryptMultiple(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	decryptor := NewDecryptor(sk)

	// Encrypt multiple values
	values := []*big.Int{
		big.NewInt(10),
		big.NewInt(20),
		big.NewInt(30),
	}

	encrypted := make([]*big.Int, len(values))
	for i, v := range values {
		encrypted[i], _ = pk.Encrypt(v)
	}

	// Decrypt all
	decrypted, err := decryptor.DecryptMultiple(encrypted)
	if err != nil {
		t.Fatalf("Multiple decryption failed: %v", err)
	}

	for i, expected := range values {
		if decrypted[i].Cmp(expected) != 0 {
			t.Errorf("Decryption %d mismatch: expected %v, got %v", i, expected, decrypted[i])
		}
	}
}

func TestVoteTallying(t *testing.T) {
	// Setup
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	counter := NewCounter(pk, sk)

	// Create some votes
	votes := []*big.Int{
		big.NewInt(1), // Vote for candidate 1
		big.NewInt(1), // Vote for candidate 1
		big.NewInt(2), // Vote for candidate 2
	}

	// Encrypt votes
	encryptedVotes := make([]*big.Int, len(votes))
	for i, v := range votes {
		encryptedVotes[i], _ = pk.Encrypt(v)
	}

	// Create vote shares using SA²
	splitter := sa2.NewVoteSplitter(pk)
	voteShares := make([]*sa2.VoteShare, len(encryptedVotes))

	for i, enc := range encryptedVotes {
		share, _ := splitter.SplitVote("voter"+string(rune(i)), enc)
		voteShares[i] = share
	}

	// Tally
	result, err := counter.TallyVotes(voteShares, "election001")
	if err != nil {
		t.Fatalf("Tallying failed: %v", err)
	}

	if result.TotalVotes != 3 {
		t.Errorf("Total votes mismatch: expected 3, got %d", result.TotalVotes)
	}

	// Check decrypted total (1+1+2 = 4)
	total := result.CandidateTallies[0]
	expected := big.NewInt(4)

	if total.Cmp(expected) != 0 {
		t.Errorf("Tally mismatch: expected %v, got %v", expected, total)
	}
}

func TestTallyByCandidate(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	counter := NewCounter(pk, sk)

	// Votes by candidate
	votesPerCandidate := make(map[int][]*big.Int)

	// Candidate 1: 3 votes
	votesPerCandidate[1] = []*big.Int{
		pk.Multiply(pk.AddMultiple([]*big.Int{pk.AddPlaintext(big.NewInt(1), big.NewInt(0))}), big.NewInt(1)),
		pk.Multiply(pk.AddMultiple([]*big.Int{pk.AddPlaintext(big.NewInt(1), big.NewInt(0))}), big.NewInt(1)),
		pk.Multiply(pk.AddMultiple([]*big.Int{pk.AddPlaintext(big.NewInt(1), big.NewInt(0))}), big.NewInt(1)),
	}

	// Encrypt properly
	enc1, _ := pk.Encrypt(big.NewInt(1))
	enc2, _ := pk.Encrypt(big.NewInt(1))
	enc3, _ := pk.Encrypt(big.NewInt(1))
	votesPerCandidate[1] = []*big.Int{enc1, enc2, enc3}

	// Candidate 2: 2 votes
	enc4, _ := pk.Encrypt(big.NewInt(1))
	enc5, _ := pk.Encrypt(big.NewInt(1))
	votesPerCandidate[2] = []*big.Int{enc4, enc5}

	// Tally
	result, err := counter.TallyByCandidate(votesPerCandidate, "election001")
	if err != nil {
		t.Fatalf("Tallying by candidate failed: %v", err)
	}

	// Candidate 1 should have 3 votes
	if result.CandidateTallies[1].Cmp(big.NewInt(3)) != 0 {
		t.Errorf("Candidate 1 tally mismatch: expected 3, got %v", result.CandidateTallies[1])
	}

	// Candidate 2 should have 2 votes
	if result.CandidateTallies[2].Cmp(big.NewInt(2)) != 0 {
		t.Errorf("Candidate 2 tally mismatch: expected 2, got %v", result.CandidateTallies[2])
	}

	if result.TotalVotes != 5 {
		t.Errorf("Total votes mismatch: expected 5, got %d", result.TotalVotes)
	}
}

func TestEmptyTally(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(1024)
	pk := sk.PublicKey

	counter := NewCounter(pk, sk)

	// Tally with no votes
	result, err := counter.TallyVotes([]*sa2.VoteShare{}, "election001")
	if err != nil {
		t.Fatalf("Empty tally failed: %v", err)
	}

	if result.TotalVotes != 0 {
		t.Errorf("Empty tally should have 0 votes, got %d", result.TotalVotes)
	}
}
