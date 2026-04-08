package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var ErrInvalidEncryptionKey = errors.New("encryption key must be 32 bytes for AES-256")

// Encrypt encrypts a plaintext string using AES-GCM and returns a base64 encoded ciphertext
func Encrypt(plaintext string, keyString string) (string, error) {
	key := []byte(keyString)

	// We only support AES-256 for maximum security, ensure exactly 32 bytes
	if len(key) != 32 {
		return "", ErrInvalidEncryptionKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded AES-GCM ciphertext using the given key
func Decrypt(ciphertextBase64 string, keyString string) (string, error) {
	key := []byte(keyString)

	if len(key) != 32 {
		return "", ErrInvalidEncryptionKey
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
