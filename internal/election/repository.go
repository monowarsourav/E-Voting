// internal/election/repository.go

package election

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository handles database operations for elections
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new election repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new election in the database
func (r *Repository) Create(election *Election) error {
	candidatesJSON, err := json.Marshal(election.Candidates)
	if err != nil {
		return fmt.Errorf("failed to marshal candidates: %w", err)
	}

	if election.ID == "" {
		election.ID = uuid.New().String()
	}

	now := time.Now()
	election.CreatedAt = now
	election.UpdatedAt = now

	query := `
		INSERT INTO elections (
			id, name, description, status, candidates, num_candidates,
			merkle_root, start_time, end_time, registration_deadline,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(
		query,
		election.ID,
		election.Name,
		election.Description,
		election.Status,
		string(candidatesJSON),
		election.NumCandidates,
		election.MerkleRoot,
		election.StartTime,
		election.EndTime,
		election.RegistrationDeadline,
		election.CreatedAt,
		election.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert election: %w", err)
	}

	return nil
}

// GetByID retrieves an election by ID
func (r *Repository) GetByID(id string) (*Election, error) {
	query := `
		SELECT id, name, description, status, candidates, num_candidates,
		       merkle_root, start_time, end_time, registration_deadline,
		       created_at, updated_at
		FROM elections
		WHERE id = ?
	`

	var election Election
	var candidatesJSON string

	err := r.db.QueryRow(query, id).Scan(
		&election.ID,
		&election.Name,
		&election.Description,
		&election.Status,
		&candidatesJSON,
		&election.NumCandidates,
		&election.MerkleRoot,
		&election.StartTime,
		&election.EndTime,
		&election.RegistrationDeadline,
		&election.CreatedAt,
		&election.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("election not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query election: %w", err)
	}

	if err := json.Unmarshal([]byte(candidatesJSON), &election.Candidates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal candidates: %w", err)
	}

	return &election, nil
}

// GetAll retrieves all elections
func (r *Repository) GetAll() ([]*Election, error) {
	query := `
		SELECT id, name, description, status, candidates, num_candidates,
		       merkle_root, start_time, end_time, registration_deadline,
		       created_at, updated_at
		FROM elections
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query elections: %w", err)
	}
	defer rows.Close()

	var elections []*Election
	for rows.Next() {
		var election Election
		var candidatesJSON string

		err := rows.Scan(
			&election.ID,
			&election.Name,
			&election.Description,
			&election.Status,
			&candidatesJSON,
			&election.NumCandidates,
			&election.MerkleRoot,
			&election.StartTime,
			&election.EndTime,
			&election.RegistrationDeadline,
			&election.CreatedAt,
			&election.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan election: %w", err)
		}

		if err := json.Unmarshal([]byte(candidatesJSON), &election.Candidates); err != nil {
			return nil, fmt.Errorf("failed to unmarshal candidates: %w", err)
		}

		elections = append(elections, &election)
	}

	return elections, nil
}

// GetByStatus retrieves elections by status
func (r *Repository) GetByStatus(status string) ([]*Election, error) {
	query := `
		SELECT id, name, description, status, candidates, num_candidates,
		       merkle_root, start_time, end_time, registration_deadline,
		       created_at, updated_at
		FROM elections
		WHERE status = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query elections: %w", err)
	}
	defer rows.Close()

	var elections []*Election
	for rows.Next() {
		var election Election
		var candidatesJSON string

		err := rows.Scan(
			&election.ID,
			&election.Name,
			&election.Description,
			&election.Status,
			&candidatesJSON,
			&election.NumCandidates,
			&election.MerkleRoot,
			&election.StartTime,
			&election.EndTime,
			&election.RegistrationDeadline,
			&election.CreatedAt,
			&election.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan election: %w", err)
		}

		if err := json.Unmarshal([]byte(candidatesJSON), &election.Candidates); err != nil {
			return nil, fmt.Errorf("failed to unmarshal candidates: %w", err)
		}

		elections = append(elections, &election)
	}

	return elections, nil
}

// Update updates an election
func (r *Repository) Update(election *Election) error {
	candidatesJSON, err := json.Marshal(election.Candidates)
	if err != nil {
		return fmt.Errorf("failed to marshal candidates: %w", err)
	}

	election.UpdatedAt = time.Now()

	query := `
		UPDATE elections
		SET name = ?, description = ?, status = ?, candidates = ?,
		    num_candidates = ?, merkle_root = ?, start_time = ?,
		    end_time = ?, registration_deadline = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = r.db.Exec(
		query,
		election.Name,
		election.Description,
		election.Status,
		string(candidatesJSON),
		election.NumCandidates,
		election.MerkleRoot,
		election.StartTime,
		election.EndTime,
		election.RegistrationDeadline,
		election.UpdatedAt,
		election.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update election: %w", err)
	}

	return nil
}

// UpdateStatus updates only the election status
func (r *Repository) UpdateStatus(id string, status string) error {
	query := `
		UPDATE elections
		SET status = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update election status: %w", err)
	}

	return nil
}

// Delete deletes an election
func (r *Repository) Delete(id string) error {
	query := `DELETE FROM elections WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete election: %w", err)
	}

	return nil
}
