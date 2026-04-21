package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/gin-gonic/gin"
)

// Server B for SA² aggregation
type AggregatorServerB struct {
	ServerID   string
	Port       string
	Aggregator *sa2.Aggregator
	PublicKey  *crypto.PaillierPublicKey
	Shares     map[string][]*big.Int // electionID -> shares
	mu         sync.RWMutex
}

func main() {
	// Get configuration from environment
	serverID := getEnv("SERVER_ID", "B")
	port := getEnv("SERVER_PORT", "8082")
	host := getEnv("SERVER_HOST", "0.0.0.0")

	log.Printf("Starting SA² Aggregator Server %s on %s:%s", serverID, host, port)

	// Create server
	server := &AggregatorServerB{
		ServerID: serverID,
		Port:     port,
		Shares:   make(map[string][]*big.Int),
	}

	// Initialize Paillier public key (in production, load from config)
	sk, err := crypto.GeneratePaillierKeyPair(2048)
	if err != nil {
		log.Fatalf("Failed to generate Paillier keys: %v", err)
	}
	server.PublicKey = sk.PublicKey

	// Initialize aggregator
	server.Aggregator = sa2.NewAggregator("ServerB", server.PublicKey)

	// Load API keys from environment
	apiKey := os.Getenv("SA2_API_KEY")
	adminKey := os.Getenv("SA2_ADMIN_KEY")
	if apiKey == "" {
		log.Fatal("SA2_API_KEY environment variable is required")
	}
	if adminKey == "" {
		log.Fatal("SA2_ADMIN_KEY environment variable is required")
	}

	// Setup router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Public routes (no auth required)
	router.GET("/health", server.healthCheck)

	// Authenticated routes
	authenticated := router.Group("/api/v1/sa2")
	authenticated.Use(apiKeyAuthMiddleware(apiKey))
	{
		authenticated.POST("/submit-share", server.submitShare)
		authenticated.POST("/aggregate", server.aggregate)
		authenticated.GET("/result/:electionId", server.getResult)

		// Admin-only route: requires both API key and admin key
		admin := authenticated.Group("")
		admin.Use(adminKeyAuthMiddleware(adminKey))
		{
			admin.POST("/clear/:electionId", server.clearShares)
		}
	}

	// Start server
	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("✓ Server B ready at http://%s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func (s *AggregatorServerB) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"server_id": s.ServerID,
		"role":      "SA² Aggregator Server B",
		"timestamp": time.Now().Unix(),
	})
}

func (s *AggregatorServerB) submitShare(c *gin.Context) {
	var req struct {
		ElectionID string `json:"election_id" binding:"required"`
		VoterID    string `json:"voter_id" binding:"required"`
		ShareB     string `json:"share_b" binding:"required"` // hex encoded big.Int
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse share
	share := new(big.Int)
	share.SetString(req.ShareB, 16)

	// Store share
	s.mu.Lock()
	if s.Shares[req.ElectionID] == nil {
		s.Shares[req.ElectionID] = make([]*big.Int, 0)
	}
	s.Shares[req.ElectionID] = append(s.Shares[req.ElectionID], share)
	s.mu.Unlock()

	log.Printf("Server B: Received share from voter %s for election %s", req.VoterID, req.ElectionID)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"server_id":    s.ServerID,
		"election_id":  req.ElectionID,
		"total_shares": len(s.Shares[req.ElectionID]),
	})
}

func (s *AggregatorServerB) aggregate(c *gin.Context) {
	var req struct {
		ElectionID string `json:"election_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.RLock()
	shares, exists := s.Shares[req.ElectionID]
	s.mu.RUnlock()

	if !exists || len(shares) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no shares found for this election"})
		return
	}

	// Aggregate shares
	log.Printf("Server B: Aggregating %d shares for election %s", len(shares), req.ElectionID)
	result := s.Aggregator.AggregateShares(shares)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"server_id":        s.ServerID,
		"election_id":      req.ElectionID,
		"aggregated_value": result.Value.Text(16),
		"num_shares":       len(shares),
	})
}

func (s *AggregatorServerB) getResult(c *gin.Context) {
	electionID := c.Param("electionId")

	s.mu.RLock()
	shares, exists := s.Shares[electionID]
	s.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "no data for this election"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id":   s.ServerID,
		"election_id": electionID,
		"num_shares":  len(shares),
	})
}

func (s *AggregatorServerB) clearShares(c *gin.Context) {
	electionID := c.Param("electionId")

	s.mu.Lock()
	delete(s.Shares, electionID)
	s.mu.Unlock()

	log.Printf("Server B: Cleared shares for election %s", electionID)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"server_id":   s.ServerID,
		"election_id": electionID,
		"message":     "shares cleared",
	})
}

// apiKeyAuthMiddleware validates the Authorization: Bearer <token> header
// against the SA2_API_KEY environment variable using constant-time comparison.
func apiKeyAuthMiddleware(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing Authorization header",
			})
			return
		}

		// Expect "Bearer <token>" format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid Authorization header format, expected 'Bearer <token>'",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "invalid API key",
			})
			return
		}

		c.Next()
	}
}

// adminKeyAuthMiddleware provides an additional layer of authentication for
// destructive operations. It checks the X-Admin-Key header against SA2_ADMIN_KEY.
func adminKeyAuthMiddleware(expectedAdminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminKey := c.GetHeader("X-Admin-Key")
		if adminKey == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "missing X-Admin-Key header, admin access required",
			})
			return
		}

		if subtle.ConstantTimeCompare([]byte(adminKey), []byte(expectedAdminKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "invalid admin key",
			})
			return
		}

		c.Next()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func init() {
	// Ensure JSON encoder doesn't escape HTML
	gin.DefaultWriter = os.Stdout
	_ = json.Marshal
}
