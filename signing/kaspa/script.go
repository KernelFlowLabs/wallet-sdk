package kaspa

import (
	"encoding/binary"
	"fmt"

	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/util"
)

const (
	op0             = 0x00
	opData1         = 0x01
	opData32        = 0x20
	opData33        = 0x21
	opPushData1     = 0x4c
	opPushData2     = 0x4d
	opPushData4     = 0x4e
	op1Negate       = 0x4f
	op1             = 0x51
	op16            = 0x60
	opEqual         = 0x87
	opBlake2b       = 0xaa
	opCheckSig      = 0xac
	opCheckSigECDSA = 0xab
)

const (
	maxScriptSize        = 10000
	maxScriptElementSize = 520
)

type ScriptClass byte

const (
	NonStandardTy ScriptClass = iota
	PubKeyTy
	PubKeyECDSATy
	ScriptHashTy
)

type ScriptBuilder struct {
	script []byte
	err    error
}

func NewScriptBuilder() *ScriptBuilder {
	return &ScriptBuilder{script: make([]byte, 0, 500)}
}

func (b *ScriptBuilder) AddOp(opcode byte) *ScriptBuilder {
	if b.err != nil {
		return b
	}
	if len(b.script)+1 > maxScriptSize {
		b.err = fmt.Errorf("adding an opcode would exceed the maximum allowed script length of %d", maxScriptSize)
		return b
	}
	b.script = append(b.script, opcode)
	return b
}

func (b *ScriptBuilder) AddData(data []byte) *ScriptBuilder {
	if b.err != nil {
		return b
	}
	dataSize := canonicalDataSize(data)
	if len(b.script)+dataSize > maxScriptSize {
		b.err = fmt.Errorf("adding %d bytes of data would exceed the maximum allowed script length of %d", dataSize, maxScriptSize)
		return b
	}
	if len(data) > maxScriptElementSize {
		b.err = fmt.Errorf("adding a data element of %d bytes would exceed the maximum allowed script element size of %d", len(data), maxScriptElementSize)
		return b
	}
	return b.addData(data)
}

func (b *ScriptBuilder) AddInt64(val int64) *ScriptBuilder {
	if b.err != nil {
		return b
	}
	if len(b.script)+1 > maxScriptSize {
		b.err = fmt.Errorf("adding an integer would exceed the maximum allowed script length of %d", maxScriptSize)
		return b
	}
	if val == 0 {
		b.script = append(b.script, op0)
		return b
	}
	if val == -1 || (val >= 1 && val <= 16) {
		b.script = append(b.script, byte((op1-1)+val))
		return b
	}
	return b.AddData(scriptNumBytes(val))
}

func (b *ScriptBuilder) Script() ([]byte, error) {
	return b.script, b.err
}

func (b *ScriptBuilder) addData(data []byte) *ScriptBuilder {
	dataLen := len(data)
	if dataLen == 0 || (dataLen == 1 && data[0] == 0) {
		b.script = append(b.script, op0)
		return b
	} else if dataLen == 1 && data[0] <= 16 {
		b.script = append(b.script, (op1-1)+data[0])
		return b
	} else if dataLen == 1 && data[0] == 0x81 {
		b.script = append(b.script, op1Negate)
		return b
	}
	if dataLen < opPushData1 {
		b.script = append(b.script, byte((opData1-1)+dataLen))
	} else if dataLen <= 0xff {
		b.script = append(b.script, opPushData1, byte(dataLen))
	} else if dataLen <= 0xffff {
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataLen))
		b.script = append(b.script, opPushData2)
		b.script = append(b.script, buf...)
	} else {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(dataLen))
		b.script = append(b.script, opPushData4)
		b.script = append(b.script, buf...)
	}
	b.script = append(b.script, data...)
	return b
}

func canonicalDataSize(data []byte) int {
	dataLen := len(data)
	if dataLen == 0 {
		return 1
	} else if dataLen == 1 && data[0] <= 16 {
		return 1
	} else if dataLen == 1 && data[0] == 0x81 {
		return 1
	}
	if dataLen < opPushData1 {
		return 1 + dataLen
	} else if dataLen <= 0xff {
		return 2 + dataLen
	} else if dataLen <= 0xffff {
		return 3 + dataLen
	}
	return 5 + dataLen
}

func scriptNumBytes(n int64) []byte {
	if n == 0 {
		return nil
	}
	isNegative := n < 0
	if isNegative {
		n = -n
	}
	result := make([]byte, 0, 9)
	for n > 0 {
		result = append(result, byte(n&0xff))
		n >>= 8
	}
	if result[len(result)-1]&0x80 != 0 {
		extraByte := byte(0x00)
		if isNegative {
			extraByte = 0x80
		}
		result = append(result, extraByte)
	} else if isNegative {
		result[len(result)-1] |= 0x80
	}
	return result
}

func PayToScriptHashScript(redeemScript []byte) ([]byte, error) {
	redeemScriptHash := util.HashBlake2b(redeemScript)
	return NewScriptBuilder().
		AddOp(opBlake2b).AddData(redeemScriptHash).
		AddOp(opEqual).Script()
}

func ExtractScriptPubKeyAddress(scriptPubKey *externalapi.ScriptPublicKey, dagParams *dagconfig.Params) (ScriptClass, util.Address, error) {
	if scriptPubKey.Version > constants.MaxScriptPublicKeyVersion {
		return NonStandardTy, nil, nil
	}
	tokens, err := parseScript(scriptPubKey.Script)
	if err != nil {
		return NonStandardTy, nil, err
	}
	switch class := classifyScript(tokens); class {
	case PubKeyTy:
		addr, err := util.NewAddressPublicKey(tokens[0].data, dagParams.Prefix)
		if err != nil {
			return class, nil, nil
		}
		return class, addr, nil
	case PubKeyECDSATy:
		addr, err := util.NewAddressPublicKeyECDSA(tokens[0].data, dagParams.Prefix)
		if err != nil {
			return class, nil, nil
		}
		return class, addr, nil
	case ScriptHashTy:
		addr, err := util.NewAddressScriptHashFromHash(tokens[1].data, dagParams.Prefix)
		if err != nil {
			return class, nil, nil
		}
		return class, addr, nil
	}
	return NonStandardTy, nil, nil
}

type scriptToken struct {
	opcode byte
	data   []byte
}

func parseScript(script []byte) ([]scriptToken, error) {
	tokens := make([]scriptToken, 0, len(script))
	for i := 0; i < len(script); {
		opcode := script[i]
		switch {
		case opcode >= opData1 && opcode < opPushData1:
			n := int(opcode)
			i++
			if i+n > len(script) {
				return nil, fmt.Errorf("malformed push, not enough data")
			}
			tokens = append(tokens, scriptToken{opcode: opcode, data: script[i : i+n]})
			i += n
		case opcode == opPushData1:
			if i+2 > len(script) {
				return nil, fmt.Errorf("malformed pushdata1")
			}
			n := int(script[i+1])
			i += 2
			if i+n > len(script) {
				return nil, fmt.Errorf("malformed pushdata1, not enough data")
			}
			tokens = append(tokens, scriptToken{opcode: opcode, data: script[i : i+n]})
			i += n
		case opcode == opPushData2:
			if i+3 > len(script) {
				return nil, fmt.Errorf("malformed pushdata2")
			}
			n := int(binary.LittleEndian.Uint16(script[i+1 : i+3]))
			i += 3
			if i+n > len(script) {
				return nil, fmt.Errorf("malformed pushdata2, not enough data")
			}
			tokens = append(tokens, scriptToken{opcode: opcode, data: script[i : i+n]})
			i += n
		case opcode == opPushData4:
			if i+5 > len(script) {
				return nil, fmt.Errorf("malformed pushdata4")
			}
			n := int(binary.LittleEndian.Uint32(script[i+1 : i+5]))
			i += 5
			if i+n > len(script) {
				return nil, fmt.Errorf("malformed pushdata4, not enough data")
			}
			tokens = append(tokens, scriptToken{opcode: opcode, data: script[i : i+n]})
			i += n
		default:
			tokens = append(tokens, scriptToken{opcode: opcode})
			i++
		}
	}
	return tokens, nil
}

func classifyScript(tokens []scriptToken) ScriptClass {
	if len(tokens) == 2 && tokens[0].opcode == opData32 && tokens[1].opcode == opCheckSig {
		return PubKeyTy
	}
	if len(tokens) == 2 && tokens[0].opcode == opData33 && tokens[1].opcode == opCheckSigECDSA {
		return PubKeyECDSATy
	}
	if len(tokens) == 3 && tokens[0].opcode == opBlake2b &&
		tokens[1].opcode == opData32 && tokens[2].opcode == opEqual {
		return ScriptHashTy
	}
	return NonStandardTy
}
