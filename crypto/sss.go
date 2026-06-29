package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/KernelFlowLabs/wallet-sdk/crypto/bip"
	"strings"

	"github.com/hashicorp/vault/shamir"
)

func SSSSplit(data []byte, n, t int) ([][]byte, error) {
	hash := sha256.Sum256(data)
	payload := append(data, hash[:4]...)
	return shamir.Split(payload, n, t)
}

func SSSCombine(shares [][]byte) ([]byte, error) {
	combined, err := shamir.Combine(shares)
	if err != nil {
		return nil, err
	}

	if len(combined) < 4 {
		return nil, errors.New("invalid combined data")
	}

	data := combined[:len(combined)-4]
	checksum := combined[len(combined)-4:]

	hash := sha256.Sum256(data)
	for i := 0; i < 4; i++ {
		if hash[i] != checksum[i] {
			return nil, errors.New("checksum mismatch")
		}
	}

	return data, nil
}

const (
	TotalShares     = 2
	ThresholdShares = 2
)

func PerformSplit(mn string) ([]string, error) {
	entropy, err := bip.EntropyFromMnemonic(mn)
	if err != nil {
		return nil, fmt.Errorf("invalid mn: %v", err)
	}

	fullHash := sha256.Sum256(entropy)
	fingerprint := hex.EncodeToString(fullHash[:4])

	shares, err := SSSSplit(entropy, TotalShares, ThresholdShares)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(shares))
	for i, s := range shares {
		result[i] = fmt.Sprintf("%s-%d-%s", fingerprint, i, hex.EncodeToString(s))
	}
	return result, nil
}

func PerformRecover(formattedShares []string) (string, error) {
	if len(formattedShares) < 2 {
		return "", errors.New("at least 2 shares required")
	}

	byteShares := make([][]byte, len(formattedShares))
	var expectedFingerprint string

	for i, s := range formattedShares {
		parts := strings.Split(s, "-")
		if len(parts) != 3 {
			return "", fmt.Errorf("invalid shard format at index %d", i)
		}

		currentFingerprint := parts[0]
		if expectedFingerprint == "" {
			expectedFingerprint = currentFingerprint
		} else if currentFingerprint != expectedFingerprint {
			return "", errors.New("shards belong to different sets (fingerprint mismatch)")
		}

		b, err := hex.DecodeString(parts[2])
		if err != nil {
			return "", fmt.Errorf("invalid hex data at index %d", i)
		}
		byteShares[i] = b
	}

	entropy, err := SSSCombine(byteShares)
	if err != nil {
		return "", err
	}

	fullHash := sha256.Sum256(entropy)
	actualFingerprint := hex.EncodeToString(fullHash[:4])
	if actualFingerprint != expectedFingerprint {
		return "", errors.New("recovered data fingerprint mismatch")
	}

	return bip.NewMnemonic(entropy)
}
