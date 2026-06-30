package shared

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func Mnemonic() string {
	return MnemonicAt(0)
}

func RPC(chain string) (url, network string) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		panic("shared: cannot locate source path")
	}
	rpcPath := filepath.Join(filepath.Dir(self), "..", ".rpc")
	data, err := os.ReadFile(rpcPath)
	if err != nil {
		panic("shared: read " + rpcPath + ": " + err.Error() + " (create example/.rpc with lines: <chain> <rpcURL> <network>)")
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == chain {
			return fields[1], fields[2]
		}
	}
	panic("shared: no rpc entry for chain " + chain + " in example/.rpc")
}

func MnemonicAt(index int) string {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		panic("shared: cannot locate source path")
	}
	mnPath := filepath.Join(filepath.Dir(self), "..", ".mn")
	data, err := os.ReadFile(mnPath)
	if err != nil {
		panic("shared: read " + mnPath + ": " + err.Error() + " (create example/.mn with your test mnemonic)")
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if s := strings.TrimSpace(line); s != "" {
			lines = append(lines, s)
		}
	}
	if index < 0 || index >= len(lines) {
		panic("shared: no mnemonic at index in example/.mn")
	}
	return lines[index]
}
