package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// PaillierBinaryProof is a non-interactive CDS OR-proof that a Paillier
// ciphertext E encrypts a value v in {0,1}, without revealing v. The proof
// operates directly on the n-th-residuosity language of Paillier (Damgård-Jurik
// 2001) and uses Strong Fiat-Shamir per Bernhard-Pereira-Warinschi 2012.
//
// Statement: E = (1+n)^v * r^n mod n^2 with v in {0,1}.
// Witness:   r in Z_n^* (the encryption randomness for E).
type PaillierBinaryProof struct {
	A0           *big.Int // simulator commitment for branch v=0
	A1           *big.Int // simulator commitment for branch v=1
	D0           *big.Int // challenge share for branch v=0
	D1           *big.Int // challenge share for branch v=1
	S0           *big.Int // response for branch v=0
	S1           *big.Int // response for branch v=1
	Nonce        []byte   // 32-byte freshness nonce
	ElectionID   string   // election context binding
	CandidateIdx int      // candidate position (statement binding)
}

// PaillierSumProof is a non-interactive Schnorr-style proof that a vector of
// Paillier ciphertexts (E_1, ..., E_m) jointly encrypts values summing to 1.
// Statement: Pi = prod_j E_j satisfies Pi/(1+n) = R^n mod n^2 for some R.
// Witness:   R = prod_j r_j mod n.
type PaillierSumProof struct {
	A          *big.Int // u^n mod n^2 commitment
	C          *big.Int // Fiat-Shamir challenge
	S          *big.Int // u * R^c mod n response
	Nonce      []byte
	ElectionID string
}

// ProvePaillierBinary constructs a binary OR-proof that E encrypts v in {0,1}.
// r is the Paillier randomness used to produce E (i.e. E = (1+n)^v * r^n mod n^2).
// candidateIdx binds the proof to a specific ballot position so a proof for
// candidate j cannot be replayed against candidate j'.
func ProvePaillierBinary(
	pk *PaillierPublicKey,
	v, r *big.Int,
	E *big.Int,
	candidateIdx int,
	nonce []byte,
	electionID string,
) (*PaillierBinaryProof, error) {
	if pk == nil || pk.N == nil || pk.N2 == nil {
		return nil, errors.New("paillier zkp: nil public key")
	}
	if v == nil || r == nil || E == nil {
		return nil, errors.New("paillier zkp: nil witness or statement")
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("paillier zkp: nonce must be %d bytes, got %d", NonceSize, len(nonce))
	}
	if electionID == "" {
		return nil, errors.New("paillier zkp: electionID must not be empty")
	}
	isZero := v.Sign() == 0
	isOne := v.Cmp(big.NewInt(1)) == 0
	if !isZero && !isOne {
		return nil, errors.New("paillier zkp: v must be 0 or 1")
	}

	n := pk.N
	n2 := pk.N2
	// (1+n)^{-1} mod n^2 = 1 - n mod n^2 = n^2 - n + 1 (since (1+n)(1-n)=1-n^2 ≡ 1 mod n^2).
	gPlusInv := new(big.Int).Sub(n2, n)
	gPlusInv.Add(gPlusInv, big.NewInt(1))
	gPlusInv.Mod(gPlusInv, n2)

	var a0, a1, d0, d1, s0, s1 *big.Int
	var err error

	if isZero {
		// Simulate v=1 branch: random d1 in Z_n, random s1 in Z_n^*,
		// compute a1 = s1^n * (E * (1+n)^{-1})^{-d1} mod n^2.
		d1, err = rand.Int(rand.Reader, n)
		if err != nil {
			return nil, fmt.Errorf("paillier zkp: entropy for d1: %w", err)
		}
		s1, err = rand.Int(rand.Reader, n)
		if err != nil {
			return nil, fmt.Errorf("paillier zkp: entropy for s1: %w", err)
		}
		// e_over_g = E * (1+n)^{-1} mod n^2 (statement for v=1 branch).
		eOverG := new(big.Int).Mul(E, gPlusInv)
		eOverG.Mod(eOverG, n2)
		// a1 = s1^n * eOverG^{-d1}.
		// In Z_{n^2}^* the order of an n-th residue divides phi(n), not n, so we
		// cannot replace -d1 by n-d1 mod n; instead we invert the base and
		// raise to d1.
		s1n := new(big.Int).Exp(s1, n, n2)
		eOverGInv := new(big.Int).ModInverse(eOverG, n2)
		if eOverGInv == nil {
			return nil, errors.New("paillier zkp: E/(1+n) not invertible mod n^2")
		}
		eOverGNegD1 := new(big.Int).Exp(eOverGInv, d1, n2)
		a1 = new(big.Int).Mul(s1n, eOverGNegD1)
		a1.Mod(a1, n2)

		// Real proof for v=0: pick u0 in Z_n^*, a0 = u0^n mod n^2.
		u0, err := rand.Int(rand.Reader, n)
		if err != nil {
			return nil, fmt.Errorf("paillier zkp: entropy for u0: %w", err)
		}
		a0 = new(big.Int).Exp(u0, n, n2)

		// Fiat-Shamir challenge c.
		c := hashPaillierBinary(n, nonce, electionID, candidateIdx, E, a0, a1)
		// d0 = c - d1 mod n
		d0 = new(big.Int).Sub(c, d1)
		d0.Mod(d0, n)
		// s0 = u0 * r^{d0} mod n
		rD0 := new(big.Int).Exp(r, d0, n)
		s0 = new(big.Int).Mul(u0, rD0)
		s0.Mod(s0, n)
	} else {
		// Simulate v=0 branch: random d0 in Z_n, random s0 in Z_n^*,
		// compute a0 = s0^n * E^{-d0} mod n^2.
		d0, err = rand.Int(rand.Reader, n)
		if err != nil {
			return nil, fmt.Errorf("paillier zkp: entropy for d0: %w", err)
		}
		s0, err = rand.Int(rand.Reader, n)
		if err != nil {
			return nil, fmt.Errorf("paillier zkp: entropy for s0: %w", err)
		}
		s0n := new(big.Int).Exp(s0, n, n2)
		eInv := new(big.Int).ModInverse(E, n2)
		if eInv == nil {
			return nil, errors.New("paillier zkp: E not invertible mod n^2")
		}
		eNegD0 := new(big.Int).Exp(eInv, d0, n2)
		a0 = new(big.Int).Mul(s0n, eNegD0)
		a0.Mod(a0, n2)

		// Real proof for v=1: pick u1 in Z_n^*, a1 = u1^n mod n^2.
		u1, err := rand.Int(rand.Reader, n)
		if err != nil {
			return nil, fmt.Errorf("paillier zkp: entropy for u1: %w", err)
		}
		a1 = new(big.Int).Exp(u1, n, n2)

		c := hashPaillierBinary(n, nonce, electionID, candidateIdx, E, a0, a1)
		d1 = new(big.Int).Sub(c, d0)
		d1.Mod(d1, n)
		rD1 := new(big.Int).Exp(r, d1, n)
		s1 = new(big.Int).Mul(u1, rD1)
		s1.Mod(s1, n)
	}

	return &PaillierBinaryProof{
		A0:           a0,
		A1:           a1,
		D0:           d0,
		D1:           d1,
		S0:           s0,
		S1:           s1,
		Nonce:        nonce,
		ElectionID:   electionID,
		CandidateIdx: candidateIdx,
	}, nil
}

// VerifyPaillierBinary checks that proof attests to E encrypting some v in {0,1}.
// Returns true iff all three verification equations hold.
func VerifyPaillierBinary(pk *PaillierPublicKey, E *big.Int, proof *PaillierBinaryProof) bool {
	if pk == nil || pk.N == nil || pk.N2 == nil || E == nil || proof == nil {
		return false
	}
	if len(proof.Nonce) != NonceSize || proof.ElectionID == "" {
		return false
	}
	if proof.A0 == nil || proof.A1 == nil || proof.D0 == nil || proof.D1 == nil ||
		proof.S0 == nil || proof.S1 == nil {
		return false
	}

	n := pk.N
	n2 := pk.N2
	gPlusInv := new(big.Int).Sub(n2, n)
	gPlusInv.Add(gPlusInv, big.NewInt(1))
	gPlusInv.Mod(gPlusInv, n2)

	// Recompute challenge.
	c := hashPaillierBinary(n, proof.Nonce, proof.ElectionID, proof.CandidateIdx, E, proof.A0, proof.A1)

	// Check d0 + d1 ≡ c (mod n).
	dSum := new(big.Int).Add(proof.D0, proof.D1)
	dSum.Mod(dSum, n)
	if dSum.Cmp(c) != 0 {
		return false
	}

	// Check s0^n ≡ a0 * E^{d0} (mod n^2).
	s0n := new(big.Int).Exp(proof.S0, n, n2)
	eD0 := new(big.Int).Exp(E, proof.D0, n2)
	rhs0 := new(big.Int).Mul(proof.A0, eD0)
	rhs0.Mod(rhs0, n2)
	if s0n.Cmp(rhs0) != 0 {
		return false
	}

	// Check s1^n ≡ a1 * (E/(1+n))^{d1} (mod n^2).
	eOverG := new(big.Int).Mul(E, gPlusInv)
	eOverG.Mod(eOverG, n2)
	s1n := new(big.Int).Exp(proof.S1, n, n2)
	eOverGD1 := new(big.Int).Exp(eOverG, proof.D1, n2)
	rhs1 := new(big.Int).Mul(proof.A1, eOverGD1)
	rhs1.Mod(rhs1, n2)
	if s1n.Cmp(rhs1) != 0 {
		return false
	}
	return true
}

// ProvePaillierSumToOne proves that the product of ciphertexts encrypts 1.
// randomness must contain the Paillier encryption randomness r_j for each E_j
// (i.e. E_j = (1+n)^{v_j} * r_j^n mod n^2 with sum_j v_j = 1).
func ProvePaillierSumToOne(
	pk *PaillierPublicKey,
	randomness []*big.Int,
	ciphertexts []*big.Int,
	nonce []byte,
	electionID string,
) (*PaillierSumProof, error) {
	if pk == nil || pk.N == nil || pk.N2 == nil {
		return nil, errors.New("paillier zkp: nil public key")
	}
	if len(randomness) == 0 || len(randomness) != len(ciphertexts) {
		return nil, errors.New("paillier zkp: randomness and ciphertext vectors must be non-empty and equal-length")
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("paillier zkp: nonce must be %d bytes, got %d", NonceSize, len(nonce))
	}
	if electionID == "" {
		return nil, errors.New("paillier zkp: electionID must not be empty")
	}

	n := pk.N
	n2 := pk.N2

	// R = prod_j r_j mod n.
	R := big.NewInt(1)
	for _, rj := range randomness {
		if rj == nil {
			return nil, errors.New("paillier zkp: nil randomness entry")
		}
		R.Mul(R, rj)
		R.Mod(R, n)
	}

	// u in Z_n^*, a = u^n mod n^2.
	u, err := rand.Int(rand.Reader, n)
	if err != nil {
		return nil, fmt.Errorf("paillier zkp: entropy for u: %w", err)
	}
	a := new(big.Int).Exp(u, n, n2)

	// Challenge c = H(n || nonce || electionID || E_1 || ... || E_m || a) mod n.
	c := hashPaillierSum(n, nonce, electionID, ciphertexts, a)

	// s = u * R^c mod n.
	rC := new(big.Int).Exp(R, c, n)
	s := new(big.Int).Mul(u, rC)
	s.Mod(s, n)

	return &PaillierSumProof{
		A:          a,
		C:          c,
		S:          s,
		Nonce:      nonce,
		ElectionID: electionID,
	}, nil
}

// VerifyPaillierSumToOne verifies a sum-to-one proof against a ciphertext vector.
func VerifyPaillierSumToOne(pk *PaillierPublicKey, ciphertexts []*big.Int, proof *PaillierSumProof) bool {
	if pk == nil || pk.N == nil || pk.N2 == nil || proof == nil {
		return false
	}
	if len(ciphertexts) == 0 {
		return false
	}
	if len(proof.Nonce) != NonceSize || proof.ElectionID == "" {
		return false
	}
	if proof.A == nil || proof.C == nil || proof.S == nil {
		return false
	}

	n := pk.N
	n2 := pk.N2

	// Recompute Pi and challenge.
	pi := big.NewInt(1)
	for _, e := range ciphertexts {
		if e == nil {
			return false
		}
		pi.Mul(pi, e)
		pi.Mod(pi, n2)
	}
	expectedC := hashPaillierSum(n, proof.Nonce, proof.ElectionID, ciphertexts, proof.A)
	if expectedC.Cmp(proof.C) != 0 {
		return false
	}

	// Compute Pi/(1+n) mod n^2.
	gPlusInv := new(big.Int).Sub(n2, n)
	gPlusInv.Add(gPlusInv, big.NewInt(1))
	gPlusInv.Mod(gPlusInv, n2)
	piOverG := new(big.Int).Mul(pi, gPlusInv)
	piOverG.Mod(piOverG, n2)

	// Check s^n ≡ a * (Pi/(1+n))^c (mod n^2).
	sn := new(big.Int).Exp(proof.S, n, n2)
	piOverGC := new(big.Int).Exp(piOverG, proof.C, n2)
	rhs := new(big.Int).Mul(proof.A, piOverGC)
	rhs.Mod(rhs, n2)
	return sn.Cmp(rhs) == 0
}

// hashPaillierBinary computes the Strong Fiat-Shamir challenge for the binary
// OR-proof. Inputs are length-prefixed and a domain-separation tag is included.
func hashPaillierBinary(n *big.Int, nonce []byte, electionID string, candidateIdx int, E, a0, a1 *big.Int) *big.Int {
	hasher := sha3.New256()
	hasher.Write([]byte("CovertVote-PaillierZKP-Binary-v1"))
	writeBig(hasher, n)
	hasher.Write(nonce)
	hasher.Write([]byte(electionID))
	idxBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(idxBuf, uint64(candidateIdx))
	hasher.Write(idxBuf)
	writeBig(hasher, E)
	writeBig(hasher, a0)
	writeBig(hasher, a1)
	h := new(big.Int).SetBytes(hasher.Sum(nil))
	h.Mod(h, n)
	return h
}

// hashPaillierSum computes the Strong Fiat-Shamir challenge for the sum-to-one proof.
func hashPaillierSum(n *big.Int, nonce []byte, electionID string, ciphertexts []*big.Int, a *big.Int) *big.Int {
	hasher := sha3.New256()
	hasher.Write([]byte("CovertVote-PaillierZKP-Sum-v1"))
	writeBig(hasher, n)
	hasher.Write(nonce)
	hasher.Write([]byte(electionID))
	mBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(mBuf, uint64(len(ciphertexts)))
	hasher.Write(mBuf)
	for _, e := range ciphertexts {
		writeBig(hasher, e)
	}
	writeBig(hasher, a)
	h := new(big.Int).SetBytes(hasher.Sum(nil))
	h.Mod(h, n)
	return h
}

// writeBig length-prefixes a big.Int into the hasher to prevent concatenation
// ambiguity attacks in Fiat-Shamir.
func writeBig(hasher interface{ Write(p []byte) (n int, err error) }, v *big.Int) {
	b := v.Bytes()
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(b)))
	hasher.Write(lenBuf)
	hasher.Write(b)
}
