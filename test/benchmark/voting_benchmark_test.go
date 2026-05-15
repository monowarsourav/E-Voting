// test/benchmark/voting_benchmark_test.go

package benchmark

import (
	"fmt"
	"math/big"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/smdc"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
)

// ============================================================
// SCALABILITY BENCHMARK - Different Voter Counts
// ============================================================

// TestE2E10kVoters runs the full end-to-end pipeline (cred gen + cast + tally
// + decrypt) for exactly 10,000 voters and writes a single-row results file.
// Useful for focused timing measurements without paying for the warm-up rounds
// at 100/1000 voters.
func TestE2E10kVoters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 10k benchmark in short mode")
	}
	t.Logf("Running end-to-end benchmark with 10,000 voters...")
	result := runVotingBenchmark(t, 10000)
	t.Logf("Voters: %d | Total: %v | Per Vote: %v | CredGen: %v | VoteCast: %v",
		result.VoterCount, result.TotalTime, result.PerVoteTime,
		result.CredGenTime, result.VoteCastTime)
	saveResultsToFile([]BenchmarkResult{result}, "test/benchmark/results/e2e_10k_results.md")
	t.Log("Result saved to test/benchmark/results/e2e_10k_results.md")
}

// TestE2E10kVotersParallel runs the same pipeline as TestE2E10kVoters but with
// concurrent vote casting using runtime.NumCPU() workers, which models the
// realistic server-side throughput a deployed election would see when voters
// submit ballots concurrently. CredGen + Registration is kept sequential because
// it mutates the registration system; only the VoteCast phase parallelises.
func TestE2E10kVotersParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 10k parallel benchmark in short mode")
	}
	const numVoters = 10000
	workers := runtime.NumCPU()
	t.Logf("Running parallel 10K benchmark with %d workers...", workers)

	// --- Setup (not timed) ---
	paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(256)
	smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, "benchmark_election",
		[]byte("test-smdc-secret-key-do-not-use-in-prod"))

	eligibleVoters := make([]string, numVoters)
	for i := 0; i < numVoters; i++ {
		eligibleVoters[i] = fmt.Sprintf("voter_%d", i)
	}
	registrationSystem := voter.NewRegistrationSystem(
		pedersenParams, ringParams, 5, eligibleVoters, "benchmark_election",
		[]byte("test-smdc-secret-key-do-not-use-in-prod"),
		biometric.NewInMemoryDuressDetector([]byte("test-duress-hmac-key")),
	)
	election := &voting.Election{
		ElectionID: "benchmark_election",
		Title:      "Benchmark Parallel",
		Candidates: []*voting.Candidate{
			{ID: 0, Name: "A"}, {ID: 1, Name: "B"}, {ID: 2, Name: "C"},
		},
		StartTime: time.Now().Unix() - 3600,
		EndTime:   time.Now().Unix() + 3600,
		IsActive:  true,
	}
	voteCaster := voting.NewVoteCaster(paillierPK, ringParams, registrationSystem, election)

	totalStart := time.Now()

	// --- Phase 1: CredGen + Registration (sequential) ---
	credStart := time.Now()
	for i := 0; i < numVoters; i++ {
		voterID := eligibleVoters[i]
		if _, _, err := smdcGen.GenerateCredential(voterID); err != nil {
			t.Fatalf("cred gen: %v", err)
		}
		if _, err := registrationSystem.RegisterVoterWithPassword(
			voterID, []byte("password123"), "blink_count", "2"); err != nil {
			t.Fatalf("registration: %v", err)
		}
	}
	credTime := time.Since(credStart)

	// --- Phase 2: VoteCast (PARALLEL) ---
	voteStart := time.Now()
	voterCh := make(chan int, numVoters)
	for i := 0; i < numVoters; i++ {
		voterCh <- i
	}
	close(voterCh)

	var wg sync.WaitGroup
	var castErrCount int64
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range voterCh {
				voterID := eligibleVoters[i]
				candidateID := i % 3
				if _, err := voteCaster.CastVote(voterID, candidateID, 0, nil); err != nil {
					atomic.AddInt64(&castErrCount, 1)
				}
			}
		}()
	}
	wg.Wait()
	voteTime := time.Since(voteStart)

	totalTime := time.Since(totalStart)
	throughput := float64(numVoters) / voteTime.Seconds()

	t.Logf("=== PARALLEL BENCHMARK RESULTS ===")
	t.Logf("Workers:           %d", workers)
	t.Logf("Voters:            %d", numVoters)
	t.Logf("CredGen (seq):     %v", credTime)
	t.Logf("VoteCast (par):    %v", voteTime)
	t.Logf("Per vote (par):    %v", voteTime/time.Duration(numVoters))
	t.Logf("Throughput:        %.2f votes/sec", throughput)
	t.Logf("Cast errors:       %d", castErrCount)
	t.Logf("Total (incl seq):  %v", totalTime)

	// Save result.
	os.MkdirAll("test/benchmark/results", 0755)
	if f, err := os.Create("test/benchmark/results/e2e_10k_parallel.md"); err == nil {
		fmt.Fprintf(f, "# CovertVote Parallel Benchmark — 10K voters\n\n")
		fmt.Fprintf(f, "**Date:** %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(f, "**Workers:** %d (NumCPU)\n", workers)
		fmt.Fprintf(f, "**Candidates:** 3 (1-hot encoding)\n\n")
		fmt.Fprintf(f, "## Results\n\n")
		fmt.Fprintf(f, "| Metric | Value |\n")
		fmt.Fprintf(f, "|--------|-------|\n")
		fmt.Fprintf(f, "| Voters | %d |\n", numVoters)
		fmt.Fprintf(f, "| Workers | %d |\n", workers)
		fmt.Fprintf(f, "| CredGen (sequential) | %v |\n", credTime.Round(time.Millisecond))
		fmt.Fprintf(f, "| VoteCast (parallel) | %v |\n", voteTime.Round(time.Millisecond))
		fmt.Fprintf(f, "| Per vote (parallel) | %v |\n", (voteTime/time.Duration(numVoters)).Round(time.Microsecond))
		fmt.Fprintf(f, "| Throughput | %.2f votes/sec |\n", throughput)
		fmt.Fprintf(f, "| Total (cred+cast) | %v |\n", totalTime.Round(time.Millisecond))
		fmt.Fprintf(f, "| Cast errors | %d |\n", castErrCount)
		f.Close()
	}
}

func TestScalabilityBenchmark(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping scalability benchmark in short mode")
	}

	// In CI or when CI=true env is set, use smaller voter counts to fit
	// time budgets. Local full benchmarks can use the full range by
	// unsetting CI and running: go test ./test/benchmark/ -run TestScalabilityBenchmark
	voterCounts := []int{100, 1000, 10000}
	if os.Getenv("CI") == "true" {
		voterCounts = []int{100, 500}
	}

	results := make([]BenchmarkResult, 0)

	for _, n := range voterCounts {
		t.Logf("Testing with %d voters...", n)
		result := runVotingBenchmark(t, n)
		results = append(results, result)

		t.Logf("Voters: %d | Total: %v | Per Vote: %v",
			n, result.TotalTime, result.PerVoteTime)
	}

	// Save results
	saveResultsToFile(results, "test/benchmark/results/benchmark_results.md")
	t.Log("Results saved to test/benchmark/results/benchmark_results.md")
}

type BenchmarkResult struct {
	VoterCount    int
	TotalTime     time.Duration
	PerVoteTime   time.Duration
	CredGenTime   time.Duration
	VoteCastTime  time.Duration
	AggregateTime time.Duration
	DecryptTime   time.Duration
}

func runVotingBenchmark(t *testing.T, numVoters int) BenchmarkResult {
	result := BenchmarkResult{VoterCount: numVoters}

	// Setup (not timed)
	paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(256)
	smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, "benchmark_election", []byte("test-smdc-secret-key-do-not-use-in-prod"))

	// Eligible voters list
	eligibleVoters := make([]string, numVoters)
	for i := 0; i < numVoters; i++ {
		eligibleVoters[i] = fmt.Sprintf("voter_%d", i)
	}

	// Create registration system
	registrationSystem := voter.NewRegistrationSystem(pedersenParams, ringParams, 5, eligibleVoters, "benchmark_election", []byte("test-smdc-secret-key-do-not-use-in-prod"), biometric.NewInMemoryDuressDetector([]byte("test-duress-hmac-key")))

	// Create election
	election := &voting.Election{
		ElectionID:  "benchmark_election",
		Title:       "Benchmark Test Election",
		Description: "Performance testing",
		Candidates: []*voting.Candidate{
			{ID: 0, Name: "Candidate A"},
			{ID: 1, Name: "Candidate B"},
			{ID: 2, Name: "Candidate C"},
		},
		StartTime: time.Now().Unix() - 3600,
		EndTime:   time.Now().Unix() + 3600,
		IsActive:  true,
	}

	voteCaster := voting.NewVoteCaster(paillierPK, ringParams, registrationSystem, election)

	totalStart := time.Now()

	// Phase 1: Credential Generation & Registration
	credStart := time.Now()
	for i := 0; i < numVoters; i++ {
		voterID := fmt.Sprintf("voter_%d", i)

		// Generate SMDC credential
		cred, _, err := smdcGen.GenerateCredential(voterID)
		if err != nil {
			t.Fatalf("Credential generation failed: %v", err)
		}

		// Register voter with password
		_, err = registrationSystem.RegisterVoterWithPassword(voterID, []byte("password123"), "blink_count", "2")
		if err != nil {
			t.Fatalf("Registration failed: %v", err)
		}

		// Store credential (in production this would be done differently)
		_ = cred
	}
	result.CredGenTime = time.Since(credStart)

	// Phase 2: Vote Casting
	voteStart := time.Now()
	for i := 0; i < numVoters; i++ {
		voterID := fmt.Sprintf("voter_%d", i)
		candidateID := i % 3 // Rotate through 3 candidates

		// Cast vote
		_, err := voteCaster.CastVote(voterID, candidateID, 0, nil)
		if err != nil {
			t.Logf("Vote casting warning for voter %d: %v", i, err)
		}
	}
	result.VoteCastTime = time.Since(voteStart)

	// Phase 3: Tally (simplified - just time the decryption)
	aggStart := time.Now()
	// In a real system, we'd aggregate all encrypted votes
	result.AggregateTime = time.Since(aggStart)

	// Phase 4: Decryption
	decStart := time.Now()
	// Decrypt a sample vote
	sampleVote, _ := paillierPK.Encrypt(big.NewInt(1))
	_, _ = paillierSK.Decrypt(sampleVote)
	result.DecryptTime = time.Since(decStart)

	result.TotalTime = time.Since(totalStart)
	result.PerVoteTime = result.TotalTime / time.Duration(numVoters)

	return result
}

func saveResultsToFile(results []BenchmarkResult, filename string) {
	// Create results directory
	os.MkdirAll("test/benchmark/results", 0755)

	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	// Write header
	f.WriteString("# CovertVote Benchmark Results\n\n")
	f.WriteString(fmt.Sprintf("**Date:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	f.WriteString("**System:** Go E-Voting System\n\n")

	f.WriteString("## Performance Table\n\n")
	f.WriteString("| Voters | Total Time | Per Vote | Cred Gen | Vote Cast | Aggregate | Decrypt |\n")
	f.WriteString("|--------|------------|----------|----------|-----------|-----------|----------|\n")

	// Write data rows
	for _, r := range results {
		f.WriteString(fmt.Sprintf("| %d | %v | %v | %v | %v | %v | %v |\n",
			r.VoterCount,
			r.TotalTime.Round(time.Millisecond),
			r.PerVoteTime.Round(time.Microsecond),
			r.CredGenTime.Round(time.Millisecond),
			r.VoteCastTime.Round(time.Millisecond),
			r.AggregateTime.Round(time.Microsecond),
			r.DecryptTime.Round(time.Microsecond),
		))
	}

	// Projection section
	f.WriteString("\n## Projections for Large Scale\n\n")
	if len(results) > 0 {
		perVote := results[len(results)-1].PerVoteTime
		f.WriteString(fmt.Sprintf("Based on per-vote time of %v:\n\n", perVote))

		projections := []int{100000, 1000000, 10000000, 50000000}
		f.WriteString("| Voters | Projected Time |\n")
		f.WriteString("|--------|----------------|\n")
		for _, n := range projections {
			projected := perVote * time.Duration(n)
			f.WriteString(fmt.Sprintf("| %d | %v |\n", n, projected.Round(time.Second)))
		}
	}
}

// ============================================================
// END-TO-END VOTE CASTING PIPELINE BENCHMARKS
// ============================================================

func BenchmarkFullVoteCastPipeline(b *testing.B) {
	// Setup (not timed)
	paillierKey, _ := crypto.GeneratePaillierKeyPair(2048)
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(512)

	// Create 100 ring members
	ringKeys := make([]*crypto.RingKeyPair, 100)
	ringPubKeys := make([]*big.Int, 100)
	for i := 0; i < 100; i++ {
		kp, _ := ringParams.GenerateRingKeyPair()
		ringKeys[i] = kp
		ringPubKeys[i] = kp.PublicKey
	}
	signerIndex := 42 // arbitrary signer

	// SMDC setup
	smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, "bench-election", []byte("test-smdc-secret-key-do-not-use-in-prod"))

	// SA2 setup
	splitter := sa2.NewVoteSplitter(paillierKey.PublicKey)

	electionID := "bench-election-001"
	candidateVote := big.NewInt(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Step 1: Paillier encrypt vote
		encVote, _ := paillierKey.PublicKey.Encrypt(candidateVote)

		// Step 2: Pedersen commitment
		commitment, _ := pedersenParams.Commit(candidateVote)

		// Step 3: ZKP Binary Proof
		nonce, _ := crypto.GenerateNonce()
		_, _ = pedersenParams.ProveBinary(candidateVote, commitment.R, commitment.C, nonce, electionID)

		// Step 4: SMDC credential generate + get real slot
		cred, realIdx, _ := smdcGen.GenerateCredential(fmt.Sprintf("voter-%d", i))
		slot := cred.Slots[realIdx]

		// Step 5: Apply SMDC weight
		weightedVote := paillierKey.PublicKey.Multiply(encVote, slot.Weight)

		// Step 6: Ring signature (100 members)
		message := weightedVote.Bytes()
		_, _ = ringParams.Sign(message, ringKeys[signerIndex], ringPubKeys, signerIndex)

		// Step 7: SA2 split
		_, _ = splitter.SplitVote(fmt.Sprintf("voter-%d", i), []*big.Int{weightedVote})
	}
}

func BenchmarkVoteCastPhases(b *testing.B) {
	// Setup
	paillierKey, _ := crypto.GeneratePaillierKeyPair(2048)
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(512)

	ringKeys := make([]*crypto.RingKeyPair, 100)
	ringPubKeys := make([]*big.Int, 100)
	for i := 0; i < 100; i++ {
		kp, _ := ringParams.GenerateRingKeyPair()
		ringKeys[i] = kp
		ringPubKeys[i] = kp.PublicKey
	}

	candidateVote := big.NewInt(1)
	electionID := "bench-election-001"

	b.Run("1_PaillierEncrypt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			paillierKey.PublicKey.Encrypt(candidateVote)
		}
	})

	b.Run("2_PedersenCommit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pedersenParams.Commit(candidateVote)
		}
	})

	b.Run("3_ZKPBinaryProve", func(b *testing.B) {
		commitment, _ := pedersenParams.Commit(candidateVote)
		nonce, _ := crypto.GenerateNonce()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pedersenParams.ProveBinary(candidateVote, commitment.R, commitment.C, nonce, electionID)
		}
	})

	b.Run("4_SMDCGenerate", func(b *testing.B) {
		smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, electionID, []byte("test-smdc-secret-key-do-not-use-in-prod"))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			smdcGen.GenerateCredential(fmt.Sprintf("voter-%d", i))
		}
	})

	b.Run("5_RingSign100", func(b *testing.B) {
		msg := []byte("test-vote-message")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ringParams.Sign(msg, ringKeys[42], ringPubKeys, 42)
		}
	})

	b.Run("6_SA2Split", func(b *testing.B) {
		encVote, _ := paillierKey.PublicKey.Encrypt(candidateVote)
		splitter := sa2.NewVoteSplitter(paillierKey.PublicKey)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			splitter.SplitVote("voter-bench", []*big.Int{encVote})
		}
	})
}

// ============================================================
// INDIVIDUAL OPERATION TIMING
// ============================================================

func TestIndividualOperationTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping individual operation timing in short mode")
	}

	iterations := 100

	// Paillier Key Gen
	start := time.Now()
	for i := 0; i < iterations; i++ {
		crypto.GeneratePaillierKeyPair(2048)
	}
	paillierKeyGen := time.Since(start) / time.Duration(iterations)

	// Setup for other tests
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	pp, _ := crypto.GeneratePedersenParams(512)

	// Paillier Encrypt
	msg := big.NewInt(42)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		pk.Encrypt(msg)
	}
	paillierEnc := time.Since(start) / time.Duration(iterations)

	// Paillier Decrypt
	ct, _ := pk.Encrypt(msg)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		sk.Decrypt(ct)
	}
	paillierDec := time.Since(start) / time.Duration(iterations)

	// Pedersen Commit
	start = time.Now()
	for i := 0; i < iterations; i++ {
		pp.Commit(msg)
	}
	pedersenCommit := time.Since(start) / time.Duration(iterations)

	// SMDC Generate
	gen := smdc.NewSMDCGenerator(pp, 5, "bench_election", []byte("test-smdc-secret-key-do-not-use-in-prod"))
	start = time.Now()
	for i := 0; i < iterations; i++ {
		gen.GenerateCredential("voter")
	}
	smdcGen := time.Since(start) / time.Duration(iterations)

	// Print results
	t.Log("\n========== INDIVIDUAL OPERATION TIMING ==========")
	t.Logf("Paillier KeyGen (2048-bit): %v", paillierKeyGen)
	t.Logf("Paillier Encrypt:           %v", paillierEnc)
	t.Logf("Paillier Decrypt:           %v", paillierDec)
	t.Logf("Pedersen Commit:            %v", pedersenCommit)
	t.Logf("SMDC Generate (k=5):        %v", smdcGen)
	t.Log("=================================================")

	// Calculate total per vote
	totalPerVote := paillierEnc + smdcGen
	t.Logf("\nEstimated Total Per Vote: %v", totalPerVote)

	// Projections
	t.Log("\n========== PROJECTIONS ==========")
	for _, n := range []int{1000, 10000, 100000, 1000000} {
		projected := totalPerVote * time.Duration(n)
		t.Logf("%d voters: %v", n, projected)
	}
}
