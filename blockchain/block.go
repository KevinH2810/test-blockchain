package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
)

type (
	Block struct {
		Hash        []byte
		Transaction []*Transaction
		PrevHash    []byte
		Height      int
		Validator   string
		Timestamp   int64
	}
)

func CreateBlock(txs []*Transaction, prevHash []byte, Validator string, height int) *Block {
	block := &Block{[]byte{}, txs, prevHash, height, Validator, time.Now().Unix()}
	//delete this and make function to generate Transactionhash
	//Txhash is formed by SHA(Tx.ID) + Tx data
	// pow := NewProof(block)
	// nonce, hash := pow.Run()

	// block.Nonce = nonce
	// block.Hash = hash
	return block
}

func (b *Block) BlockHashing() []byte {
	data := bytes.Join(
		[][]byte{
			b.PrevHash,
			b.HashTransactions(),
		},
		[]byte{},
	)

	hash := sha256.Sum256(data)

	return hash[:]
}

func Genesis(coinbase *Transaction, validator string) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{}, validator, 0)
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
