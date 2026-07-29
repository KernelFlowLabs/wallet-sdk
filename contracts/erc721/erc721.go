package erc721

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

const ABIERC721 = `[
{"inputs":[{"internalType":"address","name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},
{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"safeTransferFromWithData","outputs":[],"stateMutability":"nonpayable","type":"function"},
{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"transferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},
{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"approve","outputs":[],"stateMutability":"nonpayable","type":"function"},
{"inputs":[{"internalType":"address","name":"operator","type":"address"},{"internalType":"bool","name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},
{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"getApproved","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},
{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
{"inputs":[{"internalType":"uint256","name":"index","type":"uint256"}],"name":"tokenByIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"uint256","name":"index","type":"uint256"}],"name":"tokenOfOwnerByIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]`

type CallErc721In struct {
	Address   string `json:"address,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Operator  string `json:"operator,omitempty"`
	Approved  bool   `json:"approved,omitempty"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	TokenId   string `json:"tokenId,omitempty"`
	Index     string `json:"index,omitempty"`
	Data      string `json:"data,omitempty"`
}

func PackPayloadForErc721(function string, params []byte) (string, error) {
	p := CallErc721In{}
	if params != nil {
		if err := json.Unmarshal(params, &p); err != nil {
			return "", fmt.Errorf("wrong params type: %w", err)
		}
	}
	pp, err := p.toNative()
	if err != nil {
		return "", err
	}

	jsonABI, err := abi.JSON(strings.NewReader(ABIERC721))
	if err != nil {
		return "", err
	}

	var data []byte
	switch function {
	case "name", "symbol", "totalSupply":
		data, err = jsonABI.Pack(function)
	case "balanceOf":
		data, err = jsonABI.Pack(function, pp.Owner)
	case "ownerOf", "tokenURI", "getApproved":
		data, err = jsonABI.Pack(function, pp.TokenId)
	case "tokenByIndex":
		data, err = jsonABI.Pack(function, pp.Index)
	case "tokenOfOwnerByIndex":
		data, err = jsonABI.Pack(function, pp.Owner, pp.Index)
	case "isApprovedForAll":
		data, err = jsonABI.Pack(function, pp.Owner, pp.Operator)
	case "approve":
		data, err = jsonABI.Pack(function, pp.Recipient, pp.TokenId)
	case "setApprovalForAll":
		data, err = jsonABI.Pack(function, pp.Operator, p.Approved)
	case "transferFrom", "safeTransferFrom":
		data, err = jsonABI.Pack(function, pp.From, pp.To, pp.TokenId)
	case "safeTransferFromWithData":
		data, err = jsonABI.Pack("safeTransferFromWithData", pp.From, pp.To, pp.TokenId, pp.Data)
	default:
		return "", fmt.Errorf("unsupported erc721 function: %s", function)
	}
	if err != nil {
		return "", fmt.Errorf("pack %s: %w", function, err)
	}
	return hex.EncodeToString(data), nil
}

func UnpackReturnsForErc721(function string, returns []byte) (string, error) {
	jsonABI, err := abi.JSON(strings.NewReader(ABIERC721))
	if err != nil {
		return "", err
	}
	out, err := jsonABI.Unpack(function, returns)
	if err != nil {
		return "", err
	}
	if len(out) != 1 {
		return "", fmt.Errorf("erc721 %s: expected 1 return, got %d", function, len(out))
	}
	switch function {
	case "name", "symbol", "tokenURI":
		s, ok := out[0].(string)
		if !ok {
			return "", fmt.Errorf("erc721 %s: return not string", function)
		}
		return s, nil
	case "balanceOf", "totalSupply", "tokenByIndex", "tokenOfOwnerByIndex":
		n, ok := out[0].(*big.Int)
		if !ok {
			return "", fmt.Errorf("erc721 %s: return not uint256", function)
		}
		return n.String(), nil
	case "ownerOf", "getApproved":
		addr, ok := out[0].(common.Address)
		if !ok {
			return "", fmt.Errorf("erc721 %s: return not address", function)
		}
		return strings.ToLower(addr.Hex()), nil
	case "isApprovedForAll":
		b, ok := out[0].(bool)
		if !ok {
			return "", fmt.Errorf("erc721 %s: return not bool", function)
		}
		return strconv.FormatBool(b), nil
	}
	return "", fmt.Errorf("unsupported erc721 read function: %s", function)
}

type callErc721Native struct {
	Owner     common.Address
	Operator  common.Address
	From      common.Address
	To        common.Address
	Recipient common.Address
	TokenId   *big.Int
	Index     *big.Int
	Data      []byte
}

func (in CallErc721In) toNative() (*callErc721Native, error) {
	owner := in.Owner
	if owner == "" {
		owner = in.Address
	}
	n := &callErc721Native{
		Owner:     common.HexToAddress(owner),
		Operator:  common.HexToAddress(in.Operator),
		From:      common.HexToAddress(in.From),
		To:        common.HexToAddress(in.To),
		Recipient: common.HexToAddress(in.Recipient),
	}
	if in.TokenId != "" {
		v, ok := new(big.Int).SetString(in.TokenId, 10)
		if !ok {
			return nil, fmt.Errorf("invalid tokenId: %s", in.TokenId)
		}
		n.TokenId = v
	}
	if in.Index != "" {
		v, ok := new(big.Int).SetString(in.Index, 10)
		if !ok {
			return nil, fmt.Errorf("invalid index: %s", in.Index)
		}
		n.Index = v
	}
	if in.Data != "" {
		b, err := hex.DecodeString(strings.TrimPrefix(in.Data, "0x"))
		if err != nil {
			return nil, fmt.Errorf("invalid data hex: %w", err)
		}
		n.Data = b
	}
	return n, nil
}
