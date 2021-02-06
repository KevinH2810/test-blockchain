package blockchain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dgraph-io/badger"
)

type (
	Blockchain struct {
		LastHash []byte
		Database *badger.DB
	}
)

const (
	initialCoin = 20
)

var (
	dbPath      = "./tmp/blocks_%s"
	genesisData = "First Transaction from Genesis"
)

func DBexists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}

	return true
}

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	return db, err
}

func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	} else {
		return db, nil
	}
}

func InitBlockchain(address, nodeID string) *Blockchain {
	var lastHash []byte

	path := fmt.Sprintf(dbPath, nodeID)
	if DBexists(path) {
		fmt.Println("Blockchain Already Exists")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := openDB(path, opts)
	Handler(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData, initialCoin)
		genesis := Genesis(cbtx, address)
		hash := genesis.BlockHashing()
		genesis.Hash = hash[:]
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

func NormalBlockchainProcess(nodeID string) *Blockchain {
	var lastHash []byte

	path := fmt.Sprintf(dbPath, nodeID)
	if DBexists(path) == false {
		fmt.Println("no existing blockchain found")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	db, err := openDB(path, opts)
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

func (chain *Blockchain) ForgeBlock(transactions []*Transaction, Sender string) {
	var (
		lasthash   []byte
		lastHeight int
	)

	for _, tx := range transactions {
		if chain.VerifyTransaction(tx) != true {
			log.Panic("Invalid Transaction")
		}
	}

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handler(err)
		lastHash, err := item.Value()

		item, err = txn.Get(lastHash)
		Handler(err)
		lastBlockData, _ := item.Value()

		lastBlock := Deserialize(lastBlockData)
		lastHeight = lastBlock.Height

		return err
	})
	Handler(err)

	err = chain.Database.View(func(txn *badger.Txn) error {
		data, err := txn.Get([]byte("lh"))
		Handler(err)
		lasthash, err = data.Value()

		return err
	})

	Handler(err)

	newBlock := CreateBlock(transactions, lasthash, Sender, lastHeight+1)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handler(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})

	Handler(err)
}

func (chain *Blockchain) AddBlock(block *Block) {
	err := chain.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err == nil {
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		Handler(err)

		item, err := txn.Get([]byte("lh"))
		Handler(err)
		lastHash, _ := item.Value()

		item, err = txn.Get(lastHash)
		Handler(err)
		lastBlockData, _ := item.Value()

		lastBlock := Deserialize(lastBlockData)

		if block.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), block.Hash)
			Handler(err)
			chain.LastHash = block.Hash
		}

		return nil
	})
	Handler(err)
}

func (chain *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var (
		block Block
	)

	err := chain.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block is not found")
		} else {
			blockData, _ := item.Value()

			block = *Deserialize(blockData)
		}
		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

func (chain *Blockchain) GetBlockHashes() [][]byte {
	var (
		blocks [][]byte
	)

	iter := chain.Iterate()

	for {
		block := iter.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

func (chain *Blockchain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)
	spentTXOs := make(map[string][]int)

	iter := chain.Iterate()

	for {
		block := iter.Next()

		for _, tx := range block.Transaction {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}
			if tx.isCoinbase() == false {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.ID)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
	return UTXO
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

func (chain *Blockchain) GetLastHeight() int {
	var lastBlock Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handler(err)
		lastHash, _ := item.Value()

		item, err = txn.Get(lastHash)
		Handler(err)
		lastBlockData, _ := item.Value()

		lastBlock = *Deserialize(lastBlockData)

		return nil
	})
	Handler(err)

	return lastBlock.Height
}
