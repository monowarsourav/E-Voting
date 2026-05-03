package integration

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/covertvote/e-voting/api/handlers"
	"github.com/covertvote/e-voting/api/middleware"
	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	icrypto "github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/gin-gonic/gin"
)

// generateToken creates a random hex token and registers it in VoterSessions.
func generateToken(t *testing.T, voterID string, expiresAt int64) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	token := hex.EncodeToString(b)
	middleware.VoterSessions.Set(token, &middleware.Session{
		VoterID:   voterID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().Unix(),
	})
	t.Cleanup(func() { middleware.VoterSessions.Delete(token) })
	return token
}

// buildAuthTestRouter creates a router with real AuthMiddleware.
func buildAuthTestRouter(t *testing.T) (*gin.Engine, biometric.DuressDetector) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	paillierSK, _ := icrypto.GeneratePaillierKeyPair(2048)
	_ = paillierSK
	pedersenParams, _ := icrypto.GeneratePedersenParams(512)
	ringParams, _ := icrypto.GenerateRingParams(512)

	eligibleVoters := []string{"alice", "bob"}
	rs := voter.NewRegistrationSystem(pedersenParams, ringParams, 5, eligibleVoters, "election001")
	for _, id := range eligibleVoters {
		_, _ = rs.RegisterVoterWithPassword(id, []byte("password123"))
	}

	detector := biometric.NewInMemoryDuressDetector([]byte("auth-test-hmac-key-32-bytes-ok!"))
	duressHandler := handlers.NewDuressHandler(detector, rs)

	router := gin.New()
	auth := router.Group("")
	auth.Use(middleware.AuthMiddleware())
	auth.POST("/voters/:voterID/duress-signal", duressHandler.SetSignal)
	auth.DELETE("/voters/:voterID/duress-signal", duressHandler.RemoveSignal)

	return router, detector
}

// postJSONAuth sends a POST with an Authorization Bearer header.
func postJSONAuth(router *gin.Engine, path, token string, body interface{}) *httptest.ResponseRecorder {
	return postJSONWithToken(router, http.MethodPost, path, token, body)
}

func deleteWithToken(router *gin.Engine, path, token string) *httptest.ResponseRecorder {
	return postJSONWithToken(router, http.MethodDelete, path, token, nil)
}

func postJSONWithToken(router *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var buf *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		buf = bytes.NewReader(data)
	} else {
		buf = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestDuressEndpoint_ValidJWT_Success verifies that a valid session token
// allows a voter to set their own duress signal.
func TestDuressEndpoint_ValidJWT_Success(t *testing.T) {
	router, _ := buildAuthTestRouter(t)
	token := generateToken(t, "alice", time.Now().Add(time.Hour).Unix())

	w := postJSONAuth(router, "/voters/alice/duress-signal", token, models.SetDuressSignalRequest{
		SignalType:  "blink_count",
		SignalValue: "2",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDuressEndpoint_NoJWT_Returns401 verifies that requests without an
// Authorization header are rejected with 401.
func TestDuressEndpoint_NoJWT_Returns401(t *testing.T) {
	router, _ := buildAuthTestRouter(t)

	w := postJSONAuth(router, "/voters/alice/duress-signal", "", models.SetDuressSignalRequest{
		SignalType:  "blink_count",
		SignalValue: "2",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no token, got %d", w.Code)
	}
}

// TestDuressEndpoint_WrongVoterJWT_Returns403 verifies that voter A's session
// cannot be used to modify voter B's duress signal.
func TestDuressEndpoint_WrongVoterJWT_Returns403(t *testing.T) {
	router, _ := buildAuthTestRouter(t)
	// alice's token trying to set bob's signal.
	aliceToken := generateToken(t, "alice", time.Now().Add(time.Hour).Unix())

	w := postJSONAuth(router, "/voters/bob/duress-signal", aliceToken, models.SetDuressSignalRequest{
		SignalType:  "blink_count",
		SignalValue: "3",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-voter signal set, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDuressEndpoint_ExpiredJWT_Returns401 verifies that an expired session
// token is rejected.
func TestDuressEndpoint_ExpiredJWT_Returns401(t *testing.T) {
	router, _ := buildAuthTestRouter(t)
	// ExpiresAt is 1 second in the past.
	expiredToken := generateToken(t, "alice", time.Now().Add(-time.Second).Unix())

	w := postJSONAuth(router, "/voters/alice/duress-signal", expiredToken, models.SetDuressSignalRequest{
		SignalType:  "blink_count",
		SignalValue: "2",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDuressEndpoint_RemoveSignal_ValidAuth verifies the DELETE endpoint
// works with a valid session and is idempotent.
func TestDuressEndpoint_RemoveSignal_ValidAuth(t *testing.T) {
	router, detector := buildAuthTestRouter(t)
	token := generateToken(t, "alice", time.Now().Add(time.Hour).Unix())

	// Set a signal first.
	w := postJSONAuth(router, "/voters/alice/duress-signal", token, models.SetDuressSignalRequest{
		SignalType:  "blink_count",
		SignalValue: "2",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("SetSignal: %d %s", w.Code, w.Body.String())
	}
	if !detector.HasSignal("alice") {
		t.Fatal("signal should exist before removal")
	}

	// Delete it.
	w = deleteWithToken(router, "/voters/alice/duress-signal", token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("RemoveSignal: expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if detector.HasSignal("alice") {
		t.Error("signal should be gone after removal")
	}

	// Idempotent second delete.
	w = deleteWithToken(router, "/voters/alice/duress-signal", token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("RemoveSignal (idempotent): expected 204, got %d", w.Code)
	}
}

// TestDuressEndpoint_RemoveSignal_WrongVoter_Returns403 ensures a voter cannot
// remove another voter's signal via the DELETE endpoint.
func TestDuressEndpoint_RemoveSignal_WrongVoter_Returns403(t *testing.T) {
	router, _ := buildAuthTestRouter(t)
	aliceToken := generateToken(t, "alice", time.Now().Add(time.Hour).Unix())

	w := deleteWithToken(router, "/voters/bob/duress-signal", aliceToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-voter removal, got %d", w.Code)
	}
}
