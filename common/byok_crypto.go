package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	ErrBYOKKeyNotConfigured = errors.New("BYOK_ENCRYPTION_KEY is not configured")
	ErrBYOKKeyTooShort      = errors.New("BYOK_ENCRYPTION_KEY must be at least 16 characters")
	ErrBYOKKeyWeak          = errors.New("BYOK_ENCRYPTION_KEY is insecure or uses a forbidden default value")
	ErrBYOKEncryptionFailed = errors.New("failed to encrypt BYOK secret")
	ErrBYOKDecryptionFailed = errors.New("failed to decrypt BYOK secret")
	ErrBYOKInvalidPayload   = errors.New("invalid encrypted BYOK payload")
)

var (
	byokKeyMu      sync.RWMutex
	byokTestKey    string
	byokUseTestKey bool
)

// SetBYOKKeyForTest sets an in-memory encryption key for unit testing.
func SetBYOKKeyForTest(key string) {
	byokKeyMu.Lock()
	defer byokKeyMu.Unlock()
	byokTestKey = key
	byokUseTestKey = true
}

// ResetBYOKKeyForTest restores normal environment variable lookup for BYOK encryption key.
func ResetBYOKKeyForTest() {
	byokKeyMu.Lock()
	defer byokKeyMu.Unlock()
	byokTestKey = ""
	byokUseTestKey = false
}

func getBYOKRawKey() (string, error) {
	byokKeyMu.RLock()
	if byokUseTestKey {
		key := byokTestKey
		byokKeyMu.RUnlock()
		if strings.TrimSpace(key) == "" {
			return "", ErrBYOKKeyNotConfigured
		}
		return key, nil
	}
	byokKeyMu.RUnlock()

	raw := os.Getenv("BYOK_ENCRYPTION_KEY")
	if strings.TrimSpace(raw) == "" {
		return "", ErrBYOKKeyNotConfigured
	}
	return raw, nil
}

// DeriveBYOKKey validates and derives a 32-byte AES-256 key from BYOK_ENCRYPTION_KEY.
// Fallback to common.CryptoSecret or session secret is strictly prohibited.
func DeriveBYOKKey() ([]byte, error) {
	raw, err := getBYOKRawKey()
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if len(raw) < 16 {
		return nil, ErrBYOKKeyTooShort
	}
	switch strings.ToLower(raw) {
	case "1234567890123456", "password12345678", "byok_encryption_key", "default_secret_key":
		return nil, ErrBYOKKeyWeak
	}
	hash := sha256.Sum256([]byte(raw))
	return hash[:], nil
}

// EncryptBYOKSecret encrypts plaintext using AES-256-GCM with a random 12-byte nonce
// and binds it to additional authenticated data (aad).
// Payload format: "v1.<base64(nonce)>.<base64(ciphertext+tag)>"
func EncryptBYOKSecret(plaintext string, aad string) (string, error) {
	if plaintext == "" {
		return "", errors.New("cannot encrypt empty secret")
	}
	key, err := DeriveBYOKKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBYOKEncryptionFailed, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBYOKEncryptionFailed, err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %v", ErrBYOKEncryptionFailed, err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), []byte(aad))
	return fmt.Sprintf("v1.%s.%s",
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(ciphertext),
	), nil
}

// DecryptBYOKSecret decrypts payload using AES-256-GCM and verifies aad.
func DecryptBYOKSecret(payload string, aad string) (string, error) {
	parts := strings.Split(payload, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return "", ErrBYOKInvalidPayload
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrBYOKInvalidPayload
	}
	ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", ErrBYOKInvalidPayload
	}

	key, err := DeriveBYOKKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBYOKDecryptionFailed, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBYOKDecryptionFailed, err)
	}
	if len(nonce) != gcm.NonceSize() {
		return "", ErrBYOKInvalidPayload
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(aad))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBYOKDecryptionFailed, err)
	}
	return string(plaintext), nil
}

// MaskAPIKey produces a masked key representation (e.g. "****abcd" or "****").
func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
