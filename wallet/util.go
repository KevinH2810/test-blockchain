package wallet

import (
	"log"

	"github.com/mr-tron/base58"
)

func Base58Encode(input []byte) []byte {
	encoded := base58.Encode(input)

	return []byte(encoded)
}

func Base58Decode(input []byte) []byte {
	decoded, err := base58.Decode(string(input[:]))
	if err != nil {
		log.Panic(err)
	}

	return decoded
}
