package blockchain

type (
	Transaction struct {
		IdHash []byte
		TxIn   []TxInput
		TxOut  []TxOutput
	}

	TxOutput struct {
		Value  int
		PubKey string
	}

	TxInput struct {
		Id        []byte
		Out       int
		Signature string
	}
)

func CoinbaseTx(receiver, data string) *Transaction {

}
