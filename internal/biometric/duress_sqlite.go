package biometric

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SQLiteDuressDetector is a DuressDetector backed by the application's SQLite
// database. It survives server restarts by reading and writing the
// duress_signal_hash / duress_signal_type / duress_signal_set_at columns
// added to the voters table by migration 008_add_duress_signal.sql.
//
// Security properties preserved from InMemoryDuressDetector:
//   - Only the HMAC hash is written to disk; the raw signal value is never persisted.
//   - Constant-time comparison via crypto/subtle prevents timing attacks.
//   - The HMAC key lives only in process memory; a DB compromise leaks hashes
//     but cannot reverse-engineer signal values without the key.
type SQLiteDuressDetector struct {
	db      *sql.DB
	hmacKey []byte
}

// NewSQLiteDuressDetector constructs a detector backed by db.
// The hmacKey must be the same across all server instances sharing the DB so
// previously stored hashes remain verifiable after a restart.
func NewSQLiteDuressDetector(db *sql.DB, hmacKey []byte) *SQLiteDuressDetector {
	if len(hmacKey) == 0 {
		// NOT safe for production: replaced by the caller when DURESS_HMAC_KEY is set.
		hmacKey = []byte("dev-duress-hmac-key-replace-in-production")
	}
	return &SQLiteDuressDetector{db: db, hmacKey: hmacKey}
}

// SetSignal validates (signalType, signalValue), computes HMAC-SHA256, and
// persists the hash to the voters table row for voterID. Returns an error if
// the voter does not exist — the HTTP handler guards this before calling.
func (d *SQLiteDuressDetector) SetSignal(voterID, signalType, signalValue string) ([]byte, error) {
	if err := validateSignalType(signalType); err != nil {
		return nil, err
	}
	if err := validateSignalValue(signalType, signalValue); err != nil {
		return nil, err
	}

	hash := d.computeHMAC(signalType, signalValue)

	result, err := d.db.Exec(`
		UPDATE voters
		   SET duress_signal_hash   = ?,
		       duress_signal_type   = ?,
		       duress_signal_set_at = ?
		 WHERE voter_id = ?`,
		hash, signalType, time.Now().Unix(), voterID,
	)
	if err != nil {
		return nil, fmt.Errorf("duress SetSignal: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("duress SetSignal rows: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("duress SetSignal: voter %q not found in database", voterID)
	}
	return hash, nil
}

// VerifySignal returns true when the HMAC of (signalType, detectedValue)
// matches the stored hash for voterID. Returns true when no signal is
// registered (backward-compatible: absent signal = always passes).
// Comparison is constant-time to prevent timing attacks.
func (d *SQLiteDuressDetector) VerifySignal(voterID, signalType, detectedValue string) (bool, error) {
	var storedHash []byte
	err := d.db.QueryRow(`
		SELECT duress_signal_hash
		  FROM voters
		 WHERE voter_id = ? AND duress_signal_hash IS NOT NULL`,
		voterID,
	).Scan(&storedHash)

	if errors.Is(err, sql.ErrNoRows) {
		// No signal registered — coercion-resistance not enabled for this voter.
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("duress VerifySignal: %w", err)
	}

	presented := d.computeHMAC(signalType, detectedValue)
	// Constant-time comparison prevents HMAC oracle attacks.
	match := subtle.ConstantTimeCompare(storedHash, presented) == 1
	return match, nil
}

// HasSignal reports whether voterID has a non-null duress signal hash in the DB.
func (d *SQLiteDuressDetector) HasSignal(voterID string) bool {
	var exists int
	// Error on QueryRow is absorbed: a DB error causes exists=0 (safe default).
	d.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM voters
			 WHERE voter_id = ? AND duress_signal_hash IS NOT NULL
		)`, voterID,
	).Scan(&exists)
	return exists == 1
}

// RemoveSignal clears the duress signal for voterID. Idempotent: returns nil
// even when no signal was set.
func (d *SQLiteDuressDetector) RemoveSignal(voterID string) error {
	_, err := d.db.Exec(`
		UPDATE voters
		   SET duress_signal_hash   = NULL,
		       duress_signal_type   = NULL,
		       duress_signal_set_at = NULL
		 WHERE voter_id = ?`, voterID,
	)
	if err != nil {
		return fmt.Errorf("duress RemoveSignal: %w", err)
	}
	return nil
}

// computeHMAC mirrors InMemoryDuressDetector.computeHMAC exactly — same key,
// same separator, same output — so both implementations are interchangeable
// against the same stored hashes.
func (d *SQLiteDuressDetector) computeHMAC(signalType, signalValue string) []byte {
	mac := hmac.New(sha256.New, d.hmacKey)
	mac.Write([]byte(signalType + ":" + signalValue))
	return mac.Sum(nil)
}
