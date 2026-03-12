package tests

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/covertvote/e-voting/internal/pq"
)

// TestSaveCirclTestVectors saves test vectors BEFORE circl upgrade.
// Run this ONCE with circl v1.6.2, then upgrade to v1.6.3 and run TestCirclUpgradeCompatibility.
func TestSaveCirclTestVectors(t *testing.T) {
	// Generate Kyber keypair with current circl version
	keyPair, err := pq.GenerateKyberKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Kyber keypair: %v", err)
	}

	// Serialize keys
	pubKeyBytes, err := keyPair.PublicKey.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	privKeyBytes, err := keyPair.PrivateKey.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	// Encrypt a known message
	encap, err := pq.EncapsulateWithPublicKey(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encapsulate: %v", err)
	}

	// Save test vectors
	os.MkdirAll("testdata", 0755)
	os.WriteFile("testdata/kyber_v162_pubkey.hex", []byte(hex.EncodeToString(pubKeyBytes)), 0644)
	os.WriteFile("testdata/kyber_v162_privkey.hex", []byte(hex.EncodeToString(privKeyBytes)), 0644)
	os.WriteFile("testdata/kyber_v162_ciphertext.hex", []byte(hex.EncodeToString(encap.Ciphertext)), 0644)
	os.WriteFile("testdata/kyber_v162_sharedkey.hex", []byte(hex.EncodeToString(encap.SharedKey)), 0644)

	t.Logf("Test vectors saved to testdata/")
	t.Logf("Public key: %d bytes", len(pubKeyBytes))
	t.Logf("Private key: %d bytes", len(privKeyBytes))
	t.Logf("Ciphertext: %d bytes", len(encap.Ciphertext))
	t.Logf("Shared key: %d bytes", len(encap.SharedKey))
}

// TestCirclUpgradeCompatibility verifies v1.6.2 ciphertexts work with v1.6.3.
// Run AFTER upgrading circl to v1.6.3.
func TestCirclUpgradeCompatibility(t *testing.T) {
	// Load saved test vectors
	pubKeyHex, err := os.ReadFile("testdata/kyber_v162_pubkey.hex")
	if err != nil {
		t.Skip("No test vectors found — run TestSaveCirclTestVectors first with circl v1.6.2")
	}
	privKeyHex, _ := os.ReadFile("testdata/kyber_v162_privkey.hex")
	ciphertextHex, _ := os.ReadFile("testdata/kyber_v162_ciphertext.hex")
	expectedKeyHex, _ := os.ReadFile("testdata/kyber_v162_sharedkey.hex")

	// Decode
	pubKeyBytes, _ := hex.DecodeString(string(pubKeyHex))
	privKeyBytes, _ := hex.DecodeString(string(privKeyHex))
	ciphertext, _ := hex.DecodeString(string(ciphertextHex))
	expectedKey, _ := hex.DecodeString(string(expectedKeyHex))

	// Reconstruct keys with current (potentially upgraded) circl
	keyPair, err := pq.UnmarshalKyberKeyPair(pubKeyBytes, privKeyBytes)
	if err != nil {
		t.Fatalf("Failed to unmarshal v1.6.2 keys with current circl: %v", err)
	}

	// Decapsulate with new version
	sharedKey, err := pq.DecapsulateWithPrivateKey(keyPair.PrivateKey, ciphertext)
	if err != nil {
		t.Fatalf("CRITICAL: circl upgrade BREAKS v1.6.2 ciphertexts: %v", err)
	}

	// Compare shared keys
	if hex.EncodeToString(sharedKey) != hex.EncodeToString(expectedKey) {
		t.Fatalf("CRITICAL: Shared key mismatch after circl upgrade!\nExpected: %s\nGot:      %s",
			hex.EncodeToString(expectedKey), hex.EncodeToString(sharedKey))
	}

	t.Log("circl upgrade compatibility: PASSED")
	t.Log("v1.6.2 ciphertexts decrypt correctly with current circl version")

	_ = pubKeyBytes
}
