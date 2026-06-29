package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/KernelFlowLabs/wallet-sdk/crypto/key"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	"strconv"
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
	if err := signing.Validator.Struct(tx.Ingredient); err != nil {
		return fmt.Errorf("invalid ingredient: %v", err)
	}

	var senderBytes []byte
	var recipientBytes []byte
	var contractAddressBytes []byte
	var payload []byte
	var amount int64
	senderBytes = ConvertToBytes(tx.Ingredient.Sender)
	refBlockHash, _ := hex.DecodeString(tx.Ingredient.RefBlockHash)
	refBlockNum, _ := strconv.ParseInt(tx.Ingredient.RefBlockNumber, 10, 64)
	refBlockNumBytes := Int64ToBytes(refBlockNum)
	if len(refBlockNumBytes) < 8 {
		return fmt.Errorf("failed to Int64ToBytes for refBlockNum")
	}
	timeStamp, _ := strconv.ParseInt(tx.Ingredient.RefBlockTimestamp, 10, 64)
	expiration := timeStamp + 36000000 //that's 10 hours
	feeLimit := int64(0)
	var valueBytes []byte
	var typeUrl string
	var txType int32
	var err error
	switch tx.Ingredient.TxType {
	case signing.TxTypeTransfer:
		if tx.Ingredient.Recipient == "" {
			return fmt.Errorf("recipient required")
		}
		if tx.Ingredient.ContractAddress == signing.MagicContactAddressForNative {
			txType = contractTypeTransfer
			typeUrl = "type.googleapis.com/protocol." + contractTypeName[txType]
			recipientBytes = ConvertToBytes(tx.Ingredient.Recipient)
			amount, err = strconv.ParseInt(tx.Ingredient.Amount, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to ParseInt for Amount, err=%v", err)
			}
			valueBytes = marshalTransferContract(senderBytes, recipientBytes, amount)
		} else {
			if tx.Ingredient.FeeLimit == "" {
				return fmt.Errorf("feeLimit required")
			}
			txType = contractTypeTriggerSmart
			typeUrl = "type.googleapis.com/protocol." + contractTypeName[txType]
			feeLimit, _ = strconv.ParseInt(tx.Ingredient.FeeLimit, 10, 64)
			contractAddressBytes = ConvertToBytes(tx.Ingredient.ContractAddress)
			p := CallTrc20In{
				Recipient: tx.Ingredient.Recipient,
				Amount:    tx.Ingredient.Amount,
			}
			params, _ := json.Marshal(p)
			hexData, err := PackPayloadForTrc20("transfer", (params))
			if err != nil {
				return fmt.Errorf("failed to PackPayloadForTrc20, err=%v", err)
			}
			data, _ := hex.DecodeString(hexData)
			valueBytes = marshalTriggerSmartContract(senderBytes, contractAddressBytes, data, 0)
			payload = data
		}
	case signing.TxTypeContractCall:
		if tx.Ingredient.FeeLimit == "" {
			return fmt.Errorf("feeLimit required")
		} else if tx.Ingredient.ContractAddress == "" {
			return fmt.Errorf("contractAddress required")
		} else if tx.Ingredient.Payload == "" {
			return fmt.Errorf("payload required")
		}
		txType = contractTypeTriggerSmart
		typeUrl = "type.googleapis.com/protocol." + contractTypeName[txType]
		feeLimit, _ = strconv.ParseInt(tx.Ingredient.FeeLimit, 10, 64)
		contractAddressBytes = ConvertToBytes(tx.Ingredient.ContractAddress)
		payload, _ = hex.DecodeString(tx.Ingredient.Payload)
		if tx.Ingredient.Amount != "" {
			amount, _ = strconv.ParseInt(tx.Ingredient.Amount, 10, 64)
		}
		valueBytes = marshalTriggerSmartContract(senderBytes, contractAddressBytes, payload, amount)
	default:
		return fmt.Errorf("invalid txType")
	}
	anyBytes := marshalAny(typeUrl, valueBytes)
	contractBytes := marshalContract(txType, anyBytes)
	rawDataBytes := marshalRawTx(contractBytes, refBlockNumBytes[6:8], refBlockHash[8:16], timeStamp, expiration, feeLimit)

	hasher := sha256.New()
	hasher.Write(rawDataBytes)
	sigHash := hex.EncodeToString(hasher.Sum(nil))

	trxContractItem := &TrxContract{
		Type: contractTypeName[txType],
		Parameter: struct {
			Value   ContractValue `json:"value"`
			TypeUrl string        `json:"type_url"`
		}{
			Value: ContractValue{
				OwnerAddress:    hex.EncodeToString(senderBytes),
				ToAddress:       hex.EncodeToString(recipientBytes),
				Data:            hex.EncodeToString(payload),
				ContractAddress: hex.EncodeToString(contractAddressBytes),
				Amount:          amount,
				CallValue:       amount,
			},
			TypeUrl: typeUrl,
		},
	}
	ntx := &NativeTx{
		ID: sigHash,
		RawData: &TxRaw{
			Contracts:     []*TrxContract{trxContractItem},
			RefBlockBytes: hex.EncodeToString(refBlockNumBytes[6:8]),
			RefBlockHash:  hex.EncodeToString(refBlockHash[8:16]),
			Timestamp:     timeStamp,
			Expiration:    expiration,
			FeeLimit:      feeLimit,
		},
	}
	ntxBytes, err := json.Marshal(ntx)
	if err != nil {
		return fmt.Errorf("failed to Marshal ntx, err=%v", err)
	}
	tx.unsignedHex = hex.EncodeToString(ntxBytes)
	tx.sigHash = append(tx.sigHash, sigHash)
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
		return "", fmt.Errorf("failed to DecodeString SigHash, err=%v", err)
	}
	signature, err := key.SignWithPrivateKeyECDSAForEVM(privateKey, sigHash)
	if err != nil {
		return "", fmt.Errorf("failed to SignWithPrivateKey, err=%v", err)
	}
	return hex.EncodeToString(signature), nil
}

func (tx *TxBuilder) ConcatSignature(signature string, isDerFormat bool) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("tx == nil")
	}
	UnsignedBytes, err := hex.DecodeString(tx.unsignedHex)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString UnsignedHex, err=%v", err)
	}
	ntx := &NativeTx{}
	err = json.Unmarshal(UnsignedBytes, ntx)
	if err != nil {
		return "", fmt.Errorf("fail Unmarshal ntx, err=%v", err)
	}
	if isDerFormat {
		return "", fmt.Errorf("DER signature format not supported")
	}
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString for Signature, err=%v", err)
	}
	ntx.Signature = []string{hex.EncodeToString(sigBytes)}
	ntxBytes, err := json.Marshal(ntx)
	if err != nil {
		return "", fmt.Errorf("failed to Marshal ntx, err=%v", err)
	}
	tx.txHash = ntx.ID
	return hex.EncodeToString(ntxBytes), nil
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
type (
	FreeGas struct {
		FreeNetUsed  string `json:"freeNetUsed"`
		FreeNetLimit string `json:"freeNetLimit"`
	}
	Ingredient struct {
		TxType          string `json:"txType" validate:"required,oneof=0 1 2 3 4 5 6"`
		ContractAddress string `json:"contractAddress" validate:"required,trx_addr"`
		Sender          string `json:"sender" validate:"required,trx_addr"`
		Recipient       string `json:"recipient,omitempty" validate:"omitempty,trx_addr"`
		Amount          string `json:"amount,omitempty" validate:"omitempty,bigint_gt0"`
		Payload         string `json:"payload,omitempty" validate:"omitempty,hex_str"`
		FeeLimit        string `json:"feeLimit" validate:"required,u64"`
		//IsRecipientActivated string `json:"isRecipientActivated,omitempty" validate:"omitempty,bool_str"`
		RefBlockHash      string `json:"refBlockHash" validate:"required,hex_str"`
		RefBlockNumber    string `json:"refBlockNumber" validate:"required,u64_gt0"`
		RefBlockTimestamp string `json:"refTimestamp" validate:"required,u64_gt0"`
		//*FreeGas
	}
	TxBuilder struct {
		*Ingredient
		unsignedHex string
		sigHash     []string
		txHash      string
	}

	ContractValue struct {
		OwnerAddress    string `json:"owner_address,omitempty"`
		ToAddress       string `json:"to_address,omitempty"`
		Data            string `json:"data,omitempty"`
		ContractAddress string `json:"contract_address,omitempty"`
		Amount          int64  `json:"amount,omitempty"`
		CallValue       int64  `json:"call_value,omitempty"`
	}
	TrxContract struct {
		Type      string `json:"type"`
		Parameter struct {
			Value   ContractValue `json:"value"`
			TypeUrl string        `json:"type_url"`
		} `json:"parameter"`
	}
	TxRaw struct {
		Contracts     []*TrxContract `json:"contract"`
		RefBlockBytes string         `json:"ref_block_bytes,omitempty"`
		RefBlockHash  string         `json:"ref_block_hash,omitempty"`
		Timestamp     int64          `json:"timestamp"`
		Expiration    int64          `json:"expiration,omitempty"`
		FeeLimit      int64          `json:"fee_limit,omitempty"`
	}
	TxRet struct {
		ContractRet string `json:"contractRet"`
	}
	NativeTx struct {
		Signature []string `json:"signature,omitempty"`
		ID        string   `json:"txID,omitempty"`
		RawData   *TxRaw   `json:"raw_data,omitempty"`
		Ret       []TxRet  `json:"ret,omitempty"`
	}
)
