package tron

import (
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/KernelFlowLabs/wallet-sdk/signing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
)

func PublicKey2Address(publicKey []byte) string {
	pubKey, err := btcec.ParsePubKey(publicKey)
	if err != nil {
		return ""
	}
	ecdsaPub, err := crypto.UnmarshalPubkey(pubKey.SerializeUncompressed())
	if err != nil {
		return ""
	}
	addressHex := "41" + strings.ToLower(crypto.PubkeyToAddress(*ecdsaPub).String()[2:])
	bytes, err := hex.DecodeString(addressHex)
	if err != nil {
		return ""
	}
	return base58.CheckEncode(bytes[1:], bytes[0])
}

func ValidAddress(address string) bool {
	if address == signing.MagicContactAddressForNative {
		return true
	}
	if len(address) != 34 || address[0] != 'T' {
		return false
	}
	for _, c := range address[1:] {
		if !((c >= '1' && c <= '9') ||
			(c >= 'A' && c <= 'H') ||
			(c >= 'J' && c <= 'N') ||
			(c >= 'P' && c <= 'Z') ||
			(c >= 'a' && c <= 'k') ||
			(c >= 'm' && c <= 'z')) {
			return false
		}
	}
	return true
}

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

func ConvertFromHex(hexAddress string) string {
	b, err := hex.DecodeString(hexAddress)
	if err != nil || len(b) == 0 {
		return ""
	}
	return base58.CheckEncode(b[1:], b[0])
}

func init() {
	if err := signing.RegisterAddressValidator("trx_addr", ValidAddress); err != nil {
		panic(err)
	}
}

func Int64ToBytes(n int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(n))
	return b
}
