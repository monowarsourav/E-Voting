package handlers

import (
	"encoding/hex"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/covertvote/e-voting/internal/voting"
	"github.com/gin-gonic/gin"
)

// safeErrorMessage maps internal error contexts to generic client-safe messages.
// Internal details are logged server-side; only the generic message is returned.
func safeErrorMessage(context string, err error) string {
	log.Printf("[ERROR] %s: %v", context, err)
	messages := map[string]string{
		"bind_json":               "The request body is malformed or missing required fields.",
		"fingerprint_processing":  "Fingerprint processing failed. Please try again.",
		"vote_casting":            "Vote could not be cast. Please try again later.",
		"election_bind_json":      "Invalid election request. Check the required fields.",
		"liveness_check":          "Liveness detection encountered an error. Please retry.",
	}
	if msg, ok := messages[context]; ok {
		return msg
	}
	return "An internal error occurred. Please try again later."
}

// VotingHandler handles vote casting
type VotingHandler struct {
	VoteCaster           *voting.VoteCaster
	RegistrationSystem   *voter.RegistrationSystem
	BiometricProcessor   *biometric.FingerprintProcessor
	LivenessDetector     *biometric.LivenessDetector
	Elections            map[string]*voting.Election
	mu                   sync.RWMutex
}

// NewVotingHandler creates a new voting handler
func NewVotingHandler(
	vc *voting.VoteCaster,
	rs *voter.RegistrationSystem,
	bp *biometric.FingerprintProcessor,
	ld *biometric.LivenessDetector,
) *VotingHandler {
	return &VotingHandler{
		VoteCaster:         vc,
		RegistrationSystem: rs,
		BiometricProcessor: bp,
		LivenessDetector:   ld,
		Elections:          make(map[string]*voting.Election),
	}
}

// CastVote handles POST /api/v1/vote
// @Summary Cast a vote
// @Description Cast an anonymous encrypted vote using ring signatures and homomorphic encryption
// @Tags Voting
// @Accept json
// @Produce json
// @Param vote body models.VoteRequest true "Vote details"
// @Success 201 {object} models.VoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /vote [post]
func (h *VotingHandler) CastVote(c *gin.Context) {
	var req models.VoteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: safeErrorMessage("bind_json", err),
		})
		return
	}

	// Validate voter ID matches authenticated session
	voterID, exists := c.Get("voter_id")
	if !exists || voterID != req.VoterID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Code:    http.StatusForbidden,
			Message: "voter ID mismatch",
		})
		return
	}

	// CRITICAL: Verify fingerprint + liveness for EVERY vote
	// Step 1: Get voter record
	voterRecord, err := h.RegistrationSystem.GetVoter(req.VoterID)
	if err != nil {
		log.Printf("[ERROR] voter lookup failed for %s: %v", req.VoterID, err)
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "voter_not_found",
			Code:    http.StatusUnauthorized,
			Message: "voter not registered",
		})
		return
	}

	// Step 2: Validate fingerprint data
	if len(req.FingerprintData) < 100 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_fingerprint",
			Code:    http.StatusBadRequest,
			Message: "fingerprint data required for voting",
		})
		return
	}

	// Step 3: Verify liveness (anti-spoofing)
	livenessResult, err := h.LivenessDetector.CheckLiveness(req.LivenessData)
	if err != nil {
		log.Printf("[ERROR] liveness check error for voter %s: %v", req.VoterID, err)
	}
	if err != nil || !livenessResult.IsLive {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "liveness_failed",
			Code:    http.StatusForbidden,
			Message: "liveness detection failed - possible spoofing attempt",
		})
		return
	}

	// Step 4: Verify fingerprint matches registered voter
	// If voter registered with fingerprint, verify it
	if len(voterRecord.FingerprintHash) > 0 {
		verified := h.BiometricProcessor.VerifyFingerprint(req.FingerprintData, voterRecord.FingerprintHash)
		if !verified {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "fingerprint_mismatch",
				Code:    http.StatusForbidden,
				Message: "fingerprint does not match registered voter",
			})
			return
		}
	} else {
		// Voter registered with password, but must provide fingerprint for voting
		// We'll process and temporarily store the fingerprint hash for this vote
		fingerprintData, err := h.BiometricProcessor.ProcessFingerprint(req.VoterID, req.FingerprintData, time.Now().Unix())
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "fingerprint_processing_failed",
				Code:    http.StatusBadRequest,
				Message: safeErrorMessage("fingerprint_processing", err),
			})
			return
		}
		// Store fingerprint hash for future votes if not already set
		if len(voterRecord.FingerprintHash) == 0 {
			voterRecord.FingerprintHash = fingerprintData.Hash
		}
	}

	// Step 5: All biometric checks passed, cast the vote
	// Lock the mutex to protect concurrent vote casting
	h.mu.Lock()
	receipt, err := h.VoteCaster.CastVote(req.VoterID, req.CandidateID, req.SMDCSlotIndex)
	h.mu.Unlock()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "vote_casting_failed",
			Code:    http.StatusBadRequest,
			Message: safeErrorMessage("vote_casting", err),
		})
		return
	}

	// Build response
	response := models.VoteResponse{
		ReceiptID:      receipt.ReceiptID,
		VoterID:        receipt.VoterID,
		ElectionID:     req.ElectionID,
		Timestamp:      receipt.Timestamp,
		BlockchainTxID: receipt.BlockchainTxID,
		KeyImage:       hex.EncodeToString(receipt.KeyImage.Bytes()),
		Message:        "Vote cast successfully. Keep your receipt ID for verification.",
	}

	c.JSON(http.StatusCreated, response)
}

// CreateElection handles POST /api/v1/elections (admin only)
// @Summary Create a new election
// @Description Create a new election with candidates and time range (admin only)
// @Tags Elections
// @Accept json
// @Produce json
// @Param election body models.ElectionRequest true "Election details"
// @Success 201 {object} models.ElectionResponse
// @Failure 400 {object} models.ErrorResponse
// @Security AdminAuth
// @Router /admin/elections [post]
func (h *VotingHandler) CreateElection(c *gin.Context) {
	var req models.ElectionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: safeErrorMessage("election_bind_json", err),
		})
		return
	}

	// Validate time range
	if req.StartTime >= req.EndTime {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_time_range",
			Code:    http.StatusBadRequest,
			Message: "start time must be before end time",
		})
		return
	}

	// Convert candidates
	var candidates []*voting.Candidate
	for _, c := range req.Candidates {
		candidates = append(candidates, &voting.Candidate{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			Party:       c.Party,
		})
	}

	// Create election
	election := &voting.Election{
		ElectionID:  generateElectionID(req.Title),
		Title:       req.Title,
		Description: req.Description,
		Candidates:  candidates,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		IsActive:    req.StartTime <= time.Now().Unix(),
	}

	h.mu.Lock()
	h.Elections[election.ElectionID] = election
	h.mu.Unlock()

	// Convert candidates back for response
	var respCandidates []models.Candidate
	for _, c := range candidates {
		respCandidates = append(respCandidates, models.Candidate{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			Party:       c.Party,
		})
	}

	response := models.ElectionResponse{
		ElectionID: election.ElectionID,
		Title:      election.Title,
		Candidates: respCandidates,
		StartTime:  election.StartTime,
		EndTime:    election.EndTime,
		IsActive:   election.IsActive,
		Message:    "Election created successfully",
	}

	c.JSON(http.StatusCreated, response)
}

// GetElections handles GET /api/v1/elections
// @Summary Get all elections
// @Description Retrieve list of all elections with their candidates and status
// @Tags Elections
// @Accept json
// @Produce json
// @Success 200 {object} models.ElectionListResponse
// @Router /elections [get]
func (h *VotingHandler) GetElections(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var elections []models.ElectionInfo
	for _, election := range h.Elections {
		var candidates []models.Candidate
		for _, c := range election.Candidates {
			candidates = append(candidates, models.Candidate{
				ID:          c.ID,
				Name:        c.Name,
				Description: c.Description,
				Party:       c.Party,
			})
		}

		elections = append(elections, models.ElectionInfo{
			ElectionID:  election.ElectionID,
			Title:       election.Title,
			Description: election.Description,
			Candidates:  candidates,
			StartTime:   election.StartTime,
			EndTime:     election.EndTime,
			IsActive:    election.IsActive,
			TotalVotes:  h.VoteCaster.GetVoteCount(),
		})
	}

	response := models.ElectionListResponse{
		Elections: elections,
		Total:     len(elections),
	}

	c.JSON(http.StatusOK, response)
}

// GetElection handles GET /api/v1/elections/:id
func (h *VotingHandler) GetElection(c *gin.Context) {
	electionID := c.Param("id")

	h.mu.RLock()
	election, exists := h.Elections[electionID]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "election_not_found",
			Code:    http.StatusNotFound,
			Message: "election not found",
		})
		return
	}

	var candidates []models.Candidate
	for _, c := range election.Candidates {
		candidates = append(candidates, models.Candidate{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			Party:       c.Party,
		})
	}

	info := models.ElectionInfo{
		ElectionID:  election.ElectionID,
		Title:       election.Title,
		Description: election.Description,
		Candidates:  candidates,
		StartTime:   election.StartTime,
		EndTime:     election.EndTime,
		IsActive:    election.IsActive,
		TotalVotes:  h.VoteCaster.GetVoteCount(),
	}

	c.JSON(http.StatusOK, info)
}

// VerifyVote handles POST /api/v1/verify-vote
func (h *VotingHandler) VerifyVote(c *gin.Context) {
	var req models.VerifyVoteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: safeErrorMessage("bind_json", err),
		})
		return
	}

	// TODO: Implement actual verification logic
	// For now, return mock response
	response := models.VerifyVoteResponse{
		Valid:          true,
		ElectionID:     "election001",
		Timestamp:      time.Now().Unix(),
		BlockchainTxID: "mock-tx-id",
		Message:        "Vote verified successfully",
	}

	c.JSON(http.StatusOK, response)
}

// generateElectionID generates a unique election ID
func generateElectionID(title string) string {
	return "election-" + hex.EncodeToString([]byte(title+time.Now().String()))[:16]
}
