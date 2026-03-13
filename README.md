# CovertVote: A Seven-Protocol Blockchain E-Voting System

A secure, anonymous, and coercion-resistant electronic voting system implementing seven cryptographic protocols: Paillier homomorphic encryption, Pedersen commitments, zero-knowledge proofs (strong Fiat-Shamir), linkable ring signatures, SMDC (Self-Masking Deniable Credentials), SA2 (Samplable Anonymous Aggregation), and Kyber768 post-quantum key encapsulation.

Built in Go 1.24+ with Hyperledger Fabric blockchain integration.

## Paper

This software accompanies a research paper submitted as part of the CSE400 capstone project. The system implements the full CovertVote protocol specification with formal security properties verified through property-based tests.

**Target venue:** IEEE Transactions on Information Forensics and Security (TIFS)

## Citation

If you use this software in your research, please cite:

```bibtex
@software{covertvote2026,
  title     = {CovertVote: A Seven-Protocol Blockchain E-Voting System with SMDC and SA2},
  author    = {CSE400 Capstone Team},
  year      = {2026},
  url       = {https://github.com/covertvote/e-voting},
  license   = {MIT}
}
```

See also [`CITATION.cff`](CITATION.cff) for GitHub's built-in citation support.

## Security Properties

| Property | Protocol | Theorem | Status |
|----------|----------|---------|--------|
| **Ballot Privacy** | Paillier (DCRA) | Theorem 1 | Proven |
| **Vote Validity** | ZKP sigma-protocol (Strong Fiat-Shamir) | Theorem 2 | Proven |
| **Double-Vote Detection** | Linkable Ring Signatures (DLP) | Theorem 3 | Proven |
| **Coercion Resistance** | SMDC (Pedersen hiding) | Theorem 4 | Proven |
| **Universal Verifiability** | Homomorphic tally + ZKP | Theorem 5 | Proven |
| **Composition Security** | Hybrid argument (independent randomness) | Theorem 6 | Proven |

All security reductions and formal proofs are documented in [`internal/crypto/SECURITY.md`](internal/crypto/SECURITY.md).

## Architecture

```
CovertVote System (3-service deployment)
├── API Server (:8080)           — Voter registration, vote casting, election management
├── SA2 Leader (:8081)           — Aggregation server A (must be isolated)
├── SA2 Helper (:8082)           — Aggregation server B (must be isolated)
│
├── Cryptographic Layer
│   ├── Paillier Encryption      — 2048-bit, additive homomorphic (E(a)×E(b)=E(a+b))
│   ├── Pedersen Commitments     — 512-bit, computationally hiding & binding
│   ├── Zero-Knowledge Proofs    — OR-proof (binary), Schnorr (sum-one), Strong Fiat-Shamir
│   ├── Ring Signatures          — Linkable, key image double-vote detection
│   └── Kyber768 KEM             — NIST Level 3 post-quantum hybrid encryption
├── Privacy Layer
│   ├── SMDC Credentials         — k=5 slots (1 real + 4 fake), coercion resistance
│   └── SA2 Aggregation          — 2-server non-colluding, mask cancellation
├── Blockchain Layer
│   └── Hyperledger Fabric       — Immutable vote record, chaincode verification
└── Authentication Layer
    └── Biometric + Liveness     — SHA-3 fingerprint hashing, anti-spoofing
```

## Quick Start

```bash
# Clone and build
git clone https://github.com/covertvote/e-voting.git
cd e-voting
go mod tidy

# Run all tests
go test ./internal/... ./api/... -timeout 600s

# Run property-based crypto tests
go test -v ./internal/crypto/ -run "TestPaillierHomomorphic|TestPedersen|TestRingSignature"

# Run benchmarks
go test -bench=. -benchmem ./test/benchmark/ -run="^$" -timeout 600s

# Build binaries
go build -o bin/covertvote-api cmd/api-server/main.go
go build -o bin/aggregator-a cmd/aggregator-a/main.go
go build -o bin/aggregator-b cmd/aggregator-b/main.go

# Docker deployment (SA2 servers on separate containers)
docker-compose -f docker-compose-sa2.yml up
```

## Benchmarks

Full benchmark results with hardware specifications are in [`test/benchmark/results/PAPER_RESULTS.md`](test/benchmark/results/PAPER_RESULTS.md).

### Key Results (AMD Ryzen 5 7530U, 12 threads)

| Operation | Time | Notes |
|-----------|------|-------|
| Paillier Encrypt (2048-bit) | 8.7 ms | Dominates per-vote cost |
| Pedersen Commit (512-bit) | 80 us | |
| ZKP Binary Prove | 245 us | Strong Fiat-Shamir |
| ZKP Binary Verify | 329 us | |
| SMDC Generate (k=5) | 1.7 ms | 5 Pedersen commitments |
| Ring Sign (n=100) | 27 ms | Linear in ring size |
| SA2 Split | 35 ms | Paillier re-encryption |
| **Full Pipeline (7 steps)** | **70 ms** | **Per vote, end-to-end** |

### Scalability

| Voters | Homomorphic Tally Time | Per-Vote |
|--------|------------------------|----------|
| 1,000 | 6 ms | 6.3 us |
| 10,000 | 59 ms | 6.0 us |
| 100,000 | 628 ms | 6.3 us |
| 50,000,000 (projected) | ~5.1 min | 6.1 us |

**O(n) complexity confirmed:** per-vote tally cost is constant regardless of voter count or candidate count.

## Project Structure

```
E-voting/
├── cmd/                          # Entry points
│   ├── api-server/               # Main API server
│   ├── aggregator-a/             # SA2 Leader
│   └── aggregator-b/             # SA2 Helper
├── internal/                     # Core implementation
│   ├── crypto/                   # Paillier, Pedersen, ZKP, Ring Signatures
│   ├── pq/                       # Kyber768 post-quantum hybrid encryption
│   ├── smdc/                     # Self-Masking Deniable Credentials
│   ├── sa2/                      # Samplable Anonymous Aggregation
│   ├── voting/                   # Vote casting orchestration
│   ├── tally/                    # Homomorphic tallying & decryption
│   ├── voter/                    # Registration & Merkle eligibility
│   ├── biometric/                # Fingerprint & liveness detection
│   ├── blockchain/               # Hyperledger Fabric integration
│   └── database/                 # SQLite persistence
├── api/                          # REST API (Gin framework)
│   ├── handlers/                 # HTTP handlers
│   ├── middleware/                # Auth, rate limiting, CORS
│   └── routes/                   # Route definitions
├── chaincode/                    # Hyperledger Fabric chaincode
├── test/benchmark/               # Performance benchmarks
│   └── results/                  # Benchmark results for paper
├── docker-compose-sa2.yml        # SA2 isolated deployment
└── Thesis/                       # Paper and thesis documents
```

## Testing

```bash
# All unit + property tests
go test ./internal/... ./api/... -timeout 600s

# Property-based tests (crypto correctness)
go test -v -run "Property|Homomorphic|Binding|Linkability|Cancellation|Identity" ./internal/crypto/

# Tally correctness & SA2 integrity tests
go test -v -run "TestTallyCorrectness|TestSA2TallyIntegrity" ./internal/tally/

# Scalability benchmarks
go test -v -run "TestScalabilityTally|TestComplexityValidation" ./test/benchmark/ -timeout 30m

# Coverage report
go test -coverprofile=coverage.out ./internal/... ./api/...
go tool cover -func=coverage.out
```

## Configuration

Copy `.env.example` to `.env` and update all placeholder values. See the file for all available options.

**Critical:** SA2 Leader and Helper servers must run on separate machines/containers with independent API keys. See [`docker-compose-sa2.yml`](docker-compose-sa2.yml).

## Documentation

- [`internal/crypto/SECURITY.md`](internal/crypto/SECURITY.md) - Cryptographic security properties and theorem references
- [`SECURITY.md`](SECURITY.md) - Security policy and vulnerability reporting
- [`API_DOCUMENTATION.md`](API_DOCUMENTATION.md) - REST API reference
- [`test/benchmark/results/PAPER_RESULTS.md`](test/benchmark/results/PAPER_RESULTS.md) - Benchmark results

## License

[MIT](LICENSE)

## References

1. Paillier, P. "Public-Key Cryptosystems Based on Composite Degree Residuosity Classes." EUROCRYPT 1999.
2. Pedersen, T.P. "Non-Interactive and Information-Theoretic Secure Verifiable Secret Sharing." CRYPTO 1991.
3. Bernhard, D., Pereira, O., Warinschi, B. "How Not to Prove Yourself: Pitfalls of the Fiat-Shamir Heuristic." ASIACRYPT 2012.
4. Liu, J.K., Wei, V.K., Wong, D.S. "Linkable Spontaneous Anonymous Group Signature for Ad Hoc Groups." ACISP 2004.
5. Pointcheval, D., Stern, J. "Security Proofs for Signature Schemes." EUROCRYPT 1996.
6. NIST. "Module-Lattice-Based Key-Encapsulation Mechanism Standard (ML-KEM / Kyber)." FIPS 203, 2024.
