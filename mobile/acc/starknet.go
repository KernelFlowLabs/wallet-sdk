//go:build starknet || !custom

package acc

import (
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	"github.com/KernelFlowLabs/wallet-sdk/signing/starknet"
)

type StarknetAccount struct {
	in *starknet.Account
}

func NewStarknetFromMnemonic(mnemonic, path string) (*StarknetAccount, error) {
	acc, err := starknet.NewAccountFromMnemonic(mnemonic, path)
	if err != nil {
		return nil, err
	}
	return &StarknetAccount{in: acc.(*starknet.Account)}, nil
}

func (a *StarknetAccount) PrivateKey() []byte   { return a.in.PrivateKey() }
func (a *StarknetAccount) PublicKey() []byte    { return a.in.PublicKey() }
func (a *StarknetAccount) PublicKeyHex() string { return a.in.PublicKeyHex() }
func (a *StarknetAccount) Address() string      { return a.in.Address() }
func init() {
	registerAddressValidator(signing.FamilyOfStarknet, func(address, network string) bool {
		return starknet.ValidAddress(address)
	})
}
func (a *StarknetAccount) Wipe() { a.in.Wipe() }
