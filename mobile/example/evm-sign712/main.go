package main

import (
	"fmt"
	"log"

	"github.com/KernelFlowLabs/wallet-sdk/mobile/acc"
	"github.com/KernelFlowLabs/wallet-sdk/mobile/example/shared"
)

const path = "m/44'/60'/0'/0/0"

const typedData = `{
  "types": {
    "EIP712Domain": [
      {"name": "name", "type": "string"},
      {"name": "version", "type": "string"},
      {"name": "chainId", "type": "uint256"},
      {"name": "verifyingContract", "type": "address"}
    ],
    "Person": [
      {"name": "name", "type": "string"},
      {"name": "wallet", "type": "address"}
    ],
    "Mail": [
      {"name": "from", "type": "Person"},
      {"name": "to", "type": "Person"},
      {"name": "contents", "type": "string"}
    ]
  },
  "primaryType": "Mail",
  "domain": {
    "name": "Ether Mail",
    "version": "1",
    "chainId": 1,
    "verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
  },
  "message": {
    "from": {"name": "Cow", "wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"},
    "to": {"name": "Bob", "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"},
    "contents": "Hello, Bob!"
  }
}`

func main() {
	a, err := acc.NewEvmFromMnemonic(shared.Mnemonic(), path)
	if err != nil {
		log.Fatalf("derive account: %v", err)
	}
	fmt.Println("1. address from mnemonic:", a.Address())

	sig, err := a.SignTypedDataJSON(typedData)
	if err != nil {
		log.Fatalf("sign typed data: %v", err)
	}
	fmt.Println("2. eip-712 signature:", sig)
}
