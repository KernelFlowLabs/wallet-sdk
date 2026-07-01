package ton

import (
	"crypto/hmac"
	"crypto/sha512"

	"github.com/tonkeeper/tongo"
	"github.com/tonkeeper/tongo/ton"
	"golang.org/x/crypto/pbkdf2"
)

const (
	tonSeedIterations = 100000
	tonSeedSalt       = "TON default seed"
)

func derivePrivateKey(mnemonic string) ([]byte, error) {
	mac := hmac.New(sha512.New, []byte(mnemonic))
	mac.Write([]byte(""))
	hash := mac.Sum(nil)
	return pbkdf2.Key(hash, []byte(tonSeedSalt), tonSeedIterations, 32, sha512.New), nil
}

func ValidAddress(address string) bool {
	if len(address) != 48 {
		return false
	}
	if _, err := tongo.ParseAddress(address); err != nil {
		return false
	}
	return true
}

func AddressToRaw(s string) (string, error) {
	addr, err := tongo.ParseAddress(s)
	if err != nil {
		return "", err
	}
	return addr.ID.ToRaw(), nil
}

func RawToAddress(raw string) (string, error) {
	id, err := ton.AccountIDFromRaw(raw)
	if err != nil {
		return "", err
	}
	return id.ToHuman(false, false), nil
}

func AddressToBounce(s string) (string, error) {
	addr, err := tongo.ParseAddress(s)
	if err != nil {
		return "", err
	}
	return addr.ID.ToHuman(true, false), nil
}

func AddressToNoBounce(s string) (string, error) {
	addr, err := tongo.ParseAddress(s)
	if err != nil {
		return "", err
	}
	return addr.ID.ToHuman(false, false), nil
}
