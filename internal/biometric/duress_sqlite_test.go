package biometric

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// newTestSQLiteDB creates an in-memory SQLite database with the minimal schema
// required by SQLiteDuressDetector (voters table + duress columns from migration 008).
func newTestSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite3 in-memory: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE voters (
			id          TEXT PRIMARY KEY,
			voter_id    TEXT UNIQUE NOT NULL,
			fingerprint_hash TEXT NOT NULL DEFAULT '',
			nid_hash         TEXT NOT NULL DEFAULT '',
			ring_public_key  TEXT NOT NULL DEFAULT '',
			election_id      TEXT NOT NULL DEFAULT 'test',
			duress_signal_hash   BLOB,
			duress_signal_type   TEXT,
			duress_signal_set_at INTEGER
		)`)
	if err != nil {
		t.Fatalf("create test voters table: %v", err)
	}
	return db
}

// insertTestVoter inserts a minimal voter row so SetSignal can UPDATE it.
func insertTestVoter(t *testing.T, db *sql.DB, voterID string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO voters (id, voter_id) VALUES (?, ?)`, voterID, voterID)
	if err != nil {
		t.Fatalf("insert test voter %q: %v", voterID, err)
	}
}

func newTestSQLiteDetector(db *sql.DB) *SQLiteDuressDetector {
	return NewSQLiteDuressDetector(db, []byte("sqlite-test-hmac-key-32-bytes-ok"))
}

// TestSQLiteDuressDetector_SetAndVerify verifies that a signal set via SetSignal
// is immediately verifiable on the same detector instance.
func TestSQLiteDuressDetector_SetAndVerify(t *testing.T) {
	db := newTestSQLiteDB(t)
	insertTestVoter(t, db, "voter001")
	d := newTestSQLiteDetector(db)

	hash, err := d.SetSignal("voter001", SignalTypeBlinkCount, "3")
	if err != nil {
		t.Fatalf("SetSignal: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("expected non-empty HMAC hash from SetSignal")
	}
	if !d.HasSignal("voter001") {
		t.Error("HasSignal should be true after SetSignal")
	}

	ok, err := d.VerifySignal("voter001", SignalTypeBlinkCount, "3")
	if err != nil {
		t.Fatalf("VerifySignal: %v", err)
	}
	if !ok {
		t.Error("VerifySignal should return true for matching signal")
	}

	ok2, err := d.VerifySignal("voter001", SignalTypeBlinkCount, "2")
	if err != nil {
		t.Fatalf("VerifySignal (mismatch): %v", err)
	}
	if ok2 {
		t.Error("VerifySignal should return false for mismatched signal")
	}
}

// TestSQLiteDuressDetector_PersistAcrossInstances verifies that a signal written
// by one SQLiteDuressDetector instance is readable by a second instance that
// shares the same database — the core requirement for restart-survival.
func TestSQLiteDuressDetector_PersistAcrossInstances(t *testing.T) {
	db := newTestSQLiteDB(t)
	insertTestVoter(t, db, "voter001")

	const hmacKey = "shared-hmac-key-must-match-32-b!"
	d1 := NewSQLiteDuressDetector(db, []byte(hmacKey))
	if _, err := d1.SetSignal("voter001", SignalTypeHeadTilt, "left"); err != nil {
		t.Fatalf("d1 SetSignal: %v", err)
	}

	// New detector, same DB and key — simulates a server restart.
	d2 := NewSQLiteDuressDetector(db, []byte(hmacKey))
	if !d2.HasSignal("voter001") {
		t.Error("d2 HasSignal: signal should persist across instances")
	}
	ok, err := d2.VerifySignal("voter001", SignalTypeHeadTilt, "left")
	if err != nil {
		t.Fatalf("d2 VerifySignal: %v", err)
	}
	if !ok {
		t.Error("d2 VerifySignal: signal written by d1 must be verifiable by d2")
	}
}

// TestSQLiteDuressDetector_NoSignalReturnsTrue checks backward compatibility:
// a voter with no registered signal always passes verification (weight = 1).
func TestSQLiteDuressDetector_NoSignalReturnsTrue(t *testing.T) {
	db := newTestSQLiteDB(t)
	insertTestVoter(t, db, "voter001")
	d := newTestSQLiteDetector(db)

	if d.HasSignal("voter001") {
		t.Error("HasSignal should be false when no signal has been set")
	}

	ok, err := d.VerifySignal("voter001", SignalTypeBlinkCount, "3")
	if err != nil {
		t.Fatalf("VerifySignal: %v", err)
	}
	if !ok {
		t.Error("VerifySignal must return true when no signal is registered (backward compat)")
	}
}

// TestSQLiteDuressDetector_Replace checks that SetSignal replaces a prior signal
// and only the new value verifies correctly.
func TestSQLiteDuressDetector_Replace(t *testing.T) {
	db := newTestSQLiteDB(t)
	insertTestVoter(t, db, "voter001")
	d := newTestSQLiteDetector(db)

	if _, err := d.SetSignal("voter001", SignalTypeBlinkCount, "2"); err != nil {
		t.Fatalf("SetSignal (first): %v", err)
	}
	if _, err := d.SetSignal("voter001", SignalTypeBlinkCount, "4"); err != nil {
		t.Fatalf("SetSignal (replace): %v", err)
	}

	ok, _ := d.VerifySignal("voter001", SignalTypeBlinkCount, "4")
	if !ok {
		t.Error("new signal should verify after replacement")
	}
	old, _ := d.VerifySignal("voter001", SignalTypeBlinkCount, "2")
	if old {
		t.Error("old signal should not verify after replacement")
	}
}

// TestSQLiteDuressDetector_RemoveSignal checks that RemoveSignal clears the
// stored hash and HasSignal returns false, and that it is idempotent.
func TestSQLiteDuressDetector_RemoveSignal(t *testing.T) {
	db := newTestSQLiteDB(t)
	insertTestVoter(t, db, "voter001")
	d := newTestSQLiteDetector(db)

	if _, err := d.SetSignal("voter001", SignalTypeBlinkCount, "2"); err != nil {
		t.Fatalf("SetSignal: %v", err)
	}
	if !d.HasSignal("voter001") {
		t.Fatal("signal should exist before removal")
	}

	if err := d.RemoveSignal("voter001"); err != nil {
		t.Fatalf("RemoveSignal: %v", err)
	}
	if d.HasSignal("voter001") {
		t.Error("HasSignal should be false after RemoveSignal")
	}

	// Idempotent — second call should not error.
	if err := d.RemoveSignal("voter001"); err != nil {
		t.Fatalf("RemoveSignal (idempotent): %v", err)
	}
}

// TestSQLiteDuressDetector_SetSignal_VoterNotFound verifies that SetSignal
// returns an error when the voter row does not exist in the database.
func TestSQLiteDuressDetector_SetSignal_VoterNotFound(t *testing.T) {
	db := newTestSQLiteDB(t)
	d := newTestSQLiteDetector(db)

	_, err := d.SetSignal("ghost", SignalTypeBlinkCount, "1")
	if err == nil {
		t.Fatal("expected error when voter does not exist in DB")
	}
}
