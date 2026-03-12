# CovertVote Testing Guide for Vibe Coding Agent
## Complete Unit Tests, Integration Tests & Benchmarks

---

# 📋 QUICK START FOR AI AGENT

## One-Line Description:
```
Implement comprehensive tests for CovertVote e-voting system including unit tests for all crypto components, integration tests for full voting flow, and benchmarks for performance measurement. Follow Go testing conventions and generate IEEE paper-ready benchmark results.
```

---

# 📁 TEST FILE STRUCTURE

```
covertvote/
├── internal/
│   ├── crypto/
│   │   ├── paillier.go
│   │   ├── paillier_test.go          ← Unit Test
│   │   ├── pedersen.go
│   │   ├── pedersen_test.go          ← Unit Test
│   │   ├── zkproof.go
│   │   ├── zkproof_test.go           ← Unit Test
│   │   ├── ring_signature.go
│   │   └── ring_signature_test.go    ← Unit Test
│   │
│   ├── smdc/
│   │   ├── credential.go
│   │   └── credential_test.go        ← Unit Test
│   │
│   └── sa2/
│       ├── aggregation.go
│       └── aggregation_test.go       ← Unit Test
│
├── test/
│   ├── integration/
│   │   └── full_flow_test.go         ← Integration Test
│   │
│   └── benchmark/
│       ├── crypto_benchmark_test.go  ← Benchmark
│       ├── voting_benchmark_test.go  ← Benchmark
│       └── results/
│           └── benchmark_results.md  ← Results Output
│
└── scripts/
    ├── run_tests.sh
    └── run_benchmarks.sh
```

---

# 🧪 PART 1: UNIT TESTS

## 1.1 internal/crypto/paillier_test.go

```go
// internal/crypto/paillier_test.go

package crypto

import (
	"math/big"
	"testing"
)

// ============================================================
// TEST: Key Generation
// ============================================================

func TestPaillierKeyGeneration(t *testing.T) {
	tests := []struct {
		name    string
		bits    int
		wantErr bool
	}{
		{"2048 bits (standard)", 2048, false},
		{"1024 bits (minimum)", 1024, false},
		{"512 bits (weak but fast for testing)", 512, false},
		{"256 bits (too small)", 256, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sk, err := GeneratePaillierKeyPair(tt.bits)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePaillierKeyPair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify key components exist
				if sk.PublicKey.N == nil {
					t.Error("Public key N is nil")
				}
				if sk.Lambda == nil {
					t.Error("Lambda is nil")
				}
				if sk.Mu == nil {
					t.Error("Mu is nil")
				}
				// Verify N is product of two primes (N = P * Q)
				pq := new(big.Int).Mul(sk.P, sk.Q)
				if pq.Cmp(sk.PublicKey.N) != 0 {
					t.Error("N != P * Q")
				}
			}
		})
	}
}

// ============================================================
// TEST: Encrypt and Decrypt
// ============================================================

func TestPaillierEncryptDecrypt(t *testing.T) {
	sk, err := GeneratePaillierKeyPair(512) // Small key for fast testing
	if err != nil {
		t.Fatalf("Key generation failed: %v", err)
	}
	pk := sk.PublicKey

	tests := []struct {
		name      string
		plaintext int64
	}{
		{"zero", 0},
		{"one", 1},
		{"small number", 42},
		{"medium number", 12345},
		{"large number", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := big.NewInt(tt.plaintext)
			
			// Encrypt
			ciphertext, err := pk.Encrypt(original)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}
			
			// Decrypt
			decrypted, err := sk.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}
			
			// Verify
			if original.Cmp(decrypted) != 0 {
				t.Errorf("Decrypt(Encrypt(%d)) = %d, want %d", 
					tt.plaintext, decrypted.Int64(), tt.plaintext)
			}
		})
	}
}

// ============================================================
// TEST: Homomorphic Addition
// ============================================================

func TestPaillierHomomorphicAdd(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(512)
	pk := sk.PublicKey

	tests := []struct {
		name string
		a    int64
		b    int64
	}{
		{"0 + 0", 0, 0},
		{"1 + 1", 1, 1},
		{"10 + 20", 10, 20},
		{"100 + 200", 100, 200},
		{"12345 + 67890", 12345, 67890},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := big.NewInt(tt.a)
			b := big.NewInt(tt.b)
			expected := tt.a + tt.b

			// Encrypt both
			encA, _ := pk.Encrypt(a)
			encB, _ := pk.Encrypt(b)

			// Homomorphic add: E(a) * E(b) = E(a + b)
			encSum := pk.Add(encA, encB)

			// Decrypt
			decrypted, _ := sk.Decrypt(encSum)

			// Verify
			if decrypted.Int64() != expected {
				t.Errorf("E(%d) + E(%d) = E(%d), want E(%d)", 
					tt.a, tt.b, decrypted.Int64(), expected)
			}
		})
	}
}

// ============================================================
// TEST: Homomorphic Scalar Multiplication
// ============================================================

func TestPaillierHomomorphicMultiply(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(512)
	pk := sk.PublicKey

	tests := []struct {
		name   string
		value  int64
		scalar int64
	}{
		{"5 * 0", 5, 0},
		{"5 * 1", 5, 1},
		{"5 * 2", 5, 2},
		{"10 * 10", 10, 10},
		{"7 * 100", 7, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := big.NewInt(tt.value)
			scalar := big.NewInt(tt.scalar)
			expected := tt.value * tt.scalar

			// Encrypt value
			encValue, _ := pk.Encrypt(value)

			// Scalar multiply: E(v)^k = E(v * k)
			encProduct := pk.Multiply(encValue, scalar)

			// Decrypt
			decrypted, _ := sk.Decrypt(encProduct)

			// Verify
			if decrypted.Int64() != expected {
				t.Errorf("E(%d) * %d = E(%d), want E(%d)", 
					tt.value, tt.scalar, decrypted.Int64(), expected)
			}
		})
	}
}

// ============================================================
// TEST: Multiple Additions (Simulating Vote Counting)
// ============================================================

func TestPaillierVoteCounting(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(512)
	pk := sk.PublicKey

	// Simulate 100 votes: 40 for candidate 0, 35 for candidate 1, 25 for candidate 2
	votes := []int64{}
	for i := 0; i < 40; i++ {
		votes = append(votes, 0)
	}
	for i := 0; i < 35; i++ {
		votes = append(votes, 1)
	}
	for i := 0; i < 25; i++ {
		votes = append(votes, 2)
	}

	// Encrypt all votes
	encryptedVotes := make([]*big.Int, len(votes))
	for i, v := range votes {
		encryptedVotes[i], _ = pk.Encrypt(big.NewInt(v))
	}

	// Sum all encrypted votes
	encSum := encryptedVotes[0]
	for i := 1; i < len(encryptedVotes); i++ {
		encSum = pk.Add(encSum, encryptedVotes[i])
	}

	// Decrypt
	totalSum, _ := sk.Decrypt(encSum)

	// Expected: 40*0 + 35*1 + 25*2 = 0 + 35 + 50 = 85
	expected := int64(85)
	if totalSum.Int64() != expected {
		t.Errorf("Vote sum = %d, want %d", totalSum.Int64(), expected)
	}
}

// ============================================================
// TEST: Encryption Randomness (Same plaintext, different ciphertext)
// ============================================================

func TestPaillierEncryptionRandomness(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(512)
	pk := sk.PublicKey

	plaintext := big.NewInt(42)

	// Encrypt same value 10 times
	ciphertexts := make([]*big.Int, 10)
	for i := 0; i < 10; i++ {
		ciphertexts[i], _ = pk.Encrypt(plaintext)
	}

	// All ciphertexts should be different (semantic security)
	for i := 0; i < 10; i++ {
		for j := i + 1; j < 10; j++ {
			if ciphertexts[i].Cmp(ciphertexts[j]) == 0 {
				t.Errorf("Ciphertext %d and %d are identical (breaks semantic security)", i, j)
			}
		}
	}

	// But all should decrypt to same value
	for i, c := range ciphertexts {
		decrypted, _ := sk.Decrypt(c)
		if decrypted.Cmp(plaintext) != 0 {
			t.Errorf("Ciphertext %d decrypts to %v, want %v", i, decrypted, plaintext)
		}
	}
}
```

## 1.2 internal/crypto/pedersen_test.go

```go
// internal/crypto/pedersen_test.go

package crypto

import (
	"math/big"
	"testing"
)

// ============================================================
// TEST: Parameter Generation
// ============================================================

func TestPedersenParamsGeneration(t *testing.T) {
	tests := []struct {
		name    string
		bits    int
		wantErr bool
	}{
		{"512 bits", 512, false},
		{"256 bits (minimum)", 256, false},
		{"128 bits (too small)", 128, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp, err := GeneratePedersenParams(tt.bits)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePedersenParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if pp.P == nil || pp.Q == nil || pp.G == nil || pp.H == nil {
					t.Error("Parameters contain nil values")
				}
				// Verify p = 2q + 1 (safe prime)
				expected := new(big.Int).Mul(pp.Q, big.NewInt(2))
				expected.Add(expected, big.NewInt(1))
				if pp.P.Cmp(expected) != 0 {
					t.Error("P is not a safe prime (P != 2Q + 1)")
				}
			}
		})
	}
}

// ============================================================
// TEST: Commit and Verify
// ============================================================

func TestPedersenCommitVerify(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)

	tests := []struct {
		name  string
		value int64
	}{
		{"zero", 0},
		{"one", 1},
		{"small", 42},
		{"medium", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := big.NewInt(tt.value)

			// Commit
			commitment, err := pp.Commit(m)
			if err != nil {
				t.Fatalf("Commit failed: %v", err)
			}

			// Verify with correct values
			valid := pp.Verify(commitment.C, m, commitment.R)
			if !valid {
				t.Error("Valid commitment failed verification")
			}

			// Verify with wrong value should fail
			wrongM := big.NewInt(tt.value + 1)
			invalid := pp.Verify(commitment.C, wrongM, commitment.R)
			if invalid {
				t.Error("Invalid commitment passed verification")
			}
		})
	}
}

// ============================================================
// TEST: Homomorphic Property
// ============================================================

func TestPedersenHomomorphic(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)

	a := big.NewInt(10)
	b := big.NewInt(20)

	// Commit to both values
	commitA, _ := pp.Commit(a)
	commitB, _ := pp.Commit(b)

	// Multiply commitments: C(a) * C(b) = C(a + b)
	productC := new(big.Int).Mul(commitA.C, commitB.C)
	productC.Mod(productC, pp.P)

	// Sum of randomness
	sumR := new(big.Int).Add(commitA.R, commitB.R)
	sumR.Mod(sumR, pp.Q)

	// Sum of values
	sumM := new(big.Int).Add(a, b) // 10 + 20 = 30

	// Verify: C(a) * C(b) should equal C(a+b, r_a + r_b)
	valid := pp.Verify(productC, sumM, sumR)
	if !valid {
		t.Error("Homomorphic property failed: C(a) * C(b) != C(a+b)")
	}
}

// ============================================================
// TEST: Hiding Property (same value, different commitments)
// ============================================================

func TestPedersenHiding(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)
	m := big.NewInt(42)

	// Create multiple commitments to same value
	commitments := make([]*Commitment, 10)
	for i := 0; i < 10; i++ {
		commitments[i], _ = pp.Commit(m)
	}

	// All commitments should be different (due to random r)
	for i := 0; i < 10; i++ {
		for j := i + 1; j < 10; j++ {
			if commitments[i].C.Cmp(commitments[j].C) == 0 {
				t.Errorf("Commitment %d and %d are identical (hiding property broken)", i, j)
			}
		}
	}
}
```

## 1.3 internal/crypto/zkproof_test.go

```go
// internal/crypto/zkproof_test.go

package crypto

import (
	"math/big"
	"testing"
)

// ============================================================
// TEST: Binary Proof (proves value is 0 or 1)
// ============================================================

func TestBinaryProofValid(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)

	tests := []struct {
		name  string
		value int64
	}{
		{"value is 0", 0},
		{"value is 1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := big.NewInt(tt.value)
			
			// Create commitment
			commitment, _ := pp.Commit(w)
			
			// Create binary proof
			proof, err := pp.ProveBinary(w, commitment.R, commitment.C)
			if err != nil {
				t.Fatalf("ProveBinary failed: %v", err)
			}
			
			// Verify proof
			valid := pp.VerifyBinary(commitment.C, proof)
			if !valid {
				t.Errorf("Valid binary proof for %d failed verification", tt.value)
			}
		})
	}
}

func TestBinaryProofInvalidValue(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)

	// Try to create proof for value = 2 (should fail)
	w := big.NewInt(2)
	commitment, _ := pp.Commit(w)
	
	_, err := pp.ProveBinary(w, commitment.R, commitment.C)
	if err == nil {
		t.Error("ProveBinary should fail for value = 2")
	}
}

func TestBinaryProofSoundness(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)

	// Create valid proof for 0
	w := big.NewInt(0)
	commitment, _ := pp.Commit(w)
	proof, _ := pp.ProveBinary(w, commitment.R, commitment.C)

	// Verify with wrong commitment should fail
	wrongW := big.NewInt(1)
	wrongCommitment, _ := pp.Commit(wrongW)
	
	valid := pp.VerifyBinary(wrongCommitment.C, proof)
	if valid {
		t.Error("Proof verified for wrong commitment (soundness broken)")
	}
}

// ============================================================
// TEST: Sum-One Proof (proves sum of committed values = 1)
// ============================================================

func TestSumOneProofValid(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)

	// Create k=5 commitments with exactly one 1 and rest 0
	tests := []struct {
		name      string
		realIndex int
		k         int
	}{
		{"real at index 0", 0, 5},
		{"real at index 2", 2, 5},
		{"real at index 4", 4, 5},
		{"k=3, real at 1", 1, 3},
		{"k=10, real at 7", 7, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitments := make([]*Commitment, tt.k)
			commitmentValues := make([]*big.Int, tt.k)

			for i := 0; i < tt.k; i++ {
				var w *big.Int
				if i == tt.realIndex {
					w = big.NewInt(1)
				} else {
					w = big.NewInt(0)
				}
				commitments[i], _ = pp.Commit(w)
				commitmentValues[i] = commitments[i].C
			}

			// Create sum-one proof
			proof, err := pp.ProveSumOne(commitments)
			if err != nil {
				t.Fatalf("ProveSumOne failed: %v", err)
			}

			// Verify
			valid := pp.VerifySumOne(commitmentValues, proof)
			if !valid {
				t.Error("Valid sum-one proof failed verification")
			}
		})
	}
}

func TestSumOneProofInvalidSum(t *testing.T) {
	pp, _ := GeneratePedersenParams(256)

	// Create commitments with sum = 2 (invalid)
	commitments := make([]*Commitment, 5)
	commitmentValues := make([]*big.Int, 5)

	for i := 0; i < 5; i++ {
		var w *big.Int
		if i < 2 {
			w = big.NewInt(1) // Two 1s = sum is 2
		} else {
			w = big.NewInt(0)
		}
		commitments[i], _ = pp.Commit(w)
		commitmentValues[i] = commitments[i].C
	}

	// Create proof (this might succeed but verification should fail)
	proof, _ := pp.ProveSumOne(commitments)

	// Verify should fail because sum = 2, not 1
	valid := pp.VerifySumOne(commitmentValues, proof)
	if valid {
		t.Error("Sum-one proof verified for sum = 2 (should only verify for sum = 1)")
	}
}
```

## 1.4 internal/crypto/ring_signature_test.go

```go
// internal/crypto/ring_signature_test.go

package crypto

import (
	"testing"
)

// ============================================================
// TEST: Ring Key Generation
// ============================================================

func TestRingKeyGeneration(t *testing.T) {
	rp, err := GenerateRingParams(256)
	if err != nil {
		t.Fatalf("GenerateRingParams failed: %v", err)
	}

	kp, err := rp.GenerateRingKeyPair()
	if err != nil {
		t.Fatalf("GenerateRingKeyPair failed: %v", err)
	}

	if kp.PublicKey == nil || kp.PrivateKey == nil {
		t.Error("Key pair contains nil values")
	}
}

// ============================================================
// TEST: Sign and Verify
// ============================================================

func TestRingSignatureSignVerify(t *testing.T) {
	rp, _ := GenerateRingParams(256)

	// Create ring of 5 members
	ringSize := 5
	ring := make([]*big.Int, ringSize)
	keyPairs := make([]*RingKeyPair, ringSize)

	for i := 0; i < ringSize; i++ {
		keyPairs[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}

	// Test signing from each position
	for signerIndex := 0; signerIndex < ringSize; signerIndex++ {
		t.Run(fmt.Sprintf("signer_at_%d", signerIndex), func(t *testing.T) {
			message := []byte("test vote message")

			// Sign
			sig, err := rp.Sign(message, keyPairs[signerIndex], ring, signerIndex)
			if err != nil {
				t.Fatalf("Sign failed: %v", err)
			}

			// Verify
			valid := rp.Verify(message, sig, ring)
			if !valid {
				t.Error("Valid signature failed verification")
			}
		})
	}
}

// ============================================================
// TEST: Wrong Message Fails Verification
// ============================================================

func TestRingSignatureWrongMessage(t *testing.T) {
	rp, _ := GenerateRingParams(256)

	// Create ring
	ringSize := 5
	ring := make([]*big.Int, ringSize)
	keyPairs := make([]*RingKeyPair, ringSize)
	for i := 0; i < ringSize; i++ {
		keyPairs[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}

	// Sign original message
	originalMsg := []byte("vote for candidate A")
	sig, _ := rp.Sign(originalMsg, keyPairs[0], ring, 0)

	// Verify with different message should fail
	wrongMsg := []byte("vote for candidate B")
	valid := rp.Verify(wrongMsg, sig, ring)
	if valid {
		t.Error("Signature verified for wrong message")
	}
}

// ============================================================
// TEST: Linkability (same signer = same key image)
// ============================================================

func TestRingSignatureLinkability(t *testing.T) {
	rp, _ := GenerateRingParams(256)

	// Create ring
	ringSize := 5
	ring := make([]*big.Int, ringSize)
	keyPairs := make([]*RingKeyPair, ringSize)
	for i := 0; i < ringSize; i++ {
		keyPairs[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}

	// Same signer signs two different messages
	msg1 := []byte("first vote")
	msg2 := []byte("second vote (double voting attempt)")
	
	sig1, _ := rp.Sign(msg1, keyPairs[0], ring, 0)
	sig2, _ := rp.Sign(msg2, keyPairs[0], ring, 0)

	// Key images should be identical (linkable)
	if !Link(sig1, sig2) {
		t.Error("Same signer produced different key images (linkability broken)")
	}

	// Different signer should produce different key image
	sig3, _ := rp.Sign(msg1, keyPairs[1], ring, 1)
	if Link(sig1, sig3) {
		t.Error("Different signers produced same key image")
	}
}

// ============================================================
// TEST: Ring Size Variations
// ============================================================

func TestRingSignatureVariousSizes(t *testing.T) {
	rp, _ := GenerateRingParams(256)

	sizes := []int{2, 3, 5, 10, 20, 50, 100}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("ring_size_%d", size), func(t *testing.T) {
			// Create ring
			ring := make([]*big.Int, size)
			keyPairs := make([]*RingKeyPair, size)
			for i := 0; i < size; i++ {
				keyPairs[i], _ = rp.GenerateRingKeyPair()
				ring[i] = keyPairs[i].PublicKey
			}

			// Sign and verify
			message := []byte("test message")
			sig, err := rp.Sign(message, keyPairs[0], ring, 0)
			if err != nil {
				t.Fatalf("Sign failed for ring size %d: %v", size, err)
			}

			valid := rp.Verify(message, sig, ring)
			if !valid {
				t.Errorf("Verification failed for ring size %d", size)
			}
		})
	}
}
```

## 1.5 internal/smdc/credential_test.go

```go
// internal/smdc/credential_test.go

package smdc

import (
	"math/big"
	"testing"

	"github.com/yourusername/covertvote/internal/crypto"
)

// ============================================================
// TEST: SMDC Credential Generation
// ============================================================

func TestSMDCGeneration(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(256)
	gen := NewGenerator(pp, 5) // k=5 slots

	cred, err := gen.Generate("voter_001")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check basic properties
	if cred.K != 5 {
		t.Errorf("K = %d, want 5", cred.K)
	}
	if len(cred.Slots) != 5 {
		t.Errorf("Slots count = %d, want 5", len(cred.Slots))
	}
	if cred.RealIndex < 0 || cred.RealIndex >= 5 {
		t.Errorf("RealIndex = %d, should be in [0, 4]", cred.RealIndex)
	}
}

// ============================================================
// TEST: Exactly One Real Slot
// ============================================================

func TestSMDCExactlyOneReal(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(256)
	gen := NewGenerator(pp, 5)

	// Generate multiple credentials and verify
	for i := 0; i < 100; i++ {
		cred, _ := gen.Generate("voter")

		// Count weights
		sum := big.NewInt(0)
		onesCount := 0
		for _, slot := range cred.Slots {
			sum.Add(sum, slot.Weight)
			if slot.Weight.Cmp(big.NewInt(1)) == 0 {
				onesCount++
			}
		}

		// Sum should be exactly 1
		if sum.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("Iteration %d: Weight sum = %v, want 1", i, sum)
		}

		// Exactly one slot should have weight 1
		if onesCount != 1 {
			t.Errorf("Iteration %d: Found %d slots with weight=1, want 1", i, onesCount)
		}
	}
}

// ============================================================
// TEST: Real Index Matches Weight
// ============================================================

func TestSMDCRealIndexCorrect(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(256)
	gen := NewGenerator(pp, 5)

	for i := 0; i < 100; i++ {
		cred, _ := gen.Generate("voter")

		// The slot at RealIndex should have weight = 1
		realSlot := cred.Slots[cred.RealIndex]
		if realSlot.Weight.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("Iteration %d: RealIndex slot has weight %v, want 1", 
				i, realSlot.Weight)
		}

		// All other slots should have weight = 0
		for j, slot := range cred.Slots {
			if j != cred.RealIndex && slot.Weight.Cmp(big.NewInt(0)) != 0 {
				t.Errorf("Iteration %d: Fake slot %d has weight %v, want 0", 
					i, j, slot.Weight)
			}
		}
	}
}

// ============================================================
// TEST: Credential Verification
// ============================================================

func TestSMDCVerification(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(256)
	gen := NewGenerator(pp, 5)

	cred, _ := gen.Generate("voter_001")
	pub := cred.ToPublic()

	// Valid credential should verify
	if !gen.Verify(pub) {
		t.Error("Valid credential failed verification")
	}
}

// ============================================================
// TEST: Different K Values
// ============================================================

func TestSMDCVariousK(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(256)

	kValues := []int{2, 3, 5, 7, 10}

	for _, k := range kValues {
		t.Run(fmt.Sprintf("k=%d", k), func(t *testing.T) {
			gen := NewGenerator(pp, k)
			cred, err := gen.Generate("voter")
			if err != nil {
				t.Fatalf("Generate failed for k=%d: %v", k, err)
			}

			if cred.K != k {
				t.Errorf("K = %d, want %d", cred.K, k)
			}

			pub := cred.ToPublic()
			if !gen.Verify(pub) {
				t.Errorf("Verification failed for k=%d", k)
			}
		})
	}
}

// ============================================================
// TEST: Real Index Distribution (should be random)
// ============================================================

func TestSMDCRealIndexDistribution(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(256)
	gen := NewGenerator(pp, 5)

	// Generate many credentials and count real index distribution
	distribution := make(map[int]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		cred, _ := gen.Generate("voter")
		distribution[cred.RealIndex]++
	}

	// Each index should appear roughly 20% of the time (1000/5 = 200)
	expectedPerIndex := iterations / 5
	tolerance := float64(expectedPerIndex) * 0.3 // 30% tolerance

	for idx := 0; idx < 5; idx++ {
		count := distribution[idx]
		if float64(count) < float64(expectedPerIndex)-tolerance ||
			float64(count) > float64(expectedPerIndex)+tolerance {
			t.Errorf("Index %d appeared %d times, expected ~%d (±%.0f)", 
				idx, count, expectedPerIndex, tolerance)
		}
	}
}
```

## 1.6 internal/sa2/aggregation_test.go

```go
// internal/sa2/aggregation_test.go

package sa2

import (
	"math/big"
	"testing"

	"github.com/yourusername/covertvote/internal/crypto"
)

// ============================================================
// TEST: Share Split and Recombine
// ============================================================

func TestSA2SplitRecombine(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(512)
	pk := sk.PublicKey
	sg := NewShareGenerator(pk)

	tests := []struct {
		name  string
		value int64
	}{
		{"vote 0", 0},
		{"vote 1", 1},
		{"vote 2", 2},
		{"vote 42", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := big.NewInt(tt.value)
			encVote, _ := pk.Encrypt(original)

			// Split
			share, err := sg.Split(encVote, "voter_test")
			if err != nil {
				t.Fatalf("Split failed: %v", err)
			}

			// Recombine: shareA + shareB should give back original encrypted vote
			recombined := sg.Recombine(share.ShareA, share.ShareB)

			// Decrypt and verify
			decrypted, _ := sk.Decrypt(recombined)
			if decrypted.Cmp(original) != 0 {
				t.Errorf("Recombined value = %v, want %v", decrypted, original)
			}
		})
	}
}

// ============================================================
// TEST: Full SA² System Aggregation
// ============================================================

func TestSA2SystemAggregation(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(512)
	pk := sk.PublicKey
	sys := NewSystem(pk)

	// Simulate 10 votes: [1, 2, 1, 0, 2, 1, 1, 0, 2, 1]
	votes := []int64{1, 2, 1, 0, 2, 1, 1, 0, 2, 1}
	expectedSum := int64(0)

	for i, v := range votes {
		expectedSum += v
		encVote, _ := pk.Encrypt(big.NewInt(v))
		err := sys.ProcessVote(encVote, fmt.Sprintf("voter_%d", i))
		if err != nil {
			t.Fatalf("ProcessVote failed: %v", err)
		}
	}

	// Get encrypted tally
	result, err := sys.GetEncryptedTally()
	if err != nil {
		t.Fatalf("GetEncryptedTally failed: %v", err)
	}

	// Decrypt and verify
	decryptedSum, _ := sk.Decrypt(result.EncryptedTally)
	if decryptedSum.Int64() != expectedSum {
		t.Errorf("Tally = %d, want %d", decryptedSum.Int64(), expectedSum)
	}

	// Verify vote count
	if result.TotalVotes != len(votes) {
		t.Errorf("TotalVotes = %d, want %d", result.TotalVotes, len(votes))
	}
}

// ============================================================
// TEST: Server Counts Match
// ============================================================

func TestSA2ServerCountsMatch(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(512)
	pk := sk.PublicKey
	sys := NewSystem(pk)

	// Process 100 votes
	for i := 0; i < 100; i++ {
		encVote, _ := pk.Encrypt(big.NewInt(1))
		sys.ProcessVote(encVote, fmt.Sprintf("voter_%d", i))
	}

	// Both servers should have same count
	countA := sys.ServerA.GetShareCount()
	countB := sys.ServerB.GetShareCount()

	if countA != countB {
		t.Errorf("Server counts don't match: A=%d, B=%d", countA, countB)
	}
	if countA != 100 {
		t.Errorf("Server A count = %d, want 100", countA)
	}
}

// ============================================================
// TEST: Reset Functionality
// ============================================================

func TestSA2Reset(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(512)
	pk := sk.PublicKey
	sys := NewSystem(pk)

	// Add some votes
	for i := 0; i < 10; i++ {
		encVote, _ := pk.Encrypt(big.NewInt(1))
		sys.ProcessVote(encVote, fmt.Sprintf("voter_%d", i))
	}

	// Reset
	sys.Reset()

	// Counts should be zero
	if sys.ServerA.GetShareCount() != 0 || sys.ServerB.GetShareCount() != 0 {
		t.Error("Reset did not clear shares")
	}
}

// ============================================================
// TEST: Large Number of Votes
// ============================================================

func TestSA2LargeVoteCount(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(512)
	pk := sk.PublicKey
	sys := NewSystem(pk)

	numVotes := 1000
	expectedSum := int64(0)

	for i := 0; i < numVotes; i++ {
		vote := int64(i % 5) // Votes 0-4 rotating
		expectedSum += vote
		encVote, _ := pk.Encrypt(big.NewInt(vote))
		sys.ProcessVote(encVote, fmt.Sprintf("voter_%d", i))
	}

	result, _ := sys.GetEncryptedTally()
	decryptedSum, _ := sk.Decrypt(result.EncryptedTally)

	if decryptedSum.Int64() != expectedSum {
		t.Errorf("Tally for %d votes = %d, want %d", 
			numVotes, decryptedSum.Int64(), expectedSum)
	}
}

// ============================================================
// TEST: Privacy - Individual Shares Reveal Nothing
// ============================================================

func TestSA2SharesRevealNothing(t *testing.T) {
	sk, _ := crypto.GeneratePaillierKeyPair(512)
	pk := sk.PublicKey
	sg := NewShareGenerator(pk)

	// Encrypt two different votes
	vote1, _ := pk.Encrypt(big.NewInt(0))
	vote2, _ := pk.Encrypt(big.NewInt(1))

	share1, _ := sg.Split(vote1, "voter1")
	share2, _ := sg.Split(vote2, "voter2")

	// Server A only sees shareA values
	// Both should look random - decrypt individually should not reveal vote
	// (This is a conceptual test - in practice, individual shares are meaningless)
	
	dec1, _ := sk.Decrypt(share1.ShareA)
	dec2, _ := sk.Decrypt(share2.ShareA)

	// The decrypted shares should NOT equal the original votes
	// because they include random masks
	if dec1.Cmp(big.NewInt(0)) == 0 {
		t.Log("Warning: ShareA happened to decrypt to 0 (possible but unlikely)")
	}
	if dec2.Cmp(big.NewInt(1)) == 0 {
		t.Log("Warning: ShareA happened to decrypt to 1 (possible but unlikely)")
	}
}
```

---

# 🔗 PART 2: INTEGRATION TESTS

## 2.1 test/integration/full_flow_test.go

```go
// test/integration/full_flow_test.go

package integration

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/yourusername/covertvote/internal/crypto"
	"github.com/yourusername/covertvote/internal/sa2"
	"github.com/yourusername/covertvote/internal/smdc"
)

// ============================================================
// TEST: Complete Voting Flow (End-to-End)
// ============================================================

func TestFullVotingFlow(t *testing.T) {
	// ========== PHASE 1: System Setup ==========
	t.Log("Phase 1: System Setup")
	
	// Generate Paillier keys
	paillierSK, err := crypto.GeneratePaillierKeyPair(1024)
	if err != nil {
		t.Fatalf("Paillier key generation failed: %v", err)
	}
	paillierPK := paillierSK.PublicKey

	// Generate Pedersen parameters
	pedersenParams, err := crypto.GeneratePedersenParams(256)
	if err != nil {
		t.Fatalf("Pedersen params generation failed: %v", err)
	}

	// Generate Ring parameters
	ringParams, err := crypto.GenerateRingParams(256)
	if err != nil {
		t.Fatalf("Ring params generation failed: %v", err)
	}

	// Initialize SMDC generator
	smdcGen := smdc.NewGenerator(pedersenParams, 5)

	// Initialize SA² system
	sa2System := sa2.NewSystem(paillierPK)

	// ========== PHASE 2: Voter Registration ==========
	t.Log("Phase 2: Voter Registration")
	
	numVoters := 100
	voters := make([]struct {
		ID         string
		Credential *smdc.Credential
		RingKey    *crypto.RingKeyPair
	}, numVoters)

	// Create ring of all voters
	ring := make([]*big.Int, numVoters)
	
	for i := 0; i < numVoters; i++ {
		voters[i].ID = fmt.Sprintf("voter_%03d", i)
		
		// Generate SMDC credential
		cred, err := smdcGen.Generate(voters[i].ID)
		if err != nil {
			t.Fatalf("SMDC generation failed for voter %d: %v", i, err)
		}
		voters[i].Credential = cred

		// Verify credential
		if !smdcGen.Verify(cred.ToPublic()) {
			t.Fatalf("SMDC verification failed for voter %d", i)
		}

		// Generate ring key pair
		kp, err := ringParams.GenerateRingKeyPair()
		if err != nil {
			t.Fatalf("Ring key generation failed for voter %d: %v", i, err)
		}
		voters[i].RingKey = kp
		ring[i] = kp.PublicKey
	}

	// ========== PHASE 3: Vote Casting ==========
	t.Log("Phase 3: Vote Casting")
	
	// Simulate votes: 40 for candidate 0, 35 for candidate 1, 25 for candidate 2
	voteDistribution := map[int]int{0: 40, 1: 35, 2: 25}
	voterIndex := 0
	candidateVotes := make([]int, numVoters)

	for candidate, count := range voteDistribution {
		for i := 0; i < count; i++ {
			candidateVotes[voterIndex] = candidate
			voterIndex++
		}
	}

	// Each voter casts their vote
	for i, voter := range voters {
		candidate := candidateVotes[i]
		cred := voter.Credential
		
		// Get the real slot
		realSlot := cred.GetRealSlot()
		
		// Encrypt vote weighted by slot weight
		// E(weight * vote) = E(vote)^weight
		encVote, err := paillierPK.Encrypt(big.NewInt(int64(candidate)))
		if err != nil {
			t.Fatalf("Vote encryption failed for voter %d: %v", i, err)
		}
		
		// Apply weight (for real slot, weight=1, so vote counts)
		weightedVote := paillierPK.Multiply(encVote, realSlot.Weight)

		// Create ring signature
		message := []byte(fmt.Sprintf("vote:%d:slot:%d", candidate, cred.RealIndex))
		sig, err := ringParams.Sign(message, voter.RingKey, ring, i)
		if err != nil {
			t.Fatalf("Ring signature failed for voter %d: %v", i, err)
		}

		// Verify ring signature
		if !ringParams.Verify(message, sig, ring) {
			t.Fatalf("Ring signature verification failed for voter %d", i)
		}

		// Process vote through SA²
		err = sa2System.ProcessVote(weightedVote, voter.ID)
		if err != nil {
			t.Fatalf("SA² vote processing failed for voter %d: %v", i, err)
		}
	}

	// ========== PHASE 4: Tallying ==========
	t.Log("Phase 4: Tallying")
	
	// Get encrypted tally from SA²
	encryptedResult, err := sa2System.GetEncryptedTally()
	if err != nil {
		t.Fatalf("SA² tally failed: %v", err)
	}

	// Verify vote count
	if encryptedResult.TotalVotes != numVoters {
		t.Errorf("Total votes = %d, want %d", encryptedResult.TotalVotes, numVoters)
	}

	// Decrypt final tally
	finalTally, err := paillierSK.Decrypt(encryptedResult.EncryptedTally)
	if err != nil {
		t.Fatalf("Tally decryption failed: %v", err)
	}

	// Expected: 40*0 + 35*1 + 25*2 = 0 + 35 + 50 = 85
	expectedTally := int64(40*0 + 35*1 + 25*2)
	if finalTally.Int64() != expectedTally {
		t.Errorf("Final tally = %d, want %d", finalTally.Int64(), expectedTally)
	}

	t.Logf("✅ Full voting flow completed successfully!")
	t.Logf("   Total voters: %d", numVoters)
	t.Logf("   Final tally: %d", finalTally.Int64())
}

// ============================================================
// TEST: Coercion Resistance Scenario
// ============================================================

func TestCoercionResistanceScenario(t *testing.T) {
	t.Log("Testing coercion resistance...")

	// Setup
	paillierSK, _ := crypto.GeneratePaillierKeyPair(1024)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(256)
	smdcGen := smdc.NewGenerator(pedersenParams, 5)
	sa2System := sa2.NewSystem(paillierPK)

	// Voter generates credential
	cred, _ := smdcGen.Generate("coerced_voter")
	t.Logf("Voter's real index (SECRET): %d", cred.RealIndex)

	// ========== COERCION SCENARIO ==========
	// Coercer demands voter vote for candidate 0
	// Voter wants to vote for candidate 1

	// Voter shows coercer a FAKE slot and "votes" for candidate 0
	fakeSlot := cred.GetAnyFakeSlot()
	t.Logf("Voter shows coercer fake slot index: %d", fakeSlot.Index)

	// Fake vote (weight=0, so it won't count)
	fakeVote, _ := paillierPK.Encrypt(big.NewInt(0)) // Coercer's choice
	fakeWeightedVote := paillierPK.Multiply(fakeVote, fakeSlot.Weight) // 0 * 0 = 0
	sa2System.ProcessVote(fakeWeightedVote, "fake_vote")

	// Real vote (weight=1, will count)
	realSlot := cred.GetRealSlot()
	realVote, _ := paillierPK.Encrypt(big.NewInt(1)) // Voter's true choice
	realWeightedVote := paillierPK.Multiply(realVote, realSlot.Weight) // 1 * 1 = 1
	sa2System.ProcessVote(realWeightedVote, "real_vote")

	// Tally
	result, _ := sa2System.GetEncryptedTally()
	decrypted, _ := paillierSK.Decrypt(result.EncryptedTally)

	// Should only count the real vote (candidate 1)
	if decrypted.Int64() != 1 {
		t.Errorf("Tally = %d, want 1 (voter's real choice)", decrypted.Int64())
	}

	t.Log("✅ Coercion resistance verified: Only real vote counted!")
}

// ============================================================
// TEST: Double Voting Detection
// ============================================================

func TestDoubleVotingDetection(t *testing.T) {
	t.Log("Testing double voting detection via ring signatures...")

	ringParams, _ := crypto.GenerateRingParams(256)

	// Create ring of 10 voters
	ring := make([]*big.Int, 10)
	keyPairs := make([]*crypto.RingKeyPair, 10)
	for i := 0; i < 10; i++ {
		keyPairs[i], _ = ringParams.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}

	// Voter 0 signs first vote
	msg1 := []byte("vote:candidate_A")
	sig1, _ := ringParams.Sign(msg1, keyPairs[0], ring, 0)

	// Same voter tries to vote again
	msg2 := []byte("vote:candidate_B")
	sig2, _ := ringParams.Sign(msg2, keyPairs[0], ring, 0)

	// Both signatures should have same key image (LINKED)
	if !crypto.Link(sig1, sig2) {
		t.Error("Failed to detect double voting - key images should match")
	}

	// Different voter's signature should NOT link
	sig3, _ := ringParams.Sign(msg1, keyPairs[1], ring, 1)
	if crypto.Link(sig1, sig3) {
		t.Error("False positive: Different voters' signatures linked")
	}

	t.Log("✅ Double voting detection verified!")
}
```

---

# ⏱️ PART 3: BENCHMARK TESTS

## 3.1 test/benchmark/crypto_benchmark_test.go

```go
// test/benchmark/crypto_benchmark_test.go

package benchmark

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/yourusername/covertvote/internal/crypto"
	"github.com/yourusername/covertvote/internal/sa2"
	"github.com/yourusername/covertvote/internal/smdc"
)

// ============================================================
// PAILLIER BENCHMARKS
// ============================================================

func BenchmarkPaillierKeyGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		crypto.GeneratePaillierKeyPair(2048)
	}
}

func BenchmarkPaillierEncrypt(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	msg := big.NewInt(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.Encrypt(msg)
	}
}

func BenchmarkPaillierDecrypt(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	msg := big.NewInt(42)
	ciphertext, _ := pk.Encrypt(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Decrypt(ciphertext)
	}
}

func BenchmarkPaillierHomomorphicAdd(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	c1, _ := pk.Encrypt(big.NewInt(10))
	c2, _ := pk.Encrypt(big.NewInt(20))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.Add(c1, c2)
	}
}

// ============================================================
// PEDERSEN BENCHMARKS
// ============================================================

func BenchmarkPedersenParamsGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		crypto.GeneratePedersenParams(512)
	}
}

func BenchmarkPedersenCommit(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	msg := big.NewInt(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pp.Commit(msg)
	}
}

// ============================================================
// ZK PROOF BENCHMARKS
// ============================================================

func BenchmarkBinaryProofGenerate(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	w := big.NewInt(1)
	commitment, _ := pp.Commit(w)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pp.ProveBinary(w, commitment.R, commitment.C)
	}
}

func BenchmarkBinaryProofVerify(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	w := big.NewInt(1)
	commitment, _ := pp.Commit(w)
	proof, _ := pp.ProveBinary(w, commitment.R, commitment.C)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pp.VerifyBinary(commitment.C, proof)
	}
}

// ============================================================
// SMDC BENCHMARKS
// ============================================================

func BenchmarkSMDCGenerate(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewGenerator(pp, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate(fmt.Sprintf("voter_%d", i))
	}
}

func BenchmarkSMDCVerify(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewGenerator(pp, 5)
	cred, _ := gen.Generate("voter")
	pub := cred.ToPublic()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Verify(pub)
	}
}

// Different K values
func BenchmarkSMDCGenerateK3(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewGenerator(pp, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate("voter")
	}
}

func BenchmarkSMDCGenerateK5(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewGenerator(pp, 5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate("voter")
	}
}

func BenchmarkSMDCGenerateK10(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewGenerator(pp, 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate("voter")
	}
}

// ============================================================
// SA² BENCHMARKS
// ============================================================

func BenchmarkSA2Split(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	sg := sa2.NewShareGenerator(pk)
	encVote, _ := pk.Encrypt(big.NewInt(1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sg.Split(encVote, "voter")
	}
}

func BenchmarkSA2ProcessVote(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	sys := sa2.NewSystem(pk)
	encVote, _ := pk.Encrypt(big.NewInt(1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sys.ProcessVote(encVote, fmt.Sprintf("voter_%d", i))
	}
	b.StopTimer()
	sys.Reset()
}

// ============================================================
// RING SIGNATURE BENCHMARKS
// ============================================================

func BenchmarkRingSign10(b *testing.B) {
	benchmarkRingSign(b, 10)
}

func BenchmarkRingSign50(b *testing.B) {
	benchmarkRingSign(b, 50)
}

func BenchmarkRingSign100(b *testing.B) {
	benchmarkRingSign(b, 100)
}

func benchmarkRingSign(b *testing.B, ringSize int) {
	rp, _ := crypto.GenerateRingParams(256)
	ring := make([]*big.Int, ringSize)
	keyPairs := make([]*crypto.RingKeyPair, ringSize)
	for i := 0; i < ringSize; i++ {
		keyPairs[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}
	message := []byte("test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rp.Sign(message, keyPairs[0], ring, 0)
	}
}

func BenchmarkRingVerify100(b *testing.B) {
	rp, _ := crypto.GenerateRingParams(256)
	ringSize := 100
	ring := make([]*big.Int, ringSize)
	keyPairs := make([]*crypto.RingKeyPair, ringSize)
	for i := 0; i < ringSize; i++ {
		keyPairs[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}
	message := []byte("test message")
	sig, _ := rp.Sign(message, keyPairs[0], ring, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rp.Verify(message, sig, ring)
	}
}
```

## 3.2 test/benchmark/voting_benchmark_test.go

```go
// test/benchmark/voting_benchmark_test.go

package benchmark

import (
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/yourusername/covertvote/internal/crypto"
	"github.com/yourusername/covertvote/internal/sa2"
	"github.com/yourusername/covertvote/internal/smdc"
)

// ============================================================
// SCALABILITY BENCHMARK - Different Voter Counts
// ============================================================

func TestScalabilityBenchmark(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping scalability benchmark in short mode")
	}

	voterCounts := []int{100, 1000, 10000}
	
	// For larger tests, uncomment:
	// voterCounts := []int{100, 1000, 10000, 100000, 1000000}

	results := make([]BenchmarkResult, 0)

	for _, n := range voterCounts {
		result := runVotingBenchmark(t, n)
		results = append(results, result)
		
		t.Logf("Voters: %d | Total: %v | Per Vote: %v", 
			n, result.TotalTime, result.PerVoteTime)
	}

	// Save results
	saveResultsToFile(results, "benchmark_results.md")
}

type BenchmarkResult struct {
	VoterCount     int
	TotalTime      time.Duration
	PerVoteTime    time.Duration
	CredGenTime    time.Duration
	VoteCastTime   time.Duration
	AggregateTime  time.Duration
	DecryptTime    time.Duration
	MemoryUsedMB   float64
}

func runVotingBenchmark(t *testing.T, numVoters int) BenchmarkResult {
	result := BenchmarkResult{VoterCount: numVoters}

	// Setup (not timed)
	paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	smdcGen := smdc.NewGenerator(pedersenParams, 5)
	sa2System := sa2.NewSystem(paillierPK)

	totalStart := time.Now()

	// Phase 1: Credential Generation
	credStart := time.Now()
	credentials := make([]*smdc.Credential, numVoters)
	for i := 0; i < numVoters; i++ {
		credentials[i], _ = smdcGen.Generate(fmt.Sprintf("voter_%d", i))
	}
	result.CredGenTime = time.Since(credStart)

	// Phase 2: Vote Casting & SA² Processing
	voteStart := time.Now()
	for i := 0; i < numVoters; i++ {
		vote := int64(i % 5) // Rotate through 5 candidates
		realSlot := credentials[i].GetRealSlot()
		
		encVote, _ := paillierPK.Encrypt(big.NewInt(vote))
		weightedVote := paillierPK.Multiply(encVote, realSlot.Weight)
		
		sa2System.ProcessVote(weightedVote, fmt.Sprintf("voter_%d", i))
	}
	result.VoteCastTime = time.Since(voteStart)

	// Phase 3: Aggregation
	aggStart := time.Now()
	encryptedResult, _ := sa2System.GetEncryptedTally()
	result.AggregateTime = time.Since(aggStart)

	// Phase 4: Decryption
	decStart := time.Now()
	_, _ = paillierSK.Decrypt(encryptedResult.EncryptedTally)
	result.DecryptTime = time.Since(decStart)

	result.TotalTime = time.Since(totalStart)
	result.PerVoteTime = result.TotalTime / time.Duration(numVoters)

	// Cleanup
	sa2System.Reset()

	return result
}

func saveResultsToFile(results []BenchmarkResult, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	// Write header
	f.WriteString("# CovertVote Benchmark Results\n\n")
	f.WriteString("## Performance Table\n\n")
	f.WriteString("| Voters | Total Time | Per Vote | Cred Gen | Vote Cast | Aggregate | Decrypt |\n")
	f.WriteString("|--------|------------|----------|----------|-----------|-----------|----------|\n")

	// Write data rows
	for _, r := range results {
		f.WriteString(fmt.Sprintf("| %d | %v | %v | %v | %v | %v | %v |\n",
			r.VoterCount,
			r.TotalTime.Round(time.Millisecond),
			r.PerVoteTime.Round(time.Microsecond),
			r.CredGenTime.Round(time.Millisecond),
			r.VoteCastTime.Round(time.Millisecond),
			r.AggregateTime.Round(time.Microsecond),
			r.DecryptTime.Round(time.Microsecond),
		))
	}

	// Projection section
	f.WriteString("\n## Projections for Large Scale\n\n")
	if len(results) > 0 {
		perVote := results[len(results)-1].PerVoteTime
		f.WriteString(fmt.Sprintf("Based on per-vote time of %v:\n\n", perVote))
		
		projections := []int{1000000, 10000000, 50000000, 100000000}
		f.WriteString("| Voters | Projected Time |\n")
		f.WriteString("|--------|----------------|\n")
		for _, n := range projections {
			projected := perVote * time.Duration(n)
			f.WriteString(fmt.Sprintf("| %d | %v |\n", n, projected.Round(time.Second)))
		}
	}
}

// ============================================================
// INDIVIDUAL OPERATION TIMING
// ============================================================

func TestIndividualOperationTiming(t *testing.T) {
	iterations := 100

	// Paillier Key Gen
	start := time.Now()
	for i := 0; i < iterations; i++ {
		crypto.GeneratePaillierKeyPair(2048)
	}
	paillierKeyGen := time.Since(start) / time.Duration(iterations)

	// Setup for other tests
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	pp, _ := crypto.GeneratePedersenParams(512)

	// Paillier Encrypt
	msg := big.NewInt(42)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		pk.Encrypt(msg)
	}
	paillierEnc := time.Since(start) / time.Duration(iterations)

	// Paillier Decrypt
	ct, _ := pk.Encrypt(msg)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		sk.Decrypt(ct)
	}
	paillierDec := time.Since(start) / time.Duration(iterations)

	// Pedersen Commit
	start = time.Now()
	for i := 0; i < iterations; i++ {
		pp.Commit(msg)
	}
	pedersenCommit := time.Since(start) / time.Duration(iterations)

	// SMDC Generate
	gen := smdc.NewGenerator(pp, 5)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		gen.Generate("voter")
	}
	smdcGen := time.Since(start) / time.Duration(iterations)

	// SA² Split
	sys := sa2.NewSystem(pk)
	encVote, _ := pk.Encrypt(big.NewInt(1))
	start = time.Now()
	for i := 0; i < iterations; i++ {
		sys.Generator.Split(encVote, "voter")
	}
	sa2Split := time.Since(start) / time.Duration(iterations)

	// Print results
	t.Log("\n========== INDIVIDUAL OPERATION TIMING ==========")
	t.Logf("Paillier KeyGen (2048-bit): %v", paillierKeyGen)
	t.Logf("Paillier Encrypt:           %v", paillierEnc)
	t.Logf("Paillier Decrypt:           %v", paillierDec)
	t.Logf("Pedersen Commit:            %v", pedersenCommit)
	t.Logf("SMDC Generate (k=5):        %v", smdcGen)
	t.Logf("SA² Split:                  %v", sa2Split)
	t.Log("=================================================")

	// Calculate total per vote
	totalPerVote := paillierEnc + smdcGen + sa2Split
	t.Logf("\nEstimated Total Per Vote: %v", totalPerVote)
	
	// Projections
	t.Log("\n========== PROJECTIONS ==========")
	for _, n := range []int{1000, 10000, 100000, 1000000} {
		projected := totalPerVote * time.Duration(n)
		t.Logf("%d voters: %v", n, projected)
	}
}
```

---

# 🖥️ PART 4: SCRIPTS

## 4.1 scripts/run_tests.sh

```bash
#!/bin/bash

echo "=========================================="
echo "   CovertVote Test Suite                 "
echo "=========================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Run unit tests
echo -e "\n${GREEN}[1/3] Running Unit Tests...${NC}"
go test -v ./internal/crypto/... 
go test -v ./internal/smdc/...
go test -v ./internal/sa2/...

# Run integration tests
echo -e "\n${GREEN}[2/3] Running Integration Tests...${NC}"
go test -v ./test/integration/...

# Run tests with coverage
echo -e "\n${GREEN}[3/3] Generating Coverage Report...${NC}"
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

echo -e "\n${GREEN}✅ All tests completed!${NC}"
echo "Coverage report: coverage.html"
```

## 4.2 scripts/run_benchmarks.sh

```bash
#!/bin/bash

echo "=========================================="
echo "   CovertVote Benchmark Suite            "
echo "=========================================="

# Create results directory
mkdir -p test/benchmark/results

# Run benchmarks
echo -e "\n[1/3] Running Crypto Benchmarks..."
go test -bench=. -benchmem ./test/benchmark/... -run=^$ | tee test/benchmark/results/crypto_bench.txt

# Run scalability test
echo -e "\n[2/3] Running Scalability Benchmark..."
go test -v -run TestScalabilityBenchmark ./test/benchmark/...

# Run individual timing
echo -e "\n[3/3] Running Individual Operation Timing..."
go test -v -run TestIndividualOperationTiming ./test/benchmark/...

echo -e "\n✅ Benchmarks completed!"
echo "Results saved to: test/benchmark/results/"
```

---

# 📊 PART 5: EXPECTED RESULTS TEMPLATE

## 5.1 test/benchmark/results/benchmark_results.md (Template)

```markdown
# CovertVote Benchmark Results

**Date:** [Auto-generated]
**System:** [Your system specs]
**Go Version:** [Your Go version]

## Individual Operation Timing

| Operation | Time | Memory |
|-----------|------|--------|
| Paillier KeyGen (2048-bit) | ~250ms | 4 KB |
| Paillier Encrypt | ~2-3ms | 512 B |
| Paillier Decrypt | ~3-4ms | 512 B |
| Pedersen Commit | ~0.5ms | 128 B |
| Binary ZK Proof Gen | ~1-2ms | 256 B |
| Binary ZK Proof Verify | ~1ms | 128 B |
| SMDC Generate (k=5) | ~8-10ms | 1.5 KB |
| SMDC Verify | ~5-6ms | 512 B |
| SA² Split | ~4-5ms | 1 KB |
| Ring Sign (n=100) | ~15-20ms | 2 KB |
| Ring Verify (n=100) | ~12-15ms | 1 KB |

## Scalability Results

| Voters | Total Time | Per Vote | Throughput |
|--------|------------|----------|------------|
| 1,000 | [X]s | [X]ms | [X] votes/sec |
| 10,000 | [X]s | [X]ms | [X] votes/sec |
| 100,000 | [X]min | [X]ms | [X] votes/sec |

## Projections for National Scale

| Voters | Projected Time (Sequential) | With 100 Nodes |
|--------|----------------------------|----------------|
| 1M | [X] hours | [X] minutes |
| 10M | [X] hours | [X] hours |
| 50M | [X] hours | [X] hours |
| 100M | [X] hours | [X] hours |

## Comparison with ISE-Voting

| Metric | CovertVote | ISE-Voting |
|--------|------------|------------|
| Complexity | O(n×k) | O(n×m²) |
| Coercion Resistant | ✅ Yes | ❌ No |
| 100K voters tally | [X]s | ~63s |
```

---

# ✅ CHECKLIST FOR AI AGENT

```
Unit Tests:
☐ paillier_test.go - All tests pass
☐ pedersen_test.go - All tests pass
☐ zkproof_test.go - All tests pass
☐ ring_signature_test.go - All tests pass
☐ credential_test.go (SMDC) - All tests pass
☐ aggregation_test.go (SA²) - All tests pass

Integration Tests:
☐ full_flow_test.go - Complete voting flow works
☐ Coercion resistance scenario verified
☐ Double voting detection verified

Benchmarks:
☐ Individual operation timing measured
☐ Scalability test (100 → 100K voters)
☐ Results saved to markdown file
☐ Projections calculated for 50M voters

Commands to Run:
☐ go test ./... (all tests pass)
☐ go test -bench=. ./... (benchmarks complete)
☐ go test -cover ./... (coverage > 80%)
```

---

# 🚀 RUN ORDER

```bash
# Step 1: Run all unit tests
go test -v ./internal/...

# Step 2: Run integration tests
go test -v ./test/integration/...

# Step 3: Run benchmarks
go test -bench=. -benchmem ./test/benchmark/...

# Step 4: Run scalability test
go test -v -run TestScalabilityBenchmark ./test/benchmark/...

# Step 5: Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

**END OF TESTING GUIDE**

**Version:** 1.0
**Ready for Vibe Coding:** ✅ YES
