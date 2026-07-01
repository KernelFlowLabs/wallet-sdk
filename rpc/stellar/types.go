package stellar

import "fmt"
import "time"

// types
func (err *_ErrorResponse) Error() string {
	return fmt.Sprintf("RPC ERROR title=%s,status=%d,detail=%s,extras=%v",
		err.Title, err.Status, err.Detail, err.Extras)
}

type (
	_ErrorResponse struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
		Extras struct {
			EnvelopeXdr string `json:"envelope_xdr"`
			ResultCodes struct {
				Transaction string   `json:"transaction"`
				Operations  []string `json:"operations"`
			} `json:"result_codes"`
			ResultXdr string `json:"result_xdr"`
		} `json:"extras"`
	}
	_HalLink struct {
		Href      string `json:"href"`
		Templated bool   `json:"templated,omitempty"`
	}
	_RpcBase struct {
		Links struct {
			Self        _HalLink `json:"self"`
			Transaction _HalLink `json:"transaction"`
			Effects     _HalLink `json:"effects"`
			Succeeds    _HalLink `json:"succeeds"`
			Precedes    _HalLink `json:"precedes"`
		} `json:"_links"`
		TransactionSuccessful bool   `json:"transaction_successful"`
		SourceAccount         string `json:"source_account"`
		Type                  string `json:"type"`
		TypeI                 int32  `json:"type_i"`
		ID                    string `json:"id"`
		TransactionHash       string `json:"transaction_hash"`
	}
	_Ledger struct {
		Links struct {
			Self         _HalLink `json:"self"`
			Transactions _HalLink `json:"transactions"`
			Operations   _HalLink `json:"operations"`
			Payments     _HalLink `json:"payments"`
			Effects      _HalLink `json:"effects"`
		} `json:"_links"`
		ID                         string    `json:"id"`
		PT                         string    `json:"paging_token"`
		Hash                       string    `json:"hash"`
		PrevHash                   string    `json:"prev_hash,omitempty"`
		Sequence                   int32     `json:"sequence"`
		SuccessfulTransactionCount int32     `json:"successful_transaction_count"`
		FailedTransactionCount     *int32    `json:"failed_transaction_count"`
		OperationCount             int32     `json:"operation_count"`
		TxSetOperationCount        *int32    `json:"tx_set_operation_count"`
		ClosedAt                   time.Time `json:"closed_at"`
		TotalCoins                 string    `json:"total_coins"`
		FeePool                    string    `json:"fee_pool"`
		BaseFee                    int32     `json:"base_fee_in_stroops"`
		BaseReserve                int32     `json:"base_reserve_in_stroops"`
		MaxTxSetSize               int32     `json:"max_tx_set_size"`
		ProtocolVersion            int32     `json:"protocol_version"`
		HeaderXDR                  string    `json:"header_xdr"`
	}
	_LedgersPageRes struct {
		_ErrorResponse
		Links    _HalLink `json:"_links"`
		Embedded struct {
			Records []_Ledger
		} `json:"_embedded"`
	}
	_Balance struct {
		Balance                           string `json:"balance"`
		LiquidityPoolId                   string `json:"liquidity_pool_id,omitempty"`
		Limit                             string `json:"limit,omitempty"`
		BuyingLiabilities                 string `json:"buying_liabilities,omitempty"`
		SellingLiabilities                string `json:"selling_liabilities,omitempty"`
		Sponsor                           string `json:"sponsor,omitempty"`
		LastModifiedLedger                uint32 `json:"last_modified_ledger,omitempty"`
		IsAuthorized                      *bool  `json:"is_authorized,omitempty"`
		IsAuthorizedToMaintainLiabilities *bool  `json:"is_authorized_to_maintain_liabilities,omitempty"`
		IsClawbackEnabled                 *bool  `json:"is_clawback_enabled,omitempty"`
		AssetType                         string `json:"asset_type"`
		AssetCode                         string `json:"asset_code,omitempty"`
		AssetIssuer                       string `json:"asset_issuer,omitempty"`
	}
	_AccountRes struct {
		_ErrorResponse
		Links struct {
			Self         _HalLink `json:"self"`
			Transactions _HalLink `json:"transactions"`
			Operations   _HalLink `json:"operations"`
			Payments     _HalLink `json:"payments"`
			Effects      _HalLink `json:"effects"`
			Offers       _HalLink `json:"offers"`
			Trades       _HalLink `json:"trades"`
			Data         _HalLink `json:"data"`
		} `json:"_links"`

		ID                   string            `json:"id"`
		AccountID            string            `json:"account_id"`
		Sequence             string            `json:"sequence"`
		SubentryCount        int32             `json:"subentry_count"`
		InflationDestination string            `json:"inflation_destination,omitempty"`
		HomeDomain           string            `json:"home_domain,omitempty"`
		LastModifiedLedger   uint32            `json:"last_modified_ledger"`
		LastModifiedTime     *time.Time        `json:"last_modified_time"`
		Balances             []_Balance        `json:"balances"`
		Data                 map[string]string `json:"data"`
		NumSponsoring        uint32            `json:"num_sponsoring"`
		NumSponsored         uint32            `json:"num_sponsored"`
		Sponsor              string            `json:"sponsor,omitempty"`
		PT                   string            `json:"paging_token"`
	}
	_FeeBumpTransaction struct {
		Hash       string   `json:"hash"`
		Signatures []string `json:"signatures"`
	}
	_InnerTransaction struct {
		Hash       string   `json:"hash"`
		Signatures []string `json:"signatures"`
		MaxFee     int64    `json:"max_fee,string"`
	}
	_TransactionRes struct {
		_ErrorResponse
		Links struct {
			Self       _HalLink `json:"self"`
			Account    _HalLink `json:"account"`
			Ledger     _HalLink `json:"ledger"`
			Operations _HalLink `json:"operations"`
			Effects    _HalLink `json:"effects"`
			Precedes   _HalLink `json:"precedes"`
			Succeeds   _HalLink `json:"succeeds"`
			// Temporarily include Transaction as a link so that Transaction
			// can be fully compatible with TransactionSuccess
			// When TransactionSuccess is removed from the SDKs we can remove this HAL link
			Transaction _HalLink `json:"transaction"`
		} `json:"_links"`
		ID                 string               `json:"id"`
		PT                 string               `json:"paging_token"`
		Successful         bool                 `json:"successful"`
		Hash               string               `json:"hash"`
		Ledger             int32                `json:"ledger"`
		LedgerCloseTime    time.Time            `json:"created_at"`
		Account            string               `json:"source_account"`
		AccountMuxed       string               `json:"account_muxed,omitempty"`
		AccountMuxedID     uint64               `json:"account_muxed_id,omitempty,string"`
		AccountSequence    string               `json:"source_account_sequence"`
		FeeAccount         string               `json:"fee_account"`
		FeeAccountMuxed    string               `json:"fee_account_muxed,omitempty"`
		FeeAccountMuxedID  uint64               `json:"fee_account_muxed_id,omitempty,string"`
		FeeCharged         int64                `json:"fee_charged,string"`
		MaxFee             int64                `json:"max_fee,string"`
		OperationCount     int32                `json:"operation_count"`
		EnvelopeXdr        string               `json:"envelope_xdr"`
		ResultXdr          string               `json:"result_xdr"`
		ResultMetaXdr      string               `json:"result_meta_xdr"`
		FeeMetaXdr         string               `json:"fee_meta_xdr"`
		MemoType           string               `json:"memo_type"`
		MemoBytes          string               `json:"memo_bytes,omitempty"`
		Memo               string               `json:"memo,omitempty"`
		Signatures         []string             `json:"signatures"`
		ValidAfter         string               `json:"valid_after,omitempty"`
		ValidBefore        string               `json:"valid_before,omitempty"`
		FeeBumpTransaction *_FeeBumpTransaction `json:"fee_bump_transaction,omitempty"`
		InnerTransaction   *_InnerTransaction   `json:"inner_transaction,omitempty"`
	}
	_RpcOptionsRes struct {
		_ErrorResponse
		Links    _HalLink `json:"_links"`
		Embedded struct {
			Records []_RpcOptions
		} `json:"_embedded"`
	}
	_RpcOptions struct {
		_ErrorResponse
		Links    _HalLink `json:"_links"`
		Embedded struct {
			Records []_Ledger
		} `json:"_embedded"`
		//create_account
		_RpcBase
		StartingBalance string `json:"starting_balance"`
		Funder          string `json:"funder"`
		Account         string `json:"account"`
		//payment
		AssetType string `json:"asset_type"`
		From      string `json:"from"`
		To        string `json:"to"`
		Amount    string `json:"amount"`
	}
)
