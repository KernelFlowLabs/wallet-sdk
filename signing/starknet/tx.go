package starknet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/dontpanicdao/caigo"
	caigotypes "github.com/dontpanicdao/caigo/types"

	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

const ethContractAddress = "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7"

var snMainChainID = hexToBig("0x534e5f4d41494e")

var mask128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

func hx(n *big.Int) string { return "0x" + n.Text(16) }

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
	sender := hexToBig(tx.Sender)
	recipient := hexToBig(tx.Recipient)
	amount, ok := new(big.Int).SetString(tx.Amount, 10)
	if !ok {
		return fmt.Errorf("bad amount %s", tx.Amount)
	}
	nonce, ok := new(big.Int).SetString(tx.Nonce, 10)
	if !ok {
		return fmt.Errorf("bad nonce %s", tx.Nonce)
	}
	amtLo := new(big.Int).And(amount, mask128)
	amtHi := new(big.Int).Rsh(amount, 128)

	selector := caigotypes.GetSelectorFromName("transfer")
	calldata := []*big.Int{
		big.NewInt(1), hexToBig(contract), selector,
		big.NewInt(0), big.NewInt(3), big.NewInt(3),
		recipient, amtLo, amtHi,
	}

	l1, l2, l1data, tip, err := tx.bounds()
	if err != nil {
		return err
	}
	h := v3InvokeHash(sender, nonce, snMainChainID, calldata, l1, l2, l1data, tip)

	jtx := v3JSON{
		Type:          "INVOKE",
		SenderAddress: hx(sender),
		Calldata:      bigsToHex(calldata),
		Version:       "0x3",
		Signature:     []string{},
		Nonce:         hx(nonce),
		ResourceBounds: resourceBoundsJSON{
			L1Gas:     resBoundJSON{hx(l1.maxAmount), hx(l1.maxPrice)},
			L1DataGas: resBoundJSON{hx(l1data.maxAmount), hx(l1data.maxPrice)},
			L2Gas:     resBoundJSON{hx(l2.maxAmount), hx(l2.maxPrice)},
		},
		Tip:                   hx(new(big.Int).SetUint64(tip)),
		PaymasterData:         []string{},
		AccountDeploymentData: []string{},
		NonceDataMode:         "L1",
		FeeMode:               "L1",
	}
	b, err := json.Marshal(jtx)
	if err != nil {
		return fmt.Errorf("marshal tx: %v", err)
	}
	tx.sigHash = []string{hx(h)}
	tx.unsignedHex = hex.EncodeToString(b)
	tx.txHash = hx(h)
	return nil
}

func (tx *TxBuilder) Sign(privateKey []byte) (string, error) {
	if tx == nil || len(tx.sigHash) != 1 {
		return "", fmt.Errorf("tx not built")
	}
	hashBig := hexToBig(tx.sigHash[0])
	r, s, err := caigo.Curve.Sign(hashBig, new(big.Int).SetBytes(privateKey))
	if err != nil {
		return "", fmt.Errorf("sign: %v", err)
	}
	return hx(r) + "_" + hx(s), nil
}

func (tx *TxBuilder) ConcatSignature(signature string, isDerFormat bool) (string, error) {
	if tx == nil || tx.unsignedHex == "" {
		return "", fmt.Errorf("tx not built")
	}
	b, err := hex.DecodeString(tx.unsignedHex)
	if err != nil {
		return "", fmt.Errorf("decode unsignedHex: %v", err)
	}
	var jtx v3JSON
	if err := json.Unmarshal(b, &jtx); err != nil {
		return "", fmt.Errorf("unmarshal tx: %v", err)
	}
	parts := strings.Split(signature, "_")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid signature")
	}
	jtx.Signature = []string{parts[0], parts[1]}
	out, err := json.Marshal(jtx)
	if err != nil {
		return "", fmt.Errorf("marshal signed tx: %v", err)
	}
	return hex.EncodeToString(out), nil
}

func (tx *TxBuilder) bounds() (l1, l2, l1data resourceBound, tip uint64, err error) {
	dec := func(name, s string) (*big.Int, error) {
		if s == "" {
			return big.NewInt(0), nil
		}
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return nil, fmt.Errorf("bad %s %q", name, s)
		}
		return n, nil
	}
	var e error
	get := func(name, s string) *big.Int {
		if e != nil {
			return nil
		}
		var n *big.Int
		n, e = dec(name, s)
		return n
	}
	l1 = resourceBound{"L1_GAS", get("l1GasMaxAmount", tx.L1GasMaxAmount), get("l1GasMaxPrice", tx.L1GasMaxPrice)}
	l2 = resourceBound{"L2_GAS", get("l2GasMaxAmount", tx.L2GasMaxAmount), get("l2GasMaxPrice", tx.L2GasMaxPrice)}
	l1data = resourceBound{"L1_DATA", get("l1DataGasMaxAmount", tx.L1DataGasMaxAmount), get("l1DataGasMaxPrice", tx.L1DataGasMaxPrice)}
	tipBig := get("tip", tx.Tip)
	if e != nil {
		return l1, l2, l1data, 0, e
	}
	return l1, l2, l1data, tipBig.Uint64(), nil
}

func bigsToHex(xs []*big.Int) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = hx(x)
	}
	return out
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

	resBoundJSON struct {
		MaxAmount       string `json:"max_amount"`
		MaxPricePerUnit string `json:"max_price_per_unit"`
	}
	resourceBoundsJSON struct {
		L1Gas     resBoundJSON `json:"l1_gas"`
		L1DataGas resBoundJSON `json:"l1_data_gas"`
		L2Gas     resBoundJSON `json:"l2_gas"`
	}
	v3JSON struct {
		Type                  string             `json:"type"`
		SenderAddress         string             `json:"sender_address"`
		Calldata              []string           `json:"calldata"`
		Version               string             `json:"version"`
		Signature             []string           `json:"signature"`
		Nonce                 string             `json:"nonce"`
		ResourceBounds        resourceBoundsJSON `json:"resource_bounds"`
		Tip                   string             `json:"tip"`
		PaymasterData         []string           `json:"paymaster_data"`
		AccountDeploymentData []string           `json:"account_deployment_data"`
		NonceDataMode         string             `json:"nonce_data_availability_mode"`
		FeeMode               string             `json:"fee_data_availability_mode"`
	}
)
