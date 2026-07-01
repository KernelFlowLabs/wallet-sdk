package stellar

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/KernelFlowLabs/wallet-sdk/signing"
	xdr "github.com/KernelFlowLabs/wallet-sdk/signing/stellar/xdr"
)

const (
	// PublicNetworkPassphrase is the pass phrase for the public Stellar network.
	PublicNetworkPassphrase = "Public Global Stellar Network ; September 2015"
	// TestNetworkPassphrase is the pass phrase for the SDF test network.
	TestNetworkPassphrase = "Test SDF Network ; September 2015"
	MinBaseFee            = 100
)

func NewTxBuilder(ti *Ingredient) *TxBuilder {
	return &TxBuilder{
		Ingredient: ti,
	}
}

func (tx *TxBuilder) Build() error {
	if tx == nil {
		return fmt.Errorf("tx == nil")
	}
	if tx.Ingredient.ContractAddress != signing.MagicContactAddressForNative {
		return fmt.Errorf("only basecoin supported on this chain")
	}

	senderBytes, err := decode(VersionByteAccountID, tx.Ingredient.Sender)
	if err != nil {
		return fmt.Errorf("fail to decode Sender, err=%v", err)
	}
	var senderUint256 xdr.Uint256
	copy(senderUint256[:], senderBytes)
	senderAccount, err := xdr.NewMuxedAccount(xdr.CryptoKeyTypeKeyTypeEd25519, senderUint256)
	if err != nil {
		return fmt.Errorf("fail to NewMuxedAccount Sender, err=%v", err)
	}

	recipientBytes, err := decode(VersionByteAccountID, tx.Ingredient.Recipient)
	if err != nil {
		return fmt.Errorf("fail to decode Recipient, err=%v", err)
	}
	var recipientUint256 xdr.Uint256
	copy(recipientUint256[:], recipientBytes)
	recipientAccount, err := xdr.NewMuxedAccount(xdr.CryptoKeyTypeKeyTypeEd25519, recipientUint256)
	if err != nil {
		return fmt.Errorf("fail to NewMuxedAccount Recipient, err=%v", err)
	}

	amount, err := strconv.ParseInt(tx.Ingredient.Amount, 10, 64)
	if err != nil {
		return fmt.Errorf("fail to ParseInt for Amount, err=%v", err)
	}
	xdrAsset, err := xdr.NewAsset(xdr.AssetTypeAssetTypeNative, nil)
	if err != nil {
		return err
	}
	sequence, err := strconv.ParseInt(tx.Ingredient.Sequence, 10, 64)
	if err != nil {
		return fmt.Errorf("fail to ParseInt for Sequence, err=%v", err)
	}
	xdrMemo, err := xdr.NewMemo(xdr.MemoTypeMemoText, tx.Ingredient.Memo)
	if err != nil {
		return fmt.Errorf("fail NewMemo, err=%v", err)
	}
	xdrCond := xdr.Preconditions{
		Type: xdr.PreconditionTypePrecondTime,
		TimeBounds: &xdr.TimeBounds{
			MinTime: xdr.TimePoint(0),
			MaxTime: xdr.TimePoint(0),
		},
	}
	operation := xdr.Operation{}
	if tx.Ingredient.IsRecipientActivated == "true" {
		operation.Body = xdr.OperationBody{
			Type: xdr.OperationTypePayment,
			PaymentOp: &xdr.PaymentOp{
				Destination: recipientAccount,
				Asset:       xdrAsset,
				Amount:      xdr.Int64(amount),
			},
		}
	} else {
		if amount < 1000000 {
			return fmt.Errorf("amount must be greater than 1 since recipient is not activated yet")
		}
		operation.Body = xdr.OperationBody{
			Type: xdr.OperationTypeCreateAccount,
			CreateAccountOp: &xdr.CreateAccountOp{
				Destination: xdr.AccountId{
					Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
					Ed25519: &recipientUint256,
				},
				StartingBalance: xdr.Int64(amount),
			},
		}
	}
	envelopeType := xdr.EnvelopeTypeEnvelopeTypeTx
	transaction := xdr.Transaction{
		SourceAccount: senderAccount,
		Fee:           xdr.Uint32(MinBaseFee),
		SeqNum:        xdr.SequenceNumber(sequence + 1),
		Cond:          xdrCond,
		Memo:          xdrMemo,
		Operations:    []xdr.Operation{operation},
	}
	transactionV1Envelope := xdr.TransactionV1Envelope{
		Tx: transaction,
	}
	envelope := xdr.TransactionEnvelope{
		Type: envelopeType,
		V1:   &transactionV1Envelope,
	}
	envelopeBytes, err := envelope.MarshalBinary()
	if err != nil {
		return fmt.Errorf("fail to MarshalBinary for envelope, err=%v", err)
	}

	hashSum := sha256.Sum256([]byte(PublicNetworkPassphrase))
	taggedTx := xdr.TransactionSignaturePayloadTaggedTransaction{
		Type: envelopeType,
		Tx:   &transaction,
	}
	var txBytes bytes.Buffer
	payload := xdr.TransactionSignaturePayload{
		NetworkId:         hashSum,
		TaggedTransaction: taggedTx,
	}
	if _, err = xdr.Marshal(&txBytes, payload); err != nil {
		return fmt.Errorf("fail to Marshal for payload, err=%v", err)
	}
	txByteSum := sha256.Sum256(txBytes.Bytes())
	tx.sigHash = append(tx.sigHash, hex.EncodeToString(txByteSum[:]))
	tx.unsignedHex = hex.EncodeToString(envelopeBytes)
	return nil
}

func (tx *TxBuilder) Sign(privateKey []byte) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("tx == nil")
	} else if len(tx.sigHash) != 1 {
		return "", fmt.Errorf("tx.SigHash == nil")
	}

	sigHash, err := hex.DecodeString(tx.sigHash[0])
	if err != nil {
		return "", fmt.Errorf("fail to DecodeString for sigHash, err=%v", err)
	}
	if len(privateKey) != ed25519.SeedSize {
		return "", fmt.Errorf("invalid private key length %d", len(privateKey))
	}
	sk := ed25519.NewKeyFromSeed(privateKey)
	publicKey := sk.Public().(ed25519.PublicKey)

	var hint [4]byte
	copy(hint[:], publicKey[28:])
	sig := ed25519.Sign(sk, sigHash)
	return hex.EncodeToString(hint[:]) + "_" + hex.EncodeToString(sig), nil
}

func (tx *TxBuilder) ConcatSignature(signature string, isDerFormat bool) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("tx == nil")
	} else if tx.unsignedHex == "" {
		return "", fmt.Errorf("tx.UnsignedHex == nil")
	}

	ntxBytes, err := hex.DecodeString(tx.unsignedHex)
	if err != nil {
		return "", fmt.Errorf("fail to DecodeString for UnsignedHex, err=%v", err)
	}

	if isDerFormat {
		return "", fmt.Errorf("der format not supported")
	}
	tmp := strings.Split(signature, "_")
	if len(tmp) != 2 {
		return "", fmt.Errorf("invalid signature")
	}
	tmpHint, err := hex.DecodeString(tmp[0])
	if err != nil {
		return "", fmt.Errorf("fail to DecodeString for hint, err=%v", err)
	}
	sig, err := hex.DecodeString(tmp[1])
	if err != nil {
		return "", fmt.Errorf("fail to DecodeString for sig, err=%v", err)
	}
	var hint [4]byte
	copy(hint[:], tmpHint)
	decoratedSignature := xdr.DecoratedSignature{Hint: hint, Signature: sig}

	var envelope xdr.TransactionEnvelope
	if err := envelope.UnmarshalBinary(ntxBytes); err != nil {
		return "", fmt.Errorf("fail to UnmarshalBinary for ntxBytes, err=%v", err)
	}
	envelope.V1.Signatures = []xdr.DecoratedSignature{decoratedSignature}
	var txBytes bytes.Buffer
	if _, err := xdr.Marshal(&txBytes, envelope); err != nil {
		return "", fmt.Errorf("fail to Marshal for envelope, err=%v", err)
	}

	tx.txHash = tx.sigHash[0]
	return hex.EncodeToString(txBytes.Bytes()), nil
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

type (
	Ingredient struct {
		TxType               string `json:"txType"`
		Sender               string `json:"sender"`
		Recipient            string `json:"recipient"`
		ContractAddress      string `json:"contractAddress"`
		Amount               string `json:"amount"`
		Memo                 string `json:"memo"`
		Sequence             string `json:"sequence"`
		IsRecipientActivated string `json:"isRecipientActivated"`
	}
	TxBuilder struct {
		*Ingredient
		unsignedHex string
		sigHash     []string
		txHash      string
	}
)
