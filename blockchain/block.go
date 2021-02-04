package blockchain

type (
	Block struct {
		Hash        []byte
		Transaction []*Transaction
		PrevHash    []byte
	}
)

func Genesis() {
	return
}

func CreateBlock() {

}

func (b *Block) HashTx() []byte {

}

func (b *Block) Serialize() []byte {

}

func Deserialize(data []byte) *Block {

}
