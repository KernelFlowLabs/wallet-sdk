package signing

import (
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var Validator = validator.New()

func init() {
	_ = Validator.RegisterValidation("u64", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if s == "" {
			return false
		}
		_, err := strconv.ParseUint(s, 10, 64)
		return err == nil
	})
	_ = Validator.RegisterValidation("u64_gt0", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if s == "" {
			return false
		}
		v, err := strconv.ParseUint(s, 10, 64)
		return err == nil && v > 0
	})
	_ = Validator.RegisterValidation("i64", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if s == "" {
			return false
		}
		_, err := strconv.ParseInt(s, 10, 64)
		return err == nil
	})
	_ = Validator.RegisterValidation("bigint_gt0", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if s == "" {
			return false
		}
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return false
		}
		return n.Sign() > 0
	})
	_ = Validator.RegisterValidation("bigint_gte0", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if s == "" {
			return false
		}
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return false
		}
		return n.Sign() >= 0
	})
	_ = Validator.RegisterValidation("hex_str", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if s == "" {
			return false
		}
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			s = s[2:]
		}

		if s == "" {
			return false
		}
		for _, c := range s {
			if !unicode.IsDigit(c) &&
				(c < 'a' || c > 'f') &&
				(c < 'A' || c > 'F') {
				return false
			}
		}
		return true
	})
	_ = Validator.RegisterValidation("bool_str", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return s == "true" || s == "false"
	})
	_ = Validator.RegisterValidation("native", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return strings.EqualFold(s, MagicContactAddressForNative)
	})
}

func RegisterAddressValidator(tag string, fn func(string) bool) error {
	return Validator.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return fn(fl.Field().String())
	})
}
