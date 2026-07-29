package dex

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/kernelflowlabs/wallet-sdk/common/dexmodel"
)

const DefaultTimeout = 5 * time.Second
const ThinLiquidityRatio = 0.05

const (
	WarnThinLiquidity    = "THIN_LIQUIDITY"
	WarnPriceImpactHigh  = "PRICE_IMPACT_HIGH"
)

type Candidate struct {
	Channel      string
	Via          string
	FromChain    string
	FromToken    string
	FromAddress  string
	ToChain      string
	ToToken      string
	ToAddress    string
	ToDecimals   int
	LiquidityUsd float64
	SourceIssuer string
}

const (
	ViaDevice = "device"
	ViaServer = "server"
)

func (c *Candidate) CrossChain() bool { return c.FromChain != c.ToChain }

type Request struct {
	FromAmount    string
	SlippageBps   int
	Timeout       time.Duration
	FromAmountUsd float64
	ToPriceUsd    float64
	FeeRate       string
	FeeReceiver   string
	ScamCheck     func(chain, token string) bool
}

type Route struct {
	Rank            int     `json:"rank"`
	Recommended     bool    `json:"recommended"`
	CrossChain      bool    `json:"crossChain"`
	FromChain       string  `json:"fromChain"`
	FromToken       string  `json:"fromToken"`
	ToChain         string  `json:"toChain"`
	ToToken         string  `json:"toToken"`
	ToDecimals      int     `json:"toDecimals"`
	SourceIssuer    string  `json:"sourceIssuer,omitempty"`
	ToAmount        string  `json:"toAmount"`
	ToAmountUsd     float64 `json:"toAmountUsd"`
	ExpectedFillUsd float64 `json:"expectedFillUsd,omitempty"`
	Channel         string  `json:"channel"`
	FeeUsd          float64 `json:"feeUsd,omitempty"`
	PriceImpactPct  float64 `json:"priceImpactPct,omitempty"`
	LiquidityUsd    float64 `json:"liquidityUsd,omitempty"`
	EstSeconds      int64   `json:"estSeconds,omitempty"`
	Warnings        []Warning `json:"warnings,omitempty"`
	Reason          string  `json:"reason,omitempty"`
}

type Warning = dexmodel.DexWarning

type Response struct {
	Routes        []*Route `json:"routes"`
	Reason        string   `json:"reason,omitempty"`
	ReasonMessage string   `json:"reasonMessage,omitempty"`
}

type QuoteFn func(ctx context.Context, cand *Candidate, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error)

type Engine struct {
	Logf func(format string, args ...interface{})
}

func (e *Engine) Quote(ctx context.Context, req *Request, cands []*Candidate, quote QuoteFn) *Response {
	if req == nil || quote == nil {
		return &Response{Reason: "INVALID_REQUEST"}
	}
	resp := &Response{Routes: []*Route{}}
	if len(cands) == 0 {
		resp.Reason = "NO_CANDIDATE"
		return resp
	}
	if _, ok := new(big.Int).SetString(req.FromAmount, 10); !ok {
		resp.Reason = "INVALID_AMOUNT"
		return resp
	}

	slippagePct := "1"
	if req.SlippageBps > 0 {
		slippagePct = strconv.FormatFloat(float64(req.SlippageBps)/100.0, 'f', -1, 64)
	}
	feeRate := req.FeeRate
	if feeRate == "" {
		feeRate = "0"
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		cand *Candidate
		out  *dexmodel.DexQuoteOut
		err  error
	}
	results := make([]result, len(cands))
	var wg sync.WaitGroup
	for i, c := range cands {
		wg.Add(1)
		go func(i int, c *Candidate) {
			defer wg.Done()
			if req.ScamCheck != nil && req.ScamCheck(c.ToChain, c.ToToken) {
				results[i] = result{c, nil, fmt.Errorf("scam-filtered: %s/%s", c.ToChain, c.ToToken)}
				return
			}
			in := &dexmodel.DexQuoteIn{
				FromChain:   c.FromChain,
				ToChain:     c.ToChain,
				FromToken:   c.FromToken,
				ToToken:     c.ToToken,
				FromAddress: c.FromAddress,
				ToAddress:   c.ToAddress,
				FromAmount:  req.FromAmount,
				Slippage:    slippagePct,
				FeeRate:     feeRate,
				FeeReceiver: req.FeeReceiver,
			}
			out, err := quote(qctx, c, in)
			results[i] = result{c, out, err}
			if err != nil && e.Logf != nil {
				e.Logf("[trade-engine] %s→%s (%s/%s): %v", c.FromChain, c.ToChain, c.Channel, c.ToToken, err)
			}
		}(i, c)
	}
	wg.Wait()

	out := make([]*Route, 0, len(results))
	for _, r := range results {
		if r.err != nil || r.out == nil || len(r.out.Routes) == 0 {
			continue
		}
		best := r.out.Routes[0]
		toAmtUsdGross := RawToHuman(best.AmountOut, r.cand.ToDecimals) * req.ToPriceUsd

		expectedFillUsd := 0.0
		if best.PriceImpactPct > 0 {
			expectedFillUsd = toAmtUsdGross * (1 - best.PriceImpactPct/100.0)
		} else if req.FromAmountUsd > 0 && r.cand.LiquidityUsd > 0 {
			ratio := r.cand.LiquidityUsd / (r.cand.LiquidityUsd + req.FromAmountUsd)
			expectedFillUsd = toAmtUsdGross * ratio
		}

		feeUsd, _ := strconv.ParseFloat(best.FeeInUsd, 64)
		channel := r.out.Channel
		if channel == "" {
			channel = r.cand.Channel
		}
		route := &Route{
			CrossChain:      r.cand.CrossChain(),
			FromChain:       r.cand.FromChain,
			FromToken:       r.cand.FromToken,
			ToChain:         r.cand.ToChain,
			ToToken:         r.cand.ToToken,
			ToDecimals:      r.cand.ToDecimals,
			SourceIssuer:    r.cand.SourceIssuer,
			ToAmount:        best.AmountOut,
			ToAmountUsd:     toAmtUsdGross,
			ExpectedFillUsd: expectedFillUsd,
			Channel:         channel,
			FeeUsd:          feeUsd,
			PriceImpactPct:  best.PriceImpactPct,
			LiquidityUsd:    r.cand.LiquidityUsd,
			EstSeconds:      best.EstimatedTime,
			Warnings:        append([]Warning(nil), best.Warnings...),
			Reason:          ReasonFor(r.cand, channel),
		}
		if req.FromAmountUsd > 0 && route.LiquidityUsd > 0 && req.FromAmountUsd > route.LiquidityUsd*ThinLiquidityRatio {
			route.Warnings = append(route.Warnings, Warning{
				Code: WarnThinLiquidity,
				Message: fmt.Sprintf("this trade is %.0f%% of the pool ($%.0f of $%.0f) — real fill will be much worse than quoted",
					req.FromAmountUsd/route.LiquidityUsd*100, req.FromAmountUsd, route.LiquidityUsd),
			})
		}
		out = append(out, route)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if IsRouteUnsafe(out[i]) != IsRouteUnsafe(out[j]) {
			return !IsRouteUnsafe(out[i])
		}
		iKey := out[i].ExpectedFillUsd
		if iKey == 0 {
			iKey = out[i].ToAmountUsd
		}
		jKey := out[j].ExpectedFillUsd
		if jKey == 0 {
			jKey = out[j].ToAmountUsd
		}
		if iKey != jKey {
			return iKey > jKey
		}
		return !out[i].CrossChain && out[j].CrossChain
	})
	for i, r := range out {
		r.Rank = i + 1
		r.Recommended = i == 0 && !IsRouteUnsafe(r)
	}

	resp.Routes = out
	if len(out) == 0 {
		resp.Reason = "NO_LIQUIDITY_AT_SIZE"
		resp.ReasonMessage = "no route filled at this amount — try a smaller size"
	}
	return resp
}

func IsRouteUnsafe(r *Route) bool {
	if r == nil {
		return false
	}
	for _, w := range r.Warnings {
		if w.Code == WarnPriceImpactHigh || w.Code == WarnThinLiquidity {
			return true
		}
	}
	return false
}

func ReasonFor(c *Candidate, channel string) string {
	if c == nil {
		return ""
	}
	if c.CrossChain() {
		return fmt.Sprintf("cross-chain via %s (%s pool)", channel, c.ToChain)
	}
	if c.LiquidityUsd > 0 {
		return fmt.Sprintf("same-chain, liq %s", FormatUSD(c.LiquidityUsd))
	}
	return "same-chain"
}

func RawToHuman(raw string, decimals int) float64 {
	if raw == "" || decimals <= 0 {
		return 0
	}
	r, ok := new(big.Rat).SetString(raw)
	if !ok {
		return 0
	}
	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	r.Quo(r, new(big.Rat).SetInt(div))
	f, _ := r.Float64()
	return f
}

func FormatUSD(v float64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("$%.1fM", v/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("$%.1fk", v/1_000)
	default:
		return fmt.Sprintf("$%.0f", v)
	}
}
