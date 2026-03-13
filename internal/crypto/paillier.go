package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// GenerateRandomBigInt generates a random big.Int less than max
func GenerateRandomBigInt(max *big.Int) (*big.Int, error) {
	r, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// PaillierPublicKey holds the public key
type PaillierPublicKey struct {
	N  *big.Int // n = p*q
	G  *big.Int // g = n+1
	N2 *big.Int // n²
}

// PaillierPrivateKey holds the private key
type PaillierPrivateKey struct {
	PublicKey *PaillierPublicKey
	Lambda    *big.Int // λ = lcm(p-1, q-1)
	Mu        *big.Int // μ = L(g^λ mod n²)^(-1) mod n
	P         *big.Int // prime p (for threshold)
	Q         *big.Int // prime q (for threshold)
}

// GeneratePaillierKeyPair generates a new Paillier key pair
// bits: key size (minimum 2048 for security)
func GeneratePaillierKeyPair(bits int) (*PaillierPrivateKey, error) {
	if bits < 2048 {
		return nil, fmt.Errorf("paillier: key size must be >= 2048 bits for security, got %d", bits)
	}
	if bits%2 != 0 {
		return nil, fmt.Errorf("paillier: key size must be even, got %d", bits)
	}

	// Step 1: Generate two large primes p and q
	p, err := rand.Prime(rand.Reader, bits/2)
	if err != nil {
		return nil, err
	}

	q, err := rand.Prime(rand.Reader, bits/2)
	if err != nil {
		return nil, err
	}

	// Ensure p != q
	for p.Cmp(q) == 0 {
		q, err = rand.Prime(rand.Reader, bits/2)
		if err != nil {
			return nil, err
		}
	}

	// Step 2: Compute n = p × q
	n := new(big.Int).Mul(p, q)

	// Compute n²
	n2 := new(big.Int).Mul(n, n)

	// Step 3: Compute λ = lcm(p-1, q-1)
	p1 := new(big.Int).Sub(p, big.NewInt(1)) // p-1
	q1 := new(big.Int).Sub(q, big.NewInt(1)) // q-1
	lambda := lcm(p1, q1)

	// Step 4: g = n + 1 (standard choice)
	g := new(big.Int).Add(n, big.NewInt(1))

	// Step 5: Compute μ = L(g^λ mod n²)^(-1) mod n
	// where L(x) = (x-1)/n
	gLambda := new(big.Int).Exp(g, lambda, n2) // g^λ mod n²
	l := lFunction(gLambda, n)                  // L(g^λ mod n²)
	mu := new(big.Int).ModInverse(l, n)         // μ = L^(-1) mod n

	if mu == nil {
		return nil, errors.New("failed to compute modular inverse")
	}

	publicKey := &PaillierPublicKey{
		N:  n,
		G:  g,
		N2: n2,
	}

	return &PaillierPrivateKey{
		PublicKey: publicKey,
		Lambda:    lambda,
		Mu:        mu,
		P:         p,
		Q:         q,
	}, nil
}

// lFunction computes L(x) = (x-1)/n
func lFunction(x, n *big.Int) *big.Int {
	// L(x) = (x - 1) / n
	xMinus1 := new(big.Int).Sub(x, big.NewInt(1))
	return new(big.Int).Div(xMinus1, n)
}

// lcm computes the least common multiple of a and b
func lcm(a, b *big.Int) *big.Int {
	// lcm(a,b) = (a × b) / gcd(a,b)
	gcd := new(big.Int).GCD(nil, nil, a, b)
	ab := new(big.Int).Mul(a, b)
	return new(big.Int).Div(ab, gcd)
}

// Encrypt encrypts a plaintext message m
// Returns ciphertext c = g^m × r^n mod n²
func (pk *PaillierPublicKey) Encrypt(m *big.Int) (*big.Int, error) {
	// Validate: 0 <= m < n
	if m.Sign() < 0 || m.Cmp(pk.N) >= 0 {
		return nil, errors.New("message out of range")
	}

	// Choose random r where 0 < r < n
	r, err := rand.Int(rand.Reader, pk.N)
	if err != nil {
		return nil, err
	}
	// Ensure r > 0
	for r.Sign() == 0 {
		r, err = rand.Int(rand.Reader, pk.N)
		if err != nil {
			return nil, err
		}
	}

	// Compute g^m mod n²
	gm := new(big.Int).Exp(pk.G, m, pk.N2)

	// Compute r^n mod n²
	rn := new(big.Int).Exp(r, pk.N, pk.N2)

	// Compute c = g^m × r^n mod n²
	c := new(big.Int).Mul(gm, rn)
	c.Mod(c, pk.N2)

	return c, nil
}

// EncryptWithRandomness encrypts with specified randomness (for proofs)
func (pk *PaillierPublicKey) EncryptWithRandomness(m, r *big.Int) (*big.Int, error) {
	gm := new(big.Int).Exp(pk.G, m, pk.N2)
	rn := new(big.Int).Exp(r, pk.N, pk.N2)
	c := new(big.Int).Mul(gm, rn)
	c.Mod(c, pk.N2)
	return c, nil
}

// Decrypt decrypts a ciphertext c
// Returns plaintext m = L(c^λ mod n²) × μ mod n
func (sk *PaillierPrivateKey) Decrypt(c *big.Int) (*big.Int, error) {
	pk := sk.PublicKey

	// Compute c^λ mod n²
	cLambda := new(big.Int).Exp(c, sk.Lambda, pk.N2)

	// Compute L(c^λ mod n²)
	l := lFunction(cLambda, pk.N)

	// Compute m = L × μ mod n
	m := new(big.Int).Mul(l, sk.Mu)
	m.Mod(m, pk.N)

	return m, nil
}

// Add performs homomorphic addition: E(m1 + m2) = E(m1) × E(m2) mod n²
func (pk *PaillierPublicKey) Add(c1, c2 *big.Int) *big.Int {
	result := new(big.Int).Mul(c1, c2)
	result.Mod(result, pk.N2)
	return result
}

// AddPlaintext adds plaintext to ciphertext: E(m1 + m2) = E(m1) × g^m2 mod n²
func (pk *PaillierPublicKey) AddPlaintext(c, m *big.Int) *big.Int {
	gm := new(big.Int).Exp(pk.G, m, pk.N2)
	result := new(big.Int).Mul(c, gm)
	result.Mod(result, pk.N2)
	return result
}

// Multiply performs scalar multiplication: E(k × m) = E(m)^k mod n²
func (pk *PaillierPublicKey) Multiply(c, k *big.Int) *big.Int {
	result := new(big.Int).Exp(c, k, pk.N2)
	return result
}

// AddMultiple adds multiple ciphertexts
func (pk *PaillierPublicKey) AddMultiple(ciphertexts []*big.Int) *big.Int {
	result := big.NewInt(1)
	for _, c := range ciphertexts {
		result = pk.Add(result, c)
	}
	return result
}
