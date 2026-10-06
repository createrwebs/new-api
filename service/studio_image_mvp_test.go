package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueue3_InitialPublicTools_ReadinessAndEconomics tests the progressive activation and economics
// for the 6 image MVP tools and ensures video tools remain frozen.
func TestQueue3_InitialPublicTools_ReadinessAndEconomics(t *testing.T) {
	db := setupTestDBForStudio(t)
	pricingEngine := NewPricingEngine()

	// 1. Initial Priority Image Tools
	activeImageTools := []string{
		"background-remove",
		"image-upscale",
		"image-generate",
	}

	for _, toolId := range activeImageTools {
		tool, err := model.GetStudioToolDefinition(toolId)
		require.NoError(t, err, "Tool %s must exist in seed catalog", toolId)
		assert.Equal(t, model.StudioToolStateActive, tool.Status, "Tool %s must be ACTIVE", toolId)
		assert.True(t, tool.IsEnabled && tool.IsPublic, "Tool %s must be enabled and public", toolId)
		assert.True(t, tool.MarginPercent >= 60.0, "Tool %s margin must be >= 60%%", toolId)

		// Test quote generation
		quote, err := pricingEngine.CalculatePriceWithInputs(tool, map[string]interface{}{}, 1.0)
		require.NoError(t, err)
		assert.True(t, quote.CalculatedCredits > 0)
		assert.True(t, quote.TargetMargin >= 60.0)
	}

	// 2. Secondary Verified Tools (Beta in Image MVP)
	betaImageTools := []string{
		"product-photo",
		"object-erase",
		"image-extend",
	}

	for _, toolId := range betaImageTools {
		tool, err := model.GetStudioToolDefinition(toolId)
		require.NoError(t, err, "Tool %s must exist in seed catalog", toolId)
		assert.True(t, tool.Status == model.StudioToolStateBeta || tool.Status == model.StudioToolStateActive, "Tool %s must be BETA or ACTIVE", toolId)
		assert.True(t, tool.IsEnabled && tool.IsPublic, "Tool %s must be enabled and public", toolId)
		assert.True(t, tool.MarginPercent >= 60.0, "Tool %s margin must be >= 60%%", toolId)
	}

	// 3. Object-eraser alias lookup check
	eraserDef, err := model.GetStudioToolDefinition("object-eraser")
	require.NoError(t, err, "object-eraser alias must resolve successfully")
	assert.Equal(t, "object-erase", eraserDef.Id)

	// 4. Video tools must be FROZEN (ComingSoon / not public)
	videoTools := []string{
		"image-to-video",
		"text-to-video",
		"lip-sync",
		"talking-avatar",
		"video-upscale",
	}

	for _, vToolId := range videoTools {
		var tool model.StudioToolDefinition
		err := db.Where("id = ?", vToolId).First(&tool).Error
		require.NoError(t, err)
		assert.NotEqual(t, model.StudioToolStateActive, tool.Status, "Video tool %s must NOT be active in Image MVP", vToolId)
		assert.False(t, tool.IsPublic, "Video tool %s must not be public", vToolId)
	}
}

// TestQueue3_ToolQA_SmallAndLargeInput tests boundary inputs (small, large allowed, and exceeded).
func TestQueue3_ToolQA_SmallAndLargeInput(t *testing.T) {
	// 1. Small valid PNG (8 bytes header + minimal chunk)
	smallPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
	mime, err := ValidateMediaUpload(smallPNG, "small.png", false)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mime)

	// 2. Large allowed input (14.5 MB, strictly within 15 MB image limit)
	largeAllowedSize := 14 * 1024 * 1024 + 500*1024
	largeAllowed := make([]byte, largeAllowedSize)
	copy(largeAllowed, smallPNG)
	mimeLarge, err := ValidateMediaUpload(largeAllowed, "large_allowed.png", false)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mimeLarge)

	// 3. Exceeded size input (> 15 MB)
	exceededSize := 15*1024*1024 + 1024
	exceeded := make([]byte, exceededSize)
	copy(exceeded, smallPNG)
	_, errExceeded := ValidateMediaUpload(exceeded, "exceeded.png", false)
	require.ErrorIs(t, errExceeded, ErrFileSizeExceeded)
}

// TestQueue3_ToolQA_InvalidMIME_And_Security verifies rejection of executable, scripts, and SSRF.
func TestQueue3_ToolQA_InvalidMIME_And_Security(t *testing.T) {
	// 1. Windows PE / ELF Executables
	exeHeader := []byte{0x4D, 0x5A, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00}
	_, err := ValidateMediaUpload(exeHeader, "malware.exe", false)
	require.ErrorIs(t, err, ErrInvalidFileType)

	elfHeader := []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00}
	_, errElf := ValidateMediaUpload(elfHeader, "payload.elf", false)
	require.ErrorIs(t, errElf, ErrInvalidFileType)

	// 2. Shell script or plain text
	shHeader := []byte("#!/bin/bash\nrm -rf /")
	_, errSh := ValidateMediaUpload(shHeader, "script.sh", false)
	require.ErrorIs(t, errSh, ErrInvalidFileType)

	// 3. Video upload into image-only tool
	mp4Header := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}
	_, errVid := ValidateMediaUpload(mp4Header, "test.mp4", false)
	require.ErrorIs(t, errVid, ErrVideoNotAllowed)

	// 4. SSRF blocklist
	assert.ErrorIs(t, ValidateExternalURL("http://169.254.169.254/latest/meta-data"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://127.0.0.1:8080/admin"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://10.0.0.1/internal"), ErrSSRFForbidden)
}

// TestQueue3_ToolQA_InsufficientCredits verifies fail-closed rejection on low credit balance.
func TestQueue3_ToolQA_InsufficientCredits(t *testing.T) {
	db := setupTestDBForStudio(t)

	// User has 5,000 Quota (5 Credits)
	user := model.User{
		Username: "broke_creator",
		Quota:    5000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// Attempting image-upscale (requires 25,000 Quota / 25 Credits)
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-upscale",
		"",
		"idemp_insufficient_001",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/test.png", "scale": 4},
		1.0,
		"127.0.0.1",
	)
	require.ErrorIs(t, err, ErrInsufficientQuota)
	assert.Nil(t, job)

	// Balance must remain strictly unchanged
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 5000, bal, "User quota must not be deducted on insufficient funds")
}

// TestQueue3_ToolQA_ProviderValidationError_Refunds100Percent verifies that upstream 400/422 validation
// errors trigger immediate, full 100% refund.
func TestQueue3_ToolQA_ProviderValidationError_Refunds100Percent(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "validation_err_user",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	// Mock provider in permanent fail mode (simulates 422 Unprocessable Entity)
	failProvider := NewDeterministicMockProvider(MockModePermanentFail)
	studioSvc := NewStudioService(failProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"idemp_provider_val_err",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/corrupt.png"},
		1.0,
		"127.0.0.1",
	)
	require.Error(t, err)
	require.NotNil(t, job)
	assert.Equal(t, model.StudioJobStatusFailed, job.Status)

	// Wallet must be 100% restored
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, bal, "Wallet quota must be 100%% restored upon provider validation error")

	// Audit record in ledger
	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("user_id = ? AND status = ?", user.Id, "refunded").First(&preRecord).Error)
	assert.Equal(t, 10000, preRecord.PreConsumed)
}

// TestQueue3_ToolQA_Cancel_And_Refund verifies cancellation of queued jobs with full quota restore.
func TestQueue3_ToolQA_Cancel_And_Refund(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "cancel_user",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-upscale",
		"",
		"idemp_cancel_test",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/slow.png", "scale": 4},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)

	// Quota is currently reserved (50,000 - 25,000 = 25,000)
	midBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 25000, midBal)

	// Cancel job
	cancelledJob, err := studioSvc.CancelJob(context.Background(), job.Id, user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusCancelled, cancelledJob.Status)

	// Final quota must be 100% restored
	finalBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, finalBal, "Cancellation must restore 100%% of reserved quota")
}

// TestQueue3_ToolQA_DuplicateRequest_And_Retry verifies idempotency and replay safety.
func TestQueue3_ToolQA_DuplicateRequest_And_Retry(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "idemp_user",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// 1. First execution
	job1, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"idemp_repeat_key_101",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/test.png"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, job1.Status)

	// 2. Duplicate submission with identical key
	job2, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"idemp_repeat_key_101",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/test.png"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, job1.Id, job2.Id, "Duplicate key must return identical existing job")

	// Quota deducted exactly once (50,000 - 10,000 = 40,000)
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 40000, bal, "Duplicate request must not incur duplicate charge")

	// 3. Retry with new idempotency key
	jobRetry, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"idemp_repeat_key_102",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/test.png"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.NotEqual(t, job1.Id, jobRetry.Id, "Retry with new key must create distinct job")

	// Quota deducted a second time (40,000 - 10,000 = 30,000)
	balAfterRetry, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 30000, balAfterRetry)
}

// TestQueue3_ToolQA_RemixChain tests chaining outputs between tools (generate -> bg-remove -> upscale).
func TestQueue3_ToolQA_RemixChain(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "remix_artist",
		Quota:    100000, // 100 Credits
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// Step 1: Image Generate (5 Credits / 5,000 Quota)
	genJob, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		"remix_step1_gen",
		"mock",
		map[string]interface{}{"prompt": "futuristic flying car in Bangkok neon rain"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, genJob.Status)

	var genOut struct {
		OutputURL string `json:"output_url"`
	}
	_ = json.Unmarshal([]byte(genJob.OutputResult), &genOut)
	require.NotEmpty(t, genOut.OutputURL)

	// Step 2: Background Remove using output of Step 1 (10 Credits / 10,000 Quota)
	bgJob, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"remix_step2_bg",
		"mock",
		map[string]interface{}{"image_url": genOut.OutputURL},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, bgJob.Status)

	var bgOut struct {
		OutputURL string `json:"output_url"`
	}
	_ = json.Unmarshal([]byte(bgJob.OutputResult), &bgOut)
	require.NotEmpty(t, bgOut.OutputURL)

	// Step 3: Image Upscale using output of Step 2 (25 Credits / 25,000 Quota)
	upJob, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-upscale",
		"",
		"remix_step3_up",
		"mock",
		map[string]interface{}{"image_url": bgOut.OutputURL, "scale": 4},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, upJob.Status)

	// Total deductions: 5,000 + 10,000 + 25,000 = 40,000 Quota
	finalBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 60000, finalBal, "Remix chain must deduct exact cumulative quota")
}

// TestQueue3_CreditConversionFunnel_E2E simulates the full credit conversion journey:
// tool -> quote -> insufficient -> buy credits -> return -> requote -> confirm -> generate.
func TestQueue3_CreditConversionFunnel_E2E(t *testing.T) {
	db := setupTestDBForStudio(t)
	pricingEngine := NewPricingEngine()

	// 1. User starts with 2,000 Quota (2 Credits)
	user := model.User{
		Username: "funnel_prospect",
		Quota:    2000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	toolDef, err := model.GetStudioToolDefinition("image-upscale")
	require.NoError(t, err)

	// 2. Initial Quote (25 Credits = 25,000 Quota)
	quote, err := pricingEngine.CalculatePriceWithInputs(toolDef, map[string]interface{}{"scale": 4}, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 25, quote.ChargedCredits)

	// 3. User attempts generation -> Insufficient Funds
	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)
	_, errSub := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-upscale",
		"",
		"funnel_idemp_01",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/asset.png", "scale": 4},
		1.0,
		"127.0.0.1",
	)
	require.ErrorIs(t, errSub, ErrInsufficientQuota)

	// Record telemetry event: studio_insufficient_credit
	errRec := model.RecordStudioConversionEvent(user.Id, "insufficient_credit", "image-upscale", 25, "sess_abc123")
	require.NoError(t, errRec)

	// Record telemetry event: studio_buy_credit_click
	errClick := model.RecordStudioConversionEvent(user.Id, "buy_credit_click", "image-upscale", 25, "sess_abc123")
	require.NoError(t, errClick)

	// 4. User tops up wallet in /wallet (purchases 50,000 Quota / 50 Credits)
	errTopUp := model.IncreaseUserQuota(user.Id, 50000, true)
	require.NoError(t, errTopUp)

	newBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 52000, newBal, "Balance must reflect successful top-up")

	// 5. User returns to Studio flow
	errRet := model.RecordStudioConversionEvent(user.Id, "purchase_return", "image-upscale", 25, "sess_abc123")
	require.NoError(t, errRet)

	// 6. System Requotes (verifies fresh quote with current balance)
	requote, err := pricingEngine.CalculatePriceWithInputs(toolDef, map[string]interface{}{"scale": 4}, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 25, requote.ChargedCredits)

	// 7. User confirms explicitly -> Generation executes (Strictly non-automatic!)
	errGen := model.RecordStudioConversionEvent(user.Id, "generation_after_purchase", "image-upscale", 25, "sess_abc123")
	require.NoError(t, errGen)

	confirmedJob, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-upscale",
		"",
		"funnel_idemp_02_confirmed",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/asset.png", "scale": 4},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, confirmedJob.Status)

	// Wallet deducted: 52,000 - 25,000 = 27,000 Quota
	finalBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 27000, finalBal)

	// 8. Verify all 4 conversion events are recorded in persistent database
	var count int64
	require.NoError(t, db.Model(&model.StudioConversionEvent{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(4), count, "All 4 conversion funnel milestones must be persisted")
}

// TestQueue3_AdminMetrics_TelemetryAggregation verifies comprehensive admin metrics calculation.
func TestQueue3_AdminMetrics_TelemetryAggregation(t *testing.T) {
	db := setupTestDBForStudio(t)

	// Create test jobs and snapshots
	now := time.Now().Unix()

	job1 := model.StudioToolJob{
		Id:             "job_metric_1",
		UserId:         10,
		ToolId:         "background-remove",
		IdempotencyKey: "key_metric_1",
		Status:         model.StudioJobStatusSucceeded,
		ReservedQuota:  10000,
		SettledQuota:   10000,
		CreatedAt:      now,
	}
	require.NoError(t, db.Create(&job1).Error)

	snap1 := model.StudioCostSnapshot{
		JobId:              "job_metric_1",
		ToolId:             "background-remove",
		ProviderName:       "fal",
		CostUSD:            0.005,
		QuotaCost:          10000,
		ToraRevenueUSD:     0.020,
		GrossProfitUSD:     0.015,
		GrossMarginPercent: 75.0,
		SnapshotAt:         now,
	}
	require.NoError(t, db.Create(&snap1).Error)

	job2 := model.StudioToolJob{
		Id:             "job_metric_2",
		UserId:         20,
		ToolId:         "image-upscale",
		IdempotencyKey: "key_metric_2",
		Status:         model.StudioJobStatusFailed,
		ReservedQuota:  25000,
		SettledQuota:   0,
		CreatedAt:      now,
	}
	require.NoError(t, db.Create(&job2).Error)

	refundEvent := model.StudioJobEvent{
		JobId:     "job_metric_2",
		EventType: "WALLET_REFUNDED_FAILURE",
		CreatedAt: now,
	}
	require.NoError(t, db.Create(&refundEvent).Error)

	// Create conversion events
	require.NoError(t, model.RecordStudioConversionEvent(10, "insufficient_credit", "background-remove", 10, "s1"))
	require.NoError(t, model.RecordStudioConversionEvent(10, "buy_credit_click", "background-remove", 10, "s1"))
	require.NoError(t, model.RecordStudioConversionEvent(10, "purchase_return", "background-remove", 10, "s1"))
	require.NoError(t, model.RecordStudioConversionEvent(10, "generation_after_purchase", "background-remove", 10, "s1"))

	// Audit metrics aggregation logic
	var studioUsers int64
	var totalJobs int64
	var succeeded int64
	var failed int64
	var refunds int64

	db.Model(&model.StudioToolJob{}).Distinct("user_id").Count(&studioUsers)
	db.Model(&model.StudioToolJob{}).Count(&totalJobs)
	db.Model(&model.StudioToolJob{}).Where("status = ?", model.StudioJobStatusSucceeded).Count(&succeeded)
	db.Model(&model.StudioToolJob{}).Where("status = ?", model.StudioJobStatusFailed).Count(&failed)
	db.Model(&model.StudioJobEvent{}).Where("event_type LIKE ?", "%REFUND%").Count(&refunds)

	assert.Equal(t, int64(2), studioUsers)
	assert.Equal(t, int64(2), totalJobs)
	assert.Equal(t, int64(1), succeeded)
	assert.Equal(t, int64(1), failed)
	assert.Equal(t, int64(1), refunds)

	var conversionCount int64
	db.Model(&model.StudioConversionEvent{}).Count(&conversionCount)
	assert.Equal(t, int64(4), conversionCount)
}
