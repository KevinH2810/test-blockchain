package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/test-blockchain/blockchain"
	"gopkg.in/vrecan/death.v3"
)

const (
	protocol      = "tcp"
	version       = 1
	commandLength = 12
)

var (
	nodeAddress     string
	validateAddress string
	KnownNodes      = []string{"localhost:10111", "localhost:10112", "localhost:10113", "localhost:10114"}
	blocksInTransit = [][]byte{}
	memoryPool      = make(map[string]blockchain.Transaction)
	tempTxPool      []string
	tempStakeTxPool = make(map[string]blockchain.Transaction)
	candidateTxs    = make(chan blockchain.Transaction)
	currentChain    *blockchain.Blockchain
	validator       = make(map[string]int)
)

type (
	Addr struct {
		AddrList []string
	}

	Block struct {
		AddrFrom string
		Block    []byte
	}

	//GetBlocks to send Blocks from one nodes to another
	GetBlocks struct {
		AddrFrom string
	}

	GetData struct {
		AddrFrom string
		Type     string //Type of the data. can be Block / Transaction
		ID       []byte
	}

	Inv struct {
		AddrFrom string
		Type     string
		Items    [][]byte
	}

	Tx struct {
		AddrFrom    string
		Transaction []byte
	}

	//Version is to sync blockchain between nodes
	Version struct {
		Version    int
		LastHeight int
		AddrFrom   string
	}
)

func CmdToBytes(cmd string) []byte {
	var bytes [commandLength]byte

	for i, c := range cmd {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func NodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

func BytesToCmd(bytes []byte) string {
	var cmd []byte

	for _, b := range bytes {
		if b != 0x0 {
			cmd = append(cmd, b)
		}
	}

	return fmt.Sprintf("%s", cmd)
}

func CloseDB(chain *blockchain.Blockchain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}

func StartServer(nodeID, ForgerAddress string, forgeTime uint64) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	validateAddress = ForgerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	chain := blockchain.NormalBlockchainProcess(nodeID)
	defer chain.Database.Close()
	go CloseDB(chain)

	if nodeAddress != KnownNodes[0] {
		SendVersion(KnownNodes[0], chain)
	}

	if nodeAddress == KnownNodes[0] {
		currentChain = chain
		go func() {
			for candidateTx := range candidateTxs {
				mutex.Lock()
				tempStakeTxPool[hex.EncodeToString(candidateTx.ID)] = candidateTx
				mutex.Unlock()
			}
		}()
		go func() { StartForgeTimer(forgeTime) }()
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go HandleConnection(conn, chain)
	}
}

func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func StartForgeTimer(forgeTiming uint64) {
	schedulerNetwork := gocron.NewScheduler(time.UTC)
	schedulerNetwork.Every(forgeTiming).Seconds().Do(PickWinner)
}

func HandleConnection(conn net.Conn, chain *blockchain.Blockchain) {
	req, err := ioutil.ReadAll(conn)
	defer conn.Close()

	if err != nil {
		log.Panic(err)
	}
	command := BytesToCmd(req[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		HandleAddr(req)
	case "block":
		HandleBlock(req, chain)
	case "inv":
		HandleInv(req, chain)
	case "getblocks":
		HandleGetBlocks(req, chain)
	case "getdata":
		HandleGetData(req, chain)
	case "tx":
		HandleTx(req, chain)
	case "Staketx":
		HandleStakeTx(req, chain)
	case "version":
		HandleVersion(req, chain)
	default:
		fmt.Println("Unknown command")
	}

}

func ExtractCMD(request []byte) []byte {
	return request[:commandLength]
}

func SendData(addr string, data []byte) {

	if NodeIsKnown(addr) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}

	conn, err := net.Dial(protocol, addr)

	if err != nil {
		fmt.Printf("%s is not available", addr)

		return
	}

	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func SendAddr(address string) {
	if NodeIsKnown(address) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}

	nodes := Addr{KnownNodes}
	payload := GobEncode(nodes)
	request := append(CmdToBytes("addr"), payload...)

	SendData(address, request)
}

func SendBlock(addr string, b *blockchain.Block) {
	if NodeIsKnown(addr) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}

	data := Block{nodeAddress, b.Serialize()}
	payload := GobEncode(data)
	request := append(CmdToBytes("block"), payload...)

	SendData(addr, request)
}

func SendTx(addr string, tnx *blockchain.Transaction) {
	var (
		request []byte
	)
	if NodeIsKnown(addr) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}
	data := Tx{nodeAddress, tnx.Serialize()}
	payload := GobEncode(data)

	request = append(CmdToBytes("tx"), payload...)

	SendData(addr, request)
}

func SendStakeTx(addr string, tnx *blockchain.Transaction) {
	var (
		request []byte
	)
	if NodeIsKnown(addr) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}
	data := Tx{nodeAddress, tnx.Serialize()}
	payload := GobEncode(data)

	request = append(CmdToBytes("Staketx"), payload...)

	SendData(addr, request)
}

func SendVersion(addr string, chain *blockchain.Blockchain) {
	if NodeIsKnown(addr) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}
	bestHeight := chain.GetLastHeight()
	payload := GobEncode(Version{version, bestHeight, nodeAddress})

	request := append(CmdToBytes("version"), payload...)

	SendData(addr, request)
}

func SendInv(address, kind string, items [][]byte) {
	if NodeIsKnown(address) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}
	inventory := Inv{nodeAddress, kind, items}
	payload := GobEncode(inventory)
	request := append(CmdToBytes("inv"), payload...)

	SendData(address, request)
}

func SendGetBlocks(address string) {
	if NodeIsKnown(address) == false {
		log.Panic("Address is not in the list of Known Nodes")
	}
	payload := GobEncode(GetBlocks{nodeAddress})
	request := append(CmdToBytes("getblocks"), payload...)

	SendData(address, request)
}

func HandleAddr(request []byte) {
	var buff bytes.Buffer
	var payload Addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)

	}

	RequestBlocks()
}

func HandleBlock(request []byte, chain *blockchain.Blockchain) {
	var buff bytes.Buffer
	var payload Block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := blockchain.Deserialize(blockData)

	fmt.Println("Received a new block!")
	chain.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := blockchain.UTXOSet{chain}
		UTXOSet.Reindex()
	}
}

func HandleInv(request []byte, chain *blockchain.Blockchain) {
	var buff bytes.Buffer
	var payload Inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Received inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}

	if payload.Type == "StakeTx" {

	}
}

func HandleGetBlocks(request []byte, chain *blockchain.Blockchain) {
	var buff bytes.Buffer
	var payload GetBlocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := chain.GetBlockHashes()
	SendInv(payload.AddrFrom, "block", blocks)
}

func HandleGetData(request []byte, chain *blockchain.Blockchain) {
	var buff bytes.Buffer
	var payload GetData

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		SendBlock(payload.AddrFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := memoryPool[txID]

		SendTx(payload.AddrFrom, &tx)
	}

	if payload.Type == "Staketx" {
		txID := hex.EncodeToString(payload.ID)
		tx := memoryPool[txID]

		SendStakeTx(payload.AddrFrom, &tx)
	}
}

func HandleTx(request []byte, chain *blockchain.Blockchain) {
	var buff bytes.Buffer
	var payload Tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.ID)] = tx
	//add the ID to the temp TxPool to be forged later
	tempTxPool = append(tempTxPool, hex.EncodeToString(tx.ID))

	fmt.Printf("%s, %d", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddress && node != payload.AddrFrom {
				SendInv(node, "Staketx", [][]byte{tx.ID})
			}
		}
	}
}

func HandleStakeTx(request []byte, chain *blockchain.Blockchain) {
	var buff bytes.Buffer
	var payload Tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.ID)] = tx
	candidateTxs <- tx

	fmt.Printf("%s, %d", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddress && node != payload.AddrFrom {
				SendInv(node, "Staketx", [][]byte{tx.ID})
			}
		}
	}
}

func HandleVersion(request []byte, chain *blockchain.Blockchain) {
	var buff bytes.Buffer
	var payload Version

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	LastHeight := chain.GetLastHeight()
	otherHeight := payload.LastHeight

	if LastHeight < otherHeight {
		SendGetBlocks(payload.AddrFrom)
	} else if LastHeight > otherHeight {
		SendVersion(payload.AddrFrom, chain)
	}
}

func RequestBlocks() {
	for _, node := range KnownNodes {
		SendGetBlocks(node)
	}
}

func SendGetData(address, kind string, id []byte) {
	payload := GobEncode(GetData{nodeAddress, kind, id})
	request := append(CmdToBytes("getdata"), payload...)

	SendData(address, request)
}
