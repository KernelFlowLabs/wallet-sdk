package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func RsaEncryptWithPub(pub []byte, plainText []byte) ([]byte, error) {
	block, _ := pem.Decode(pub)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the public key")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, plainText, nil)
}

func RsaDecryptWithPriv(priv []byte, cipherText []byte) ([]byte, error) {
	block, _ := pem.Decode(priv)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	var privKey *rsa.PrivateKey
	var err error

	privKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		p8, errP8 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if errP8 != nil {
			return nil, errors.New("failed to parse private key: " + errP8.Error())
		}
		var ok bool
		privKey, ok = p8.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
	}
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, cipherText, nil)
}
