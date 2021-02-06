package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/test-blockchain/blockchain"
	"github.com/test-blockchain/network"
	"github.com/test-blockchain/wallet"
)

type (
	CommandLine struct {
		blockchain *blockchain.Blockchain
	}
)

func (cli *CommandLine) printUsage() {
	fmt.Println("Print Usage :")
	fmt.Println("getBalance - address ADDRESS - get balance for the ADDRESS")
	fmt.Println("createblockchain - address ADDRESS - create blockchain for the ADDRESS")
	fmt.Println("send -sender SENDER -receiver RECEIVER -amount AMOUNT -forge - send amount from Sender to Receiver")
	fmt.Println("printchain - prints the block in the chain")
	fmt.Println("createwallet - Create new wallet")
	fmt.Println("listaddress - list addresses in our wallet")
	fmt.Println("reindexutxo - Rebuilds the UTXO set")
	fmt.Println("startnode -forger ADDRESS - Start a node with specific id in NODE_ID env. -forget enables forge blocks candidate")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) printChain(NodeId string) {
	chain := blockchain.NormalBlockchainProcess(NodeId)
	defer chain.Database.Close()
	iter := chain.Iterate()

	for {
		block := iter.Next()

		for _, tx := range block.Transaction {
			fmt.Println(tx)
		}
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}

	}
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Println("NODE_IS is not set in environment!")
		runtime.Goexit()
	}

	getBalanceCmd := flag.NewFlagSet("getBalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createBlockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createNewWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	getAllWalletAddressCmd := flag.NewFlagSet("getaddress", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "the address of ownder")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "the address of the blockchain maker")
	sendFrom := sendCmd.String("from", "", "Source wallet addres")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendForge := sendCmd.Bool("forge", false, "set being a forge network")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	startNodeMiner := startNodeCmd.String("forger", "", "Enable forger mode to send reward to ADDRESS")
	startNodeTimeForge := startNodeCmd.Uint64("timeforge", 0, "Enable mining mode and send reward to ADDRESS")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		blockchain.Handler(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		blockchain.Handler(err)
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		blockchain.Handler(err)
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handler(err)
	case "getaddress":
		err := getAllWalletAddressCmd.Parse(os.Args[2:])
		blockchain.Handler(err)
	case "createwallet":
		err := createNewWalletCmd.Parse(os.Args[2:])
		blockchain.Handler(err)
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if createNewWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if getAllWalletAddressCmd.Parsed() {
		cli.listWalletAddress(nodeID)
	}

	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}

		cli.getBalance(*getBalanceAddress, nodeID)
	}
	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			runtime.Goexit()
		}
		cli.startNode(nodeID, *startNodeMiner, *startNodeTimeForge)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, nodeID, *sendAmount, *sendForge)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockchain(*createBlockchainAddress, nodeID)
	}
}

func (cli *CommandLine) createBlockchain(address, NodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid!")
	}
	chain := blockchain.InitBlockchain(address, NodeId)
	defer chain.Database.Close()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(address, NodeId string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid!")
	}
	chain := blockchain.NormalBlockchainProcess(NodeId)
	UTXOSet := blockchain.UTXOSet{chain}
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[:len(pubKeyHash)-1]
	unspentTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)

	for _, out := range unspentTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of address %s: %d\n", address, balance)
}

//send function with param Sender, Receiver and Amount. to send normal sendTx function
//fill all parameters
//empty Receiver && Amount is a StakeTx
func (cli *CommandLine) send(Sender, Receiver, NodeId string, amount int, proposeForge bool) {
	if !wallet.ValidateAddress(Sender) {
		log.Panic("Sender is not valid!")
	}

	if Receiver != "" && !wallet.ValidateAddress(Receiver) {
		log.Panic("Receiver is not valid!")
	}

	chain := blockchain.NormalBlockchainProcess(NodeId)
	UTXOSet := blockchain.UTXOSet{chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallet(NodeId)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWalletFromAddress(NodeId)

	tx := blockchain.NewTransaction(&wallet, Sender, Receiver, amount, &UTXOSet)

	fmt.Println(tx)

	if proposeForge {
		//after we make the transaction proposal, we sent it
		if Receiver != "" {
			log.Panic("please empty the sender address")
		}
		cbTx := blockchain.CoinbaseTx(Sender, "", amount)
		network.SendTx(network.KnownNodes[0], cbTx)
		fmt.Println("Transaction Proposal has been sent")
	} else {
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("Transaction Proposal has been sent")
	}

	//the normal transaction should be pushed to the txPool to be forged into one block later

	//gonna change this later so that the winner of the PoS will get the rewards
	//Generate random rewards, should be fees but not implemented
	// randData := make([]byte, 24)
	// _, err := rand.Read(randData)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// valueString := fmt.Sprintf("%x", randData)
	// value, err := strconv.Atoi(valueString)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// cbTx := blockchain.CoinbaseTx(Sender, "", value)

	// chain.ForgeBlock([]*blockchain.Transaction{cbTx, tx}, Sender)

	fmt.Println("Success!", tx)
}

func (cli *CommandLine) listWalletAddress(NodeID string) {
	wallets, _ := wallet.CreateWallet(NodeID)
	addresses := wallets.GetAllAddressFromWallet()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet(NodeId string) {
	wallets, _ := wallet.CreateWallet(NodeId)
	address := wallets.AddNewWallet()
	wallets.SaveFile(NodeId)

	fmt.Printf("New address is %s\n", address)
}

func (cli *CommandLine) startNode(NodeID, forgerAddress string, forgeTime uint64) {
	fmt.Printf("Starting Node :%s\n", NodeID)

	if len(forgerAddress) > 0 {
		if wallet.ValidateAddress(forgerAddress) {
			fmt.Println("Forging priviledge is activated. address to receive rewards : %s", forgerAddress)
		} else {
			log.Panic("Wrong Forger address")
		}
	}

	if forgeTime == 0 {
		forgeTime = uint64(30)
	}

	network.StartServer(NodeID, forgerAddress, forgeTime)
}

func (cli *CommandLine) reindexUTXO(NodeID string) {
	chain := blockchain.NormalBlockchainProcess(NodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Println("Done! There are %d transactions in the UTXO Set.\n", count)
}
