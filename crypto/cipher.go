package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	cipherSaltSize = 32
	cipherKeySize  = 32
	cipherScryptN  = 32768
	cipherScryptR  = 8
	cipherScryptP  = 1
)

func EncryptText(clearText, password []byte) ([]byte, error) {
	if len(clearText) == 0 || len(password) == 0 {
		return nil, errors.New("invalid input")
	}

	salt := make([]byte, cipherSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := scrypt.Key(password, salt, cipherScryptN, cipherScryptR, cipherScryptP, cipherKeySize)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, clearText, nil)

	result := make([]byte, len(salt)+len(nonce)+len(ciphertext))
	copy(result, salt)
	copy(result[len(salt):], nonce)
	copy(result[len(salt)+len(nonce):], ciphertext)

	return result, nil
}

func DecryptText(cipherText, password []byte) ([]byte, error) {
	if cipherText == nil || len(password) == 0 {
		return nil, errors.New("invalid input")
	}
	if len(cipherText) < cipherSaltSize+12 {
		return nil, errors.New("invalid encrypted data")
	}

	salt := cipherText[:cipherSaltSize]
	nonce := cipherText[cipherSaltSize : cipherSaltSize+12]
	ciphertext := cipherText[cipherSaltSize+12:]

	key, err := scrypt.Key(password, salt, cipherScryptN, cipherScryptR, cipherScryptP, cipherKeySize)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed")
	}

	return plaintext, nil
}
