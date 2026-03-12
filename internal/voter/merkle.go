package voter

import (
	"bytes"
	"crypto/sha256"
	"errors"
)

// MerkleTree represents a Merkle tree for voter eligibility
type MerkleTree struct {
	Root   []byte
	Leaves [][]byte
}

// NewMerkleTree creates a new Merkle tree from voter IDs
func NewMerkleTree(voterIDs []string) *MerkleTree {
	if len(voterIDs) == 0 {
		return &MerkleTree{
			Root:   nil,
			Leaves: [][]byte{},
		}
	}

	// Create leaf nodes (hash of voter IDs)
	leaves := make([][]byte, len(voterIDs))
	for i, id := range voterIDs {
		hash := sha256.Sum256([]byte(id))
		leaves[i] = hash[:]
	}

	root := buildMerkleRoot(leaves)

	return &MerkleTree{
		Root:   root,
		Leaves: leaves,
	}
}

// buildMerkleRoot builds the Merkle root from leaves
func buildMerkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return nil
	}

	if len(leaves) == 1 {
		return leaves[0]
	}

	// Build next level
	var nextLevel [][]byte

	for i := 0; i < len(leaves); i += 2 {
		if i+1 < len(leaves) {
			// Pair exists
			combined := append(leaves[i], leaves[i+1]...)
			hash := sha256.Sum256(combined)
			nextLevel = append(nextLevel, hash[:])
		} else {
			// Odd node, promote it
			nextLevel = append(nextLevel, leaves[i])
		}
	}

	return buildMerkleRoot(nextLevel)
}

// GetProof generates a Merkle proof for a voter ID
func (mt *MerkleTree) GetProof(voterID string) ([][]byte, error) {
	// Find the leaf
	targetHash := sha256.Sum256([]byte(voterID))
	leafIndex := -1

	for i, leaf := range mt.Leaves {
		if bytes.Equal(leaf, targetHash[:]) {
			leafIndex = i
			break
		}
	}

	if leafIndex == -1 {
		return nil, errors.New("voter ID not found in tree")
	}

	// Build proof path
	proof := [][]byte{}
	currentLevel := mt.Leaves
	index := leafIndex

	for len(currentLevel) > 1 {
		// Find sibling
		var sibling []byte
		if index%2 == 0 {
			// Left node, sibling is right
			if index+1 < len(currentLevel) {
				sibling = currentLevel[index+1]
			}
		} else {
			// Right node, sibling is left
			sibling = currentLevel[index-1]
		}

		if sibling != nil {
			proof = append(proof, sibling)
		}

		// Move to next level
		nextLevel := [][]byte{}
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				combined := append(currentLevel[i], currentLevel[i+1]...)
				hash := sha256.Sum256(combined)
				nextLevel = append(nextLevel, hash[:])
			} else {
				nextLevel = append(nextLevel, currentLevel[i])
			}
		}

		currentLevel = nextLevel
		index = index / 2
	}

	return proof, nil
}

// VerifyProof verifies a Merkle proof
func VerifyProof(voterID string, proof [][]byte, root []byte) bool {
	// Start with leaf hash
	currentHash := sha256.Sum256([]byte(voterID))

	// Apply proof path
	for _, sibling := range proof {
		// Determine order (smaller hash first for determinism)
		var combined []byte
		if bytes.Compare(currentHash[:], sibling) < 0 {
			combined = append(currentHash[:], sibling...)
		} else {
			combined = append(sibling, currentHash[:]...)
		}

		hash := sha256.Sum256(combined)
		currentHash = hash
	}

	return bytes.Equal(currentHash[:], root)
}

// IsEligible checks if a voter is eligible
func (mt *MerkleTree) IsEligible(voterID string) bool {
	targetHash := sha256.Sum256([]byte(voterID))

	for _, leaf := range mt.Leaves {
		if bytes.Equal(leaf, targetHash[:]) {
			return true
		}
	}

	return false
}
