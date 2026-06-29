package kaspa

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	chainrpc "github.com/KernelFlowLabs/wallet-sdk/rpc"
	"github.com/KernelFlowLabs/wallet-sdk/signing"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
)

var _ chainrpc.BasicChainHandler = (*Handler)(nil)

type Handler struct {
	client     rpcclient.RPCClient
	scanApi    *chainrpc.Request
	kasplexApi *chainrpc.Request
}

func NewHandler(url string) (*Handler, error) {
	h := &Handler{
		scanApi:    chainrpc.NewRequest("https://api.kaspa.org", nil),
		kasplexApi: chainrpc.NewRequest("https://api.kasplex.org/v1", nil),
	}
	client, err := rpcclient.NewRPCClient(url)
	if err != nil {
		return nil, fmt.Errorf("failed to NewRPCClient, err=%v", err)
	}
	client.SetTimeout(10 * time.Second)
	h.client = *client
	return h, nil
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	res, err := h.client.GetBlockCount()
	if err != nil {
		return "", fmt.Errorf("failed to GetBlockCount, err=%v", err)
	} else if res.Error != nil {
		return "", fmt.Errorf("failed to GetBlockCount, errMSg=%v", res.Error.Error())
	}
	return strconv.FormatUint(res.BlockCount, 10), nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	if contractAddress == signing.MagicContactAddressForNative ||
		contractAddress == "KAS" {
		getUTXOsByAddressesResponse, err := h.client.GetUTXOsByAddresses([]string{address})
		if err != nil {
			return "", err
		}
		dagInfo, err := h.client.GetBlockDAGInfo()
		if err != nil {
			return "", err
		}
		var balance uint64
		for _, entry := range getUTXOsByAddressesResponse.Entries {
			if !isUTXOSpendable(entry, dagInfo.VirtualDAAScore) {
				continue
			}
			balance += entry.UTXOEntry.Amount
		}
		return strconv.FormatUint(balance, 10), nil
	}

	path := "krc20/address/" + address + "/token/" + contractAddress
	res := &_KasplexGetBalanceRes{}
	err := h.kasplexApi.Get(ctx, res, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to GetBalance via krc20Api2, err=%v", err)
	} else if res.Message != "successful" {
		return "", fmt.Errorf("failed to GetBalance via krc20Api2, errMSg=%v", res.Message)
	}

	return res.Result[0].Balance, nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	tmp := strings.Split(hash, "_")
	var ticker string
	if len(tmp) == 1 {
		hash = tmp[0]
	} else if len(tmp) == 2 {
		hash = tmp[0]
		ticker = tmp[1]
	} else {
		return nil, fmt.Errorf("invalid hash")
	}

	r := &chainrpc.TxResult{}
	re := regexp.MustCompile(`^[0-9a-f]{64}$`)
	matched := re.MatchString(hash)
	if !matched {
		r.Status = signing.TxStatusFailed
		r.ErrMsg = "invalid hash"
		return r, nil
	}

	if ticker == "" {
		parentBlueScore, err := h.client.GetVirtualSelectedParentBlueScore()
		if err != nil {
			return nil, fmt.Errorf("failed to GetVirtualSelectedParentBlueScore, err=%v", err)
		} else if parentBlueScore.Error != nil {
			return nil, fmt.Errorf("failed to GetVirtualSelectedParentBlueScore, errMsg=%v",
				parentBlueScore.Error.Error())
		}
		rawTx, isPending, err := getNativeTx(ctx, h.scanApi, hash)
		if err != nil {
			return nil, fmt.Errorf("failed to getNativeTx, err=%v", err)
		} else if isPending || (rawTx != nil && rawTx.BlockTime > 0 && !rawTx.IsAccepted) {
			r.Status = signing.TxStatusPending
		} else if rawTx != nil && rawTx.BlockTime > 0 && rawTx.IsAccepted {
			r.Status = signing.TxStatusSucceeded
		}
		return r, nil
	}

	path := "krc20/op/" + hash
	res := &_KasplexGetTransactionRes{}
	err := h.kasplexApi.Get(ctx, res, path, nil)
	if err != nil {
		r.Status = signing.TxStatusPending
	} else if res.Message != "successful" {
		r.Status = signing.TxStatusFailed
		r.ErrMsg = res.Message
		if strings.Contains(res.Message, "op not found") {
			r.Status = signing.TxStatusPending
		} else {
			return r, fmt.Errorf("not successful, msg=%s", res.Message)
		}
	} else if res.Result[0].TxAccept != "1" || res.Result[0].OpAccept != "1" {
		r.Status = signing.TxStatusFailed
		r.ErrMsg = fmt.Sprintf("TxAccept or OpAccept !=1, opError=%s", res.Result[0].OpError)
		return r, nil
	} else if res.Result[0].From != "" && res.Result[0].To != "" && res.Result[0].Tick != "" {
		if res.Result[0].Amt == "" && (res.Result[0].From == res.Result[0].To) {
			r.Status = signing.TxStatusSucceeded
		} else if res.Result[0].Amt != "" {
			r.Status = signing.TxStatusSucceeded
		}
	} else {
		r.Status = signing.TxStatusPending
	}

	return r, nil
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	signedTxBytes, err := hex.DecodeString(signedHex)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString signedHex, err=%v", err)
	}
	stx := &signedTx{}
	err = json.Unmarshal(signedTxBytes, stx)
	if err != nil {
		return "", fmt.Errorf("failed to Unmarshal signedHex, err=%v", err)
	}
	rpcTransactionBytes, err := hex.DecodeString(stx.SignedHex)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString, err=%v", err)
	}
	transaction := &appmessage.RPCTransaction{}
	err = json.Unmarshal(rpcTransactionBytes, transaction)
	if err != nil {
		return "", fmt.Errorf("failed to Unmarshal, err=%v", err)
	}
	submitTransactionResponse, err := h.client.SubmitTransaction(transaction, stx.TxHash, false)
	if err != nil {
		if strings.Contains(err.Error(), "Rejected transaction") &&
			strings.Contains(err.Error(), "already spent by transaction") &&
			strings.Contains(err.Error(), "in the mempool") ||
			strings.Contains(err.Error(), "Rejected transaction") &&
				strings.Contains(err.Error(), "is an orphan where orphan is disallowed") {
			return "", fmt.Errorf("SEND_RETRY_ONCE")
		}
		return "", fmt.Errorf("error submitting transaction err=%v", err)
	}
	return submitTransactionResponse.TransactionID, nil
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, payload, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("CallContract not supported")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "estimateFee":
	case "getUtxo":
		res := &signing.UtxoList{}
		getUTXOsByAddressesResponse, err := h.client.GetUTXOsByAddresses([]string{params})
		if err != nil {
			return "", fmt.Errorf("failed to GetUTXOsByAddresses, err=%v", err)
		}
		dagInfo, err := h.client.GetBlockDAGInfo()
		if err != nil {
			return "", fmt.Errorf("failed to GetBlockDAGInfo, err=%v", err)
		}
		spendableUTXOs := make(map[appmessage.RPCOutpoint]*appmessage.RPCUTXOEntry, 0)
		for _, entry := range getUTXOsByAddressesResponse.Entries {
			if !isUTXOSpendable(entry, dagInfo.VirtualDAAScore) {
				continue
			}
			spendableUTXOs[*entry.Outpoint] = entry.UTXOEntry
		}
		for outpoint, utxo := range spendableUTXOs {
			outpointCopy := outpoint
			utxoInfo := &signing.UtxoInfo{
				Hash:          outpointCopy.TransactionID,
				Index:         strconv.FormatUint(uint64(outpointCopy.Index), 10),
				Script:        utxo.ScriptPublicKey.Script,
				Value:         strconv.FormatUint(utxo.Amount, 10),
				Version:       strconv.FormatUint(uint64(utxo.ScriptPublicKey.Version), 10),
				IsCoinbase:    strconv.FormatBool(utxo.IsCoinbase),
				BlockDAAScore: strconv.FormatUint(utxo.BlockDAAScore, 10),
			}
			if utxoInfo.Hash != "e0cedb45a7231772245bcf668c428dad8e3c64ee2761dcff0d95fa149f3dd4a2" {
				res.List = append(res.List, utxoInfo)
			}
		}
		resBytes, err := json.Marshal(res)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal for res, err=%v", err)
		}
		return string(resBytes), nil
	case "isTxInMemPool":
		mempoolEntries, err := h.client.GetMempoolEntries(true, false)
		if err != nil {
			return "", fmt.Errorf("failed to GetMempoolEntries, err=%v", err)
		}
		for _, entry := range mempoolEntries.Entries {
			dtx, err := appmessage.RPCTransactionToDomainTransaction(entry.Transaction)
			if err != nil {
				return "", fmt.Errorf("failed to RPCTransactionToDomainTransaction, err=%v", err)
			}
			txid := consensushashing.TransactionID(dtx).String()
			if txid == params {
				return "true", nil
			}
		}
		return "false", nil
	case "getMempoolEntriesByAddresses":
		_, err := h.client.GetMempoolEntriesByAddresses([]string{params},
			false, false)
		if err != nil {
			return "", fmt.Errorf("failed to GetMempoolEntries, err=%v", err)
		}
		return "false", nil
	case "submitTransaction":
		signedTxBytes, err := hex.DecodeString(params)
		if err != nil {
			return "", fmt.Errorf("failed to DecodeString signedHex, err=%v", err)
		}
		stx := &signedTx{}
		err = json.Unmarshal(signedTxBytes, stx)
		if err != nil {
			return "", fmt.Errorf("failed to Unmarshal signedHex, err=%v", err)
		}
		rpcTransactionBytes, err := hex.DecodeString(stx.SignedHex)
		if err != nil {
			return "", fmt.Errorf("failed to DecodeString, err=%v", err)
		}
		transaction := &appmessage.RPCTransaction{}
		err = json.Unmarshal(rpcTransactionBytes, transaction)
		if err != nil {
			return "", fmt.Errorf("failed to Unmarshal, err=%v", err)
		}
		submitTransactionResponse, err := h.client.SubmitTransaction(transaction, stx.TxHash, false)
		if err != nil {
			return "", fmt.Errorf("error submitting transaction err=%v", err)
		}
		return submitTransactionResponse.TransactionID, nil
	case "getVirtualSelectedParentBlueScore":
		parentBlueScore, err := h.client.GetVirtualSelectedParentBlueScore()
		if err != nil {
			return "", fmt.Errorf("failed to GetVirtualSelectedParentBlueScore, err=%v", err)
		}
		return strconv.FormatUint(parentBlueScore.BlueScore, 10), nil
	case "getPointHash":
		dagInfo, err := h.client.GetBlockDAGInfo()
		if err != nil {
			return "", fmt.Errorf("failed to GetBlockDAGInfo, err=%v", err)
		}
		return dagInfo.PruningPointHash, nil
	case "getAddedBlockHashes":
		block, err := h.client.GetVirtualSelectedParentChainFromBlock(params, false)
		if err != nil {
			return "", fmt.Errorf("failed to GetVirtualSelectedParentChainFromBlock, err=%v", err)
		}
		var added []string
		for _, hash := range block.AddedChainBlockHashes {
			added = append(added, hash)
		}
		addedBytes, err := json.Marshal(added)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal added, err=%v", err)
		}
		return string(addedBytes), nil
	case "getBlock":
		block, err := h.client.GetBlock(params, true)
		if err != nil {
			return "", fmt.Errorf("failed to GetBlock, err=%v", err)
		}
		blockBytes, err := json.Marshal(block)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal block, err=%v", err)
		}
		return string(blockBytes), nil
	case "getTokenDecimals":
		if params == signing.MagicContactAddressForNative {
			return "8", nil
		}
	}
	return "", fmt.Errorf("unsupported function %s", instruction)
}

// Verify UTXO is spendable (check if a minimum of 10 confirmations have been processed since UTXO creation)
func isUTXOSpendable(entry *appmessage.UTXOsByAddressesEntry, virtualSelectedParentBlueScore uint64) bool {
	blockDAAScore := entry.UTXOEntry.BlockDAAScore
	if !entry.UTXOEntry.IsCoinbase {
		const minConfirmations = 10
		return blockDAAScore+minConfirmations < virtualSelectedParentBlueScore
	}
	coinbaseMaturity := dagconfig.MainnetParams.BlockCoinbaseMaturity
	return blockDAAScore+coinbaseMaturity < virtualSelectedParentBlueScore
}

// types
type (
	_GetTransactionRes struct {
		SubnetworkID            string     `json:"subnetwork_id"`
		TransactionID           string     `json:"transaction_id"`
		Hash                    string     `json:"hash"`
		Mass                    string     `json:"mass"`
		BlockHash               []string   `json:"block_hash"`
		BlockTime               int64      `json:"block_time"`
		IsAccepted              bool       `json:"is_accepted"`
		AcceptingBlockHash      string     `json:"accepting_block_hash"`
		AcceptingBlockBlueScore uint64     `json:"accepting_block_blue_score"`
		Inputs                  []_Inputs  `json:"inputs"`
		Outputs                 []_Outputs `json:"outputs"`
		Detail                  string     `json:"detail"`
	}

	_Inputs struct {
		ID                    int    `json:"id"`
		TransactionID         string `json:"transaction_id"`
		Index                 int    `json:"index"`
		PreviousOutpointHash  string `json:"previous_outpoint_hash"`
		PreviousOutpointIndex string `json:"previous_outpoint_index"`
		SignatureScript       string `json:"signature_script"`
		SigOpCount            string `json:"sig_op_count"`
	}

	_Outputs struct {
		ID                     int         `json:"id"`
		TransactionID          string      `json:"transaction_id"`
		Index                  int         `json:"index"`
		Amount                 int         `json:"amount"`
		ScriptPublicKey        string      `json:"script_public_key"`
		ScriptPublicKeyAddress string      `json:"script_public_key_address"`
		ScriptPublicKeyType    string      `json:"script_public_key_type"`
		AcceptingBlockHash     interface{} `json:"accepting_block_hash"`
	}
)

type (
	_KasScanGetTransactionRes struct {
		Message                 string      `json:"message"`
		Detail                  string      `json:"detail"`
		SubnetworkId            string      `json:"subnetwork_id"`
		TransactionId           string      `json:"transaction_id"`
		Hash                    string      `json:"hash"`
		Mass                    string      `json:"mass"`
		Payload                 interface{} `json:"payload"`
		BlockHash               []string    `json:"block_hash"`
		BlockTime               int64       `json:"block_time"`
		IsAccepted              bool        `json:"is_accepted"`
		AcceptingBlockHash      string      `json:"accepting_block_hash"`
		AcceptingBlockBlueScore int         `json:"accepting_block_blue_score"`
		AcceptingBlockTime      int64       `json:"accepting_block_time"`
		Inputs                  []struct {
			TransactionId           string      `json:"transaction_id"`
			Index                   int         `json:"index"`
			PreviousOutpointHash    string      `json:"previous_outpoint_hash"`
			PreviousOutpointIndex   string      `json:"previous_outpoint_index"`
			PreviousOutpointAddress interface{} `json:"previous_outpoint_address"`
			PreviousOutpointAmount  interface{} `json:"previous_outpoint_amount"`
			SignatureScript         string      `json:"signature_script"`
			SigOpCount              string      `json:"sig_op_count"`
		} `json:"inputs"`
		Outputs []struct {
			TransactionId          string `json:"transaction_id"`
			Index                  int    `json:"index"`
			Amount                 int    `json:"amount"`
			ScriptPublicKey        string `json:"script_public_key"`
			ScriptPublicKeyAddress string `json:"script_public_key_address"`
			ScriptPublicKeyType    string `json:"script_public_key_type"`
		} `json:"outputs"`
	}
)

type (
	_KasplexGetBalanceRes struct {
		Message string `json:"message"`
		Result  []struct {
			Tick       string `json:"tick"`
			Balance    string `json:"balance"`
			Locked     string `json:"locked"`
			Dec        string `json:"dec"`
			OpScoreMod string `json:"opScoreMod"`
		} `json:"result"`
	}
	_KasplexGetTransactionRes struct {
		Message string `json:"message"`
		Result  []struct {
			P          string `json:"p"`
			Op         string `json:"op"`
			Tick       string `json:"tick"`
			Amt        string `json:"amt"`
			From       string `json:"from"`
			To         string `json:"to"`
			OpScore    string `json:"opScore"`
			HashRev    string `json:"hashRev"`
			FeeRev     string `json:"feeRev"`
			TxAccept   string `json:"txAccept"`
			OpAccept   string `json:"opAccept"`
			OpError    string `json:"opError"`
			Checkpoint string `json:"checkpoint"`
			MtsAdd     string `json:"mtsAdd"`
			MtsMod     string `json:"mtsMod"`
		} `json:"result"`
	}
	_KASSendRequest struct {
		Recipient string `json:"recipient"`
		Amount    string `json:"amount"`
		OrderId   string `json:"orderId"`
	}
	_KasplexSendResponse struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OrderInfo struct {
				CommitHash string `json:"commitHash"`
				Amount     string `json:"amount"`
				Tick       string `json:"tick"`
				Recipient  string `json:"recipient"`
				Error      string `json:"error"`
				RevealHash string `json:"revealHash"`
				OrderId    string `json:"orderId"`
			} `json:"orderInfo"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}
	_KRC20SendRequest struct {
		Tick      string `json:"tick"`
		Recipient string `json:"recipient"`
		Amount    string `json:"amount"`
		Minter    string `json:"minter"`
		OrderId   string `json:"orderId"`
	}
	_KRC20SubmitTransactionResponse struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OrderInfo struct {
				CommitHash string `json:"commitHash"`
				Amount     string `json:"amount"`
				Tick       string `json:"tick"`
				Recipient  string `json:"recipient"`
				Error      string `json:"error"`
				RevealHash string `json:"revealHash"`
				OrderId    string `json:"orderId"`
			} `json:"orderInfo"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}
	_KRC20EstimateFee struct {
		PriorityBucket struct {
			Feerate          int `json:"feerate"`
			EstimatedSeconds int `json:"estimatedSeconds"`
		} `json:"priorityBucket"`
		NormalBuckets []struct {
			Feerate          int     `json:"feerate"`
			EstimatedSeconds float64 `json:"estimatedSeconds"`
		} `json:"normalBuckets"`
		LowBuckets []struct {
			Feerate          int     `json:"feerate"`
			EstimatedSeconds float64 `json:"estimatedSeconds"`
		} `json:"lowBuckets"`
	}
	_KRC20DelOrderResponse struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Timestamp string `json:"timestamp"`
			OrderId   string `json:"orderId"`
		} `json:"data"`
	}
)

func getNativeTx(ctx context.Context, api *chainrpc.Request, hash string) (*_KasScanGetTransactionRes, bool, error) {
	out := &_KasScanGetTransactionRes{}
	err := api.Get(ctx, out, "transactions/"+hash, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get transactions, hash=%s, err=%v", hash, err)
	} else if out.Detail != "" {
		if strings.Contains(out.Detail, "Transaction not found") {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("failed to get transactions, hash=%s, detail=%s",
			hash, out.Detail)
	} else if out.Message != "" {
		return nil, false, fmt.Errorf("failed to get transactions, hash=%s, message=%s",
			hash, out.Message)
	}
	return out, false, nil
}

type signedTx struct {
	SignedHex string
	TxHash    string
}
