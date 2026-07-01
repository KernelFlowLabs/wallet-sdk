package starknet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/account"
	"github.com/NethermindEth/starknet.go/curve"
	"github.com/NethermindEth/starknet.go/hash"
	"github.com/NethermindEth/starknet.go/rpc"
	"github.com/NethermindEth/starknet.go/utils"

	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

const ethContractAddress = "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7"

// SN_MAIN chain id ("SN_MAIN" as a short string).
var snMain, _ = utils.HexToFelt("0x534e5f4d41494e")

var mask128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

func NewTxBuilder(ti *Ingredient) *TxBuilder {
	return &TxBuilder{Ingredient: ti}
}

func (tx *TxBuilder) Build() error {
	if tx == nil {
		return fmt.Errorf("tx == nil")
	}
	if tx.TxType != signing.TxTypeTransfer {
		return fmt.Errorf("only transfer (INVOKE v3) supported")
	}
	contract := tx.ContractAddress
	if contract == "" {
		return fmt.Errorf("contractAddress required")
	} else if contract == signing.MagicContactAddressForNative {
		contract = ethContractAddress
	}
	contractFelt, err := utils.HexToFelt(contract)
	if err != nil {
		return fmt.Errorf("bad contractAddress: %v", err)
	}
	recipientFelt, err := utils.HexToFelt(tx.Recipient)
	if err != nil {
		return fmt.Errorf("bad recipient: %v", err)
	}
	senderFelt, err := utils.HexToFelt(tx.Sender)
	if err != nil {
		return fmt.Errorf("bad sender: %v", err)
	}
	amt, ok := new(big.Int).SetString(tx.Amount, 10)
	if !ok {
		return fmt.Errorf("bad amount %s", tx.Amount)
	}
	low := new(big.Int).And(amt, mask128)
	high := new(big.Int).Rsh(amt, 128)
	nonce, ok := new(big.Int).SetString(tx.Nonce, 10)
	if !ok {
		return fmt.Errorf("bad nonce %s", tx.Nonce)
	}
	rb, err := tx.resourceBounds()
	if err != nil {
		return err
	}
	tip, err := decToU64(orZero(tx.Tip))
	if err != nil {
		return fmt.Errorf("bad tip: %v", err)
	}

	call := rpc.FunctionCall{
		ContractAddress:    contractFelt,
		EntryPointSelector: utils.GetSelectorFromNameFelt("transfer"),
		Calldata:           []*felt.Felt{recipientFelt, utils.BigIntToFelt(low), utils.BigIntToFelt(high)},
	}
	txn := &rpc.InvokeTxnV3{
		Type:                  rpc.TransactionTypeInvoke,
		SenderAddress:         senderFelt,
		Calldata:              account.FmtCallDataCairo0([]rpc.FunctionCall{call}),
		Version:               rpc.TransactionV3,
		Signature:             []*felt.Felt{},
		Nonce:                 utils.BigIntToFelt(nonce),
		ResourceBounds:        rb,
		Tip:                   tip,
		PayMasterData:         []*felt.Felt{},
		AccountDeploymentData: []*felt.Felt{},
		NonceDataMode:         rpc.DAModeL1,
		FeeMode:               rpc.DAModeL1,
	}
	h, err := hash.TransactionHashInvokeV3(txn, snMain)
	if err != nil {
		return fmt.Errorf("TransactionHashInvokeV3: %v", err)
	}
	b, err := json.Marshal(txn)
	if err != nil {
		return fmt.Errorf("marshal txn: %v", err)
	}
	tx.sigHash = []string{h.String()}
	tx.unsignedHex = hex.EncodeToString(b)
	tx.txHash = h.String()
	return nil
}

func (tx *TxBuilder) Sign(privateKey []byte) (string, error) {
	if tx == nil || len(tx.sigHash) != 1 {
		return "", fmt.Errorf("tx not built")
	}
	hashBig, ok := new(big.Int).SetString(strings.TrimPrefix(tx.sigHash[0], "0x"), 16)
	if !ok {
		return "", fmt.Errorf("bad sigHash")
	}
	r, s, err := curve.Sign(hashBig, new(big.Int).SetBytes(privateKey))
	if err != nil {
		return "", fmt.Errorf("sign: %v", err)
	}
	return "0x" + r.Text(16) + "_0x" + s.Text(16), nil
}

func (tx *TxBuilder) ConcatSignature(signature string, isDerFormat bool) (string, error) {
	if tx == nil || tx.unsignedHex == "" {
		return "", fmt.Errorf("tx not built")
	}
	b, err := hex.DecodeString(tx.unsignedHex)
	if err != nil {
		return "", fmt.Errorf("decode unsignedHex: %v", err)
	}
	var txn rpc.InvokeTxnV3
	if err := json.Unmarshal(b, &txn); err != nil {
		return "", fmt.Errorf("unmarshal txn: %v", err)
	}
	parts := strings.Split(signature, "_")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid signature")
	}
	rFelt, err := utils.HexToFelt(parts[0])
	if err != nil {
		return "", fmt.Errorf("bad r: %v", err)
	}
	sFelt, err := utils.HexToFelt(parts[1])
	if err != nil {
		return "", fmt.Errorf("bad s: %v", err)
	}
	txn.Signature = []*felt.Felt{rFelt, sFelt}
	out, err := json.Marshal(&txn)
	if err != nil {
		return "", fmt.Errorf("marshal signed txn: %v", err)
	}
	return hex.EncodeToString(out), nil
}

func (tx *TxBuilder) resourceBounds() (*rpc.ResourceBoundsMapping, error) {
	l1a, err := decToU64(orZero(tx.L1GasMaxAmount))
	if err != nil {
		return nil, err
	}
	l1p, err := decToU128(orZero(tx.L1GasMaxPrice))
	if err != nil {
		return nil, err
	}
	l1da, err := decToU64(orZero(tx.L1DataGasMaxAmount))
	if err != nil {
		return nil, err
	}
	l1dp, err := decToU128(orZero(tx.L1DataGasMaxPrice))
	if err != nil {
		return nil, err
	}
	l2a, err := decToU64(orZero(tx.L2GasMaxAmount))
	if err != nil {
		return nil, err
	}
	l2p, err := decToU128(orZero(tx.L2GasMaxPrice))
	if err != nil {
		return nil, err
	}
	return &rpc.ResourceBoundsMapping{
		L1Gas:     rpc.ResourceBounds{MaxAmount: l1a, MaxPricePerUnit: l1p},
		L1DataGas: rpc.ResourceBounds{MaxAmount: l1da, MaxPricePerUnit: l1dp},
		L2Gas:     rpc.ResourceBounds{MaxAmount: l2a, MaxPricePerUnit: l2p},
	}, nil
}

func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func decToU64(dec string) (rpc.U64, error) {
	n, ok := new(big.Int).SetString(dec, 10)
	if !ok {
		return "", fmt.Errorf("bad uint %q", dec)
	}
	return rpc.U64("0x" + n.Text(16)), nil
}

func decToU128(dec string) (rpc.U128, error) {
	n, ok := new(big.Int).SetString(dec, 10)
	if !ok {
		return "", fmt.Errorf("bad uint %q", dec)
	}
	return rpc.U128("0x" + n.Text(16)), nil
}

func (tx *TxBuilder) GetTxHash() string       { return tx.txHash }
func (tx *TxBuilder) GetSigHash() []string    { return tx.sigHash }
func (tx *TxBuilder) GetUnsignedHex() string  { return tx.unsignedHex }
func (tx *TxBuilder) SetSigHash(s []string)   { tx.sigHash = s }
func (tx *TxBuilder) SetUnsignedHex(s string) { tx.unsignedHex = s }

type (
	Ingredient struct {
		TxType          string `json:"txType"`
		ContractAddress string `json:"contractAddress"`
		Sender          string `json:"sender,omitempty"`
		SenderPublicKey string `json:"senderPublicKey,omitempty"`
		Recipient       string `json:"recipient"`
		Amount          string `json:"amount"`
		Nonce           string `json:"nonce"`
		// v3 resource bounds (decimal strings)
		L1GasMaxAmount     string `json:"l1GasMaxAmount"`
		L1GasMaxPrice      string `json:"l1GasMaxPrice"`
		L1DataGasMaxAmount string `json:"l1DataGasMaxAmount"`
		L1DataGasMaxPrice  string `json:"l1DataGasMaxPrice"`
		L2GasMaxAmount     string `json:"l2GasMaxAmount"`
		L2GasMaxPrice      string `json:"l2GasMaxPrice"`
		Tip                string `json:"tip"`
	}
	TxBuilder struct {
		*Ingredient
		unsignedHex string
		sigHash     []string
		txHash      string
	}
)
