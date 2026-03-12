package handlers

import (
	"net/http"
	"time"

	"github.com/covertvote/e-voting/api/models"
	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// HealthHandler handles health checks
type HealthHandler struct {
	Version string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{
		Version: version,
	}
}

// HealthCheck handles GET /health
// @Summary Health check
// @Description Returns the health status of the API server
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	uptime := time.Since(startTime).Seconds()

	response := models.HealthResponse{
		Status:  "healthy",
		Version: h.Version,
		Uptime:  int64(uptime),
	}

	c.JSON(http.StatusOK, response)
}

// ReadinessCheck handles GET /ready
// @Summary Readiness check
// @Description Returns whether the system is ready to accept requests
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /ready [get]
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// Check if system is ready
	// For now, always return ready
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"database":    "ok",
			"crypto":      "ok",
			"blockchain":  "pending",
		},
	})
}

// LivenessCheck handles GET /live
// @Summary Liveness check
// @Description Returns whether the system is alive
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /live [get]
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}
