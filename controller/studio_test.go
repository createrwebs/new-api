package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func setupTestRouterForStudio(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	dsn := fmt.Sprintf("file:test_controller_studio_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	oldDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
		_ = sqlDB.Close()
	})

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.WalletPreConsumeRecord{},
	))
	require.NoError(t, model.EnsureStudioTables(db))
	require.NoError(t, service.SeedStudioCatalog(db))

	// Init Studio Singleton
	mockProvider := service.NewDeterministicMockProvider(service.MockModeInstantSuccess)
	service.GlobalStudioService = service.NewStudioService(mockProvider)

	r.GET("/api/studio/tools", GetStudioTools)
	r.POST("/api/studio/quote", QuoteStudioJob)

	// Authenticated mock route
	authGroup := r.Group("/api/studio")
	authGroup.Use(func(c *gin.Context) {
		c.Set("id", 1)
		c.Set("role", common.RoleCommonUser)
		c.Next()
	})
	authGroup.POST("/jobs", CreateStudioJob)

	return r, db
}

func TestController_QuoteStudioJob(t *testing.T) {
	r, _ := setupTestRouterForStudio(t)

	// 1. Valid Quote for image-generate
	payload := map[string]interface{}{
		"tool_id": "image-generate",
		"input_params": map[string]interface{}{
			"prompt":      "Cinematic cyberpunk Bangkok",
			"num_outputs": 2,
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/studio/quote", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			QuoteId        string  `json:"quote_id"`
			ToolId         string  `json:"tool_id"`
			ChargedCredits int     `json:"charged_credits"`
			ChargedQuota   int     `json:"charged_quota"`
			TargetMargin   float64 `json:"target_margin"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Data.QuoteId)
	assert.Equal(t, "image-generate", resp.Data.ToolId)
	assert.Equal(t, 10, resp.Data.ChargedCredits, "2 outputs should charge 10 credits (5 * 2)")
	assert.Equal(t, 10000, resp.Data.ChargedQuota)
	assert.True(t, resp.Data.TargetMargin >= 60.0)
}

func TestController_CreateStudioJob_RejectsBase64Video(t *testing.T) {
	r, db := setupTestRouterForStudio(t)

	user := model.User{
		Username: "video_uploader_test",
		Quota:    200000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	// Attempt submitting raw base64 video
	payload := map[string]interface{}{
		"tool_id": "image-to-video",
		"input_params": map[string]interface{}{
			"image_url": "data:video/mp4;base64,AAAAHGZ0eXBtcDQyAAAAAG1wNDJpc29t...",
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/studio/jobs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "video uploads via raw base64 are disabled")
}

func TestController_CreateStudioJob_RejectsSSRFInParams(t *testing.T) {
	r, db := setupTestRouterForStudio(t)

	user := model.User{
		Username: "ssrf_tester",
		Quota:    200000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	// Attempt submitting internal IP in image_url
	payload := map[string]interface{}{
		"tool_id": "image-upscale",
		"input_params": map[string]interface{}{
			"image_url": "http://169.254.169.254/latest/meta-data",
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/studio/jobs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "SSRF protection blocked request")
}

func TestController_StudioWebhook_ProcessesCallback(t *testing.T) {
	r, db := setupTestRouterForStudio(t)
	r.POST("/api/studio/webhook/:provider", StudioWebhook)

	user := model.User{
		Username: "webhook_ctl_user",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	// Pre-create a job in PROCESSING state
	job := model.StudioToolJob{
		Id:             "job_ctl_webhook_1",
		UserId:         user.Id,
		ToolId:         "image-generate",
		RequestId:      "req_ctl_webhook_1",
		IdempotencyKey: "idemp_ctl_webhook_1",
		ProviderName:   "fal",
		ProviderJobId:  "fal_job_req_9988",
		Status:         model.StudioJobStatusProcessing,
		ReservedQuota:  5000,
		SettledQuota:   0,
		CreatedAt:      common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&job).Error)

	// Send webhook
	webhookPayload := map[string]interface{}{
		"request_id": "fal_job_req_9988",
		"status":     "completed",
		"output_url": "https://cdn.toraapi.com/fal_out_1.png",
	}
	body, _ := json.Marshal(webhookPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/studio/webhook/fal", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)

	// Check DB
	var updated model.StudioToolJob
	require.NoError(t, db.Where("id = ?", job.Id).First(&updated).Error)
	assert.Equal(t, model.StudioJobStatusSucceeded, updated.Status)
	assert.Equal(t, 5000, updated.SettledQuota)
}

func TestController_TriggerStudioProviderCanary(t *testing.T) {
	r, db := setupTestRouterForStudio(t)

	// Register admin route
	adminGroup := r.Group("/api/admin/studio")
	adminGroup.Use(func(c *gin.Context) {
		roleHeader := c.GetHeader("X-Role")
		if roleHeader == "admin" {
			c.Set("id", 99)
			c.Set("role", common.RoleAdminUser)
		} else {
			c.Set("id", 1)
			c.Set("role", common.RoleCommonUser)
		}
		c.Next()
	})
	adminGroup.POST("/provider-canary", TriggerStudioProviderCanary)

	// Create test admin user with wallet balance
	adminUser := model.User{
		Id:       99,
		Username: "canary_admin",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&adminUser).Error)

	// Test 1: Non-admin rejected
	payloadNonAdmin := map[string]interface{}{
		"provider":            "fal",
		"tool_id":             "background-remove",
		"confirm_live_charge": true,
		"max_spend_usd":       0.02,
		"idempotency_key":     "idemp_canary_ctrl_001",
	}
	body, _ := json.Marshal(payloadNonAdmin)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/studio/provider-canary", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// Test 2: Invalid provider rejected
	payloadBadProv := map[string]interface{}{
		"provider":            "unauthorized_provider",
		"tool_id":             "background-remove",
		"confirm_live_charge": true,
		"max_spend_usd":       0.02,
		"idempotency_key":     "idemp_canary_ctrl_002",
	}
	body, _ = json.Marshal(payloadBadProv)
	req = httptest.NewRequest(http.MethodPost, "/api/admin/studio/provider-canary", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Role", "admin")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "only 'fal' is permitted")

	// Test 3: Missing confirm_live_charge rejected
	payloadNoConfirm := map[string]interface{}{
		"provider":            "fal",
		"tool_id":             "background-remove",
		"confirm_live_charge": false,
		"max_spend_usd":       0.02,
		"idempotency_key":     "idemp_canary_ctrl_003",
	}
	body, _ = json.Marshal(payloadNoConfirm)
	req = httptest.NewRequest(http.MethodPost, "/api/admin/studio/provider-canary", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Role", "admin")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "confirm_live_charge must be true")

	// Test 4: Exceeds hard ceiling ($0.05) rejected
	payloadOverCeiling := map[string]interface{}{
		"provider":            "fal",
		"tool_id":             "background-remove",
		"confirm_live_charge": true,
		"max_spend_usd":       0.50, // exceeds $0.05
		"idempotency_key":     "idemp_canary_ctrl_004",
	}
	body, _ = json.Marshal(payloadOverCeiling)
	req = httptest.NewRequest(http.MethodPost, "/api/admin/studio/provider-canary", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Role", "admin")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "ceiling")

	// Test 5: Missing FAL_KEY returns PreconditionFailed (OPERATOR_BLOCKED)
	payloadValid := map[string]interface{}{
		"provider":            "fal",
		"tool_id":             "background-remove",
		"confirm_live_charge": true,
		"max_spend_usd":       0.02,
		"idempotency_key":     "idemp_canary_ctrl_005",
	}
	body, _ = json.Marshal(payloadValid)
	req = httptest.NewRequest(http.MethodPost, "/api/admin/studio/provider-canary", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Role", "admin")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusPreconditionFailed, w.Code)
	assert.Contains(t, w.Body.String(), "FAL_NOT_CONFIGURED")
}
