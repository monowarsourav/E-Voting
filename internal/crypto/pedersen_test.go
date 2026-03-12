package crypto

import (
	"math/big"
	"testing"
)

func TestPedersenCommitVerify(t *testing.T) {
	pp, err := GeneratePedersenParams(512) // Small for testing
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	message := big.NewInt(42)

	commitment, err := pp.Commit(message)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Should verify correctly
	if !pp.Verify(commitment, message) {
		t.Error("Valid commitment verification failed")
	}

	// Should fail with wrong message
	wrongMessage := big.NewInt(43)
	if pp.Verify(commitment, wrongMessage) {
		t.Error("Invalid commitment verification should fail")
	}
}

func TestPedersenHomomorphic(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	m1 := big.NewInt(10)
	m2 := big.NewInt(20)
	expectedSum := big.NewInt(30)

	c1, _ := pp.Commit(m1)
	c2, _ := pp.Commit(m2)

	// Add commitments
	cSum := pp.AddCommitments(c1, c2)

	// Verify sum
	if !pp.Verify(cSum, expectedSum) {
		t.Error("Homomorphic addition failed")
	}
}

func TestPedersenScalarMultiply(t *testing.T) {
	pp, _ := GeneratePedersenParams(512)

	m := big.NewInt(5)
	k := big.NewInt(3)
	expected := big.NewInt(15) // 5 × 3

	c, _ := pp.Commit(m)

	// Scalar multiply
	cMul := pp.ScalarMultiply(c, k)

	// Verify
	if !pp.Verify(cMul, expected) {
		t.Error("Scalar multiplication failed")
	}
}
