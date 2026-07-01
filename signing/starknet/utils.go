package starknet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/NethermindEth/starknet.go/curve"
	"github.com/NethermindEth/starknet.go/utils"
)

// STARK curve order.
var starkCurveN, _ = new(big.Int).SetString(
	"3618502788666131213697322783095070105526743751716087489154079457884512865583", 10)

// ArgentX (Cairo 0 proxy) class hashes + address-prefix — must match the
// already-deployed account so the derived address is identical.
var (
	argentXproxyClassHashContract   = "0x25ec026985a3bf9d0cc1fe17326b245dfdc3ff89b8fde106542a3ea56c5a918"
	argentXaccountClassHashContract = "0x033434ad846cdd5f23eb73ff09fe6fddd568284a0fb7d1be20ee482f044dabe2"
	contractAddressPrefix           = "0x535441524b4e45545f434f4e54524143545f41444452455353"
)

func hexToBig(s string) *big.Int {
	n, _ := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	return n
}

// PublicKey2Address derives the ArgentX account address from a Starknet
// public key (x-coordinate) given as big-endian bytes.
func PublicKey2Address(publicKey []byte) (string, error) {
	pubHex := "0x" + new(big.Int).SetBytes(publicKey).Text(16)
	account, err := precalculatedArgentAddress(pubHex)
	if err != nil {
		return "", err
	}
	return expandAddress("0x" + account.Text(16)), nil
}

func ValidAddress(address string) bool {
	if len(address) != 66 || address[:2] != "0x" {
		return false
	}
	for _, c := range address[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func precalculatedArgentAddress(pubKeyHex string) (*big.Int, error) {
	pubKey := hexToBig(pubKeyHex)
	initializeCallData := []*big.Int{pubKey, big.NewInt(0)}
	axProxyConstructorCallData := _compile(hexToBig(argentXaccountClassHashContract),
		"initialize", initializeCallData)
	constructorCalldataHash := curve.ComputeHashOnElements(axProxyConstructorCallData)
	elems := []*big.Int{hexToBig(contractAddressPrefix), big.NewInt(0), pubKey,
		hexToBig(argentXproxyClassHashContract), constructorCalldataHash}
	return curve.ComputeHashOnElements(elems), nil
}

func _compile(implementation *big.Int, selector string, calldata []*big.Int) []*big.Int {
	out := []*big.Int{implementation, utils.GetSelectorFromName(selector)}
	out = append(out, big.NewInt(int64(len(calldata))))
	out = append(out, calldata...)
	return out
}

func expandAddress(address string) string {
	rest := strings.TrimPrefix(address, "0x")
	if len(rest) < 64 {
		return "0x" + strings.Repeat("0", 64-len(rest)) + rest
	}
	return address
}

// grindKey folds a seed into the STARK curve order (EIP-2645 style).
func grindKey(seed []byte) ([]byte, error) {
	sha256mask, _ := new(big.Int).SetString(
		"115792089237316195423570985008687907853269984665640564039457584007913129639936", 10)
	limit := new(big.Int).Sub(sha256mask, new(big.Int).Mod(sha256mask, starkCurveN))
	for i := int64(0); ; i++ {
		iHex := big.NewInt(i).Text(16)
		if len(iHex) == 1 {
			iHex = "0" + iHex
		}
		iBytes, _ := hex.DecodeString(iHex)
		h := sha256.Sum256(append(seed, iBytes...))
		key := new(big.Int).SetBytes(h[:])
		if key.Cmp(limit) == -1 {
			return new(big.Int).Mod(key, starkCurveN).Bytes(), nil
		}
		if i == 100000 {
			return nil, fmt.Errorf("grindKey is broken: tried 100k vals")
		}
	}
}
