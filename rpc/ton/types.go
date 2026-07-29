package ton

type (
	ApiBlock struct {
		Error        string           `json:"error,omitempty"`
		Transactions []ApiTransaction `json:"transactions"`
	}
	ApiTransaction struct {
		Error   string `json:"error,omitempty"`
		Hash    string `json:"hash"`
		Lt      uint64 `json:"lt"`
		Account struct {
			Address  string `json:"address"`
			Name     string `json:"name"`
			IsScam   bool   `json:"is_scam"`
			IsWallet bool   `json:"is_wallet"`
		} `json:"account"`
		Success         bool          `json:"success"`
		Utime           int64         `json:"utime"`
		OrigStatus      string        `json:"orig_status"`
		EndStatus       string        `json:"end_status"`
		TotalFees       uint64        `json:"total_fees"`
		EndBalance      int64         `json:"end_balance"`
		TransactionType string        `json:"transaction_type"`
		StateUpdateOld  string        `json:"state_update_old"`
		StateUpdateNew  string        `json:"state_update_new"`
		InMsg           interface{}   `json:"in_msg,omitempty"`
		OutMsgs         []interface{} `json:"out_msgs"`
		Block           string        `json:"block"`
		PrevTransHash   string        `json:"prev_trans_hash,omitempty"`
		PrevTransLt     uint64        `json:"prev_trans_lt,omitempty"`
		ComputePhase    struct {
			Skipped             bool   `json:"skipped"`
			Success             bool   `json:"success"`
			GasFees             int    `json:"gas_fees"`
			GasUsed             int    `json:"gas_used"`
			VMSteps             int    `json:"vm_steps"`
			ExitCode            int    `json:"exit_code"`
			ExitCodeDescription string `json:"exit_code_description"`
		} `json:"compute_phase"`
		ActionPhase struct {
			Success        bool   `json:"success"`
			ResultCode     int    `json:"result_code"`
			TotalActions   int    `json:"total_actions"`
			SkippedActions int    `json:"skipped_actions"`
			FwdFees        uint64 `json:"fwd_fees"`
			TotalFees      uint64 `json:"total_fees"`
		} `json:"action_phase,omitempty"`
		Aborted     bool   `json:"aborted"`
		Destroyed   bool   `json:"destroyed"`
		Raw         string `json:"raw"`
		CreditPhase struct {
			FeesCollected int   `json:"fees_collected"`
			Credit        int64 `json:"credit"`
		} `json:"credit_phase,omitempty"`
		StoragePhase struct {
			FeesCollected uint64 `json:"fees_collected"`
			StatusChange  string `json:"status_change"`
		} `json:"storage_phase,omitempty"`
		BouncePhase string `json:"bounce_phase,omitempty"`
	}
	InMsgDetails struct {
		MsgType     string `json:"msg_type"`
		CreatedLt   uint64 `json:"created_lt"`
		IhrDisabled bool   `json:"ihr_disabled"`
		Bounce      bool   `json:"bounce"`
		Bounced     bool   `json:"bounced"`
		Value       uint64 `json:"value"`
		FwdFee      uint64 `json:"fwd_fee"`
		IhrFee      uint64 `json:"ihr_fee"`
		Destination struct {
			Address  string `json:"address"`
			IsScam   bool   `json:"is_scam"`
			IsWallet bool   `json:"is_wallet"`
		} `json:"destination"`
		ImportFee     uint64 `json:"import_fee"`
		CreatedAt     uint64 `json:"created_at"`
		Hash          string `json:"hash"`
		RawBody       string `json:"raw_body"`
		DecodedOpName string `json:"decoded_op_name"`
		DecodedBody   struct {
			Signature   string `json:"signature"`
			SubwalletId uint64 `json:"subwallet_id"`
			ValidUntil  uint64 `json:"valid_until"`
			Seqno       int64  `json:"seqno"`
			Op          int    `json:"op"`
			Payload     []struct {
				Mode    int `json:"mode"`
				Message struct {
					SumType         string `json:"sum_type"`
					MessageInternal struct {
						IhrDisabled bool   `json:"ihr_disabled"`
						Bounce      bool   `json:"bounce"`
						Bounced     bool   `json:"bounced"`
						Src         string `json:"src"`
						Dest        string `json:"dest"`
						Value       struct {
							Grams string `json:"grams"`
							Other struct {
							} `json:"other"`
						} `json:"value"`
						IhrFee    string      `json:"ihr_fee"`
						FwdFee    string      `json:"fwd_fee"`
						CreatedLt uint64      `json:"created_lt"`
						CreatedAt uint64      `json:"created_at"`
						Init      interface{} `json:"init"`
						Body      struct {
							IsRight bool `json:"is_right"`
							Value   struct {
								SumType string `json:"sum_type"`
								OpCode  int    `json:"op_code"`
								Value   struct {
									QueryId             float64     `json:"query_id"`
									Amount              string      `json:"amount"`
									Destination         string      `json:"destination"`
									ResponseDestination string      `json:"response_destination"`
									CustomPayload       interface{} `json:"custom_payload"`
									ForwardTonAmount    string      `json:"forward_ton_amount"`
									ForwardPayload      struct {
										IsRight bool `json:"is_right"`
										Value   struct {
											SumType string `json:"sum_type"`
											OpCode  int    `json:"op_code"`
											Value   struct {
												Text string `json:"text"`
											} `json:"value"`
										} `json:"value"`
									} `json:"forward_payload"`
								} `json:"value"`
							} `json:"value"`
						} `json:"body"`
					} `json:"message_internal"`
				} `json:"message"`
			} `json:"payload"`
		} `json:"decoded_body"`
	}
	OutMsgDetails struct {
		MsgType     string `json:"msg_type"`
		CreatedLt   uint64 `json:"created_lt"`
		IhrDisabled bool   `json:"ihr_disabled"`
		Bounce      bool   `json:"bounce"`
		Bounced     bool   `json:"bounced"`
		Value       uint64 `json:"value"`
		FwdFee      uint64 `json:"fwd_fee"`
		IhrFee      uint64 `json:"ihr_fee"`
		Destination struct {
			Address  string `json:"address"`
			IsScam   bool   `json:"is_scam"`
			IsWallet bool   `json:"is_wallet"`
		} `json:"destination"`
		Source struct {
			Address  string `json:"address"`
			IsScam   bool   `json:"is_scam"`
			IsWallet bool   `json:"is_wallet"`
		} `json:"source"`
		ImportFee     uint64 `json:"import_fee"`
		CreatedAt     uint64 `json:"created_at"`
		OpCode        string `json:"op_code"`
		Hash          string `json:"hash"`
		RawBody       string `json:"raw_body"`
		DecodedOpName string `json:"decoded_op_name"`
		DecodedBody   struct {
			Text                string      `json:"text,omitempty"`
			QueryId             float64     `json:"query_id"`
			Amount              string      `json:"amount"`
			Destination         string      `json:"destination"`
			ResponseDestination string      `json:"response_destination"`
			CustomPayload       interface{} `json:"custom_payload"`
			ForwardTonAmount    string      `json:"forward_ton_amount"`
			ForwardPayload      struct {
				IsRight bool `json:"is_right"`
				Value   struct {
					SumType string `json:"sum_type"`
					OpCode  int    `json:"op_code"`
					Value   struct {
						Text string `json:"text"`
					} `json:"value"`
				} `json:"value"`
			} `json:"forward_payload"`
		} `json:"decoded_body"`
	}
)

type (
	RemotesTokenMeta struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Image       string `json:"image"`
		Symbol      string `json:"symbol"`
	}
)
