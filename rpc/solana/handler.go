package solana

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	chainrpc "github.com/KernelFlowLabs/wallet-sdk/rpc"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	walletsolana "github.com/KernelFlowLabs/wallet-sdk/signing/solana"

	"github.com/blocto/solana-go-sdk/common"
)

var _ chainrpc.BasicChainHandler = (*Handler)(nil)

type Handler struct {
	rpc           *chainrpc.Request
	heliusApi     *chainrpc.Request
	heliusApiMain *chainrpc.Request
}

func NewHandler(rpcUrl, heliusAPIKey string) (*Handler, error) {
	h := &Handler{}

	url := rpcUrl
	h.rpc = chainrpc.NewRequest(url,
		map[string]string{"Content-Type": "application/json"})
	h.heliusApi = chainrpc.NewRequest("https://api.helius.xyz/v0/token-metadata?api-key="+heliusAPIKey, nil)
	h.heliusApiMain = chainrpc.NewRequest("https://mainnet.helius-rpc.com/?api-key="+heliusAPIKey, nil)
	return h, nil
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      0,
		Method:  "getSlot",
		Params:  nil,
	}
	res := &_GetSlotRes{}
	err := h.rpc.Post(ctx, res, "", req)
	if err != nil {
		return "", fmt.Errorf("failed to getSlot, err=%v", err)
	} else if res.Error.Code != 0 {
		return "", fmt.Errorf("failed to getSlot, errMsg=%s", res.Error.Message)
	}

	return strconv.FormatUint(res.Result, 10), nil
}

func (h *Handler) GetBlockHeight(ctx context.Context) (string, error) {
	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      0,
		Method:  "getBlockHeight",
		Params:  nil,
	}
	res := &_GetBlockHeightRes{}
	err := h.rpc.Post(ctx, res, "", req)
	if err != nil {
		return "", fmt.Errorf("failed to getBlockHeight, err=%v", err)
	} else if res.Error.Code != 0 {
		return "", fmt.Errorf("failed to getBlockHeight, errMsg=%s", res.Error.Message)
	}

	return strconv.FormatUint(res.Result, 10), nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	if contractAddress == signing.MagicContactAddressForNative ||
		contractAddress == signing.MagicContactAddressForNativeSOL {
		value, err := h.getBaseCoinBalance(ctx, address, blockNumber)
		if err != nil {
			return "", err
		}
		return value.String(), nil
	} else {
		if !walletsolana.ValidAddress(address) {
			return "", fmt.Errorf("invalid address")
		} else if !walletsolana.ValidAddress(contractAddress) {
			return "", fmt.Errorf("invalid contractAddress")
		}
		value, err := h.getTokenBalance(ctx, address, contractAddress)
		if err != nil {
			return "", err
		}
		return value.String(), nil
	}
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	tx, err := hex.DecodeString(strings.TrimPrefix(signedHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString, err=%v", err)
	}
	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      0,
		Method:  "sendTransaction",
		Params: []interface{}{
			base64.StdEncoding.EncodeToString(tx),
			_SendTransactionConfig{
				SkipPreflight:       false,
				MaxRetries:          5,
				PreflightCommitment: "finalized",
				Encoding:            "base64",
			},
		},
	}
	res := &_SendTransactionRes{}
	err = h.rpc.Post(ctx, res, "", req)
	if err != nil {
		return "", fmt.Errorf("failed to sendTransaction, err=%v", err)
	} else if res.Error.Code != 0 {
		return "", fmt.Errorf("failed to sendTransaction, errMsg=%s", res.Error.Message)
	} else if res.Result == "" {
		return "", fmt.Errorf("failed to sendTransaction, result is empty")
	}

	return res.Result, nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	r := &chainrpc.TxResult{}

	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      0,
		Method:  "getSignatureStatuses",
		Params: []interface{}{[]string{
			hash,
		},
			_SearchTransactionHistory{
				SearchTransactionHistory: true,
			},
		},
	}
	var err error
	res := &_GetSignatureStatusRes{}
	err = h.rpc.Post(ctx, res, "", req)
	if err != nil {
		return nil, fmt.Errorf("failed to getSignatureStatuses, err=%v", err)
	} else if res.Error.Code != 0 {
		return nil, fmt.Errorf("failed to getSignatureStatuses, err=%s", res.Error.Message)
	} else if len(res.Result.Value) == 0 || res.Result.Value[0] == nil {
		r.Status = signing.TxStatusPending
		return r, nil
	}

	rv := res.Result.Value[0]
	r.Height = strconv.FormatUint(rv.Slot, 10)
	r.Time = strconv.FormatInt(time.Now().Unix(), 10)
	if rv.Err != nil {
		r.Status = signing.TxStatusFailed
		if e, ok := rv.Err.(map[string]interface{}); ok {
			for k, _ := range e {
				r.ErrMsg = k
			}
		}
	} else if rv.ConfirmationStatus == "processed" ||
		rv.ConfirmationStatus == "confirmed" {
		r.Status = signing.TxStatusPending
	} else if rv.ConfirmationStatus == "finalized" {
		r.Status = signing.TxStatusSucceeded
	} else {
		r.Status = signing.TxStatusUnknown
	}
	return r, nil
}

func (h *Handler) GetTxByHashWithLogs(ctx context.Context, hash string) (*GetTransactionWithLog, error) {
	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      1,
		Method:  "getTransaction",
		Params: []interface{}{
			hash,
			_SearchTransaction{
				Commitment:                     "finalized",
				MaxSupportedTransactionVersion: 0,
			},
		},
	}
	res := &GetTransactionWithLog{}
	err := h.rpc.Post(ctx, res, "", req)
	if err != nil {
		return nil, fmt.Errorf("failed to _GetTransaction, err=%v", err)
	} else if res.Error.Code != 0 {
		return nil, fmt.Errorf("failed to _GetTransaction, err=%s", res.Error.Message)
	}
	return res, nil
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("CallContract not supported")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "getAccountInfo":
		req := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      0,
			Method:  "getAccountInfo",
			Params: []interface{}{
				params,
				struct {
					Encoding string `json:"encoding"`
				}{
					"jsonParsed",
				},
			},
		}
		res := &_GetAccountInfoJsonRes{}
		err := h.rpc.Post(ctx, res, "", req)
		if err != nil {
			return "", fmt.Errorf("failed to getAccountInfo, err=%v", err)
		} else if res.Error.Code != 0 {
			return "", fmt.Errorf("failed to getAccountInfo, errMsg=%s", res.Error.Message)
		}
		valueBytes, err := json.Marshal(res.Result.Value)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal for result, err=%v", err)
		}
		return string(valueBytes), nil
	case "getLatestBlockHash":
		req := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      0,
			Method:  "getLatestBlockhash",
			Params:  nil,
		}
		res := &_GetLatestBlockhash{}
		var lastErr error
		for i := 0; i < 3; i++ {
			err := h.rpc.Post(ctx, res, "", req)
			if err == nil && res.Error.Code == 0 && res.Result.Value.Blockhash != "" {
				return res.Result.Value.Blockhash, nil
			}

			if err != nil {
				lastErr = fmt.Errorf("attempt %d failed to getFees, err=%v", i+1, err)
			} else if res.Error.Code != 0 {
				lastErr = fmt.Errorf("attempt %d failed to getFees, errMsg=%s", i+1, res.Error.Message)
			}
			if i < 2 {
				time.Sleep(time.Second)
			}
		}
		return "", lastErr
	case "hasATA":
		tmp := strings.Split(params, "_")
		if len(tmp) != 2 {
			return "", fmt.Errorf("len(tmp) != 2")
		}
		address := tmp[0]
		tokenAddress := tmp[1]
		addressTokenPublic, _, err := common.FindAssociatedTokenAddress(common.PublicKeyFromString(address),
			common.PublicKeyFromString(tokenAddress))
		if err != nil {
			return "", fmt.Errorf("failed to FindAssociatedTokenAddress, err=%v", err)
		}
		addressTokenPublicB58 := addressTokenPublic.ToBase58()
		req := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      0,
			Method:  "getTokenAccountsByOwner",
			Params: []interface{}{
				address,
				struct {
					Mint string `json:"mint"`
				}{
					tokenAddress,
				},
				struct {
					Encoding string `json:"encoding"`
				}{
					"jsonParsed",
				},
			},
		}
		res := &_GetTokenAccountsByOwnerResponse{}
		err = h.rpc.Post(ctx, res, "", req)
		if err != nil {
			return "", fmt.Errorf("failed to getTokenAccountsByOwner, err=%v", err)
		} else if res.Error.Code != 0 {
			return "", fmt.Errorf("failed to getTokenAccountsByOwner, err=%s", res.Error.Message)
		}
		for _, v := range res.Result.Value {
			if v.Pubkey != addressTokenPublicB58 {
				continue
			} else if common.PublicKeyFromString(v.Account.Owner) != common.TokenProgramID {
				continue
			} else if v.Account.Data.Program != "spl-token" {
				continue
			} else if v.Account.Data.Parsed.Type != "account" {
				continue
			} else if v.Account.Data.Parsed.Info.Mint != tokenAddress {
				continue
			} else if v.Account.Data.Parsed.Info.State != "initialized" {
				continue
			}
			return "true", nil
		}
		return "false", nil
	case "getBlockByNumber":
		height, err := strconv.ParseUint(params, 10, 64)
		if err != nil {
			return "", fmt.Errorf("failed to ParseUint for height, err=%v", err)
		}
		block, err := h.getBlockByNumber(ctx, height)
		if err != nil {
			return "", fmt.Errorf("failed to getBlockByNumber, height=%d, err=%v", height, err)
		}
		var blockTx RpcBlock
		blockTx.Transactions = block.Result.Transactions
		blockTxBytes, err := json.Marshal(blockTx)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal, err=%v", err)
		}
		return string(blockTxBytes), nil
	case "isBlockHashValid":
		req := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      0,
			Method:  "isBlockhashValid",
			Params:  []string{params},
		}
		res := &_IsBlockHashValidRes{}
		err := h.rpc.Post(ctx, res, "", req)
		if err != nil {
			return "", fmt.Errorf("failed to getFees, err=%v", err)
		} else if res.Error.Code != 0 {
			return "", fmt.Errorf("failed to getFees, errMsg=%s", res.Error.Message)
		}
		return strconv.FormatBool(res.Result.Value), nil
	case "getTokenInfo":
		//first try
		tokenInfo := &chainrpc.TokenInfo{}
		{
			var out []HeliusTokenMetaResItem
			req := &HeliusTokenMetaReq{
				MintAccounts: []string{
					params,
				},
			}
			err := h.heliusApi.Post(ctx, &out, "", req)
			if err != nil {
				goto SECOND_TRY
			} else if len(out) != 1 {
				goto SECOND_TRY
			}

			if out[0].LegacyMetadata.Address != "" &&
				out[0].LegacyMetadata.Name != "" &&
				out[0].LegacyMetadata.Symbol != "" {
				tokenInfo.Name = out[0].LegacyMetadata.Name
				tokenInfo.Symbol = out[0].LegacyMetadata.Symbol
				tokenInfo.Decimals = strconv.FormatInt(int64(out[0].LegacyMetadata.Decimals), 10)
			} else if out[0].OnChainMetadata.Metadata.Data.Name != "" &&
				out[0].OnChainMetadata.Metadata.Data.Symbol != "" &&
				out[0].OnChainAccountInfo.AccountInfo.Data.Parsed.Info.Decimals != 0 {
				tokenInfo.Name = out[0].OnChainMetadata.Metadata.Data.Name
				tokenInfo.Symbol = out[0].OnChainMetadata.Metadata.Data.Symbol
				tokenInfo.Decimals = strconv.FormatInt(int64(out[0].OnChainAccountInfo.AccountInfo.Data.Parsed.Info.Decimals), 10)
			} else {
				goto SECOND_TRY
			}
			if tokenInfo.Name == "" || tokenInfo.Symbol == "" ||
				tokenInfo.Decimals == "" || tokenInfo.Decimals == "0" {
				goto SECOND_TRY
			}
			tokenInfoBytes, _ := json.Marshal(tokenInfo)
			return string(tokenInfoBytes), nil
		}
	SECOND_TRY:
		{
			var out HeliusTokenMetaMainRes
			req := &HeliusTokenMetaMainReq{}
			req.Jsonrpc = "2.0"
			req.Id = "my-id"
			req.Params.Id = params
			req.Method = "getAsset"
			req.Params.DisplayOptions.ShowFungible = true

			err := h.heliusApiMain.Post(ctx, &out, "", req)
			if err != nil {
				return "", fmt.Errorf("failed to heliusApiMain, err=%v", err)
			}
			tokenInfo.Name = out.Result.Content.Metadata.Name
			tokenInfo.Symbol = out.Result.Content.Metadata.Symbol
			tokenInfo.Decimals = strconv.FormatInt(out.Result.TokenInfo.Decimals, 10)
			if tokenInfo.Name == "" || tokenInfo.Symbol == "" ||
				tokenInfo.Decimals == "" || tokenInfo.Decimals == "0" {
				return "", fmt.Errorf("failed to get meta info 2, err=%v", err)
			}
			tokenInfoBytes, _ := json.Marshal(tokenInfo)
			return string(tokenInfoBytes), nil
		}
	case "getMinimumBalanceForRentExemption":
		space, err := strconv.ParseUint(params, 10, 64)
		if err != nil {
			return "", fmt.Errorf("failed to ParseUint for space, err=%v", err)
		}
		req := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      0,
			Method:  "getMinimumBalanceForRentExemption",
			Params:  []uint64{space},
		}
		res := &_GetminimumbalanceforrentexemptionRes{}
		err = h.rpc.Post(ctx, res, "", req)
		if err != nil {
			return "", fmt.Errorf("failed to getMinimumBalanceForRentExemption, err=%v", err)
		} else if res.Error.Code != 0 {
			return "", fmt.Errorf("failed to getMinimumBalanceForRentExemption, errMsg=%s", res.Error.Message)
		}
		return strconv.FormatUint(res.Result, 10), nil
	case "getNonceAccountBlockHash":
		req := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      0,
			Method:  "getAccountInfo",
			Params: []interface{}{
				params,
				struct {
					Encoding string `json:"encoding"`
				}{
					"jsonParsed",
				},
			},
		}
		res := &_GetAccountInfoJsonRes{}
		err := h.rpc.Post(ctx, res, "", req)
		if err != nil {
			return "", fmt.Errorf("failed to getAccountInfo, err=%v", err)
		} else if res.Error.Code != 0 {
			return "", fmt.Errorf("failed to getAccountInfo, errMsg=%s", res.Error.Message)
		} else if res.Result.Value.Data.Parsed.Info.Blockhash == "" {
			return "", fmt.Errorf("failed to getAccountInfo, Blockhash is empty")
		}
		return res.Result.Value.Data.Parsed.Info.Blockhash, nil
	case "getPriorityFee":
		req := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      0,
			Method:  "getRecentPrioritizationFees",
			Params:  []interface{}{},
		}
		res := &_GetPrioritizationFee{}
		err := h.rpc.Post(ctx, res, "", req)
		if err != nil {
			return "", fmt.Errorf("failed to getRecentPrioritizationFees, err=%v", err)
		} else if res.Error.Code != 0 {
			return "", fmt.Errorf("failed to getRecentPrioritizationFees, errMsg=%s", res.Error.Message)
		}
		var maxFee uint64
		for _, it := range res.Result {
			if it.PrioritizationFee > maxFee {
				maxFee = it.PrioritizationFee
			}
		}
		return strconv.FormatUint(maxFee, 10), nil
	case "getTokenDecimals":
		if params == signing.MagicContactAddressForNative {
			return "9", nil
		}
		{
			var out []HeliusTokenMetaResItem
			req := &HeliusTokenMetaReq{
				MintAccounts: []string{
					params,
				},
			}
			err := h.heliusApi.Post(ctx, &out, "", req)
			if err != nil {
				goto DECIMALS_SECOND_TRY
			} else if len(out) != 1 {
				goto DECIMALS_SECOND_TRY
			}

			var decimals string
			if out[0].LegacyMetadata.Address != "" {
				decimals = strconv.FormatInt(int64(out[0].LegacyMetadata.Decimals), 10)
			} else if out[0].OnChainAccountInfo.AccountInfo.Data.Parsed.Info.Decimals != 0 {
				decimals = strconv.FormatInt(int64(out[0].OnChainAccountInfo.AccountInfo.Data.Parsed.Info.Decimals), 10)
			} else {
				goto DECIMALS_SECOND_TRY
			}
			if decimals == "" || decimals == "0" {
				goto DECIMALS_SECOND_TRY
			}
			return decimals, nil
		}
	DECIMALS_SECOND_TRY:
		{
			var out HeliusTokenMetaMainRes
			req := &HeliusTokenMetaMainReq{}
			req.Jsonrpc = "2.0"
			req.Id = "my-id"
			req.Params.Id = params
			req.Method = "getAsset"
			req.Params.DisplayOptions.ShowFungible = true

			err := h.heliusApiMain.Post(ctx, &out, "", req)
			if err != nil {
				return "", fmt.Errorf("failed to heliusApiMain, err=%v", err)
			}
			decimals := strconv.FormatInt(out.Result.TokenInfo.Decimals, 10)
			if decimals == "" || decimals == "0" {
				return "", fmt.Errorf("failed to get decimals 2, err=%v", err)
			}
			return decimals, nil
		}
	}
	return "", fmt.Errorf("unsupported funciton")
}

// unexported
func (h *Handler) getBaseCoinBalance(ctx context.Context, address, rpc string) (*big.Int, error) {
	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      0,
		Method:  "getBalance",
		Params:  []string{address},
	}
	res := &_GetBalanceRes{}
	var err error
	if strings.EqualFold(rpc, "rpc1") {
		err = h.rpc.Post(ctx, res, "", req)
	} else {
		rpc = "rpc"
		err = h.rpc.Post(ctx, res, "", req)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to getBalance from %s, err=%v", rpc, err)
	} else if res.Error.Code != 0 {
		return nil, fmt.Errorf("failed to getBalance from %s, errMsg=%s", rpc, res.Error.Message)
	}

	return res.Result.Value, nil
}
func (h *Handler) getTokenBalance(ctx context.Context, address, contractAddress string) (*big.Int, error) {
	ownerTokenPublic, _, err := common.FindAssociatedTokenAddress(
		common.PublicKeyFromString(address),
		common.PublicKeyFromString(contractAddress))
	if err != nil {
		return nil, fmt.Errorf("failed to FindAssociatedTokenAddress for"+
			"address=%s and contract=%s, err=%v", address, contractAddress, err)
	}
	ownerTokenAddr := ownerTokenPublic.ToBase58()
	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      0,
		Method:  "getTokenAccountBalance",
		Params:  []string{ownerTokenAddr},
	}
	res := &_GetTokenBalanceRes{}
	err = h.rpc.Post(ctx, res, "", req)
	if err != nil {
		return nil, fmt.Errorf("failed to getTokenAccountBalance, err=%v", err)
	} else if res.Error.Code != 0 {
		if strings.Contains(res.Error.Message, "could not find account") {
			return big.NewInt(0), nil
		}
		return nil, fmt.Errorf("failed to getTokenAccountBalance from, err=%s", res.Error.Message)
	}
	amount, ok := new(big.Int).SetString(res.Result.Value.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("SetString failed")
	}

	return amount, nil
}
func (h *Handler) getBlockByNumber(ctx context.Context, num uint64) (*_GetBlockByNumberRes, error) {
	req := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      1,
		Method:  "getBlock",
		Params: []interface{}{
			num,
			map[string]interface{}{
				"encoding":                       "json",
				"maxSupportedTransactionVersion": 0,
				"transactionDetails":             "full",
				"rewards":                        false,
			},
		},
	}
	res := &_GetBlockByNumberRes{}
	err := h.rpc.Post(ctx, res, "", req)
	if err != nil {
		return nil, fmt.Errorf("failed to getBlockByNumber, err=%v", err)
	} else if res.Error.Code != 0 {
		return nil, fmt.Errorf("failed to getBlockByNumber, err=%s", res.Error.Message)
	}
	return res, nil
}

// types
type (
	_BaseRequest struct {
		JsonRPC string      `json:"jsonrpc"`
		ID      uint64      `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}
	_BaseResponse struct {
		JsonRPC string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
	}
	_Context struct {
		ApiVersion string `json:"apiVersion,omitempty"`
		Slot       uint64 `json:"slot" json:"slot,omitempty"`
	}
	_GetSlotRes struct {
		_BaseResponse
		Result uint64 `json:"result"`
	}

	_GetBlockHeightRes struct {
		_BaseResponse
		Result uint64 `json:"result"`
	}

	_GetBalanceRes struct {
		_BaseResponse
		Result struct {
			Context *_Context `json:"context"`
			Value   *big.Int  `json:"value"`
		} `json:"result"`
	}

	AccountInfo struct {
		Data struct {
			Parsed struct {
				Info struct {
					Decimals        int    `json:"decimals"`
					FreezeAuthority string `json:"freezeAuthority"`
					IsInitialized   bool   `json:"isInitialized"`
					MintAuthority   string `json:"mintAuthority"`
					Supply          string `json:"supply"`

					Authority     string `json:"authority"`
					Blockhash     string `json:"blockhash"`
					FeeCalculator struct {
						LamportsPerSignature string `json:"lamportsPerSignature"`
					} `json:"feeCalculator"`
				} `json:"info"`
				Type string `json:"type"`
			} `json:"parsed"`
			Program string `json:"program"`
			Space   int    `json:"space"`
		} `json:"data,omitempty"`
		//Data1      []string `json:"data,omitempty"`
		Executable bool   `json:"executable" json:"executable,omitempty"`
		Lamports   uint64 `json:"lamports" json:"lamports,omitempty"`
	}

	_GetAccountInfoJsonRes struct {
		_BaseResponse
		Result struct {
			Context *_Context   `json:"context"`
			Value   AccountInfo `json:"value"`
		} `json:"result"`
	}

	_GetTokenBalanceRes struct {
		_BaseResponse
		Result struct {
			Context *_Context `json:"context"`
			Value   struct {
				Amount   string  `json:"amount"`
				Decimals int     `json:"decimals"`
				UiAmount float64 `json:"uiAmount"`
			} `json:"value"`
		} `json:"result"`
	}

	//Fees struct {
	//	Blockhash     string `json:"blockhash"`
	//	FeeCalculator struct {
	//		LamportsPerSignature uint64 `json:"lamportsPerSignature"`
	//	} `json:"feeCalculator"`
	//	LastValidSlot        uint64 `json:"lastValidSlot"`
	//	LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
	//}
	_GetLatestBlockhash struct {
		_BaseResponse
		Result struct {
			Context struct {
				Slot int `json:"slot"`
			} `json:"context"`
			Value struct {
				Blockhash            string `json:"blockhash"`
				LastValidBlockHeight int    `json:"lastValidBlockHeight"`
			} `json:"value"`
		} `json:"result"`
	}
	_IsBlockHashValidRes struct {
		_BaseResponse
		Result struct {
			Context *_Context `json:"context"`
			Value   bool      `json:"value"`
		} `json:"result"`
	}

	_GetminimumbalanceforrentexemptionRes struct {
		_BaseResponse
		Result uint64 `json:"result"`
	}

	_SendTransactionRes struct {
		_BaseResponse
		Result string `json:"result"`
	}

	_GetSignatureStatus struct {
		Slot               uint64      `json:"slot"`
		Confirmations      uint64      `json:"confirmations"`
		Err                interface{} `json:"err"`
		ConfirmationStatus string      `json:"confirmationStatus"` //finalized, confirmed, processed
	}
	_GetSignatureStatusRes struct {
		_BaseResponse
		Result struct {
			Context *_Context              `json:"context"`
			Value   []*_GetSignatureStatus `json:"value"`
		} `json:"result"`
	}
	_GetTokenAccountsByOwnerResponse struct {
		_BaseResponse
		Result struct {
			Context *_Context                   `json:"context"`
			Value   []*_GetTokenAccountsByOwner `json:"value"`
		} `json:"result"`
	}
	_GetTokenAccountsByOwner struct {
		Pubkey  string `json:"pubkey"`
		Account struct {
			Data struct {
				Parsed struct {
					Info struct {
						IsNative    bool   `json:"isNative"`
						Mint        string `json:"mint"`
						Owner       string `json:"owner"`
						State       string `json:"state"`
						TokenAmount struct {
							Amount   string  `json:"amount"`
							Decimals int     `json:"decimals"`
							UiAmount float64 `json:"uiAmount"`
						} `json:"tokenAmount"`
					} `json:"info"`
					Type string `json:"type"`
				} `json:"parsed"`
				Program string `json:"program"`
				Space   uint64 `json:"space"`
			} `json:"data"`
			Executable bool   `json:"executable"`
			Lamports   uint64 `json:"lamports"`
			Owner      string `json:"owner"`
			RentEpoch  uint64 `json:"rentEpoch"`
		} `json:"account"`
	}
	GetTransaction struct {
		_BaseResponse
		Result struct {
			BlockTime int `json:"blockTime"`
			Meta      struct {
				ComputeUnitsConsumed int           `json:"computeUnitsConsumed"`
				Err                  interface{}   `json:"err"`
				Fee                  int           `json:"fee"`
				InnerInstructions    []interface{} `json:"innerInstructions"`
				LogMessages          []string      `json:"logMessages"`
				PostBalances         []int         `json:"postBalances"`
				PostTokenBalances    []struct {
					AccountIndex  int    `json:"accountIndex"`
					Mint          string `json:"mint"`
					Owner         string `json:"owner"`
					ProgramID     string `json:"programId"`
					UITokenAmount struct {
						Amount         string  `json:"amount"`
						Decimals       int     `json:"decimals"`
						UIAmount       float64 `json:"uiAmount"`
						UIAmountString string  `json:"uiAmountString"`
					} `json:"uiTokenAmount"`
				} `json:"postTokenBalances"`
				PreBalances      []int `json:"preBalances"`
				PreTokenBalances []struct {
					AccountIndex  int    `json:"accountIndex"`
					Mint          string `json:"mint"`
					Owner         string `json:"owner"`
					ProgramID     string `json:"programId"`
					UITokenAmount struct {
						Amount         string  `json:"amount"`
						Decimals       int     `json:"decimals"`
						UIAmount       float64 `json:"uiAmount"`
						UIAmountString string  `json:"uiAmountString"`
					} `json:"uiTokenAmount"`
				} `json:"preTokenBalances"`
				Rewards []interface{} `json:"rewards"`
				Status  struct {
					Ok interface{} `json:"Ok"`
				} `json:"status"`
			} `json:"meta"`
			Slot        int `json:"slot"`
			Transaction struct {
				Message struct {
					AccountKeys []struct {
						Pubkey   string `json:"pubkey"`
						Signer   bool   `json:"signer"`
						Source   string `json:"source"`
						Writable bool   `json:"writable"`
					} `json:"accountKeys"`
					//AccountKeys  []string `json:"accountKeys"`
					Instructions []struct {
						Accounts    []interface{}   `json:"accounts,omitempty"`
						Data        string          `json:"data,omitempty"`
						ProgramID   string          `json:"programId"`
						StackHeight interface{}     `json:"stackHeight"`
						Parsed      json.RawMessage `json:"parsed"`
						//Parsed      struct {
						//	Info struct {
						//		Amount      string `json:"amount"`
						//		Authority   string `json:"authority"`
						//		Destination string `json:"destination"`
						//		Source      string `json:"source"`
						//	} `json:"info"`
						//	Type string `json:"type"`
						//} `json:"parsed,omitempty"`
						Program string `json:"program,omitempty"`
					} `json:"instructions"`
					RecentBlockhash string `json:"recentBlockhash"`
				} `json:"message"`
				Signatures []string `json:"signatures"`
			} `json:"transaction"`
		} `json:"result"`
		ID int `json:"id"`
	}
	GetTransactionWithLog struct {
		_BaseResponse
		Result struct {
			BlockTime int `json:"blockTime"`
			Meta      struct {
				ComputeUnitsConsumed int           `json:"computeUnitsConsumed"`
				Err                  interface{}   `json:"err"`
				Fee                  int           `json:"fee"`
				InnerInstructions    []interface{} `json:"innerInstructions"`
				LogMessages          []string      `json:"logMessages"`
				PostBalances         []int         `json:"postBalances"`
				PostTokenBalances    []struct {
					AccountIndex  int    `json:"accountIndex"`
					Mint          string `json:"mint"`
					Owner         string `json:"owner"`
					ProgramID     string `json:"programId"`
					UITokenAmount struct {
						Amount         string  `json:"amount"`
						Decimals       int     `json:"decimals"`
						UIAmount       float64 `json:"uiAmount"`
						UIAmountString string  `json:"uiAmountString"`
					} `json:"uiTokenAmount"`
				} `json:"postTokenBalances"`
				PreBalances      []int `json:"preBalances"`
				PreTokenBalances []struct {
					AccountIndex  int    `json:"accountIndex"`
					Mint          string `json:"mint"`
					Owner         string `json:"owner"`
					ProgramID     string `json:"programId"`
					UITokenAmount struct {
						Amount         string  `json:"amount"`
						Decimals       int     `json:"decimals"`
						UIAmount       float64 `json:"uiAmount"`
						UIAmountString string  `json:"uiAmountString"`
					} `json:"uiTokenAmount"`
				} `json:"preTokenBalances"`
				Rewards []interface{} `json:"rewards"`
				Status  struct {
					Ok interface{} `json:"Ok"`
				} `json:"status"`
			} `json:"meta"`
			Slot        int `json:"slot"`
			Transaction struct {
				Message struct {
					// AccountKeys []struct {
					// 	Pubkey   string `json:"pubkey"`
					// 	Signer   bool   `json:"signer"`
					// 	Source   string `json:"source"`
					// 	Writable bool   `json:"writable"`
					// } `json:"accountKeys"`
					AccountKeys  []string `json:"accountKeys"`
					Instructions []struct {
						Accounts    []interface{}   `json:"accounts,omitempty"`
						Data        string          `json:"data,omitempty"`
						ProgramID   string          `json:"programId"`
						StackHeight interface{}     `json:"stackHeight"`
						Parsed      json.RawMessage `json:"parsed"`
						//Parsed      struct {
						//	Info struct {
						//		Amount      string `json:"amount"`
						//		Authority   string `json:"authority"`
						//		Destination string `json:"destination"`
						//		Source      string `json:"source"`
						//	} `json:"info"`
						//	Type string `json:"type"`
						//} `json:"parsed,omitempty"`
						Program string `json:"program,omitempty"`
					} `json:"instructions"`
					RecentBlockhash string `json:"recentBlockhash"`
				} `json:"message"`
				Signatures []string `json:"signatures"`
			} `json:"transaction"`
		} `json:"result"`
		ID int `json:"id"`
	}
	_GetPrioritizationFee struct {
		_BaseResponse
		Result []PrioritizationFeeItem `json:"result"`
	}
	PrioritizationFeeItem struct {
		Slot              uint64 `json:"slot"`
		PrioritizationFee uint64 `json:"prioritizationFee"`
	}

	TransferInfo struct {
		Info struct {
			Amount            string `json:"amount"`
			Lamports          int    `json:"lamports"`
			Authority         string `json:"authority"`
			MultisigAuthority string `json:"multisigAuthority"`
			Destination       string `json:"destination"`
			Source            string `json:"source"`
		} `json:"info"`
		Type string `json:"type"`
	}

	_GetBlockByNumberRes struct {
		_BaseResponse
		Result struct {
			BlockHeight       int64   `json:"blockHeight"`
			BlockTime         int64   `json:"blockTime"`
			Blockhash         string  `json:"blockhash"`
			ParentSlot        int64   `json:"parentSlot"`
			PreviousBlockhash string  `json:"previousBlockhash"`
			Transactions      []RpcTx `json:"transactions"`
		} `json:"result"`
		ID int `json:"id"`
	}

	SPLTokenTransfer struct {
		AccountIndex  int    `json:"accountIndex"`
		Mint          string `json:"mint"`
		Owner         string `json:"owner"`
		UiTokenAmount struct {
			Amount         string  `json:"amount"`
			Decimals       int     `json:"decimals"`
			UiAmount       float64 `json:"uiAmount"`
			UiAmountString string  `json:"uiAmountString"`
		} `json:"uiTokenAmount"`
	}
	RpcTx struct {
		Meta struct {
			ComputeUnitsConsumed int           `json:"computeUnitsConsumed"`
			Err                  interface{}   `json:"err"`
			Fee                  int           `json:"fee"`
			InnerInstructions    []interface{} `json:"innerInstructions"`
			LoadedAddresses      struct {
				Readonly []interface{} `json:"readonly"`
				Writable []interface{} `json:"writable"`
			} `json:"loadedAddresses"`
			LogMessages       []string            `json:"logMessages"`
			PreBalances       []uint64            `json:"preBalances"`
			PostBalances      []uint64            `json:"postBalances"`
			PreTokenBalances  []*SPLTokenTransfer `json:"preTokenBalances"`
			PostTokenBalances []*SPLTokenTransfer `json:"postTokenBalances"`
			Rewards           interface{}         `json:"rewards"`
			Status            struct {
				Ok interface{} `json:"Ok"`
			} `json:"status"`
		} `json:"meta"`
		Transaction struct {
			Message struct {
				AccountKeys []string `json:"accountKeys"`
				//AccountKeys []struct {
				//	Pubkey   string `json:"pubkey"`
				//	Signer   bool   `json:"signer"`
				//	Source   string `json:"source"`
				//	Writable bool   `json:"writable"`
				//} `json:"accountKeys"`
				Header struct {
					NumReadonlySignedAccounts   int `json:"numReadonlySignedAccounts"`
					NumReadonlyUnsignedAccounts int `json:"numReadonlyUnsignedAccounts"`
					NumRequiredSignatures       int `json:"numRequiredSignatures"`
				} `json:"header"`
				Instructions []struct {
					Accounts       []int       `json:"accounts"`
					Data           string      `json:"data"`
					ProgramIDIndex int         `json:"programIdIndex"`
					StackHeight    interface{} `json:"stackHeight"`
				} `json:"instructions"`
				RecentBlockhash string `json:"recentBlockhash"`
			} `json:"message"`
			Signatures []string `json:"signatures"`
		} `json:"transaction"`
		Version interface{} `json:"version"`
	}
	RpcBlock struct {
		BlockTime    int64   `json:"blockTime"`
		Transactions []RpcTx `json:"transactions"`
	}
)

type (
	_SendTransactionConfig struct {
		SkipPreflight       bool   `json:"skipPreflight"` // default: false
		MaxRetries          int64  `json:"maxRetries"`
		PreflightCommitment string `json:"preflightCommitment"` // default: max
		Encoding            string `json:"encoding"`            // base58 or base64
	}
	_SearchTransactionHistory struct {
		SearchTransactionHistory bool `json:"searchTransactionHistory"`
	}
	_SearchTransaction struct {
		Commitment                     string `json:"commitment"`
		MaxSupportedTransactionVersion int64  `json:"maxSupportedTransactionVersion"`
	}
)

type (
	HeliusTokenMetaReq struct {
		MintAccounts []string `json:"mintAccounts"`
	}
	HeliusTokenMetaResItem struct {
		Account            string `json:"account"`
		OnChainAccountInfo struct {
			AccountInfo struct {
				Key        string `json:"key"`
				IsSigner   bool   `json:"isSigner"`
				IsWritable bool   `json:"isWritable"`
				Lamports   int64  `json:"lamports"`
				Data       struct {
					Parsed struct {
						Info struct {
							Decimals        int    `json:"decimals"`
							FreezeAuthority string `json:"freezeAuthority"`
							IsInitialized   bool   `json:"isInitialized"`
							MintAuthority   string `json:"mintAuthority"`
							Supply          string `json:"supply"`
						} `json:"info"`
						Type string `json:"type"`
					} `json:"parsed"`
					Program string `json:"program"`
					Space   int    `json:"space"`
				} `json:"data"`
				Owner      string  `json:"owner"`
				Executable bool    `json:"executable"`
				RentEpoch  float64 `json:"rentEpoch"`
			} `json:"accountInfo"`
			Error string `json:"error"`
		} `json:"onChainAccountInfo"`
		OnChainMetadata struct {
			Metadata struct {
				TokenStandard   string `json:"tokenStandard"`
				Key             string `json:"key"`
				UpdateAuthority string `json:"updateAuthority"`
				Mint            string `json:"mint"`
				Data            struct {
					Name                 string      `json:"name"`
					Symbol               string      `json:"symbol"`
					Uri                  string      `json:"uri"`
					SellerFeeBasisPoints int         `json:"sellerFeeBasisPoints"`
					Creators             interface{} `json:"creators"`
				} `json:"data"`
				PrimarySaleHappened bool `json:"primarySaleHappened"`
				IsMutable           bool `json:"isMutable"`
				EditionNonce        int  `json:"editionNonce"`
				Uses                struct {
					UseMethod string `json:"useMethod"`
					Remaining int    `json:"remaining"`
					Total     int    `json:"total"`
				} `json:"uses"`
				Collection        interface{} `json:"collection"`
				CollectionDetails interface{} `json:"collectionDetails"`
			} `json:"metadata"`
			Error string `json:"error"`
		} `json:"onChainMetadata"`
		LegacyMetadata struct {
			ChainId    int      `json:"chainId"`
			Address    string   `json:"address"`
			Symbol     string   `json:"symbol"`
			Name       string   `json:"name"`
			Decimals   int      `json:"decimals"`
			LogoURI    string   `json:"logoURI"`
			Tags       []string `json:"tags"`
			Extensions struct {
				CoingeckoId string `json:"coingeckoId"`
				SerumV3Usdc string `json:"serumV3Usdc"`
				Website     string `json:"website"`
			} `json:"extensions"`
		} `json:"legacyMetadata"`
	}
	HeliusTokenMetaMainReq struct {
		Jsonrpc string `json:"jsonrpc"`
		Id      string `json:"id"`
		Method  string `json:"method"`
		Params  struct {
			Id             string `json:"id"`
			DisplayOptions struct {
				ShowFungible bool `json:"showFungible"`
			} `json:"displayOptions"`
		} `json:"params"`
	}
	HeliusTokenMetaMainRes struct {
		Jsonrpc string `json:"jsonrpc"`
		Result  struct {
			Content struct {
				Metadata struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				} `json:"metadata"`
			} `json:"content"`
			TokenInfo struct {
				Decimals int64 `json:"decimals"`
			} `json:"token_info"`
		} `json:"result"`
	}
)
