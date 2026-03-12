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
	hashBytes := SHA3Hash(g.Bytes())

	hashInt := new(big.Int).SetBytes(hashBytes)
	hashInt.Mod(hashInt, p)

	pMinus1 := new(big.Int).Sub(p, big.NewInt(1))
	exp := new(big.Int).Div(pMinus1, q)

	h := new(big.Int).Exp(hashInt, exp, p)

	// Ensure h != 1 and h != g
	if h.Cmp(big.NewInt(1)) == 0 || h.Cmp(g) == 0 {
		// Add salt and retry
		hashBytes = SHA3HashMultiple(g.Bytes(), []byte("salt"))
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
	gm := new(big.Int).Exp(pp.G, m, pp.P) // g^m mod p
	hr := new(big.Int).Exp(pp.H, r, pp.P) // h^r mod p
	c := new(big.Int).Mul(gm, hr)          // g^m × h^r
	c.Mod(c, pp.P)                         // mod p

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
