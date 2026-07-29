package uniswap

// TokenInfo is the subset of Uniswap V3 subgraph's Token type we care
// about. derivedETH is the on-chain price in ETH; multiply by the
// current ETH/USD spot to get a USD figure. totalValueLockedUSD is the
// sum across every whitelisted pool — useful as a "is this trade-able?"
// filter on the catalog.
type TokenInfo struct {
	ID                  string  `json:"id"` // token contract address (lowercased)
	Symbol              string  `json:"symbol"`
	Decimals            string  `json:"decimals"`            // string in GraphQL (it's actually BigInt upstream)
	DerivedETH          string  `json:"derivedETH"`          // string-encoded decimal
	TotalValueLockedUSD string  `json:"totalValueLockedUSD"` // string-encoded decimal
	VolumeUSD           string  `json:"volumeUSD"`           // 24h volume across pools
	WhitelistPools      []*Pool `json:"whitelistPools,omitempty"`
}

// Pool is one Uniswap V3 pool the token participates in. FeeTier is
// expressed in hundredths-of-bps (500 = 0.05%, 3000 = 0.3%, 10000 =
// 1%). TotalValueLockedUSD is the per-pool TVL — pick the deepest pool
// as the trade-routing primary.
type Pool struct {
	ID                  string `json:"id"` // pool contract address
	FeeTier             string `json:"feeTier"`
	Liquidity           string `json:"liquidity"`
	TotalValueLockedUSD string `json:"totalValueLockedUSD"`
	Token0              struct {
		Symbol string `json:"symbol"`
		ID     string `json:"id"`
	} `json:"token0"`
	Token1 struct {
		Symbol string `json:"symbol"`
		ID     string `json:"id"`
	} `json:"token1"`
}

// graphResp is the standard GraphQL envelope; Data is decoded into the
// caller-provided struct.
type graphResp[T any] struct {
	Data   T            `json:"data"`
	Errors []graphError `json:"errors,omitempty"`
}

type graphError struct {
	Message string `json:"message"`
}
