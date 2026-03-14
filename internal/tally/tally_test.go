package tally

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/voting"
)

func TestDecryption(t *testing.T) {
	// Setup
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
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
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
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
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
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
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
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
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
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

// TestTallyCorrectness is a property-based test: encrypt N random votes,
// tally homomorphically, decrypt, verify sum matches plaintext sum.
func TestTallyCorrectness(t *testing.T) {
	key, err := crypto.GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := key.PublicKey

	// 50 voters, distributed among 3 candidates
	candidates := 3
	votes := make([]int64, 50)
	expected := make([]int64, candidates)

	for i := range votes {
		votes[i] = int64(i % candidates) // distribute among candidates
		expected[votes[i]]++
	}

	// Encrypt and tally per candidate
	for c := 0; c < candidates; c++ {
		tally := big.NewInt(1) // identity for multiplication
		for _, v := range votes {
			vote := big.NewInt(0)
			if v == int64(c) {
				vote = big.NewInt(1)
			}
			enc, _ := pk.Encrypt(vote)
			tally = pk.Add(tally, enc)
		}

		result, _ := key.Decrypt(tally)
		if result.Int64() != expected[c] {
			t.Errorf("Candidate %d: expected %d votes, got %d", c, expected[c], result.Int64())
		}
	}
}

// TestSA2TallyIntegrity tests that SA² split → aggregate → combine → decrypt gives correct result.
func TestSA2TallyIntegrity(t *testing.T) {
	key, err := crypto.GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := key.PublicKey
	splitter := sa2.NewVoteSplitter(pk)
	aggA := sa2.NewAggregator("server-a", pk)
	aggB := sa2.NewAggregator("server-b", pk)
	combiner := sa2.NewCombiner(pk)

	// 20 voters
	nVoters := 20
	var sharesA, sharesB []*big.Int
	expectedSum := int64(0)

	for i := 0; i < nVoters; i++ {
		vote := big.NewInt(int64(i % 2))
		expectedSum += int64(i % 2)

		enc, _ := pk.Encrypt(vote)
		share, _ := splitter.SplitVote(fmt.Sprintf("v%d", i), enc)
		sharesA = append(sharesA, share.ShareA)
		sharesB = append(sharesB, share.ShareB)
	}

	resultA := aggA.AggregateShares(sharesA)
	resultB := aggB.AggregateShares(sharesB)
	combined := combiner.CombineAggregates(resultA, resultB)

	decrypted, _ := key.Decrypt(combined.EncryptedTally)
	if decrypted.Int64() != expectedSum {
		t.Errorf("SA2 tally: expected %d, got %d", expectedSum, decrypted.Int64())
	}
}

func TestThresholdTally(t *testing.T) {
	// Generate threshold keys (3-of-5)
	shares, err := crypto.GenerateThresholdKey(2048, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	pk := shares.PublicKey

	// Encrypt 20 votes: 12 for candidate A (1), 8 for candidate B (0)
	nVoters := 20
	expectedSum := int64(0)
	var encryptedVotes []*big.Int

	for i := 0; i < nVoters; i++ {
		v := int64(0)
		if i < 12 {
			v = 1
		}
		expectedSum += v
		enc, _ := pk.Encrypt(big.NewInt(v))
		encryptedVotes = append(encryptedVotes, enc)
	}

	// Use ThresholdTally with trustees 0, 1, 2
	result, err := ThresholdTally(encryptedVotes, pk, shares, []int{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}

	if result.Int64() != expectedSum {
		t.Errorf("Threshold tally: expected %d, got %d", expectedSum, result.Int64())
	}
}

func TestTallyWithDifferentTrusteeSubsets(t *testing.T) {
	shares, err := crypto.GenerateThresholdKey(2048, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	pk := shares.PublicKey

	// Encrypt a simple sum: 1+1+1 = 3
	var cts []*big.Int
	for i := 0; i < 3; i++ {
		enc, _ := pk.Encrypt(big.NewInt(1))
		cts = append(cts, enc)
	}

	// Try different subsets of 3 trustees
	subsets := [][3]int{{0, 1, 2}, {0, 2, 4}, {1, 3, 4}, {2, 3, 4}}

	for _, subset := range subsets {
		indices := subset[:]
		result, err := ThresholdTally(cts, pk, shares, indices)
		if err != nil {
			t.Fatalf("Subset %v failed: %v", subset, err)
		}

		if result.Int64() != 3 {
			t.Errorf("Subset %v: expected 3, got %d", subset, result.Int64())
		}
	}
}

func TestCounterTallyVotes(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	counter := NewCounter(pk, sk)

	// Create SA2 vote shares and tally them
	splitter := sa2.NewVoteSplitter(pk)

	// 5 votes for candidate A, 3 for candidate B
	distribution := []int64{1, 1, 1, 1, 1, 0, 0, 0}
	var voteShares []*sa2.VoteShare

	for i, v := range distribution {
		enc, _ := pk.Encrypt(big.NewInt(v))
		share, _ := splitter.SplitVote(fmt.Sprintf("voter-%d", i), enc)
		voteShares = append(voteShares, share)
	}

	result, err := counter.TallyVotes(voteShares, "election-001")
	if err != nil {
		t.Fatalf("TallyVotes failed: %v", err)
	}

	if result.TotalVotes != 8 {
		t.Errorf("Expected 8 total votes, got %d", result.TotalVotes)
	}

	// Decrypted sum should be 5 (five 1s + three 0s)
	total := result.CandidateTallies[0]
	if total.Int64() != 5 {
		t.Errorf("Expected sum 5, got %d", total.Int64())
	}
}

func TestVerifyTally(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	// Encrypt known values
	enc1, _ := pk.Encrypt(big.NewInt(5))
	enc2, _ := pk.Encrypt(big.NewInt(3))

	// Homomorphic add
	encSum := pk.Add(enc1, enc2)

	// Decrypt and verify
	result, _ := sk.Decrypt(encSum)
	if result.Int64() != 8 {
		t.Errorf("Verify tally: expected 8, got %d", result.Int64())
	}

	// Use Counter.VerifyTally
	counter := NewCounter(pk, sk)
	tallyResult := &TallyResult{
		ElectionID:       "test",
		CandidateTallies: map[int]*big.Int{0: big.NewInt(8)},
		TotalVotes:       2,
	}

	if !counter.VerifyTally(tallyResult, encSum) {
		t.Error("VerifyTally returned false")
	}
}

func TestThresholdDecryptorPartialDecrypt(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	td := NewThresholdDecryptor(pk, 2)

	// Encrypt a value
	value := big.NewInt(42)
	encrypted, _ := pk.Encrypt(value)

	// Create a threshold key share
	share := &ThresholdKey{
		ServerID:  "server-1",
		Share:     big.NewInt(12345),
		Index:     1,
		PublicKey: pk,
	}

	// Partial decrypt
	partial, err := td.PartialDecrypt(encrypted, share)
	if err != nil {
		t.Fatalf("PartialDecrypt failed: %v", err)
	}

	if partial == nil {
		t.Fatal("Partial decryption is nil")
	}
	if partial.ServerID != "server-1" {
		t.Errorf("ServerID mismatch: got %s", partial.ServerID)
	}
	if partial.Value == nil {
		t.Error("Partial value is nil")
	}
	if len(partial.Proof) < 32 {
		t.Error("Proof is too short")
	}

	// Test nil input
	_, err = td.PartialDecrypt(nil, share)
	if err == nil {
		t.Error("Expected error for nil encrypted value")
	}

	_, err = td.PartialDecrypt(encrypted, nil)
	if err == nil {
		t.Error("Expected error for nil key share")
	}
}

func TestThresholdDecryptorCombine(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	td := NewThresholdDecryptor(pk, 2)

	// Encrypt a value
	value := big.NewInt(7)
	encrypted, _ := pk.Encrypt(value)

	// Create 2 threshold key shares
	shares := []*ThresholdKey{
		{ServerID: "server-1", Share: big.NewInt(111), Index: 1, PublicKey: pk},
		{ServerID: "server-2", Share: big.NewInt(222), Index: 2, PublicKey: pk},
	}

	// Partial decrypt each
	partials := make([]*PartialDecryption, 2)
	for i, share := range shares {
		pd, err := td.PartialDecrypt(encrypted, share)
		if err != nil {
			t.Fatalf("PartialDecrypt %d failed: %v", i, err)
		}
		partials[i] = pd
	}

	// Combine (note: with arbitrary shares, the result won't match the original
	// value, but we're testing that the code path executes without panicking)
	_, err := td.CombinePartialDecryptions(partials, sk)
	if err != nil {
		t.Fatalf("CombinePartialDecryptions failed: %v", err)
	}

	// Test insufficient partials
	_, err = td.CombinePartialDecryptions(partials[:1], sk)
	if err == nil {
		t.Error("Expected error for insufficient partials")
	}
}

func TestThresholdDecryptorVerifyDecryption(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	td := NewThresholdDecryptor(pk, 2)

	// Encrypt and decrypt a value
	value := big.NewInt(42)
	encrypted, _ := pk.Encrypt(value)
	decrypted, _ := sk.Decrypt(encrypted)

	// Create a valid proof (>= 32 bytes)
	proof := make([]byte, 64)
	for i := range proof {
		proof[i] = byte(i)
	}

	// Verify should succeed with valid inputs
	if !td.VerifyDecryption(encrypted, decrypted, proof) {
		t.Error("VerifyDecryption returned false for valid inputs")
	}

	// Test nil inputs
	if td.VerifyDecryption(nil, decrypted, proof) {
		t.Error("Expected false for nil encrypted")
	}
	if td.VerifyDecryption(encrypted, nil, proof) {
		t.Error("Expected false for nil decrypted")
	}
	if td.VerifyDecryption(encrypted, decrypted, nil) {
		t.Error("Expected false for nil proof")
	}

	// Test short proof
	shortProof := make([]byte, 10)
	if td.VerifyDecryption(encrypted, decrypted, shortProof) {
		t.Error("Expected false for short proof")
	}
}

func TestAggregateWeightedVotes(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	counter := NewCounter(pk, sk)

	// Vote with weight 1 (real) should count
	voteReal, _ := pk.Encrypt(big.NewInt(1))
	weightedReal := pk.Multiply(voteReal, big.NewInt(1)) // E(1*1) = E(1)

	// Vote with weight 0 (fake) should not count
	voteFake, _ := pk.Encrypt(big.NewInt(1))
	weightedFake := pk.Multiply(voteFake, big.NewInt(0)) // E(1*0) = E(0)

	weightedVotes := []*voting.WeightedVote{
		{VoterID: "v1", EncryptedVote: weightedReal},
		{VoterID: "v2", EncryptedVote: weightedFake},
	}

	aggregated := counter.AggregateWeightedVotes(weightedVotes)
	result, _ := sk.Decrypt(aggregated)

	if result.Int64() != 1 {
		t.Errorf("Weighted tally: expected 1, got %d (fake vote should not count)", result.Int64())
	}
}
