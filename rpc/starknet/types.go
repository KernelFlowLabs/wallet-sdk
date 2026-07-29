package starknet

type (
	_BaseRequest struct {
		JsonRPC string      `json:"jsonrpc"`
		ID      string      `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}
	_ErrorResponse struct {
		Code    int    `json:"code"`
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	_BaseResponse struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      string          `json:"id"`
		Error   *_ErrorResponse `json:"error"`
	}

	_GetHeightRes struct {
		_BaseResponse
		Result uint64 `json:"result"`
	}
	_GetNonceRes struct {
		_BaseResponse
		Result string `json:"result"`
	}
	_GetBalanceRes struct {
		_BaseResponse
		Result []string `json:"result"`
	}
	_GetDecimalRes struct {
		_BaseResponse
		Result []string `json:"result"`
	}
	_FeeEstimate struct {
		L1GasConsumed     string `json:"l1_gas_consumed"`
		L1GasPrice        string `json:"l1_gas_price"`
		L2GasConsumed     string `json:"l2_gas_consumed"`
		L2GasPrice        string `json:"l2_gas_price"`
		L1DataGasConsumed string `json:"l1_data_gas_consumed"`
		L1DataGasPrice    string `json:"l1_data_gas_price"`
		OverallFee        string `json:"overall_fee"`
	}
	_EstimateFeeRes struct {
		_BaseResponse
		Result []_FeeEstimate `json:"result"`
	}
	_SendTxRes struct {
		_BaseResponse
		Result struct {
			TransactionHash string `json:"transaction_hash"`
		} `json:"result"`
	}
	_GetTransactionByHash struct {
		_BaseResponse
		Result struct {
			Calldata        []string `json:"calldata"`
			SenderAddress   string   `json:"sender_address"`
			TransactionHash string   `json:"transaction_hash"`
			Type            string   `json:"type"`
			Version         string   `json:"version"`
		} `json:"result"`
	}
	_GetTransaction struct {
		_BaseResponse
		Result struct {
			BlockNumber     int    `json:"block_number"`
			ExecutionStatus string `json:"execution_status"`
			FinalityStatus  string `json:"finality_status"`
			TransactionHash string `json:"transaction_hash"`
		} `json:"result"`
	}
)
