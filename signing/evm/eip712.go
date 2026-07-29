package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func Pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func ABIEncode(parts ...[]byte) []byte {
	res := make([]byte, 0, len(parts)*32)
	for _, p := range parts {
		res = append(res, Pad32(p)...)
	}
	return res
}

func EIP712Hash(domainSeparator, structHash []byte) []byte {
	return crypto.Keccak256(append([]byte{0x19, 0x01}, append(domainSeparator, structHash...)...))
}

func EIP712DomainSeparator(name, version string, chainID int64, contract common.Address) []byte {
	typeHash := crypto.Keccak256([]byte(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)",
	))
	return crypto.Keccak256(ABIEncode(
		typeHash,
		crypto.Keccak256([]byte(name)),
		crypto.Keccak256([]byte(version)),
		Pad32(big.NewInt(chainID).Bytes()),
		Pad32(contract.Bytes()),
	))
}

func EIP712DomainSeparatorNameVersion(name, version string) []byte {
	typeHash := crypto.Keccak256([]byte("EIP712Domain(string name,string version)"))
	return crypto.Keccak256(ABIEncode(
		typeHash,
		crypto.Keccak256([]byte(name)),
		crypto.Keccak256([]byte(version)),
	))
}
