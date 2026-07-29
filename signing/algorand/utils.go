package algorand

import (
	"bytes"
	"crypto/sha512"
	"encoding/base32"
	"fmt"
)

const (
	checksumLenBytes = 4
	hashLenBytes     = sha512.Size256
)

var MinBaseCoinValue uint64 = 200000

type (
	Address [hashLenBytes]byte
	Digest  [hashLenBytes]byte
)

func (a Address) String() string {
	checksumHash := sha512.Sum512_256(a[:])
	checksumBytes := checksumHash[hashLenBytes-checksumLenBytes:]
	checksumAddress := append(a[:], checksumBytes...)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(checksumAddress)
}

func PublicKey2Address(publicKey []byte) (string, error) {
	if len(publicKey) != hashLenBytes {
		return "", fmt.Errorf("invalid public key length %d", len(publicKey))
	}
	var pubBytes [hashLenBytes]byte
	copy(pubBytes[:], publicKey)
	checksumHash := sha512.Sum512_256(pubBytes[:])
	checksumHashBytes := checksumHash[hashLenBytes-checksumLenBytes:]
	checksumAddress := append(pubBytes[:], checksumHashBytes...)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(checksumAddress), nil
}

func ValidAddress(address string) bool {
	_, err := decodeAddress(address)
	return err == nil
}

func decodeAddress(address string) (Address, error) {
	r := Address{}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(address)
	if err != nil {
		return r, fmt.Errorf("fail to StdEncoding, err=%v", err)
	}
	if len(decoded) != len(r)+checksumLenBytes {
		return r, fmt.Errorf("len(decoded) != len(r)+checksumLenBytes")
	}
	addressBytes := decoded[:len(r)]
	checksumBytes := decoded[len(r):]
	checksumHash := sha512.Sum512_256(addressBytes)
	expectedChecksumBytes := checksumHash[hashLenBytes-checksumLenBytes:]
	if !bytes.Equal(expectedChecksumBytes, checksumBytes) {
		return r, fmt.Errorf("wrong checkSum")
	}
	copy(r[:], addressBytes)
	return r, nil
}
