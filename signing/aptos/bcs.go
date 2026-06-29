package aptos

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	rawTransactionSalt = "APTOS::RawTransaction"
	transactionSalt    = "APTOS::Transaction"
)

const (
	payloadTagEntryFunction = 2
	typeTagVariantStruct    = 7
	authTagEd25519          = 0
)

type bcsEncoder struct {
	buf []byte
}

func (e *bcsEncoder) u8(v uint8) {
	e.buf = append(e.buf, v)
}

func (e *bcsEncoder) u64(v uint64) {
	e.buf = append(e.buf,
		byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}

func (e *bcsEncoder) uleb128(v uint64) {
	for v >= 0x80 {
		e.buf = append(e.buf, byte(v)|0x80)
		v >>= 7
	}
	e.buf = append(e.buf, byte(v))
}

func (e *bcsEncoder) raw(b []byte) {
	e.buf = append(e.buf, b...)
}

func (e *bcsEncoder) lenPrefixed(b []byte) {
	e.uleb128(uint64(len(b)))
	e.buf = append(e.buf, b...)
}

func (e *bcsEncoder) str(s string) {
	e.lenPrefixed([]byte(s))
}

type aptAddress [32]byte

func parseHexAddress(s string) (aptAddress, error) {
	var a aptAddress
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return a, err
	}
	if len(b) > len(a) {
		return a, fmt.Errorf("address too long")
	}
	copy(a[len(a)-len(b):], b)
	return a, nil
}

type typeTag interface {
	encodeTag(*bcsEncoder)
}

type typeTagStruct struct {
	addr     aptAddress
	module   string
	name     string
	typeArgs []typeTag
}

func (t typeTagStruct) encodeTag(e *bcsEncoder) {
	e.uleb128(typeTagVariantStruct)
	e.raw(t.addr[:])
	e.str(t.module)
	e.str(t.name)
	e.uleb128(uint64(len(t.typeArgs)))
	for _, a := range t.typeArgs {
		a.encodeTag(e)
	}
}

func parseTypeTagStruct(s string) (typeTagStruct, error) {
	var t typeTagStruct
	if strings.Contains(s, "<") {
		return t, fmt.Errorf("nested type args not supported")
	}
	parts := strings.Split(s, "::")
	if len(parts) != 3 {
		return t, fmt.Errorf("invalid struct tag %q", s)
	}
	addr, err := parseHexAddress(parts[0])
	if err != nil {
		return t, err
	}
	t.addr = addr
	t.module = parts[1]
	t.name = parts[2]
	return t, nil
}

type entryFunction struct {
	moduleAddr aptAddress
	moduleName string
	function   string
	typeArgs   []typeTag
	args       [][]byte
}

func (p entryFunction) encodePayload(e *bcsEncoder) {
	e.uleb128(payloadTagEntryFunction)
	e.raw(p.moduleAddr[:])
	e.str(p.moduleName)
	e.str(p.function)
	e.uleb128(uint64(len(p.typeArgs)))
	for _, t := range p.typeArgs {
		t.encodeTag(e)
	}
	e.uleb128(uint64(len(p.args)))
	for _, a := range p.args {
		e.lenPrefixed(a)
	}
}

func parseModuleId(s string) (aptAddress, string, error) {
	parts := strings.Split(s, "::")
	if len(parts) != 2 {
		return aptAddress{}, "", fmt.Errorf("invalid module id %q", s)
	}
	addr, err := parseHexAddress(parts[0])
	if err != nil {
		return aptAddress{}, "", err
	}
	return addr, parts[1], nil
}

type rawTransaction struct {
	sender                  aptAddress
	sequenceNumber          uint64
	payload                 entryFunction
	maxGasAmount            uint64
	gasUnitPrice            uint64
	expirationTimestampSecs uint64
	chainId                 uint8
}

func (r rawTransaction) encode() []byte {
	e := &bcsEncoder{}
	e.raw(r.sender[:])
	e.u64(r.sequenceNumber)
	r.payload.encodePayload(e)
	e.u64(r.maxGasAmount)
	e.u64(r.gasUnitPrice)
	e.u64(r.expirationTimestampSecs)
	e.u8(r.chainId)
	return e.buf
}

func encodeU64Arg(v uint64) []byte {
	e := &bcsEncoder{}
	e.u64(v)
	return e.buf
}

func encodeSignedTransaction(rawBytes, publicKey, signature []byte) []byte {
	e := &bcsEncoder{}
	e.raw(rawBytes)
	e.uleb128(authTagEd25519)
	e.lenPrefixed(publicKey)
	e.lenPrefixed(signature)
	return e.buf
}
