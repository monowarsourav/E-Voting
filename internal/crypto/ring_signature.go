package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sort"
)

// RING_SIZE is the fixed ring size for O(n) scalability
// Using a constant ring size ensures O(n) complexity regardless of total voter count
// This is standard practice (Monero uses 11, academic papers use 50-200)
const RING_SIZE = 100

// RingParams holds ring signature parameters
type RingParams struct {
	P *big.Int // Prime modulus
	Q *big.Int // Order
	G *big.Int // Generator
}

// RingKeyPair represents a member's key pair
type RingKeyPair struct {
	PublicKey  *big.Int // pk = g^sk mod p
	PrivateKey *big.Int // sk
}

// RingSignature represents a linkable ring signature
type RingSignature struct {
	KeyImage  *big.Int   // I = sk × H(pk) - for linking
	Challenge *big.Int   // c0
	Responses []*big.Int // r0, r1, ..., rn-1
}

// GenerateRingParams generates ring signature parameters
func GenerateRingParams(bits int) (*RingParams, error) {
	pp, err := GeneratePedersenParams(bits)
	if err != nil {
		return nil, err
	}
	return &RingParams{
		P: pp.P,
		Q: pp.Q,
		G: pp.G,
	}, nil
}

// GenerateRingKeyPair generates a key pair for ring member
func (rp *RingParams) GenerateRingKeyPair() (*RingKeyPair, error) {
	// sk = random in Zq
	sk, err := rand.Int(rand.Reader, rp.Q)
	if err != nil {
		return nil, err
	}

	// pk = g^sk mod p
	pk := new(big.Int).Exp(rp.G, sk, rp.P)

	return &RingKeyPair{
		PublicKey:  pk,
		PrivateKey: sk,
	}, nil
}

// SelectRandomRing selects a fixed-size ring from all available public keys
// This ensures O(n) complexity by using a constant ring size regardless of total voters
// Returns: (fixedSizeRing, newSignerIndex)
func SelectRandomRing(allPublicKeys []*big.Int, signerKey *big.Int, signerIndex int) ([]*big.Int, int, error) {
	totalKeys := len(allPublicKeys)

	// If total keys <= RING_SIZE, use all keys
	if totalKeys <= RING_SIZE {
		return allPublicKeys, signerIndex, nil
	}

	// Validate signerIndex
	if signerIndex < 0 || signerIndex >= totalKeys {
		return nil, -1, errors.New("invalid signer index")
	}

	// Create a map to track selected indices (excluding signer)
	selectedIndices := make(map[int]bool)
	selectedIndices[signerIndex] = true // Mark signer as already selected

	// Randomly select (RING_SIZE - 1) other members
	for len(selectedIndices) < RING_SIZE {
		// Generate random index
		randomBig, err := rand.Int(rand.Reader, big.NewInt(int64(totalKeys)))
		if err != nil {
			return nil, -1, err
		}
		randomIdx := int(randomBig.Int64())

		// Add if not already selected
		if !selectedIndices[randomIdx] {
			selectedIndices[randomIdx] = true
		}
	}

	// Convert selected indices to sorted array for deterministic ordering
	indices := make([]int, 0, RING_SIZE)
	for idx := range selectedIndices {
		indices = append(indices, idx)
	}

	// Sort indices to maintain deterministic ordering (O(n log n)).
	sort.Ints(indices)

	// Build the fixed-size ring
	ring := make([]*big.Int, RING_SIZE)
	newSignerIndex := -1

	for i, idx := range indices {
		ring[i] = allPublicKeys[idx]
		if idx == signerIndex {
			newSignerIndex = i
		}
	}

	if newSignerIndex == -1 {
		return nil, -1, errors.New("signer not found in ring")
	}

	return ring, newSignerIndex, nil
}

// hashToPoint hashes a public key to a group element
func (rp *RingParams) hashToPoint(pk *big.Int) *big.Int {
	hashBytes := SHA3Hash(pk.Bytes())

	h := new(big.Int).SetBytes(hashBytes)
	h.Mod(h, rp.P)

	// Ensure it's in the subgroup
	pMinus1 := new(big.Int).Sub(rp.P, big.NewInt(1))
	exp := new(big.Int).Div(pMinus1, rp.Q)
	h.Exp(h, exp, rp.P)

	return h
}

// Sign creates a linkable ring signature
func (rp *RingParams) Sign(message []byte, signerKey *RingKeyPair, ring []*big.Int, signerIndex int) (*RingSignature, error) {
	n := len(ring)
	if signerIndex < 0 || signerIndex >= n {
		return nil, errors.New("invalid signer index")
	}

	// Step 1: Compute key image I = sk × H(pk)
	hp := rp.hashToPoint(signerKey.PublicKey)
	keyImage := new(big.Int).Exp(hp, signerKey.PrivateKey, rp.P)

	// Step 2: Initialize arrays
	challenges := make([]*big.Int, n)
	responses := make([]*big.Int, n)

	// Step 3: Generate random commitment for signer.
	// Entropy failures here would silently produce an insecure signature, so
	// we propagate the error rather than ignoring it.
	alpha, err := rand.Int(rand.Reader, rp.Q)
	if err != nil {
		return nil, fmt.Errorf("ring sign: read entropy for alpha: %w", err)
	}

	// L_s = g^alpha
	Ls := new(big.Int).Exp(rp.G, alpha, rp.P)
	// R_s = H(pk_s)^alpha
	Rs := new(big.Int).Exp(hp, alpha, rp.P)

	// Step 4: Compute starting challenge
	challenges[(signerIndex+1)%n] = rp.hashRing(message, Ls, Rs)

	// Step 5: Fill in simulated responses
	for i := 1; i < n; i++ {
		idx := (signerIndex + i) % n
		nextIdx := (idx + 1) % n

		// Random response
		responses[idx], err = rand.Int(rand.Reader, rp.Q)
		if err != nil {
			return nil, fmt.Errorf("ring sign: read entropy for response: %w", err)
		}

		// L_i = g^r_i × pk_i^c_i
		gri := new(big.Int).Exp(rp.G, responses[idx], rp.P)
		pkci := new(big.Int).Exp(ring[idx], challenges[idx], rp.P)
		Li := new(big.Int).Mul(gri, pkci)
		Li.Mod(Li, rp.P)

		// R_i = H(pk_i)^r_i × I^c_i
		hpi := rp.hashToPoint(ring[idx])
		hpri := new(big.Int).Exp(hpi, responses[idx], rp.P)
		Ici := new(big.Int).Exp(keyImage, challenges[idx], rp.P)
		Ri := new(big.Int).Mul(hpri, Ici)
		Ri.Mod(Ri, rp.P)

		// Next challenge
		challenges[nextIdx] = rp.hashRing(message, Li, Ri)
	}

	// Step 6: Close the ring - compute signer's response
	// r_s = alpha - c_s × sk mod q
	responses[signerIndex] = new(big.Int).Mul(challenges[signerIndex], signerKey.PrivateKey)
	responses[signerIndex].Sub(alpha, responses[signerIndex])
	responses[signerIndex].Mod(responses[signerIndex], rp.Q)

	return &RingSignature{
		KeyImage:  keyImage,
		Challenge: challenges[0],
		Responses: responses,
	}, nil
}

// Verify verifies a ring signature
func (rp *RingParams) Verify(message []byte, sig *RingSignature, ring []*big.Int) bool {
	n := len(ring)
	if len(sig.Responses) != n {
		return false
	}

	currentChallenge := sig.Challenge

	for i := 0; i < n; i++ {
		// L_i = g^r_i × pk_i^c_i
		gri := new(big.Int).Exp(rp.G, sig.Responses[i], rp.P)
		pkci := new(big.Int).Exp(ring[i], currentChallenge, rp.P)
		Li := new(big.Int).Mul(gri, pkci)
		Li.Mod(Li, rp.P)

		// R_i = H(pk_i)^r_i × I^c_i
		hpi := rp.hashToPoint(ring[i])
		hpri := new(big.Int).Exp(hpi, sig.Responses[i], rp.P)
		Ici := new(big.Int).Exp(sig.KeyImage, currentChallenge, rp.P)
		Ri := new(big.Int).Mul(hpri, Ici)
		Ri.Mod(Ri, rp.P)

		// Next challenge
		currentChallenge = rp.hashRing(message, Li, Ri)
	}

	// Ring should close: final challenge should equal initial
	return currentChallenge.Cmp(sig.Challenge) == 0
}

// Link checks if two signatures are from the same signer
func Link(sig1, sig2 *RingSignature) bool {
	return sig1.KeyImage.Cmp(sig2.KeyImage) == 0
}

// hashRing creates a hash for ring computation
func (rp *RingParams) hashRing(message []byte, L, R *big.Int) *big.Int {
	hashBytes := SHA3HashMultiple(message, L.Bytes(), R.Bytes())

	c := new(big.Int).SetBytes(hashBytes)
	c.Mod(c, rp.Q)
	return c
}
