package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveStoreAccountToken(t *testing.T) {
	// 1. Must produce a valid RFC 4122 UUID string
	tokenProd := DeriveStoreAccountToken(42, "production")
	parsed, err := uuid.Parse(tokenProd)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(5), parsed.Version())

	// 2. Deterministic
	assert.Equal(t, tokenProd, DeriveStoreAccountToken(42, "production"))
	assert.Equal(t, tokenProd, DeriveStoreAccountToken(42, "Production"))
	assert.Equal(t, tokenProd, DeriveStoreAccountToken(42, ""))

	// 3. User isolation
	tokenOther := DeriveStoreAccountToken(43, "production")
	assert.NotEqual(t, tokenProd, tokenOther)

	// 4. Environment isolation
	tokenSandbox := DeriveStoreAccountToken(42, "sandbox")
	assert.NotEqual(t, tokenProd, tokenSandbox)
	assert.Equal(t, tokenSandbox, DeriveStoreAccountToken(42, "Sandbox"))
	assert.Equal(t, tokenSandbox, DeriveStoreAccountToken(42, "Xcode"))

	// 5. Validation helper
	assert.True(t, ValidateStoreAccountToken(tokenProd, 42, "production"))
	assert.True(t, ValidateStoreAccountToken(tokenProd, 42, "Production"))
	assert.False(t, ValidateStoreAccountToken(tokenProd, 43, "production"))
	assert.False(t, ValidateStoreAccountToken(tokenProd, 42, "sandbox"))
	assert.False(t, ValidateStoreAccountToken("", 42, "production"))
	assert.False(t, ValidateStoreAccountToken(tokenProd, 0, "production"))

	t.Logf("User 42 (production): %s", tokenProd)
	t.Logf("User 42 (sandbox):    %s", tokenSandbox)
	t.Logf("User 43 (production): %s", tokenOther)
}
