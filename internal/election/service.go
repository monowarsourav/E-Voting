// internal/election/service.go

package election

import (
	"fmt"
	"time"
)

// LoggerInterface defines the logging interface
type LoggerInterface interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// Service handles election business logic
type Service struct {
	repo   *Repository
	logger LoggerInterface
}

// NewService creates a new election service
func NewService(repo *Repository, logger LoggerInterface) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// CreateElection creates a new election
func (s *Service) CreateElection(req *CreateElectionRequest) (*Election, error) {
	// Validate time constraints
	now := time.Now()
	if req.StartTime.Before(now) {
		return nil, fmt.Errorf("start time must be in the future")
	}
	if req.EndTime.Before(req.StartTime) {
		return nil, fmt.Errorf("end time must be after start time")
	}
	if req.RegistrationDeadline.After(req.StartTime) {
		return nil, fmt.Errorf("registration deadline must be before start time")
	}
	if req.RegistrationDeadline.Before(now) {
		return nil, fmt.Errorf("registration deadline must be in the future")
	}

	// Validate candidates
	if len(req.Candidates) < 2 {
		return nil, fmt.Errorf("election must have at least 2 candidates")
	}
	if len(req.Candidates) > 100 {
		return nil, fmt.Errorf("election cannot have more than 100 candidates")
	}

	// Check for duplicate candidates
	candidateMap := make(map[string]bool)
	for _, candidate := range req.Candidates {
		if candidate == "" {
			return nil, fmt.Errorf("candidate name cannot be empty")
		}
		if candidateMap[candidate] {
			return nil, fmt.Errorf("duplicate candidate: %s", candidate)
		}
		candidateMap[candidate] = true
	}

	election := &Election{
		Name:                 req.Name,
		Description:          req.Description,
		Candidates:           req.Candidates,
		NumCandidates:        len(req.Candidates),
		StartTime:            req.StartTime,
		EndTime:              req.EndTime,
		RegistrationDeadline: req.RegistrationDeadline,
		Status:               string(StatusPending),
		MerkleRoot:           "", // Will be set when voters register
	}

	if err := s.repo.Create(election); err != nil {
		s.logger.Error("Failed to create election", "error", err)
		return nil, fmt.Errorf("failed to create election: %w", err)
	}

	s.logger.Info("Election created", "id", election.ID, "name", election.Name)
	return election, nil
}

// GetElection retrieves an election by ID
func (s *Service) GetElection(id string) (*Election, error) {
	election, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update status based on current time
	s.updateElectionStatus(election)

	return election, nil
}

// GetAllElections retrieves all elections
func (s *Service) GetAllElections() ([]*Election, error) {
	elections, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	// Update statuses
	for _, election := range elections {
		s.updateElectionStatus(election)
	}

	return elections, nil
}

// GetActiveElections retrieves all active elections
func (s *Service) GetActiveElections() ([]*Election, error) {
	elections, err := s.repo.GetByStatus(string(StatusActive))
	if err != nil {
		return nil, err
	}

	return elections, nil
}

// UpdateElection updates an election
func (s *Service) UpdateElection(id string, req *UpdateElectionRequest) (*Election, error) {
	election, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Only allow updates to pending elections
	if election.Status != string(StatusPending) {
		return nil, fmt.Errorf("cannot update election with status: %s", election.Status)
	}

	if req.Name != "" {
		election.Name = req.Name
	}
	if req.Description != "" {
		election.Description = req.Description
	}
	if len(req.Candidates) > 0 {
		if len(req.Candidates) < 2 {
			return nil, fmt.Errorf("election must have at least 2 candidates")
		}
		election.Candidates = req.Candidates
		election.NumCandidates = len(req.Candidates)
	}

	if err := s.repo.Update(election); err != nil {
		s.logger.Error("Failed to update election", "error", err, "id", id)
		return nil, fmt.Errorf("failed to update election: %w", err)
	}

	s.logger.Info("Election updated", "id", election.ID)
	return election, nil
}

// StartElection starts an election (changes status to active)
func (s *Service) StartElection(id string) error {
	election, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	now := time.Now()
	if now.Before(election.StartTime) {
		return fmt.Errorf("election cannot be started before scheduled start time")
	}

	if election.Status != string(StatusPending) {
		return fmt.Errorf("election is not in pending status")
	}

	if err := s.repo.UpdateStatus(id, string(StatusActive)); err != nil {
		s.logger.Error("Failed to start election", "error", err, "id", id)
		return err
	}

	s.logger.Info("Election started", "id", id, "name", election.Name)
	return nil
}

// CloseElection closes an election
func (s *Service) CloseElection(id string) error {
	election, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if election.Status != string(StatusActive) {
		return fmt.Errorf("election is not active")
	}

	if err := s.repo.UpdateStatus(id, string(StatusClosed)); err != nil {
		s.logger.Error("Failed to close election", "error", err, "id", id)
		return err
	}

	s.logger.Info("Election closed", "id", id, "name", election.Name)
	return nil
}

// SetElectionTallied marks an election as tallied
func (s *Service) SetElectionTallied(id string) error {
	election, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if election.Status != string(StatusClosed) {
		return fmt.Errorf("election must be closed before tallying")
	}

	if err := s.repo.UpdateStatus(id, string(StatusTallied)); err != nil {
		s.logger.Error("Failed to set election as tallied", "error", err, "id", id)
		return err
	}

	s.logger.Info("Election marked as tallied", "id", id, "name", election.Name)
	return nil
}

// UpdateMerkleRoot updates the Merkle root for an election
func (s *Service) UpdateMerkleRoot(id string, merkleRoot string) error {
	election, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	election.MerkleRoot = merkleRoot

	if err := s.repo.Update(election); err != nil {
		s.logger.Error("Failed to update Merkle root", "error", err, "id", id)
		return err
	}

	s.logger.Info("Merkle root updated", "id", id)
	return nil
}

// updateElectionStatus updates election status based on current time
func (s *Service) updateElectionStatus(election *Election) {
	now := time.Now()

	// Don't change status if already tallied
	if election.Status == string(StatusTallied) {
		return
	}

	// Check if election should be active
	if now.After(election.StartTime) && now.Before(election.EndTime) {
		if election.Status == string(StatusPending) {
			s.repo.UpdateStatus(election.ID, string(StatusActive))
			election.Status = string(StatusActive)
		}
	}

	// Check if election should be closed
	if now.After(election.EndTime) {
		if election.Status == string(StatusActive) {
			s.repo.UpdateStatus(election.ID, string(StatusClosed))
			election.Status = string(StatusClosed)
		}
	}
}

// DeleteElection deletes an election (only if not started)
func (s *Service) DeleteElection(id string) error {
	election, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if election.Status != string(StatusPending) {
		return fmt.Errorf("cannot delete election with status: %s", election.Status)
	}

	if err := s.repo.Delete(id); err != nil {
		s.logger.Error("Failed to delete election", "error", err, "id", id)
		return err
	}

	s.logger.Info("Election deleted", "id", id)
	return nil
}
