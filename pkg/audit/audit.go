// pkg/audit/audit.go

package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	// Election events
	EventElectionCreated AuditEventType = "election_created"
	EventElectionStarted AuditEventType = "election_started"
	EventElectionClosed  AuditEventType = "election_closed"
	EventElectionTallied AuditEventType = "election_tallied"

	// Voter events
	EventVoterRegistered AuditEventType = "voter_registered"
	EventVoterVerified   AuditEventType = "voter_verified"

	// Voting events
	EventVoteCast     AuditEventType = "vote_cast"
	EventVoteVerified AuditEventType = "vote_verified"

	// Admin events
	EventAdminLogin    AuditEventType = "admin_login"
	EventAdminAction   AuditEventType = "admin_action"
	EventKeyGenerated  AuditEventType = "key_generated"
	EventConfigChanged AuditEventType = "config_changed"

	// Security events
	EventAuthFailed        AuditEventType = "auth_failed"
	EventRateLimitHit      AuditEventType = "rate_limit_hit"
	EventInvalidRequest    AuditEventType = "invalid_request"
	EventDoubleVoteAttempt AuditEventType = "double_vote_attempt"
)

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID         int64          `json:"id"`
	Timestamp  time.Time      `json:"timestamp"`
	EventType  AuditEventType `json:"event_type"`
	UserID     string         `json:"user_id,omitempty"` // Voter ID or Admin ID
	ElectionID string         `json:"election_id,omitempty"`
	IPAddress  string         `json:"ip_address,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource,omitempty"`
	Result     string         `json:"result"` // success, failure, error
	ErrorMsg   string         `json:"error_msg,omitempty"`
	Metadata   string         `json:"metadata,omitempty"` // JSON-encoded additional data
}

// AuditLogger handles audit logging
type AuditLogger struct {
	db *sql.DB
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *sql.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// Log logs an audit event. Nil-safe: a nil logger silently no-ops so callers
// don't need to branch on whether auditing is configured.
func (al *AuditLogger) Log(event *AuditEvent) error {
	if al == nil {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	query := `
		INSERT INTO audit_logs (
			timestamp, event_type, user_id, election_id, ip_address,
			user_agent, action, resource, result, error_msg, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := al.db.Exec(
		query,
		event.Timestamp,
		event.EventType,
		event.UserID,
		event.ElectionID,
		event.IPAddress,
		event.UserAgent,
		event.Action,
		event.Resource,
		event.Result,
		event.ErrorMsg,
		event.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		event.ID = id
	}

	return nil
}

// LogSuccess logs a successful event
func (al *AuditLogger) LogSuccess(eventType AuditEventType, action string, userID string, metadata map[string]interface{}) error {
	if al == nil {
		return nil
	}
	metadataJSON, _ := json.Marshal(metadata)

	return al.Log(&AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Action:    action,
		Result:    "success",
		Metadata:  string(metadataJSON),
	})
}

// LogFailure logs a failed event
func (al *AuditLogger) LogFailure(eventType AuditEventType, action string, userID string, errorMsg string, metadata map[string]interface{}) error {
	if al == nil {
		return nil
	}
	metadataJSON, _ := json.Marshal(metadata)

	return al.Log(&AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Action:    action,
		Result:    "failure",
		ErrorMsg:  errorMsg,
		Metadata:  string(metadataJSON),
	})
}

// LogElectionCreated logs election creation
func (al *AuditLogger) LogElectionCreated(electionID string, adminID string, metadata map[string]interface{}) error {
	if al == nil {
		return nil
	}
	metadataJSON, _ := json.Marshal(metadata)

	return al.Log(&AuditEvent{
		EventType:  EventElectionCreated,
		UserID:     adminID,
		ElectionID: electionID,
		Action:     "create_election",
		Result:     "success",
		Metadata:   string(metadataJSON),
	})
}

// LogVoteCast logs a vote being cast
func (al *AuditLogger) LogVoteCast(electionID string, voterID string, ipAddress string) error {
	return al.Log(&AuditEvent{
		EventType:  EventVoteCast,
		UserID:     voterID,
		ElectionID: electionID,
		IPAddress:  ipAddress,
		Action:     "cast_vote",
		Result:     "success",
	})
}

// LogDoubleVoteAttempt logs an attempted double vote
func (al *AuditLogger) LogDoubleVoteAttempt(electionID string, voterID string, ipAddress string) error {
	return al.Log(&AuditEvent{
		EventType:  EventDoubleVoteAttempt,
		UserID:     voterID,
		ElectionID: electionID,
		IPAddress:  ipAddress,
		Action:     "attempted_double_vote",
		Result:     "failure",
		ErrorMsg:   "Voter has already cast a vote",
	})
}

// LogAuthFailure logs an authentication failure
func (al *AuditLogger) LogAuthFailure(userID string, ipAddress string, reason string) error {
	return al.Log(&AuditEvent{
		EventType: EventAuthFailed,
		UserID:    userID,
		IPAddress: ipAddress,
		Action:    "authentication",
		Result:    "failure",
		ErrorMsg:  reason,
	})
}

// Query retrieves audit logs with filters
func (al *AuditLogger) Query(filters *AuditQueryFilters) ([]*AuditEvent, error) {
	query := `
		SELECT id, timestamp, event_type, user_id, election_id, ip_address,
		       user_agent, action, resource, result, error_msg, metadata
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}

	if filters.EventType != "" {
		query += " AND event_type = ?"
		args = append(args, filters.EventType)
	}

	if filters.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filters.UserID)
	}

	if filters.ElectionID != "" {
		query += " AND election_id = ?"
		args = append(args, filters.ElectionID)
	}

	if !filters.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filters.StartTime)
	}

	if !filters.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filters.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	rows, err := al.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	events := make([]*AuditEvent, 0)
	for rows.Next() {
		var event AuditEvent
		var userID, electionID, ipAddress, userAgent, resource, errorMsg, metadata sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&event.EventType,
			&userID,
			&electionID,
			&ipAddress,
			&userAgent,
			&event.Action,
			&resource,
			&event.Result,
			&errorMsg,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if userID.Valid {
			event.UserID = userID.String
		}
		if electionID.Valid {
			event.ElectionID = electionID.String
		}
		if ipAddress.Valid {
			event.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			event.UserAgent = userAgent.String
		}
		if resource.Valid {
			event.Resource = resource.String
		}
		if errorMsg.Valid {
			event.ErrorMsg = errorMsg.String
		}
		if metadata.Valid {
			event.Metadata = metadata.String
		}

		events = append(events, &event)
	}

	return events, nil
}

// AuditQueryFilters defines filters for querying audit logs
type AuditQueryFilters struct {
	EventType  AuditEventType
	UserID     string
	ElectionID string
	StartTime  time.Time
	EndTime    time.Time
	Limit      int
}

// InitAuditTable creates the audit_logs table if it doesn't exist
func InitAuditTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			event_type VARCHAR(50) NOT NULL,
			user_id VARCHAR(255),
			election_id VARCHAR(255),
			ip_address VARCHAR(45),
			user_agent TEXT,
			action VARCHAR(100) NOT NULL,
			resource VARCHAR(255),
			result VARCHAR(20) NOT NULL,
			error_msg TEXT,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_audit_event_type ON audit_logs(event_type);
		CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit_logs(user_id);
		CREATE INDEX IF NOT EXISTS idx_audit_election_id ON audit_logs(election_id);
		CREATE INDEX IF NOT EXISTS idx_audit_result ON audit_logs(result);
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create audit_logs table: %w", err)
	}

	return nil
}
