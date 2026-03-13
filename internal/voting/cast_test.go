package voting

import (
	"math/big"
	"testing"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
)

func TestBallotCreation(t *testing.T) {
	// Setup
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	bc := NewBallotCreator(pk)

	// Create ballot
	ballot, err := bc.CreateBallot("voter001", 1)
	if err != nil {
		t.Fatalf("Ballot creation failed: %v", err)
	}

	if ballot.VoterID != "voter001" {
		t.Errorf("VoterID mismatch")
	}

	if ballot.CandidateID != 1 {
		t.Errorf("CandidateID mismatch")
	}

	// Decrypt to verify
	decrypted, _ := sk.Decrypt(ballot.EncryptedVote)
	if decrypted.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Decrypted vote mismatch: expected 1, got %v", decrypted)
	}
}

func TestWeightApplication(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	bc := NewBallotCreator(pk)

	// Create ballot for candidate 5
	ballot, _ := bc.CreateBallot("voter001", 5)

	// Apply weight 1 (real slot)
	weightedVote := bc.ApplyWeight(ballot.EncryptedVote, big.NewInt(1))

	// Decrypt
	decrypted, _ := sk.Decrypt(weightedVote)
	expected := big.NewInt(5) // 5 × 1 = 5

	if decrypted.Cmp(expected) != 0 {
		t.Errorf("Weighted vote mismatch: expected %v, got %v", expected, decrypted)
	}

	// Apply weight 0 (fake slot)
	weightedVoteZero := bc.ApplyWeight(ballot.EncryptedVote, big.NewInt(0))
	decryptedZero, _ := sk.Decrypt(weightedVoteZero)

	if decryptedZero.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Zero-weighted vote should be 0, got %v", decryptedZero)
	}
}

func TestVoteCasting(t *testing.T) {
	// Setup cryptographic parameters
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	pp, _ := crypto.GeneratePedersenParams(512)
	rp, _ := crypto.GenerateRingParams(512)

	// Create election
	election := &Election{
		ElectionID: "election001",
		Title:      "Test Election",
		Candidates: []*Candidate{
			{ID: 1, Name: "Candidate A"},
			{ID: 2, Name: "Candidate B"},
		},
		StartTime: time.Now().Add(-1 * time.Hour).Unix(),
		EndTime:   time.Now().Add(1 * time.Hour).Unix(),
		IsActive:  true,
	}

	// Create registration system with eligible voters
	eligibleVoters := []string{"voter001", "voter002", "voter003"}
	rs := voter.NewRegistrationSystem(pp, rp, 5, eligibleVoters, "election001")

	// Register some voters
	fingerprint1 := make([]byte, 200)
	for i := range fingerprint1 {
		fingerprint1[i] = byte(i % 256)
	}
	// Note: In real system, voter001 must be in eligible list
	// For testing, we'll skip actual registration as it requires complex setup

	// Create vote caster
	vc := NewVoteCaster(pk, rp, rs, election)

	// Test vote count
	if vc.GetVoteCount() != 0 {
		t.Errorf("Initial vote count should be 0, got %d", vc.GetVoteCount())
	}

	// Test invalid candidate
	// (We can't fully test voting without registered voters, but we can test validation)
	if vc.isValidCandidate(1) != true {
		t.Error("Candidate 1 should be valid")
	}

	if vc.isValidCandidate(99) != false {
		t.Error("Candidate 99 should be invalid")
	}
}

func TestElectionTiming(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	pp, _ := crypto.GeneratePedersenParams(512)
	rp, _ := crypto.GenerateRingParams(512)

	// Election that hasn't started
	futureElection := &Election{
		ElectionID: "future",
		StartTime:  time.Now().Add(1 * time.Hour).Unix(),
		EndTime:    time.Now().Add(2 * time.Hour).Unix(),
		IsActive:   true,
	}

	rs := voter.NewRegistrationSystem(pp, rp, 5, []string{}, "future")
	vc := NewVoteCaster(pk, rp, rs, futureElection)

	// Try to vote (should fail - not in voting period)
	_, err := vc.CastVote("voter001", 1, 0)
	if err == nil {
		t.Error("Should not allow voting before election starts")
	}

	// Past election
	pastElection := &Election{
		ElectionID: "past",
		StartTime:  time.Now().Add(-2 * time.Hour).Unix(),
		EndTime:    time.Now().Add(-1 * time.Hour).Unix(),
		IsActive:   true,
	}

	vc2 := NewVoteCaster(pk, rp, rs, pastElection)
	_, err = vc2.CastVote("voter001", 1, 0)
	if err == nil {
		t.Error("Should not allow voting after election ends")
	}
}
