package filecoin

import (
	"bytes"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"strconv"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/dchest/blake2b"
)

func PublicKey2Address(publicKey []byte) (string, error) {
	pubKey, err := btcec.ParsePubKey(publicKey)
	if err != nil {
		return "", err
	}
	addrHash, err := hash(pubKey.SerializeUncompressed(), payloadHashConfig)
	if err != nil {
		return "", err
	} else if len(addrHash) != PayloadHashLength {
		return "", fmt.Errorf("invalid payload hash length")
	}
	buf := make([]byte, 1+len(addrHash))
	buf[0] = byte(SECP256K1)
	copy(buf[1:], addrHash)
	checkHash, err := hash(buf, checksumHashConfig)
	if err != nil {
		return "", err
	}
	payload := append([]byte{}, addrHash...)
	address := MainnetPrefix + fmt.Sprintf("%d", SECP256K1) +
		base32.NewEncoding(encodeStd).WithPadding(-1).EncodeToString(append(payload, checkHash...))
	return address, nil
}

func ValidAddress(address string) bool {
	return addressToBytes(address) != nil
}

type Protocol = int8

const (
	ID Protocol = iota
	SECP256K1
	Actor
	BLS
)
const PayloadHashLength = 20
const ChecksumHashLength = 4

var payloadHashConfig = &blake2b.Config{Size: PayloadHashLength}
var checksumHashConfig = &blake2b.Config{Size: ChecksumHashLength}

const encodeStd = "abcdefghijklmnopqrstuvwxyz234567"
const MainnetPrefix = "f"

var AddressEncoding = base32.NewEncoding(encodeStd)

func hash(ingest []byte, cfg *blake2b.Config) ([]byte, error) {
	hasher, err := blake2b.New(cfg)
	if err != nil {
		return nil, err
	}
	if _, err := hasher.Write(ingest); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func addressToBytes(addr string) []byte {
	if len(addr) == 0 {
		return nil
	}
	if string(addr[0]) != MainnetPrefix {
		return nil
	}
	var protocol int8
	switch addr[1] {
	case '0':
		protocol = ID
	case '1':
		protocol = SECP256K1
	case '2':
		protocol = Actor
	case '3':
		protocol = BLS
	default:
		return nil
	}
	raw := addr[2:]
	if protocol == ID {
		if len(raw) > 20 {
			return nil
		}
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil
		}
		return toBytes(protocol, toUvarint(id))
	}

	payloadcksm, err := AddressEncoding.WithPadding(-1).DecodeString(raw)
	if err != nil {
		return nil
	}
	payload := payloadcksm[:len(payloadcksm)-4]
	cksm := payloadcksm[len(payloadcksm)-4:]
	if protocol == SECP256K1 || protocol == Actor {
		if len(payload) != 20 {
			return nil
		}
	}
	if !validateChecksum(append([]byte{byte(protocol)}, payload...), cksm) {
		return nil
	}
	return toBytes(protocol, payload)
}

func toBytes(protocol int8, payload []byte) []byte {
	switch protocol {
	case ID:
		_, n, err := fromUvarint(payload)
		if err != nil {
			return nil
		}
		if n != len(payload) {
			return nil
		}
	case SECP256K1, Actor:
		if len(payload) != 20 {
			return nil
		}
	case BLS:
		if len(payload) != 48 {
			return nil
		}
	default:
		return nil
	}
	explen := 1 + len(payload)
	buf := make([]byte, explen)

	buf[0] = byte(protocol)
	copy(buf[1:], payload)

	return buf
}

const (
	MaxLenUvarint63   = 9
	MaxValueUvarint63 = (1 << 63) - 1
)

func fromUvarint(buf []byte) (uint64, int, error) {
	var x uint64
	var s uint
	for i, b := range buf {
		if (i == 8 && b >= 0x80) || i >= MaxLenUvarint63 {
			return 0, 0, errors.New("varints larger than uint63 not supported")
		}
		if b < 0x80 {
			if b == 0 && s > 0 {
				return 0, 0, errors.New("varint not minimally encoded")
			}
			return x | uint64(b)<<s, i + 1, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0, errors.New("varints malformed, could not reach the end")
}
func toUvarint(num uint64) []byte {
	buf := make([]byte, uvarintSize(num))
	n := binary.PutUvarint(buf, uint64(num))
	return buf[:n]
}
func uvarintSize(num uint64) int {
	bits := bits.Len64(num)
	q, r := bits/7, bits%7
	size := q
	if r > 0 || size == 0 {
		size++
	}
	return size
}
func validateChecksum(ingest, expect []byte) bool {
	digest, _ := hash(ingest, checksumHashConfig)
	return bytes.Equal(digest, expect)
}
