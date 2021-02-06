package wallet

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ripemd160"
)

type (
	Wallet struct {
		PrivateKey []byte
		Publickey  []byte
	}
)

const (
	checksumLength = 4
)

func NewPairKey() ([]byte, []byte) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Cannot generate RSA key\n")
		os.Exit(1)
	}
	public := &private.PublicKey

	publicKey := x509.MarshalPKCS1PublicKey(public)
	privateKey := x509.MarshalPKCS1PrivateKey(private)

	return privateKey, publicKey
}

func MakeWallet() *Wallet {
	privateKey, publicKey := NewPairKey()
	wallet := Wallet{privateKey, publicKey}

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

func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))

	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]
	pubKeyHash = pubKeyHash[:len(pubKeyHash)-checksumLength]
	targetChecksum := Checksum(pubKeyHash)

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func (wallet *Wallet) Address() []byte {
	pubHash := PublicKeyHash(wallet.Publickey)
	checksum := Checksum(pubHash)

	fullHash := append(pubHash, checksum...)

	address := Base58Encode(fullHash)
	fmt.Printf("address: %x\n", address)

	return address
}
