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
	"github.com/gin-gonic/gin"
)

// duressRateLimiter enforces a per-voter ceiling on SetSignal calls using a
// sliding-window counter. This guards against signal-spam that could DoS the
// system or flood the audit log. The limit (5 calls/hour per voter) is tight
// enough to stop abuse while allowing a voter to update their pattern several
// times during a single session.
type duressRateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time // voter ID → timestamps of recent calls
	window  time.Duration
	max     int
}

func newDuressRateLimiter() *duressRateLimiter {
	return &duressRateLimiter{
		buckets: make(map[string][]time.Time),
		window:  time.Hour,
		max:     5,
	}
}

// allow returns true when voterID has not exceeded the call budget. Stale
// entries outside the sliding window are pruned on each check.
func (r *duressRateLimiter) allow(voterID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	times := r.buckets[voterID]
	// Drop entries outside the window.
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	r.buckets[voterID] = valid

	if len(valid) >= r.max {
		return false
	}
	r.buckets[voterID] = append(r.buckets[voterID], now)
	return true
}

// DuressHandler handles behavioral duress signal registration for voters.
// It exposes endpoints that let an authenticated voter store or remove a
// secret behavioral pattern (e.g. "2 blinks"). During vote casting the
// vote caster verifies the pattern; a mismatch silently zeros the vote weight
// so a coercer cannot distinguish a coerced vote from a genuine one.
type DuressHandler struct {
	Detector           biometric.DuressDetector
	RegistrationSystem *voter.RegistrationSystem
	rateLimiter        *duressRateLimiter
}

// NewDuressHandler creates a new DuressHandler.
func NewDuressHandler(d biometric.DuressDetector, rs *voter.RegistrationSystem) *DuressHandler {
	return &DuressHandler{
		Detector:           d,
		RegistrationSystem: rs,
		rateLimiter:        newDuressRateLimiter(),
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
// @Failure      403      {object}  models.ErrorResponse
// @Failure      404      {object}  models.ErrorResponse
// @Failure      429      {object}  models.ErrorResponse
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

	// Per-voter rate limit: 5 SetSignal calls per hour.
	// This is tighter than the global StrictLimiter and keyed on voter ID
	// (not IP) so a NAT'd coercer cannot exhaust a victim's budget.
	if !h.rateLimiter.allow(voterID) {
		c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
			Error:   "rate_limit_exceeded",
			Code:    http.StatusTooManyRequests,
			Message: "too many signal updates — try again later",
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

// RemoveSignal handles DELETE /api/v1/voters/:voterID/duress-signal
//
// @Summary      Remove a behavioral duress signal
// @Description  Clears the voter's registered duress signal. Idempotent:
//               returns 204 even when no signal was registered.
// @Tags         Voters
// @Produce      json
// @Param        voterID  path  string  true  "Voter ID"
// @Success      204
// @Failure      403  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /voters/{voterID}/duress-signal [delete]
func (h *DuressHandler) RemoveSignal(c *gin.Context) {
	voterID := c.Param("voterID")

	if authedID, exists := c.Get("voter_id"); exists && authedID != voterID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Code:    http.StatusForbidden,
			Message: "voter ID mismatch",
		})
		return
	}

	if err := h.Detector.RemoveSignal(voterID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Code:    http.StatusInternalServerError,
			Message: "failed to remove signal",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
