package multiversx

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"

	chainrpc "github.com/KernelFlowLabs/wallet-sdk/rpc"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
	walletmvx "github.com/KernelFlowLabs/wallet-sdk/signing/multiversx"
)

var _ chainrpc.BasicChainHandler = (*Handler)(nil)

type Handler struct {
	rpc *chainrpc.Request
}

func NewHandler(rpcUrl string) (*Handler, error) {
	h := &Handler{}
	h.rpc = chainrpc.NewRequest(rpcUrl, map[string]string{
		"Content-Type": "application/json",
	})
	return h, nil
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	res := &_StatusRes{}
	err := h.rpc.Get(ctx, res, "network/status/4294967295", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get latest block, err=%v", err)
	} else if res.Code != "successful" {
		return "", fmt.Errorf("failed to get latest block, errMsg=%s", res.Error)
	}

	return strconv.FormatUint(res.Data.Status.ErdHighestFinalNonce, 10), nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	if contractAddress == signing.MagicContactAddressForNative {
		res := &_AddressRes{}
		path := "address/" + address
		err := h.rpc.Get(ctx, res, path, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get balance, err=%v", err)
		} else if res.Code != "successful" {
			return "", fmt.Errorf("failed to get balance, errMsg=%s", res.Error)
		}
		if bal, ok := big.NewInt(0).SetString(res.Data.Account.Balance, 10); ok {
			return bal.String(), nil
		}
		return "", fmt.Errorf("wrong return type")
	}

	res := &_EsdtBalanceRes{}
	path := "address/" + address + "/esdt/" + contractAddress
	err := h.rpc.Get(context.Background(), res, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get getTokenBalance, err=%v", err)
	} else if res.Code != "successful" {
		if strings.Contains(res.Error, "account was not found") {
			return "0", nil
		}
		return "", fmt.Errorf("failed to get getTokenBalance, err=%s", res.Error)
	}
	if bal, ok := big.NewInt(0).SetString(res.Data.TokenData.Balance, 10); ok {
		return bal.String(), nil
	}
	return "", fmt.Errorf("wrong return type")
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	signedBytes, err := hex.DecodeString(strings.TrimPrefix(signedHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString for signedHex, err=%v", err)
	}
	req := &nativeTx{}
	err = json.Unmarshal(signedBytes, req)
	if err != nil {
		return "", fmt.Errorf("failed to Unmarshal for signedBytes, err=%v", err)
	}

	res := &_TransactionSendRes{}
	path := "transaction/send"
	err = h.rpc.Post(ctx, res, path, req)
	if err != nil {
		return "", fmt.Errorf("failed to sendTransaction, err=%v", err)
	} else if res.Code != "successful" {
		return "", fmt.Errorf("failed to sendTransaction, errMsg=%s", res.Error)
	}

	return res.Data.TxHash, nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	tmp := strings.Split(hash, ":")
	if len(tmp) != 2 {
		return nil, fmt.Errorf("invalid params, pattern should be hash:from")
	}
	hash = tmp[0]
	sender := tmp[1]
	r := &chainrpc.TxResult{}

	tx, err := h.getTransaction(ctx, hash, sender)
	if err != nil {
		return nil, err
	}
	if tx.Status == "pending" {
		r.Status = signing.TxStatusPending
	} else if tx.Status == "invalid" {
		r.Status = signing.TxStatusFailed
	} else if tx.Status == "success" {
		if tx.Operation == "transfer" {
			r.Status = signing.TxStatusSucceeded
		} else if tx.Operation == "ESDTTransfer" {
			ifContractFailed := false

			if len(tx.SmartContractResults) == 0 {
				ifContractFailed = true
			}
			for _, v := range tx.SmartContractResults {
				if v.ReturnMessage != "" {
					ifContractFailed = true
				}
			}
			if ifContractFailed {
				r.Status = signing.TxStatusFailed
			} else {
				r.Status = signing.TxStatusSucceeded
			}
		} else {
			return r, fmt.Errorf("unsupported operation")
		}
	} else {
		r.Status = signing.TxStatusUnknown
	}
	return r, nil
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("CallContract not supported")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "getNonce":
		res := &_AddressRes{}
		path := "address/" + params
		err := h.rpc.Get(ctx, res, path, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get getNonce, err=%v", err)
		} else if res.Code != "successful" {
			return "", fmt.Errorf("failed to get getNonce, errMsg=%s", res.Error)
		}

		return strconv.FormatUint(res.Data.Account.Nonce, 10), nil
	case "getNetworkConfig":
		res := &_NetWorkConfigRes{}
		result := &walletmvx.NetWorkConfig{}
		err := h.rpc.Get(ctx, res, "network/config", nil)
		if err != nil {
			return "", fmt.Errorf("failed to network/config, err=%v", err)
		} else if res.Code != "successful" {
			return "", fmt.Errorf("failed to network/config, errMsg=%s", res.Error)
		}
		result.GasPrice = strconv.FormatUint(res.Data.Config.MinGasPrice, 10)
		result.GasLimit = strconv.FormatUint(res.Data.Config.MinGasLimit, 10)
		result.ChainID = res.Data.Config.ChainID
		result.Version = strconv.FormatUint(uint64(res.Data.Config.MinTransactionVersion), 10)
		resultBytes, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal for result, err=%s", err)
		}
		return string(resultBytes), nil
	case "getTokenDecimals":
		if params == signing.MagicContactAddressForNative {
			return "18", nil
		}
		res := &_TokenInfoRes{}
		path := "tokens/" + params
		err := h.rpc.Get(ctx, res, path, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get token decimals, err=%v", err)
		} else if res.Code != "successful" {
			return "", fmt.Errorf("failed to get token decimals, errMsg=%s", res.Error)
		}
		return strconv.FormatUint(uint64(res.Data.Decimals), 10), nil
	}
	return "", fmt.Errorf("unsupported function")
}

// unexported
func (h *Handler) getTransaction(ctx context.Context, hash string, sender string) (*_EgldTransaction, error) {
	res := &_TransactionRes{}
	path := "transaction/" + hash
	query := url.Values{}
	query.Set("withResults", "true")
	query.Set("sender", sender)
	err := h.rpc.Get(ctx, res, path, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get getTransaction %s, err=%v", hash, err)
	} else if res.Code != "successful" {
		return nil, fmt.Errorf("failed to get getTransaction %s, errMsg=%s", hash, res.Error)
	}

	return res.Data.Transaction, nil
}

// types
type (
	_NetWorkConfigRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Config *_RpcNetworkConfig `json:"config"`
		}
	}
	_RpcNetworkConfig struct {
		ChainID                  string  `json:"erd_chain_id"`
		Denomination             int     `json:"erd_denomination"`
		GasPerDataByte           uint64  `json:"erd_gas_per_data_byte"`
		LatestTagSoftwareVersion string  `json:"erd_latest_tag_software_version"`
		MetaConsensusGroup       uint32  `json:"erd_meta_consensus_group_size"`
		MinGasLimit              uint64  `json:"erd_min_gas_limit"`
		MinGasPrice              uint64  `json:"erd_min_gas_price"`
		MinTransactionVersion    uint32  `json:"erd_min_transaction_version"`
		NumMetachainNodes        uint32  `json:"erd_num_metachain_nodes"`
		NumNodesInShard          uint32  `json:"erd_num_nodes_in_shard"`
		NumShardsWithoutMeta     uint32  `json:"erd_num_shards_without_meta"`
		RoundDuration            int64   `json:"erd_round_duration"`
		ShardConsensusGroupSize  uint64  `json:"erd_shard_consensus_group_size"`
		StartTime                int64   `json:"erd_start_time"`
		Adaptivity               bool    `json:"erd_adaptivity,string"`
		Hysteresys               float32 `json:"erd_hysteresis,string"`
	}
	_StatusRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Status struct {
				ErdCurrentRound               uint64 `json:"erd_current_round"`
				ErdEpochNumber                uint64 `json:"erd_epoch_number"`
				ErdHighestFinalNonce          uint64 `json:"erd_highest_final_nonce"`
				ErdNonce                      uint64 `json:"erd_nonce"`
				ErdNonceAtEpochStart          uint64 `json:"erd_nonce_at_epoch_start"`
				ErdNoncesPassedInCurrentEpoch uint64 `json:"erd_nonces_passed_in_current_epoch"`
				ErdRoundAtEpochStart          uint64 `json:"erd_round_at_epoch_start"`
				ErdRoundsPassedInCurrentEpoch uint64 `json:"erd_rounds_passed_in_current_epoch"`
				ErdRoundsPerEpoch             uint64 `json:"erd_rounds_per_epoch"`
			} `json:"status"`
		} `json:"data"`
	}
	_AddressRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Account struct {
				Address string `json:"address"`
				Nonce   uint64 `json:"nonce"`
				Balance string `json:"balance"`
			} `json:"account"`
		} `json:"data"`
	}
	_TopicsArr     []string
	_EgldEventItem struct {
		Address    string     `json:"address"`
		Identifier string     `json:"identifier"`
		Data       string     `json:"data"`
		Topics     _TopicsArr `json:"topics"`
	}
	_EgldLogsItem struct {
		Address string            `json:"address"`
		Events  []*_EgldEventItem `json:"events"`
	}
	_EgldSmartContractResultsItem struct {
		Hash          string         `json:"hash"`
		Sender        string         `json:"sender"`
		Receiver      string         `json:"receiver"`
		Nonce         uint64         `json:"nonce"`
		Data          string         `json:"data"`
		CallType      int            `json:"callType"`
		ReturnMessage string         `json:"returnMessage"`
		Logs          *_EgldLogsItem `json:"logs"`
	}
	_EgldTransaction struct {
		Type                              string `json:"type"`
		ProcessingTypeOnSource            string `json:"processingTypeOnSource"`
		ProcessingTypeOnDestination       string `json:"processingTypeOnDestination"`
		Hash                              string `json:"hash"`
		Nonce                             int    `json:"nonce"`
		Round                             int    `json:"round"`
		Epoch                             int    `json:"epoch"`
		Value                             string `json:"value"`
		Receiver                          string `json:"receiver"`
		Sender                            string `json:"sender"`
		GasPrice                          int    `json:"gasPrice"`
		GasLimit                          int    `json:"gasLimit"`
		Data                              string `json:"data"`
		Signature                         string `json:"signature"`
		SourceShard                       int    `json:"sourceShard"`
		DestinationShard                  int    `json:"destinationShard"`
		BlockNonce                        int    `json:"blockNonce"`
		BlockHash                         string `json:"blockHash"`
		NotarizedAtSourceInMetaNonce      int    `json:"notarizedAtSourceInMetaNonce"`
		NotarizedAtSourceInMetaHash       string `json:"NotarizedAtSourceInMetaHash"`
		NotarizedAtDestinationInMetaNonce int    `json:"notarizedAtDestinationInMetaNonce"`
		NotarizedAtDestinationInMetaHash  string `json:"notarizedAtDestinationInMetaHash"`
		MiniblockType                     string `json:"miniblockType"`
		MiniblockHash                     string `json:"miniblockHash"`
		Timestamp                         int    `json:"timestamp"`
		SmartContractResults              []struct {
			Hash           string         `json:"hash"`
			Nonce          int            `json:"nonce"`
			Value          int64          `json:"value"`
			Receiver       string         `json:"receiver"`
			Sender         string         `json:"sender"`
			Data           string         `json:"data"`
			PrevTxHash     string         `json:"prevTxHash"`
			OriginalTxHash string         `json:"originalTxHash"`
			GasLimit       int            `json:"gasLimit"`
			GasPrice       int            `json:"gasPrice"`
			CallType       int            `json:"callType"`
			Operation      string         `json:"operation"`
			IsRefund       bool           `json:"isRefund"`
			ReturnMessage  string         `json:"returnMessage"`
			Logs           *_EgldLogsItem `json:"logs"`
		} `json:"smartContractResults"`
		Logs struct {
			Address string `json:"address"`
			Events  []struct {
				Address    string   `json:"address"`
				Identifier string   `json:"identifier"`
				Topics     []string `json:"topics"`
				Data       *string  `json:"data"`
			} `json:"events"`
		} `json:"logs"`
		Status           string   `json:"status"`
		Tokens           []string `json:"tokens"`
		EsdtValues       []string `json:"esdtValues"`
		Operation        string   `json:"operation"`
		InitiallyPaidFee string   `json:"initiallyPaidFee"`
	}
	_EgldShard struct {
		Hash  string `json:"hash"`
		Nonce uint64 `json:"nonce"`
		Shard int    `json:"shard"`
	}
	_Block struct {
		Nonce        uint64              `json:"nonce"`
		Round        uint64              `json:"round"`
		Hash         string              `json:"hash"`
		Epoch        uint64              `json:"epoch"`
		NumTxs       uint64              `json:"numTxs"`
		Timestamp    int64               `json:"timestamp"`
		Status       string              `jons:"status"`
		ShardBlocks  []*_EgldShard       `json:"shardBlocks"`
		Transactions []*_EgldTransaction `json:"transactions"`
	}
	_HyperblockRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Hyperblock *_Block `json:"hyperblock"`
		} `json:"data"`
	}
	_TransactionRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Transaction *_EgldTransaction `json:"transaction"`
		} `json:"data"`
	}
	_TransactionStatusRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_TransactionSendRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			TxHash string `json:"txHash"`
		} `json:"data"`
	}
	_TransactionSimulateRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Result struct {
				Status string `json:"status"`
				Hash   string `json:"hash"`
			} `json:"result"`
		} `json:"data"`
	}
	_EgldTokenData struct {
		Symbol          string `json:"symbol,omitempty"`
		Balance         string `json:"balance"`
		Properties      string `json:"properties"`
		TokenIdentifier string `json:"tokenIdentifier"`
	}
	_EsdtBalanceRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			TokenData *_EgldTokenData `json:"tokenData"`
		} `json:"data"`
	}
	_TokenInfoRes struct {
		Code  string `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Decimals int64  `json:"decimals"`
			Name     string `json:"name"`
			Ticker   string `json:"ticker"`
		} `json:"data"`
	}
)

type nativeTx struct {
	Nonce     uint64 `json:"nonce"`
	Value     string `json:"value"`
	RcvAddr   string `json:"receiver"`
	SndAddr   string `json:"sender"`
	GasPrice  uint64 `json:"gasPrice,omitempty"`
	GasLimit  uint64 `json:"gasLimit,omitempty"`
	Data      string `json:"data,omitempty"`
	Signature string `json:"signature,omitempty"`
	ChainID   string `json:"chainID"`
	Version   uint32 `json:"version"`
	Options   uint32 `json:"options,omitempty"`
}
