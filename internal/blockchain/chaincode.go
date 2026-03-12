// internal/blockchain/chaincode.go

package blockchain

// ChaincodeInterface defines the interface for interacting with chaincode
type ChaincodeInterface interface {
	// Election operations
	CreateElection(election *ElectionOnChain) (*ChaincodeResponse, error)
	GetElection(electionID string) (*ElectionOnChain, error)
	UpdateElectionStatus(electionID string, status string) (*ChaincodeResponse, error)
	ElectionExists(electionID string) (bool, error)
	GetAllElections() ([]*ElectionOnChain, error)

	// Vote operations
	CastVote(vote *VoteOnChain) (*ChaincodeResponse, error)
	GetVote(voteID string) (*VoteOnChain, error)
	GetVotesByElection(electionID string) ([]*VoteOnChain, error)
	VoteExists(keyImage string) (bool, error)

	// Credential operations
	StoreCredential(credential *CredentialOnChain) (*ChaincodeResponse, error)
	GetCredential(credentialID string) (*CredentialOnChain, error)
	GetCredentialsByElection(electionID string) ([]*CredentialOnChain, error)

	// Tally operations
	StoreTallyResult(result *TallyResultOnChain) (*ChaincodeResponse, error)
	GetTallyResult(electionID string) (*TallyResultOnChain, error)
	VerifyTallyResult(electionID string) (bool, error)

	// Query operations
	QueryByKey(key string) (*QueryResult, error)
	QueryByRange(startKey, endKey string) ([]*QueryResult, error)

	// Event operations
	SubscribeToEvents(eventName string, callback func(*BlockchainEvent) error) error
	UnsubscribeFromEvents(eventName string) error
}

// MockChaincode provides a mock implementation for testing
type MockChaincode struct {
	elections  map[string]*ElectionOnChain
	votes      map[string]*VoteOnChain
	credentials map[string]*CredentialOnChain
	tallies    map[string]*TallyResultOnChain
}

// NewMockChaincode creates a new mock chaincode
func NewMockChaincode() *MockChaincode {
	return &MockChaincode{
		elections:  make(map[string]*ElectionOnChain),
		votes:      make(map[string]*VoteOnChain),
		credentials: make(map[string]*CredentialOnChain),
		tallies:    make(map[string]*TallyResultOnChain),
	}
}

// CreateElection creates an election in mock storage
func (m *MockChaincode) CreateElection(election *ElectionOnChain) (*ChaincodeResponse, error) {
	m.elections[election.ID] = election
	return &ChaincodeResponse{
		TxID:    "mock-tx-" + election.ID,
		Success: true,
		Message: "Election created successfully",
	}, nil
}

// GetElection retrieves an election from mock storage
func (m *MockChaincode) GetElection(electionID string) (*ElectionOnChain, error) {
	if election, exists := m.elections[electionID]; exists {
		return election, nil
	}
	return nil, nil
}

// UpdateElectionStatus updates election status in mock storage
func (m *MockChaincode) UpdateElectionStatus(electionID string, status string) (*ChaincodeResponse, error) {
	if election, exists := m.elections[electionID]; exists {
		election.Status = status
		return &ChaincodeResponse{
			TxID:    "mock-tx-update-" + electionID,
			Success: true,
			Message: "Status updated successfully",
		}, nil
	}
	return &ChaincodeResponse{
		Success: false,
		Message: "Election not found",
	}, nil
}

// ElectionExists checks if an election exists
func (m *MockChaincode) ElectionExists(electionID string) (bool, error) {
	_, exists := m.elections[electionID]
	return exists, nil
}

// GetAllElections retrieves all elections
func (m *MockChaincode) GetAllElections() ([]*ElectionOnChain, error) {
	elections := make([]*ElectionOnChain, 0, len(m.elections))
	for _, election := range m.elections {
		elections = append(elections, election)
	}
	return elections, nil
}

// CastVote stores a vote in mock storage
func (m *MockChaincode) CastVote(vote *VoteOnChain) (*ChaincodeResponse, error) {
	m.votes[vote.ID] = vote
	return &ChaincodeResponse{
		TxID:    "mock-tx-vote-" + vote.ID,
		Success: true,
		Message: "Vote cast successfully",
	}, nil
}

// GetVote retrieves a vote from mock storage
func (m *MockChaincode) GetVote(voteID string) (*VoteOnChain, error) {
	if vote, exists := m.votes[voteID]; exists {
		return vote, nil
	}
	return nil, nil
}

// GetVotesByElection retrieves all votes for an election
func (m *MockChaincode) GetVotesByElection(electionID string) ([]*VoteOnChain, error) {
	votes := make([]*VoteOnChain, 0)
	for _, vote := range m.votes {
		if vote.ElectionID == electionID {
			votes = append(votes, vote)
		}
	}
	return votes, nil
}

// VoteExists checks if a vote with the given key image exists
func (m *MockChaincode) VoteExists(keyImage string) (bool, error) {
	for _, vote := range m.votes {
		if vote.KeyImage == keyImage {
			return true, nil
		}
	}
	return false, nil
}

// StoreCredential stores a credential in mock storage
func (m *MockChaincode) StoreCredential(credential *CredentialOnChain) (*ChaincodeResponse, error) {
	m.credentials[credential.ID] = credential
	return &ChaincodeResponse{
		TxID:    "mock-tx-cred-" + credential.ID,
		Success: true,
		Message: "Credential stored successfully",
	}, nil
}

// GetCredential retrieves a credential from mock storage
func (m *MockChaincode) GetCredential(credentialID string) (*CredentialOnChain, error) {
	if cred, exists := m.credentials[credentialID]; exists {
		return cred, nil
	}
	return nil, nil
}

// GetCredentialsByElection retrieves all credentials for an election
func (m *MockChaincode) GetCredentialsByElection(electionID string) ([]*CredentialOnChain, error) {
	creds := make([]*CredentialOnChain, 0)
	for _, cred := range m.credentials {
		if cred.ElectionID == electionID {
			creds = append(creds, cred)
		}
	}
	return creds, nil
}

// StoreTallyResult stores a tally result in mock storage
func (m *MockChaincode) StoreTallyResult(result *TallyResultOnChain) (*ChaincodeResponse, error) {
	m.tallies[result.ElectionID] = result
	return &ChaincodeResponse{
		TxID:    "mock-tx-tally-" + result.ElectionID,
		Success: true,
		Message: "Tally result stored successfully",
	}, nil
}

// GetTallyResult retrieves a tally result from mock storage
func (m *MockChaincode) GetTallyResult(electionID string) (*TallyResultOnChain, error) {
	if result, exists := m.tallies[electionID]; exists {
		return result, nil
	}
	return nil, nil
}

// VerifyTallyResult verifies a tally result
func (m *MockChaincode) VerifyTallyResult(electionID string) (bool, error) {
	if result, exists := m.tallies[electionID]; exists {
		return result.Verified, nil
	}
	return false, nil
}

// QueryByKey queries by a specific key
func (m *MockChaincode) QueryByKey(key string) (*QueryResult, error) {
	// Mock implementation
	return &QueryResult{
		Key:   key,
		Value: []byte("mock-value"),
	}, nil
}

// QueryByRange queries by key range
func (m *MockChaincode) QueryByRange(startKey, endKey string) ([]*QueryResult, error) {
	// Mock implementation
	return []*QueryResult{}, nil
}

// SubscribeToEvents subscribes to chaincode events
func (m *MockChaincode) SubscribeToEvents(eventName string, callback func(*BlockchainEvent) error) error {
	// Mock implementation
	return nil
}

// UnsubscribeFromEvents unsubscribes from chaincode events
func (m *MockChaincode) UnsubscribeFromEvents(eventName string) error {
	// Mock implementation
	return nil
}
