package biometric

import (
	"testing"
	"time"
)

func TestFingerprintProcessing(t *testing.T) {
	fp := NewFingerprintProcessor()

	// Create sample fingerprint data
	rawData := make([]byte, 200)
	for i := range rawData {
		rawData[i] = byte(i % 256)
	}

	timestamp := time.Now().Unix()

	// Process fingerprint
	fpData, err := fp.ProcessFingerprint("voter001", rawData, timestamp)
	if err != nil {
		t.Fatalf("Processing failed: %v", err)
	}

	if fpData.VoterID != "voter001" {
		t.Errorf("VoterID mismatch")
	}

	if len(fpData.Hash) == 0 {
		t.Error("Hash should not be empty")
	}
}

func TestFingerprintVerification(t *testing.T) {
	fp := NewFingerprintProcessor()

	rawData := make([]byte, 200)
	for i := range rawData {
		rawData[i] = byte(i % 256)
	}

	fpData, _ := fp.ProcessFingerprint("voter001", rawData, time.Now().Unix())

	// Should verify with same data
	if !fp.VerifyFingerprint(rawData, fpData.Hash) {
		t.Error("Verification should succeed with correct data")
	}

	// Should fail with different data
	tamperedData := make([]byte, 200)
	for i := range tamperedData {
		tamperedData[i] = byte((i + 1) % 256)
	}

	if fp.VerifyFingerprint(tamperedData, fpData.Hash) {
		t.Error("Verification should fail with tampered data")
	}
}

func TestVoterIDGeneration(t *testing.T) {
	fp := NewFingerprintProcessor()

	rawData := make([]byte, 200)
	fpData, _ := fp.ProcessFingerprint("voter001", rawData, time.Now().Unix())

	voterID := fp.GenerateVoterID(fpData.Hash, "NID123456")

	if len(voterID) != 32 {
		t.Errorf("VoterID should be 32 chars, got %d", len(voterID))
	}

	// Same inputs should produce same ID
	voterID2 := fp.GenerateVoterID(fpData.Hash, "NID123456")
	if voterID != voterID2 {
		t.Error("Same inputs should produce same voter ID")
	}

	// Different NID should produce different ID
	voterID3 := fp.GenerateVoterID(fpData.Hash, "NID789012")
	if voterID == voterID3 {
		t.Error("Different NID should produce different voter ID")
	}
}

func TestLivenessDetection(t *testing.T) {
	ld := NewLivenessDetector(0.7)

	// Good quality data
	goodData := make([]byte, 1000)
	for i := range goodData {
		goodData[i] = byte((i * 137) % 256) // Pseudo-random
	}

	result, err := ld.CheckLiveness(goodData)
	if err != nil {
		t.Fatalf("Liveness check failed: %v", err)
	}

	if !result.IsLive {
		t.Logf("Liveness result: %v (confidence: %.2f)", result.Reason, result.Confidence)
	}

	// Poor quality data
	poorData := make([]byte, 100)
	result2, _ := ld.CheckLiveness(poorData)

	if result2.IsLive {
		t.Error("Poor quality data should fail liveness check")
	}
}
