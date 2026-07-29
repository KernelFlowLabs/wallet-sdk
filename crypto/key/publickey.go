package key

import (
	"crypto/ed25519"

	"github.com/ChainSafe/go-schnorrkel"
	"github.com/btcsuite/btcd/btcec/v2"
)

func PrivateKey2PublicKeyECDSA(privateKey []byte) ([]byte, error) {
	_, pubKey := btcec.PrivKeyFromBytes(privateKey)
	return pubKey.SerializeCompressed(), nil
}

func PrivateKey2PublicKeyED25519(privateKey []byte) ([]byte, error) {
	sk := ed25519.NewKeyFromSeed(privateKey)
	publicKey := sk.Public().(ed25519.PublicKey)
	return publicKey, nil
}

func PrivateKey2PublicKeySR25519(privateKey []byte) ([]byte, error) {
	ms := &schnorrkel.MiniSecretKey{}
	var tmp [32]byte
	copy(tmp[:], privateKey)
	err := ms.Decode(tmp)
	if err != nil {
		return nil, err
	}
	sk := ms.ExpandEd25519()
	pub, err := sk.Public()
	if err != nil {
		return nil, err
	}
	pubBytes := pub.Encode()
	return pubBytes[:], nil
}
