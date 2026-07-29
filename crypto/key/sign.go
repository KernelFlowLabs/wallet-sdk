package key

import (
	"crypto/ed25519"
	"fmt"

	schnorrkel "github.com/ChainSafe/go-schnorrkel"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/crypto"
)

func SignWithPrivateKeyECDSAForEVM(privateKeyBytes, data []byte) ([]byte, error) {
	ecdsaKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("convert to ECDSA: %w", err)
	}

	signature, err := crypto.Sign(data, ecdsaKey)
	if err != nil {
		return nil, fmt.Errorf("sign data: %w", err)
	}

	return signature, nil
}

func VerifySignatureECDSAForEVM(publicKeyBytes, data, signature []byte) bool {
	if len(signature) < 65 {
		return false
	}

	return crypto.VerifySignature(publicKeyBytes, data, signature[:len(signature)-1])
}

func SignWithPrivateKeyECDSAForUTXO(privateKeyBytes, data []byte) ([]byte, error) {
	btcPrivateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	sig := ecdsa.Sign(btcPrivateKey, data)
	return sig.Serialize(), nil
}

func VerifySignatureECDSAForUTXO(publicKey, data, sig []byte) bool {
	pub, err := btcec.ParsePubKey(publicKey)
	if err != nil {
		return false
	}
	signature, err := ecdsa.ParseSignature(sig)
	if err != nil {
		return false
	}
	return signature.Verify(data, pub)
}

func SignWithPrivateKeyED25519(privateKeyBytes, data []byte) ([]byte, error) {
	sk := ed25519.NewKeyFromSeed(privateKeyBytes)
	return ed25519.Sign(sk, data), nil
}

func VerifySignatureED25519(publicKeyBytes, data, signature []byte) bool {
	if !ed25519.Verify(publicKeyBytes, data, signature) {
		return false
	}
	return true
}

func SignWithPrivateKeySchnorr(privateKeyBytes, data []byte) ([]byte, error) {
	prvKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	signature, err := schnorr.Sign(prvKey, data)
	if err != nil {
		return nil, err
	}
	return signature.Serialize(), err
}

func VerifySignatureSchnorr(publicKey, data, sig []byte) bool {
	pub, err := schnorr.ParsePubKey(publicKey)
	if err != nil {
		return false
	}
	signature, err := schnorr.ParseSignature(sig)
	if err != nil {
		return false
	}
	return signature.Verify(data, pub)
}

func SignWithPrivateKeySR25519(privateKeyBytes, data []byte) ([]byte, error) {
	ms := &schnorrkel.MiniSecretKey{}
	var tmp [32]byte
	copy(tmp[:], privateKeyBytes)
	if err := ms.Decode(tmp); err != nil {
		return nil, fmt.Errorf("decode mini secret key: %w", err)
	}
	sk := ms.ExpandEd25519()
	sig, err := sk.Sign(schnorrkel.NewSigningContext([]byte("substrate"), data))
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	enc := sig.Encode()
	return enc[:], nil
}

func VerifySignatureSR25519(publicKey, data, sig []byte) bool {
	pub := &schnorrkel.PublicKey{}
	var pubTmp [32]byte
	copy(pubTmp[:], publicKey)
	if err := pub.Decode(pubTmp); err != nil {
		return false
	}
	signature := &schnorrkel.Signature{}
	var sigTmp [64]byte
	copy(sigTmp[:], sig)
	if err := signature.Decode(sigTmp); err != nil {
		return false
	}
	ok, err := pub.Verify(signature, schnorrkel.NewSigningContext([]byte("substrate"), data))
	return err == nil && ok
}
