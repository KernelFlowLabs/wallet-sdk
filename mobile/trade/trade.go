package trade

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/kernelflowlabs/wallet-sdk/dex/jupiter"
)

var (
	initOnce sync.Once
	initErr  string
	jup      *jupiter.Client
	server   string
)

func Init(configJSON string) string {
	initOnce.Do(func() {
		var cfg struct {
			JupiterAPIKey string `json:"jupiter_api_key"`
			SolanaRPC     string `json:"solana_rpc"`
			FeeAccount    string `json:"fee_account"`
			ServerURL     string `json:"server_url"`
		}
		if s := strings.TrimSpace(configJSON); s != "" {
			if err := json.Unmarshal([]byte(s), &cfg); err != nil {
				initErr = fmt.Sprintf("parse config: %v", err)
				return
			}
		}
		jup = jupiter.NewClientWithOptions(cfg.SolanaRPC, cfg.FeeAccount, cfg.JupiterAPIKey)
		server = strings.TrimRight(cfg.ServerURL, "/")
	})
	return initErr
}

func Version() string {
	return "wallet-sdk/mobile/trade 0.3.0 (autoQuote + buildTx + checkTx + submitIntent, device/server hybrid)"
}
