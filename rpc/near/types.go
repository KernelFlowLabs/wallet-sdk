package near

import "fmt"

// types
func (err *_ErrorResponse) Error() string {
	return fmt.Sprintf("RPC ERROR code=%d,message=%s,data=%s", err.Code, err.Message, err.Data)
}

type (
	_BaseRequest struct {
		JsonRPC string      `json:"jsonrpc"`
		ID      string      `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}
	_ErrorResponse struct {
		Code    int         `json:"code"`
		Name    string      `json:"name"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
		Cause   interface{} `json:"cause"`
	}
	_BaseResponse struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      string          `json:"id"`
		Error   *_ErrorResponse `json:"error"`
	}
)

type (
	_GetBlockRes struct {
		_BaseResponse
		Result _Block `json:"result"`
	}
	_Block struct {
		Author string `json:"author"`
		Header struct {
			Approvals             []interface{} `json:"approvals"`
			BlockMerkleRoot       string        `json:"block_merkle_root"`
			ChallengesResult      []interface{} `json:"challenges_result"`
			ChallengesRoot        string        `json:"challenges_root"`
			ChunkHeadersRoot      string        `json:"chunk_headers_root"`
			ChunkMask             []bool        `json:"chunk_mask"`
			ChunkReceiptsRoot     string        `json:"chunk_receipts_root"`
			ChunkTxRoot           string        `json:"chunk_tx_root"`
			ChunksIncluded        int64         `json:"chunks_included"`
			EpochID               string        `json:"epoch_id"`
			GasPrice              string        `json:"gas_price"`
			Hash                  string        `json:"hash"`
			Height                int64         `json:"height"`
			LastDsFinalBlock      string        `json:"last_ds_final_block"`
			LastFinalBlock        string        `json:"last_final_block"`
			LatestProtocolVersion int           `json:"latest_protocol_version"`
			NextBpHash            string        `json:"next_bp_hash"`
			NextEpochID           string        `json:"next_epoch_id"`
			OutcomeRoot           string        `json:"outcome_root"`
			PrevHash              string        `json:"prev_hash"`
			PrevStateRoot         string        `json:"prev_state_root"`
			RandomValue           string        `json:"random_value"`
			RentPaid              string        `json:"rent_paid"`
			Signature             string        `json:"signature"`
			Timestamp             int64         `json:"timestamp"`
			TimestampNanosec      string        `json:"timestamp_nanosec"`
			TotalSupply           string        `json:"total_supply"`
			ValidatorProposals    []interface{} `json:"validator_proposals"`
			ValidatorReward       string        `json:"validator_reward"`
		} `json:"header"`
		Chunks []struct {
			BalanceBurnt         string        `json:"balance_burnt"`
			ChunkHash            string        `json:"chunk_hash"`
			EncodedLength        int           `json:"encoded_length"`
			EncodedMerkleRoot    string        `json:"encoded_merkle_root"`
			GasLimit             int64         `json:"gas_limit"`
			GasUsed              int           `json:"gas_used"`
			HeightCreated        int           `json:"height_created"`
			HeightIncluded       int           `json:"height_included"`
			OutcomeRoot          string        `json:"outcome_root"`
			OutgoingReceiptsRoot string        `json:"outgoing_receipts_root"`
			PrevBlockHash        string        `json:"prev_block_hash"`
			PrevStateRoot        string        `json:"prev_state_root"`
			RentPaid             string        `json:"rent_paid"`
			ShardID              int           `json:"shard_id"`
			Signature            string        `json:"signature"`
			TxRoot               string        `json:"tx_root"`
			ValidatorProposals   []interface{} `json:"validator_proposals"`
			ValidatorReward      string        `json:"validator_reward"`
		} `json:"chunks"`
	}

	_GetChunkRes struct {
		_BaseResponse
		Result _Chunk `json:"result"`
	}
	_Chunk struct {
		Author string `json:"author"`
		Header struct {
			BalanceBurnt         string        `json:"balance_burnt"`
			ChunkHash            string        `json:"chunk_hash"`
			EncodedLength        int           `json:"encoded_length"`
			EncodedMerkleRoot    string        `json:"encoded_merkle_root"`
			GasLimit             int64         `json:"gas_limit"`
			GasUsed              int64         `json:"gas_used"`
			HeightCreated        int64         `json:"height_created"`
			HeightIncluded       int64         `json:"height_included"`
			OutcomeRoot          string        `json:"outcome_root"`
			OutgoingReceiptsRoot string        `json:"outgoing_receipts_root"`
			PrevBlockHash        string        `json:"prev_block_hash"`
			PrevStateRoot        string        `json:"prev_state_root"`
			RentPaid             string        `json:"rent_paid"`
			ShardID              int           `json:"shard_id"`
			Signature            string        `json:"signature"`
			TxRoot               string        `json:"tx_root"`
			ValidatorProposals   []interface{} `json:"validator_proposals"`
			ValidatorReward      string        `json:"validator_reward"`
		} `json:"header"`
		Receipts     []interface{} `json:"receipts"`
		Transactions []struct {
			Actions []struct {
				Transfer *struct {
					Deposit string `json:"deposit"`
				} `json:"Transfer,omitempty"`
			} `json:"actions"`
			Hash       string `json:"hash"`
			Nonce      uint64 `json:"nonce"`
			PublicKey  string `json:"public_key"`
			ReceiverID string `json:"receiver_id"`
			Signature  string `json:"signature"`
			SignerID   string `json:"signer_id"`
		} `json:"transactions"`
	}

	_GetBalanceRes struct {
		_BaseResponse
		Result _Account `json:"result"`
	}
	_Account struct {
		Amount string `json:"amount"`
	}

	_ReceiptRes struct {
		_BaseResponse
		Result _TransactionReceipt `json:"result"`
	}
	_TransactionReceipt struct {
		ReceiptsOutcome []struct {
			BlockHash string `json:"block_hash"`
			ID        string `json:"id"`
			Outcome   struct {
				ExecutorID string        `json:"executor_id"`
				GasBurnt   int64         `json:"gas_burnt"`
				Logs       []interface{} `json:"logs"`
				ReceiptIds []string      `json:"receipt_ids"`
				Status     struct {
					SuccessValue string `json:"SuccessValue"`
				} `json:"status"`
				TokensBurnt string `json:"tokens_burnt"`
			} `json:"outcome"`
			Proof []struct {
				Direction string `json:"direction"`
				Hash      string `json:"hash"`
			} `json:"proof"`
		} `json:"receipts_outcome"`
		Status struct {
			SuccessValue     string      `json:"SuccessValue,omitempty"`
			SuccessReceiptId string      `json:"SuccessReceiptId,omitempty"`
			Failure          interface{} `json:"Failure,omitempty"`
			Unknown          string      `json:"Unknown,omitempty"`
		} `json:"status"`
		Transaction struct {
			Actions []struct {
				Transfer struct {
					Deposit string `json:"deposit"`
				} `json:"Transfer"`
				FunctionCall struct {
					MethodName string `json:"method_name"`
					Args       string `json:"args"`
					Gas        uint64 `json:"gas"`
					Deposit    string `json:"deposit"`
				} `json:"FunctionCall"`
			} `json:"actions"`
			Hash       string `json:"hash"`
			Nonce      int    `json:"nonce"`
			PublicKey  string `json:"public_key"`
			ReceiverID string `json:"receiver_id"`
			Signature  string `json:"signature"`
			SignerID   string `json:"signer_id"`
		} `json:"transaction"`
		TransactionOutcome struct {
			BlockHash string `json:"block_hash"`
			ID        string `json:"id"`
			Outcome   struct {
				ExecutorID string        `json:"executor_id"`
				GasBurnt   int64         `json:"gas_burnt"`
				Logs       []interface{} `json:"logs"`
				ReceiptIds []string      `json:"receipt_ids"`
				Status     struct {
					SuccessReceiptID string `json:"SuccessReceiptId"`
				} `json:"status"`
				TokensBurnt string `json:"tokens_burnt"`
			} `json:"outcome"`
			Proof []struct {
				Direction string `json:"direction"`
				Hash      string `json:"hash"`
			} `json:"proof"`
		} `json:"transaction_outcome"`
	}
	_GetNonceRes struct {
		_BaseResponse
		Result _AccountNonce `json:"result"`
	}
	_AccountNonce struct {
		Error       string `json:"error"`
		Nonce       uint64 `json:"nonce"`
		Permission  string `json:"permission"`
		BlockHeight int    `json:"block_height"`
		BlockHash   string `json:"block_hash"`
	}

	_ContractTokenRes struct {
		_BaseResponse
		Result _ContractRes `json:"result"`
	}

	_ContractRes struct {
		BlockHash   string        `json:"block_hash"`
		BlockHeight int           `json:"block_height"`
		Logs        []interface{} `json:"logs"`
		Result      []uint8       `json:"result"`
	}

	_TokenStorageBounds struct {
		Min string `json:"min"`
		Max string `json:"max"`
	}
	_TokenStorageBalance struct {
		Total     string `json:"total"`
		Available string `json:"available"`
	}

	nearTokenTransfer struct {
		ReceiverId string `json:"receiver_id"`
		Amount     string `json:"amount"`
	}
)
