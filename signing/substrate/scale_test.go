package substrate

import (
	"crypto/ed25519"
	"encoding/hex"
	"math/big"
	"testing"
)

func TestSubstrateCodecAgainstOKXVector(t *testing.T) {
	priv, _ := hex.DecodeString("45d3bd794c5bc6ed91ae41c93c0baed679935703dfac72c48d27f8321b8d3a40")
	edKey := ed25519.NewKeyFromSeed(priv)
	pub := []byte(edKey.Public().(ed25519.PublicKey))

	callIndex, _ := hex.DecodeString("0500")
	call := encodeTransferCall(callIndex, pub, big.NewInt(10000000000))
	era := encodeEraMortal(10672081, 64)
	genesis, _ := hex.DecodeString("91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3")
	block, _ := hex.DecodeString("569e9705bdcd3cf15edb1378433148d437f585a21ad0e2691f0d8c0083021580")

	payload := encodeSigningPayload(call, era, big.NewInt(18), big.NewInt(0), 9220, 12, genesis, block, false, false)
	sig := ed25519.Sign(edKey, payload)
	signed := encodeSignedExtrinsic(pub, 0x00, sig, era, big.NewInt(18), big.NewInt(0), call, false, false)

	got := "0x" + hex.EncodeToString(signed)
	want := "0x410284000c2f3c6dabb4a0600eccae87aeaa39242042f9a576aa8dca01e1b419cf17d7a200823181d175794c0438f88340b8f314d1e0e1f0e7fda5b0c0375be35482468ea6284e3831ce67b622322ad984f5a1d1868e7536e4558735fc1c9050443e1c8503150148000500000c2f3c6dabb4a0600eccae87aeaa39242042f9a576aa8dca01e1b419cf17d7a20700e40b5402"
	if got != want {
		t.Fatalf("OKX vector mismatch\ngot  %s\nwant %s", got, want)
	}
}
