package util

import (
	"strings"

	"github.com/KernelFlowLabs/wallet-sdk/crypto"
)

func SssSplit(mnemonic string) (string, error) {
	shares, err := crypto.PerformSplit(mnemonic)
	if err != nil {
		return "", err
	}
	return strings.Join(shares, ","), nil
}

func SssRecover(shares string) (string, error) {
	var clean []string
	for _, line := range strings.Split(shares, ",") {
		if s := strings.TrimSpace(line); s != "" {
			clean = append(clean, s)
		}
	}
	return crypto.PerformRecover(clean)
}
