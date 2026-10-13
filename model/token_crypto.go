package model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const tokenEncryptedPrefix = "enc:v1:"

var (
	ErrTokenKeyDecryptionFailed = errors.New("failed to decrypt token key")
	ErrTokenKeyInvalidPayload   = errors.New("invalid encrypted token key payload")
)

// TokenHash computes a deterministic SHA-256 hex digest of a token plaintext key for fast indexed lookup.
func TokenHash(plaintextKey string) string {
	plaintextKey = strings.TrimSpace(plaintextKey)
	if plaintextKey == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(plaintextKey))
	return hex.EncodeToString(hash[:])
}

func deriveTokenCryptoKey() ([]byte, error) {
	secret := strings.TrimSpace(common.CryptoSecret)
	if secret == "" {
		return nil, errors.New("common.CryptoSecret is not configured")
	}
	hash := sha256.Sum256([]byte(secret))
	return hash[:], nil
}

// EncryptTokenKey encrypts a plaintext token key using AES-256-GCM.
func EncryptTokenKey(plaintextKey string) (string, error) {
	plaintextKey = strings.TrimSpace(plaintextKey)
	if plaintextKey == "" {
		return "", nil
	}
	key, err := deriveTokenCryptoKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher init error: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm init error: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation error: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintextKey), nil)
	return fmt.Sprintf("%s%s:%s",
		tokenEncryptedPrefix,
		base64.RawURLEncoding.EncodeToString(nonce),
		base64.RawURLEncoding.EncodeToString(ciphertext),
	), nil
}

// DecryptTokenKey decrypts an encrypted token key. If the key does not start with the encrypted prefix,
// it is returned as-is to preserve compatibility with legacy unencrypted keys.
func DecryptTokenKey(encryptedKey string) (string, error) {
	encryptedKey = strings.TrimSpace(encryptedKey)
	if encryptedKey == "" {
		return "", nil
	}
	if !strings.HasPrefix(encryptedKey, tokenEncryptedPrefix) {
		return encryptedKey, nil
	}

	payload := strings.TrimPrefix(encryptedKey, tokenEncryptedPrefix)
	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		return "", ErrTokenKeyInvalidPayload
	}

	nonce, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrTokenKeyInvalidPayload
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrTokenKeyInvalidPayload
	}

	key, err := deriveTokenCryptoKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenKeyDecryptionFailed, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenKeyDecryptionFailed, err)
	}
	if len(nonce) != gcm.NonceSize() {
		return "", ErrTokenKeyInvalidPayload
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenKeyDecryptionFailed, err)
	}
	return string(plaintext), nil
}
