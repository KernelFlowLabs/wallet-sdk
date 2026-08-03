package bungee

import (
	dexmodel "github.com/kernelflowlabs/wallet-sdk/common/dexmodel"
	"encoding/json"
	"fmt"
)

type (
	QuoteRequest struct {
		OriginChainId      string
		DestinationChainId string
		InputToken         string
		OutputToken        string
		UserAddress        string
		ReceiverAddress    string
		InputAmount        string
		Slippage           string
	}
	Token struct {
		ChainId  int    `json:"chainId"`
		Address  string `json:"address"`
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals int    `json:"decimals"`
		LogoURI  string `json:"logoURI"`
		Icon     string `json:"icon"`
	}
	Protocol struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Icon        string `json:"icon"`
	}
	QuoteResponse struct {
		Success    bool     `json:"success"`
		StatusCode int      `json:"statusCode"`
		Message    string   `json:"message,omitempty"`
		Result     QuoteOut `json:"result"`
	}
	QuoteOut struct {
		OriginChainId      int    `json:"originChainId"`
		DestinationChainId int    `json:"destinationChainId"`
		UserAddress        string `json:"userAddress"`
		ReceiverAddress    string `json:"receiverAddress"`
		Input              struct {
			Token      Token   `json:"token"`
			Amount     string  `json:"amount"`
			PriceInUsd float64 `json:"priceInUsd"`
			ValueInUsd float64 `json:"valueInUsd"`
		} `json:"input"`
		Routes []QuoteRoute `json:"routes"`
	}
	QuoteRoute struct {
		UserOp    string `json:"userOp"`
		QuoteId   string `json:"quoteId"`
		ExpiresAt int64  `json:"expiresAt"`
		Output    struct {
			Token        Token   `json:"token"`
			Amount       string  `json:"amount"`
			MinAmountOut string  `json:"minAmountOut"`
			PriceInUsd   float64 `json:"priceInUsd"`
			ValueInUsd   float64 `json:"valueInUsd"`
		} `json:"output"`
		EstimatedTime     int      `json:"estimatedTime"`
		Slippage          float64  `json:"slippage"`
		SuggestedSlippage float64  `json:"suggestedSlippage"`
		RouteTags         []string `json:"routeTags"`
		RouteDetails      struct {
			DexDetails *struct {
				Protocol Protocol `json:"protocol"`
			} `json:"dexDetails"`
			BridgeDetails *struct {
				Protocol Protocol `json:"protocol"`
			} `json:"bridgeDetails"`
		} `json:"routeDetails"`
		Approval *struct {
			SpenderAddress string `json:"spenderAddress"`
			Amount         string `json:"amount"`
			TokenAddress   string `json:"tokenAddress"`
			UserAddress    string `json:"userAddress"`
		} `json:"approval"`
		TxData struct {
			Kind   string          `json:"kind"`
			Object json.RawMessage `json:"object"`
		} `json:"txData"`
		GasFee struct {
			GasToken     Token      `json:"gasToken"`
			GasLimit     flexString `json:"gasLimit"`
			GasPrice     flexString `json:"gasPrice"`
			EstimatedFee flexString `json:"estimatedFee"`
			FeeInUsd     float64    `json:"feeInUsd"`
		} `json:"gasFee"`
		StatusCheck struct {
			Endpoint       string `json:"endpoint"`
			Method         string `json:"method"`
			IntervalSec    int    `json:"intervalSec"`
			MaxDurationSec int    `json:"maxDurationSec"`
		} `json:"statusCheck"`
	}
	evmTxObject struct {
		ChainId int        `json:"chainId"`
		To      string     `json:"to"`
		Data    string     `json:"data"`
		Value   flexString `json:"value"`
	}
	flexString string
	StatusResponse struct {
		Success    bool      `json:"success"`
		StatusCode int       `json:"statusCode"`
		Message    string    `json:"message,omitempty"`
		Result     StatusOut `json:"result"`
	}
	StatusOut struct {
		QuoteId    string `json:"quoteId"`
		UserOp     string `json:"userOp"`
		Status     string `json:"status"`
		StatusCode string `json:"statusCode"`
		Origin     struct {
			ChainId int    `json:"chainId"`
			Status  string `json:"status"`
			TxHash  string `json:"txHash"`
		} `json:"origin"`
		Destination struct {
			ChainId int    `json:"chainId"`
			Status  string `json:"status"`
			TxHash  string `json:"txHash"`
		} `json:"destination"`
		Refund *struct {
			ChainId int    `json:"chainId"`
			TxHash  string `json:"txHash"`
		} `json:"refund"`
	}
)

func (f *flexString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*f = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(b)
	return nil
}

func (c *Client) toSocketQuoteReq(in *dexmodel.DexQuoteIn) *QuoteRequest {
	fromChainId, ok := idChainMapperReverse[in.FromChain]
	if !ok {
		return nil
	}
	toChainId, ok := idChainMapperReverse[in.ToChain]
	if !ok {
		return nil
	}
	return &QuoteRequest{
		OriginChainId:      fromChainId,
		DestinationChainId: toChainId,
		InputToken:         in.FromToken,
		OutputToken:        in.ToToken,
		UserAddress:        in.FromAddress,
		ReceiverAddress:    in.ToAddress,
		InputAmount:        in.FromAmount,
		Slippage:           in.Slippage,
	}
}

func routeName(rt *QuoteRoute) string {
	if b := rt.RouteDetails.BridgeDetails; b != nil && b.Protocol.DisplayName != "" {
		return b.Protocol.DisplayName
	}
	if d := rt.RouteDetails.DexDetails; d != nil && d.Protocol.DisplayName != "" {
		return d.Protocol.DisplayName
	}
	return "Socket"
}

func (c *Client) toStandardQuoteRes(res *QuoteOut) *dexmodel.DexQuoteOut {
	if res == nil {
		return nil
	}

	out := &dexmodel.DexQuoteOut{
		Routes: make([]*dexmodel.DexRoute, 0, len(res.Routes)),
	}
	for i := range res.Routes {
		rt := &res.Routes[i]
		route := &dexmodel.DexRoute{
			RouteId:           rt.QuoteId,
			Name:              routeName(rt),
			AmountOut:         rt.Output.Amount,
			AmountOutMin:      rt.Output.MinAmountOut,
			Slippage:          rt.Slippage,
			SuggestedSlippage: rt.SuggestedSlippage,
			FeeInUsd:          fmt.Sprintf("%.2f", rt.GasFee.FeeInUsd),
			EstimatedTime:     int64(rt.EstimatedTime),
			NeedBuild:         false,
			UserOp:            rt.UserOp,
			GasLimit:          string(rt.GasFee.GasLimit),
			ExpiresAt:         rt.ExpiresAt,
		}
		switch rt.TxData.Kind {
		case "", "evm_tx":
			var obj evmTxObject
			if err := json.Unmarshal(rt.TxData.Object, &obj); err == nil && obj.To != "" {
				route.TxData = &dexmodel.DexTx{
					To:       obj.To,
					Data:     obj.Data,
					Value:    string(obj.Value),
					GasPrice: string(rt.GasFee.GasPrice),
					GasLimit: string(rt.GasFee.GasLimit),
				}
			}
		default:
			if len(rt.TxData.Object) > 0 {
				route.TxData = &dexmodel.DexTx{Data: string(rt.TxData.Object)}
				route.UserOp = rt.TxData.Kind
			}
		}
		if rt.Approval != nil && rt.Approval.SpenderAddress != "" {
			route.ApprovalData = &dexmodel.DexApproval{
				Token:   rt.Approval.TokenAddress,
				Spender: rt.Approval.SpenderAddress,
				Amount:  rt.Approval.Amount,
			}
			route.TrustedSpenders = append(route.TrustedSpenders, rt.Approval.SpenderAddress)
		}
		out.Routes = append(out.Routes, route)
	}
	return out
}

var idChainMapper = map[string]string{
	"1":          "ETH",
	"56":         "BNB",
	"89999":      "SOL",
	"137":        "POLYGON",
	"43114":      "AVAXC",
	"42161":      "ARB",
	"10":         "OP",
	"8453":       "BASE",
	"59144":      "LINEA",
	"324":        "ZKSYNC",
	"1101":       "ZKEVM",
	"534352":     "SCROLL",
	"5000":       "MANTLE",
	"169":        "MANTA",
	"81457":      "BLAST",
	"34443":      "MODE",
	"252":        "FRAXTAL",
	"7777777":    "ZORA",
	"666666666":  "DEGEN",
	"1088":       "METIS",
	"100":        "GNOSIS",
	"250":        "FTM",
	"1284":       "MOONBEAM",
	"1285":       "MOONRIVER",
	"42220":      "CELO",
	"1313161554": "AURORA",
	"25":         "CRONOS",
	"288":        "BOBA",
	"122":        "FUSE",
	"1666600000": "HARMONY",
	"2222":       "KAVA",
	"321":        "KCC",
	"106":        "VELAS",
	"128":        "HECO",
	"66":         "OKEX",
	"40":         "TELOS",
	"167000":     "TAIKO",
	"196":        "XLAYER",
	"480":        "WORLDCHAIN",
	"1135":       "LISK",
	"1868":       "SONEIUM",
	"1923":       "SWELL",
	"60808":      "BOB",
	"33139":      "APECHAIN",
	"2741":       "ABSTRACT",
	"130":        "UNICHAIN",
	"57073":      "INK",
}

var idChainMapperReverse = func() map[string]string {
	m := make(map[string]string, len(idChainMapper))
	for k, v := range idChainMapper {
		m[v] = k
	}
	return m
}()
