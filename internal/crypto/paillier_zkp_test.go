package crypto

import (
	"crypto/rand"
	"math/big"
	"testing"
)

// generatePaillierEncryption produces a Paillier ciphertext of v together with
// the randomness used (so the prover has the witness).
func generatePaillierEncryption(t *testing.T, pk *PaillierPublicKey, v *big.Int) (*big.Int, *big.Int) {
	t.Helper()
	r, err := rand.Int(rand.Reader, pk.N)
	if err != nil {
		t.Fatalf("rand: %v", err)
	}
	E, err := pk.EncryptWithRandomness(v, r)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return E, r
}

func TestPaillierBinaryProof_Zero(t *testing.T) {
	sk, err := GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	pk := sk.PublicKey
	E, r := generatePaillierEncryption(t, pk, big.NewInt(0))

	nonce, _ := GenerateNonce()
	proof, err := ProvePaillierBinary(pk, big.NewInt(0), r, E, 3, nonce, "test-election")
	if err != nil {
		t.Fatalf("prove: %v", err)
	}
	if !VerifyPaillierBinary(pk, E, proof) {
		t.Fatal("honest v=0 proof rejected")
	}
}

func TestPaillierBinaryProof_One(t *testing.T) {
	sk, err := GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	pk := sk.PublicKey
	E, r := generatePaillierEncryption(t, pk, big.NewInt(1))

	nonce, _ := GenerateNonce()
	proof, err := ProvePaillierBinary(pk, big.NewInt(1), r, E, 0, nonce, "test-election")
	if err != nil {
		t.Fatalf("prove: %v", err)
	}
	if !VerifyPaillierBinary(pk, E, proof) {
		t.Fatal("honest v=1 proof rejected")
	}
}

func TestPaillierBinaryProof_RejectInvalidVotes(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey

	// Try to prove v=2 (out of {0,1}) — generator should reject.
	r, _ := rand.Int(rand.Reader, pk.N)
	E, _ := pk.EncryptWithRandomness(big.NewInt(2), r)
	nonce, _ := GenerateNonce()
	if _, err := ProvePaillierBinary(pk, big.NewInt(2), r, E, 0, nonce, "test"); err == nil {
		t.Fatal("expected error for v=2 generation")
	}
}

func TestPaillierBinaryProof_TamperedRejected(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	E, r := generatePaillierEncryption(t, pk, big.NewInt(1))
	nonce, _ := GenerateNonce()
	proof, _ := ProvePaillierBinary(pk, big.NewInt(1), r, E, 5, nonce, "test")

	// Verify against a different ciphertext.
	Eother, _ := generatePaillierEncryption(t, pk, big.NewInt(1))
	if VerifyPaillierBinary(pk, Eother, proof) {
		t.Fatal("proof verified against a different ciphertext")
	}

	// Replay against a different candidate index.
	proof2 := *proof
	proof2.CandidateIdx = 6
	if VerifyPaillierBinary(pk, E, &proof2) {
		t.Fatal("proof verified after candidate-index replay")
	}

	// Replay against a different election.
	proof3 := *proof
	proof3.ElectionID = "other-election"
	if VerifyPaillierBinary(pk, E, &proof3) {
		t.Fatal("proof verified after election-id replay")
	}

	// Mutate response.
	proof4 := *proof
	proof4.S0 = new(big.Int).Add(proof.S0, big.NewInt(1))
	if VerifyPaillierBinary(pk, E, &proof4) {
		t.Fatal("proof verified after S0 tamper")
	}
}

func TestPaillierSumProof_Valid(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	const m = 4
	const real = 2 // candidate index of the "1" position

	rs := make([]*big.Int, m)
	es := make([]*big.Int, m)
	for j := 0; j < m; j++ {
		v := big.NewInt(0)
		if j == real {
			v = big.NewInt(1)
		}
		es[j], rs[j] = generatePaillierEncryption(t, pk, v)
	}

	nonce, _ := GenerateNonce()
	proof, err := ProvePaillierSumToOne(pk, rs, es, nonce, "test")
	if err != nil {
		t.Fatalf("prove sum: %v", err)
	}
	if !VerifyPaillierSumToOne(pk, es, proof) {
		t.Fatal("honest sum proof rejected")
	}
}

func TestPaillierSumProof_RejectsSumZero(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	const m = 3

	rs := make([]*big.Int, m)
	es := make([]*big.Int, m)
	for j := 0; j < m; j++ {
		es[j], rs[j] = generatePaillierEncryption(t, pk, big.NewInt(0))
	}
	nonce, _ := GenerateNonce()
	proof, err := ProvePaillierSumToOne(pk, rs, es, nonce, "test")
	if err != nil {
		t.Fatalf("prove: %v", err)
	}
	// The prover happily generates a proof, but verification must fail
	// because sum=0, not 1, so Pi/(1+n) is not an n-th residue.
	if VerifyPaillierSumToOne(pk, es, proof) {
		t.Fatal("sum proof verified despite sum=0")
	}
}

func TestPaillierSumProof_RejectsSumTwo(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	const m = 3

	rs := make([]*big.Int, m)
	es := make([]*big.Int, m)
	// Two candidates set to 1 — sum = 2.
	for j := 0; j < m; j++ {
		v := big.NewInt(0)
		if j == 0 || j == 1 {
			v = big.NewInt(1)
		}
		es[j], rs[j] = generatePaillierEncryption(t, pk, v)
	}
	nonce, _ := GenerateNonce()
	proof, _ := ProvePaillierSumToOne(pk, rs, es, nonce, "test")
	if VerifyPaillierSumToOne(pk, es, proof) {
		t.Fatal("sum proof verified despite sum=2")
	}
}

func TestPaillierSumProof_TamperedCiphertextRejected(t *testing.T) {
	sk, _ := GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	const m = 3
	rs := make([]*big.Int, m)
	es := make([]*big.Int, m)
	for j := 0; j < m; j++ {
		v := big.NewInt(0)
		if j == 0 {
			v = big.NewInt(1)
		}
		es[j], rs[j] = generatePaillierEncryption(t, pk, v)
	}
	nonce, _ := GenerateNonce()
	proof, _ := ProvePaillierSumToOne(pk, rs, es, nonce, "test")

	// Swap one ciphertext for a freshly encrypted 0 — Pi changes, so verification
	// must fail at the challenge recomputation step.
	esTampered := append([]*big.Int(nil), es...)
	esTampered[1], _ = generatePaillierEncryption(t, pk, big.NewInt(0))
	if VerifyPaillierSumToOne(pk, esTampered, proof) {
		t.Fatal("sum proof verified despite ciphertext substitution")
	}
}
