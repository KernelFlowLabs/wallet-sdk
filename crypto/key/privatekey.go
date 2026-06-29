package key

import (
	"encoding/binary"
	"fmt"
	"github.com/KernelFlowLabs/wallet-sdk/crypto/bip"
	"strconv"
	"strings"

	schnorrkel "github.com/ChainSafe/go-schnorrkel"
	"github.com/gtank/merlin"
	"golang.org/x/crypto/blake2b"
)

func DerivePrivateKeyECDSA(mnemonic, path string) ([]byte, error) {
	if mnemonic == "" {
		return nil, fmt.Errorf("mnemonic is required")
	}
	seed, err := bip.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, fmt.Errorf("failed to NewSeedWithErrorChecking, err=%v", err)
	}
	k, err := bip.SeedToKeyForECDSA(seed, path)
	if err != nil {
		return nil, fmt.Errorf("failed to SeedToKeyForECDSA, err=%v", err)
	}
	return k.Key, nil
}

func DerivePrivateKeyED25519(mnemonic, path string) ([]byte, error) {
	if mnemonic == "" {
		return nil, fmt.Errorf("mnemonic is required")
	}
	seed, err := bip.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, fmt.Errorf("failed to NewSeedWithErrorChecking, err=%v", err)
	}
	k, err := bip.SeedToKeyForED25519(seed, path)
	if err != nil {
		return nil, fmt.Errorf("failed to SeedToKeyForED25519, err=%v", err)
	}
	return k.Key, nil
}

func DerivePrivateKeySR25519(mnemonic, path string) ([]byte, error) {
	if mnemonic == "" {
		return nil, fmt.Errorf("mnemonic is required")
	}
	seed, err := bip.NewSeedFromMnemonicSr25519(mnemonic, "")
	if err != nil {
		return nil, fmt.Errorf("failed to NewSeedFromMnemonicSr25519: %w", err)
	}

	var secretBytes [32]byte
	copy(secretBytes[:], seed[:32])
	ms, err := schnorrkel.NewMiniSecretKeyFromRaw(secretBytes)
	if err != nil {
		return nil, err
	}

	if path == "" {
		keyBytes := ms.Encode()
		return keyBytes[:], nil
	}

	junctions, err := parseSubstratePath(path)
	if err != nil {
		return nil, err
	}
	for _, j := range junctions {
		ms, err = substrateHardDerive(ms, j)
		if err != nil {
			return nil, err
		}
	}

	keyBytes := ms.Encode()
	return keyBytes[:], nil
}

func substrateHardDerive(ms *schnorrkel.MiniSecretKey, junctionID []byte) (*schnorrkel.MiniSecretKey, error) {
	sk := ms.ExpandEd25519()
	skenc := sk.Encode() // [scalar_32 || nonce_32]

	t := merlin.NewTranscript("SchnorrRistrettoHDKD")
	t.AppendMessage([]byte("sign-bytes"), []byte{}) // empty, matches SigningContext.bytes(EMPTY)
	t.AppendMessage([]byte("chain-code"), junctionID)
	t.AppendMessage([]byte("secret-key"), skenc[:32]) // first 32 bytes = scalar

	var newMiniSecret [32]byte
	copy(newMiniSecret[:], t.ExtractBytes([]byte("HDKD-hard"), 32))

	return schnorrkel.NewMiniSecretKeyFromRaw(newMiniSecret)
}

func parseSubstratePath(path string) ([][]byte, error) {
	var junctions [][]byte
	for path != "" {
		if !strings.HasPrefix(path, "//") {
			return nil, fmt.Errorf("only hard derivation (//) is supported, got: %s", path)
		}
		path = path[2:]
		end := strings.Index(path, "/")
		var name string
		if end == -1 {
			name = path
			path = ""
		} else {
			name = path[:end]
			path = path[end:]
		}
		junctions = append(junctions, substrateJunctionID(name))
	}
	return junctions, nil
}

func substrateJunctionID(name string) []byte {
	var buf [32]byte
	if n, err := strconv.ParseUint(name, 10, 64); err == nil {
		binary.LittleEndian.PutUint64(buf[:8], n)
		return buf[:]
	}
	b := []byte(name)
	if len(b) > 32 {
		h := blake2b.Sum256(b)
		return h[:]
	}
	copy(buf[:], b)
	return buf[:]
}
