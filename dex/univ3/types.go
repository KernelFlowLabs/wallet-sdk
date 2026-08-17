// Package univ3 safely encodes and decodes legacy Uniswap V3 SwapRouter and
// SwapRouter02 exactInputSingle calldata. It is a calldata codec, not a quote
// engine or a Universal Router implementation.
package univ3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	RouterVersionV1 = 1
	RouterVersion02 = 2
	MaxUint24       = uint32(1<<24 - 1)
)

// Canonical legacy router deployments. These addresses are deployed on
// Arbitrum One and several other EVM networks; callers must still verify the
// target chain before trusting an address.
const (
	AddrSwapRouterV1 = "0xE592427A0AEce92De3Edee1F18E0157C05861564" // SwapRouter (deadline in params)
	AddrSwapRouter02 = "0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45" // SwapRouter02 (deadline via multicall)
)

// 4-byte function selectors (hex, no 0x prefix).
const (
	SelectorExactInputSingleV1 = "414bf389"
	SelectorExactInputSingleV2 = "04e45aaf"
	SelectorMulticall          = "ac9650d8" // multicall(bytes[])
	SelectorMulticallDeadline  = "5ae401dc" // multicall(uint256,bytes[])
	SelectorMulticallBlockhash = "1f0464d1" // multicall(bytes32,bytes[])
)

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

// SwapInfo holds decoded exactInputSingle parameters. Deadline can come from a
// SwapRouter V1 tuple or an outer SwapRouter02 deadline multicall.
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
}
