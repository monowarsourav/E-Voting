package crypto

import (
	"math/big"
	"sync"
	"testing"
)

// sharedThresholdShares is generated once per test binary execution
// and reused across all tests. GenerateThresholdKey(2048, 5, 3) takes
// 2-3 minutes; sharing the result brings test suite runtime from
// ~15 minutes to ~4 minutes while preserving production-grade 2048-bit
// keys across all correctness tests.
var (
	sharedThresholdShares     *ThresholdKeyShares
	sharedThresholdSharesOnce sync.Once
	sharedThresholdSharesErr  error
)

// getSharedThresholdShares returns production-grade 2048-bit threshold
// shares with n=5 trustees and threshold t=3. Generated exactly once
// per test binary execution via sync.Once.
func getSharedThresholdShares(t *testing.T) *ThresholdKeyShares {
	t.Helper()
	sharedThresholdSharesOnce.Do(func() {
		sharedThresholdShares, sharedThresholdSharesErr = GenerateThresholdKey(2048, 5, 3)
	})
	if sharedThresholdSharesErr != nil {
		t.Fatal(sharedThresholdSharesErr)
	}
	return sharedThresholdShares
}

func TestThresholdKeyGeneration(t *testing.T) {
	shares := getSharedThresholdShares(t)

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

	for i, s := range shares.Shares {
		if s.Index != i+1 {
			t.Errorf("share %d has index %d, expected %d", i, s.Index, i+1)
		}
	}
}

func TestThresholdEncryptDecrypt(t *testing.T) {
	shares := getSharedThresholdShares(t)
	pk := shares.PublicKey

	vote := big.NewInt(42)
	ciphertext, err := pk.Encrypt(vote)
	if err != nil {
		t.Fatal(err)
	}

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
	shares := getSharedThresholdShares(t)
	pk := shares.PublicKey

	votes := []int64{1, 0, 1, 1, 0, 1, 0, 1, 1, 0}
	var ciphertexts []*big.Int
	for _, v := range votes {
		enc, err := pk.Encrypt(big.NewInt(v))
		if err != nil {
			t.Fatal(err)
		}
		ciphertexts = append(ciphertexts, enc)
	}

	tally := pk.AddMultiple(ciphertexts)

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
	shares := getSharedThresholdShares(t)
	pk := shares.PublicKey

	ciphertext, err := pk.Encrypt(big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}

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
	shares := getSharedThresholdShares(t)
	pk := shares.PublicKey

	ciphertext, err := pk.Encrypt(big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}

	pd, err := shares.Shares[0].PartialDecrypt(
		ciphertext, pk, shares.Params, shares.VerifyKeys[0], shares.V)
	if err != nil {
		t.Fatal(err)
	}

	valid := VerifyPartialDecryption(pd, ciphertext, pk, shares.VerifyKeys[0], shares.V)
	if !valid {
		t.Fatal("valid partial decryption should verify")
	}

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
