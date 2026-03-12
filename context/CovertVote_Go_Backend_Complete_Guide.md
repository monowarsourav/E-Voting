# CovertVote: Complete Go Backend Development Guide
## Blockchain-Based E-Voting System with SMDC + SA²

---

# 📚 Table of Contents

1. [Project Overview](#1-project-overview)
2. [System Architecture](#2-system-architecture)
3. [Project Structure](#3-project-structure)
4. [Required Libraries](#4-required-libraries)
5. [Algorithm Flow Summary](#5-algorithm-flow-summary)
6. [Step 1: Paillier Encryption](#step-1-paillier-encryption)
7. [Step 2: Pedersen Commitment](#step-2-pedersen-commitment)
8. [Step 3: ZK Proofs (Σ-Protocol)](#step-3-zk-proofs-σ-protocol)
9. [Step 4: SMDC Credential System](#step-4-smdc-credential-system)
10. [Step 5: Ring Signature](#step-5-ring-signature)
11. [Step 6: SA² Aggregation](#step-6-sa²-aggregation)
12. [Step 7: Biometric (Fingerprint)](#step-7-biometric-fingerprint)
13. [Step 8: Voter Registration](#step-8-voter-registration)
14. [Step 9: Vote Casting](#step-9-vote-casting)
15. [Step 10: Tallying & Decryption](#step-10-tallying--decryption)
16. [Step 11: Hyperledger Fabric Integration](#step-11-hyperledger-fabric-integration)
17. [Step 12: API Endpoints](#step-12-api-endpoints)
18. [Step 13: Testing Guide](#step-13-testing-guide)
19. [Step 14: Post-Quantum Hybrid (Kyber)](#step-14-post-quantum-hybrid-kyber)
20. [Performance Optimization](#performance-optimization)
21. [Security Checklist](#security-checklist)

---

# 1. Project Overview

## 1.1 কী বানাচ্ছি?

**CovertVote** - একটি blockchain-based e-voting system যেখানে:
- **SMDC**: Self-Masking Deniable Credentials (coercion resistance)
- **SA²**: Samplable Anonymous Aggregation (privacy)
- **Paillier**: Homomorphic encryption (secure counting)
- **Hyperledger Fabric**: Blockchain storage

## 1.2 Key Features

| Feature | Technology | Purpose |
|---------|------------|---------|
| Coercion Resistance | SMDC (k=5 slots) | Voter কে force করা যাবে না |
| Anonymity | Ring Signature + SA² | কে কাকে vote দিয়েছে জানা যাবে না |
| Verifiability | ZK Proofs | সবাই verify করতে পারবে |
| Scalability | Off-chain compute | 200M voters handle করতে পারবে |
| Immutability | Hyperledger Fabric | Vote tamper করা যাবে না |

## 1.3 Complexity

```
CovertVote: O(n × k) where k = 5 (constant)
         = O(n) - Linear time!

vs ISE-Voting: O(n × m²) - Quadratic in candidates
```

---

# 2. System Architecture

## 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLIENT LAYER                            │
│  [Mobile App] [Web App] [Biometric Device]                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         API LAYER (Go)                          │
│  [REST API] [gRPC] [WebSocket]                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    COMPUTATION LAYER (Off-Chain)                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │ Paillier │ │ Pedersen │ │ ZK Proof │ │   Ring   │           │
│  │ Encrypt  │ │ Commit   │ │ Generate │ │   Sign   │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
│  ┌──────────┐ ┌──────────┐                                      │
│  │   SMDC   │ │   SA²    │                                      │
│  │ Generate │ │Aggregate │                                      │
│  └──────────┘ └──────────┘                                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    STORAGE LAYER (On-Chain)                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              HYPERLEDGER FABRIC                          │   │
│  │  - Voter Merkle Root                                     │   │
│  │  - SMDC Commitments                                      │   │
│  │  - Encrypted Votes                                       │   │
│  │  - Final Tally + Proofs                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      SA² SERVER LAYER                           │
│  ┌─────────────────┐           ┌─────────────────┐             │
│  │    SERVER A     │           │    SERVER B     │             │
│  │  (Aggregator 1) │           │  (Aggregator 2) │             │
│  └─────────────────┘           └─────────────────┘             │
│         │                               │                       │
│         └───────────┬───────────────────┘                       │
│                     ▼                                           │
│          ┌─────────────────┐                                    │
│          │ Threshold Decrypt│                                   │
│          │   (Combined)     │                                   │
│          └─────────────────┘                                    │
└─────────────────────────────────────────────────────────────────┘
```

## 2.2 Data Flow

```
Fingerprint → SHA3 Hash → Merkle Verify → SMDC Generate
     │
     ▼
[k=5 slots: 1 real, 4 fake] → Pedersen Commit → ZK Proof
     │
     ▼
Vote Encrypt (Paillier) → Ring Sign → Blockchain Store
     │
     ▼
SA² Split → Server A + Server B → Aggregate
     │
     ▼
Threshold Decrypt → Final Tally → NIZK Proof → Publish
```

---

# 3. Project Structure

```
covertvote/
├── cmd/
│   ├── server/
│   │   └── main.go                 # Main API server
│   ├── aggregator-a/
│   │   └── main.go                 # SA² Server A
│   └── aggregator-b/
│       └── main.go                 # SA² Server B
│
├── internal/
│   ├── crypto/
│   │   ├── paillier.go             # Paillier encryption
│   │   ├── paillier_test.go
│   │   ├── pedersen.go             # Pedersen commitment
│   │   ├── pedersen_test.go
│   │   ├── zkproof.go              # Zero-knowledge proofs
│   │   ├── zkproof_test.go
│   │   ├── ring_signature.go       # Ring signatures
│   │   ├── ring_signature_test.go
│   │   ├── kyber.go                # Post-quantum (hybrid)
│   │   └── hash.go                 # SHA-3 utilities
│   │
│   ├── smdc/
│   │   ├── credential.go           # SMDC credential generation
│   │   ├── credential_test.go
│   │   ├── proof.go                # SMDC proofs
│   │   └── types.go                # SMDC types
│   │
│   ├── sa2/
│   │   ├── aggregation.go          # SA² aggregation
│   │   ├── aggregation_test.go
│   │   ├── server.go               # SA² server logic
│   │   ├── share.go                # Secret sharing
│   │   └── types.go                # SA² types
│   │
│   ├── biometric/
│   │   ├── fingerprint.go          # Fingerprint processing
│   │   ├── fingerprint_test.go
│   │   ├── liveness.go             # Liveness detection
│   │   └── hash.go                 # Biometric hashing
│   │
│   ├── voter/
│   │   ├── registration.go         # Voter registration
│   │   ├── registration_test.go
│   │   ├── merkle.go               # Merkle tree for eligibility
│   │   └── types.go                # Voter types
│   │
│   ├── voting/
│   │   ├── cast.go                 # Vote casting
│   │   ├── cast_test.go
│   │   ├── ballot.go               # Ballot structure
│   │   └── types.go                # Voting types
│   │
│   ├── tally/
│   │   ├── decrypt.go              # Threshold decryption
│   │   ├── decrypt_test.go
│   │   ├── count.go                # Vote counting
│   │   └── proof.go                # Tally proofs
│   │
│   └── blockchain/
│       ├── hyperledger.go          # Hyperledger Fabric SDK
│       ├── hyperledger_test.go
│       ├── chaincode.go            # Smart contract interface
│       └── types.go                # Blockchain types
│
├── pkg/
│   ├── utils/
│   │   ├── bigint.go               # Big integer utilities
│   │   ├── random.go               # Secure random
│   │   └── convert.go              # Type conversions
│   │
│   └── config/
│       ├── config.go               # Configuration
│       └── config.yaml             # Config file
│
├── api/
│   ├── handlers/
│   │   ├── register.go             # Registration handlers
│   │   ├── vote.go                 # Voting handlers
│   │   ├── tally.go                # Tally handlers
│   │   └── verify.go               # Verification handlers
│   │
│   ├── middleware/
│   │   ├── auth.go                 # Authentication
│   │   └── ratelimit.go            # Rate limiting
│   │
│   └── router.go                   # API routes
│
├── chaincode/
│   └── covertvote/
│       ├── covertvote.go           # Hyperledger chaincode
│       └── go.mod
│
├── scripts/
│   ├── setup.sh                    # Setup script
│   ├── test.sh                     # Test script
│   └── benchmark.sh                # Benchmark script
│
├── test/
│   ├── integration/
│   │   └── full_flow_test.go       # Full flow test
│   └── benchmark/
│       └── performance_test.go     # Performance benchmarks
│
├── docs/
│   ├── API.md                      # API documentation
│   └── ARCHITECTURE.md             # Architecture docs
│
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

---

# 4. Required Libraries

## 4.1 Go Modules Installation

```bash
# Initialize project
mkdir covertvote && cd covertvote
go mod init github.com/yourusername/covertvote

# Core crypto
go get golang.org/x/crypto/sha3
go get github.com/cloudflare/circl/kem/kyber/kyber768

# Big number operations (built-in)
# math/big - already available

# Merkle Tree
go get github.com/cbergoon/merkletree

# Hyperledger Fabric SDK
go get github.com/hyperledger/fabric-sdk-go

# API Framework
go get github.com/gin-gonic/gin

# Configuration
go get github.com/spf13/viper

# Logging
go get go.uber.org/zap

# Testing
go get github.com/stretchr/testify
```

## 4.2 Library Summary Table

| Purpose | Library | Import Path |
|---------|---------|-------------|
| SHA-3 Hash | x/crypto | `golang.org/x/crypto/sha3` |
| Big Numbers | stdlib | `math/big` |
| Secure Random | stdlib | `crypto/rand` |
| Merkle Tree | merkletree | `github.com/cbergoon/merkletree` |
| Post-Quantum Kyber | circl | `github.com/cloudflare/circl/kem/kyber/kyber768` |
| Hyperledger SDK | fabric-sdk-go | `github.com/hyperledger/fabric-sdk-go` |
| HTTP Server | gin | `github.com/gin-gonic/gin` |
| Config | viper | `github.com/spf13/viper` |
| Logging | zap | `go.uber.org/zap` |
| Testing | testify | `github.com/stretchr/testify` |

## 4.3 go.mod Example

```go
module github.com/yourusername/covertvote

go 1.21

require (
    golang.org/x/crypto v0.17.0
    github.com/cloudflare/circl v1.3.7
    github.com/cbergoon/merkletree v0.2.0
    github.com/hyperledger/fabric-sdk-go v1.0.0
    github.com/gin-gonic/gin v1.9.1
    github.com/spf13/viper v1.18.2
    go.uber.org/zap v1.26.0
    github.com/stretchr/testify v1.8.4
)
```

---

# 5. Algorithm Flow Summary

## 5.1 Complete Flow Table

### Phase 1: Registration
| Step | User Action | Algorithm | Why |
|------|-------------|-----------|-----|
| 1.1 | Fingerprint দেয় | **SHA-3 Hash** | Fingerprint কে fixed-size credential এ convert |
| 1.2 | Liveness check | **CNN Model** | Deepfake/photo attack detect |
| 1.3 | NID verify | **AES-256 Decrypt** | NID data securely read |
| 1.4 | Eligibility check | **Merkle Tree Lookup** | O(log n) এ voter valid কিনা check |
| 1.5 | Voter ID generate | **HMAC-SHA256** | Unique, deterministic ID create |

### Phase 2: SMDC Credential Generation
| Step | Action | Algorithm | Why |
|------|--------|-----------|-----|
| 2.1 | k slots create (k=5) | **Secure Random** | Fake slots এর জন্য |
| 2.2 | 1টা real slot select | **Random Choice** | কোনটা real কেউ জানবে না |
| 2.3 | Weights commit | **Pedersen Commitment** | w hide করে |
| 2.4 | Binary proof | **Σ-Protocol** | w ∈ {0,1} prove |
| 2.5 | Sum proof | **Σ-Protocol** | Σw = 1 prove |

### Phase 3: Vote Casting
| Step | Action | Algorithm | Why |
|------|--------|-----------|-----|
| 3.1 | Vote encrypt | **Paillier** | Homomorphic encryption |
| 3.2 | Weighted vote | **Paillier Multiply** | E(w×v) compute |
| 3.3 | Validity proof | **ZK Proof** | Valid vote prove |
| 3.4 | Anonymous sign | **Ring Signature** | Identity hide |
| 3.5 | Submit | **Hyperledger TX** | Store on chain |

### Phase 4: SA² Aggregation
| Step | Action | Algorithm | Why |
|------|--------|-----------|-----|
| 4.1 | Vote split | **Secret Sharing** | 2 shares create |
| 4.2 | Server A sum | **Paillier Add** | Aggregate share A |
| 4.3 | Server B sum | **Paillier Add** | Aggregate share B |
| 4.4 | Combine | **Modular Add** | Final encrypted sum |

### Phase 5: Tallying
| Step | Action | Algorithm | Why |
|------|--------|-----------|-----|
| 5.1 | Partial decrypt A | **Threshold Decrypt** | Server A's part |
| 5.2 | Partial decrypt B | **Threshold Decrypt** | Server B's part |
| 5.3 | Final result | **Lagrange** | Combine decryptions |
| 5.4 | Proof | **NIZK** | Correct decrypt prove |

---

# Step 1: Paillier Encryption

## 1.1 Algorithm

```
╔══════════════════════════════════════════════════════════════╗
║                    PAILLIER CRYPTOSYSTEM                     ║
╠══════════════════════════════════════════════════════════════╣
║ KEY GENERATION:                                              ║
║   1. Choose two large primes p, q                            ║
║   2. Compute n = p × q                                       ║
║   3. Compute λ = lcm(p-1, q-1)                               ║
║   4. Choose g = n + 1                                        ║
║   5. Compute μ = (L(g^λ mod n²))^(-1) mod n                  ║
║   where L(x) = (x-1)/n                                       ║
║                                                              ║
║ Public Key: (n, g)                                           ║
║ Private Key: (λ, μ)                                          ║
╠══════════════════════════════════════════════════════════════╣
║ ENCRYPTION: E(m)                                             ║
║   1. Choose random r where 0 < r < n                         ║
║   2. Compute c = g^m × r^n mod n²                            ║
╠══════════════════════════════════════════════════════════════╣
║ DECRYPTION: D(c)                                             ║
║   1. Compute m = L(c^λ mod n²) × μ mod n                     ║
╠══════════════════════════════════════════════════════════════╣
║ HOMOMORPHIC PROPERTIES:                                      ║
║   Addition: E(m1) × E(m2) = E(m1 + m2) mod n²                ║
║   Scalar:   E(m)^k = E(k × m) mod n²                         ║
╚══════════════════════════════════════════════════════════════╝
```

## 1.2 Go Implementation

```go
// internal/crypto/paillier.go

package crypto

import (
    "crypto/rand"
    "errors"
    "math/big"
)

// PaillierPublicKey holds the public key
type PaillierPublicKey struct {
    N  *big.Int // n = p*q
    G  *big.Int // g = n+1
    N2 *big.Int // n²
}

// PaillierPrivateKey holds the private key
type PaillierPrivateKey struct {
    PublicKey *PaillierPublicKey
    Lambda    *big.Int // λ = lcm(p-1, q-1)
    Mu        *big.Int // μ = L(g^λ mod n²)^(-1) mod n
    P         *big.Int // prime p (for threshold)
    Q         *big.Int // prime q (for threshold)
}

// GeneratePaillierKeyPair generates a new Paillier key pair
// bits: key size (2048 recommended for security)
func GeneratePaillierKeyPair(bits int) (*PaillierPrivateKey, error) {
    // Step 1: Generate two large primes p and q
    p, err := rand.Prime(rand.Reader, bits/2)
    if err != nil {
        return nil, err
    }
    
    q, err := rand.Prime(rand.Reader, bits/2)
    if err != nil {
        return nil, err
    }
    
    // Ensure p != q
    for p.Cmp(q) == 0 {
        q, err = rand.Prime(rand.Reader, bits/2)
        if err != nil {
            return nil, err
        }
    }
    
    // Step 2: Compute n = p × q
    n := new(big.Int).Mul(p, q)
    
    // Compute n²
    n2 := new(big.Int).Mul(n, n)
    
    // Step 3: Compute λ = lcm(p-1, q-1)
    p1 := new(big.Int).Sub(p, big.NewInt(1)) // p-1
    q1 := new(big.Int).Sub(q, big.NewInt(1)) // q-1
    lambda := lcm(p1, q1)
    
    // Step 4: g = n + 1 (standard choice)
    g := new(big.Int).Add(n, big.NewInt(1))
    
    // Step 5: Compute μ = L(g^λ mod n²)^(-1) mod n
    // where L(x) = (x-1)/n
    gLambda := new(big.Int).Exp(g, lambda, n2) // g^λ mod n²
    l := lFunction(gLambda, n)                  // L(g^λ mod n²)
    mu := new(big.Int).ModInverse(l, n)         // μ = L^(-1) mod n
    
    if mu == nil {
        return nil, errors.New("failed to compute modular inverse")
    }
    
    publicKey := &PaillierPublicKey{
        N:  n,
        G:  g,
        N2: n2,
    }
    
    return &PaillierPrivateKey{
        PublicKey: publicKey,
        Lambda:    lambda,
        Mu:        mu,
        P:         p,
        Q:         q,
    }, nil
}

// lFunction computes L(x) = (x-1)/n
func lFunction(x, n *big.Int) *big.Int {
    // L(x) = (x - 1) / n
    xMinus1 := new(big.Int).Sub(x, big.NewInt(1))
    return new(big.Int).Div(xMinus1, n)
}

// lcm computes the least common multiple of a and b
func lcm(a, b *big.Int) *big.Int {
    // lcm(a,b) = (a × b) / gcd(a,b)
    gcd := new(big.Int).GCD(nil, nil, a, b)
    ab := new(big.Int).Mul(a, b)
    return new(big.Int).Div(ab, gcd)
}

// Encrypt encrypts a plaintext message m
// Returns ciphertext c = g^m × r^n mod n²
func (pk *PaillierPublicKey) Encrypt(m *big.Int) (*big.Int, error) {
    // Validate: 0 <= m < n
    if m.Sign() < 0 || m.Cmp(pk.N) >= 0 {
        return nil, errors.New("message out of range")
    }
    
    // Choose random r where 0 < r < n
    r, err := rand.Int(rand.Reader, pk.N)
    if err != nil {
        return nil, err
    }
    // Ensure r > 0
    for r.Sign() == 0 {
        r, err = rand.Int(rand.Reader, pk.N)
        if err != nil {
            return nil, err
        }
    }
    
    // Compute g^m mod n²
    gm := new(big.Int).Exp(pk.G, m, pk.N2)
    
    // Compute r^n mod n²
    rn := new(big.Int).Exp(r, pk.N, pk.N2)
    
    // Compute c = g^m × r^n mod n²
    c := new(big.Int).Mul(gm, rn)
    c.Mod(c, pk.N2)
    
    return c, nil
}

// EncryptWithRandomness encrypts with specified randomness (for proofs)
func (pk *PaillierPublicKey) EncryptWithRandomness(m, r *big.Int) (*big.Int, error) {
    gm := new(big.Int).Exp(pk.G, m, pk.N2)
    rn := new(big.Int).Exp(r, pk.N, pk.N2)
    c := new(big.Int).Mul(gm, rn)
    c.Mod(c, pk.N2)
    return c, nil
}

// Decrypt decrypts a ciphertext c
// Returns plaintext m = L(c^λ mod n²) × μ mod n
func (sk *PaillierPrivateKey) Decrypt(c *big.Int) (*big.Int, error) {
    pk := sk.PublicKey
    
    // Compute c^λ mod n²
    cLambda := new(big.Int).Exp(c, sk.Lambda, pk.N2)
    
    // Compute L(c^λ mod n²)
    l := lFunction(cLambda, pk.N)
    
    // Compute m = L × μ mod n
    m := new(big.Int).Mul(l, sk.Mu)
    m.Mod(m, pk.N)
    
    return m, nil
}

// Add performs homomorphic addition: E(m1 + m2) = E(m1) × E(m2) mod n²
func (pk *PaillierPublicKey) Add(c1, c2 *big.Int) *big.Int {
    result := new(big.Int).Mul(c1, c2)
    result.Mod(result, pk.N2)
    return result
}

// AddPlaintext adds plaintext to ciphertext: E(m1 + m2) = E(m1) × g^m2 mod n²
func (pk *PaillierPublicKey) AddPlaintext(c, m *big.Int) *big.Int {
    gm := new(big.Int).Exp(pk.G, m, pk.N2)
    result := new(big.Int).Mul(c, gm)
    result.Mod(result, pk.N2)
    return result
}

// Multiply performs scalar multiplication: E(k × m) = E(m)^k mod n²
func (pk *PaillierPublicKey) Multiply(c, k *big.Int) *big.Int {
    result := new(big.Int).Exp(c, k, pk.N2)
    return result
}

// AddMultiple adds multiple ciphertexts
func (pk *PaillierPublicKey) AddMultiple(ciphertexts []*big.Int) *big.Int {
    result := big.NewInt(1)
    for _, c := range ciphertexts {
        result = pk.Add(result, c)
    }
    return result
}
```

## 1.3 Test File

```go
// internal/crypto/paillier_test.go

package crypto

import (
    "math/big"
    "testing"
)

func TestPaillierEncryptDecrypt(t *testing.T) {
    // Generate keys (1024 bits for testing, use 2048 in production)
    sk, err := GeneratePaillierKeyPair(1024)
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
    sk, _ := GeneratePaillierKeyPair(1024)
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
    sk, _ := GeneratePaillierKeyPair(1024)
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
```

---

# Step 2: Pedersen Commitment

## 2.1 Algorithm

```
╔══════════════════════════════════════════════════════════════╗
║                   PEDERSEN COMMITMENT                        ║
╠══════════════════════════════════════════════════════════════╣
║ PROPERTIES:                                                  ║
║   - Perfectly Hiding: Commitment reveals nothing about m     ║
║   - Computationally Binding: Cannot open to different value  ║
║   - Homomorphic: C(m1) × C(m2) = C(m1 + m2)                  ║
╠══════════════════════════════════════════════════════════════╣
║ SETUP:                                                       ║
║   1. Choose large prime p and prime order q where q|(p-1)    ║
║   2. Choose generators g, h ∈ Gq                             ║
║   3. CRITICAL: Nobody knows log_g(h)                         ║
║                                                              ║
║ Public Parameters: (p, q, g, h)                              ║
╠══════════════════════════════════════════════════════════════╣
║ COMMIT(m):                                                   ║
║   1. Choose random r ∈ Zq                                    ║
║   2. Compute C = g^m × h^r mod p                             ║
║   3. Return (C, r) where r is the opening                    ║
╠══════════════════════════════════════════════════════════════╣
║ VERIFY(C, m, r):                                             ║
║   1. Check: C == g^m × h^r mod p                             ║
╚══════════════════════════════════════════════════════════════╝
```

## 2.2 Go Implementation

```go
// internal/crypto/pedersen.go

package crypto

import (
    "crypto/rand"
    "errors"
    "math/big"
)

// PedersenParams holds the public parameters for Pedersen commitments
type PedersenParams struct {
    P *big.Int // Large prime
    Q *big.Int // Prime order of subgroup
    G *big.Int // Generator 1
    H *big.Int // Generator 2 (discrete log to g unknown)
}

// Commitment represents a Pedersen commitment
type Commitment struct {
    C *big.Int // Commitment value
    R *big.Int // Randomness (kept secret until opening)
}

// GeneratePedersenParams generates secure Pedersen parameters
func GeneratePedersenParams(bits int) (*PedersenParams, error) {
    // Generate safe prime p = 2q + 1 where q is also prime
    // For simplicity, we generate q first
    q, err := rand.Prime(rand.Reader, bits-1)
    if err != nil {
        return nil, err
    }
    
    // p = 2q + 1
    p := new(big.Int).Mul(q, big.NewInt(2))
    p.Add(p, big.NewInt(1))
    
    // Check if p is prime (for safe prime)
    // In production, use proper safe prime generation
    if !p.ProbablyPrime(20) {
        // Retry or use different method
        return GeneratePedersenParams(bits)
    }
    
    // Find generator g
    g, err := findGenerator(p, q)
    if err != nil {
        return nil, err
    }
    
    // Find generator h (must be independent of g)
    // We hash g to get h, ensuring nobody knows log_g(h)
    h, err := deriveIndependentGenerator(p, q, g)
    if err != nil {
        return nil, err
    }
    
    return &PedersenParams{
        P: p,
        Q: q,
        G: g,
        H: h,
    }, nil
}

// findGenerator finds a generator of the subgroup of order q
func findGenerator(p, q *big.Int) (*big.Int, error) {
    one := big.NewInt(1)
    pMinus1 := new(big.Int).Sub(p, one)
    exp := new(big.Int).Div(pMinus1, q) // (p-1)/q
    
    for i := 0; i < 1000; i++ {
        // Random element
        h, err := rand.Int(rand.Reader, p)
        if err != nil {
            return nil, err
        }
        
        // g = h^((p-1)/q) mod p
        g := new(big.Int).Exp(h, exp, p)
        
        // Check g != 1
        if g.Cmp(one) != 0 {
            return g, nil
        }
    }
    
    return nil, errors.New("failed to find generator")
}

// deriveIndependentGenerator creates h from g using hash
func deriveIndependentGenerator(p, q, g *big.Int) (*big.Int, error) {
    // Use hash-to-group to derive h
    // This ensures log_g(h) is unknown
    
    // Simple method: h = hash(g)^((p-1)/q) mod p
    hasher := sha3.New256()
    hasher.Write(g.Bytes())
    hashBytes := hasher.Sum(nil)
    
    hashInt := new(big.Int).SetBytes(hashBytes)
    hashInt.Mod(hashInt, p)
    
    pMinus1 := new(big.Int).Sub(p, big.NewInt(1))
    exp := new(big.Int).Div(pMinus1, q)
    
    h := new(big.Int).Exp(hashInt, exp, p)
    
    // Ensure h != 1 and h != g
    if h.Cmp(big.NewInt(1)) == 0 || h.Cmp(g) == 0 {
        // Add salt and retry
        hasher.Write([]byte("salt"))
        hashBytes = hasher.Sum(nil)
        hashInt.SetBytes(hashBytes)
        hashInt.Mod(hashInt, p)
        h = new(big.Int).Exp(hashInt, exp, p)
    }
    
    return h, nil
}

// Commit creates a Pedersen commitment to message m
// Returns commitment C = g^m × h^r mod p
func (pp *PedersenParams) Commit(m *big.Int) (*Commitment, error) {
    // Generate random r ∈ Zq
    r, err := rand.Int(rand.Reader, pp.Q)
    if err != nil {
        return nil, err
    }
    
    // C = g^m × h^r mod p
    gm := new(big.Int).Exp(pp.G, m, pp.P)  // g^m mod p
    hr := new(big.Int).Exp(pp.H, r, pp.P)  // h^r mod p
    c := new(big.Int).Mul(gm, hr)          // g^m × h^r
    c.Mod(c, pp.P)                          // mod p
    
    return &Commitment{
        C: c,
        R: r,
    }, nil
}

// CommitWithRandomness creates commitment with specified randomness
func (pp *PedersenParams) CommitWithRandomness(m, r *big.Int) *big.Int {
    gm := new(big.Int).Exp(pp.G, m, pp.P)
    hr := new(big.Int).Exp(pp.H, r, pp.P)
    c := new(big.Int).Mul(gm, hr)
    c.Mod(c, pp.P)
    return c
}

// Verify verifies a commitment opening
// Returns true if C == g^m × h^r mod p
func (pp *PedersenParams) Verify(commitment *Commitment, m *big.Int) bool {
    expected := pp.CommitWithRandomness(m, commitment.R)
    return expected.Cmp(commitment.C) == 0
}

// AddCommitments homomorphically adds two commitments
// C1 × C2 = g^(m1+m2) × h^(r1+r2) = Commit(m1+m2, r1+r2)
func (pp *PedersenParams) AddCommitments(c1, c2 *Commitment) *Commitment {
    // New commitment value
    newC := new(big.Int).Mul(c1.C, c2.C)
    newC.Mod(newC, pp.P)
    
    // New randomness
    newR := new(big.Int).Add(c1.R, c2.R)
    newR.Mod(newR, pp.Q)
    
    return &Commitment{
        C: newC,
        R: newR,
    }
}

// ScalarMultiply multiplies commitment by scalar
// C^k = g^(k×m) × h^(k×r) = Commit(k×m, k×r)
func (pp *PedersenParams) ScalarMultiply(c *Commitment, k *big.Int) *Commitment {
    newC := new(big.Int).Exp(c.C, k, pp.P)
    newR := new(big.Int).Mul(c.R, k)
    newR.Mod(newR, pp.Q)
    
    return &Commitment{
        C: newC,
        R: newR,
    }
}
```

## 2.3 Test File

```go
// internal/crypto/pedersen_test.go

package crypto

import (
    "math/big"
    "testing"
)

func TestPedersenCommitVerify(t *testing.T) {
    pp, err := GeneratePedersenParams(512) // Small for testing
    if err != nil {
        t.Fatalf("Setup failed: %v", err)
    }
    
    message := big.NewInt(42)
    
    commitment, err := pp.Commit(message)
    if err != nil {
        t.Fatalf("Commit failed: %v", err)
    }
    
    // Should verify correctly
    if !pp.Verify(commitment, message) {
        t.Error("Valid commitment verification failed")
    }
    
    // Should fail with wrong message
    wrongMessage := big.NewInt(43)
    if pp.Verify(commitment, wrongMessage) {
        t.Error("Invalid commitment verification should fail")
    }
}

func TestPedersenHomomorphic(t *testing.T) {
    pp, _ := GeneratePedersenParams(512)
    
    m1 := big.NewInt(10)
    m2 := big.NewInt(20)
    expectedSum := big.NewInt(30)
    
    c1, _ := pp.Commit(m1)
    c2, _ := pp.Commit(m2)
    
    // Add commitments
    cSum := pp.AddCommitments(c1, c2)
    
    // Verify sum
    if !pp.Verify(cSum, expectedSum) {
        t.Error("Homomorphic addition failed")
    }
}
```

---

# Step 3: ZK Proofs (Σ-Protocol)

## 3.1 Algorithm

```
╔══════════════════════════════════════════════════════════════╗
║              Σ-PROTOCOL FOR BINARY PROOF                     ║
║              (Prove w ∈ {0, 1} without revealing w)          ║
╠══════════════════════════════════════════════════════════════╣
║ Given: Commitment C = g^w × h^r where w ∈ {0,1}              ║
║ Prove: w is either 0 or 1 without revealing which            ║
╠══════════════════════════════════════════════════════════════╣
║ PROVER (knows w, r):                                         ║
║   If w = 0:                                                  ║
║     1. Choose random r1, d1, w1                              ║
║     2. a0 = g^w0 × h^r0 (real commitment to 0)               ║
║     3. a1 = g^w1 × h^r1 × (C/g)^(-d1) (simulated for w=1)    ║
║     4. Get challenge c = Hash(C, a0, a1)                     ║
║     5. d0 = c - d1 mod q                                     ║
║     6. f0 = r0 + d0×r mod q                                  ║
║     7. f1 = r1 (already set)                                 ║
║                                                              ║
║   If w = 1: (symmetric)                                      ║
║     Similar but swap 0 and 1                                 ║
╠══════════════════════════════════════════════════════════════╣
║ VERIFIER:                                                    ║
║   1. Check c = d0 + d1 mod q                                 ║
║   2. Check a0 = g^0 × h^f0 × C^(-d0) = h^f0 × C^(-d0)        ║
║   3. Check a1 = g^1 × h^f1 × (C/g)^(-d1)                     ║
╚══════════════════════════════════════════════════════════════╝
```

## 3.2 Go Implementation

```go
// internal/crypto/zkproof.go

package crypto

import (
    "crypto/rand"
    "math/big"
    
    "golang.org/x/crypto/sha3"
)

// BinaryProof proves that a commitment contains 0 or 1
type BinaryProof struct {
    A0 *big.Int // First announcement
    A1 *big.Int // Second announcement
    D0 *big.Int // Challenge for w=0 case
    D1 *big.Int // Challenge for w=1 case
    F0 *big.Int // Response for w=0 case
    F1 *big.Int // Response for w=1 case
}

// SumProof proves that sum of weights equals 1
type SumProof struct {
    ProductCommitment *big.Int   // Product of all commitments
    Challenge         *big.Int
    Response          *big.Int
}

// ProveBinary creates a ZK proof that w ∈ {0, 1}
func (pp *PedersenParams) ProveBinary(w, r *big.Int, C *big.Int) (*BinaryProof, error) {
    isZero := w.Cmp(big.NewInt(0)) == 0
    isOne := w.Cmp(big.NewInt(1)) == 0
    
    if !isZero && !isOne {
        return nil, errors.New("w must be 0 or 1")
    }
    
    var a0, a1, d0, d1, f0, f1 *big.Int
    
    if isZero {
        // Real proof for w=0, simulate w=1
        
        // Simulate w=1 branch
        d1, _ = rand.Int(rand.Reader, pp.Q)
        r1, _ := rand.Int(rand.Reader, pp.Q)
        
        // a1 = g × h^r1 × (C/g)^(-d1)
        // Simplified: a1 = g^1 × h^r1 × C^(-d1) × g^(d1)
        gInv := new(big.Int).ModInverse(pp.G, pp.P)
        CdivG := new(big.Int).Mul(C, gInv)
        CdivG.Mod(CdivG, pp.P)
        
        CdivGNegD1 := new(big.Int).Exp(CdivG, new(big.Int).Sub(pp.Q, d1), pp.P)
        g1 := new(big.Int).Set(pp.G)
        hr1 := new(big.Int).Exp(pp.H, r1, pp.P)
        
        a1 = new(big.Int).Mul(g1, hr1)
        a1.Mul(a1, CdivGNegD1)
        a1.Mod(a1, pp.P)
        
        f1 = r1
        
        // Real proof for w=0
        r0, _ := rand.Int(rand.Reader, pp.Q)
        a0 = new(big.Int).Exp(pp.H, r0, pp.P) // g^0 × h^r0 = h^r0
        
        // Get challenge
        c := hashChallenge(pp.Q, C, a0, a1)
        
        // d0 = c - d1 mod q
        d0 = new(big.Int).Sub(c, d1)
        d0.Mod(d0, pp.Q)
        
        // f0 = r0 + d0 × r mod q
        f0 = new(big.Int).Mul(d0, r)
        f0.Add(f0, r0)
        f0.Mod(f0, pp.Q)
        
    } else {
        // Real proof for w=1, simulate w=0
        
        // Simulate w=0 branch
        d0, _ = rand.Int(rand.Reader, pp.Q)
        r0, _ := rand.Int(rand.Reader, pp.Q)
        
        // a0 = h^r0 × C^(-d0)
        hr0 := new(big.Int).Exp(pp.H, r0, pp.P)
        CNegD0 := new(big.Int).Exp(C, new(big.Int).Sub(pp.Q, d0), pp.P)
        a0 = new(big.Int).Mul(hr0, CNegD0)
        a0.Mod(a0, pp.P)
        
        f0 = r0
        
        // Real proof for w=1
        r1, _ := rand.Int(rand.Reader, pp.Q)
        
        // a1 = g × h^r1 for w=1
        g1 := new(big.Int).Set(pp.G)
        hr1 := new(big.Int).Exp(pp.H, r1, pp.P)
        a1 = new(big.Int).Mul(g1, hr1)
        a1.Mod(a1, pp.P)
        
        // Get challenge
        c := hashChallenge(pp.Q, C, a0, a1)
        
        // d1 = c - d0 mod q
        d1 = new(big.Int).Sub(c, d0)
        d1.Mod(d1, pp.Q)
        
        // f1 = r1 + d1 × r mod q
        f1 = new(big.Int).Mul(d1, r)
        f1.Add(f1, r1)
        f1.Mod(f1, pp.Q)
    }
    
    return &BinaryProof{
        A0: a0,
        A1: a1,
        D0: d0,
        D1: d1,
        F0: f0,
        F1: f1,
    }, nil
}

// VerifyBinary verifies a binary ZK proof
func (pp *PedersenParams) VerifyBinary(C *big.Int, proof *BinaryProof) bool {
    // Recompute challenge
    c := hashChallenge(pp.Q, C, proof.A0, proof.A1)
    
    // Check d0 + d1 = c mod q
    dSum := new(big.Int).Add(proof.D0, proof.D1)
    dSum.Mod(dSum, pp.Q)
    if dSum.Cmp(c) != 0 {
        return false
    }
    
    // Check a0: h^f0 × C^(-d0) should equal a0
    hf0 := new(big.Int).Exp(pp.H, proof.F0, pp.P)
    CNegD0 := new(big.Int).Exp(C, new(big.Int).Sub(pp.Q, proof.D0), pp.P)
    check0 := new(big.Int).Mul(hf0, CNegD0)
    check0.Mod(check0, pp.P)
    if check0.Cmp(proof.A0) != 0 {
        return false
    }
    
    // Check a1: g × h^f1 × (C/g)^(-d1) should equal a1
    gInv := new(big.Int).ModInverse(pp.G, pp.P)
    CdivG := new(big.Int).Mul(C, gInv)
    CdivG.Mod(CdivG, pp.P)
    
    g1 := new(big.Int).Set(pp.G)
    hf1 := new(big.Int).Exp(pp.H, proof.F1, pp.P)
    CdivGNegD1 := new(big.Int).Exp(CdivG, new(big.Int).Sub(pp.Q, proof.D1), pp.P)
    
    check1 := new(big.Int).Mul(g1, hf1)
    check1.Mul(check1, CdivGNegD1)
    check1.Mod(check1, pp.P)
    if check1.Cmp(proof.A1) != 0 {
        return false
    }
    
    return true
}

// ProveSumOne proves that commitments sum to 1
// Given C1, C2, ..., Ck where each Ci = g^wi × h^ri
// Prove: Σwi = 1
func (pp *PedersenParams) ProveSumOne(commitments []*Commitment) (*SumProof, error) {
    // Product of commitments = g^(Σwi) × h^(Σri)
    // If Σwi = 1, then product = g × h^(Σri)
    
    // Compute product
    product := big.NewInt(1)
    totalR := big.NewInt(0)
    
    for _, c := range commitments {
        product.Mul(product, c.C)
        product.Mod(product, pp.P)
        totalR.Add(totalR, c.R)
        totalR.Mod(totalR, pp.Q)
    }
    
    // Now prove product = g × h^totalR
    // This is a standard Schnorr proof
    
    // Random commitment
    k, _ := rand.Int(rand.Reader, pp.Q)
    a := new(big.Int).Exp(pp.H, k, pp.P) // h^k
    
    // Challenge
    c := hashChallenge(pp.Q, product, a, pp.G)
    
    // Response: s = k + c × totalR mod q
    s := new(big.Int).Mul(c, totalR)
    s.Add(s, k)
    s.Mod(s, pp.Q)
    
    return &SumProof{
        ProductCommitment: product,
        Challenge:         c,
        Response:          s,
    }, nil
}

// VerifySumOne verifies that sum of weights equals 1
func (pp *PedersenParams) VerifySumOne(commitments []*big.Int, proof *SumProof) bool {
    // Recompute product
    product := big.NewInt(1)
    for _, c := range commitments {
        product.Mul(product, c)
        product.Mod(product, pp.P)
    }
    
    // Check product matches
    if product.Cmp(proof.ProductCommitment) != 0 {
        return false
    }
    
    // Verify: h^s = a × (product/g)^c
    // i.e., h^s × g^c × product^(-c) = a
    
    hs := new(big.Int).Exp(pp.H, proof.Response, pp.P)
    gc := new(big.Int).Exp(pp.G, proof.Challenge, pp.P)
    
    gInv := new(big.Int).ModInverse(pp.G, pp.P)
    productDivG := new(big.Int).Mul(product, gInv)
    productDivG.Mod(productDivG, pp.P)
    
    // a should be h^s × (product/g)^(-c)
    // Simplified check
    
    // Recompute a from proof values
    negC := new(big.Int).Sub(pp.Q, proof.Challenge)
    productDivGNegC := new(big.Int).Exp(productDivG, negC, pp.P)
    computedA := new(big.Int).Mul(hs, productDivGNegC)
    computedA.Mod(computedA, pp.P)
    
    // Hash and verify
    expectedC := hashChallenge(pp.Q, product, computedA, pp.G)
    
    return expectedC.Cmp(proof.Challenge) == 0
}

// hashChallenge creates a Fiat-Shamir challenge
func hashChallenge(q *big.Int, values ...*big.Int) *big.Int {
    hasher := sha3.New256()
    for _, v := range values {
        hasher.Write(v.Bytes())
    }
    hashBytes := hasher.Sum(nil)
    c := new(big.Int).SetBytes(hashBytes)
    c.Mod(c, q)
    return c
}
```

---

# Step 4: SMDC Credential System

## 4.1 Algorithm

```
╔══════════════════════════════════════════════════════════════╗
║         SMDC - SELF-MASKING DENIABLE CREDENTIALS             ║
╠══════════════════════════════════════════════════════════════╣
║ CONCEPT:                                                     ║
║   - Voter creates k credential slots (typically k=5)         ║
║   - Exactly 1 slot is "real" (weight = 1)                    ║
║   - Other k-1 slots are "fake" (weight = 0)                  ║
║   - Nobody can distinguish which slot is real                ║
║   - Coercer can be shown any fake slot                       ║
╠══════════════════════════════════════════════════════════════╣
║ GENERATE(k):                                                 ║
║   1. real_index = SecureRandom(0, k-1)                       ║
║   2. FOR i = 0 TO k-1:                                       ║
║        IF i == real_index:                                   ║
║          weights[i] = 1                                      ║
║        ELSE:                                                 ║
║          weights[i] = 0                                      ║
║   3. FOR each weight w[i]:                                   ║
║        commitment[i] = Pedersen.Commit(w[i])                 ║
║        binaryProof[i] = ProveBinary(w[i])                    ║
║   4. sumProof = ProveSumOne(commitments)                     ║
║   5. RETURN (commitments, binaryProofs, sumProof, real_index)║
╠══════════════════════════════════════════════════════════════╣
║ VERIFY(commitments, binaryProofs, sumProof):                 ║
║   1. FOR each commitment[i]:                                 ║
║        IF NOT VerifyBinary(commitment[i], binaryProof[i]):   ║
║          RETURN false                                        ║
║   2. IF NOT VerifySumOne(commitments, sumProof):             ║
║        RETURN false                                          ║
║   3. RETURN true                                             ║
╚══════════════════════════════════════════════════════════════╝
```

## 4.2 Go Implementation

```go
// internal/smdc/types.go

package smdc

import (
    "math/big"
    
    "github.com/yourusername/covertvote/internal/crypto"
)

// SMDCCredential represents a voter's SMDC credential set
type SMDCCredential struct {
    VoterID      string                   // Voter identifier
    K            int                      // Number of slots
    Slots        []*CredentialSlot        // All k slots
    RealIndex    int                      // Which slot is real (SECRET!)
    SumProof     *crypto.SumProof         // Proof that weights sum to 1
}

// CredentialSlot represents one slot in SMDC
type CredentialSlot struct {
    Index       int                      // Slot index (0 to k-1)
    Weight      *big.Int                 // 0 or 1 (SECRET!)
    Randomness  *big.Int                 // Pedersen randomness (SECRET!)
    Commitment  *crypto.Commitment       // Public commitment
    BinaryProof *crypto.BinaryProof      // Proof that weight ∈ {0,1}
}

// PublicCredential is what gets published (no secrets)
type PublicCredential struct {
    VoterID      string
    K            int
    Commitments  []*big.Int
    BinaryProofs []*crypto.BinaryProof
    SumProof     *crypto.SumProof
}
```

```go
// internal/smdc/credential.go

package smdc

import (
    "crypto/rand"
    "errors"
    "math/big"
    
    "github.com/yourusername/covertvote/internal/crypto"
)

// SMDCGenerator generates SMDC credentials
type SMDCGenerator struct {
    PedersenParams *crypto.PedersenParams
    K              int // Number of slots
}

// NewSMDCGenerator creates a new SMDC generator
func NewSMDCGenerator(pp *crypto.PedersenParams, k int) *SMDCGenerator {
    return &SMDCGenerator{
        PedersenParams: pp,
        K:              k,
    }
}

// GenerateCredential generates a new SMDC credential for a voter
func (gen *SMDCGenerator) GenerateCredential(voterID string) (*SMDCCredential, error) {
    k := gen.K
    pp := gen.PedersenParams
    
    // Step 1: Randomly select the real slot index
    realIndexBig, err := rand.Int(rand.Reader, big.NewInt(int64(k)))
    if err != nil {
        return nil, err
    }
    realIndex := int(realIndexBig.Int64())
    
    // Step 2: Create all k slots
    slots := make([]*CredentialSlot, k)
    commitments := make([]*crypto.Commitment, k)
    
    for i := 0; i < k; i++ {
        var weight *big.Int
        
        if i == realIndex {
            weight = big.NewInt(1) // Real slot
        } else {
            weight = big.NewInt(0) // Fake slot
        }
        
        // Create Pedersen commitment
        commitment, err := pp.Commit(weight)
        if err != nil {
            return nil, err
        }
        
        // Create binary proof
        binaryProof, err := pp.ProveBinary(weight, commitment.R, commitment.C)
        if err != nil {
            return nil, err
        }
        
        slots[i] = &CredentialSlot{
            Index:       i,
            Weight:      weight,
            Randomness:  commitment.R,
            Commitment:  commitment,
            BinaryProof: binaryProof,
        }
        
        commitments[i] = commitment
    }
    
    // Step 3: Create sum proof (Σweights = 1)
    sumProof, err := pp.ProveSumOne(commitments)
    if err != nil {
        return nil, err
    }
    
    return &SMDCCredential{
        VoterID:   voterID,
        K:         k,
        Slots:     slots,
        RealIndex: realIndex,
        SumProof:  sumProof,
    }, nil
}

// GetPublicCredential extracts the public part of credential
func (cred *SMDCCredential) GetPublicCredential() *PublicCredential {
    commitments := make([]*big.Int, cred.K)
    binaryProofs := make([]*crypto.BinaryProof, cred.K)
    
    for i, slot := range cred.Slots {
        commitments[i] = slot.Commitment.C
        binaryProofs[i] = slot.BinaryProof
    }
    
    return &PublicCredential{
        VoterID:      cred.VoterID,
        K:            cred.K,
        Commitments:  commitments,
        BinaryProofs: binaryProofs,
        SumProof:     cred.SumProof,
    }
}

// GetRealSlot returns the real slot (ONLY for internal use by voter)
func (cred *SMDCCredential) GetRealSlot() *CredentialSlot {
    return cred.Slots[cred.RealIndex]
}

// GetFakeSlot returns a fake slot (for coercion scenario)
func (cred *SMDCCredential) GetFakeSlot(index int) (*CredentialSlot, error) {
    if index == cred.RealIndex {
        return nil, errors.New("cannot return real slot as fake")
    }
    if index < 0 || index >= cred.K {
        return nil, errors.New("invalid slot index")
    }
    return cred.Slots[index], nil
}

// VerifyCredential verifies a public credential
func (gen *SMDCGenerator) VerifyCredential(pub *PublicCredential) bool {
    pp := gen.PedersenParams
    
    // Verify each binary proof
    for i := 0; i < pub.K; i++ {
        if !pp.VerifyBinary(pub.Commitments[i], pub.BinaryProofs[i]) {
            return false
        }
    }
    
    // Verify sum proof
    if !pp.VerifySumOne(pub.Commitments, pub.SumProof) {
        return false
    }
    
    return true
}
```

## 4.3 Test File

```go
// internal/smdc/credential_test.go

package smdc

import (
    "testing"
    
    "github.com/yourusername/covertvote/internal/crypto"
)

func TestSMDCGeneration(t *testing.T) {
    // Setup
    pp, _ := crypto.GeneratePedersenParams(512)
    gen := NewSMDCGenerator(pp, 5) // k=5 slots
    
    // Generate credential
    cred, err := gen.GenerateCredential("voter123")
    if err != nil {
        t.Fatalf("Generation failed: %v", err)
    }
    
    // Check structure
    if cred.K != 5 {
        t.Errorf("Expected 5 slots, got %d", cred.K)
    }
    
    // Count weights
    totalWeight := int64(0)
    realCount := 0
    for _, slot := range cred.Slots {
        totalWeight += slot.Weight.Int64()
        if slot.Weight.Int64() == 1 {
            realCount++
        }
    }
    
    if totalWeight != 1 {
        t.Errorf("Total weight should be 1, got %d", totalWeight)
    }
    
    if realCount != 1 {
        t.Errorf("Should have exactly 1 real slot, got %d", realCount)
    }
}

func TestSMDCVerification(t *testing.T) {
    pp, _ := crypto.GeneratePedersenParams(512)
    gen := NewSMDCGenerator(pp, 5)
    
    cred, _ := gen.GenerateCredential("voter456")
    pub := cred.GetPublicCredential()
    
    // Should verify
    if !gen.VerifyCredential(pub) {
        t.Error("Valid credential should verify")
    }
}

func TestSMDCCoercionResistance(t *testing.T) {
    pp, _ := crypto.GeneratePedersenParams(512)
    gen := NewSMDCGenerator(pp, 5)
    
    cred, _ := gen.GenerateCredential("voter789")
    
    // Get real slot
    realSlot := cred.GetRealSlot()
    
    // Get a fake slot (any index except real)
    fakeIndex := (cred.RealIndex + 1) % cred.K
    fakeSlot, _ := cred.GetFakeSlot(fakeIndex)
    
    // Verify they're different
    if realSlot.Weight.Cmp(fakeSlot.Weight) == 0 {
        t.Error("Real and fake slots should have different weights")
    }
    
    // But both should verify as valid binary!
    if !pp.VerifyBinary(realSlot.Commitment.C, realSlot.BinaryProof) {
        t.Error("Real slot binary proof failed")
    }
    
    if !pp.VerifyBinary(fakeSlot.Commitment.C, fakeSlot.BinaryProof) {
        t.Error("Fake slot binary proof failed")
    }
}
```

---

# Step 5: Ring Signature

## 5.1 Algorithm

```
╔══════════════════════════════════════════════════════════════╗
║              LINKABLE RING SIGNATURE                         ║
╠══════════════════════════════════════════════════════════════╣
║ PURPOSE:                                                     ║
║   - Sign anonymously within a group (ring)                   ║
║   - Linkable: Same key produces same tag (detect double-vote)║
║   - Anonymous: Cannot identify actual signer                 ║
╠══════════════════════════════════════════════════════════════╣
║ SETUP:                                                       ║
║   - Ring R = {PK1, PK2, ..., PKn} (all voter public keys)    ║
║   - Signer has (pk_s, sk_s) where pk_s ∈ R                   ║
╠══════════════════════════════════════════════════════════════╣
║ SIGN(message, sk_s, R):                                      ║
║   1. Compute key image: I = sk_s × H_p(pk_s)                 ║
║   2. FOR each member i in R:                                 ║
║        IF i == signer_index:                                 ║
║          Generate real response                              ║
║        ELSE:                                                 ║
║          Generate simulated response                         ║
║   3. Create challenge ring                                   ║
║   4. Return σ = (I, c1, r1, r2, ..., rn)                     ║
╠══════════════════════════════════════════════════════════════╣
║ VERIFY(message, σ, R):                                       ║
║   1. Reconstruct challenge ring                              ║
║   2. Check ring closes properly                              ║
║   3. Return true/false                                       ║
╠══════════════════════════════════════════════════════════════╣
║ LINK(σ1, σ2):                                                ║
║   - IF I1 == I2: Same signer (double-vote!)                  ║
║   - ELSE: Different signers                                  ║
╚══════════════════════════════════════════════════════════════╝
```

## 5.2 Go Implementation

```go
// internal/crypto/ring_signature.go

package crypto

import (
    "crypto/rand"
    "errors"
    "math/big"
    
    "golang.org/x/crypto/sha3"
)

// RingParams holds ring signature parameters
type RingParams struct {
    P *big.Int // Prime modulus
    Q *big.Int // Order
    G *big.Int // Generator
}

// RingKeyPair represents a member's key pair
type RingKeyPair struct {
    PublicKey  *big.Int // pk = g^sk mod p
    PrivateKey *big.Int // sk
}

// RingSignature represents a linkable ring signature
type RingSignature struct {
    KeyImage   *big.Int   // I = sk × H(pk) - for linking
    Challenge  *big.Int   // c0
    Responses  []*big.Int // r0, r1, ..., rn-1
}

// GenerateRingParams generates ring signature parameters
func GenerateRingParams(bits int) (*RingParams, error) {
    pp, err := GeneratePedersenParams(bits)
    if err != nil {
        return nil, err
    }
    return &RingParams{
        P: pp.P,
        Q: pp.Q,
        G: pp.G,
    }, nil
}

// GenerateRingKeyPair generates a key pair for ring member
func (rp *RingParams) GenerateRingKeyPair() (*RingKeyPair, error) {
    // sk = random in Zq
    sk, err := rand.Int(rand.Reader, rp.Q)
    if err != nil {
        return nil, err
    }
    
    // pk = g^sk mod p
    pk := new(big.Int).Exp(rp.G, sk, rp.P)
    
    return &RingKeyPair{
        PublicKey:  pk,
        PrivateKey: sk,
    }, nil
}

// hashToPoint hashes a public key to a group element
func (rp *RingParams) hashToPoint(pk *big.Int) *big.Int {
    hasher := sha3.New256()
    hasher.Write(pk.Bytes())
    hashBytes := hasher.Sum(nil)
    
    h := new(big.Int).SetBytes(hashBytes)
    h.Mod(h, rp.P)
    
    // Ensure it's in the subgroup
    pMinus1 := new(big.Int).Sub(rp.P, big.NewInt(1))
    exp := new(big.Int).Div(pMinus1, rp.Q)
    h.Exp(h, exp, rp.P)
    
    return h
}

// Sign creates a linkable ring signature
func (rp *RingParams) Sign(message []byte, signerKey *RingKeyPair, ring []*big.Int, signerIndex int) (*RingSignature, error) {
    n := len(ring)
    if signerIndex < 0 || signerIndex >= n {
        return nil, errors.New("invalid signer index")
    }
    
    // Step 1: Compute key image I = sk × H(pk)
    hp := rp.hashToPoint(signerKey.PublicKey)
    keyImage := new(big.Int).Exp(hp, signerKey.PrivateKey, rp.P)
    
    // Step 2: Initialize arrays
    challenges := make([]*big.Int, n)
    responses := make([]*big.Int, n)
    
    // Step 3: Generate random commitment for signer
    alpha, _ := rand.Int(rand.Reader, rp.Q)
    
    // L_s = g^alpha
    Ls := new(big.Int).Exp(rp.G, alpha, rp.P)
    // R_s = H(pk_s)^alpha
    Rs := new(big.Int).Exp(hp, alpha, rp.P)
    
    // Step 4: Compute starting challenge
    challenges[(signerIndex+1)%n] = rp.hashRing(message, Ls, Rs)
    
    // Step 5: Fill in simulated responses
    for i := 1; i < n; i++ {
        idx := (signerIndex + i) % n
        nextIdx := (idx + 1) % n
        
        // Random response
        responses[idx], _ = rand.Int(rand.Reader, rp.Q)
        
        // L_i = g^r_i × pk_i^c_i
        gri := new(big.Int).Exp(rp.G, responses[idx], rp.P)
        pkci := new(big.Int).Exp(ring[idx], challenges[idx], rp.P)
        Li := new(big.Int).Mul(gri, pkci)
        Li.Mod(Li, rp.P)
        
        // R_i = H(pk_i)^r_i × I^c_i
        hpi := rp.hashToPoint(ring[idx])
        hpri := new(big.Int).Exp(hpi, responses[idx], rp.P)
        Ici := new(big.Int).Exp(keyImage, challenges[idx], rp.P)
        Ri := new(big.Int).Mul(hpri, Ici)
        Ri.Mod(Ri, rp.P)
        
        // Next challenge
        if nextIdx != signerIndex {
            challenges[nextIdx] = rp.hashRing(message, Li, Ri)
        }
    }
    
    // Step 6: Close the ring - compute signer's response
    // r_s = alpha - c_s × sk mod q
    responses[signerIndex] = new(big.Int).Mul(challenges[signerIndex], signerKey.PrivateKey)
    responses[signerIndex].Sub(alpha, responses[signerIndex])
    responses[signerIndex].Mod(responses[signerIndex], rp.Q)
    
    return &RingSignature{
        KeyImage:  keyImage,
        Challenge: challenges[0],
        Responses: responses,
    }, nil
}

// Verify verifies a ring signature
func (rp *RingParams) Verify(message []byte, sig *RingSignature, ring []*big.Int) bool {
    n := len(ring)
    if len(sig.Responses) != n {
        return false
    }
    
    currentChallenge := sig.Challenge
    
    for i := 0; i < n; i++ {
        // L_i = g^r_i × pk_i^c_i
        gri := new(big.Int).Exp(rp.G, sig.Responses[i], rp.P)
        pkci := new(big.Int).Exp(ring[i], currentChallenge, rp.P)
        Li := new(big.Int).Mul(gri, pkci)
        Li.Mod(Li, rp.P)
        
        // R_i = H(pk_i)^r_i × I^c_i
        hpi := rp.hashToPoint(ring[i])
        hpri := new(big.Int).Exp(hpi, sig.Responses[i], rp.P)
        Ici := new(big.Int).Exp(sig.KeyImage, currentChallenge, rp.P)
        Ri := new(big.Int).Mul(hpri, Ici)
        Ri.Mod(Ri, rp.P)
        
        // Next challenge
        currentChallenge = rp.hashRing(message, Li, Ri)
    }
    
    // Ring should close: final challenge should equal initial
    return currentChallenge.Cmp(sig.Challenge) == 0
}

// Link checks if two signatures are from the same signer
func Link(sig1, sig2 *RingSignature) bool {
    return sig1.KeyImage.Cmp(sig2.KeyImage) == 0
}

// hashRing creates a hash for ring computation
func (rp *RingParams) hashRing(message []byte, L, R *big.Int) *big.Int {
    hasher := sha3.New256()
    hasher.Write(message)
    hasher.Write(L.Bytes())
    hasher.Write(R.Bytes())
    hashBytes := hasher.Sum(nil)
    
    c := new(big.Int).SetBytes(hashBytes)
    c.Mod(c, rp.Q)
    return c
}
```

---

# Step 6: SA² Aggregation

## 6.1 Algorithm

```
╔══════════════════════════════════════════════════════════════╗
║        SA² - SAMPLABLE ANONYMOUS AGGREGATION                 ║
╠══════════════════════════════════════════════════════════════╣
║ CONCEPT:                                                     ║
║   - 2-server model: Server A and Server B                    ║
║   - Each vote is split into 2 shares                         ║
║   - Neither server can see individual votes                  ║
║   - Only combined result reveals tally                       ║
╠══════════════════════════════════════════════════════════════╣
║ SPLIT(encrypted_vote):                                       ║
║   1. Generate random mask m                                  ║
║   2. share_A = encrypted_vote × E(m)                         ║
║   3. share_B = E(-m)                                         ║
║   4. Note: share_A × share_B = encrypted_vote                ║
╠══════════════════════════════════════════════════════════════╣
║ AGGREGATE_A(shares_A[]):                                     ║
║   1. agg_A = Π(shares_A[i]) for all i                        ║
║   2. Send agg_A to combiner                                  ║
╠══════════════════════════════════════════════════════════════╣
║ AGGREGATE_B(shares_B[]):                                     ║
║   1. agg_B = Π(shares_B[i]) for all i                        ║
║   2. Send agg_B to combiner                                  ║
╠══════════════════════════════════════════════════════════════╣
║ COMBINE(agg_A, agg_B):                                       ║
║   1. final = agg_A × agg_B                                   ║
║   2. Note: All masks cancel out!                             ║
║   3. final = E(Σ votes) = E(tally)                           ║
╚══════════════════════════════════════════════════════════════╝
```

## 6.2 Go Implementation

```go
// internal/sa2/types.go

package sa2

import (
    "math/big"
)

// VoteShare represents a split vote share
type VoteShare struct {
    VoterID string
    ShareA  *big.Int // For Server A
    ShareB  *big.Int // For Server B
}

// AggregatedShare represents aggregated shares from one server
type AggregatedShare struct {
    ServerID string
    Value    *big.Int
    Count    int
}

// CombinedResult represents the final combined tally
type CombinedResult struct {
    EncryptedTally *big.Int
    TotalVotes     int
}
```

```go
// internal/sa2/share.go

package sa2

import (
    "crypto/rand"
    "math/big"
    
    "github.com/yourusername/covertvote/internal/crypto"
)

// ShareGenerator handles vote splitting
type ShareGenerator struct {
    PaillierPK *crypto.PaillierPublicKey
}

// NewShareGenerator creates a new share generator
func NewShareGenerator(pk *crypto.PaillierPublicKey) *ShareGenerator {
    return &ShareGenerator{
        PaillierPK: pk,
    }
}

// SplitVote splits an encrypted vote into two shares
// Input: E(w × v) - encrypted weighted vote
// Output: (share_A, share_B) where share_A × share_B = E(w × v)
func (sg *ShareGenerator) SplitVote(encryptedVote *big.Int, voterID string) (*VoteShare, error) {
    pk := sg.PaillierPK
    
    // Generate random mask m in range [0, n)
    mask, err := rand.Int(rand.Reader, pk.N)
    if err != nil {
        return nil, err
    }
    
    // Encrypt the mask: E(m)
    encMask, err := pk.Encrypt(mask)
    if err != nil {
        return nil, err
    }
    
    // Encrypt negative mask: E(-m) = E(n - m) since we work mod n
    negativeMask := new(big.Int).Sub(pk.N, mask)
    encNegMask, err := pk.Encrypt(negativeMask)
    if err != nil {
        return nil, err
    }
    
    // share_A = E(w×v) × E(m) = E(w×v + m)
    shareA := pk.Add(encryptedVote, encMask)
    
    // share_B = E(-m)
    shareB := encNegMask
    
    // Verification: shareA × shareB = E(w×v + m) × E(-m) = E(w×v + m - m) = E(w×v)
    
    return &VoteShare{
        VoterID: voterID,
        ShareA:  shareA,
        ShareB:  shareB,
    }, nil
}

// RecombineShares recombines shares (for verification only)
func (sg *ShareGenerator) RecombineShares(shareA, shareB *big.Int) *big.Int {
    return sg.PaillierPK.Add(shareA, shareB)
}
```

```go
// internal/sa2/aggregation.go

package sa2

import (
    "math/big"
    "sync"
    
    "github.com/yourusername/covertvote/internal/crypto"
)

// Server represents a SA² aggregation server
type Server struct {
    ID         string
    PaillierPK *crypto.PaillierPublicKey
    shares     []*big.Int
    mutex      sync.Mutex
}

// NewServer creates a new SA² server
func NewServer(id string, pk *crypto.PaillierPublicKey) *Server {
    return &Server{
        ID:         id,
        PaillierPK: pk,
        shares:     make([]*big.Int, 0),
    }
}

// ReceiveShare receives a vote share
func (s *Server) ReceiveShare(share *big.Int) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    s.shares = append(s.shares, share)
}

// ReceiveShares receives multiple vote shares
func (s *Server) ReceiveShares(shares []*big.Int) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    s.shares = append(s.shares, shares...)
}

// Aggregate computes the homomorphic sum of all shares
func (s *Server) Aggregate() *AggregatedShare {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    if len(s.shares) == 0 {
        // Return encryption of 0
        zero, _ := s.PaillierPK.Encrypt(big.NewInt(0))
        return &AggregatedShare{
            ServerID: s.ID,
            Value:    zero,
            Count:    0,
        }
    }
    
    // Homomorphic sum: Π(shares[i]) = E(Σ values[i])
    result := s.shares[0]
    for i := 1; i < len(s.shares); i++ {
        result = s.PaillierPK.Add(result, s.shares[i])
    }
    
    return &AggregatedShare{
        ServerID: s.ID,
        Value:    result,
        Count:    len(s.shares),
    }
}

// Reset clears the server's shares
func (s *Server) Reset() {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    s.shares = make([]*big.Int, 0)
}

// Combiner combines aggregated shares from both servers
type Combiner struct {
    PaillierPK *crypto.PaillierPublicKey
}

// NewCombiner creates a new combiner
func NewCombiner(pk *crypto.PaillierPublicKey) *Combiner {
    return &Combiner{
        PaillierPK: pk,
    }
}

// Combine combines the aggregated shares from Server A and Server B
func (c *Combiner) Combine(aggA, aggB *AggregatedShare) (*CombinedResult, error) {
    // Combined = aggA.Value × aggB.Value
    // = E(Σ(w×v + m)) × E(Σ(-m))
    // = E(Σ(w×v + m - m))
    // = E(Σ(w×v))
    // All masks cancel out!
    
    combined := c.PaillierPK.Add(aggA.Value, aggB.Value)
    
    return &CombinedResult{
        EncryptedTally: combined,
        TotalVotes:     aggA.Count, // Should equal aggB.Count
    }, nil
}
```

```go
// internal/sa2/server.go

package sa2

import (
    "github.com/yourusername/covertvote/internal/crypto"
)

// SA2System represents the complete SA² system
type SA2System struct {
    ServerA    *Server
    ServerB    *Server
    Combiner   *Combiner
    Generator  *ShareGenerator
    PaillierPK *crypto.PaillierPublicKey
}

// NewSA2System creates a new SA² system
func NewSA2System(pk *crypto.PaillierPublicKey) *SA2System {
    return &SA2System{
        ServerA:    NewServer("ServerA", pk),
        ServerB:    NewServer("ServerB", pk),
        Combiner:   NewCombiner(pk),
        Generator:  NewShareGenerator(pk),
        PaillierPK: pk,
    }
}

// ProcessVote processes a single encrypted vote
func (sys *SA2System) ProcessVote(encryptedVote *big.Int, voterID string) error {
    // Split the vote
    share, err := sys.Generator.SplitVote(encryptedVote, voterID)
    if err != nil {
        return err
    }
    
    // Send to servers
    sys.ServerA.ReceiveShare(share.ShareA)
    sys.ServerB.ReceiveShare(share.ShareB)
    
    return nil
}

// ProcessVotes processes multiple votes in batch
func (sys *SA2System) ProcessVotes(votes []*EncryptedVote) error {
    sharesA := make([]*big.Int, len(votes))
    sharesB := make([]*big.Int, len(votes))
    
    for i, vote := range votes {
        share, err := sys.Generator.SplitVote(vote.Ciphertext, vote.VoterID)
        if err != nil {
            return err
        }
        sharesA[i] = share.ShareA
        sharesB[i] = share.ShareB
    }
    
    sys.ServerA.ReceiveShares(sharesA)
    sys.ServerB.ReceiveShares(sharesB)
    
    return nil
}

// GetEncryptedTally returns the combined encrypted tally
func (sys *SA2System) GetEncryptedTally() (*CombinedResult, error) {
    aggA := sys.ServerA.Aggregate()
    aggB := sys.ServerB.Aggregate()
    
    return sys.Combiner.Combine(aggA, aggB)
}

// Reset resets the system for a new election
func (sys *SA2System) Reset() {
    sys.ServerA.Reset()
    sys.ServerB.Reset()
}

// EncryptedVote represents an encrypted vote
type EncryptedVote struct {
    VoterID    string
    Ciphertext *big.Int
}
```

---

# Step 7: Biometric (Fingerprint)

## 7.1 Algorithm

```
╔══════════════════════════════════════════════════════════════╗
║              FINGERPRINT PROCESSING                          ║
╠══════════════════════════════════════════════════════════════╣
║ NOTE: We don't store actual fingerprints!                    ║
║       We only store and compare hashes.                      ║
╠══════════════════════════════════════════════════════════════╣
║ REGISTER(fingerprint_image):                                 ║
║   1. Extract features (minutiae points)                      ║
║   2. Normalize features                                      ║
║   3. Hash: credential = SHA3-256(features)                   ║
║   4. Store: credential (NOT the fingerprint!)                ║
╠══════════════════════════════════════════════════════════════╣
║ AUTHENTICATE(fingerprint_image, stored_credential):          ║
║   1. Extract features from new image                         ║
║   2. Normalize features                                      ║
║   3. Hash: new_credential = SHA3-256(features)               ║
║   4. Compare: new_credential == stored_credential?           ║
╠══════════════════════════════════════════════════════════════╣
║ LIVENESS CHECK (simplified):                                 ║
║   1. Check image quality                                     ║
║   2. Check for known fake patterns                           ║
║   3. In production: use dedicated liveness SDK               ║
╚══════════════════════════════════════════════════════════════╝
```

## 7.2 Go Implementation

```go
// internal/biometric/fingerprint.go

package biometric

import (
    "crypto/subtle"
    "encoding/hex"
    "errors"
    
    "golang.org/x/crypto/sha3"
)

// FingerprintCredential represents a hashed fingerprint
type FingerprintCredential struct {
    Hash      []byte // SHA3-256 hash of fingerprint features
    HashHex   string // Hex representation
    CreatedAt int64  // Timestamp
}

// FingerprintProcessor handles fingerprint operations
type FingerprintProcessor struct {
    // In production, add configuration for feature extraction
}

// NewFingerprintProcessor creates a new processor
func NewFingerprintProcessor() *FingerprintProcessor {
    return &FingerprintProcessor{}
}

// ProcessFingerprint processes raw fingerprint data and creates credential
// In production, 'data' would be processed through feature extraction
// For thesis prototype, we directly hash the input
func (fp *FingerprintProcessor) ProcessFingerprint(data []byte) (*FingerprintCredential, error) {
    if len(data) == 0 {
        return nil, errors.New("empty fingerprint data")
    }
    
    // Step 1: Feature extraction (simplified for prototype)
    // In production, use a proper minutiae extraction library
    features := fp.extractFeatures(data)
    
    // Step 2: Normalize features
    normalized := fp.normalizeFeatures(features)
    
    // Step 3: Hash with SHA3-256
    hasher := sha3.New256()
    hasher.Write(normalized)
    hash := hasher.Sum(nil)
    
    return &FingerprintCredential{
        Hash:    hash,
        HashHex: hex.EncodeToString(hash),
    }, nil
}

// extractFeatures extracts features from fingerprint image
// SIMPLIFIED for prototype - in production use proper library
func (fp *FingerprintProcessor) extractFeatures(data []byte) []byte {
    // In production:
    // 1. Binarize image
    // 2. Thin ridges
    // 3. Extract minutiae points (ridge endings, bifurcations)
    // 4. Create feature vector
    
    // For prototype, we just use the raw data
    // In real implementation, use libraries like:
    // - SourceAFIS (open source)
    // - Neurotechnology SDK
    return data
}

// normalizeFeatures normalizes feature vectors
func (fp *FingerprintProcessor) normalizeFeatures(features []byte) []byte {
    // In production: 
    // - Sort minutiae by position
    // - Normalize coordinates
    // - Apply consistent orientation
    
    // For prototype, return as-is
    return features
}

// Verify verifies a fingerprint against stored credential
func (fp *FingerprintProcessor) Verify(data []byte, stored *FingerprintCredential) bool {
    newCred, err := fp.ProcessFingerprint(data)
    if err != nil {
        return false
    }
    
    // Constant-time comparison to prevent timing attacks
    return subtle.ConstantTimeCompare(newCred.Hash, stored.Hash) == 1
}

// LivenessCheck performs basic liveness detection
// In production, use dedicated liveness SDK
func (fp *FingerprintProcessor) LivenessCheck(imageData []byte) (bool, float64) {
    // Simplified checks for prototype
    // In production, use:
    // - BioID
    // - FaceTec
    // - iProov
    
    // Basic checks:
    if len(imageData) < 1000 {
        return false, 0.0 // Too small, likely fake
    }
    
    // Check for known fake patterns (simplified)
    // In production, use ML model
    
    // Return confidence score
    confidence := 0.95 // Placeholder
    isLive := confidence > 0.90
    
    return isLive, confidence
}
```

```go
// internal/biometric/hash.go

package biometric

import (
    "golang.org/x/crypto/sha3"
)

// GenerateVoterID generates a unique voter ID from biometric + NID
func GenerateVoterID(fingerprintHash []byte, nidNumber string, salt []byte) string {
    hasher := sha3.New256()
    hasher.Write(fingerprintHash)
    hasher.Write([]byte(nidNumber))
    hasher.Write(salt)
    
    hash := hasher.Sum(nil)
    return hex.EncodeToString(hash[:16]) // First 16 bytes = 32 hex chars
}

// HashNID creates a hash of NID for storage (not storing raw NID)
func HashNID(nidNumber string) []byte {
    hasher := sha3.New256()
    hasher.Write([]byte(nidNumber))
    return hasher.Sum(nil)
}
```

---

# Step 8: Voter Registration

## 8.1 Go Implementation

```go
// internal/voter/types.go

package voter

import (
    "time"
    
    "github.com/yourusername/covertvote/internal/biometric"
    "github.com/yourusername/covertvote/internal/smdc"
)

// Voter represents a registered voter
type Voter struct {
    VoterID           string                           // Unique voter ID
    FingerprintCred   *biometric.FingerprintCredential // Fingerprint hash
    NIDHash           []byte                           // Hashed NID
    SMDCCredential    *smdc.SMDCCredential             // SMDC credential
    PublicCredential  *smdc.PublicCredential           // Public part
    RingPublicKey     *big.Int                         // For ring signature
    MerkleLeaf        []byte                           // Leaf in Merkle tree
    RegisteredAt      time.Time
    IsEligible        bool
}

// RegistrationRequest represents a registration request
type RegistrationRequest struct {
    NIDNumber         string
    FingerprintData   []byte
    FingerprintImage  []byte // For liveness check
}

// RegistrationResult represents registration outcome
type RegistrationResult struct {
    Success     bool
    VoterID     string
    Error       string
    MerkleProof [][]byte
}
```

```go
// internal/voter/registration.go

package voter

import (
    "crypto/rand"
    "errors"
    "sync"
    "time"
    
    "github.com/yourusername/covertvote/internal/biometric"
    "github.com/yourusername/covertvote/internal/crypto"
    "github.com/yourusername/covertvote/internal/smdc"
)

// RegistrationService handles voter registration
type RegistrationService struct {
    fpProcessor    *biometric.FingerprintProcessor
    smdcGenerator  *smdc.SMDCGenerator
    ringParams     *crypto.RingParams
    voterStore     map[string]*Voter
    merkleTree     *MerkleTree
    mutex          sync.RWMutex
    salt           []byte
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
    pp *crypto.PedersenParams,
    rp *crypto.RingParams,
    k int,
) (*RegistrationService, error) {
    salt := make([]byte, 32)
    if _, err := rand.Read(salt); err != nil {
        return nil, err
    }
    
    return &RegistrationService{
        fpProcessor:   biometric.NewFingerprintProcessor(),
        smdcGenerator: smdc.NewSMDCGenerator(pp, k),
        ringParams:    rp,
        voterStore:    make(map[string]*Voter),
        merkleTree:    NewMerkleTree(),
        salt:          salt,
    }, nil
}

// Register registers a new voter
func (rs *RegistrationService) Register(req *RegistrationRequest) (*RegistrationResult, error) {
    rs.mutex.Lock()
    defer rs.mutex.Unlock()
    
    // Step 1: Liveness check
    isLive, confidence := rs.fpProcessor.LivenessCheck(req.FingerprintImage)
    if !isLive {
        return &RegistrationResult{
            Success: false,
            Error:   fmt.Sprintf("liveness check failed (confidence: %.2f)", confidence),
        }, nil
    }
    
    // Step 2: Process fingerprint
    fpCred, err := rs.fpProcessor.ProcessFingerprint(req.FingerprintData)
    if err != nil {
        return nil, err
    }
    
    // Step 3: Check if already registered
    voterID := biometric.GenerateVoterID(fpCred.Hash, req.NIDNumber, rs.salt)
    if _, exists := rs.voterStore[voterID]; exists {
        return &RegistrationResult{
            Success: false,
            Error:   "voter already registered",
        }, nil
    }
    
    // Step 4: Validate NID (simplified - in production call external API)
    if !rs.validateNID(req.NIDNumber) {
        return &RegistrationResult{
            Success: false,
            Error:   "invalid NID",
        }, nil
    }
    
    // Step 5: Generate SMDC credential
    smdcCred, err := rs.smdcGenerator.GenerateCredential(voterID)
    if err != nil {
        return nil, err
    }
    
    // Step 6: Generate ring signature key pair
    ringKey, err := rs.ringParams.GenerateRingKeyPair()
    if err != nil {
        return nil, err
    }
    
    // Step 7: Create voter record
    voter := &Voter{
        VoterID:          voterID,
        FingerprintCred:  fpCred,
        NIDHash:          biometric.HashNID(req.NIDNumber),
        SMDCCredential:   smdcCred,
        PublicCredential: smdcCred.GetPublicCredential(),
        RingPublicKey:    ringKey.PublicKey,
        RegisteredAt:     time.Now(),
        IsEligible:       true,
    }
    
    // Step 8: Add to Merkle tree
    leaf := rs.merkleTree.AddLeaf(voterID, ringKey.PublicKey)
    voter.MerkleLeaf = leaf
    
    // Step 9: Store voter
    rs.voterStore[voterID] = voter
    
    // Step 10: Return result
    return &RegistrationResult{
        Success: true,
        VoterID: voterID,
    }, nil
}

// validateNID validates NID format and eligibility
func (rs *RegistrationService) validateNID(nid string) bool {
    // Simplified validation
    // In production: call government API
    if len(nid) < 10 {
        return false
    }
    return true
}

// GetVoter returns a voter by ID
func (rs *RegistrationService) GetVoter(voterID string) (*Voter, error) {
    rs.mutex.RLock()
    defer rs.mutex.RUnlock()
    
    voter, exists := rs.voterStore[voterID]
    if !exists {
        return nil, errors.New("voter not found")
    }
    return voter, nil
}

// GetMerkleRoot returns the current Merkle root
func (rs *RegistrationService) GetMerkleRoot() []byte {
    return rs.merkleTree.GetRoot()
}

// GetMerkleProof returns Merkle proof for a voter
func (rs *RegistrationService) GetMerkleProof(voterID string) ([][]byte, error) {
    return rs.merkleTree.GetProof(voterID)
}

// GetAllPublicKeys returns all voter public keys (for ring signature)
func (rs *RegistrationService) GetAllPublicKeys() []*big.Int {
    rs.mutex.RLock()
    defer rs.mutex.RUnlock()
    
    keys := make([]*big.Int, 0, len(rs.voterStore))
    for _, voter := range rs.voterStore {
        keys = append(keys, voter.RingPublicKey)
    }
    return keys
}
```

```go
// internal/voter/merkle.go

package voter

import (
    "golang.org/x/crypto/sha3"
    "math/big"
    "sync"
)

// MerkleTree represents a Merkle tree for voter eligibility
type MerkleTree struct {
    leaves  [][]byte
    voterMap map[string]int // voterID -> leaf index
    mutex   sync.RWMutex
}

// NewMerkleTree creates a new Merkle tree
func NewMerkleTree() *MerkleTree {
    return &MerkleTree{
        leaves:   make([][]byte, 0),
        voterMap: make(map[string]int),
    }
}

// AddLeaf adds a voter to the Merkle tree
func (mt *MerkleTree) AddLeaf(voterID string, publicKey *big.Int) []byte {
    mt.mutex.Lock()
    defer mt.mutex.Unlock()
    
    // Create leaf: hash(voterID || publicKey)
    hasher := sha3.New256()
    hasher.Write([]byte(voterID))
    hasher.Write(publicKey.Bytes())
    leaf := hasher.Sum(nil)
    
    mt.voterMap[voterID] = len(mt.leaves)
    mt.leaves = append(mt.leaves, leaf)
    
    return leaf
}

// GetRoot computes the Merkle root
func (mt *MerkleTree) GetRoot() []byte {
    mt.mutex.RLock()
    defer mt.mutex.RUnlock()
    
    if len(mt.leaves) == 0 {
        return make([]byte, 32)
    }
    
    return mt.computeRoot(mt.leaves)
}

// computeRoot recursively computes Merkle root
func (mt *MerkleTree) computeRoot(nodes [][]byte) []byte {
    if len(nodes) == 1 {
        return nodes[0]
    }
    
    // Pad if odd number
    if len(nodes)%2 == 1 {
        nodes = append(nodes, nodes[len(nodes)-1])
    }
    
    var nextLevel [][]byte
    for i := 0; i < len(nodes); i += 2 {
        hasher := sha3.New256()
        hasher.Write(nodes[i])
        hasher.Write(nodes[i+1])
        nextLevel = append(nextLevel, hasher.Sum(nil))
    }
    
    return mt.computeRoot(nextLevel)
}

// GetProof returns the Merkle proof for a voter
func (mt *MerkleTree) GetProof(voterID string) ([][]byte, error) {
    mt.mutex.RLock()
    defer mt.mutex.RUnlock()
    
    index, exists := mt.voterMap[voterID]
    if !exists {
        return nil, errors.New("voter not in tree")
    }
    
    return mt.computeProof(index), nil
}

// computeProof computes Merkle proof for given index
func (mt *MerkleTree) computeProof(index int) [][]byte {
    proof := make([][]byte, 0)
    nodes := mt.leaves
    
    for len(nodes) > 1 {
        // Pad if odd
        if len(nodes)%2 == 1 {
            nodes = append(nodes, nodes[len(nodes)-1])
        }
        
        // Add sibling to proof
        if index%2 == 0 {
            proof = append(proof, nodes[index+1])
        } else {
            proof = append(proof, nodes[index-1])
        }
        
        // Move to next level
        var nextLevel [][]byte
        for i := 0; i < len(nodes); i += 2 {
            hasher := sha3.New256()
            hasher.Write(nodes[i])
            hasher.Write(nodes[i+1])
            nextLevel = append(nextLevel, hasher.Sum(nil))
        }
        
        nodes = nextLevel
        index = index / 2
    }
    
    return proof
}

// VerifyProof verifies a Merkle proof
func VerifyMerkleProof(leaf []byte, proof [][]byte, root []byte, index int) bool {
    current := leaf
    
    for _, sibling := range proof {
        hasher := sha3.New256()
        if index%2 == 0 {
            hasher.Write(current)
            hasher.Write(sibling)
        } else {
            hasher.Write(sibling)
            hasher.Write(current)
        }
        current = hasher.Sum(nil)
        index = index / 2
    }
    
    return subtle.ConstantTimeCompare(current, root) == 1
}
```

---

# Step 9: Vote Casting

## 9.1 Go Implementation

```go
// internal/voting/types.go

package voting

import (
    "math/big"
    "time"
    
    "github.com/yourusername/covertvote/internal/crypto"
)

// Ballot represents a voter's ballot
type Ballot struct {
    VoterID       string
    SlotBallots   []*SlotBallot // One ballot per SMDC slot
    RingSignature *crypto.RingSignature
    Timestamp     time.Time
}

// SlotBallot represents a ballot for one SMDC slot
type SlotBallot struct {
    SlotIndex         int
    EncryptedVote     *big.Int // E(v) - encrypted vote choice
    EncryptedWeighted *big.Int // E(w × v) - encrypted weighted vote
    VoteProof         *VoteValidityProof
}

// VoteValidityProof proves vote is valid without revealing choice
type VoteValidityProof struct {
    // Proof that vote is for valid candidate (0 to m-1)
    Commitments []*big.Int
    Challenges  []*big.Int
    Responses   []*big.Int
}

// CastVoteRequest represents a vote casting request
type CastVoteRequest struct {
    VoterID         string
    FingerprintData []byte      // For authentication
    Votes           []int       // Vote choices for each slot
}

// CastVoteResult represents vote casting outcome
type CastVoteResult struct {
    Success      bool
    BallotID     string
    Error        string
    Timestamp    time.Time
}
```

```go
// internal/voting/cast.go

package voting

import (
    "crypto/rand"
    "errors"
    "fmt"
    "math/big"
    "sync"
    "time"
    
    "github.com/yourusername/covertvote/internal/crypto"
    "github.com/yourusername/covertvote/internal/voter"
)

// VotingService handles vote casting
type VotingService struct {
    paillierPK     *crypto.PaillierPublicKey
    ringParams     *crypto.RingParams
    voterService   *voter.RegistrationService
    ballotStore    map[string]*Ballot
    usedKeyImages  map[string]bool // For double-vote detection
    numCandidates  int
    mutex          sync.RWMutex
}

// NewVotingService creates a new voting service
func NewVotingService(
    pk *crypto.PaillierPublicKey,
    rp *crypto.RingParams,
    vs *voter.RegistrationService,
    numCandidates int,
) *VotingService {
    return &VotingService{
        paillierPK:    pk,
        ringParams:    rp,
        voterService:  vs,
        ballotStore:   make(map[string]*Ballot),
        usedKeyImages: make(map[string]bool),
        numCandidates: numCandidates,
    }
}

// CastVote casts a vote
func (vs *VotingService) CastVote(req *CastVoteRequest, privateKey *crypto.RingKeyPair) (*CastVoteResult, error) {
    vs.mutex.Lock()
    defer vs.mutex.Unlock()
    
    // Step 1: Authenticate voter
    voterRecord, err := vs.voterService.GetVoter(req.VoterID)
    if err != nil {
        return nil, errors.New("voter not found")
    }
    
    // Verify fingerprint
    if !vs.voterService.fpProcessor.Verify(req.FingerprintData, voterRecord.FingerprintCred) {
        return &CastVoteResult{
            Success: false,
            Error:   "authentication failed",
        }, nil
    }
    
    // Step 2: Validate votes
    k := voterRecord.SMDCCredential.K
    if len(req.Votes) != k {
        return nil, fmt.Errorf("expected %d votes, got %d", k, len(req.Votes))
    }
    
    for _, v := range req.Votes {
        if v < 0 || v >= vs.numCandidates {
            return nil, fmt.Errorf("invalid vote choice: %d", v)
        }
    }
    
    // Step 3: Create encrypted ballots for each slot
    slotBallots := make([]*SlotBallot, k)
    
    for i := 0; i < k; i++ {
        slot := voterRecord.SMDCCredential.Slots[i]
        vote := big.NewInt(int64(req.Votes[i]))
        
        // Encrypt vote: E(v)
        encVote, err := vs.paillierPK.Encrypt(vote)
        if err != nil {
            return nil, err
        }
        
        // Compute encrypted weighted vote: E(w × v)
        // If w = 0: E(0 × v) = E(0)
        // If w = 1: E(1 × v) = E(v)
        var encWeighted *big.Int
        if slot.Weight.Cmp(big.NewInt(0)) == 0 {
            // Fake slot: encrypt 0
            encWeighted, _ = vs.paillierPK.Encrypt(big.NewInt(0))
        } else {
            // Real slot: E(w × v) = E(v)^w = E(v)
            encWeighted = vs.paillierPK.Multiply(encVote, slot.Weight)
        }
        
        // Create validity proof (simplified)
        proof := vs.createVoteProof(vote)
        
        slotBallots[i] = &SlotBallot{
            SlotIndex:         i,
            EncryptedVote:     encVote,
            EncryptedWeighted: encWeighted,
            VoteProof:         proof,
        }
    }
    
    // Step 4: Create ring signature
    ring := vs.voterService.GetAllPublicKeys()
    signerIndex := vs.findSignerIndex(ring, privateKey.PublicKey)
    if signerIndex < 0 {
        return nil, errors.New("voter not in ring")
    }
    
    // Sign the ballot
    ballotBytes := vs.serializeBallots(slotBallots)
    ringSig, err := vs.ringParams.Sign(ballotBytes, privateKey, ring, signerIndex)
    if err != nil {
        return nil, err
    }
    
    // Step 5: Check for double voting
    keyImageStr := ringSig.KeyImage.String()
    if vs.usedKeyImages[keyImageStr] {
        return &CastVoteResult{
            Success: false,
            Error:   "double voting detected",
        }, nil
    }
    vs.usedKeyImages[keyImageStr] = true
    
    // Step 6: Create and store ballot
    ballotID := vs.generateBallotID()
    ballot := &Ballot{
        VoterID:       req.VoterID,
        SlotBallots:   slotBallots,
        RingSignature: ringSig,
        Timestamp:     time.Now(),
    }
    
    vs.ballotStore[ballotID] = ballot
    
    return &CastVoteResult{
        Success:   true,
        BallotID:  ballotID,
        Timestamp: ballot.Timestamp,
    }, nil
}

// createVoteProof creates a proof that vote is valid (simplified)
func (vs *VotingService) createVoteProof(vote *big.Int) *VoteValidityProof {
    // In production, implement proper ZK proof
    // This is a placeholder
    return &VoteValidityProof{
        Commitments: []*big.Int{vote},
        Challenges:  []*big.Int{big.NewInt(0)},
        Responses:   []*big.Int{big.NewInt(0)},
    }
}

// findSignerIndex finds the index of voter's public key in ring
func (vs *VotingService) findSignerIndex(ring []*big.Int, pk *big.Int) int {
    for i, key := range ring {
        if key.Cmp(pk) == 0 {
            return i
        }
    }
    return -1
}

// serializeBallots serializes ballots for signing
func (vs *VotingService) serializeBallots(ballots []*SlotBallot) []byte {
    var data []byte
    for _, b := range ballots {
        data = append(data, b.EncryptedWeighted.Bytes()...)
    }
    return data
}

// generateBallotID generates a unique ballot ID
func (vs *VotingService) generateBallotID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return fmt.Sprintf("ballot-%x", b)
}

// GetBallot returns a ballot by ID
func (vs *VotingService) GetBallot(ballotID string) (*Ballot, error) {
    vs.mutex.RLock()
    defer vs.mutex.RUnlock()
    
    ballot, exists := vs.ballotStore[ballotID]
    if !exists {
        return nil, errors.New("ballot not found")
    }
    return ballot, nil
}

// GetAllBallots returns all cast ballots
func (vs *VotingService) GetAllBallots() []*Ballot {
    vs.mutex.RLock()
    defer vs.mutex.RUnlock()
    
    ballots := make([]*Ballot, 0, len(vs.ballotStore))
    for _, b := range vs.ballotStore {
        ballots = append(ballots, b)
    }
    return ballots
}
```

---

# Step 10: Tallying & Decryption

## 10.1 Go Implementation

```go
// internal/tally/types.go

package tally

import (
    "math/big"
    "time"
)

// TallyResult represents the election result
type TallyResult struct {
    CandidateResults []CandidateResult
    TotalVotes       int
    Timestamp        time.Time
    Proof            *TallyProof
}

// CandidateResult represents one candidate's result
type CandidateResult struct {
    CandidateID   int
    CandidateName string
    VoteCount     int64
    Percentage    float64
}

// TallyProof proves correct tallying
type TallyProof struct {
    EncryptedTally   *big.Int
    DecryptionProof  *DecryptionProof
    PartialDecrypts  []*PartialDecrypt
}

// PartialDecrypt represents one server's partial decryption
type PartialDecrypt struct {
    ServerID string
    Value    *big.Int
    Proof    *big.Int
}

// DecryptionProof proves correct decryption
type DecryptionProof struct {
    Challenge *big.Int
    Response  *big.Int
}
```

```go
// internal/tally/decrypt.go

package tally

import (
    "math/big"
    
    "github.com/yourusername/covertvote/internal/crypto"
    "github.com/yourusername/covertvote/internal/sa2"
    "github.com/yourusername/covertvote/internal/voting"
)

// TallyService handles vote tallying
type TallyService struct {
    paillierSK    *crypto.PaillierPrivateKey
    paillierPK    *crypto.PaillierPublicKey
    sa2System     *sa2.SA2System
    votingService *voting.VotingService
    numCandidates int
}

// NewTallyService creates a new tally service
func NewTallyService(
    sk *crypto.PaillierPrivateKey,
    sa2Sys *sa2.SA2System,
    vs *voting.VotingService,
    numCandidates int,
) *TallyService {
    return &TallyService{
        paillierSK:    sk,
        paillierPK:    sk.PublicKey,
        sa2System:     sa2Sys,
        votingService: vs,
        numCandidates: numCandidates,
    }
}

// ProcessAllVotes processes all ballots through SA²
func (ts *TallyService) ProcessAllVotes() error {
    ballots := ts.votingService.GetAllBallots()
    
    for _, ballot := range ballots {
        // For each slot in the ballot
        for _, slotBallot := range ballot.SlotBallots {
            // Process the encrypted weighted vote
            err := ts.sa2System.ProcessVote(
                slotBallot.EncryptedWeighted,
                ballot.VoterID,
            )
            if err != nil {
                return err
            }
        }
    }
    
    return nil
}

// ComputeTally computes the final tally
func (ts *TallyService) ComputeTally() (*TallyResult, error) {
    // Step 1: Get encrypted tally from SA²
    combined, err := ts.sa2System.GetEncryptedTally()
    if err != nil {
        return nil, err
    }
    
    // Step 2: Decrypt the tally
    // In production, use threshold decryption with multiple parties
    decryptedTally, err := ts.paillierSK.Decrypt(combined.EncryptedTally)
    if err != nil {
        return nil, err
    }
    
    // Step 3: Parse the tally
    // Since we're using simple encoding, the tally is the sum of votes
    // For multiple candidates, we'd need separate tallies per candidate
    
    // Simplified: assuming single candidate vote count
    // In production, implement proper multi-candidate tallying
    
    totalVotes := decryptedTally.Int64()
    
    // Create result
    results := make([]CandidateResult, ts.numCandidates)
    for i := 0; i < ts.numCandidates; i++ {
        results[i] = CandidateResult{
            CandidateID:   i,
            CandidateName: fmt.Sprintf("Candidate %d", i+1),
            VoteCount:     0, // Would be computed per candidate
            Percentage:    0,
        }
    }
    
    // For demo: put all votes to first candidate
    // In production, properly decode per-candidate votes
    results[0].VoteCount = totalVotes
    results[0].Percentage = 100.0
    
    // Create proof
    proof := &TallyProof{
        EncryptedTally: combined.EncryptedTally,
    }
    
    return &TallyResult{
        CandidateResults: results,
        TotalVotes:       combined.TotalVotes,
        Timestamp:        time.Now(),
        Proof:            proof,
    }, nil
}

// VerifyTally verifies the tally is correct
func (ts *TallyService) VerifyTally(result *TallyResult) bool {
    // Verify the decryption proof
    // In production, implement full verification
    return result.Proof != nil
}
```

---

# Step 11: Hyperledger Fabric Integration

## 11.1 Chaincode (Smart Contract)

```go
// chaincode/covertvote/covertvote.go

package main

import (
    "encoding/json"
    "fmt"
    
    "github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CovertVoteContract implements the voting smart contract
type CovertVoteContract struct {
    contractapi.Contract
}

// Election represents an election
type Election struct {
    ID            string   `json:"id"`
    Name          string   `json:"name"`
    Candidates    []string `json:"candidates"`
    MerkleRoot    string   `json:"merkleRoot"`
    Status        string   `json:"status"` // "created", "registration", "voting", "tallying", "completed"
    StartTime     int64    `json:"startTime"`
    EndTime       int64    `json:"endTime"`
}

// StoredCredential represents a stored SMDC credential
type StoredCredential struct {
    VoterID     string   `json:"voterId"`
    Commitments []string `json:"commitments"`
    Proofs      string   `json:"proofs"` // JSON encoded
}

// StoredBallot represents a stored ballot
type StoredBallot struct {
    BallotID     string `json:"ballotId"`
    EncryptedData string `json:"encryptedData"`
    RingSignature string `json:"ringSignature"`
    Timestamp     int64  `json:"timestamp"`
}

// TallyRecord represents the tally result
type TallyRecord struct {
    ElectionID    string            `json:"electionId"`
    Results       map[string]int64  `json:"results"`
    Proof         string            `json:"proof"`
    Timestamp     int64             `json:"timestamp"`
}

// CreateElection creates a new election
func (c *CovertVoteContract) CreateElection(ctx contractapi.TransactionContextInterface, id, name string, candidates []string) error {
    election := Election{
        ID:         id,
        Name:       name,
        Candidates: candidates,
        Status:     "created",
    }
    
    electionJSON, err := json.Marshal(election)
    if err != nil {
        return err
    }
    
    return ctx.GetStub().PutState("election_"+id, electionJSON)
}

// SetMerkleRoot sets the voter Merkle root
func (c *CovertVoteContract) SetMerkleRoot(ctx contractapi.TransactionContextInterface, electionID, merkleRoot string) error {
    electionJSON, err := ctx.GetStub().GetState("election_" + electionID)
    if err != nil {
        return err
    }
    
    var election Election
    json.Unmarshal(electionJSON, &election)
    
    election.MerkleRoot = merkleRoot
    election.Status = "registration"
    
    updatedJSON, _ := json.Marshal(election)
    return ctx.GetStub().PutState("election_"+electionID, updatedJSON)
}

// StoreCredential stores a voter's SMDC credential
func (c *CovertVoteContract) StoreCredential(ctx contractapi.TransactionContextInterface, electionID, voterID string, commitments []string, proofs string) error {
    cred := StoredCredential{
        VoterID:     voterID,
        Commitments: commitments,
        Proofs:      proofs,
    }
    
    credJSON, _ := json.Marshal(cred)
    key := fmt.Sprintf("cred_%s_%s", electionID, voterID)
    return ctx.GetStub().PutState(key, credJSON)
}

// StoreBallot stores an encrypted ballot
func (c *CovertVoteContract) StoreBallot(ctx contractapi.TransactionContextInterface, electionID, ballotID, encryptedData, ringSignature string, timestamp int64) error {
    ballot := StoredBallot{
        BallotID:      ballotID,
        EncryptedData: encryptedData,
        RingSignature: ringSignature,
        Timestamp:     timestamp,
    }
    
    ballotJSON, _ := json.Marshal(ballot)
    key := fmt.Sprintf("ballot_%s_%s", electionID, ballotID)
    return ctx.GetStub().PutState(key, ballotJSON)
}

// StoreTally stores the tally result
func (c *CovertVoteContract) StoreTally(ctx contractapi.TransactionContextInterface, electionID string, results map[string]int64, proof string, timestamp int64) error {
    tally := TallyRecord{
        ElectionID: electionID,
        Results:    results,
        Proof:      proof,
        Timestamp:  timestamp,
    }
    
    tallyJSON, _ := json.Marshal(tally)
    return ctx.GetStub().PutState("tally_"+electionID, tallyJSON)
}

// GetElection retrieves an election
func (c *CovertVoteContract) GetElection(ctx contractapi.TransactionContextInterface, electionID string) (*Election, error) {
    electionJSON, err := ctx.GetStub().GetState("election_" + electionID)
    if err != nil {
        return nil, err
    }
    
    var election Election
    json.Unmarshal(electionJSON, &election)
    return &election, nil
}

// GetTally retrieves the tally
func (c *CovertVoteContract) GetTally(ctx contractapi.TransactionContextInterface, electionID string) (*TallyRecord, error) {
    tallyJSON, err := ctx.GetStub().GetState("tally_" + electionID)
    if err != nil {
        return nil, err
    }
    
    var tally TallyRecord
    json.Unmarshal(tallyJSON, &tally)
    return &tally, nil
}

func main() {
    chaincode, err := contractapi.NewChaincode(&CovertVoteContract{})
    if err != nil {
        panic(err)
    }
    
    if err := chaincode.Start(); err != nil {
        panic(err)
    }
}
```

## 11.2 Fabric SDK Integration

```go
// internal/blockchain/hyperledger.go

package blockchain

import (
    "encoding/json"
    "fmt"
    
    "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
    "github.com/hyperledger/fabric-sdk-go/pkg/core/config"
    "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

// FabricClient represents a Hyperledger Fabric client
type FabricClient struct {
    sdk           *fabsdk.FabricSDK
    channelClient *channel.Client
    chaincodeName string
}

// NewFabricClient creates a new Fabric client
func NewFabricClient(configPath, channelID, chaincodeName, userID, orgID string) (*FabricClient, error) {
    // Create SDK instance
    sdk, err := fabsdk.New(config.FromFile(configPath))
    if err != nil {
        return nil, err
    }
    
    // Create channel context
    clientContext := sdk.ChannelContext(channelID, fabsdk.WithUser(userID), fabsdk.WithOrg(orgID))
    
    // Create channel client
    channelClient, err := channel.New(clientContext)
    if err != nil {
        return nil, err
    }
    
    return &FabricClient{
        sdk:           sdk,
        channelClient: channelClient,
        chaincodeName: chaincodeName,
    }, nil
}

// CreateElection creates a new election on the blockchain
func (fc *FabricClient) CreateElection(id, name string, candidates []string) error {
    candidatesJSON, _ := json.Marshal(candidates)
    
    _, err := fc.channelClient.Execute(
        channel.Request{
            ChaincodeID: fc.chaincodeName,
            Fcn:         "CreateElection",
            Args:        [][]byte{[]byte(id), []byte(name), candidatesJSON},
        },
    )
    return err
}

// StoreCredential stores an SMDC credential
func (fc *FabricClient) StoreCredential(electionID, voterID string, commitments []string, proofs string) error {
    commitmentsJSON, _ := json.Marshal(commitments)
    
    _, err := fc.channelClient.Execute(
        channel.Request{
            ChaincodeID: fc.chaincodeName,
            Fcn:         "StoreCredential",
            Args:        [][]byte{[]byte(electionID), []byte(voterID), commitmentsJSON, []byte(proofs)},
        },
    )
    return err
}

// StoreBallot stores an encrypted ballot
func (fc *FabricClient) StoreBallot(electionID, ballotID, encryptedData, ringSignature string, timestamp int64) error {
    _, err := fc.channelClient.Execute(
        channel.Request{
            ChaincodeID: fc.chaincodeName,
            Fcn:         "StoreBallot",
            Args:        [][]byte{
                []byte(electionID),
                []byte(ballotID),
                []byte(encryptedData),
                []byte(ringSignature),
                []byte(fmt.Sprintf("%d", timestamp)),
            },
        },
    )
    return err
}

// StoreTally stores the tally result
func (fc *FabricClient) StoreTally(electionID string, results map[string]int64, proof string, timestamp int64) error {
    resultsJSON, _ := json.Marshal(results)
    
    _, err := fc.channelClient.Execute(
        channel.Request{
            ChaincodeID: fc.chaincodeName,
            Fcn:         "StoreTally",
            Args:        [][]byte{
                []byte(electionID),
                resultsJSON,
                []byte(proof),
                []byte(fmt.Sprintf("%d", timestamp)),
            },
        },
    )
    return err
}

// Close closes the SDK
func (fc *FabricClient) Close() {
    fc.sdk.Close()
}
```

---

# Step 12: API Endpoints

## 12.1 Router Setup

```go
// api/router.go

package api

import (
    "github.com/gin-gonic/gin"
    
    "github.com/yourusername/covertvote/api/handlers"
    "github.com/yourusername/covertvote/api/middleware"
)

// SetupRouter sets up the API router
func SetupRouter(h *handlers.Handlers) *gin.Engine {
    r := gin.Default()
    
    // Middleware
    r.Use(middleware.CORS())
    r.Use(middleware.RateLimit())
    
    // API v1
    v1 := r.Group("/api/v1")
    {
        // Election management
        election := v1.Group("/election")
        {
            election.POST("/create", h.CreateElection)
            election.GET("/:id", h.GetElection)
            election.POST("/:id/start", h.StartElection)
            election.POST("/:id/end", h.EndElection)
        }
        
        // Voter registration
        voter := v1.Group("/voter")
        {
            voter.POST("/register", h.RegisterVoter)
            voter.GET("/:id", h.GetVoter)
            voter.POST("/verify", h.VerifyVoter)
        }
        
        // Voting
        vote := v1.Group("/vote")
        {
            vote.POST("/cast", h.CastVote)
            vote.GET("/ballot/:id", h.GetBallot)
            vote.POST("/verify", h.VerifyBallot)
        }
        
        // Tally
        tally := v1.Group("/tally")
        {
            tally.POST("/compute", h.ComputeTally)
            tally.GET("/:electionId", h.GetTally)
            tally.POST("/verify", h.VerifyTally)
        }
        
        // Verification
        verify := v1.Group("/verify")
        {
            verify.POST("/credential", h.VerifyCredential)
            verify.POST("/signature", h.VerifySignature)
            verify.POST("/merkle", h.VerifyMerkleProof)
        }
    }
    
    return r
}
```

## 12.2 API Handlers

```go
// api/handlers/handlers.go

package handlers

import (
    "github.com/yourusername/covertvote/internal/voter"
    "github.com/yourusername/covertvote/internal/voting"
    "github.com/yourusername/covertvote/internal/tally"
    "github.com/yourusername/covertvote/internal/blockchain"
)

// Handlers holds all API handlers
type Handlers struct {
    voterService  *voter.RegistrationService
    votingService *voting.VotingService
    tallyService  *tally.TallyService
    fabricClient  *blockchain.FabricClient
}

// NewHandlers creates new handlers
func NewHandlers(
    vs *voter.RegistrationService,
    vts *voting.VotingService,
    ts *tally.TallyService,
    fc *blockchain.FabricClient,
) *Handlers {
    return &Handlers{
        voterService:  vs,
        votingService: vts,
        tallyService:  ts,
        fabricClient:  fc,
    }
}
```

```go
// api/handlers/register.go

package handlers

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/yourusername/covertvote/internal/voter"
)

// RegisterVoterRequest represents registration request
type RegisterVoterRequest struct {
    NIDNumber       string `json:"nidNumber" binding:"required"`
    FingerprintData string `json:"fingerprintData" binding:"required"` // Base64 encoded
}

// RegisterVoter handles voter registration
func (h *Handlers) RegisterVoter(c *gin.Context) {
    var req RegisterVoterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Decode fingerprint
    fpData, err := base64.StdEncoding.DecodeString(req.FingerprintData)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fingerprint data"})
        return
    }
    
    // Register voter
    result, err := h.voterService.Register(&voter.RegistrationRequest{
        NIDNumber:       req.NIDNumber,
        FingerprintData: fpData,
    })
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    if !result.Success {
        c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "voterId": result.VoterID,
    })
}

// GetVoter gets voter information
func (h *Handlers) GetVoter(c *gin.Context) {
    voterID := c.Param("id")
    
    voterRecord, err := h.voterService.GetVoter(voterID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "voter not found"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "voterId":      voterRecord.VoterID,
        "isEligible":   voterRecord.IsEligible,
        "registeredAt": voterRecord.RegisteredAt,
    })
}
```

```go
// api/handlers/vote.go

package handlers

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/yourusername/covertvote/internal/voting"
)

// CastVoteRequest represents vote casting request
type CastVoteRequest struct {
    VoterID         string `json:"voterId" binding:"required"`
    FingerprintData string `json:"fingerprintData" binding:"required"`
    Votes           []int  `json:"votes" binding:"required"`
}

// CastVote handles vote casting
func (h *Handlers) CastVote(c *gin.Context) {
    var req CastVoteRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Decode fingerprint
    fpData, _ := base64.StdEncoding.DecodeString(req.FingerprintData)
    
    // Cast vote
    result, err := h.votingService.CastVote(&voting.CastVoteRequest{
        VoterID:         req.VoterID,
        FingerprintData: fpData,
        Votes:           req.Votes,
    }, nil) // Private key would be provided by client
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    if !result.Success {
        c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success":   true,
        "ballotId":  result.BallotID,
        "timestamp": result.Timestamp,
    })
}
```

---

# Step 13: Testing Guide

## 13.1 Unit Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/crypto/...
go test ./internal/smdc/...
go test ./internal/sa2/...

# Run with verbose output
go test -v ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 13.2 Integration Test

```go
// test/integration/full_flow_test.go

package integration

import (
    "testing"
    
    "github.com/yourusername/covertvote/internal/crypto"
    "github.com/yourusername/covertvote/internal/smdc"
    "github.com/yourusername/covertvote/internal/sa2"
    "github.com/yourusername/covertvote/internal/voter"
    "github.com/yourusername/covertvote/internal/voting"
    "github.com/yourusername/covertvote/internal/tally"
)

func TestFullVotingFlow(t *testing.T) {
    // Step 1: Setup cryptographic parameters
    paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
    paillierPK := paillierSK.PublicKey
    
    pedersenPP, _ := crypto.GeneratePedersenParams(512)
    ringParams, _ := crypto.GenerateRingParams(512)
    
    // Step 2: Initialize services
    voterService, _ := voter.NewRegistrationService(pedersenPP, ringParams, 5)
    sa2System := sa2.NewSA2System(paillierPK)
    votingService := voting.NewVotingService(paillierPK, ringParams, voterService, 3)
    tallyService := tally.NewTallyService(paillierSK, sa2System, votingService, 3)
    
    // Step 3: Register voters
    for i := 0; i < 10; i++ {
        result, err := voterService.Register(&voter.RegistrationRequest{
            NIDNumber:       fmt.Sprintf("NID%d", i),
            FingerprintData: []byte(fmt.Sprintf("fingerprint%d", i)),
        })
        if err != nil || !result.Success {
            t.Fatalf("Registration failed for voter %d", i)
        }
    }
    
    // Step 4: Cast votes
    // ... voting logic ...
    
    // Step 5: Compute tally
    err := tallyService.ProcessAllVotes()
    if err != nil {
        t.Fatalf("Processing votes failed: %v", err)
    }
    
    result, err := tallyService.ComputeTally()
    if err != nil {
        t.Fatalf("Computing tally failed: %v", err)
    }
    
    // Step 6: Verify
    t.Logf("Tally result: %+v", result)
}
```

## 13.3 Benchmark Test

```go
// test/benchmark/performance_test.go

package benchmark

import (
    "testing"
    
    "github.com/yourusername/covertvote/internal/crypto"
)

func BenchmarkPaillierEncrypt(b *testing.B) {
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey
    m := big.NewInt(42)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pk.Encrypt(m)
    }
}

func BenchmarkPaillierDecrypt(b *testing.B) {
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey
    m := big.NewInt(42)
    c, _ := pk.Encrypt(m)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sk.Decrypt(c)
    }
}

func BenchmarkSMDCGeneration(b *testing.B) {
    pp, _ := crypto.GeneratePedersenParams(512)
    gen := smdc.NewSMDCGenerator(pp, 5)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        gen.GenerateCredential("voter")
    }
}
```

---

# Step 14: Post-Quantum Hybrid (Kyber)

## 14.1 Kyber Integration

```go
// internal/crypto/kyber.go

package crypto

import (
    "github.com/cloudflare/circl/kem/kyber/kyber768"
)

// KyberKeyPair represents a Kyber key pair
type KyberKeyPair struct {
    PublicKey  []byte
    PrivateKey []byte
}

// GenerateKyberKeyPair generates a new Kyber key pair
func GenerateKyberKeyPair() (*KyberKeyPair, error) {
    pk, sk, err := kyber768.GenerateKeyPair(nil)
    if err != nil {
        return nil, err
    }
    
    pkBytes := make([]byte, kyber768.PublicKeySize)
    skBytes := make([]byte, kyber768.PrivateKeySize)
    
    pk.Pack(pkBytes)
    sk.Pack(skBytes)
    
    return &KyberKeyPair{
        PublicKey:  pkBytes,
        PrivateKey: skBytes,
    }, nil
}

// HybridEncrypt encrypts using both Kyber and classical
func HybridEncrypt(message []byte, kyberPK []byte, classicalPK *PaillierPublicKey) (*HybridCiphertext, error) {
    // Kyber encapsulation
    var pk kyber768.PublicKey
    pk.Unpack(kyberPK)
    
    ct, ss, err := kyber768.Encapsulate(&pk)
    if err != nil {
        return nil, err
    }
    
    // Use shared secret to derive symmetric key
    symKey := sha3.Sum256(ss)
    
    // Encrypt message with symmetric key (AES-GCM)
    encMessage, err := aesGCMEncrypt(symKey[:], message)
    if err != nil {
        return nil, err
    }
    
    // Also encrypt with classical (Paillier) for redundancy
    msgInt := new(big.Int).SetBytes(message)
    classicalCT, _ := classicalPK.Encrypt(msgInt)
    
    return &HybridCiphertext{
        KyberCT:     ct,
        SymmetricCT: encMessage,
        ClassicalCT: classicalCT,
    }, nil
}

// HybridCiphertext represents hybrid encrypted data
type HybridCiphertext struct {
    KyberCT     []byte
    SymmetricCT []byte
    ClassicalCT *big.Int
}
```

---

# Performance Optimization

## Optimization Techniques

```go
// pkg/utils/parallel.go

package utils

import (
    "sync"
)

// ParallelProcess processes items in parallel
func ParallelProcess[T any, R any](items []T, workers int, process func(T) R) []R {
    results := make([]R, len(items))
    var wg sync.WaitGroup
    
    jobs := make(chan int, len(items))
    
    // Start workers
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for i := range jobs {
                results[i] = process(items[i])
            }
        }()
    }
    
    // Send jobs
    for i := range items {
        jobs <- i
    }
    close(jobs)
    
    wg.Wait()
    return results
}

// BatchProcess processes in batches
func BatchProcess[T any](items []T, batchSize int, process func([]T) error) error {
    for i := 0; i < len(items); i += batchSize {
        end := i + batchSize
        if end > len(items) {
            end = len(items)
        }
        
        if err := process(items[i:end]); err != nil {
            return err
        }
    }
    return nil
}
```

---

# Security Checklist

## Pre-Deployment Checklist

```
□ All cryptographic parameters use secure sizes (2048+ bits for Paillier)
□ Random number generation uses crypto/rand only
□ No secret keys logged or exposed in errors
□ Constant-time comparisons for sensitive data
□ Input validation on all API endpoints
□ Rate limiting enabled
□ TLS/HTTPS configured
□ Hyperledger Fabric channels properly configured
□ Access control policies set
□ Audit logging enabled
□ Backup and recovery procedures documented
□ Penetration testing completed
□ Code review completed
□ All tests passing
□ Performance benchmarks acceptable
```

---

# Quick Reference

## Build Commands

```bash
# Build server
go build -o bin/server ./cmd/server

# Build with optimizations
go build -ldflags="-s -w" -o bin/server ./cmd/server

# Run server
./bin/server

# Run with environment
ENV=production ./bin/server
```

## Docker Commands

```bash
# Build Docker image
docker build -t covertvote:latest .

# Run container
docker run -p 8080:8080 covertvote:latest

# Docker Compose
docker-compose up -d
```

---

**Document Version:** 1.0  
**Last Updated:** January 2026  
**Author:** CovertVote Thesis Project

---

# END OF DOCUMENT
