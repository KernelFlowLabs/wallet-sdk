package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/KernelFlowLabs/wallet-sdk/mobile/acc"
	"github.com/KernelFlowLabs/wallet-sdk/mobile/tx"
	starknetrpc "github.com/KernelFlowLabs/wallet-sdk/rpc/starknet"
	"github.com/KernelFlowLabs/wallet-sdk/signing"
)

const path = "m/44'/9004'/0'/0/0"

const (
	mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	rpcURL   = "https://rpc.starknet.lava.build"
)

type bounds struct {
	L1GasMaxAmount     string `json:"l1GasMaxAmount"`
	L1GasMaxPrice      string `json:"l1GasMaxPrice"`
	L1DataGasMaxAmount string `json:"l1DataGasMaxAmount"`
	L1DataGasMaxPrice  string `json:"l1DataGasMaxPrice"`
	L2GasMaxAmount     string `json:"l2GasMaxAmount"`
	L2GasMaxPrice      string `json:"l2GasMaxPrice"`
	OverallFee         string `json:"overallFee"`
}

func build(a *acc.StarknetAccount, sender, nonce string, b bounds) (string, string, error) {
	tb := tx.NewStarknetTxBuilder().
		SetTxType(signing.TxTypeTransfer).
		SetContractAddress(signing.MagicContactAddressForNative).
		SetSender(sender).
		SetRecipient(sender).
		SetAmount("1000000000000").
		SetNonce(nonce).
		SetL1GasMaxAmount(b.L1GasMaxAmount).SetL1GasMaxPrice(b.L1GasMaxPrice).
		SetL1DataGasMaxAmount(b.L1DataGasMaxAmount).SetL1DataGasMaxPrice(b.L1DataGasMaxPrice).
		SetL2GasMaxAmount(b.L2GasMaxAmount).SetL2GasMaxPrice(b.L2GasMaxPrice).
		SetTip("0")
	if err := tb.Build(); err != nil {
		return "", "", err
	}
	sig, err := tb.Sign(a.PrivateKey())
	if err != nil {
		return "", "", err
	}
	signed, err := tb.ConcatSignature(sig)
	if err != nil {
		return "", "", err
	}
	return tb.SigHash(), signed, nil
}

func main() {
	ctx := context.Background()

	a, err := acc.NewStarknetFromMnemonic(mnemonic, path)
	if err != nil {
		log.Fatalf("derive account: %v", err)
	}
	sender := a.Address()
	fmt.Println("1. address (ArgentX) from mnemonic:", sender)

	h, err := starknetrpc.NewHandler(rpcURL)
	if err != nil {
		log.Fatalf("rpc handler: %v", err)
	}

	nonce, err := h.InquireChain(ctx, "getNonce", sender)
	if err != nil {
		log.Fatalf("getNonce: %v", err)
	}
	fmt.Println("2. nonce:", nonce)

	prov := bounds{
		L1GasMaxAmount: "1000", L1GasMaxPrice: "100000000000000",
		L1DataGasMaxAmount: "10000", L1DataGasMaxPrice: "100000000000000",
		L2GasMaxAmount: "10000000", L2GasMaxPrice: "100000000000000",
	}
	_, provSigned, err := build(a, sender, nonce, prov)
	if err != nil {
		log.Fatalf("build provisional: %v", err)
	}

	estStr, err := h.InquireChain(ctx, "estimateFee", provSigned)
	if err != nil {
		log.Fatalf("estimateFee (signature validation): %v", err)
	}
	var real bounds
	if err := json.Unmarshal([]byte(estStr), &real); err != nil {
		log.Fatalf("parse estimate: %v", err)
	}
	fmt.Println("3. estimateFee OK -> __validate__ accepted the signature (v3 hash + sig correct)")
	fmt.Printf("   real bounds: l2Amount=%s l2Price=%s overallFee(fri)=%s\n",
		real.L2GasMaxAmount, real.L2GasMaxPrice, real.OverallFee)

	sigHash, finalSigned, err := build(a, sender, nonce, real)
	if err != nil {
		log.Fatalf("build final: %v", err)
	}
	fmt.Println("4. final signed tx built, sigHash:", sigHash)

	txHash, err := h.SendTx(ctx, finalSigned)
	if err != nil {
		fmt.Printf("5. broadcast rejected (expected: account has 0 STRK for v3 fees): %v\n", err)
		return
	}
	fmt.Println("5. broadcasted, txHash:", txHash)
}
