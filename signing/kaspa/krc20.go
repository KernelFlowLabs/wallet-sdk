package kaspa

import (
	"encoding/hex"
	"encoding/json"

	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/util"
)

type Krc20Params struct {
	P    string `json:"p"`
	Op   string `json:"op"`
	Tick string `json:"tick"`
	Amt  string `json:"amt"`
	To   string `json:"to"`
}

func GetKrc20Params(t *Krc20Params, pubKey []byte) (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	sb := NewScriptBuilder()
	sb.AddData(pubKey)
	sb.AddOp(OpCheckSig)
	sb.AddOp(OpFalse)
	sb.AddOp(OpIf)
	sb.AddData([]byte("kasplex"))
	sb.AddInt64(0)
	sb.AddData(data)
	sb.AddOp(OpEndIf)

	redeemScript, err := sb.Script()
	if err != nil {
		return "", err
	}
	p2shScript, err := PayToScriptHashScript(redeemScript)
	if err != nil {
		return "", err
	}
	prefix, err := util.ParsePrefix("kaspa")
	if err != nil {
		return "", err
	}
	_, p2shAcc, err := ExtractScriptPubKeyAddress(&externalapi.ScriptPublicKey{
		Script:  p2shScript,
		Version: 0,
	}, &dagconfig.Params{Prefix: prefix})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(redeemScript) + "_" +
		hex.EncodeToString(p2shScript) + "_" +
		p2shAcc.String(), nil
}
