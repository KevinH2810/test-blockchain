package blockchain

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/test-blockchain/wallet"
)

type (
	Transaction struct {
		ID      []byte
		Inputs  []TxInput
		Outputs []TxOutput
	}
)

//CoinbaseTx is reward function
func CoinbaseTx(to, data string, value int) *Transaction {
	if data == "" {
		randData := make([]byte, 24)
		_, err := rand.Read(randData)
		Handler(err)

		data = fmt.Sprintf("%x", randData)
	}

	txin := TxInput{[]byte{}, "", -1, nil, []byte(data)}
	txout := NewTxOutput(value, to)

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}}
	tx.ID = tx.Hash()

	return &tx
}

func NewTransaction(w *wallet.Wallet, Sender, Receiver string, amount int, UTXO *UTXOSet) *Transaction {
	var (
		inputs  []TxInput
		outputs []TxOutput
	)

	pubKeyHash := wallet.PublicKeyHash(w.Publickey)
	acc, validOutputs := UTXO.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("Error: Not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		Handler(err)

		for _, out := range outs {
			input := TxInput{txID, Sender, out, nil, w.Publickey}
			inputs = append(inputs, input)
		}
	}

	from := fmt.Sprintf("%s", w.Address())

	outputs = append(outputs, *NewTxOutput(amount, Receiver))

	// fee := amount / 10

	// if acc > amount+fee {
	// 	outputs = append(outputs, TxOutput{fee, acc - (fee + amount), pubKeyHash})
	// }

	if acc > amount {
		outputs = append(outputs, TxOutput{acc - amount, from, pubKeyHash})
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXO.Blockchain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func (tx *Transaction) SetID() {
	var (
		encoded bytes.Buffer
		hash    [32]byte
	)

	encoder := gob.NewEncoder(&encoded)
	err := encoder.Encode(tx)
	Handler(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func (tx *Transaction) isCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	Handler(err)

	return encoded.Bytes()
}

func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	Handler(err)
	return transaction
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) Sign(privKey *rsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.isCoinbase() {
		return
	}

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTX := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTX.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		signature, err := rsa.SignPSS(rand.Reader, privKey, crypto.SHA256, txCopy.ID, nil)
		if err != nil {
			panic(err)
		}

		tx.Inputs[inId].Signature = signature

	}
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.isCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, in := range tx.Inputs {
		prevTx := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		pubKeyHash, err := x509.ParsePKCS1PublicKey(in.PubKey)
		if err != nil {
			log.Panic(err)
		}
		err = rsa.VerifyPSS(pubKeyHash, crypto.SHA256, txCopy.ID, in.Signature, nil)
		if err != nil {
			return false
		}
	}

	return true
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.SenderAddress, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		// outputs = append(outputs, TxOutput{out.Fees, out.Value, out.PubKeyHash})
		outputs = append(outputs, TxOutput{out.Value, out.Address, out.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privKey []byte) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		Handler(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	parsedPrivKey, err := x509.ParsePKCS1PrivateKey(privKey)
	if err != nil {
		log.Panic("Error = ", err)
	}

	tx.Sign(parsedPrivKey, prevTXs)
}
