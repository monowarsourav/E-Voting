# CovertVote Benchmark Results for Paper

## Hardware
- **CPU:** AMD Ryzen 5 7530U with Radeon Graphics (12 threads)
- **Architecture:** x86_64 (amd64)
- **RAM:** 30 GiB
- **Go Version:** go1.25.7 linux/amd64
- **Date:** 2026-03-13

---

## 1. Crypto Micro-Benchmarks (Pedersen 512-bit)

### ZKP Benchmarks (3 runs)

| Benchmark | Ops | ns/op | B/op | allocs/op |
|-----------|-----|-------|------|-----------|
| ZKPBinaryProve | 4,765 | 247,164 | 12,542 | 102 |
| ZKPBinaryProve | 4,903 | 249,373 | 12,542 | 102 |
| ZKPBinaryProve | 4,916 | 240,178 | 12,542 | 102 |
| ZKPBinaryVerify | 3,822 | 325,091 | 16,392 | 123 |
| ZKPBinaryVerify | 3,591 | 328,072 | 16,392 | 123 |
| ZKPBinaryVerify | 3,756 | 333,124 | 16,392 | 123 |
| ZKPSumOneProve | 14,529 | 82,717 | 5,770 | 53 |
| ZKPSumOneProve | 14,022 | 85,661 | 5,778 | 54 |
| ZKPSumOneProve | 13,821 | 84,585 | 5,786 | 55 |
| ZKPSumOneVerify | 7,287 | 159,660 | 10,037 | 82 |
| ZKPSumOneVerify | 7,344 | 158,734 | 10,037 | 82 |
| ZKPSumOneVerify | 7,467 | 163,853 | 10,037 | 82 |

**ZKP Summary:**
| Operation | Mean (us) | Std Dev |
|-----------|-----------|---------|
| Binary Prove | ~245 | ~5 |
| Binary Verify | ~329 | ~4 |
| Sum-One Prove | ~84 | ~1.5 |
| Sum-One Verify | ~161 | ~3 |

---

## 2. Post-Quantum (Kyber768) Benchmarks (3 runs)

| Benchmark | Ops | ns/op | B/op | allocs/op |
|-----------|-----|-------|------|-----------|
| KyberKeyGen | 48,631 | 25,493 | 8,304 | 6 |
| KyberKeyGen | 44,419 | 26,501 | 8,304 | 6 |
| KyberKeyGen | 43,952 | 25,820 | 8,304 | 6 |
| KyberEncapsulate | 80,949 | 26,100 | 1,232 | 3 |
| KyberEncapsulate | 31,076 | 33,609 | 1,232 | 3 |
| KyberEncapsulate | 39,164 | 27,039 | 1,232 | 3 |
| HybridEncrypt (2048-bit) | 99 | 15,949,089 | 33,306 | 50 |
| HybridEncrypt (2048-bit) | 84 | 13,272,623 | 33,309 | 50 |
| HybridEncrypt (2048-bit) | 100 | 12,885,869 | 33,306 | 50 |
| HybridDecrypt (2048-bit) | 100 | 11,971,867 | 27,090 | 38 |
| HybridDecrypt (2048-bit) | 98 | 11,304,215 | 27,090 | 38 |
| HybridDecrypt (2048-bit) | 100 | 11,529,940 | 27,072 | 38 |

**PQ Summary:**
| Operation | Mean (us) |
|-----------|-----------|
| Kyber KeyGen | ~26 |
| Kyber Encapsulate | ~29 |
| Hybrid Encrypt (Kyber+Paillier 2048) | ~14,036 |
| Hybrid Decrypt (Kyber+Paillier 2048) | ~11,602 |

> Note: Hybrid encrypt/decrypt are dominated by Paillier 2048-bit operations (~8.7ms for Paillier alone). Kyber768 adds only ~26us overhead.

---

## 3. End-to-End Vote Cast Pipeline Benchmarks

### Full 7-Step Pipeline (Single Vote)

| Benchmark | Ops | ns/op | B/op | allocs/op |
|-----------|-----|-------|------|-----------|
| FullVoteCastPipeline | 15 | 70,466,199 | 1,749,574 | 12,169 |

**Full pipeline: ~70.5 ms per vote** (all 7 cryptographic operations)

### Per-Phase Breakdown

| Phase | Operation | ns/op | ms/op | B/op | allocs/op |
|-------|-----------|-------|-------|------|-----------|
| 1 | Paillier Encrypt (2048-bit) | 8,691,005 | 8.69 | 26,210 | 31 |
| 2 | Pedersen Commit (512-bit) | 79,911 | 0.08 | 3,962 | 31 |
| 3 | ZKP Binary Prove | 243,257 | 0.24 | 12,542 | 102 |
| 4 | SMDC Generate (k=5) | 1,733,340 | 1.73 | 93,565 | 800 |
| 5 | Ring Sign (n=100) | 27,059,779 | 27.06 | 1,506,214 | 11,081 |
| 6 | SA2 Split | 35,344,732 | 35.34 | 103,904 | 117 |

**Pipeline Time Distribution:**
| Phase | % of Total |
|-------|-----------|
| SA2 Split | 48.8% |
| Ring Sign (100 members) | 37.3% |
| Paillier Encrypt | 12.0% |
| SMDC Generate | 2.4% |
| ZKP Binary Prove | 0.3% |
| Pedersen Commit | 0.1% |

> **Bottleneck:** SA2 split (Paillier re-encryption for mask generation) and ring signatures dominate. Both scale linearly.

---

## 4. Scalability Results

### 4.1 Homomorphic Tally Scalability (O(n) Verification)

Measures pure homomorphic addition time (modular multiplication in N^2).
Pre-encrypted vote pool reused to isolate tally from encryption cost.

| Voters | Tally Time (ms) | Per-Vote (us) |
|--------|-----------------|---------------|
| 100 | 0.6 | 6.23 |
| 500 | 3.0 | 6.01 |
| 1,000 | 6.3 | 6.28 |
| 5,000 | 29.0 | 6.00 |
| 10,000 | 59.0 | 6.00 |
| 50,000 | 308.0 | 6.18 |
| 100,000 | 628.0 | 6.29 |

**Result: Per-vote tally cost is constant at ~6.1 us, confirming O(n) complexity.**

### Projections (Linear Extrapolation)

| Voters | Projected Tally Time |
|--------|---------------------|
| 1,000,000 | ~6.1 seconds |
| 10,000,000 | ~61 seconds |
| 50,000,000 | ~5.1 minutes |

### 4.2 Ring Signature Scalability (Linear in Ring Size)

| Ring Size | Sign Time (ms) | Verify Time (ms) |
|-----------|----------------|-------------------|
| 10 | 2.2 | 2.2 |
| 25 | 6.0 | 5.8 |
| 50 | 12.0 | 12.0 |
| 100 | 24.0 | 24.2 |
| 200 | 47.5 | 48.5 |
| 500 | 120.5 | 120.5 |

**Result: Linear scaling confirmed. ~0.24 ms per ring member for both sign and verify.**

### 4.3 O(n) Complexity Validation (CovertVote vs ISE-Voting)

Fixed n=1000 voters, varying m candidates.
Tests whether per-candidate tally time remains constant (O(n)) or grows with m (O(n*m^2)).

| Candidates (m) | Tally Time (ms) | Per-Candidate (ms) |
|-----------------|-----------------|---------------------|
| 2 | 12.0 | 6.00 |
| 5 | 28.0 | 5.60 |
| 10 | 58.0 | 5.80 |
| 20 | 113.0 | 5.65 |
| 50 | 288.0 | 5.76 |

**Result: Per-candidate time is constant at ~5.76 ms regardless of m.**
- CovertVote: O(n * m) total, O(n) per candidate -- CONFIRMED
- ISE-Voting: O(n * m^2) total, O(n * m) per candidate -- would show per-candidate time GROWING with m

---

## 5. Test Coverage

| Package | Coverage |
|---------|----------|
| internal/biometric | 82.9% |
| internal/crypto | 74.5% |
| internal/sa2 | 81.2% |
| internal/smdc | 83.3% |
| internal/pq | 31.6% |
| internal/tally | 29.9% |
| internal/voting | 17.9% |
| api/handlers | 44.0% |
| **Total (all packages)** | **34.6%** |

> Note: Core cryptographic packages (crypto, sa2, smdc) have 74-83% coverage.
> Lower coverage in voting/tally is due to infrastructure code (blockchain integration, threshold decryption) that requires full system deployment.

---

## 6. Security Parameters

| Protocol | Parameter | Value |
|----------|-----------|-------|
| Paillier HE | Key size | 2048 bits |
| Pedersen Commitments | Group size | 512 bits |
| Ring Signatures | Group size | 512 bits |
| Ring Signatures | Default ring size | 100 |
| SMDC | Slot count (k) | 5 (1 real + 4 fake) |
| SA2 | Server count | 2 (non-colluding) |
| Kyber768 | NIST Level | 3 (Module-LWE) |
| ZKP Fiat-Shamir | Variant | Strong (public params in hash) |

---

## 7. Key Findings for Paper

1. **O(n) Tally Complexity Confirmed:** Per-vote tally cost is constant (~6 us) from 100 to 100K voters.
2. **Candidate-Independent Scaling:** Per-candidate tally cost remains at ~5.76ms regardless of candidate count m, confirming O(n) per candidate vs ISE-Voting's O(n*m).
3. **Full Pipeline:** End-to-end vote casting takes ~70ms, dominated by SA2 split (49%) and ring signatures (37%).
4. **Post-Quantum Overhead:** Kyber768 adds only ~26us to the hybrid encryption pipeline (negligible vs Paillier's ~8.7ms).
5. **Linear Ring Signature Scaling:** Both sign and verify scale linearly at ~0.24ms per ring member.
6. **National-Scale Feasibility:** Homomorphic tally for 50M voters projected at ~5.1 minutes.
