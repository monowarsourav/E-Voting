# CovertVote ProVerif Formal Verification

## Overview
This directory contains ProVerif models for formal verification of CovertVote's security properties. Each property is in a separate file because ProVerif's diff-equivalence mode (for privacy) is incompatible with query mode (for correspondence properties).

## Files
- `privacy.pv` — Ballot Privacy via observational equivalence (diff-equivalence)
- `verifiability.pv` — Individual Verifiability via correspondence property
- `eligibility.pv` — Voter Eligibility via correspondence property
- `run_verification.sh` — Script to run all 3 checks

## Security Properties Verified
1. **Ballot Privacy** — Observational equivalence (diff-equivalence): adversary cannot distinguish which voter cast which vote
2. **Individual Verifiability** — Correspondence property: every recorded ballot was cast by a legitimate voter
3. **Eligibility** — Correspondence property: only registered voters can cast valid ballots

## What ProVerif CANNOT Verify (by design)
- Paillier homomorphic tally correctness (requires algebraic reasoning)
- Ring signature linkability (requires group theory)
- Kyber768 post-quantum security (requires lattice assumptions)
- SMDC k-slot indistinguishability (requires computational indistinguishability)

These are covered by pen-and-paper proofs in the paper (Section 6).

## How to Run

```bash
eval $(opam env)
cd proverif/

# Run all 3 checks
bash run_verification.sh

# Or run individually:
proverif privacy.pv          # Ballot Privacy
proverif verifiability.pv    # Individual Verifiability
proverif eligibility.pv      # Voter Eligibility
```

### Expected Results
| Property | File | Expected Output |
|----------|------|-----------------|
| Ballot Privacy | privacy.pv | `Observational equivalence is true` |
| Verifiability | verifiability.pv | `event(BallotRecorded(b)) ==> inj-event(VoterCastBallot(pkV,b)) is true` |
| Eligibility | eligibility.pv | `event(ValidBallotCast(pkV)) ==> event(VoterRegistered(pkV)) is true` |

## References
- Baloglu et al., "Election Verifiability in Receipt-free Voting Protocols" (ProVerif models: github.com/sbaloglu/proverif-codes)
- Cortier et al., "Machine-checked proofs for electronic voting: Privacy and verifiability for Belenios" (CSF 2018)
- Blanchet, "ProVerif: Cryptographic Protocol Verifier in the Formal Model" (https://proverif.inria.fr)

## Paper Citation
When referencing this verification in the paper:
"We formally verify ballot privacy via observational equivalence, individual verifiability and eligibility via correspondence properties using ProVerif [Blanchet, 2001]. The model abstracts Paillier encryption as IND-CPA secure public-key encryption and SA2 as ideal secret sharing. Homomorphic and ring signature properties are proven separately via game-based reductions (Section 6)."
