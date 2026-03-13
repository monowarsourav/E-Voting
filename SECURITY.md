# Security Policy

## Cryptographic Security Properties

CovertVote implements seven cryptographic protocols with formally analyzed security properties.
For detailed security assumptions, theorem references, and parameter justifications, see:

- [`internal/crypto/SECURITY.md`](internal/crypto/SECURITY.md) - Full security properties documentation

## Security Parameters

| Protocol | Parameter | Value | Security Level |
|----------|-----------|-------|----------------|
| Paillier HE | Key size | 2048 bits | 112-bit classical |
| Pedersen Commitments | Group size | 512 bits | DLP-based |
| Ring Signatures | Group size | 512 bits | DLP-based |
| Kyber768 | NIST Level | 3 | 192-bit post-quantum |
| ZKP Fiat-Shamir | Variant | Strong (BPW12) | ROM soundness |

## Threat Model

- **Adversary:** Dolev-Yao network adversary (full control of communication channels)
- **SA2 Assumption:** Non-colluding 2-server model (Leader and Helper must run on separate infrastructure)
- **Coercion Model:** SMDC provides resistance against coercive adversaries who can observe but not modify voter actions

## Reporting Vulnerabilities

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Contact the maintainers directly via university email
3. Allow reasonable time for a fix before public disclosure

## Deployment Considerations

- SA2 Leader and Helper servers **MUST** run on separate machines/containers
- All `.env` values must be changed from defaults before deployment
- Paillier key size minimum of 2048 bits is enforced in code
- TLS should be used for all network communication in production
