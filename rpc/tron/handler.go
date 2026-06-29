package tron

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
	wallettron "github.com/KernelFlowLabs/wallet-sdk/signing/tron"
)

var _ chainrpc.BasicChainHandler = (*Handler)(nil)

type Handler struct {
	rpc      *chainrpc.Request
	tronGrid *chainrpc.Request
}

func NewHandler(rpcUrl string) (*Handler, error) {
	h := &Handler{}

	h.rpc = chainrpc.NewRequest(rpcUrl, nil)
	h.tronGrid = chainrpc.NewRequest("https://api.trongrid.io", map[string]string{
		"TRON_PRO_API_KEY": "8dc8fb48-b87d-427e-821f-95b7160f2b51",
	})
	return h, nil
}

func (h *Handler) GetHeight(ctx context.Context) (string, error) {
	block := &_Block{}
	err := h.rpc.Post(ctx, block, "wallet/getnowblock", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get latest block, err=%v", err)
	}
	height := block.BlockHeader.Data.Number
	if height == 0 {
		return "", fmt.Errorf("got zero")
	}
	return strconv.FormatUint(height, 10), nil
}

func (h *Handler) GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error) {
	if contractAddress == signing.MagicContactAddressForNative ||
		contractAddress == signing.MagicContactAddressForNativeTRX {
		bal, err := h.getBaseCoinBalance(ctx, address)
		if err != nil {
			return "", fmt.Errorf("failed to getBaseCoinBalance, err=%v", err)
		}
		return strconv.FormatUint(bal, 10), nil
	} else {
		balance, err := h.getTokenBalance(ctx, address, contractAddress)
		if err != nil {
			return "", fmt.Errorf("failed to getTokenBalance, err=%v", err)
		}
		return balance.String(), nil
	}
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
		if latestHeight-height < confirmation {
			result.ErrMsg = fmt.Sprintf("tx succeeded.But current confirmation number %d hasn't meet "+
				"expected number %d", latestHeight-height, confirmation)
			return result, nil
		}
	}

	in := &_GetTransactionInfoByIdReq{
		Value: hash,
	}
	tx := &Tx{}
	err = h.rpc.Post(ctx, tx, "wallet/gettransactionbyid", in)
	if err != nil {
		return nil, fmt.Errorf("failed to gettransactionbyid, err=%v", err)
	}

	transfer := &chainrpc.Transfer{}
	if tx.Data.Contracts[0].Type == "TransferContract" {
		transfer.Sender = wallettron.ConvertFromHex(tx.Data.Contracts[0].Parameter.Value.OwnerAddress)
		transfer.Recipient = wallettron.ConvertFromHex(tx.Data.Contracts[0].Parameter.Value.ToAddress)
		amt := tx.Data.Contracts[0].Parameter.Value.Amount
		transfer.Amount = amt.String()
		transfer.ContractAddress = signing.MagicContactAddressForNative

		if transfer.Sender != "" &&
			transfer.Recipient != "" &&
			transfer.Amount != "" &&
			transfer.ContractAddress != "" {
			result.Transfers = append(result.Transfers, transfer)
			result.BalanceChange = append(result.BalanceChange, &chainrpc.BalanceChange{
				Address:         transfer.Sender,
				ContractAddress: transfer.ContractAddress,
				Change:          new(big.Int).Neg(amt).String(),
			})
			result.BalanceChange = append(result.BalanceChange, &chainrpc.BalanceChange{
				Address:         transfer.Recipient,
				ContractAddress: transfer.ContractAddress,
				Change:          amt.String(),
			})
		}
	} else if tx.Data.Contracts[0].Type == "TriggerSmartContract" {
		data := tx.Data.Contracts[0].Parameter.Value.Data
		if len(data) != 128+len(SignatureTransferMethod) ||
			data[0:8] != SignatureTransferMethod {
			result.Rejected = true
			result.ErrMsg = "not a token transfer"
			return result, nil
		}
		transfer.Sender = wallettron.ConvertFromHex(tx.Data.Contracts[0].Parameter.Value.OwnerAddress)
		transfer.ContractAddress = wallettron.ConvertFromHex(tx.Data.Contracts[0].Parameter.Value.ContractAddress)
		recipientHexAddr := "41" + data[len(SignatureTransferMethod)+
			24:len(SignatureTransferMethod)+64]
		transfer.Recipient = wallettron.ConvertFromHex(recipientHexAddr)
		amt, ok := big.NewInt(0).SetString(data[len(SignatureTransferMethod)+64:], 16)
		if !ok {
			result.Rejected = true
			result.ErrMsg = "failed to setString for amount"
			return result, nil
		}
		transfer.Amount = amt.String()

		if transfer.Sender != "" &&
			transfer.Recipient != "" &&
			transfer.Amount != "" &&
			transfer.ContractAddress != "" {
			result.Transfers = append(result.Transfers, transfer)
			result.BalanceChange = append(result.BalanceChange, &chainrpc.BalanceChange{
				Address:         transfer.Sender,
				ContractAddress: transfer.ContractAddress,
				Change:          new(big.Int).Neg(amt).String(),
			})
			result.BalanceChange = append(result.BalanceChange, &chainrpc.BalanceChange{
				Address:         transfer.Recipient,
				ContractAddress: signing.MagicContactAddressForNative,
				Change:          amt.String(),
			})
		}
	} else {
		result.Rejected = true
		result.ErrMsg = "invalid tx type"
		return result, nil
	}

	return result, nil
}

func (h *Handler) GetAddressFee(ctx context.Context, address string) (wallettron.FreeGas, error) {
	res := wallettron.FreeGas{}
	ar, err := h.getAccountResource(ctx, address)
	if err != nil {
		return res, fmt.Errorf("failed to getAccountResource, err=%v", err)
	}
	res.FreeNetUsed = strconv.FormatInt(ar.FreeNetLimit, 10)
	res.FreeNetLimit = strconv.FormatInt(ar.FreeNetLimit, 10)
	return res, nil
}

func (h *Handler) SendTx(ctx context.Context, signedHex string) (string, error) {
	signedBytes, err := hex.DecodeString(strings.TrimPrefix(signedHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString, err=%v", err)
	}
	ntx := &wallettron.NativeTx{}
	err = json.Unmarshal(signedBytes, ntx)
	if err != nil {
		return "", fmt.Errorf("failed to Unmarshal for ntx, err=%v", err)
	}
	req := &_BroadcastJsonRawTransactionReq{
		Signature: ntx.Signature,
		ID:        ntx.ID,
		RawData:   ntx.RawData,
	}
	res := &_BroadcastJsonRawTransactionRes{}
	err = h.rpc.Post(ctx, res, "wallet/broadcasttransaction", req)
	if err != nil {
		return "", fmt.Errorf("failed to broadcasttransaction, err=%v", err)
	} else if res.Result == false {
		return "", fmt.Errorf("failed to broadcasttransaction, errCode=%v, errMsg=%v", res.Code, res.Message)
	}

	return res.TxID, nil
}

func (h *Handler) CheckTx(ctx context.Context, hash string) (*chainrpc.TxResult, error) {
	if hash == "" {
		return nil, fmt.Errorf("hash is empty")
	}
	r := &chainrpc.TxResult{}

	req := &_GetTransactionInfoByIdReq{
		Value: hash,
	}
	res := &_GetTransactionInfoByIdRes{}
	err := h.rpc.Post(ctx, res, "wallet/gettransactioninfobyid", req)
	if err != nil {
		return nil, fmt.Errorf("failed to gettransactioninfobyid, err=%v", err)
	}

	r.Height = strconv.FormatUint(res.BlockNumber, 10)
	r.Time = strconv.FormatInt(res.BlockTimeStamp/1000, 10)
	r.GasUsed = strconv.FormatUint(res.Receipt.NetUsage, 10)
	if res.Result == "FAILED" {
		r.Status = signing.TxStatusFailed
		r.ErrMsg = res.Result
	} else {
		ntx := &wallettron.NativeTx{}
		err = h.rpc.Post(ctx, ntx, "wallet/gettransactionbyid", req)
		if err != nil {
			return nil, fmt.Errorf("failed to gettransactionbyid, err=%v", err)
		} else if len(ntx.Ret) < 1 {
			r.Status = signing.TxStatusPending
		} else if ntx.Ret[0].ContractRet == "SUCCESS" && r.Height != "0" {
			r.Status = signing.TxStatusSucceeded

			var logs []chainrpc.EvmLog
			for _, v := range res.Log {
				log := chainrpc.EvmLog{
					Address: "0x" + v.Address,
					Topics:  strings.Join(v.Topics, ","),
					Data:    v.Data,
				}
				logs = append(logs, log)
			}
			r.Logs = logs
		} else {
			r.Status = signing.TxStatusPending
		}
	}
	return r, nil
}

func (h *Handler) CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error) {
	return nil, fmt.Errorf("CallContract not supported")
}

func (h *Handler) InquireChain(ctx context.Context, instruction, params string) (string, error) {
	switch instruction {
	case "getRefInfo":
		var blockHeight string
		var err error
		if params == "" {
			blockHeight, err = h.GetHeight(ctx)
			if err != nil {
				return "", err
			}
		}
		blockNumberUint64, err := strconv.ParseUint(blockHeight, 10, 64)
		if err != nil {
			return "", fmt.Errorf("failed to ParseUint, err=%v", err)
		}
		block, err := h.getBlockByNumber(ctx, blockNumberUint64)
		if err != nil {
			return "", fmt.Errorf("failed to getBlockByNumber, err=%v", err)
		}
		return block.BlockId + "_" +
			blockHeight + "_" +
			strconv.FormatInt(block.BlockHeader.Data.Timestamp, 10), nil
	case "getAddressFee":
		res := wallettron.FreeGas{}
		ar, err := h.getAccountResource(ctx, params)
		if err != nil {
			return "", fmt.Errorf("failed to getAccountResource, err=%v", err)
		}
		res.FreeNetUsed = strconv.FormatInt(ar.FreeNetLimit, 10)
		res.FreeNetLimit = strconv.FormatInt(ar.FreeNetLimit, 10)
		resBytes, err := json.Marshal(res)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal for res, err=%v", err)
		}
		return string(resBytes), nil
	case "getBlockByNumber":
		blockNumberUint64, err := strconv.ParseUint(params, 10, 64)
		if err != nil {
			return "", fmt.Errorf("failed to ParseUint, err=%v", err)
		}
		block, err := h.getBlockByNumber(ctx, blockNumberUint64)
		if err != nil {
			return "", fmt.Errorf("failed to getBlockByNumber, err=%v", err)
		}
		resBytes, err := json.Marshal(block)
		if err != nil {
			return "", fmt.Errorf("failed to Marshal, err=%v", err)
		}
		return string(resBytes), nil
	case "getAllowance":
		tmp := strings.Split(params, ":")
		if len(tmp) != 3 {
			return "", fmt.Errorf("len(tmp) != 3")
		}
		contractAddressHex := wallettron.ConvertToHex(tmp[0])
		if len(contractAddressHex) == 0 {
			return "", fmt.Errorf("contractAddressHex is empty")
		}
		ownerHex := wallettron.ConvertToHex(tmp[1])
		if len(ownerHex) == 0 {
			return "", fmt.Errorf("ownerHex is empty")
		}
		spenderHex := wallettron.ConvertToHex(tmp[2])
		if len(spenderHex) == 0 {
			return "", fmt.Errorf("spenderHex is empty")
		}
		in := &_SmartContractReq{
			OwnerAddress:     ownerHex,
			ContractAddress:  contractAddressHex,
			FunctionSelector: "allowance(address,address)",
			Parameter: "000000000000000000000000" + ownerHex[2:] +
				"000000000000000000000000" + spenderHex[2:],
		}
		out := &_ConstantSmartContractRes{}
		err := h.rpc.Post(ctx, out, "wallet/triggerconstantcontract", in)
		if err != nil {
			return "", err
		} else if out.Result.Result == false {
			return "", fmt.Errorf("get Result==false")
		}
		r := parseContractNumber(out.ConstantResult[0])
		return r.String(), nil
	case "getTokenDecimals":
		if params == signing.MagicContactAddressForNative {
			return "6", nil
		}
		contractAddressHex := wallettron.ConvertToHex(params)
		if len(contractAddressHex) == 0 {
			return "", fmt.Errorf("contractAddressHex is empty")
		}
		in := &_SmartContractReq{
			OwnerAddress:     contractAddressHex,
			ContractAddress:  contractAddressHex,
			FunctionSelector: "decimals()",
			Parameter:        "",
		}
		out := &_ConstantSmartContractRes{}
		err := h.rpc.Post(ctx, out, "wallet/triggerconstantcontract", in)
		if err != nil {
			return "", err
		} else if out.Result.Result == false {
			return "", fmt.Errorf("get Result==false")
		}
		r := parseContractNumber(out.ConstantResult[0])
		return r.String(), nil
	}
	return "", fmt.Errorf("unsupported function")
}

// unexported
func (h *Handler) getBaseCoinBalance(ctx context.Context, address string) (uint64, error) {
	in := &_GetAccountReq{
		wallettron.ConvertToHex(address),
	}
	out := &_GetAccountRes{}
	err := h.rpc.Post(ctx, out, "wallet/getaccount", in)
	if err != nil {
		return 0, err
	}
	return out.Balance, nil
}
func (h *Handler) getTokenBalance(ctx context.Context, address string, contract string) (*big.Int, error) {
	addressHex := wallettron.ConvertToHex(address)
	if len(addressHex) == 0 {
		return nil, fmt.Errorf("addressHex is empty")
	}
	contractHex := wallettron.ConvertToHex(contract)
	if len(contract) == 0 {
		return nil, fmt.Errorf("contract is empty")
	}
	in := &_SmartContractReq{
		OwnerAddress:     addressHex,
		ContractAddress:  contractHex,
		FunctionSelector: "balanceOf(address)",
		Parameter:        "000000000000000000000000" + addressHex[2:],
	}
	out := &_ConstantSmartContractRes{}
	err := h.rpc.Post(ctx, out, "wallet/triggerconstantcontract", in)
	if err != nil {
		return nil, err
	} else if out.Result.Result == false {
		return nil, fmt.Errorf("get Result==false")
	}
	r := parseContractNumber(out.ConstantResult[0])
	return r, nil
}
func (h *Handler) getBlockByNumber(ctx context.Context, num uint64) (*_Block, error) {
	blocks := &_Blocks{}
	err := h.rpc.Post(ctx, blocks, "wallet/getblockbylimitnext",
		_BlockRequest{StartNum: num, EndNum: num + 1})
	if err != nil {
		return nil, err
	}
	if blocks.Blocks == nil {
		return nil, fmt.Errorf("get blocks.Blocks == nil")
	}
	if len(blocks.Blocks) == 0 {
		return nil, fmt.Errorf("len(blocks.Blocks) == 0")
	}
	return &blocks.Blocks[0], nil
}
func (h *Handler) getAccountResource(ctx context.Context, address string) (*_GetAccountResourceOut, error) {
	out := &_GetAccountResourceOut{}
	err := h.rpc.Post(ctx, out, "wallet/getaccountresource",
		_GetAccountResourceIn{Address: address, Visible: true})
	if err != nil {
		return nil, err
	}
	return out, nil
}
func parseContractNumber(data string) *big.Int {
	if len(strings.TrimPrefix(data, "0x")) == 64 {
		var n big.Int
		_, ok := n.SetString(data, 16)
		if ok {
			return &n
		}
	}
	return nil
}

// SignatureTransferMethod is the TRC20 transfer(address,uint256) method selector.
const SignatureTransferMethod = "a9059cbb"

// types
type (
	_GetAccountReq struct {
		Address string `json:"address"`
	}
	_GetAccountRes struct {
		Address    string `json:"address"`
		Balance    uint64 `json:"balance"`
		CreateTime int64  `json:"create_time"`
	}

	_GetAccountResourceIn struct {
		Address string `json:"address"`
		Visible bool   `json:"visible"`
	}
	_GetAccountResourceOut struct {
		FreeNetUsed  int64 `json:"freeNetUsed"`
		FreeNetLimit int64 `json:"freeNetLimit"`
	}

	_GetTransactionInfoByIdReq struct {
		Value string `json:"value"`
	}
	_GetTransactionInfoByIdRes struct {
		Result         string `json:"result"`
		TxID           string `json:"id"`
		BlockNumber    uint64 `json:"blockNumber"`
		BlockTimeStamp int64  `json:"blockTimeStamp"`
		Receipt        struct {
			NetUsage uint64 `json:"net_usage"`
			Result   string `json:"result"`
		} `json:"receipt"`
		Log []struct {
			Address string   `json:"address"`
			Topics  []string `json:"topics"`
			Data    string   `json:"data"`
		} `json:"log"`
	}

	_SmartContractReq struct {
		OwnerAddress     string   `json:"owner_address"`
		ContractAddress  string   `json:"contract_address"`
		FunctionSelector string   `json:"function_selector"`
		Parameter        string   `json:"parameter"`
		FeeLimit         int64    `json:"fee_limit"`
		CallValue        *big.Int `json:"call_value"`
		Visible          bool     `json:"visible"`
	}

	_ConstantSmartContractRes struct {
		Result struct {
			Result bool `json:"result"`
		} `json:"result"`
		ConstantResult []string `json:"constant_result"`
	}

	_BlockRequest struct {
		StartNum uint64 `json:"startNum"`
		EndNum   uint64 `json:"endNum"`
	}

	_Blocks struct {
		Blocks []_Block `json:"block"`
	}

	_Block struct {
		BlockId     string `json:"blockID"`
		Txs         []Tx   `json:"transactions"`
		BlockHeader struct {
			Data BlockData `json:"raw_data"`
		} `json:"block_header"`
	}

	BlockData struct {
		Number         uint64 `json:"number"`
		TxTrieRoot     string `json:"txTrieRoot"`
		WitnessAddress string `json:"witness_address"`
		ParentHash     string `json:"parentHash"`
		Version        int    `json:"version"`
		Timestamp      int64  `json:"timestamp"`
	}
	Tx struct {
		Ret       []_TxRet `json:"ret,omitempty"`
		Signature []string `json:"signature,omitempty"`
		ID        string   `json:"txID,omitempty"`
		BlockTime int64    `json:"block_timestamp,omitempty"`
		Data      _TxData  `json:"raw_data,omitempty"`
		Visible   bool     `json:"visible,omitempty"`
	}

	_TxRet struct {
		ContractRet string `json:"contractRet"`
	}

	_TxData struct {
		Contracts     []_TRXContract `json:"contract"`
		RefBlockBytes string         `json:"ref_block_bytes,omitempty"`
		RefBlockHash  string         `json:"ref_block_hash,omitempty"`
		Expiration    int64          `json:"expiration,omitempty"`
		FeeLimit      int64          `json:"fee_limit,omitempty"`
		Timestamp     int64          `json:"timestamp"`
	}

	_TRXContract struct {
		Type      string `json:"type"`
		Parameter struct {
			Value   _TransferValue `json:"value"`
			TypeUrl string         `json:"type_url"`
		} `json:"parameter"`
	}
	_TransferValue struct {
		OwnerAddress    string   `json:"owner_address,omitempty"`
		ToAddress       string   `json:"to_address,omitempty"`
		Data            string   `json:"data,omitempty"`
		ContractAddress string   `json:"contract_address,omitempty"`
		Amount          *big.Int `json:"amount,omitempty"`
		CallValue       *big.Int `json:"call_value,omitempty"`
		Visible         bool     `json:"visible"`
	}

	_BroadcastJsonRawTransactionReq struct {
		Signature []string          `json:"signature,omitempty"`
		ID        string            `json:"txID,omitempty"`
		RawData   *wallettron.TxRaw `json:"raw_data,omitempty"`
	}
	_BroadcastJsonRawTransactionRes struct {
		Result  bool   `json:"result,omitempty"`
		Code    string `json:"code,omitempty"`
		TxID    string `json:"txid,omitempty"`
		Message string `json:"message,omitempty"`
	}

	_BroadcastProtoRawTransactionReq struct {
		Transaction string `json:"transaction"`
	}
	_BroadcastProtoRawTransactionRes struct {
		Result      bool   `json:"result"`
		Code        string `json:"code"`
		Error       string `json:"error"`
		TxID        string `json:"txID"`
		Message     string `json:"message"`
		Transaction string `json:"transaction"`
	}
	_TriggerSmartContractReq struct {
		ContractAddress []byte
		Data            []byte
		Owner           []byte
		BlockNumber     int64
	}

	RpcBlock struct {
		BlockId     string `json:"blockID"`
		Txs         []Tx   `json:"transactions"`
		BlockHeader struct {
			Data BlockData `json:"raw_data"`
		} `json:"block_header"`
	}
)
