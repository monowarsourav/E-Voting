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
- Paper Reference: Theorem 6 (Composition Security via hybrid argument)
