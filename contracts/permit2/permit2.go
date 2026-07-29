package permit2

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

const Address = "0x000000000022D473030F116dDEE9F6B43aC78BA3"
const DomainName = "Permit2"

const ABI = `[
{"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint160","name":"amount","type":"uint160"},{"internalType":"uint48","name":"expiration","type":"uint48"}],"name":"approve","outputs":[],"stateMutability":"nonpayable","type":"function"},
{"inputs":[{"components":[{"components":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"internalType":"struct ISignatureTransfer.TokenPermissions","name":"permitted","type":"tuple"},{"internalType":"uint256","name":"nonce","type":"uint256"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"internalType":"struct ISignatureTransfer.PermitTransferFrom","name":"permit","type":"tuple"},{"components":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"requestedAmount","type":"uint256"}],"internalType":"struct ISignatureTransfer.SignatureTransferDetails","name":"transferDetails","type":"tuple"},{"internalType":"address","name":"owner","type":"address"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"permitTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"}
]`

type TokenPermissions struct {
	Token  string `json:"token"`
	Amount string `json:"amount"`
}

type PermitTransferFrom struct {
	Permitted TokenPermissions `json:"permitted"`
	Spender   string           `json:"spender"`
	Nonce     string           `json:"nonce"`
	Deadline  string           `json:"deadline"`
}

type PermitBatchTransferFrom struct {
	Permitted []TokenPermissions `json:"permitted"`
	Spender   string             `json:"spender"`
	Nonce     string             `json:"nonce"`
	Deadline  string             `json:"deadline"`
}

type TypedData struct {
	Types       map[string][]TypedField `json:"types"`
	PrimaryType string                  `json:"primaryType"`
	Domain      Domain                  `json:"domain"`
	Message     map[string]interface{}  `json:"message"`
}

type TypedField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Domain struct {
	Name              string `json:"name"`
	ChainId           string `json:"chainId"`
	VerifyingContract string `json:"verifyingContract"`
}

func BuildPermitTransferFrom(chainId int64, p PermitTransferFrom) (*TypedData, error) {
	if err := validate(chainId, p.Permitted.Token, p.Permitted.Amount, p.Spender, p.Nonce, p.Deadline); err != nil {
		return nil, err
	}
	return &TypedData{
		Types: map[string][]TypedField{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"PermitTransferFrom": {
				{Name: "permitted", Type: "TokenPermissions"},
				{Name: "spender", Type: "address"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
			},
			"TokenPermissions": {
				{Name: "token", Type: "address"},
				{Name: "amount", Type: "uint256"},
			},
		},
		PrimaryType: "PermitTransferFrom",
		Domain: Domain{
			Name:              DomainName,
			ChainId:           big.NewInt(chainId).String(),
			VerifyingContract: Address,
		},
		Message: map[string]interface{}{
			"permitted": map[string]interface{}{
				"token":  p.Permitted.Token,
				"amount": p.Permitted.Amount,
			},
			"spender":  p.Spender,
			"nonce":    p.Nonce,
			"deadline": p.Deadline,
		},
	}, nil
}

func BuildPermitBatchTransferFrom(chainId int64, p PermitBatchTransferFrom) (*TypedData, error) {
	if len(p.Permitted) == 0 {
		return nil, fmt.Errorf("permit2 batch: empty permitted list")
	}
	if err := validate(chainId, "", "", p.Spender, p.Nonce, p.Deadline); err != nil {
		return nil, err
	}
	perms := make([]map[string]interface{}, 0, len(p.Permitted))
	for i, tp := range p.Permitted {
		if tp.Token == "" || !isDecimal(tp.Amount) {
			return nil, fmt.Errorf("permit2 batch entry %d: invalid token/amount", i)
		}
		perms = append(perms, map[string]interface{}{
			"token":  tp.Token,
			"amount": tp.Amount,
		})
	}
	return &TypedData{
		Types: map[string][]TypedField{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"PermitBatchTransferFrom": {
				{Name: "permitted", Type: "TokenPermissions[]"},
				{Name: "spender", Type: "address"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
			},
			"TokenPermissions": {
				{Name: "token", Type: "address"},
				{Name: "amount", Type: "uint256"},
			},
		},
		PrimaryType: "PermitBatchTransferFrom",
		Domain: Domain{
			Name:              DomainName,
			ChainId:           big.NewInt(chainId).String(),
			VerifyingContract: Address,
		},
		Message: map[string]interface{}{
			"permitted": perms,
			"spender":   p.Spender,
			"nonce":     p.Nonce,
			"deadline":  p.Deadline,
		},
	}, nil
}

func MarshalTypedData(t *TypedData) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil typed data")
	}
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func validate(chainId int64, token, amount, spender, nonce, deadline string) error {
	if chainId <= 0 {
		return fmt.Errorf("permit2: chainId required")
	}
	if token != "" && !isAddr(token) {
		return fmt.Errorf("permit2: invalid token address")
	}
	if amount != "" && !isDecimal(amount) {
		return fmt.Errorf("permit2: invalid amount")
	}
	if !isAddr(spender) {
		return fmt.Errorf("permit2: invalid spender address")
	}
	if !isDecimal(nonce) {
		return fmt.Errorf("permit2: invalid nonce")
	}
	if !isDecimal(deadline) {
		return fmt.Errorf("permit2: invalid deadline")
	}
	return nil
}

func isAddr(s string) bool {
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func isDecimal(s string) bool {
	if s == "" {
		return false
	}
	_, ok := new(big.Int).SetString(s, 10)
	return ok
}
