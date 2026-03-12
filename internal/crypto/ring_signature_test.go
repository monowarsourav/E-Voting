package crypto

import (
	"math/big"
	"testing"
)

func TestRingSignatureBasic(t *testing.T) {
	// Setup
	rp, _ := GenerateRingParams(512)

	// Create ring of 5 members
	ring := make([]*big.Int, 5)
	keys := make([]*RingKeyPair, 5)

	for i := 0; i < 5; i++ {
		keys[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keys[i].PublicKey
	}

	// Sign with member 2
	message := []byte("Vote for Candidate A")
	signerIndex := 2

	sig, err := rp.Sign(message, keys[signerIndex], ring, signerIndex)
	if err != nil {
		t.Fatalf("Signing failed: %v", err)
	}

	// Verify
	if !rp.Verify(message, sig, ring) {
		t.Error("Valid signature verification failed")
	}
}

func TestRingSignatureLinkability(t *testing.T) {
	rp, _ := GenerateRingParams(512)

	// Create ring
	ring := make([]*big.Int, 3)
	keys := make([]*RingKeyPair, 3)

	for i := 0; i < 3; i++ {
		keys[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keys[i].PublicKey
	}

	// Sign two messages with same key
	msg1 := []byte("Vote 1")
	msg2 := []byte("Vote 2")

	sig1, _ := rp.Sign(msg1, keys[0], ring, 0)
	sig2, _ := rp.Sign(msg2, keys[0], ring, 0)

	// Should link (same signer)
	if !Link(sig1, sig2) {
		t.Error("Signatures from same signer should link")
	}

	// Sign with different key
	sig3, _ := rp.Sign(msg1, keys[1], ring, 1)

	// Should not link (different signer)
	if Link(sig1, sig3) {
		t.Error("Signatures from different signers should not link")
	}
}

func TestRingSignatureInvalidMessage(t *testing.T) {
	rp, _ := GenerateRingParams(512)

	ring := make([]*big.Int, 3)
	keys := make([]*RingKeyPair, 3)

	for i := 0; i < 3; i++ {
		keys[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keys[i].PublicKey
	}

	message := []byte("Original message")
	sig, _ := rp.Sign(message, keys[0], ring, 0)

	// Try to verify with different message
	tamperedMessage := []byte("Tampered message")
	if rp.Verify(tamperedMessage, sig, ring) {
		t.Error("Signature should not verify with tampered message")
	}
}
