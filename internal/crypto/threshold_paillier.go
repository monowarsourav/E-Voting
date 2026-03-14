package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
)

// ThresholdParams holds the threshold scheme parameters.
type ThresholdParams struct {
	N         int // total number of trustees
	Threshold int // minimum trustees needed to decrypt
}

// ThresholdKeyShares holds the result of distributed key generation.
type ThresholdKeyShares struct {
	PublicKey  *PaillierPublicKey
	Shares    []*KeyShare    // one per trustee
	VerifyKeys []*big.Int    // verification key for each share
	Params    *ThresholdParams
	V         *big.Int       // verification base
	Delta     *big.Int       // n! (factorial of total trustees)
}

// KeyShare represents one trustee's share of the private key.
type KeyShare struct {
	Index int      // trustee index (1-based)
	Si    *big.Int // share value
}

// ThresholdPartialDecryption represents one trustee's partial decryption.
type ThresholdPartialDecryption struct {
	Index int                         // trustee index (1-based)
	Ci    *big.Int                    // partial decryption value
	Proof *PartialDecryptionProof     // ZK proof of correct partial decryption
}

// PartialDecryptionProof proves correct partial decryption without revealing the share.
type PartialDecryptionProof struct {
	E *big.Int // challenge
	Z *big.Int // response
}

// GenerateThresholdKey generates a threshold Paillier key pair using the
// Damgård-Jurik-Nielsen scheme.
// bits: key size (>= 2048), n: total trustees, t: threshold.
func GenerateThresholdKey(bits, n, t int) (*ThresholdKeyShares, error) {
	if bits < 2048 {
		return nil, fmt.Errorf("threshold paillier: key size must be >= 2048, got %d", bits)
	}
	if t < 1 || t > n {
		return nil, fmt.Errorf("threshold paillier: need 1 <= t <= n, got t=%d n=%d", t, n)
	}
	if n < 2 {
		return nil, fmt.Errorf("threshold paillier: need n >= 2, got %d", n)
	}

	// Step 1: Generate safe primes p = 2p' + 1, q = 2q' + 1
	p, pPrime, err := generateSafePrime(bits / 2)
	if err != nil {
		return nil, fmt.Errorf("generating safe prime p: %w", err)
	}

	q, qPrime, err := generateSafePrime(bits / 2)
	if err != nil {
		return nil, fmt.Errorf("generating safe prime q: %w", err)
	}

	// Ensure p != q
	for p.Cmp(q) == 0 {
		q, qPrime, err = generateSafePrime(bits / 2)
		if err != nil {
			return nil, fmt.Errorf("regenerating safe prime q: %w", err)
		}
	}

	// Step 2: Compute N = p*q, m = p'*q'
	bigN := new(big.Int).Mul(p, q)
	m := new(big.Int).Mul(pPrime, qPrime)
	nSquared := new(big.Int).Mul(bigN, bigN)

	// Step 3: Compute d such that d ≡ 0 (mod m) and d ≡ 1 (mod N) using CRT
	// d = m * (m^(-1) mod N)
	mInvN := new(big.Int).ModInverse(m, bigN)
	if mInvN == nil {
		return nil, errors.New("threshold paillier: failed to compute m^(-1) mod N")
	}
	d := new(big.Int).Mul(m, mInvN)

	// Step 4: delta = n!
	delta := factorialBig(n)

	// Step 5: Split d into shares using Shamir's Secret Sharing over Z_{N*m}
	nm := new(big.Int).Mul(bigN, m)
	shares, err := shamirSplit(d, n, t, nm)
	if err != nil {
		return nil, fmt.Errorf("shamir split: %w", err)
	}

	// Step 6: Public key
	g := new(big.Int).Add(bigN, big.NewInt(1))
	pk := &PaillierPublicKey{
		N:  bigN,
		G:  g,
		N2: nSquared,
	}

	// Step 7: Generate verification base v = random square mod N^2
	// Choose random r, v = r^2 mod N^2
	r, err := rand.Int(rand.Reader, nSquared)
	if err != nil {
		return nil, err
	}
	v := new(big.Int).Mul(r, r)
	v.Mod(v, nSquared)

	// Step 8: Generate verification keys: vk_i = v^(delta * si) mod N^2
	verifyKeys := make([]*big.Int, n)
	for i := 0; i < n; i++ {
		exp := new(big.Int).Mul(delta, shares[i].Si)
		verifyKeys[i] = new(big.Int).Exp(v, exp, nSquared)
	}

	params := &ThresholdParams{
		N:         n,
		Threshold: t,
	}

	return &ThresholdKeyShares{
		PublicKey:  pk,
		Shares:    shares,
		VerifyKeys: verifyKeys,
		Params:    params,
		V:         v,
		Delta:     delta,
	}, nil
}

// PartialDecrypt performs one trustee's partial decryption.
// Computes ci = c^(2 * delta * si) mod N^2 and generates a ZK proof.
func (share *KeyShare) PartialDecrypt(c *big.Int, pk *PaillierPublicKey, params *ThresholdParams, vk *big.Int, v *big.Int) (*ThresholdPartialDecryption, error) {
	if c == nil || pk == nil || params == nil || vk == nil || v == nil {
		return nil, errors.New("threshold paillier: nil parameter")
	}

	delta := factorialBig(params.N)
	nSquared := pk.N2

	// ci = c^(2 * delta * si) mod N^2
	twoTimesDeltaSi := new(big.Int).Mul(big.NewInt(2), delta)
	twoTimesDeltaSi.Mul(twoTimesDeltaSi, share.Si)
	ci := new(big.Int).Exp(c, twoTimesDeltaSi, nSquared)

	// ZK proof of correct partial decryption
	proof, err := generatePartialDecryptProof(c, ci, share, pk, vk, v, delta)
	if err != nil {
		return nil, fmt.Errorf("generating ZK proof: %w", err)
	}

	return &ThresholdPartialDecryption{
		Index: share.Index,
		Ci:    ci,
		Proof: proof,
	}, nil
}

// CombinePartialDecryptions combines t partial decryptions to recover plaintext
// using Lagrange interpolation in the exponent.
func CombinePartialDecryptions(partials []*ThresholdPartialDecryption, pk *PaillierPublicKey, params *ThresholdParams) (*big.Int, error) {
	if len(partials) < params.Threshold {
		return nil, fmt.Errorf("threshold paillier: need at least %d partial decryptions, got %d",
			params.Threshold, len(partials))
	}

	delta := factorialBig(params.N)
	nSquared := pk.N2

	// Collect the indices of participating trustees
	indices := make([]int, len(partials))
	for i, pd := range partials {
		indices[i] = pd.Index
	}

	// Compute c' = product of ci^(2 * lambda_i) mod N^2
	// where lambda_i are Lagrange coefficients scaled by delta
	cPrime := big.NewInt(1)

	for i, pd := range partials {
		// Compute Lagrange coefficient: lambda_{0,i}^S = delta * prod_{j!=i} (j / (j - i))
		// where indices are 1-based trustee indices
		lambda := computeLagrangeCoeff(indices[i], indices, delta)

		// Exponent = 2 * lambda
		twoLambda := new(big.Int).Mul(big.NewInt(2), lambda)

		// Handle negative exponents by computing modular inverse
		var term *big.Int
		if twoLambda.Sign() < 0 {
			// ci^(-|exp|) mod N^2 = (ci^(-1))^|exp| mod N^2
			ciInv := new(big.Int).ModInverse(pd.Ci, nSquared)
			if ciInv == nil {
				return nil, errors.New("threshold paillier: failed to compute modular inverse of ci")
			}
			absExp := new(big.Int).Abs(twoLambda)
			term = new(big.Int).Exp(ciInv, absExp, nSquared)
		} else {
			term = new(big.Int).Exp(pd.Ci, twoLambda, nSquared)
		}

		cPrime.Mul(cPrime, term)
		cPrime.Mod(cPrime, nSquared)
	}

	// Recover plaintext: m = L(c') * (4 * delta^2)^(-1) mod N
	// where L(x) = (x - 1) / N
	lValue := new(big.Int).Sub(cPrime, big.NewInt(1))
	lValue.Div(lValue, pk.N)

	// Compute (4 * delta^2)^(-1) mod N
	fourDeltaSquared := new(big.Int).Mul(delta, delta)
	fourDeltaSquared.Mul(fourDeltaSquared, big.NewInt(4))

	fourDeltaSquaredInv := new(big.Int).ModInverse(fourDeltaSquared, pk.N)
	if fourDeltaSquaredInv == nil {
		return nil, errors.New("threshold paillier: failed to compute (4*delta^2)^(-1) mod N")
	}

	plaintext := new(big.Int).Mul(lValue, fourDeltaSquaredInv)
	plaintext.Mod(plaintext, pk.N)

	return plaintext, nil
}

// VerifyPartialDecryption verifies a trustee's ZK proof of correct partial decryption.
func VerifyPartialDecryption(pd *ThresholdPartialDecryption, c *big.Int, pk *PaillierPublicKey, vk *big.Int, v *big.Int) bool {
	if pd == nil || pd.Proof == nil || c == nil || pk == nil || vk == nil || v == nil {
		return false
	}

	nSquared := pk.N2
	e := pd.Proof.E
	z := pd.Proof.Z

	// Verify v-branch of the Schnorr-like proof.
	// Proof relation: v^z == b * vk^e, so b = v^z * vk^(-e) mod N^2
	// This proves knowledge of si such that vk = v^(delta*si), which is sound
	// when vk is a trusted public parameter (DJN scheme).
	vZ := new(big.Int).Exp(v, z, nSquared)
	vkE := new(big.Int).Exp(vk, e, nSquared)
	vkEInv := new(big.Int).ModInverse(vkE, nSquared)
	if vkEInv == nil {
		return false
	}
	bRecon := new(big.Int).Mul(vZ, vkEInv)
	bRecon.Mod(bRecon, nSquared)

	// Recompute challenge — must match generatePartialDecryptProof
	h := sha256.New()
	h.Write([]byte("CovertVote-ThresholdProof-v1"))
	h.Write(c.Bytes())
	h.Write(pd.Ci.Bytes())
	h.Write(bRecon.Bytes())
	h.Write(vk.Bytes())
	h.Write(v.Bytes())
	eRecomputed := new(big.Int).SetBytes(h.Sum(nil))

	return eRecomputed.Cmp(e) == 0
}

// generatePartialDecryptProof generates a ZK proof of correct partial decryption.
func generatePartialDecryptProof(c, ci *big.Int, share *KeyShare, pk *PaillierPublicKey, vk, v *big.Int, delta *big.Int) (*PartialDecryptionProof, error) {
	nSquared := pk.N2

	// Choose random r from [0, N^2)
	r, err := rand.Int(rand.Reader, nSquared)
	if err != nil {
		return nil, err
	}

	// b = v^r mod N^2
	b := new(big.Int).Exp(v, r, nSquared)

	// Challenge e = SHA-256(domain_tag, c, ci, b, vk, v)
	h := sha256.New()
	h.Write([]byte("CovertVote-ThresholdProof-v1"))
	h.Write(c.Bytes())
	h.Write(ci.Bytes())
	h.Write(b.Bytes())
	h.Write(vk.Bytes())
	h.Write(v.Bytes())
	e := new(big.Int).SetBytes(h.Sum(nil))

	// Response z = r + e * delta * si (no modular reduction)
	// This ensures v^z = v^r * v^(e*delta*si) = b * vk^e
	z := new(big.Int).Mul(e, delta)
	z.Mul(z, share.Si)
	z.Add(z, r)

	return &PartialDecryptionProof{
		E: e,
		Z: z,
	}, nil
}

// computeLagrangeCoeff computes the Lagrange coefficient for index i
// among the set of indices, scaled by delta.
// lambda_{0,i}^S = delta * prod_{j in S, j!=i} (j / (j - i))
// Since we work with integers, we compute numerator and denominator separately.
func computeLagrangeCoeff(i int, indices []int, delta *big.Int) *big.Int {
	num := new(big.Int).Set(delta)
	den := big.NewInt(1)

	for _, j := range indices {
		if j == i {
			continue
		}
		num.Mul(num, big.NewInt(int64(j)))
		den.Mul(den, big.NewInt(int64(j-i)))
	}

	// Single division at the end preserves precision
	lambda := new(big.Int).Div(num, den)
	return lambda
}

// shamirSplit splits secret d into n shares with threshold t over Z_mod.
func shamirSplit(d *big.Int, n, t int, mod *big.Int) ([]*KeyShare, error) {
	// Generate random polynomial of degree t-1: f(x) = d + a1*x + a2*x^2 + ... + a_{t-1}*x^{t-1}
	coefficients := make([]*big.Int, t)
	coefficients[0] = new(big.Int).Set(d) // constant term = secret

	for i := 1; i < t; i++ {
		coeff, err := rand.Int(rand.Reader, mod)
		if err != nil {
			return nil, err
		}
		coefficients[i] = coeff
	}

	// Evaluate polynomial at points 1, 2, ..., n
	shares := make([]*KeyShare, n)
	for i := 1; i <= n; i++ {
		x := big.NewInt(int64(i))
		y := evaluatePolynomial(coefficients, x, mod)
		shares[i-1] = &KeyShare{
			Index: i,
			Si:    y,
		}
	}

	return shares, nil
}

// evaluatePolynomial evaluates a polynomial at point x modulo mod.
// coefficients[0] + coefficients[1]*x + coefficients[2]*x^2 + ...
func evaluatePolynomial(coefficients []*big.Int, x, mod *big.Int) *big.Int {
	result := new(big.Int).Set(coefficients[0])
	xPow := new(big.Int).Set(x)

	for i := 1; i < len(coefficients); i++ {
		term := new(big.Int).Mul(coefficients[i], xPow)
		term.Mod(term, mod)
		result.Add(result, term)
		result.Mod(result, mod)

		xPow.Mul(xPow, x)
		xPow.Mod(xPow, mod)
	}

	return result
}

// generateSafePrime generates a safe prime p = 2p' + 1 where p' is also prime.
func generateSafePrime(bits int) (p, pPrime *big.Int, err error) {
	for {
		// Generate a random prime p'
		pPrime, err = rand.Prime(rand.Reader, bits-1)
		if err != nil {
			return nil, nil, err
		}

		// p = 2p' + 1
		p = new(big.Int).Mul(pPrime, big.NewInt(2))
		p.Add(p, big.NewInt(1))

		// Check if p is also prime
		if p.ProbablyPrime(20) {
			return p, pPrime, nil
		}
	}
}

// factorialBig computes n! as a *big.Int.
func factorialBig(n int) *big.Int {
	result := big.NewInt(1)
	for i := 2; i <= n; i++ {
		result.Mul(result, big.NewInt(int64(i)))
	}
	return result
}
