package blockchain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger"
)

type (
	Blockchain struct {
		LastHash []byte
		Database *badger.DB
	}

	BlockchainIterate struct {
		CurrentHash []byte
		Database    *badger.DB
	}
)

var (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction From Genesis"
)

func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func InitBlockchain(address string) *Blockchain {
	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain Already Exists")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	Handler(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handler(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash

		return err
	})

	Handler(err)

	blockchain := Blockchain{lastHash, db}
	return &blockchain
}

func NormalBlockchainProcess(address string) *Blockchain {
	var lastHash []byte

	if DBexists() == false {
		fmt.Println("no existing blockchain found")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := badger.Open(opts)
	Handler(err)

	err = db.Update(func(txn *badger.Txn) error {
		data, err := txn.Get([]byte("lh"))
		Handler(err)

		lastHash, err = data.Value()
		return err
	})

	Handler(err)

	chain := Blockchain{lastHash, db}

	return &chain
}

func (chain *Blockchain) FindUnspentTransactions(PubKeyHash []byte) []Transaction {
	var unspentTx []Transaction

	spentTXs := make(map[string][]int)

	iter := chain.Iterate()

	for {

		block := iter.Next()

		for _, tx := range block.Transaction {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXs[txID] != nil {
					for _, spentOut := range spentTXs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.IsLockedWithKey(PubKeyHash) {
					unspentTx = append(unspentTx, *tx)
				}
			}
			if tx.isCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.UsesKey(PubKeyHash) {
						inTxId := hex.EncodeToString(in.ID)
						spentTXs[inTxId] = append(spentTXs[inTxId], in.Out)
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTx
}

func (chain *Blockchain) AddBlock(transactions []*Transaction) {
	var lasthash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		data, err := txn.Get([]byte("lh"))
		Handler(err)
		lasthash, err = data.Value()

		return err
	})

	Handler(err)

	newBlock := CreateBlock(transactions, lasthash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handler(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})

	Handler(err)
}

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

func (chain *Blockchain) FindUTXO(PubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	unspentTransaction := chain.FindUnspentTransactions(PubKeyHash)

	for _, tx := range unspentTransaction {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(PubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (chain *Blockchain) FindSpendableOutputs(PubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(PubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			//to validate so the user wont be able to sent money if they didnt have enough balances
			if out.IsLockedWithKey(PubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}

func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	iter := bc.Iterate()

	for {
		block := iter.Next()

		for _, tx := range block.Transaction {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction does not exist")
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {

	if tx.isCoinbase() {
		return true
	}

	prevTxs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTx, err := bc.FindTransaction(in.ID)
		Handler(err)
		prevTxs[hex.EncodeToString(prevTx.ID)] = prevTx
	}

	return tx.Verify(prevTxs)
}
