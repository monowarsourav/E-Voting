package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/gin-gonic/gin"
)

func setupTestRegistrationHandler() *RegistrationHandler {
	// Initialize crypto params
	pedersenParams, _ := crypto.GeneratePedersenParams(256)
	ringParams, _ := crypto.GenerateRingParams(256)

	// Eligible voters
	eligibleVoters := []string{"voter001", "voter002", "test_voter", "alice", "bob"}

	// Registration system
	registrationSystem := voter.NewRegistrationSystem(
		pedersenParams,
		ringParams,
		5, // SMDC slots
		eligibleVoters,
		"test-election-001",
	)

	// Biometric components
	fingerprintProcessor := biometric.NewFingerprintProcessor()
	livenessDetector := biometric.NewLivenessDetector(0.5)

	return NewRegistrationHandler(registrationSystem, fingerprintProcessor, livenessDetector)
}

func TestRegisterWithPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestRegistrationHandler()

	tests := []struct {
		name           string
		voterID        string
		password       string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid registration",
			voterID:        "voter001",
			password:       "Password123",
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name:           "Duplicate registration",
			voterID:        "voter001",
			password:       "Password123",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "Ineligible voter",
			voterID:        "invalid_voter",
			password:       "Password123",
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:           "Weak password",
			voterID:        "voter002",
			password:       "short",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			reqBody := models.RegistrationRequest{
				VoterID:  tt.voterID,
				Password: tt.password,
			}
			jsonData, _ := json.Marshal(reqBody)
			c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Register(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				if _, exists := response["error"]; !exists {
					t.Error("Expected error in response")
				}
			} else {
				if voterID, exists := response["voter_id"]; !exists || voterID != tt.voterID {
					t.Errorf("Expected voter_id %s in response", tt.voterID)
				}
			}
		})
	}
}

func TestRegisterWithBiometric(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestRegistrationHandler()

	// Generate valid biometric data with proper entropy (minimum 500 bytes for liveness)
	fingerprintData := make([]byte, 512)
	for i := range fingerprintData {
		fingerprintData[i] = byte((i * 37 + i/3*17 + 13) % 256) // Varied pseudo-random pattern
	}
	livenessData := make([]byte, 512)
	for i := range livenessData {
		livenessData[i] = byte((i * 73 + i/5*41 + 29) % 256) // Varied pseudo-random pattern with good entropy
	}

	tests := []struct {
		name            string
		voterID         string
		fingerprintData []byte
		livenessData    []byte
		expectedStatus  int
		expectError     bool
	}{
		{
			name:            "Valid biometric registration",
			voterID:         "alice",
			fingerprintData: fingerprintData,
			livenessData:    livenessData,
			expectedStatus:  http.StatusCreated,
			expectError:     false,
		},
		{
			name:            "Invalid fingerprint (too short)",
			voterID:         "bob",
			fingerprintData: []byte{1, 2, 3},
			livenessData:    livenessData,
			expectedStatus:  http.StatusBadRequest,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			reqBody := models.RegistrationRequest{
				VoterID:         tt.voterID,
				FingerprintData: tt.fingerprintData,
				LivenessData:    tt.livenessData,
			}
			jsonData, _ := json.Marshal(reqBody)
			c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Register(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s, LivenessDataLen: %d",
					tt.expectedStatus, w.Code, w.Body.String(), len(tt.livenessData))
			}
		})
	}
}

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestRegistrationHandler()

	// Pre-register a voter
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	regReq := models.RegistrationRequest{
		VoterID:  "test_voter",
		Password: "TestPassword123",
	}
	jsonData, _ := json.Marshal(regReq)
	c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Register(c)

	tests := []struct {
		name           string
		voterID        string
		password       string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid login",
			voterID:        "test_voter",
			password:       "TestPassword123",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Wrong password",
			voterID:        "test_voter",
			password:       "WrongPassword",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "Non-existent voter",
			voterID:        "nonexistent",
			password:       "Password123",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			loginReq := models.LoginRequest{
				VoterID:  tt.voterID,
				Password: tt.password,
			}
			jsonData, _ := json.Marshal(loginReq)
			c.Request = httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Login(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if tt.expectError {
				if _, exists := response["error"]; !exists {
					t.Error("Expected error in response")
				}
			} else {
				if _, exists := response["auth_token"]; !exists {
					t.Error("Expected auth_token in response")
				}
			}
		})
	}
}

func TestRegisterMissingAuthMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestRegistrationHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// No password or fingerprint
	reqBody := models.RegistrationRequest{
		VoterID: "voter001",
	}
	jsonData, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "missing_auth_method" {
		t.Errorf("Expected error 'missing_auth_method', got '%s'", response["error"])
	}
}

func TestRegisterMultipleAuthMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestRegistrationHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	fingerprintData := make([]byte, 256)
	livenessData := make([]byte, 128)

	// Both password and fingerprint
	reqBody := models.RegistrationRequest{
		VoterID:         "voter001",
		Password:        "Password123",
		FingerprintData: fingerprintData,
		LivenessData:    livenessData,
	}
	jsonData, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "multiple_auth_methods" {
		t.Errorf("Expected error 'multiple_auth_methods', got '%s'", response["error"])
	}
}
