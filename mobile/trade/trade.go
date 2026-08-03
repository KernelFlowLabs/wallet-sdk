package trade

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/kernelflowlabs/wallet-sdk/dex/jupiter"
)

var (
	tradeMu sync.RWMutex
	jup     *jupiter.Client
	server  string
)

func Init(configJSON string) string {
	var cfg struct {
		JupiterAPIKey string `json:"jupiter_api_key"`
		SolanaRPC     string `json:"solana_rpc"`
		FeeAccount    string `json:"fee_account"`
		ServerURL     string `json:"server_url"`
	}
	if s := strings.TrimSpace(configJSON); s != "" {
		if err := json.Unmarshal([]byte(s), &cfg); err != nil {
			return fmt.Sprintf("parse config: %v", err)
		}
	}
	client := jupiter.NewClientWithOptions(cfg.SolanaRPC, cfg.FeeAccount, cfg.JupiterAPIKey)
	srv := strings.TrimRight(cfg.ServerURL, "/")

	tradeMu.Lock()
	jup = client
	server = srv
	proxy = nil
	tradeMu.Unlock()
	return ""
}

func jupClient() *jupiter.Client {
	tradeMu.RLock()
	defer tradeMu.RUnlock()
	return jup
}

func Version() string {
	return "wallet-sdk/mobile/trade 0.3.0 (autoQuote + buildTx + checkTx + submitIntent, device/server hybrid)"
}
