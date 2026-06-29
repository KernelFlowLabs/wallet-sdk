package utxo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	chainrpc "github.com/KernelFlowLabs/wallet-sdk/rpc"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	walletutxo "github.com/KernelFlowLabs/wallet-sdk/signing/utxo"

	"github.com/shopspring/decimal"
)

var _ chainrpc.BasicChainHandler = (*Handler)(nil)

type Handler struct {
	rpc              *chainrpc.Request
	scanApi          *chainrpc.Request
	network          string
	blockcypherToken string
	bcMu             sync.Mutex
	bcLastReq        time.Time
}

func NewHandler(url, network, blockcypherToken string) (*Handler, error) {
	if network == walletutxo.NetworkEnumForSYS {
		h := &Handler{blockcypherToken: blockcypherToken}
		rpc := chainrpc.NewRequest(url, map[string]string{
			"content-type": "text/plain",
		})
		h.rpc = rpc
		h.network = network
		return h, nil
	}

	parts := strings.SplitN(url, ";", 2)
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return nil, fmt.Errorf("invalid url: empty rpc part")
	}

	rpcPart := strings.TrimSpace(parts[0])
	scanApiUrl := ""
	if len(parts) == 2 {
		scanApiUrl = strings.TrimSpace(parts[1])
	}

	var (
		rpcBase string
		rpcUser string
		rpcPass string
	)

	if strings.Contains(rpcPart, "&") {
		tmp := strings.SplitN(rpcPart, "&", 2)
		rpcBase = strings.TrimSpace(tmp[0])
		authStr := strings.TrimSpace(tmp[1])

		up := strings.SplitN(authStr, "@", 2)
		if len(up) != 2 {
			return nil, fmt.Errorf("rpc authorization string error")
		}
		rpcUser = up[0]
		rpcPass = up[1]
	} else {
		rpcBase = rpcPart
	}

	h := &Handler{blockcypherToken: blockcypherToken}

	rpcHeaders := map[string]string{
		"content-type": "text/plain",
	}

	if rpcUser != "" && rpcPass != "" {
		authorization := base64.StdEncoding.EncodeToString([]byte(rpcUser + ":" + rpcPass))
		rpcHeaders["Authorization"] = "Basic " + authorization
	}

	rpc := chainrpc.NewRequest(rpcBase, rpcHeaders)
	h.rpc = rpc

	if scanApiUrl != "" {
		scanHeaders := map[string]string{}
		h.scanApi = chainrpc.NewRequest(scanApiUrl, scanHeaders)
	}

	h.network = network
	return h, nil
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	req := &_BaseRequest{
		JsonRPC: "1.0",
		ID:      "1",
		Method:  "getblockcount",
	}
	out := &_GetBlockCountRes{}
	err := h.rpc.Post(ctx, out, "", req)
	if err != nil {
		return "", fmt.Errorf("failed to GetHeight, err=%v", err)
	} else if out.Error.Code != 0 {
		return "", fmt.Errorf("failed to GetHeight, errMsg=%v", out.Error.Message)
	} else if out.Result == 0 {
		return "", fmt.Errorf("got zero")
	}

	return strconv.FormatUint(out.Result, 10), nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	if contractAddress != signing.MagicContactAddressForNative {
		return "", fmt.Errorf("invalid contractAddress")
	}
	switch h.network {
	case walletutxo.NetworkEnumForBTC, walletutxo.NetworkEnumForBTCP2TR:
		return h.getBalanceForBTC(ctx, address)
	case walletutxo.NetworkEnumForLTC:
		return h.getBalanceForLTC(ctx, address)
	case walletutxo.NetworkEnumForDOGE:
		return h.getBalanceForDOGE(ctx, address)
	}
	return "", fmt.Errorf("failed to get balance, invalid network")
}

func (h *Handler) GetTransfersByHash(ctx context.Context, hash string,
	confirmation uint64, withInternal bool) (*chainrpc.TxTransfers, error) {
	result := &chainrpc.TxTransfers{
		Hash: hash,
	}
	txResult, err := h.CheckTx(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to CheckTx, err=%v", err)
	} else if txResult.Status == signing.TxStatusFailed {
		result.Rejected = true
		if txResult.ErrMsg != "" {
			result.ErrMsg = txResult.ErrMsg
		} else {
			result.ErrMsg = "not a succeeded tx"
		}
		return result, nil
	} else if txResult.Status == signing.TxStatusPending {
		result.ErrMsg = "pending tx"
		return result, nil
	} else if txResult.Status != signing.TxStatusSucceeded {
		result.ErrMsg = "not a succeeded tx"
		return result, nil
	}

	height, _ := strconv.ParseUint(txResult.Height, 10, 64)
	latestHeightStr, _ := h.GetHeight(ctx)
	latestHeight, _ := strconv.ParseUint(latestHeightStr, 10, 64)
	if height != 0 && latestHeight != 0 {
		//confirmation := getConfirmation(h.network)
		if latestHeight-height < confirmation {
			result.ErrMsg = fmt.Sprintf("tx succeeded.But current confirmation number %d hasn't meet "+
				"expected number %d", latestHeight-height, confirmation)
			return result, nil
		}
	}

	switch h.network {
	case walletutxo.NetworkEnumForBTC, walletutxo.NetworkEnumForBTCP2TR:
		transfers, balanceChange, err := h.getTxTransferForBTC(ctx, hash)
		if err != nil {
			result.ErrMsg = err.Error()
			result.Rejected = true
			return result, nil
		}
		result.Transfers = transfers
		result.BalanceChange = balanceChange
		return result, nil
	case walletutxo.NetworkEnumForLTC:
	case walletutxo.NetworkEnumForDOGE:
	case walletutxo.NetworkEnumForSYS:
	}
	return nil, fmt.Errorf("invalid network")
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	req := &_BaseRequest{
		JsonRPC: "1.0",
		ID:      "1",
		Method:  "sendrawtransaction",
		Params: []string{
			signedHex,
		},
	}
	out := &_SendRawTransactionRes{}
	err := h.rpc.Post(ctx, out, "", req)
	if err != nil {
		return "", fmt.Errorf("failed to sendrawtransaction, err=%v", err)
	} else if out.Error.Code != 0 {
		return "", fmt.Errorf("failed to sendrawtransaction, errMsg=%v", out.Error.Message)
	}
	return out.Result, nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	switch h.network {
	case walletutxo.NetworkEnumForBTC, walletutxo.NetworkEnumForBTCP2TR:
		return h.getTxStatusForBTC(ctx, hash)
	case walletutxo.NetworkEnumForLTC:
		return h.getTxStatusForLTC(ctx, hash)
	case walletutxo.NetworkEnumForDOGE:
		return h.getTxStatusForDOGE(ctx, hash)
	}
	return nil, fmt.Errorf("failed to get hash status, invalid network")
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("CallContract not supported")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "getUtxo":
		res := &signing.UtxoList{}
		var err error
		switch h.network {
		case walletutxo.NetworkEnumForBTC, walletutxo.NetworkEnumForBTCP2TR:
			res, err = h.getUtxoForBTC(ctx, params)
			if err != nil {
				return "", err
			}
		case walletutxo.NetworkEnumForLTC:
			res, err = h.getUtxoForLTC(ctx, params)
			if err != nil {
				return "", err
			}
		case walletutxo.NetworkEnumForDOGE:
			res, err = h.getUtxoForDOGE(ctx, params)
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("failed to get utxo, invalid network")
		}
		resBytes, err := json.Marshal(res)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal for res, err=%v", err)
		}
		return string(resBytes), nil
	case "getByteFee":
		switch h.network {
		case walletutxo.NetworkEnumForBTC, walletutxo.NetworkEnumForBTCP2TR:
			return h.getByteFeeForBTC(ctx)
		case walletutxo.NetworkEnumForLTC:
			return h.getByteFeeForLTC(ctx)
		case walletutxo.NetworkEnumForDOGE:
			return h.getByteFeeForDOGE(ctx)
		default:
			return "", fmt.Errorf("failed to get utxo, invalid network")
		}
	case "getBlockByNumber":
		blockNumber, err := strconv.ParseInt(params, 10, 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse block number, err=%v", err)
		}
		blockHash, err := h.getBlockHashByNumber(ctx, blockNumber)
		if err != nil {
			return "", fmt.Errorf("failed to get block hash by number, err=%v", err)
		}

		block, err := h.getBlockByHash(ctx, blockHash)
		if err != nil {
			return "", fmt.Errorf("failed to get block by hash, err=%v", err)
		}
		return block, nil
	case "getTokenDecimals":
		if params == signing.MagicContactAddressForNative {
			return "8", nil
		}
		return "", fmt.Errorf("unsupported token address")
	}
	return "", fmt.Errorf("unsupported function")
}

// unexported
func (h *Handler) getBlockHashByNumber(ctx context.Context, blockNumber int64) (string, error) {
	req := &_BaseRequest{
		JsonRPC: "1.0",
		ID:      "1",
		Method:  "getblockhash",
		Params:  []interface{}{blockNumber},
	}

	out := &_GetBlockHashRes{}
	err := h.rpc.Post(ctx, out, "", req)
	if err != nil {
		return "", fmt.Errorf("failed to get block hash, err=%v", err)
	} else if out.Error.Code != 0 {
		return "", fmt.Errorf("failed to get block hash, errMsg=%v", out.Error.Message)
	}

	return out.Result, nil
}
func (h *Handler) getBlockByHash(ctx context.Context, blockHash string) (string, error) {
	req := &_BaseRequest{
		JsonRPC: "1.0",
		ID:      "1",
		Method:  "getblock",
		Params:  []interface{}{blockHash, 2},
	}

	out := &_GetBlockVerboseRes{}
	err := h.rpc.Post(ctx, out, "", req)
	if err != nil {
		return "", fmt.Errorf("failed to get block, err=%v", err)
	} else if out.Error.Code != 0 {
		return "", fmt.Errorf("failed to get block, errMsg=%v", out.Error.Message)
	}
	blockJSON, err := json.Marshal(out.Result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal block, err=%v", err)
	}

	return string(blockJSON), nil
}
func (h *Handler) getTxTransferForBTC(ctx context.Context, hash string) ([]*chainrpc.Transfer, []*chainrpc.BalanceChange, error) {
	path := "tx/" + hash
	res := &_ElectrsTxRes{}
	err := h.scanApi.Get(ctx, res, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tx from Electrs, err=%v", err)
	}

	inputAmounts := make(map[string]*big.Int)
	for _, vin := range res.Vin {
		addr := vin.Prevout.ScriptpubkeyAddress
		if addr == "" {
			continue
		}
		if inputAmounts[addr] == nil {
			inputAmounts[addr] = big.NewInt(0)
		}
		inputAmounts[addr].Add(inputAmounts[addr], big.NewInt(vin.Prevout.Value))
	}

	outputAmounts := make(map[string]*big.Int)
	for _, vout := range res.Vout {
		addr := vout.ScriptpubkeyAddress
		if addr == "" {
			continue
		}
		if outputAmounts[addr] == nil {
			outputAmounts[addr] = big.NewInt(0)
		}
		outputAmounts[addr].Add(outputAmounts[addr], big.NewInt(vout.Value))
	}

	allAddresses := make(map[string]bool)
	for addr := range inputAmounts {
		allAddresses[addr] = true
	}
	for addr := range outputAmounts {
		allAddresses[addr] = true
	}

	balanceChanges := make([]*chainrpc.BalanceChange, 0)
	realSenders := make([]string, 0)
	realReceivers := make(map[string]*big.Int)

	for addr := range allAddresses {
		inputAmt := inputAmounts[addr]
		if inputAmt == nil {
			inputAmt = big.NewInt(0)
		}
		outputAmt := outputAmounts[addr]
		if outputAmt == nil {
			outputAmt = big.NewInt(0)
		}

		change := new(big.Int).Sub(outputAmt, inputAmt)

		if change.Cmp(big.NewInt(0)) != 0 {
			balanceChanges = append(balanceChanges, &chainrpc.BalanceChange{
				Address:         addr,
				ContractAddress: signing.MagicContactAddressForNative,
				Change:          change.String(),
			})

			if change.Cmp(big.NewInt(0)) > 0 {
				realReceivers[addr] = change
			} else {
				realSenders = append(realSenders, addr)
			}
		}
	}

	if len(realSenders) == 0 || len(realReceivers) == 0 {
		return nil, balanceChanges, fmt.Errorf("no valid transfers found")
	}

	primarySender := realSenders[0]

	transfers := make([]*chainrpc.Transfer, 0)
	for recipient, amount := range realReceivers {
		transfers = append(transfers, &chainrpc.Transfer{
			Sender:          primarySender,
			Recipient:       recipient,
			Amount:          amount.String(),
			ContractAddress: signing.MagicContactAddressForNative,
		})
	}

	return transfers, balanceChanges, nil
}

func (h *Handler) getByteFeeForBTC(ctx context.Context) (string, error) {
	return h.getByteFeeFromBlockcypher(ctx, "BTC")
}
func (h *Handler) getBalanceForBTC(ctx context.Context, address string) (string, error) {
	return h.getBalanceFromBlockcypher(ctx, "BTC", address)
}
func (h *Handler) getUtxoForBTC(ctx context.Context, address string) (*signing.UtxoList, error) {
	return h.getUtxoFromBlockcypher(ctx, "BTC", address)
}
func (h *Handler) getTxStatusForBTC(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	return h.getTxStatusFromBlockcypher(ctx, "BTC", hash)
}

func (h *Handler) getByteFeeForLTC(ctx context.Context) (string, error) {
	return h.getByteFeeFromBlockcypher(ctx, "LTC")
}
func (h *Handler) getBalanceForLTC(ctx context.Context, address string) (string, error) {
	return h.getBalanceFromBlockcypher(ctx, "LTC", address)
}
func (h *Handler) getUtxoForLTC(ctx context.Context, address string) (*signing.UtxoList, error) {
	return h.getUtxoFromBlockcypher(ctx, "LTC", address)
}
func (h *Handler) getTxStatusForLTC(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	return h.getTxStatusFromBlockcypher(ctx, "LTC", hash)
}

func (h *Handler) getByteFeeForDOGE(ctx context.Context) (string, error) {
	return h.getByteFeeFromBlockcypherByCurl(ctx, "DOGE")
}
func (h *Handler) getBalanceForDOGE(ctx context.Context, address string) (string, error) {
	return h.getBalanceFromBlockcypherByCurl(ctx, "DOGE", address)
}
func (h *Handler) getUtxoForDOGE(ctx context.Context, address string) (*signing.UtxoList, error) {
	return h.getUtxoFromBlockcypherByCurl(ctx, "DOGE", address)
}
func (h *Handler) getTxStatusForDOGE(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	return h.getTxStatusFromBlockcypherByCurl(ctx, "DOGE", hash)
}

// Blockcypher

func (h *Handler) getByteFeeFromBlockcypher(ctx context.Context, chainName string) (string, error) {
	h.bcLimit()
	out := &_BlockcypherFeeRes{}
	path := "v1/" + strings.ToLower(chainName) + "/main?token=" + h.blockcypherToken
	err := h.scanApi.Get(ctx, out, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get byteFee, err=%v", err)
	} else if out == nil {
		return "", fmt.Errorf("failed to get byteFee, return is nil")
	}
	feePerByte := float64(1)
	if out.MediumFeePerKb != 0 {
		feePerByte = out.MediumFeePerKb / 1000
	} else if out.HighFeePerKb != 0 {
		feePerByte = out.HighFeePerKb / 1000
	} else if out.LowFeePerKb != 0 {
		feePerByte = out.LowFeePerKb / 1000
	}
	return decimal.NewFromFloat(feePerByte).Ceil().String(), nil
}
func (h *Handler) getBalanceFromBlockcypher(ctx context.Context, chainName, address string) (string, error) {
	h.bcLimit()
	path := "v1/" + strings.ToLower(chainName) + "/main/addrs/" + address + "?token=" + h.blockcypherToken
	res := &_BlockcypherUtxoRes{}

	req := url.Values{}
	req.Set("unspentOnly", "true")
	err := h.scanApi.Get(ctx, res, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get balance from blockcypher, err=%v", err)
	} else if res == nil {
		return "", fmt.Errorf("failed to get balance from blockcypher, return is nil")
	}
	return strconv.FormatInt(res.FinalBalance, 10), nil
}
func (h *Handler) getTxStatusFromBlockcypher(ctx context.Context, chainName, hash string) (*chainrpc.TxResult, error) {
	h.bcLimit()
	path := "v1/" + strings.ToLower(chainName) + "/main/txs/" + hash + "?token=" + h.blockcypherToken
	res := &_BlockcypherTxRes{}
	err := h.scanApi.Get(ctx, res, path, nil)
	result := &chainrpc.TxResult{}
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(strings.ToLower(errMsg), "not found") ||
			strings.Contains(errMsg, "404") ||
			strings.Contains(errMsg, "500") {
			result.Status = signing.TxStatusPending
			return result, nil
		}
		return nil, fmt.Errorf("failed to get hash status, err=%v", err)
	} else if res == nil {
		return nil, fmt.Errorf("failed to get hash status, return nil")
	} else if res.BlockHeight > 0 &&
		res.DoubleSpend == false &&
		res.Confirmations > 0 &&
		res.Confidence == 1 {
		result.Status = signing.TxStatusSucceeded
		result.Height = strconv.FormatInt(res.BlockHeight, 10)
		return result, nil
	} else {
		result.Status = signing.TxStatusPending
		return result, nil
	}
}
func (h *Handler) getUtxoFromBlockcypher(ctx context.Context, chainName, address string) (*signing.UtxoList, error) {
	h.bcLimit()
	path := "v1/" + strings.ToLower(chainName) + "/main/addrs/" + address + "?token=" + h.blockcypherToken
	res := &_BlockcypherUtxoRes{}

	req := url.Values{}
	req.Set("unspentOnly", "true")
	err := h.scanApi.Get(ctx, res, path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get utxo from blockcypher, err=%v", err)
	} else if res == nil {
		return nil, fmt.Errorf("failed to get balance from blockcypher, return is nil")
	}

	utxoList := &signing.UtxoList{}
	network := getNetwork(chainName)
	if network == "" {
		return nil, fmt.Errorf("failed to get network for %s", chainName)
	}
	for _, v := range res.Txrefs {
		if v.Confirmed == "" {
			continue
		} else if v.Confirmations == 0 {
			continue
		}
		pubKeyScript, err := walletutxo.AddressToScriptPubKey(address, network)
		if err != nil {
			return nil, fmt.Errorf("failed to AddressToScriptPubKey, addr=%s, err=%v", address, err)
		}
		utxo := &signing.UtxoInfo{
			Hash:   v.TxHash,
			Script: pubKeyScript,
			Index:  strconv.FormatInt(v.TxOutputN, 10),
			Value:  strconv.FormatInt(v.Value, 10),
		}
		utxoList.List = append(utxoList.List, utxo)
	}
	return utxoList, nil
}
func (h *Handler) getByteFeeFromBlockcypherByCurl(ctx context.Context, chainName string) (string, error) {
	h.bcLimit()
	url := fmt.Sprintf("https://api.blockcypher.com/v1/%s/main?token=%s",
		strings.ToLower(chainName),
		h.blockcypherToken)
	cmd := exec.CommandContext(ctx, "curl",
		"-s",
		"-H", "User-Agent: curl/7.81.0",
		url)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("failed to get byteFee via curl, err=%v, stderr=%s",
				err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to get byteFee via curl, err=%v", err)
	}
	if len(output) == 0 {
		return "", fmt.Errorf("failed to get byteFee, empty response")
	}
	if strings.Contains(string(output), "error") || strings.Contains(string(output), "Limits reached") {
		return "", fmt.Errorf("failed to get byteFee, api error: %s", string(output))
	}
	out := &_BlockcypherFeeRes{}
	if err := json.Unmarshal(output, out); err != nil {
		return "", fmt.Errorf("failed to parse byteFee response, err=%v, output=%s",
			err, string(output))
	}
	feePerByte := float64(1)
	if out.MediumFeePerKb != 0 {
		feePerByte = out.MediumFeePerKb / 1000
	} else if out.HighFeePerKb != 0 {
		feePerByte = out.HighFeePerKb / 1000
	} else if out.LowFeePerKb != 0 {
		feePerByte = out.LowFeePerKb / 1000
	}

	return decimal.NewFromFloat(feePerByte).Ceil().String(), nil
}
func (h *Handler) getBalanceFromBlockcypherByCurl(ctx context.Context, chainName, address string) (string, error) {
	h.bcLimit()

	url := fmt.Sprintf("https://api.blockcypher.com/v1/%s/main/addrs/%s?token=%s&unspentOnly=true",
		strings.ToLower(chainName),
		address,
		h.blockcypherToken)

	cmd := exec.CommandContext(ctx, "curl", "-s", "-H", "User-Agent: curl/7.81.0", url)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("failed to get balance from blockcypher, err=%v, stderr=%s",
				err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to get balance from blockcypher, err=%v", err)
	}

	if len(output) == 0 {
		return "", fmt.Errorf("failed to get balance from blockcypher, empty response")
	}

	if strings.Contains(string(output), "\"error\"") {
		return "", fmt.Errorf("failed to get balance from blockcypher, api error: %s", string(output))
	}

	res := &_BlockcypherUtxoRes{}
	if err := json.Unmarshal(output, res); err != nil {
		return "", fmt.Errorf("failed to parse balance response, err=%v, output=%s",
			err, string(output))
	}

	if res == nil {
		return "", fmt.Errorf("failed to get balance from blockcypher, return is nil")
	}

	return strconv.FormatInt(res.FinalBalance, 10), nil
}
func (h *Handler) getTxStatusFromBlockcypherByCurl(ctx context.Context, chainName, hash string) (*chainrpc.TxResult, error) {
	h.bcLimit()

	url := fmt.Sprintf("https://api.blockcypher.com/v1/%s/main/txs/%s?token=%s",
		strings.ToLower(chainName),
		hash,
		h.blockcypherToken)

	cmd := exec.CommandContext(ctx, "curl", "-s", "-H", "User-Agent: curl/7.81.0", url)
	output, err := cmd.Output()

	result := &chainrpc.TxResult{}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			errMsg := string(exitErr.Stderr)
			if strings.Contains(strings.ToLower(errMsg), "not found") ||
				strings.Contains(errMsg, "404") ||
				strings.Contains(errMsg, "500") {
				result.Status = signing.TxStatusPending
				return result, nil
			}
		}
		return nil, fmt.Errorf("failed to get hash status, err=%v", err)
	}

	if len(output) == 0 {
		result.Status = signing.TxStatusPending
		return result, nil
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "not found") ||
		strings.Contains(outputStr, "404") ||
		strings.Contains(outputStr, "500") ||
		strings.Contains(outputStr, "\"error\"") {
		result.Status = signing.TxStatusPending
		return result, nil
	}

	res := &_BlockcypherTxRes{}
	if err := json.Unmarshal(output, res); err != nil {
		return nil, fmt.Errorf("failed to parse tx status response, err=%v", err)
	}

	if res == nil {
		return nil, fmt.Errorf("failed to get hash status, return nil")
	}

	if res.BlockHeight > 0 &&
		res.DoubleSpend == false &&
		res.Confirmations > 0 &&
		res.Confidence == 1 {
		result.Status = signing.TxStatusSucceeded
		result.Height = strconv.FormatInt(res.BlockHeight, 10)
		return result, nil
	}

	result.Status = signing.TxStatusPending
	return result, nil
}
func (h *Handler) getUtxoFromBlockcypherByCurl(ctx context.Context, chainName, address string) (*signing.UtxoList, error) {
	h.bcLimit()

	url := fmt.Sprintf("https://api.blockcypher.com/v1/%s/main/addrs/%s?token=%s&unspentOnly=true",
		strings.ToLower(chainName),
		address,
		h.blockcypherToken)

	cmd := exec.CommandContext(ctx, "curl", "-s", "-H", "User-Agent: curl/7.81.0", url)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("failed to get utxo from blockcypher, err=%v, stderr=%s",
				err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to get utxo from blockcypher, err=%v", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("failed to get utxo from blockcypher, empty response")
	}

	if strings.Contains(string(output), "\"error\"") {
		return nil, fmt.Errorf("failed to get utxo from blockcypher, api error: %s", string(output))
	}

	res := &_BlockcypherUtxoRes{}
	if err := json.Unmarshal(output, res); err != nil {
		return nil, fmt.Errorf("failed to parse utxo response, err=%v, output=%s",
			err, string(output))
	}

	if res == nil {
		return nil, fmt.Errorf("failed to get balance from blockcypher, return is nil")
	}

	utxoList := &signing.UtxoList{}
	network := getNetwork(chainName)
	if network == "" {
		return nil, fmt.Errorf("failed to get network for %s", chainName)
	}

	for _, v := range res.Txrefs {
		if v.Confirmed == "" {
			continue
		} else if v.Confirmations == 0 {
			continue
		} else if v.Value <= 1000000 {
			continue
		}

		pubKeyScript, err := walletutxo.AddressToScriptPubKey(address, network)
		if err != nil {
			return nil, fmt.Errorf("failed to AddressToScriptPubKey, addr=%s, err=%v", address, err)
		}

		utxo := &signing.UtxoInfo{
			Hash:   v.TxHash,
			Script: pubKeyScript,
			Index:  strconv.FormatInt(v.TxOutputN, 10),
			Value:  strconv.FormatInt(v.Value, 10),
		}
		utxoList.List = append(utxoList.List, utxo)
	}

	return utxoList, nil
}
func (h *Handler) bcLimit() {
	h.bcMu.Lock()
	defer h.bcMu.Unlock()

	minInterval := 5 * time.Second

	now := time.Now()
	elapsed := now.Sub(h.bcLastReq)

	if elapsed < minInterval {
		time.Sleep(minInterval - elapsed)
	}

	h.bcLastReq = time.Now()
}

// types
type (
	_BaseRequest struct {
		JsonRPC string      `json:"jsonrpc"`
		ID      string      `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}
	_BaseResponse struct {
		ID    string `json:"id"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
	}
	_GetBlockCountRes struct {
		_BaseResponse
		Result uint64 `json:"result"`
	}
	_SendRawTransactionRes struct {
		_BaseResponse
		Result string `json:"result"`
	}
	_GetBlockHashRes struct {
		_BaseResponse
		Result string `json:"result"`
	}
	_GetBlockRes struct {
		_BaseResponse
		Result string `json:"result"`
	}
	_GetBlockVerboseRes struct {
		_BaseResponse
		Result RpcBlock `json:"result"`
	}
	RpcBlock struct {
		Hash          string  `json:"hash"`
		Confirmations int64   `json:"confirmations"`
		Height        int64   `json:"height"`
		Version       int32   `json:"version"`
		MerkleRoot    string  `json:"merkleroot"`
		Time          int64   `json:"time"`
		Nonce         uint32  `json:"nonce"`
		Bits          string  `json:"bits"`
		Difficulty    float64 `json:"difficulty"`
		PreviousHash  string  `json:"previousblockhash"`
		NextHash      string  `json:"nextblockhash,omitempty"`
		Tx            []RpcTx `json:"tx"`
	}
	RpcTx struct {
		Txid     string    `json:"txid"`
		Version  int32     `json:"version"`
		Locktime uint32    `json:"locktime"`
		Vin      []RpcVin  `json:"vin"`
		Vout     []RpcVout `json:"vout"`
	}
	RpcVin struct {
		Coinbase  string `json:"coinbase,omitempty"`
		Txid      string `json:"txid"`
		Vout      uint32 `json:"vout"`
		ScriptSig struct {
			Asm string `json:"asm"`
			Hex string `json:"hex"`
		} `json:"scriptSig"`
		TxinWitness []string `json:"txinwitness,omitempty"` //only for btc
		Sequence    uint32   `json:"sequence"`
	}
	RpcVout struct {
		Value        float64         `json:"value"`
		N            int             `json:"n"`
		ScriptPubKey RpcScriptPubKey `json:"scriptPubKey"`
	}
	RpcScriptPubKey struct {
		Asm       string   `json:"asm"`
		Hex       string   `json:"hex"`
		ReqSigs   int      `json:"reqSigs,omitempty"`
		Type      string   `json:"type"`
		Address   string   `json:"address,omitempty"`   // for BTC new version
		Addresses []string `json:"addresses,omitempty"` // for BTC old version and DOGE
	}
)

// electrs
type (
	_ElectrsAddressRes struct {
		ChainStats struct {
			FundedTxoCount int    `json:"funded_txo_count"`
			FundedTxoSum   uint64 `json:"funded_txo_sum"`
			SpentTxoCount  int    `json:"spent_txo_count"`
			SpentTxoSum    uint64 `json:"spent_txo_sum"`
			TxCount        int    `json:"tx_count"`
		} `json:"chain_stats"`
		MempoolStats struct {
			FundedTxoCount int    `json:"funded_txo_count"`
			FundedTxoSum   uint64 `json:"funded_txo_sum"`
			SpentTxoCount  int    `json:"spent_txo_count"`
			SpentTxoSum    uint64 `json:"spent_txo_sum"`
			TxCount        int    `json:"tx_count"`
		} `json:"mempool_stats"`
	}
	_ElectrsUtxo struct {
		TxID   string `json:"txid"`
		Vout   uint64 `json:"vout"`
		Status struct {
			Confirmed bool `json:"confirmed"`
		} `json:"status"`
		Value uint64 `json:"value"`
	}
	_ElectrsUtxoRes []_ElectrsUtxo

	_ElectrsTxRes struct {
		Fee      uint64 `json:"fee"`
		Weight   int    `json:"weight"`
		Size     int    `json:"size"`
		Version  int    `json:"version"`
		Locktime int    `json:"locktime"`
		Txid     string `json:"txid"`
		Vin      []struct {
			Txid    string `json:"txid"`
			Vout    int    `json:"vout"`
			Prevout struct {
				Scriptpubkey        string `json:"scriptpubkey"`
				ScriptpubkeyAsm     string `json:"scriptpubkey_asm"`
				ScriptpubkeyType    string `json:"scriptpubkey_type"`
				ScriptpubkeyAddress string `json:"scriptpubkey_address"`
				Value               int64  `json:"value"`
			} `json:"prevout"`
			Scriptsig    string   `json:"scriptsig"`
			ScriptsigAsm string   `json:"scriptsig_asm"`
			Witness      []string `json:"witness"`
			IsCoinbase   bool     `json:"is_coinbase"`
			Sequence     int64    `json:"sequence"`
		} `json:"vin"`
		Vout []struct {
			Scriptpubkey        string `json:"scriptpubkey"`
			ScriptpubkeyAsm     string `json:"scriptpubkey_asm"`
			ScriptpubkeyType    string `json:"scriptpubkey_type"`
			ScriptpubkeyAddress string `json:"scriptpubkey_address"`
			Value               int64  `json:"value"`
		} `json:"vout"`
		Status struct {
			Confirmed   bool  `json:"confirmed"`
			BlockHeight int64 `json:"block_height"`
			BlockTime   int64 `json:"block_time"`
		} `json:"status"`
	}

	_ElectrsFeeRes struct {
		One   float64 `json:"1"`
		Two   float64 `json:"2"`
		Three float64 `json:"3"`
	}
)

// ltc blockcypher
type (
	_BlockcypherFeeRes struct {
		Name             string    `json:"name"`
		Height           int64     `json:"height"`
		Hash             string    `json:"hash"`
		Time             time.Time `json:"time"`
		LatestUrl        string    `json:"latest_url"`
		PreviousHash     string    `json:"previous_hash"`
		PreviousUrl      string    `json:"previous_url"`
		PeerCount        int64     `json:"peer_count"`
		UnconfirmedCount int64     `json:"unconfirmed_count"`
		HighFeePerKb     float64   `json:"high_fee_per_kb"`
		MediumFeePerKb   float64   `json:"medium_fee_per_kb"`
		LowFeePerKb      float64   `json:"low_fee_per_kb"`
		LastForkHeight   int64     `json:"last_fork_height"`
		LastForkHash     string    `json:"last_fork_hash"`
	}
	_BlockcypherUtxoRes struct {
		Address            string `json:"address"`
		TotalReceived      int64  `json:"total_received"`
		TotalSent          int64  `json:"total_sent"`
		Balance            int64  `json:"balance"`
		UnconfirmedBalance int64  `json:"unconfirmed_balance"`
		FinalBalance       int64  `json:"final_balance"`
		NTx                int64  `json:"n_tx"`
		UnconfirmedNTx     int64  `json:"unconfirmed_n_tx"`
		FinalNTx           int64  `json:"final_n_tx"`
		Txrefs             []struct {
			TxHash        string `json:"tx_hash"`
			BlockHeight   uint64 `json:"block_height"`
			TxInputN      int64  `json:"tx_input_n"`
			TxOutputN     int64  `json:"tx_output_n"`
			Value         int64  `json:"value"`
			RefBalance    int64  `json:"ref_balance"`
			Spent         bool   `json:"spent"`
			Confirmations int64  `json:"confirmations"`
			Confirmed     string `json:"confirmed"`
			DoubleSpend   bool   `json:"double_spend"`
		} `json:"txrefs"`
		TxUrl string `json:"tx_url"`
	}
	_BlockcypherTxRes struct {
		BlockHash     string   `json:"block_hash"`
		BlockHeight   int64    `json:"block_height"`
		BlockIndex    int64    `json:"block_index"`
		Hash          string   `json:"hash"`
		Addresses     []string `json:"addresses"`
		Total         int64    `json:"total"`
		Fees          int64    `json:"fees"`
		Size          int64    `json:"size"`
		Vsize         int64    `json:"vsize"`
		Preference    string   `json:"preference"`
		RelayedBy     string   `json:"relayed_by"`
		Confirmed     string   `json:"confirmed"`
		Received      string   `json:"received"`
		Ver           int64    `json:"ver"`
		DoubleSpend   bool     `json:"double_spend"`
		VinSz         int64    `json:"vin_sz"`
		VoutSz        int64    `json:"vout_sz"`
		OptInRbf      bool     `json:"opt_in_rbf"`
		Confirmations int64    `json:"confirmations"`
		Confidence    int64    `json:"confidence"`
		Inputs        []struct {
			PrevHash    string   `json:"prev_hash"`
			OutputIndex int      `json:"output_index"`
			Script      string   `json:"script"`
			OutputValue int      `json:"output_value"`
			Sequence    int64    `json:"sequence"`
			Addresses   []string `json:"addresses"`
			ScriptType  string   `json:"script_type"`
			Age         int      `json:"age"`
		} `json:"inputs"`
		Outputs []struct {
			Value      int64    `json:"value"`
			Script     string   `json:"script"`
			Addresses  []string `json:"addresses"`
			ScriptType string   `json:"script_type"`
		} `json:"outputs"`
	}
)

// dogeinfo
type (
	_DogeInfoAddressRes struct {
		Status string `json:"status"`
		Data   struct {
			Confirmed   string `json:"confirmed"`
			Unconfirmed string `json:"unconfirmed"`
			Error       string `json:"error_message"`
		} `json:"data"`
	}
	_DogeInfoUtxoRes struct {
		Status string `json:"status"`
		Data   struct {
			Outputs []struct {
				Hash    string `json:"hash"`
				Index   uint64 `json:"index"`
				Script  string `json:"script"`
				Address string `json:"address"`
				Value   string `json:"value"`
				Block   int    `json:"block"`
				TxHex   string `json:"tx_hex"`
			} `json:"outputs"`
			Error string `json:"error_message"`
		} `json:"data"`
	}

	_DogeTransaction struct {
		Hash          string `json:"hash"`
		Confirmations int    `json:"confirmations"`
		Size          int    `json:"size"`
		Version       int    `json:"version"`
		Locktime      int    `json:"locktime"`
		BlockHash     string `json:"block_hash"`
		Time          int    `json:"time"`
		InputsN       int    `json:"inputs_n"`
		Inputs        []struct {
			Pos       int    `json:"pos"`
			Value     string `json:"value"`
			Type      string `json:"type"`
			Address   string `json:"address"`
			ScriptSig struct {
				Hex string `json:"hex"`
			} `json:"scriptSig"`
			PreviousOutput struct {
				Hash string `json:"hash"`
				Pos  int    `json:"pos"`
			} `json:"previous_output"`
		} `json:"inputs"`
		InputsValue string `json:"inputs_value"`
		OutputsN    int    `json:"outputs_n"`
		Outputs     []struct {
			Pos     int    `json:"pos"`
			Value   string `json:"value"`
			Type    string `json:"type"`
			Address string `json:"address"`
		} `json:"outputs"`
		OutputsValue string `json:"outputs_value"`
		Fee          string `json:"fee"`
	}
	_DogeTransactionRes struct {
		Status string `json:"status"`
		Data   struct {
			Error         string      `json:"error_message"`
			Hash          string      `json:"hash"`
			Confirmations uint64      `json:"confirmations"`
			Size          int         `json:"size"`
			Vsize         int         `json:"vsize"`
			Weight        interface{} `json:"weight"`
			Version       int         `json:"version"`
			Time          uint64      `json:"time"`
			BlockHash     string      `json:"block_hash"`
			BlockHeight   int         `json:"block_height"`
			Fee           string      `json:"fee"`
			Inputs        []struct {
				Index     int    `json:"index"`
				Value     string `json:"value"`
				Address   string `json:"address"`
				ScriptSig struct {
					Hex string `json:"hex"`
					Asm string `json:"asm"`
				} `json:"scriptSig"`
				Witness        interface{} `json:"witness"`
				PreviousOutput struct {
					Hash  string `json:"hash"`
					Index int    `json:"index"`
				} `json:"previous_output"`
			} `json:"inputs"`
			Outputs []struct {
				Index   int    `json:"index"`
				Value   string `json:"value"`
				Type    string `json:"type"`
				Address string `json:"address"`
				Script  struct {
					Hex string `json:"hex"`
					Asm string `json:"asm"`
				} `json:"script"`
				Spent interface{} `json:"spent"`
			} `json:"outputs"`
		} `json:"data"`
	}
)

// syscoin
type (
	_BaseResponseSysCoin struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	}
	_GetBlockCountResSysCoin struct {
		_BaseResponseSysCoin
		Blockbook struct {
			Coin       string `json:"coin"`
			Host       string `json:"host"`
			Version    string `json:"version"`
			BestHeight uint64 `json:"bestHeight"`
		} `json:"blockbook"`
	}

	_GetBalanceResSysCoin struct {
		_BaseResponseSysCoin
		Address            string `json:"address"`
		Balance            string `json:"balance"`
		TotalReceived      string `json:"totalReceived"`
		TotalSent          string `json:"totalSent"`
		UnconfirmedBalance string `json:"unconfirmedBalance"`
	}

	_GetUtxoResSysCoin struct {
		_BaseResponseSysCoin
		Utxos []struct {
			Txid          string `json:"txid"`
			Vout          uint64 `json:"vout"`
			Value         string `json:"value"`
			Height        int    `json:"height"`
			Confirmations int    `json:"confirmations"`
		} `json:"utxos"`
	}

	_SendRawTransactionResSysCoin struct {
		_BaseResponseSysCoin
		Result string `json:"result"`
	}

	_GetTransactionRes struct {
		_BaseResponseSysCoin
		Txid    string `json:"txid"`
		Version int    `json:"version"`
		Vin     []struct {
			Txid      string   `json:"txid"`
			Vout      int      `json:"vout"`
			Sequence  int64    `json:"sequence"`
			N         int      `json:"n"`
			Addresses []string `json:"addresses"`
			IsAddress bool     `json:"isAddress"`
			Value     string   `json:"value"`
		} `json:"vin"`
		Vout []struct {
			Value     string   `json:"value"`
			N         int      `json:"n"`
			Hex       string   `json:"hex"`
			Addresses []string `json:"addresses"`
			IsAddress bool     `json:"isAddress"`
		} `json:"vout"`
		BlockHeight   int    `json:"blockHeight"`
		Confirmations int    `json:"confirmations"`
		BlockTime     int    `json:"blockTime"`
		Value         string `json:"value"`
		ValueIn       string `json:"valueIn"`
		Fees          string `json:"fees"`
		Hex           string `json:"hex"`
		Rbf           bool   `json:"rbf"`
	}
)

func getConfirmation(network string) uint64 {
	switch network {
	case walletutxo.NetworkEnumForBTC:
		return 0
	case walletutxo.NetworkEnumForDOGE:
		return 1
	case walletutxo.NetworkEnumForSYS:
		return 2
	}
	return 1
}

func getNetwork(chainName string) string {
	switch chainName {
	case "BTC":
		return walletutxo.NetworkEnumForBTC
	case "LTC":
		return walletutxo.NetworkEnumForLTC
	case "DOGE":
		return walletutxo.NetworkEnumForDOGE
	}
	return ""
}
