package trade

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/kernelflowlabs/wallet-sdk/common/dexmodel"
)

type checkTxReq struct {
	Channel   string `json:"channel"`
	Via       string `json:"via"`
	HashType  string `json:"hash_type"`
	Hash      string `json:"hash"`
	FromChain string `json:"from_chain"`
	ToChain   string `json:"to_chain"`
	Bridge    string `json:"bridge,omitempty"`
	TimeoutMs int    `json:"timeout_ms"`
}

type checkTxResp struct {
	Channel string `json:"channel"`
	Status  string `json:"status"`
	ToChain string `json:"toChain,omitempty"`
	ToHash  string `json:"toHash,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Error   string `json:"error,omitempty"`
}

func CheckTx(reqJSON string) string {
	var req checkTxReq
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return marshal(&checkTxResp{Error: fmt.Sprintf("parse request: %v", err)})
	}
	if req.Channel == "" || req.Hash == "" || req.HashType == "" {
		return marshal(&checkTxResp{Error: "channel, hash, hash_type required"})
	}
	timeout := 8 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if req.Via == "server" {
		return checkTxViaServer(ctx, &req)
	}
	in := &dexmodel.DexCheckTxIn{
		Channel:   req.Channel,
		HashType:  dexmodel.DexHashType(req.HashType),
		Hash:      req.Hash,
		FromChain: req.FromChain,
		ToChain:   req.ToChain,
		Bridge:    req.Bridge,
	}
	var out *dexmodel.DexCheckTxOut
	var err error
	switch req.Channel {
	case "jupiter":
		if jup == nil {
			return marshal(&checkTxResp{Error: "jupiter not initialized — call Init"})
		}
		out, err = jup.Status(ctx, in)
	case "bungee":
		out, err = autoQuoteBungee().Status(ctx, in)
	case "lifi":
		out, err = autoQuoteLiFi().Status(ctx, in)
	default:
		return marshal(&checkTxResp{Error: fmt.Sprintf("unknown channel: %q", req.Channel)})
	}
	if err != nil {
		return marshal(&checkTxResp{Error: err.Error()})
	}
	if out == nil {
		return marshal(&checkTxResp{Error: "empty response"})
	}
	return marshal(&checkTxResp{
		Channel: out.Channel,
		Status:  string(out.Status),
		ToChain: out.ToChain,
		ToHash:  out.ToHash,
		Msg:     out.Msg,
	})
}

func checkTxViaServer(ctx context.Context, req *checkTxReq) string {
	p := serverProxy()
	if p == nil {
		return marshal(&checkTxResp{Error: "via=server but server_url not configured — pass it to Init"})
	}
	q := url.Values{}
	q.Set("channel", req.Channel)
	q.Set("hashType", req.HashType)
	q.Set("hash", req.Hash)
	q.Set("fromChain", req.FromChain)
	q.Set("toChain", req.ToChain)
	if req.Bridge != "" {
		q.Set("bridge", req.Bridge)
	}
	var env struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Status  string `json:"status"`
			ToChain string `json:"toChain"`
			ToHash  string `json:"toHash"`
		} `json:"data"`
	}
	if err := p.Get(ctx, &env, "public/trade/checkTx", q); err != nil {
		return marshal(&checkTxResp{Error: fmt.Sprintf("proxy checkTx: %v", err)})
	}
	if env.Code != 0 {
		return marshal(&checkTxResp{Error: fmt.Sprintf("proxy code=%d msg=%s", env.Code, env.Msg)})
	}
	return marshal(&checkTxResp{
		Channel: req.Channel,
		Status:  env.Data.Status,
		ToChain: env.Data.ToChain,
		ToHash:  env.Data.ToHash,
	})
}
