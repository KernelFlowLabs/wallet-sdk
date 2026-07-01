package starknet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/NethermindEth/starknet.go/utils"

	chainrpc "github.com/KernelFlowLabs/wallet-sdk/rpc"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

var _ chainrpc.BasicChainHandler = (*Handler)(nil)

const ethContractAddress = "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7"

type Handler struct {
	rpc *chainrpc.Request
}

func NewHandler(url string) (*Handler, error) {
	return &Handler{rpc: chainrpc.NewRequest(url, map[string]string{"Content-Type": "application/json"})}, nil
}

func balanceOfSelector() string { return utils.GetSelectorFromNameFelt("balanceOf").String() }

func expandAddress(address string) string {
	rest := strings.TrimPrefix(address, "0x")
	if len(rest) < 64 {
		return "0x" + strings.Repeat("0", 64-len(rest)) + rest
	}
	return address
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_blockNumber"}
	res := &_GetHeightRes{}
	if err := h.rpc.Post(ctx, res, "", req); err != nil {
		return "", fmt.Errorf("fail to GetHeight, err=%v", err)
	} else if res.Error != nil {
		return "", fmt.Errorf("fail to GetHeight, errMsg=%v", res.Error.Message)
	}
	return strconv.FormatUint(res.Result, 10), nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	if contractAddress == signing.MagicContactAddressForNative {
		contractAddress = ethContractAddress
	}
	req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_call",
		Params: []interface{}{
			map[string]interface{}{
				"calldata":             []string{address},
				"contract_address":     contractAddress,
				"entry_point_selector": balanceOfSelector(),
			},
			"latest",
		}}
	res := &_GetBalanceRes{}
	if err := h.rpc.Post(ctx, res, "", req); err != nil {
		return "", fmt.Errorf("fail to GetBalance, err=%v", err)
	} else if res.Error != nil {
		return "", fmt.Errorf("fail to GetBalance, errMsg=%v", res.Error.Message)
	}
	// uint256 low + high
	low := hexToBig(res.Result[0])
	if len(res.Result) > 1 {
		high := hexToBig(res.Result[1])
		low.Add(low, new(big.Int).Lsh(high, 128))
	}
	return low.String(), nil
}

func (h *Handler) GetTransfersByHash(ctx context.Context, hash string, confirmation uint64) (*chainrpc.TxTransfers, error) {
	r := &chainrpc.TxTransfers{Hash: hash}
	txResult, err := h.CheckTx(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("fail to CheckTx, err=%v", err)
	} else if txResult.Status != signing.TxStatusSucceeded {
		if txResult.Status == signing.TxStatusFailed {
			r.Rejected = true
		}
		r.ErrMsg = "not a succeeded tx"
		return r, nil
	}
	// v3 invoke calldata = [num_calls, to, selector, calldata_len, recipient, amount_lo, amount_hi]
	req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_getTransactionByHash",
		Params: []string{hash}}
	res := &_GetTransactionByHash{}
	if err := h.rpc.Post(ctx, res, "", req); err != nil {
		return nil, fmt.Errorf("fail to getTransactionByHash, err=%v", err)
	} else if res.Error != nil {
		return nil, fmt.Errorf("fail to getTransactionByHash, errMsg=%v", res.Error.Message)
	} else if len(res.Result.Calldata) != 7 {
		r.Rejected = true
		r.ErrMsg = "unsupported calldata shape"
		return r, nil
	}
	contractAddress := expandAddress(res.Result.Calldata[1])
	if contractAddress == ethContractAddress {
		contractAddress = signing.MagicContactAddressForNative
	}
	r.Transfers = append(r.Transfers, &chainrpc.Transfer{
		Sender:          expandAddress(res.Result.SenderAddress),
		Recipient:       expandAddress(res.Result.Calldata[4]),
		Amount:          hexToBig(res.Result.Calldata[5]).String(),
		ContractAddress: contractAddress,
	})
	return r, nil
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	signedBytes, err := hex.DecodeString(strings.TrimPrefix(signedHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("SendTx DecodeString, err=%v", err)
	}
	var txn json.RawMessage = signedBytes
	req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_addInvokeTransaction",
		Params: map[string]interface{}{"invoke_transaction": txn}}
	res := &_SendTxRes{}
	if err := h.rpc.Post(ctx, res, "", req); err != nil {
		return "", fmt.Errorf("fail to SendTx, err=%v", err)
	} else if res.Error != nil {
		return "", fmt.Errorf("fail to SendTx, errMsg=%v", res.Error.Message)
	}
	return expandAddress(res.Result.TransactionHash), nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	result := &chainrpc.TxResult{}
	req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_getTransactionReceipt",
		Params: []string{hash}}
	res := &_GetTransaction{}
	if err := h.rpc.Post(ctx, res, "", req); err != nil {
		return nil, fmt.Errorf("fail to CheckTx, err=%v", err)
	} else if res.Error != nil {
		if strings.Contains(res.Error.Message, "hash not found") || strings.Contains(res.Error.Message, "not found") {
			result.Status = signing.TxStatusPending
			return result, nil
		}
		result.Status = signing.TxStatusFailed
		result.ErrMsg = res.Error.Message
		return result, nil
	}
	switch res.Result.ExecutionStatus {
	case "SUCCEEDED":
		result.Status = signing.TxStatusSucceeded
		result.Height = strconv.FormatInt(int64(res.Result.BlockNumber), 10)
	case "REVERTED":
		result.Status = signing.TxStatusFailed
		result.Height = strconv.FormatInt(int64(res.Result.BlockNumber), 10)
	default:
		result.Status = signing.TxStatusPending
	}
	return result, nil
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "getNonce":
		req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_getNonce",
			Params: []string{"latest", params}}
		res := &_GetNonceRes{}
		if err := h.rpc.Post(ctx, res, "", req); err != nil {
			return "", fmt.Errorf("fail to GetNonce, err=%v", err)
		} else if res.Error != nil {
			if strings.Contains(res.Error.Message, "not found") {
				return "0", nil
			}
			return "", fmt.Errorf("fail to GetNonce, errMsg=%v", res.Error.Message)
		}
		if res.Result == "" || res.Result == "0" {
			return "0", nil
		}
		return hexToBig(res.Result).String(), nil
	case "getContractDecimals":
		req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_call",
			Params: []interface{}{
				map[string]interface{}{
					"calldata":             []string{},
					"contract_address":     params,
					"entry_point_selector": utils.GetSelectorFromNameFelt("decimals").String(),
				},
				"latest",
			}}
		res := &_GetDecimalRes{}
		if err := h.rpc.Post(ctx, res, "", req); err != nil {
			return "", fmt.Errorf("fail to GetDecimal, err=%v", err)
		} else if res.Error != nil {
			return "", fmt.Errorf("fail to GetDecimal, errMsg=%v", res.Error.Message)
		}
		return hexToBig(res.Result[0]).String(), nil
	case "estimateFee":
		// params: hex-encoded signed InvokeTxnV3 JSON.
		txBytes, err := hex.DecodeString(strings.TrimPrefix(params, "0x"))
		if err != nil {
			return "", fmt.Errorf("estimateFee: bad params, err=%v", err)
		}
		txn := json.RawMessage(txBytes)
		req := &_BaseRequest{JsonRPC: "2.0", ID: "0", Method: "starknet_estimateFee",
			Params: []interface{}{[]json.RawMessage{txn}, []interface{}{}, "pending"}}
		res := &_EstimateFeeRes{}
		if err := h.rpc.Post(ctx, res, "", req); err != nil {
			return "", fmt.Errorf("fail to estimateFee, err=%v", err)
		} else if res.Error != nil {
			return "", fmt.Errorf("fail to estimateFee, errMsg=%v", res.Error.Message)
		} else if len(res.Result) == 0 {
			return "", fmt.Errorf("estimateFee: empty result")
		}
		fe := res.Result[0]
		bounds := map[string]string{
			"l1GasMaxAmount":     bump(fe.L1GasConsumed),
			"l1GasMaxPrice":      bump(fe.L1GasPrice),
			"l1DataGasMaxAmount": bump(fe.L1DataGasConsumed),
			"l1DataGasMaxPrice":  bump(fe.L1DataGasPrice),
			"l2GasMaxAmount":     bump(fe.L2GasConsumed),
			"l2GasMaxPrice":      bump(fe.L2GasPrice),
			"overallFee":         hexToBig(fe.OverallFee).String(),
		}
		out, _ := json.Marshal(bounds)
		return string(out), nil
	}
	return "", fmt.Errorf("unsupported function")
}

func hexToBig(s string) *big.Int {
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	if !ok {
		return big.NewInt(0)
	}
	return n
}

// bump adds a 50% safety margin to an estimated resource value (hex in, dec out).
func bump(hexVal string) string {
	n := hexToBig(hexVal)
	n.Mul(n, big.NewInt(3))
	n.Div(n, big.NewInt(2))
	return n.String()
}
