package biometric

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
)

// LivenessDetector handles liveness detection to prevent spoofing
type LivenessDetector struct {
	Threshold float64 // Confidence threshold (0.0 to 1.0)
}

// NewLivenessDetector creates a new liveness detector
func NewLivenessDetector(threshold float64) *LivenessDetector {
	return &LivenessDetector{
		Threshold: threshold,
	}
}

// LivenessResult represents the result of liveness detection
type LivenessResult struct {
	IsLive     bool
	Confidence float64
	Reason     string
}

// CheckLiveness performs liveness detection on fingerprint data
// NOTE: This is a simplified implementation. In production, use a proper
// liveness detection algorithm (CNN-based, challenge-response, etc.)
func (ld *LivenessDetector) CheckLiveness(fingerprintData []byte) (*LivenessResult, error) {
	if len(fingerprintData) == 0 {
		return nil, errors.New("empty fingerprint data")
	}

	// Simplified liveness check based on data characteristics
	// In production, this would use ML models, texture analysis, etc.

	// Check 1: Data entropy (randomness)
	entropy := ld.calculateEntropy(fingerprintData)
	if entropy < 3.0 {
		return &LivenessResult{
			IsLive:     false,
			Confidence: 0.2,
			Reason:     "Low entropy - possible fake/photo",
		}, nil
	}

	// Check 2: Data size and quality indicators
	if len(fingerprintData) < 500 {
		return &LivenessResult{
			IsLive:     false,
			Confidence: 0.3,
			Reason:     "Insufficient data quality",
		}, nil
	}

	// Simulate liveness confidence score using crypto/rand
	// In production, this would come from a trained ML model
	confidence, err := cryptoRandFloat64()
	if err != nil {
		return nil, err
	}
	confidence = 0.7 + (confidence * 0.25) // 0.7 to 0.95

	isLive := confidence >= ld.Threshold

	reason := "Live fingerprint detected"
	if !isLive {
		reason = "Failed liveness check"
	}

	return &LivenessResult{
		IsLive:     isLive,
		Confidence: confidence,
		Reason:     reason,
	}, nil
}

// cryptoRandFloat64 generates a cryptographically secure random float64 in [0.0, 1.0).
// Uses crypto/rand instead of math/rand to avoid predictable output in security-critical paths.
func cryptoRandFloat64() (float64, error) {
	var buf [8]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return 0, err
	}
	// Use top 53 bits of a uint64 to produce a uniform float64 in [0, 1).
	// IEEE 754 double has 53 bits of mantissa precision.
	val := binary.BigEndian.Uint64(buf[:])
	return float64(val>>11) / float64(1<<53), nil
}

// calculateEntropy calculates Shannon entropy of data
func (ld *LivenessDetector) calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	// Count byte frequencies
	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	// Calculate entropy
	entropy := 0.0
	length := float64(len(data))

	for _, count := range freq {
		if count > 0 {
			p := float64(count) / length
			entropy -= p * logBase2(p)
		}
	}

	return entropy
}

// logBase2 calculates log base 2
func logBase2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// log2(x) = ln(x) / ln(2)
	return log(x) / log(2)
}

// log calculates natural logarithm (simplified)
func log(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Simplified - in production use math.Log
	n := 0.0
	for x > 2 {
		x /= 2
		n += 1
	}
	// Taylor series approximation for ln(x) near 1
	y := x - 1
	sum := y
	term := y
	for i := 2; i < 20; i++ {
		term *= -y
		sum += term / float64(i)
	}
	return sum + n*0.693147 // ln(2) ~ 0.693147
}
