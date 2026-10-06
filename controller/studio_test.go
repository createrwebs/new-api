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
