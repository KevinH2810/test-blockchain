package blockchain

import (
	"bytes"
	"encoding/binary"
	"log"
)

func Handler(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
