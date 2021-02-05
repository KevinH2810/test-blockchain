package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
)

type (
	Block struct {
		Hash        []byte
		Transaction []*Transaction
		PrevHash    []byte
		Nonce       int
	}
)

func CreateBlock(txs []*Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash
	return block
}

func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{})
}

//turn Block to []byte data
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)

	if err != nil {
		log.Panic(err)
	}

	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block

	//decode the data
	decoder := gob.NewDecoder(bytes.NewReader(data))

	//pass the decoded data to block
	err := decoder.Decode(&block)

	Handler(err)

	return &block
}

func (b *Block) HashTransactions() []byte {
	var (
		txHashes [][]byte
	)

	for _, tx := range b.Transaction {
		txHashes = append(txHashes, tx.Serialize())
	}

	tree := NewMerkleTree(txHashes)

	return tree.RootNode.Data
}
