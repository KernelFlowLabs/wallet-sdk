package algorand

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	chainrpc "github.com/KernelFlowLabs/wallet-sdk/rpc"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

var _ chainrpc.BasicChainHandler = (*Handler)(nil)

type Handler struct {
	rpc *chainrpc.Request
}

func NewHandler(rpcUrl string) (*Handler, error) {
	rpcBase := ""
	tmp := strings.Split(rpcUrl, "@")
	headers := map[string]string{"Content-Type": "application/json"}
	if len(tmp) == 1 {
		rpcBase = tmp[0]
	} else if len(tmp) == 2 {
		rpcBase = tmp[0]
		headers["X-Algo-API-Token"] = tmp[1]
	} else {
		return nil, fmt.Errorf("invalid params")
	}
	return &Handler{rpc: chainrpc.NewRequest(rpcBase, headers)}, nil
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	out := &_NodeStatusResponse{}
	err := h.rpc.Get(ctx, out, "v2/status", nil)
	if err != nil {
		return "", err
	} else if out.Message != "" {
		return "", fmt.Errorf("fail to get latest block, err=%v", out.Message)
	} else if out.LastRound == 0 {
		return "", fmt.Errorf("fail to get latest block, return value is 0")
	}
	return strconv.FormatUint(out.LastRound, 10), nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	if contractAddress == signing.MagicContactAddressForNative {
		value, err := h.getBaseCoinBalance(ctx, address)
		if err != nil {
			return "", err
		}
		return value.String(), nil
	}
	value, err := h.getTokenBalance(ctx, address, contractAddress)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

func (h *Handler) GetTransfersByHash(ctx context.Context, hash string, confirmation uint64) (*chainrpc.TxTransfers, error) {
	r := &chainrpc.TxTransfers{Hash: hash}
	txResult, err := h.CheckTx(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("fail to CheckTx, err=%v", err)
	} else if txResult.Status == signing.TxStatusFailed {
		r.Rejected = true
		if txResult.ErrMsg != "" {
			r.ErrMsg = txResult.ErrMsg
		} else {
			r.ErrMsg = "a failed tx"
		}
		return r, nil
	} else if txResult.Status == signing.TxStatusPending {
		r.ErrMsg = "pending tx"
		return r, nil
	} else if txResult.Status != signing.TxStatusSucceeded {
		r.ErrMsg = "not a succeeded tx"
		return r, nil
	}

	height, _ := strconv.ParseUint(txResult.Height, 10, 64)
	latestHeightStr, _ := h.GetHeight(ctx)
	latestHeight, _ := strconv.ParseUint(latestHeightStr, 10, 64)
	if height != 0 && latestHeight != 0 {
		if latestHeight-height < confirmation {
			r.ErrMsg = fmt.Sprintf("tx succeeded.But current confirmation number %d hasn't meet "+
				"expected number %d", latestHeight-height, confirmation)
			return r, nil
		}
	}

	tx, err := h.checkTransactionById(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("fail to checkTransactionById, err=%v", err)
	}
	transfer := &chainrpc.Transfer{}
	if tx.Txn.Txn.Type == "pay" {
		transfer.Sender = tx.Txn.Txn.Snd
		transfer.Recipient = tx.Txn.Txn.Rcv
		transfer.Amount = strconv.FormatUint(tx.Txn.Txn.Amt, 10)
		transfer.ContractAddress = signing.MagicContactAddressForNative
	} else if tx.Txn.Txn.Type == "axfer" {
		transfer.Sender = tx.Txn.Txn.Snd
		transfer.Recipient = tx.Txn.Txn.Arcv
		transfer.Amount = strconv.FormatUint(tx.Txn.Txn.Aamt, 10)
		transfer.ContractAddress = strconv.FormatUint(tx.Txn.Txn.Xaid, 10)
	} else {
		r.Rejected = true
		r.ErrMsg = "invalid tx Type"
		return r, nil
	}

	if transfer.Sender == "" || transfer.Recipient == "" || transfer.Amount == "" || transfer.ContractAddress == "" {
		r.ErrMsg = "incomplete return data"
		r.Rejected = true
		return r, nil
	}
	r.Transfers = append(r.Transfers, transfer)
	return r, nil
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	signedBytes, err := hex.DecodeString(strings.TrimPrefix(signedHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("fail to DecodeString signedHex, err=%v", err)
	}
	out := &_SendTxOut{}
	h.rpc.SetHeader("Content-Type", "application/x-binary")
	err = h.rpc.PostWithOutEncoded(ctx, out, "v2/transactions", signedBytes)
	if err != nil {
		return "", fmt.Errorf("fail to send transaction, err=%v", err)
	} else if out.Message != "" {
		return "", fmt.Errorf("fail to send transaction, err=%v", out.Message)
	} else if out.TxId == "" {
		return "", fmt.Errorf("fail to send transaction, return TxId is empty")
	}
	return out.TxId, nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	r := &chainrpc.TxResult{}
	tx, err := h.checkTransactionById(ctx, hash)
	if err != nil {
		if strings.Contains(err.Error(), "could not find the transaction") {
			r.Status = signing.TxStatusPending
			return r, nil
		}
		return nil, err
	}
	if tx.ConfirmedRound > 0 {
		r.Status = signing.TxStatusSucceeded
		r.Height = strconv.FormatUint(tx.ConfirmedRound, 10)
	} else if tx.PoolError != "" {
		r.Status = signing.TxStatusFailed
		r.ErrMsg = tx.PoolError
	} else {
		r.Status = signing.TxStatusPending
	}
	return r, nil
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "transactionsParams":
		out := &_TransactionParams{}
		err := h.rpc.Get(ctx, out, "v2/transactions/params", nil)
		if err != nil {
			return "", fmt.Errorf("fail to get latest transactions params, err=%v", err)
		} else if out.Message != "" {
			return "", fmt.Errorf("fail to get latest transactions params, err=%v", out.Message)
		}
		res := map[string]string{
			"genesisID":   out.GenesisId,
			"genesisHash": out.GenesisHash,
			"firstValid":  strconv.FormatUint(out.LastRound, 10),
		}
		resBytes, err := json.Marshal(res)
		if err != nil {
			return "", fmt.Errorf("fail to Marshal for res, err=%v", err)
		}
		return string(resBytes), nil
	case "isAddressActivatedForToken":
		tmp := strings.Split(params, ":")
		if len(tmp) != 2 {
			return "", fmt.Errorf("invalid params")
		}
		assetsId, err := strconv.ParseUint(tmp[1], 10, 64)
		if err != nil {
			return "", fmt.Errorf("fail to ParseUint for contractAddress, err=%v", err)
		}
		out := &_AccountAddress{}
		err = h.rpc.Get(ctx, out, "v2/accounts/"+tmp[0], nil)
		if err != nil {
			return "", fmt.Errorf("fail to get balance, err=%v", err)
		} else if out.Message != "" {
			return "", fmt.Errorf("fail to get balance, errMSg=%v", out.Message)
		}
		for _, v := range out.Assets {
			if v.AssetId == assetsId {
				return "true", nil
			}
		}
		return "false", nil
	case "getMinFee":
		out := &_TransactionParams{}
		err := h.rpc.Get(ctx, out, "v2/transactions/params", nil)
		if err != nil {
			return "", fmt.Errorf("fail to get latest transactions params, err=%v", err)
		} else if out.Message != "" {
			return "", fmt.Errorf("fail to get latest transactions params, err=%v", out.Message)
		}
		return strconv.FormatUint(out.MinFee, 10), nil
	}
	return "", fmt.Errorf("unsupported function")
}

func (h *Handler) getBaseCoinBalance(ctx context.Context, address string) (*big.Int, error) {
	out := &_AccountAddress{}
	err := h.rpc.Get(ctx, out, "v2/accounts/"+address, nil)
	if err != nil {
		return nil, fmt.Errorf("fail to get balance, err=%v", err)
	} else if out.Message != "" {
		return nil, fmt.Errorf("fail to get balance, err=%v", out.Message)
	}
	return big.NewInt(0).SetUint64(out.AmountWithoutPendingRewards), nil
}

func (h *Handler) getTokenBalance(ctx context.Context, address, contract string) (*big.Int, error) {
	assetsId, err := strconv.ParseUint(contract, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("fail to ParseUint for contract, err=%v", err)
	}
	out := &_AccountAddress{}
	err = h.rpc.Get(ctx, out, "v2/accounts/"+address, nil)
	if err != nil {
		return nil, fmt.Errorf("fail to get balance, err=%v", err)
	} else if out.Message != "" {
		return nil, fmt.Errorf("fail to get balance, err=%v", out.Message)
	}
	for _, v := range out.Assets {
		if v.AssetId == assetsId {
			return big.NewInt(0).SetUint64(v.Amount), nil
		}
	}
	return big.NewInt(0), nil
}

func (h *Handler) checkTransactionById(ctx context.Context, txID string) (*_PendingTransactionResponse, error) {
	out := &_PendingTransactionResponse{}
	err := h.rpc.Get(ctx, out, "v2/transactions/pending/"+txID, nil)
	if err != nil {
		return nil, fmt.Errorf("fail to get pending tx, err=%v", err)
	} else if out.Message != "" {
		return nil, fmt.Errorf("fail to get pending tx, errMsg=%v", out.Message)
	}
	return out, nil
}
