package trade

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/kernelflowlabs/wallet-sdk/dex/jupiter"
	"github.com/kernelflowlabs/wallet-sdk/dex/univ2"
	"github.com/kernelflowlabs/wallet-sdk/dex/univ3"
)

var (
	tradeInitMu       sync.Mutex
	tradeMu           sync.RWMutex
	jup               *jupiter.Client
	jupiterAPIKey     string
	jupiterSolanaRPC  string
	jupiterFeeAccount string
	server            string
	bungeeAPIKey      string
	bungeeAffiliateID string
)

func Init(configJSON string) string {
	tradeInitMu.Lock()
	defer tradeInitMu.Unlock()

	var cfg struct {
		JupiterAPIKey     string         `json:"jupiter_api_key"`
		SolanaRPC         string         `json:"solana_rpc"`
		FeeAccount        string         `json:"fee_account"`
		ServerURL         string         `json:"server_url"`
		BungeeAPIKey      string         `json:"bungee_api_key"`
		BungeeAffiliateID string         `json:"bungee_affiliate_id"`
		Univ2             []univ2.Config `json:"univ2,omitempty"`
		Univ3             []univ3.Config `json:"univ3,omitempty"`
	}
	if s := strings.TrimSpace(configJSON); s != "" {
		if err := json.Unmarshal([]byte(s), &cfg); err != nil {
			return fmt.Sprintf("parse config: %v", err)
		}
	}
	tradeMu.RLock()
	client := jup
	reuseJupiter := client != nil &&
		jupiterAPIKey == cfg.JupiterAPIKey &&
		jupiterSolanaRPC == cfg.SolanaRPC &&
		jupiterFeeAccount == cfg.FeeAccount
	currentUniv2 := univ2Clients
	currentUniv3 := univ3Clients
	tradeMu.RUnlock()
	if !reuseJupiter {
		client = jupiter.NewClientWithOptions(cfg.SolanaRPC, cfg.FeeAccount, cfg.JupiterAPIKey)
	}
	univ2ByChain, createdUniv2, err := newUniv2Clients(cfg.Univ2, currentUniv2)
	if err != nil {
		return fmt.Sprintf("configure univ2: %v", err)
	}
	univ3ByChain, _, err := newUniv3Clients(cfg.Univ3, currentUniv3)
	if err != nil {
		closeUniv2ClientList(createdUniv2)
		return fmt.Sprintf("configure univ3: %v", err)
	}
	srv := strings.TrimRight(cfg.ServerURL, "/")

	tradeMu.Lock()
	previousUniv2 := univ2Clients
	previousUniv3 := univ3Clients
	jup = client
	jupiterAPIKey = cfg.JupiterAPIKey
	jupiterSolanaRPC = cfg.SolanaRPC
	jupiterFeeAccount = cfg.FeeAccount
	if server != srv {
		proxy = nil
	}
	server = srv
	if bungeeAPIKey != cfg.BungeeAPIKey || bungeeAffiliateID != cfg.BungeeAffiliateID {
		bng = nil
	}
	bungeeAPIKey = cfg.BungeeAPIKey
	bungeeAffiliateID = cfg.BungeeAffiliateID
	univ2Clients = univ2ByChain
	univ3Clients = univ3ByChain
	tradeMu.Unlock()
	closeRetiredUniv2Clients(previousUniv2, univ2ByChain)
	closeRetiredUniv3Clients(previousUniv3, univ3ByChain)
	return ""
}

func jupClient() *jupiter.Client {
	tradeMu.RLock()
	defer tradeMu.RUnlock()
	return jup
}

func Version() string {
	return "wallet-sdk/mobile/trade 1.0.0 (dynamic univ2/univ3 + EIP-2612 permit + lifecycle-safe reconfiguration)"
}
