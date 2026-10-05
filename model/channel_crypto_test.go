package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelKeyEncryptionLifecycle(t *testing.T) {
	setupChannelStatusTest(t)

	const keyCanary = "CHANNEL_KEY_CANARY_sk-secret12345"

	channel := Channel{
		Name:   "crypto-test-channel",
		Type:   1, // OpenAI
		Key:    keyCanary,
		Status: common.ChannelStatusEnabled,
		Group:  "default",
	}

	require.NoError(t, channel.Insert())
	require.NotZero(t, channel.Id)

	// In memory, channel.Key remains plaintext
	assert.Equal(t, keyCanary, channel.Key)

	// Query raw database row directly without GORM hooks
	var rawKeyInDB string
	require.NoError(t, DB.Table("channels").Select("`key`").Where("id = ?", channel.Id).Scan(&rawKeyInDB).Error)
	if rawKeyInDB == "" {
		// Postgres uses double quotes
		require.NoError(t, DB.Table("channels").Select(`"key"`).Where("id = ?", channel.Id).Scan(&rawKeyInDB).Error)
	}

	// Raw key in DB must be encrypted with AES-256-GCM and prefixed with enc:v1:
	assert.True(t, len(rawKeyInDB) > 0)
	assert.Contains(t, rawKeyInDB, "enc:v1:")
	assert.NotContains(t, rawKeyInDB, keyCanary, "plaintext key must not exist at rest in database")

	// Loading channel via GetChannelById must transparently decrypt
	loaded, err := GetChannelById(channel.Id, true)
	require.NoError(t, err)
	assert.Equal(t, keyCanary, loaded.Key)

	// GetKeys() must return the plaintext key
	keys := loaded.GetKeys()
	require.Len(t, keys, 1)
	assert.Equal(t, keyCanary, keys[0])

	// GetNextEnabledKey() must return the plaintext key
	nextKey, _, nextErr := loaded.GetNextEnabledKey()
	require.Nil(t, nextErr)
	assert.Equal(t, keyCanary, nextKey)
}

func TestLegacyPlaintextChannelKeyCompatibility(t *testing.T) {
	setupChannelStatusTest(t)

	const legacyKey = "sk-legacy-unencrypted-key-999"

	// Insert legacy plaintext row directly by raw SQL (bypassing BeforeCreate hook)
	var insertSQL string
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		insertSQL = `INSERT INTO channels (name, type, "key", status, "group", created_time) VALUES (?, ?, ?, ?, ?, ?)`
	} else {
		insertSQL = `INSERT INTO channels (name, type, ` + "`key`" + `, status, ` + "`group`" + `, created_time) VALUES (?, ?, ?, ?, ?, ?)`
	}
	require.NoError(t, DB.Exec(insertSQL, "legacy-channel", 1, legacyKey, common.ChannelStatusEnabled, "default", 1234567890).Error)

	var channelID int
	require.NoError(t, DB.Table("channels").Select("id").Where("name = ?", "legacy-channel").Scan(&channelID).Error)
	require.NotZero(t, channelID)

	// Loading legacy unencrypted channel must work transparently
	loaded, err := GetChannelById(channelID, true)
	require.NoError(t, err)
	assert.Equal(t, legacyKey, loaded.Key)
	assert.Equal(t, []string{legacyKey}, loaded.GetKeys())

	// Updating the channel should automatically migrate/encrypt the key at rest
	loaded.Name = "migrated-legacy-channel"
	require.NoError(t, loaded.Update())

	// Check raw DB row again: should now be encrypted
	var updatedRawKey string
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		require.NoError(t, DB.Table("channels").Select(`"key"`).Where("id = ?", channelID).Scan(&updatedRawKey).Error)
	} else {
		require.NoError(t, DB.Table("channels").Select("`key`").Where("id = ?", channelID).Scan(&updatedRawKey).Error)
	}
	assert.Contains(t, updatedRawKey, "enc:v1:")
	assert.NotContains(t, updatedRawKey, legacyKey)
}

func TestMultiKeyChannelEncryption(t *testing.T) {
	setupChannelStatusTest(t)

	const multiKeyStr = "sk-multi-key-1\nsk-multi-key-2\nsk-multi-key-3"

	channel := Channel{
		Name:   "multi-key-channel",
		Type:   1,
		Key:    multiKeyStr,
		Status: common.ChannelStatusEnabled,
		Group:  "default",
	}

	require.NoError(t, channel.Insert())

	// Check raw DB row: must be encrypted
	var rawKey string
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		require.NoError(t, DB.Table("channels").Select(`"key"`).Where("id = ?", channel.Id).Scan(&rawKey).Error)
	} else {
		require.NoError(t, DB.Table("channels").Select("`key`").Where("id = ?", channel.Id).Scan(&rawKey).Error)
	}
	assert.Contains(t, rawKey, "enc:v1:")
	assert.NotContains(t, rawKey, "sk-multi-key-1")

	// Load and verify GetKeys
	loaded, err := GetChannelById(channel.Id, true)
	require.NoError(t, err)
	keys := loaded.GetKeys()
	require.Len(t, keys, 3)
	assert.Equal(t, "sk-multi-key-1", keys[0])
	assert.Equal(t, "sk-multi-key-2", keys[1])
	assert.Equal(t, "sk-multi-key-3", keys[2])
}
