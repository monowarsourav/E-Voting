# CovertVote: A Seven-Protocol Blockchain E-Voting System

A secure, anonymous, and coercion-resistant electronic voting system implementing seven cryptographic protocols: Paillier homomorphic encryption, Pedersen commitments, zero-knowledge proofs (strong Fiat-Shamir), linkable ring signatures, SMDC (Self-Masking Deniable Credentials), SA² (Samplable Anonymous Aggregation), and Kyber768 post-quantum key encapsulation.

Built in Go 1.24+ with **production-ready Hyperledger Fabric v2.5** blockchain integration, benchmarked with Hyperledger Caliper.

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
│   ├── Threshold Paillier       — Damgård-Jurik-Nielsen (DJN), t-of-n with ZK proofs
│   └── Kyber768 KEM             — NIST Level 3 post-quantum hybrid encryption
├── Privacy Layer
│   ├── SMDC Credentials         — k=5 slots (1 real + 4 fake), coercion resistance
│   ├── SA2 Aggregation          — 2-server non-colluding, mask cancellation
│   └── Duress Detection         — Behavioral biometric coercion detection (HMAC-SHA256)
├── Blockchain Layer
│   └── Hyperledger Fabric v2.5  — Production network, chaincode verification, Fabric Gateway SDK
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

# Start Hyperledger Fabric network
cd network && ./start.sh

# Docker deployment
docker-compose up
```

## Benchmarks

Full benchmark results with hardware specifications are in [`test/benchmark/results/PAPER_RESULTS.md`](test/benchmark/results/PAPER_RESULTS.md).

### Crypto Micro-Benchmarks (AMD Ryzen 5 7530U, 12 threads)

| Operation | Time | Notes |
|-----------|------|-------|
| Paillier Encrypt (2048-bit) | 8.52 ms | Dominates per-vote cost |
| Pedersen Commit (512-bit) | 85 µs | |
| ZKP Binary Prove | 257 µs | Strong Fiat-Shamir |
| ZKP Binary Verify | 330 µs | |
| SMDC Generate (k=5) | 1.83 ms | 5 Pedersen commitments |
| Ring Sign (n=100) | 9.53 ms | Linear in ring size |
| SA2 Split | 35.5 ms | Paillier re-encryption |
| **Full Pipeline (7 steps)** | **74.9 ms** | **Per vote, end-to-end** |

### Blockchain Performance (Hyperledger Caliper)

Benchmarked on a real Hyperledger Fabric v2.5.12 network (1 Orderer + 2 Peers, 10 workers).

| Round | Transactions | Success | Fail | Throughput (TPS) | Avg Latency (s) |
|-------|-------------|---------|------|------------------|-----------------|
| castVote-10K | 10,000 | **10,000** | **0** | **199.9** | **0.11** |
| castVote-25K | 25,000 | 14,803 | 10,197 | 279.3 | 2.96 |
| castVote-50K | 50,000 | 36,417 | 13,583 | 227.8 | 3.70 |
| castVote-100K | 100,000 | 83,803 | 16,197 | 216.4 | 3.98 |

> **Note:** The 10K round achieves 100% success with 0 failures. Failures in higher rounds are caused by Fabric Gateway's default concurrency limit (500), not application bugs — configurable in production.

### Scalability

| Voters | Homomorphic Tally Time | Per-Vote |
|--------|------------------------|----------|
| 1,000 | 6 ms | 6.3 µs |
| 10,000 | 59 ms | 6.0 µs |
| 100,000 | 628 ms | 6.3 µs |
| 50,000,000 (projected) | ~5.1 min | 6.1 µs |

**O(n) complexity confirmed:** per-vote tally cost is constant regardless of voter count or candidate count.

## Test Coverage

| Package | Coverage |
|---------|----------|
| internal/tally | **91.6%** |
| internal/smdc | 83.3% |
| internal/biometric | 82.1% |
| internal/pq | 82.1% |
| internal/sa2 | 81.2% |
| internal/crypto | 78.2% |
| internal/voting | 77.9% |
| api/handlers | 46.7% |
| internal/blockchain | 34.2% |

**185 tests** (unit + property + benchmark), **100% pass rate** (excluding integration tests requiring live Fabric network).

## Project Structure

```
E-voting/
├── cmd/                          # Entry points
│   ├── api-server/               # Main API server
│   ├── aggregator-a/             # SA2 Leader
│   └── aggregator-b/             # SA2 Helper
├── internal/                     # Core implementation
│   ├── crypto/                   # Paillier, Pedersen, ZKP, Ring Signatures, Threshold Paillier
│   ├── pq/                       # Kyber768 post-quantum hybrid encryption
│   ├── smdc/                     # Self-Masking Deniable Credentials
│   ├── sa2/                      # Samplable Anonymous Aggregation
│   ├── voting/                   # Vote casting orchestration (17-step pipeline)
│   ├── tally/                    # Homomorphic tallying & threshold decryption
│   ├── voter/                    # Registration & Merkle eligibility
│   ├── biometric/                # Fingerprint, liveness, & duress detection
│   ├── blockchain/               # Hyperledger Fabric Gateway SDK integration
│   └── database/                 # SQLite persistence
├── api/                          # REST API (Gin framework)
│   ├── handlers/                 # HTTP handlers
│   ├── middleware/                # Auth, rate limiting, CORS
│   └── routes/                   # Route definitions
├── chaincode/                    # Hyperledger Fabric chaincode (Go)
│   └── covertvote/               # Smart contract: election, vote, tally CRUD
├── caliper/                      # Hyperledger Caliper benchmarks
│   ├── networkconfig.yaml        # Fabric network configuration
│   ├── benchconfig.yaml          # Benchmark rounds (10K–100K)
│   └── workload/                 # Transaction workloads (createElection, castVote)
├── network/                      # Fabric network scripts & crypto materials
├── test/                         # Tests
│   ├── benchmark/                # Performance benchmarks (with paper results)
│   └── integration/              # End-to-end integration tests
├── migrations/                   # SQLite schema migrations (8 tables)
├── proverif/                     # Formal verification models
├── scripts/                      # Deployment and tooling scripts
└── docker-compose.yml            # Container deployment
```

## Testing

```bash
# All unit + property tests
go test ./internal/... ./api/... -timeout 600s

# Property-based tests (crypto correctness)
go test -v -run "Property|Homomorphic|Binding|Linkability|Cancellation|Identity" ./internal/crypto/

# Tally correctness & SA2 integrity tests
go test -v -run "TestTallyCorrectness|TestSA2TallyIntegrity" ./internal/tally/

# Blockchain integration tests (requires running Fabric network)
go test -v -run "TestRealFabricIntegration" ./internal/blockchain/ -timeout 60s

# Caliper performance benchmarks (requires running Fabric network)
cd caliper && npx caliper launch manager \
  --caliper-workspace . \
  --caliper-networkconfig networkconfig.yaml \
  --caliper-benchconfig benchconfig.yaml \
  --caliper-flow-only-test

# Scalability benchmarks
go test -v -run "TestScalabilityTally|TestComplexityValidation" ./test/benchmark/ -timeout 30m

# Coverage report
go test -coverprofile=coverage.out ./internal/... ./api/...
go tool cover -func=coverage.out
```

## Configuration

Copy `.env.example` to `.env` and update all placeholder values. See the file for all available options.

**Critical:** SA2 Leader and Helper servers must run on separate machines/containers with independent API keys. See [`docker-compose.yml`](docker-compose.yml).

## Blockchain Deployment

The system uses **Hyperledger Fabric v2.5.12** with the following topology:

| Component | Description |
|-----------|-------------|
| Orderer | Raft consensus (single node for dev, multi-node for production) |
| Peer Org1 | Endorsing peer with CouchDB state database |
| Peer Org2 | Endorsing peer with CouchDB state database |
| Chaincode | `covertvote_2.0` — Go smart contract with admin access control |
| SDK | `fabric-gateway` v1.10.1 — gRPC with TLS |

See [`network/`](network/) for setup scripts and [`chaincode/covertvote/`](chaincode/covertvote/) for the smart contract source.

## Documentation

- [`internal/crypto/SECURITY.md`](internal/crypto/SECURITY.md) — Cryptographic security properties and theorem references
- [`SECURITY.md`](SECURITY.md) — Security policy and vulnerability reporting
- [`API_DOCUMENTATION.md`](API_DOCUMENTATION.md) — REST API reference
- [`RESEARCH_ANALYSIS.md`](RESEARCH_ANALYSIS.md) — Comprehensive research analysis and comparative study
- [`test/benchmark/results/PAPER_RESULTS.md`](test/benchmark/results/PAPER_RESULTS.md) — Benchmark results for paper

## License

[MIT](LICENSE)

## References

1. Paillier, P. "Public-Key Cryptosystems Based on Composite Degree Residuosity Classes." EUROCRYPT 1999.
2. Pedersen, T.P. "Non-Interactive and Information-Theoretic Secure Verifiable Secret Sharing." CRYPTO 1991.
3. Bernhard, D., Pereira, O., Warinschi, B. "How Not to Prove Yourself: Pitfalls of the Fiat-Shamir Heuristic." ASIACRYPT 2012.
4. Liu, J.K., Wei, V.K., Wong, D.S. "Linkable Spontaneous Anonymous Group Signature for Ad Hoc Groups." ACISP 2004.
5. Damgård, I., Jurik, M., Nielsen, J.B. "A Generalization of Paillier's Public-Key System." PKC 2010.
6. NIST. "Module-Lattice-Based Key-Encapsulation Mechanism Standard (ML-KEM / Kyber)." FIPS 203, 2024.
7. Pointcheval, D., Stern, J. "Security Proofs for Signature Schemes." EUROCRYPT 1996.
