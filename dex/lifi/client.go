package lifi

import (
	dexmodel "github.com/kernelflowlabs/wallet-sdk/common/dexmodel"
	"github.com/kernelflowlabs/wallet-sdk/common/httpc"
	"context"
	"fmt"
	"net/url"
)

type Client struct {
	client     *httpc.Request
	integrator string
}

func NewClient(opts ...ClientOption) *Client {
	cfg := clientConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	headers := map[string]string{"Content-Type": "application/json"}
	if cfg.apiKey != "" {
		headers["x-lifi-api-key"] = cfg.apiKey
	}
	c := httpc.NewRequest("https://li.quest", headers)
	cfg.rateLimit.Apply(c)
	return &Client{
		client:     c,
		integrator: cfg.integrator,
	}
}

func (c *Client) Chains(ctx context.Context) ([]string, error) {
	path := "v1/chains"

	query := url.Values{}
	query.Set("chainTypes", "EVM,UTXO")

	out := &ChainsRes{}
	err := c.client.Get(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to get chains, err=%v", err)
	}

	var res []string
	for _, v := range out.Chains {
		if chain, ok := ChainMapper[v.Key]; ok {
			res = append(res, chain)
		}
	}
	return res, nil
}

func (c *Client) Tokens(ctx context.Context) error {
	path := "v1/tokens"

	query := url.Values{}
	//query.Set("chainTypes", "EVM,UTXO,SVM,MVM")
	query.Set("chainTypes", "EVM")

	out := &TokensRes{}
	err := c.client.Get(ctx, out, path, query)
	if err != nil {
		return fmt.Errorf("fail to get chains, err=%v", err)
	}
	return nil
}

func (c *Client) Quote(ctx context.Context, in *dexmodel.DexQuoteIn) (*dexmodel.DexQuoteOut, error) {
	req := c.toLifiQuoteReq(in)
	if req == nil {
		return nil, fmt.Errorf("fail to convert request to QuoteReq")
	}

	path := "v1/quote"

	query := url.Values{}
	query.Set("fromChain", req.FromChain)
	query.Set("toChain", req.ToChain)
	query.Set("fromToken", req.FromToken)
	query.Set("toToken", req.ToToken)
	query.Set("fromAddress", req.FromAddress)
	query.Set("toAddress", req.ToAddress)
	query.Set("fromAmount", req.FromAmount)
	query.Set("slippage", req.Slippage)
	if c.integrator != "" {
		query.Set("integrator", c.integrator)
	}
	query.Set("fee", req.Fee)
	if req.FromAmountForGas != "" && req.FromAmountForGas != "0" {
		query.Set("fromAmountForGas", req.FromAmountForGas)
	}

	out := &QuoteRes{}

	err := c.client.Get(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to get quote, from=%s, to=%s, err=%v",
			in.FromChain, in.ToChain, err)
	}
	res := c.toStandardQuoteRes(out)
	if res == nil {
		return nil, fmt.Errorf("fail to convert response")
	}
	return res, nil
}

func (c *Client) Status(ctx context.Context, in *dexmodel.DexCheckTxIn) (*dexmodel.DexCheckTxOut, error) {
	path := "v1/status"

	query := url.Values{}
	query.Set("txHash", in.Hash)
	fromChainId, ok := idChainMapperReverse[in.FromChain]
	if !ok {
		return nil, fmt.Errorf("invalid fromChain")
	}
	query.Set("fromChain", fromChainId)
	toChainId, ok := idChainMapperReverse[in.ToChain]
	if !ok {
		return nil, fmt.Errorf("invalid toChain")
	}
	query.Set("toChain", toChainId)
	out := &StatusRes{}
	err := c.client.Get(ctx, out, path, query)
	if err != nil {
		return nil, fmt.Errorf("fail to get status, txHash=%s, err=%v", in.Hash, err)
	}
	res := c.toStandardStatusRes(out)
	if res == nil {
		return nil, fmt.Errorf("fail to get status, txHash=%s", in.Hash)
	}
	res.ToChain = in.ToChain
	return res, nil
}

//func (c *Client) GetConnections(ctx context.Context, fromChain, toChain string) (*ConnectionsResponse, error) {
//	params := map[string]string{}
//	if fromChain != "" {
//		params["fromChain"] = fromChain
//	}
//	if toChain != "" {
//		params["toChain"] = toChain
//	}
//
//	res := &ConnectionsResponse{}
//	err := c.client.Get(ctx, res, "connections", params)
//	if err != nil {
//		return nil, fmt.Errorf("fail to get connections, from=%s, to=%s, err=%v",
//			fromChain, toChain, err)
//	}
//	return res, nil
//}
