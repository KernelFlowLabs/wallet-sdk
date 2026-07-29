package univ3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// UniswapV3 router addresses on Arbitrum One (chain ID 42161)
const (
	AddrSwapRouterV1 = "0xE592427A0AEce92De3Edee1F18E0157C05861564" // SwapRouter (has deadline)
	AddrSwapRouter02 = "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45" // SwapRouter02 (no deadline)
)

// 4-byte function selectors (hex, no 0x prefix)
const (
	SelectorExactInputSingleV1 = "414bf389" // exactInputSingle – SwapRouter v1
	SelectorExactInputSingleV2 = "04e45aaf" // exactInputSingle – SwapRouter02
	SelectorMulticall          = "ac9650d8" // multicall(bytes[])
	SelectorMulticallDeadline  = "5ae401dc" // multicall(uint256,bytes[]) – SwapRouter02
)

// SwapRouter v1 ABI – exactInputSingle includes deadline
const abiSwapRouterV1 = `[
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
  {"inputs":[{"internalType":"bytes[]","name":"data","type":"bytes[]"}],
  "name":"multicall","outputs":[{"internalType":"bytes[]","name":"results","type":"bytes[]"}],
  "stateMutability":"payable","type":"function"}
]`

// SwapRouter02 ABI – exactInputSingle has no deadline field
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

// SwapInfo holds decoded parameters from a UniV3 exactInputSingle call.
type SwapInfo struct {
	TokenIn           common.Address
	TokenOut          common.Address
	Fee               uint32 // uint24 in Solidity; go-ethereum maps uint24 → uint32
	Recipient         common.Address
	Deadline          *big.Int // nil for SwapRouter02 (no deadline field)
	AmountIn          *big.Int
	AmountOutMinimum  *big.Int
	SqrtPriceLimitX96 *big.Int
	RouterVersion     int // 1 = SwapRouterV1, 2 = SwapRouter02
}
