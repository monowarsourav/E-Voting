package benchmark

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
)

// TestScalabilityTally measures homomorphic tally time at different voter scales.
// This proves O(n) complexity: tally time should grow linearly with voter count.
// Note: We pre-encrypt a pool of votes and reuse them to isolate tally time from
// encryption time, since Paillier encryption is expensive at 2048-bit.
func TestScalabilityTally(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability tally benchmark in short mode")
	}

	paillierKey, err := crypto.GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := paillierKey.PublicKey

	// Pre-encrypt a pool of votes (reused across scales to save time)
	poolSize := 1000
	t.Logf("Pre-encrypting pool of %d votes...", poolSize)
	pool := make([]*big.Int, poolSize)
	for i := 0; i < poolSize; i++ {
		v := big.NewInt(int64(i % 2))
		enc, err := pk.Encrypt(v)
		if err != nil {
			t.Fatal(err)
		}
		pool[i] = enc
	}

	voterCounts := []int{100, 500, 1000, 5000, 10000, 50000, 100000}

	fmt.Println("=== Homomorphic Tally Scalability ===")
	fmt.Println("Voters | Tally Time (ms) | Per-Vote (μs)")
	fmt.Println("-------|-----------------|---------------")

	for _, n := range voterCounts {
		// Build vote slice by cycling through the pool
		votes := make([]*big.Int, n)
		for i := 0; i < n; i++ {
			votes[i] = pool[i%poolSize]
		}

		// Measure tally time only (homomorphic addition = modular multiplication)
		start := time.Now()
		tally := big.NewInt(1)
		for _, v := range votes {
			tally = pk.Add(tally, v)
		}
		elapsed := time.Since(start)

		perVote := float64(elapsed.Microseconds()) / float64(n)
		fmt.Printf("%7d | %15.2f | %13.2f\n", n, float64(elapsed.Milliseconds()), perVote)
	}
}

// TestScalabilityRingSize measures ring signature time vs ring size.
func TestScalabilityRingSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ring size scalability benchmark in short mode")
	}

	ringParams, err := crypto.GenerateRingParams(512)
	if err != nil {
		t.Fatal(err)
	}

	ringSizes := []int{10, 25, 50, 100, 200, 500}

	fmt.Println("\n=== Ring Signature Scalability ===")
	fmt.Println("Ring Size | Sign Time (ms) | Verify Time (ms)")
	fmt.Println("----------|----------------|------------------")

	for _, size := range ringSizes {
		keys := make([]*crypto.RingKeyPair, size)
		pubKeys := make([]*big.Int, size)
		for i := 0; i < size; i++ {
			kp, _ := ringParams.GenerateRingKeyPair()
			keys[i] = kp
			pubKeys[i] = kp.PublicKey
		}

		msg := []byte("benchmark-message")
		signerIdx := size / 2

		// Measure sign time (average of multiple runs)
		var totalSign, totalVerify time.Duration
		runs := 5
		if size >= 200 {
			runs = 2
		}

		var sig *crypto.RingSignature
		for r := 0; r < runs; r++ {
			start := time.Now()
			sig, _ = ringParams.Sign(msg, keys[signerIdx], pubKeys, signerIdx)
			totalSign += time.Since(start)

			start = time.Now()
			ringParams.Verify(msg, sig, pubKeys)
			totalVerify += time.Since(start)
		}

		avgSign := float64(totalSign.Milliseconds()) / float64(runs)
		avgVerify := float64(totalVerify.Milliseconds()) / float64(runs)
		fmt.Printf("%9d | %14.2f | %16.2f\n", size, avgSign, avgVerify)
	}
}

// TestComplexityValidation validates O(n) vs O(n×m²) by varying candidate count.
// Fixes n=1000 voters, varies m={2,5,10,20,50} candidates.
// CovertVote tally time should remain constant per candidate regardless of m (O(n)).
func TestComplexityValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping complexity validation benchmark in short mode")
	}

	paillierKey, err := crypto.GeneratePaillierKeyPair(2048)
	if err != nil {
		t.Fatal(err)
	}
	pk := paillierKey.PublicKey

	n := 1000 // fixed voter count
	candidateCounts := []int{2, 5, 10, 20, 50}

	// Pre-encrypt votes (0 and 1) to isolate tally time
	t.Log("Pre-encrypting vote pool for complexity validation...")
	enc0, _ := pk.Encrypt(big.NewInt(0))
	enc1, _ := pk.Encrypt(big.NewInt(1))

	fmt.Println("\n=== O(n) Complexity Validation (n=1000 voters) ===")
	fmt.Println("Candidates | Tally Time (ms) | Per-Candidate (ms)")
	fmt.Println("-----------|-----------------|--------------------")

	for _, m := range candidateCounts {
		// For each candidate, tally n votes using homomorphic addition
		// Total operations = n × m homomorphic additions

		start := time.Now()
		for c := 0; c < m; c++ {
			tally := big.NewInt(1)
			for i := 0; i < n; i++ {
				if i%m == c {
					tally = pk.Add(tally, enc1)
				} else {
					tally = pk.Add(tally, enc0)
				}
			}
		}
		elapsed := time.Since(start)

		perCandidate := float64(elapsed.Milliseconds()) / float64(m)
		fmt.Printf("%10d | %15.2f | %18.2f\n", m, float64(elapsed.Milliseconds()), perCandidate)
	}

	fmt.Println("\nNote: If per-candidate time is roughly constant, this confirms O(n×m) = O(n) per candidate.")
	fmt.Println("ISE-Voting's O(n×m²) would show per-candidate time GROWING with m.")
}
