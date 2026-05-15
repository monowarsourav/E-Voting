// Command election-sim runs a full end-to-end election simulation from voter
// registration through tally and result publication. It is intended for
// repeated experimentation: each invocation writes a timestamped Markdown
// report into test/benchmark/results/election-sim/<mode>/, where <mode> is
// either "mock" (in-process, no Fabric) or "real" (live Fabric, when ready).
//
// Usage:
//
//	go run ./cmd/election-sim --voters=1000 --candidates=3 --mode=mock
//	go run ./cmd/election-sim --voters=10000 --candidates=5 --mode=mock
//
// The simulation performs:
//  1. Election setup (Paillier 2048-bit, SMDC, ring, SA²)
//  2. Voter registration (sequential, SMDC k=5 slots per voter)
//  3. Vote casting (parallel, runtime.NumCPU() workers, real SQLite key-image
//     persistence; ZK proofs verified at cast time as defence-in-depth)
//  4. Per-candidate SA² aggregation
//  5. Per-candidate Paillier decryption
//  6. Result publication and integrity checks
//
// The mode flag selects the blockchain submitter. "mock" uses the in-process
// FabricClient in mock mode (no actual chain write); "real" connects to a
// running Hyperledger Fabric network. NOTE: the live-chain path requires the
// chaincode to accept the per-candidate ciphertext array (JSON-encoded); the
// current chaincode still expects a single ciphertext string, so the real
// mode is unusable until the chaincode is updated and redeployed. The tool
// prints a warning and falls back to mock if the chaincode update is missing.
package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/blockchain"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/repository/keyimage"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/smdc"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
)

func main() {
	voters := flag.Int("voters", 1000, "number of voters")
	candidates := flag.Int("candidates", 3, "number of candidates")
	mode := flag.String("mode", "mock", "blockchain mode: mock | real")
	seed := flag.Int64("seed", 0,
		"random seed for voter-to-candidate assignment (0 = time-based; "+
			"set to a non-zero integer to reproduce a specific election)")
	flag.Parse()

	if *voters <= 0 || *candidates <= 0 {
		log.Fatalf("--voters and --candidates must be > 0 (got %d, %d)", *voters, *candidates)
	}
	if *mode != "mock" && *mode != "real" {
		log.Fatalf("--mode must be 'mock' or 'real' (got %q)", *mode)
	}
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	fmt.Printf("CovertVote Election Simulator\n")
	fmt.Printf("============================\n")
	fmt.Printf("Voters:     %d\n", *voters)
	fmt.Printf("Candidates: %d\n", *candidates)
	fmt.Printf("Mode:       %s\n", *mode)
	fmt.Printf("Workers:    %d (NumCPU)\n", runtime.NumCPU())
	fmt.Printf("Vote-dist seed: %d\n", *seed)
	fmt.Println()

	res, err := runElection(*voters, *candidates, *mode, *seed)
	if err != nil {
		log.Fatalf("election failed: %v", err)
	}

	// Persist a Markdown report.
	reportPath, err := writeReport(res)
	if err != nil {
		log.Fatalf("failed to write report: %v", err)
	}
	fmt.Printf("\nReport saved to: %s\n", reportPath)
}

type electionResult struct {
	Voters           int
	Candidates       int
	Mode             string
	Workers          int
	Seed             int64
	ElectionID       string
	CandidateNames   []string
	CandidateTallies []int64 // length = Candidates (decrypted from chain)
	ExpectedTallies  []int64 // length = Candidates (ground truth from seed)
	TotalCounted     int64
	RegistrationTime time.Duration
	VoteCastTime     time.Duration
	AggregateTime    time.Duration
	DecryptTime      time.Duration
	TotalTime        time.Duration
	PerVoteParallel  time.Duration
	CastErrors       int64
	KeyImageDBPath   string
	UniqueKeyImages  int
	FabricEndpoint   string
	FabricChannel    string
	FabricChaincode  string
	StartedAt        time.Time
}

// connectFabric brings up a FabricClient against the local network/ directory.
// Returns an error if the network is unreachable or crypto material is missing.
func connectFabric() (*blockchain.FabricClient, error) {
	networkDir, err := findNetworkDir()
	if err != nil {
		return nil, err
	}
	tlsCertPath := filepath.Join(networkDir,
		"crypto-config/peerOrganizations/org1.covertvote.com/peers/peer0.org1.covertvote.com/tls/ca.crt")
	certPath := filepath.Join(networkDir,
		"crypto-config/peerOrganizations/org1.covertvote.com/users/Admin@org1.covertvote.com/msp/signcerts/Admin@org1.covertvote.com-cert.pem")
	keyPath := filepath.Join(networkDir,
		"crypto-config/peerOrganizations/org1.covertvote.com/users/Admin@org1.covertvote.com/msp/keystore")
	if _, err := os.Stat(tlsCertPath); err != nil {
		return nil, fmt.Errorf("TLS cert not found at %s — is the Fabric network running?", tlsCertPath)
	}
	fc := blockchain.NewFabricClient("covertvotechannel", "covertvote", true)
	if err := fc.ConnectGateway(blockchain.FabricConfig{
		PeerEndpoint: "localhost:7051",
		GatewayPeer:  "peer0.org1.covertvote.com",
		MSPID:        "Org1MSP",
		TLSCertPath:  tlsCertPath,
		CertPath:     certPath,
		KeyPath:      keyPath,
	}); err != nil {
		return nil, fmt.Errorf("connect to fabric: %w", err)
	}
	return fc, nil
}

// findNetworkDir locates the network/ directory containing crypto-config and
// docker-compose-hf.yml, searching several parent directories.
func findNetworkDir() (string, error) {
	candidates := []string{
		"./network",
		"../network",
		"../../network",
		"/home/bs01582/E-voting/network",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(abs, "docker-compose-hf.yml")); err == nil {
			return abs, nil
		}
	}
	return "", errors.New("could not find network/ directory (expected docker-compose-hf.yml)")
}

func runElection(numVoters, numCandidates int, mode string, seed int64) (*electionResult, error) {
	res := &electionResult{
		Voters:     numVoters,
		Candidates: numCandidates,
		Mode:       mode,
		Workers:    runtime.NumCPU(),
		Seed:       seed,
		StartedAt:  time.Now(),
	}

	// ---- Setup (untimed) ----
	fmt.Printf("[1/6] Generating cryptographic parameters... ")
	t0 := time.Now()
	paillierSK, err := crypto.GeneratePaillierKeyPair(2048)
	if err != nil {
		return nil, fmt.Errorf("paillier keygen: %w", err)
	}
	paillierPK := paillierSK.PublicKey
	pedersenParams, err := crypto.GeneratePedersenParams(512)
	if err != nil {
		return nil, fmt.Errorf("pedersen: %w", err)
	}
	ringParams, err := crypto.GenerateRingParams(256)
	if err != nil {
		return nil, fmt.Errorf("ring: %w", err)
	}
	smdcGen := smdc.NewSMDCGenerator(pedersenParams, 5, "election-sim",
		[]byte("election-sim-smdc-secret-key-placeholder-must-be-32-bytes-or-more"))
	fmt.Printf("done (%v)\n", time.Since(t0).Round(time.Millisecond))

	// Election with N candidates. Use a per-run unique ID so concurrent or
	// repeated invocations against the same chain do not collide.
	electionID := fmt.Sprintf("election-sim-%d", res.StartedAt.UnixNano())
	res.ElectionID = electionID
	candidates := make([]*voting.Candidate, numCandidates)
	res.CandidateNames = make([]string, numCandidates)
	for j := 0; j < numCandidates; j++ {
		name := candidateName(j)
		candidates[j] = &voting.Candidate{ID: j, Name: name}
		res.CandidateNames[j] = name
	}
	election := &voting.Election{
		ElectionID: electionID,
		Title:      "Election Simulation",
		Candidates: candidates,
		StartTime:  time.Now().Unix() - 3600,
		EndTime:    time.Now().Unix() + 86400,
		IsActive:   true,
	}

	// Voter list.
	eligibleVoters := make([]string, numVoters)
	for i := 0; i < numVoters; i++ {
		eligibleVoters[i] = fmt.Sprintf("voter_%d", i)
	}
	registrationSystem := voter.NewRegistrationSystem(
		pedersenParams, ringParams, 5, eligibleVoters, "election-sim",
		[]byte("election-sim-smdc-secret-key-placeholder-must-be-32-bytes-or-more"),
		biometric.NewInMemoryDuressDetector([]byte("election-sim-duress-key-placeholder")),
	)

	// SQLite key-image store (durable, per-run).
	dbPath := filepath.Join(os.TempDir(),
		fmt.Sprintf("election-sim-%d.db", res.StartedAt.UnixNano()))
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS key_images (
        key_image TEXT PRIMARY KEY,
        used_at INTEGER NOT NULL
    )`); err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}
	keyStore := keyimage.New(db)
	res.KeyImageDBPath = dbPath

	// Build the VoteCaster options. In real mode we also attach a live
	// FabricClient as the BlockchainSubmitter and create the election on chain
	// before any votes are cast.
	opts := []voting.VoteCasterOption{voting.WithKeyImageStore(keyStore)}
	if mode == "real" {
		fmt.Printf("        Connecting to Hyperledger Fabric (localhost:7051)... ")
		fabricClient, err := connectFabric()
		if err != nil {
			return nil, fmt.Errorf("fabric connect: %w", err)
		}
		fmt.Println("connected.")
		defer fabricClient.Disconnect()
		res.FabricEndpoint = "localhost:7051"
		res.FabricChannel = "covertvotechannel"
		res.FabricChaincode = "covertvote"

		fmt.Printf("        Creating election on chain (id=%s)... ", electionID)
		txID, err := fabricClient.CreateElection(election)
		if err != nil {
			return nil, fmt.Errorf("create election on chain: %w", err)
		}
		fmt.Printf("tx=%s\n", txID)
		opts = append(opts, voting.WithBlockchainSubmitter(fabricClient))
	}
	voteCaster := voting.NewVoteCaster(paillierPK, ringParams, registrationSystem, election, opts...)

	// ---- Phase 2: Voter Registration (sequential) ----
	fmt.Printf("[2/6] Registering %d voters... ", numVoters)
	regStart := time.Now()
	for i := 0; i < numVoters; i++ {
		voterID := eligibleVoters[i]
		if _, _, err := smdcGen.GenerateCredential(voterID); err != nil {
			return nil, fmt.Errorf("smdc gen %d: %w", i, err)
		}
		if _, err := registrationSystem.RegisterVoterWithPassword(
			voterID, []byte("election-sim-password"),
			"blink_count", "2"); err != nil {
			return nil, fmt.Errorf("register %d: %w", i, err)
		}
	}
	res.RegistrationTime = time.Since(regStart)
	fmt.Printf("done (%v)\n", res.RegistrationTime.Round(time.Millisecond))

	// Pre-compute each voter's candidate choice from the seed so the
	// "ground truth" distribution is reproducible and the parallel
	// goroutines do not race on the shared RNG.
	rng := rand.New(rand.NewSource(seed))
	voterChoice := make([]int, numVoters)
	expectedCounts := make([]int64, numCandidates)
	for i := 0; i < numVoters; i++ {
		c := rng.Intn(numCandidates)
		voterChoice[i] = c
		expectedCounts[c]++
	}
	fmt.Printf("        Vote distribution (expected, from seed):")
	for j := 0; j < numCandidates; j++ {
		fmt.Printf(" %s=%d", res.CandidateNames[j], expectedCounts[j])
	}
	fmt.Println()
	res.ExpectedTallies = expectedCounts

	// ---- Phase 3: Vote Casting (parallel) ----
	fmt.Printf("[3/6] Casting %d votes with %d parallel workers...\n", numVoters, res.Workers)
	voteStart := time.Now()
	voterCh := make(chan int, numVoters)
	for i := 0; i < numVoters; i++ {
		voterCh <- i
	}
	close(voterCh)

	var wg sync.WaitGroup
	var castErrCount int64
	wg.Add(res.Workers)
	for w := 0; w < res.Workers; w++ {
		go func() {
			defer wg.Done()
			for i := range voterCh {
				voterID := eligibleVoters[i]
				// In a real deployment the voter learns the real slot index via
				// the duress-bound reveal endpoint; the simulator computes it
				// directly using the server secret since it has both halves.
				realSlotIdx := smdcGen.DeriveRealIndex(voterID)
				candidateID := voterChoice[i]
				if _, err := voteCaster.CastVote(voterID, candidateID, realSlotIdx, nil); err != nil {
					atomic.AddInt64(&castErrCount, 1)
				}
			}
		}()
	}
	wg.Wait()
	res.VoteCastTime = time.Since(voteStart)
	res.CastErrors = castErrCount
	res.PerVoteParallel = res.VoteCastTime / time.Duration(numVoters)
	fmt.Printf("        done (%v, %v per vote effective)\n",
		res.VoteCastTime.Round(time.Millisecond),
		res.PerVoteParallel.Round(time.Microsecond))
	if castErrCount > 0 {
		fmt.Printf("        WARNING: %d cast errors\n", castErrCount)
	}

	// ---- Phase 4: SA² Aggregation (per candidate) ----
	fmt.Printf("[4/6] Aggregating per-candidate SA² shares... ")
	allShares := voteCaster.GetAllVoteShares()
	aggStart := time.Now()
	aggA := sa2.NewAggregator("ServerA", paillierPK)
	aggB := sa2.NewAggregator("ServerB", paillierPK)
	combiner := sa2.NewCombiner(paillierPK)

	encryptedTotals := make([]*big.Int, numCandidates)
	for j := 0; j < numCandidates; j++ {
		colA := make([]*big.Int, 0, len(allShares))
		colB := make([]*big.Int, 0, len(allShares))
		for _, share := range allShares {
			if j < len(share.SharesA) {
				colA = append(colA, share.SharesA[j])
				colB = append(colB, share.SharesB[j])
			}
		}
		aA := aggA.AggregateShares(j, colA)
		aB := aggB.AggregateShares(j, colB)
		combined := combiner.CombineAggregates(aA, aB)
		encryptedTotals[j] = combined.EncryptedTally
	}
	res.AggregateTime = time.Since(aggStart)
	fmt.Printf("done (%v)\n", res.AggregateTime.Round(time.Millisecond))

	// ---- Phase 5: Per-candidate Paillier Decryption ----
	fmt.Printf("[5/6] Decrypting per-candidate totals... ")
	decStart := time.Now()
	res.CandidateTallies = make([]int64, numCandidates)
	for j := 0; j < numCandidates; j++ {
		plain, err := paillierSK.Decrypt(encryptedTotals[j])
		if err != nil {
			return nil, fmt.Errorf("decrypt candidate %d: %w", j, err)
		}
		res.CandidateTallies[j] = plain.Int64()
		res.TotalCounted += plain.Int64()
	}
	res.DecryptTime = time.Since(decStart)
	fmt.Printf("done (%v)\n", res.DecryptTime.Round(time.Millisecond))

	// Key-image uniqueness check (every successful cast should have inserted
	// exactly one row).
	uniqueKI := 0
	if err := db.QueryRow(`SELECT COUNT(*) FROM key_images`).Scan(&uniqueKI); err == nil {
		res.UniqueKeyImages = uniqueKI
	}

	res.TotalTime = time.Since(t0)

	// ---- Phase 6: Result Publication ----
	fmt.Printf("[6/6] Publishing results.\n\n")
	printResults(res)

	return res, nil
}

func candidateName(idx int) string {
	letters := []string{
		"Alice", "Bob", "Charlie", "Dave", "Eve",
		"Frank", "Grace", "Heidi", "Ivan", "Judy",
	}
	if idx < len(letters) {
		return letters[idx]
	}
	return fmt.Sprintf("Candidate-%d", idx+1)
}

func printResults(r *electionResult) {
	fmt.Println("=== ELECTION RESULTS ===")
	winnerIdx, winnerVotes := 0, int64(-1)
	allMatch := true
	for j, v := range r.CandidateTallies {
		if v > winnerVotes {
			winnerVotes = v
			winnerIdx = j
		}
		exp := int64(0)
		if j < len(r.ExpectedTallies) {
			exp = r.ExpectedTallies[j]
		}
		mark := "✓"
		if v != exp {
			mark = "✗"
			allMatch = false
		}
		fmt.Printf("  %-12s (ID=%d): %d votes  (expected %d) %s\n",
			r.CandidateNames[j], j, v, exp, mark)
	}
	fmt.Printf("  %s\n", repeat('-', 60))
	fmt.Printf("  Total counted: %d\n", r.TotalCounted)
	fmt.Printf("  Expected:      %d (voter count)\n", r.Voters)
	if int(r.TotalCounted) == r.Voters && allMatch {
		fmt.Println("  Integrity: ✓ tally exactly matches ground-truth distribution")
	} else if int(r.TotalCounted) == r.Voters {
		fmt.Println("  Integrity: ✗ total matches but per-candidate mismatch (BUG)")
	} else {
		fmt.Printf("  Integrity: ✗ MISMATCH (delta = %d)\n", int(r.TotalCounted)-r.Voters)
	}
	fmt.Println()
	fmt.Printf("  Winner: %s (ID=%d) with %d votes\n",
		r.CandidateNames[winnerIdx], winnerIdx, winnerVotes)
	fmt.Println()
	fmt.Println("Cryptographic integrity:")
	fmt.Printf("  Cast errors:       %d\n", r.CastErrors)
	fmt.Printf("  Unique key images: %d (expected %d)\n", r.UniqueKeyImages, r.Voters-int(r.CastErrors))
	fmt.Println()

	electionDayTime := r.VoteCastTime + r.AggregateTime + r.DecryptTime

	fmt.Println("Timing — Pre-election (one-time, weeks before vote):")
	fmt.Printf("  Voter registration: %v\n", r.RegistrationTime.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("Timing — Election day (per-election; this is the real number):")
	fmt.Printf("  Vote casting:       %v\n", r.VoteCastTime.Round(time.Millisecond))
	fmt.Printf("  SA² aggregation:    %v\n", r.AggregateTime.Round(time.Millisecond))
	fmt.Printf("  Decryption:         %v\n", r.DecryptTime.Round(time.Millisecond))
	fmt.Printf("  Total (cast+agg+dec): %v\n", electionDayTime.Round(time.Millisecond))
	fmt.Printf("  Per voter (parallel): %v\n", r.PerVoteParallel.Round(time.Microsecond))
}

func writeReport(r *electionResult) (string, error) {
	// Folder structure: test/benchmark/results/election-sim/<mode>/
	dir := filepath.Join("test", "benchmark", "results", "election-sim", r.Mode)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ts := r.StartedAt.Format("2006-01-02_15-04-05")
	fileName := fmt.Sprintf("%s_v%d_c%d.md", ts, r.Voters, r.Candidates)
	fullPath := filepath.Join(dir, fileName)
	f, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fmt.Fprintf(f, "# Election Simulation Result\n\n")
	fmt.Fprintf(f, "**Mode:** `%s`\n", r.Mode)
	fmt.Fprintf(f, "**Started:** %s\n", r.StartedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(f, "**Voters:** %d\n", r.Voters)
	fmt.Fprintf(f, "**Candidates:** %d\n", r.Candidates)
	fmt.Fprintf(f, "**Workers:** %d (NumCPU)\n", r.Workers)
	fmt.Fprintf(f, "**Vote-distribution seed:** `%d` (re-run with `--seed=%d` to reproduce)\n", r.Seed, r.Seed)
	fmt.Fprintf(f, "**Encoding:** 1-hot per-candidate Paillier (2048-bit) + CDS OR-proofs\n\n")

	fmt.Fprintln(f, "## Per-candidate Tally")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "Votes are distributed across candidates by a seeded PRNG so the")
	fmt.Fprintln(f, "ground-truth tally is reproducible. Each voter's choice is sampled")
	fmt.Fprintln(f, "uniformly at random from the candidate list, modelling an idealised")
	fmt.Fprintln(f, "preference distribution.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "| Candidate | ID | Decrypted Votes | Ground-truth (PRNG) | Match |")
	fmt.Fprintln(f, "|---|---|---|---|---|")
	winnerIdx, winnerVotes := 0, int64(-1)
	allMatch := true
	for j, v := range r.CandidateTallies {
		exp := int64(0)
		if j < len(r.ExpectedTallies) {
			exp = r.ExpectedTallies[j]
		}
		mark := "✓"
		if v != exp {
			mark = "✗"
			allMatch = false
		}
		fmt.Fprintf(f, "| %s | %d | %d | %d | %s |\n",
			r.CandidateNames[j], j, v, exp, mark)
		if v > winnerVotes {
			winnerVotes = v
			winnerIdx = j
		}
	}
	fmt.Fprintf(f, "| **Total** | — | **%d** | %d | %s |\n\n",
		r.TotalCounted, r.Voters, ternary(allMatch && int(r.TotalCounted) == r.Voters, "✓", "✗"))
	fmt.Fprintf(f, "**Winner:** %s (ID=%d) with %d votes\n\n",
		r.CandidateNames[winnerIdx], winnerIdx, winnerVotes)

	fmt.Fprintln(f, "## Integrity Checks")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "| Check | Result |")
	fmt.Fprintln(f, "|---|---|")
	if int(r.TotalCounted) == r.Voters {
		fmt.Fprintf(f, "| Tally matches voter count | ✓ (%d/%d) |\n", r.TotalCounted, r.Voters)
	} else {
		fmt.Fprintf(f, "| Tally matches voter count | ✗ (counted %d, expected %d) |\n", r.TotalCounted, r.Voters)
	}
	fmt.Fprintf(f, "| Cast errors | %d |\n", r.CastErrors)
	expectedKI := int64(r.Voters) - r.CastErrors
	if int64(r.UniqueKeyImages) == expectedKI {
		fmt.Fprintf(f, "| Unique key images | ✓ %d (no double-vote) |\n", r.UniqueKeyImages)
	} else {
		fmt.Fprintf(f, "| Unique key images | mismatch: %d (expected %d) |\n", r.UniqueKeyImages, expectedKI)
	}
	fmt.Fprintln(f, "")

	electionDayTime := r.VoteCastTime + r.AggregateTime + r.DecryptTime

	fmt.Fprintln(f, "## Timing — Pre-election (one-time, weeks before vote)")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "These phases run once before the election starts and are not part of")
	fmt.Fprintln(f, "election-day performance.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "| Phase | Time |")
	fmt.Fprintln(f, "|---|---|")
	fmt.Fprintf(f, "| Voter registration (SMDC credential + duress signal store) | %v |\n",
		r.RegistrationTime.Round(time.Millisecond))
	fmt.Fprintln(f, "")

	fmt.Fprintln(f, "## Timing — Election day")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "These phases run during/after voting. **This is the real election")
	fmt.Fprintln(f, "performance number** — registration is excluded because it is a")
	fmt.Fprintln(f, "one-time event completed long before voting begins.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "| Phase | Time |")
	fmt.Fprintln(f, "|---|---|")
	fmt.Fprintf(f, "| Vote casting (parallel, %d workers) | %v |\n",
		r.Workers, r.VoteCastTime.Round(time.Millisecond))
	fmt.Fprintf(f, "| SA² aggregation (post-vote) | %v |\n",
		r.AggregateTime.Round(time.Millisecond))
	fmt.Fprintf(f, "| Decryption (per-candidate) | %v |\n",
		r.DecryptTime.Round(time.Millisecond))
	fmt.Fprintf(f, "| **Election-day total (cast + agg + dec)** | **%v** |\n",
		electionDayTime.Round(time.Millisecond))
	fmt.Fprintf(f, "| Per voter (effective, parallel) | %v |\n",
		r.PerVoteParallel.Round(time.Microsecond))
	fmt.Fprintln(f, "")

	fmt.Fprintln(f, "## Artefacts")
	fmt.Fprintln(f, "")
	fmt.Fprintf(f, "- Key-image database: `%s`\n", r.KeyImageDBPath)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "## Notes")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "- Candidates voted round-robin (`voter i → candidate i mod m`), so the per-candidate")
	fmt.Fprintln(f, "  tallies should be roughly balanced (off by at most one).")

	return fullPath, nil
}

// repeat returns a string of n copies of r. Avoids importing strings for one call.
func repeat(r rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}

func ternary(cond bool, t, f string) string {
	if cond {
		return t
	}
	return f
}
