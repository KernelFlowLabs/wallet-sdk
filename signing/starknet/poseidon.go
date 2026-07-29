package starknet

import (
	"math/big"
	"sync"
)

var starkFieldPrime = func() *big.Int {
	p := new(big.Int).Lsh(big.NewInt(1), 251)
	p.Add(p, new(big.Int).Lsh(big.NewInt(17), 192))
	return p.Add(p, big.NewInt(1))
}()

const (
	poseidonFullRounds    = 8
	poseidonPartialRounds = 83
)

var (
	poseidonKeysOnce sync.Once
	poseidonKeys     [91][3]*big.Int
)

func loadPoseidonKeys() {
	for r := range poseidonRoundKeysDec {
		for j := 0; j < 3; j++ {
			n, ok := new(big.Int).SetString(poseidonRoundKeysDec[r][j], 10)
			if !ok {
				panic("starknet: bad poseidon round key")
			}
			poseidonKeys[r][j] = n
		}
	}
}

func fMod(x *big.Int) *big.Int {
	x.Mod(x, starkFieldPrime)
	if x.Sign() < 0 {
		x.Add(x, starkFieldPrime)
	}
	return x
}

func fAdd(a, b *big.Int) *big.Int { return fMod(new(big.Int).Add(a, b)) }
func fSub(a, b *big.Int) *big.Int { return fMod(new(big.Int).Sub(a, b)) }
func fMul(a, b *big.Int) *big.Int { return fMod(new(big.Int).Mul(a, b)) }

func fCube(x *big.Int) *big.Int { return fMul(fMul(x, x), x) }

func hadesPermutation(s [3]*big.Int) [3]*big.Int {
	poseidonKeysOnce.Do(loadPoseidonKeys)
	total := poseidonFullRounds + poseidonPartialRounds
	for i := 0; i < total; i++ {
		full := i < poseidonFullRounds/2 || total-i <= poseidonFullRounds/2

		s[0] = fAdd(s[0], poseidonKeys[i][0])
		s[1] = fAdd(s[1], poseidonKeys[i][1])
		s[2] = fAdd(s[2], poseidonKeys[i][2])

		s[2] = fCube(s[2])
		if full {
			s[0] = fCube(s[0])
			s[1] = fCube(s[1])
		}

		sum := fAdd(fAdd(s[0], s[1]), s[2])
		s0 := fAdd(sum, fMul(s[0], big.NewInt(2)))
		s1 := fSub(sum, fMul(s[1], big.NewInt(2)))
		s2 := fSub(sum, fMul(s[2], big.NewInt(3)))
		s[0], s[1], s[2] = s0, s1, s2
	}
	return s
}

func poseidonHashMany(elems []*big.Int) *big.Int {
	s := [3]*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0)}
	n := len(elems)
	for i := 0; i < n/2; i++ {
		s[0] = fAdd(s[0], elems[2*i])
		s[1] = fAdd(s[1], elems[2*i+1])
		s = hadesPermutation(s)
	}
	rem := n % 2
	if rem == 1 {
		s[0] = fAdd(s[0], elems[n-1])
	}
	s[rem] = fAdd(s[rem], big.NewInt(1))
	s = hadesPermutation(s)
	return s[0]
}
