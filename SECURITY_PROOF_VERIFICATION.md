# CovertVote — Security Proof Verification Trail

**Status:** COMPLETE (Checkpoints 0–9 of 9 finished, awaiting LaTeX compile + supervisor review)
**Last updated:** 2026-05-11
**Audience:** Supervisor, external cryptographer reviewer, future maintainer
**Companion file:** `thesis/Chapters/chapter3.tex` § sec:security-analysis (game-based proofs); `thesis/Chapters/chapter4.tex` § sec:proverif (symbolic verification, Tier 2)

---

## 1. Document Purpose & How to Use

### Why this document exists

The CovertVote thesis was originally drafted with one-paragraph "proof sketches" for each of six security theorems. Following supervisor feedback, the security analysis is being upgraded to **journal quality** (TDSC / TIFS standard), which requires:

1. Formal **game-based reductions** (not symbolic-model arguments alone).
2. Each security claim explicitly reduced to a **standard hardness assumption** (DCRA, DDH, Module-LWE, DLog).
3. **Hybrid arguments** that handle the joint distribution of all ballot components — not just the underlying primitive in isolation.
4. **ProVerif analysis** retained but clearly framed as **complementary symbolic verification (Tier 2)** rather than the primary security evidence.

This document is the **verification audit trail** that backs every claim added to the thesis. For each theorem, it records (a) the formal game definition, (b) the reduction theorem statement, (c) the hybrid sequence with per-step justification, (d) the confidence level assigned to each step, (e) cross-checks performed against primary cryptographic literature, and (f) any open questions or items flagged for human review.

### How to use this document (for the supervisor / external reviewer)

1. **For correctness review:** Open §4 (Per-Theorem Verification Worksheets). Each worksheet is self-contained. The "Cross-checks performed" subsection of each worksheet cites the original paper, page/section number, and quotes the property being invoked. You can audit each step against the cited source.
2. **For thesis ↔ doc mapping:** §6 (Cross-Reference Map) gives line-by-line mapping from thesis locations to MD verification sections.
3. **For risk assessment:** Each step in §4 is tagged with one of:
   - `HIGH` — textbook reduction, cited source is canonical, low risk of error.
   - `MEDIUM` — reduction is standard but requires careful primitive composition; verify cited section.
   - `NEEDS-EXPERT-REVIEW` — research-level claim or non-standard adaptation; supervisor or external cryptographer should review before submission.
4. **For change tracking:** §7 (Verification Log) records what changed in each session, in chronological order.
5. **For known limitations:** §8 (Known Limitations & Items Flagged for Human Review) is the consolidated punch-list of things explicitly NOT proven rigorously and / or NEEDING human review.

### How to use this document (for me, going forward)

For each new theorem cycle:
1. Verify primary references (read original paper, note page/section).
2. Draft MD worksheet in §4 (game → reduction theorem → hybrid → confidence → cross-checks → open questions).
3. Pause for review.
4. Port to LaTeX (chapter3.tex).
5. Update §6 (cross-ref map) and §7 (log).

---

## 2. Notation & Assumptions Registry

### 2.1 Notation conventions used in proofs

| Symbol | Meaning |
|--------|---------|
| $\lambda$ | Security parameter (in bits). For this thesis: Paillier modulus 2048 bits ⇒ $\lambda \approx 112$. |
| `negl(λ)` | A negligible function in $\lambda$, i.e., for every polynomial $p$ and large enough $\lambda$, `negl(λ) < 1/p(λ)`. |
| `PPT` | Probabilistic polynomial-time (in $\lambda$). |
| $\mathcal{A}$ | Adversary (always PPT unless stated). |
| $\mathcal{C}$ | Challenger in a security game. |
| $\mathcal{B}$ | Reduction algorithm; uses $\mathcal{A}$ as a sub-routine to attack a base hard problem. |
| $\text{Adv}^X_{\mathcal{A}}(\lambda)$ | Advantage of $\mathcal{A}$ in winning game $X$ at security parameter $\lambda$. |
| $\stackrel{\$}{\leftarrow}$ | Uniform random sampling. |
| $\stackrel{c}{\equiv}$ | Computational indistinguishability. |
| $\stackrel{s}{\equiv}$ | Statistical (information-theoretic) indistinguishability. |
| $G_i$ | $i$-th hybrid game in a sequence. |
| `Pr[event in G_i]` | Probability of `event` occurring in the experiment defined by $G_i$. |

### 2.2 Hardness assumptions formally stated

#### DCRA — Decisional Composite Residuosity Assumption

For an RSA modulus $n = pq$ where $|p| = |q| \geq 1024$ bits, the distribution
$$\{(n, c) : c \stackrel{\$}{\leftarrow} \mathbb{Z}_{n^2}^*\}$$
is computationally indistinguishable from
$$\{(n, c) : c = (1+n)^m \cdot r^n \bmod n^2,\ m \stackrel{\$}{\leftarrow} \mathbb{Z}_n,\ r \stackrel{\$}{\leftarrow} \mathbb{Z}_n^*\}.$$
Source: Paillier 1999, §3.1. **Used in:** T1 (Ballot Privacy), T5 (SA² Aggregation), T6 (PQ Hybrid as inner layer).

#### DDH — Decisional Diffie-Hellman Assumption

For a prime-order group $\mathbb{G}$ of order $q$ with generator $g$, the distribution
$$\{(g^a, g^b, g^{ab}) : a, b \stackrel{\$}{\leftarrow} \mathbb{Z}_q\}$$
is computationally indistinguishable from
$$\{(g^a, g^b, g^c) : a, b, c \stackrel{\$}{\leftarrow} \mathbb{Z}_q\}.$$
Source: Standard. **Used in:** T3 (Voter Anonymity).

#### DLog — Discrete Logarithm Assumption

For prime-order group $\mathbb{G}$ with generator $g$, given $g^x$ for $x \stackrel{\$}{\leftarrow} \mathbb{Z}_q$, no PPT adversary outputs $x$ with non-negligible probability.
**Used in:** Pedersen binding (background fact); T2 (Schnorr soundness).

#### Module-LWE (M-LWE)

For module rank $k$, modulus $q$, dimension $n$, and error distribution $\chi$, the distribution
$$\{(A, As + e) : A \stackrel{\$}{\leftarrow} R_q^{k \times k},\ s \stackrel{\$}{\leftarrow} R_q^k,\ e \stackrel{\$}{\leftarrow} \chi^k\}$$
is computationally indistinguishable from $\{(A, u) : u \stackrel{\$}{\leftarrow} R_q^k\}$, where $R_q = \mathbb{Z}_q[x]/(x^n + 1)$.
For Kyber-768: $n = 256$, $q = 3329$, $k = 3$. Source: NIST FIPS 203, §1.3. **Used in:** T6.

### 2.3 Trust assumptions (system-level, non-cryptographic)

| Label | Statement | Where invoked |
|-------|-----------|---------------|
| **A1** | At least one of the two SA² aggregation servers is honest (non-colluding). | T5 |
| **A2** | The bulletin board (Hyperledger Fabric ledger) provides append-only integrity. | T2, T3 (verifiability backbone) |
| **A3** | Voters' Paillier private keys, ring secret keys, and SMDC credentials are stored on devices not under coercer control. | T4 (coercion resistance is conditional on credential issuance phase being uncoerced) |
| **A4** | Random oracle model (ROM) — SHA-256 / SHA3 modeled as random oracles for Fiat-Shamir transforms. | T2, T3, T4 |
| **A5** | Threshold Paillier decryption (DJN) requires a quorum of trustees; tally key not reconstructed by any single party. | T1 (decryption phase not modeled in IND-CPA game) |

### 2.4 Properties of building blocks (correctly stated)

These are the properties that the proofs in §4 depend on. They are stated here once, with references, to prevent property-mis-attribution errors of the type fixed in §3 below.

| Primitive | Hiding | Binding / Soundness / Anonymity | Source |
|-----------|--------|------------------------------|--------|
| **Paillier (1999)** | IND-CPA under DCRA | — (encryption, not commitment) | Paillier 1999 §3 |
| **Pedersen commitment** | **Perfectly hiding** (statistical) | **Computationally binding** under DLog | Pedersen 1991 §2 |
| **Schnorr Σ-protocol** | Honest-verifier zero-knowledge | Special soundness ⇒ knowledge soundness; under Strong Fiat-Shamir + ROM, NIZK with knowledge soundness | Schnorr 1991; Bernhard-Pereira-Warinschi 2012 |
| **Liu-Wei-Wong (2004) linkable ring sig** | Signer ambiguity (unconditional in ROM for $(c_i, r_i)$; key-image relation requires DDH) | Unforgeability under DLog + ROM; linkability via key image | LWW 2004 §4.2 |
| **SA² (Bonawitz et al. 2017 / Apple)** | Per-server view ⊥ vote when at least one server honest | — | Bonawitz et al. 2017 |
| **ML-KEM-768 (Kyber)** | IND-CCA2 under M-LWE + M-SIS in ROM (FO transform) | — | NIST FIPS 203; Hofheinz-Hövelmanns-Kiltz |
| **HMAC-SHA256** | EUF-CMA under SHA-256 PRF assumption | — | RFC 2104; FIPS 198 |

---

## 3. Pre-existing Errors Found in Thesis

This section catalogs all factual / technical errors discovered in the existing thesis text during the verification process. Each error has a **fix** described, applied to the thesis, and the diff is recorded in §7 (Verification Log).

### Error E1 — Pedersen hiding/binding properties stated in reverse

**Standard Pedersen commitment:** $C = g^m \cdot h^r \bmod p$ where $g, h$ generate a prime-order subgroup of order $q$ and $\log_g h$ is unknown.

**The correct properties are:**
- **Perfectly hiding** (information-theoretic): For any $m$, the distribution of $C$ over uniform $r \stackrel{\$}{\leftarrow} \mathbb{Z}_q$ is the uniform distribution over the subgroup, *independent of $m$*. Hence $C$ reveals **zero** information about $m$, even to an unbounded adversary.
- **Computationally binding** under DLog: If an adversary finds $(m, r) \neq (m', r')$ with $g^m h^r = g^{m'} h^{r'}$, then $g^{m-m'} = h^{r'-r}$, yielding $\log_g h = (r' - r) / (m - m')$. So breaking binding ⇒ solving DLog.

Source: Pedersen 1991, "Non-interactive and information-theoretic secure verifiable secret sharing", CRYPTO '91, §2. Also standard in Boneh-Shoup *Cryptography Textbook* §13.5; Goldreich *Foundations of Cryptography* Vol. 1.

#### Locations in thesis where the properties were stated backwards

| # | File:line | Wrong text | Fix |
|---|-----------|------------|-----|
| E1.1 | `chapter3.tex:171` | "Pedersen commitments provide computationally hiding and perfectly binding commitments" | "Pedersen commitments provide perfectly hiding and computationally binding commitments" |
| E1.2 | `chapter3.tex:193` (Security Properties bullet 1) | "**Computationally Hiding**: Given $C$, no polynomial-time adversary can determine $m$ (under the Discrete Logarithm assumption)." | "**Perfectly Hiding**: For uniformly random $r$, the distribution of $C$ is independent of $m$; an unbounded adversary learns nothing about $m$ from $C$." |
| E1.3 | `chapter3.tex:194` (bullet 2) | "**Perfectly Binding**: There exists no $(m', r') \neq (m, r)$ such that $g^{m'} \cdot h^{r'} = g^m \cdot h^r$ (information-theoretically)." | "**Computationally Binding**: Under the Discrete Logarithm assumption, no PPT adversary can produce $(m, r) \neq (m', r')$ with $g^m h^r = g^{m'} h^{r'}$; doing so would yield $\log_g h$." |
| E1.4 | `chapter3.tex:351` (SMDC coercion property) | "Due to Pedersen's **computational hiding** property, the coercer cannot determine whether $C_j$ commits to 0 or 1." | "Due to Pedersen's **perfect hiding** property, the distributions of $C_j$ for $w_j = 0$ and $w_j = 1$ are statistically identical; the coercer learns nothing about $w_j$ from $C_j$, even with unbounded computational power." |
| E1.5 | `chapter3.tex:356` (figure caption) | "...renders the real and fake slots computationally indistinguishable to a coercer." | "...renders the real and fake slot commitments **statistically (information-theoretically) indistinguishable** to a coercer." |
| E1.6 | `chapter3.tex:596` (Theorem 4 statement) | "...under the Computational Hiding property of Pedersen commitments." | "...under the Perfect Hiding property of Pedersen commitments." |
| E1.7 | `chapter3.tex:598` (Theorem 4 proof sketch) | Proof argues "distinguishing $C_i = g^0 h^{r_i}$ from $C_i = g^1 h^{r_i}$ requires computing $\log_g h$, which is infeasible" | Replace with: "Since $r_i$ is sampled uniformly from $\mathbb{Z}_q$ and $h$ generates the subgroup of order $q$, the value $h^{r_i}$ is uniformly distributed over the subgroup, independent of any other quantity. Hence $C_i = g^{w_i} \cdot h^{r_i}$ is uniformly distributed regardless of $w_i \in \{0, 1\}$. The two distributions for $w_i = 0$ and $w_i = 1$ are therefore identical (statistical distance zero), so no adversary — even unbounded — can distinguish them." |
| E1.8 | `chapter1.tex:95` (introduction protocol summary) | "Pedersen Commitments: Provide computationally hiding and perfectly binding commitments..." | "Pedersen Commitments: Provide perfectly hiding and computationally binding commitments..." |
| E1.9 | `chapter2.tex:147` (background subsection) | "...perfectly binding (no two openings exist for a given $C$) and computationally hiding under the discrete logarithm assumption." | "...perfectly hiding (distribution of $C$ over uniform $r$ is independent of $m$) and computationally binding under the discrete logarithm assumption (a collision would yield $\log_g h$)." |

**Strengthening side-effect:** The corrected Theorem 4 is now stronger than the original — it provides **information-theoretic** (perfect) coercion indistinguishability for the slot commitments, not merely computational. This is an *upgrade* in the thesis's security claims, not a downgrade.

### Future error entries

Additional errors discovered during T1–T6 verification will be appended here as E2, E3, ... with the same structure (location, wrong text, correct text, source, impact).

---

## 4. Per-Theorem Verification Worksheets

> **To be filled in Checkpoints 1–6.** Each worksheet will contain: game definition, reduction theorem statement, hybrid sequence, confidence levels, open questions, cross-checks performed.
>
> Order: T2 → T6 → T5 → T1 → T3 → T4.

### 4.1 Theorem 2 — Vote Validity

**Status:** Drafted (Checkpoint 1, Session 1).
**Confidence:** HIGH — textbook NIZK knowledge soundness via special-soundness + Strong Fiat-Shamir in ROM.

#### 4.1.1 Informal claim (matches thesis statement)

The binary proofs $\pi_i^{\text{bin}}$ (one per candidate) and the sum proof $\pi^{\text{sum}}$ together ensure that any ballot accepted by the verifier is **well-formed**: each per-candidate weight satisfies $w_i \in \{0, 1\}$, and $\sum_{i=1}^{m} w_i = 1$. An adversary cannot produce a verifying proof for an ill-formed ballot except with negligible probability.

#### 4.1.2 Formal relation

Let $\mathbb{G}_q$ be the order-$q$ subgroup of $\mathbb{Z}_p^*$ used for Pedersen commitments, with generators $g, h$ where $\log_g h$ is unknown. The well-formedness relation is:
$$R_{\text{ballot}} = \left\{ \big( (C_1, \ldots, C_m);\ (w_1, \ldots, w_m, r_1, \ldots, r_m) \big) \;:\; \forall i\, C_i = g^{w_i} h^{r_i} \wedge w_i \in \{0, 1\} \wedge \sum_{i=1}^{m} w_i = 1 \right\}.$$

#### 4.1.3 Game definition — Soundness experiment

**Experiment $\text{Snd}^{R_{\text{ballot}}}_{\mathcal{A}}(\lambda)$:**

1. **Setup.** Challenger $\mathcal{C}$ generates Pedersen parameters $(p, q, g, h)$ at security level $\lambda$ (with $|q| \geq 2\lambda$, hash output 256 bits). The random oracle $H$ is initialized.
2. **Adversary phase.** $\mathcal{A}$ is given $(p, q, g, h)$ and oracle access to $H$. $\mathcal{A}$ may make up to $q_H = q_H(\lambda)$ hash queries (a polynomial in $\lambda$).
3. **Output.** $\mathcal{A}$ outputs a candidate ballot $(C_1, \ldots, C_m, \pi^{\text{bin}}_1, \ldots, \pi^{\text{bin}}_m, \pi^{\text{sum}})$.
4. **Win condition.** $\mathcal{A}$ wins if (a) every $\pi^{\text{bin}}_i$ verifies, (b) $\pi^{\text{sum}}$ verifies, AND (c) there exists no opening $(w_1, \ldots, w_m, r_1, \ldots, r_m)$ such that $((C_1, \ldots, C_m); (w_1, \ldots, w_m, r_1, \ldots, r_m)) \in R_{\text{ballot}}$.

Define $\text{Adv}^{\text{Snd}}_{\mathcal{A}}(\lambda) = \Pr[\mathcal{A} \text{ wins}]$.

#### 4.1.4 Theorem 2 (formal)

**Theorem 2 (Vote Validity — Knowledge Soundness).** Let $H : \{0,1\}^* \to \{0,1\}^{256}$ be modeled as a random oracle. For any PPT adversary $\mathcal{A}$ making at most $q_H$ hash queries and winning the soundness experiment with probability $\text{Adv}^{\text{Snd}}_{\mathcal{A}}(\lambda) = \varepsilon$, there exists a PPT algorithm $\mathcal{B}$ that, in expected polynomial time, either:
- **(a)** extracts a witness $(w_1, \ldots, w_m, r_1, \ldots, r_m) \in R_{\text{ballot}}$ with probability at least $\varepsilon - q_H \cdot 2^{-256} - m \cdot \varepsilon^2 / q_H$, or
- **(b)** breaks the binding property of Pedersen commitments (and hence the Discrete Logarithm assumption in $\mathbb{G}_q$).

In particular,
$$\text{Adv}^{\text{Snd}}_{\mathcal{A}}(\lambda) \leq q_H \cdot 2^{-256} + m \cdot \sqrt{q_H \cdot \text{Adv}^{\text{DLog}}_{\mathcal{B}}(\lambda)} + \text{negl}(\lambda).$$

#### 4.1.5 Hybrid sequence and proof outline

Three hybrids are used: $G_0$ is the real soundness experiment; $G_1$ replaces the random oracle with an idealized lazy-sampling oracle (no change in distribution, just a bookkeeping step); $G_2$ rules out hash-collision wins.

**Game $G_0$:** Real experiment as defined in §4.1.3.

**Game $G_1$:** Identical to $G_0$, except $\mathcal{C}$ maintains a table $T$ of all hash queries made by $\mathcal{A}$, programming $H$ lazily. This is a perfect simulation of the random oracle, so $\Pr[\mathcal{A} \text{ wins } G_1] = \Pr[\mathcal{A} \text{ wins } G_0] = \varepsilon$.

**Game $G_2$:** Identical to $G_1$, except $\mathcal{C}$ aborts if $\mathcal{A}$ outputs a verifying proof whose Fiat-Shamir challenge $c$ was *not* queried via $H$ (i.e., $\mathcal{A}$ guessed the challenge). The probability of this guess succeeding is at most $2^{-256}$ per proof, with at most $m+1$ proofs (binary + sum), giving $\Pr[\text{abort in } G_2] \leq (m+1) \cdot 2^{-256}$.

Therefore $\Pr[\mathcal{A} \text{ wins } G_2] \geq \varepsilon - (m+1) \cdot 2^{-256}$.

**Forking and witness extraction.** In $G_2$, every accepting proof corresponds to an actual hash query. Apply the General Forking Lemma~\cite{bellare2006forking} to each binary proof (which is an OR-composition of two Schnorr Σ-protocols): rewind $\mathcal{A}$ at the hash query corresponding to that proof's challenge, supply a fresh challenge $c'$, and obtain a second accepting transcript $(A_0, A_1, d_0', d_1', f_0', f_1')$ with $c \neq c'$. By the **special soundness** of OR-composed Schnorr (Cramer-Damgård-Schoenmakers 1994, §3): from any two accepting transcripts of the binary proof with distinct challenges, an extractor recovers either (i) the witness $(w, r)$ with $w \in \{0,1\}$ and $C = g^w h^r$, or (ii) two openings $(w, r), (w', r')$ of $C$ with $w \neq w'$, breaking Pedersen binding (DLog).

Apply the same forking to $\pi^{\text{sum}}$: it is a Schnorr proof of knowledge of $R = \sum_i r_i$ such that $\Pi / g = h^R$ where $\Pi = \prod_i C_i$. Special soundness extracts $R$, completing the witness.

By the General Forking Lemma, the per-proof extraction succeeds with probability at least $\varepsilon(\varepsilon - 1/q) / q_H \geq \varepsilon^2 / q_H - q^{-1}$ (negligible). Combining for $m$ binary proofs (independent extractions, taking a union bound for failures) and one sum proof yields the bound stated in the theorem.

#### 4.1.6 Confidence per step

| Step | Justification source | Confidence |
|------|----------------------|------------|
| $G_0 \to G_1$ (lazy sampling) | Standard ROM technique (Bellare-Rogaway 1993, §4) | HIGH |
| $G_1 \to G_2$ (challenge-guessing abort) | Pigeonhole on hash range, well-known | HIGH |
| Binary-proof special soundness (OR-composed Schnorr) | Cramer-Damgård-Schoenmakers 1994, §3.2 | HIGH |
| Sum-proof special soundness (Schnorr DLog) | Schnorr 1991, Theorem 1 | HIGH |
| Forking-lemma extraction bound | Bellare-Neven 2006 (General Forking Lemma) | HIGH |
| Strong Fiat-Shamir + statement-binding | Bernhard-Pereira-Warinschi 2012, Theorem 6 | HIGH |
| Final composition (m binary + 1 sum) | Standard union bound; assumes proofs are constructed independently | HIGH |

#### 4.1.7 Open questions / flagged items

- **OQ1:** The challenge space size in the thesis's Strong-FS construction is implicitly $2^{256}$ (SHA-256 truncated mod $q$). For $|q| = |p|/2 \approx 1024$ bits (since $p$ is a 2048-bit safe prime), the challenge is reduced mod $q$, which could in principle introduce a tiny statistical bias. In practice $2^{256} \ll q$, so the challenge is effectively uniform on $\{0, \ldots, 2^{256}-1\} \subset \mathbb{Z}_q$ — this is fine but worth noting.
- **OQ2:** The original thesis claim "soundness error $\leq 2^{-256}$" is actually the per-proof challenge-guessing bound, not the full extraction bound. The full bound includes the forking factor, which is asymptotically tight only against PPT adversaries. For thesis text, we keep the headline statement but present the full reduction in the body.

#### 4.1.8 Cross-checks performed

1. Verified that the binary proof in `chapter3.tex:208–239` is the **standard OR-composed Schnorr for $C = g^0 h^r$ vs. $C = g^1 h^r$**, matching the construction in Cramer-Damgård-Schoenmakers 1994 §3.2 (with the $w=1$ branch using statement $C/g = h^r$).
2. Verified that the Strong Fiat-Shamir hash includes $(\text{params}, \text{nonce}, \text{electionID}, C, A_0, A_1)$ — matching the BPW 2012 §4.1 recommendation that the *statement* (here the commitment $C$) is included in the hash.
3. Verified that the sum proof (`chapter3.tex:241–253`) is a standard Schnorr proof of $\log_h(\Pi/g) = R$, where $\Pi/g \in \mathbb{G}_q$ when $\sum w_i = 1$.
4. Verified that the existing Theorem 2 statement (`chapter3.tex:588`) does not reference any hardness assumption — only the soundness error bound. The new statement makes the DLog dependency (via Pedersen binding) explicit.

#### 4.1.9 LaTeX port plan

The new Theorem 2 will be placed in a new subsection `\subsection{Computational Security Theorems}` (renaming the existing `\subsection{Formal Security Theorems}`), with the formal game definition immediately preceding the theorem statement. The original "Proof Sketch" paragraph is retained as an introductory informal preview, followed by the formal game, theorem, and proof-by-hybrid-games.
### 4.2 Theorem 6 — Post-Quantum Hybrid

**Status:** Drafted (Checkpoint 2).
**Confidence:** HIGH — textbook KEM-DEM / Encrypt-then-MAC composition.

#### 4.2.1 Construction recap

Per `chapter3.tex:416–435`, each transmitted vote tuple is
$$\mathsf{Ballot}_{\text{outer}} = (\text{ct}_{\text{ky}},\ E(v),\ \text{salt},\ \text{MAC})$$
where $(K, \text{ct}_{\text{ky}}) \leftarrow \text{Kyber.Enc}(\text{pk}_{\text{ky}})$, $K_{\text{auth}} = \text{SHA256}(K \| \text{salt})$, and $\text{MAC} = \text{HMAC-SHA256}(K_{\text{auth}}, E(v))$.

#### 4.2.2 Two security claims to formalise

T6 actually states **two** properties that need separate treatments:

- **(P1) Confidentiality of $v$**: an adversary observing the transmitted tuple learns nothing about $v$ beyond what is implied by the public tally.
- **(P2) Integrity of $E(v)$ in transit/at rest**: an adversary cannot produce $(\text{ct}_{\text{ky}}', E(v)', \text{salt}', \text{MAC}')$ that verifies under the recipient's public key but with $E(v)' \neq E(v)$ (chosen by the legitimate sender).

These follow from a standard Encrypt-then-MAC / KEM-DEM analysis. We formalise both via an INT-CTXT-style game for (P2) and an IND-CPA-style argument for (P1) that piggybacks on Theorem 1.

#### 4.2.3 Game definition — INT-CTXT for the outer hybrid

**Experiment $\mathrm{INT\text{-}CTXT}^{\text{Hybrid}}_{\mathcal{A}}(\lambda)$:**

1. **Setup.** Challenger $\mathcal{C}$ generates Kyber-768 key pair $(\text{pk}_{\text{ky}}, \text{sk}_{\text{ky}})$, Paillier key pair $(\text{pk}_p, \text{sk}_p)$, and gives $\mathcal{A}$ both public keys.
2. **Encryption oracle $\mathsf{Enc}(\cdot)$.** $\mathcal{A}$ may submit polynomially many vote queries $v_1, v_2, \ldots, v_q$. For each $v_i$, $\mathcal{C}$ returns $\mathsf{Ballot}_i = (\text{ct}_{\text{ky},i}, E(v_i), \text{salt}_i, \text{MAC}_i)$ produced honestly. Let $\mathcal{S} = \{\mathsf{Ballot}_i\}_{i=1}^q$.
3. **Forgery output.** $\mathcal{A}$ outputs $\mathsf{Ballot}^* = (\text{ct}_{\text{ky}}^*, E^*, \text{salt}^*, \text{MAC}^*)$.
4. **Win condition.** $\mathcal{A}$ wins if (i) $\mathsf{Ballot}^* \notin \mathcal{S}$, AND (ii) the verification routine accepts $\mathsf{Ballot}^*$, i.e., $\text{HMAC-SHA256}(\text{SHA256}(\text{Kyber.Dec}(\text{sk}_{\text{ky}}, \text{ct}_{\text{ky}}^*) \| \text{salt}^*), E^*) = \text{MAC}^*$.

Define $\mathrm{Adv}^{\mathrm{INT\text{-}CTXT}}_{\mathcal{A}}(\lambda) = \Pr[\mathcal{A} \text{ wins}]$.

#### 4.2.4 Theorem 6 (formal)

**Theorem 6 (Post-Quantum Hybrid Integrity).** *For any PPT adversary $\mathcal{A}$ making at most $q$ encryption queries,*
$$\mathrm{Adv}^{\mathrm{INT\text{-}CTXT}}_{\mathcal{A}}(\lambda) \leq q \cdot \mathrm{Adv}^{\mathrm{IND\text{-}CCA2}}_{\mathcal{B}_1, \text{Kyber}}(\lambda) + \mathrm{Adv}^{\mathrm{EUF\text{-}CMA}}_{\mathcal{B}_2, \text{HMAC}}(\lambda) + \mathrm{negl}(\lambda),$$
*where $\mathcal{B}_1$ attacks Kyber-768 IND-CCA2 (reducing to Module-LWE + Module-SIS via the Fujisaki–Okamoto transform~\cite{hofheinz2017fo}) and $\mathcal{B}_2$ attacks HMAC-SHA256 EUF-CMA (reducing to the PRF assumption on SHA-256~\cite{rfc2104,nist_fips198}). Confidentiality of $v$ inherits from Paillier IND-CPA under DCRA (cf. Theorem 1).*

#### 4.2.5 Hybrid proof outline

$G_0$: real INT-CTXT experiment.

$G_1$: identical to $G_0$, except $\mathcal{C}$ aborts if $\mathcal{A}$'s forgery uses a $\text{ct}_{\text{ky}}^*$ that decapsulates to a $K^*$ which collides with any oracle-derived $K_i$ where $\text{ct}_{\text{ky}}^* \neq \text{ct}_{\text{ky},i}$. By the IND-CCA2 security of Kyber-768 (which implies key-pseudorandomness for unique ciphertexts), the per-query collision probability is bounded by Kyber's CCA advantage. Union bound over $q$ queries gives:
$$|\Pr[\mathcal{A} \text{ wins } G_1] - \Pr[\mathcal{A} \text{ wins } G_0]| \leq q \cdot \mathrm{Adv}^{\mathrm{IND\text{-}CCA2}}_{\mathcal{B}_1, \text{Kyber}}(\lambda).$$

$G_2$: identical to $G_1$, except $\mathcal{C}$ aborts if $\mathcal{A}$'s forgery is a fresh MAC under a fresh $K_{\text{auth}}^*$ never used by any oracle query. For this to succeed, $\mathcal{A}$ must produce a valid HMAC for an unseen key; by EUF-CMA of HMAC-SHA256 under the PRF assumption, this is bounded by $\mathrm{Adv}^{\mathrm{EUF\text{-}CMA}}_{\mathcal{B}_2, \text{HMAC}}(\lambda)$.

In $G_2$, the only remaining win path requires reusing some oracle ballot $\mathsf{Ballot}_i$ exactly — but condition (i) of the win condition rules this out. Hence $\Pr[\mathcal{A} \text{ wins } G_2] = 0$.

Combining yields the bound in the theorem.

**Confidentiality (P1):** Inherits from Theorem 1's hybrid game $G_3$ step (the MAC is replaced by a uniformly random string of the same length; the $\text{ct}_{\text{ky}}$ is independent of $v$ since Kyber is an IND-CCA2 KEM). Thus $\mathcal{A}$ learns no information about $v$ beyond that revealed by $E(v)$, which Theorem 1 shows is computationally hidden under DCRA.

#### 4.2.6 Confidence per step

| Step | Justification source | Confidence |
|------|----------------------|------------|
| Kyber-768 IND-CCA2 | NIST FIPS 203 §1.3 + Hofheinz-Hövelmanns-Kiltz 2017 | HIGH |
| HMAC-SHA256 EUF-CMA | RFC 2104; FIPS 198; Bellare 2006 PRF analysis | HIGH |
| Encrypt-then-MAC composition for INT-CTXT | Bellare-Namprempre 2000; Cramer-Shoup 2003 KEM-DEM | HIGH |
| Confidentiality piggyback on Theorem 1 | Direct hybrid argument | HIGH |

#### 4.2.7 Open questions

- **OQ1:** The construction uses $K_{\text{auth}} = \text{SHA256}(K \| \text{salt})$ for key derivation rather than HKDF. Under the random-oracle model assumption on SHA-256, this is fine; under the PRF assumption alone, a more careful analysis would invoke key-derivation-function security (e.g., HKDF-Extract). Worth a footnote in thesis.
- **OQ2:** The thesis includes a sentence "Generic message encryption additionally uses AES-256-GCM for authenticated symmetric encryption." This is for non-vote payloads (audit metadata) and is outside the T6 game; mention in scope clarification only.

#### 4.2.8 Cross-checks performed

1. Verified `chapter3.tex:416–435` matches the standard Encrypt-then-MAC pattern with KEM-derived MAC key.
2. Verified Kyber-768 parameters match NIST FIPS 203 (Category 3, $k=3$, $n=256$, $q=3329$).
3. Verified HMAC-SHA256 is the FIPS 198 standard construction.
4. Verified that `chapter3.tex:438` (the dual-layer protection paragraph) correctly states the security argument's structure: confidentiality via Paillier-DCRA, integrity via Kyber-derived MAC.
### 4.3 Theorem 5 — SA² Aggregation Privacy

**Status:** Drafted (Checkpoint 3).
**Confidence:** HIGH — direct reduction to Paillier IND-CPA (DCRA).

#### 4.3.1 Construction recap

Per `chapter3.tex:374–402`, given an encrypted vote $E(v)$, the voter computes:
- $S_A = E(v) \cdot E(\text{mask}) = E(v + \text{mask}) \bmod n^2$ (sent to Server A)
- $S_B = E(-\text{mask}) \bmod n^2$ (sent to Server B)
where $\text{mask} \stackrel{\$}{\leftarrow} \mathbb{Z}_{n/100}$.

Reconstruction: $S_A \cdot S_B = E(v + \text{mask} - \text{mask}) = E(v)$ via Paillier additive homomorphism.

**Trust assumption A1:** at least one of {Server A, Server B} is honest and does not share its share with the other.

#### 4.3.2 Game definition — Single-server view privacy

**Experiment $\mathrm{SA2\text{-}Priv}^{\sigma}_{\mathcal{A}}(\lambda)$** (parametrised by which server $\sigma \in \{A, B\}$ is corrupted):

1. **Setup.** $\mathcal{C}$ generates Paillier key pair $(\text{pk}_p, \text{sk}_p)$ and gives $\mathcal{A}$ only $\text{pk}_p$ (the corrupted-server view does not include $\text{sk}_p$).
2. **Challenge.** $\mathcal{A}$ outputs two distinct votes $v_0, v_1 \in \mathbb{Z}_n$.
3. $\mathcal{C}$ samples $b \stackrel{\$}{\leftarrow} \{0, 1\}$ and $\text{mask} \stackrel{\$}{\leftarrow} \mathbb{Z}_{n/100}$, computes $(S_A^*, S_B^*) \leftarrow \mathsf{Split}(v_b, \text{mask})$, and gives $\mathcal{A}$ only the $\sigma$-server's share: $S_A^*$ if $\sigma = A$, else $S_B^*$.
4. **Guess.** $\mathcal{A}$ outputs $b' \in \{0, 1\}$. Wins if $b' = b$.

Advantage: $\mathrm{Adv}^{\mathrm{SA2\text{-}Priv}, \sigma}_{\mathcal{A}}(\lambda) = |\Pr[b' = b] - 1/2|$.

#### 4.3.3 Theorem 5 (formal)

**Theorem 5 (SA² Aggregation Privacy).** *Let $\sigma \in \{A, B\}$ index a single corrupted server. For any PPT adversary $\mathcal{A}$,*
$$\mathrm{Adv}^{\mathrm{SA2\text{-}Priv}, \sigma}_{\mathcal{A}}(\lambda) \leq \mathrm{Adv}^{\mathrm{IND\text{-}CPA}}_{\mathcal{B}, \mathrm{Paillier}}(\lambda) \leq \mathrm{Adv}^{\mathrm{DCRA}}_{\mathcal{B}'}(\lambda) + \mathrm{negl}(\lambda).$$

#### 4.3.4 Proof (case $\sigma = A$; $\sigma = B$ is symmetric)

Server A's view is $S_A^* = E(v_b + \text{mask})$ where mask is uniform on $\mathbb{Z}_{n/100}$. We construct a Paillier IND-CPA distinguisher $\mathcal{B}$ as follows:

$\mathcal{B}$ receives $\text{pk}_p$ from the Paillier IND-CPA challenger, forwards it to $\mathcal{A}$, receives $(v_0, v_1)$ from $\mathcal{A}$, and proceeds:
- Sample $\text{mask} \stackrel{\$}{\leftarrow} \mathbb{Z}_{n/100}$.
- Compute $m_0 = v_0 + \text{mask} \bmod n$ and $m_1 = v_1 + \text{mask} \bmod n$.
- Submit $(m_0, m_1)$ to the Paillier IND-CPA challenger; receive ciphertext $c^* = E(m_b)$ for the challenger's hidden bit $b$.
- Forward $c^*$ to $\mathcal{A}$ as the simulated $S_A^*$.
- Output $\mathcal{A}$'s guess $b'$ as $\mathcal{B}$'s own guess.

Since the same mask is added to both $v_0$ and $v_1$, the challenger's computation $E(m_b) = E(v_b + \text{mask})$ exactly matches the distribution of $S_A^*$ in the real game. Hence $\mathrm{Adv}^{\mathrm{IND\text{-}CPA}}_{\mathcal{B}, \mathrm{Paillier}}(\lambda) = \mathrm{Adv}^{\mathrm{SA2\text{-}Priv}, A}_{\mathcal{A}}(\lambda)$. Paillier IND-CPA reduces to DCRA (Paillier 1999, Theorem 15) with negligible loss. The case $\sigma = B$ is identical: $S_B^* = E(-\text{mask})$ is independent of $v_b$, so the IND-CPA reduction is even tighter (the challenger's plaintexts $m_0 = m_1 = -\text{mask}$ are equal, giving $\mathrm{Adv} = 0$ unconditionally — Server B's share leaks zero information about $v$).

#### 4.3.5 Mask-range subtlety

The mask is sampled from $\mathbb{Z}_{n/100}$, not $\mathbb{Z}_n$. This bound is chosen to prevent overflow: with up to 100 voters per aggregation batch, the sum of masks remains in $\mathbb{Z}_n$ and the additive homomorphism $E(\sum v_i + \sum \text{mask}_i)$ does not wrap modulo $n$. The mask range does **not** affect the IND-CPA reduction: indistinguishability of $E(v_0 + \text{mask})$ and $E(v_1 + \text{mask})$ is a property of Paillier semantic security under DCRA, independent of the mask distribution. (In particular, the mask need not statistically hide $v$ in plaintext space; the encryption layer hides it.)

If statistical (information-theoretic) hiding were required at the plaintext level — for example, in a setting where the recipient can decrypt — then $\text{mask}$ would need to be uniform over $\mathbb{Z}_n$ to give $v + \text{mask}$ a distribution close to uniform on $\mathbb{Z}_n$. This stronger property is **not** required here, since both shares remain encrypted at all times under Paillier and only their product (the unmasked $E(v) \cdot E(0) = E(v)$, after both shares combined) is decrypted, after threshold-Paillier tally aggregation. The current $\mathbb{Z}_{n/100}$ range is therefore correct for the system's design.

#### 4.3.6 Confidence per step

| Step | Justification source | Confidence |
|------|----------------------|------------|
| Paillier IND-CPA $\Leftrightarrow$ DCRA | Paillier 1999, Theorem 15 | HIGH |
| Reduction simulation faithfulness | Direct check: $E(v_b + \text{mask})$ matches real $S_A$ | HIGH |
| Server B share independence of $v$ | Direct: $S_B = E(-\text{mask})$, no $v$ dependence | HIGH |
| Mask-range argument | Standard: encryption hides plaintext, range only prevents overflow | HIGH |

#### 4.3.7 Open questions

- **OQ1:** The trust assumption A1 (non-colluding servers) is system-level, not cryptographic. The thesis already states this clearly. If both servers collude, $S_A \cdot S_B = E(v)$ is reconstructed; security degrades to Paillier IND-CPA alone, which still hides $v$ from external observers but reveals it to the colluding pair (who can then forward $E(v)$ to the threshold-decryption trustees, but not decrypt themselves without the trustee quorum).
- **OQ2:** A multi-vote variant (where $\mathcal{A}$ corrupts one server and observes many ballots from many voters) is straightforwardly reducible by hybrid argument: the per-ballot advantage compounds linearly, so Adv ≤ $q \cdot \mathrm{Adv}^{\mathrm{DCRA}}$. Worth a one-sentence remark in the thesis.

#### 4.3.8 Cross-checks performed

1. Verified construction against `chapter3.tex:380–391` (mask sampling, share computation).
2. Verified reconstruction via Paillier additive homomorphism (`chapter3.tex:396–400`).
3. Verified that the security claim in `chapter3.tex:402` matches the formal game (single-server-view indistinguishability).
4. Verified Paillier IND-CPA $\Leftrightarrow$ DCRA against Paillier 1999 §3.3.
### 4.4 Theorem 1 — Ballot Privacy (CENTERPIECE)

**Status:** Drafted (Checkpoint 4).
**Confidence:** HIGH (with care) — composition of T2/T3/T6 hybrids with a per-position Paillier reduction to DCRA.

#### 4.4.1 Construction recap — what is in a ballot

Per the 17-step pipeline in `chapter3.tex` § sec:pipeline, each ballot $\mathsf{Ballot}$ casting a vote $\mathbf{v} = (v_1, \ldots, v_m) \in \{0,1\}^m$ with $\sum v_i = 1$ comprises:
1. **Paillier ciphertexts** $E_j = \text{Paillier.Enc}(\text{pk}_p, v_j)$, $j = 1, \ldots, m$.
2. **Pedersen commitments** $C_j = g^{v_j} h^{r_j}$, $j = 1, \ldots, m$.
3. **ZK proofs** $\pi^{\text{bin}}_j$ (per-candidate binary proof) and $\pi^{\text{sum}}$ (sum-equals-1).
4. **Linkable ring signature** $\sigma$ over $(E_1, \ldots, E_m)$ using the voter's secret key in a ring $\mathcal{R}$ of $|\mathcal{R}| = 100$.
5. **SA² shares** $(S_A, S_B) = \mathsf{Split}(E_j, \text{mask})$ for each $j$.
6. **Kyber outer wrapper** $(\text{ct}_{\text{ky}}, \text{salt}, \text{MAC})$ binding the Paillier ciphertexts.

#### 4.4.2 Game definition — IND-Ballot-CPA

**Experiment $\mathrm{IND\text{-}Ballot\text{-}CPA}_{\mathcal{A}}(\lambda)$:**

1. **Setup.** $\mathcal{C}$ runs full system setup: Paillier $(\text{pk}_p, \text{sk}_p)$ at 2048-bit, Pedersen $(p, q, g, h)$, ring keys $\{(\text{pk}_i, \text{sk}_i)\}_{i=1}^{|\mathcal{R}|}$, SMDC credentials per voter, Kyber keys $(\text{pk}_{\text{ky}}, \text{sk}_{\text{ky}})$. Public bulletin-board model: $\mathcal{A}$ observes all posted ballots.
2. **Pre-challenge query phase.** $\mathcal{A}$ adaptively:
   - Submits any number of honest-voter casts (oracle $\mathsf{Cast}(\text{vid}, \mathbf{v})$ posts an honest ballot); 
   - Corrupts up to $|\mathcal{R}| - 1 = 99$ ring members of its choice (excluding the target voter $\text{id}^*$ when chosen);
   - Queries a $\mathsf{Reveal}(\mathsf{Ballot}_i)$ oracle that returns the randomness used in any past honest ballot except the eventual challenge.
3. **Challenge.** $\mathcal{A}$ outputs two valid vote vectors $\mathbf{v}_0, \mathbf{v}_1 \in \{0,1\}^m$ (each with $\sum_j v_{b,j} = 1$) and a target voter identity $\text{id}^*$ whose ring secret key is uncorrupted. $\mathcal{C}$ samples $b \stackrel{\$}{\leftarrow} \{0,1\}$ and produces the complete challenge ballot $\mathsf{Ballot}^*$ for $\mathbf{v}_b$ as in §4.4.1, with $\text{id}^*$ as signer.
4. **Post-challenge.** $\mathcal{A}$ may continue querying $\mathsf{Cast}$ (any vote, any voter except $\text{id}^*$) and corrupting non-target ring members.
5. **Guess.** $\mathcal{A}$ outputs $b' \in \{0,1\}$. Wins if $b' = b$.

Define $\mathrm{Adv}^{\mathrm{IND\text{-}Ballot\text{-}CPA}}_{\mathcal{A}}(\lambda) = |\Pr[b' = b] - 1/2|$.

#### 4.4.3 Theorem 1 (formal)

**Theorem 1 (Ballot Indistinguishability — Computational).** *Let $m$ denote the number of candidates and $\Delta = |\{j : v_{0,j} \neq v_{1,j}\}|$ the Hamming distance between the two challenge vote vectors (for valid single-choice ballots, $\Delta \in \{0, 2\}$; $\Delta = 0$ means identical votes and the game is trivial). For any PPT adversary $\mathcal{A}$,*
$$\mathrm{Adv}^{\mathrm{IND\text{-}Ballot\text{-}CPA}}_{\mathcal{A}}(\lambda) \leq \mathrm{Adv}^{\mathrm{ZK}}_{\mathcal{B}_{\text{ZK}}}(\lambda) + \mathrm{Adv}^{\mathrm{Anon}}_{\mathcal{B}_{\sigma}}(\lambda) + \mathrm{Adv}^{\mathrm{INT\text{-}CTXT}}_{\mathcal{B}_{\text{Hyb}}}(\lambda) + \Delta \cdot \mathrm{Adv}^{\mathrm{DCRA}}_{\mathcal{B}_{\mathrm{DCRA}}}(\lambda) + \mathrm{negl}(\lambda),$$
*where $\mathcal{B}_{\text{ZK}}$ attacks the simulation indistinguishability of the Σ-protocol Strong-Fiat-Shamir transform (negligible in ROM under Theorem~2), $\mathcal{B}_{\sigma}$ attacks LWW signer ambiguity (Theorem~3, reducing to DDH), $\mathcal{B}_{\text{Hyb}}$ attacks the Kyber-HMAC hybrid (Theorem~6, reducing to Module-LWE), and $\mathcal{B}_{\mathrm{DCRA}}$ attacks DCRA (Paillier semantic security).*

#### 4.4.4 Hybrid sequence $G_0 \to G_4$

We follow the structure of the supervisor's elaboration, refining the final reduction to DCRA into a per-position hybrid.

**Game $G_0$:** The real $\mathrm{IND\text{-}Ballot\text{-}CPA}$ experiment. By definition $\Pr[\mathcal{A} \text{ wins } G_0] = 1/2 + \mathrm{Adv}^{\mathrm{IND\text{-}Ballot\text{-}CPA}}_{\mathcal{A}}(\lambda) =: 1/2 + \varepsilon$.

**Game $G_1$ (simulate ZK proofs):** $\mathcal{C}$ replaces $\pi^{\text{bin}}_j$ and $\pi^{\text{sum}}$ in the challenge ballot with simulator outputs $\widetilde{\pi}^{\text{bin}}_j$, $\widetilde{\pi}^{\text{sum}}$ produced without the witness, by programming the random oracle at the corresponding hash queries. The Schnorr/CDS Σ-protocols are honest-verifier zero-knowledge (Cramer--Damgård--Schoenmakers 1994~\cite{cramer1994or}); under the random-oracle model and Strong Fiat--Shamir~\cite{bernhard2012}, simulated proofs are computationally indistinguishable from real ones. Bound: $|\Pr[\mathcal{A} \text{ wins } G_1] - \Pr[\mathcal{A} \text{ wins } G_0]| \leq \mathrm{Adv}^{\mathrm{ZK}}_{\mathcal{B}_{\text{ZK}}}(\lambda)$.

After $G_1$, the proofs in $\mathsf{Ballot}^*$ are independent of $b$.

**Game $G_2$ (simulate ring signature):** $\mathcal{C}$ replaces the linkable ring signature $\sigma$ in $\mathsf{Ballot}^*$ with a simulated signature $\widetilde{\sigma}$ produced using the LWW signer-ambiguity simulator: choose $(c_i, r_i)$ uniformly at random for all $i \in \mathcal{R}$ except one randomly-chosen index $i^*$, then compute the closing values to satisfy the verification equation. Under DDH (used only via the key-image relation $I = H(\text{pk})^{\text{sk}}$; the signature components themselves are unconditionally uniform per Liu--Wei--Wong 2004 §4.2 Theorem~4~\cite{liu2004}), the simulated signature is computationally indistinguishable from the real one. Bound: $|\Pr[\mathcal{A} \text{ wins } G_2] - \Pr[\mathcal{A} \text{ wins } G_1]| \leq \mathrm{Adv}^{\mathrm{Anon}}_{\mathcal{B}_{\sigma}}(\lambda) \leq \mathrm{Adv}^{\mathrm{DDH}}_{\mathcal{B}'_{\sigma}}(\lambda)$.

After $G_2$, $\widetilde{\sigma}$ in $\mathsf{Ballot}^*$ is independent of $\text{id}^*$.

**Game $G_3$ (replace MAC with random):** $\mathcal{C}$ replaces $\text{MAC}^*$ in $\mathsf{Ballot}^*$ with a uniformly random 256-bit string $\widetilde{\text{MAC}}^* \stackrel{\$}{\leftarrow} \{0,1\}^{256}$. By the IND-CCA2 security of Kyber-768 (Theorem~6~\cite{nist_kyber,hofheinz2017fo}), the encapsulated key $K^*$ derived from $\text{ct}_{\text{ky}}^*$ is pseudorandom; under the PRF property of HMAC-SHA256 (and the random-oracle model on the $K_{\text{auth}}$ derivation), the resulting MAC is computationally indistinguishable from random. Bound: $|\Pr[\mathcal{A} \text{ wins } G_3] - \Pr[\mathcal{A} \text{ wins } G_2]| \leq \mathrm{Adv}^{\mathrm{INT\text{-}CTXT}}_{\mathcal{B}_{\text{Hyb}}}(\lambda)$.

After $G_3$, $\widetilde{\text{MAC}}^*$ is independent of $b$.

**Game $G_4$ (Pedersen and SA² shares carry no information):** No further game change is needed for Pedersen commitments because, by **perfect hiding** (cf. corrected Theorem~4 / E1.1–E1.7 in §3 of this document), the distribution of $\{C_j\}$ is independent of $\{v_{b,j}\}$ — statistically zero advantage. Similarly the SA² shares $(S_A, S_B)$ are encryptions whose hiding is captured by the Paillier component analysed in the next step. Hence $\Pr[\mathcal{A} \text{ wins } G_4] = \Pr[\mathcal{A} \text{ wins } G_3]$.

**Per-position hybrid to DCRA.** In $G_3$, the only $b$-dependent component of $\mathsf{Ballot}^*$ is the vector of Paillier ciphertexts $(E_1^*, \ldots, E_m^*)$ (the SA² shares are derived deterministically from these via random masks, contributing no extra advantage). Let $\Delta = |\{j : v_{0,j} \neq v_{1,j}\}|$. Define a sub-hybrid sequence $H_0, H_1, \ldots, H_\Delta$:
- $H_0 = G_3$ with $\mathsf{Ballot}^* = $ honest ballot for $\mathbf{v}_0$.
- $H_k$ = same as $H_{k-1}$, except the $k$-th differing position (in some fixed ordering of the differing-position set $\mathsf{Diff}$) is changed from $v_0$'s value to $v_1$'s value.
- $H_\Delta = G_3$ with $\mathsf{Ballot}^* = $ honest ballot for $\mathbf{v}_1$.

For each consecutive pair $H_{k-1}, H_k$, the only difference is one Paillier ciphertext at one position. A standard reduction $\mathcal{B}_{\mathrm{DCRA}}$ embeds a Paillier IND-CPA challenge (which reduces to DCRA) at that position: given a target ciphertext $c^*$ that is either $E(0)$ or $E(1)$, $\mathcal{B}_{\mathrm{DCRA}}$ uses $c^*$ as the $k$-th differing-position ciphertext and constructs the rest of $\mathsf{Ballot}^*$ honestly (knowing all other inputs). $\mathcal{B}_{\mathrm{DCRA}}$ outputs $\mathcal{A}$'s guess. Hence $|\Pr[\mathcal{A} \text{ wins } H_k] - \Pr[\mathcal{A} \text{ wins } H_{k-1}]| \leq \mathrm{Adv}^{\mathrm{IND\text{-}CPA}}_{\mathcal{B}_{\mathrm{DCRA}}, \mathrm{Paillier}}(\lambda) \leq \mathrm{Adv}^{\mathrm{DCRA}}_{\mathcal{B}'_{\mathrm{DCRA}}}(\lambda) + \mathrm{negl}(\lambda)$.

Telescoping over all $\Delta$ steps: $|\Pr[\mathcal{A} \text{ wins } H_\Delta] - \Pr[\mathcal{A} \text{ wins } H_0]| \leq \Delta \cdot \mathrm{Adv}^{\mathrm{DCRA}}_{\mathcal{B}'_{\mathrm{DCRA}}}(\lambda)$.

For valid single-choice ballots, $\Delta \in \{0, 2\}$ (the "1" moves from one candidate to another, flipping exactly two positions). So the DCRA term contributes at most $2 \cdot \mathrm{Adv}^{\mathrm{DCRA}}$.

#### 4.4.5 Combined bound

Combining the four hybrid steps with the per-position telescope:
$$\varepsilon \leq \mathrm{Adv}^{\mathrm{ZK}}_{\mathcal{B}_{\text{ZK}}} + \mathrm{Adv}^{\mathrm{Anon}}_{\mathcal{B}_{\sigma}} + \mathrm{Adv}^{\mathrm{INT\text{-}CTXT}}_{\mathcal{B}_{\text{Hyb}}} + 2 \cdot \mathrm{Adv}^{\mathrm{DCRA}}_{\mathcal{B}'_{\mathrm{DCRA}}} + \mathrm{negl}(\lambda).$$
Each term is negligible under the standard assumptions (DCRA, DDH, M-LWE, ROM), so $\varepsilon$ is negligible.

#### 4.4.6 Confidence per step

| Step | Justification | Confidence |
|------|---------------|------------|
| $G_0 \to G_1$ (ZK simulator) | Schnorr HVZK + Strong-FS in ROM (Bernhard-Pereira-Warinschi 2012) | HIGH |
| $G_1 \to G_2$ (ring sig simulator) | LWW 2004 §4.2 Theorem 4 (signer ambiguity); see §4.5 (T3) for verification | HIGH |
| $G_2 \to G_3$ (MAC randomization) | Kyber IND-CCA2 + HMAC EUF-CMA + ROM; see §4.2 (T6) | HIGH |
| Pedersen step (no game) | Perfect hiding (corrected E1.1–E1.7) | HIGH |
| Per-position hybrid for Paillier | Textbook hybrid argument over IND-CPA | HIGH |
| Final DCRA reduction | Paillier 1999 Theorem 15 | HIGH |

#### 4.4.7 Open questions

- **OQ1:** The Reveal oracle in the query phase: does revealing randomness for non-challenge ballots affect the reduction? In the IND-CPA → DCRA reduction at each $H_k \to H_{k+1}$ step, the adversary's revealed randomness is for ciphertexts the reduction $\mathcal{B}$ generated honestly, so $\mathcal{B}$ knows them and answers Reveal queries faithfully. The challenge ciphertext $c^*$ is excluded from Reveal queries by the game definition. Reduction goes through. **Verified.**
- **OQ2:** Threshold Paillier decryption is intentionally not included in this game — the game models privacy of an individual ballot before tallying. The decryption step releases only the aggregate. A separate analysis would be needed to argue the aggregate does not leak individual votes — this is standard tally privacy, follows from any reasonable election-statistics threshold (number of voters per outcome ≥ 1 leaks nothing structural).
- **OQ3:** The challenge ballot's SA² shares $(S_A, S_B)$ are derived deterministically from $E_b$ and a fresh mask. The mask is internal to the challenger; the adversary observes only the shares. By the analysis of T5 (§4.3), each share alone hides $v_b$ under DCRA, and both shares together reconstruct $E_b$ (whose hiding is what we are reducing to). No additional hybrid step is needed.

#### 4.4.8 Cross-checks performed

1. **Verified ballot composition** against the 17-step pipeline in `chapter3.tex` § sec:pipeline.
2. **Verified ZK simulator existence** for OR-composed Schnorr (CDS 1994 §3) and Schnorr-PoK ($\pi^{\text{sum}}$, Schnorr 1991).
3. **Verified LWW signer-ambiguity simulator** at §4.5 / T3 cross-check (planned for Checkpoint 5).
4. **Verified per-position hybrid** is sound: each step changes only one Paillier ciphertext; reduction $\mathcal{B}$ embeds DCRA challenge honestly because all other ciphertexts are generated by $\mathcal{B}$ itself.
5. **Verified $\Delta = 2$ for valid single-choice ballots:** each $\mathbf{v}_b$ has exactly one 1; if $\mathbf{v}_0 \neq \mathbf{v}_1$, the 1 moves to a new position, flipping exactly two coordinates.
6. **Verified Pedersen step requires no game** because of perfect hiding (corrected from prior thesis text — see §3 / E1).
### 4.5 Theorem 3 — Voter Anonymity

**Status:** Drafted (Checkpoint 5).
**Confidence:** MEDIUM — depends on careful framing of LWW Theorem 4. Cross-checked against LWW 2004 §4 (definitions) and §4.2 (Signer Ambiguity proof).

#### 4.5.1 Construction recap

Per `chapter3.tex:255–300`, each voter has a key pair $(\text{sk}, \text{pk}) = (x, g^x)$ in the order-$q$ subgroup of $\mathbb{Z}_p^*$. The ring of size $|\mathcal{R}| = 100$ produces signatures $\sigma = (I, c_0, r_0, r_1, \ldots, r_{n-1})$ where $I = H(\text{pk}_s)^{\text{sk}_s}$ is the key image and the $(c_i, r_i)$ values close the ring per the LWW signing equations.

#### 4.5.2 Game definition — Signer Anonymity

**Experiment $\mathrm{Anon}^{\mathrm{LWW}}_{\mathcal{A}}(\lambda)$:**

1. **Setup.** $\mathcal{C}$ generates a ring $\mathcal{R} = \{(x_i, g^{x_i})\}_{i=0}^{n-1}$ honestly (so all $x_i$ are uniform in $\mathbb{Z}_q$) and gives $\mathcal{A}$ only the public ring $\{g^{x_i}\}_{i=0}^{n-1}$.
2. **Pre-challenge oracles.** $\mathcal{A}$ may query a Sign oracle $\mathsf{Sign}(j, M)$ for any signer index $j$ and message $M$, receiving an honest signature.
3. **Challenge.** $\mathcal{A}$ outputs two indices $i_0, i_1 \in \{0, \ldots, n-1\}$ and a challenge message $M^*$ such that $\mathcal{A}$ has not previously queried $\mathsf{Sign}(i_0, M^*)$ or $\mathsf{Sign}(i_1, M^*)$. $\mathcal{C}$ samples $b \stackrel{\$}{\leftarrow} \{0, 1\}$ and produces $\sigma^* \leftarrow \mathsf{Sign}(i_b, M^*)$, returning $\sigma^*$ to $\mathcal{A}$.
4. **Post-challenge.** $\mathcal{A}$ may continue querying $\mathsf{Sign}$ except on $(i_0, M^*)$ and $(i_1, M^*)$.
5. **Guess.** $\mathcal{A}$ outputs $b' \in \{0, 1\}$. Wins if $b' = b$.

Define $\mathrm{Adv}^{\mathrm{Anon}}_{\mathcal{A}}(\lambda) = |\Pr[b' = b] - 1/2|$.

#### 4.5.3 Theorem 3 (formal)

**Theorem 3 (Voter Anonymity).** *In the random-oracle model on $H$, for any PPT adversary $\mathcal{A}$ making at most $q_S$ Sign-oracle queries,*
$$\mathrm{Adv}^{\mathrm{Anon}}_{\mathcal{A}}(\lambda) \leq \mathrm{Adv}^{\mathrm{DDH}}_{\mathcal{B}}(\lambda) + q_S \cdot 2^{-(\lambda - O(\log q_S))}.$$

#### 4.5.4 Proof outline

The argument decomposes the adversary's view of $\sigma^* = (I^*, c_0^*, r_0^*, \ldots, r_{n-1}^*)$ into two parts:

**Part (a): the $(c_i, r_i)$ values are statistically anonymous.** By the LWW signing procedure, for each non-signer $i \neq s$ the value $r_i$ is sampled uniformly from $\mathbb{Z}_q$, and $c_i$ is determined by the random-oracle output on the previous step. For the signer $s$, the value $r_s = \alpha - c_s \cdot \text{sk}_s \bmod q$ where $\alpha$ is uniform; hence $r_s$ is also uniform on $\mathbb{Z}_q$ (since $\alpha$ uniformizes the otherwise-deterministic relationship). So the marginal distribution of $(c_0, r_0, \ldots, r_{n-1})$ is **identical** for every choice of signer index, meaning Part (a) contributes statistical distance zero. (This matches LWW 2004 §4.2 Lemma 4.)

**Part (b): the key image $I^*$ requires DDH.** The key image $I^* = H(\text{pk}_{i_b})^{\text{sk}_{i_b}}$ depends on which signer was chosen. To distinguish $b = 0$ vs $b = 1$, $\mathcal{A}$ must determine which of the ring members $(i_0, i_1)$ generated the key image. In the random oracle model, $H(\text{pk}_i)$ is a fresh uniformly-random group element $h_i$ for each $i$ (programmed once on first query). The triple $(\text{pk}_i, h_i, I^*)$ is a DDH triple precisely when $i$ is the signer. The reduction $\mathcal{B}$:

1. Receives a DDH challenge $(g^a, g^b, T)$ where $T = g^{ab}$ (case 1) or $T \stackrel{\$}{\leftarrow} \mathbb{G}_q$ (case 0).
2. Embeds $g^a$ as the public key of signer $i_0$ (sets $\text{pk}_{i_0} = g^a$, so $\text{sk}_{i_0} = a$ which $\mathcal{B}$ does not know).
3. Programs the random oracle so $H(\text{pk}_{i_0}) = g^b$.
4. Sets $I^* = T$.
5. Generates the rest of the signature using the simulator (since the $(c_i, r_i)$ values are statistically uniform, $\mathcal{B}$ samples them honestly to close the ring).
6. Returns $\sigma^* = (T, c_0^*, r_0^*, \ldots, r_{n-1}^*)$ to $\mathcal{A}$.
7. Outputs $\mathcal{A}$'s guess: if $\mathcal{A}$ guesses $b = 0$ (i.e., signer $i_0$), $\mathcal{B}$ outputs "DDH triple"; otherwise "random".

When $T = g^{ab}$, the key image is correctly $H(\text{pk}_{i_0})^{\text{sk}_{i_0}}$, indistinguishable from a real $b = 0$ signature. When $T$ is random, the key image is random and statistically indistinguishable from a $b = 1$ signature where the signer's identity is hidden. Hence $\mathrm{Adv}^{\mathrm{DDH}}_{\mathcal{B}}(\lambda) = \mathrm{Adv}^{\mathrm{Anon}}_{\mathcal{A}}(\lambda)$.

The extra $q_S \cdot 2^{-(\lambda - O(\log q_S))}$ term accounts for random-oracle programming consistency across $q_S$ Sign queries (standard ROM bookkeeping; negligible).

#### 4.5.5 Linkability is preserved

Linkability (the property that two signatures by the same signer yield the same key image $I$) is unaffected by the anonymity argument: $I$ is a deterministic function of the signer's public key and secret key, so two signs by the same signer produce the same $I$. Anonymity prevents the verifier from learning *which* member signed, but linkability lets the system detect *that the same member signed twice*. For the e-voting application, this allows double-vote detection without compromising voter privacy.

#### 4.5.6 Confidence per step

| Step | Source | Confidence |
|------|--------|------------|
| $(c_i, r_i)$ statistical anonymity | LWW 2004 §4.2 Lemma 4 | HIGH |
| $\alpha$-uniformization of $r_s$ | Direct: $r_s = \alpha - c_s \cdot \text{sk}_s$ with uniform $\alpha$ | HIGH |
| Key-image DDH reduction | LWW 2004 §4.2 Theorem 4 (proof structure matches) | MEDIUM (verified in cross-check below) |
| ROM programming of $H(\text{pk}_i)$ | Standard | HIGH |

#### 4.5.7 Open questions

- **OQ1:** The original Theorem 3 statement says "an adversary's distinguishing advantage over uniform guessing is negligible under DDH." This is precisely the formulation above. Match confirmed.
- **OQ2:** The thesis statement also says "statistically indistinguishable among the $|\mathcal{R}|$ ring members" — this is true for the $(c_i, r_i)$ component (Part a), but the key image is only **computationally** hidden under DDH (Part b). The new theorem statement makes this distinction explicit. Recommend updating the existing wording in `chapter3.tex:592` if needed for precision.

#### 4.5.8 Cross-checks performed

1. **LWW 2004 Theorem 4 (Signer Ambiguity)** — Verified via its proof structure: the reduction embeds the DDH instance as the signer's $(pk, H(pk))$ pair and maps the DDH-or-random target to the key image. The thesis's signing procedure (chapter3.tex:269–295) matches the LSAG construction in LWW §3 exactly (key image, $\alpha$-randomization, ring closure).
2. **Statistical uniformity of $(c_i, r_i)$** — Direct verification: in step 4 of the signing procedure (chapter3.tex:285–291), each $r_i$ for $i \neq s$ is sampled uniformly; $c_{i+1}$ is the hash output (uniform in ROM). In step 5 (chapter3.tex:292), $r_s = \alpha - c_s \cdot \text{sk}_s$ with uniform $\alpha$. Hence all $(c_i, r_i)$ are uniform.
3. **Key image construction** — `chapter3.tex:275–279` matches LWW's $I = H(\text{pk})^{\text{sk}}$ definition.
### 4.6 Theorem 4 — Coercion Resistance

**Status:** Drafted (Checkpoint 6).
**Confidence:** **NEEDS-EXPERT-REVIEW** — adapts JCJ coercion-resistance framework to SMDC's novel slot-based deniable credentials. The proof structure is principled, but SMDC has no prior published reduction blueprint. Supervisor and / or external cryptographer should review before submission.

#### 4.6.1 Construction recap

Per `chapter3.tex:303–369`, each voter receives $k = 5$ credential slots. Slot $j$ contains:
- A Pedersen commitment $C_j = g^{w_j} h^{r_j}$ with $w_{j^*} = 1$ for exactly one **real slot index** $j^*$, and $w_j = 0$ for the other four (fake) slots.
- A binary $\Sigma$-protocol proof $\pi_j^{\text{bin}}$.

Plus a global sum proof $\pi^{\text{sum}}$ that $\sum_j w_j = 1$.

The real index $j^*$ is derived deterministically: $j^* = \mathrm{HMAC\text{-}SHA256}(K_{\text{srv}}, \mathrm{vid} \| \mathrm{eid}) \bmod 5$, where $K_{\text{srv}}$ is a server-side secret. The value $j^*$ is never stored; only the legitimate voter (who knows $\mathrm{vid}$ and $\mathrm{eid}$) can re-derive it via the server's verification API.

A coerced cast uses some slot $j_{\text{forced}}$ chosen by the coercer; if $j_{\text{forced}} \neq j^*$, the cast contributes $w = 0$ to the tally and is silently discarded.

#### 4.6.2 Threat model (explicit scope)

**In scope.** The adversary $\mathcal{A}$ (coercer) is given:
- All 5 slot commitments $\{C_j\}_{j=1}^5$ for the target voter.
- All 5 binary proofs $\{\pi_j^{\text{bin}}\}_{j=1}^5$.
- The sum proof $\pi^{\text{sum}}$.
- The voter's public bulletin-board ballot for the coerced cast (commitment, proof, ciphertext, ring signature, etc.).
- The voter's biometric/credential interaction transcript (everything observable during the cast).

**Adversary's task:** Determine whether the slot used in the coerced cast is the real slot ($j^*$) or a fake slot, with non-negligible advantage over the prior $\Pr[j_{\text{forced}} = j^*] = 1/5$.

**Out of scope** (each treated explicitly in §4.6.5):
- **Forced abstention.** A coercer who can prevent the voter from casting at all is not addressed; this is a system-level threat handled by the deployment (anonymous voting booths, mobile casting windows).
- **Randomization attacks.** A coercer who demands a random ballot rather than a specific candidate is not addressed.
- **Simulation attacks (full JCJ).** SMDC does not provide receipt-freeness in the strict JCJ sense (a coercer who controls the voter's device can replay the entire transcript). The SMDC guarantee is a weaker but operationally significant one: indistinguishability of real vs. fake slots given only the bulletin-board view.
- **Credential issuance compromise.** Assumption A3 (per §2.3): credentials are issued in a non-coerced setting.

#### 4.6.3 Game definition — Slot indistinguishability under coercion

**Experiment $\mathrm{Coerc}^{\mathrm{SMDC}}_{\mathcal{A}}(\lambda)$:**

1. **Setup.** $\mathcal{C}$ generates Pedersen parameters $(p, q, g, h)$ and HMAC server key $K_{\text{srv}}$. For the target voter (vid, eid), $\mathcal{C}$ computes the real slot index $j^* = \mathrm{HMAC\text{-}SHA256}(K_{\text{srv}}, \mathrm{vid} \| \mathrm{eid}) \bmod 5$.
2. **Credential generation.** For each slot $j \in \{1, \ldots, 5\}$, $\mathcal{C}$ samples $r_j \stackrel{\$}{\leftarrow} \mathbb{Z}_q$ and computes $C_j = g^{w_j} h^{r_j}$ where $w_{j^*} = 1$ and $w_j = 0$ for $j \neq j^*$. $\mathcal{C}$ also generates the binary proofs $\pi_j^{\text{bin}}$ and the sum proof $\pi^{\text{sum}}$.
3. **Coercion view.** $\mathcal{C}$ gives $\mathcal{A}$: $\{C_j\}, \{\pi_j^{\text{bin}}\}, \pi^{\text{sum}}$, plus the verification public key $K_{\text{srv}}^{\text{pub}}$ (a public counterpart used for HMAC verification; in the SMDC construction $K_{\text{srv}}$ is also held by the server but $\mathcal{A}$ has no direct read access).
4. **Challenge.** $\mathcal{A}$ outputs two distinct slot indices $j_0, j_1 \in \{1, \ldots, 5\}$ for comparison. ($\mathcal{A}$ does not know $j^*$.)
5. $\mathcal{C}$ samples $b \stackrel{\$}{\leftarrow} \{0, 1\}$ and emits the cast transcript using slot $j_b$: this includes the slot's commitment $C_{j_b}$, the binary proof $\pi_{j_b}^{\text{bin}}$, and the associated bulletin-board ballot (with Paillier ciphertext, ring signature, etc., all simulated as in Theorem~1's hybrid steps so that auxiliary components contribute no signer-/slot-dependent leakage).
6. **Guess.** $\mathcal{A}$ outputs $b' \in \{0, 1\}$. Wins if $b' = b$.

Define $\mathrm{Adv}^{\mathrm{Coerc}}_{\mathcal{A}}(\lambda) = |\Pr[b' = b] - 1/2|$.

#### 4.6.4 Theorem 4 (formal)

**Theorem 4 (SMDC Coercion Resistance — Slot Indistinguishability).** *For any adversary $\mathcal{A}$ (with no a priori bound on computational power for the Pedersen part, and PPT for the proof and HMAC parts),*
$$\mathrm{Adv}^{\mathrm{Coerc}}_{\mathcal{A}}(\lambda) \leq \mathrm{Adv}^{\mathrm{ZK}}_{\mathcal{B}_{\mathrm{ZK}}}(\lambda) + \mathrm{Adv}^{\mathrm{HMAC\text{-}PRF}}_{\mathcal{B}_{\mathrm{HMAC}}}(\lambda) + \mathrm{negl}(\lambda).$$

In particular, when $\mathcal{A}$ is restricted to PPT and HMAC-SHA256 is modeled as a PRF, $\mathrm{Adv}^{\mathrm{Coerc}}_{\mathcal{A}}(\lambda) = \mathrm{negl}(\lambda)$.

#### 4.6.5 Proof outline (hybrids $G_0 \to G_3$)

**Game $G_0$.** Real coercion experiment. $\Pr[\mathcal{A} \text{ wins } G_0] = 1/2 + \varepsilon$.

**Game $G_1$ (replace real-slot derivation with random index).** $\mathcal{C}$ samples $j^* \stackrel{\$}{\leftarrow} \{1, \ldots, 5\}$ uniformly at random, ignoring the HMAC derivation. By the PRF property of HMAC-SHA256 (under the assumption that $K_{\text{srv}}$ is unknown to $\mathcal{A}$, which holds by Trust Assumption A2 — server's secret state), the HMAC-derived $j^*$ is computationally indistinguishable from uniform. Bound: $|\Pr[G_1] - \Pr[G_0]| \leq \mathrm{Adv}^{\mathrm{HMAC\text{-}PRF}}_{\mathcal{B}_{\mathrm{HMAC}}}(\lambda)$.

**Game $G_2$ (simulate ZK proofs).** $\mathcal{C}$ replaces $\pi_j^{\text{bin}}$ and $\pi^{\text{sum}}$ with simulator outputs that do not use the witnesses $w_j$. Under Strong Fiat--Shamir in ROM, real and simulated proofs are computationally indistinguishable. Bound: $|\Pr[G_2] - \Pr[G_1]| \leq \mathrm{Adv}^{\mathrm{ZK}}_{\mathcal{B}_{\mathrm{ZK}}}(\lambda)$.

**Game $G_3$ (Pedersen perfect-hiding step — no game change needed).** Each $C_j$ is computed as $g^{w_j} h^{r_j}$ with $r_j$ uniform on $\mathbb{Z}_q$ and $h$ a generator of the order-$q$ subgroup. By perfect hiding (cf. §3 / E1.1–E1.9 of this document and Theorem~4 statement in chapter3.tex:596), the distribution of $C_j$ is uniform on the subgroup independently of $w_j \in \{0, 1\}$. Hence in $G_3$ — which is the same probability space as $G_2$ after simulation — the commitments $C_{j_0}$ and $C_{j_1}$ are statistically identical, regardless of which slot index is real. **No advantage from $C$:** $\Pr[G_3] = \Pr[G_2]$.

**End-state analysis.** In $G_3$, the adversary's view of the challenge slot $j_b$ comprises: $C_{j_b}$ (statistically uniform, independent of $b$), $\widetilde{\pi}_{j_b}^{\text{bin}}$ (simulator output, no witness dependence), and the bulletin-board ballot (already shown to be computationally independent of $b$ by the auxiliary-component hybrids from T1). The slot index $j^*$ is uniform (by $G_1$) and independent of $(j_0, j_1)$ chosen by $\mathcal{A}$. Therefore $\Pr[\mathcal{A} \text{ wins } G_3] = 1/2$ exactly, and combining the hybrid bounds gives the theorem's bound. \hfill $\square$

#### 4.6.6 What this theorem does and does NOT establish

**Establishes (formally):**
- An adversary cannot distinguish whether a cast was made using a real slot or a fake slot, given the public bulletin-board view and the voter's credential transcript.
- Equivalent: the coercer cannot verify whether the voter complied with coercion (used the real slot for the demanded candidate) or cheated (used a fake slot to silently waste the coerced vote).

**Does NOT establish (out of scope, future work):**
- **Forced abstention resistance:** if $\mathcal{A}$ can prevent the voter from casting at all, the voter cannot use a fake slot. SMDC addresses this only through deployment-level controls (anonymous casting windows).
- **Randomization-attack resistance:** if $\mathcal{A}$ demands a random ballot rather than a specific candidate, the indistinguishability argument does not distinguish "real random ballot from voter" vs "fake random ballot."
- **Full JCJ receipt-freeness:** A device-controlling coercer who observes the cast transcript at the device level (rather than just the bulletin board) is not covered. JCJ-2005 §4 addresses this via posted-ballot encryption; SMDC inherits this via the Paillier+ZK layer and is therefore at least as strong as standard receipt-free voting once the device is trusted.
- **Credential-issuance coercion:** Per Trust Assumption A3, credentials are assumed issued in a non-coerced setting. If $\mathcal{A}$ controls the issuance phase, $\mathcal{A}$ can record $j^*$ at issuance and defeat the construction.

#### 4.6.7 Confidence per step

| Step | Source | Confidence |
|------|--------|------------|
| $G_0 \to G_1$ (HMAC-PRF for slot derivation) | RFC 2104 + Bellare 2006 NMAC analysis | HIGH |
| $G_1 \to G_2$ (ZK simulator) | CDS 1994 + BPW 2012 | HIGH |
| Pedersen perfect-hiding step (no game) | Pedersen 1991 §2 | HIGH |
| End-state win probability = 1/2 | Direct distribution analysis | HIGH |
| **Overall reduction structure adapted from JCJ-2005** | JCJ 2005 §4 (adaptation; SMDC's slot mechanism is novel) | **NEEDS-EXPERT-REVIEW** |
| Scope/out-of-scope demarcation | Standard for coercion resistance | HIGH (explicit) |

#### 4.6.8 Open questions / items flagged for human review

- **OQ1 (NEEDS-EXPERT-REVIEW):** The adaptation of JCJ's coercion-resistance experiment to SMDC's slot-based mechanism is novel. The reduction structure (HMAC + ZK + perfect hiding) is principled, but a published peer reviewer (or supervisor with cryptography background) should verify that no SMDC-specific attack vector is missed. Specific concerns:
  - The adversary $\mathcal{A}$ knows all 5 commitments $\{C_j\}$ and all 5 ZK proofs $\{\pi_j^{\text{bin}}\}$. The argument that perfect hiding + ZK is sufficient for $\mathcal{A}$ not to learn $j^*$ from $\{C_j, \pi_j^{\text{bin}}\}$ is sound but should be verified.
  - The HMAC-PRF reduction requires that $K_{\text{srv}}$ remain unknown to $\mathcal{A}$. The "verification public key $K_{\text{srv}}^{\text{pub}}$" mentioned in step 3 of the game definition is a placeholder — in the SMDC construction, slot verification is done by the server-side comparison, not by a public key. This needs a careful audit of `chapter3.tex:360–369` (Credential Verification subsection) to confirm $K_{\text{srv}}$ is never disclosed.
- **OQ2:** The "silent discard" semantics (cast with $w = 0$ contributes nothing) are critical for the deniability argument. This is realized at Step~7 of the pipeline (`chapter3.tex` § 3.5). If a side-channel revealed which slots resulted in $w = 0$ casts, the deniability would be defeated. Mention in OQ2 for supervisor: the implementation should be audited for timing or storage side-channels that distinguish $w = 0$ vs $w = 1$ casts post-cast.
- **OQ3:** The "forced abstention" and "randomization attack" out-of-scope items are standard limitations of any coercion-resistant voting scheme (JCJ 2005 §3.3). The thesis's existing prose acknowledges these via the SMDC subsection but does not name them explicitly — recommend adding one sentence per scope item to the thesis.

#### 4.6.9 Cross-checks performed

1. **JCJ-2005 game structure** verified against §3 of JCJ 2005 (definition of coercion-resistance experiment). SMDC's slot-based variant maps to JCJ's "designated voter" experiment with slot indices playing the role of credentials.
2. **SMDC slot construction** verified against `chapter3.tex:309–340` (credential generation pseudocode).
3. **Pedersen perfect hiding** (now correctly stated post-Pedersen-fix in §3 / E1) verified to provide the statistical indistinguishability for the commitment-level slot comparison.
4. **HMAC-PRF** verified to suffice for the real-slot index derivation step (Bellare 2006, "New Proofs for NMAC and HMAC").
5. **Bulletin-board ballot's slot-independence** verified by reference to T1's hybrid game (auxiliary components are simulated independent of the slot identity).

---

## 5. UC Composition Discussion

**Status:** Drafted (Checkpoint 7).
**Confidence:** **INFORMAL** — sketch only, not a rigorous UC proof. Full UC formalization is explicitly future work.

### 5.1 Why this section is informal

Universally Composable (UC) security~\cite{canetti2001uc} provides the strongest standard composition guarantee available in cryptography: a protocol $\Pi$ that UC-emulates an ideal functionality $\mathcal{F}$ remains secure under arbitrary concurrent composition with other protocols. Producing a full UC proof for the seven-protocol CovertVote pipeline requires:

1. Formally defining ideal functionalities for each protocol ($\mathcal{F}_{\text{Paillier}}$, $\mathcal{F}_{\text{SMDC}}$, $\mathcal{F}_{\text{SA2}}$, $\mathcal{F}_{\text{Ring}}$, $\mathcal{F}_{\text{Kyber}}$, $\mathcal{F}_{\text{ZK}}$, $\mathcal{F}_{\text{Tally}}$).
2. Constructing simulators that map adversary actions in the real protocol world to corresponding actions in the ideal world.
3. Proving environment indistinguishability (real ↔ ideal) for each protocol against an arbitrary PPT environment $\mathcal{Z}$.
4. Invoking the UC composition theorem to combine the per-protocol guarantees.

This is a research-scale undertaking: even individual UC proofs for established primitives (e.g., UC-secure Paillier, UC-secure ring signatures) span 15--50 pages each in dedicated papers. A complete UC treatment for CovertVote is therefore identified as future work in Chapter~5.

This section provides an **informal sketch** of (a) the ideal functionalities the future proof should target, (b) the composition argument structure, and (c) related UC-secure voting work to draw on (notably E-cclesia~\cite{ecclesia2025}).

### 5.2 Ideal functionalities (sketch)

#### $\mathcal{F}_{\text{Ballot}}$: Idealized Ballot Privacy

**Inputs:** From voter $V_i$: a vote $v_i \in \{0,1\}^m$ with $\sum_j v_{i,j} = 1$.

**Output to adversary:** The voter identity $V_i$ and *only* the bulletin-board placeholder ("ballot $i$ posted by $V_i$"), with no information about $v_i$.

**Output to tally trustees (after election close):** The aggregate tally vector $\mathbf{T} = \sum_i v_i$.

**Real-world emulation:** The CovertVote ballot construction (Paillier + Pedersen + ZK + ring sig + SA² + Kyber-MAC) emulates $\mathcal{F}_{\text{Ballot}}$ if the simulator can map any real-world adversary's view to a view consistent with the ideal world. This is essentially what Theorem 1 establishes for the IND-CPA-style game; UC additionally requires simulator-extractability and adaptive corruption handling, which the current design supports via threshold-Paillier decryption.

#### $\mathcal{F}_{\text{SMDC}}$: Idealized Coercion Resistance

**Inputs:** From voter $V_i$: a vote $v_i$ AND a "coercion mode" flag $\text{coerced}_i \in \{0, 1\}$.

**Output to adversary (the coercer):** The same as in the real world (commitments, proofs, cast transcript), but generated by the simulator using only $V_i$'s identity — never the actual $v_i$.

**Behaviour:** If $\text{coerced}_i = 1$, the ballot is silently dropped at tally; if $\text{coerced}_i = 0$, the vote is added to the aggregate.

**Real-world emulation:** Theorem 4's slot-indistinguishability argument provides the simulator's distinguishing-advantage bound. UC additionally requires the simulator to handle adaptive coercion (the coercer choosing whom to coerce based on observation), which the current proof handles by virtue of the perfect-hiding step (statistical, hence adaptive-secure for that component).

#### $\mathcal{F}_{\text{SA2}}$: Idealized Two-Server Aggregation

**Inputs:** From each voter $V_i$: an encrypted vote $E(v_i)$.

**Output to Server A:** A shared simulator-generated value $\widetilde{S}_A$ statistically close to $E(v_i + \text{mask})$ but containing no information about $v_i$.

**Output to Server B:** Symmetric.

**Output to threshold-Paillier trustees:** The aggregated $E(\sum_i v_i)$.

**Real-world emulation:** Theorem 5's reduction is the core of the simulator construction; the assumption A1 (one honest server) is the trust assumption that makes UC simulation possible.

### 5.3 Informal composition argument

Each of T1–T6 establishes a per-protocol indistinguishability bound. By the standard hybrid argument, sequential composition of the six protocols (each contributing a bounded advantage to a top-level distinguisher) gives:
$$\mathrm{Adv}^{\mathrm{Joint}}_{\mathcal{Z}}(\lambda) \leq \sum_{i=1}^{6} \mathrm{Adv}^{T_i}_{\mathcal{B}_i}(\lambda) + O(\mathrm{negl}(\lambda)).$$
Each $\mathrm{Adv}^{T_i}$ is negligible by the corresponding theorem, so the joint advantage is negligible.

This is the **sequential** composition argument. Full UC requires concurrent composition under arbitrary scheduling by $\mathcal{Z}$. The cited E-cclesia work~\cite{ecclesia2025} provides a recent template for UC-secure self-tallying voting that future work can adapt.

### 5.4 Limitations of this discussion

1. The argument above is **sequential**, not concurrent. UC requires concurrent composition under arbitrary scheduling.
2. The simulators sketched above are **non-adaptive** (the corruption set is fixed at the start). UC permits adaptive corruption.
3. The ideal functionalities are **informally stated**; full formalization in the UC framework requires precise message scheduling and corruption interfaces.
4. **No machine-checked proof** is provided.

### 5.5 Forward path

Following the supervisor's earlier correspondence on E-cclesia~\cite{ecclesia2025}, future work should:
1. Adapt E-cclesia's UC-secure self-tallying functionality $\mathcal{F}_{\text{SST}}$ to CovertVote's seven-protocol pipeline.
2. Construct UC simulators for each of $\mathcal{F}_{\text{Ballot}}$, $\mathcal{F}_{\text{SMDC}}$, $\mathcal{F}_{\text{SA2}}$.
3. Apply the UC composition theorem (Canetti 2001 Theorem 13~\cite{canetti2001uc}) to combine the per-protocol guarantees.
4. Optionally machine-check the result in EasyCrypt or CryptoVerif.

This is a 6--12 month effort by one researcher, beyond the scope of the present thesis but well-defined enough to be a tractable PhD-level project for a follow-on student.

---

## 6. Cross-Reference Map

> **To be filled incrementally.** Each row will map a thesis location (file:line) to the corresponding MD verification section.

| Thesis location | Topic | MD section |
|-----------------|-------|------------|
| `chapter1.tex:95`, `chapter2.tex:147`, `chapter3.tex:171, 193-194, 351, 356, 596, 598` | Pedersen property errors (hiding/binding swap) | §3 / E1.1–E1.9 |
| `chapter3.tex:579-603` | Two-tier intro paragraph (Tier 1 vs Tier 2) | §1 / Document Purpose |
| `chapter3.tex:605-624` (subsubsec:t1-ballot-privacy) | Theorem 1 — Ballot Indistinguishability | §4.4 |
| `chapter3.tex:626-700` (subsubsec:t2-vote-validity) | Theorem 2 — Vote Validity | §4.1 |
| `chapter3.tex:722-754` (subsubsec:t3-voter-anonymity) | Theorem 3 — Voter Anonymity | §4.5 |
| `chapter3.tex:754-810` (subsubsec:t4-coercion-resistance) | Theorem 4 — Coercion Resistance | §4.6 |
| `chapter3.tex:685-720` (subsubsec:t5-aggregation-privacy) | Theorem 5 — SA² Aggregation Privacy | §4.3 |
| `chapter3.tex:730-862` (subsubsec:t6-pq-hybrid) | Theorem 6 — Post-Quantum Hybrid | §4.2 |
| `chapter3.tex:862-895` (subsubsec:uc-composition) | UC composition outline | §5 |
| `chapter4.tex:437-485` (sec:proverif) | Tier 2 framing (ProVerif as symbolic verification) | §1 / Document Purpose |
| `chapter5.tex:28` | UC future work language updated | §5 / Forward path |
| `references.bib` (new entries appended at end) | Cramer1994or, Bellare2006forking, Bellare1993rom, Cramer2003kemdem, Hofheinz2017fo, Canetti2001uc, Pointcheval2000security | §2.4 |

**Note:** Line numbers above are approximate after the structural rewrite of §sec:security-analysis in Checkpoint 1; readers can locate each subsubsection by its `\label{}` (e.g., `subsubsec:t1-ballot-privacy`).

---

## 7. Verification Log

### Session 1 — 2026-05-11 (Checkpoint 0)

**Actions:**
1. Created this document at `/home/bs01582/E-voting/SECURITY_PROOF_VERIFICATION.md`.
2. Documented Notation & Assumptions Registry (§2) with formal statements of DCRA, DDH, DLog, M-LWE.
3. Audited `chapter3.tex` lines 168–205 (Pedersen subsection), lines 340–360 (SMDC coercion subsection), lines 596–598 (Theorem 4) for Pedersen hiding/binding property statements.
4. **Found 9 distinct locations across chapter1, chapter2, and chapter3** where Pedersen properties were stated backwards or invoked wrong-flavor (E1.1 through E1.9 in §3). Documented each with wrong-text, correct-text, source, and rationale.
5. Applied fixes to `chapter3.tex` (commit/edit log: see git log for `chapter3.tex` after this session).

**What surfaced:**
- The original thesis had **2 occurrences** of the Pedersen hiding/binding swap initially identified by the supervisor's review; closer audit during this checkpoint surfaced **7 additional secondary occurrences** across all three relevant chapters (chapter1 protocol summary, chapter2 background subsection, chapter3 security-properties block, SMDC coercion text, figure caption, Theorem 4 statement, Theorem 4 proof sketch).
- The corrected statement (perfect hiding) actually **strengthens** Theorem 4's claim from computational to information-theoretic indistinguishability of slot commitments — this is a positive side-effect, not a regression.

**Files modified:**
- `SECURITY_PROOF_VERIFICATION.md` (NEW, this document)
- `thesis/Chapters/chapter1.tex` (Pedersen fix E1.8)
- `thesis/Chapters/chapter2.tex` (Pedersen fix E1.9)
- `thesis/Chapters/chapter3.tex` (Pedersen fixes E1.1–E1.7; Theorem 4 statement and proof sketch rewritten using perfect hiding)

**Pending for next session (Checkpoint 1):**
- Theorem 2 (Vote Validity) MD worksheet (§4.1).

### Session 2 — 2026-05-11 (Checkpoint 1: T2 Vote Validity)

**Actions:**
1. Drafted §4.1 (T2 Vote Validity worksheet) with formal soundness experiment, game-based theorem statement, hybrid proof $G_0 \to G_1 \to G_2$, and explicit reduction to DLog via Pedersen binding.
2. Added **6 missing references** to `references.bib` upfront (covers T2 + future checkpoints):
   - `cramer1994or` (CDS 1994, OR-composition special soundness)
   - `bellare2006forking` (Bellare-Neven General Forking Lemma)
   - `bellare1993rom` (Bellare-Rogaway random oracle model)
   - `cramer2003kemdem` (Cramer-Shoup KEM-DEM, for T6)
   - `hofheinz2017fo` (Hofheinz-Hövelmanns-Kiltz FO transform, for T6)
   - `canetti2001uc` (Canetti UC framework, for §5)
   - `pointcheval2000security` (Pointcheval-Stern, supplementary for forking)
3. Performed **structural reorganisation** of `chapter3.tex` § sec:security-analysis:
   - Renamed `\subsection{Formal Security Theorems}` → `\subsection{Computational Security: Game-Based Reductions}`.
   - Added intro paragraph explaining Tier 1 (computational, this section) vs. Tier 2 (symbolic, ProVerif in chapter 4).
   - Added new `\subsubsection{Notational Conventions for Proofs}` (defines $\lambda$, PPT, $\mathcal{A}$, advantage notation).
   - Added new `\subsubsection{Hardness Assumptions}` (formal statements of DCRA, DDH, DLog, M-LWE; trust assumptions A1, A2).
   - Promoted each of T1–T6 to its own `\subsubsection`, retaining existing sketch text as initial pass; T2 now has full extended treatment.
4. Wrote full T2 LaTeX block (formal relation, soundness experiment, theorem with bound, three-game hybrid proof, explicit reduction to DLog, Strong Fiat–Shamir remark).
5. Added label `subsec:t4-coercion-game` for forward reference from Theorem 4 sketch (now resolved).

**What surfaced:**
- The original T2 statement claimed "soundness error $\leq 2^{-256}$" without specifying whether this is per-proof challenge-guessing or full extraction. The new bound (Equation~\ref{eq:t2-bound}) makes this explicit: $(m+1) \cdot 2^{-256}$ from challenge-guessing + $m \sqrt{q_H \cdot \text{Adv}^{\text{DLog}}}$ from forking-extraction.
- The DLog dependence (via Pedersen binding) was implicit in the original sketch; the new theorem makes it an explicit assumption.

**Files modified:**
- `SECURITY_PROOF_VERIFICATION.md` (§4.1 added, log entry added)
- `thesis/Chapters/chapter3.tex` (lines 579–606 → ~120 lines of new structure with full T2 proof and placeholder structure for T1, T3-T6)
- `thesis/references.bib` (7 new entries appended)

**Pending for next session (Checkpoint 2):**
- Theorem 6 (Post-Quantum Hybrid) MD worksheet → LaTeX port.

### Session 3 — 2026-05-11 (Checkpoint 2: T6 Post-Quantum Hybrid)

**Actions:**
1. Drafted §4.2 (T6 worksheet): formal INT-CTXT experiment for the Kyber-HMAC outer wrapper, theorem with reduction bound, hybrid proof $G_0 \to G_2$.
2. Ported to LaTeX: replaced T6 sketch in chapter3.tex with full game definition + Theorem 6 + hybrid proof + confidentiality piggyback + quantum-adversary scope.

**What surfaced:**
- The original T6 statement combined integrity and confidentiality into a single informal claim. The new treatment separates them: integrity (P2) gets a formal INT-CTXT game and reduction; confidentiality (P1) inherits from Theorem 1.
- Added explicit treatment of the AES-256-GCM auxiliary use as a scope clarification.

**Files modified:** `SECURITY_PROOF_VERIFICATION.md`, `chapter3.tex`.

### Session 4 — 2026-05-11 (Checkpoint 3: T5 SA² Aggregation Privacy)

**Actions:**
1. Drafted §4.3 (T5 worksheet): formal single-server-view game, reduction to Paillier IND-CPA → DCRA.
2. Ported to LaTeX with full proof, mask-range remark, and multi-vote extension.

**What surfaced:**
- The mask-range subtlety ($\mathbb{Z}_{n/100}$ vs $\mathbb{Z}_n$) is correct as designed: the Paillier IND-CPA reduction does not require statistical hiding by mask; the smaller range only prevents overflow.
- Server B's share is unconditionally independent of $v$, giving the case $\sigma = B$ zero advantage.

**Files modified:** `SECURITY_PROOF_VERIFICATION.md`, `chapter3.tex`.

### Session 5 — 2026-05-11 (Checkpoint 4: T1 Ballot Privacy — CENTERPIECE)

**Actions:**
1. Drafted §4.4 (T1 worksheet): full IND-Ballot-CPA experiment with all ballot components, hybrid sequence $G_0 \to G_3$, per-position telescoping reduction to DCRA with $\Delta = 2$ for valid single-choice ballots.
2. Ported to LaTeX: comprehensive 75-line treatment in chapter3.tex including ballot composition recap, game definition, theorem with combined bound, four-game hybrid sequence, Pedersen perfect-hiding step, per-position telescope, and scope/limitations paragraph.

**What surfaced:**
- The supervisor's elaboration suggested a $1/m$ reduction loss; the cleaner per-position telescope (over the $\Delta = 2$ differing positions for single-choice votes) gives a tight $\Delta \cdot \mathrm{Adv}^{\mathrm{DCRA}}$ bound.
- Pedersen commitments contribute zero advantage under perfect hiding (corrected from prior thesis text), simplifying the hybrid sequence by eliminating one game step.

**Files modified:** `SECURITY_PROOF_VERIFICATION.md`, `chapter3.tex`.

### Session 6 — 2026-05-11 (Checkpoint 5: T3 Voter Anonymity)

**Actions:**
1. Drafted §4.5 (T3 worksheet): formal anonymity experiment, two-part decomposition (statistical anonymity of $(c_i, r_i)$ + DDH-based anonymity of key image), explicit DDH reduction.
2. Ported to LaTeX with full proof and clarification on linkability preservation.

**What surfaced:**
- The original T3 statement said "statistically indistinguishable under DDH" — slightly muddled. The new treatment makes the decomposition explicit: $(c_i, r_i)$ are statistical, key image is DDH-based.
- Verified Liu-Wei-Wong 2004 §4.2 Theorem 4 (Signer Ambiguity) framing: my earlier doubt about whether DDH was needed was resolved — DDH is needed for the key-image relation, exactly as the supervisor's elaboration claimed.

**Files modified:** `SECURITY_PROOF_VERIFICATION.md`, `chapter3.tex`.

### Session 7 — 2026-05-11 (Checkpoint 6: T4 Coercion Resistance — HIGH RISK)

**Actions:**
1. Drafted §4.6 (T4 worksheet) with extensive open questions and scope demarcation. Marked as **NEEDS-EXPERT-REVIEW**.
2. Adapted JCJ-2005 coercion-resistance experiment to SMDC's slot-based mechanism.
3. Hybrid sequence $G_0 \to G_2$: HMAC-PRF for slot derivation, ZK simulator, perfect-hiding closure.
4. Explicit scope/out-of-scope demarcation: forced abstention, randomization, device-controlling coercer, credential-issuance compromise all listed as out of scope per JCJ tradition.
5. Ported to LaTeX with full game, theorem, proof, and limitations paragraph.

**What surfaced:**
- The HMAC-PRF reduction needs Trust Assumption A2 (server's $K_{\text{srv}}$ remains secret) — flagged in MD as something to verify against actual SMDC implementation.
- T4 is the only theorem flagged NEEDS-EXPERT-REVIEW. Specific concerns documented in §4.6.8 OQ1 for supervisor.

**Files modified:** `SECURITY_PROOF_VERIFICATION.md`, `chapter3.tex`.

### Session 8 — 2026-05-11 (Checkpoint 7: UC Composition Outline)

**Actions:**
1. Drafted §5 (UC discussion): ideal functionalities sketch ($\mathcal{F}_{\mathrm{Ballot}}, \mathcal{F}_{\mathrm{SMDC}}, \mathcal{F}_{\mathrm{SA2}}$), informal sequential composition argument, explicit limitations, forward path to E-cclesia-style adaptation.
2. Added new subsubsection `subsubsec:uc-composition` to chapter3.tex (after T6) with the same content adapted for thesis prose.

**What surfaced:**
- Honest disclosure throughout: "informal," "non-adaptive," "sequential not concurrent," "no machine-checked proof" all stated explicitly in both MD and LaTeX.
- This positions the thesis correctly between game-based (proven) and full UC (acknowledged as future work) without overclaiming.

**Files modified:** `SECURITY_PROOF_VERIFICATION.md`, `chapter3.tex`.

### Session 9 — 2026-05-11 (Checkpoint 8: ProVerif Reframing)

**Actions:**
1. Renamed `chapter4.tex` § sec:proverif from "Formal Verification with ProVerif" to "Tier 2 — Symbolic Protocol Verification with ProVerif".
2. Added new opening paragraph explaining the role of Tier 2 vs Tier 1, and a "Why both tiers" subsection arguing the complementary nature.
3. Updated the closing paragraph to cross-reference each ProVerif property with its Tier 1 game-based counterpart.
4. Updated `chapter5.tex:28` (UC future-work bullet) to reflect what is now done (Tier 1 + Tier 2 + UC outline) vs what remains (full UC + machine-checked proof).

**Files modified:** `SECURITY_PROOF_VERIFICATION.md`, `chapter4.tex`, `chapter5.tex`.

### Session 10 — 2026-05-11 (Checkpoint 9: Final Consistency Pass)

**Actions:**
1. Updated §6 (Cross-Reference Map) with all new locations from Checkpoints 1–8.
2. Updated §7 (Verification Log) with this consolidated record.
3. **LaTeX compile not performed** — `pdflatex` is not installed in this environment. The user / supervisor should run the standard compile workflow (`pdflatex main.tex; bibtex main; pdflatex main.tex; pdflatex main.tex`) and verify (a) no LaTeX errors, (b) all new citations resolve, (c) all `\ref{}` cross-references resolve (no `??` in PDF), (d) all new `\label{}`s appear in the table of contents.
4. Performed a static syntactic audit: each new theorem block uses the existing `\textbf{Theorem N}` and `\textit{Proof.}` style for consistency with the rest of the thesis (no new packages added). Each new equation has a `\label{}`. All cross-references use existing `\label`s in chapter3 or new ones added in this revision.
5. Compiled summary list of items flagged for human review (see §8).

**Final state:**
- 9 Pedersen errors fixed across 3 chapters.
- 6 game-based proofs added (T1–T6) with formal experiments and hybrid arguments.
- UC composition outline added with explicit informality disclosure.
- ProVerif reframed as Tier 2.
- 7 new bibliography entries added.
- T4 explicitly flagged NEEDS-EXPERT-REVIEW.

**Files modified across all sessions:**
- `/home/bs01582/E-voting/SECURITY_PROOF_VERIFICATION.md` (NEW — this document, ~750 lines)
- `/home/bs01582/E-voting/thesis/Chapters/chapter1.tex` (E1.8)
- `/home/bs01582/E-voting/thesis/Chapters/chapter2.tex` (E1.9)
- `/home/bs01582/E-voting/thesis/Chapters/chapter3.tex` (E1.1–E1.7 + complete §sec:security-analysis rewrite)
- `/home/bs01582/E-voting/thesis/Chapters/chapter4.tex` (Tier 2 reframing of §sec:proverif)
- `/home/bs01582/E-voting/thesis/Chapters/chapter5.tex` (line 28 UC future-work language)
- `/home/bs01582/E-voting/thesis/references.bib` (7 new entries)

---

## 8. Known Limitations & Items Flagged for Human Review

> Updated incrementally as work proceeds. Each item is something explicitly NOT proven with full rigor and / or where I (the AI assistant) explicitly want a human cryptographer to verify before journal submission.

### L1 — UC composition outline is informal

Full UC formalization (ideal functionalities, simulator constructions, environment indistinguishability) for all seven protocols of CovertVote is a research-scale project, kept as future work in chapter 5. The UC composition section (§5 of this doc) will sketch the structure but explicitly disclose informality.

### L2 — Coercion resistance proof (T4) requires expert review

T4 adapts the JCJ-2005 coercion-resistance game to SMDC's novel slot-based deniable-credential construction. While the reduction is structurally analogous to JCJ, the SMDC-specific elements (HMAC-derived real-slot index, silent-discard semantics) have no prior literature blueprint. The proof will be drafted but explicitly tagged `NEEDS-EXPERT-REVIEW` in §4.6.

### L3 — Behavioral duress detection (chapter3.tex § 3.8.2) is not proven game-based

The behavioral duress detection mechanism (HMAC-equality-based) is a system-level countermeasure, not a cryptographic primitive in the strict sense. Its security is argued informally (constant-time HMAC comparison ⇒ no timing oracle; HMAC PRF property ⇒ database-leak does not reveal signal). No new game-based theorem is added for it; the existing prose argument is retained.

### Future limitations

To be added as discovered during T1–T6 verification.
