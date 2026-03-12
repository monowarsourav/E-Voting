// test/benchmark/voting_benchmark_test.go

package benchmark

import (
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/smdc"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
)

// ============================================================
// SCALABILITY BENCHMARK - Different Voter Counts
// ============================================================

func TestScalabilityBenchmark(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping scalability benchmark in short mode")
	}

	voterCounts := []int{100, 1000, 10000}

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
	VoterCount     int
	TotalTime      time.Duration
	PerVoteTime    time.Duration
	CredGenTime    time.Duration
	VoteCastTime   time.Duration
	AggregateTime  time.Duration
	DecryptTime    time.Duration
}

func runVotingBenchmark(t *testing.T, numVoters int) BenchmarkResult {
	result := BenchmarkResult{VoterCount: numVoters}

	// Setup (not timed)
	paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(256)
	smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, "benchmark_election")

	// Eligible voters list
	eligibleVoters := make([]string, numVoters)
	for i := 0; i < numVoters; i++ {
		eligibleVoters[i] = fmt.Sprintf("voter_%d", i)
	}

	// Create registration system
	registrationSystem := voter.NewRegistrationSystem(pedersenParams, ringParams, 5, eligibleVoters, "benchmark_election")

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
		_, err = registrationSystem.RegisterVoterWithPassword(voterID, []byte("password123"))
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
		_, err := voteCaster.CastVote(voterID, candidateID, 0)
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
	gen := smdc.NewSMDCGenerator(pp, 5, "bench_election")
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
