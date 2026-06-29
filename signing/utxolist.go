package signing

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type (
	UtxoList struct {
		List []*UtxoInfo `json:"list" validate:"required,min=1,dive"`
	}
	UtxoInfo struct {
		Hash   string `json:"hash" validate:"required,hex_str"`
		Script string `json:"script" validate:"required,hex_str"`
		Index  string `json:"index" validate:"required,u64"`
		Value  string `json:"value" validate:"required,u64_gt0"`

		Version       string `json:"version,omitempty" validate:"omitempty,u64"`
		IsCoinbase    string `json:"isCoinbase,omitempty" validate:"omitempty,bool_str"`
		BlockDAAScore string `json:"blockDAAScore,omitempty" validate:"omitempty,u64"`
	}
)

func NewUtxoList() *UtxoList {
	return &UtxoList{
		List: make([]*UtxoInfo, 0),
	}
}

func (u *UtxoList) SerializeFromStr(jsonStr string) error {
	if u == nil {
		return fmt.Errorf("UtxoList is nil")
	}

	err := json.Unmarshal([]byte(jsonStr), u)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

func (u *UtxoList) AddUtxoInfo(info *UtxoInfo) {
	if info == nil {
		return
	}

	if u == nil {
		return
	}

	if (*u).List == nil {
		(*u).List = make([]*UtxoInfo, 0)
	}

	(*u).List = append((*u).List, info)
}

func (u *UtxoList) SelectUtxo(targetValue string) error {
	if u == nil || u.List == nil {
		return fmt.Errorf("UtxoList is nil or empty")
	}

	target, err := strconv.ParseUint(targetValue, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target value: %w", err)
	}

	sort.Slice(u.List, func(i, j int) bool {
		vi, _ := strconv.ParseUint(u.List[i].Value, 10, 64)
		vj, _ := strconv.ParseUint(u.List[j].Value, 10, 64)
		return vi < vj
	})

	var selectedUtxos []*UtxoInfo
	var totalValue uint64 = 0

	for _, utxo := range u.List {
		if totalValue >= target {
			break
		}

		value, err := strconv.ParseUint(utxo.Value, 10, 64)
		if err != nil {
			continue
		}

		selectedUtxos = append(selectedUtxos, utxo)
		totalValue += value
	}

	if totalValue < target {
		return fmt.Errorf("insufficient balance: need %s, have %d", targetValue, totalValue)
	}

	u.List = selectedUtxos

	return nil
}

func (u *UtxoList) CalcValue() string {
	if u == nil || u.List == nil {
		return "0"
	}
	var totalValue uint64 = 0
	for _, utxo := range u.List {
		if utxo == nil {
			continue
		}
		value, err := strconv.ParseUint(utxo.Value, 10, 64)
		if err != nil {
			return "0"
		}

		totalValue += value
	}

	return strconv.FormatUint(totalValue, 10)
}

func (u *UtxoList) IsEmpty() bool {
	if u == nil {
		return true
	} else if len(u.List) == 0 {
		return true
	}
	return false
}

func (u *UtxoList) String() string {
	if u == nil || len(u.List) == 0 {
		return "UtxoList is empty"
	}

	var sb strings.Builder
	for i, utxo := range u.List {
		fmt.Fprintf(&sb, "[%d]\n", i)
		fmt.Fprintf(&sb, "  Hash:          %s\n", utxo.Hash)
		fmt.Fprintf(&sb, "  Index:         %s\n", utxo.Index)
		fmt.Fprintf(&sb, "  Value:         %s\n", utxo.Value)
		fmt.Fprintf(&sb, "  Script:        %s\n", utxo.Script)

		if utxo.Version != "" {
			fmt.Fprintf(&sb, "  Version:       %s\n", utxo.Version)
		}
		if utxo.IsCoinbase != "" {
			fmt.Fprintf(&sb, "  IsCoinbase:    %s\n", utxo.IsCoinbase)
		}
		if utxo.BlockDAAScore != "" {
			fmt.Fprintf(&sb, "  BlockDAAScore: %s\n", utxo.BlockDAAScore)
		}
	}
	return sb.String()
}
