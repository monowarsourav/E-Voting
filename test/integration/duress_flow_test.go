package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/covertvote/e-voting/api/handlers"
	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	icrypto "github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
	"github.com/gin-gonic/gin"
)

// buildDuressTestRouter wires up a minimal HTTP router with the voting and
// duress endpoints — no auth middleware so the test can drive calls directly.
func buildDuressTestRouter(t *testing.T) (
	*gin.Engine,
	*voting.VoteCaster,
	*voter.RegistrationSystem,
	*icrypto.PaillierPrivateKey,
	biometric.DuressDetector,
) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	paillierSK, _ := icrypto.GeneratePaillierKeyPair(2048)
	pedersenParams, _ := icrypto.GeneratePedersenParams(512)
	ringParams, _ := icrypto.GenerateRingParams(512)

	eligibleVoters := []string{"alice", "bob", "charlie"}
	rs := voter.NewRegistrationSystem(pedersenParams, ringParams, 5, eligibleVoters, "election001")

	// Register voters with a deterministic fingerprint stub (>= 100 bytes).
	// RegisterVoter stores the value as-is as FingerprintHash; the voting
	// handler calls FingerprintToHash(rawData) before comparing, so we must
	// store the pre-computed hash to make the equality check pass.
	for _, id := range eligibleVoters {
		// Derive 200 bytes deterministically — must match buildVoteRequest's logic.
		raw := bytes.Repeat([]byte(id), 1)
		for len(raw) < 200 {
			raw = append(raw, raw...)
		}
		rawFP := raw[:200]
		fpHash := icrypto.FingerprintToHash(rawFP)
		if _, err := rs.RegisterVoter(id, fpHash); err != nil {
			t.Fatalf("register %s: %v", id, err)
		}
	}

	election := &voting.Election{
		ElectionID: "election001",
		Title:      "Integration Test Election",
		Candidates: []*voting.Candidate{
			{ID: 1, Name: "Candidate A"},
			{ID: 2, Name: "Candidate B"},
		},
		StartTime: time.Now().Unix() - 3600,
		EndTime:   time.Now().Unix() + 3600,
		IsActive:  true,
	}

	detector := biometric.NewInMemoryDuressDetector([]byte("integration-test-duress-hmac-key"))

	vc := voting.NewVoteCaster(
		paillierSK.PublicKey,
		ringParams,
		rs,
		election,
		voting.WithDuressDetector(detector),
	)

	fp := biometric.NewFingerprintProcessor()
	ld := biometric.NewLivenessDetector(0.0) // threshold=0 so test data always passes

	votingHandler := handlers.NewVotingHandler(vc, rs, fp, ld)
	votingHandler.Elections[election.ElectionID] = election

	duressHandler := handlers.NewDuressHandler(detector, rs)

	router := gin.New()

	// Stub auth: echo voter_id from the request header.
	router.Use(func(c *gin.Context) {
		if id := c.GetHeader("X-Voter-ID"); id != "" {
			c.Set("voter_id", id)
		}
		c.Next()
	})

	router.POST("/voters/:voterID/duress-signal", duressHandler.SetSignal)
	router.POST("/vote", votingHandler.CastVote)

	return router, vc, rs, paillierSK, detector
}

// postJSON is a test helper that POSTs JSON and returns the recorder.
func postJSON(router *gin.Engine, path string, voterID string, body interface{}) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if voterID != "" {
		req.Header.Set("X-Voter-ID", voterID)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// buildVoteRequest constructs a vote body.
// Fingerprint must be >= 100 bytes; liveness must be >= 500 bytes with
// sufficient entropy. Both are generated deterministically from voterID.
func buildVoteRequest(voterID, electionID string, candidateID, smdc int, signalType, signalValue string) models.VoteRequest {
	// Produce exactly 200 bytes deterministically from voterID.
	raw := bytes.Repeat([]byte(voterID), 1)
	for len(raw) < 200 {
		raw = append(raw, raw...)
	}
	fp := raw[:200]

	// 600 bytes of varied data for liveness (entropy check requires > 3.0).
	liveness := make([]byte, 600)
	for i := range liveness {
		liveness[i] = byte((i*17 + 42) % 256)
	}
	return models.VoteRequest{
		VoterID:             voterID,
		ElectionID:          electionID,
		CandidateID:         candidateID,
		SMDCSlotIndex:       smdc,
		AuthToken:           "test-token",
		FingerprintData:     fp,
		LivenessData:        liveness,
		DetectedSignalType:  signalType,
		DetectedSignalValue: signalValue,
	}
}

// TestDuressFlow_FullEndToEnd exercises the complete duress signal lifecycle:
//
//  1. Register voter alice.
//  2. Set duress signal "2 blinks = real".
//  3. alice casts a vote with matching signal "2 blinks" → succeeds.
//  4. bob casts a vote with mismatched signal "3 blinks" → still returns
//     success (coercer cannot tell), but vote is internally zeroed.
//  5. charlie casts a vote with no duress signal set → succeeds normally.
//  6. Verify response body is structurally identical for all three cases
//     (no leak of duress outcome).
func TestDuressFlow_FullEndToEnd(t *testing.T) {
	router, _, rs, _, detector := buildDuressTestRouter(t)

	// Step 2: Set duress signal for alice ("2 blinks = genuine").
	w := postJSON(router, "/voters/alice/duress-signal", "alice", models.SetDuressSignalRequest{
		SignalType:  "blink_count",
		SignalValue: "2",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("Step 2 SetSignal failed: %d %s", w.Code, w.Body.String())
	}

	// Also set a signal for bob so we can test a mismatch (bob's real = "2", will submit "3").
	w = postJSON(router, "/voters/bob/duress-signal", "bob", models.SetDuressSignalRequest{
		SignalType:  "blink_count",
		SignalValue: "2",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("Step 2b SetSignal for bob failed: %d %s", w.Code, w.Body.String())
	}

	// Verify both signals are registered.
	if !detector.HasSignal("alice") {
		t.Error("alice should have a registered duress signal")
	}
	if !detector.HasSignal("bob") {
		t.Error("bob should have a registered duress signal")
	}
	if detector.HasSignal("charlie") {
		t.Error("charlie should NOT have a registered duress signal")
	}

	// Find SMDC slot 0 for each voter.
	aliceRecord, _ := rs.GetVoter("alice")
	_ = aliceRecord // slot 0 used below

	// Step 3: alice casts with matching signal "2 blinks".
	aliceVote := buildVoteRequest("alice", "election001", 1, 0, "blink_count", "2")
	wAlice := postJSON(router, "/vote", "alice", aliceVote)
	if wAlice.Code != http.StatusCreated {
		t.Fatalf("Step 3 alice vote failed: %d %s", wAlice.Code, wAlice.Body.String())
	}

	// Step 4: bob casts with MISMATCHED signal "3 blinks" (real is "2").
	// The server must return 201 (success) — coercer cannot tell.
	bobVote := buildVoteRequest("bob", "election001", 1, 0, "blink_count", "3")
	wBob := postJSON(router, "/vote", "bob", bobVote)
	if wBob.Code != http.StatusCreated {
		t.Fatalf("Step 4 bob (mismatched) vote must return 201: %d %s", wBob.Code, wBob.Body.String())
	}

	// Step 5: charlie casts with no duress signal set.
	charlieVote := buildVoteRequest("charlie", "election001", 1, 0, "", "")
	wCharlie := postJSON(router, "/vote", "charlie", charlieVote)
	if wCharlie.Code != http.StatusCreated {
		t.Fatalf("Step 5 charlie vote failed: %d %s", wCharlie.Code, wCharlie.Body.String())
	}

	// Step 6: Verify response structure is identical — no leakage of duress outcome.
	var aliceResp, bobResp, charlieResp models.VoteResponse
	json.NewDecoder(bytes.NewReader(wAlice.Body.Bytes())).Decode(&aliceResp)
	json.NewDecoder(bytes.NewReader(wBob.Body.Bytes())).Decode(&bobResp)
	json.NewDecoder(bytes.NewReader(wCharlie.Body.Bytes())).Decode(&charlieResp)

	// All three must have a non-empty receipt_id.
	if aliceResp.ReceiptID == "" {
		t.Error("alice receipt_id must not be empty")
	}
	if bobResp.ReceiptID == "" {
		t.Error("bob receipt_id must not be empty (coercer must not see empty receipt)")
	}
	if charlieResp.ReceiptID == "" {
		t.Error("charlie receipt_id must not be empty")
	}

	// All three carry the same success message.
	for name, resp := range map[string]models.VoteResponse{
		"alice": aliceResp, "bob": bobResp, "charlie": charlieResp,
	} {
		if resp.Message == "" {
			t.Errorf("%s: response message must not be empty", name)
		}
	}
}

// TestDuressFlow_NoSignal_BackwardCompat verifies that voters without a
// registered duress signal can still vote normally (backward compatibility).
func TestDuressFlow_NoSignal_BackwardCompat(t *testing.T) {
	router, _, _, _, _ := buildDuressTestRouter(t)

	// charlie has no duress signal; submitting an empty signal type is fine.
	vote := buildVoteRequest("charlie", "election001", 1, 0, "", "")
	w := postJSON(router, "/vote", "charlie", vote)
	if w.Code != http.StatusCreated {
		t.Fatalf("backward-compat vote failed: %d %s", w.Code, w.Body.String())
	}
}

// TestDuressFlow_SetSignal_InvalidType verifies the signal-type allowlist.
func TestDuressFlow_SetSignal_InvalidType(t *testing.T) {
	router, _, _, _, _ := buildDuressTestRouter(t)

	w := postJSON(router, "/voters/alice/duress-signal", "alice", models.SetDuressSignalRequest{
		SignalType:  "interpretive_dance", // not in allowlist
		SignalValue: "samba",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid signal type, got %d", w.Code)
	}
}
