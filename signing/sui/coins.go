package sui

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type (
	CoinItem struct {
		CoinType            string `json:"coinType"`
		CoinObjectId        string `json:"coinObjectId"`
		Version             string `json:"version"`
		Digest              string `json:"digest"`
		Balance             string `json:"balance"`
		PreviousTransaction string `json:"previousTransaction"`
	}
	Coins []CoinItem
)

func parseCoinRefs(jsonStr string) ([]suiObjectRef, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var coins Coins
	if err := json.Unmarshal([]byte(jsonStr), &coins); err != nil {
		return nil, err
	}
	refs := make([]suiObjectRef, 0, len(coins))
	for _, c := range coins {
		version, err := strconv.ParseUint(c.Version, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid coin version %q: %w", c.Version, err)
		}
		refs = append(refs, suiObjectRef{objectID: c.CoinObjectId, version: version, digest: c.Digest})
	}
	return refs, nil
}
