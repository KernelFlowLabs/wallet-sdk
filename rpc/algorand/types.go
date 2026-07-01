package algorand

type (
	_SendTxOut struct {
		Message string `json:"message"`
		TxId    string `json:"txId"`
	}
	_TransactionParams struct {
		Message          string `json:"message"`
		ConsensusVersion string `json:"consensus-version"`
		Fee              uint64 `json:"fee"`
		GenesisHash      string `json:"genesis-hash"`
		GenesisId        string `json:"genesis-id"`
		LastRound        uint64 `json:"last-round"`
		MinFee           uint64 `json:"min-fee"`
	}
	_NodeStatusResponse struct {
		Message                     string `json:"message"`
		Catchpoint                  string `json:"catchpoint,omitempty"`
		CatchpointAcquiredBlocks    uint64 `json:"catchpoint-acquired-blocks,omitempty"`
		CatchpointProcessedAccounts uint64 `json:"catchpoint-processed-accounts,omitempty"`
		CatchpointTotalAccounts     uint64 `json:"catchpoint-total-accounts,omitempty"`
		CatchpointTotalBlocks       uint64 `json:"catchpoint-total-blocks,omitempty"`
		CatchpointVerifiedAccounts  uint64 `json:"catchpoint-verified-accounts,omitempty"`
		CatchupTime                 uint64 `json:"catchup-time"`
		LastCatchpoint              string `json:"last-catchpoint,omitempty"`
		LastRound                   uint64 `json:"last-round"`
		LastVersion                 string `json:"last-version"`
		NextVersion                 string `json:"next-version"`
		NextVersionRound            uint64 `json:"next-version-round"`
		NextVersionSupported        bool   `json:"next-version-supported"`
		StoppedAtUnsupportedRound   bool   `json:"stopped-at-unsupported-round"`
		TimeSinceLastRound          uint64 `json:"time-since-last-round"`
	}
	_ApplicationStateSchema struct {
		NumByteSlice uint64 `json:"num-byte-slice"`
		NumUint      uint64 `json:"num-uint"`
	}
	_AssetHolding struct {
		Amount          uint64 `json:"amount"`
		AssetId         uint64 `json:"asset-id"`
		Creator         string `json:"creator"`
		Deleted         bool   `json:"deleted,omitempty"`
		IsFrozen        bool   `json:"is-frozen"`
		OptedInAtRound  uint64 `json:"opted-in-at-round,omitempty"`
		OptedOutAtRound uint64 `json:"opted-out-at-round,omitempty"`
	}
	_TealValue struct {
		Bytes string `json:"bytes"`
		Type  uint64 `json:"type"`
		Uint  uint64 `json:"uint"`
	}

	_TealKeyValue struct {
		Key   string     `json:"key"`
		Value _TealValue `json:"value"`
	}

	_ApplicationParams struct {
		ApprovalProgram   []byte                  `json:"approval-program"`
		ClearStateProgram []byte                  `json:"clear-state-program"`
		Creator           string                  `json:"creator,omitempty"`
		ExtraProgramPages uint64                  `json:"extra-program-pages,omitempty"`
		GlobalState       []_TealKeyValue         `json:"global-state,omitempty"`
		GlobalStateSchema _ApplicationStateSchema `json:"global-state-schema,omitempty"`
		LocalStateSchema  _ApplicationStateSchema `json:"local-state-schema,omitempty"`
	}

	_Application struct {
		CreatedAtRound uint64             `json:"created-at-round,omitempty"`
		Deleted        bool               `json:"deleted,omitempty"`
		DeletedAtRound uint64             `json:"deleted-at-round,omitempty"`
		Id             uint64             `json:"id"`
		Params         _ApplicationParams `json:"params"`
	}
	_ApplicationLocalState struct {
		ClosedOutAtRound uint64                  `json:"closed-out-at-round,omitempty"`
		Deleted          bool                    `json:"deleted,omitempty"`
		Id               uint64                  `json:"id"`
		KeyValue         []_TealKeyValue         `json:"key-value,omitempty"`
		OptedInAtRound   uint64                  `json:"opted-in-at-round,omitempty"`
		Schema           _ApplicationStateSchema `json:"schema"`
	}
	_AssetParams struct {
		Clawback      string `json:"clawback,omitempty"`
		Creator       string `json:"creator"`
		Decimals      uint64 `json:"decimals"`
		DefaultFrozen bool   `json:"default-frozen,omitempty"`
		Freeze        string `json:"freeze,omitempty"`
		Manager       string `json:"manager,omitempty"`
		MetadataHash  []byte `json:"metadata-hash,omitempty"`
		Name          string `json:"name,omitempty"`
		NameB64       []byte `json:"name-b64,omitempty"`
		Reserve       string `json:"reserve,omitempty"`
		Total         uint64 `json:"total"`
		UnitName      string `json:"unit-name,omitempty"`
		UnitNameB64   []byte `json:"unit-name-b64,omitempty"`
		Url           string `json:"url,omitempty"`
		UrlB64        []byte `json:"url-b64,omitempty"`
	}

	_Asset struct {
		CreatedAtRound   uint64       `json:"created-at-round,omitempty"`
		Deleted          bool         `json:"deleted,omitempty"`
		DestroyedAtRound uint64       `json:"destroyed-at-round,omitempty"`
		Index            uint64       `json:"index"`
		Params           _AssetParams `json:"params"`
	}

	_AccountParticipation struct {
		SelectionParticipationKey []byte `json:"selection-participation-key"`
		VoteFirstValid            uint64 `json:"vote-first-valid"`
		VoteKeyDilution           uint64 `json:"vote-key-dilution"`
		VoteLastValid             uint64 `json:"vote-last-valid"`
		VoteParticipationKey      []byte `json:"vote-participation-key"`
	}
	_AccountAddress struct {
		Message                     string                   `json:"message"`
		Address                     string                   `json:"address"`
		Amount                      uint64                   `json:"amount"`
		AmountWithoutPendingRewards uint64                   `json:"amount-without-pending-rewards"`
		AppsLocalState              []_ApplicationLocalState `json:"apps-local-state,omitempty"`
		AppsTotalExtraPages         uint64                   `json:"apps-total-extra-pages,omitempty"`
		AppsTotalSchema             _ApplicationStateSchema  `json:"apps-total-schema,omitempty"`
		Assets                      []_AssetHolding          `json:"assets,omitempty"`
		AuthAddr                    string                   `json:"auth-addr,omitempty"`
		ClosedAtRound               uint64                   `json:"closed-at-round,omitempty"`
		CreatedApps                 []_Application           `json:"created-apps,omitempty"`
		CreatedAssets               []_Asset                 `json:"created-assets,omitempty"`
		CreatedAtRound              uint64                   `json:"created-at-round,omitempty"`
		Deleted                     bool                     `json:"deleted,omitempty"`
		Participation               _AccountParticipation    `json:"participation,omitempty"`
		PendingRewards              uint64                   `json:"pending-rewards"`
		RewardBase                  uint64                   `json:"reward-base,omitempty"`
		Rewards                     uint64                   `json:"rewards"`
		Round                       uint64                   `json:"round"`
		SigType                     string                   `json:"sig-type,omitempty"`
		Status                      string                   `json:"status"`
	}

	_PendingTransactionResponse struct {
		Message            string `json:"message,omitempty"`
		ApplicationIndex   uint64 `json:"application-index,omitempty"`
		AssetClosingAmount uint64 `json:"asset-closing-amount,omitempty"`
		AssetIndex         uint64 `json:"asset-index,omitempty"`
		CloseRewards       uint64 `json:"close-rewards,omitempty"`
		ClosingAmount      uint64 `json:"closing-amount,omitempty"`
		ConfirmedRound     uint64 `json:"confirmed-round,omitempty"`
		PoolError          string `json:"pool-error"`
		Txn                struct {
			Sig string `json:"sig"`
			Txn struct {
				Type string `json:"type"`
				Snd  string `json:"snd"`
				Rcv  string `json:"rcv"`
				Arcv string `json:"arcv"`
				Aamt uint64 `json:"aamt"`
				Amt  uint64 `json:"amt"`
				Xaid uint64 `json:"xaid"`
				Fee  uint64 `json:"fee"`
				Note string `json:"note"`
			} `json:"txn"`
		} `json:"txn"`
	}

	//_PendingTransactionResponse struct {
	//	Message            string     `json:"message,omitempty"`
	//	ApplicationIndex   uint64     `json:"application-index,omitempty"`
	//	AssetClosingAmount uint64     `json:"asset-closing-amount,omitempty"`
	//	AssetIndex         uint64     `json:"asset-index,omitempty"`
	//	CloseRewards       uint64     `json:"close-rewards,omitempty"`
	//	ClosingAmount      uint64     `json:"closing-amount,omitempty"`
	//	ConfirmedRound     uint64     `json:"confirmed-round,omitempty"`
	//	Logs               [][]byte   `json:"logs,omitempty"`
	//	PoolError          string     `json:"pool-error"`
	//	ReceiverRewards    uint64     `json:"receiver-rewards,omitempty"`
	//	SenderRewards      uint64     `json:"sender-rewards,omitempty"`
	//	Transaction        _SignedTxn `json:"txn"`
	//}
	//_SignedTxn struct {
	//	_struct  struct{}     `codec:",omitempty,omitemptyarray"`
	//	Sig      string       `codec:"sig"`
	//	Txn      _Transaction `codec:"txn"`
	//	AuthAddr Address      `codec:"sgnr"`
	//}
	//_Transaction struct {
	//	_struct struct{} `codec:",omitempty,omitemptyarray"`
	//	Type    string   `codec:"type"`
	//	_Header
	//	_PaymentTxnFields
	//	_AssetTransferTxnFields
	//}
	//_Header struct {
	//	_struct     struct{} `codec:",omitempty,omitemptyarray"`
	//	Sender      Address  `codec:"snd"`
	//	Fee         uint64   `codec:"fee"`
	//	FirstValid  uint64   `codec:"fv"`
	//	LastValid   uint64   `codec:"lv"`
	//	Note        []byte   `codec:"note"`
	//	GenesisID   string   `codec:"gen"`
	//	GenesisHash Digest   `codec:"gh"`
	//	Group       Digest   `codec:"grp"`
	//	Lease       [32]byte `codec:"lx"`
	//	RekeyTo     Address  `codec:"rekey"`
	//}
	//_PaymentTxnFields struct {
	//	_struct          struct{} `codec:",omitempty,omitemptyarray"`
	//	Receiver         Address  `codec:"rcv"`
	//	Amount           uint64   `codec:"amt"`
	//	CloseRemainderTo Address  `codec:"close"`
	//}
	//_AssetTransferTxnFields struct {
	//	_struct       struct{} `codec:",omitempty,omitemptyarray"`
	//	XferAsset     uint64   `codec:"xaid"`
	//	AssetAmount   uint64   `codec:"aamt"`
	//	AssetSender   Address  `codec:"asnd"`
	//	AssetReceiver Address  `codec:"arcv"`
	//	AssetCloseTo  Address  `codec:"aclose"`
	//}
)
