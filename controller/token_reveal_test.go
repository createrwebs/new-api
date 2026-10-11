package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

	rawKeyA := "sk-TEST-RELAY-SECRET-DO-NOT-LEAK-A"
	tokenA := &model.Token{
		UserId:         userA.Id,
		Name:           "tokenA",
		Key:            rawKeyA,
		Status:         common.TokenStatusEnabled,
		UnlimitedQuota: true,
	}
	require.NoError(t, tokenA.Insert())

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
		assert.NotContains(t, resp.Body.String(), rawKeyA, "GetToken body must never contain raw plaintext secret")
		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		data, ok := body["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, model.MaskTokenKey(rawKeyA), data["key"])
		assert.NotEqual(t, rawKeyA, data["key"], "GetToken must never return full plaintext key")
	}

	// 2. GetAllTokens (GET list) returns masked key, NOT plaintext key
	{
		req, _ := http.NewRequest(http.MethodGet, "/api/token/?p=0", nil)
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Set("id", userA.Id)
		c.Set("role", userA.Role)
		GetAllTokens(c)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.NotContains(t, resp.Body.String(), rawKeyA, "GetAllTokens body must never contain raw plaintext secret")
	}

	// 3. Owner userA reveals tokenA key -> 200 OK with full plaintext key
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

	// 4. UserB attempts to reveal tokenA key (IDOR attempt) -> Rejected (error)
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
		assert.NotContains(t, resp.Body.String(), rawKeyA, "Response to unauthorized user must never contain secret")
	}

	// 5. Admin reveals tokenA key -> 200 OK with full plaintext key
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

	// 6. Batch reveal: UserB cannot get tokenA
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
		assert.NotContains(t, resp.Body.String(), rawKeyA)
	}

	// 7. Revoked token -> reveal denied!
	{
		rawKeyRevoked := "sk-TEST-RELAY-SECRET-REVOKED"
		tokenRevoked := &model.Token{
			UserId:         userA.Id,
			Name:           "revokedToken",
			Key:            rawKeyRevoked,
			Status:         common.TokenStatusDisabled, // revoked/disabled
			UnlimitedQuota: true,
		}
		require.NoError(t, tokenRevoked.Insert())

		// Single reveal -> denied
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/token/%d/key", tokenRevoked.Id), nil)
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(tokenRevoked.Id)}}
		c.Set("id", userA.Id)
		c.Set("role", userA.Role)
		GetTokenKey(c)

		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.Equal(t, false, body["success"], "Revoked token must not be revealed")
		assert.NotContains(t, resp.Body.String(), rawKeyRevoked)

		// Batch reveal -> excludes revoked token
		reqBatch, _ := http.NewRequest(http.MethodPost, "/api/token/batch/keys", bytes.NewReader([]byte(fmt.Sprintf(`{"ids": [%d]}`, tokenRevoked.Id))))
		reqBatch.Header.Set("Content-Type", "application/json")
		respBatch := httptest.NewRecorder()
		cBatch, _ := gin.CreateTestContext(respBatch)
		cBatch.Request = reqBatch
		cBatch.Set("id", userA.Id)
		cBatch.Set("role", userA.Role)
		GetTokenKeysBatch(cBatch)

		assert.Equal(t, http.StatusOK, respBatch.Code)
		var bodyBatch map[string]any
		require.NoError(t, json.Unmarshal(respBatch.Body.Bytes(), &bodyBatch))
		dataBatch, ok := bodyBatch["data"].(map[string]any)
		require.True(t, ok)
		keysBatch, ok := dataBatch["keys"].(map[string]any)
		require.True(t, ok)
		assert.Empty(t, keysBatch, "Batch reveal must exclude revoked token")
		assert.NotContains(t, respBatch.Body.String(), rawKeyRevoked)
	}

	// 8. Deleted token -> reveal denied!
	{
		rawKeyDeleted := "sk-TEST-RELAY-SECRET-DELETED"
		tokenDeleted := &model.Token{
			UserId:         userA.Id,
			Name:           "deletedToken",
			Key:            rawKeyDeleted,
			Status:         common.TokenStatusEnabled,
			UnlimitedQuota: true,
		}
		require.NoError(t, tokenDeleted.Insert())
		require.NoError(t, tokenDeleted.Delete()) // Soft delete

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/token/%d/key", tokenDeleted.Id), nil)
		resp := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(resp)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(tokenDeleted.Id)}}
		c.Set("id", userA.Id)
		c.Set("role", userA.Role)
		GetTokenKey(c)

		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.Equal(t, false, body["success"], "Deleted token must not be revealed")
		assert.NotContains(t, resp.Body.String(), rawKeyDeleted)
	}

	// 9. Inspect raw DB row directly: plaintext marker must NOT exist in the database table!
	{
		var rawKeyInDB string
		require.NoError(t, db.Table("tokens").Select("`key`").Where("id = ?", tokenA.Id).Scan(&rawKeyInDB).Error)
		assert.NotEqual(t, rawKeyA, rawKeyInDB, "Database must never contain plaintext secret")
		assert.True(t, strings.HasPrefix(rawKeyInDB, "enc:v1:"), "Database row must be encrypted with enc:v1: prefix")
	}
}
