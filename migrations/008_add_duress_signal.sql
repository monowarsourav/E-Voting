-- migrations/008_add_duress_signal.sql
--
-- Adds behavioral duress signal columns to the voters table.
--
-- These columns support future DB-backed DuressDetector implementations.
-- The current InMemoryDuressDetector stores signals in process memory only;
-- this migration is applied at startup to keep the schema always ready for
-- the persistence layer once it is wired up.
--
-- Column semantics:
--   duress_signal_hash    BLOB    HMAC-SHA256 of (signalType+":"+signalValue)
--   duress_signal_type    TEXT    one of: blink_count, head_tilt, long_press,
--                                         time_delay, voice_command
--   duress_signal_set_at  INTEGER Unix timestamp (seconds) when last updated

ALTER TABLE voters ADD COLUMN duress_signal_hash BLOB;
ALTER TABLE voters ADD COLUMN duress_signal_type TEXT;
ALTER TABLE voters ADD COLUMN duress_signal_set_at INTEGER;

CREATE INDEX IF NOT EXISTS idx_voters_duress_type ON voters(duress_signal_type);
