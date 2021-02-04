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

func Genesis() *Block {
	return CreateBlock([]*Transaction{}, []byte{})
}

func CreateBlock(txs []*Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0}
	// pow := NewProof(block)
	// nonce, hash := pow.Run()

	// block.Nonce = nonce
	// block.Hash = hash
	return block
}

func (b *Block) HashTx() []byte {

}

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
