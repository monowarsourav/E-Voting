package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// NonceSize is the size in bytes of the random nonce used in Fiat-Shamir challenges
const NonceSize = 32

// GenerateNonce generates a cryptographically secure random 32-byte nonce
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	return nonce, nil
}

// BinaryProof proves that a commitment contains 0 or 1
type BinaryProof struct {
	A0         *big.Int // First announcement
	A1         *big.Int // Second announcement
	D0         *big.Int // Challenge for w=0 case
	D1         *big.Int // Challenge for w=1 case
	F0         *big.Int // Response for w=0 case
	F1         *big.Int // Response for w=1 case
	Nonce      []byte   // Random nonce to prevent replay attacks
	ElectionID string   // Election context binding
}

// SumProof proves that sum of weights equals 1
type SumProof struct {
	ProductCommitment *big.Int // Product of all commitments
	Challenge         *big.Int
	Response          *big.Int
	Nonce             []byte // Random nonce to prevent replay attacks
	ElectionID        string // Election context binding
}

// ProveBinary creates a ZK proof that w in {0, 1}
func (pp *PedersenParams) ProveBinary(w, r *big.Int, C *big.Int, nonce []byte, electionID string) (*BinaryProof, error) {
	isZero := w.Cmp(big.NewInt(0)) == 0
	isOne := w.Cmp(big.NewInt(1)) == 0

	if !isZero && !isOne {
		return nil, errors.New("w must be 0 or 1")
	}

	if len(nonce) != NonceSize {
		return nil, errors.New("nonce must be 32 bytes")
	}

	if electionID == "" {
		return nil, errors.New("electionID must not be empty")
	}

	var a0, a1, d0, d1, f0, f1 *big.Int
	var err error

	if isZero {
		// Real proof for w=0, simulate w=1
		d1, err = rand.Int(rand.Reader, pp.Q)
		if err != nil {
			return nil, fmt.Errorf("zkp: entropy for d1: %w", err)
		}
		r1, err := rand.Int(rand.Reader, pp.Q)
		if err != nil {
			return nil, fmt.Errorf("zkp: entropy for r1: %w", err)
		}

		// a1 = g * h^r1 * (C/g)^(-d1)
		gInv := new(big.Int).ModInverse(pp.G, pp.P)
		CdivG := new(big.Int).Mul(C, gInv)
		CdivG.Mod(CdivG, pp.P)

		CdivGNegD1 := new(big.Int).Exp(CdivG, new(big.Int).Sub(pp.Q, d1), pp.P)
		g1 := new(big.Int).Set(pp.G)
		hr1 := new(big.Int).Exp(pp.H, r1, pp.P)

		a1 = new(big.Int).Mul(g1, hr1)
		a1.Mul(a1, CdivGNegD1)
		a1.Mod(a1, pp.P)

		f1 = r1

		// Real proof for w=0
		r0, err := rand.Int(rand.Reader, pp.Q)
		if err != nil {
			return nil, fmt.Errorf("zkp: entropy for r0: %w", err)
		}
		a0 = new(big.Int).Exp(pp.H, r0, pp.P) // g^0 * h^r0 = h^r0

		// Get challenge with nonce and election context
		c := hashChallenge(pp.Q, nonce, electionID, pp, C, a0, a1)

		// d0 = c - d1 mod q
		d0 = new(big.Int).Sub(c, d1)
		d0.Mod(d0, pp.Q)

		// f0 = r0 + d0 * r mod q
		f0 = new(big.Int).Mul(d0, r)
		f0.Add(f0, r0)
		f0.Mod(f0, pp.Q)

	} else {
		// Real proof for w=1, simulate w=0
		d0, err = rand.Int(rand.Reader, pp.Q)
		if err != nil {
			return nil, fmt.Errorf("zkp: entropy for d0: %w", err)
		}
		r0, err := rand.Int(rand.Reader, pp.Q)
		if err != nil {
			return nil, fmt.Errorf("zkp: entropy for r0: %w", err)
		}

		// a0 = h^r0 * C^(-d0)
		hr0 := new(big.Int).Exp(pp.H, r0, pp.P)
		CNegD0 := new(big.Int).Exp(C, new(big.Int).Sub(pp.Q, d0), pp.P)
		a0 = new(big.Int).Mul(hr0, CNegD0)
		a0.Mod(a0, pp.P)

		f0 = r0

		// Real proof for w=1
		r1, err := rand.Int(rand.Reader, pp.Q)
		if err != nil {
			return nil, fmt.Errorf("zkp: entropy for r1: %w", err)
		}

		// a1 = g * h^r1 for w=1
		g1 := new(big.Int).Set(pp.G)
		hr1 := new(big.Int).Exp(pp.H, r1, pp.P)
		a1 = new(big.Int).Mul(g1, hr1)
		a1.Mod(a1, pp.P)

		// Get challenge with nonce and election context
		c := hashChallenge(pp.Q, nonce, electionID, pp, C, a0, a1)

		// d1 = c - d0 mod q
		d1 = new(big.Int).Sub(c, d0)
		d1.Mod(d1, pp.Q)

		// f1 = r1 + d1 * r mod q
		f1 = new(big.Int).Mul(d1, r)
		f1.Add(f1, r1)
		f1.Mod(f1, pp.Q)
	}

	return &BinaryProof{
		A0:         a0,
		A1:         a1,
		D0:         d0,
		D1:         d1,
		F0:         f0,
		F1:         f1,
		Nonce:      nonce,
		ElectionID: electionID,
	}, nil
}

// VerifyBinary verifies a binary ZK proof
func (pp *PedersenParams) VerifyBinary(C *big.Int, proof *BinaryProof) bool {
	if len(proof.Nonce) != NonceSize {
		return false
	}
	if proof.ElectionID == "" {
		return false
	}

	// Recompute challenge with nonce and election context
	c := hashChallenge(pp.Q, proof.Nonce, proof.ElectionID, pp, C, proof.A0, proof.A1)

	// Check d0 + d1 = c mod q
	dSum := new(big.Int).Add(proof.D0, proof.D1)
	dSum.Mod(dSum, pp.Q)
	if dSum.Cmp(c) != 0 {
		return false
	}

	// Check a0: h^f0 * C^(-d0) should equal a0
	hf0 := new(big.Int).Exp(pp.H, proof.F0, pp.P)
	CNegD0 := new(big.Int).Exp(C, new(big.Int).Sub(pp.Q, proof.D0), pp.P)
	check0 := new(big.Int).Mul(hf0, CNegD0)
	check0.Mod(check0, pp.P)
	if check0.Cmp(proof.A0) != 0 {
		return false
	}

	// Check a1: g * h^f1 * (C/g)^(-d1) should equal a1
	gInv := new(big.Int).ModInverse(pp.G, pp.P)
	CdivG := new(big.Int).Mul(C, gInv)
	CdivG.Mod(CdivG, pp.P)

	g1 := new(big.Int).Set(pp.G)
	hf1 := new(big.Int).Exp(pp.H, proof.F1, pp.P)
	CdivGNegD1 := new(big.Int).Exp(CdivG, new(big.Int).Sub(pp.Q, proof.D1), pp.P)

	check1 := new(big.Int).Mul(g1, hf1)
	check1.Mul(check1, CdivGNegD1)
	check1.Mod(check1, pp.P)
	return check1.Cmp(proof.A1) == 0
}

// ProveSumOne proves that commitments sum to 1
// Given C1, C2, ..., Ck where each Ci = g^wi * h^ri
// Prove: sum(wi) = 1
func (pp *PedersenParams) ProveSumOne(commitments []*Commitment, nonce []byte, electionID string) (*SumProof, error) {
	if len(nonce) != NonceSize {
		return nil, errors.New("nonce must be 32 bytes")
	}

	if electionID == "" {
		return nil, errors.New("electionID must not be empty")
	}

	// Product of commitments = g^(sum(wi)) * h^(sum(ri))
	// If sum(wi) = 1, then product = g * h^(sum(ri))

	// Compute product
	product := big.NewInt(1)
	totalR := big.NewInt(0)

	for _, c := range commitments {
		product.Mul(product, c.C)
		product.Mod(product, pp.P)
		totalR.Add(totalR, c.R)
		totalR.Mod(totalR, pp.Q)
	}

	// Now prove product = g * h^totalR
	// This is a standard Schnorr proof

	// Random commitment
	k, _ := rand.Int(rand.Reader, pp.Q)
	a := new(big.Int).Exp(pp.H, k, pp.P) // h^k

	// Challenge with nonce and election context
	c := hashChallenge(pp.Q, nonce, electionID, pp, product, a, pp.G)

	// Response: s = k + c * totalR mod q
	s := new(big.Int).Mul(c, totalR)
	s.Add(s, k)
	s.Mod(s, pp.Q)

	return &SumProof{
		ProductCommitment: product,
		Challenge:         c,
		Response:          s,
		Nonce:             nonce,
		ElectionID:        electionID,
	}, nil
}

// VerifySumOne verifies that sum of weights equals 1
func (pp *PedersenParams) VerifySumOne(commitments []*big.Int, proof *SumProof) bool {
	if len(proof.Nonce) != NonceSize {
		return false
	}
	if proof.ElectionID == "" {
		return false
	}

	// Recompute product
	product := big.NewInt(1)
	for _, c := range commitments {
		product.Mul(product, c)
		product.Mod(product, pp.P)
	}

	// Check product matches
	if product.Cmp(proof.ProductCommitment) != 0 {
		return false
	}

	// Verify: h^s = a * (product/g)^c
	hs := new(big.Int).Exp(pp.H, proof.Response, pp.P)

	gInv := new(big.Int).ModInverse(pp.G, pp.P)
	productDivG := new(big.Int).Mul(product, gInv)
	productDivG.Mod(productDivG, pp.P)

	// Recompute a from proof values
	negC := new(big.Int).Sub(pp.Q, proof.Challenge)
	productDivGNegC := new(big.Int).Exp(productDivG, negC, pp.P)
	computedA := new(big.Int).Mul(hs, productDivGNegC)
	computedA.Mod(computedA, pp.P)

	// Hash and verify with nonce and election context
	expectedC := hashChallenge(pp.Q, proof.Nonce, proof.ElectionID, pp, product, computedA, pp.G)

	return expectedC.Cmp(proof.Challenge) == 0
}

// hashChallenge creates a Strong Fiat-Shamir challenge per Bernhard-Pereira-Warinschi
// (ASIACRYPT 2012). It includes the public parameters (the "statement") in the hash
// to prevent attacks demonstrated against Helios voting system.
// The nonce prevents replay attacks across sessions, and the electionID binds the
// proof to a specific election, preventing cross-election proof reuse.
func hashChallenge(q *big.Int, nonce []byte, electionID string, pp *PedersenParams, values ...*big.Int) *big.Int {
	hasher := sha3.New256()

	// Domain separation tag (prevents cross-protocol attacks)
	hasher.Write([]byte("CovertVote-ZKP-v1"))

	// Include public parameters (the "statement" in Strong Fiat-Shamir)
	hasher.Write(pp.P.Bytes())
	hasher.Write(pp.Q.Bytes())
	hasher.Write(pp.G.Bytes())
	hasher.Write(pp.H.Bytes())

	// Context binding
	hasher.Write(nonce)
	hasher.Write([]byte(electionID))

	// Length-prefix each value to prevent concatenation ambiguity
	for _, v := range values {
		vBytes := v.Bytes()
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(vBytes)))
		hasher.Write(lenBuf)
		hasher.Write(vBytes)
	}

	hashBytes := hasher.Sum(nil)
	c := new(big.Int).SetBytes(hashBytes)
	c.Mod(c, q)
	return c
}
