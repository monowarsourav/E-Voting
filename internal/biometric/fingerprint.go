package biometric

import (
	"errors"

	"github.com/covertvote/e-voting/internal/crypto"
)

// FingerprintData represents raw fingerprint data
type FingerprintData struct {
	VoterID   string
	RawData   []byte // Raw fingerprint image or template
	Hash      []byte // SHA3 hash of fingerprint
	Timestamp int64
}

// FingerprintProcessor handles fingerprint processing
type FingerprintProcessor struct {
	MinDataSize int // Minimum size of fingerprint data
}

// NewFingerprintProcessor creates a new fingerprint processor
func NewFingerprintProcessor() *FingerprintProcessor {
	return &FingerprintProcessor{
		MinDataSize: 100, // Minimum 100 bytes
	}
}

// ProcessFingerprint processes raw fingerprint data and creates a hash
func (fp *FingerprintProcessor) ProcessFingerprint(voterID string, rawData []byte, timestamp int64) (*FingerprintData, error) {
	// Validate data size
	if len(rawData) < fp.MinDataSize {
		return nil, errors.New("fingerprint data too small")
	}

	// Compute SHA3 hash of fingerprint
	hash := crypto.FingerprintToHash(rawData)

	return &FingerprintData{
		VoterID:   voterID,
		RawData:   rawData,
		Hash:      hash,
		Timestamp: timestamp,
	}, nil
}

// VerifyFingerprint verifies a fingerprint against stored hash
func (fp *FingerprintProcessor) VerifyFingerprint(rawData []byte, storedHash []byte) bool {
	computedHash := crypto.FingerprintToHash(rawData)

	// Compare hashes
	if len(computedHash) != len(storedHash) {
		return false
	}

	for i := range computedHash {
		if computedHash[i] != storedHash[i] {
			return false
		}
	}

	return true
}

// GenerateVoterID generates a deterministic voter ID from fingerprint
func (fp *FingerprintProcessor) GenerateVoterID(fingerprintHash []byte, nid string) string {
	// Combine fingerprint hash with NID
	combined := append(fingerprintHash, []byte(nid)...)
	idHash := crypto.SHA3Hash(combined)

	// Convert to hex string (first 16 bytes = 32 hex chars)
	voterID := ""
	for i := 0; i < 16 && i < len(idHash); i++ {
		voterID += string("0123456789abcdef"[idHash[i]>>4])
		voterID += string("0123456789abcdef"[idHash[i]&0x0f])
	}

	return voterID
}
