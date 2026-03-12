package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminTokens stores valid admin tokens (in production, use secure storage)
var AdminTokens = map[string]bool{
	"admin-token-example-12345": true,
}

// SessionStore provides concurrency-safe access to voter sessions.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionStore creates an initialised SessionStore.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// Get retrieves a session by token. Returns nil, false if not found.
func (s *SessionStore) Get(token string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	return sess, ok
}

// Set stores a session under the given token.
func (s *SessionStore) Set(token string, sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = sess
}

// Delete removes a session by token.
func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// CleanupExpired removes all sessions whose ExpiresAt is in the past.
func (s *SessionStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	for token, session := range s.sessions {
		if now > session.ExpiresAt {
			delete(s.sessions, token)
		}
	}
}

// VoterSessions is the package-level session store used by the middleware and handlers.
var VoterSessions = NewSessionStore()

// Session represents an authenticated session
type Session struct {
	VoterID   string
	Token     string
	ExpiresAt int64
	CreatedAt int64
}

// AuthMiddleware validates authentication tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid authorization format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate session token
		session, exists := VoterSessions.Get(token)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// Check expiration
		if time.Now().Unix() > session.ExpiresAt {
			VoterSessions.Delete(token)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "token expired",
			})
			c.Abort()
			return
		}

		// Set voter ID in context
		c.Set("voter_id", session.VoterID)
		c.Next()
	}
}

// AdminAuthMiddleware validates admin tokens
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing admin authorization",
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid authorization format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate admin token
		if !AdminTokens[token] {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "invalid admin credentials",
			})
			c.Abort()
			return
		}

		c.Set("is_admin", true)
		c.Next()
	}
}

// CreateSession creates a new voter session
func CreateSession(voterID string) *Session {
	// Generate session token
	hash := sha256.Sum256([]byte(voterID + time.Now().String()))
	token := hex.EncodeToString(hash[:])

	session := &Session{
		VoterID:   voterID,
		Token:     token,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(), // 24 hour session
	}

	VoterSessions.Set(token, session)
	return session
}

// CleanupExpiredSessions removes expired sessions.
// Delegates to the SessionStore's CleanupExpired method.
func CleanupExpiredSessions() {
	VoterSessions.CleanupExpired()
}

// CORSMiddleware is now in middleware/cors.go to avoid duplication
