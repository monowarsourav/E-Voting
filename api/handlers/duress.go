package handlers

import (
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/gin-gonic/gin"
)

// DuressHandler handles behavioral duress signal registration for voters.
// It exposes a single endpoint that lets an authenticated voter store a
// secret behavioral pattern (e.g. "2 blinks"). During vote casting the
// vote caster verifies the pattern; a mismatch silently zeros the vote weight
// so a coercer cannot distinguish a coerced vote from a genuine one.
type DuressHandler struct {
	Detector           biometric.DuressDetector
	RegistrationSystem *voter.RegistrationSystem
}

// NewDuressHandler creates a new DuressHandler.
func NewDuressHandler(d biometric.DuressDetector, rs *voter.RegistrationSystem) *DuressHandler {
	return &DuressHandler{
		Detector:           d,
		RegistrationSystem: rs,
	}
}

// SetSignal handles POST /api/v1/voters/:voterID/duress-signal
//
// @Summary      Register a behavioral duress signal
// @Description  Sets (or replaces) a secret behavioral pattern for a voter.
//               During a coerced vote the voter presents the WRONG pattern;
//               the server silently zeros the vote weight without informing
//               the coercer.
// @Tags         Voters
// @Accept       json
// @Produce      json
// @Param        voterID  path      string                          true  "Voter ID"
// @Param        body     body      models.SetDuressSignalRequest   true  "Signal details"
// @Success      200      {object}  models.SetDuressSignalResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      404      {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /voters/{voterID}/duress-signal [post]
func (h *DuressHandler) SetSignal(c *gin.Context) {
	voterID := c.Param("voterID")

	// The authenticated voter_id (set by AuthMiddleware) must match the path param.
	// This prevents one voter from setting another voter's duress signal.
	if authedID, exists := c.Get("voter_id"); exists && authedID != voterID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Code:    http.StatusForbidden,
			Message: "voter ID mismatch",
		})
		return
	}

	// Verify voter is registered.
	if _, err := h.RegistrationSystem.GetVoter(voterID); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "voter_not_found",
			Code:    http.StatusNotFound,
			Message: "voter not registered",
		})
		return
	}

	var req models.SetDuressSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: safeErrorMessage("bind_json", err),
		})
		return
	}

	hash, err := h.Detector.SetSignal(voterID, req.SignalType, req.SignalValue)
	if err != nil {
		log.Printf("[ERROR] duress SetSignal for voter %s: %v", voterID, err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_signal",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Return the first 8 bytes of the HMAC as a short, opaque signal_id.
	// This lets the client confirm registration without exposing the full hash.
	signalID := hex.EncodeToString(hash[:8])

	c.JSON(http.StatusOK, models.SetDuressSignalResponse{
		Status:   "registered",
		SignalID: signalID,
		SetAt:    time.Now().Unix(),
	})
}
