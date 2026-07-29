package tron

const (
	contractTypeTransfer     = 1
	contractTypeTriggerSmart = 31
)

var contractTypeName = map[int32]string{
	contractTypeTransfer:     "TransferContract",
	contractTypeTriggerSmart: "TriggerSmartContract",
}

func pbUvarint(v uint64) []byte {
	var b []byte
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func pbTag(field, wire int) []byte {
	return pbUvarint(uint64(field)<<3 | uint64(wire))
}

func pbVarintField(field int, v uint64) []byte {
	if v == 0 {
		return nil
	}
	return append(pbTag(field, 0), pbUvarint(v)...)
}

func pbBytesField(field int, data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := pbTag(field, 2)
	out = append(out, pbUvarint(uint64(len(data)))...)
	return append(out, data...)
}

func marshalTransferContract(owner, to []byte, amount int64) []byte {
	var b []byte
	b = append(b, pbBytesField(1, owner)...)
	b = append(b, pbBytesField(2, to)...)
	b = append(b, pbVarintField(3, uint64(amount))...)
	return b
}

func marshalTriggerSmartContract(owner, contract, data []byte, callValue int64) []byte {
	var b []byte
	b = append(b, pbBytesField(1, owner)...)
	b = append(b, pbBytesField(2, contract)...)
	b = append(b, pbVarintField(3, uint64(callValue))...)
	b = append(b, pbBytesField(4, data)...)
	return b
}

func marshalAny(typeURL string, value []byte) []byte {
	var b []byte
	b = append(b, pbBytesField(1, []byte(typeURL))...)
	b = append(b, pbBytesField(2, value)...)
	return b
}

func marshalContract(contractType int32, parameter []byte) []byte {
	var b []byte
	b = append(b, pbVarintField(1, uint64(contractType))...)
	b = append(b, pbBytesField(2, parameter)...)
	return b
}

func marshalRawTx(contract, refBlockBytes, refBlockHash []byte, timestamp, expiration, feeLimit int64) []byte {
	var b []byte
	b = append(b, pbBytesField(1, refBlockBytes)...)
	b = append(b, pbBytesField(4, refBlockHash)...)
	b = append(b, pbVarintField(8, uint64(expiration))...)
	b = append(b, pbBytesField(11, contract)...)
	b = append(b, pbVarintField(14, uint64(timestamp))...)
	b = append(b, pbVarintField(18, uint64(feeLimit))...)
	return b
}
