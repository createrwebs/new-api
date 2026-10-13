package service

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMJImageAccessLifecycle(t *testing.T) {
	origSecret := common.CryptoSecret
	common.CryptoSecret = "test-crypto-secret-key-12345678"
	defer func() { common.CryptoSecret = origSecret }()

	taskID := "1234567890abcdef"
	ownerUserID := 100

	token, err := IssueMJImageAccess(taskID, ownerUserID, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Valid verification with matching owner
	assert.True(t, VerifyMJImageAccess(token, taskID, ownerUserID, common.RoleCommonUser))

	// Admin caller bypass
	assert.True(t, VerifyMJImageAccess(token, taskID, 999, common.RoleAdminUser))

	// Invalid when caller is different user
	assert.False(t, VerifyMJImageAccess(token, taskID, 200, common.RoleCommonUser))

	// Invalid when caller is unauthenticated
	assert.False(t, VerifyMJImageAccess(token, taskID, 0, 0))

	// Invalid for another task ID
	assert.False(t, VerifyMJImageAccess(token, "another-task-id", ownerUserID, common.RoleCommonUser))

	// Tampered signature
	parts := strings.Split(token, ".")
	require.Len(t, parts, 4)
	tamperedSig := parts[0] + "." + parts[1] + "." + parts[2] + ".tamperedSignature123"
	assert.False(t, VerifyMJImageAccess(tamperedSig, taskID, ownerUserID, common.RoleCommonUser))

	// Tampered expiration
	tamperedExpiry := parts[0] + ".9999999999." + parts[2] + "." + parts[3]
	assert.False(t, VerifyMJImageAccess(tamperedExpiry, taskID, ownerUserID, common.RoleCommonUser))

	// Tampered owner_user_id (e.g. changing 100 to 200)
	tamperedOwner := parts[0] + "." + parts[1] + ".200." + parts[3]
	assert.False(t, VerifyMJImageAccess(tamperedOwner, taskID, 200, common.RoleCommonUser))

	// Wrong purpose
	wrongPurposeToken, err := IssueMJImageAccessWithPurpose("avatar", taskID, ownerUserID, time.Hour)
	require.NoError(t, err)
	assert.False(t, VerifyMJImageAccess(wrongPurposeToken, taskID, ownerUserID, common.RoleCommonUser))
}

func TestMJImageAccessExpired(t *testing.T) {
	origSecret := common.CryptoSecret
	common.CryptoSecret = "test-crypto-secret-key-12345678"
	defer func() { common.CryptoSecret = origSecret }()

	taskID := "task-expired-test"
	ownerUserID := 100

	// Create token with 10ms ttl and wait
	token, err := IssueMJImageAccess(taskID, ownerUserID, 10*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond) // unix seconds precision
	assert.False(t, VerifyMJImageAccess(token, taskID, ownerUserID, common.RoleCommonUser))
}

func TestBuildMJImageURL(t *testing.T) {
	origSecret := common.CryptoSecret
	common.CryptoSecret = "test-crypto-secret-key-12345678"
	defer func() { common.CryptoSecret = origSecret }()

	origAddress := system_setting.ServerAddress
	system_setting.ServerAddress = "https://api.toraai.com"
	defer func() { system_setting.ServerAddress = origAddress }()

	url := BuildMJImageURL("mj_task_123", 100, time.Hour)
	assert.True(t, strings.HasPrefix(url, "https://api.toraai.com/mj/image/mj_task_123?access="))

	// Extract access token from URL and verify it
	parts := strings.Split(url, "?access=")
	require.Len(t, parts, 2)
	assert.True(t, VerifyMJImageAccess(parts[1], "mj_task_123", 100, common.RoleCommonUser))
	assert.False(t, VerifyMJImageAccess(parts[1], "mj_task_123", 200, common.RoleCommonUser))
}
