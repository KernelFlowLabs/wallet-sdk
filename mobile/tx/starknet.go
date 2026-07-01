//go:build starknet || !custom

package tx

import (
	"strings"

	"github.com/KernelFlowLabs/wallet-sdk/signing/starknet"
)

type StarknetTxBuilder struct {
	in *starknet.TxBuilder
}

func NewStarknetTxBuilder() *StarknetTxBuilder {
	return &StarknetTxBuilder{in: starknet.NewTxBuilder(&starknet.Ingredient{})}
}

func (b *StarknetTxBuilder) SetTxType(v string) *StarknetTxBuilder {
	b.in.Ingredient.TxType = v
	return b
}
func (b *StarknetTxBuilder) SetContractAddress(v string) *StarknetTxBuilder {
	b.in.Ingredient.ContractAddress = v
	return b
}
func (b *StarknetTxBuilder) SetSender(v string) *StarknetTxBuilder {
	b.in.Ingredient.Sender = v
	return b
}
func (b *StarknetTxBuilder) SetRecipient(v string) *StarknetTxBuilder {
	b.in.Ingredient.Recipient = v
	return b
}
func (b *StarknetTxBuilder) SetAmount(v string) *StarknetTxBuilder {
	b.in.Ingredient.Amount = v
	return b
}
func (b *StarknetTxBuilder) SetNonce(v string) *StarknetTxBuilder {
	b.in.Ingredient.Nonce = v
	return b
}
func (b *StarknetTxBuilder) SetL1GasMaxAmount(v string) *StarknetTxBuilder {
	b.in.Ingredient.L1GasMaxAmount = v
	return b
}
func (b *StarknetTxBuilder) SetL1GasMaxPrice(v string) *StarknetTxBuilder {
	b.in.Ingredient.L1GasMaxPrice = v
	return b
}
func (b *StarknetTxBuilder) SetL1DataGasMaxAmount(v string) *StarknetTxBuilder {
	b.in.Ingredient.L1DataGasMaxAmount = v
	return b
}
func (b *StarknetTxBuilder) SetL1DataGasMaxPrice(v string) *StarknetTxBuilder {
	b.in.Ingredient.L1DataGasMaxPrice = v
	return b
}
func (b *StarknetTxBuilder) SetL2GasMaxAmount(v string) *StarknetTxBuilder {
	b.in.Ingredient.L2GasMaxAmount = v
	return b
}
func (b *StarknetTxBuilder) SetL2GasMaxPrice(v string) *StarknetTxBuilder {
	b.in.Ingredient.L2GasMaxPrice = v
	return b
}
func (b *StarknetTxBuilder) SetTip(v string) *StarknetTxBuilder { b.in.Ingredient.Tip = v; return b }

func (b *StarknetTxBuilder) Build() error { return b.in.Build() }

func (b *StarknetTxBuilder) SigHash() string {
	sh := b.in.GetSigHash()
	if len(sh) == 0 {
		return ""
	}
	return sh[0]
}

func (b *StarknetTxBuilder) UnsignedHex() string { return b.in.GetUnsignedHex() }

func (b *StarknetTxBuilder) Sign(privateKey []byte) (string, error) {
	sig, err := b.in.Sign(privateKey)
	if err != nil {
		return "", err
	}
	WipeByte(privateKey)
	return sig, nil
}

func (b *StarknetTxBuilder) ConcatSignature(signature string) (string, error) {
	return b.in.ConcatSignature(strings.TrimPrefix(signature, "0x"), false)
}

func (b *StarknetTxBuilder) TxHash() string { return b.in.GetTxHash() }
