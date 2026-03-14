# TASK 1: Threshold Paillier Implementation
# Copy this ENTIRE prompt into your IDE (Cursor/Copilot)

```
Implement Threshold Paillier Decryption for CovertVote e-voting system.

## Context
Currently `internal/tally/decrypt.go` uses a SINGLE Paillier private key for decryption. This is a single point of failure — one entity can decrypt all votes alone. We need threshold decryption where the private key is split into shares, and at least t-of-n trustees must cooperate to decrypt.

## What to implement

### 1. Create `internal/crypto/threshold_paillier.go` (NEW FILE)

Implement threshold Paillier following Damgård-Jurik-Nielsen (2010) scheme:

```go
package crypto

// ThresholdParams holds the threshold scheme parameters
type ThresholdParams struct {
    N          int // total number of trustees (e.g., 5)
    Threshold  int // minimum trustees needed (e.g., 3)
}

// ThresholdKeyShares holds the result of distributed key generation
type ThresholdKeyShares struct {
    PublicKey   *PaillierPublicKey
    Shares      []*KeyShare        // one per trustee
    VerifyKeys  []*big.Int         // verification keys for each share
    Params      *ThresholdParams
    V           *big.Int           // verification base
    Delta       *big.Int           // n! (factorial of n)
}

// KeyShare represents one trustee's share of the private key
type KeyShare struct {
    Index    int       // trustee index (1-based)
    Si       *big.Int  // share value
}

// PartialDecryption represents one trustee's partial decryption
type PartialDecryption struct {
    Index    int       // trustee index
    Ci       *big.Int  // partial decryption value: c^(2 * delta * si) mod n^2
    Proof    *PartialDecryptionProof  // ZK proof of correct partial decryption
}

// PartialDecryptionProof proves correct partial decryption without revealing the share
type PartialDecryptionProof struct {
    E    *big.Int  // challenge
    Z    *big.Int  // response
}
```

Key functions to implement:

```go
// GenerateThresholdKey generates a threshold Paillier key pair.
// bits: key size (2048), n: total trustees, t: threshold
// Returns shares for each trustee and the public key.
func GenerateThresholdKey(bits, n, t int) (*ThresholdKeyShares, error)

// PartialDecrypt performs one trustee's partial decryption.
// The trustee computes ci = c^(2 * delta * si) mod n^2
// and generates a ZK proof of correct computation.
func (share *KeyShare) PartialDecrypt(c *big.Int, pk *PaillierPublicKey, params *ThresholdParams, vk *big.Int, v *big.Int) (*PartialDecryption, error)

// CombinePartialDecryptions combines t partial decryptions to recover plaintext.
// Uses Lagrange interpolation in the exponent.
// partials: at least t valid partial decryptions
func CombinePartialDecryptions(partials []*PartialDecryption, pk *PaillierPublicKey, params *ThresholdParams) (*big.Int, error)

// VerifyPartialDecryption verifies a trustee's ZK proof of correct partial decryption.
func VerifyPartialDecryption(pd *PartialDecryption, c *big.Int, pk *PaillierPublicKey, vk *big.Int, v *big.Int) bool
```

Implementation details:

a) **GenerateThresholdKey:**
   - Generate safe primes p, q where p = 2p' + 1, q = 2q' + 1
   - Compute n = p*q, m = p'*q'
   - Compute d such that d = 0 mod m and d = 1 mod n (CRT)
   - Split d into shares using Shamir's Secret Sharing over Z_m
   - delta = n! (factorial of total trustees count)
   - Generate verification keys: v = random square mod n^2, vk_i = v^(delta * si) mod n^2

b) **PartialDecrypt:**
   - Compute ci = c^(2 * delta * si) mod n^2
   - Generate ZK proof: choose random r, compute a = c^(4 * delta * r), b = v^r
   - Challenge e = SHA-256(c, ci, a, b, ...)
   - Response z = r + e * si (no modular reduction needed for the proof)

c) **CombinePartialDecryptions:**
   - Compute Lagrange coefficients lambda_i for the subset S of trustees
   - Compute c' = product of ci^(2 * lambda_i) mod n^2 for i in S
   - Recover plaintext: m = L(c') * (4 * delta^2)^(-1) mod n
   - where L(x) = (x - 1) / n

### 2. Create `internal/crypto/threshold_paillier_test.go` (NEW FILE)

```go
func TestThresholdKeyGeneration(t *testing.T) {
    // Generate 3-of-5 threshold key with 2048-bit modulus
    shares, err := GenerateThresholdKey(2048, 5, 3)
    if err != nil {
        t.Fatal(err)
    }
    // Verify we got 5 shares
    if len(shares.Shares) != 5 {
        t.Fatalf("expected 5 shares, got %d", len(shares.Shares))
    }
}

func TestThresholdEncryptDecrypt(t *testing.T) {
    shares, _ := GenerateThresholdKey(2048, 5, 3)
    pk := shares.PublicKey
    
    // Encrypt a vote
    vote := big.NewInt(42)
    ciphertext, _ := pk.Encrypt(vote)
    
    // Get partial decryptions from trustees 1, 3, 5 (any 3 of 5)
    partials := make([]*PartialDecryption, 3)
    indices := []int{0, 2, 4} // trustees 1, 3, 5
    for i, idx := range indices {
        pd, err := shares.Shares[idx].PartialDecrypt(
            ciphertext, pk, shares.Params, shares.VerifyKeys[idx], shares.V)
        if err != nil {
            t.Fatal(err)
        }
        partials[i] = pd
    }
    
    // Combine and verify
    result, err := CombinePartialDecryptions(partials, pk, shares.Params)
    if err != nil {
        t.Fatal(err)
    }
    if result.Cmp(vote) != 0 {
        t.Errorf("expected %d, got %d", vote.Int64(), result.Int64())
    }
}

func TestThresholdHomomorphicTally(t *testing.T) {
    shares, _ := GenerateThresholdKey(2048, 5, 3)
    pk := shares.PublicKey
    
    // Encrypt 10 votes
    votes := []int64{1, 0, 1, 1, 0, 1, 0, 1, 1, 0} // expected sum = 6
    var ciphertexts []*big.Int
    for _, v := range votes {
        enc, _ := pk.Encrypt(big.NewInt(v))
        ciphertexts = append(ciphertexts, enc)
    }
    
    // Homomorphic tally
    tally := pk.AddMultiple(ciphertexts)
    
    // Threshold decrypt with trustees 2, 3, 4
    partials := make([]*PartialDecryption, 3)
    for i := 0; i < 3; i++ {
        pd, _ := shares.Shares[i+1].PartialDecrypt(
            tally, pk, shares.Params, shares.VerifyKeys[i+1], shares.V)
        partials[i] = pd
    }
    
    result, _ := CombinePartialDecryptions(partials, pk, shares.Params)
    if result.Int64() != 6 {
        t.Errorf("expected tally 6, got %d", result.Int64())
    }
}

func TestThresholdInsufficientShares(t *testing.T) {
    shares, _ := GenerateThresholdKey(2048, 5, 3)
    pk := shares.PublicKey
    
    ciphertext, _ := pk.Encrypt(big.NewInt(1))
    
    // Only 2 partial decryptions (need 3)
    partials := make([]*PartialDecryption, 2)
    for i := 0; i < 2; i++ {
        pd, _ := shares.Shares[i].PartialDecrypt(
            ciphertext, pk, shares.Params, shares.VerifyKeys[i], shares.V)
        partials[i] = pd
    }
    
    _, err := CombinePartialDecryptions(partials, pk, shares.Params)
    if err == nil {
        t.Fatal("expected error with insufficient shares")
    }
}

func TestThresholdPartialDecryptionVerification(t *testing.T) {
    shares, _ := GenerateThresholdKey(2048, 5, 3)
    pk := shares.PublicKey
    
    ciphertext, _ := pk.Encrypt(big.NewInt(7))
    
    // Get a valid partial decryption
    pd, _ := shares.Shares[0].PartialDecrypt(
        ciphertext, pk, shares.Params, shares.VerifyKeys[0], shares.V)
    
    // Verify the ZK proof
    valid := VerifyPartialDecryption(pd, ciphertext, pk, shares.VerifyKeys[0], shares.V)
    if !valid {
        t.Fatal("valid partial decryption should verify")
    }
    
    // Tamper with the partial decryption
    pd.Ci.Add(pd.Ci, big.NewInt(1))
    invalid := VerifyPartialDecryption(pd, ciphertext, pk, shares.VerifyKeys[0], shares.V)
    if invalid {
        t.Fatal("tampered partial decryption should NOT verify")
    }
}
```

### 3. Update `internal/tally/decrypt.go`

Add a new function that uses threshold decryption alongside the existing single-key decryption:

```go
// ThresholdTally performs homomorphic tallying with threshold decryption.
// This is the secure production method — no single entity can decrypt alone.
func ThresholdTally(
    encryptedVotes []*big.Int,
    pk *crypto.PaillierPublicKey,
    partialDecryptions [][]*crypto.PartialDecryption,
    params *crypto.ThresholdParams,
) (*big.Int, error) {
    // Step 1: Homomorphic addition of all encrypted votes
    tally := pk.AddMultiple(encryptedVotes)
    
    // Step 2: Combine partial decryptions
    // We expect len(partialDecryptions) >= params.Threshold sets
    if len(partialDecryptions) < params.Threshold {
        return nil, fmt.Errorf("need at least %d partial decryptions, got %d",
            params.Threshold, len(partialDecryptions))
    }
    
    result, err := crypto.CombinePartialDecryptions(
        partialDecryptions[0], // first threshold-many partial decryptions
        pk, params)
    if err != nil {
        return nil, fmt.Errorf("threshold decryption failed: %w", err)
    }
    
    return result, nil
}
```

Keep the existing single-key Decrypt functions for backward compatibility and testing.

### 4. Add Benchmark

In `test/benchmark/crypto_benchmark_test.go`, add:

```go
func BenchmarkThresholdKeyGen(b *testing.B) {
    for i := 0; i < b.N; i++ {
        crypto.GenerateThresholdKey(2048, 5, 3)
    }
}

func BenchmarkThresholdPartialDecrypt(b *testing.B) {
    shares, _ := crypto.GenerateThresholdKey(2048, 5, 3)
    pk := shares.PublicKey
    ct, _ := pk.Encrypt(big.NewInt(42))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        shares.Shares[0].PartialDecrypt(ct, pk, shares.Params, shares.VerifyKeys[0], shares.V)
    }
}

func BenchmarkThresholdCombine3of5(b *testing.B) {
    shares, _ := crypto.GenerateThresholdKey(2048, 5, 3)
    pk := shares.PublicKey
    ct, _ := pk.Encrypt(big.NewInt(42))
    partials := make([]*crypto.PartialDecryption, 3)
    for i := 0; i < 3; i++ {
        partials[i], _ = shares.Shares[i].PartialDecrypt(
            ct, pk, shares.Params, shares.VerifyKeys[i], shares.V)
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        crypto.CombinePartialDecryptions(partials, pk, shares.Params)
    }
}
```

### Important Notes:
- Use `crypto/rand` for ALL random number generation (never math/rand)
- Use `crypto/sha256` or `golang.org/x/crypto/sha3` for hashing in ZK proofs
- All big.Int operations must use proper modular arithmetic
- Do NOT use any external threshold Paillier library — implement from scratch following the DJN scheme so we have full control
- Make sure existing tests still pass: `go test ./internal/crypto/... -v`
- Make sure new tests pass: `go test ./internal/crypto/... -run Threshold -v`
- Run benchmarks: `go test -bench=BenchmarkThreshold -benchmem ./test/benchmark/...`
```
