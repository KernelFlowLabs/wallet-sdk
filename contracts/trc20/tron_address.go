package trc20

import (
	"encoding/hex"
	"math/big"

	"github.com/btcsuite/btcd/btcutil/base58"
)

func ConvertToBytes(address string) []byte {
	result, version, err := base58.CheckDecode(address)
	if err != nil {
		return nil
	}
	return append([]byte{version}, result...)
}

func ConvertToHex(address string) string {
	payload := ConvertToBytes(address)
	if payload == nil {
		return ""
	}
	if payload[0] == 0 {
		return new(big.Int).SetBytes(payload).String()
	}
	h := hex.EncodeToString(payload)
	if h == "" {
		h = "0"
	}
	return h
}
