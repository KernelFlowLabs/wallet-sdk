//go:build substrate || !custom

package acc

import (
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	"github.com/KernelFlowLabs/wallet-sdk/signing/substrate"
)

type SubstrateAccount struct {
	in *substrate.Account
}

func NewSubstrateFromMnemonic(mnemonic, path, network string) (*SubstrateAccount, error) {
	acc, err := substrate.NewAccountFromMnemonic(mnemonic, path, network)
	if err != nil {
		return nil, err
	}
	return &SubstrateAccount{in: acc.(*substrate.Account)}, nil
}

func (a *SubstrateAccount) PrivateKey() []byte {
	return a.in.PrivateKey()
}

func (a *SubstrateAccount) PublicKey() []byte {
	return a.in.PublicKey()
}

func (a *SubstrateAccount) Address() string {
	return a.in.Address()
}

const (
	SubstrateNetworkEnumForDOT          string = substrate.NetworkEnumForDOT
	SubstrateNetworkEnumForKSM          string = substrate.NetworkEnumForKSM
	SubstrateNetworkEnumForASTR         string = substrate.NetworkEnumForASTR
	SubstrateNetworkEnumForACA          string = substrate.NetworkEnumForACA
	SubstrateNetworkEnumForAZERO        string = substrate.NetworkEnumForAZERO
	SubstrateNetworkEnumForTAO          string = substrate.NetworkEnumForTAO
	SubstrateNetworkEnumForDOTASSETSHUB string = substrate.NetworkEnumForDOTASSETSHUB
)

func init() {
	registerAddressValidator(signing.FamilyOfSUBSTRATE, func(address, network string) bool {
		return substrate.ValidAddress(address, network)
	})
}

func (a *SubstrateAccount) Wipe() {
	a.in.Wipe()
}
