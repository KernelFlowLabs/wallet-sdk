package okx

import (
	dexmodel "github.com/kernelflowlabs/wallet-sdk/common/dexmodel"
	"github.com/kernelflowlabs/wallet-sdk/common/httpc"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"
)

type Client struct {
	client     *httpc.Request
	apiKey     string
	apiSecret  string
	passphrase string
	mu         sync.Mutex
}

func NewClient(apiKey, apiSecret, passphrase string) *Client {
	return &Client{
		client: httpc.NewRequest("https://web3.okx.com",
			map[string]string{
				"Content-Type": "application/json",
			}),
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		passphrase: passphrase,
	}
}

func (c *Client) SupportedChains(ctx context.Context) ([]string, error) {
	path := "api/v6/dex/aggregator/supported/chain"
	query := url.Values{}

	out := &SupportedChainsRes{}
	err := c.signedGet(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to get supported chains, err=%v", err)
	} else if !out.IsSuccess() {
		return nil, fmt.Errorf("fail to get supported chains, code=%s, msg=%s", out.Code, out.Msg)
	}

	var res []string
	for _, v := range out.Data {
		res = append(res, v.ChainName)
	}
	return res, nil
}

func (c *Client) SupportedCrossChains(ctx context.Context) ([]string, error) {
	path := "api/v5/dex/cross-chain/supported/chain"
	query := url.Values{}

	out := &SupportedChainsRes{}
	err := c.signedGet(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to get supported chains, err=%v", err)
	} else if !out.IsSuccess() {
		return nil, fmt.Errorf("fail to get supported chains, code=%s, msg=%s", out.Code, out.Msg)
	}

	var res []string
	for _, v := range out.Data {
		res = append(res, v.ChainName)
	}
	return res, nil
}

func (c *Client) PreSingleChainQuote(ctx context.Context, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	req := c.toOkxSingleChainQuoteReq(in)
	if req == nil {
		return nil, fmt.Errorf("fail to convert request to QuoteReq")
	}

	path := "api/v6/dex/aggregator/quote"
	query := url.Values{}
	query.Set("chainIndex", req.ChainIndex)
	query.Set("fromTokenAddress", req.FromToken)
	query.Set("toTokenAddress", req.ToToken)
	query.Set("amount", req.Amount)
	query.Set("slippagePercent", req.Slippage)

	out := &SingleChainQuoteRes{}
	err := c.signedGet(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to quote, err=%v", err)
	} else if !out.IsSuccess() {
		return nil, fmt.Errorf("fail to quote, code=%v, msg=%s", out.Code, out.Msg)
	}
	res := c.toStandardQuoteResPre(out, in)
	if res == nil {
		return nil, fmt.Errorf("fail to convert response")
	}
	return res, nil
}

func (c *Client) Quote(ctx context.Context, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	if in.FromChain == in.ToChain {
		return c.singleChainQuote(ctx, in)
	}
	return c.crossChainQuote(ctx, in)
}

func (c *Client) singleChainQuote(ctx context.Context, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	req := c.toOkxSingleChainQuoteReq(in)
	if req == nil {
		return nil, fmt.Errorf("fail to convert request to QuoteReq")
	}

	path := "api/v6/dex/aggregator/swap"
	query := url.Values{}
	query.Set("chainIndex", req.ChainIndex)
	query.Set("fromTokenAddress", req.FromToken)
	query.Set("toTokenAddress", req.ToToken)
	query.Set("amount", req.Amount)
	query.Set("slippagePercent", req.Slippage)
	query.Set("userWalletAddress", req.UserAddress)
	query.Set("approveTransaction", "true")
	query.Set("approveAmount", req.Amount)
	query.Set("singleRouteOnly", "true")
	//query.Set("fromTokenReferrerWalletAddress", req.ReferrerAddress)
	query.Set("toTokenReferrerWalletAddress", req.ReferrerAddress)
	query.Set("feePercent", req.FeePercent)

	out := &SingleChainSwapRes{}
	err := c.signedGet(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to swap, err=%v", err)
	} else if !out.IsSuccess() {
		return nil, fmt.Errorf("fail to swap, code=%s, msg=%s", out.Code, out.Msg)
	}

	data, err := c.parseSwapResData(out.Data)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("fail to swap, no data")
	}
	res := c.toStandardQuoteResFromSwapResData(data[0], in)
	if res == nil {
		return nil, fmt.Errorf("fail to convert response")
	}
	return res, nil
}

func (c *Client) crossChainQuote(ctx context.Context, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	req := c.toOkxCrossChainQuoteReq(in)
	if req == nil {
		return nil, fmt.Errorf("fail to convert request to QuoteReq")
	}

	path := "api/v5/dex/cross-chain/build-tx"
	query := url.Values{}
	query.Set("fromChainId", req.FromChainIndex)
	query.Set("toChainId", req.ToChainIndex)
	query.Set("fromTokenAddress", req.FromToken)
	query.Set("toTokenAddress", req.ToToken)
	query.Set("userWalletAddress", req.UserAddress)
	query.Set("receiveAddress", req.ReceiverAddress)
	query.Set("amount", req.Amount)
	query.Set("slippage", req.Slippage)
	query.Set("referrerAddress", req.ReferrerAddress)
	query.Set("feePercent", req.FeePercent)

	out := &CrossChainQuoteRes{}
	err := c.signedGet(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to swap, err=%v", err)
	} else if !out.IsSuccess() {
		return nil, fmt.Errorf("fail to swap, code=%s, msg=%s", out.Code, out.Msg)
	}

	data, err := c.parseCrossResData(out.Data)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("fail to swap, no data")
	}
	res := c.toStandardQuoteResFromCrossResData(data[0], in)
	if res == nil {
		return nil, fmt.Errorf("fail to convert response")
	}
	return res, nil
}

func (c *Client) Status(ctx context.Context, in *dexmodel.DexCheckTxIn) (*dexmodel.DexCheckTxOut, error) {
	chainId, ok := idChainMapperReverse[in.FromChain]
	if !ok {
		log.Printf("not found chain id %s", in.FromChain)
		return nil, fmt.Errorf("unsupported chain")
	}

	path := "api/v6/dex/aggregator/history"
	query := url.Values{}
	query.Set("chainIndex", chainId)
	query.Set("txHash", in.Hash)
	//query.Set("isFromMyProject", "true")

	out := &HistoryRes{}
	err := c.signedGet(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to get status, txHash=%s, err=%v", in.Hash, err)
	} else if !out.IsSuccess() {
		return nil, fmt.Errorf("fail to get status, code=%s, msg=%s", out.Code, out.Msg)
	}

	res := c.toStandardStatusRes(out)
	if res == nil {
		return nil, fmt.Errorf("fail to convert response")
	}
	res.Channel = in.Channel
	res.ToHash = in.ToChain
	return res, nil
}

func (c *Client) signedGet(ctx context.Context, res interface{}, path string, query url.Values) error {
	c.mu.Lock()

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	requestPath := path
	if len(query) > 0 {
		requestPath = path + "?" + query.Encode()
	}
	sign := c.hmacSign(timestamp + "GET" + "/" + requestPath)

	c.client.SetHeader("OK-ACCESS-KEY", c.apiKey)
	c.client.SetHeader("OK-ACCESS-SIGN", sign)
	c.client.SetHeader("OK-ACCESS-PASSPHRASE", c.passphrase)
	c.client.SetHeader("OK-ACCESS-TIMESTAMP", timestamp)
	c.mu.Unlock()

	return c.client.Get(ctx, res, path, query)
}

func (c *Client) hmacSign(message string) string {
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
