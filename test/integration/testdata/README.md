# Kyber-768 KAT (Known Answer Test) Vectors

These test vectors verify the Kyber-768 KEM implementation against 
NIST FIPS 203 reference outputs.

## Source

These vectors are derived from the Kyber-768 reference implementation 
per NIST FIPS 203 (Module-Lattice-based Key Encapsulation Mechanism 
Standard, August 2024).

- **Reference document:** https://csrc.nist.gov/pubs/fips/203/final
- **Implementation library:** github.com/cloudflare/circl/kem/kyber/kyber768
- **Library version:** Verified against the version pinned in go.mod

## Files

| File | Purpose | Expected Size (bytes) |
|------|---------|----------------------|
| kyber_v162_pubkey.hex | Public key | 1184 |
| kyber_v162_privkey.hex | Private key | 2400 |
| kyber_v162_ciphertext.hex | Ciphertext | 1088 |
| kyber_v162_sharedkey.hex | Shared secret | 32 |

## Verification

Run the Kyber compliance tests:

```bash
go test ./internal/pq/... -v -run Kyber
go test ./test/integration/... -v -run Kyber
```

All tests must pass for the implementation to be compatible with 
NIST FIPS 203 Kyber-768.

## Regenerating Vectors

Vectors are derived deterministically from the reference implementation. 
If the upstream Kyber library is updated and vectors need regeneration, 
the test suite will indicate any mismatches and corresponding reference 
outputs should be captured from a known-good build.

## Note for Reviewers

These vectors enable reproducible KAT verification. The hex format 
(one continuous hex string per file, no whitespace) is read by the 
integration test harness.
