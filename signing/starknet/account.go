package starknet

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/NethermindEth/starknet.go/curve"

	"github.com/KernelFlowLabs/wallet-sdk/crypto/bip"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

type Account struct {
	privateKey []byte // STARK field element, big-endian
	publicKey  []byte // public key x-coordinate, big-endian
	address    string
}

func NewAccountFromMnemonic(mnemonic, path string) (signing.AccountHandler, error) {
	privateKey, err := derivePrivateKey(mnemonic, path)
	if err != nil {
		return nil, err
	}
	return NewAccountFromPrivateKey(privateKey)
}

func NewAccountFromPrivateKey(privateKey []byte) (signing.AccountHandler, error) {
	x, _ := curve.PrivateKeyToPoint(new(big.Int).SetBytes(privateKey))
	publicKey := x.Bytes()
	address, err := PublicKey2Address(publicKey)
	if err != nil {
		return nil, fmt.Errorf("PublicKey2Address: %v", err)
	}
	if !ValidAddress(address) {
		return nil, fmt.Errorf("invalid address generated")
	}
	return &Account{privateKey: privateKey, publicKey: publicKey, address: address}, nil
}

func NewAccountFromPrivateKeyHex(privateKey string) (signing.AccountHandler, error) {
	privateKeyBytes, err := hex.DecodeString(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to DecodeString privateKey: %v", err)
	}
	return NewAccountFromPrivateKey(privateKeyBytes)
}

// derivePrivateKey implements Starknet's key derivation: a BIP-44 secp256k1
// key at m/44'/60'/0'/0/0, re-derived at the caller path, then grindKey'd
// into the STARK curve order.
func derivePrivateKey(mnemonic, path string) ([]byte, error) {
	seed, err := bip.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, fmt.Errorf("NewSeedWithErrorChecking: %v", err)
	}
	k, err := bip.SeedToKeyForECDSA(seed, "m/44'/60'/0'/0/0")
	if err != nil {
		return nil, fmt.Errorf("SeedToKeyForECDSA: %v", err)
	}
	k1, err := bip.SeedToKeyForECDSA(k.Key, path)
	if err != nil {
		return nil, fmt.Errorf("SeedToKeyForECDSA: %v", err)
	}
	return grindKey(k1.Key)
}

func (a *Account) PrivateKey() []byte    { return a.privateKey }
func (a *Account) PublicKey() []byte     { return a.publicKey }
func (a *Account) PrivateKeyHex() string { return hex.EncodeToString(a.privateKey) }
func (a *Account) PublicKeyHex() string  { return "0x" + new(big.Int).SetBytes(a.publicKey).Text(16) }
func (a *Account) Address() string       { return a.address }

// SignData signs a field element (data as a big-endian big.Int) with the STARK
// curve and returns r||s (each 32-byte big-endian).
func (a *Account) SignData(data []byte) ([]byte, error) {
	r, s, err := curve.Sign(new(big.Int).SetBytes(data), new(big.Int).SetBytes(a.privateKey))
	if err != nil {
		return nil, err
	}
	out := make([]byte, 64)
	r.FillBytes(out[:32])
	s.FillBytes(out[32:])
	return out, nil
}

func (a *Account) VerifySignData(data, sig []byte) bool {
	if len(sig) != 64 {
		return false
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	ok, err := curve.Verify(new(big.Int).SetBytes(data), r, s, new(big.Int).SetBytes(a.publicKey))
	return err == nil && ok
}

func (a *Account) Wipe() {
	for i := range a.privateKey {
		a.privateKey[i] = 0
	}
}
