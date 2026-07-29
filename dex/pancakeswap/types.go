package pancakeswap

// TokenInfo mirrors Uniswap's TokenInfo shape — PancakeSwap V3's
// subgraph schema is forked from Uniswap V3 so field semantics are
// identical (derivedBNB instead of derivedETH for the chain-native
// quote, but the rest line up).
type TokenInfo struct {
	ID                  string  `json:"id"`
	Symbol              string  `json:"symbol"`
	Decimals            string  `json:"decimals"`
	DerivedBNB          string  `json:"derivedBNB"`          // string-encoded decimal
	TotalValueLockedUSD string  `json:"totalValueLockedUSD"` // string-encoded decimal
	VolumeUSD           string  `json:"volumeUSD"`
	WhitelistPools      []*Pool `json:"whitelistPools,omitempty"`
}

type Pool struct {
	ID                  string `json:"id"`
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

type graphResp[T any] struct {
	Data   T            `json:"data"`
	Errors []graphError `json:"errors,omitempty"`
}

type graphError struct {
	Message string `json:"message"`
}
