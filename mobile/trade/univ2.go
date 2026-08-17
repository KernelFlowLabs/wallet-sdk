package trade

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kernelflowlabs/wallet-sdk/dex/univ2"
)

const (
	loamChainName     = "LOAM"
	loamChainID       = uint64(12192)
	loamTestnetRPC    = "https://rpc.loamchain.com/testnet"
	loamFactory       = "0x379644CA7ED677b42bDe3053108Bbd2c693F68E4"
	loamRouter02      = "0x2587e1A0A933c17740BC260134042B59e5451A73"
	loamWrappedNative = "0x017A39461F54E5871aCff276c61bc949933154eC"
	loamUSDC          = "0xCd80b98eC2c1BE1314CBC42b3f2ee329bED0f29B"
	loamUSDT          = "0x1EA2b7f60749d7C3A378c68AfC49B974a5e08f3d"
)

var univ2Clients map[string]*univ2.Client

func defaultUniv2Configs() []univ2.Config {
	return []univ2.Config{{
		ChainName:       loamChainName,
		ChainID:         loamChainID,
		RPC:             loamTestnetRPC,
		Factory:         loamFactory,
		Router02:        loamRouter02,
		WrappedNative:   loamWrappedNative,
		QuoteBaseTokens: []string{loamUSDC, loamUSDT},
	}}
}

func newUniv2Clients(configs []univ2.Config) (map[string]*univ2.Client, error) {
	if len(configs) == 0 {
		configs = defaultUniv2Configs()
	}
	clients := make(map[string]*univ2.Client, len(configs)*2)
	created := make([]*univ2.Client, 0, len(configs))
	closeCreated := func() {
		for _, client := range created {
			client.Close()
		}
	}
	for i, config := range configs {
		client, err := univ2.NewClient(config)
		if err != nil {
			closeCreated()
			return nil, fmt.Errorf("univ2[%d]: %w", i, err)
		}
		created = append(created, client)
		normalized := client.Config()
		nameKey := strings.ToUpper(normalized.ChainName)
		idKey := strconv.FormatUint(normalized.ChainID, 10)
		if _, exists := clients[nameKey]; exists {
			closeCreated()
			return nil, fmt.Errorf("duplicate univ2 chain %q", normalized.ChainName)
		}
		if _, exists := clients[idKey]; exists {
			closeCreated()
			return nil, fmt.Errorf("duplicate univ2 chainId %d", normalized.ChainID)
		}
		clients[nameKey] = client
		clients[idKey] = client
	}
	return clients, nil
}

func autoQuoteUniv2(chain string) *univ2.Client {
	tradeMu.RLock()
	defer tradeMu.RUnlock()
	return univ2Clients[strings.ToUpper(strings.TrimSpace(chain))]
}
