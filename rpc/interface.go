package chainrpc

import "context"

type BasicChainHandler interface {
	GetHeight(ctx context.Context) (string, error)
	GetBalance(ctx context.Context, address, contractAddress, blockNumber string) (string, error)
	SendTx(ctx context.Context, signedHex string) (string, error)
	CheckTx(ctx context.Context, hash string) (*TxResult, error)
	CallContract(ctx context.Context, contractAddress, params, blockNumber string) ([]byte, error)
	InquireChain(ctx context.Context, instruction, params string) (string, error)
}
