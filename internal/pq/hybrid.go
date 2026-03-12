package pq

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"github.com/covertvote/e-voting/internal/crypto"
)

// HybridKeyPair combines Kyber and Paillier keys
type HybridKeyPair struct {
	KyberPublicKey    kem.PublicKey
	KyberPrivateKey   kem.PrivateKey
	PaillierPublicKey *crypto.PaillierPublicKey
	PaillierSecretKey *crypto.PaillierPrivateKey
}

// HybridCiphertext represents a hybrid encrypted message
type HybridCiphertext struct {
	KyberCiphertext    []byte   `json:"kyber_ciphertext"`
	PaillierCiphertext *big.Int `json:"paillier_ciphertext"`
	Salt               []byte   `json:"salt"`
	MAC                []byte   `json:"mac"`
}

// GenerateHybridKeyPair generates both Kyber and Paillier key pairs
func GenerateHybridKeyPair(paillierBits int) (*HybridKeyPair, error) {
	// Generate Kyber key pair
	kyberPair, err := GenerateKyberKeyPair()
	if err != nil {
		return nil, err
	}

	// Generate Paillier key pair
	paillierSK, err := crypto.GeneratePaillierKeyPair(paillierBits)
	if err != nil {
		return nil, err
	}

	return &HybridKeyPair{
		KyberPublicKey:    kyberPair.PublicKey,
		KyberPrivateKey:   kyberPair.PrivateKey,
		PaillierPublicKey: paillierSK.PublicKey,
		PaillierSecretKey: paillierSK,
	}, nil
}

// HybridEncrypt performs hybrid encryption using both Kyber and Paillier
// This provides both post-quantum security (Kyber) and homomorphic properties (Paillier)
func HybridEncrypt(
	message *big.Int,
	kyberPublicKey kem.PublicKey,
	paillierPublicKey *crypto.PaillierPublicKey,
) (*HybridCiphertext, error) {
	// Step 1: Encrypt with Paillier (homomorphic encryption)
	paillierCiphertext, err := paillierPublicKey.Encrypt(message)
	if err != nil {
		return nil, err
	}

	// Step 2: Encapsulate with Kyber to get shared secret
	encap, err := EncapsulateWithPublicKey(kyberPublicKey)
	if err != nil {
		return nil, err
	}

	// Step 3: Generate salt
	salt, err := GenerateRandomSalt()
	if err != nil {
		return nil, err
	}

	// Step 4: Derive key from Kyber shared secret
	derivedKey := DeriveKey(encap.SharedKey, salt)

	// Step 5: Compute MAC over Paillier ciphertext
	mac := computeMAC(paillierCiphertext.Bytes(), derivedKey)

	return &HybridCiphertext{
		KyberCiphertext:    encap.Ciphertext,
		PaillierCiphertext: paillierCiphertext,
		Salt:               salt,
		MAC:                mac,
	}, nil
}

// HybridDecrypt performs hybrid decryption
func HybridDecrypt(
	ciphertext *HybridCiphertext,
	kyberPrivateKey kem.PrivateKey,
	paillierSecretKey *crypto.PaillierPrivateKey,
) (*big.Int, error) {
	// Step 1: Decapsulate Kyber to get shared secret
	scheme := kyber768.Scheme()
	sharedSecret, err := scheme.Decapsulate(kyberPrivateKey, ciphertext.KyberCiphertext)
	if err != nil {
		return nil, err
	}

	// Step 2: Derive key
	derivedKey := DeriveKey(sharedSecret, ciphertext.Salt)

	// Step 3: Verify MAC
	expectedMAC := computeMAC(ciphertext.PaillierCiphertext.Bytes(), derivedKey)
	if !verifyMAC(ciphertext.MAC, expectedMAC) {
		return nil, errors.New("MAC verification failed - possible tampering")
	}

	// Step 4: Decrypt with Paillier
	plaintext, err := paillierSecretKey.Decrypt(ciphertext.PaillierCiphertext)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// HybridHomomorphicAdd performs homomorphic addition on hybrid ciphertexts
func HybridHomomorphicAdd(
	ct1, ct2 *HybridCiphertext,
	paillierPublicKey *crypto.PaillierPublicKey,
) *HybridCiphertext {
	// Add Paillier ciphertexts (homomorphic property)
	sumCiphertext := paillierPublicKey.Add(ct1.PaillierCiphertext, ct2.PaillierCiphertext)

	// NOTE: The returned MAC is stale because the PaillierCiphertext has changed.
	// The caller must re-encapsulate (via ReEncapsulate) after the final tally
	// to produce a valid MAC. Re-encapsulating after every intermediate operation
	// is unnecessary and expensive; it should be done once after all homomorphic
	// operations are complete.
	return &HybridCiphertext{
		KyberCiphertext:    ct1.KyberCiphertext,
		PaillierCiphertext: sumCiphertext,
		Salt:               ct1.Salt,
		MAC:                ct1.MAC,
	}
}

// HybridHomomorphicMultiply performs scalar multiplication on hybrid ciphertext
func HybridHomomorphicMultiply(
	ct *HybridCiphertext,
	scalar *big.Int,
	paillierPublicKey *crypto.PaillierPublicKey,
) *HybridCiphertext {
	// Multiply Paillier ciphertext by scalar
	productCiphertext := paillierPublicKey.Multiply(ct.PaillierCiphertext, scalar)

	// NOTE: The returned MAC is stale because the PaillierCiphertext has changed.
	// The caller must re-encapsulate (via ReEncapsulate) after the final tally
	// to produce a valid MAC. Re-encapsulating after every intermediate operation
	// is unnecessary and expensive; it should be done once after all homomorphic
	// operations are complete.
	return &HybridCiphertext{
		KyberCiphertext:    ct.KyberCiphertext,
		PaillierCiphertext: productCiphertext,
		Salt:               ct.Salt,
		MAC:                ct.MAC,
	}
}

// SerializeHybridCiphertext serializes a hybrid ciphertext to JSON
func SerializeHybridCiphertext(ct *HybridCiphertext) ([]byte, error) {
	return json.Marshal(ct)
}

// DeserializeHybridCiphertext deserializes a hybrid ciphertext from JSON
func DeserializeHybridCiphertext(data []byte) (*HybridCiphertext, error) {
	var ct HybridCiphertext
	err := json.Unmarshal(data, &ct)
	if err != nil {
		return nil, err
	}
	return &ct, nil
}

// computeMAC computes HMAC-SHA256 using the standard crypto/hmac construction.
func computeMAC(data []byte, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// verifyMAC verifies a MAC using constant-time comparison to prevent timing attacks.
func verifyMAC(mac1, mac2 []byte) bool {
	return subtle.ConstantTimeCompare(mac1, mac2) == 1
}

// ReEncapsulate re-encapsulates a hybrid ciphertext with new Kyber encryption
// Useful after homomorphic operations to refresh the Kyber layer
func ReEncapsulate(
	ct *HybridCiphertext,
	kyberPublicKey kem.PublicKey,
) (*HybridCiphertext, error) {
	// Encapsulate with new Kyber encryption
	encap, err := EncapsulateWithPublicKey(kyberPublicKey)
	if err != nil {
		return nil, err
	}

	// Generate new salt
	salt, err := GenerateRandomSalt()
	if err != nil {
		return nil, err
	}

	// Derive new key
	derivedKey := DeriveKey(encap.SharedKey, salt)

	// Compute new MAC
	mac := computeMAC(ct.PaillierCiphertext.Bytes(), derivedKey)

	return &HybridCiphertext{
		KyberCiphertext:    encap.Ciphertext,
		PaillierCiphertext: ct.PaillierCiphertext,
		Salt:               salt,
		MAC:                mac,
	}, nil
}
