package evm

import (
	"fmt"
	"math/big"
)

type Fee struct {
	BaseFee     string
	PriorityFee string // wei, for 1559
	MaxFeeCap   string // wei, for 1559
	GasPrice    string // wei, for legacy
}
type FeeTiers struct {
	Slow, Avg, Fast *Fee
}

func SuggestFees(baseFeeStr, tipCapStr, floorTipCapStr, gasPriceStr string, isLegacy bool) (FeeTiers, error) {
	if isLegacy {
		gasPrice, ok := big.NewInt(0).SetString(gasPriceStr, 10)
		if !ok {
			return FeeTiers{}, fmt.Errorf("failed to SetString for gasPriceStr")
		}
		return suggestLegacy(gasPrice)
	}

	baseFee, ok := big.NewInt(0).SetString(baseFeeStr, 10)
	if !ok {
		return FeeTiers{}, fmt.Errorf("failed to SetString for baseFeeStr")
	}
	tipCap, ok := big.NewInt(0).SetString(tipCapStr, 10)
	if !ok {
		return FeeTiers{}, fmt.Errorf("failed to SetString for tipCapStr")
	}
	floorTipCap, ok := big.NewInt(0).SetString(floorTipCapStr, 10)
	if !ok {
		return FeeTiers{}, fmt.Errorf("failed to SetString for floorTipCapStr")
	}
	return suggest1559(baseFee, tipCap, floorTipCap)
}
func suggestLegacy(gasPrice *big.Int) (FeeTiers, error) {
	if gasPrice == nil {
		return FeeTiers{}, fmt.Errorf("gasPrice is nil")
	}
	slow := mulIntFloat(gasPrice, 0.8)
	avg := new(big.Int).Set(gasPrice)
	fast := mulIntFloat(gasPrice, 1.5)

	return FeeTiers{
		Slow: &Fee{GasPrice: slow.String()},
		Avg:  &Fee{GasPrice: avg.String()},
		Fast: &Fee{GasPrice: fast.String()},
	}, nil
}
func suggest1559(baseFee, nodeTip, floorTipCap *big.Int) (FeeTiers, error) {
	if nodeTip.Cmp(big.NewInt(0)) == 0 {
		fee := &Fee{
			BaseFee:     baseFee.String(),
			PriorityFee: floorTipCap.String(),
			MaxFeeCap:   new(big.Int).Add(baseFee, floorTipCap).String(),
		}
		return FeeTiers{
			Slow: fee,
			Avg:  fee,
			Fast: fee,
		}, nil
	}

	bSlow := mulIntFloat(baseFee, 1.80)
	bAvg := mulIntFloat(baseFee, 1.42)
	bFast := mulIntFloat(baseFee, 1.125)

	tSlow := maxInt(mulIntFloat(nodeTip, 0.8), floorTipCap)
	tAvg := maxInt(nodeTip, floorTipCap)
	tFast := maxInt(mulIntFloat(nodeTip, 1.5), floorTipCap)

	cSlow := safeCap(bSlow, tSlow, baseFee)
	cAvg := safeCap(bAvg, tAvg, baseFee)
	cFast := safeCap(bFast, tFast, baseFee)

	return FeeTiers{
		Slow: &Fee{BaseFee: baseFee.String(), PriorityFee: tSlow.String(), MaxFeeCap: cSlow.String()},
		Avg:  &Fee{BaseFee: baseFee.String(), PriorityFee: tAvg.String(), MaxFeeCap: cAvg.String()},
		Fast: &Fee{BaseFee: baseFee.String(), PriorityFee: tFast.String(), MaxFeeCap: cFast.String()},
	}, nil
}
func mulIntFloat(x *big.Int, m float64) *big.Int {
	fx := new(big.Float).SetInt(x)
	fx.Mul(fx, big.NewFloat(m))
	r, _ := fx.Int(nil)
	return r
}
func maxInt(a, b *big.Int) *big.Int {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.Cmp(b) >= 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}
func safeCap(projectedBase, tip, base *big.Int) *big.Int {
	cap1 := new(big.Int).Add(projectedBase, tip)
	minCap := new(big.Int).Add(base, tip)
	if cap1.Cmp(minCap) < 0 {
		return minCap
	}
	return cap1
}

// for GuaranteeNativeAmt
func GuaranteeNativeAmt(fee *Fee, gasLimitStr, nativeBalStr string, l1FeeStr string, isLegacy bool) (bool, error) {
	gasLimit, ok := big.NewInt(0).SetString(gasLimitStr, 10)
	if !ok {
		return false, fmt.Errorf("failed to SetString for gasLimitStr")
	}
	nativeBal, ok := big.NewInt(0).SetString(nativeBalStr, 10)
	if !ok {
		return false, fmt.Errorf("failed to SetString for nativeBalStr")
	}

	total := big.NewInt(0)
	if isLegacy {
		gasPrice, ok := big.NewInt(0).SetString(fee.GasPrice, 10)
		if !ok {
			return false, fmt.Errorf("failed to SetString for gasPrice")
		}
		total = gasPrice.Mul(gasPrice, gasLimit)
	} else {
		baseFee, ok := big.NewInt(0).SetString(fee.BaseFee, 10)
		if !ok {
			return false, fmt.Errorf("failed to SetString for baseFee")
		}
		priorityFee, ok := big.NewInt(0).SetString(fee.PriorityFee, 10)
		if !ok {
			return false, fmt.Errorf("failed to SetString for priorityFee")
		}
		maxFeeCap, ok := big.NewInt(0).SetString(fee.MaxFeeCap, 10)
		if !ok {
			return false, fmt.Errorf("failed to SetString for maxFeeCap")
		}
		effective := new(big.Int).Add(baseFee, priorityFee)
		if effective.Cmp(maxFeeCap) == 1 {
			effective = maxFeeCap
		}
		total = effective.Mul(effective, gasLimit)
		if l1FeeStr != "" {
			l1Fee, ok := big.NewInt(0).SetString(l1FeeStr, 10)
			if !ok {
				return false, fmt.Errorf("failed to SetString for l1Fee")
			}
			total.Add(total, l1Fee)
		}
	}
	return nativeBal.Cmp(total) >= 0, nil
}
