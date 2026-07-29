package univ3

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var (
	parsedV1 abi.ABI
	parsedV2 abi.ABI
)

func init() {
	var err error
	parsedV1, err = abi.JSON(strings.NewReader(abiSwapRouterV1))
	if err != nil {
		panic(fmt.Sprintf("univ3: failed to parse SwapRouter v1 ABI: %v", err))
	}
	parsedV2, err = abi.JSON(strings.NewReader(abiSwapRouter02))
	if err != nil {
		panic(fmt.Sprintf("univ3: failed to parse SwapRouter02 ABI: %v", err))
	}
}

// IsUniV3Router returns (routerVersion, true) when addr is a known UniV3 router.
func IsUniV3Router(addr string) (int, bool) {
	lower := strings.ToLower(addr)
	switch lower {
	case strings.ToLower(AddrSwapRouterV1):
		return 1, true
	case strings.ToLower(AddrSwapRouter02):
		return 2, true
	}
	return 0, false
}

// DecodeSwapInfo decodes a UniV3 exactInputSingle calldata (direct call or inside multicall).
// routerVer is 1 or 2 and is used only when the selector is multicall.
func DecodeSwapInfo(calldata []byte, routerVer int) (*SwapInfo, error) {
	if len(calldata) < 4 {
		return nil, fmt.Errorf("calldata too short (%d bytes)", len(calldata))
	}
	sel := hex.EncodeToString(calldata[:4])
	switch sel {
	case SelectorExactInputSingleV1:
		return decodeV1(calldata)
	case SelectorExactInputSingleV2:
		return decodeV2(calldata)
	case SelectorMulticall, SelectorMulticallDeadline:
		return decodeMulticall(calldata, routerVer)
	default:
		return nil, fmt.Errorf("unsupported selector: %s", sel)
	}
}

// EncodeExactInputSingle builds calldata for exactInputSingle on the given router version.
// For V1, deadline defaults to now+60s when nil.
func EncodeExactInputSingle(info *SwapInfo, recipient common.Address, deadline *big.Int, routerVer int) ([]byte, error) {
	if routerVer == 2 {
		return encodeV2(info, recipient)
	}
	return encodeV1(info, recipient, deadline)
}

// ---------- decode helpers ----------

func decodeV1(calldata []byte) (*SwapInfo, error) {
	method := parsedV1.Methods["exactInputSingle"]
	vals, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, fmt.Errorf("unpack v1 exactInputSingle: %w", err)
	}
	if len(vals) != 1 {
		return nil, fmt.Errorf("unexpected arg count %d", len(vals))
	}
	info, err := extractSwapFields(vals[0], true)
	if err != nil {
		return nil, err
	}
	info.RouterVersion = 1
	return info, nil
}

func decodeV2(calldata []byte) (*SwapInfo, error) {
	method := parsedV2.Methods["exactInputSingle"]
	vals, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, fmt.Errorf("unpack v2 exactInputSingle: %w", err)
	}
	if len(vals) != 1 {
		return nil, fmt.Errorf("unexpected arg count %d", len(vals))
	}
	info, err := extractSwapFields(vals[0], false)
	if err != nil {
		return nil, err
	}
	info.RouterVersion = 2
	return info, nil
}

func decodeMulticall(calldata []byte, routerVer int) (*SwapInfo, error) {
	sel := hex.EncodeToString(calldata[:4])
	var innerCalls [][]byte

	if sel == SelectorMulticall {
		// multicall(bytes[]) – exists on both V1 and V2; V1 ABI is fine to parse either
		method := parsedV1.Methods["multicall"]
		vals, err := method.Inputs.Unpack(calldata[4:])
		if err != nil {
			return nil, fmt.Errorf("unpack multicall(bytes[]): %w", err)
		}
		raw, ok := vals[0].([][]byte)
		if !ok {
			return nil, fmt.Errorf("multicall: unexpected data type %T", vals[0])
		}
		innerCalls = raw
	} else {
		// multicall(uint256 deadline, bytes[]) – SwapRouter02 only
		method := parsedV2.Methods["multicall"]
		vals, err := method.Inputs.Unpack(calldata[4:])
		if err != nil {
			return nil, fmt.Errorf("unpack multicall(uint256,bytes[]): %w", err)
		}
		if len(vals) < 2 {
			return nil, fmt.Errorf("multicall: too few args (%d)", len(vals))
		}
		raw, ok := vals[1].([][]byte)
		if !ok {
			return nil, fmt.Errorf("multicall: unexpected data type %T", vals[1])
		}
		innerCalls = raw
	}

	for _, inner := range innerCalls {
		info, err := DecodeSwapInfo(inner, routerVer)
		if err == nil {
			return info, nil
		}
	}
	return nil, fmt.Errorf("no exactInputSingle found in multicall (%d calls)", len(innerCalls))
}

// extractSwapFields reads the ABI-unpacked tuple struct via reflection.
// go-ethereum creates an anonymous struct whose fields are PascalCase versions of the ABI component names.
// uint24 (fee) maps to uint32; uint160 (sqrtPriceLimitX96) maps to *big.Int.
func extractSwapFields(v interface{}, hasDeadline bool) (*SwapInfo, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", v)
	}

	addr := func(name string) (common.Address, error) {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			return common.Address{}, fmt.Errorf("field %s not found", name)
		}
		a, ok := f.Interface().(common.Address)
		if !ok {
			return common.Address{}, fmt.Errorf("field %s: expected common.Address, got %T", name, f.Interface())
		}
		return a, nil
	}

	bigint := func(name string) (*big.Int, error) {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			return nil, fmt.Errorf("field %s not found", name)
		}
		n, ok := f.Interface().(*big.Int)
		if !ok {
			return nil, fmt.Errorf("field %s: expected *big.Int, got %T", name, f.Interface())
		}
		if n == nil {
			return big.NewInt(0), nil
		}
		return n, nil
	}

	tokenIn, err := addr("TokenIn")
	if err != nil {
		return nil, err
	}
	tokenOut, err := addr("TokenOut")
	if err != nil {
		return nil, err
	}
	recipient, err := addr("Recipient")
	if err != nil {
		return nil, err
	}
	amountIn, err := bigint("AmountIn")
	if err != nil {
		return nil, err
	}
	amountOutMin, err := bigint("AmountOutMinimum")
	if err != nil {
		return nil, err
	}
	sqrtLimit, err := bigint("SqrtPriceLimitX96")
	if err != nil {
		return nil, err
	}

	// Fee: uint24 → uint32 in go-ethereum; handle *big.Int fallback just in case
	feeField := rv.FieldByName("Fee")
	if !feeField.IsValid() {
		return nil, fmt.Errorf("field Fee not found")
	}
	var fee uint32
	switch feeField.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		fee = uint32(feeField.Uint())
	case reflect.Ptr:
		n, ok := feeField.Interface().(*big.Int)
		if !ok || n == nil {
			return nil, fmt.Errorf("field Fee: unexpected type %T", feeField.Interface())
		}
		fee = uint32(n.Uint64())
	default:
		return nil, fmt.Errorf("field Fee: unexpected kind %s", feeField.Kind())
	}

	info := &SwapInfo{
		TokenIn:           tokenIn,
		TokenOut:          tokenOut,
		Fee:               fee,
		Recipient:         recipient,
		AmountIn:          amountIn,
		AmountOutMinimum:  amountOutMin,
		SqrtPriceLimitX96: sqrtLimit,
	}

	if hasDeadline {
		dl, err := bigint("Deadline")
		if err != nil {
			return nil, err
		}
		info.Deadline = dl
	}
	return info, nil
}

// ---------- encode helpers ----------

func encodeV1(info *SwapInfo, recipient common.Address, deadline *big.Int) ([]byte, error) {
	if deadline == nil {
		deadline = big.NewInt(time.Now().Unix() + 60)
	}
	// Local struct whose field names match the ABI component names (case-insensitive).
	type params struct {
		TokenIn           common.Address
		TokenOut          common.Address
		Fee               *big.Int // go-ethereum Pack accepts *big.Int for uint24
		Recipient         common.Address
		Deadline          *big.Int
		AmountIn          *big.Int
		AmountOutMinimum  *big.Int
		SqrtPriceLimitX96 *big.Int
	}
	p := params{
		TokenIn:           info.TokenIn,
		TokenOut:          info.TokenOut,
		Fee:               new(big.Int).SetUint64(uint64(info.Fee)),
		Recipient:         recipient,
		Deadline:          deadline,
		AmountIn:          info.AmountIn,
		AmountOutMinimum:  info.AmountOutMinimum,
		SqrtPriceLimitX96: info.SqrtPriceLimitX96,
	}
	return parsedV1.Pack("exactInputSingle", p)
}

func encodeV2(info *SwapInfo, recipient common.Address) ([]byte, error) {
	type params struct {
		TokenIn           common.Address
		TokenOut          common.Address
		Fee               *big.Int
		Recipient         common.Address
		AmountIn          *big.Int
		AmountOutMinimum  *big.Int
		SqrtPriceLimitX96 *big.Int
	}
	p := params{
		TokenIn:           info.TokenIn,
		TokenOut:          info.TokenOut,
		Fee:               new(big.Int).SetUint64(uint64(info.Fee)),
		Recipient:         recipient,
		AmountIn:          info.AmountIn,
		AmountOutMinimum:  info.AmountOutMinimum,
		SqrtPriceLimitX96: info.SqrtPriceLimitX96,
	}
	return parsedV2.Pack("exactInputSingle", p)
}
