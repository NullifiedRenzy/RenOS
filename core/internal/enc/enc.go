package enc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrEmptyKey    = errors.New("encryption key is empty")
	ErrInvalidData = errors.New("invalid encrypted data")
)

// DeriveKey derives a 32-byte AES key from a passphrase using SHA-256.
func DeriveKey(passphrase string) []byte {
	h := sha256.Sum256([]byte(passphrase))
	return h[:]
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Returns base64-encoded ciphertext: nonce + ciphertext.
func Encrypt(key []byte, plaintext []byte) (string, error) {
	if len(key) == 0 {
		return "", ErrEmptyKey
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
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded AES-256-GCM ciphertext.
func Decrypt(key []byte, encoded string) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrEmptyKey
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrInvalidData
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrInvalidData
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
