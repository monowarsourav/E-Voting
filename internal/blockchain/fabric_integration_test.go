package blockchain

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

// TestRealFabricIntegration tests the full blockchain integration against a
// running Hyperledger Fabric network. Skip if the network is not running.
//
// Prerequisites:
//   - docker compose -f network/docker-compose-hf.yml up -d
//   - Chaincode "covertvote" deployed on channel "covertvotechannel"
func TestRealFabricIntegration(t *testing.T) {
	// Determine the project root (4 levels up from this test file).
	networkDir := findNetworkDir()
	if networkDir == "" {
		t.Skip("Skipping: could not find network/ directory")
	}

	tlsCertPath := filepath.Join(networkDir,
		"crypto-config/peerOrganizations/org1.covertvote.com/peers/peer0.org1.covertvote.com/tls/ca.crt")
	certPath := filepath.Join(networkDir,
		"crypto-config/peerOrganizations/org1.covertvote.com/users/Admin@org1.covertvote.com/msp/signcerts/Admin@org1.covertvote.com-cert.pem")
	keyPath := filepath.Join(networkDir,
		"crypto-config/peerOrganizations/org1.covertvote.com/users/Admin@org1.covertvote.com/msp/keystore")

	// Check if crypto materials exist
	if _, err := os.Stat(tlsCertPath); os.IsNotExist(err) {
		t.Skip("Skipping: TLS cert not found — Fabric network not set up")
	}

	fc := NewFabricClient("covertvotechannel", "covertvote", true)
	err := fc.ConnectGateway(FabricConfig{
		PeerEndpoint: "localhost:7051",
		GatewayPeer:  "peer0.org1.covertvote.com",
		MSPID:        "Org1MSP",
		TLSCertPath:  tlsCertPath,
		CertPath:     certPath,
		KeyPath:      keyPath,
	})
	if err != nil {
		t.Skipf("Skipping: cannot connect to Fabric: %v", err)
	}
	defer fc.Disconnect()

	t.Log("✅ Connected to real Hyperledger Fabric network")

	// Test 1: GetElection (created earlier via CLI)
	t.Run("GetElection", func(t *testing.T) {
		result, err := fc.GetTallyResult("election001")
		// This may fail if no tally was stored — that's OK
		if err != nil {
			t.Logf("GetTallyResult (expected if no tally): %v", err)
		} else {
			t.Logf("Tally result: %+v", result)
		}
	})

	// Test 2: GetVote (vote001 created via CLI)
	t.Run("GetVote", func(t *testing.T) {
		vote, err := fc.GetVote("vote001")
		if err != nil {
			t.Fatalf("GetVote failed: %v", err)
		}
		if vote.VoteID != "vote001" {
			t.Fatalf("Expected vote001, got %s", vote.VoteID)
		}
		if vote.ElectionID != "election001" {
			t.Fatalf("Expected election001, got %s", vote.ElectionID)
		}
		t.Logf("✅ Retrieved vote from blockchain: %+v", vote)
	})

	// Test 3: CastVote via SDK
	t.Run("CastVoteSDK", func(t *testing.T) {
		voteID := fmt.Sprintf("sdk-vote-%d", os.Getpid())
		txID, err := fc.SubmitVote(
			voteID,
			"election001",
			newBigInt(12345),
			nil, // ring signature not needed for blockchain test
			newBigInt(67890),
			newBigInt(11111),
			[][]byte{{0x01, 0x02}},
		)
		// CastVote marshals nil ringSignature as "null" — chaincode stores it
		if err != nil {
			t.Fatalf("CastVote via SDK failed: %v", err)
		}
		t.Logf("✅ Vote submitted via SDK, txID: %s", txID)

		// Verify the vote was stored
		vote, err := fc.GetVote(voteID)
		if err != nil {
			t.Fatalf("GetVote after SDK cast failed: %v", err)
		}
		if vote.VoteID != voteID {
			t.Fatalf("Expected %s, got %s", voteID, vote.VoteID)
		}
		t.Logf("✅ Vote verified on blockchain: %s", vote.VoteID)
	})

	// Test 4: VerifyVote
	t.Run("VerifyVote", func(t *testing.T) {
		exists, err := fc.VerifyVote("vote001")
		if err != nil {
			t.Fatalf("VerifyVote failed: %v", err)
		}
		if !exists {
			t.Fatal("vote001 should exist on blockchain")
		}
		t.Log("✅ Vote verified on blockchain")
	})
}

func findNetworkDir() string {
	// Try common paths
	candidates := []string{
		"../../network",
		"../../../network",
		"/home/bs01582/E-voting/network",
	}
	for _, c := range candidates {
		abs, _ := filepath.Abs(c)
		if _, err := os.Stat(filepath.Join(abs, "docker-compose-hf.yml")); err == nil {
			return abs
		}
	}
	return ""
}

func newBigInt(v int64) *big.Int {
	return big.NewInt(v)
}
