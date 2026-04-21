package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/tally"
	"github.com/covertvote/e-voting/internal/voting"
	"github.com/gin-gonic/gin"
)

// TallyHandler handles vote tallying
type TallyHandler struct {
	Counter    *tally.Counter
	VoteCaster *voting.VoteCaster
	Elections  map[string]*voting.Election // shared reference to election state

	// Result cache to prevent re-tallying on every GetResults call (DoS mitigation)
	cacheMu     sync.RWMutex
	resultCache map[string]*cachedResult
}

// cachedResult stores a tally result with an expiration time.
type cachedResult struct {
	response  models.TallyResponse
	expiresAt time.Time
}

const resultCacheTTL = 30 * time.Second

// NewTallyHandler creates a new tally handler
func NewTallyHandler(counter *tally.Counter, vc *voting.VoteCaster, elections map[string]*voting.Election) *TallyHandler {
	return &TallyHandler{
		Counter:     counter,
		VoteCaster:  vc,
		Elections:   elections,
		resultCache: make(map[string]*cachedResult),
	}
}

// TallyVotes handles POST /api/v1/tally (admin only)
func (h *TallyHandler) TallyVotes(c *gin.Context) {
	var req models.TallyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[ERROR] tally bind_json: %v", err)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: "Invalid tally request. Check required fields.",
		})
		return
	}

	// Get all vote shares
	voteShares := h.VoteCaster.GetAllVoteShares()

	if len(voteShares) == 0 {
		c.JSON(http.StatusOK, models.TallyResponse{
			ElectionID:       req.ElectionID,
			CandidateTallies: make(map[int]int64),
			TotalVotes:       0,
			TallyTime:        time.Now().Unix(),
			Verified:         true,
		})
		return
	}

	// Tally votes
	result, err := h.Counter.TallyVotes(voteShares, req.ElectionID)
	if err != nil {
		log.Printf("[ERROR] tally failed for election %s: %v", req.ElectionID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "tally_failed",
			Code:    http.StatusInternalServerError,
			Message: "Vote tallying failed. Please try again later.",
		})
		return
	}

	// Convert result
	candidateTallies := make(map[int]int64)
	for candidateID, tally := range result.CandidateTallies {
		candidateTallies[candidateID] = tally.Int64()
	}

	response := models.TallyResponse{
		ElectionID:       req.ElectionID,
		CandidateTallies: candidateTallies,
		TotalVotes:       int64(result.TotalVotes),
		TallyTime:        time.Now().Unix(),
		Verified:         true,
	}

	// Invalidate cache after admin re-tally so GetResults picks up the new result
	h.cacheMu.Lock()
	h.resultCache[req.ElectionID] = &cachedResult{
		response:  response,
		expiresAt: time.Now().Add(resultCacheTTL),
	}
	h.cacheMu.Unlock()

	c.JSON(http.StatusOK, response)
}

// GetResults handles GET /api/v1/results/:electionId
// Only returns results for completed elections (election EndTime has passed).
// Results are cached to prevent expensive re-tallying on every request (DoS mitigation).
func (h *TallyHandler) GetResults(c *gin.Context) {
	electionID := c.Param("electionId")

	if electionID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Code:    http.StatusBadRequest,
			Message: "election ID is required",
		})
		return
	}

	// Verify election exists and is completed
	election, exists := h.Elections[electionID]
	if !exists {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "election_not_found",
			Code:    http.StatusNotFound,
			Message: "election not found",
		})
		return
	}

	now := time.Now().Unix()
	if election.EndTime > now {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "election_not_completed",
			Code:    http.StatusForbidden,
			Message: "results are only available after the election has ended",
		})
		return
	}

	// Check the result cache first
	h.cacheMu.RLock()
	cached, cacheHit := h.resultCache[electionID]
	h.cacheMu.RUnlock()

	if cacheHit && time.Now().Before(cached.expiresAt) {
		c.JSON(http.StatusOK, cached.response)
		return
	}

	// Cache miss or expired -- perform the tally
	voteShares := h.VoteCaster.GetAllVoteShares()

	if len(voteShares) == 0 {
		response := models.TallyResponse{
			ElectionID:       electionID,
			CandidateTallies: make(map[int]int64),
			TotalVotes:       0,
			TallyTime:        time.Now().Unix(),
			Verified:         true,
		}
		h.cacheMu.Lock()
		h.resultCache[electionID] = &cachedResult{
			response:  response,
			expiresAt: time.Now().Add(resultCacheTTL),
		}
		h.cacheMu.Unlock()
		c.JSON(http.StatusOK, response)
		return
	}

	// Tally votes
	result, err := h.Counter.TallyVotes(voteShares, electionID)
	if err != nil {
		log.Printf("[ERROR] GetResults tally failed for election %s: %v", electionID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "tally_failed",
			Code:    http.StatusInternalServerError,
			Message: "Unable to compute results. Please try again later.",
		})
		return
	}

	// Convert result
	candidateTallies := make(map[int]int64)
	for candidateID, tally := range result.CandidateTallies {
		candidateTallies[candidateID] = tally.Int64()
	}

	response := models.TallyResponse{
		ElectionID:       electionID,
		CandidateTallies: candidateTallies,
		TotalVotes:       int64(result.TotalVotes),
		TallyTime:        time.Now().Unix(),
		Verified:         true,
	}

	// Cache the result
	h.cacheMu.Lock()
	h.resultCache[electionID] = &cachedResult{
		response:  response,
		expiresAt: time.Now().Add(resultCacheTTL),
	}
	h.cacheMu.Unlock()

	c.JSON(http.StatusOK, response)
}

// GetVoteCount handles GET /api/v1/vote-count
func (h *TallyHandler) GetVoteCount(c *gin.Context) {
	count := h.VoteCaster.GetVoteCount()

	c.JSON(http.StatusOK, gin.H{
		"total_votes": count,
		"timestamp":   time.Now().Unix(),
	})
}
