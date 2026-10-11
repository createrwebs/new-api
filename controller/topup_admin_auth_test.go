package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTopUpAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	require.NoError(t, i18n.Init())
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&model.User{}, &model.TopUp{}, &model.Option{}, &model.Log{}, &model.AuditLog{}))
	model.DB, model.LOG_DB = database, database
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, model.EnsureLegacyAccessTokenRetireAt(time.Now().Add(24*time.Hour).Unix()))
	previousRedis := common.RedisEnabled
	previousMemory := common.MemoryCacheEnabled
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = previousRedis
		common.MemoryCacheEnabled = previousMemory
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousType, previousLogType)
		require.NoError(t, sqlDB.Close())
	})
	return database
}

func TestAdminCompleteTopUpAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTopUpAdminTestDB(t)

	// Create test target user
	targetUser := &model.User{
		Username: "target-user",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Quota:    1000,
		AffCode:  "aff-target",
	}
	require.NoError(t, db.Create(targetUser).Error)

	// Create a normal user with access token
	normalToken := "normal-user-token"
	normalUser := &model.User{
		Username:    "normal-caller",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AccessToken: &normalToken,
		AffCode:     "aff-normal",
	}
	require.NoError(t, db.Create(normalUser).Error)

	// Create an admin user with access token
	adminToken := "admin-user-token"
	adminUser := &model.User{
		Username:    "admin-caller",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		AccessToken: &adminToken,
		AffCode:     "aff-admin",
	}
	require.NoError(t, db.Create(adminUser).Error)

	// Create a pending top-up order for targetUser
	tradeNo := "TRADE_TEST_123456"
	topUp := &model.TopUp{
		UserId:          targetUser.Id,
		Amount:          10, // $10 -> 10 * 500,000 = 5,000,000 quota
		Money:           10.0,
		TradeNo:         tradeNo,
		PaymentMethod:   "epay",
		PaymentProvider: model.PaymentProviderEpay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(topUp).Error)

	// Setup router replicating api-router.go:
	// adminRoute := userRoute.Group("/")
	// adminRoute.Use(middleware.AdminAuth())
	// adminRoute.POST("/topup/complete", controller.AdminCompleteTopUp)
	router := gin.New()
	adminRoute := router.Group("/api/user")
	adminRoute.Use(middleware.AdminAuth())
	adminRoute.POST("/topup/complete", AdminCompleteTopUp)

	reqPayload, _ := json.Marshal(map[string]string{"trade_no": tradeNo})

	// 1. Unauthenticated request -> 401 Unauthorized
	{
		req, _ := http.NewRequest(http.MethodPost, "/api/user/topup/complete", bytes.NewReader(reqPayload))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	}

	// 2. Normal user (RoleCommonUser) -> 403 Forbidden
	{
		req, _ := http.NewRequest(http.MethodPost, "/api/user/topup/complete", bytes.NewReader(reqPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+normalToken)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
	}

	// 3. Admin user (RoleAdminUser) completes pending order -> 200 OK & user quota credited
	{
		req, _ := http.NewRequest(http.MethodPost, "/api/user/topup/complete", bytes.NewReader(reqPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &respBody))
		assert.Equal(t, true, respBody["success"])

		// Verify order is now SUCCESS
		var updatedTopUp model.TopUp
		require.NoError(t, db.Where("trade_no = ?", tradeNo).First(&updatedTopUp).Error)
		assert.Equal(t, common.TopUpStatusSuccess, updatedTopUp.Status)

		// Verify target user's quota increased by 10 * QuotaPerUnit
		var updatedUser model.User
		require.NoError(t, db.First(&updatedUser, targetUser.Id).Error)
		expectedQuota := targetUser.Quota + int(10*common.QuotaPerUnit)
		assert.Equal(t, expectedQuota, updatedUser.Quota)

		// 4. Idempotent re-execution does not double credit
		resp2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(http.MethodPost, "/api/user/topup/complete", bytes.NewReader(reqPayload))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+adminToken)
		router.ServeHTTP(resp2, req2)
		assert.Equal(t, http.StatusOK, resp2.Code)

		var updatedUser2 model.User
		require.NoError(t, db.First(&updatedUser2, targetUser.Id).Error)
		assert.Equal(t, expectedQuota, updatedUser2.Quota, "Idempotent completion must not double credit")
	}
}
