package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupFreeTierMiddlewareTestDB(t *testing.T) *model.User {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSubscription{}, &model.FreeTierDailyUsage{}))
	model.DB = db
	model.LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
	})

	user := &model.User{
		Username:    "free-tier-middleware-" + common.GetRandomString(8),
		Password:    "unused-password",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
		AffCode:     "free-tier-aff-" + common.GetRandomString(8),
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func TestFreeTierQuotaRejectsTwentyFirstRequest(t *testing.T) {
	user := setupFreeTierMiddlewareTestDB(t)
	router := gin.New()
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		FreeTierQuota()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusNoContent)
	})

	for requestNumber := 1; requestNumber <= int(model.FreeTierDailyLimit)+1; requestNumber++ {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		router.ServeHTTP(response, request)
		if requestNumber <= int(model.FreeTierDailyLimit) {
			assert.Equal(t, http.StatusNoContent, response.Code, requestNumber)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, response.Code)
			assert.Contains(t, response.Body.String(), "FREE_DAILY_LIMIT")
		}
	}
}

func TestFreeTierQuotaEnforcedOnResponsesEndpoint(t *testing.T) {
	user := setupFreeTierMiddlewareTestDB(t)
	router := gin.New()
	router.POST("/v1/responses", func(c *gin.Context) {
		c.Set("id", user.Id)
		FreeTierQuota()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusNoContent)
	})

	for requestNumber := 1; requestNumber <= int(model.FreeTierDailyLimit)+1; requestNumber++ {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		router.ServeHTTP(response, request)
		if requestNumber <= int(model.FreeTierDailyLimit) {
			assert.Equal(t, http.StatusNoContent, response.Code, requestNumber)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, response.Code)
			assert.Contains(t, response.Body.String(), "FREE_DAILY_LIMIT")
		}
	}
}

func TestFreeTierQuotaSharedBetweenChatCompletionsAndResponses(t *testing.T) {
	user := setupFreeTierMiddlewareTestDB(t)
	router := gin.New()
	handler := func(c *gin.Context) {
		c.Set("id", user.Id)
		FreeTierQuota()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusNoContent)
	}
	router.POST("/v1/chat/completions", handler)
	router.POST("/v1/responses", handler)

	// 10 requests to chat/completions
	for i := 1; i <= 10; i++ {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
		assert.Equal(t, http.StatusNoContent, response.Code)
	}

	// 10 requests to responses
	for i := 11; i <= 20; i++ {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/responses", nil))
		assert.Equal(t, http.StatusNoContent, response.Code)
	}

	// 21st request to chat/completions should be rejected
	chat21 := httptest.NewRecorder()
	router.ServeHTTP(chat21, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
	assert.Equal(t, http.StatusTooManyRequests, chat21.Code)
	assert.Contains(t, chat21.Body.String(), "FREE_DAILY_LIMIT")

	// 22nd request to responses should also be rejected
	resp22 := httptest.NewRecorder()
	router.ServeHTTP(resp22, httptest.NewRequest(http.MethodPost, "/v1/responses", nil))
	assert.Equal(t, http.StatusTooManyRequests, resp22.Code)
	assert.Contains(t, resp22.Body.String(), "FREE_DAILY_LIMIT")
}

func TestFreeTierQuotaBypassesPaidAndAdminUsers(t *testing.T) {
	user := setupFreeTierMiddlewareTestDB(t)
	paid := &model.User{
		Username:    "paid-tier-middleware-" + common.GetRandomString(8),
		Password:    "unused-password",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
		AffCode:     "paid-tier-aff-" + common.GetRandomString(8),
	}
	require.NoError(t, model.DB.Create(paid).Error)
	require.NoError(t, model.DB.Create(&model.UserSubscription{
		UserId: paid.Id, AmountTotal: 0, Status: "active", EndTime: time.Now().Unix() + 3600,
	}).Error)

	// Paid user bypass
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("id", paid.Id)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	FreeTierQuota()(context)
	assert.NotEqual(t, http.StatusTooManyRequests, response.Code)

	// Admin user bypass
	require.NoError(t, model.DB.Model(user).Update("role", common.RoleAdminUser).Error)
	response = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(response)
	context.Set("id", user.Id)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	FreeTierQuota()(context)
	assert.NotEqual(t, http.StatusTooManyRequests, response.Code)

	// Root user bypass
	require.NoError(t, model.DB.Model(user).Update("role", common.RoleRootUser).Error)
	response = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(response)
	context.Set("id", user.Id)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	FreeTierQuota()(context)
	assert.NotEqual(t, http.StatusTooManyRequests, response.Code)
}

func TestFreeTierQuotaDoesNotApplyToModelListingRoute(t *testing.T) {
	user := setupFreeTierMiddlewareTestDB(t)
	router := gin.New()
	router.GET("/v1/models", func(c *gin.Context) {
		c.Set("id", user.Id)
		c.JSON(http.StatusOK, gin.H{"data": []string{"model"}})
	})
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", user.Id)
		FreeTierQuota()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusNoContent)
	})

	modelsResponse := httptest.NewRecorder()
	router.ServeHTTP(modelsResponse, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	assert.Equal(t, http.StatusOK, modelsResponse.Code)

	chatResponse := httptest.NewRecorder()
	router.ServeHTTP(chatResponse, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
	assert.Equal(t, http.StatusNoContent, chatResponse.Code)
}

func TestFreeTierQuotaExcludesOutOfScopeProtocols(t *testing.T) {
	user := setupFreeTierMiddlewareTestDB(t)
	router := gin.New()
	router.POST("/v1/embeddings", func(c *gin.Context) {
		c.Set("id", user.Id)
		FreeTierQuota()(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	// Even if user exceeded 20 requests on free tier, /v1/embeddings is out of scope and not restricted by FreeTierQuota
	day := time.Now()
	for i := 0; i < int(model.FreeTierDailyLimit); i++ {
		_, err := model.ReserveFreeTierRequest(user.Id, day)
		require.NoError(t, err)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil))
	assert.Equal(t, http.StatusOK, response.Code)
}
