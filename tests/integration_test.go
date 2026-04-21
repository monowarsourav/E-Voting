package tests

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
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
	"github.com/gin-gonic/gin"
)

// TestCompleteVotingFlowWithPassword tests the full flow: register -> login -> vote
func TestCompleteVotingFlowWithPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(512)

	eligibleVoters := []string{"alice", "bob"}
	registrationSystem := voter.NewRegistrationSystem(pedersenParams, ringParams, 5, eligibleVoters, "election001")

	fingerprintProcessor := biometric.NewFingerprintProcessor()
	livenessDetector := biometric.NewLivenessDetector(0.5)

	regHandler := handlers.NewRegistrationHandler(registrationSystem, fingerprintProcessor, livenessDetector)

	// Create test election
	testElection := &voting.Election{
		ElectionID:  "election001",
		Title:       "Test Election",
		Description: "Test",
		Candidates: []*voting.Candidate{
			{ID: 1, Name: "Candidate A"},
			{ID: 2, Name: "Candidate B"},
		},
		StartTime: time.Now().Unix() - 3600,
		EndTime:   time.Now().Unix() + 3600,
		IsActive:  true,
	}

	voteCaster := voting.NewVoteCaster(paillierPK, ringParams, registrationSystem, testElection)
	voteHandler := handlers.NewVotingHandler(voteCaster, registrationSystem, fingerprintProcessor, livenessDetector)

	// Step 1: Register with password
	t.Run("Step 1: Register", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		regReq := models.RegistrationRequest{
			VoterID:  "alice",
			Password: "SecurePassword123",
		}
		jsonData, _ := json.Marshal(regReq)
		c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		regHandler.Register(c)

		if w.Code != http.StatusCreated {
			t.Fatalf("Registration failed with status %d: %s", w.Code, w.Body.String())
		}

		var regResp models.RegistrationResponse
		json.Unmarshal(w.Body.Bytes(), &regResp)

		if regResp.VoterID != "alice" {
			t.Errorf("Expected voter_id 'alice', got '%s'", regResp.VoterID)
		}
	})

	// Step 2: Login
	var authToken string
	t.Run("Step 2: Login", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		loginReq := models.LoginRequest{
			VoterID:  "alice",
			Password: "SecurePassword123",
		}
		jsonData, _ := json.Marshal(loginReq)
		c.Request = httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		regHandler.Login(c)

		if w.Code != http.StatusOK {
			t.Fatalf("Login failed with status %d: %s", w.Code, w.Body.String())
		}

		var loginResp models.LoginResponse
		json.Unmarshal(w.Body.Bytes(), &loginResp)

		authToken = loginResp.AuthToken
		if authToken == "" {
			t.Fatal("No auth_token received")
		}

		if loginResp.ExpiresIn != 86400 {
			t.Errorf("Expected expires_in 86400, got %d", loginResp.ExpiresIn)
		}
	})

	// Step 3: Cast vote WITH biometrics
	t.Run("Step 3: Cast Vote", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("voter_id", "alice")

		// Generate valid biometric data with proper entropy (minimum 500 bytes for liveness)
		fingerprintData := make([]byte, 512)
		for i := range fingerprintData {
			fingerprintData[i] = byte((i*37 + i/3*17 + 13) % 256) // Varied pseudo-random pattern
		}
		livenessData := make([]byte, 512)
		for i := range livenessData {
			livenessData[i] = byte((i*73 + i/5*41 + 29) % 256) // Varied pseudo-random pattern with good entropy
		}

		voteReq := models.VoteRequest{
			VoterID:         "alice",
			ElectionID:      "election001",
			CandidateID:     1,
			SMDCSlotIndex:   0,
			AuthToken:       authToken,
			FingerprintData: fingerprintData,
			LivenessData:    livenessData,
		}
		jsonData, _ := json.Marshal(voteReq)
		c.Request = httptest.NewRequest("POST", "/vote", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		voteHandler.CastVote(c)

		if w.Code != http.StatusCreated {
			t.Fatalf("Vote casting failed with status %d: %s", w.Code, w.Body.String())
		}

		var voteResp models.VoteResponse
		json.Unmarshal(w.Body.Bytes(), &voteResp)

		if voteResp.ReceiptID == "" {
			t.Error("No receipt_id received")
		}

		if voteResp.VoterID != "alice" {
			t.Errorf("Expected voter_id 'alice', got '%s'", voteResp.VoterID)
		}
	})
}

// TestCompleteVotingFlowWithBiometric tests registration with biometric
func TestCompleteVotingFlowWithBiometric(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(512)

	eligibleVoters := []string{"bob"}
	registrationSystem := voter.NewRegistrationSystem(pedersenParams, ringParams, 5, eligibleVoters, "election001")

	fingerprintProcessor := biometric.NewFingerprintProcessor()
	livenessDetector := biometric.NewLivenessDetector(0.5)

	regHandler := handlers.NewRegistrationHandler(registrationSystem, fingerprintProcessor, livenessDetector)

	// Create test election
	testElection := &voting.Election{
		ElectionID:  "election001",
		Title:       "Test Election",
		Description: "Test",
		Candidates: []*voting.Candidate{
			{ID: 1, Name: "Candidate A"},
			{ID: 2, Name: "Candidate B"},
		},
		StartTime: time.Now().Unix() - 3600,
		EndTime:   time.Now().Unix() + 3600,
		IsActive:  true,
	}

	voteCaster := voting.NewVoteCaster(paillierPK, ringParams, registrationSystem, testElection)
	voteHandler := handlers.NewVotingHandler(voteCaster, registrationSystem, fingerprintProcessor, livenessDetector)

	// Generate valid biometric data with proper entropy (minimum 500 bytes for liveness)
	fingerprintData := make([]byte, 512)
	for i := range fingerprintData {
		fingerprintData[i] = byte((i*37 + i/3*17 + 13) % 256) // Varied pseudo-random pattern
	}
	livenessData := make([]byte, 512)
	for i := range livenessData {
		livenessData[i] = byte((i*73 + i/5*41 + 29) % 256) // Varied pseudo-random pattern with good entropy
	}

	// Step 1: Register with biometric
	t.Run("Step 1: Register with Biometric", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		regReq := models.RegistrationRequest{
			VoterID:         "bob",
			FingerprintData: fingerprintData,
			LivenessData:    livenessData,
		}
		jsonData, _ := json.Marshal(regReq)
		c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		regHandler.Register(c)

		if w.Code != http.StatusCreated {
			t.Fatalf("Registration failed with status %d: %s", w.Code, w.Body.String())
		}
	})

	// Step 2: Login with fingerprint
	var authToken string
	t.Run("Step 2: Login with Fingerprint", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		loginReq := models.LoginRequest{
			VoterID:         "bob",
			FingerprintData: fingerprintData,
		}
		jsonData, _ := json.Marshal(loginReq)
		c.Request = httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		regHandler.Login(c)

		if w.Code != http.StatusOK {
			t.Fatalf("Login failed with status %d: %s", w.Code, w.Body.String())
		}

		var loginResp models.LoginResponse
		json.Unmarshal(w.Body.Bytes(), &loginResp)

		authToken = loginResp.AuthToken
		if authToken == "" {
			t.Fatal("No auth_token received")
		}
	})

	// Step 3: Cast vote
	t.Run("Step 3: Cast Vote", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("voter_id", "bob")

		voteReq := models.VoteRequest{
			VoterID:         "bob",
			ElectionID:      "election001",
			CandidateID:     2,
			SMDCSlotIndex:   0,
			AuthToken:       authToken,
			FingerprintData: fingerprintData,
			LivenessData:    livenessData,
		}
		jsonData, _ := json.Marshal(voteReq)
		c.Request = httptest.NewRequest("POST", "/vote", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		voteHandler.CastVote(c)

		if w.Code != http.StatusCreated {
			t.Fatalf("Vote casting failed with status %d: %s", w.Code, w.Body.String())
		}

		var voteResp models.VoteResponse
		json.Unmarshal(w.Body.Bytes(), &voteResp)

		if voteResp.ReceiptID == "" {
			t.Error("No receipt_id received")
		}
	})
}

// TestVotingWithoutBiometricsRejected tests that voting without biometrics fails
func TestVotingWithoutBiometricsRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	paillierSK, _ := crypto.GeneratePaillierKeyPair(2048)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(512)
	ringParams, _ := crypto.GenerateRingParams(512)

	eligibleVoters := []string{"charlie"}
	registrationSystem := voter.NewRegistrationSystem(pedersenParams, ringParams, 5, eligibleVoters, "election001")

	fingerprintProcessor := biometric.NewFingerprintProcessor()
	livenessDetector := biometric.NewLivenessDetector(0.5)

	regHandler := handlers.NewRegistrationHandler(registrationSystem, fingerprintProcessor, livenessDetector)

	// Create test election
	testElection := &voting.Election{
		ElectionID:  "election001",
		Title:       "Test Election",
		Description: "Test",
		Candidates: []*voting.Candidate{
			{ID: 1, Name: "Candidate A"},
			{ID: 2, Name: "Candidate B"},
		},
		StartTime: time.Now().Unix() - 3600,
		EndTime:   time.Now().Unix() + 3600,
		IsActive:  true,
	}

	voteCaster := voting.NewVoteCaster(paillierPK, ringParams, registrationSystem, testElection)
	voteHandler := handlers.NewVotingHandler(voteCaster, registrationSystem, fingerprintProcessor, livenessDetector)

	// Register with password
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	regReq := models.RegistrationRequest{
		VoterID:  "charlie",
		Password: "Password123",
	}
	jsonData, _ := json.Marshal(regReq)
	c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	regHandler.Register(c)

	// Try to vote WITHOUT biometrics
	t.Run("Vote without biometrics should fail", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("voter_id", "charlie")

		voteReq := models.VoteRequest{
			VoterID:       "charlie",
			ElectionID:    "election001",
			CandidateID:   1,
			SMDCSlotIndex: 0,
			AuthToken:     "test_token",
			// NO fingerprint or liveness data
		}
		jsonData, _ := json.Marshal(voteReq)
		c.Request = httptest.NewRequest("POST", "/vote", bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")

		voteHandler.CastVote(c)

		if w.Code == http.StatusCreated {
			t.Error("Vote should have been rejected without biometrics")
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if _, exists := response["error"]; !exists {
			t.Error("Expected error in response")
		}
	})
}
