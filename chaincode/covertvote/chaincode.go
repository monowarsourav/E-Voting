package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// AdminMSPID is the MSP ID that has administrative privileges for election management.
const AdminMSPID = "ElectionCommissionMSP"

// VoteChaincode provides functions for managing votes on the blockchain
type VoteChaincode struct {
	contractapi.Contract
}

// Vote represents a vote on the blockchain
type Vote struct {
	VoteID         string `json:"vote_id"`
	ElectionID     string `json:"election_id"`
	EncryptedVote  string `json:"encrypted_vote"`
	RingSignature  string `json:"ring_signature"`
	KeyImage       string `json:"key_image"`
	SMDCCommitment string `json:"smdc_commitment"`
	MerkleProof    string `json:"merkle_proof"`
	Timestamp      int64  `json:"timestamp"`
	BlockNumber    uint64 `json:"block_number"`
	OwnerID        string `json:"owner_id"`
}

// VoteSummary is a redacted view of a vote, omitting privacy-sensitive fields.
type VoteSummary struct {
	VoteID     string `json:"vote_id"`
	ElectionID string `json:"election_id"`
	Timestamp  int64  `json:"timestamp"`
}

// Election represents an election on the blockchain
type Election struct {
	ElectionID  string   `json:"election_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Candidates  []string `json:"candidates"`
	StartTime   int64    `json:"start_time"`
	EndTime     int64    `json:"end_time"`
	IsActive    bool     `json:"is_active"`
	CreatedAt   int64    `json:"created_at"`
}

// TallyResult represents tally results on the blockchain
type TallyResult struct {
	ElectionID       string         `json:"election_id"`
	CandidateTallies map[string]int `json:"candidate_tallies"`
	TotalVotes       int            `json:"total_votes"`
	TallyTime        int64          `json:"tally_time"`
	Verified         bool           `json:"verified"`
}

// QueryResult structure used for handling result of query
type QueryResult struct {
	Key    string `json:"key"`
	Record *Vote  `json:"record"`
}

// couchDBQuery represents a CouchDB selector query, built safely via json.Marshal.
type couchDBQuery struct {
	Selector map[string]interface{} `json:"selector"`
}

// isAdmin returns true if the caller's MSP ID matches the admin org.
func (vc *VoteChaincode) isAdmin(ctx contractapi.TransactionContextInterface) (bool, error) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return false, fmt.Errorf("failed to get client MSP ID: %v", err)
	}
	return mspID == AdminMSPID, nil
}

// requireAdmin returns an error if the caller is not from the admin org.
func (vc *VoteChaincode) requireAdmin(ctx contractapi.TransactionContextInterface) error {
	admin, err := vc.isAdmin(ctx)
	if err != nil {
		return err
	}
	if !admin {
		return fmt.Errorf("access denied: this function requires admin org (%s)", AdminMSPID)
	}
	return nil
}

// requireAuthenticated returns the caller's identity ID, or an error if the
// identity cannot be determined (i.e. the caller is not authenticated).
func (vc *VoteChaincode) requireAuthenticated(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := cid.GetID(ctx.GetStub())
	if err != nil || id == "" {
		return "", fmt.Errorf("access denied: caller does not have a valid identity")
	}
	return id, nil
}

// InitLedger adds base data to the ledger
func (vc *VoteChaincode) InitLedger(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("Initializing CovertVote chaincode ledger")
	return nil
}

// CreateElection creates a new election on the blockchain.
// ACCESS: Only the admin org (ElectionCommissionMSP) may call this function.
func (vc *VoteChaincode) CreateElection(
	ctx contractapi.TransactionContextInterface,
	electionID string,
	title string,
	description string,
	candidatesJSON string,
	startTime int64,
	endTime int64,
) error {
	// --- Access control: admin only ---
	if err := vc.requireAdmin(ctx); err != nil {
		return err
	}

	// Check if election already exists
	exists, err := vc.ElectionExists(ctx, electionID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("election %s already exists", electionID)
	}

	// Parse candidates
	var candidates []string
	if err := json.Unmarshal([]byte(candidatesJSON), &candidates); err != nil {
		return fmt.Errorf("failed to parse candidates: %v", err)
	}

	election := Election{
		ElectionID:  electionID,
		Title:       title,
		Description: description,
		Candidates:  candidates,
		StartTime:   startTime,
		EndTime:     endTime,
		IsActive:    startTime <= time.Now().Unix(),
		CreatedAt:   time.Now().Unix(),
	}

	electionJSON, err := json.Marshal(election)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(electionID, electionJSON)
}

// CastVote records a vote on the blockchain.
// ACCESS: Any authenticated member may cast a vote.
func (vc *VoteChaincode) CastVote(
	ctx contractapi.TransactionContextInterface,
	voteID string,
	electionID string,
	encryptedVote string,
	ringSignature string,
	keyImage string,
	smdcCommitment string,
	merkleProof string,
) error {
	// --- Access control: authenticated member ---
	callerID, err := vc.requireAuthenticated(ctx)
	if err != nil {
		return err
	}

	// Check if vote already exists
	exists, err := vc.VoteExists(ctx, voteID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("vote %s already exists", voteID)
	}

	// Check if key image was already used (double-vote detection)
	keyImageKey := "keyimage_" + keyImage
	keyImageBytes, err := ctx.GetStub().GetState(keyImageKey)
	if err != nil {
		return fmt.Errorf("failed to read key image: %v", err)
	}
	if keyImageBytes != nil {
		return fmt.Errorf("double-vote detected: key image already used")
	}

	// Create vote record
	vote := Vote{
		VoteID:         voteID,
		ElectionID:     electionID,
		EncryptedVote:  encryptedVote,
		RingSignature:  ringSignature,
		KeyImage:       keyImage,
		SMDCCommitment: smdcCommitment,
		MerkleProof:    merkleProof,
		Timestamp:      time.Now().Unix(),
		OwnerID:        callerID,
	}

	voteJSON, err := json.Marshal(vote)
	if err != nil {
		return err
	}

	// Store vote
	if err := ctx.GetStub().PutState(voteID, voteJSON); err != nil {
		return err
	}

	// Mark key image as used
	if err := ctx.GetStub().PutState(keyImageKey, []byte(voteID)); err != nil {
		return err
	}

	// Emit event
	ctx.GetStub().SetEvent("VoteCast", voteJSON)

	return nil
}

// GetVote returns the vote stored in the world state with given id.
// ACCESS: Only the vote owner or admin org may read a specific vote.
func (vc *VoteChaincode) GetVote(ctx contractapi.TransactionContextInterface, voteID string) (*Vote, error) {
	voteJSON, err := ctx.GetStub().GetState(voteID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if voteJSON == nil {
		return nil, fmt.Errorf("vote %s does not exist", voteID)
	}

	var vote Vote
	err = json.Unmarshal(voteJSON, &vote)
	if err != nil {
		return nil, err
	}

	// --- Access control: owner or admin ---
	admin, err := vc.isAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if !admin {
		callerID, err := vc.requireAuthenticated(ctx)
		if err != nil {
			return nil, err
		}
		if callerID != vote.OwnerID {
			return nil, fmt.Errorf("access denied: only the vote owner or admin org can read this vote")
		}
	}

	return &vote, nil
}

// GetElection returns the election stored in the world state with given id
func (vc *VoteChaincode) GetElection(ctx contractapi.TransactionContextInterface, electionID string) (*Election, error) {
	electionJSON, err := ctx.GetStub().GetState(electionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if electionJSON == nil {
		return nil, fmt.Errorf("election %s does not exist", electionID)
	}

	var election Election
	err = json.Unmarshal(electionJSON, &election)
	if err != nil {
		return nil, err
	}

	return &election, nil
}

// GetAllVotes returns vote summaries (redacted) for all votes in world state.
// ACCESS: Only the admin org (ElectionCommissionMSP) may call this function.
// Returns VoteSummary (redacted) to limit exposure of privacy-sensitive fields.
func (vc *VoteChaincode) GetAllVotes(ctx contractapi.TransactionContextInterface) ([]*VoteSummary, error) {
	// --- Access control: admin only ---
	if err := vc.requireAdmin(ctx); err != nil {
		return nil, err
	}

	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var summaries []*VoteSummary
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var vote Vote
		err = json.Unmarshal(queryResponse.Value, &vote)
		if err != nil {
			continue
		}

		// Only include actual vote records (skip elections, tallies, key images, etc.)
		if vote.VoteID == "" {
			continue
		}

		summaries = append(summaries, &VoteSummary{
			VoteID:     vote.VoteID,
			ElectionID: vote.ElectionID,
			Timestamp:  vote.Timestamp,
		})
	}

	return summaries, nil
}

// GetVotesByElection returns all votes for a specific election.
// ACCESS: Only the admin org (ElectionCommissionMSP) may call this function.
func (vc *VoteChaincode) GetVotesByElection(ctx contractapi.TransactionContextInterface, electionID string) ([]*Vote, error) {
	// --- Access control: admin only ---
	if err := vc.requireAdmin(ctx); err != nil {
		return nil, err
	}

	// Build query safely using json.Marshal instead of string interpolation
	// to prevent CouchDB injection.
	query := couchDBQuery{
		Selector: map[string]interface{}{
			"election_id": electionID,
		},
	}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %v", err)
	}

	return vc.getQueryResultForQueryString(ctx, string(queryBytes))
}

// getQueryResultForQueryString executes the passed in query string
func (vc *VoteChaincode) getQueryResultForQueryString(ctx contractapi.TransactionContextInterface, queryString string) ([]*Vote, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var votes []*Vote
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var vote Vote
		err = json.Unmarshal(queryResponse.Value, &vote)
		if err != nil {
			return nil, err
		}
		votes = append(votes, &vote)
	}

	return votes, nil
}

// StoreTallyResult stores tally results on the blockchain.
// ACCESS: Only the admin org (ElectionCommissionMSP) may call this function.
func (vc *VoteChaincode) StoreTallyResult(
	ctx contractapi.TransactionContextInterface,
	electionID string,
	candidateTalliesJSON string,
	totalVotes int,
) error {
	// --- Access control: admin only ---
	if err := vc.requireAdmin(ctx); err != nil {
		return err
	}

	// Parse candidate tallies
	var candidateTallies map[string]int
	if err := json.Unmarshal([]byte(candidateTalliesJSON), &candidateTallies); err != nil {
		return fmt.Errorf("failed to parse candidate tallies: %v", err)
	}

	result := TallyResult{
		ElectionID:       electionID,
		CandidateTallies: candidateTallies,
		TotalVotes:       totalVotes,
		TallyTime:        time.Now().Unix(),
		Verified:         true,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}

	tallyKey := "tally_" + electionID
	return ctx.GetStub().PutState(tallyKey, resultJSON)
}

// GetTallyResult returns the tally result for a specific election
func (vc *VoteChaincode) GetTallyResult(ctx contractapi.TransactionContextInterface, electionID string) (*TallyResult, error) {
	tallyKey := "tally_" + electionID
	resultJSON, err := ctx.GetStub().GetState(tallyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read tally result: %v", err)
	}
	if resultJSON == nil {
		return nil, fmt.Errorf("tally result for election %s does not exist", electionID)
	}

	var result TallyResult
	err = json.Unmarshal(resultJSON, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// VoteExists returns true when vote with given ID exists in world state
func (vc *VoteChaincode) VoteExists(ctx contractapi.TransactionContextInterface, voteID string) (bool, error) {
	voteJSON, err := ctx.GetStub().GetState(voteID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return voteJSON != nil, nil
}

// ElectionExists returns true when election with given ID exists in world state
func (vc *VoteChaincode) ElectionExists(ctx contractapi.TransactionContextInterface, electionID string) (bool, error) {
	electionJSON, err := ctx.GetStub().GetState(electionID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return electionJSON != nil, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&VoteChaincode{})
	if err != nil {
		fmt.Printf("Error creating CovertVote chaincode: %v", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting CovertVote chaincode: %v", err)
	}
}
