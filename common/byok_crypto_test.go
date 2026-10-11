package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBYOKKey = "secure-production-test-key-32bytes!!"

func TestBYOKCryptoRoundTrip(t *testing.T) {
	SetBYOKKeyForTest(testBYOKKey)
	defer ResetBYOKKeyForTest()

	plaintext := "sk-or-v1-abcdef1234567890testsecret"
	aad := "user_provider:42:openrouter"

	encrypted, err := EncryptBYOKSecret(plaintext, aad)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(encrypted, "v1."))

	decrypted, err := DecryptBYOKSecret(encrypted, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestBYOKCryptoRandomNonceMakesCiphertextsDiffer(t *testing.T) {
	SetBYOKKeyForTest(testBYOKKey)
	defer ResetBYOKKeyForTest()

	plaintext := "sk-same-secret-key-123456"
	aad := "user_provider:1:custom"

	encrypted1, err := EncryptBYOKSecret(plaintext, aad)
	require.NoError(t, err)

	encrypted2, err := EncryptBYOKSecret(plaintext, aad)
	require.NoError(t, err)

	assert.NotEqual(t, encrypted1, encrypted2, "Each encryption must use a fresh random nonce")

	// Both should decrypt to the same plaintext
	decrypted1, err := DecryptBYOKSecret(encrypted1, aad)
	require.NoError(t, err)
	decrypted2, err := DecryptBYOKSecret(encrypted2, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)
	assert.Equal(t, plaintext, decrypted2)
}

func TestBYOKCryptoTamperedCiphertextFails(t *testing.T) {
	SetBYOKKeyForTest(testBYOKKey)
	defer ResetBYOKKeyForTest()

	plaintext := "sk-sensitive-api-key"
	aad := "user_provider:99:openrouter"

	encrypted, err := EncryptBYOKSecret(plaintext, aad)
	require.NoError(t, err)

	parts := strings.Split(encrypted, ".")
	require.Len(t, parts, 3)

	// Tamper with the ciphertext body
	tamperedBody := parts[2]
	if len(tamperedBody) > 4 {
		tamperedBody = "AAAA" + tamperedBody[4:]
	}
	tampered := parts[0] + "." + parts[1] + "." + tamperedBody

	_, err = DecryptBYOKSecret(tampered, aad)
	assert.ErrorIs(t, err, ErrBYOKDecryptionFailed)
}

func TestBYOKCryptoWrongKeyFails(t *testing.T) {
	SetBYOKKeyForTest("initial-encryption-key-for-test-32b")
	plaintext := "sk-secret-payload"
	aad := "user_provider:10:openrouter"

	encrypted, err := EncryptBYOKSecret(plaintext, aad)
	require.NoError(t, err)

	// Switch to a different valid key
	SetBYOKKeyForTest("different-secret-key-for-test-32b!!")
	defer ResetBYOKKeyForTest()

	_, err = DecryptBYOKSecret(encrypted, aad)
	assert.ErrorIs(t, err, ErrBYOKDecryptionFailed)
}

func TestBYOKCryptoAADMismatchFails(t *testing.T) {
	SetBYOKKeyForTest(testBYOKKey)
	defer ResetBYOKKeyForTest()

	plaintext := "sk-key-bound-to-user-1"
	correctAAD := "user_provider:1:openrouter"
	wrongUserAAD := "user_provider:2:openrouter"
	wrongProviderAAD := "user_provider:1:custom"

	encrypted, err := EncryptBYOKSecret(plaintext, correctAAD)
	require.NoError(t, err)

	// Attempt decrypt with different user ID
	_, err = DecryptBYOKSecret(encrypted, wrongUserAAD)
	assert.ErrorIs(t, err, ErrBYOKDecryptionFailed, "Ciphertext must not be decryptable with another user's AAD")

	// Attempt decrypt with different provider type
	_, err = DecryptBYOKSecret(encrypted, wrongProviderAAD)
	assert.ErrorIs(t, err, ErrBYOKDecryptionFailed, "Ciphertext must not be decryptable with another provider's AAD")
}

func TestBYOKCryptoMalformedCiphertextRejected(t *testing.T) {
	SetBYOKKeyForTest(testBYOKKey)
	defer ResetBYOKKeyForTest()

	aad := "user_provider:1:openrouter"

	malformedCases := []string{
		"",
		"invalid",
		"v2.not.enough.parts.here",
		"v1.onlyonepart",
		"v1.badbase64!nonce.badbase64!cipher",
	}

	for _, malformed := range malformedCases {
		_, err := DecryptBYOKSecret(malformed, aad)
		assert.ErrorIs(t, err, ErrBYOKInvalidPayload, "Case: %s", malformed)
	}
}

func TestBYOKCryptoKeyValidation(t *testing.T) {
	// Not configured
	SetBYOKKeyForTest("")
	_, err := DeriveBYOKKey()
	assert.ErrorIs(t, err, ErrBYOKKeyNotConfigured)

	// Too short (< 16 chars)
	SetBYOKKeyForTest("short")
	_, err = DeriveBYOKKey()
	assert.ErrorIs(t, err, ErrBYOKKeyTooShort)

	// Forbidden weak key
	SetBYOKKeyForTest("password12345678")
	_, err = DeriveBYOKKey()
	assert.ErrorIs(t, err, ErrBYOKKeyWeak)

	// Valid key
	SetBYOKKeyForTest(testBYOKKey)
	key, err := DeriveBYOKKey()
	require.NoError(t, err)
	assert.Len(t, key, 32)
	ResetBYOKKeyForTest()
}

func TestMaskAPIKey(t *testing.T) {
	assert.Equal(t, "****", MaskAPIKey(""))
	assert.Equal(t, "****", MaskAPIKey("abc"))
	assert.Equal(t, "****", MaskAPIKey("abcd"))
	assert.Equal(t, "****bcde", MaskAPIKey("abcde"))
	assert.Equal(t, "****1234", MaskAPIKey("sk-or-v1-abcdef1234"))
	assert.Equal(t, "****abcd", MaskAPIKey("  sk-or-v1-testabcd  "))
}
