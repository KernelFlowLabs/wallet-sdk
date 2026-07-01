package filecoin

type _BaseRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      uint64      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type _BaseResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Error   struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	}
}

// _Message mirrors the JSON shape a signed Filecoin message serializes to
// (BigInt fields are strings on the wire).
type _Message struct {
	Version    uint64 `json:"Version"`
	To         string `json:"To"`
	From       string `json:"From"`
	Nonce      uint64 `json:"Nonce"`
	Value      string `json:"Value"`
	GasLimit   int64  `json:"GasLimit"`
	GasFeeCap  string `json:"GasFeeCap"`
	GasPremium string `json:"GasPremium"`
	Method     uint64 `json:"Method"`
	Params     []byte `json:"Params"`
}

type _SignedMessage struct {
	Message   *_Message `json:"Message"`
	Signature struct {
		Type byte   `json:"Type"`
		Data []byte `json:"Data"`
	} `json:"Signature"`
}

type (
	_GetBlockHeight struct {
		Height uint64
	}
	_GetBlockHeightRes struct {
		_BaseResponse
		Result *_GetBlockHeight `json:"result"`
	}
	_GetBalanceResponse struct {
		_BaseResponse
		Result string `json:"result"`
	}
	_GetTipSetByHeight struct {
		Blocks []struct {
			Timestamp int64 `json:"Timestamp"`
		}
		Height uint64 `json:"Height"`
	}
	_GetTipSetByHeightRes struct {
		_BaseResponse
		Result *_GetTipSetByHeight `json:"result"`
	}
	_GetMessageResponse struct {
		_BaseResponse
		Result *_Message `json:"result"`
	}
	_GetMpoolGetNonceRes struct {
		_BaseResponse
		Result uint64 `json:"result"`
	}
	_MpoolPushResponse struct {
		_BaseResponse
		Result struct {
			Cid string `json:"/"`
		} `json:"result"`
	}
	_GasEstimateRes struct {
		_BaseResponse
		Result struct {
			GasLimit   int64  `json:"GasLimit"`
			GasFeeCap  string `json:"GasFeeCap"`
			GasPremium string `json:"GasPremium"`
		} `json:"result"`
	}
	_StateSearchMsgLimited struct {
		Receipt struct {
			ExitCode int64
			Return   string
			GasUsed  uint64
		}
		Height uint64
	}
	_StateSearchMsgLimitedRes struct {
		_BaseResponse
		Result *_StateSearchMsgLimited `json:"result"`
	}
)
