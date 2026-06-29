package evm

import (
	"bytes"
	"testing"

	"github.com/KernelFlowLabs/wallet-sdk/crypto/key"
)

// Well-known secp256k1 test vector: private key = 1.
const (
	testPrivKeyHex = "0x0000000000000000000000000000000000000000000000000000000000000001"
	testAddress    = "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf" // EIP-55 checksummed
)

func TestAccountFromPrivateKeyHex(t *testing.T) {
	acc, err := NewAccountFromPrivateKeyHex(testPrivKeyHex)
	if err != nil {
		t.Fatalf("NewAccountFromPrivateKeyHex: %v", err)
	}
	if got := acc.Address(); got != testAddress {
		t.Fatalf("address mismatch: got %s, want %s", got, testAddress)
	}
}

// Address derivation must be deterministic across calls.
func TestAccountDeterministic(t *testing.T) {
	a1, err := NewAccountFromPrivateKeyHex(testPrivKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	a2, err := NewAccountFromPrivateKeyHex(testPrivKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	if a1.Address() != a2.Address() {
		t.Fatalf("nondeterministic address: %s vs %s", a1.Address(), a2.Address())
	}
	if !bytes.Equal(a1.PublicKey(), a2.PublicKey()) {
		t.Fatal("nondeterministic public key")
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	acc, err := NewAccountFromPrivateKeyHex(testPrivKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	digest := key.Keccak256([]byte("kernelflow wallet-sdk"))
	sig, err := acc.SignData(digest)
	if err != nil {
		t.Fatalf("SignData: %v", err)
	}
	if !acc.VerifySignData(digest, sig) {
		t.Fatal("VerifySignData returned false for a valid signature")
	}
	// A tampered digest must not verify.
	bad := key.Keccak256([]byte("tampered"))
	if acc.VerifySignData(bad, sig) {
		t.Fatal("VerifySignData returned true for a mismatched digest")
	}
}
