package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/covertvote/e-voting/api/models"
	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// HealthHandler handles health-check endpoints. A non-nil DB enables real
// readiness checks (ping) rather than always reporting ready.
type HealthHandler struct {
	Version string
	DB      *sql.DB
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(version string, db *sql.DB) *HealthHandler {
	return &HealthHandler{Version: version, DB: db}
}

// HealthCheck handles GET /health — process liveness. Always returns 200 if
// the process is running; useful for container orchestrators.
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:  "healthy",
		Version: h.Version,
		Uptime:  int64(time.Since(startTime).Seconds()),
	})
}

// ReadinessCheck handles GET /ready — reports whether the system is ready
// to serve traffic. Returns 503 if any dependency check fails so load
// balancers can route traffic away.
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	checks := gin.H{"crypto": "ok"}
	status := http.StatusOK
	overall := "ready"

	if h.DB != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.DB.PingContext(ctx); err != nil {
			checks["database"] = "unhealthy: " + err.Error()
			status = http.StatusServiceUnavailable
			overall = "not_ready"
		} else {
			checks["database"] = "ok"
		}
	} else {
		checks["database"] = "not_configured"
	}

	c.JSON(status, gin.H{"status": overall, "checks": checks})
}

// LivenessCheck handles GET /live — always alive if we can respond.
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}
