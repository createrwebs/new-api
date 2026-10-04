package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testModelBYOKKey = "secure-model-test-key-32bytes-ok!"

func setupUserProviderTestDB(t *testing.T) {
	t.Helper()
	common.SetBYOKKeyForTest(testModelBYOKKey)
	previousDB := DB
	previousType := common.MainDatabaseType()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&User{},
		&UserProvider{},
		&TwoFABackupCode{},
		&TwoFA{},
		&UserSession{},
		&AuthFlow{},
		&PasskeyCredential{},
		&Token{},
		&UserAccessToken{},
		&UserOAuthBinding{},
		&ExternalIdentityClaim{},
	))

	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.ResetBYOKKeyForTest()
	})
}

func createTestUser(t *testing.T, prefix string) *User {
	t.Helper()
	user := &User{
		Username: prefix + "-" + common.GetRandomString(8),
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "aff-" + common.GetRandomString(10),
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestUserProviderCRUDAndEncryption(t *testing.T) {
	setupUserProviderTestDB(t)

	user := createTestUser(t, "byok-user")

	rawSecretKey := "sk-or-v1-my-secret-openrouter-key"

	provider := &UserProvider{
		UserId:   user.Id,
		Provider: ProviderOpenRouter,
		Name:     "My OpenRouter",
		BaseURL:  DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey(rawSecretKey))
	require.NoError(t, CreateUserProvider(provider))
	assert.Positive(t, provider.Id)

	// Verify plaintext key is NOT stored in DB
	var dbRecord UserProvider
	require.NoError(t, DB.First(&dbRecord, provider.Id).Error)
	assert.NotEqual(t, rawSecretKey, dbRecord.APIKeyEncrypted)
	assert.True(t, strings.HasPrefix(dbRecord.APIKeyEncrypted, "v1."))
	assert.Equal(t, "****-key", dbRecord.APIKeyMasked)

	// Decrypt through model
	decrypted, err := dbRecord.DecryptAPIKey()
	require.NoError(t, err)
	assert.Equal(t, rawSecretKey, decrypted)

	// List
	list, err := GetUserProviders(user.Id)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, provider.Name, list[0].Name)

	// Update without changing key
	dbRecord.Name = "Updated OpenRouter Name"
	dbRecord.Enabled = false
	require.NoError(t, UpdateUserProvider(&dbRecord))

	updated, err := GetUserProviderByID(user.Id, dbRecord.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated OpenRouter Name", updated.Name)
	assert.False(t, updated.Enabled)
	decryptedAfterUpdate, err := updated.DecryptAPIKey()
	require.NoError(t, err)
	assert.Equal(t, rawSecretKey, decryptedAfterUpdate)

	// Update with new key
	newSecretKey := "sk-or-v1-new-secret-different"
	require.NoError(t, updated.SetAPIKey(newSecretKey))
	require.NoError(t, UpdateUserProvider(updated))

	updatedWithNewKey, err := GetUserProviderByID(user.Id, dbRecord.Id)
	require.NoError(t, err)
	decryptedNew, err := updatedWithNewKey.DecryptAPIKey()
	require.NoError(t, err)
	assert.Equal(t, newSecretKey, decryptedNew)
	assert.Equal(t, "****rent", updatedWithNewKey.APIKeyMasked)

	// Delete
	require.NoError(t, DeleteUserProvider(user.Id, dbRecord.Id))
	_, err = GetUserProviderByID(user.Id, dbRecord.Id)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestUserProviderUserIsolation(t *testing.T) {
	setupUserProviderTestDB(t)

	userA := createTestUser(t, "user-a")
	userB := createTestUser(t, "user-b")

	providerA := &UserProvider{
		UserId:   userA.Id,
		Provider: ProviderOpenRouter,
		Name:     "User A Provider",
		BaseURL:  DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, providerA.SetAPIKey("sk-secret-a-12345"))
	require.NoError(t, CreateUserProvider(providerA))

	// User B cannot GET User A's provider
	_, err := GetUserProviderByID(userB.Id, providerA.Id)
	assert.ErrorIs(t, err, ErrProviderNotFound)

	// User B cannot UPDATE User A's provider
	tampered := &UserProvider{
		Id:       providerA.Id,
		UserId:   userB.Id,
		Provider: ProviderOpenRouter,
		Name:     "Hacked Name",
	}
	err = UpdateUserProvider(tampered)
	assert.ErrorIs(t, err, ErrProviderNotFound)

	// User B cannot DELETE User A's provider
	err = DeleteUserProvider(userB.Id, providerA.Id)
	assert.ErrorIs(t, err, ErrProviderNotFound)

	// User A's provider remains intact
	remaining, err := GetUserProviderByID(userA.Id, providerA.Id)
	require.NoError(t, err)
	assert.Equal(t, "User A Provider", remaining.Name)
}

func TestUserProviderDuplicateConstraint(t *testing.T) {
	setupUserProviderTestDB(t)

	user := createTestUser(t, "user-dup")

	p1 := &UserProvider{
		UserId:   user.Id,
		Provider: ProviderOpenRouter,
		Name:     "First OpenRouter",
		BaseURL:  DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, p1.SetAPIKey("sk-key-1-12345"))
	require.NoError(t, CreateUserProvider(p1))

	// Duplicate openrouter for the same user should fail
	p2 := &UserProvider{
		UserId:   user.Id,
		Provider: ProviderOpenRouter,
		Name:     "Second OpenRouter",
		BaseURL:  DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, p2.SetAPIKey("sk-key-2-12345"))
	err := CreateUserProvider(p2)
	assert.ErrorIs(t, err, ErrProviderDuplicate)

	// But "custom" provider for the same user is allowed
	customP := &UserProvider{
		UserId:   user.Id,
		Provider: ProviderCustom,
		Name:     "Custom Provider",
		BaseURL:  "https://custom-ai.example.com/v1",
		Enabled:  true,
	}
	require.NoError(t, customP.SetAPIKey("sk-key-custom-12345"))
	require.NoError(t, CreateUserProvider(customP))
}

func TestValidateUserProviderInput(t *testing.T) {
	// Invalid provider type
	_, err := ValidateUserProviderInput("anthropic", "Name", "https://api.example.com")
	assert.ErrorIs(t, err, ErrProviderInvalidType)

	// Empty name
	_, err = ValidateUserProviderInput("openrouter", "", "")
	assert.ErrorIs(t, err, ErrProviderNameEmpty)

	// Custom without base_url
	_, err = ValidateUserProviderInput("custom", "My Custom", "")
	assert.ErrorIs(t, err, ErrProviderBaseURLEmpty)

	// Custom with invalid base_url
	_, err = ValidateUserProviderInput("custom", "My Custom", "not-a-url")
	assert.ErrorIs(t, err, ErrProviderBaseURLInvalid)

	// Custom with valid base_url (normalizes trailing slash)
	cleanURL, err := ValidateUserProviderInput("custom", "My Custom", "https://openrouter.ai/api/v1///")
	require.NoError(t, err)
	assert.Equal(t, "https://openrouter.ai/api/v1", cleanURL)

	// OpenRouter with empty base_url gets default
	cleanOpenRouterURL, err := ValidateUserProviderInput("openrouter", "OpenRouter", "")
	require.NoError(t, err)
	assert.Equal(t, DefaultOpenRouterBaseURL, cleanOpenRouterURL)
}

func TestUserDeletionDeletesUserProviders(t *testing.T) {
	setupUserProviderTestDB(t)

	// 1. create user
	user := createTestUser(t, "user-del")

	// 2. create provider
	provider := &UserProvider{
		UserId:   user.Id,
		Provider: ProviderOpenRouter,
		Name:     "OpenRouter To Delete",
		BaseURL:  DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-secret-delete-test"))
	require.NoError(t, CreateUserProvider(provider))

	// Verify provider was stored
	stored, err := GetUserProviderByID(user.Id, provider.Id)
	require.NoError(t, err)
	assert.Equal(t, provider.Name, stored.Name)

	// 3. delete user (via user.Delete())
	_, err = user.Delete()
	require.NoError(t, err)

	// 4. verify provider record ถูกลบ
	_, err = GetUserProviderByID(user.Id, provider.Id)
	assert.ErrorIs(t, err, ErrProviderNotFound)

	var count int64
	require.NoError(t, DB.Model(&UserProvider{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Zero(t, count, "All provider records for deleted user must be purged")
}

func TestUserHardDeletionDeletesUserProviders(t *testing.T) {
	setupUserProviderTestDB(t)

	// 1. create user
	user := createTestUser(t, "user-hard-del")

	// 2. create provider
	provider := &UserProvider{
		UserId:   user.Id,
		Provider: ProviderOpenRouter,
		Name:     "OpenRouter Hard Delete",
		BaseURL:  DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-secret-hard-delete-test"))
	require.NoError(t, CreateUserProvider(provider))

	// 3. hard delete user
	_, err := user.HardDelete()
	require.NoError(t, err)

	// 4. verify provider record ถูกลบ
	var count int64
	require.NoError(t, DB.Model(&UserProvider{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Zero(t, count, "All provider records for hard-deleted user must be purged")
}

