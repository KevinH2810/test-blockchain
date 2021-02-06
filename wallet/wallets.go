package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const (
	walletFilePath = "./tmp/wallets_%s.data"
)

type (
	Wallets struct {
		Wallets map[string]*Wallet
	}
)

func CreateWallet(NodeId string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile(NodeId)

	return &wallets, err
}

func (ws Wallets) GetWalletFromAddress(address string) Wallet {
	return *ws.Wallets[address]
}

func (ws *Wallets) GetAllAddressFromWallet() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) AddNewWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[address] = wallet

	return address
}

func (ws *Wallets) SaveFile(nodeId string) {
	var (
		content bytes.Buffer
	)

	walletFile := fmt.Sprintf(walletFilePath, nodeId)
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

func (ws *Wallets) LoadFile(nodeId string) error {
	var (
		wallets Wallets
	)

	walletFile := fmt.Sprintf(walletFilePath, nodeId)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets

	return nil
}
