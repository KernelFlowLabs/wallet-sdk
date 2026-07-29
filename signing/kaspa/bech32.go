package kaspa

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func encodeAddress(prefix string, payload []byte, version byte) string {
	data := make([]byte, len(payload)+1)
	data[0] = version
	copy(data[1:], payload)
	converted := convertBits(data, eightToFiveBits)
	return encode(prefix, converted)
}

func convertBits(data []byte, conversionType conversionType) []byte {
	var regrouped []byte
	nextByte := byte(0)
	filledBits := uint8(0)

	for _, b := range data {
		b = b << (8 - conversionType.fromBits)
		remainingFromBits := conversionType.fromBits
		for remainingFromBits > 0 {
			remainingToBits := conversionType.toBits - filledBits
			toExtract := remainingFromBits
			if remainingToBits < toExtract {
				toExtract = remainingToBits
			}
			nextByte = (nextByte << toExtract) | (b >> (8 - toExtract))
			b = b << toExtract
			remainingFromBits -= toExtract
			filledBits += toExtract
			if filledBits == conversionType.toBits {
				regrouped = append(regrouped, nextByte)
				filledBits = 0
				nextByte = 0
			}
		}
	}
	if conversionType.pad && filledBits > 0 {
		nextByte = nextByte << (conversionType.toBits - filledBits)
		regrouped = append(regrouped, nextByte)
		filledBits = 0
		nextByte = 0
	}
	return regrouped
}

func encode(prefix string, data []byte) string {
	checksum := calculateChecksum(prefix, data)
	combined := append(data, checksum...)
	base32String := encodeToBase32(combined)

	return fmt.Sprintf("%s:%s", prefix, base32String)
}

func calculateChecksum(prefix string, payload []byte) []byte {
	prefixLower5Bits := prefixToUint5Array(prefix)
	payloadInts := ints(payload)
	templateZeroes := []int{0, 0, 0, 0, 0, 0, 0, 0}

	concat := append(prefixLower5Bits, 0)
	concat = append(concat, payloadInts...)
	concat = append(concat, templateZeroes...)

	polyModResult := polyMod(concat)
	var res []byte
	for i := 0; i < checksumLength; i++ {
		res = append(res, byte((polyModResult>>uint(5*(checksumLength-1-i)))&31))
	}

	return res
}

func polyMod(values []int) int {
	var generator = []int64{}
	for _, v := range generatorStr {
		vv, _ := strconv.ParseInt(v, 10, 64)
		generator = append(generator, vv)
	}
	var valuesTmp []int64
	for _, v := range values {
		valuesTmp = append(valuesTmp, int64(v))
	}

	checksum := int64(1)
	for _, value := range valuesTmp {
		topBits := checksum >> 35
		checksum = ((checksum & int64(0x07ffffffff)) << 5) ^ value
		for i := 0; i < len(generator); i++ {
			if ((topBits >> uint(i)) & 1) == 1 {
				checksum ^= generator[i]
			}
		}
	}

	return int(checksum ^ 1)
}

func encodeToBase32(data []byte) string {
	result := make([]byte, 0, len(data))
	for _, b := range data {
		if int(b) >= len(charset) {
			return ""
		}
		result = append(result, charset[b])
	}
	return string(result)
}

func prefixToUint5Array(prefix string) []int {
	prefixLower5Bits := make([]int, len(prefix))
	for i := 0; i < len(prefix); i++ {
		char := prefix[i]
		charLower5Bits := int(char & 31)
		prefixLower5Bits[i] = charLower5Bits
	}

	return prefixLower5Bits
}

func ints(payload []byte) []int {
	payloadInts := make([]int, len(payload))
	for i, b := range payload {
		payloadInts[i] = int(b)
	}

	return payloadInts
}

func verifyChecksum(prefix string, payload []byte) bool {
	prefixLower5Bits := prefixToUint5Array(prefix)
	payloadInts := ints(payload)
	dataToVerify := append(prefixLower5Bits, 0)
	dataToVerify = append(dataToVerify, payloadInts...)
	return polyMod(dataToVerify) == 0
}

func decodeFromBase32(base32String string) ([]byte, error) {
	decoded := make([]byte, 0, len(base32String))
	for i := 0; i < len(base32String); i++ {
		index := strings.IndexByte(charset, base32String[i])
		if index < 0 {
			return nil, errors.Errorf("invalid character not part of "+
				"charset: %c", base32String[i])
		}
		decoded = append(decoded, byte(index))
	}
	return decoded, nil
}

func decode(encoded string) (string, []byte, error) {
	if len(encoded) < checksumLength+2 {
		return "", nil, errors.Errorf("invalid bech32 string length %d",
			len(encoded))
	}
	for i := 0; i < len(encoded); i++ {
		if encoded[i] < 33 || encoded[i] > 126 {
			return "", nil, errors.Errorf("invalid character in "+
				"string: '%c'", encoded[i])
		}
	}
	lower := strings.ToLower(encoded)
	upper := strings.ToUpper(encoded)
	if encoded != lower && encoded != upper {
		return "", nil, errors.Errorf("string not all lowercase or all " +
			"uppercase")
	}
	encoded = lower
	colonIndex := strings.LastIndexByte(encoded, ':')
	if colonIndex < 1 || colonIndex+checksumLength+1 > len(encoded) {
		return "", nil, errors.Errorf("invalid index of ':'")
	}
	prefix := encoded[:colonIndex]
	data := encoded[colonIndex+1:]
	decoded, err := decodeFromBase32(data)
	if err != nil {
		return "", nil, errors.Errorf("failed converting data to bytes: "+
			"%s", err)
	}
	if !verifyChecksum(prefix, decoded) {
		checksum := encoded[len(encoded)-checksumLength:]
		expected := encodeToBase32(calculateChecksum(prefix,
			decoded[:len(decoded)-checksumLength]))

		return "", nil, errors.Errorf("checksum failed. Expected %s, got %s", expected, checksum)
	}
	return prefix, decoded[:len(decoded)-checksumLength], nil
}

func decodeAddress(addr string, expectedPrefix string) (string, []byte, error) {
	prefix, decoded, err := decode(addr)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decode addr, err=%v", err)
	}
	if prefix != expectedPrefix {
		return "", nil, fmt.Errorf("invalid prefix")
	}
	converted := convertBits(decoded, fiveToEightBits)
	payload := converted[1:]
	return prefix, payload, nil
}

func correctAddress(address string) string {
	_, _, err := decodeAddress(address, bech32PrefixKaspaMainnet)
	if err != nil && strings.Contains(err.Error(), "checksum failed. Expected") {
		r, _ := regexp.Compile("checksum failed. Expected (.*), got (.*)")
		rr := r.FindAllSubmatch([]byte(err.Error()), -1)
		if len(rr) == 1 && len(rr[0]) == 3 {
			length := len(rr[0][1])
			main := address[:len(address)-length]
			checkSum := string(rr[0][1])
			return main + checkSum
		}
	}
	return address
}

var fiveToEightBits = conversionType{fromBits: 5, toBits: 8, pad: false}
var eightToFiveBits = conversionType{fromBits: 8, toBits: 5, pad: true}
var generatorStr = []string{"656907472481", "522768456162", "1044723512260", "748107326120", "130178868336"}

type conversionType struct {
	fromBits uint8
	toBits   uint8
	pad      bool
}

const (
	bech32PrefixKaspaTestNet = "kaspatest"
	bech32PrefixKaspaMainnet = "kaspa"
	pubKeyAddrID             = 0x00
)
const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
const checksumLength = 8
