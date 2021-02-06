package blockchain

import (
	"bytes"
	"encoding/gob"

	"github.com/test-blockchain/wallet"
)

type (
	TxOutput struct {
		Value      int
		Address    string
		PubKeyHash []byte
	}

	TxInput struct {
		ID            []byte
		SenderAddress string
		Out           int
		Signature     []byte
		PubKey        []byte
	}

	TxOutputs struct {
		Outputs []TxOutput
	}
)

func NewTxOutput(value int, address string) *TxOutput {
	txo := &TxOutput{value, address, nil}
	txo.Lock([]byte(address))
	return txo
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (out *TxOutput) Lock(address []byte) {
	fullhash := wallet.Base58Decode(address)
	pubKeyHash := fullhash[:len(fullhash)-4]
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyhash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyhash) == 0
}

func (outs TxOutputs) Serialize() []byte {
	var buffer bytes.Buffer
	encode := gob.NewEncoder(&buffer)
	err := encode.Encode(outs)
	Handler(err)
	return buffer.Bytes()
}

func DeserializeOutputs(data []byte) TxOutputs {
	var outputs TxOutputs
	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&outputs)
	Handler(err)
	return outputs
}
