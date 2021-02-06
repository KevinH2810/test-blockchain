package network

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/test-blockchain/blockchain"
)

type (
	lottpool struct {
		address string
		amount  int
	}

	ProofOfStake struct {
		lastHeight int
		lastHash   []byte
	}
)

var (
	mutex      = &sync.Mutex{}
	pendingTxs []*blockchain.Transaction
)

func PickWinner() {
	time.Sleep(30 * time.Second)
	mutex.Lock()
	//lock the variables
	for _, tempId := range tempTxPool {
		tx := memoryPool[tempId]
		if currentChain.VerifyTransaction(&tx) {
			pendingTxs = append(pendingTxs, &tx)
		}

	}

	temp := tempStakeTxPool
	mutex.Unlock()

	lotterypool := []string{}
	if len(temp) > 0 {

		// slightly modified traditional proof of stake algorithm
		// from all validators who submitted a transaction, weight them by the number of staked token
		// in traditional proof of stake, validators can participate without submitting a block to be forged
	OUTER:
		//check each Stake transactions if already in the lottery pool
		for _, tx := range temp {
			// if already in lottery pool, skip
			for _, LotteryTx := range lotterypool {
				if tx.Outputs[0].Address == LotteryTx {
					continue OUTER
				}
			}

			// lock list of validators to prevent data race
			mutex.Lock()
			setValidators := validator
			mutex.Unlock()

			k, ok := setValidators[tx.Inputs[0].SenderAddress]
			if ok {
				for i := 0; i < k; i++ {
					lotterypool = append(lotterypool, tx.Inputs[0].SenderAddress)
				}
			}
		}

		// randomly pick winner from lottery pool
		s := rand.NewSource(time.Now().Unix())
		r := rand.New(s)
		lotteryWinner := lotterypool[r.Intn(len(lotterypool))]

		TxLotteryWinner := temp[lotteryWinner]

		// add block of winner to blockchain and let all the other nodes know

		//	LINK TO THE FUNCTION TO ADD BLOCK TO BLOCKCHAIN AND BROADCAST
		pendingTxs = append(pendingTxs, &TxLotteryWinner)
		pos := NewProofOfStake()
		pos.GetLastHash(currentChain)
		lastHash := pos.lastHash
		lastHeight := pos.lastHeight
		block := &blockchain.Block{[]byte{}, pendingTxs, lastHash, lastHeight, lotteryWinner, time.Now().Unix()}
		hash := block.BlockHashing()
		block.Hash = hash[:]
		currentChain.AddBlock(block)
	}

	mutex.Lock()
	for k := range tempStakeTxPool {
		delete(tempStakeTxPool, k)
	}
	mutex.Unlock()
}

func (pos *ProofOfStake) GetLastHash(chain *blockchain.Blockchain) {
	err := chain.Database.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte("lh"))
		if err != nil {
			log.Panic(err)
		}

		lastHash, err := item.Value()

		item, err = txn.Get(lastHash)
		if err != nil {
			log.Panic(err)
		}
		lastBlockData, _ := item.Value()

		lastBlock := blockchain.Deserialize(lastBlockData)
		lastHeight := lastBlock.Height

		pos.lastHash = lastHash
		pos.lastHeight = lastHeight

		return err
	})
	if err != nil {
		log.Panic(err)
	}
}

func NewProofOfStake() *ProofOfStake {
	return &ProofOfStake{}
}
