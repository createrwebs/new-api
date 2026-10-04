package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenKeyEncryptionLifecycle(t *testing.T) {
	truncateTables(t)
	origSecret := common.CryptoSecret
	common.CryptoSecret = "token-encryption-test-secret-123456"
	defer func() { common.CryptoSecret = origSecret }()

	rawKey := "sk-test-relay-key-abcdef1234567890"

	token := &Token{
		UserId:         100,
		Name:           "test-encrypted-token",
		Key:            rawKey,
		Status:         common.TokenStatusEnabled,
		UnlimitedQuota: true,
	}

	// 1. Insert token
	require.NoError(t, token.Insert())
	assert.True(t, token.Id > 0)
	// In-memory struct has decrypted plaintext key
	assert.Equal(t, rawKey, token.Key)

	// 2. Verify Database storage: Key must be ENCRYPTED, not plaintext!
	var rawRow struct {
		Key     string `gorm:"column:key"`
		KeyHash string `gorm:"column:key_hash"`
	}
	require.NoError(t, DB.Table("tokens").Where("id = ?", token.Id).Scan(&rawRow).Error)
	assert.True(t, strings.HasPrefix(rawRow.Key, "enc:v1:"), "raw DB key must be encrypted with enc:v1: prefix")
	assert.NotEqual(t, rawKey, rawRow.Key, "raw DB key must not be plaintext")
	assert.Equal(t, TokenHash(rawKey), rawRow.KeyHash, "raw DB key_hash must be SHA-256 of plaintext key")

	// 3. Lookup by key (blind index lookup via key_hash)
	found, err := GetTokenByKey(rawKey, true)
	require.NoError(t, err)
	assert.Equal(t, token.Id, found.Id)
	assert.Equal(t, rawKey, found.Key) // In-memory struct is decrypted

	// 4. Lookup by wrong key fails
	_, err = GetTokenByKey("sk-wrong-key-999", true)
	assert.Error(t, err)

	// 5. Masking
	assert.Equal(t, MaskTokenKey(rawKey), found.GetMaskedKey())
	assert.Equal(t, rawKey, found.GetFullKey())
}

func TestLegacyTokenPlaintextCompatibility(t *testing.T) {
	truncateTables(t)
	origSecret := common.CryptoSecret
	common.CryptoSecret = "token-encryption-test-secret-123456"
	defer func() { common.CryptoSecret = origSecret }()

	legacyRawKey := "sk-legacy-unencrypted-key-xyz"

	// Directly insert a legacy row with plaintext key and empty key_hash
	require.NoError(t, DB.Exec("INSERT INTO tokens (id, user_id, `key`, key_hash, status, name, created_time, accessed_time, expired_time, remain_quota, unlimited_quota) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		200, 100, legacyRawKey, "", common.TokenStatusEnabled, "legacy-token", 1000, 1000, -1, 0, true,
	).Error)

	// Lookup legacy token
	found, err := GetTokenByKey(legacyRawKey, true)
	require.NoError(t, err)
	assert.Equal(t, 200, found.Id)
	assert.Equal(t, legacyRawKey, found.Key)

	// GetTokenByIds also returns plaintext
	foundById, err := GetTokenByIds(200, 100)
	require.NoError(t, err)
	assert.Equal(t, legacyRawKey, foundById.Key)
}

func TestTokenEncryptionWrongSecretAndRevokedStatus(t *testing.T) {
	truncateTables(t)
	origSecret := common.CryptoSecret
	common.CryptoSecret = "secret-key-phase-6b-test-primary"
	defer func() { common.CryptoSecret = origSecret }()

	rawKey := "sk-test-crypto-secret-key-12345"

	token := &Token{
		UserId:         101,
		Name:           "status-test-token",
		Key:            rawKey,
		Status:         common.TokenStatusEnabled,
		UnlimitedQuota: true,
	}
	require.NoError(t, token.Insert())

	// 1. Verify normal lookup succeeds
	found, err := GetTokenByKey(rawKey, true)
	require.NoError(t, err)
	assert.Equal(t, rawKey, found.Key)

	// Fetch actual ciphertext from DB
	var rawRow struct {
		Key string `gorm:"column:key"`
	}
	require.NoError(t, DB.Table("tokens").Where("id = ?", token.Id).Scan(&rawRow).Error)
	assert.True(t, strings.HasPrefix(rawRow.Key, "enc:v1:"))

	// 2. Disable/Revoke token: status = TokenStatusDisabled
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Update("status", common.TokenStatusDisabled).Error)
	disabledToken, err := GetTokenByKey(rawKey, true)
	require.NoError(t, err)
	assert.Equal(t, common.TokenStatusDisabled, disabledToken.Status, "token status must be disabled")

	// 3. Delete token: Soft-delete
	require.NoError(t, DB.Where("id = ?", token.Id).Delete(&Token{}).Error)
	_, err = GetTokenByKey(rawKey, true)
	assert.Error(t, err, "deleted token cannot be found via GetTokenByKey")

	// 4. Rotated/Wrong CryptoSecret fails safely
	common.CryptoSecret = "different-wrong-secret-key-rotated"
	_, err = DecryptTokenKey(rawRow.Key)
	assert.Error(t, err, "decryption with wrong secret must fail")
}
