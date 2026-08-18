package univ3

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	defaultDeadlineTTL = 60 * time.Second
	maxMulticallDepth  = 8
)

var (
	ErrInvalidRouterVersion = errors.New("univ3: router version must be 1 or 2")
	ErrNoSwapFound          = errors.New("univ3: no exact-input swap found")
	ErrUnsupportedSelector  = errors.New("univ3: unsupported selector")

	parsedV1                 abi.ABI
	parsedV2                 abi.ABI
	parsedMulticallBlockhash abi.ABI
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
	parsedMulticallBlockhash, err = abi.JSON(strings.NewReader(abiMulticallBlockhash))
	if err != nil {
		panic(fmt.Sprintf("univ3: failed to parse blockhash multicall ABI: %v", err))
	}
}

func IsUniV3Router(addr string) (int, bool) {
	addr = strings.TrimSpace(addr)
	if len(addr) != 42 || !common.IsHexAddress(addr) {
		return 0, false
	}
	switch common.HexToAddress(addr) {
	case common.HexToAddress(AddrSwapRouterV1):
		return RouterVersionV1, true
	case common.HexToAddress(AddrSwapRouter02):
		return RouterVersion02, true
	default:
		return 0, false
	}
}

func DecodeSwapInfo(calldata []byte, routerVer int) (*SwapInfo, error) {
	infos, err := DecodeSwapInfos(calldata, routerVer)
	if err != nil {
		return nil, err
	}
	if len(infos) != 1 {
		return nil, fmt.Errorf("univ3: calldata contains %d swaps; use DecodeSwapInfos", len(infos))
	}
	return infos[0], nil
}

func DecodeSwapInfos(calldata []byte, routerVer int) ([]*SwapInfo, error) {
	return decodeSwapInfos(calldata, routerVer, 0)
}

func decodeSwapInfos(calldata []byte, routerVer, depth int) ([]*SwapInfo, error) {
	if len(calldata) < 4 {
		return nil, fmt.Errorf("univ3: calldata too short (%d bytes)", len(calldata))
	}
	if depth > maxMulticallDepth {
		return nil, fmt.Errorf("univ3: multicall nesting exceeds %d", maxMulticallDepth)
	}

	selector := hex.EncodeToString(calldata[:4])
	switch selector {
	case SelectorExactInputSingleV1:
		info, err := decodeV1(calldata)
		if err != nil {
			return nil, err
		}
		return []*SwapInfo{info}, nil
	case SelectorExactInputV1:
		info, err := decodeExactInputV1(calldata)
		if err != nil {
			return nil, err
		}
		return []*SwapInfo{info}, nil
	case SelectorExactInputSingleV2:
		info, err := decodeV2(calldata)
		if err != nil {
			return nil, err
		}
		return []*SwapInfo{info}, nil
	case SelectorMulticall, SelectorMulticallDeadline, SelectorMulticallBlockhash:
		if err := validateRouterVersion(routerVer); err != nil {
			return nil, err
		}
		if selector != SelectorMulticall && routerVer != RouterVersion02 {
			return nil, fmt.Errorf("univ3: selector %s requires SwapRouter02", selector)
		}
		calls, deadline, blockhash, err := decodeMulticallEnvelope(calldata, selector)
		if err != nil {
			return nil, err
		}
		infos := make([]*SwapInfo, 0, len(calls))
		for index, inner := range calls {
			if !hasSupportedSelector(inner) {
				continue
			}
			innerInfos, err := decodeSwapInfos(inner, routerVer, depth+1)
			if err != nil {
				if errors.Is(err, ErrNoSwapFound) {
					continue
				}
				return nil, fmt.Errorf("univ3: multicall item %d: %w", index, err)
			}
			infos = append(infos, innerInfos...)
		}
		if len(infos) == 0 {
			return nil, ErrNoSwapFound
		}
		for _, info := range infos {
			if info.RouterVersion != routerVer {
				return nil, fmt.Errorf("univ3: router version %d contains version %d swap calldata", routerVer, info.RouterVersion)
			}
			if deadline != nil {
				if info.Deadline == nil || deadline.Cmp(info.Deadline) < 0 {
					info.Deadline = new(big.Int).Set(deadline)
				}
			}
			if blockhash != nil {
				if info.PreviousBlockhash != nil && *info.PreviousBlockhash != *blockhash {
					return nil, fmt.Errorf("univ3: nested multicalls require different previous block hashes")
				}
				if info.PreviousBlockhash == nil {
					hashCopy := *blockhash
					info.PreviousBlockhash = &hashCopy
				}
			}
		}
		return infos, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedSelector, selector)
	}
}

func hasSupportedSelector(calldata []byte) bool {
	if len(calldata) < 4 {
		return false
	}
	switch hex.EncodeToString(calldata[:4]) {
	case SelectorExactInputSingleV1, SelectorExactInputV1, SelectorExactInputSingleV2,
		SelectorMulticall, SelectorMulticallDeadline, SelectorMulticallBlockhash:
		return true
	default:
		return false
	}
}

func decodeMulticallEnvelope(calldata []byte, selector string) ([][]byte, *big.Int, *common.Hash, error) {
	var method abi.Method
	switch selector {
	case SelectorMulticall:
		method = parsedV1.Methods["multicall"]
	case SelectorMulticallDeadline:
		method = parsedV2.Methods["multicall"]
	case SelectorMulticallBlockhash:
		method = parsedMulticallBlockhash.Methods["multicall"]
	default:
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedSelector, selector)
	}

	values, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("univ3: unpack multicall %s: %w", selector, err)
	}
	if selector == SelectorMulticall {
		if len(values) != 1 {
			return nil, nil, nil, fmt.Errorf("univ3: multicall has %d arguments", len(values))
		}
		calls, ok := values[0].([][]byte)
		if !ok {
			return nil, nil, nil, fmt.Errorf("univ3: multicall data has type %T", values[0])
		}
		return calls, nil, nil, nil
	}
	if len(values) != 2 {
		return nil, nil, nil, fmt.Errorf("univ3: multicall has %d arguments", len(values))
	}
	calls, ok := values[1].([][]byte)
	if !ok {
		return nil, nil, nil, fmt.Errorf("univ3: multicall data has type %T", values[1])
	}
	if selector == SelectorMulticallDeadline {
		deadline, ok := values[0].(*big.Int)
		if !ok || deadline == nil {
			return nil, nil, nil, fmt.Errorf("univ3: multicall deadline has type %T", values[0])
		}
		return calls, deadline, nil, nil
	}
	rawHash, ok := values[0].([32]byte)
	if !ok {
		return nil, nil, nil, fmt.Errorf("univ3: multicall blockhash has type %T", values[0])
	}
	hash := common.BytesToHash(rawHash[:])
	return calls, nil, &hash, nil
}

func EncodeExactInputSingle(info *SwapInfo, recipient common.Address, deadline *big.Int, routerVer int) ([]byte, error) {
	if err := validateRouterVersion(routerVer); err != nil {
		return nil, err
	}
	if err := validateSwapInfo(info, recipient, routerVer); err != nil {
		return nil, err
	}
	deadline, err := effectiveDeadline(deadline)
	if err != nil {
		return nil, err
	}
	if routerVer == RouterVersionV1 {
		return encodeV1(info, recipient, deadline)
	}
	inner, err := encodeV2(info, recipient)
	if err != nil {
		return nil, err
	}
	return parsedV2.Pack("multicall", deadline, [][]byte{inner})
}

func EncodeExactInputSingleCall(info *SwapInfo, recipient common.Address, deadline *big.Int, routerVer int) ([]byte, error) {
	if err := validateRouterVersion(routerVer); err != nil {
		return nil, err
	}
	if err := validateSwapInfo(info, recipient, routerVer); err != nil {
		return nil, err
	}
	if routerVer == RouterVersionV1 {
		deadline, err := effectiveDeadline(deadline)
		if err != nil {
			return nil, err
		}
		return encodeV1(info, recipient, deadline)
	}
	if deadline != nil {
		return nil, fmt.Errorf("univ3: raw SwapRouter02 exactInputSingle has no deadline; use EncodeExactInputSingle")
	}
	return encodeV2(info, recipient)
}

func validateRouterVersion(routerVer int) error {
	if routerVer != RouterVersionV1 && routerVer != RouterVersion02 {
		return ErrInvalidRouterVersion
	}
	return nil
}

func validateSwapInfo(info *SwapInfo, recipient common.Address, routerVer int) error {
	if info == nil {
		return fmt.Errorf("univ3: nil swap info")
	}
	if info.RouterVersion != 0 && info.RouterVersion != routerVer {
		return fmt.Errorf("univ3: swap info router version %d does not match %d", info.RouterVersion, routerVer)
	}
	if info.TokenIn == (common.Address{}) || info.TokenOut == (common.Address{}) || info.TokenIn == info.TokenOut {
		return fmt.Errorf("univ3: invalid token pair")
	}
	if recipient == (common.Address{}) {
		return fmt.Errorf("univ3: zero recipient")
	}
	if info.Fee > MaxUint24 {
		return fmt.Errorf("univ3: fee exceeds uint24")
	}
	if info.AmountIn == nil || info.AmountIn.Sign() <= 0 || info.AmountIn.BitLen() > 256 {
		return fmt.Errorf("univ3: amountIn must be a positive uint256")
	}
	if info.AmountOutMinimum == nil || info.AmountOutMinimum.Sign() <= 0 || info.AmountOutMinimum.BitLen() > 256 {
		return fmt.Errorf("univ3: amountOutMinimum must be a positive uint256")
	}
	if info.SqrtPriceLimitX96 == nil || info.SqrtPriceLimitX96.Sign() < 0 || info.SqrtPriceLimitX96.BitLen() > 160 {
		return fmt.Errorf("univ3: sqrtPriceLimitX96 must be a uint160")
	}
	return nil
}

func effectiveDeadline(deadline *big.Int) (*big.Int, error) {
	if deadline == nil {
		return big.NewInt(time.Now().Add(defaultDeadlineTTL).Unix()), nil
	}
	if deadline.Sign() <= 0 || deadline.BitLen() > 256 {
		return nil, fmt.Errorf("univ3: deadline must be a positive uint256")
	}
	return new(big.Int).Set(deadline), nil
}

func decodeV1(calldata []byte) (*SwapInfo, error) {
	method := parsedV1.Methods["exactInputSingle"]
	values, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, fmt.Errorf("univ3: unpack v1 exactInputSingle: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("univ3: unexpected v1 argument count %d", len(values))
	}
	info, err := extractSwapFields(values[0], true)
	if err != nil {
		return nil, err
	}
	info.RouterVersion = RouterVersionV1
	return info, nil
}

func decodeV2(calldata []byte) (*SwapInfo, error) {
	method := parsedV2.Methods["exactInputSingle"]
	values, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, fmt.Errorf("univ3: unpack v2 exactInputSingle: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("univ3: unexpected v2 argument count %d", len(values))
	}
	info, err := extractSwapFields(values[0], false)
	if err != nil {
		return nil, err
	}
	info.RouterVersion = RouterVersion02
	return info, nil
}

func decodeExactInputV1(calldata []byte) (*SwapInfo, error) {
	method := parsedV1.Methods["exactInput"]
	values, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return nil, fmt.Errorf("univ3: unpack v1 exactInput: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("univ3: unexpected v1 exactInput argument count %d", len(values))
	}
	rv := reflect.ValueOf(values[0])
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("univ3: expected exactInput tuple struct, got %T", values[0])
	}
	field := func(name string) (reflect.Value, error) {
		value := rv.FieldByName(name)
		if !value.IsValid() {
			return reflect.Value{}, fmt.Errorf("univ3: exactInput field %s not found", name)
		}
		return value, nil
	}
	pathField, err := field("Path")
	if err != nil {
		return nil, err
	}
	path, ok := pathField.Interface().([]byte)
	if !ok {
		return nil, fmt.Errorf("univ3: exactInput Path has type %T", pathField.Interface())
	}
	tokens, fees, err := decodeV3Path(path)
	if err != nil {
		return nil, err
	}
	recipientField, err := field("Recipient")
	if err != nil {
		return nil, err
	}
	recipient, ok := recipientField.Interface().(common.Address)
	if !ok || recipient == (common.Address{}) {
		return nil, fmt.Errorf("univ3: exactInput has invalid recipient")
	}
	bigField := func(name string) (*big.Int, error) {
		value, err := field(name)
		if err != nil {
			return nil, err
		}
		number, ok := value.Interface().(*big.Int)
		if !ok || number == nil {
			return nil, fmt.Errorf("univ3: exactInput field %s has type %T", name, value.Interface())
		}
		return new(big.Int).Set(number), nil
	}
	deadline, err := bigField("Deadline")
	if err != nil {
		return nil, err
	}
	amountIn, err := bigField("AmountIn")
	if err != nil {
		return nil, err
	}
	amountOutMinimum, err := bigField("AmountOutMinimum")
	if err != nil {
		return nil, err
	}
	if deadline.Sign() <= 0 || deadline.BitLen() > 256 || amountIn.Sign() <= 0 || amountIn.BitLen() > 256 || amountOutMinimum.Sign() <= 0 || amountOutMinimum.BitLen() > 256 {
		return nil, fmt.Errorf("univ3: exactInput contains invalid numeric fields")
	}
	return &SwapInfo{
		TokenIn: tokens[0], TokenOut: tokens[len(tokens)-1], Recipient: recipient,
		Deadline: deadline, AmountIn: amountIn, AmountOutMinimum: amountOutMinimum,
		SqrtPriceLimitX96: new(big.Int), RouterVersion: RouterVersionV1,
		Path: append([]byte(nil), path...), Tokens: tokens, Fees: fees,
	}, nil
}

func extractSwapFields(value any, hasDeadline bool) (*SwapInfo, error) {
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("univ3: expected tuple struct, got %T", value)
	}

	addressField := func(name string) (common.Address, error) {
		field := rv.FieldByName(name)
		if !field.IsValid() {
			return common.Address{}, fmt.Errorf("univ3: field %s not found", name)
		}
		address, ok := field.Interface().(common.Address)
		if !ok {
			return common.Address{}, fmt.Errorf("univ3: field %s has type %T", name, field.Interface())
		}
		return address, nil
	}
	bigIntField := func(name string) (*big.Int, error) {
		field := rv.FieldByName(name)
		if !field.IsValid() {
			return nil, fmt.Errorf("univ3: field %s not found", name)
		}
		number, ok := field.Interface().(*big.Int)
		if !ok || number == nil {
			return nil, fmt.Errorf("univ3: field %s has type %T", name, field.Interface())
		}
		return number, nil
	}

	tokenIn, err := addressField("TokenIn")
	if err != nil {
		return nil, err
	}
	tokenOut, err := addressField("TokenOut")
	if err != nil {
		return nil, err
	}
	recipient, err := addressField("Recipient")
	if err != nil {
		return nil, err
	}
	amountIn, err := bigIntField("AmountIn")
	if err != nil {
		return nil, err
	}
	amountOutMinimum, err := bigIntField("AmountOutMinimum")
	if err != nil {
		return nil, err
	}
	sqrtPriceLimitX96, err := bigIntField("SqrtPriceLimitX96")
	if err != nil {
		return nil, err
	}

	feeField := rv.FieldByName("Fee")
	if !feeField.IsValid() {
		return nil, fmt.Errorf("univ3: field Fee not found")
	}
	var fee uint32
	switch feeField.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		fee = uint32(feeField.Uint())
	case reflect.Ptr:
		number, ok := feeField.Interface().(*big.Int)
		if !ok || number == nil || number.Sign() < 0 || number.BitLen() > 24 {
			return nil, fmt.Errorf("univ3: invalid Fee value")
		}
		fee = uint32(number.Uint64())
	default:
		return nil, fmt.Errorf("univ3: field Fee has kind %s", feeField.Kind())
	}

	info := &SwapInfo{
		TokenIn:           tokenIn,
		TokenOut:          tokenOut,
		Fee:               fee,
		Recipient:         recipient,
		AmountIn:          amountIn,
		AmountOutMinimum:  amountOutMinimum,
		SqrtPriceLimitX96: sqrtPriceLimitX96,
		Tokens:            []common.Address{tokenIn, tokenOut},
		Fees:              []uint32{fee},
	}
	if hasDeadline {
		info.Deadline, err = bigIntField("Deadline")
		if err != nil {
			return nil, err
		}
	}
	return info, nil
}

func encodeV1(info *SwapInfo, recipient common.Address, deadline *big.Int) ([]byte, error) {
	type params struct {
		TokenIn           common.Address
		TokenOut          common.Address
		Fee               *big.Int
		Recipient         common.Address
		Deadline          *big.Int
		AmountIn          *big.Int
		AmountOutMinimum  *big.Int
		SqrtPriceLimitX96 *big.Int
	}
	return parsedV1.Pack("exactInputSingle", params{
		TokenIn:           info.TokenIn,
		TokenOut:          info.TokenOut,
		Fee:               new(big.Int).SetUint64(uint64(info.Fee)),
		Recipient:         recipient,
		Deadline:          deadline,
		AmountIn:          info.AmountIn,
		AmountOutMinimum:  info.AmountOutMinimum,
		SqrtPriceLimitX96: info.SqrtPriceLimitX96,
	})
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
	return parsedV2.Pack("exactInputSingle", params{
		TokenIn:           info.TokenIn,
		TokenOut:          info.TokenOut,
		Fee:               new(big.Int).SetUint64(uint64(info.Fee)),
		Recipient:         recipient,
		AmountIn:          info.AmountIn,
		AmountOutMinimum:  info.AmountOutMinimum,
		SqrtPriceLimitX96: info.SqrtPriceLimitX96,
	})
}
