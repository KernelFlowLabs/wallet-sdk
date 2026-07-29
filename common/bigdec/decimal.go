package bigdec

import (
	"errors"
	"math/big"
	"strings"

	"github.com/shopspring/decimal"
)

const precision = 60

func trimDecimal(s string) string {
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func parseRat(s string) (*big.Rat, error) {
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return nil, errors.New("invalid number: " + s)
	}
	return r, nil
}

func BigAdd(a, b string) (string, error) {
	ar, err := parseRat(a)
	if err != nil {
		return "", err
	}

	br, err := parseRat(b)
	if err != nil {
		return "", err
	}

	ar.Add(ar, br)

	return trimDecimal(ar.FloatString(precision)), nil
}

func BigSub(a, b string) (string, error) {
	ar, err := parseRat(a)
	if err != nil {
		return "", err
	}

	br, err := parseRat(b)
	if err != nil {
		return "", err
	}

	ar.Sub(ar, br)

	return trimDecimal(ar.FloatString(precision)), nil
}

func BigMul(a, b string) (string, error) {
	ar, err := parseRat(a)
	if err != nil {
		return "", err
	}

	br, err := parseRat(b)
	if err != nil {
		return "", err
	}

	ar.Mul(ar, br)

	return trimDecimal(ar.FloatString(precision)), nil
}

func BigDiv(a, b string) (string, error) {
	ar, err := parseRat(a)
	if err != nil {
		return "", err
	}

	br, err := parseRat(b)
	if err != nil {
		return "", err
	}

	if br.Sign() == 0 {
		return "", errors.New("division by zero")
	}

	ar.Quo(ar, br)

	return trimDecimal(ar.FloatString(precision)), nil
}

func HexToDecimal(hex string) string {
	hex = strings.TrimPrefix(hex, "0x")
	n := new(big.Int)
	n.SetString(hex, 16)
	return n.String()
}

func CalcAmountOutMin(amountOut string, slippage float64) string {
	amount, err := decimal.NewFromString(amountOut)
	if err != nil {
		return "0"
	}
	rate := decimal.NewFromFloat(1 - slippage/100)
	return amount.Mul(rate).Truncate(0).String()
}
