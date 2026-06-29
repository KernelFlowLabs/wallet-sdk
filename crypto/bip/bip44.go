package bip

import (
	"fmt"
	"strconv"
	"strings"
)

func SeedToKeyForECDSA(seed []byte, path string) (*Key, error) {
	masterKey, err := NewMasterKey(seed, PrivateWalletVersionDefault)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return masterKey, nil
	}

	path = strings.TrimPrefix(path, "m/")
	for _, v := range strings.Split(path, "/") {
		hardened := strings.HasSuffix(v, `'`)
		n, err := strconv.ParseUint(strings.TrimSuffix(v, `'`), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to ParseUint, err=%v", err)
		}
		child := uint32(n)
		if hardened {
			child += FirstHardenedChild
		}
		masterKey, err = masterKey.NewChildKey(child)
		if err != nil {
			return nil, err
		}
	}
	return masterKey, nil
}

func SeedToKeyForED25519(seed []byte, path string) (*Key, error) {
	masterKey, err := NewMasterKeyED25519(seed)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return masterKey, nil
	}

	path = strings.TrimPrefix(path, "m/")
	tmp := strings.Split(path, "/")
	var paths []uint32
	for _, v := range tmp {
		n, err := strconv.ParseUint(strings.TrimSuffix(v, `'`), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to ParseUint, err=%v", err)
		}
		paths = append(paths, uint32(n))
	}

	for _, v := range paths {
		masterKey, err = masterKey.NewChildKeyED25519(FirstHardenedChild + v)
		if err != nil {
			return nil, err
		}
	}
	return masterKey, nil
}
