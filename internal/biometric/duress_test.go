package biometric

import (
	"bytes"
	"testing"
)

func newTestDetector() *InMemoryDuressDetector {
	return NewInMemoryDuressDetector([]byte("test-hmac-key-32-bytes-unit-tests"))
}

func TestSetSignal_Valid(t *testing.T) {
	d := newTestDetector()
	hash, err := d.SetSignal("voter001", SignalTypeBlinkCount, "3")
	if err != nil {
		t.Fatalf("SetSignal returned unexpected error: %v", err)
	}
	if len(hash) == 0 {
		t.Error("expected non-empty HMAC hash")
	}
	if !d.HasSignal("voter001") {
		t.Error("signal should be stored after SetSignal")
	}
}

func TestSetSignal_InvalidType(t *testing.T) {
	d := newTestDetector()
	_, err := d.SetSignal("voter001", "eye_roll", "2")
	if err == nil {
		t.Fatal("expected error for invalid signal type, got nil")
	}
}

func TestSetSignal_InvalidValue_BlinkCount(t *testing.T) {
	d := newTestDetector()
	// blink_count must be 1-5
	_, err := d.SetSignal("voter001", SignalTypeBlinkCount, "9")
	if err == nil {
		t.Fatal("expected error for out-of-range blink_count, got nil")
	}
	_, err = d.SetSignal("voter001", SignalTypeBlinkCount, "notanumber")
	if err == nil {
		t.Fatal("expected error for non-integer blink_count, got nil")
	}
}

func TestVerifySignal_Match(t *testing.T) {
	d := newTestDetector()
	if _, err := d.SetSignal("voter001", SignalTypeBlinkCount, "2"); err != nil {
		t.Fatalf("SetSignal: %v", err)
	}

	ok, err := d.VerifySignal("voter001", SignalTypeBlinkCount, "2")
	if err != nil {
		t.Fatalf("VerifySignal returned error: %v", err)
	}
	if !ok {
		t.Error("matching signal should return true")
	}
}

func TestVerifySignal_Mismatch(t *testing.T) {
	d := newTestDetector()
	if _, err := d.SetSignal("voter001", SignalTypeBlinkCount, "2"); err != nil {
		t.Fatalf("SetSignal: %v", err)
	}

	ok, err := d.VerifySignal("voter001", SignalTypeBlinkCount, "3") // wrong value
	if err != nil {
		t.Fatalf("VerifySignal returned error: %v", err)
	}
	if ok {
		t.Error("mismatched signal should return false")
	}
}

func TestVerifySignal_TypeMismatch(t *testing.T) {
	d := newTestDetector()
	if _, err := d.SetSignal("voter001", SignalTypeBlinkCount, "2"); err != nil {
		t.Fatalf("SetSignal: %v", err)
	}

	// Same value but different type — HMAC input differs, should mismatch.
	ok, err := d.VerifySignal("voter001", SignalTypeLongPress, "2")
	if err != nil {
		t.Fatalf("VerifySignal returned error: %v", err)
	}
	if ok {
		t.Error("different signal type with same value should return false")
	}
}

func TestVerifySignal_NoSignalSet(t *testing.T) {
	d := newTestDetector()
	// No signal registered → backward-compatible: treat as match (weight 1).
	ok, err := d.VerifySignal("unknown-voter", SignalTypeBlinkCount, "2")
	if err != nil {
		t.Fatalf("VerifySignal returned error for unregistered voter: %v", err)
	}
	if !ok {
		t.Error("VerifySignal for voter with no registered signal should return true")
	}
}

func TestHMACDeterministic(t *testing.T) {
	d := newTestDetector()
	h1 := d.computeHMAC(SignalTypeBlinkCount, "3")
	h2 := d.computeHMAC(SignalTypeBlinkCount, "3")
	if !bytes.Equal(h1, h2) {
		t.Error("HMAC must be deterministic for identical inputs")
	}

	// Different value → different hash.
	h3 := d.computeHMAC(SignalTypeBlinkCount, "4")
	if bytes.Equal(h1, h3) {
		t.Error("HMAC should differ for different signal values")
	}

	// Different type, same value → different hash (separator prevents collisions).
	h4 := d.computeHMAC(SignalTypeLongPress, "3")
	if bytes.Equal(h1, h4) {
		t.Error("HMAC should differ for different signal types with the same value")
	}
}

func TestSetSignal_Replace(t *testing.T) {
	d := newTestDetector()
	hash1, _ := d.SetSignal("voter001", SignalTypeBlinkCount, "2")
	hash2, _ := d.SetSignal("voter001", SignalTypeBlinkCount, "4") // replace

	if bytes.Equal(hash1, hash2) {
		t.Error("hash should change after replacing signal")
	}

	// Only new signal should match.
	ok, _ := d.VerifySignal("voter001", SignalTypeBlinkCount, "4")
	if !ok {
		t.Error("new signal should match after replacement")
	}
	ok2, _ := d.VerifySignal("voter001", SignalTypeBlinkCount, "2")
	if ok2 {
		t.Error("old signal should not match after replacement")
	}
}
