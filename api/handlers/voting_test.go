package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
	"github.com/gin-gonic/gin"
)

func setupTestVotingHandler() (*VotingHandler, *voter.RegistrationSystem) {
	// Initialize crypto params
	paillierSK, _ := crypto.GeneratePaillierKeyPair(512)
	paillierPK := paillierSK.PublicKey
	pedersenParams, _ := crypto.GeneratePedersenParams(256)
	ringParams, _ := crypto.GenerateRingParams(256)

	// Eligible voters
	eligibleVoters := []string{"voter001", "voter002", "test_voter"}

	// Registration system
	registrationSystem := voter.NewRegistrationSystem(
		pedersenParams,
		ringParams,
		5,
		eligibleVoters,
		"test-election-001",
	)

	// Biometric components
	fingerprintProcessor := biometric.NewFingerprintProcessor()
	livenessDetector := biometric.NewLivenessDetector(0.5)

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

	// Voting system
	voteCaster := voting.NewVoteCaster(
		paillierPK,
		ringParams,
		registrationSystem,
		testElection,
	)

	handler := NewVotingHandler(
		voteCaster,
		registrationSystem,
		fingerprintProcessor,
		livenessDetector,
	)

	// Add test election to handler
	handler.Elections["election001"] = testElection

	return handler, registrationSystem
}

func TestCastVoteWithBiometrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, regSystem := setupTestVotingHandler()

	// Register voter with password
	fingerprintData := make([]byte, 512)
	for i := range fingerprintData {
		fingerprintData[i] = byte((i * 37 + i/3*17 + 13) % 256) // Varied pseudo-random pattern
	}

	_, err := regSystem.RegisterVoterWithPassword("voter001", []byte("password123"))
	if err != nil {
		t.Fatalf("Failed to register voter: %v", err)
	}

	// Generate valid biometric data with proper entropy (minimum 500 bytes)
	livenessData := make([]byte, 512)
	for i := range livenessData {
		livenessData[i] = byte((i * 73 + i/5*41 + 29) % 256) // Varied pseudo-random pattern with good entropy
	}

	tests := []struct {
		name            string
		voterID         string
		fingerprintData []byte
		livenessData    []byte
		candidateID     int
		expectedStatus  int
		expectError     bool
		errorContains   string
	}{
		{
			name:            "Valid vote with biometrics",
			voterID:         "voter001",
			fingerprintData: fingerprintData,
			livenessData:    livenessData,
			candidateID:     1,
			expectedStatus:  http.StatusCreated,
			expectError:     false,
		},
		{
			name:            "Missing fingerprint data",
			voterID:         "voter001",
			fingerprintData: []byte{1, 2, 3}, // Too short
			livenessData:    livenessData,
			candidateID:     1,
			expectedStatus:  http.StatusBadRequest,
			expectError:     true,
			errorContains:   "invalid_fingerprint",
		},
		{
			name:            "Missing liveness data",
			voterID:         "voter001",
			fingerprintData: fingerprintData,
			livenessData:    []byte{}, // Empty
			candidateID:     1,
			expectedStatus:  http.StatusForbidden,
			expectError:     true,
			errorContains:   "liveness_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set voter_id in context (from auth middleware)
			c.Set("voter_id", tt.voterID)

			voteReq := models.VoteRequest{
				VoterID:         tt.voterID,
				ElectionID:      "election001",
				CandidateID:     tt.candidateID,
				SMDCSlotIndex:   0,
				AuthToken:       "test_token",
				FingerprintData: tt.fingerprintData,
				LivenessData:    tt.livenessData,
			}
			jsonData, _ := json.Marshal(voteReq)
			c.Request = httptest.NewRequest("POST", "/vote", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.CastVote(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				if _, exists := response["error"]; !exists {
					t.Error("Expected error in response")
				}
				if tt.errorContains != "" {
					if errCode, ok := response["error"].(string); !ok || errCode != tt.errorContains {
						t.Errorf("Expected error containing '%s', got '%v'",
							tt.errorContains, response["error"])
					}
				}
			} else {
				if _, exists := response["receipt_id"]; !exists {
					t.Error("Expected receipt_id in response")
				}
			}
		})
	}
}

func TestCastVotePasswordUserRequiresBiometrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, regSystem := setupTestVotingHandler()

	// Register voter with PASSWORD only
	_, err := regSystem.RegisterVoterWithPassword("voter002", []byte("password123"))
	if err != nil {
		t.Fatalf("Failed to register voter: %v", err)
	}

	// Try to vote WITH biometrics
	fingerprintData := make([]byte, 512)
	for i := range fingerprintData {
		fingerprintData[i] = byte((i * 41 + i/7*23 + 19) % 256) // Different varied pseudo-random pattern
	}
	livenessData := make([]byte, 512)
	for i := range livenessData {
		livenessData[i] = byte((i * 73 + i/5*41 + 29) % 256) // Varied pseudo-random pattern with good entropy
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("voter_id", "voter002")

	voteReq := models.VoteRequest{
		VoterID:         "voter002",
		ElectionID:      "election001",
		CandidateID:     1,
		SMDCSlotIndex:   0,
		AuthToken:       "test_token",
		FingerprintData: fingerprintData,
		LivenessData:    livenessData,
	}
	jsonData, _ := json.Marshal(voteReq)
	c.Request = httptest.NewRequest("POST", "/vote", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CastVote(c)

	// Should succeed - password users can provide biometrics for voting
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, exists := response["receipt_id"]; !exists {
		t.Error("Expected receipt_id in successful vote")
	}
}

func TestCastVoteUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, regSystem := setupTestVotingHandler()

	_, err := regSystem.RegisterVoterWithPassword("voter001", []byte("password123"))
	if err != nil {
		t.Fatalf("Failed to register voter: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// No voter_id in context (auth middleware not run)
	fingerprintData := make([]byte, 256)
	livenessData := make([]byte, 128)

	voteReq := models.VoteRequest{
		VoterID:         "voter001",
		ElectionID:      "election001",
		CandidateID:     1,
		SMDCSlotIndex:   0,
		AuthToken:       "test_token",
		FingerprintData: fingerprintData,
		LivenessData:    livenessData,
	}
	jsonData, _ := json.Marshal(voteReq)
	c.Request = httptest.NewRequest("POST", "/vote", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CastVote(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestGetElections(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupTestVotingHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/elections", nil)

	handler.GetElections(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.ElectionListResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Total != 1 {
		t.Errorf("Expected 1 election, got %d", response.Total)
	}

	if len(response.Elections) != 1 {
		t.Errorf("Expected 1 election in list, got %d", len(response.Elections))
	}

	if response.Elections[0].ElectionID != "election001" {
		t.Errorf("Expected election_id 'election001', got '%s'",
			response.Elections[0].ElectionID)
	}
}

func TestGetElection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := setupTestVotingHandler()

	tests := []struct {
		name           string
		electionID     string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid election",
			electionID:     "election001",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Non-existent election",
			electionID:     "invalid_id",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.electionID}}
			c.Request = httptest.NewRequest("GET", "/elections/"+tt.electionID, nil)

			handler.GetElection(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
