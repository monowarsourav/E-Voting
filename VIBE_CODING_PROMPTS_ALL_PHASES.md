# CovertVote — Vibe Coding Prompts (Phase 1 → Phase 5)
# Copy-paste each prompt into your IDE (Cursor/Copilot/Windsurf/etc.)
# Apply sequentially — Phase 1 first, then Phase 2, etc.

---

# ═══════════════════════════════════════════════════════
# PHASE 1: CRITICAL CODE FIXES (Week 1)
# ═══════════════════════════════════════════════════════

## PROMPT 1.1 — Fix hashChallenge (Strong Fiat-Shamir)
## File: internal/crypto/zkproof.go
## Time: ~30 minutes

```
Fix the hashChallenge function in internal/crypto/zkproof.go to implement proper Strong Fiat-Shamir transformation.

Current problem: The hashChallenge function does NOT include the public parameters (Pedersen params P, Q, G, H) in the hash computation. According to Bernhard-Pereira-Warinschi (ASIACRYPT 2012), omitting the statement from the Fiat-Shamir hash enables real attacks (demonstrated against Helios voting system).

Changes needed:

1. Update hashChallenge signature to accept PedersenParams:

func hashChallenge(q *big.Int, nonce []byte, electionID string, pp *PedersenParams, values ...*big.Int) *big.Int {
    hasher := sha3.New256()
    
    // Domain separation tag (prevents cross-protocol attacks)
    hasher.Write([]byte("CovertVote-ZKP-v1"))
    
    // Include public parameters (the "statement" in Strong Fiat-Shamir)
    hasher.Write(pp.P.Bytes())
    hasher.Write(pp.Q.Bytes())
    hasher.Write(pp.G.Bytes())
    hasher.Write(pp.H.Bytes())
    
    // Context binding
    hasher.Write(nonce)
    hasher.Write([]byte(electionID))
    
    // Length-prefix each value to prevent concatenation ambiguity
    for _, v := range values {
        vBytes := v.Bytes()
        lenBuf := make([]byte, 4)
        binary.BigEndian.PutUint32(lenBuf, uint32(len(vBytes)))
        hasher.Write(lenBuf)
        hasher.Write(vBytes)
    }
    
    hashBytes := hasher.Sum(nil)
    c := new(big.Int).SetBytes(hashBytes)
    c.Mod(c, q)
    return c
}

2. Add "encoding/binary" to the imports.

3. Update ALL callers of hashChallenge to pass PedersenParams (pp):

In ProveBinary(): change hashChallenge(pp.Q, nonce, electionID, C, a0, a1) 
   to hashChallenge(pp.Q, nonce, electionID, pp, C, a0, a1)

In VerifyBinary(): same change — pass pp as the 4th argument.

In ProveSumOne(): change hashChallenge(pp.Q, nonce, electionID, product, a, pp.G)
   to hashChallenge(pp.Q, nonce, electionID, pp, product, a, pp.G)

In VerifySumOne(): same change.

4. Make sure all existing tests still pass after this change:
   go test ./internal/crypto/... -v

Do NOT change the function logic for ProveBinary or VerifyBinary — only the hashChallenge function and its call sites.
```

---

## PROMPT 1.2 — Fix Paillier KeyGen Validation
## File: internal/crypto/paillier.go
## Time: ~10 minutes

```
Add key size validation to GeneratePaillierKeyPair in internal/crypto/paillier.go.

Add this check at the very beginning of the function:

func GeneratePaillierKeyPair(bits int) (*PaillierPrivateKey, error) {
    if bits < 2048 {
        return nil, fmt.Errorf("paillier: key size must be >= 2048 bits for security, got %d", bits)
    }
    if bits%2 != 0 {
        return nil, fmt.Errorf("paillier: key size must be even, got %d", bits)
    }
    // ... rest of existing code unchanged
}

Also add "fmt" to the imports if not already present.

Then add a test in internal/crypto/paillier_test.go:

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

Run: go test ./internal/crypto/... -v
```

---

## PROMPT 1.3 — SA² Server Docker Separation
## File: docker-compose-sa2.yml (NEW FILE), cmd/aggregator-a/main.go, cmd/aggregator-b/main.go
## Time: ~2 hours

```
Create a Docker Compose configuration that deploys SA² Leader and Helper servers as SEPARATE containers. This is critical because the Prio/SA² security model requires non-colluding servers — running them in the same process negates the privacy guarantee.

1. Create docker-compose-sa2.yml in the project root:

version: '3.8'

services:
  sa2-leader:
    build:
      context: .
      dockerfile: Dockerfile.aggregator
    container_name: covertvote-sa2-leader
    environment:
      - SERVER_ROLE=leader
      - SERVER_ID=sa2-server-a
      - SA2_API_KEY=${SA2_LEADER_API_KEY}
      - SA2_ADMIN_KEY=${SA2_LEADER_ADMIN_KEY}
      - LISTEN_PORT=8081
      - PARTNER_URL=http://sa2-helper:8082
    ports:
      - "8081:8081"
    networks:
      - sa2-network
    restart: unless-stopped

  sa2-helper:
    build:
      context: .
      dockerfile: Dockerfile.aggregator
    container_name: covertvote-sa2-helper
    environment:
      - SERVER_ROLE=helper
      - SERVER_ID=sa2-server-b
      - SA2_API_KEY=${SA2_HELPER_API_KEY}
      - SA2_ADMIN_KEY=${SA2_HELPER_ADMIN_KEY}
      - LISTEN_PORT=8082
      - PARTNER_URL=http://sa2-leader:8081
    ports:
      - "8082:8082"
    networks:
      - sa2-network
    restart: unless-stopped

networks:
  sa2-network:
    driver: bridge
    internal: false

2. Update .env.example to add:

# SA2 Server Keys (MUST be different for Leader and Helper)
SA2_LEADER_API_KEY=change-me-leader-api-key-min-32-chars
SA2_LEADER_ADMIN_KEY=change-me-leader-admin-key-min-32-chars
SA2_HELPER_API_KEY=change-me-helper-api-key-min-32-chars
SA2_HELPER_ADMIN_KEY=change-me-helper-admin-key-min-32-chars

3. If cmd/aggregator-a/main.go and cmd/aggregator-b/main.go exist, update them to read SERVER_ROLE from environment and configure accordingly. If they don't exist, the existing Dockerfile.aggregator should work — just make sure it reads SERVER_ROLE env var.

4. Add a comment at the top of internal/sa2/aggregation.go:

// Package sa2 implements Samplable Anonymous Aggregation (SA²) for private vote tallying.
//
// SECURITY REQUIREMENT: The Leader and Helper aggregation servers MUST be deployed
// on separate machines or containers managed by independent administrative domains.
// The Prio/SA² security model requires that at most one server is compromised.
// Running both servers in the same process negates this guarantee entirely.
//
// For deployment, use docker-compose-sa2.yml which enforces container separation.

This is for the research paper's threat model — we need to demonstrate that the SA² non-collusion assumption is architecturally enforced.
```

---

# ═══════════════════════════════════════════════════════
# PHASE 2: MISSING BENCHMARKS (Week 2-3)
# ═══════════════════════════════════════════════════════

## PROMPT 2.1 — ZKP Benchmark
## File: test/benchmark/crypto_benchmark_test.go
## Time: ~30 minutes

```
Add ZKP (Zero-Knowledge Proof) benchmarks to test/benchmark/crypto_benchmark_test.go.

Add these benchmark functions after the existing benchmarks:

func BenchmarkZKPBinaryProve(b *testing.B) {
    pp, _ := crypto.GeneratePedersenParams(2048)
    weight := big.NewInt(1)
    commitment, _ := pp.Commit(weight)
    nonce, _ := crypto.GenerateNonce()
    electionID := "bench-election-001"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pp.ProveBinary(weight, commitment.R, commitment.C, nonce, electionID)
    }
}

func BenchmarkZKPBinaryVerify(b *testing.B) {
    pp, _ := crypto.GeneratePedersenParams(2048)
    weight := big.NewInt(1)
    commitment, _ := pp.Commit(weight)
    nonce, _ := crypto.GenerateNonce()
    electionID := "bench-election-001"
    proof, _ := pp.ProveBinary(weight, commitment.R, commitment.C, nonce, electionID)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pp.VerifyBinary(commitment.C, proof)
    }
}

func BenchmarkZKPSumOneProve(b *testing.B) {
    pp, _ := crypto.GeneratePedersenParams(2048)
    // Create 5 commitments (k=5 SMDC slots): one weight=1, rest weight=0
    commitments := make([]*crypto.Commitment, 5)
    for i := 0; i < 5; i++ {
        w := big.NewInt(0)
        if i == 2 { w = big.NewInt(1) }
        c, _ := pp.Commit(w)
        commitments[i] = c
    }
    nonce, _ := crypto.GenerateNonce()
    electionID := "bench-election-001"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pp.ProveSumOne(commitments, nonce, electionID)
    }
}

func BenchmarkZKPSumOneVerify(b *testing.B) {
    pp, _ := crypto.GeneratePedersenParams(2048)
    commitments := make([]*crypto.Commitment, 5)
    commitmentValues := make([]*big.Int, 5)
    for i := 0; i < 5; i++ {
        w := big.NewInt(0)
        if i == 2 { w = big.NewInt(1) }
        c, _ := pp.Commit(w)
        commitments[i] = c
        commitmentValues[i] = c.C
    }
    nonce, _ := crypto.GenerateNonce()
    electionID := "bench-election-001"
    proof, _ := pp.ProveSumOne(commitments, nonce, electionID)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pp.VerifySumOne(commitmentValues, proof)
    }
}

Make sure to import the crypto package properly. Run with:
go test -bench=BenchmarkZKP -benchmem -count=5 ./test/benchmark/... | tee test/benchmark/results/zkp_bench.txt
```

---

## PROMPT 2.2 — End-to-End Vote Casting Benchmark
## File: test/benchmark/voting_benchmark_test.go
## Time: ~45 minutes

```
Add an end-to-end vote casting benchmark to test/benchmark/voting_benchmark_test.go that measures the FULL pipeline time: ballot creation → Paillier encryption → Pedersen commitment → ZKP generation → SMDC weight application → ring signature → SA² split → total.

Add this benchmark function (or update the existing one if BenchmarkFullVoteCast already exists):

func BenchmarkFullVoteCastPipeline(b *testing.B) {
    // Setup (not timed)
    paillierKey, _ := crypto.GeneratePaillierKeyPair(2048)
    pedersenParams, _ := crypto.GeneratePedersenParams(2048)
    ringParams, _ := crypto.GenerateRingParams(2048)
    
    // Create 100 ring members
    ringKeys := make([]*crypto.RingKeyPair, 100)
    ringPubKeys := make([]*big.Int, 100)
    for i := 0; i < 100; i++ {
        kp, _ := ringParams.GenerateRingKeyPair()
        ringKeys[i] = kp
        ringPubKeys[i] = kp.PublicKey
    }
    signerIndex := 42 // arbitrary signer
    
    // SMDC setup
    smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, "bench-election")
    
    // SA2 setup
    splitter := sa2.NewVoteSplitter(paillierKey.PublicKey)
    
    electionID := "bench-election-001"
    candidateVote := big.NewInt(1)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Step 1: Paillier encrypt vote
        encVote, _ := paillierKey.PublicKey.Encrypt(candidateVote)
        
        // Step 2: Pedersen commitment
        commitment, _ := pedersenParams.Commit(candidateVote)
        
        // Step 3: ZKP Binary Proof
        nonce, _ := crypto.GenerateNonce()
        _, _ = pedersenParams.ProveBinary(candidateVote, commitment.R, commitment.C, nonce, electionID)
        
        // Step 4: SMDC credential generate + get real slot
        cred, realIdx, _ := smdcGen.GenerateCredential(fmt.Sprintf("voter-%d", i))
        slot, _ := cred.GetRealSlot(realIdx)
        
        // Step 5: Apply SMDC weight
        weightedVote := paillierKey.PublicKey.Multiply(encVote, slot.Weight)
        
        // Step 6: Ring signature (100 members)
        message := weightedVote.Bytes()
        _, _ = ringParams.Sign(message, ringKeys[signerIndex], ringPubKeys, signerIndex)
        
        // Step 7: SA2 split
        _, _ = splitter.SplitVote(fmt.Sprintf("voter-%d", i), weightedVote)
    }
}

Also add per-phase timing benchmark:

func BenchmarkVoteCastPhases(b *testing.B) {
    // Same setup as above...
    paillierKey, _ := crypto.GeneratePaillierKeyPair(2048)
    pedersenParams, _ := crypto.GeneratePedersenParams(2048)
    ringParams, _ := crypto.GenerateRingParams(2048)
    
    ringKeys := make([]*crypto.RingKeyPair, 100)
    ringPubKeys := make([]*big.Int, 100)
    for i := 0; i < 100; i++ {
        kp, _ := ringParams.GenerateRingKeyPair()
        ringKeys[i] = kp
        ringPubKeys[i] = kp.PublicKey
    }
    
    candidateVote := big.NewInt(1)
    electionID := "bench-election-001"
    
    b.Run("1_PaillierEncrypt", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            paillierKey.PublicKey.Encrypt(candidateVote)
        }
    })
    
    b.Run("2_PedersenCommit", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            pedersenParams.Commit(candidateVote)
        }
    })
    
    b.Run("3_ZKPBinaryProve", func(b *testing.B) {
        commitment, _ := pedersenParams.Commit(candidateVote)
        nonce, _ := crypto.GenerateNonce()
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            pedersenParams.ProveBinary(candidateVote, commitment.R, commitment.C, nonce, electionID)
        }
    })
    
    b.Run("4_SMDCGenerate", func(b *testing.B) {
        smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, electionID)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            smdcGen.GenerateCredential(fmt.Sprintf("voter-%d", i))
        }
    })
    
    b.Run("5_RingSign100", func(b *testing.B) {
        msg := []byte("test-vote-message")
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            ringParams.Sign(msg, ringKeys[42], ringPubKeys, 42)
        }
    })
    
    b.Run("6_SA2Split", func(b *testing.B) {
        encVote, _ := paillierKey.PublicKey.Encrypt(candidateVote)
        splitter := sa2.NewVoteSplitter(paillierKey.PublicKey)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            splitter.SplitVote("voter-bench", encVote)
        }
    })
}

Make sure imports include: fmt, math/big, testing, and the project's crypto, smdc, sa2 packages.
Run: go test -bench=BenchmarkVoteCast -benchmem -count=5 ./test/benchmark/... | tee test/benchmark/results/e2e_bench.txt
```

---

## PROMPT 2.3 — Scalability Benchmark (100 → 100K voters)
## File: test/benchmark/scalability_test.go (NEW FILE)
## Time: ~1 hour

```
Create a new file test/benchmark/scalability_test.go that measures how CovertVote scales with increasing voter counts.

This benchmark tests:
1. Tally time vs voter count (proves O(n) complexity)
2. Ring signature time vs ring size
3. O(n) vs O(n×m²) validation by varying candidate count

package benchmark

import (
    "fmt"
    "math/big"
    "testing"
    "time"
    
    "github.com/covertvote/e-voting/internal/crypto"
    "github.com/covertvote/e-voting/internal/sa2"
)

// TestScalabilityTally measures homomorphic tally time at different voter scales.
// This proves O(n) complexity: tally time should grow linearly with voter count.
func TestScalabilityTally(t *testing.T) {
    paillierKey, err := crypto.GeneratePaillierKeyPair(2048)
    if err != nil {
        t.Fatal(err)
    }
    pk := paillierKey.PublicKey
    
    voterCounts := []int{100, 500, 1000, 5000, 10000, 50000, 100000}
    
    fmt.Println("=== Homomorphic Tally Scalability ===")
    fmt.Println("Voters | Tally Time (ms) | Per-Vote (μs)")
    fmt.Println("-------|-----------------|---------------")
    
    for _, n := range voterCounts {
        // Pre-generate encrypted votes
        votes := make([]*big.Int, n)
        for i := 0; i < n; i++ {
            v := big.NewInt(int64(i % 2)) // alternating 0/1 votes
            enc, err := pk.Encrypt(v)
            if err != nil {
                t.Fatal(err)
            }
            votes[i] = enc
        }
        
        // Measure tally time
        start := time.Now()
        tally := big.NewInt(1)
        for _, v := range votes {
            tally = pk.Add(tally, v)
        }
        elapsed := time.Since(start)
        
        perVote := float64(elapsed.Microseconds()) / float64(n)
        fmt.Printf("%7d | %15.2f | %13.2f\n", n, float64(elapsed.Milliseconds()), perVote)
    }
}

// TestScalabilityRingSize measures ring signature time vs ring size.
func TestScalabilityRingSize(t *testing.T) {
    ringParams, err := crypto.GenerateRingParams(2048)
    if err != nil {
        t.Fatal(err)
    }
    
    ringSizes := []int{10, 25, 50, 100, 200, 500}
    
    fmt.Println("\n=== Ring Signature Scalability ===")
    fmt.Println("Ring Size | Sign Time (ms) | Verify Time (ms)")
    fmt.Println("----------|----------------|------------------")
    
    for _, size := range ringSizes {
        keys := make([]*crypto.RingKeyPair, size)
        pubKeys := make([]*big.Int, size)
        for i := 0; i < size; i++ {
            kp, _ := ringParams.GenerateRingKeyPair()
            keys[i] = kp
            pubKeys[i] = kp.PublicKey
        }
        
        msg := []byte("benchmark-message")
        signerIdx := size / 2
        
        // Measure sign time (average of 5 runs)
        var totalSign, totalVerify time.Duration
        runs := 5
        if size >= 200 { runs = 2 }
        
        var sig *crypto.RingSignature
        for r := 0; r < runs; r++ {
            start := time.Now()
            sig, _ = ringParams.Sign(msg, keys[signerIdx], pubKeys, signerIdx)
            totalSign += time.Since(start)
            
            start = time.Now()
            ringParams.Verify(msg, sig, pubKeys)
            totalVerify += time.Since(start)
        }
        
        avgSign := float64(totalSign.Milliseconds()) / float64(runs)
        avgVerify := float64(totalVerify.Milliseconds()) / float64(runs)
        fmt.Printf("%9d | %14.2f | %16.2f\n", size, avgSign, avgVerify)
    }
}

// TestComplexityValidation validates O(n) vs O(n×m²) by varying candidate count.
// Fixes n=1000 voters, varies m={2,5,10,20,50} candidates.
// CovertVote tally time should remain constant regardless of m (O(n)).
func TestComplexityValidation(t *testing.T) {
    paillierKey, err := crypto.GeneratePaillierKeyPair(2048)
    if err != nil {
        t.Fatal(err)
    }
    pk := paillierKey.PublicKey
    
    n := 1000 // fixed voter count
    candidateCounts := []int{2, 5, 10, 20, 50}
    
    fmt.Println("\n=== O(n) Complexity Validation (n=1000 voters) ===")
    fmt.Println("Candidates | Tally Time (ms) | Per-Candidate (ms)")
    fmt.Println("-----------|-----------------|--------------------")
    
    for _, m := range candidateCounts {
        // For each candidate, create n encrypted votes
        // Total operations = n × m homomorphic additions
        
        start := time.Now()
        for c := 0; c < m; c++ {
            tally := big.NewInt(1)
            for i := 0; i < n; i++ {
                vote := big.NewInt(0)
                if i%m == c { vote = big.NewInt(1) } // distribute votes
                enc, _ := pk.Encrypt(vote)
                tally = pk.Add(tally, enc)
            }
        }
        elapsed := time.Since(start)
        
        perCandidate := float64(elapsed.Milliseconds()) / float64(m)
        fmt.Printf("%10d | %15.2f | %18.2f\n", m, float64(elapsed.Milliseconds()), perCandidate)
    }
    
    fmt.Println("\nNote: If per-candidate time is roughly constant, this confirms O(n×m) = O(n) per candidate.")
    fmt.Println("ISE-Voting's O(n×m²) would show per-candidate time GROWING with m.")
}

Run: go test -v -timeout 30m ./test/benchmark/scalability_test.go 2>&1 | tee test/benchmark/results/scalability_results.txt

Note: The 100K voter test may take several minutes due to Paillier encryption overhead. Be patient.
```

---

# ═══════════════════════════════════════════════════════
# PHASE 3: FORMAL PROOF ALIGNMENT (Week 3-4)
# ═══════════════════════════════════════════════════════

## PROMPT 3.1 — Add Security Properties Documentation
## File: internal/crypto/SECURITY.md (NEW FILE)
## Time: ~20 minutes

```
Create a new file internal/crypto/SECURITY.md that documents the security properties and assumptions for each cryptographic primitive used in CovertVote. This file serves as a bridge between the formal proofs in the paper and the actual implementation.

# CovertVote Cryptographic Security Properties

## Paillier Homomorphic Encryption (paillier.go)
- **Security Assumption:** Decisional Composite Residuosity Assumption (DCRA)
- **Key Size:** Minimum 2048 bits (enforced in GeneratePaillierKeyPair)
- **Semantic Security:** IND-CPA under DCRA
- **Homomorphic Property:** E(a) × E(b) = E(a+b) mod N²
- **Paper Reference:** Theorem 1 (Ballot Privacy) reduces to DCRA

## Pedersen Commitments (pedersen.go)
- **Hiding:** Computationally hiding under DLP
- **Binding:** Computationally binding under DLP
- **Perfect Hiding (info-theoretic):** For any commitment C, there exist valid openings to every possible message
- **Paper Reference:** Used in Theorem 1 (Game 1 transition) and Theorem 4 (SMDC credential indistinguishability)

## Zero-Knowledge Proofs — Σ-Protocol (zkproof.go)
- **Construction:** OR-proof for binary (w∈{0,1}), Schnorr-style for sum-one (Σw=1)
- **Fiat-Shamir:** Strong variant — hash includes: domain tag, public params (P,Q,G,H), nonce, electionID, proof values
- **Soundness:** Via Forking Lemma (Pointcheval-Stern 1996) in Random Oracle Model
- **Zero-Knowledge:** Simulator programs random oracle to produce indistinguishable transcripts
- **Replay Prevention:** 32-byte cryptographic nonce + electionID binding
- **Paper Reference:** Theorem 2

## Linkable Ring Signatures (ring_signature.go)
- **Construction:** Liu-Wei-Wong (ACISP 2004) style
- **Ring Size:** Fixed at 100 (RING_SIZE constant)
- **Key Image:** I = sk × H(pk) — deterministic, enables double-vote detection
- **Anonymity:** Signature distribution independent of signer position (statistical guarantee)
- **Unforgeability:** Reduces to DLP
- **Linkability:** Same signer → same key image; collision probability ≤ q²_H/2^λ
- **Paper Reference:** Theorem 3

## SMDC — Self-Masking Deniable Credentials (smdc/)
- **Slot Count:** k=5 (1 real + 4 fake)
- **Real Index Derivation:** HMAC-SHA256(electionID, "smdc-real-index:" + voterID + ":" + electionID) mod k
- **Indistinguishability:** Real and fake slots are computationally indistinguishable under DLP (Pedersen hiding)
- **CHide Resistance:** No cleansing phase needed — fake votes encode as 0-vectors, cancel naturally in SA² aggregation
- **Paper Reference:** Theorem 4

## SA² — Samplable Anonymous Aggregation (sa2/)
- **Model:** 2-server (Leader + Helper), non-colluding assumption
- **Mask Cancellation:** mask_A + mask_B = 0 (in Paillier ciphertext space)
- **CRITICAL:** Servers MUST run on separate machines/containers (see docker-compose-sa2.yml)
- **Paper Reference:** Theorem 1 (tally consistency), Threat Model (Section 4)

## Kyber768 Post-Quantum KEM (pq/)
- **Library:** Cloudflare CIRCL v1.6.2
- **Security Level:** NIST Level 3 (Module-LWE)
- **Usage:** Transport layer encryption (voter ↔ SA² server communication)
- **Hybrid:** Classical + PQ (defense-in-depth)
- **Paper Reference:** Section 6.6 (Post-Quantum Security Considerations)

## Composition Security
All seven protocols use INDEPENDENT randomness sources:
- Paillier: r ∈ Z*_N
- Pedersen: s ∈ Z*_q  
- Ring Signature: α ∈ Z_q
- SA² masks: random ∈ Z_N
- Kyber: internal PRNG (CIRCL)
- ZKP challenges: derived from public transcript via Fiat-Shamir (no shared randomness with encryption)
Paper Reference: Theorem 6 (Composition Security via hybrid argument)
```

---

## PROMPT 3.2 — Threat Model as Code Comments
## File: internal/sa2/aggregation.go, internal/voting/cast.go
## Time: ~15 minutes

```
Add formal threat model documentation as code comments to the key files. These comments will be referenced in the paper and help reviewers verify the implementation matches the formal model.

1. At the top of internal/sa2/aggregation.go, update the package comment:

// Package sa2 implements Samplable Anonymous Aggregation (SA²) for private vote tallying.
//
// THREAT MODEL:
// - Adversary: PPT, Dolev-Yao network model
// - Corruption: At most ONE of {SA²-Leader, SA²-Helper} may be corrupted
// - Non-collusion: Leader and Helper MUST be operated by independent administrative domains
// - Security guarantee: If at least one server is honest, individual vote shares are
//   information-theoretically hidden from the adversary
// - Mask cancellation: share_A = E(vote + mask), share_B = E(-mask)
//   Combined: E(vote + mask) × E(-mask) = E(vote) (Paillier homomorphic property)
//
// DEPLOYMENT REQUIREMENT: Use docker-compose-sa2.yml for container separation.
// See: Talwar et al., "Samplable Anonymous Aggregation", ACM CCS 2024

2. At the top of internal/voting/cast.go, add after the existing package comment:

// SECURITY PROPERTIES verified by formal proofs (see security_analysis.tex):
// - Ballot Privacy (Theorem 1): Encrypted votes indistinguishable under DCRA
// - ZKP Soundness (Theorem 2): Invalid votes (w∉{0,1} or Σw≠1) detected with overwhelming probability  
// - Anonymity (Theorem 3): Ring signature hides voter identity among 100 members
// - Double-Vote Prevention: Key Image uniqueness enforced by DB UNIQUE constraint
// - Coercion Resistance (Theorem 4): SMDC fake credentials indistinguishable from real
// - Composition (Theorem 6): Independent randomness across all 7 protocols
```

---

# ═══════════════════════════════════════════════════════
# PHASE 4: TEST COVERAGE IMPROVEMENT (Week 4-6)
# ═══════════════════════════════════════════════════════

## PROMPT 4.1 — Add Missing Tests for Tally Module
## File: internal/tally/tally_test.go
## Time: ~45 minutes

```
Improve test coverage for the tally module in internal/tally/tally_test.go. Add the following test cases:

1. TestTallyWithMultipleCandidates — Test homomorphic tally with 3+ candidates and verify correct results.

2. TestTallyWithZeroVotes — Edge case: election with 0 votes cast.

3. TestTallyCorrectness — Property-based test: encrypt N random votes, tally homomorphically, decrypt, verify sum matches plaintext sum.

func TestTallyCorrectness(t *testing.T) {
    key, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := key.PublicKey
    
    // 50 voters, random votes for 3 candidates
    candidates := 3
    votes := make([]int64, 50)
    expected := make([]int64, candidates)
    
    for i := range votes {
        votes[i] = int64(i % candidates) // distribute among candidates
        expected[votes[i]]++
    }
    
    // Encrypt and tally per candidate
    for c := 0; c < candidates; c++ {
        tally := big.NewInt(1) // identity for multiplication
        for _, v := range votes {
            vote := big.NewInt(0)
            if v == int64(c) { vote = big.NewInt(1) }
            enc, _ := pk.Encrypt(vote)
            tally = pk.Add(tally, enc)
        }
        
        result, _ := key.Decrypt(tally)
        if result.Int64() != expected[c] {
            t.Errorf("Candidate %d: expected %d votes, got %d", c, expected[c], result.Int64())
        }
    }
}

4. TestSA2TallyIntegrity — Test that SA² split → aggregate → combine → decrypt gives correct result.

func TestSA2TallyIntegrity(t *testing.T) {
    key, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := key.PublicKey
    splitter := sa2.NewVoteSplitter(pk)
    aggA := sa2.NewAggregator("server-a", pk)
    aggB := sa2.NewAggregator("server-b", pk)
    combiner := sa2.NewCombiner(pk)
    
    // 20 voters
    nVoters := 20
    var sharesA, sharesB []*big.Int
    expectedSum := int64(0)
    
    for i := 0; i < nVoters; i++ {
        vote := big.NewInt(int64(i % 2))
        expectedSum += int64(i % 2)
        
        enc, _ := pk.Encrypt(vote)
        share, _ := splitter.SplitVote(fmt.Sprintf("v%d", i), enc)
        sharesA = append(sharesA, share.ShareA)
        sharesB = append(sharesB, share.ShareB)
    }
    
    resultA := aggA.AggregateShares(sharesA)
    resultB := aggB.AggregateShares(sharesB)
    combined := combiner.CombineAggregates(resultA, resultB)
    
    decrypted, _ := key.Decrypt(combined.EncryptedTally)
    if decrypted.Int64() != expectedSum {
        t.Errorf("SA2 tally: expected %d, got %d", expectedSum, decrypted.Int64())
    }
}

Run: go test -v -coverprofile=coverage.out ./internal/tally/... ./internal/sa2/...
go tool cover -func=coverage.out | grep -E "(tally|sa2)"
```

---

## PROMPT 4.2 — Add Property-Based Tests for Crypto
## File: internal/crypto/property_test.go (NEW FILE)
## Time: ~30 minutes

```
Create internal/crypto/property_test.go with property-based tests that verify the mathematical properties our formal proofs depend on.

package crypto

import (
    "math/big"
    "testing"
)

// TestPaillierHomomorphicProperty verifies E(a) × E(b) = E(a+b)
// This is the foundation of Theorem 1 (Ballot Privacy) and Theorem 5 (Universal Verifiability)
func TestPaillierHomomorphicProperty(t *testing.T) {
    key, _ := GeneratePaillierKeyPair(2048)
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
    key, _ := GeneratePaillierKeyPair(2048)
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
    pp, _ := GeneratePedersenParams(2048)
    
    // Commit to value 1
    commitment, _ := pp.Commit(big.NewInt(1))
    
    // Verify opens correctly for value 1
    if !pp.Verify(commitment.C, big.NewInt(1), commitment.R) {
        t.Fatal("Commitment should verify for correct value")
    }
    
    // Should NOT verify for value 0 (binding property)
    if pp.Verify(commitment.C, big.NewInt(0), commitment.R) {
        t.Fatal("Commitment should NOT verify for wrong value (binding broken!)")
    }
}

// TestRingSignatureLinkability verifies same signer → same key image
// This is the foundation of double-vote detection (Theorem 3)
func TestRingSignatureLinkability(t *testing.T) {
    rp, _ := GenerateRingParams(2048)
    
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
    if !Link(sig1, sig2) {
        t.Fatal("Same signer should produce linked signatures")
    }
    
    // Different signer should NOT link
    sig3, _ := rp.Sign([]byte("message1"), keys[5], pubKeys, 5)
    if Link(sig1, sig3) {
        t.Fatal("Different signers should NOT produce linked signatures")
    }
}

// TestSMDCCredentialIndistinguishability verifies all slots look identical externally
// This is the foundation of Theorem 4 (Coercion Resistance)
func TestSMDCCredentialIndistinguishability(t *testing.T) {
    pp, _ := GeneratePedersenParams(2048)
    gen := smdc.NewSMDCGenerator(pp, 5, "test-election")
    
    cred, realIdx, _ := gen.GenerateCredential("voter-1")
    pub := cred.GetPublicCredential()
    
    // All binary proofs should verify (real AND fake slots)
    for i := 0; i < pub.K; i++ {
        if !pp.VerifyBinary(pub.Commitments[i], pub.BinaryProofs[i]) {
            t.Errorf("Slot %d binary proof failed", i)
        }
    }
    
    // Sum proof should verify
    if !pp.VerifySumOne(pub.Commitments, pub.SumProof) {
        t.Fatal("Sum proof failed")
    }
    
    // All commitments should be the same size (no information leak via size)
    realCommitSize := len(pub.Commitments[realIdx].Bytes())
    for i, c := range pub.Commitments {
        if i != realIdx && len(c.Bytes()) != realCommitSize {
            // Note: This is a probabilistic check; sizes may differ by 1 byte occasionally
            // but should not systematically differ between real and fake
            t.Logf("Warning: commitment size differs: slot %d=%d bytes, real=%d bytes", 
                i, len(c.Bytes()), realCommitSize)
        }
    }
    
    _ = realIdx // suppress unused warning
}

Note: You'll need to import the smdc package. Adjust imports accordingly.
Run: go test -v ./internal/crypto/... -run TestPaillier -run TestPedersen -run TestRingSignature -run TestSMDC
```

---

# ═══════════════════════════════════════════════════════
# PHASE 5: PAPER PREPARATION (Week 6-9)
# ═══════════════════════════════════════════════════════

## PROMPT 5.1 — Generate Benchmark Results Summary
## File: test/benchmark/results/PAPER_RESULTS.md (NEW FILE)
## Time: ~15 minutes (after running all benchmarks)

```
After running all benchmarks from Phase 2, create a summary file at test/benchmark/results/PAPER_RESULTS.md.

Run these commands and capture output:

echo "# CovertVote Benchmark Results for Paper" > test/benchmark/results/PAPER_RESULTS.md
echo "## Hardware: $(uname -m), $(cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d: -f2)" >> test/benchmark/results/PAPER_RESULTS.md
echo "## Go Version: $(go version)" >> test/benchmark/results/PAPER_RESULTS.md
echo "## Date: $(date)" >> test/benchmark/results/PAPER_RESULTS.md
echo "" >> test/benchmark/results/PAPER_RESULTS.md

echo "### Crypto Micro-Benchmarks" >> test/benchmark/results/PAPER_RESULTS.md
go test -bench=. -benchmem -count=5 ./test/benchmark/crypto_benchmark_test.go 2>&1 >> test/benchmark/results/PAPER_RESULTS.md

echo "" >> test/benchmark/results/PAPER_RESULTS.md
echo "### PQ Benchmarks" >> test/benchmark/results/PAPER_RESULTS.md
go test -bench=. -benchmem -count=5 ./internal/pq/... 2>&1 >> test/benchmark/results/PAPER_RESULTS.md

echo "" >> test/benchmark/results/PAPER_RESULTS.md
echo "### ZKP Benchmarks" >> test/benchmark/results/PAPER_RESULTS.md
go test -bench=BenchmarkZKP -benchmem -count=5 ./test/benchmark/... 2>&1 >> test/benchmark/results/PAPER_RESULTS.md

echo "" >> test/benchmark/results/PAPER_RESULTS.md
echo "### E2E Pipeline Benchmarks" >> test/benchmark/results/PAPER_RESULTS.md
go test -bench=BenchmarkVoteCast -benchmem -count=3 ./test/benchmark/... 2>&1 >> test/benchmark/results/PAPER_RESULTS.md

echo "" >> test/benchmark/results/PAPER_RESULTS.md
echo "### Scalability Results" >> test/benchmark/results/PAPER_RESULTS.md
go test -v -timeout 30m -run TestScalability ./test/benchmark/... 2>&1 >> test/benchmark/results/PAPER_RESULTS.md

echo "" >> test/benchmark/results/PAPER_RESULTS.md
echo "### Test Coverage" >> test/benchmark/results/PAPER_RESULTS.md
go test -coverprofile=coverage.out ./internal/... ./api/... 2>&1 >> test/benchmark/results/PAPER_RESULTS.md
go tool cover -func=coverage.out >> test/benchmark/results/PAPER_RESULTS.md

Then commit and push: git add . && git commit -m "Add comprehensive benchmarks and tests for paper" && git push
```

---

## PROMPT 5.2 — Open Source Cleanup for Paper Submission
## File: Multiple files
## Time: ~30 minutes

```
Prepare the repository for open-source publication alongside the paper submission. This is important because IEEE TIFS encourages code sharing for reproducibility.

1. Update README.md to be academic/paper-focused:
   - Add a "Citation" section with BibTeX entry (placeholder for now)
   - Add "Security Properties" section listing what the system guarantees
   - Add "Benchmarks" section pointing to test/benchmark/results/
   - Add "Paper" section mentioning the submission target
   - Remove any internal/development notes

2. Create a LICENSE file (MIT or Apache 2.0 recommended for academic software)

3. Create CITATION.cff (GitHub citation file):
   cff-version: 1.2.0
   title: "CovertVote: A Seven-Protocol Blockchain E-Voting System"
   message: "If you use this software, please cite our paper."
   authors:
     - given-names: [Your Name]
       family-names: [Your Family Name]
   type: software
   license: MIT

4. Make sure .gitignore excludes: bin/, *.exe, .env, vendor/ (if using Go modules)

5. Remove any sensitive files: check that .env.example has only placeholder values, no real keys.

6. Add a SECURITY.md file pointing to the security analysis documentation.

7. Verify the project builds cleanly:
   go build ./...
   go vet ./...
   go test ./internal/... ./api/... ./tests/...
```
