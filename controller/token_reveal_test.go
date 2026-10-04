package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenRevealAuthorization(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))

	origSecret := common.CryptoSecret
	common.CryptoSecret = "token-reveal-secret-12345678"
	defer func() { common.CryptoSecret = origSecret }()

	userA := &model.User{Username: "userA", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff-a"}
	require.NoError(t, db.Create(userA).Error)

	userB := &model.User{Username: "userB", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff-b"}
	require.NoError(t, db.Create(userB).Error)

	admin := &model.User{Username: "admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "aff-admin"}
	require.NoError(t, db.Create(admin).Error)

	rawKeyA := "sk-user-a-secret-key-11111"
	tokenA := &model.Token{
		UserId:         userA.Id,
		Name:           "tokenA",
		Key:            rawKeyA,
		Status:         common.TokenStatusEnabled,
		UnlimitedQuota: true,
	}
	require.NoError(t, tokenA.Insert())

	router := gin.New()
	router.POST("/api/token/:id/key", func(c *gin.Context) {
		GetTokenKey(c)
	})
	router.GET("/api/token/:id", func(c *gin.Context) {
		GetToken(c)
	})
	router.POST("/api/token/batch/keys", func(c *gin.Context) {
		GetTokenKeysBatch(c)
	})

	// 1. GetToken (GET) returns masked key, NOT full plaintext key
	{
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/token/%d", tokenA.Id), nil)
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(tokenA.Id)}}
		c.Set("id", userA.Id)
		c.Set("role", userA.Role)
		GetToken(c)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		data, ok := body["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, model.MaskTokenKey(rawKeyA), data["key"])
		assert.NotEqual(t, rawKeyA, data["key"], "GetToken must never return full plaintext key")
	}

	// 2. Owner userA reveals tokenA key -> 200 OK with full plaintext key
	{
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/token/%d/key", tokenA.Id), nil)
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(tokenA.Id)}}
		c.Set("id", userA.Id)
		c.Set("role", userA.Role)
		GetTokenKey(c)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		data, ok := body["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, rawKeyA, data["key"])
	}

	// 3. UserB attempts to reveal tokenA key -> Rejected (error)
	{
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/token/%d/key", tokenA.Id), nil)
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(tokenA.Id)}}
		c.Set("id", userB.Id)
		c.Set("role", userB.Role)
		GetTokenKey(c)

		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.Equal(t, false, body["success"], "UserB must not be allowed to reveal UserA's token")
	}

	// 4. Admin reveals tokenA key -> 200 OK with full plaintext key
	{
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/token/%d/key", tokenA.Id), nil)
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(tokenA.Id)}}
		c.Set("id", admin.Id)
		c.Set("role", admin.Role)
		GetTokenKey(c)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		data, ok := body["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, rawKeyA, data["key"])
	}

	// 5. Batch reveal: UserB cannot get tokenA
	{
		reqBody := []byte(fmt.Sprintf(`{"ids": [%d]}`, tokenA.Id))
		req, _ := http.NewRequest(http.MethodPost, "/api/token/batch/keys", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Set("id", userB.Id)
		c.Set("role", userB.Role)
		GetTokenKeysBatch(c)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		data, ok := body["data"].(map[string]any)
		require.True(t, ok)
		keys, ok := data["keys"].(map[string]any)
		require.True(t, ok)
		assert.Empty(t, keys, "UserB must receive empty keys for tokens owned by UserA")
	}
}
