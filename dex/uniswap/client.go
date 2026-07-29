// Package uniswap wraps the Uniswap V3 subgraph on Ethereum mainnet.
// Used by the EVM source resolver to fill priceUsd / liquidity /
// primaryPoolAddr on each AssetSource issued on ETH. The subgraph is
// GraphQL-only; this client speaks the minimum subset of the schema we
// actually consume (Token + whitelistPools).
//
// SubgraphURL is injectable so callers can swap between The Graph's
// hosted service and the decentralized network without changing call
// sites. As of 2026 the canonical mainnet V3 subgraph is published at
//
//	https://gateway.thegraph.com/api/<API_KEY>/subgraphs/id/5zvR82QoaXYFyDEKLZ9t6v9adgnptxYpKpSbxtgVENFV
//
// — the gateway requires an API key. For local/CI dev a no-key fallback
// is the community mirror at
//
//	https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3
//
// which still serves traffic but isn't guaranteed past 2026.
package uniswap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/kernelflowlabs/wallet-sdk/common/httpc"
)

const DefaultSubgraphURL = "https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3"

type Client struct {
	subgraphURL string
	http        *httpc.Request
}

// NewClient builds a subgraph-query client. Pass "" to use the public
// hosted-service URL (no API key). When/if that mirror disappears,
// callers move to The Graph gateway URL with a key baked in.
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

// GetTokenInfo loads the Token row + whitelisted pools for one
// contract. Address must be lowercased (subgraph IDs are case-sensitive
// lowercased hex). Returns (nil, nil) when the token isn't indexed —
// trade routing should fall through to LiFi/1inch in that case.
func (c *Client) GetTokenInfo(ctx context.Context, contractAddr string) (*TokenInfo, error) {
	query := `query($id:ID!){
		token(id:$id){
			id symbol decimals derivedETH totalValueLockedUSD volumeUSD
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
	resp, err := c.query(ctx, query, map[string]any{"id": contractAddr})
	if err != nil {
		return nil, err
	}
	var out graphResp[respData]
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, fmt.Errorf("uniswap decode: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("uniswap graphql: %s", out.Errors[0].Message)
	}
	return out.Data.Token, nil
}

// query issues a raw GraphQL request and returns the body bytes. The
// caller is responsible for shape-decoding via graphResp[T].
func (c *Client) query(ctx context.Context, q string, vars map[string]any) ([]byte, error) {
	payload, err := json.Marshal(map[string]any{
		"query":     q,
		"variables": vars,
	})
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	if err := c.http.GetRaw(ctx, buf, "", nil); err == nil && false {
		// no-op — placeholder; we actually POST below
	}
	// httpc.Request lacks a generic POST-raw-bytes helper that bypasses
	// the JSON-decode round-trip, so we use PostWithOutEncoded which
	// sends a []byte body untouched. The subgraph echoes JSON which we
	// parse on the caller side.
	var raw json.RawMessage
	if err := c.http.PostWithOutEncoded(ctx, &raw, c.subgraphURL, payload); err != nil {
		return nil, fmt.Errorf("uniswap subgraph post: %w", err)
	}
	_ = buf
	return raw, nil
}
