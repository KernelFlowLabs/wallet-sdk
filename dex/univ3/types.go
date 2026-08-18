package univ3

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	RouterVersionV1 = 1
	RouterVersion02 = 2
	MaxUint24       = uint32(1<<24 - 1)
	Channel         = "univ3"

	DefaultDeadlineTTL = 20 * time.Minute
)

type Config struct {
	ChainName       string        `json:"chain_name"`
	ChainID         uint64        `json:"chain_id"`
	RPC             string        `json:"rpc"`
	Factory         string        `json:"factory"`
	SwapRouter      string        `json:"swap_router"`
	Quoter          string        `json:"quoter"`
	WrappedNative   string        `json:"wrapped_native"`
	FeeTiers        []uint32      `json:"fee_tiers,omitempty"`
	QuoteBaseTokens []string      `json:"quote_base_tokens,omitempty"`
	DeadlineTTL     time.Duration `json:"deadline_ttl,omitempty"`
}

const (
	AddrSwapRouterV1 = "0xE592427A0AEce92De3Edee1F18E0157C05861564" // SwapRouter (deadline in params)
	AddrSwapRouter02 = "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45" // SwapRouter02 (deadline via multicall)
)

const (
	SelectorExactInputSingleV1 = "414bf389"
	SelectorExactInputV1       = "c04b8d59"
	SelectorExactInputSingleV2 = "04e45aaf"
	SelectorMulticall          = "ac9650d8" // multicall(bytes[])
	SelectorMulticallDeadline  = "5ae401dc" // multicall(uint256,bytes[])
	SelectorMulticallBlockhash = "1f0464d1" // multicall(bytes32,bytes[])
)

const abiSwapRouterV1 = `[
	{"inputs":[],"name":"factory","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"WETH9","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"components":[
    {"internalType":"address","name":"tokenIn","type":"address"},
    {"internalType":"address","name":"tokenOut","type":"address"},
    {"internalType":"uint24","name":"fee","type":"uint24"},
    {"internalType":"address","name":"recipient","type":"address"},
    {"internalType":"uint256","name":"deadline","type":"uint256"},
    {"internalType":"uint256","name":"amountIn","type":"uint256"},
    {"internalType":"uint256","name":"amountOutMinimum","type":"uint256"},
    {"internalType":"uint160","name":"sqrtPriceLimitX96","type":"uint160"}
  ],"internalType":"struct ISwapRouter.ExactInputSingleParams","name":"params","type":"tuple"}],
  "name":"exactInputSingle","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],
  "stateMutability":"payable","type":"function"},
	{"inputs":[{"components":[
		{"internalType":"bytes","name":"path","type":"bytes"},
		{"internalType":"address","name":"recipient","type":"address"},
		{"internalType":"uint256","name":"deadline","type":"uint256"},
		{"internalType":"uint256","name":"amountIn","type":"uint256"},
		{"internalType":"uint256","name":"amountOutMinimum","type":"uint256"}
	],"internalType":"struct ISwapRouter.ExactInputParams","name":"params","type":"tuple"}],
	"name":"exactInput","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],
	"stateMutability":"payable","type":"function"},
  {"inputs":[{"internalType":"bytes[]","name":"data","type":"bytes[]"}],
  "name":"multicall","outputs":[{"internalType":"bytes[]","name":"results","type":"bytes[]"}],
  "stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"amountMinimum","type":"uint256"},{"internalType":"address","name":"recipient","type":"address"}],"name":"unwrapWETH9","outputs":[],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amountMinimum","type":"uint256"},{"internalType":"address","name":"recipient","type":"address"}],"name":"sweepToken","outputs":[],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"amountMinimum","type":"uint256"},{"internalType":"address","name":"recipient","type":"address"},{"internalType":"uint256","name":"feeBips","type":"uint256"},{"internalType":"address","name":"feeRecipient","type":"address"}],"name":"unwrapWETH9WithFee","outputs":[],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amountMinimum","type":"uint256"},{"internalType":"address","name":"recipient","type":"address"},{"internalType":"uint256","name":"feeBips","type":"uint256"},{"internalType":"address","name":"feeRecipient","type":"address"}],"name":"sweepTokenWithFee","outputs":[],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"uint256","name":"deadline","type":"uint256"},{"internalType":"uint8","name":"v","type":"uint8"},{"internalType":"bytes32","name":"r","type":"bytes32"},{"internalType":"bytes32","name":"s","type":"bytes32"}],"name":"selfPermit","outputs":[],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"uint256","name":"deadline","type":"uint256"},{"internalType":"uint8","name":"v","type":"uint8"},{"internalType":"bytes32","name":"r","type":"bytes32"},{"internalType":"bytes32","name":"s","type":"bytes32"}],"name":"selfPermitIfNecessary","outputs":[],"stateMutability":"payable","type":"function"}
]`

const abiFactoryV3 = `[
	{"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"}],"name":"getPool","outputs":[{"internalType":"address","name":"pool","type":"address"}],"stateMutability":"view","type":"function"}
]`

const abiQuoterV1 = `[
	{"inputs":[],"name":"factory","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"WETH9","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"bytes","name":"path","type":"bytes"},{"internalType":"uint256","name":"amountIn","type":"uint256"}],"name":"quoteExactInput","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"},{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint160","name":"sqrtPriceLimitX96","type":"uint160"}],"name":"quoteExactInputSingle","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],"stateMutability":"nonpayable","type":"function"}
]`

const abiSwapRouter02 = `[
  {"inputs":[{"components":[
    {"internalType":"address","name":"tokenIn","type":"address"},
    {"internalType":"address","name":"tokenOut","type":"address"},
    {"internalType":"uint24","name":"fee","type":"uint24"},
    {"internalType":"address","name":"recipient","type":"address"},
    {"internalType":"uint256","name":"amountIn","type":"uint256"},
    {"internalType":"uint256","name":"amountOutMinimum","type":"uint256"},
    {"internalType":"uint160","name":"sqrtPriceLimitX96","type":"uint160"}
  ],"internalType":"struct IV3SwapRouter.ExactInputSingleParams","name":"params","type":"tuple"}],
  "name":"exactInputSingle","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],
  "stateMutability":"payable","type":"function"},
  {"inputs":[{"internalType":"uint256","name":"deadline","type":"uint256"},{"internalType":"bytes[]","name":"data","type":"bytes[]"}],
  "name":"multicall","outputs":[{"internalType":"bytes[]","name":"results","type":"bytes[]"}],
  "stateMutability":"payable","type":"function"}
]`

const abiMulticallBlockhash = `[
  {"inputs":[{"internalType":"bytes32","name":"previousBlockhash","type":"bytes32"},{"internalType":"bytes[]","name":"data","type":"bytes[]"}],
  "name":"multicall","outputs":[{"internalType":"bytes[]","name":"results","type":"bytes[]"}],
  "stateMutability":"payable","type":"function"}
]`

type SwapInfo struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               uint32
	Recipient         common.Address
	Deadline          *big.Int
	PreviousBlockhash *common.Hash
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
	RouterVersion     int
	Path              []byte
	Tokens            []common.Address
	Fees              []uint32
}
