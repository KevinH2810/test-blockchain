package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/test-blockchain/blockchain"
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
	fmt.Println("send -sender SENDER -receiver RECEIVER -amount AMOUNT - send acmount from Sender to Receiver")
	fmt.Println("printchain - prints the block in the chain")
	fmt.Println("createwallet - Create new wallet")
	fmt.Println("listaddress - list addresses in our wallet")

}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) printChain() {
	chain := blockchain.NormalBlockchainProcess("")
	defer chain.Database.Close()
	iter := chain.Iterate()

	for {
		block := iter.Next()

		pow := blockchain.NewProof(block)
		fmt.Printf("Pow: %s\n", strconv.FormatBool(pow.Validation()))
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

	getBalanceCmd := flag.NewFlagSet("getBalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createBlockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createNewWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	getAllWalletAddressCmd := flag.NewFlagSet("getaddress", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "the address of ownder")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "the address of the blockchain maker")
	sendFrom := sendCmd.String("from", "", "Source wallet addres")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

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

	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if createNewWalletCmd.Parsed() {
		cli.createWallet()
	}

	if getAllWalletAddressCmd.Parsed() {
		cli.listWalletAddress()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}

		cli.getBalance(*getBalanceAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if createBlockchainCmd.Parsed() {
		fmt.Println("Address = ", *createBlockchainAddress)
		if *createBlockchainAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockchain(*createBlockchainAddress)
	}
}

func (cli *CommandLine) createBlockchain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid!")
	}
	chain := blockchain.InitBlockchain(address)
	defer chain.Database.Close()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid!")
	}
	chain := blockchain.NormalBlockchainProcess(address)
	defer chain.Database.Close()

	balance := 0
	wallets, err := wallet.CreateWallet()
	blockchain.Handler(err)
	w := wallets.GetWalletFromAddress(address)
	pubKeyHash := wallet.PublicKeyHash(w.Publickey)
	unspentTXOs := chain.FindUTXO(pubKeyHash)

	for _, out := range unspentTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of address %s: %d\n", address, balance)
}

func (cli *CommandLine) send(Sender, Receiver string, amount int) {
	if !wallet.ValidateAddress(Sender) {
		log.Panic("Sender is not valid!")
	}

	if !wallet.ValidateAddress(Receiver) {
		log.Panic("Receiver is not valid!")
	}
	chain := blockchain.NormalBlockchainProcess(Sender)
	defer chain.Database.Close()

	tx := blockchain.NewTransaction(Sender, Receiver, amount, chain)

	//gonna change this later so that the winner of the PoS will get the rewards
	cbTx := blockchain.CoinbaseTx(Sender, "")

	chain.AddBlock([]*blockchain.Transaction{cbTx, tx})
	fmt.Println("Success!")
}

func (cli *CommandLine) listWalletAddress() {
	wallets, _ := wallet.CreateWallet()
	addresses := wallets.GetAllAddressFromWallet()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet() {
	wallets, _ := wallet.CreateWallet()
	address := wallets.AddNewWallet()
	wallets.SaveFile()

	fmt.Printf("New address is %s\n", address)
}
