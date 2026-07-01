package cosmos

import (
	"context"
	"fmt"
)

func (h *Handler) getHeightSei(ctx context.Context) (string, error) {
	out := &_SeiStatus{}
	path := "status"
	err := h.rpc.Get(ctx, out, path, nil)
	if err != nil {
		return "", fmt.Errorf("fail to get latest block,err=%s", err)
	} else if out.SyncInfo.LatestBlockHeight == "0" || out.SyncInfo.LatestBlockHeight == "" {
		return "", fmt.Errorf("fail to get latest block, height==0")
	}
	return out.SyncInfo.LatestBlockHeight, nil
}

type _SeiStatus struct {
	SyncInfo struct {
		LatestBlockHeight string `json:"latest_block_height"`
	} `json:"sync_info"`
}
