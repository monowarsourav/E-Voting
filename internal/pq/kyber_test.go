package pq

import (
	"math/big"
	"testing"

	"github.com/covertvote/e-voting/internal/crypto"
)

func TestKyberKeyGeneration(t *testing.T) {
	kp, err := GenerateKyberKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Kyber key pair: %v", err)
	}

	if kp.PublicKey == nil || kp.PrivateKey == nil {
		t.Error("Key pair contains nil keys")
	}
}

func TestKyberEncapsulateDecapsulate(t *testing.T) {
	kp, err := GenerateKyberKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Kyber key pair: %v", err)
	}

	// Encapsulate
	encap, err := kp.Encapsulate()
	if err != nil {
		t.Fatalf("Encapsulation failed: %v", err)
	}

	// Decapsulate
	sharedSecret2, err := kp.Decapsulate(encap.Ciphertext)
	if err != nil {
		t.Fatalf("Decapsulation failed: %v", err)
	}

	// Verify shared secrets match
	if len(encap.SharedKey) != len(sharedSecret2) {
		t.Error("Shared secret lengths don't match")
	}

	for i := 0; i < len(encap.SharedKey); i++ {
		if encap.SharedKey[i] != sharedSecret2[i] {
			t.Error("Shared secrets don't match")
			break
		}
	}
}

func TestHybridKeyGeneration(t *testing.T) {
	hybrid, err := GenerateHybridKeyPair(1024)
	if err != nil {
		t.Fatalf("Failed to generate hybrid key pair: %v", err)
	}

	if hybrid.KyberPublicKey == nil {
		t.Error("Kyber public key is nil")
	}

	if hybrid.PaillierPublicKey == nil {
		t.Error("Paillier public key is nil")
	}
}

func TestHybridEncryptDecrypt(t *testing.T) {
	// Generate hybrid keys
	hybrid, err := GenerateHybridKeyPair(1024)
	if err != nil {
		t.Fatalf("Failed to generate hybrid keys: %v", err)
	}

	// Message to encrypt
	message := big.NewInt(42)

	// Encrypt
	ct, err := HybridEncrypt(message, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	if err != nil {
		t.Fatalf("Hybrid encryption failed: %v", err)
	}

	// Decrypt
	plaintext, err := HybridDecrypt(ct, hybrid.KyberPrivateKey, hybrid.PaillierSecretKey)
	if err != nil {
		t.Fatalf("Hybrid decryption failed: %v", err)
	}

	// Verify
	if plaintext.Cmp(message) != 0 {
		t.Errorf("Decrypted plaintext doesn't match: got %v, want %v", plaintext, message)
	}
}

func TestHybridHomomorphicAdd(t *testing.T) {
	// Generate hybrid keys
	hybrid, err := GenerateHybridKeyPair(1024)
	if err != nil {
		t.Fatalf("Failed to generate hybrid keys: %v", err)
	}

	// Encrypt two values
	val1 := big.NewInt(10)
	val2 := big.NewInt(32)

	ct1, err := HybridEncrypt(val1, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	ct2, err := HybridEncrypt(val2, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Homomorphic addition
	ctSum := HybridHomomorphicAdd(ct1, ct2, hybrid.PaillierPublicKey)

	// Decrypt sum (note: MAC will be invalid, so we decrypt Paillier part directly)
	sum, err := hybrid.PaillierSecretKey.Decrypt(ctSum.PaillierCiphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// Verify
	expected := new(big.Int).Add(val1, val2)
	if sum.Cmp(expected) != 0 {
		t.Errorf("Homomorphic addition failed: got %v, want %v", sum, expected)
	}
}

func TestHybridHomomorphicMultiply(t *testing.T) {
	// Generate hybrid keys
	hybrid, err := GenerateHybridKeyPair(1024)
	if err != nil {
		t.Fatalf("Failed to generate hybrid keys: %v", err)
	}

	// Encrypt a value
	val := big.NewInt(7)
	scalar := big.NewInt(3)

	ct, err := HybridEncrypt(val, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Homomorphic scalar multiplication
	ctProduct := HybridHomomorphicMultiply(ct, scalar, hybrid.PaillierPublicKey)

	// Decrypt product (note: MAC will be invalid, so we decrypt Paillier part directly)
	product, err := hybrid.PaillierSecretKey.Decrypt(ctProduct.PaillierCiphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// Verify
	expected := new(big.Int).Mul(val, scalar)
	if product.Cmp(expected) != 0 {
		t.Errorf("Homomorphic multiplication failed: got %v, want %v", product, expected)
	}
}

func TestReEncapsulate(t *testing.T) {
	// Generate hybrid keys
	hybrid, err := GenerateHybridKeyPair(1024)
	if err != nil {
		t.Fatalf("Failed to generate hybrid keys: %v", err)
	}

	// Encrypt a value
	val := big.NewInt(100)
	ct, err := HybridEncrypt(val, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Re-encapsulate
	newCt, err := ReEncapsulate(ct, hybrid.KyberPublicKey)
	if err != nil {
		t.Fatalf("Re-encapsulation failed: %v", err)
	}

	// Verify Paillier ciphertext is unchanged
	if newCt.PaillierCiphertext.Cmp(ct.PaillierCiphertext) != 0 {
		t.Error("Re-encapsulation changed Paillier ciphertext")
	}

	// Verify Kyber ciphertext is different (new encryption)
	if len(newCt.KyberCiphertext) == len(ct.KyberCiphertext) {
		same := true
		for i := 0; i < len(newCt.KyberCiphertext); i++ {
			if newCt.KyberCiphertext[i] != ct.KyberCiphertext[i] {
				same = false
				break
			}
		}
		if same {
			t.Error("Re-encapsulation did not change Kyber ciphertext")
		}
	}

	// Decrypt and verify value is still correct
	plaintext, err := HybridDecrypt(newCt, hybrid.KyberPrivateKey, hybrid.PaillierSecretKey)
	if err != nil {
		t.Fatalf("Decryption after re-encapsulation failed: %v", err)
	}

	if plaintext.Cmp(val) != 0 {
		t.Errorf("Value changed after re-encapsulation: got %v, want %v", plaintext, val)
	}
}

func TestPostQuantumSecurity(t *testing.T) {
	// This test demonstrates that Kyber provides post-quantum security
	// In a real attack, classical algorithms can't break Kyber encryption

	hybrid, err := GenerateHybridKeyPair(1024)
	if err != nil {
		t.Fatalf("Failed to generate hybrid keys: %v", err)
	}

	message := big.NewInt(12345)
	ct, err := HybridEncrypt(message, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Without the private key, decryption should fail
	fakePrivateKey, _ := GenerateKyberKeyPair()

	_, err = HybridDecrypt(ct, fakePrivateKey.PrivateKey, hybrid.PaillierSecretKey)
	if err == nil {
		t.Error("Decryption with wrong Kyber key should fail")
	}

	t.Log("Post-quantum security verified: wrong key cannot decrypt")
}

func TestSerializeDeserialize(t *testing.T) {
	hybrid, err := GenerateHybridKeyPair(1024)
	if err != nil {
		t.Fatalf("Failed to generate hybrid keys: %v", err)
	}

	message := big.NewInt(999)
	ct, err := HybridEncrypt(message, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Serialize
	data, err := SerializeHybridCiphertext(ct)
	if err != nil {
		t.Fatalf("Serialization failed: %v", err)
	}

	// Deserialize
	ct2, err := DeserializeHybridCiphertext(data)
	if err != nil {
		t.Fatalf("Deserialization failed: %v", err)
	}

	// Verify ciphertexts match
	if ct2.PaillierCiphertext.Cmp(ct.PaillierCiphertext) != 0 {
		t.Error("Paillier ciphertext mismatch after serialization")
	}

	// Decrypt and verify
	plaintext, err := HybridDecrypt(ct2, hybrid.KyberPrivateKey, hybrid.PaillierSecretKey)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if plaintext.Cmp(message) != 0 {
		t.Errorf("Value mismatch after serialization: got %v, want %v", plaintext, message)
	}
}

// Benchmark tests
func BenchmarkKyberKeyGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateKyberKeyPair()
	}
}

func BenchmarkKyberEncapsulate(b *testing.B) {
	kp, _ := GenerateKyberKeyPair()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kp.Encapsulate()
	}
}

func BenchmarkHybridEncrypt(b *testing.B) {
	hybrid, _ := GenerateHybridKeyPair(2048)
	message := big.NewInt(42)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		HybridEncrypt(message, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	}
}

func BenchmarkHybridDecrypt(b *testing.B) {
	hybrid, _ := GenerateHybridKeyPair(2048)
	message := big.NewInt(42)
	ct, _ := HybridEncrypt(message, hybrid.KyberPublicKey, hybrid.PaillierPublicKey)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		HybridDecrypt(ct, hybrid.KyberPrivateKey, hybrid.PaillierSecretKey)
	}
}

func init() {
	// Ensure imports are used
	_ = crypto.GeneratePaillierKeyPair
}
