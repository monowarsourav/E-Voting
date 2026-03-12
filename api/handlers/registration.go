package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/covertvote/e-voting/api/middleware"
	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/gin-gonic/gin"
)

// RegistrationHandler handles voter registration
type RegistrationHandler struct {
	RegistrationSystem *voter.RegistrationSystem
	BiometricProcessor *biometric.FingerprintProcessor
	LivenessDetector   *biometric.LivenessDetector
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(
	rs *voter.RegistrationSystem,
	bp *biometric.FingerprintProcessor,
	ld *biometric.LivenessDetector,
) *RegistrationHandler {
	return &RegistrationHandler{
		RegistrationSystem: rs,
		BiometricProcessor: bp,
		LivenessDetector:   ld,
	}
}

// Register handles POST /api/v1/register
// @Summary Register a new voter
// @Description Register with either biometric (fingerprint + liveness) OR password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param registration body models.RegistrationRequest true "Registration details"
// @Success 201 {object} models.RegistrationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Router /register [post]
// Supports EITHER biometric OR password registration
func (h *RegistrationHandler) Register(c *gin.Context) {
	var req models.RegistrationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	var fingerprintHash []byte
	var passwordHash []byte
	var voterID string

	// Check which authentication method is being used
	hasBiometric := len(req.FingerprintData) > 0
	hasPassword := len(req.Password) > 0

	if !hasBiometric && !hasPassword {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_auth_method",
			Code:    http.StatusBadRequest,
			Message: "must provide either fingerprint or password",
		})
		return
	}

	if hasBiometric && hasPassword {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "multiple_auth_methods",
			Code:    http.StatusBadRequest,
			Message: "provide only one authentication method",
		})
		return
	}

	// Option 1: Biometric registration
	if hasBiometric {
		// Validate fingerprint
		if len(req.FingerprintData) < 100 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_fingerprint",
				Code:    http.StatusBadRequest,
				Message: "fingerprint data too short",
			})
			return
		}

		// Process fingerprint
		fingerprintData, err := h.BiometricProcessor.ProcessFingerprint(req.VoterID, req.FingerprintData, time.Now().Unix())
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "fingerprint_processing_failed",
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			return
		}

		// Check liveness
		livenessResult, err := h.LivenessDetector.CheckLiveness(req.LivenessData)
		if err != nil || !livenessResult.IsLive {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "liveness_failed",
				Code:    http.StatusForbidden,
				Message: "liveness detection failed - possible spoofing attempt",
			})
			return
		}

		fingerprintHash = fingerprintData.Hash
		voterID = req.VoterID // Use requested voterID for consistency with password registration
	} else {
		// Option 2: Password registration
		if len(req.Password) < 8 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "weak_password",
				Code:    http.StatusBadRequest,
				Message: "password must be at least 8 characters",
			})
			return
		}

		// Hash password using SHA-256
		hash := sha256.Sum256([]byte(req.Password))
		passwordHash = hash[:]
		voterID = req.VoterID
	}

	// Register voter with appropriate auth method
	var voterRecord *voter.Voter
	var err error

	if hasBiometric {
		voterRecord, err = h.RegistrationSystem.RegisterVoter(voterID, fingerprintHash)
	} else {
		voterRecord, err = h.RegistrationSystem.RegisterVoterWithPassword(voterID, passwordHash)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "registration_failed",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Create session
	session := middleware.CreateSession(voterID)

	// Get public credential
	publicCred := voterRecord.SMDCCredential.GetPublicCredential()

	// Build response
	response := models.RegistrationResponse{
		VoterID:          voterID,
		PublicKey:        hex.EncodeToString(voterRecord.RingPublicKey.PublicKey.Bytes()),
		SMDCPublicCred:   hex.EncodeToString(publicCred.Commitments[0].Bytes()),
		MerkleRoot:       hex.EncodeToString(h.RegistrationSystem.GetMerkleRoot()),
		RegistrationTime: time.Now().Unix(),
		Message:          "Registration successful. Session token provided in Set-Cookie header.",
	}

	// Set session token in cookie
	c.SetCookie(
		"session_token",
		session.Token,
		int(24*time.Hour.Seconds()),
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusCreated, response)
}

// Login handles POST /api/v1/login
// @Summary Login voter
// @Description Login with either fingerprint OR password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param login body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /login [post]
// Supports EITHER fingerprint OR password login
func (h *RegistrationHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Get voter record
	voterRecord, err := h.RegistrationSystem.GetVoter(req.VoterID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "authentication_failed",
			Code:    http.StatusUnauthorized,
			Message: "invalid credentials",
		})
		return
	}

	// Check which authentication method is being used
	hasBiometric := len(req.FingerprintData) > 0
	hasPassword := len(req.Password) > 0

	if !hasBiometric && !hasPassword {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_auth_method",
			Code:    http.StatusBadRequest,
			Message: "must provide either fingerprint or password",
		})
		return
	}

	// Verify authentication
	authenticated := false

	if hasBiometric {
		// Verify fingerprint
		authenticated = h.BiometricProcessor.VerifyFingerprint(req.FingerprintData, voterRecord.FingerprintHash)
	} else {
		// Verify password
		hash := sha256.Sum256([]byte(req.Password))
		passwordHash := hash[:]

		// Compare hashes
		if len(passwordHash) == len(voterRecord.PasswordHash) {
			match := true
			for i := range passwordHash {
				if passwordHash[i] != voterRecord.PasswordHash[i] {
					match = false
					break
				}
			}
			authenticated = match
		}
	}

	if !authenticated {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "authentication_failed",
			Code:    http.StatusUnauthorized,
			Message: "invalid credentials",
		})
		return
	}

	// Create session
	session := middleware.CreateSession(req.VoterID)

	// Set session token in cookie
	c.SetCookie(
		"session_token",
		session.Token,
		int(24*time.Hour.Seconds()),
		"/",
		"",
		false,
		true,
	)

	// Return response
	response := models.LoginResponse{
		VoterID:   req.VoterID,
		AuthToken: session.Token,
		ExpiresIn: int64(24 * time.Hour.Seconds()),
		Message:   "Login successful",
	}

	c.JSON(http.StatusOK, response)
}

// GetVoterInfo handles GET /api/v1/voter/:id
func (h *RegistrationHandler) GetVoterInfo(c *gin.Context) {
	voterID := c.Param("id")

	if voterID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: "voter ID is required",
		})
		return
	}

	// Get voter
	voterRecord, err := h.RegistrationSystem.GetVoter(voterID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "voter_not_found",
			Code:    http.StatusNotFound,
			Message: "voter not found",
		})
		return
	}

	response := models.VoterInfoResponse{
		VoterID:          voterID,
		Registered:       true,
		RegistrationTime: time.Now().Unix(),
		HasVoted:         false, // TODO: Check actual vote status
	}

	// Only return info if requested by the voter themselves or admin
	requestedBy, _ := c.Get("voter_id")
	isAdmin, _ := c.Get("is_admin")

	if requestedBy != voterID && !isAdmin.(bool) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Code:    http.StatusForbidden,
			Message: "cannot access other voter's information",
		})
		return
	}

	_ = voterRecord // Use the variable
	c.JSON(http.StatusOK, response)
}

// VerifyEligibility handles POST /api/v1/verify-eligibility
func (h *RegistrationHandler) VerifyEligibility(c *gin.Context) {
	var req struct {
		VoterID string `json:"voter_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Get Merkle proof
	proof, err := h.RegistrationSystem.GetMerkleProof(req.VoterID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "voter_not_found",
			Code:    http.StatusNotFound,
			Message: "voter not registered",
		})
		return
	}

	// Verify proof
	merkleRoot := h.RegistrationSystem.GetMerkleRoot()
	valid := voter.VerifyProof(req.VoterID, proof, merkleRoot)

	c.JSON(http.StatusOK, gin.H{
		"voter_id":  req.VoterID,
		"eligible":  valid,
		"verified":  true,
		"timestamp": time.Now().Unix(),
	})
}

// GetAllVoters handles GET /api/v1/voters (admin only)
func (h *RegistrationHandler) GetAllVoters(c *gin.Context) {
	// Get all public keys (voter count)
	allKeys := h.RegistrationSystem.GetAllPublicKeys()

	c.JSON(http.StatusOK, gin.H{
		"total_voters": len(allKeys),
		"merkle_root":  hex.EncodeToString(h.RegistrationSystem.GetMerkleRoot()),
	})
}
