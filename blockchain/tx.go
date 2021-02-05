package blockchain

import (
	"bytes"

	"github.com/test-blockchain/wallet"
)

type (
	TxOutput struct {
		Value      int
		PubKeyHash []byte
	}

	TxInput struct {
		ID        []byte
		Out       int
		Signature []byte
		PubKey    []byte
	}
)

func NewTxOutput(value int, address string) *TxOutput {
	txo := &TxOutput{value, nil}
	txo.Lock([]byte(address))
	return txo
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (out *TxOutput) Lock(address []byte) {
	fullhash := wallet.Base58Decode(address)
	pubKeyHash := fullhash[0 : len(fullhash)-4]
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyhash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyhash) == 0
}
