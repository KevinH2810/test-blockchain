package blockchain

import "github.com/dgraph-io/badger"

type (
	BlockchainIterate struct {
		CurrentHash []byte
		Database    *badger.DB
	}
)

func (chain *Blockchain) Iterate() *BlockchainIterate {

	iter := &BlockchainIterate{chain.LastHash, chain.Database}
	return iter
}

func (iter *BlockchainIterate) Next() *Block {
	var (
		block        *Block
		encodedBlock []byte
	)

	err := iter.Database.View(func(txn *badger.Txn) error {
		//get the block based on currenthash
		data, err := txn.Get(iter.CurrentHash)
		encodedBlock, err = data.Value()
		block = Deserialize(encodedBlock)

		return err
	})

	Handler(err)

	iter.CurrentHash = block.PrevHash

	return block
}
