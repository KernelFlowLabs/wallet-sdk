package utxo

import (
	"encoding/hex"
	"fmt"
	"github.com/KernelFlowLabs/wallet-sdk/signing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

func PublicKey2Address(publicKey []byte, network string) string {
	var (
		params  chaincfg.Params
		address string
	)

	switch network {
	case NetworkEnumForBTC, NetworkEnumForBTCP2TR:
		params = btcParams
		program := btcutil.Hash160(publicKey)
		if network == NetworkEnumForBTCP2TR {
			btcPub, err := schnorr.ParsePubKey(publicKey[1:])
			if err != nil {
				return ""
			}
			addr, err := btcutil.NewAddressTaproot(
				schnorr.SerializePubKey(txscript.ComputeTaprootKeyNoScript(btcPub)),
				&params,
			)
			if err != nil {
				return ""
			}
			address = addr.String()
		} else {
			addr, err := btcutil.NewAddressWitnessPubKeyHash(program, &params)
			if err != nil {
				return ""
			}
			address = addr.String()
		}

	case NetworkEnumForLTC:
		params = ltcParams
		program := btcutil.Hash160(publicKey)
		addr, err := btcutil.NewAddressWitnessPubKeyHash(program, &params)
		if err != nil {
			return ""
		}
		address = addr.String()

	case NetworkEnumForDOGE:
		params = dogeParams
		program := btcutil.Hash160(publicKey)
		addr, err := btcutil.NewAddressPubKeyHash(program, &params)
		if err != nil {
			return ""
		}
		address = addr.String()

	case NetworkEnumForSYS:
		params = sysParams
		program := btcutil.Hash160(publicKey)
		addr, err := btcutil.NewAddressWitnessPubKeyHash(program, &params)
		if err != nil {
			return ""
		}
		address = addr.String()

	default:
		return ""
	}

	return address
}

func ValidAddress(address, network string) bool {
	switch network {
	case NetworkEnumForBTC, NetworkEnumForBTCP2TR:
		return ValidAddressForBTC(address)
	case NetworkEnumForLTC:
		return ValidAddressForLTC(address)
	case NetworkEnumForDOGE:
		return ValidAddressForDOGE(address)
	case NetworkEnumForSYS:
		return ValidAddressForSYS(address)
	}
	return false
}

func ValidAddressForBTC(address string) bool {
	if address == signing.MagicContactAddressForNative {
		return true
	}
	_, err := btcutil.DecodeAddress(address, &btcParams)
	if err == nil {
		return true
	}
	return false
}

func ValidAddressForLTC(address string) bool {
	if address == signing.MagicContactAddressForNative {
		return true
	}
	_, err := btcutil.DecodeAddress(address, &ltcParams)
	if err == nil {
		return true
	}
	return false
}

func ValidAddressForDOGE(address string) bool {
	if address == signing.MagicContactAddressForNative {
		return true
	}
	_, err := btcutil.DecodeAddress(address, &dogeParams)
	if err == nil {
		return true
	}
	return false
}

func ValidAddressForSYS(address string) bool {
	if address == signing.MagicContactAddressForNative {
		return true
	}
	_, err := btcutil.DecodeAddress(address, &sysParams)
	if err == nil {
		return true
	}
	return false
}

func AddressToScriptPubKey(address string, network string) (string, error) {
	var params *chaincfg.Params

	switch network {
	case NetworkEnumForBTC, NetworkEnumForBTCP2TR:
		params = &btcParams
	case NetworkEnumForLTC:
		params = &ltcParams
	case NetworkEnumForDOGE:
		params = &dogeParams
	default:
		return "", fmt.Errorf("unsupported network")
	}

	addr, err := btcutil.DecodeAddress(address, params)
	if err != nil {
		return "", fmt.Errorf("invalid address: %v", err)
	}

	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return "", fmt.Errorf("failed to create script: %v", err)
	}

	return hex.EncodeToString(script), nil
}

func PrivateKeyHexToWIF(privateKey string) (string, error) {
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex: %w", err)
	}
	priv, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

	privateKeyWif, err := btcutil.NewWIF(priv, &chaincfg.MainNetParams, true)
	if err != nil {
		return "", fmt.Errorf("failed to create WIF: %w", err)
	}
	return privateKeyWif.String(), nil
}

func PrivateKeyWIFToHex(privateKey string) (string, error) {
	wif, err := btcutil.DecodeWIF(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode WIF: %w", err)
	}
	privateKeyBytes := wif.PrivKey.Serialize()
	return hex.EncodeToString(privateKeyBytes), nil
}

const (
	NetworkEnumForBTC     string = "0"
	NetworkEnumForLTC     string = "1"
	NetworkEnumForDOGE    string = "2"
	NetworkEnumForBTCP2TR string = "3"
	NetworkEnumForSYS     string = "4"
)

func init() {
	if err := signing.RegisterAddressValidator("btc_addr", ValidAddressForBTC); err != nil {
		panic(err)
	}
	if err := signing.RegisterAddressValidator("ltc_addr", ValidAddressForBTC); err != nil {
		panic(err)
	}
	if err := signing.RegisterAddressValidator("doge_addr", ValidAddressForBTC); err != nil {
		panic(err)
	}
	if err := signing.RegisterAddressValidator("btc_addr", ValidAddressForBTC); err != nil {
		panic(err)
	}
}
