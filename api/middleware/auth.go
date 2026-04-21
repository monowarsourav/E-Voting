package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// adminTokens holds the set of valid admin tokens loaded from configuration.
// Use SetAdminTokens to populate at application startup. Comparisons use
// constant-time equality to avoid timing side-channels.
var (
	adminTokensMu sync.RWMutex
	adminTokens   []string
)

// SetAdminTokens configures the list of valid admin tokens. Call once during
// application startup from configuration (never hardcode). Passing an empty
// slice disables admin access entirely.
func SetAdminTokens(tokens []string) {
	adminTokensMu.Lock()
	defer adminTokensMu.Unlock()
	adminTokens = make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t != "" {
			adminTokens = append(adminTokens, t)
		}
	}
}

// isValidAdminToken checks the provided token against the configured admin
// tokens using constant-time comparison.
func isValidAdminToken(token string) bool {
	adminTokensMu.RLock()
	defer adminTokensMu.RUnlock()
	tokenBytes := []byte(token)
	for _, configured := range adminTokens {
		if subtle.ConstantTimeCompare(tokenBytes, []byte(configured)) == 1 {
			return true
		}
	}
	return false
}

// SessionStore provides concurrency-safe access to voter sessions.
//
// For production deployments, sessions should be persisted via a backend
// implementing PersistentSessionStore so that server restarts do not
// invalidate live sessions. The in-memory map serves as a fast path; when
// Backend is non-nil, reads fall through to it on miss and writes are
// replicated to it.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	Backend  PersistentSessionStore
}

// PersistentSessionStore is the optional durable backend for sessions.
type PersistentSessionStore interface {
	Get(token string) (*Session, error)
	Set(session *Session) error
	Delete(token string) error
	DeleteExpired(now int64) error
	ErrNotFound() error
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
	sess, ok := s.sessions[token]
	s.mu.RUnlock()
	if ok {
		return sess, true
	}
	if s.Backend == nil {
		return nil, false
	}
	sess, err := s.Backend.Get(token)
	if err != nil || sess == nil {
		return nil, false
	}
	s.mu.Lock()
	s.sessions[token] = sess
	s.mu.Unlock()
	return sess, true
}

// Set stores a session under the given token.
func (s *SessionStore) Set(token string, sess *Session) {
	s.mu.Lock()
	s.sessions[token] = sess
	s.mu.Unlock()
	if s.Backend != nil {
		_ = s.Backend.Set(sess)
	}
}

// Delete removes a session by token.
func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
	if s.Backend != nil {
		_ = s.Backend.Delete(token)
	}
}

// CleanupExpired removes all sessions whose ExpiresAt is in the past.
func (s *SessionStore) CleanupExpired() {
	now := time.Now().Unix()
	s.mu.Lock()
	for token, session := range s.sessions {
		if now > session.ExpiresAt {
			delete(s.sessions, token)
		}
	}
	s.mu.Unlock()
	if s.Backend != nil {
		_ = s.Backend.DeleteExpired(now)
	}
}

// VoterSessions is the package-level session store used by the middleware and handlers.
var VoterSessions = NewSessionStore()

// Session represents an authenticated session.
type Session struct {
	VoterID   string
	Token     string
	ExpiresAt int64
	CreatedAt int64
}

// AuthMiddleware validates authentication tokens.
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

		session, exists := VoterSessions.Get(token)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}

		if time.Now().Unix() > session.ExpiresAt {
			VoterSessions.Delete(token)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "token expired",
			})
			c.Abort()
			return
		}

		c.Set("voter_id", session.VoterID)
		// Explicitly mark non-admin requests so downstream handlers can rely
		// on the key always being present (prevents nil type-assertion panics).
		c.Set("is_admin", false)
		c.Next()
	}
}

// AdminAuthMiddleware validates admin tokens.
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
		if !isValidAdminToken(token) {
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

// sessionTokenBytes is the entropy used for session tokens. 32 bytes of
// crypto/rand output provides 256 bits of entropy — far beyond guessing range.
const sessionTokenBytes = 32

// CreateSession creates a new voter session with a cryptographically random token.
func CreateSession(voterID string) (*Session, error) {
	buf := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(buf)

	session := &Session{
		VoterID:   voterID,
		Token:     token,
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	VoterSessions.Set(token, session)
	return session, nil
}

// CleanupExpiredSessions removes expired sessions.
func CleanupExpiredSessions() {
	VoterSessions.CleanupExpired()
}
