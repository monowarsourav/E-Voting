package crypto

import (
	"math/big"
	"testing"
)

// TestPaillierHomomorphicProperty verifies E(a) × E(b) = E(a+b)
// This is the foundation of Theorem 1 (Ballot Privacy) and Theorem 5 (Universal Verifiability)
func TestPaillierHomomorphicProperty(t *testing.T) {
	key, err := GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := key.PublicKey

	testCases := [][2]int64{{0, 0}, {1, 0}, {0, 1}, {1, 1}, {42, 58}, {100, 200}, {0, 999}}

	for _, tc := range testCases {
		a, b := big.NewInt(tc[0]), big.NewInt(tc[1])

		encA, _ := pk.Encrypt(a)
		encB, _ := pk.Encrypt(b)

		// Homomorphic addition
		encSum := pk.Add(encA, encB)

		// Decrypt and verify
		sum, _ := key.Decrypt(encSum)
		expected := new(big.Int).Add(a, b)

		if sum.Cmp(expected) != 0 {
			t.Errorf("Homomorphic property failed: E(%d) + E(%d) = %d, expected %d",
				tc[0], tc[1], sum.Int64(), expected.Int64())
		}
	}
}

// TestPaillierScalarMultiplication verifies E(m)^k = E(k×m)
func TestPaillierScalarMultiplication(t *testing.T) {
	key, err := GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := key.PublicKey

	m := big.NewInt(7)
	k := big.NewInt(5)

	encM, _ := pk.Encrypt(m)
	encKM := pk.Multiply(encM, k)

	result, _ := key.Decrypt(encKM)
	expected := new(big.Int).Mul(m, k) // 35

	if result.Cmp(expected) != 0 {
		t.Errorf("Scalar mult failed: E(%d)^%d = %d, expected %d",
			m.Int64(), k.Int64(), result.Int64(), expected.Int64())
	}
}

// TestPedersenBindingProperty verifies commitment binding (cannot open to different value)
func TestPedersenBindingProperty(t *testing.T) {
	pp, err := GeneratePedersenParams(512)
	if err != nil {
		t.Fatal(err)
	}

	// Commit to value 1
	commitment, _ := pp.Commit(big.NewInt(1))

	// Verify opens correctly for value 1
	if !pp.Verify(commitment, big.NewInt(1)) {
		t.Fatal("Commitment should verify for correct value")
	}

	// Create a fake commitment with same C but claimed value 0
	fakeCommitment := &Commitment{C: commitment.C, R: commitment.R}
	if pp.Verify(fakeCommitment, big.NewInt(0)) {
		t.Fatal("Commitment should NOT verify for wrong value (binding broken!)")
	}
}

// TestRingSignatureLinkabilityProperty verifies same signer → same key image
// This is the foundation of double-vote detection (Theorem 3)
func TestRingSignatureLinkabilityProperty(t *testing.T) {
	rp, err := GenerateRingParams(512)
	if err != nil {
		t.Fatal(err)
	}

	// Create ring of 10 members
	keys := make([]*RingKeyPair, 10)
	pubKeys := make([]*big.Int, 10)
	for i := 0; i < 10; i++ {
		kp, _ := rp.GenerateRingKeyPair()
		keys[i] = kp
		pubKeys[i] = kp.PublicKey
	}

	// Same signer signs two different messages
	sig1, _ := rp.Sign([]byte("message1"), keys[3], pubKeys, 3)
	sig2, _ := rp.Sign([]byte("message2"), keys[3], pubKeys, 3)

	// Key images must be identical (linkable)
	if sig1.KeyImage.Cmp(sig2.KeyImage) != 0 {
		t.Fatal("Same signer should produce identical key images (linkability broken!)")
	}

	// Different signer should produce different key image
	sig3, _ := rp.Sign([]byte("message1"), keys[5], pubKeys, 5)
	if sig1.KeyImage.Cmp(sig3.KeyImage) == 0 {
		t.Fatal("Different signers should produce different key images")
	}
}

// TestPaillierZeroWeightCancellation verifies E(vote)^0 = E(0)
// This is critical for SMDC: fake slots with weight 0 must contribute nothing to tally
func TestPaillierZeroWeightCancellation(t *testing.T) {
	key, err := GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := key.PublicKey

	// Encrypt a vote
	vote := big.NewInt(42)
	encVote, _ := pk.Encrypt(vote)

	// Apply weight 0 (fake SMDC slot)
	zeroWeighted := pk.Multiply(encVote, big.NewInt(0))

	// Decrypt should be 0
	result, _ := key.Decrypt(zeroWeighted)
	if result.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("E(vote)^0 should decrypt to 0, got %v", result)
	}
}

// TestPaillierAdditiveIdentity verifies E(0) is the additive identity
func TestPaillierAdditiveIdentity(t *testing.T) {
	key, err := GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := key.PublicKey

	vote := big.NewInt(7)
	encVote, _ := pk.Encrypt(vote)
	encZero, _ := pk.Encrypt(big.NewInt(0))

	// E(vote) + E(0) should equal E(vote)
	sum := pk.Add(encVote, encZero)
	result, _ := key.Decrypt(sum)

	if result.Cmp(vote) != 0 {
		t.Errorf("E(vote) + E(0) should equal vote (%d), got %d", vote.Int64(), result.Int64())
	}
}
