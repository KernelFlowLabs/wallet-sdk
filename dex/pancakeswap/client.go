// Package pancakeswap wraps the PancakeSwap V3 subgraph on BNB Chain.
// Same role as the uniswap client but for BSC sources — fills
// priceUsd / liquidity / primaryPoolAddr on each AssetSource issued on
// BNB. PancakeSwap V3's subgraph forks Uniswap V3's so the GraphQL
// schema lines up; only derivedETH → derivedBNB differs.
//
// As of 2026 PancakeSwap publishes the V3 BSC subgraph at
//
//	https://api.studio.thegraph.com/query/45376/pancakeswap-v3-bsc/version/latest
//
// — open access, no key. The Graph gateway also serves it behind a
// key; either URL is valid (pass via NewClient).
package pancakeswap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kernelflowlabs/wallet-sdk/common/httpc"
)

const DefaultSubgraphURL = "https://api.studio.thegraph.com/query/45376/pancakeswap-v3-bsc/version/latest"

type Client struct {
	subgraphURL string
	http        *httpc.Request
}

func NewClient(subgraphURL string) *Client {
	if subgraphURL == "" {
		subgraphURL = DefaultSubgraphURL
	}
	return &Client{
		subgraphURL: subgraphURL,
		http: httpc.NewRequest("", map[string]string{
			"Content-Type": "application/json",
		}),
	}
}

// GetTokenInfo mirrors uniswap.GetTokenInfo on BSC. Address must be
// lowercased hex. Returns (nil, nil) when the token isn't indexed.
func (c *Client) GetTokenInfo(ctx context.Context, contractAddr string) (*TokenInfo, error) {
	query := `query($id:ID!){
		token(id:$id){
			id symbol decimals derivedBNB totalValueLockedUSD volumeUSD
			whitelistPools(first:5, orderBy:totalValueLockedUSD, orderDirection:desc){
				id feeTier liquidity totalValueLockedUSD
				token0{id symbol}
				token1{id symbol}
			}
		}
	}`
	type respData struct {
		Token *TokenInfo `json:"token"`
	}
	payload, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]any{"id": contractAddr},
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := c.http.PostWithOutEncoded(ctx, &raw, c.subgraphURL, payload); err != nil {
		return nil, fmt.Errorf("pancakeswap subgraph post: %w", err)
	}
	var out graphResp[respData]
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("pancakeswap decode: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("pancakeswap graphql: %s", out.Errors[0].Message)
	}
	return out.Data.Token, nil
}
