package wallet

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ripemd160"
)

type (
	Wallet struct {
		PrivateKey rsa.PrivateKey
		Publickey  []byte
	}
)

func NewPairKey() (*rsa.PrivateKey, []byte) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Cannot generate RSA key\n")
		os.Exit(1)
	}
	public := &privateKey.PublicKey

	publicKey := x509.MarshalPKCS1PublicKey(public)

	return privateKey, publicKey
}

func MakeWallet() *Wallet {
	privateKey, publicKey := NewPairKey()
	wallet := Wallet{*privateKey, publicKey}

	return &wallet
}

func PublicKeyHash(pubKey []byte) []byte {
	hasher := ripemd160.New()
	_, err := hasher.Write(pubKey)
	if err != nil {
		log.Panic(err)
	}

	publicRipMD := hasher.Sum(nil)

	return publicRipMD
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))

	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLength]
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func (wallet *Wallet) Address() []byte {
	pubHash := PublicKeyHash(wallet.Publickey)

	address := Base58Encode(pubHash)
	fmt.Printf("public key : %x\n", wallet.Publickey)
	fmt.Printf("private hash : %x\n", pubHash)
	fmt.Printf("address: %x\n", address)

	return address
}
