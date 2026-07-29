package bungee

import (
	dexmodel "github.com/kernelflowlabs/wallet-sdk/common/dexmodel"
	"github.com/kernelflowlabs/wallet-sdk/common/httpc"
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Client speaks the Socket Swap V3 API (the successor of Bungee v1;
// V3 is tx-mode only — no intent submit step, routes carry txData).
type Client struct {
	client *httpc.Request
}

func NewClient(opts ...ClientOption) *Client {
	cfg := clientConfig{rateLimit: RateLimitPublic}
	for _, opt := range opts {
		opt(&cfg)
	}
	// Keyed access goes through the dedicated backend.
	base := "https://public-backend.socket.tech"
	if cfg.apiKey != "" {
		base = "https://dedicated-backend.socket.tech"
	}
	c := httpc.NewRequest(base,
		map[string]string{"Content-Type": "application/json"})
	if cfg.apiKey != "" {
		c.SetHeader("x-api-key", cfg.apiKey)
	}
	if cfg.affiliateId != "" {
		c.SetHeader("affiliate", cfg.affiliateId)
	}
	cfg.rateLimit.Apply(c)
	return &Client{client: c}
}

// GetSupportedChainIds returns the SDK's static chain map; Socket V3
// has no supported-chains endpoint.
func (c *Client) GetSupportedChainIds(ctx context.Context) ([]string, error) {
	ids := make([]string, 0, len(idChainMapper))
	for id := range idChainMapper {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func (c *Client) Quote(ctx context.Context, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	req := c.toSocketQuoteReq(in)
	if req == nil {
		return nil, fmt.Errorf("fail to convert request to QuoteReq")
	}

	path := "v3/swap/quote"
	query := url.Values{}
	query.Set("userOps", "tx")
	query.Set("originChainId", req.OriginChainId)
	query.Set("destinationChainId", req.DestinationChainId)
	query.Set("inputToken", req.InputToken)
	query.Set("outputToken", req.OutputToken)
	query.Set("userAddress", req.UserAddress)
	query.Set("receiverAddress", req.ReceiverAddress)
	query.Set("inputAmount", req.InputAmount)
	if req.Slippage != "" {
		query.Set("slippage", req.Slippage)
	}

	out := &QuoteResponse{}
	err := c.client.Get(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to quote, err=%v", err)
	} else if !out.Success {
		return nil, fmt.Errorf("fail to quote, statusCode=%d, msg=%s", out.StatusCode, out.Message)
	}
	res := c.toStandardQuoteRes(&out.Result)
	if res == nil {
		return nil, fmt.Errorf("fail to convert response")
	}
	return res, nil
}

// Status polls /v3/swap/status; in.Hash must carry the route quoteId.
func (c *Client) Status(ctx context.Context, in *dexmodel.DexCheckTxIn) (*dexmodel.DexCheckTxOut, error) {
	if in == nil {
		return nil, fmt.Errorf("in is nil")
	} else if in.Hash == "" {
		return nil, fmt.Errorf("hash (quoteId) is empty")
	}

	path := "v3/swap/status"
	query := url.Values{}
	query.Set("quoteId", in.Hash)

	out := &dexmodel.DexCheckTxOut{}
	res := &StatusResponse{}
	err := c.client.Get(ctx, res, path, query)
	if err != nil {
		// Unknown quoteId is an HTTP 404 — not-found, not an error.
		if strings.Contains(err.Error(), "status=404") {
			out.Status = dexmodel.DexStatusNotFound
			return out, nil
		}
		return nil, fmt.Errorf("fail to get status, quoteId=%s, err=%v", in.Hash, err)
	} else if !res.Success {
		if res.StatusCode == 404 {
			out.Status = dexmodel.DexStatusNotFound
			return out, nil
		}
		return nil, fmt.Errorf("fail to get status, quoteId=%s, statusCode=%d, msg=%s", in.Hash, res.StatusCode, res.Message)
	}

	switch res.Result.StatusCode {
	case "COMPLETED":
		out.Status = dexmodel.DexStatusSucceeded
		out.ToChain = in.ToChain
		out.ToHash = res.Result.Destination.TxHash
	case "REFUNDED":
		out.Status = dexmodel.DexStatusRefunded
		out.ToChain = in.ToChain
		if res.Result.Refund != nil && res.Result.Refund.TxHash != "" {
			out.ToHash = res.Result.Refund.TxHash
		} else {
			out.ToHash = res.Result.Destination.TxHash
		}
	case "FAILED", "EXPIRED":
		out.Status = dexmodel.DexStatusFailed
	default:
		// PENDING / IN_PROGRESS and anything new keep polling.
		out.Status = dexmodel.DexStatusPending
	}
	return out, nil
}
