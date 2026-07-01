package evm

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/KernelFlowLabs/wallet-sdk/crypto/key"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewTxBuilder(ti *Ingredient, network string) *TxBuilder {
	return &TxBuilder{
		Ingredient: ti,
		network:    network,
	}
}

func (tx *TxBuilder) Build() error {
	if tx == nil {
		return fmt.Errorf("tx == nil")
	}
	if err := signing.Validator.Struct(tx.Ingredient); err != nil {
		return fmt.Errorf("invalid ingredient: %v", err)
	}

	nonce, _ := strconv.ParseUint(tx.Ingredient.Nonce, 10, 64)
	gasLimit, _ := strconv.ParseUint(tx.Ingredient.GasLimit, 10, 64)

	var data []byte
	var to common.Address
	var value *big.Int
	var ok bool
	var err error
	switch tx.Ingredient.TxType {
	case signing.TxTypeTransfer:
		if tx.Ingredient.Recipient == "" {
			return fmt.Errorf("empty Recipient")
		}
		if tx.Ingredient.ContractAddress == signing.MagicContactAddressForNative {
			value, ok = big.NewInt(0).SetString(tx.Ingredient.Amount, 10)
			if !ok {
				return fmt.Errorf("failed to SetString for Amount")
			}
			to = common.HexToAddress(tx.Ingredient.Recipient)
			if tx.Ingredient.Memo != "" {
				data, _ = hex.DecodeString(tx.Ingredient.Memo)
			}
		} else {
			to = common.HexToAddress(tx.Ingredient.ContractAddress)
			p := CallErc20In{
				Recipient: tx.Ingredient.Recipient,
				Amount:    tx.Ingredient.Amount,
				Memo:      tx.Ingredient.Memo,
			}
			params, _ := json.Marshal(p)
			hexData, err := PackPayloadForErc20("transfer", params)
			if err != nil {
				return fmt.Errorf("failed to PackPayloadForErc20, err=%v", err)
			}
			data, _ = hex.DecodeString(hexData)
		}
	case signing.TxTypeContractCall:
		to = common.HexToAddress(tx.Ingredient.ContractAddress)
		if len(tx.Ingredient.Payload) < 8 {
			return fmt.Errorf("payload too short")
		}
		data, _ = hex.DecodeString(strings.TrimPrefix(tx.Ingredient.Payload, "0x"))
		if tx.Ingredient.Amount != "" {
			value, _ = big.NewInt(0).SetString(tx.Ingredient.Amount, 10)
		}
	default:
		return fmt.Errorf("invalid txType")
	}
	unsignedTx := &types.Transaction{}
	if tx.Ingredient.IsLegacyTx == "true" {
		gasPrice, _ := new(big.Int).SetString(tx.Ingredient.GasPrice, 10)
		unsignedTx = types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &to,
			Value:    value,
			Data:     data,
		})
	} else {
		gasFeeCap, _ := new(big.Int).SetString(tx.Ingredient.GasFeeCap, 10)
		gasTipCap, _ := new(big.Int).SetString(tx.Ingredient.GasTipCap, 10)
		unsignedTx = types.NewTx(&types.DynamicFeeTx{
			Nonce:     nonce,
			To:        &to,
			Value:     value,
			Gas:       gasLimit,
			GasFeeCap: gasFeeCap,
			GasTipCap: gasTipCap,
			Data:      data,
		})
	}
	unsignedBytes, err := unsignedTx.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to MarshalBinary for unsigned, err=%v", err)
	}
	tx.unsignedHex = hex.EncodeToString(unsignedBytes)
	chainId, err := strconv.ParseInt(tx.network, 10, 64)
	if err != nil {
		return fmt.Errorf("unsuppored chainId %d", chainId)
	}
	tx.sigHash = append(tx.sigHash, strings.TrimPrefix(types.NewLondonSigner(
		big.NewInt(chainId)).Hash(unsignedTx).Hex(), "0x"))
	return nil
}

func (tx *TxBuilder) Sign(privateKey []byte) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("tx == nil")
	} else if len(tx.sigHash) != 1 {
		return "", fmt.Errorf("len(tx.SigHash) != 1, =%d", len(tx.sigHash))
	}
	sigHash, err := hex.DecodeString(tx.sigHash[0])
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString for SigHash, err=%v", err)
	}
	signature, err := key.SignWithPrivateKeyECDSAForEVM(privateKey, sigHash)
	if err != nil {
		return "", fmt.Errorf("failed to SignWithPrivateKey, err=%v", err)
	}
	return hex.EncodeToString(signature), nil
}

func (tx *TxBuilder) ConcatSignature(signature string, isDerFormat bool) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("tx == nil")
	}

	unsignedBytes, err := hex.DecodeString(tx.unsignedHex)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString for UnsignedHex, err=%v", err)
	}
	var ntx = types.Transaction{}
	err = ntx.UnmarshalBinary(unsignedBytes)
	if err != nil {
		return "", fmt.Errorf("failed to UnmarshalBinary for unsignedBytes, err=%v", err)
	}

	if isDerFormat {
		return "", fmt.Errorf("DER signature format not supported")
	}
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString for Signature, err=%v", err)
	}
	chainIdInt, err := strconv.ParseInt(tx.network, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to ParseInt for Network, err=%v", err)
	}
	stx, err := ntx.WithSignature(types.NewLondonSigner(big.NewInt(chainIdInt)), sigBytes)
	if err != nil {
		return "", fmt.Errorf("failed to WithSignature, err=%v", err)
	}
	tx.txHash = stx.Hash().Hex()
	signedBytes, err := stx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to MarshalBinary for stx, err=%v", err)
	}
	return hex.EncodeToString(signedBytes), nil
}

func (tx *TxBuilder) GetTxHash() string {
	return tx.txHash
}

func (tx *TxBuilder) GetSigHash() []string {
	return tx.sigHash
}

func (tx *TxBuilder) GetUnsignedHex() string {
	return tx.unsignedHex
}

func (tx *TxBuilder) SetSigHash(sigHash []string) {
	tx.sigHash = sigHash
}

func (tx *TxBuilder) SetUnsignedHex(unsignedHex string) {
	tx.unsignedHex = unsignedHex
}

// types
type (
	Ingredient struct {
		TxType          string `json:"txType" validate:"required,oneof=0 1 2 3 4 5 6"`
		ContractAddress string `json:"contractAddress" validate:"required,evm_addr"`
		Sender          string `json:"sender" validate:"required,evm_addr"`
		Recipient       string `json:"recipient,omitempty" validate:"omitempty,evm_addr"`
		Amount          string `json:"amount,omitempty" validate:"omitempty,bigint_gte0"`
		Payload         string `json:"payload,omitempty" validate:"omitempty,hex_str"`
		Memo            string `json:"memo,omitempty" validate:"omitempty,hex_str"`
		Nonce           string `json:"nonce" validate:"required,u64"`
		GasPrice        string `json:"gasPrice,omitempty" validate:"omitempty,bigint_gt0"`
		GasFeeCap       string `json:"gasFeeCap,omitempty" validate:"omitempty,bigint_gt0"`
		GasTipCap       string `json:"gasTipCap,omitempty" validate:"omitempty,bigint_gt0"`
		GasLimit        string `json:"gasLimit" validate:"required,u64_gt0"`
		IsLegacyTx      string `json:"isLegacyTx" validate:"required,bool_str"`
	}
	TxBuilder struct {
		*Ingredient
		network     string
		unsignedHex string
		sigHash     []string
		txHash      string
	}
)
