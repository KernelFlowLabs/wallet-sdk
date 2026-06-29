package sui

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	h := &Handler{}
	h.rpc = chainrpc.NewRequest(rpcUrl, map[string]string{
		"Content-Type": "application/json",
	})
	return h, nil
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	out := &_GetBlockCountRes{}
	body := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      1,
		Method:  "sui_getLatestCheckpointSequenceNumber",
		Params:  nil,
	}
	err := h.rpc.Post(ctx, out, "", body)
	if err != nil {
		return "", fmt.Errorf("failed to get latest block,err=%s", err)
	} else if out.Error.Code != 0 {
		return "", fmt.Errorf("failed to get latest block,errMsg=%s", out.Error.Message)
	}
	return out.Result, nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	coinType := contractAddress
	if contractAddress == signing.MagicContactAddressForNative {
		coinType = "0x2::sui::SUI"
	}

	out := &_GetBalanceRes{}
	body := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      1,
		Method:  "suix_getBalance",
		Params:  []string{address, coinType},
	}
	err := h.rpc.Post(ctx, out, "", body)
	if err != nil {
		return "", fmt.Errorf("failed to get latest balance, err=%s", err)
	} else if out.Error.Code != 0 {
		return "", fmt.Errorf("failed to get latest balance, errMsg=%s", out.Error.Message)
	}
	return out.Result.TotalBalance, nil
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	signedTxBytes, err := hex.DecodeString(signedHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode signedHex, err=%s", err)
	}
	var st signedTx
	err = json.Unmarshal(signedTxBytes, &st)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal signedTx, err=%s", err)
	}
	out := &_SendRawTransactionRes{}
	body := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      1,
		Method:  "sui_executeTransactionBlock",
		Params: []interface{}{
			st.Tx,
			[]string{st.Signature},
			map[string]bool{
				"showInput":          true,
				"showEffects":        true,
				"showEvents":         true,
				"showObjectChanges":  true,
				"showBalanceChanges": true,
				"showRawInput":       true,
				"showRawEffects":     true,
			},
			"WaitForLocalExecution",
		},
	}
	err = h.rpc.Post(ctx, out, "", body)
	if err != nil {
		return "", fmt.Errorf("failed to executeTransactionBlock, err=%s", err)
	} else if out.Error.Code != 0 {
		return "", fmt.Errorf("failed to executeTransactionBlock, errMsg=%s", out.Error.Message)
	}

	return out.Result.Digest, nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	out := &_GetTransactionsRes{}
	body := &_BaseRequest{
		JsonRPC: "2.0",
		ID:      1,
		Method:  "sui_getTransactionBlock",
		Params:  []interface{}{hash, map[string]bool{"showEffects": true}},
	}
	err := h.rpc.Post(ctx, out, "", body)
	if err != nil {
		return nil, fmt.Errorf("failed to getTransactionBlock, err=%s", err)
	} else if out.Error.Code != 0 {
		return nil, fmt.Errorf("failed to getTransactionBlock, errMsg=%s", out.Error.Message)
	}

	r := &chainrpc.TxResult{}
	if out.Result.Checkpoint == nil {
		r.Status = signing.TxStatusPending
		return r, nil
	} else if out.Result.Effects.Status.Status == "success" {
		r.Status = signing.TxStatusSucceeded
		r.Height = *out.Result.Checkpoint
		return r, nil
	}
	r.Status = signing.TxStatusFailed
	r.Height = *out.Result.Checkpoint
	return r, nil
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("CallContract not supported")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "getGasPrice":
		out := &_GetGasPriceRes{}
		body := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      1,
			Method:  "suix_getReferenceGasPrice",
			Params:  []string{params},
		}
		err := h.rpc.Post(ctx, out, "", body)
		if err != nil {
			return "", fmt.Errorf("failed to get gas price, err=%s", err)
		} else if out.Error.Code != 0 {
			return "", fmt.Errorf("failed to get gas price, errMsg=%s", out.Error.Message)
		}
		return out.Result, nil
	case "getCoinMetadata":
		out := &_GetCoinMetadataRes{}
		body := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      1,
			Method:  "suix_getCoinMetadata",
			Params:  []string{params},
		}
		err := h.rpc.Post(ctx, out, "", body)
		if err != nil {
			return "", fmt.Errorf("failed to get coin metadata, err=%s", err)
		} else if out.Error.Code != 0 {
			return "", fmt.Errorf("failed to get coin metadata, errMsg=%s", out.Error.Message)
		}
		rBytes, err := json.Marshal(out.Result)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal, err=%v", err)
		}
		return string(rBytes), nil
	case "getCoins":
		tmp := strings.Split(params, "_")
		if len(tmp) != 2 {
			return "", fmt.Errorf("len(tmp) != 2")
		}
		address := tmp[0]
		contractAddress := tmp[1]
		if contractAddress == signing.MagicContactAddressForNative {
			contractAddress = "0x2::sui::SUI"
		}
		out := &_GetCoinsRes{}
		cursor := ""
		var coins Coins
		for {
			request := struct {
				JsonRPC string   `json:"jsonrpc"`
				ID      int      `json:"id"`
				Method  string   `json:"method"`
				Params  []string `json:"params"`
			}{
				JsonRPC: "2.0",
				ID:      1,
				Method:  "suix_getCoins",
				Params:  []string{address, contractAddress},
			}

			if cursor != "" {
				request.Params = append(request.Params, cursor)
			}

			if err := h.rpc.Post(ctx, &out, "", request); err != nil {
				return "", fmt.Errorf("failed to get coins, err=%s", err)
			} else if out.Error.Code != 0 {
				return "", fmt.Errorf("failed to get coins, errMsg=%s", out.Error.Message)
			}
			coins = append(coins, out.Result.Data...)
			if !out.Result.HasNextPage {
				break
			}
			cursor = out.Result.NextCursor
		}
		rBytes, err := json.Marshal(coins)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal, err=%v", err)
		}
		return string(rBytes), nil
	case "getTokenDecimals":
		if params == signing.MagicContactAddressForNative {
			return "9", nil
		}
		out := &_GetCoinMetadataRes{}
		body := &_BaseRequest{
			JsonRPC: "2.0",
			ID:      1,
			Method:  "suix_getCoinMetadata",
			Params:  []string{params},
		}
		err := h.rpc.Post(ctx, out, "", body)
		if err != nil {
			return "", fmt.Errorf("failed to get coin metadata, err=%s", err)
		} else if out.Error.Code != 0 {
			return "", fmt.Errorf("failed to get coin metadata, errMsg=%s", out.Error.Message)
		}
		decimals := strconv.FormatInt(out.Result.Decimals, 10)
		if decimals == "" || decimals == "0" {
			return "", fmt.Errorf("got zero")
		}
		return decimals, nil
	}

	return "", fmt.Errorf("unsupported function")
}

// signedTx mirrors the serialized signed-transaction envelope produced by the
// wallet-sdk Sui TxBuilder (where the equivalent type is unexported).
type signedTx struct {
	Tx        string
	Signature string
}

// types
type (
	_BaseRequest struct {
		JsonRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}
	_BaseResponse struct {
		ID    int `json:"id"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
	}
	_GetBlockCountRes struct {
		_BaseResponse
		Result string `json:"result"`
	}
	_GetBalanceRes struct {
		_BaseResponse
		Result struct {
			CoinType        string      `json:"coinType"`
			CoinObjectCount int         `json:"coinObjectCount"`
			TotalBalance    string      `json:"totalBalance"`
			LockedBalance   interface{} `json:"lockedBalance"`
		}
	}

	MetaData struct {
		Decimals    int64  `json:"decimals"`
		Name        string `json:"name"`
		Symbol      string `json:"symbol"`
		Description string `json:"description"`
		IconUrl     string `json:"iconUrl"`
		Id          string `json:"id"`
	}
	_GetCoinMetadataRes struct {
		_BaseResponse
		Result MetaData `json:"result"`
	}
	_GetGasPriceRes struct {
		_BaseResponse
		Result string `json:"result"`
	}
	CoinItem struct {
		CoinType            string `json:"coinType"`
		CoinObjectId        string `json:"coinObjectId"`
		Version             string `json:"version"`
		Digest              string `json:"digest"`
		Balance             string `json:"balance"`
		PreviousTransaction string `json:"previousTransaction"`
	}
	Coins        []CoinItem
	_GetCoinsRes struct {
		_BaseResponse
		Result struct {
			Data        Coins  `json:"data"`
			NextCursor  string `json:"nextCursor"`
			HasNextPage bool   `json:"hasNextPage"`
		} `json:"result"`
	}
	_GetTransactionsRes struct {
		_BaseResponse
		Result struct {
			Digest  string `json:"digest"`
			Effects struct {
				MessageVersion string `json:"messageVersion"`
				Status         struct {
					Status string `json:"status"`
				} `json:"status"`
				ExecutedEpoch string `json:"executedEpoch"`
				GasUsed       struct {
					ComputationCost         string `json:"computationCost"`
					StorageCost             string `json:"storageCost"`
					StorageRebate           string `json:"storageRebate"`
					NonRefundableStorageFee string `json:"nonRefundableStorageFee"`
				} `json:"gasUsed"`
				TransactionDigest string `json:"transactionDigest"`
			} `json:"effects"`
			TimestampMs string  `json:"timestampMs"`
			Checkpoint  *string `json:"checkpoint"`
		} `json:"result"`
	}

	_SendRawTransactionRes struct {
		_BaseResponse
		Result struct {
			Digest      string `json:"digest"`
			Transaction struct {
				Data struct {
					MessageVersion string `json:"messageVersion"`
					Transaction    struct {
						Kind   string `json:"kind"`
						Inputs []struct {
							Type       string          `json:"type"`
							ValueType  string          `json:"valueType,omitempty"`
							Value      json.RawMessage `json:"value,omitempty"`
							ObjectType string          `json:"objectType,omitempty"`
							ObjectId   string          `json:"objectId,omitempty"`
							Version    string          `json:"version,omitempty"`
							Digest     string          `json:"digest,omitempty"`
						} `json:"inputs"`
						Transactions []struct {
							TransferObjects []interface{} `json:"TransferObjects"`
						} `json:"transactions"`
					} `json:"transaction"`
					Sender  string `json:"sender"`
					GasData struct {
						Payment []struct {
							ObjectId string `json:"objectId"`
							Version  int    `json:"version"`
							Digest   string `json:"digest"`
						} `json:"payment"`
						Owner  string `json:"owner"`
						Price  string `json:"price"`
						Budget string `json:"budget"`
					} `json:"gasData"`
				} `json:"data"`
				TxSignatures []string `json:"txSignatures"`
			} `json:"transaction"`
			RawTransaction string `json:"rawTransaction"`
			Effects        struct {
				MessageVersion string `json:"messageVersion"`
				Status         struct {
					Status string `json:"status"`
				} `json:"status"`
				ExecutedEpoch string `json:"executedEpoch"`
				GasUsed       struct {
					ComputationCost         string `json:"computationCost"`
					StorageCost             string `json:"storageCost"`
					StorageRebate           string `json:"storageRebate"`
					NonRefundableStorageFee string `json:"nonRefundableStorageFee"`
				} `json:"gasUsed"`
				TransactionDigest string `json:"transactionDigest"`
				Mutated           []struct {
					Owner struct {
						AddressOwner string `json:"AddressOwner"`
					} `json:"owner"`
					Reference struct {
						ObjectId string `json:"objectId"`
						Version  int    `json:"version"`
						Digest   string `json:"digest"`
					} `json:"reference"`
				} `json:"mutated"`
				GasObject struct {
					Owner struct {
						ObjectOwner string `json:"ObjectOwner"`
					} `json:"owner"`
					Reference struct {
						ObjectId string `json:"objectId"`
						Version  int    `json:"version"`
						Digest   string `json:"digest"`
					} `json:"reference"`
				} `json:"gasObject"`
				EventsDigest string `json:"eventsDigest"`
			} `json:"effects"`
			ObjectChanges []struct {
				Type      string `json:"type"`
				Sender    string `json:"sender"`
				Recipient struct {
					AddressOwner string `json:"AddressOwner"`
				} `json:"recipient"`
				ObjectType string `json:"objectType"`
				ObjectId   string `json:"objectId"`
				Version    string `json:"version"`
				Digest     string `json:"digest"`
			} `json:"objectChanges"`
		} `json:"result"`
	}
)
