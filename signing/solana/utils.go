package solana

import (
	"github.com/kernelflowlabs/wallet-sdk/signing"

	"github.com/blocto/solana-go-sdk/common"
	"github.com/mr-tron/base58"
)

func TokenProgramOf(token2022 bool) common.PublicKey {
	if token2022 {
		return common.Token2022ProgramID
	}
	return common.TokenProgramID
}

// FindAssociatedTokenAddressWithProgram derives the ATA with the owning
// token program in the seed; legacy and Token-2022 ATAs differ.
func FindAssociatedTokenAddressWithProgram(wallet, mint common.PublicKey, token2022 bool) (common.PublicKey, uint8, error) {
	seeds := [][]byte{
		wallet.Bytes(),
		TokenProgramOf(token2022).Bytes(),
		mint.Bytes(),
	}
	return common.FindProgramAddress(seeds, common.SPLAssociatedTokenAccountProgramID)
}

func PublicKey2Address(publicKey []byte) (string, error) {
	address := base58.Encode(publicKey[:])
	return address, nil
}

// IsNativeSentinel accepts both native markers: EVM-style 0xeee... and
// So111...112, matching the rpc GetBalance convention.
func IsNativeSentinel(address string) bool {
	return address == signing.MagicContactAddressForNative ||
		address == signing.MagicContactAddressForNativeSOL
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
