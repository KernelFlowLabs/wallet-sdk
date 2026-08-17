package trade

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/kernelflowlabs/wallet-sdk/dex/jupiter"
	"github.com/kernelflowlabs/wallet-sdk/dex/univ2"
)

var (
	tradeMu           sync.RWMutex
	jup               *jupiter.Client
	server            string
	bungeeAPIKey      string
	bungeeAffiliateID string
)

func Init(configJSON string) string {
	var cfg struct {
		JupiterAPIKey     string         `json:"jupiter_api_key"`
		SolanaRPC         string         `json:"solana_rpc"`
		FeeAccount        string         `json:"fee_account"`
		ServerURL         string         `json:"server_url"`
		BungeeAPIKey      string         `json:"bungee_api_key"`
		BungeeAffiliateID string         `json:"bungee_affiliate_id"`
		Univ2             []univ2.Config `json:"univ2,omitempty"`
	}
	if s := strings.TrimSpace(configJSON); s != "" {
		if err := json.Unmarshal([]byte(s), &cfg); err != nil {
			return fmt.Sprintf("parse config: %v", err)
		}
	}
	client := jupiter.NewClientWithOptions(cfg.SolanaRPC, cfg.FeeAccount, cfg.JupiterAPIKey)
	univ2ByChain, err := newUniv2Clients(cfg.Univ2)
	if err != nil {
		return fmt.Sprintf("configure univ2: %v", err)
	}
	srv := strings.TrimRight(cfg.ServerURL, "/")

	tradeMu.Lock()
	jup = client
	server = srv
	proxy = nil
	bungeeAPIKey = cfg.BungeeAPIKey
	bungeeAffiliateID = cfg.BungeeAffiliateID
	bng = nil
	univ2Clients = univ2ByChain
	tradeMu.Unlock()
	return ""
}

func jupClient() *jupiter.Client {
	tradeMu.RLock()
	defer tradeMu.RUnlock()
	return jup
}

func Version() string {
	return "wallet-sdk/mobile/trade 0.4.0 (autoQuote + inline univ2 swaps + buildTx + checkTx + submitIntent, device/server hybrid)"
}
