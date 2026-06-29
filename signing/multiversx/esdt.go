package multiversx

import (
	"encoding/hex"
	"fmt"
	"math/big"
)

func PackPayloadForESDT(assetsID, amount string) (string, error) {
	assetsIDHex := hex.EncodeToString([]byte(assetsID))
	amountBig, ok := big.NewInt(0).SetString(amount, 10)
	if !ok {
		return "", fmt.Errorf("failed to SetString for amount %s", amount)
	}
	amountHex := fmt.Sprintf("%x", amountBig)
	if len(amountHex)%2 != 0 {
		amountHex = "0" + amountHex
	}

	payload := "ESDTTransfer" + "@" + assetsIDHex + "@" + amountHex
	return payload, nil
}
