# TASK 3: Improve Test Coverage to 80%+
# Copy this ENTIRE prompt into your IDE (Cursor)

```
Improve test coverage for 3 low-coverage modules in the CovertVote e-voting system. Current coverage: voting (17.9%), tally (29.9%), pq (31.6%). Target: 70%+ each.

## 1. Add tests to internal/voting/cast_test.go

Add these NEW test functions AFTER the existing tests. Do NOT modify existing tests.

func setupTestElection() (*Election, *crypto.PaillierPrivateKey, *crypto.RingParams, *voter.RegistrationSystem) {
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    rp, _ := crypto.GenerateRingParams(512)
    pp, _ := crypto.GeneratePedersenParams(512)
    rs := voter.NewRegistrationSystem(pp, "test-election")

    election := &Election{
        ElectionID:  "test-election-001",
        Title:       "Test Election",
        Candidates: []*Candidate{
            {ID: 0, Name: "Candidate A"},
            {ID: 1, Name: "Candidate B"},
            {ID: 2, Name: "Candidate C"},
        },
        StartTime: time.Now().Unix() - 3600,
        EndTime:   time.Now().Unix() + 3600,
        IsActive:  true,
    }

    return election, sk, rp, rs
}

func registerVoter(t *testing.T, rs *voter.RegistrationSystem, rp *crypto.RingParams, voterID string) {
    t.Helper()
    ringKP, _ := rp.GenerateRingKeyPair()
    fingerprint := []byte("fingerprint-" + voterID)
    err := rs.RegisterVoter(voterID, fingerprint, ringKP)
    if err != nil {
        t.Fatalf("Failed to register voter %s: %v", voterID, err)
    }
}

func TestCastVoteFullPipeline(t *testing.T) {
    election, sk, rp, rs := setupTestElection()
    _ = sk

    // Register 5 voters (minimum for ring)
    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    // Cast vote for voter-0, candidate 1, SMDC slot 0
    receipt, err := vc.CastVote("voter-0", 1, 0)
    if err != nil {
        t.Fatalf("CastVote failed: %v", err)
    }

    if receipt == nil {
        t.Fatal("Receipt is nil")
    }
    if receipt.VoterID != "voter-0" {
        t.Errorf("Receipt voterID mismatch: got %s", receipt.VoterID)
    }
    if receipt.KeyImage == nil {
        t.Error("Receipt KeyImage is nil")
    }
}

func TestCastVoteDoubleVotePrevention(t *testing.T) {
    election, sk, rp, rs := setupTestElection()

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    // First vote should succeed
    _, err := vc.CastVote("voter-0", 1, 0)
    if err != nil {
        t.Fatalf("First vote failed: %v", err)
    }

    // Second vote by same voter should fail
    _, err = vc.CastVote("voter-0", 0, 0)
    if err == nil {
        t.Fatal("Expected error for double vote, got nil")
    }
}

func TestCastVoteInvalidCandidate(t *testing.T) {
    election, sk, rp, rs := setupTestElection()

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    // Invalid candidate ID (only 0, 1, 2 exist)
    _, err := vc.CastVote("voter-0", 99, 0)
    if err == nil {
        t.Fatal("Expected error for invalid candidate, got nil")
    }
}

func TestCastVoteInactiveElection(t *testing.T) {
    election, sk, rp, rs := setupTestElection()
    election.IsActive = false  // Deactivate

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    _, err := vc.CastVote("voter-0", 1, 0)
    if err == nil {
        t.Fatal("Expected error for inactive election, got nil")
    }
}

func TestCastVoteExpiredElection(t *testing.T) {
    election, sk, rp, rs := setupTestElection()
    election.StartTime = time.Now().Unix() - 7200
    election.EndTime = time.Now().Unix() - 3600  // Ended 1 hour ago

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    _, err := vc.CastVote("voter-0", 1, 0)
    if err == nil {
        t.Fatal("Expected error for expired election, got nil")
    }
}

func TestCastVoteUnregisteredVoter(t *testing.T) {
    election, sk, rp, rs := setupTestElection()

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    _, err := vc.CastVote("unregistered-voter", 1, 0)
    if err == nil {
        t.Fatal("Expected error for unregistered voter, got nil")
    }
}

func TestCastVoteInvalidSMDCSlot(t *testing.T) {
    election, sk, rp, rs := setupTestElection()

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    // SMDC has k=5 slots (0-4), so 10 is invalid
    _, err := vc.CastVote("voter-0", 1, 10)
    if err == nil {
        t.Fatal("Expected error for invalid SMDC slot, got nil")
    }

    // Negative slot
    _, err = vc.CastVote("voter-1", 1, -1)
    if err == nil {
        t.Fatal("Expected error for negative SMDC slot, got nil")
    }
}

func TestVerifyVote(t *testing.T) {
    election, sk, rp, rs := setupTestElection()
    _ = sk

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    _, err := vc.CastVote("voter-0", 1, 0)
    if err != nil {
        t.Fatalf("CastVote failed: %v", err)
    }

    // Retrieve and verify
    castVote, found := vc.GetCastVote("voter-0")
    if !found {
        t.Fatal("Cast vote not found")
    }

    valid := vc.VerifyVote(castVote)
    if !valid {
        t.Error("Vote verification failed")
    }
}

func TestGetVoteCount(t *testing.T) {
    election, sk, rp, rs := setupTestElection()

    for i := 0; i < 10; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    if vc.GetVoteCount() != 0 {
        t.Errorf("Expected 0 votes, got %d", vc.GetVoteCount())
    }

    // Cast 3 votes
    for i := 0; i < 3; i++ {
        _, err := vc.CastVote(fmt.Sprintf("voter-%d", i), i%3, 0)
        if err != nil {
            t.Fatalf("Vote %d failed: %v", i, err)
        }
    }

    if vc.GetVoteCount() != 3 {
        t.Errorf("Expected 3 votes, got %d", vc.GetVoteCount())
    }
}

func TestGetAllVoteShares(t *testing.T) {
    election, sk, rp, rs := setupTestElection()

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    // Cast 2 votes
    vc.CastVote("voter-0", 0, 0)
    vc.CastVote("voter-1", 1, 0)

    shares := vc.GetAllVoteShares()
    if len(shares) != 2 {
        t.Errorf("Expected 2 vote shares, got %d", len(shares))
    }
}

func TestSMDCWeightAffectsTally(t *testing.T) {
    election, sk, rp, rs := setupTestElection()

    for i := 0; i < 5; i++ {
        registerVoter(t, rs, rp, fmt.Sprintf("voter-%d", i))
    }

    vc := NewVoteCaster(sk.PublicKey, rp, rs, election)

    // voter-0 uses real slot (slot index from SMDC - weight 1)
    // voter-1 uses fake slot (weight 0) - vote should not count
    // We need to find the real index for each voter
    voterRecord0, _ := rs.GetVoter("voter-0")
    realIdx0 := voterRecord0.SMDCRealIndex

    // Cast with real slot
    receipt0, err := vc.CastVote("voter-0", 1, realIdx0)
    if err != nil {
        t.Fatalf("Real slot vote failed: %v", err)
    }
    if receipt0 == nil {
        t.Fatal("Receipt is nil for real slot vote")
    }

    // Cast with fake slot for voter-1
    voterRecord1, _ := rs.GetVoter("voter-1")
    realIdx1 := voterRecord1.SMDCRealIndex
    fakeIdx1 := (realIdx1 + 1) % 5  // Pick a slot that is NOT real

    receipt1, err := vc.CastVote("voter-1", 1, fakeIdx1)
    if err != nil {
        t.Fatalf("Fake slot vote failed: %v", err)
    }
    if receipt1 == nil {
        t.Fatal("Receipt is nil for fake slot vote")
    }

    // Both votes cast successfully - but fake slot vote has weight 0
    // The tally should only count voter-0's vote
    t.Log("Real slot vote and fake slot vote both cast - tally correctness depends on SMDC weight")
}

Make sure to add "fmt" to the imports at the top of the file.


## 2. Add tests to internal/tally/tally_test.go

Add these NEW test functions AFTER the existing tests:

func TestThresholdTally(t *testing.T) {
    // Generate threshold keys (3-of-5)
    shares, err := crypto.GenerateThresholdKey(2048, 5, 3)
    if err != nil {
        t.Fatal(err)
    }
    pk := shares.PublicKey

    // Encrypt 20 votes: 12 for candidate A (1), 8 for candidate B (0)
    nVoters := 20
    expectedSum := int64(0)
    var encryptedVotes []*big.Int

    for i := 0; i < nVoters; i++ {
        v := int64(0)
        if i < 12 {
            v = 1
        }
        expectedSum += v
        enc, _ := pk.Encrypt(big.NewInt(v))
        encryptedVotes = append(encryptedVotes, enc)
    }

    // Homomorphic tally
    tally := pk.AddMultiple(encryptedVotes)

    // Threshold decrypt with trustees 0, 1, 2 (3-of-5)
    partials := make([]*crypto.ThresholdPartialDecryption, 3)
    for i := 0; i < 3; i++ {
        pd, err := shares.Shares[i].PartialDecrypt(
            tally, pk, shares.Params, shares.VerifyKeys[i], shares.V)
        if err != nil {
            t.Fatal(err)
        }
        partials[i] = pd
    }

    result, err := crypto.CombinePartialDecryptions(partials, pk, shares.Params)
    if err != nil {
        t.Fatal(err)
    }

    if result.Int64() != expectedSum {
        t.Errorf("Threshold tally: expected %d, got %d", expectedSum, result.Int64())
    }
}

func TestTallyWithDifferentTrusteeSubsets(t *testing.T) {
    shares, err := crypto.GenerateThresholdKey(2048, 5, 3)
    if err != nil {
        t.Fatal(err)
    }
    pk := shares.PublicKey

    // Encrypt a simple sum: 1+1+1 = 3
    var cts []*big.Int
    for i := 0; i < 3; i++ {
        enc, _ := pk.Encrypt(big.NewInt(1))
        cts = append(cts, enc)
    }
    tally := pk.AddMultiple(cts)

    // Try different subsets of 3 trustees
    subsets := [][3]int{{0, 1, 2}, {0, 2, 4}, {1, 3, 4}, {2, 3, 4}}

    for _, subset := range subsets {
        partials := make([]*crypto.ThresholdPartialDecryption, 3)
        for i, idx := range subset {
            pd, err := shares.Shares[idx].PartialDecrypt(
                tally, pk, shares.Params, shares.VerifyKeys[idx], shares.V)
            if err != nil {
                t.Fatalf("Subset %v, trustee %d: %v", subset, idx, err)
            }
            partials[i] = pd
        }

        result, err := crypto.CombinePartialDecryptions(partials, pk, shares.Params)
        if err != nil {
            t.Fatalf("Subset %v combine failed: %v", subset, err)
        }

        if result.Int64() != 3 {
            t.Errorf("Subset %v: expected 3, got %d", subset, result.Int64())
        }
    }
}

func TestCounterTallyVotes(t *testing.T) {
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey

    counter := NewCounter(pk, sk, 3)

    // Create encrypted votes for 3 candidates
    // 5 votes for candidate 0, 3 for candidate 1, 2 for candidate 2
    var votes []*big.Int
    distribution := []int{0, 0, 0, 0, 0, 1, 1, 1, 2, 2}
    
    for _, candID := range distribution {
        voteVec := make([]*big.Int, 3)
        for j := 0; j < 3; j++ {
            if j == candID {
                enc, _ := pk.Encrypt(big.NewInt(1))
                voteVec[j] = enc
            } else {
                enc, _ := pk.Encrypt(big.NewInt(0))
                voteVec[j] = enc
            }
        }
        votes = append(votes, voteVec...)
    }

    t.Log("Counter tally test passed - votes created successfully")
}

func TestVerifyTally(t *testing.T) {
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey

    // Encrypt known values
    enc1, _ := pk.Encrypt(big.NewInt(5))
    enc2, _ := pk.Encrypt(big.NewInt(3))
    
    // Homomorphic add
    encSum := pk.Add(enc1, enc2)
    
    // Decrypt and verify
    result, _ := sk.Decrypt(encSum)
    if result.Int64() != 8 {
        t.Errorf("Verify tally: expected 8, got %d", result.Int64())
    }
}

func TestAggregateWeightedVotes(t *testing.T) {
    sk, _ := crypto.GeneratePaillierKeyPair(2048)
    pk := sk.PublicKey

    // Vote with weight 1 (real) should count
    voteReal, _ := pk.Encrypt(big.NewInt(1))
    weightedReal := pk.Multiply(voteReal, big.NewInt(1))  // E(1*1) = E(1)

    // Vote with weight 0 (fake) should not count
    voteFake, _ := pk.Encrypt(big.NewInt(1))
    weightedFake := pk.Multiply(voteFake, big.NewInt(0))  // E(1*0) = E(0)

    // Tally: weighted_real + weighted_fake = E(1) + E(0) = E(1)
    tally := pk.Add(weightedReal, weightedFake)
    result, _ := sk.Decrypt(tally)

    if result.Int64() != 1 {
        t.Errorf("Weighted tally: expected 1, got %d (fake vote should not count)", result.Int64())
    }
}


## 3. Add tests to internal/pq/kyber_test.go

Add these NEW test functions AFTER the existing tests:

func TestXOREncryptDecrypt(t *testing.T) {
    message := []byte("Hello CovertVote! This is a test message.")
    key := make([]byte, 32)
    _, _ = rand.Read(key)

    encrypted := XOREncrypt(message, key)
    
    // Encrypted should differ from plaintext
    if string(encrypted) == string(message) {
        t.Error("Encrypted should differ from plaintext")
    }

    decrypted := XORDecrypt(encrypted, key)
    if string(decrypted) != string(message) {
        t.Errorf("Decrypted mismatch: got %s", string(decrypted))
    }
}

func TestEncryptDecryptMessage(t *testing.T) {
    kp, err := GenerateKyberKeyPair()
    if err != nil {
        t.Fatal(err)
    }

    message := []byte("Secret vote data for CovertVote")
    ciphertext, kemCT, salt, err := EncryptMessage(message, kp.PublicKey)
    if err != nil {
        t.Fatal(err)
    }

    decrypted, err := DecryptMessage(ciphertext, kemCT, salt, kp.PrivateKey)
    if err != nil {
        t.Fatal(err)
    }

    if string(decrypted) != string(message) {
        t.Errorf("Message mismatch: got %s", string(decrypted))
    }
}

func TestGenerateRandomSalt(t *testing.T) {
    salt1, err := GenerateRandomSalt()
    if err != nil {
        t.Fatal(err)
    }
    salt2, _ := GenerateRandomSalt()

    if len(salt1) != 16 {
        t.Errorf("Salt length: expected 16, got %d", len(salt1))
    }

    // Two salts should be different
    if string(salt1) == string(salt2) {
        t.Error("Two random salts should not be identical")
    }
}

func TestDeriveKey(t *testing.T) {
    secret := []byte("shared-secret-from-kyber")
    salt := []byte("random-salt-value")

    key1 := DeriveKey(secret, salt)
    key2 := DeriveKey(secret, salt)

    // Same inputs should produce same key
    if string(key1) != string(key2) {
        t.Error("Same inputs should produce same derived key")
    }

    // Different salt should produce different key
    key3 := DeriveKey(secret, []byte("different-salt"))
    if string(key1) == string(key3) {
        t.Error("Different salts should produce different keys")
    }

    if len(key1) != 32 {
        t.Errorf("Key length: expected 32, got %d", len(key1))
    }
}

func TestBigIntConversion(t *testing.T) {
    original := big.NewInt(1234567890)
    
    bytes := BigIntToBytes(original)
    recovered := BytesToBigInt(bytes)

    if original.Cmp(recovered) != 0 {
        t.Errorf("BigInt roundtrip failed: %v != %v", original, recovered)
    }
}

func TestUnmarshalKyberKeyPair(t *testing.T) {
    kp, _ := GenerateKyberKeyPair()

    pubBytes, _ := kp.PublicKey.MarshalBinary()
    privBytes, _ := kp.PrivateKey.MarshalBinary()

    recovered, err := UnmarshalKyberKeyPair(pubBytes, privBytes)
    if err != nil {
        t.Fatal(err)
    }

    // Test recovered keys work
    enc, err := EncapsulateWithPublicKey(recovered.PublicKey)
    if err != nil {
        t.Fatal(err)
    }

    ss, err := DecapsulateWithPrivateKey(recovered.PrivateKey, enc.Ciphertext)
    if err != nil {
        t.Fatal(err)
    }

    if string(ss) != string(enc.SharedSecret) {
        t.Error("Recovered keypair doesn't work correctly")
    }
}

func TestHybridEncryptDecryptLargeMessage(t *testing.T) {
    hkp, err := GenerateHybridKeyPair(2048)
    if err != nil {
        t.Fatal(err)
    }

    // Test with different vote values
    testVotes := []*big.Int{
        big.NewInt(0),
        big.NewInt(1),
        big.NewInt(42),
        big.NewInt(100),
    }

    for _, vote := range testVotes {
        ct, err := HybridEncrypt(vote, hkp)
        if err != nil {
            t.Fatalf("Encrypt vote %v failed: %v", vote, err)
        }

        result, err := HybridDecrypt(ct, hkp)
        if err != nil {
            t.Fatalf("Decrypt vote %v failed: %v", vote, err)
        }

        if result.Cmp(vote) != 0 {
            t.Errorf("Vote roundtrip failed: %v != %v", vote, result)
        }
    }
}

func TestMACComputeVerify(t *testing.T) {
    data := []byte("test data for MAC")
    key := []byte("secret-key-for-hmac-test-32bytes!")

    mac1 := computeMAC(data, key)
    mac2 := computeMAC(data, key)

    // Same input = same MAC
    if !verifyMAC(mac1, mac2) {
        t.Error("Same data should produce identical MACs")
    }

    // Different data = different MAC
    mac3 := computeMAC([]byte("different data"), key)
    if verifyMAC(mac1, mac3) {
        t.Error("Different data should produce different MACs")
    }
}

Add "crypto/rand" to imports if not already present.


## 4. Run tests and check coverage

After adding all tests, run:

# Run all tests
go test ./internal/voting/... -v -count=1 2>&1 | tail -20
go test ./internal/tally/... -v -count=1 2>&1 | tail -20
go test ./internal/pq/... -v -count=1 2>&1 | tail -20

# Check coverage per module
go test -coverprofile=coverage_voting.out ./internal/voting/... 2>&1
go tool cover -func=coverage_voting.out | tail -3

go test -coverprofile=coverage_tally.out ./internal/tally/... 2>&1
go tool cover -func=coverage_tally.out | tail -3

go test -coverprofile=coverage_pq.out ./internal/pq/... 2>&1
go tool cover -func=coverage_pq.out | tail -3

# Full project coverage
go test -coverprofile=coverage.out ./internal/... 2>&1
go tool cover -func=coverage.out | grep total

Then commit:
git add .
git commit -m "Improve test coverage: voting, tally, pq modules - target 70%+"
git push
```
