# CovertVote - Quick Start Guide

## 🚀 Getting Started

### Prerequisites
- Go 1.21 or higher
- Make (for build automation)

### Installation

```bash
# Navigate to project directory
cd /home/bs01582/E-voting

# Install dependencies
make deps

# Verify installation
go version
go mod verify
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Test specific modules
make test-crypto      # Cryptographic primitives
make test-smdc        # SMDC credentials
make test-sa2         # SA² aggregation
make test-biometric   # Biometric processing
make test-voting      # Vote casting
make test-tally       # Vote tallying
```

### Project Structure

```
E-voting/
├── internal/          # Core implementation
│   ├── crypto/        # ✅ Cryptographic primitives
│   ├── smdc/          # ✅ SMDC credentials
│   ├── sa2/           # ✅ SA² aggregation
│   ├── biometric/     # ✅ Biometric auth
│   ├── voter/         # ✅ Voter registration
│   ├── voting/        # ✅ Vote casting
│   └── tally/         # ✅ Vote tallying
├── pkg/               # Utilities & config
├── api/               # API layer (TODO)
├── cmd/               # Server applications (TODO)
└── docs/              # Documentation
```

## 📚 Usage Examples

### Example 1: Paillier Encryption

```go
package main

import (
    "fmt"
    "math/big"
    "github.com/covertvote/e-voting/internal/crypto"
)

func main() {
    // Generate keys
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey

    // Encrypt votes
    vote1, _ := pk.Encrypt(big.NewInt(5))
    vote2, _ := pk.Encrypt(big.NewInt(3))

    // Homomorphic addition
    sum := pk.Add(vote1, vote2)

    // Decrypt
    result, _ := sk.Decrypt(sum)
    fmt.Printf("Total: %v\n", result) // Output: Total: 8
}
```

### Example 2: SMDC Credentials

```go
package main

import (
    "fmt"
    "github.com/covertvote/e-voting/internal/crypto"
    "github.com/covertvote/e-voting/internal/smdc"
)

func main() {
    // Setup
    pp, _ := crypto.GeneratePedersenParams(1024)
    gen := smdc.NewSMDCGenerator(pp, 5)

    // Generate credential
    cred, _ := gen.GenerateCredential("voter123")

    // Get public credential (for verification)
    pub := cred.GetPublicCredential()

    // Verify credential
    valid := gen.VerifyCredential(pub)
    fmt.Printf("Credential valid: %v\n", valid)

    // Get real slot (for actual voting)
    realSlot := cred.GetRealSlot()
    fmt.Printf("Real slot weight: %v\n", realSlot.Weight)
}
```

### Example 3: Complete Voting Flow

```go
package main

import (
    "fmt"
    "time"
    "github.com/covertvote/e-voting/internal/crypto"
    "github.com/covertvote/e-voting/internal/voting"
    "github.com/covertvote/e-voting/internal/voter"
)

func main() {
    // Setup cryptography
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey
    pp, _ := crypto.GeneratePedersenParams(1024)
    rp, _ := crypto.GenerateRingParams(1024)

    // Create election
    election := &voting.Election{
        ElectionID: "election001",
        Title:      "Presidential Election",
        Candidates: []*voting.Candidate{
            {ID: 1, Name: "Candidate A"},
            {ID: 2, Name: "Candidate B"},
        },
        StartTime: time.Now().Unix(),
        EndTime:   time.Now().Add(24 * time.Hour).Unix(),
        IsActive:  true,
    }

    // Setup registration
    eligibleVoters := []string{"voter001", "voter002"}
    rs := voter.NewRegistrationSystem(pp, rp, 5, eligibleVoters)

    // Create vote caster
    vc := voting.NewVoteCaster(pk, rp, rs, election)

    fmt.Printf("Voting system ready for %d candidates\n",
               len(election.Candidates))
}
```

### Example 4: SA² Vote Aggregation

```go
package main

import (
    "fmt"
    "math/big"
    "github.com/covertvote/e-voting/internal/crypto"
    "github.com/covertvote/e-voting/internal/sa2"
)

func main() {
    // Setup
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey

    // Encrypt votes
    votes := []*big.Int{
        big.NewInt(1), // Candidate 1
        big.NewInt(2), // Candidate 2
        big.NewInt(1), // Candidate 1
    }

    var encVotes []*big.Int
    for _, v := range votes {
        enc, _ := pk.Encrypt(v)
        encVotes = append(encVotes, enc)
    }

    // Split votes for SA²
    splitter := sa2.NewVoteSplitter(pk)
    var sharesA, sharesB []*big.Int

    for i, enc := range encVotes {
        share, _ := splitter.SplitVote(fmt.Sprintf("voter%d", i), enc)
        sharesA = append(sharesA, share.ShareA)
        sharesB = append(sharesB, share.ShareB)
    }

    // Aggregate at each server
    aggA := sa2.NewAggregator("ServerA", pk)
    aggB := sa2.NewAggregator("ServerB", pk)

    resultA := aggA.AggregateShares(sharesA)
    resultB := aggB.AggregateShares(sharesB)

    // Combine
    combiner := sa2.NewCombiner(pk)
    final := combiner.CombineAggregates(resultA, resultB)

    // Decrypt
    tally, _ := sk.Decrypt(final.EncryptedTally)
    fmt.Printf("Total votes: %v\n", tally) // 1+2+1 = 4
}
```

### Example 5: Vote Tallying

```go
package main

import (
    "fmt"
    "math/big"
    "github.com/covertvote/e-voting/internal/crypto"
    "github.com/covertvote/e-voting/internal/tally"
)

func main() {
    // Setup
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey

    // Create counter
    counter := tally.NewCounter(pk, sk)

    // Votes by candidate
    votesPerCandidate := make(map[int][]*big.Int)

    // Candidate 1: 3 votes
    for i := 0; i < 3; i++ {
        enc, _ := pk.Encrypt(big.NewInt(1))
        votesPerCandidate[1] = append(votesPerCandidate[1], enc)
    }

    // Candidate 2: 2 votes
    for i := 0; i < 2; i++ {
        enc, _ := pk.Encrypt(big.NewInt(1))
        votesPerCandidate[2] = append(votesPerCandidate[2], enc)
    }

    // Tally
    result, _ := counter.TallyByCandidate(votesPerCandidate, "election001")

    // Display results
    for candidateID, count := range result.CandidateTallies {
        fmt.Printf("Candidate %d: %v votes\n", candidateID, count)
    }
    fmt.Printf("Total votes: %d\n", result.TotalVotes)
}
```

## 🧪 Running Examples

Create a test file `examples/basic_test.go`:

```go
package examples

import (
    "math/big"
    "testing"
    "github.com/covertvote/e-voting/internal/crypto"
)

func TestBasicFlow(t *testing.T) {
    // Your example code here
    sk, _ := crypto.GeneratePaillierKeyPair(1024)
    pk := sk.PublicKey

    vote, _ := pk.Encrypt(big.NewInt(42))
    decrypted, _ := sk.Decrypt(vote)

    if decrypted.Cmp(big.NewInt(42)) != 0 {
        t.Errorf("Mismatch: got %v", decrypted)
    }
}
```

Run with:
```bash
go test ./examples/ -v
```

## 📊 Available Commands

```bash
# Development
make deps        # Install dependencies
make fmt         # Format code
make vet         # Run go vet

# Testing
make test               # Run all tests
make test-coverage      # Generate coverage report
make test-<module>      # Test specific module

# Cleaning
make clean       # Remove build artifacts
```

## 🔧 Configuration

Edit `pkg/config/config.yaml`:

```yaml
crypto:
  paillier_key_size: 2048  # Increase for production
  smdc_slots: 5            # Coercion resistance level

election:
  max_votes_per_second: 1000
  voting_period_hours: 24
```

Load configuration in code:

```go
cfg, _ := config.LoadConfig()
fmt.Printf("Key size: %d bits\n", cfg.Crypto.PaillierKeySize)
```

## 📖 Documentation

- `README.md` - Project overview
- `PROJECT_STATUS.md` - Initial implementation status
- `IMPLEMENTATION_COMPLETE.md` - Complete feature list
- `QUICKSTART.md` - This guide
- Inline documentation in all source files

## 🐛 Troubleshooting

### Tests failing?
```bash
# Clean and rebuild
make clean
go clean -cache
make deps
make test
```

### Import errors?
```bash
# Verify module name
grep module go.mod
# Should show: module github.com/covertvote/e-voting

# Tidy dependencies
go mod tidy
```

### Performance issues?
```bash
# Run benchmarks
go test -bench=. ./internal/crypto/
```

## 🎯 Next Steps

1. ✅ Core system is complete and tested
2. 🚧 Optional: Build REST API layer
3. 🚧 Optional: Add Hyperledger Fabric integration
4. 🚧 Optional: Create web/mobile frontend
5. 🚧 Optional: Deploy to production

## 📞 Support

- Check inline documentation in source files
- Review test files for usage examples
- See `IMPLEMENTATION_COMPLETE.md` for full feature list

## 🎉 You're Ready!

The CovertVote e-voting system core is fully implemented and tested. Start by running the examples above or writing your own tests.

Happy voting! 🗳️
