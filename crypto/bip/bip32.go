// This file is derived from github.com/FactomProject/go-bip32, itself a fork of
// github.com/tyler-smith/go-bip32 (MIT License, Copyright (c) 2014 Tyler Smith),
// with local modifications. Vendored intentionally so the BIP-32 derivation can
// never change out from under the wallet. See crypto/bip/LICENSE.go-bip32.

package bip

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"
)

const (
	FirstHardenedChild        = uint32(0x80000000)
	PublicKeyCompressedLength = 33
)

var (
	PrivateWalletVersionDefault, _  = hex.DecodeString("0488ADE4")
	PrivateWalletVersionForKaspa, _ = hex.DecodeString("038F2EF4")
	PublicWalletVersionDefault, _   = hex.DecodeString("0488B21E")
	ErrSerializedKeyWrongSize       = errors.New("Serialized keys should by exactly 82 bytes")
	ErrHardnedChildPublicKey        = errors.New("Can't create hardened child for public key")
	ErrInvalidChecksum              = errors.New("Checksum doesn't match")
	ErrInvalidPrivateKey            = errors.New("Invalid private key")
	ErrInvalidPublicKey             = errors.New("Invalid public key")
)

type Key struct {
	Key         []byte
	Version     []byte
	ChildNumber []byte
	FingerPrint []byte
	ChainCode   []byte
	Depth       [1]byte
	IsPrivate   bool
}

func NewMasterKey(seed []byte, version []byte) (*Key, error) {

	hmacHash := hmac.New(sha512.New, []byte("Bitcoin seed"))
	_, err := hmacHash.Write(seed)
	if err != nil {
		return nil, err
	}
	intermediary := hmacHash.Sum(nil)

	keyBytes := intermediary[:32]
	chainCode := intermediary[32:]

	err = validatePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	if version == nil {
		version = PrivateWalletVersionDefault
	}

	key := &Key{
		Version:     version,
		ChainCode:   chainCode,
		Key:         keyBytes,
		Depth:       [1]byte{0x0},
		ChildNumber: []byte{0x00, 0x00, 0x00, 0x00},
		FingerPrint: []byte{0x00, 0x00, 0x00, 0x00},
		IsPrivate:   true,
	}

	return key, nil
}

func (key *Key) NewChildKey(childIdx uint32) (*Key, error) {
	if !key.IsPrivate && childIdx >= FirstHardenedChild {
		return nil, ErrHardnedChildPublicKey
	}

	intermediary, err := key.getIntermediary(childIdx)
	if err != nil {
		return nil, err
	}

	childKey := &Key{
		ChildNumber: uint32Bytes(childIdx),
		ChainCode:   intermediary[32:],
		Depth:       [1]byte{key.Depth[0] + 1},
		IsPrivate:   key.IsPrivate,
	}

	if key.IsPrivate {
		childKey.Version = PrivateWalletVersionDefault
		fingerprint, err := hash160(compressedPublicKeyForPrivateKey(key.Key))
		if err != nil {
			return nil, err
		}
		childKey.FingerPrint = fingerprint[:4]
		childKey.Key = addPrivateKeys(intermediary[:32], key.Key)

		err = validatePrivateKey(childKey.Key)
		if err != nil {
			return nil, err
		}

	} else {
		keyBytes := compressedPublicKeyForPrivateKey(intermediary[:32])

		err := validateChildPublicKey(keyBytes)
		if err != nil {
			return nil, err
		}

		childKey.Version = PrivateWalletVersionDefault
		fingerprint, err := hash160(key.Key)
		if err != nil {
			return nil, err
		}
		childKey.FingerPrint = fingerprint[:4]
		childKey.Key = addPublicKeys(keyBytes, key.Key)
	}

	return childKey, nil
}

func NewMasterKeyED25519(seed []byte) (*Key, error) {

	hmacHash := hmac.New(sha512.New, []byte("ed25519 seed"))
	_, err := hmacHash.Write(seed)
	if err != nil {
		return nil, err
	}
	intermediary := hmacHash.Sum(nil)

	keyBytes := intermediary[:32]
	chainCode := intermediary[32:]

	err = validatePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	key := &Key{
		Version:     PrivateWalletVersionDefault,
		ChainCode:   chainCode,
		Key:         keyBytes,
		Depth:       [1]byte{0x0},
		ChildNumber: []byte{0x00, 0x00, 0x00, 0x00},
		FingerPrint: []byte{0x00, 0x00, 0x00, 0x00},
		IsPrivate:   true,
	}

	return key, nil
}

func (key *Key) NewChildKeyED25519(childIdx uint32) (*Key, error) {
	if childIdx < FirstHardenedChild {
		return nil, ErrHardnedChildPublicKey
	}

	iBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(iBytes, childIdx)
	key1 := append([]byte{0x0}, key.Key...)
	data := append(key1, iBytes...)

	hmacHash := hmac.New(sha512.New, key.ChainCode)
	_, err := hmacHash.Write(data)
	if err != nil {
		return nil, err
	}
	sum := hmacHash.Sum(nil)
	newKey := &Key{
		Key:       sum[:32],
		ChainCode: sum[32:],
	}
	return newKey, nil
}

func (key *Key) getIntermediary(childIdx uint32) ([]byte, error) {
	childIndexBytes := uint32Bytes(childIdx)
	var data []byte
	if childIdx >= FirstHardenedChild {
		data = append([]byte{0x0}, key.Key...)
	} else {
		if key.IsPrivate {
			data = compressedPublicKeyForPrivateKey(key.Key)
		} else {
			data = key.Key
		}
	}
	data = append(data, childIndexBytes...)

	hmacHash := hmac.New(sha512.New, key.ChainCode)
	_, err := hmacHash.Write(data)
	if err != nil {
		return nil, err
	}
	return hmacHash.Sum(nil), nil
}

func (key *Key) CompressedPublicKey() *Key {
	keyBytes := key.Key

	if key.IsPrivate {
		keyBytes = compressedPublicKeyForPrivateKey(keyBytes)
	}

	return &Key{
		Version:     PrivateWalletVersionDefault,
		Key:         keyBytes,
		Depth:       key.Depth,
		ChildNumber: key.ChildNumber,
		FingerPrint: key.FingerPrint,
		ChainCode:   key.ChainCode,
		IsPrivate:   false,
	}
}

func (key *Key) UncompressedPublicKey() *Key {
	keyBytes := key.Key

	if key.IsPrivate {
		keyBytes = uncompressedPublicKeyForPrivateKey(keyBytes)
	}

	return &Key{
		Version:     PrivateWalletVersionDefault,
		Key:         keyBytes,
		Depth:       key.Depth,
		ChildNumber: key.ChildNumber,
		FingerPrint: key.FingerPrint,
		ChainCode:   key.ChainCode,
		IsPrivate:   false,
	}
}

func (key *Key) Serialize() ([]byte, error) {

	keyBytes := key.Key
	if key.IsPrivate {
		keyBytes = append([]byte{0x0}, keyBytes...)
	}

	buffer := new(bytes.Buffer)
	buffer.Write(key.Version)
	buffer.Write(key.Depth[:])
	buffer.Write(key.FingerPrint)
	buffer.Write(key.ChildNumber)
	buffer.Write(key.ChainCode)
	buffer.Write(keyBytes)

	serializedKey, err := addChecksumToBytes(buffer.Bytes())
	if err != nil {
		return nil, err
	}

	return serializedKey, nil
}

func (key *Key) B58Serialize() string {
	serializedKey, err := key.Serialize()
	if err != nil {
		return ""
	}

	return base58Encode(serializedKey)
}

func (key *Key) String() string {
	return key.B58Serialize()
}

func Deserialize(data []byte) (*Key, error) {
	if len(data) != 82 {
		return nil, ErrSerializedKeyWrongSize
	}
	var key = &Key{}
	key.Version = data[0:4]
	key.Depth = [1]byte{data[4]}
	key.FingerPrint = data[5:9]
	key.ChildNumber = data[9:13]
	key.ChainCode = data[13:45]

	if data[45] == byte(0) {
		key.IsPrivate = true
		key.Key = data[46:78]
	} else {
		key.IsPrivate = false
		key.Key = data[45:78]
	}

	cs1, err := checksum(data[0 : len(data)-4])
	if err != nil {
		return nil, err
	}

	cs2 := data[len(data)-4:]
	for i := range cs1 {
		if cs1[i] != cs2[i] {
			return nil, ErrInvalidChecksum
		}
	}
	return key, nil
}

func B58Deserialize(data string) (*Key, error) {
	b, err := base58Decode(data)
	if err != nil {
		return nil, err
	}
	return Deserialize(b)
}

func uncompressedPublicKeyForPrivateKey(key []byte) []byte {
	return uncompressPublicKey(curve.ScalarBaseMult(key))
}

func uncompressPublicKey(x *big.Int, y *big.Int) (b []byte) {
	xbytes := x.Bytes()
	ybytes := y.Bytes()

	padded_x := append(bytes.Repeat([]byte{0x00}, 32-len(xbytes)), xbytes...)
	padded_y := append(bytes.Repeat([]byte{0x00}, 32-len(ybytes)), ybytes...)

	return append([]byte{0x04}, append(padded_x, padded_y...)...)
}

func compressedPublicKeyForPrivateKey(key []byte) []byte {
	return compressPublicKey(curve.ScalarBaseMult(key))
}
