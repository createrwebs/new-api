package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testControllerBYOKKey = "secure-controller-test-key-32bytes"

func setupUserProviderControllerTestDB(t *testing.T) {
	t.Helper()
	common.SetBYOKKeyForTest(testControllerBYOKKey)
	gin.SetMode(gin.TestMode)

	previousDB := model.DB
	previousType := common.MainDatabaseType()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserProvider{}))

	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.ResetBYOKKeyForTest()
	})
}

func createControllerTestUser(t *testing.T, prefix string) *model.User {
	t.Helper()
	user := &model.User{
		Username: prefix + "-" + common.GetRandomString(8),
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "aff-" + common.GetRandomString(10),
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

type mockTester struct {
	shouldFail bool
	failError  error
	calledWith string
}

func (m *mockTester) Test(ctx context.Context, provider *model.UserProvider, apiKey string) error {
	m.calledWith = apiKey
	if m.shouldFail {
		if m.failError != nil {
			return m.failError
		}
		return errors.New("mock connection failure")
	}
	return nil
}

func setupProviderRouter(userID int) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("id", userID)
		c.Next()
	})
	r.GET("/api/user/providers", ListUserProviders)
	r.POST("/api/user/providers", CreateUserProvider)
	r.PUT("/api/user/providers/:id", UpdateUserProvider)
	r.DELETE("/api/user/providers/:id", DeleteUserProvider)
	r.POST("/api/user/providers/:id/test", TestUserProvider)
	return r
}

func TestCreateAndListUserProvider(t *testing.T) {
	setupUserProviderControllerTestDB(t)
	user := createControllerTestUser(t, "ctrl-user")
	router := setupProviderRouter(user.Id)

	// Create OpenRouter provider
	createPayload := CreateUserProviderRequest{
		Provider: "openrouter",
		Name:     "Personal OpenRouter",
		APIKey:   "sk-or-v1-abcdef12345678",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/user/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var createRes map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &createRes))
	assert.True(t, createRes["success"].(bool))

	data := createRes["data"].(map[string]any)
	assert.Equal(t, "openrouter", data["provider"])
	assert.Equal(t, "Personal OpenRouter", data["name"])
	assert.Equal(t, model.DefaultOpenRouterBaseURL, data["base_url"])
	assert.Equal(t, "****5678", data["api_key"])
	assert.True(t, data["enabled"].(bool))

	// Plaintext key MUST NOT appear anywhere in the response body
	assert.NotContains(t, resp.Body.String(), "sk-or-v1-abcdef12345678")

	// List providers
	listReq := httptest.NewRequest(http.MethodGet, "/api/user/providers", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)

	require.Equal(t, http.StatusOK, listResp.Code)
	var listRes map[string]any
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listRes))
	assert.True(t, listRes["success"].(bool))
	providers := listRes["data"].([]any)
	require.Len(t, providers, 1)

	firstProvider := providers[0].(map[string]any)
	assert.Equal(t, "****5678", firstProvider["api_key"])
	assert.NotContains(t, listResp.Body.String(), "sk-or-v1-abcdef12345678")
}

func TestCreateCustomProviderValidation(t *testing.T) {
	setupUserProviderControllerTestDB(t)
	user := createControllerTestUser(t, "ctrl-custom")
	router := setupProviderRouter(user.Id)

	// Custom without base_url must fail
	body, _ := json.Marshal(CreateUserProviderRequest{
		Provider: "custom",
		Name:     "Custom AI",
		APIKey:   "sk-custom-secret-key",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/user/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var res map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.False(t, res["success"].(bool))
	assert.Contains(t, res["message"].(string), "base_url is required")

	// Custom with invalid base_url must fail
	body, _ = json.Marshal(CreateUserProviderRequest{
		Provider: "custom",
		Name:     "Custom AI",
		BaseURL:  "ftp://not-http.example.com",
		APIKey:   "sk-custom-secret-key",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/user/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.False(t, res["success"].(bool))

	// Custom with valid base_url must succeed
	body, _ = json.Marshal(CreateUserProviderRequest{
		Provider: "custom",
		Name:     "Custom AI",
		BaseURL:  "https://openrouter.ai/api/v1///",
		APIKey:   "sk-custom-secret-key-1234",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/user/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.True(t, res["success"].(bool))
	data := res["data"].(map[string]any)
	assert.Equal(t, "https://openrouter.ai/api/v1", data["base_url"])
	assert.Equal(t, "****1234", data["api_key"])
}

func TestUpdateProviderRetainsOrReplacesKey(t *testing.T) {
	setupUserProviderControllerTestDB(t)
	user := createControllerTestUser(t, "ctrl-update")
	router := setupProviderRouter(user.Id)

	initialKey := "sk-initial-secret-1111"
	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: "openrouter",
		Name:     "Initial Name",
		BaseURL:  model.DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey(initialKey))
	require.NoError(t, model.CreateUserProvider(provider))

	// Update only name and disabled (no api_key provided)
	disabled := false
	body, _ := json.Marshal(UpdateUserProviderRequest{
		Name:    "Updated Name Only",
		Enabled: &disabled,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/user/providers/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var res map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.True(t, res["success"].(bool))

	// Verify key in DB is still decryptable to initialKey
	stored, err := model.GetUserProviderByID(user.Id, provider.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name Only", stored.Name)
	assert.False(t, stored.Enabled)
	decrypted, err := stored.DecryptAPIKey()
	require.NoError(t, err)
	assert.Equal(t, initialKey, decrypted)

	// Update with new API key
	newKey := "sk-replacement-secret-9999"
	body, _ = json.Marshal(UpdateUserProviderRequest{
		APIKey: newKey,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/user/providers/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.True(t, res["success"].(bool))
	data := res["data"].(map[string]any)
	assert.Equal(t, "****9999", data["api_key"])

	storedAfterKeyUpdate, err := model.GetUserProviderByID(user.Id, provider.Id)
	require.NoError(t, err)
	decryptedNew, err := storedAfterKeyUpdate.DecryptAPIKey()
	require.NoError(t, err)
	assert.Equal(t, newKey, decryptedNew)
}

func TestDeleteUserProvider(t *testing.T) {
	setupUserProviderControllerTestDB(t)
	user := createControllerTestUser(t, "ctrl-del")
	router := setupProviderRouter(user.Id)

	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: "openrouter",
		Name:     "To Delete",
		BaseURL:  model.DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey("sk-temp-key"))
	require.NoError(t, model.CreateUserProvider(provider))

	req := httptest.NewRequest(http.MethodDelete, "/api/user/providers/1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var res map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.True(t, res["success"].(bool))

	_, err := model.GetUserProviderByID(user.Id, provider.Id)
	assert.ErrorIs(t, err, model.ErrProviderNotFound)
}

func TestControllerUserIsolation(t *testing.T) {
	setupUserProviderControllerTestDB(t)
	userA := createControllerTestUser(t, "user-a")
	userB := createControllerTestUser(t, "user-b")

	providerA := &model.UserProvider{
		UserId:   userA.Id,
		Provider: "openrouter",
		Name:     "User A Provider",
		BaseURL:  model.DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, providerA.SetAPIKey("sk-secret-user-a"))
	require.NoError(t, model.CreateUserProvider(providerA))

	// User B router
	routerB := setupProviderRouter(userB.Id)

	// User B trying to UPDATE User A's provider
	updateBody, _ := json.Marshal(UpdateUserProviderRequest{Name: "Stolen Name"})
	req := httptest.NewRequest(http.MethodPut, "/api/user/providers/1", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	routerB.ServeHTTP(resp, req)

	var res map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.False(t, res["success"].(bool), "User B must not be able to update User A's provider")

	// User B trying to DELETE User A's provider
	req = httptest.NewRequest(http.MethodDelete, "/api/user/providers/1", nil)
	resp = httptest.NewRecorder()
	routerB.ServeHTTP(resp, req)

	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.False(t, res["success"].(bool), "User B must not be able to delete User A's provider")

	// User B trying to TEST User A's provider
	req = httptest.NewRequest(http.MethodPost, "/api/user/providers/1/test", nil)
	resp = httptest.NewRecorder()
	routerB.ServeHTTP(resp, req)

	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.False(t, res["success"].(bool), "User B must not be able to test User A's provider")

	// Verify User A provider untouched
	intact, err := model.GetUserProviderByID(userA.Id, providerA.Id)
	require.NoError(t, err)
	assert.Equal(t, "User A Provider", intact.Name)
}

func TestTestUserProviderEndpoint(t *testing.T) {
	setupUserProviderControllerTestDB(t)
	user := createControllerTestUser(t, "test-prov")
	router := setupProviderRouter(user.Id)

	rawSecret := "sk-real-test-secret-key"
	provider := &model.UserProvider{
		UserId:   user.Id,
		Provider: "openrouter",
		Name:     "Testable Provider",
		BaseURL:  model.DefaultOpenRouterBaseURL,
		Enabled:  true,
	}
	require.NoError(t, provider.SetAPIKey(rawSecret))
	require.NoError(t, model.CreateUserProvider(provider))

	// Mock successful tester
	mock := &mockTester{shouldFail: false}
	previousTester := DefaultProviderTester
	DefaultProviderTester = mock
	defer func() { DefaultProviderTester = previousTester }()

	req := httptest.NewRequest(http.MethodPost, "/api/user/providers/1/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var res map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	assert.True(t, res["success"].(bool))
	assert.Contains(t, res["data"].(map[string]any)["message"], "successful")
	assert.Equal(t, rawSecret, mock.calledWith)

	// Mock failed tester (should not echo secrets in error response)
	mock.shouldFail = true
	mock.failError = errors.New("authentication failed: invalid API key")
	respFail := httptest.NewRecorder()
	router.ServeHTTP(respFail, httptest.NewRequest(http.MethodPost, "/api/user/providers/1/test", nil))

	var resFail map[string]any
	require.NoError(t, json.Unmarshal(respFail.Body.Bytes(), &resFail))
	assert.False(t, resFail["success"].(bool))
	assert.Contains(t, resFail["message"].(string), "invalid API key")
	assert.NotContains(t, respFail.Body.String(), rawSecret)
}

func TestTestUserProviderEndpointSSRFProtection(t *testing.T) {
	setupUserProviderControllerTestDB(t)

	// Ensure DefaultProviderTester is using the real HTTP tester
	DefaultProviderTester = &defaultHTTPProviderTester{}

	privateTargets := []string{
		"http://127.0.0.1:8000",
		"http://169.254.169.254",
		"http://localhost:8080",
		"http://[::1]:8080",
	}

	for _, target := range privateTargets {
		user := createControllerTestUser(t, "ssrf-test")
		router := setupProviderRouter(user.Id)

		provider := &model.UserProvider{
			UserId:   user.Id,
			Provider: "custom",
			Name:     "Private Provider",
			BaseURL:  target,
			Enabled:  true,
		}
		require.NoError(t, provider.SetAPIKey("sk-ssrf-secret"))
		// Directly insert to DB bypassing NormalizeBaseURL
		require.NoError(t, model.DB.Create(provider).Error)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/user/providers/%d/test", provider.Id), nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		var res map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res), "target %s failed to parse json", target)
		assert.False(t, res["success"].(bool), "Target %s must fail test due to SSRF protection", target)
		assert.Contains(t, res["message"].(string), "SSRF protection")
	}
}

func TestCreateUserProviderRejectsPrivateBaseURL(t *testing.T) {
	setupUserProviderControllerTestDB(t)

	targets := []string{
		"http://127.0.0.1:8000",
		"http://localhost:8080",
		"http://169.254.169.254",
		"http://192.168.1.5",
	}

	for _, target := range targets {
		user := createControllerTestUser(t, "create-ssrf")
		router := setupProviderRouter(user.Id)

		createPayload := CreateUserProviderRequest{
			Provider: "custom",
			Name:     "Malicious Custom Provider",
			BaseURL:  target,
			APIKey:   "sk-test-key",
		}
		body, _ := json.Marshal(createPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/user/providers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		var res map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		assert.False(t, res["success"].(bool), "Target %s must be rejected on creation", target)
		assert.Contains(t, res["message"].(string), "base_url must be a valid http or https URL")
	}
}

