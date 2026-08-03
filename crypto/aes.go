package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

func AesEncryptCTR(data, key, iv []byte) ([]byte, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("invalid iv length %d", len(iv))
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(data))
	stream.XORKeyStream(outText, data)
	return outText, err
}

func AesDecryptCTR(data, key, iv []byte) ([]byte, error) {
	return AesEncryptCTR(data, key, iv)
}
