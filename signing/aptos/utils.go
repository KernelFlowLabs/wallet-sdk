package aptos

import (
	"crypto/sha3"
	"encoding/hex"
	"strings"

	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

func PublicKey2Address(publicKey []byte) (string, error) {
	data := append(publicKey, 0x00)
	authKey := sha3.Sum256(data)
	return "0x" + hex.EncodeToString(authKey[:]), nil
}

func ValidAddress(address string) bool {
	if address == signing.MagicContactAddressForNative {
		return true
	}
	parts := strings.Split(address, "::")
	if len(parts) == 1 || len(parts) == 3 {
		if len(parts[0]) != 66 {
			return false
		}
		if len(address) != 66 || address[:2] != "0x" {
			return false
		}
		for _, c := range address[2:] {
			isValid := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if isValid {
				return true
			}
		}
	} else {
		return false
	}
	return false
}

func init() {
	if err := signing.RegisterAddressValidator("apt_addr", ValidAddress); err != nil {
		panic(err)
	}
}
