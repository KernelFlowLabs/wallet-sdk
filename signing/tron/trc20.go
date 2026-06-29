package tron

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const ABITRC20 = `[{"inputs":[{"internalType":"string","name":"name","type":"string"},{"internalType":"string","name":"symbol","type":"string"},{"internalType":"uint8","name":"decimal","type":"uint8"}],"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"spender","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"account","type":"address"}],"name":"Paused","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"account","type":"address"}],"name":"Unpaused","type":"event"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"spender","type":"address"}],"name":"allowance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"approve","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"burn","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"subtractedValue","type":"uint256"}],"name":"decreaseAllowance","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"addedValue","type":"uint256"}],"name":"increaseAllowance","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"pause","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"paused","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"new_controller","type":"address"},{"internalType":"address","name":"new_pauser","type":"address"}],"name":"setAdmin","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"recipient","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"transfer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"sender","type":"address"},{"internalType":"address","name":"recipient","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"transferFrom","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"unpause","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

// for call contract
type (
	CallTrc20In struct {
		Address   string `json:"address,omitempty"`
		Owner     string `json:"owner,omitempty"`
		Spender   string `json:"spender,omitempty"`
		From      string `json:"from,omitempty"`
		To        string `json:"to,omitempty"`
		Recipient string `json:"recipient,omitempty"`
		Amount    string `json:"amount,omitempty"`
	}
	callTrc20NativeIn struct {
		Address   common.Address
		Owner     common.Address
		Spender   common.Address
		From      common.Address
		To        common.Address
		Recipient common.Address
		Amount    *big.Int
	}

	UnpackTrc20 struct {
		Function string
		CallTrc20In
	}
)

func PackPayloadForTrc20(function string, params []byte) (string, error) {
	p := CallTrc20In{}
	if params != nil {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return "", fmt.Errorf("wrong params type")
		}
	}

	pp := &callTrc20NativeIn{}
	{
		pp.Address = common.HexToAddress(ConvertToHex(p.Address))
		pp.Owner = common.HexToAddress(ConvertToHex(p.Owner))
		pp.Spender = common.HexToAddress(ConvertToHex(p.Spender))
		pp.From = common.HexToAddress(ConvertToHex(p.From))
		pp.To = common.HexToAddress(ConvertToHex(p.To))
		pp.Recipient = common.HexToAddress(ConvertToHex(p.Recipient))
		if p.Amount != "" {
			amount, ok := big.NewInt(0).SetString(p.Amount, 10)
			if !ok {
				return "", fmt.Errorf("invalid amount")
			}
			pp.Amount = amount
		}
	}

	jsonABI, err := abi.JSON(strings.NewReader(ABITRC20))
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
	case "mint", "burn", "transfer":
		data, err = jsonABI.Pack(function, pp.Recipient, pp.Amount)
		if err != nil {
			return "", err
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
		return "", fmt.Errorf("failed to packpayload for trc20, function not found")
	}
	return hex.EncodeToString(data), nil
}

func UnpackPayloadForTrc20(payload string) (*UnpackTrc20, error) {
	payload = strings.TrimPrefix(payload, "0x")
	if len(payload) < 8 {
		return nil, fmt.Errorf("payload too short")
	}
	jsonABI, err := abi.JSON(strings.NewReader(ABITRC20))
	if err != nil {
		return nil, fmt.Errorf("failed to abi.JSON, err=%v", err)
	}
	methodSig, err := hex.DecodeString(payload[0:8])
	if err != nil {
		return nil, fmt.Errorf("failed to DecodeString, err=%v", err)
	}
	method, err := jsonABI.MethodById(methodSig)
	if err != nil {
		return nil, fmt.Errorf("failed to MethodById, err=%v", err)
	}
	paramsBytes, err := hex.DecodeString(payload[8:])
	if err != nil {
		return nil, fmt.Errorf("failed to DecodeString, err=%v", err)
	}
	paramsList, err := method.Inputs.Unpack(paramsBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to method.Inputs.Unpack, err=%v", err)
	}
	function := method.Name

	out := &UnpackTrc20{
		Function: function,
	}

	switch function {
	case "transferOwnership":
		if len(payload) != 72 {
			return nil, fmt.Errorf("inappropriate length of payload of %s", function)
		} else if len(paramsList) != 1 {
			return nil, fmt.Errorf("inappropriate length of paramsList of %s", function)
		}
		address, ok := paramsList[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for address of %s", function)
		}
		out.CallTrc20In = CallTrc20In{
			Address: strings.ToLower(address.Hex()),
		}
	case "mint", "burn", "transfer":
		if len(payload) != 136 {
			return nil, fmt.Errorf("inappropriate length of payload of %s", function)
		} else if len(paramsList) != 2 {
			return nil, fmt.Errorf("inappropriate length of paramsList of %s", function)
		}
		recipient, ok := paramsList[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for recipient of %s", function)
		}
		amount, ok := paramsList[1].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for amount of %s", function)
		}
		out.CallTrc20In = CallTrc20In{
			Recipient: strings.ToLower(recipient.Hex()),
			Amount:    amount.String(),
		}
	case "transferFrom":
		if len(payload) != 200 {
			return nil, fmt.Errorf("inappropriate length of payload of %s", function)
		} else if len(paramsList) != 3 {
			return nil, fmt.Errorf("inappropriate length of paramsList of %s", function)
		}
		from, ok := paramsList[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for from of %s", function)
		}
		to, ok := paramsList[1].(common.Address)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for to of %s", function)
		}
		amount, ok := paramsList[2].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("inappropriate type for amount of %s", function)
		}
		out.CallTrc20In = CallTrc20In{
			From:   strings.ToLower(from.Hex()),
			To:     strings.ToLower(to.Hex()),
			Amount: amount.String(),
		}
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
		out.CallTrc20In = CallTrc20In{
			Spender: strings.ToLower(spender.Hex()),
			Amount:  amount.String(),
		}
	default:
		return nil, fmt.Errorf("function %s for trc20 not found", function)
	}

	return out, nil
}
