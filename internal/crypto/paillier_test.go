package crypto

import (
	"math/big"
	"testing"
)

func TestPaillierEncryptDecrypt(t *testing.T) {
	sk, err := GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatalf("Key generation failed: %v", err)
	}

	// Test encryption/decryption
	message := big.NewInt(42)

	ciphertext, err := sk.PublicKey.Encrypt(message)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := sk.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if message.Cmp(decrypted) != 0 {
		t.Errorf("Decryption mismatch: expected %v, got %v", message, decrypted)
	}
}

func TestPaillierHomomorphicAdd(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	m1 := big.NewInt(15)
	m2 := big.NewInt(27)
	expected := big.NewInt(42) // 15 + 27

	c1, _ := pk.Encrypt(m1)
	c2, _ := pk.Encrypt(m2)

	// Homomorphic addition
	cSum := pk.Add(c1, c2)

	// Decrypt sum
	result, _ := sk.Decrypt(cSum)

	if expected.Cmp(result) != 0 {
		t.Errorf("Homomorphic add failed: expected %v, got %v", expected, result)
	}
}

func TestPaillierHomomorphicMultiply(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	m := big.NewInt(7)
	k := big.NewInt(6)
	expected := big.NewInt(42) // 7 × 6

	c, _ := pk.Encrypt(m)

	// Scalar multiplication
	cProduct := pk.Multiply(c, k)

	// Decrypt
	result, _ := sk.Decrypt(cProduct)

	if expected.Cmp(result) != 0 {
		t.Errorf("Scalar multiply failed: expected %v, got %v", expected, result)
	}
}

func TestPaillierAddMultiple(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	// Encrypt multiple values
	values := []*big.Int{
		big.NewInt(10),
		big.NewInt(20),
		big.NewInt(30),
	}

	var ciphertexts []*big.Int
	for _, v := range values {
		c, _ := pk.Encrypt(v)
		ciphertexts = append(ciphertexts, c)
	}

	// Add all ciphertexts
	cSum := pk.AddMultiple(ciphertexts)

	// Decrypt
	result, _ := sk.Decrypt(cSum)

	expected := big.NewInt(60) // 10 + 20 + 30
	if expected.Cmp(result) != 0 {
		t.Errorf("AddMultiple failed: expected %v, got %v", expected, result)
	}
}

func TestPaillierKeyGenRejectsSmallKeys(t *testing.T) {
	_, err := GeneratePaillierKeyPair(1024)
	if err == nil {
		t.Fatal("expected error for 1024-bit key, got nil")
	}
	_, err = GeneratePaillierKeyPair(512)
	if err == nil {
		t.Fatal("expected error for 512-bit key, got nil")
	}
}
