package blockchain

import "crypto/sha256"

type (
	MerkleTree struct {
		RootNode *MerkleNode
	}

	//recursive tree structure
	MerkleNode struct {
		Left  *MerkleNode
		Right *MerkleNode
		Data  []byte
	}
)

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	return &node
}

func NewMerkleTree(data [][]byte) *MerkleTree {
	var (
		nodes []MerkleNode
	)

	//check to make the Merkle tree leaves even
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	//adding data to array but not on Merkle Tree format
	for _, dat := range data {
		node := NewMerkleNode(nil, nil, dat)
		nodes = append(nodes, *node)
	}

	//Making the data on array to Merkle Tree Format
	for i := 0; i < len(data)/2; i++ {
		var level []MerkleNode

		for k := 0; k < len(nodes); k += 2 {
			node := NewMerkleNode(&nodes[k], &nodes[k+1], nil)
			level = append(level, *node)
		}

		nodes = level
	}

	tree := MerkleTree{&nodes[0]}

	return &tree
}
