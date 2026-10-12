package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type huobaoIdentityResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		UserID         int    `json:"user_id"`
		TokenID        int    `json:"token_id"`
		Username       string `json:"username"`
		DisplayName    string `json:"display_name"`
		AccountStatus  int    `json:"account_status"`
		TokenStatus    int    `json:"token_status"`
		ExpiresAt      int64  `json:"expires_at"`
		Quota          int    `json:"quota"`
		UnlimitedQuota bool   `json:"unlimited_quota"`
		UserQuota      int    `json:"user_quota"`
		// Fields that MUST NOT exist
		Password     *string `json:"password,omitempty"`
		Key          *string `json:"key,omitempty"`
		OriginalKey  *string `json:"original_key,omitempty"`
		AdminPrivilege *bool `json:"is_admin,omitempty"`
	} `json:"data"`
}

type huobaoCreditsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		UserID           int     `json:"user_id"`
		TokenID          int     `json:"token_id"`
		QuotaPerCredit   int     `json:"quota_per_credit"`
		UserQuota        int     `json:"user_quota"`
		TokenQuota       int     `json:"token_quota"`
		UnlimitedQuota   bool    `json:"unlimited_quota"`
		SpendableQuota   int     `json:"spendable_quota"`
		SpendableCredits float64 `json:"spendable_credits"`
	} `json:"data"`
}

func setupHuobaoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	previousDB := model.DB
	previousRedis := common.RedisEnabled
	model.DB = db
	common.RedisEnabled = false

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.WalletPreConsumeRecord{}))

	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedis
	})
	return db
}

func setupHuobaoRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/huobao")
	{
		api.GET("/identity", GetHuobaoIdentity)
		api.GET("/credits", GetHuobaoCredits)
		api.POST("/video/quote", QuoteHuobaoVideoTask)
		api.POST("/video/reserve", ReserveHuobaoVideoTask)
		api.POST("/video/settle", SettleHuobaoVideoTask)
	}
	v1 := router.Group("/v1/huobao")
	{
		v1.GET("/identity", GetHuobaoIdentity)
		v1.GET("/credits", GetHuobaoCredits)
		v1.POST("/video/quote", QuoteHuobaoVideoTask)
		v1.POST("/video/reserve", ReserveHuobaoVideoTask)
		v1.POST("/video/settle", SettleHuobaoVideoTask)
	}
	return router
}

func TestHuobaoIdentity_A0_Contract(t *testing.T) {
	db := setupHuobaoTestDB(t)
	router := setupHuobaoRouter()

	now := time.Now().Unix()

	// 1. Create User A (Enabled, with quota)
	userA := model.User{
		Username:    "user_a",
		DisplayName: "User A Display",
		Status:      common.UserStatusEnabled,
		Quota:       10000000, // 10,000 credits
		AffCode:     "aff_user_a",
	}
	require.NoError(t, db.Create(&userA).Error)

	// 2. Create User B (Enabled, with quota)
	userB := model.User{
		Username:    "user_b",
		DisplayName: "User B Display",
		Status:      common.UserStatusEnabled,
		Quota:       5000000,
		AffCode:     "aff_user_b",
	}
	require.NoError(t, db.Create(&userB).Error)

	// 3. Create User C (Banned/Disabled)
	userC := model.User{
		Username:    "user_banned",
		DisplayName: "Banned User",
		Status:      common.UserStatusDisabled,
		Quota:       2000000,
		AffCode:     "aff_user_c",
	}
	require.NoError(t, db.Create(&userC).Error)

	// User A - Key 1 (Active, Limited Quota)
	tokenA1 := model.Token{
		UserId:         userA.Id,
		Key:            "tok_a1_valid",
		KeyHash:        model.TokenHash("tok_a1_valid"),
		Status:         common.TokenStatusEnabled,
		Name:           "Key A1",
		RemainQuota:    500000,
		UnlimitedQuota: false,
		ExpiredTime:    -1,
	}
	require.NoError(t, db.Create(&tokenA1).Error)

	// User A - Key 2 (Active, Different Key string, Same User)
	tokenA2 := model.Token{
		UserId:         userA.Id,
		Key:            "tok_a2_second_key",
		KeyHash:        model.TokenHash("tok_a2_second_key"),
		Status:         common.TokenStatusEnabled,
		Name:           "Key A2",
		RemainQuota:    200000,
		UnlimitedQuota: false,
		ExpiredTime:    -1,
	}
	require.NoError(t, db.Create(&tokenA2).Error)

	// User A - Key 3 (Active, BUT Zero Quota / Exhausted)
	tokenA3ZeroQuota := model.Token{
		UserId:         userA.Id,
		Key:            "tok_a3_zero_quota",
		KeyHash:        model.TokenHash("tok_a3_zero_quota"),
		Status:         common.TokenStatusExhausted,
		Name:           "Key A3 Zero Quota",
		RemainQuota:    0,
		UnlimitedQuota: false,
		ExpiredTime:    -1,
	}
	require.NoError(t, db.Create(&tokenA3ZeroQuota).Error)

	// User A - Key 4 (Disabled Token)
	tokenA4Disabled := model.Token{
		UserId:         userA.Id,
		Key:            "tok_a4_disabled",
		KeyHash:        model.TokenHash("tok_a4_disabled"),
		Status:         common.TokenStatusDisabled,
		Name:           "Key A4 Disabled",
		RemainQuota:    1000000,
		UnlimitedQuota: false,
		ExpiredTime:    -1,
	}
	require.NoError(t, db.Create(&tokenA4Disabled).Error)

	// User A - Key 5 (Expired Token)
	tokenA5Expired := model.Token{
		UserId:         userA.Id,
		Key:            "tok_a5_expired",
		KeyHash:        model.TokenHash("tok_a5_expired"),
		Status:         common.TokenStatusEnabled,
		Name:           "Key A5 Expired",
		RemainQuota:    1000000,
		UnlimitedQuota: false,
		ExpiredTime:    now - 3600, // expired 1 hour ago
	}
	require.NoError(t, db.Create(&tokenA5Expired).Error)

	// User B - Key 1 (Active)
	tokenB1 := model.Token{
		UserId:         userB.Id,
		Key:            "tok_b1_valid",
		KeyHash:        model.TokenHash("tok_b1_valid"),
		Status:         common.TokenStatusEnabled,
		Name:           "Key B1",
		RemainQuota:    100000,
		UnlimitedQuota: false,
		ExpiredTime:    -1,
	}
	require.NoError(t, db.Create(&tokenB1).Error)

	// User C (Banned User) - Key 1
	tokenC1 := model.Token{
		UserId:         userC.Id,
		Key:            "tok_c1_banned_user",
		KeyHash:        model.TokenHash("tok_c1_banned_user"),
		Status:         common.TokenStatusEnabled,
		Name:           "Key C1",
		RemainQuota:    1000000,
		UnlimitedQuota: false,
		ExpiredTime:    -1,
	}
	require.NoError(t, db.Create(&tokenC1).Error)

	t.Run("A0.1: Key A1 succeeds and returns User A", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_a1_valid")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp huobaoIdentityResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, userA.Id, resp.Data.UserID)
		assert.Equal(t, tokenA1.Id, resp.Data.TokenID)
		assert.Equal(t, "user_a", resp.Data.Username)
		assert.Equal(t, "User A Display", resp.Data.DisplayName)
		assert.Equal(t, 500000, resp.Data.Quota)
		assert.Equal(t, 10000000, resp.Data.UserQuota)
	})

	t.Run("A0.2: Multiple keys for the same user return the exact same user_id", func(t *testing.T) {
		// Key A2 request
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer tok_a2_second_key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp huobaoIdentityResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		// MUST be userA.Id
		assert.Equal(t, userA.Id, resp.Data.UserID)
		assert.Equal(t, tokenA2.Id, resp.Data.TokenID)
	})

	t.Run("A0.3: Key for user B returns different user_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_b1_valid")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp huobaoIdentityResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, userB.Id, resp.Data.UserID)
		assert.NotEqual(t, userA.Id, resp.Data.UserID)
	})

	t.Run("A0.4: Key with zero quota / exhausted STILL SUCCEEDS to log in (Invariant A0)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_a3_zero_quota")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp huobaoIdentityResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, userA.Id, resp.Data.UserID)
		assert.Equal(t, 0, resp.Data.Quota)
	})

	t.Run("A0.5: Disabled token is rejected with 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_a4_disabled")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var resp huobaoIdentityResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "disabled")
	})

	t.Run("A0.6: Expired token is rejected with 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_a5_expired")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var resp huobaoIdentityResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "expired")
	})

	t.Run("A0.7: Banned or disabled user account is rejected with 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_c1_banned_user")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var resp huobaoIdentityResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Message, "banned")
	})

	t.Run("A0.8: Missing or non-existent token is rejected with 401 Unauthorized", func(t *testing.T) {
		// Missing header
		req1 := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusUnauthorized, w1.Code)

		// Non-existent key
		req2 := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req2.Header.Set("Authorization", "Bearer sk-completely-non-existent-key")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("A0.9: No secrets or credentials leaked in response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/identity", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_a1_valid")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		bodyStr := w.Body.String()
		assert.NotContains(t, bodyStr, "tok_a1_valid")
		assert.NotContains(t, bodyStr, "password")
		assert.NotContains(t, bodyStr, "secret")
	})

	t.Run("A0.10: Credits endpoint calculates spendable quota correctly", func(t *testing.T) {
		// For token A1: User Quota = 10,000,000, Token Quota = 500,000
		// Spendable Quota = min(10000000, 500000) = 500,000
		// Spendable Credits = 500,000 / 1,000 = 500.0
		req := httptest.NewRequest(http.MethodGet, "/api/huobao/credits", nil)
		req.Header.Set("Authorization", "Bearer sk-tok_a1_valid")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp huobaoCreditsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, service.QuotaPerCredit, resp.Data.QuotaPerCredit)
		assert.Equal(t, 500000, resp.Data.SpendableQuota)
		assert.Equal(t, float64(500), resp.Data.SpendableCredits)
	})
}

func TestHuobaoVideoBilling_A5(t *testing.T) {
	_ = setupHuobaoTestDB(t)
	router := setupHuobaoRouter()

	// Setup a user with quota
	user := model.User{
		Username: "video_user",
		Status:   common.UserStatusEnabled,
		Quota:    100000, // 100,000 Quota = 100 Credits
		AffCode:  "aff_video_user",
	}
	require.NoError(t, model.DB.Create(&user).Error)

	token := model.Token{
		UserId:         user.Id,
		Key:            "tok_video_token_valid",
		Status:         common.TokenStatusEnabled,
		RemainQuota:    100000,
		UnlimitedQuota: false,
	}
	require.NoError(t, model.DB.Create(&token).Error)

	// Setup zero-quota user
	zeroUser := model.User{
		Username: "zero_user",
		Status:   common.UserStatusEnabled,
		Quota:    0,
		AffCode:  "aff_zero_user",
	}
	require.NoError(t, model.DB.Create(&zeroUser).Error)

	zeroToken := model.Token{
		UserId:         zeroUser.Id,
		Key:            "tok_zero_video_token",
		Status:         common.TokenStatusEnabled,
		RemainQuota:    0,
		UnlimitedQuota: false,
	}
	require.NoError(t, model.DB.Create(&zeroToken).Error)

	t.Run("A5.1: Video quote returns correct rate and estimated credits", func(t *testing.T) {
		body := `{"model":"wan3.0-video","resolution":"720P","duration_seconds":6}`
		req := httptest.NewRequest(http.MethodPost, "/api/huobao/video/quote", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer sk-tok_video_token_valid")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Model            string  `json:"model"`
				Resolution       string  `json:"resolution"`
				DurationSeconds  int     `json:"duration_seconds"`
				QuotaPerSecond   int     `json:"quota_per_second"`
				EstimatedQuota   int     `json:"estimated_quota"`
				EstimatedCredits float64 `json:"estimated_credits"`
				CanAfford        bool    `json:"can_afford"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, 2000, resp.Data.QuotaPerSecond) // 720P rate = 2000 quota/s
		assert.Equal(t, 12000, resp.Data.EstimatedQuota) // 6s * 2000 = 12000 quota
		assert.Equal(t, 12.0, resp.Data.EstimatedCredits)
		assert.True(t, resp.Data.CanAfford)
	})

	t.Run("A5.2: Zero-quota user cannot reserve video task (402 Payment Required)", func(t *testing.T) {
		body := `{"request_id":"req-zero-test-1","model":"wan3.0-video","resolution":"720P","max_duration_seconds":10}`
		req := httptest.NewRequest(http.MethodPost, "/api/huobao/video/reserve", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer sk-tok_zero_video_token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusPaymentRequired, w.Code)
		assert.Contains(t, w.Body.String(), "insufficient wallet or token quota")
	})

	t.Run("A5.3: Valid user can reserve video task (atomic pre-consume created, status pending)", func(t *testing.T) {
		body := `{"request_id":"req-valid-1","model":"wan3.0-video","resolution":"720P","max_duration_seconds":10}`
		req := httptest.NewRequest(http.MethodPost, "/api/huobao/video/reserve", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer sk-tok_video_token_valid")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				RequestId     string `json:"request_id"`
				ReservedQuota int    `json:"reserved_quota"`
				Status        string `json:"status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, "req-valid-1", resp.Data.RequestId)
		assert.Equal(t, 20000, resp.Data.ReservedQuota) // 10s * 2000 = 20,000 quota
		assert.Equal(t, "pending", resp.Data.Status)

		// Verify record in DB
		var rec model.WalletPreConsumeRecord
		require.NoError(t, model.DB.Where("request_id = ?", "req-valid-1").First(&rec).Error)
		assert.Equal(t, 20000, rec.PreConsumed)
		assert.Equal(t, "pending", rec.Status)
	})

	t.Run("A5.4: Failed video task settles with 0 charge and refunds full reservation", func(t *testing.T) {
		// First reserve another task
		resBody := `{"request_id":"req-fail-1","model":"wan3.0-video","resolution":"720P","max_duration_seconds":10}`
		req1 := httptest.NewRequest(http.MethodPost, "/api/huobao/video/reserve", strings.NewReader(resBody))
		req1.Header.Set("Authorization", "Bearer sk-tok_video_token_valid")
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Record quota before settle
		var uBefore model.User
		require.NoError(t, model.DB.Where("id = ?", user.Id).First(&uBefore).Error)

		// Now settle with success=false
		settleBody := `{"request_id":"req-fail-1","success":false,"actual_duration_seconds":0,"error_message":"prompt violation"}`
		req2 := httptest.NewRequest(http.MethodPost, "/api/huobao/video/settle", strings.NewReader(settleBody))
		req2.Header.Set("Authorization", "Bearer sk-tok_video_token_valid")
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Contains(t, w2.Body.String(), `"status":"refunded"`)
		assert.Contains(t, w2.Body.String(), `"settled_quota":0`)

		// Verify user quota was restored (increased by 20,000)
		var uAfter model.User
		require.NoError(t, model.DB.Where("id = ?", user.Id).First(&uAfter).Error)
		assert.Equal(t, uBefore.Quota+20000, uAfter.Quota)
	})

	t.Run("A5.5: Successful video task settles exact output seconds and refunds excess reservation", func(t *testing.T) {
		// Settle req-valid-1 (which reserved 20,000 for 10s) with 6s actual duration = 12,000 quota
		// Excess 8,000 quota should be refunded to user wallet!
		var uBefore model.User
		require.NoError(t, model.DB.Where("id = ?", user.Id).First(&uBefore).Error)

		settleBody := `{"request_id":"req-valid-1","success":true,"actual_duration_seconds":6}`
		req := httptest.NewRequest(http.MethodPost, "/api/huobao/video/settle", strings.NewReader(settleBody))
		req.Header.Set("Authorization", "Bearer sk-tok_video_token_valid")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				RequestId             string `json:"request_id"`
				ActualDurationSeconds int    `json:"actual_duration_seconds"`
				SettledQuota          int    `json:"settled_quota"`
				Status                string `json:"status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, 12000, resp.Data.SettledQuota) // 6s * 2000 = 12,000 quota
		assert.Equal(t, "settled", resp.Data.Status)

		// User quota should receive 8,000 refund (20,000 - 12,000)
		var uAfter model.User
		require.NoError(t, model.DB.Where("id = ?", user.Id).First(&uAfter).Error)
		assert.Equal(t, uBefore.Quota+8000, uAfter.Quota)
	})

	t.Run("A5.6: Duplicate settle calls are idempotent and do not double-charge or double-refund", func(t *testing.T) {
		var uBefore model.User
		require.NoError(t, model.DB.Where("id = ?", user.Id).First(&uBefore).Error)

		// Call settle again on already settled req-valid-1
		settleBody := `{"request_id":"req-valid-1","success":true,"actual_duration_seconds":6}`
		req := httptest.NewRequest(http.MethodPost, "/api/huobao/video/settle", strings.NewReader(settleBody))
		req.Header.Set("Authorization", "Bearer sk-tok_video_token_valid")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// User quota must NOT change
		var uAfter model.User
		require.NoError(t, model.DB.Where("id = ?", user.Id).First(&uAfter).Error)
		assert.Equal(t, uBefore.Quota, uAfter.Quota)
	})
}
