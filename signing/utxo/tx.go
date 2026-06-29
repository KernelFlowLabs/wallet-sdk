package utxo

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/KernelFlowLabs/wallet-sdk/crypto/key"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func NewTxBuilder(ti *Ingredient, network string) *TxBuilder {
	return &TxBuilder{
		Ingredient: ti,
		network:    network,
	}
}

func NewTxBuilderFromUnsignedHex(unsignedHex, publicKey, network string) (*TxBuilder, error) {
	raw, err := hex.DecodeString(unsignedHex)
	if err != nil {
		return nil, fmt.Errorf("failed to DecodeString, err=%v", err)
	}

	packet, err := psbt.NewFromRawBytes(bytes.NewReader(raw), false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse as PSBT, err=%v", err)
	}

	sigHashes, err := extractSigHashes(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to extract sigHashes, err=%v", err)
	}

	return &TxBuilder{
		unsignedHex: unsignedHex,
		sigHash:     sigHashes,
		network:     network,
		Ingredient: &Ingredient{
			SenderPublicKey: publicKey,
			IsPSBT:          true,
		},
	}, nil
}

func (tx *TxBuilder) Build() error {
	if tx == nil {
		return fmt.Errorf("tx == nil")
	}

	var sigHash []string
	var unsignedHex string
	var err error
	switch tx.network {
	case NetworkEnumForBTC:
		sigHash, unsignedHex, err = buildForBTC(tx.Ingredient)
	case NetworkEnumForLTC:
		sigHash, unsignedHex, err = buildForLTC(tx.Ingredient)
	case NetworkEnumForDOGE:
		sigHash, unsignedHex, err = buildForDOGE(tx.Ingredient)
	case NetworkEnumForSYS:
		sigHash, unsignedHex, err = buildForSYS(tx.Ingredient)
	default:
		return fmt.Errorf("unsupported network")
	}
	if err != nil {
		return err
	}
	tx.sigHash = sigHash
	tx.unsignedHex = unsignedHex
	return nil
}

func (tx *TxBuilder) Sign(privateKey []byte) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("tx == nil")
	} else if len(tx.sigHash) < 1 {
		return "", fmt.Errorf("tx.SigHash too short")
	}
	var signatureList string
	for idx, v := range tx.sigHash {
		hash, err := hex.DecodeString(v)
		if err != nil {
			return "", fmt.Errorf("failed to DecodeString for hash, err=%v", err)
		}
		sig, err := key.SignWithPrivateKeyECDSAForUTXO(privateKey, hash)
		if err != nil {
			return "", fmt.Errorf("failed to SignWithPrivateKeyECDSAForUTXO, err=%v", err)
		}
		signature := append(sig, byte(txscript.SigHashAll))
		if idx == 0 {
			signatureList = hex.EncodeToString(signature)
		} else {
			signatureList = signatureList + "_" + hex.EncodeToString(signature)
		}
	}
	return signatureList, nil
}

func (tx *TxBuilder) ConcatSignature(signature string, isDerFormat bool) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("tx == nil")
	} else if len(tx.sigHash) < 1 {
		return "", fmt.Errorf("tx.SigHash == nil")
	}

	pkData, err := hex.DecodeString(tx.Ingredient.SenderPublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString for SenderPublicKey, err=%v", err)
	}

	unsignedBytes, err := hex.DecodeString(tx.unsignedHex)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString, err=%v", err)
	}
	msgTx := &wire.MsgTx{}
	packet := &psbt.Packet{}
	if tx.Ingredient.IsPSBT {
		packet, err = psbt.NewFromRawBytes(bytes.NewReader(unsignedBytes), false)
		if err != nil {
			return "", fmt.Errorf("failed to Deserialize for packet, err=%v", err)
		}
		msgTx = packet.UnsignedTx
	} else {
		err = msgTx.Deserialize(bytes.NewReader(unsignedBytes))
		if err != nil {
			return "", fmt.Errorf("failed to Deserialize for msgTx, err=%v", err)
		}
	}

	tmp := strings.Split(signature, "_")
	if len(tmp) != len(msgTx.TxIn) {
		return "", fmt.Errorf("length of signature not match")
	}
	if isDerFormat {
		return "", fmt.Errorf("DER signature format not supported")
	}
	sig := make([][]byte, len(tmp))
	for k, v := range tmp {
		vv, err := hex.DecodeString(v)
		if err != nil {
			return "", fmt.Errorf("failed to DecodeString for signature, err=%v", err)
		}
		sig[k] = vv
	}

	var txHash string
	switch tx.network {
	case NetworkEnumForBTC:
		if tx.IsPSBT {
			updater, err := psbt.NewUpdater(packet)
			if err != nil {
				return "", fmt.Errorf("failed to NewUpdater, err=%v", err)
			}
			for idx := range msgTx.TxIn {
				outcome, err := updater.Sign(idx, sig[idx], pkData, nil, nil)
				if err != nil {
					return "", fmt.Errorf("failed to updater.Sign, err=%v", err)
				} else if outcome != 0 {
					return "", fmt.Errorf("failed to updater.Sign, outcome != 0")
				}
			}
			err = psbt.MaybeFinalizeAll(packet)
			if err != nil {
				return "", fmt.Errorf("failed to MaybeFinalizeAll, err=%v", err)
			}
			msgTx, err = psbt.Extract(packet)
			if err != nil {
				return "", fmt.Errorf("failed to Extract, err=%v", err)
			}
			txHash = msgTx.TxHash().String()
		} else {
			for idx, txin := range msgTx.TxIn {
				txin.Witness = wire.TxWitness{sig[idx], pkData}
			}
			txHash = msgTx.TxHash().String()
		}
	case NetworkEnumForLTC:
		for idx, txin := range msgTx.TxIn {
			txin.Witness = wire.TxWitness{sig[idx], pkData}
		}
		txHash = msgTx.TxHash().String()
	case NetworkEnumForDOGE:
		for idx, txin := range msgTx.TxIn {
			scriptBuiler, err := txscript.NewScriptBuilder().AddData(sig[idx]).AddData(pkData).Script()
			if err != nil {
				return "", fmt.Errorf("failed to NewScriptBuilder, err=%v", err)
			}
			txin.SignatureScript = scriptBuiler
		}
		txHash = msgTx.TxHash().String()
	case NetworkEnumForSYS:
		updater, err := psbt.NewUpdater(packet)
		if err != nil {
			return "", fmt.Errorf("failed to NewUpdater, err=%v", err)
		}
		for idx, _ := range msgTx.TxIn {
			outcome, err := updater.Sign(idx, sig[idx], pkData, nil, nil)
			if err != nil {
				return "", fmt.Errorf("failed to updater.Sign, err=%v", err)
			} else if outcome != 0 {
				return "", fmt.Errorf("failed to updater.Sign, outcome != 0")
			}
		}
		err = psbt.MaybeFinalizeAll(packet)
		if err != nil {
			return "", fmt.Errorf("failed to MaybeFinalizeAll, err=%v", err)
		}
		msgTx, err = psbt.Extract(packet)
		if err != nil {
			return "", fmt.Errorf("failed to Extract, err=%v", err)
		}
		txHash = msgTx.TxHash().String()
	}
	var signedBytes bytes.Buffer
	err = msgTx.Serialize(&signedBytes)
	if err != nil {
		return "", fmt.Errorf("failed to Serialize, err=%v", err)
	}
	tx.txHash = txHash
	return hex.EncodeToString(signedBytes.Bytes()), nil
}

func (tx *TxBuilder) GetTxHash() string {
	return tx.txHash
}

func (tx *TxBuilder) GetSigHash() []string {
	return tx.sigHash
}

func (tx *TxBuilder) GetUnsignedHex() string {
	return tx.unsignedHex
}

func (tx *TxBuilder) SetSigHash(sigHash []string) {
	tx.sigHash = sigHash
}

func (tx *TxBuilder) SetUnsignedHex(unsignedHex string) {
	tx.unsignedHex = unsignedHex
}

// types
const DefaultInputBytes = 148
const DefaultOutputBytes = 34
const DefaultDust = 546

type (
	Ingredient struct {
		TxType                string            `json:"txType" validate:"required,oneof=0 1 2 3 4 5 6"`
		ContractAddress       string            `json:"contractAddress" validate:"required,native"`
		Sender                string            `json:"sender" validate:"required"`
		SenderPublicKey       string            `json:"senderPublicKey" validate:"required,hex_str"`
		Recipient             string            `json:"recipient" validate:"required"`
		Amount                string            `json:"amount" validate:"required,i64"`
		ByteFee               string            `json:"byteFee" validate:"required,i64"`
		Utxos                 *signing.UtxoList `json:"utxos" validate:"required"`
		IsPSBT                bool              `json:"isPSBT,omitempty" validate:"omitempty,bool_hex"`
		InscriptionIdsForHash map[string]string `json:"inscriptionIdsForHash,omitempty"`
		Memo                  string            `json:"memo,omitempty" validate:"omitempty,hex_str"`
	}
	TxBuilder struct {
		*Ingredient
		network     string
		unsignedHex string
		sigHash     []string
		txHash      string
	}
)

func ToBigEndianHash(hash string) string {
	if len(hash) != 64 {
		return ""
	}
	var hashBigEndian string
	for i := 0; i < 64; {
		hashBigEndian = hashBigEndian + hash[64-i-2:64-i]
		i = i + 2
	}
	return hashBigEndian
}

// by network
func buildForBTC(i *Ingredient) ([]string, string, error) {
	var sigHash []string
	var unsignedHex string
	txVersion := int32(wire.TxVersion)
	sequenceNum := wire.MaxTxInSequenceNum
	params := btcParams
	totalHas := int64(0)
	for _, v := range i.Utxos.List {
		value, _ := strconv.ParseInt(v.Value, 10, 64)
		totalHas += value
	}

	byteFee, _ := strconv.ParseInt(i.ByteFee, 10, 64)
	bytesLen := int64(0)
	var toAddrArr []string
	var toAmountArr []int64
	if i.Amount == signing.MagicNumberForMaxAmount {
		bytesLen = int64(len(i.Utxos.List)*180 + (1)*34 + 10)
		fee := bytesLen * byteFee
		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, totalHas-fee)
	} else {
		value, _ := strconv.ParseInt(i.Amount, 10, 64)
		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, value)
		bytesLen = int64(len(i.Utxos.List)*180 + (len(toAddrArr))*34 + 10)
		fee := bytesLen * byteFee
		change := totalHas - fee - value
		if change > 0 {
			if change < DefaultDust {
				return nil, "", fmt.Errorf("change < DefaultDust")
			}
			toAddrArr = append(toAddrArr, i.Sender)
			toAmountArr = append(toAmountArr, change)
		} else if change == 0 {

		} else {
			return nil, "", fmt.Errorf("value too large, totalHas=%d, value=%d, fee=%d", totalHas, value, fee)
		}
	}

	//build msgTx
	msgTx := wire.NewMsgTx(txVersion)
	for _, utxo := range i.Utxos.List {
		utxoHash, err := chainhash.NewHashFromStr(utxo.Hash)
		if err != nil {
			return nil, "", fmt.Errorf("failed to NewHashFromStr, err=%v", err)
		}
		index, _ := strconv.ParseUint(utxo.Index, 10, 64)
		outPoint := wire.NewOutPoint(utxoHash, uint32(index))
		txIn := wire.NewTxIn(outPoint, nil, nil)
		txIn.Sequence = sequenceNum
		msgTx.AddTxIn(txIn)
	}
	for i, toAddr := range toAddrArr {
		toAddress, err := btcutil.DecodeAddress(toAddr, &params)
		if err != nil {
			return nil, "", fmt.Errorf("failed to DecodeAddress for toAddr, err=%v", err)
		}
		toAddressBytes, err := txscript.PayToAddrScript(toAddress)
		if err != nil {
			return nil, "", fmt.Errorf("failed to PayToAddrScript for toAddress, err=%v", err)
		}
		txOut := wire.NewTxOut(toAmountArr[i], toAddressBytes)
		msgTx.AddTxOut(txOut)
	}

	if i.Memo != "" {
		if len(i.Memo) > 80 {
			return nil, "", fmt.Errorf("memo too long")
		}
		sb := txscript.NewScriptBuilder()
		sb.AddOp(txscript.OP_RETURN)
		sb.AddData([]byte(i.Memo))
		sbScript, err := sb.Script()
		if err != nil {
			return nil, "", fmt.Errorf("failed to Script for memo, err=%v", err)
		}
		msgTx.AddTxOut(wire.NewTxOut(0, sbScript))
	}
	/*
		{
			if i.Memo != "" {
				memoArr := util.StringSplitByLen(i.Memo, 1)
				for _, v := range memoArr {
					sb := txscript.NewScriptBuilder()
					sb.AddOp(txscript.OP_RETURN)
					sb.AddData([]byte(v))
					sbScript, err := sb.Script()
					if err != nil {
						return nil, "", fmt.Errorf("failed to Script for memo, err=%v", err)
					}
					msgTx.AddTxOut(wire.NewTxOut(0, sbScript))
				}
			}
		}
	*/

	//cal txhash
	senderAddr, err := btcutil.DecodeAddress(i.Sender, &params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to DecodeAddress for sender, err=%v", err)
	}
	pkData, err := txscript.PayToAddrScript(senderAddr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to PayToAddrScript for senderAddr, err=%v", err)
	}
	for idx, utxo := range i.Utxos.List {
		var pkScript []byte
		if utxo.Script == "" {
			pkScript = pkData
		} else {
			pkScript, _ = hex.DecodeString(utxo.Script)
		}
		var hash []byte
		amount, _ := strconv.ParseInt(utxo.Value, 10, 64)
		txSigHashes := txscript.NewTxSigHashes(msgTx, txscript.NewCannedPrevOutputFetcher(pkScript, amount))
		hash, err = txscript.CalcWitnessSigHash(pkScript, txSigHashes, txscript.SigHashAll, msgTx, idx, amount)
		if err != nil {
			return nil, "", fmt.Errorf("failed to CalcWitnessSigHash, err=%v", err)
		}
		sigHash = append(sigHash, hex.EncodeToString(hash))
	}

	var unsignedBytes bytes.Buffer
	err = msgTx.Serialize(&unsignedBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to Serialize for msgTx, err=%v", err)
	}
	unsignedHex = hex.EncodeToString(unsignedBytes.Bytes())

	return sigHash, unsignedHex, nil
}
func buildForLTC(i *Ingredient) ([]string, string, error) {
	var sigHash []string
	var unsignedHex string
	txVersion := int32(wire.TxVersion)
	sequenceNum := wire.MaxTxInSequenceNum
	params := ltcParams
	totalHas := int64(0)
	for _, v := range i.Utxos.List {
		value, _ := strconv.ParseInt(v.Value, 10, 64)
		totalHas += value
	}
	byteFee, _ := strconv.ParseInt(i.ByteFee, 10, 64)
	bytesLen := int64(0)
	var toAddrArr []string
	var toAmountArr []int64
	if i.Amount == signing.MagicNumberForMaxAmount {
		bytesLen = int64(len(i.Utxos.List)*68 + 1*31 + 11)
		fee := bytesLen * byteFee
		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, totalHas-fee)
	} else {
		value, _ := strconv.ParseInt(i.Amount, 10, 64)
		numOutputs := 2
		bytesLen = int64(len(i.Utxos.List)*68 + numOutputs*31 + 11)
		fee := bytesLen * byteFee

		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, value)
		change := totalHas - fee - value

		if change > 0 {
			if change < DefaultDust {
				return nil, "", fmt.Errorf("change < DefaultDust")
			}
			toAddrArr = append(toAddrArr, i.Sender)
			toAmountArr = append(toAmountArr, change)
		} else if change == 0 {
		} else {
			return nil, "", fmt.Errorf("value too large, totalHas=%d, value=%d, fee=%d", totalHas, value, fee)
		}
	}

	//build msgTx
	msgTx := wire.NewMsgTx(txVersion)
	for _, utxo := range i.Utxos.List {
		utxoHash, err := chainhash.NewHashFromStr(utxo.Hash)
		if err != nil {
			return nil, "", fmt.Errorf("failed to NewHashFromStr, err=%v", err)
		}
		index, _ := strconv.ParseInt(utxo.Index, 10, 64)
		outPoint := wire.NewOutPoint(utxoHash, uint32(index))
		txIn := wire.NewTxIn(outPoint, nil, nil)
		txIn.Sequence = sequenceNum
		msgTx.AddTxIn(txIn)
	}
	for i, toAddr := range toAddrArr {
		toAddress, err := btcutil.DecodeAddress(toAddr, &params)
		if err != nil {
			return nil, "", fmt.Errorf("failed to DecodeAddress for toAddr, err=%v", err)
		}
		toAddressBytes, err := txscript.PayToAddrScript(toAddress)
		if err != nil {
			return nil, "", fmt.Errorf("failed to PayToAddrScript for toAddress, err=%v", err)
		}
		txOut := wire.NewTxOut(toAmountArr[i], toAddressBytes)
		msgTx.AddTxOut(txOut)
	}

	//cal txhash
	senderAddr, err := btcutil.DecodeAddress(i.Sender, &params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to DecodeAddress for sender, err=%v", err)
	}
	pkData, err := txscript.PayToAddrScript(senderAddr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to PayToAddrScript for senderAddr, err=%v", err)
	}

	for idx, utxo := range i.Utxos.List {
		var pkScript []byte
		if utxo.Script == "" {
			pkScript = pkData
		} else {
			pkScript, _ = hex.DecodeString(utxo.Script)
		}

		amount, _ := strconv.ParseInt(utxo.Value, 10, 64)
		txSigHashes := txscript.NewTxSigHashes(msgTx, txscript.NewCannedPrevOutputFetcher(pkScript, amount))
		hash, err := txscript.CalcWitnessSigHash(pkScript, txSigHashes, txscript.SigHashAll, msgTx, idx, amount)
		if err != nil {
			return nil, "", fmt.Errorf("failed to CalcWitnessSigHash, err=%v", err)
		}
		sigHash = append(sigHash, hex.EncodeToString(hash))
	}

	var unsignedBytes bytes.Buffer
	err = msgTx.Serialize(&unsignedBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to Serialize for msgTx, err=%v", err)
	}
	unsignedHex = hex.EncodeToString(unsignedBytes.Bytes())

	return sigHash, unsignedHex, nil
}
func buildForDOGE(i *Ingredient) ([]string, string, error) {
	var sigHash []string
	var unsignedHex string
	txVersion := int32(wire.TxVersion)
	sequenceNum := wire.MaxTxInSequenceNum
	params := dogeParams
	totalHas := int64(0)
	for _, v := range i.Utxos.List {
		value, _ := strconv.ParseInt(v.Value, 10, 64)
		totalHas += value
	}

	byteFee, _ := strconv.ParseInt(i.ByteFee, 10, 64)
	bytesLen := int64(0)
	var toAddrArr []string
	var toAmountArr []int64
	if i.Amount == signing.MagicNumberForMaxAmount {
		bytesLen = int64(len(i.Utxos.List)*180 + (1)*34 + 10)
		fee := bytesLen * byteFee
		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, totalHas-fee)
	} else {
		value, _ := strconv.ParseInt(i.Amount, 10, 64)
		bytesLen = int64(len(i.Utxos.List)*180 + (2)*34 + 10)
		fee := bytesLen * byteFee
		if fee < 5000000 {
			fee = 5000000
		}
		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, value)
		change := totalHas - fee - value
		if change > 0 {
			if change < DefaultDust {
				return nil, "", fmt.Errorf("change < DefaultDust")
			}
			toAddrArr = append(toAddrArr, i.Sender)
			toAmountArr = append(toAmountArr, change)
		} else if change == 0 {

		} else {
			return nil, "", fmt.Errorf("value too large")
		}
	}

	//build msgTx
	msgTx := wire.NewMsgTx(txVersion)
	for _, utxo := range i.Utxos.List {
		utxoHash, err := chainhash.NewHashFromStr(utxo.Hash)
		if err != nil {
			return nil, "", fmt.Errorf("failed to NewHashFromStr, err=%v", err)
		}
		index, _ := strconv.ParseUint(utxo.Index, 10, 64)
		outPoint := wire.NewOutPoint(utxoHash, uint32(index))
		txIn := wire.NewTxIn(outPoint, nil, nil)
		txIn.Sequence = sequenceNum
		msgTx.AddTxIn(txIn)
	}
	for i, toAddr := range toAddrArr {
		toAddress, err := btcutil.DecodeAddress(toAddr, &params)
		if err != nil {
			return nil, "", fmt.Errorf("failed to DecodeAddress for toAddr, err=%v", err)
		}
		toAddressBytes, err := txscript.PayToAddrScript(toAddress)
		if err != nil {
			return nil, "", fmt.Errorf("failed to PayToAddrScript for toAddress, err=%v", err)
		}
		txOut := wire.NewTxOut(toAmountArr[i], toAddressBytes)
		msgTx.AddTxOut(txOut)
	}

	if i.Memo != "" {
		if len(i.Memo) > 80 {
			return nil, "", fmt.Errorf("memo too long")
		}
		sb := txscript.NewScriptBuilder()
		sb.AddOp(txscript.OP_RETURN)
		sb.AddData([]byte(i.Memo))
		sbScript, err := sb.Script()
		if err != nil {
			return nil, "", fmt.Errorf("failed to Script for memo, err=%v", err)
		}
		msgTx.AddTxOut(wire.NewTxOut(0, sbScript))
	}
	/*
		{
			if i.Memo != "" {
				memoArr := util.StringSplitByLen(i.Memo, 1)
				for _, v := range memoArr {
					sb := txscript.NewScriptBuilder()
					sb.AddOp(txscript.OP_RETURN)
					sb.AddData([]byte(v))
					sbScript, err := sb.Script()
					if err != nil {
						return nil, "", fmt.Errorf("failed to Script for memo, err=%v", err)
					}
					msgTx.AddTxOut(wire.NewTxOut(0, sbScript))
				}
			}
		}
	*/

	//cal txhash
	senderAddr, err := btcutil.DecodeAddress(i.Sender, &params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to DecodeAddress for sender, err=%v", err)
	}
	pkData, err := txscript.PayToAddrScript(senderAddr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to PayToAddrScript for senderAddr, err=%v", err)
	}
	for idx, utxo := range i.Utxos.List {
		var pkScript []byte
		if utxo.Script == "" {
			pkScript = pkData
		} else {
			pkScript, _ = hex.DecodeString(utxo.Script)
		}
		var hash []byte
		hash, err = txscript.CalcSignatureHash(pkScript, txscript.SigHashAll, msgTx, idx)
		if err != nil {
			return nil, "", fmt.Errorf("failed to CalcSignatureHash, err=%v", err)
		}
		sigHash = append(sigHash, hex.EncodeToString(hash))
	}
	var unsignedBytes bytes.Buffer
	err = msgTx.Serialize(&unsignedBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to Serialize for msgTx, err=%v", err)
	}
	unsignedHex = hex.EncodeToString(unsignedBytes.Bytes())

	return sigHash, unsignedHex, nil
}
func buildForSYS(i *Ingredient) ([]string, string, error) {
	i.IsPSBT = true
	var sigHash []string
	var unsignedHex string
	txVersion := int32(2)
	sequenceNum := wire.MaxTxInSequenceNum - 2
	params := sysParams
	totalHas := int64(0)
	for _, v := range i.Utxos.List {
		value, _ := strconv.ParseInt(v.Value, 10, 64)
		totalHas += value
	}

	byteFee, _ := strconv.ParseInt(i.ByteFee, 10, 64)
	bytesLen := int64(0)
	var toAddrArr []string
	var toAmountArr []int64
	if i.Amount == signing.MagicNumberForMaxAmount {
		bytesLen = int64(len(i.Utxos.List)*180 + (1)*34 + 10)
		fee := bytesLen * byteFee
		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, totalHas-fee)
	} else {
		value, _ := strconv.ParseInt(i.Amount, 10, 64)
		bytesLen = int64(len(i.Utxos.List)*180 + (2)*34 + 10)
		fee := bytesLen * byteFee
		toAddrArr = append(toAddrArr, i.Recipient)
		toAmountArr = append(toAmountArr, value)
		change := totalHas - fee - value
		if change > 0 {
			if change < DefaultDust {
				return nil, "", fmt.Errorf("change < DefaultDust")
			}
			toAddrArr = append(toAddrArr, i.Sender)
			toAmountArr = append(toAmountArr, change)
		} else if change == 0 {

		} else {
			return nil, "", fmt.Errorf("value too large")
		}
	}

	//build msgTx
	msgTx := wire.NewMsgTx(txVersion)
	for _, utxo := range i.Utxos.List {
		utxoHash, err := chainhash.NewHashFromStr(utxo.Hash)
		if err != nil {
			return nil, "", fmt.Errorf("failed to NewHashFromStr, err=%v", err)
		}
		index, _ := strconv.ParseUint(utxo.Index, 10, 64)
		outPoint := wire.NewOutPoint(utxoHash, uint32(index))
		txIn := wire.NewTxIn(outPoint, nil, nil)
		txIn.Sequence = sequenceNum
		msgTx.AddTxIn(txIn)
	}
	for i, toAddr := range toAddrArr {
		toAddress, err := btcutil.DecodeAddress(toAddr, &params)
		if err != nil {
			return nil, "", fmt.Errorf("failed to DecodeAddress for toAddr, err=%v", err)
		}
		toAddressBytes, err := txscript.PayToAddrScript(toAddress)
		if err != nil {
			return nil, "", fmt.Errorf("failed to PayToAddrScript for toAddress, err=%v", err)
		}
		txOut := wire.NewTxOut(toAmountArr[i], toAddressBytes)
		msgTx.AddTxOut(txOut)
	}

	//cal txhash
	senderAddr, err := btcutil.DecodeAddress(i.Sender, &params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to DecodeAddress for sender, err=%v", err)
	}
	pkData, err := txscript.PayToAddrScript(senderAddr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to PayToAddrScript for senderAddr, err=%v", err)
	}
	for idx, utxo := range i.Utxos.List {
		var pkScript []byte
		if utxo.Script == "" {
			pkScript = pkData
		} else {
			pkScript, _ = hex.DecodeString(utxo.Script)
		}
		var hash []byte
		amount, _ := strconv.ParseInt(utxo.Value, 10, 64)
		txSigHashes := txscript.NewTxSigHashes(msgTx, txscript.NewCannedPrevOutputFetcher(pkScript, amount))
		hash, err = txscript.CalcWitnessSigHash(pkScript, txSigHashes, txscript.SigHashAll, msgTx, idx, amount)
		if err != nil {
			return nil, "", fmt.Errorf("failed to CalcWitnessSigHash, err=%v", err)
		}
		sigHash = append(sigHash, hex.EncodeToString(hash))
	}
	packet, err := psbt.NewFromUnsignedTx(msgTx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to NewFromUnsignedTx, err=%v", err)
	}
	for idx, utxo := range i.Utxos.List {
		amount, _ := strconv.ParseInt(utxo.Value, 10, 64)
		packet.Inputs[idx].WitnessUtxo = wire.NewTxOut(amount, pkData)
		if err := packet.SanityCheck(); err != nil {
			return nil, "", fmt.Errorf("failed to SanityCheck, err=%v", err)
		}
	}
	var unsignedBytes bytes.Buffer
	err = packet.Serialize(&unsignedBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to Serialize for packet, err=%v", err)
	}
	unsignedHex = hex.EncodeToString(unsignedBytes.Bytes())

	return sigHash, unsignedHex, nil
}

func extractSigHashes(packet *psbt.Packet) ([]string, error) {
	var sigHashes []string

	for i, input := range packet.Inputs {
		switch {
		case input.WitnessUtxo != nil:
			prevOuts := txscript.NewCannedPrevOutputFetcher(
				input.WitnessUtxo.PkScript,
				input.WitnessUtxo.Value,
			)
			sigHash := txscript.NewTxSigHashes(packet.UnsignedTx, prevOuts)

			hash, err := txscript.CalcWitnessSigHash(
				input.WitnessUtxo.PkScript,
				sigHash,
				txscript.SigHashAll,
				packet.UnsignedTx,
				i,
				input.WitnessUtxo.Value,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to CalcWitnessSigHash for input %d, err=%v", i, err)
			}
			sigHashes = append(sigHashes, hex.EncodeToString(hash))

		case input.NonWitnessUtxo != nil:
			utxo := input.NonWitnessUtxo
			txIn := packet.UnsignedTx.TxIn[i]
			pkScript := utxo.TxOut[txIn.PreviousOutPoint.Index].PkScript

			hash, err := txscript.CalcSignatureHash(
				pkScript,
				txscript.SigHashAll,
				packet.UnsignedTx,
				i,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to CalcSignatureHash for input %d, err=%v", i, err)
			}
			sigHashes = append(sigHashes, hex.EncodeToString(hash))

		default:
			return nil, fmt.Errorf("input %d missing both WitnessUtxo and NonWitnessUtxo", i)
		}
	}

	return sigHashes, nil
}
