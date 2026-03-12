package smdc

import (
	"testing"

	"github.com/covertvote/e-voting/internal/crypto"
)

func TestSMDCGeneration(t *testing.T) {
	// Setup
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := NewSMDCGenerator(pp, 5, "test-election-001") // k=5 slots

	// Generate credential
	cred, realIndex, err := gen.GenerateCredential("voter123")
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	// Check structure
	if cred.K != 5 {
		t.Errorf("Expected 5 slots, got %d", cred.K)
	}

	// Verify realIndex is in valid range
	if realIndex < 0 || realIndex >= 5 {
		t.Errorf("realIndex = %d, should be in [0, 4]", realIndex)
	}

	// Count weights
	totalWeight := int64(0)
	realCount := 0
	for _, slot := range cred.Slots {
		totalWeight += slot.Weight.Int64()
		if slot.Weight.Int64() == 1 {
			realCount++
		}
	}

	if totalWeight != 1 {
		t.Errorf("Total weight should be 1, got %d", totalWeight)
	}

	if realCount != 1 {
		t.Errorf("Should have exactly 1 real slot, got %d", realCount)
	}
}

func TestSMDCVerification(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := NewSMDCGenerator(pp, 5, "test-election-001")

	cred, _, _ := gen.GenerateCredential("voter456")
	pub := cred.GetPublicCredential()

	// Should verify
	if !gen.VerifyCredential(pub) {
		t.Error("Valid credential should verify")
	}
}

func TestSMDCCoercionResistance(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := NewSMDCGenerator(pp, 5, "test-election-001")

	cred, realIndex, _ := gen.GenerateCredential("voter789")

	// Get real slot
	realSlot, err := cred.GetRealSlot(realIndex)
	if err != nil {
		t.Fatalf("GetRealSlot failed: %v", err)
	}

	// Get a fake slot (any index except real)
	fakeIndex := (realIndex + 1) % cred.K
	fakeSlot, _ := cred.GetFakeSlot(fakeIndex, realIndex)

	// Verify they're different
	if realSlot.Weight.Cmp(fakeSlot.Weight) == 0 {
		t.Error("Real and fake slots should have different weights")
	}

	// But both should verify as valid binary!
	if !pp.VerifyBinary(realSlot.Commitment.C, realSlot.BinaryProof) {
		t.Error("Real slot binary proof failed")
	}

	if !pp.VerifyBinary(fakeSlot.Commitment.C, fakeSlot.BinaryProof) {
		t.Error("Fake slot binary proof failed")
	}
}

func TestSMDCCannotGetRealAsFake(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := NewSMDCGenerator(pp, 5, "test-election-001")

	cred, realIndex, _ := gen.GenerateCredential("voter000")

	// Try to get real slot as fake - should fail
	_, err := cred.GetFakeSlot(realIndex, realIndex)
	if err == nil {
		t.Error("Should not be able to get real slot as fake")
	}
}

func TestSMDCDeriveRealIndexDeterministic(t *testing.T) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := NewSMDCGenerator(pp, 5, "test-election-001")

	// DeriveRealIndex should return the same value for the same inputs
	idx1 := gen.DeriveRealIndex("voter123")
	idx2 := gen.DeriveRealIndex("voter123")

	if idx1 != idx2 {
		t.Errorf("DeriveRealIndex not deterministic: %d != %d", idx1, idx2)
	}

	// Different voterID should (very likely) produce a different index
	// but we mainly test determinism, not distribution
	if idx1 < 0 || idx1 >= 5 {
		t.Errorf("DeriveRealIndex out of range: %d", idx1)
	}
}
