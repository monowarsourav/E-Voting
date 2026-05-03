// Package biometric provides biometric verification primitives.
// This file implements the behavioral duress signal subsystem — a coercion-
// resistance mechanism that lets a voter register a secret behavioral pattern
// (e.g. "2 blinks") during registration and silently zero their vote weight
// if that pattern is absent or wrong during coerced voting.
//
// Security properties:
//   - HMAC-SHA256 with a server-side key: the server can recompute the hash
//     for any presented signal and compare in constant time.
//   - Coercer cannot distinguish a weight-0 (duress) vote from a weight-1
//     (real) vote: the API response is identical in both cases.
//   - Hash stored server-side only: the raw signal value is never persisted.
package biometric

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// Signal type constants define the supported behavioral duress patterns.
const (
	SignalTypeBlinkCount   = "blink_count"   // integer 1-5: number of deliberate blinks
	SignalTypeHeadTilt     = "head_tilt"     // string: direction/angle label
	SignalTypeLongPress    = "long_press"    // integer 1-5: seconds of press duration
	SignalTypeTimeDelay    = "time_delay"    // string: timing pattern label
	SignalTypeVoiceCommand = "voice_command" // string: spoken command label
)

var validSignalTypes = map[string]bool{
	SignalTypeBlinkCount:   true,
	SignalTypeHeadTilt:     true,
	SignalTypeLongPress:    true,
	SignalTypeTimeDelay:    true,
	SignalTypeVoiceCommand: true,
}

// ErrInvalidSignalType is returned when an unrecognised signal type is presented.
var ErrInvalidSignalType = errors.New("invalid signal type")

// ErrInvalidSignalValue is returned when the signal value fails type-specific validation.
var ErrInvalidSignalValue = errors.New("invalid signal value")

// DuressSignal holds the registered duress signal for a single voter.
// The raw SignalValue is stored only for diagnostic purposes on the server;
// only the HMAC Hash is used for verification comparisons.
type DuressSignal struct {
	SignalType  string
	SignalValue string
	Hash        []byte // HMAC-SHA256(serverKey, signalType+":"+signalValue)
	Timestamp   int64  // Unix seconds when the signal was registered
}

// DetectedSignal carries the client-reported behavioral signal submitted with
// a vote-cast request. A nil DetectedSignal means the voter did not include
// a behavioral signal (backward-compatible path — no duress check is applied).
type DetectedSignal struct {
	SignalType  string
	SignalValue string
}

// DuressDetector is the storage and verification interface for behavioral
// duress signals. All methods must be safe for concurrent use.
type DuressDetector interface {
	// SetSignal registers (or replaces) the duress signal for voterID.
	// Returns the HMAC hash of the accepted signal so callers can store a
	// receipt without persisting the raw value.
	SetSignal(voterID, signalType, signalValue string) ([]byte, error)

	// VerifySignal returns true when the voter has a registered signal AND the
	// presented (signalType, detectedValue) pair produces the same HMAC hash.
	// Returns true (not an error) when no signal is registered — the no-signal
	// case is treated as "always matches" for backward compatibility.
	VerifySignal(voterID, signalType, detectedValue string) (bool, error)

	// HasSignal reports whether voterID currently has a registered duress signal.
	HasSignal(voterID string) bool
}

// InMemoryDuressDetector is the default DuressDetector implementation backed
// by a plain Go map. Signals survive only for the lifetime of the process.
// The SQL migration 008_add_duress_signal.sql defines the columns for a
// future database-backed implementation that can be swapped in without
// changing call sites.
type InMemoryDuressDetector struct {
	mu      sync.RWMutex
	signals map[string]*DuressSignal
	hmacKey []byte // server-side secret; must never be exposed to clients
}

// NewInMemoryDuressDetector constructs a detector with the provided HMAC key.
// The key should be loaded from the DURESS_HMAC_KEY environment variable.
// If the key is empty a hard-coded development fallback is used — the caller
// should log a warning in that case.
func NewInMemoryDuressDetector(hmacKey []byte) *InMemoryDuressDetector {
	if len(hmacKey) == 0 {
		// NOT safe for production: replaced by the caller when DURESS_HMAC_KEY is set.
		hmacKey = []byte("dev-duress-hmac-key-replace-in-production")
	}
	return &InMemoryDuressDetector{
		signals: make(map[string]*DuressSignal),
		hmacKey: hmacKey,
	}
}

// SetSignal validates the (signalType, signalValue) pair and stores the HMAC
// of the signal for voterID, replacing any previously registered signal.
func (d *InMemoryDuressDetector) SetSignal(voterID, signalType, signalValue string) ([]byte, error) {
	if err := validateSignalType(signalType); err != nil {
		return nil, err
	}
	if err := validateSignalValue(signalType, signalValue); err != nil {
		return nil, err
	}

	hash := d.computeHMAC(signalType, signalValue)

	d.mu.Lock()
	d.signals[voterID] = &DuressSignal{
		SignalType:  signalType,
		SignalValue: signalValue,
		Hash:        hash,
		Timestamp:   time.Now().Unix(),
	}
	d.mu.Unlock()

	return hash, nil
}

// VerifySignal compares the HMAC of the presented signal against the stored
// hash using constant-time comparison to prevent timing side-channel leaks.
// Returns (true, nil) when no signal is registered (backward-compatible).
func (d *InMemoryDuressDetector) VerifySignal(voterID, signalType, detectedValue string) (bool, error) {
	d.mu.RLock()
	stored, ok := d.signals[voterID]
	d.mu.RUnlock()

	if !ok {
		// No signal registered — coercion-resistance not enabled for this voter.
		return true, nil
	}

	presented := d.computeHMAC(signalType, detectedValue)
	// Constant-time comparison prevents timing attacks on the HMAC value.
	match := subtle.ConstantTimeCompare(stored.Hash, presented) == 1
	return match, nil
}

// HasSignal reports whether voterID has a registered duress signal.
func (d *InMemoryDuressDetector) HasSignal(voterID string) bool {
	d.mu.RLock()
	_, ok := d.signals[voterID]
	d.mu.RUnlock()
	return ok
}

// computeHMAC returns HMAC-SHA256(serverKey, signalType+":"+signalValue).
// The ":" separator prevents type/value concatenation collisions
// (e.g. type="blink" value="count3" ≠ type="blink_count" value="3" after separation).
func (d *InMemoryDuressDetector) computeHMAC(signalType, signalValue string) []byte {
	mac := hmac.New(sha256.New, d.hmacKey)
	mac.Write([]byte(signalType + ":" + signalValue))
	return mac.Sum(nil)
}

// --- validation helpers ---

func validateSignalType(t string) error {
	if !validSignalTypes[t] {
		return fmt.Errorf("%w: %q (accepted: blink_count, head_tilt, long_press, time_delay, voice_command)", ErrInvalidSignalType, t)
	}
	return nil
}

// validateSignalValue enforces per-type value constraints:
//   - blink_count, long_press: integer in [1, 5]
//   - head_tilt, time_delay, voice_command: non-empty string ≤ 64 chars
func validateSignalValue(signalType, value string) error {
	switch signalType {
	case SignalTypeBlinkCount, SignalTypeLongPress:
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 5 {
			return fmt.Errorf("%w: %s requires an integer between 1 and 5, got %q", ErrInvalidSignalValue, signalType, value)
		}
	default:
		if value == "" || len(value) > 64 {
			return fmt.Errorf("%w: value must be 1–64 characters for signal type %s", ErrInvalidSignalValue, signalType)
		}
	}
	return nil
}

// ValidSignalTypes returns the ordered list of accepted signal type strings.
// Exposed for use by API validation and documentation layers.
func ValidSignalTypes() []string {
	return []string{
		SignalTypeBlinkCount,
		SignalTypeHeadTilt,
		SignalTypeLongPress,
		SignalTypeTimeDelay,
		SignalTypeVoiceCommand,
	}
}
