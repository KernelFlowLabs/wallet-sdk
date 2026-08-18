package trade

import (
	"encoding/json"
	"fmt"
)

type submitReq struct {
	Channel       string          `json:"channel"`
	Request       json.RawMessage `json:"request"`
	UserSignature string          `json:"user_signature"`
	RequestHash   string          `json:"request_hash"`
	TimeoutMs     int             `json:"timeout_ms"`
}

type submitResp struct {
	Hash  string `json:"hash,omitempty"`
	Error string `json:"error,omitempty"`
}

func SubmitIntent(reqJSON string) string {
	var req submitReq
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return marshal(&submitResp{Error: fmt.Sprintf("parse request: %v", err)})
	}
	switch req.Channel {
	case "bungee", "jupiter", "lifi", "univ2", "univ3":
		// Bungee intent mode is gone upstream (Socket V3 is tx-mode only).
		return marshal(&submitResp{Error: fmt.Sprintf("channel %q has no intent submission — read route.txData, sign, broadcast normally", req.Channel)})
	case "":
		return marshal(&submitResp{Error: "channel required"})
	}
	return marshal(&submitResp{Error: fmt.Sprintf("unknown channel: %q", req.Channel)})
}
