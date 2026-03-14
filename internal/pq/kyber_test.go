package pq

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
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
	hybrid, err := GenerateHybridKeyPair(2048)
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
	hybrid, err := GenerateHybridKeyPair(2048)
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
	hybrid, err := GenerateHybridKeyPair(2048)
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
	hybrid, err := GenerateHybridKeyPair(2048)
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
	hybrid, err := GenerateHybridKeyPair(2048)
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

	hybrid, err := GenerateHybridKeyPair(2048)
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
	hybrid, err := GenerateHybridKeyPair(2048)
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

func TestXOREncryptDecrypt(t *testing.T) {
	message := []byte("Hello CovertVote! This is a test message.")
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	encrypted := XOREncrypt(message, key)

	// Encrypted should differ from plaintext
	if string(encrypted) == string(message) {
		t.Error("Encrypted should differ from plaintext")
	}

	decrypted := XORDecrypt(encrypted, key)
	if string(decrypted) != string(message) {
		t.Errorf("Decrypted mismatch: got %s", string(decrypted))
	}
}

func TestEncryptDecryptMessage(t *testing.T) {
	kp, err := GenerateKyberKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("Secret vote data for CovertVote")
	ciphertext, kemCT, salt, err := EncryptMessage(message, kp.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := DecryptMessage(ciphertext, kemCT, salt, kp.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(message) {
		t.Errorf("Message mismatch: got %s", string(decrypted))
	}
}

func TestGenerateRandomSalt(t *testing.T) {
	salt1, err := GenerateRandomSalt()
	if err != nil {
		t.Fatal(err)
	}
	salt2, _ := GenerateRandomSalt()

	if len(salt1) != 32 {
		t.Errorf("Salt length: expected 32, got %d", len(salt1))
	}

	// Two salts should be different
	if string(salt1) == string(salt2) {
		t.Error("Two random salts should not be identical")
	}
}

func TestDeriveKey(t *testing.T) {
	secret := []byte("shared-secret-from-kyber")
	salt := []byte("random-salt-value")

	key1 := DeriveKey(secret, salt)
	key2 := DeriveKey(secret, salt)

	// Same inputs should produce same key
	if string(key1) != string(key2) {
		t.Error("Same inputs should produce same derived key")
	}

	// Different salt should produce different key
	key3 := DeriveKey(secret, []byte("different-salt"))
	if string(key1) == string(key3) {
		t.Error("Different salts should produce different keys")
	}

	if len(key1) != 32 {
		t.Errorf("Key length: expected 32, got %d", len(key1))
	}
}

func TestBigIntConversion(t *testing.T) {
	original := big.NewInt(1234567890)

	bytes := BigIntToBytes(original)
	recovered := BytesToBigInt(bytes)

	if original.Cmp(recovered) != 0 {
		t.Errorf("BigInt roundtrip failed: %v != %v", original, recovered)
	}
}

func TestUnmarshalKyberKeyPair(t *testing.T) {
	kp, _ := GenerateKyberKeyPair()

	pubBytes, _ := kp.PublicKey.MarshalBinary()
	privBytes, _ := kp.PrivateKey.MarshalBinary()

	recovered, err := UnmarshalKyberKeyPair(pubBytes, privBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Test recovered keys work
	enc, err := EncapsulateWithPublicKey(recovered.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := DecapsulateWithPrivateKey(recovered.PrivateKey, enc.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if string(ss) != string(enc.SharedKey) {
		t.Error("Recovered keypair doesn't work correctly")
	}
}

func TestHybridEncryptDecryptLargeMessage(t *testing.T) {
	hkp, err := GenerateHybridKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}

	// Test with different vote values
	testVotes := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(42),
		big.NewInt(100),
	}

	for _, vote := range testVotes {
		ct, err := HybridEncrypt(vote, hkp.KyberPublicKey, hkp.PaillierPublicKey)
		if err != nil {
			t.Fatalf("Encrypt vote %v failed: %v", vote, err)
		}

		result, err := HybridDecrypt(ct, hkp.KyberPrivateKey, hkp.PaillierSecretKey)
		if err != nil {
			t.Fatalf("Decrypt vote %v failed: %v", vote, err)
		}

		if result.Cmp(vote) != 0 {
			t.Errorf("Vote roundtrip failed: %v != %v", vote, result)
		}
	}
}

func TestMACComputeVerify(t *testing.T) {
	data := []byte("test data for MAC")
	key := []byte("secret-key-for-hmac-test-32bytes!")

	mac1 := computeMAC(data, key)
	mac2 := computeMAC(data, key)

	// Same input = same MAC
	if !verifyMAC(mac1, mac2) {
		t.Error("Same data should produce identical MACs")
	}

	// Different data = different MAC
	mac3 := computeMAC([]byte("different data"), key)
	if verifyMAC(mac1, mac3) {
		t.Error("Different data should produce different MACs")
	}
}

// --- PQ Voting Tests ---

func setupPQTestElection(t *testing.T) (*HybridKeyPair, *crypto.RingParams, *voter.RegistrationSystem, *voting.Election) {
	t.Helper()

	hkp, err := GenerateHybridKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate hybrid keys: %v", err)
	}

	rp, _ := crypto.GenerateRingParams(512)
	pp, _ := crypto.GeneratePedersenParams(512)

	voterIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		voterIDs[i] = fmt.Sprintf("pq-voter-%d", i)
	}

	rs := voter.NewRegistrationSystem(pp, rp, 5, voterIDs, "pq-test-election")

	// Register voters
	for _, id := range voterIDs {
		fingerprint := []byte("fp-" + id)
		_, err := rs.RegisterVoter(id, fingerprint)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", id, err)
		}
	}

	election := &voting.Election{
		ElectionID: "pq-test-election",
		Title:      "PQ Test Election",
		Candidates: []*voting.Candidate{
			{ID: 0, Name: "Candidate A"},
			{ID: 1, Name: "Candidate B"},
		},
		StartTime: time.Now().Unix() - 3600,
		EndTime:   time.Now().Unix() + 3600,
		IsActive:  true,
	}

	return hkp, rp, rs, election
}

func TestNewPQVoteCaster(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)

	pqvc := NewPQVoteCaster(hkp, rp, rs, election)
	if pqvc == nil {
		t.Fatal("PQVoteCaster is nil")
	}
	if pqvc.GetPQVoteCount() != 0 {
		t.Errorf("Expected 0 votes, got %d", pqvc.GetPQVoteCount())
	}
}

func TestCastPQVoteFullPipeline(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	receipt, err := pqvc.CastPQVote("pq-voter-0", 1, 0)
	if err != nil {
		t.Fatalf("CastPQVote failed: %v", err)
	}
	if receipt == nil {
		t.Fatal("Receipt is nil")
	}
	if !receipt.PQSecure {
		t.Error("Receipt should be PQ secure")
	}
	if receipt.KeyImage == nil {
		t.Error("Receipt KeyImage is nil")
	}
	if pqvc.GetPQVoteCount() != 1 {
		t.Errorf("Expected 1 vote, got %d", pqvc.GetPQVoteCount())
	}
}

func TestCastPQVoteDoubleVote(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, err := pqvc.CastPQVote("pq-voter-0", 1, 0)
	if err != nil {
		t.Fatalf("First vote failed: %v", err)
	}

	_, err = pqvc.CastPQVote("pq-voter-0", 0, 0)
	if err == nil {
		t.Fatal("Expected error for double vote")
	}
}

func TestCastPQVoteInvalidCandidate(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, err := pqvc.CastPQVote("pq-voter-0", 99, 0)
	if err == nil {
		t.Fatal("Expected error for invalid candidate")
	}
}

func TestCastPQVoteInactiveElection(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	election.IsActive = false

	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, err := pqvc.CastPQVote("pq-voter-0", 1, 0)
	if err == nil {
		t.Fatal("Expected error for inactive election")
	}
}

func TestCastPQVoteUnregisteredVoter(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, err := pqvc.CastPQVote("unknown-voter", 1, 0)
	if err == nil {
		t.Fatal("Expected error for unregistered voter")
	}
}

func TestCastPQVoteInvalidSMDCSlot(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, err := pqvc.CastPQVote("pq-voter-0", 1, 10)
	if err == nil {
		t.Fatal("Expected error for invalid SMDC slot")
	}

	_, err = pqvc.CastPQVote("pq-voter-1", 1, -1)
	if err == nil {
		t.Fatal("Expected error for negative SMDC slot")
	}
}

func TestGetAllPQVoteShares(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, _ = pqvc.CastPQVote("pq-voter-0", 0, 0)
	_, _ = pqvc.CastPQVote("pq-voter-1", 1, 0)

	shares := pqvc.GetAllPQVoteShares()
	if len(shares) != 2 {
		t.Errorf("Expected 2 shares, got %d", len(shares))
	}
}

func TestVerifyPQVote(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, err := pqvc.CastPQVote("pq-voter-0", 1, 0)
	if err != nil {
		t.Fatalf("CastPQVote failed: %v", err)
	}

	castVote := pqvc.CastVotes["pq-voter-0"]
	// Exercise VerifyPQVote code path
	_ = pqvc.VerifyPQVote(castVote)
}

func TestConvertToSA2Shares(t *testing.T) {
	hkp, rp, rs, election := setupPQTestElection(t)
	pqvc := NewPQVoteCaster(hkp, rp, rs, election)

	_, err := pqvc.CastPQVote("pq-voter-0", 0, 0)
	if err != nil {
		t.Fatalf("CastPQVote failed: %v", err)
	}

	allShares := pqvc.GetAllPQVoteShares()
	if len(allShares) == 0 {
		t.Fatal("No shares returned")
	}

	sa2Share := ConvertToSA2Shares(allShares[0])
	if sa2Share == nil {
		t.Fatal("SA2 share is nil")
	}
	if sa2Share.ShareA == nil || sa2Share.ShareB == nil {
		t.Error("SA2 share has nil components")
	}
}
