package substrate

import "math/big"

func encodeTransferCall(callIndex, recipientPubKey []byte, amount *big.Int) []byte {
	b := append([]byte{}, callIndex...)
	b = append(b, 0x00)
	b = append(b, recipientPubKey...)
	b = append(b, encodeCompact(amount)...)
	return b
}

func encodeSigningPayload(callBytes, era []byte, nonce, tip *big.Int, specVersion, txVersion uint32, genesisHash, blockHash []byte, hasAssetTxPayment, hasMetadataHash bool) []byte {
	b := append([]byte{}, callBytes...)
	b = append(b, era...)
	b = append(b, encodeCompact(nonce)...)
	b = append(b, encodeCompact(tip)...)
	if hasAssetTxPayment {
		b = append(b, 0x00)
	}
	if hasMetadataHash {
		b = append(b, 0x00)
	}
	b = append(b, encodeU32LE(specVersion)...)
	b = append(b, encodeU32LE(txVersion)...)
	b = append(b, genesisHash...)
	b = append(b, blockHash...)
	if hasMetadataHash {
		b = append(b, 0x00)
	}
	return b
}

func encodeSignedExtrinsic(senderPubKey []byte, sigType byte, signature, era []byte, nonce, tip *big.Int, callBytes []byte, hasAssetTxPayment, hasMetadataHash bool) []byte {
	inner := []byte{0x84, 0x00}
	inner = append(inner, senderPubKey...)
	inner = append(inner, sigType)
	inner = append(inner, signature...)
	inner = append(inner, era...)
	inner = append(inner, encodeCompact(nonce)...)
	inner = append(inner, encodeCompact(tip)...)
	if hasAssetTxPayment {
		inner = append(inner, 0x00)
	}
	if hasMetadataHash {
		inner = append(inner, 0x00)
	}
	inner = append(inner, callBytes...)
	return append(encodeCompact(big.NewInt(int64(len(inner)))), inner...)
}
