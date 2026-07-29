package starknet

import (
	"encoding/binary"
	"math/big"
)

type resourceBound struct {
	name      string
	maxAmount *big.Int
	maxPrice  *big.Int
}

func (rb resourceBound) pack() *big.Int {
	buf := []byte{0}
	buf = append(buf, []byte(rb.name)...)
	amt := make([]byte, 8)
	binary.BigEndian.PutUint64(amt, rb.maxAmount.Uint64())
	buf = append(buf, amt...)
	price := make([]byte, 16)
	rb.maxPrice.FillBytes(price)
	buf = append(buf, price...)
	return new(big.Int).SetBytes(buf)
}

func shortStringFelt(s string) *big.Int { return new(big.Int).SetBytes([]byte(s)) }

func v3InvokeHash(sender, nonce, chainID *big.Int, calldata []*big.Int, l1, l2, l1data resourceBound, tip uint64) *big.Int {
	tipAndResources := poseidonHashMany([]*big.Int{
		new(big.Int).SetUint64(tip), l1.pack(), l2.pack(), l1data.pack(),
	})
	return poseidonHashMany([]*big.Int{
		shortStringFelt("invoke"),
		big.NewInt(3),
		sender,
		tipAndResources,
		poseidonHashMany(nil),
		chainID,
		nonce,
		big.NewInt(0),
		poseidonHashMany(nil),
		poseidonHashMany(calldata),
	})
}
