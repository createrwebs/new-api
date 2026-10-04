package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMJImageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousType := common.MainDatabaseType()
	previousCache := common.MemoryCacheEnabled
	previousRedis := common.RedisEnabled
	common.RedisEnabled = false
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&model.Midjourney{}, &model.User{}, &model.Token{}, &model.Channel{}, &model.UserAccessToken{}))
	model.DB = database
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.MemoryCacheEnabled = previousCache
		common.RedisEnabled = previousRedis
		require.NoError(t, sqlDB.Close())
	})
	return database
}

func TestRelayMidjourneyImageAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupMJImageTestDB(t)

	origSecret := common.CryptoSecret
	common.CryptoSecret = "mj-auth-test-secret-key-12345"
	defer func() { common.CryptoSecret = origSecret }()

	// Allow mock upstream in test environment
	originalFetchSetting := *system_setting.GetFetchSetting()
	system_setting.GetFetchSetting().EnableSSRFProtection = false
	system_setting.GetFetchSetting().AllowPrivateIp = true
	system_setting.GetFetchSetting().AllowedPorts = []string{"1-65535"}
	t.Cleanup(func() { *system_setting.GetFetchSetting() = originalFetchSetting })

	// Start a mock upstream server returning a dummy image
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("fake-image-bytes"))
	}))
	defer mockUpstream.Close()

	chanSetting := fmt.Sprintf(`{"proxy":"%s"}`, mockUpstream.URL)
	channel := &model.Channel{
		Type:    constant.ChannelTypeMidjourney,
		Status:  common.ChannelStatusEnabled,
		Setting: &chanSetting,
	}
	require.NoError(t, db.Create(channel).Error)

	// Create User A, User B, Admin
	userA := &model.User{Username: "userA", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff-a"}
	require.NoError(t, db.Create(userA).Error)

	userB := &model.User{Username: "userB", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "aff-b"}
	require.NoError(t, db.Create(userB).Error)

	admin := &model.User{Username: "admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, AffCode: "aff-admin"}
	require.NoError(t, db.Create(admin).Error)

	// Create Midjourney Task owned by User A
	taskA := &model.Midjourney{
		UserId:    userA.Id,
		ChannelId: channel.Id,
		MjId:      "mj-task-user-a-123",
		ImageUrl:  mockUpstream.URL + "/image.png",
		Status:    "SUCCESS",
	}
	require.NoError(t, db.Create(taskA).Error)

	// Helper to create router with optional authenticated user context
	createRouter := func(userId int, userRole int) *gin.Engine {
		r := gin.New()
		r.GET("/mj/image/:id", func(c *gin.Context) {
			if userId > 0 {
				c.Set("id", userId)
				c.Set("role", userRole)
			}
			RelayMidjourneyImage(c)
		})
		return r
	}

	// 1. User A + valid capability of User A -> PASS (200, fake-image-bytes)
	{
		validTokenA, err := service.IssueMJImageAccess("mj-task-user-a-123", userA.Id, time.Hour)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", validTokenA), nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "fake-image-bytes", resp.Body.String())
	}

	// 2. User B + valid capability of User A -> DENY (403, invalid_or_expired_access_token)
	{
		validTokenA, err := service.IssueMJImageAccess("mj-task-user-a-123", userA.Id, time.Hour)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", validTokenA), nil)
		resp := httptest.NewRecorder()
		routerB := createRouter(userB.Id, userB.Role)
		routerB.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_or_expired_access_token")
	}

	// 3. User A + capability with modified owner_user_id -> DENY (403, signature mismatch)
	{
		validTokenA, err := service.IssueMJImageAccess("mj-task-user-a-123", userA.Id, time.Hour)
		require.NoError(t, err)

		parts := strings.Split(validTokenA, ".")
		require.Len(t, parts, 4)
		// Alter owner_user_id to userB's id while keeping signature
		tamperedToken := fmt.Sprintf("%s.%s.%d.%s", parts[0], parts[1], userB.Id, parts[3])

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", tamperedToken), nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_or_expired_access_token")
	}

	// 4. User A + capability of another mj_id -> DENY (403, resource mismatch)
	{
		tokenForOther, err := service.IssueMJImageAccess("mj-task-other-456", userA.Id, time.Hour)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", tokenForOther), nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_or_expired_access_token")
	}

	// 5. Expired capability -> DENY (403, expired token)
	{
		expiredToken, err := service.IssueMJImageAccess("mj-task-user-a-123", userA.Id, 10*time.Millisecond)
		require.NoError(t, err)
		time.Sleep(1100 * time.Millisecond) // wait past 1-second unix timestamp resolution

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", expiredToken), nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_or_expired_access_token")
	}

	// 6. Modified signature -> DENY (403, tampered signature)
	{
		validTokenA, err := service.IssueMJImageAccess("mj-task-user-a-123", userA.Id, time.Hour)
		require.NoError(t, err)

		parts := strings.Split(validTokenA, ".")
		require.Len(t, parts, 4)
		tamperedSigToken := fmt.Sprintf("%s.%s.%s.badSignature999", parts[0], parts[1], parts[2])

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", tamperedSigToken), nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_or_expired_access_token")
	}

	// 7. Wrong purpose -> DENY (403, purpose isolation)
	{
		wrongPurposeToken, err := service.IssueMJImageAccessWithPurpose("avatar", "mj-task-user-a-123", userA.Id, time.Hour)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", wrongPurposeToken), nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_or_expired_access_token")
	}

	// 8. Unknown mj_id -> DENY (400, midjourney_task_not_found)
	{
		tokenUnknown, err := service.IssueMJImageAccess("mj-task-nonexistent-999", userA.Id, time.Hour)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-nonexistent-999?access=%s", tokenUnknown), nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "midjourney_task_not_found")
	}

	// 9. No capability + owner session -> PASS (200, fake-image-bytes)
	{
		req, _ := http.NewRequest(http.MethodGet, "/mj/image/mj-task-user-a-123", nil)
		resp := httptest.NewRecorder()
		routerA := createRouter(userA.Id, userA.Role)
		routerA.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "fake-image-bytes", resp.Body.String())
	}

	// 10. No capability + non-owner session -> DENY (403, permission_denied)
	{
		req, _ := http.NewRequest(http.MethodGet, "/mj/image/mj-task-user-a-123", nil)
		resp := httptest.NewRecorder()
		routerB := createRouter(userB.Id, userB.Role)
		routerB.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "permission_denied")
	}

	// 11. Admin behavior:
	// 11a. Admin with capability of User A -> PASS (200, fake-image-bytes)
	{
		validTokenA, err := service.IssueMJImageAccess("mj-task-user-a-123", userA.Id, time.Hour)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", validTokenA), nil)
		resp := httptest.NewRecorder()
		routerAdmin := createRouter(admin.Id, admin.Role)
		routerAdmin.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "fake-image-bytes", resp.Body.String())
	}

	// 11b. Admin without capability -> PASS (200, fake-image-bytes)
	{
		req, _ := http.NewRequest(http.MethodGet, "/mj/image/mj-task-user-a-123", nil)
		resp := httptest.NewRecorder()
		routerAdmin := createRouter(admin.Id, admin.Role)
		routerAdmin.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "fake-image-bytes", resp.Body.String())
	}

	// 12. Unauthenticated request without capability -> 401 (authentication_required)
	{
		req, _ := http.NewRequest(http.MethodGet, "/mj/image/mj-task-user-a-123", nil)
		resp := httptest.NewRecorder()
		routerUnauth := createRouter(0, 0)
		routerUnauth.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "authentication_required")
	}

	// 13. Unauthenticated request with capability of User A -> 403 (caller identity check fails)
	{
		validTokenA, err := service.IssueMJImageAccess("mj-task-user-a-123", userA.Id, time.Hour)
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/mj/image/mj-task-user-a-123?access=%s", validTokenA), nil)
		resp := httptest.NewRecorder()
		routerUnauth := createRouter(0, 0)
		routerUnauth.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_or_expired_access_token")
	}

	// 14. Bearer API Relay token of User A -> 200 (fake-image-bytes)
	{
		tokenA := &model.Token{
			UserId:         userA.Id,
			Name:           "token-a",
			Key:            "tokenkeyusera",
			Status:         common.TokenStatusEnabled,
			UnlimitedQuota: true,
		}
		require.NoError(t, db.Create(tokenA).Error)

		req, _ := http.NewRequest(http.MethodGet, "/mj/image/mj-task-user-a-123", nil)
		req.Header.Set("Authorization", "Bearer sk-tokenkeyusera")
		resp := httptest.NewRecorder()
		routerUnauth := createRouter(0, 0)
		routerUnauth.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "fake-image-bytes", resp.Body.String())
	}

	// 15. Bearer API Relay token of User B on User A's task -> 403 (permission_denied)
	{
		tokenB := &model.Token{
			UserId:         userB.Id,
			Name:           "token-b",
			Key:            "tokenkeyuserb",
			Status:         common.TokenStatusEnabled,
			UnlimitedQuota: true,
		}
		require.NoError(t, db.Create(tokenB).Error)

		req, _ := http.NewRequest(http.MethodGet, "/mj/image/mj-task-user-a-123", nil)
		req.Header.Set("Authorization", "Bearer sk-tokenkeyuserb")
		resp := httptest.NewRecorder()
		routerUnauth := createRouter(0, 0)
		routerUnauth.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "permission_denied")
	}
}
