package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

func setupBYOKTestEnv(t *testing.T) (*model.User, func()) {
	t.Helper()
	prevKey := os.Getenv("BYOK_ENCRYPTION_KEY")
	testKey := "byok-test-secret-key-32-bytes-len"
	os.Setenv("BYOK_ENCRYPTION_KEY", testKey)

	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousType := common.MainDatabaseType()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.UserSubscription{},
		&model.FreeTierDailyUsage{},
		&model.UserProvider{},
	))

	model.DB = db
	model.LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	user := &model.User{
		Username:    "byok-user-" + common.GetRandomString(8),
		Password:    "unused-password",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
		AffCode:     "byok-aff-" + common.GetRandomString(8),
	}
	require.NoError(t, db.Create(user).Error)

	cleanup := func() {
		os.Setenv("BYOK_ENCRYPTION_KEY", prevKey)
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
	}

	return user, cleanup
}

func TestDetectBYOKRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Case 1: Header X-Provider
	c1, _ := gin.CreateTestContext(httptest.NewRecorder())
	c1.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c1.Request.Header.Set("X-Provider", "openrouter")
	info1 := detectBYOKRequest(c1)
	assert.True(t, info1.isBYOK)
	assert.Equal(t, model.ProviderOpenRouter, info1.provider)

	// Case 2: Header X-BYOK
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c2.Request.Header.Set("X-BYOK", "custom")
	info2 := detectBYOKRequest(c2)
	assert.True(t, info2.isBYOK)
	assert.Equal(t, model.ProviderCustom, info2.provider)

	// Case 3: Query parameter
	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions?provider=openrouter", nil)
	info3 := detectBYOKRequest(c3)
	assert.True(t, info3.isBYOK)
	assert.Equal(t, model.ProviderOpenRouter, info3.provider)

	// Case 4: JSON model prefix openrouter/
	c4, _ := gin.CreateTestContext(httptest.NewRecorder())
	body4 := `{"model": "openrouter/anthropic/claude-3.5-sonnet", "messages": []}`
	c4.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body4))
	c4.Request.Header.Set("Content-Type", "application/json")
	info4 := detectBYOKRequest(c4)
	assert.True(t, info4.isBYOK)
	assert.Equal(t, model.ProviderOpenRouter, info4.provider)
	assert.True(t, info4.hasPrefix)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", info4.cleanModel)

	// Case 5: JSON model prefix custom/
	c5, _ := gin.CreateTestContext(httptest.NewRecorder())
	body5 := `{"model": "custom/meta-llama-3", "messages": []}`
	c5.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body5))
	c5.Request.Header.Set("Content-Type", "application/json")
	info5 := detectBYOKRequest(c5)
	assert.True(t, info5.isBYOK)
	assert.Equal(t, model.ProviderCustom, info5.provider)
	assert.True(t, info5.hasPrefix)
	assert.Equal(t, "meta-llama-3", info5.cleanModel)

	// Case 6: Non-BYOK standard request
	c6, _ := gin.CreateTestContext(httptest.NewRecorder())
	body6 := `{"model": "gpt-4o", "messages": []}`
	c6.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body6))
	c6.Request.Header.Set("Content-Type", "application/json")
	info6 := detectBYOKRequest(c6)
	assert.False(t, info6.isBYOK)
}

func TestBYOKRouterInvalidProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("X-Provider", "unsupported-provider")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "byok_invalid_provider")
}

func TestBYOKRouterProviderNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("X-Provider", "openrouter")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "byok_provider_not_found")
}

func TestBYOKRouterProviderDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderOpenRouter,
		Name:     "Disabled OpenRouter",
		Enabled:  false,
	}
	require.NoError(t, provider.SetAPIKey("sk-or-test-key-1234"))
	require.NoError(t, model.CreateUserProvider(provider))

	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("X-Provider", "openrouter")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "byok_provider_not_found")
}

func TestBYOKRouterSuccessOpenRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	plaintextKey := "sk-or-v1-my-secret-openrouter-key"
	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderOpenRouter,
		Name:     "My OpenRouter",
		BaseURL:  "https://openrouter.ai/api/v1",
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey(plaintextKey))
	require.NoError(t, model.CreateUserProvider(provider))

	var capturedChannelType int
	var capturedChannelKey string
	var capturedBaseURL string
	var capturedModel string
	var capturedIsBYOK bool
	var capturedRewrittenBody []byte

	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}

		capturedIsBYOK = c.GetBool("is_byok")
		capturedChannelType = common.GetContextKeyInt(c, constant.ContextKeyChannelType)
		capturedChannelKey = common.GetContextKeyString(c, constant.ContextKeyChannelKey)
		capturedBaseURL = common.GetContextKeyString(c, constant.ContextKeyChannelBaseUrl)
		capturedModel = common.GetContextKeyString(c, constant.ContextKeyOriginalModel)

		storage, err := common.GetBodyStorage(c)
		if err == nil && storage != nil {
			capturedRewrittenBody, _ = storage.Bytes()
		}

		c.Status(http.StatusOK)
	})

	reqBody := `{"model": "openrouter/anthropic/claude-3.5-sonnet", "messages": [{"role": "user", "content": "hi"}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, capturedIsBYOK)
	assert.Equal(t, constant.ChannelTypeOpenRouter, capturedChannelType)
	assert.Equal(t, plaintextKey, capturedChannelKey)
	assert.Equal(t, "https://openrouter.ai/api", capturedBaseURL)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", capturedModel)
	assert.Equal(t, "anthropic/claude-3.5-sonnet", gjson.GetBytes(capturedRewrittenBody, "model").String())
}

func TestBYOKRouterSuccessCustom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	plaintextKey := "sk-custom-secret-key-9999"
	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderCustom,
		Name:     "Local vLLM",
		BaseURL:  "https://ai.example.com/v1",
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey(plaintextKey))
	require.NoError(t, model.CreateUserProvider(provider))

	var capturedChannelType int
	var capturedChannelKey string
	var capturedBaseURL string
	var capturedIsBYOK bool

	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}

		capturedIsBYOK = c.GetBool("is_byok")
		capturedChannelType = common.GetContextKeyInt(c, constant.ContextKeyChannelType)
		capturedChannelKey = common.GetContextKeyString(c, constant.ContextKeyChannelKey)
		capturedBaseURL = common.GetContextKeyString(c, constant.ContextKeyChannelBaseUrl)
		c.Status(http.StatusOK)
	})

	reqBody := `{"model": "llama-3-70b", "messages": []}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Provider", "custom")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, capturedIsBYOK)
	assert.Equal(t, constant.ChannelTypeOpenAI, capturedChannelType)
	assert.Equal(t, plaintextKey, capturedChannelKey)
	assert.Equal(t, "https://ai.example.com", capturedBaseURL)
}

func TestBYOKBypassesFreeTierQuota(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	// 1. Configure OpenRouter BYOK for the user
	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderOpenRouter,
		Name:     "My OpenRouter",
		BaseURL:  "https://openrouter.ai/api",
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-or-test-secret-key"))
	require.NoError(t, model.CreateUserProvider(provider))

	// 2. Exhaust user's free tier quota (set used = 20)
	now := time.Now()
	for i := 0; i < int(model.FreeTierDailyLimit); i++ {
		allowed, err := model.ReserveFreeTierRequest(user.Id, now)
		require.NoError(t, err)
		require.True(t, allowed)
	}

	// Verify next standard free request is rejected
	twentyFirst, err := model.ReserveFreeTierRequest(user.Id, now)
	require.NoError(t, err)
	require.False(t, twentyFirst, "twenty-first standard request must be blocked")

	// 3. Setup router with BYOKRouter + FreeTierQuota in pipeline
	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		FreeTierQuota()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusNoContent)
	})

	// 4. Send non-BYOK request -> should be rejected with 429
	wNormal := httptest.NewRecorder()
	normalBody := `{"model": "gpt-4o", "messages": []}`
	reqNormal := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(normalBody))
	reqNormal.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wNormal, reqNormal)
	assert.Equal(t, http.StatusTooManyRequests, wNormal.Code)
	assert.Contains(t, wNormal.Body.String(), "FREE_DAILY_LIMIT")

	// 5. Send BYOK request -> MUST succeed and bypass FreeTierQuota!
	wBYOK := httptest.NewRecorder()
	byokBody := `{"model": "openrouter/anthropic/claude-3.5-sonnet", "messages": []}`
	reqBYOK := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(byokBody))
	reqBYOK.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wBYOK, reqBYOK)
	assert.Equal(t, http.StatusNoContent, wBYOK.Code)
}

func TestBYOKBypassesResponsesFreeTierQuota(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderCustom,
		Name:     "Custom Endpoint",
		BaseURL:  "https://custom-ai.example.com",
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-custom-secret-key"))
	require.NoError(t, model.CreateUserProvider(provider))

	// Exhaust quota
	now := time.Now()
	for i := 0; i < int(model.FreeTierDailyLimit); i++ {
		_, _ = model.ReserveFreeTierRequest(user.Id, now)
	}

	router := gin.New()
	router.POST("/v1/responses", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		FreeTierQuota()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusNoContent)
	})

	// BYOK request to /v1/responses
	w := httptest.NewRecorder()
	body := `{"model": "custom/deepseek-v3", "input": "hi"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestBYOKUserIsolationRelay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userA, cleanupA := setupBYOKTestEnv(t)
	defer cleanupA()

	// Create User B in the same DB
	userB := &model.User{
		Username: "user-b-" + common.GetRandomString(8),
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "aff-b-" + common.GetRandomString(8),
	}
	require.NoError(t, model.DB.Create(userB).Error)

	// User B configures an OpenRouter provider
	providerB := &model.UserProvider{
		UserId:   userB.Id,
		Provider: model.ProviderOpenRouter,
		Name:     "User B OpenRouter",
		BaseURL:  "https://openrouter.ai/api",
		Enabled:  true,
	}
	require.NoError(t, providerB.SetAPIKey("sk-user-b-secret-key-12345"))
	require.NoError(t, model.CreateUserProvider(providerB))

	// User A tries to invoke BYOK relay, even injecting "user_id": userB.Id in body
	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		// Authenticated as User A
		c.Set("id", userA.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	body := fmt.Sprintf(`{"user_id": %d, "model": "openrouter/anthropic/claude-3.5-sonnet", "messages": []}`, userB.Id)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Must fail because User A does NOT own this provider. user_id in body must NOT be trusted!
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "byok_provider_not_found")
	assert.NotContains(t, w.Body.String(), "sk-user-b-secret-key-12345")
}

func TestBYOKDistributeBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderOpenRouter,
		Name:     "My OpenRouter",
		BaseURL:  "https://openrouter.ai/api",
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-or-test-secret-key"))
	require.NoError(t, model.CreateUserProvider(provider))

	// Router with BYOKRouter and Distribute, but ZERO channels in DB
	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		Distribute()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	body := `{"model": "openrouter/meta-llama/llama-3-8b", "messages": []}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Must succeed through Distribute() because BYOK bypasses system channel selection
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBYOKRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	// Use dedicated user ID to avoid sharing in-memory rate limit counts with prior tests
	user.Id = 88888

	// Set test rate limits: 3 requests per 60 seconds
	resetLimits := SetBYOKRateLimitsForTest(3, 60, 10)
	defer resetLimits()

	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderOpenRouter,
		Name:     "Rate Limited Provider",
		BaseURL:  "https://openrouter.ai/api",
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-or-test-secret-key"))
	require.NoError(t, model.CreateUserProvider(provider))

	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		BYOKRouter()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	body := `{"model": "openrouter/meta-llama/llama-3-8b", "messages": []}`

	// First 3 requests should succeed
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should pass rate limit", i)
	}

	// 4th request must be rejected with 429
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), "byok_rate_limit_exceeded")
}

func TestBYOKConcurrencyLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, cleanup := setupBYOKTestEnv(t)
	defer cleanup()

	// Use dedicated user ID for concurrency test
	user.Id = 99999

	// Set test limits: max 2 concurrent requests
	resetLimits := SetBYOKRateLimitsForTest(100, 60, 2)
	defer resetLimits()

	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: model.ProviderOpenRouter,
		Name:     "Concurrency Limited Provider",
		BaseURL:  "https://openrouter.ai/api",
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-or-test-secret-key"))
	require.NoError(t, model.CreateUserProvider(provider))

	holdCh1 := make(chan struct{})
	holdCh2 := make(chan struct{})
	readyCh := make(chan struct{}, 2)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", user.Id)
		c.Next()
	})
	router.Use(BYOKRouter())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		readyCh <- struct{}{}
		// Hold handler open
		if c.GetHeader("X-Req-ID") == "1" {
			<-holdCh1
		} else if c.GetHeader("X-Req-ID") == "2" {
			<-holdCh2
		}
		c.Status(http.StatusOK)
	})

	body := `{"model": "openrouter/meta-llama/llama-3-8b", "messages": []}`

	// Start request 1 (async)
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Req-ID", "1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}()
	<-readyCh

	// Start request 2 (async)
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Req-ID", "2")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}()
	<-readyCh

	// Start request 3 (should immediately fail with 429 concurrency limit exceeded)
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("X-Req-ID", "3")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Contains(t, w3.Body.String(), "byok_concurrency_limit_exceeded")

	// Release request 1
	close(holdCh1)
	time.Sleep(30 * time.Millisecond)

	// Now request 4 should succeed
	req4 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("X-Req-ID", "4")
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	close(holdCh2)
}

// Ensure unused io import doesn't cause compiler error
var _ = io.EOF


