package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/covertvote/e-voting/api/models"
	"github.com/covertvote/e-voting/internal/biometric"
	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voter"
	"github.com/gin-gonic/gin"
)

// setupDuressTest constructs a registered voter, a detector, and a handler
// wired together — the minimum needed to exercise the endpoint.
func setupDuressTest(t *testing.T) (*DuressHandler, *voter.RegistrationSystem, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	pp, _ := crypto.GeneratePedersenParams(512)
	rp, _ := crypto.GenerateRingParams(512)
	rs := voter.NewRegistrationSystem(pp, rp, 5, []string{"alice", "bob"}, "election001")

	// Register alice so the handler can find her.
	fp := make([]byte, 200)
	for i := range fp {
		fp[i] = byte(i)
	}
	if _, err := rs.RegisterVoter("alice", fp); err != nil {
		t.Fatalf("failed to register alice: %v", err)
	}

	detector := biometric.NewInMemoryDuressDetector([]byte("handler-test-hmac-key-32bytes!!"))
	handler := NewDuressHandler(detector, rs)

	router := gin.New()
	// Simulate auth middleware by setting voter_id in context.
	router.Use(func(c *gin.Context) {
		c.Set("voter_id", c.Param("voterID"))
		c.Next()
	})
	router.POST("/api/v1/voters/:voterID/duress-signal", handler.SetSignal)

	return handler, rs, router
}

func postDuressSignal(router *gin.Engine, voterID, signalType, signalValue string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(models.SetDuressSignalRequest{
		SignalType:  signalType,
		SignalValue: signalValue,
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/voters/"+voterID+"/duress-signal",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestSetupDuressEndpoint_Success(t *testing.T) {
	_, _, router := setupDuressTest(t)

	w := postDuressSignal(router, "alice", "blink_count", "2")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.SetDuressSignalResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "registered" {
		t.Errorf("expected status=registered, got %q", resp.Status)
	}
	if resp.SignalID == "" {
		t.Error("signal_id should not be empty")
	}
	if resp.SetAt == 0 {
		t.Error("set_at should not be zero")
	}
}

func TestSetupDuressEndpoint_InvalidVoter(t *testing.T) {
	_, _, router := setupDuressTest(t)

	w := postDuressSignal(router, "nonexistent-voter", "blink_count", "2")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetupDuressEndpoint_InvalidSignalType(t *testing.T) {
	_, _, router := setupDuressTest(t)

	w := postDuressSignal(router, "alice", "eye_roll", "2") // not a valid type

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetupDuressEndpoint_InvalidSignalValue(t *testing.T) {
	_, _, router := setupDuressTest(t)

	// blink_count must be 1-5; 99 is out of range.
	w := postDuressSignal(router, "alice", "blink_count", "99")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-range blink_count, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetupDuressEndpoint_Replace(t *testing.T) {
	_, _, router := setupDuressTest(t)

	// First registration.
	w1 := postDuressSignal(router, "alice", "blink_count", "2")
	if w1.Code != http.StatusOK {
		t.Fatalf("first registration failed: %d %s", w1.Code, w1.Body.String())
	}

	// Second registration (replace) with a different value.
	w2 := postDuressSignal(router, "alice", "blink_count", "4")
	if w2.Code != http.StatusOK {
		t.Fatalf("replacement registration failed: %d %s", w2.Code, w2.Body.String())
	}

	var r1, r2 models.SetDuressSignalResponse
	json.NewDecoder(bytes.NewReader(w1.Body.Bytes())).Decode(&r1)
	json.NewDecoder(bytes.NewReader(w2.Body.Bytes())).Decode(&r2)

	// The signal_id (first 8 bytes of HMAC) must differ between the two registrations.
	if r1.SignalID == r2.SignalID {
		t.Error("signal_id should change after replacing the duress signal")
	}
}
