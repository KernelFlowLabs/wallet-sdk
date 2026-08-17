package pancakeswap

// Config selects the PancakeSwap V3 subgraph endpoint. The official Graph
// Network endpoint requires an API key. A custom endpoint (for example, a
// private indexer) may omit APIKey.
type Config struct {
	SubgraphURL string `json:"subgraph_url,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
}

// TokenInfo contains the PancakeSwap V3 token metadata used by the SDK. The
// subgraph schema keeps the historical
// derivedETH JSON field name on BNB Chain; DerivedBNB maps that field to its
// actual chain-native meaning.
type TokenInfo struct {
	ID                  string  `json:"id"`
	Symbol              string  `json:"symbol"`
	Decimals            string  `json:"decimals"`
	DerivedBNB          string  `json:"derivedETH"`          // BNB price; schema retains the derivedETH name
	DerivedUSD          string  `json:"derivedUSD"`          // string-encoded decimal
	TotalValueLockedUSD string  `json:"totalValueLockedUSD"` // string-encoded decimal
	VolumeUSD           string  `json:"volumeUSD"`           // cumulative, not 24-hour volume
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
	Data    T            `json:"data"`
	Errors  []graphError `json:"errors,omitempty"`
	Message string       `json:"message,omitempty"`
}

type graphError struct {
	Message string `json:"message"`
}
