package key

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func Keccak256(data []byte) []byte {
	return crypto.Keccak256(data)
}

func Keccak256Hash(data []byte) common.Hash {
	return crypto.Keccak256Hash(data)
}
