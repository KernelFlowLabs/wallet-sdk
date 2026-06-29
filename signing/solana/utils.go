package solana

import (
	"github.com/KernelFlowLabs/wallet-sdk/signing"

	"github.com/mr-tron/base58"
)

func PublicKey2Address(publicKey []byte) (string, error) {
	address := base58.Encode(publicKey[:])
	return address, nil
}

func ValidAddress(address string) bool {
	if address == signing.MagicContactAddressForNative {
		return true
	}
	val, err := base58.Decode(address)
	if err != nil {
		return false
	}
	if len(val) != PublicKeyLength {
		return false
	}
	return true
}

const (
	PublicKeyLength = 32
)

func init() {
	if err := signing.RegisterAddressValidator("sol_addr", ValidAddress); err != nil {
		panic(err)
	}
}
