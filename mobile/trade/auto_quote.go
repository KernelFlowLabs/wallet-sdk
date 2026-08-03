package trade

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/kernelflowlabs/wallet-sdk/common/dexmodel"
	"github.com/kernelflowlabs/wallet-sdk/common/httpc"
	"github.com/kernelflowlabs/wallet-sdk/dex"
	"github.com/kernelflowlabs/wallet-sdk/dex/bungee"
	"github.com/kernelflowlabs/wallet-sdk/dex/lifi"
)

var (
	bngOnce sync.Once
	bng     *bungee.Client
	lfiOnce sync.Once
	lfi     *lifi.Client
	proxy   *httpc.Request
)

func autoQuoteBungee() *bungee.Client {
	bngOnce.Do(func() { bng = bungee.NewClient() })
	return bng
}

func autoQuoteLiFi() *lifi.Client {
	// Device-side keeps the previous free-tier defaults explicitly.
	lfiOnce.Do(func() {
		lfi = lifi.NewClient(
			lifi.WithRateLimit(lifi.RateLimitFree),
			lifi.WithIntegrator("onerootapp"),
		)
	})
	return lfi
}

func serverProxy() *httpc.Request {
	tradeMu.RLock()
	if proxy != nil || server == "" {
		p := proxy
		tradeMu.RUnlock()
		return p
	}
	tradeMu.RUnlock()

	tradeMu.Lock()
	defer tradeMu.Unlock()
	if proxy == nil && server != "" {
		proxy = httpc.NewRequest(server, nil)
	}
	return proxy
}

type autoQuoteReq struct {
	FromAmount    string           `json:"from_amount"`
	SlippageBps   int              `json:"slippage_bps"`
	FromAmountUsd float64          `json:"from_amount_usd"`
	ToPriceUsd    float64          `json:"to_price_usd"`
	TimeoutMs     int              `json:"timeout_ms"`
	FeeRate       string           `json:"fee_rate,omitempty"`
	FeeReceiver   string           `json:"fee_receiver,omitempty"`
	Candidates    []*wireCandidate `json:"candidates"`
}

type wireCandidate struct {
	Channel      string  `json:"channel"`
	Via          string  `json:"via"`
	FromChain    string  `json:"from_chain"`
	FromToken    string  `json:"from_token"`
	FromAddress  string  `json:"from_address"`
	ToChain      string  `json:"to_chain"`
	ToToken      string  `json:"to_token"`
	ToAddress    string  `json:"to_address"`
	ToDecimals   int     `json:"to_decimals"`
	LiquidityUsd float64 `json:"liquidity_usd"`
	SourceIssuer string  `json:"source_issuer,omitempty"`
}

func AutoQuote(reqJSON string) string {
	var req autoQuoteReq
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return errRespAutoQuote(fmt.Sprintf("parse request: %v", err))
	}
	cands := make([]*dex.Candidate, 0, len(req.Candidates))
	for _, c := range req.Candidates {
		cands = append(cands, &dex.Candidate{
			Channel:      c.Channel,
			Via:          c.Via,
			FromChain:    c.FromChain,
			FromToken:    c.FromToken,
			FromAddress:  c.FromAddress,
			ToChain:      c.ToChain,
			ToToken:      c.ToToken,
			ToAddress:    c.ToAddress,
			ToDecimals:   c.ToDecimals,
			LiquidityUsd: c.LiquidityUsd,
			SourceIssuer: c.SourceIssuer,
		})
	}
	sdkReq := &dex.Request{
		FromAmount:    req.FromAmount,
		SlippageBps:   req.SlippageBps,
		FromAmountUsd: req.FromAmountUsd,
		ToPriceUsd:    req.ToPriceUsd,
		Timeout:       time.Duration(req.TimeoutMs) * time.Millisecond,
		FeeRate:       req.FeeRate,
		FeeReceiver:   req.FeeReceiver,
	}
	engine := &dex.Engine{}
	resp := engine.Quote(context.Background(), sdkReq, cands, dispatch)
	b, _ := json.Marshal(resp)
	return string(b)
}

func dispatch(ctx context.Context, cand *dex.Candidate, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	if cand.Via == dex.ViaServer {
		return quoteViaServer(ctx, in)
	}
	if cand.Via == "" && cand.Channel == "jupiter" && cand.FromChain == "SOL" && cand.ToChain == "SOL" {
		return quoteViaServer(ctx, in)
	}
	switch cand.Channel {
	case "jupiter":
		j := jupClient()
		if j == nil {
			return nil, fmt.Errorf("jupiter not initialized — call Init")
		}
		return j.Quote(ctx, in)
	case "bungee":
		return autoQuoteBungee().Quote(ctx, in)
	case "lifi":
		return autoQuoteLiFi().Quote(ctx, in)
	}
	return nil, fmt.Errorf("unknown channel: %q", cand.Channel)
}

func quoteViaServer(ctx context.Context, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	p := serverProxy()
	if p == nil {
		return nil, fmt.Errorf("via=server but server_url not configured — pass it to Init")
	}
	q := url.Values{}
	q.Set("fromChain", in.FromChain)
	q.Set("toChain", in.ToChain)
	q.Set("fromToken", in.FromToken)
	q.Set("toToken", in.ToToken)
	q.Set("fromAddress", in.FromAddress)
	q.Set("toAddress", in.ToAddress)
	q.Set("fromAmount", in.FromAmount)
	if in.Slippage != "" {
		q.Set("slippage", in.Slippage)
	}
	if in.FromValueUsd != "" {
		q.Set("fromValueUsd", in.FromValueUsd)
	}
	if in.GasOnDestination != "" {
		q.Set("gasOnDestination", in.GasOnDestination)
	}
	if in.FeeRate != "" {
		q.Set("feeRate", in.FeeRate)
	}
	if in.FeeReceiver != "" {
		q.Set("feeReceiver", in.FeeReceiver)
	}
	var env struct {
		Code int                   `json:"code"`
		Msg  string                `json:"msg"`
		Data *dexmodel.DexQuoteOut `json:"data"`
	}
	if err := p.Get(ctx, &env, "public/trade/quote", q); err != nil {
		return nil, fmt.Errorf("proxy quote: %w", err)
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("proxy quote code=%d msg=%s", env.Code, env.Msg)
	}
	return env.Data, nil
}

func errRespAutoQuote(msg string) string {
	b, _ := json.Marshal(&dex.Response{
		Routes:        []*dex.Route{},
		Reason:        "INVALID_REQUEST",
		ReasonMessage: msg,
	})
	return string(b)
}
