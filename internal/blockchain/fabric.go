package blockchain

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/voting"
)

// FabricClient represents a Hyperledger Fabric client
type FabricClient struct {
	ChannelName   string
	ChaincodeName string
	Enabled       bool
	MockMode      bool // If true, use mock implementation
	chaincode     ChaincodeInterface

	// Real Fabric Gateway SDK fields
	gateway  *client.Gateway
	contract *client.Contract
	grpcConn *grpc.ClientConn
}

// FabricConfig holds configuration for connecting to a real Fabric network.
type FabricConfig struct {
	// PeerEndpoint is the gRPC address of the peer (e.g. "localhost:7051").
	PeerEndpoint string
	// GatewayPeer is the TLS hostname override for the peer.
	GatewayPeer string
	// MSPID is the MSP ID of the connecting org (e.g. "Org1MSP").
	MSPID string
	// CryptoPath is the root path to the org's crypto-config material.
	CryptoPath string
	// TLSCertPath is the path to the peer's TLS CA certificate.
	TLSCertPath string
	// CertPath is the path to the user's signing certificate.
	CertPath string
	// KeyPath is the path to the directory containing the user's private key.
	KeyPath string
}

// NewFabricClient creates a new Fabric client
func NewFabricClient(channelName, chaincodeName string, enabled bool) *FabricClient {
	fc := &FabricClient{
		ChannelName:   channelName,
		ChaincodeName: chaincodeName,
		Enabled:       enabled,
		MockMode:      true, // Default to mock mode
	}

	// Initialize mock chaincode by default
	fc.chaincode = NewMockChaincode()

	return fc
}

// ConnectGateway connects to a real Hyperledger Fabric network using the
// Fabric Gateway SDK (recommended for Fabric 2.4+).
func (fc *FabricClient) ConnectGateway(cfg FabricConfig) error {
	if !fc.Enabled {
		fmt.Println("Blockchain integration is disabled")
		return nil
	}

	// 1. Load peer TLS certificate
	pemBytes, err := os.ReadFile(cfg.TLSCertPath)
	if err != nil {
		return fmt.Errorf("read TLS cert: %w", err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemBytes) {
		return fmt.Errorf("failed to add TLS cert to pool")
	}

	// 2. Create gRPC connection to the peer
	transportCreds := credentials.NewClientTLSFromCert(certPool, cfg.GatewayPeer)
	conn, err := grpc.NewClient(cfg.PeerEndpoint, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		return fmt.Errorf("grpc connect: %w", err)
	}
	fc.grpcConn = conn

	// 3. Load user identity (X.509 certificate)
	certPEM, err := os.ReadFile(cfg.CertPath)
	if err != nil {
		return fmt.Errorf("read user cert: %w", err)
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return fmt.Errorf("parse user cert: %w", err)
	}
	id, err := identity.NewX509Identity(cfg.MSPID, cert)
	if err != nil {
		return fmt.Errorf("create identity: %w", err)
	}

	// 4. Load user private key for signing
	keyPEM, err := readFirstFile(cfg.KeyPath)
	if err != nil {
		return fmt.Errorf("read user key: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return fmt.Errorf("create signer: %w", err)
	}

	// 5. Connect to the Fabric Gateway
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("gateway connect: %w", err)
	}
	fc.gateway = gw

	// 6. Get network (channel) and contract (chaincode)
	network := gw.GetNetwork(fc.ChannelName)
	fc.contract = network.GetContract(fc.ChaincodeName)
	fc.MockMode = false

	fmt.Printf("✅ Connected to Hyperledger Fabric: channel=%s, chaincode=%s, peer=%s\n",
		fc.ChannelName, fc.ChaincodeName, cfg.PeerEndpoint)

	return nil
}

// Connect connects to the Fabric network (legacy interface — uses mock mode).
func (fc *FabricClient) Connect(connectionProfile, walletPath, identityName string) error {
	if !fc.Enabled {
		fmt.Println("Blockchain integration is disabled")
		return nil
	}
	fmt.Println("Running in MOCK mode — use ConnectGateway for real Fabric")
	fc.chaincode = NewMockChaincode()
	return nil
}

// CreateElection creates a new election on the blockchain.
// The chaincode persists candidates as a flat list of names ([]string), so we
// flatten the (ID, Name, ...) candidate structs into a name-only slice for the
// chain payload. The richer in-process Candidate metadata stays off-chain.
func (fc *FabricClient) CreateElection(election *voting.Election) (string, error) {
	candidateNames := make([]string, len(election.Candidates))
	for i, c := range election.Candidates {
		candidateNames[i] = c.Name
	}
	candidatesJSON, err := json.Marshal(candidateNames)
	if err != nil {
		return "", fmt.Errorf("marshal candidates: %v", err)
	}

	if !fc.MockMode && fc.contract != nil {
		// REAL Fabric transaction
		result, err := fc.contract.SubmitTransaction(
			"CreateElection",
			election.ElectionID,
			election.Title,
			election.Description,
			string(candidatesJSON),
			fmt.Sprintf("%d", election.StartTime),
			fmt.Sprintf("%d", election.EndTime),
		)
		if err != nil {
			return "", fmt.Errorf("submit CreateElection: %w", err)
		}
		txID := string(result)
		if txID == "" {
			txID = "tx-" + election.ElectionID
		}
		return txID, nil
	}

	// Mock fallback
	fmt.Printf("Creating election on blockchain (mock): %s\n", election.ElectionID)
	_ = candidatesJSON
	return "mock-tx-" + election.ElectionID, nil
}

// SubmitVote submits a vote to the blockchain. encryptedVotes is the
// per-candidate weighted Paillier ciphertext vector E_w_j; it is serialised as
// a JSON array of decimal strings so the chaincode can store all m positions
// without losing structure.
func (fc *FabricClient) SubmitVote(
	voteID string,
	electionID string,
	encryptedVotes []*big.Int,
	ringSignature *crypto.RingSignature,
	keyImage *big.Int,
	smdcCommitment *big.Int,
	merkleProof [][]byte,
) (string, error) {
	encryptedVoteStrs := make([]string, len(encryptedVotes))
	for j, e := range encryptedVotes {
		if e != nil {
			encryptedVoteStrs[j] = e.String()
		}
	}
	encryptedVotesJSON, _ := json.Marshal(encryptedVoteStrs)
	ringSignatureJSON, _ := json.Marshal(ringSignature)
	keyImageStr := keyImage.String()
	smdcCommitmentStr := smdcCommitment.String()
	merkleProofJSON, _ := json.Marshal(merkleProof)

	if !fc.MockMode && fc.contract != nil {
		// REAL Fabric transaction
		result, err := fc.contract.SubmitTransaction(
			"CastVote",
			voteID,
			electionID,
			string(encryptedVotesJSON),
			string(ringSignatureJSON),
			keyImageStr,
			smdcCommitmentStr,
			string(merkleProofJSON),
		)
		if err != nil {
			return "", fmt.Errorf("submit CastVote: %w", err)
		}
		txID := string(result)
		if txID == "" {
			txID = "tx-vote-" + voteID
		}
		return txID, nil
	}

	// Mock fallback
	fmt.Printf("Submitting vote to blockchain (mock): %s\n", voteID)
	_ = encryptedVotesJSON
	_ = ringSignatureJSON
	_ = keyImageStr
	_ = smdcCommitmentStr
	_ = merkleProofJSON
	return "mock-tx-" + voteID, nil
}

// GetVote retrieves a vote from the blockchain
func (fc *FabricClient) GetVote(voteID string) (*BlockchainVote, error) {
	if !fc.MockMode && fc.contract != nil {
		// REAL Fabric query
		result, err := fc.contract.EvaluateTransaction("GetVote", voteID)
		if err != nil {
			return nil, fmt.Errorf("evaluate GetVote: %w", err)
		}
		var vote BlockchainVote
		if err := json.Unmarshal(result, &vote); err != nil {
			return nil, fmt.Errorf("unmarshal vote: %w", err)
		}
		return &vote, nil
	}

	// Mock fallback
	return &BlockchainVote{
		VoteID:     voteID,
		ElectionID: "election001",
		Timestamp:  0,
	}, nil
}

// GetVotesByElection retrieves all votes for an election
func (fc *FabricClient) GetVotesByElection(electionID string) ([]*BlockchainVote, error) {
	if !fc.MockMode && fc.contract != nil {
		result, err := fc.contract.EvaluateTransaction("GetVotesByElection", electionID)
		if err != nil {
			return nil, fmt.Errorf("evaluate GetVotesByElection: %w", err)
		}
		var votes []*BlockchainVote
		if err := json.Unmarshal(result, &votes); err != nil {
			return nil, fmt.Errorf("unmarshal votes: %w", err)
		}
		return votes, nil
	}

	return []*BlockchainVote{}, nil
}

// StoreTallyResult stores tally results on the blockchain
func (fc *FabricClient) StoreTallyResult(
	electionID string,
	candidateTallies map[int]int64,
	totalVotes int64,
) (string, error) {
	talliesJSON, err := json.Marshal(candidateTallies)
	if err != nil {
		return "", err
	}

	if !fc.MockMode && fc.contract != nil {
		// REAL Fabric transaction
		result, err := fc.contract.SubmitTransaction(
			"StoreTallyResult",
			electionID,
			string(talliesJSON),
			fmt.Sprintf("%d", totalVotes),
		)
		if err != nil {
			return "", fmt.Errorf("submit StoreTallyResult: %w", err)
		}
		txID := string(result)
		if txID == "" {
			txID = "tx-tally-" + electionID
		}
		return txID, nil
	}

	// Mock fallback
	fmt.Printf("Storing tally result on blockchain (mock): %s\n", electionID)
	_ = talliesJSON
	return "mock-tx-tally-" + electionID, nil
}

// GetTallyResult retrieves tally results from the blockchain
func (fc *FabricClient) GetTallyResult(electionID string) (*BlockchainTallyResult, error) {
	if !fc.MockMode && fc.contract != nil {
		result, err := fc.contract.EvaluateTransaction("GetTallyResult", electionID)
		if err != nil {
			return nil, fmt.Errorf("evaluate GetTallyResult: %w", err)
		}
		var tally BlockchainTallyResult
		if err := json.Unmarshal(result, &tally); err != nil {
			return nil, fmt.Errorf("unmarshal tally: %w", err)
		}
		return &tally, nil
	}

	return &BlockchainTallyResult{
		ElectionID:       electionID,
		CandidateTallies: make(map[string]int),
		TotalVotes:       0,
		Verified:         true,
	}, nil
}

// VerifyVote verifies a vote exists on the blockchain
func (fc *FabricClient) VerifyVote(voteID string) (bool, error) {
	if !fc.MockMode && fc.contract != nil {
		result, err := fc.contract.EvaluateTransaction("VoteExists", voteID)
		if err != nil {
			return false, fmt.Errorf("evaluate VoteExists: %w", err)
		}
		return string(result) == "true", nil
	}

	return true, nil
}

// Disconnect closes the connection to the Fabric network
func (fc *FabricClient) Disconnect() error {
	if fc.gateway != nil {
		fc.gateway.Close()
	}
	if fc.grpcConn != nil {
		fc.grpcConn.Close()
	}
	fmt.Println("Disconnected from Hyperledger Fabric network")
	return nil
}

// readFirstFile reads the first file in the given directory.
// This is used for the keystore directory which contains a single private key.
func readFirstFile(dir string) ([]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return os.ReadFile(filepath.Join(dir, entry.Name()))
		}
	}
	return nil, fmt.Errorf("no files found in %s", dir)
}

// BlockchainVote represents a vote stored on the blockchain
type BlockchainVote struct {
	VoteID         string `json:"vote_id"`
	ElectionID     string `json:"election_id"`
	EncryptedVote  string `json:"encrypted_vote"`
	RingSignature  string `json:"ring_signature"`
	KeyImage       string `json:"key_image"`
	SMDCCommitment string `json:"smdc_commitment"`
	MerkleProof    string `json:"merkle_proof"`
	Timestamp      int64  `json:"timestamp"`
	BlockNumber    uint64 `json:"block_number"`
}

// BlockchainTallyResult represents tally results on the blockchain
type BlockchainTallyResult struct {
	ElectionID       string         `json:"election_id"`
	CandidateTallies map[string]int `json:"candidate_tallies"`
	TotalVotes       int            `json:"total_votes"`
	TallyTime        int64          `json:"tally_time"`
	Verified         bool           `json:"verified"`
}
