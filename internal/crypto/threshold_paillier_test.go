package crypto

import (
	"math/big"
	"testing"
)

func TestThresholdKeyGeneration(t *testing.T) {
	shares, err := GenerateThresholdKey(2048, 5, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(shares.Shares) != 5 {
		t.Fatalf("expected 5 shares, got %d", len(shares.Shares))
	}

	if len(shares.VerifyKeys) != 5 {
		t.Fatalf("expected 5 verify keys, got %d", len(shares.VerifyKeys))
	}

	if shares.PublicKey == nil {
		t.Fatal("public key is nil")
	}

	if shares.Params.N != 5 || shares.Params.Threshold != 3 {
		t.Fatalf("params mismatch: got n=%d t=%d", shares.Params.N, shares.Params.Threshold)
	}

	// Verify indices are 1-based
	for i, s := range shares.Shares {
		if s.Index != i+1 {
			t.Errorf("share %d has index %d, expected %d", i, s.Index, i+1)
		}
	}
}

func TestThresholdEncryptDecrypt(t *testing.T) {
	shares, err := GenerateThresholdKey(2048, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	pk := shares.PublicKey

	vote := big.NewInt(42)
	ciphertext, err := pk.Encrypt(vote)
	if err != nil {
		t.Fatal(err)
	}

	// Get partial decryptions from trustees 1, 3, 5 (any 3 of 5)
	partials := make([]*ThresholdPartialDecryption, 3)
	indices := []int{0, 2, 4}
	for i, idx := range indices {
		pd, err := shares.Shares[idx].PartialDecrypt(
			ciphertext, pk, shares.Params, shares.VerifyKeys[idx], shares.V)
		if err != nil {
			t.Fatal(err)
		}
		partials[i] = pd
	}

	result, err := CombinePartialDecryptions(partials, pk, shares.Params)
	if err != nil {
		t.Fatal(err)
	}

	if result.Cmp(vote) != 0 {
		t.Errorf("expected %d, got %d", vote.Int64(), result.Int64())
	}
}

func TestThresholdHomomorphicTally(t *testing.T) {
	shares, err := GenerateThresholdKey(2048, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	pk := shares.PublicKey

	// Encrypt 10 votes
	votes := []int64{1, 0, 1, 1, 0, 1, 0, 1, 1, 0} // expected sum = 6
	var ciphertexts []*big.Int
	for _, v := range votes {
		enc, err := pk.Encrypt(big.NewInt(v))
		if err != nil {
			t.Fatal(err)
		}
		ciphertexts = append(ciphertexts, enc)
	}

	// Homomorphic tally
	tally := pk.AddMultiple(ciphertexts)

	// Threshold decrypt with trustees 2, 3, 4
	partials := make([]*ThresholdPartialDecryption, 3)
	for i := 0; i < 3; i++ {
		pd, err := shares.Shares[i+1].PartialDecrypt(
			tally, pk, shares.Params, shares.VerifyKeys[i+1], shares.V)
		if err != nil {
			t.Fatal(err)
		}
		partials[i] = pd
	}

	result, err := CombinePartialDecryptions(partials, pk, shares.Params)
	if err != nil {
		t.Fatal(err)
	}

	if result.Int64() != 6 {
		t.Errorf("expected tally 6, got %d", result.Int64())
	}
}

func TestThresholdInsufficientShares(t *testing.T) {
	shares, err := GenerateThresholdKey(2048, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	pk := shares.PublicKey

	ciphertext, err := pk.Encrypt(big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}

	// Only 2 partial decryptions (need 3)
	partials := make([]*ThresholdPartialDecryption, 2)
	for i := 0; i < 2; i++ {
		pd, err := shares.Shares[i].PartialDecrypt(
			ciphertext, pk, shares.Params, shares.VerifyKeys[i], shares.V)
		if err != nil {
			t.Fatal(err)
		}
		partials[i] = pd
	}

	_, err = CombinePartialDecryptions(partials, pk, shares.Params)
	if err == nil {
		t.Fatal("expected error with insufficient shares")
	}
}

func TestThresholdPartialDecryptionVerification(t *testing.T) {
	shares, err := GenerateThresholdKey(2048, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	pk := shares.PublicKey

	ciphertext, err := pk.Encrypt(big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}

	// Get a valid partial decryption
	pd, err := shares.Shares[0].PartialDecrypt(
		ciphertext, pk, shares.Params, shares.VerifyKeys[0], shares.V)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the ZK proof
	valid := VerifyPartialDecryption(pd, ciphertext, pk, shares.VerifyKeys[0], shares.V)
	if !valid {
		t.Fatal("valid partial decryption should verify")
	}

	// Tamper with the partial decryption
	tamperedCi := new(big.Int).Add(pd.Ci, big.NewInt(1))
	tampered := &ThresholdPartialDecryption{
		Index: pd.Index,
		Ci:    tamperedCi,
		Proof: pd.Proof,
	}
	invalid := VerifyPartialDecryption(tampered, ciphertext, pk, shares.VerifyKeys[0], shares.V)
	if invalid {
		t.Fatal("tampered partial decryption should NOT verify")
	}
}
