package bip

import (
	"fmt"
	"hash/crc32"
	"testing"
)

func TestEnglishWordlistSize(t *testing.T) {
	if len(English) != 2048 {
		t.Fatalf("English wordlist must have 2048 words, got %d", len(English))
	}
}

func TestEnglishWordlistChecksum(t *testing.T) {
	const want = "c1dbd296"
	got := fmt.Sprintf("%x", crc32.ChecksumIEEE([]byte(english)))
	if got != want {
		t.Fatalf("English wordlist checksum changed: got %s, want %s — the wordlist has been altered", got, want)
	}
}

func TestReverseWordMapConsistent(t *testing.T) {
	for i, w := range English {
		if ReverseWordMap[w] != i {
			t.Fatalf("ReverseWordMap[%q] = %d, want %d", w, ReverseWordMap[w], i)
		}
	}
}
