// This file is derived from github.com/tyler-smith/go-bip39 (MIT License,
// Copyright (c) 2014 Tyler Smith), with local modifications (e.g.
// NewSeedFromMnemonicSr25519). It is vendored intentionally so the BIP-39
// implementation and wordlist can never change out from under the wallet.
// See crypto/bip/LICENSE.go-bip39 for the upstream license.

package bip

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

var (
	last11BitsMask                 = big.NewInt(2047)
	shift11BitsMask                = big.NewInt(2048)
	bigOne                         = big.NewInt(1)
	bigTwo                         = big.NewInt(2)
	wordLengthChecksumMasksMapping = map[int]*big.Int{
		12: big.NewInt(15),
		15: big.NewInt(31),
		18: big.NewInt(63),
		21: big.NewInt(127),
		24: big.NewInt(255),
	}
	wordLengthChecksumShiftMapping = map[int]*big.Int{
		12: big.NewInt(16),
		15: big.NewInt(8),
		18: big.NewInt(4),
		21: big.NewInt(2),
	}
	wordList []string
	wordMap  map[string]int
)

var (
	ErrInvalidMnemonic             = errors.New("Invalid mnenomic")
	ErrEntropyLengthInvalid        = errors.New("Entropy length must be [128, 256] and a multiple of 32")
	ErrValidatedSeedLengthMismatch = errors.New("Seed length does not match validated seed length")
	ErrChecksumIncorrect           = errors.New("Checksum incorrect")
)

func init() {
	SetWordList(English)
}

func SetWordList(list []string) {
	wordList = list
	wordMap = map[string]int{}
	for i, v := range wordList {
		wordMap[v] = i
	}
}

func GetWordList() []string {
	return wordList
}

func NewEntropy(bitSize int) ([]byte, error) {
	err := validateEntropyBitSize(bitSize)
	if err != nil {
		return nil, err
	}

	entropy := make([]byte, bitSize/8)
	_, err = rand.Read(entropy)
	return entropy, err
}

func EntropyFromMnemonic(mnemonic string) ([]byte, error) {
	mnemonicSlice, isValid := splitMnemonicWords(mnemonic)
	if !isValid {
		return nil, ErrInvalidMnemonic
	}
	b := big.NewInt(0)
	for _, v := range mnemonicSlice {
		index, found := wordMap[v]
		if found == false {
			return nil, fmt.Errorf("word `%v` not found in reverse map", v)
		}
		var wordBytes [2]byte
		binary.BigEndian.PutUint16(wordBytes[:], uint16(index))
		b = b.Mul(b, shift11BitsMask)
		b = b.Or(b, big.NewInt(0).SetBytes(wordBytes[:]))
	}

	checksum := big.NewInt(0)
	checksumMask := wordLengthChecksumMasksMapping[len(mnemonicSlice)]
	checksum = checksum.And(b, checksumMask)

	b.Div(b, big.NewInt(0).Add(checksumMask, bigOne))

	entropy := b.Bytes()
	entropy = padByteSlice(entropy, len(mnemonicSlice)/3*4)

	entropyChecksumBytes := computeChecksum(entropy)
	entropyChecksum := big.NewInt(int64(entropyChecksumBytes[0]))
	if l := len(mnemonicSlice); l != 24 {
		checksumShift := wordLengthChecksumShiftMapping[l]
		entropyChecksum.Div(entropyChecksum, checksumShift)
	}

	if checksum.Cmp(entropyChecksum) != 0 {
		return nil, ErrChecksumIncorrect
	}

	return entropy, nil
}

func NewMnemonic(entropy []byte) (string, error) {
	entropyBitLength := len(entropy) * 8
	checksumBitLength := entropyBitLength / 32
	sentenceLength := (entropyBitLength + checksumBitLength) / 11

	err := validateEntropyBitSize(entropyBitLength)
	if err != nil {
		return "", err
	}

	entropy = addChecksum(entropy)
	entropyInt := new(big.Int).SetBytes(entropy)
	words := make([]string, sentenceLength)
	word := big.NewInt(0)

	for i := sentenceLength - 1; i >= 0; i-- {
		word.And(entropyInt, last11BitsMask)
		entropyInt.Div(entropyInt, shift11BitsMask)
		wordBytes := padByteSlice(word.Bytes(), 2)
		words[i] = wordList[binary.BigEndian.Uint16(wordBytes)]
	}

	return strings.Join(words, " "), nil
}

func MnemonicToByteArray(mnemonic string, raw ...bool) ([]byte, error) {
	var (
		mnemonicSlice    = strings.Split(mnemonic, " ")
		entropyBitSize   = len(mnemonicSlice) * 11
		checksumBitSize  = entropyBitSize % 32
		fullByteSize     = (entropyBitSize-checksumBitSize)/8 + 1
		checksumByteSize = fullByteSize - (fullByteSize % 4)
	)
	if !IsMnemonicValid(mnemonic) {
		return nil, ErrInvalidMnemonic
	}

	checksummedEntropy := big.NewInt(0)
	modulo := big.NewInt(2048)
	for _, v := range mnemonicSlice {
		index := big.NewInt(int64(wordMap[v]))
		checksummedEntropy.Mul(checksummedEntropy, modulo)
		checksummedEntropy.Add(checksummedEntropy, index)
	}

	checksumModulo := big.NewInt(0).Exp(bigTwo, big.NewInt(int64(checksumBitSize)), nil)
	rawEntropy := big.NewInt(0).Div(checksummedEntropy, checksumModulo)

	rawEntropyBytes := padByteSlice(rawEntropy.Bytes(), checksumByteSize)
	checksummedEntropyBytes := padByteSlice(checksummedEntropy.Bytes(), fullByteSize)

	newChecksummedEntropyBytes := padByteSlice(addChecksum(rawEntropyBytes), fullByteSize)
	if !compareByteSlices(checksummedEntropyBytes, newChecksummedEntropyBytes) {
		return nil, ErrChecksumIncorrect
	}

	if len(raw) > 0 && raw[0] {
		return rawEntropyBytes, nil
	}

	return checksummedEntropyBytes, nil
}

func NewSeedWithErrorChecking(mnemonic string, password string) ([]byte, error) {
	_, err := MnemonicToByteArray(mnemonic)
	if err != nil {
		return nil, err
	}
	return NewSeed(mnemonic, password), nil
}

func NewSeed(mnemonic string, password string) []byte {
	mnemonic = strings.Join(strings.Fields(mnemonic), " ")
	return pbkdf2.Key([]byte(mnemonic), []byte("mnemonic"+password), 2048, 64, sha512.New)
}

func IsMnemonicValid(mnemonic string) bool {
	_, err := EntropyFromMnemonic(mnemonic)
	return err == nil
}

func addChecksum(data []byte) []byte {
	hash := computeChecksum(data)
	firstChecksumByte := hash[0]

	checksumBitLength := uint(len(data) / 4)

	dataBigInt := new(big.Int).SetBytes(data)
	for i := uint(0); i < checksumBitLength; i++ {
		dataBigInt.Mul(dataBigInt, bigTwo)
		if uint8(firstChecksumByte&(1<<(7-i))) > 0 {
			dataBigInt.Or(dataBigInt, bigOne)
		}
	}

	return dataBigInt.Bytes()
}

func computeChecksum(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func validateEntropyBitSize(bitSize int) error {
	if (bitSize%32) != 0 || bitSize < 128 || bitSize > 256 {
		return ErrEntropyLengthInvalid
	}
	return nil
}

func padByteSlice(slice []byte, length int) []byte {
	offset := length - len(slice)
	if offset <= 0 {
		return slice
	}
	newSlice := make([]byte, length)
	copy(newSlice[offset:], slice)
	return newSlice
}

func compareByteSlices(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func splitMnemonicWords(mnemonic string) ([]string, bool) {
	tmp := strings.Fields(mnemonic)

	var words []string
	for _, v := range tmp {
		word := strings.TrimSpace(v)
		words = append(words, word)
	}

	numOfWords := len(words)

	if numOfWords%3 != 0 || numOfWords < 12 || numOfWords > 24 {
		return nil, false
	}
	return words, true
}

func NewSeedFromMnemonicSr25519(mnemonic string, password string) ([64]byte, error) {
	entropy, err := EntropyFromMnemonic(mnemonic)
	if err != nil {
		return [64]byte{}, err
	}

	if len(entropy) < 16 || len(entropy) > 32 || len(entropy)%4 != 0 {
		return [64]byte{}, errors.New("invalid entropy")
	}

	bz := pbkdf2.Key(entropy, []byte("mnemonic"+password), 2048, 64, sha512.New)
	var bzArr [64]byte
	copy(bzArr[:], bz[:64])

	return bzArr, nil
}
