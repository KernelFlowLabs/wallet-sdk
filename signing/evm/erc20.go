package evm

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const ABIERC20 = `[{"inputs":[{"internalType":"string","name":"_name","type":"string"},{"internalType":"string","name":"_symbol","type":"string"}],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"allowance","type":"uint256"},{"internalType":"uint256","name":"needed","type":"uint256"}],"name":"ERC20InsufficientAllowance","type":"error"},{"inputs":[{"internalType":"address","name":"sender","type":"address"},{"internalType":"uint256","name":"balance","type":"uint256"},{"internalType":"uint256","name":"needed","type":"uint256"}],"name":"ERC20InsufficientBalance","type":"error"},{"inputs":[{"internalType":"address","name":"approver","type":"address"}],"name":"ERC20InvalidApprover","type":"error"},{"inputs":[{"internalType":"address","name":"receiver","type":"address"}],"name":"ERC20InvalidReceiver","type":"error"},{"inputs":[{"internalType":"address","name":"sender","type":"address"}],"name":"ERC20InvalidSender","type":"error"},{"inputs":[{"internalType":"address","name":"spender","type":"address"}],"name":"ERC20InvalidSpender","type":"error"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"spender","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"_from","type":"address"},{"indexed":true,"internalType":"address","name":"_to","type":"address"},{"indexed":false,"internalType":"uint256","name":"_value","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"data","type":"bytes"}],"name":"DataDelivery","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"}],"name":"allowance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"_from","type":"address"},{"internalType":"uint256","name":"_value","type":"uint256"}],"name":"burn","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"callBack","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"factory","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"_maxSupply","type":"uint256"},{"internalType":"uint8","name":"_decimals","type":"uint8"}],"name":"initialize","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"maxSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"_to","type":"address"},{"internalType":"uint256","name":"_value","type":"uint256"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"_callBack","type":"address"}],"name":"setCallBack","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transfer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"bytes","name":"_data","type":"bytes"}],"name":"transfer_WITHDATA","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}]`
const ABIERC20ForTransferWithData = `[{"inputs":[{"internalType":"string","name":"_name","type":"string"},{"internalType":"string","name":"_symbol","type":"string"}],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"allowance","type":"uint256"},{"internalType":"uint256","name":"needed","type":"uint256"}],"name":"ERC20InsufficientAllowance","type":"error"},{"inputs":[{"internalType":"address","name":"sender","type":"address"},{"internalType":"uint256","name":"balance","type":"uint256"},{"internalType":"uint256","name":"needed","type":"uint256"}],"name":"ERC20InsufficientBalance","type":"error"},{"inputs":[{"internalType":"address","name":"approver","type":"address"}],"name":"ERC20InvalidApprover","type":"error"},{"inputs":[{"internalType":"address","name":"receiver","type":"address"}],"name":"ERC20InvalidReceiver","type":"error"},{"inputs":[{"internalType":"address","name":"sender","type":"address"}],"name":"ERC20InvalidSender","type":"error"},{"inputs":[{"internalType":"address","name":"spender","type":"address"}],"name":"ERC20InvalidSpender","type":"error"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"spender","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"_from","type":"address"},{"indexed":true,"internalType":"address","name":"_to","type":"address"},{"indexed":false,"internalType":"uint256","name":"_value","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"data","type":"bytes"}],"name":"DataDelivery","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"}],"name":"allowance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"_from","type":"address"},{"internalType":"uint256","name":"_value","type":"uint256"}],"name":"burn","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"callBack","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"factory","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"_maxSupply","type":"uint256"},{"internalType":"uint8","name":"_decimals","type":"uint8"}],"name":"initialize","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"maxSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"_to","type":"address"},{"internalType":"uint256","name":"_value","type":"uint256"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"_callBack","type":"address"}],"name":"setCallBack","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"bytes","name":"_data","type":"bytes"}],"name":"transfer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}]`

// for call contract
type (
	CallErc20In struct {
		Address   string `json:"address,omitempty"`
		Owner     string `json:"owner,omitempty"`
		Spender   string `json:"spender,omitempty"`
		From      string `json:"from,omitempty"`
		To        string `json:"to,omitempty"`
		Recipient string `json:"recipient,omitempty"`

		Amount string `json:"amount,omitempty"`
		Memo   string `json:"data,omitempty"`
	}
	callErc20InNative struct {
		Address   common.Address
		Owner     common.Address
		Spender   common.Address
		From      common.Address
		To        common.Address
		Recipient common.Address

		Amount *big.Int
		Memo   []byte
	}
	UnpackErc20 struct {
		Function string
		CallErc20In
	}
)

func PackPayloadForErc20(function string, params []byte) (string, error) {
	p := CallErc20In{}
	if params != nil {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return "", fmt.Errorf("wrong params type")
		}
	}

	var ok bool
	pp := &callErc20InNative{}
	{
		pp.Address = common.HexToAddress(p.Address)
		pp.Owner = common.HexToAddress(p.Owner)
		pp.Spender = common.HexToAddress(p.Spender)
		pp.From = common.HexToAddress(p.From)
		pp.To = common.HexToAddress(p.To)
		pp.Recipient = common.HexToAddress(p.Recipient)

		if p.Amount != "" {
			pp.Amount, ok = big.NewInt(0).SetString(p.Amount, 10)
			if !ok {
				return "", fmt.Errorf("invalid amount")
			}
		}
		if p.Memo != "" {
			memo, err := hex.DecodeString(strings.TrimPrefix(p.Memo, "0x"))
			if err != nil {
				return "", fmt.Errorf("failed to DecodeString for Memo")
			}
			pp.Memo = memo
		}
	}

	jsonABI, err := abi.JSON(strings.NewReader(ABIERC20))
	if err != nil {
		return "", err
	}
	jsonABI1, err := abi.JSON(strings.NewReader(ABIERC20ForTransferWithData))
	if err != nil {
		return "", err
	}
	var data []byte

	switch function {
	case "name", "symbol", "decimals", "totalSupply":
		data, err = jsonABI.Pack(function)
		if err != nil {
			return "", fmt.Errorf("failed to Pack, err=%v", err)
		}
	case "balanceOf", "transferOwnership":
		data, err = jsonABI.Pack(function, pp.Address)
		if err != nil {
			return "", fmt.Errorf("failed to Pack, err=%v", err)
		}
	case "allowance":
		data, err = jsonABI.Pack(function, pp.Owner, pp.Spender)
		if err != nil {
			return "", fmt.Errorf("failed to Pack, err=%v", err)
		}
	case "mint", "burn":
		data, err = jsonABI.Pack(function, pp.Recipient, pp.Amount)
		if err != nil {
			return "", fmt.Errorf("failed to Pack, err=%v", err)
		}
	case "transfer":
		if p.Memo != "" {
			data, err = jsonABI1.Pack(function, pp.Recipient, pp.Amount, pp.Memo)
		} else {
			data, err = jsonABI.Pack(function, pp.Recipient, pp.Amount)
		}
		if err != nil {
			return "", fmt.Errorf("failed to Pack, err=%v", err)
		}
	case "transferFrom":
		data, err = jsonABI.Pack(function, pp.From, pp.To, pp.Amount)
		if err != nil {
			return "", fmt.Errorf("failed to Pack, err=%v", err)
		}
	case "approve":
		data, err = jsonABI.Pack(function, pp.Spender, pp.Amount)
		if err != nil {
			return "", fmt.Errorf("failed to Pack, err=%v", err)
		}
	default:
		return "", fmt.Errorf("failed to packpayload for erc20, function not found")
	}
	return hex.EncodeToString(data), nil
}
func UnpackPayloadForErc20(payload string) (*UnpackErc20, error) {
	payload = strings.TrimPrefix(payload, "0x")
	if len(payload) < 8 {
		return nil, fmt.Errorf("payload too short")
	}
	jsonABI, err := abi.JSON(strings.NewReader(ABIERC20))
	if err != nil {
		return nil, err
	}
	jsonABI1, err := abi.JSON(strings.NewReader(ABIERC20ForTransferWithData))
	if err != nil {
		return nil, err
	}
	methodSig, err := hex.DecodeString(payload[0:8])
	if err != nil {
		return nil, err
	}
	method, err := jsonABI.MethodById(methodSig)
	if err != nil {
		method, err = jsonABI1.MethodById(methodSig)
		if err != nil {
			return nil, err
		}
	}
	paramsBytes, err := hex.DecodeString(payload[8:])
	if err != nil {
		return nil, err
	}
	paramsList, err := method.Inputs.Unpack(paramsBytes)
	if err != nil {
		return nil, err
	}
	function := method.Name

	out := &UnpackErc20{
		Function: function,
	}
	switch function {
	case "transfer":
		if len(paramsList) != 2 && len(paramsList) != 3 {
			return nil, fmt.Errorf("inappropriate length paramsList of %s", function)
		}
		recipient, ok := paramsList[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for common.Address of %s", function)
		}
		amount, ok := paramsList[1].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for big.Int of %s", function)
		}
		var memo string
		if len(paramsList) == 3 {
			memoBytes, ok := paramsList[2].([]byte)
			if !ok {
				return nil, fmt.Errorf("inappropriate type for []byte of %s", function)
			}
			memo = hex.EncodeToString(memoBytes)
		}
		out.Recipient = strings.ToLower(recipient.Hex())
		out.Amount = amount.String()
		out.Memo = memo
	case "approve":
		if len(payload) != 136 {
			return nil, fmt.Errorf("inappropriate length of payload of %s", function)
		} else if len(paramsList) != 2 {
			return nil, fmt.Errorf("inappropriate length of paramsList of %s", function)
		}
		spender, ok := paramsList[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for common.Address of %s", function)
		}
		amount, ok := paramsList[1].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for bool of %s", function)
		}
		out.Spender = strings.ToLower(spender.Hex())
		out.Amount = amount.String()
	case "mint", "burn":
		if len(payload) != 136 {
			return nil, fmt.Errorf("inappropriate length of payload of %s", function)
		} else if len(paramsList) != 2 {
			return nil, fmt.Errorf("inappropriate length of paramsList of %s", function)
		}
		recipient, ok := paramsList[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for common.Address of %s", function)
		}
		amount, ok := paramsList[1].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for bool of %s", function)
		}
		out.Recipient = strings.ToLower(recipient.Hex())
		out.Amount = amount.String()
	default:
		return nil, fmt.Errorf("function %s for erc20 not found", function)
	}
	return out, nil
}
func UnpackReturnsForErc20(function string, returns []byte) (string, error) {
	jsonABI, err := abi.JSON(strings.NewReader(ABIERC20))
	if err != nil {
		return "", err
	}
	out, err := jsonABI.Unpack(function, returns)
	if err != nil {
		return "", err
	}
	switch function {
	case "name":
		if len(out) != 1 {
			return "", fmt.Errorf("invalid length of result")
		}
		name, ok := out[0].(string)
		if !ok {
			return "", fmt.Errorf("out[0] is not string")
		}
		return string(name), nil
	case "symbol":
		if len(out) != 1 {
			return "", fmt.Errorf("invalid length of result")
		}
		symbol, ok := out[0].(string)
		if !ok {
			return "", fmt.Errorf("out[0] is not string")
		}
		return string(symbol), nil
	case "decimals":
		if len(out) != 1 {
			return "", fmt.Errorf("invalid length of result")
		}
		decimals, ok := out[0].(uint8)
		if !ok {
			return "", fmt.Errorf("out[0] is not big.Int")
		}
		return strconv.FormatInt(int64(decimals), 10), nil
	case "allowance":
		if len(out) != 1 {
			return "", fmt.Errorf("invalid length of result")
		}
		allowance, ok := out[0].(*big.Int)
		if !ok {
			return "", fmt.Errorf("out[0] is not big.Int")
		}
		return allowance.String(), nil
	}
	return "", fmt.Errorf("unsupported function")
}
