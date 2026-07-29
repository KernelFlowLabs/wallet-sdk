package substrate

import "math/big"

func encodeCompact(v *big.Int) []byte {
	if v == nil || v.Sign() <= 0 {
		return []byte{0}
	}
	if v.BitLen() <= 30 {
		u := v.Uint64()
		switch {
		case u < 1<<6:
			return []byte{byte(u) << 2}
		case u < 1<<14:
			x := uint16(u)<<2 | 0b01
			return []byte{byte(x), byte(x >> 8)}
		default:
			x := uint32(u)<<2 | 0b10
			return []byte{byte(x), byte(x >> 8), byte(x >> 16), byte(x >> 24)}
		}
	}
	be := v.Bytes()
	le := make([]byte, len(be))
	for i := range be {
		le[len(be)-1-i] = be[i]
	}
	return append([]byte{byte((len(le)-4)<<2 | 0b11)}, le...)
}

func encodeU32LE(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}

func encodeEraImmortal() []byte {
	return []byte{0x00}
}

func encodeEraMortal(height, period uint64) []byte {
	if period == 0 {
		period = 64
	}
	phase := height % period
	tz := uint64(0)
	for p := period; p > 0 && p&1 == 0; p >>= 1 {
		tz++
	}
	exp := tz
	if exp >= 1 {
		exp--
	}
	if exp < 1 {
		exp = 1
	}
	if exp > 15 {
		exp = 15
	}
	quant := period >> 12
	if quant < 1 {
		quant = 1
	}
	encoded := exp | ((phase / quant) << 4)
	return []byte{byte(encoded), byte(encoded >> 8)}
}
