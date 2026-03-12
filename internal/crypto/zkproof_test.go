package crypto

import (
	"math/big"
	"testing"
)

func TestBinaryProofZero(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	// Commit to 0
	w := big.NewInt(0)
	commitment, _ := pp.Commit(w)

	// Generate nonce for replay protection
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("Nonce generation failed: %v", err)
	}
	electionID := "test-election-001"

	// Create binary proof
	proof, err := pp.ProveBinary(w, commitment.R, commitment.C, nonce, electionID)
	if err != nil {
		t.Fatalf("Proof generation failed: %v", err)
	}

	// Verify
	if !pp.VerifyBinary(commitment.C, proof) {
		t.Error("Valid binary proof (w=0) verification failed")
	}
}

func TestBinaryProofOne(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	// Commit to 1
	w := big.NewInt(1)
	commitment, _ := pp.Commit(w)

	// Generate nonce for replay protection
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("Nonce generation failed: %v", err)
	}
	electionID := "test-election-001"

	// Create binary proof
	proof, err := pp.ProveBinary(w, commitment.R, commitment.C, nonce, electionID)
	if err != nil {
		t.Fatalf("Proof generation failed: %v", err)
	}

	// Verify
	if !pp.VerifyBinary(commitment.C, proof) {
		t.Error("Valid binary proof (w=1) verification failed")
	}
}

func TestBinaryProofInvalid(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	// Commit to 2 (invalid)
	w := big.NewInt(2)
	commitment, _ := pp.Commit(w)

	nonce, _ := GenerateNonce()
	electionID := "test-election-001"

	// Should fail to create proof
	_, err := pp.ProveBinary(w, commitment.R, commitment.C, nonce, electionID)
	if err == nil {
		t.Error("Should not be able to prove w=2")
	}
}

func TestBinaryProofMissingNonce(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	w := big.NewInt(0)
	commitment, _ := pp.Commit(w)

	// Empty nonce should be rejected
	_, err := pp.ProveBinary(w, commitment.R, commitment.C, nil, "test-election")
	if err == nil {
		t.Error("Should reject nil nonce")
	}

	// Wrong-size nonce should be rejected
	_, err = pp.ProveBinary(w, commitment.R, commitment.C, []byte("short"), "test-election")
	if err == nil {
		t.Error("Should reject short nonce")
	}
}

func TestBinaryProofMissingElectionID(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	w := big.NewInt(0)
	commitment, _ := pp.Commit(w)

	nonce, _ := GenerateNonce()

	// Empty electionID should be rejected
	_, err := pp.ProveBinary(w, commitment.R, commitment.C, nonce, "")
	if err == nil {
		t.Error("Should reject empty electionID")
	}
}

func TestSumProof(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	// Create 5 commitments: 1 weight=1, 4 weight=0
	commitments := make([]*Commitment, 5)
	commitmentValues := make([]*big.Int, 5)

	// One real
	commitments[0], _ = pp.Commit(big.NewInt(1))
	commitmentValues[0] = commitments[0].C

	// Four fakes
	for i := 1; i < 5; i++ {
		commitments[i], _ = pp.Commit(big.NewInt(0))
		commitmentValues[i] = commitments[i].C
	}

	// Generate nonce for replay protection
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("Nonce generation failed: %v", err)
	}
	electionID := "test-election-001"

	// Create sum proof
	sumProof, err := pp.ProveSumOne(commitments, nonce, electionID)
	if err != nil {
		t.Fatalf("Sum proof generation failed: %v", err)
	}

	// Verify
	if !pp.VerifySumOne(commitmentValues, sumProof) {
		t.Error("Valid sum proof verification failed")
	}
}

func TestSumProofInvalid(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	// Create commitments that don't sum to 1
	commitments := make([]*Commitment, 3)
	commitmentValues := make([]*big.Int, 3)

	// All zeros - sum = 0, not 1
	for i := 0; i < 3; i++ {
		commitments[i], _ = pp.Commit(big.NewInt(0))
		commitmentValues[i] = commitments[i].C
	}

	nonce, _ := GenerateNonce()
	electionID := "test-election-001"

	// This proof should still be created but verification with wrong values should fail
	sumProof, _ := pp.ProveSumOne(commitments, nonce, electionID)

	// Verification should fail because sum != 1
	if pp.VerifySumOne(commitmentValues, sumProof) {
		t.Error("Invalid sum proof should not verify")
	}
}
