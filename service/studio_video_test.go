package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueue5_VideoToolActivation verifies image-to-video is ACTIVE and other video tools remain FROZEN.
func TestQueue5_VideoToolActivation(t *testing.T) {
	db := setupTestDBForStudio(t)

	// 1. Verify image-to-video tool definition in catalog
	tool, err := model.GetStudioToolDefinition("image-to-video")
	require.NoError(t, err, "image-to-video tool must exist in catalog")
	assert.Equal(t, model.StudioToolStateActive, tool.Status, "image-to-video must be ACTIVE in Queue 5")
	assert.True(t, tool.IsEnabled && tool.IsPublic, "image-to-video must be enabled and public")
	assert.Equal(t, "high", tool.RiskClass, "image-to-video must have risk class 'high'")
	assert.True(t, tool.MarginPercent >= 60.0, "image-to-video margin floor must be >= 60%%")

	// 2. Verify Schema contains duration, resolution, quality, audio
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(tool.InputSchema), &schema))
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "schema must contain properties")
	assert.Contains(t, props, "duration", "schema must contain duration")
	assert.Contains(t, props, "resolution", "schema must contain resolution")
	assert.Contains(t, props, "quality", "schema must contain quality")
	assert.Contains(t, props, "audio", "schema must contain audio")

	// 3. Verify other video tools remain FROZEN (ComingSoon / not public)
	frozenVideoTools := []string{
		"text-to-video",
		"lip-sync",
		"talking-avatar",
		"video-upscale",
	}

	for _, vId := range frozenVideoTools {
		var vTool model.StudioToolDefinition
		err := db.Where("id = ?", vId).First(&vTool).Error
		require.NoError(t, err, "tool %s must exist in catalog", vId)
		assert.Equal(t, model.StudioToolStateComingSoon, vTool.Status, "tool %s must remain ComingSoon", vId)
		assert.False(t, vTool.IsPublic, "tool %s must not be public", vId)
		assert.False(t, vTool.IsEnabled, "tool %s must remain disabled", vId)
	}
}

// TestQueue5_DynamicVideoQuote_And_Economics validates dynamic video quoting and margin floor.
func TestQueue5_DynamicVideoQuote_And_Economics(t *testing.T) {
	_ = setupTestDBForStudio(t)

	pricingEngine := NewPricingEngine()
	toolDef, err := model.GetStudioToolDefinition("image-to-video")
	require.NoError(t, err)

	// Case 1: Base 5s 720p standard quality (Wan 2.2)
	params720p := map[string]interface{}{
		"duration":    5,
		"resolution":  "720p",
		"quality":     "standard",
		"num_outputs": 4, // Must be clamped to single output for video
	}
	sn720p, err := pricingEngine.CalculatePriceWithInputs(toolDef, params720p, 1.0)
	require.NoError(t, err)

	assert.Equal(t, 125, sn720p.ChargedCredits, "720p 5s should charge 125 Credits")
	assert.Equal(t, 125000, sn720p.ChargedQuota, "720p 5s should reserve 125,000 Quota")
	assert.InDelta(t, 0.080, sn720p.ProviderEstimatedCostUSD, 0.001, "720p 5s COGS should be $0.080")
	// Reference USD = 125 * $0.002 = $0.250. Margin = ($0.250 - $0.080)/$0.250 = 68.0%
	assert.InDelta(t, 68.0, sn720p.TargetMargin, 0.5, "Gross margin should be ~68.0%")
	assert.True(t, sn720p.TargetMargin >= 60.0, "Must satisfy >= 60%% margin floor")

	// Case 2: 1080p 5s standard quality
	params1080p := map[string]interface{}{
		"duration":   5,
		"resolution": "1080p",
		"quality":    "standard",
	}
	sn1080p, err := pricingEngine.CalculatePriceWithInputs(toolDef, params1080p, 1.0)
	require.NoError(t, err)

	assert.Equal(t, 157, sn1080p.ChargedCredits, "1080p 5s should charge 157 Credits (ceil(125 * 1.25))")
	assert.Equal(t, 157000, sn1080p.ChargedQuota)
	assert.InDelta(t, 0.100, sn1080p.ProviderEstimatedCostUSD, 0.001, "1080p 5s COGS should be $0.100")
	// Reference USD = 157 * $0.002 = $0.314. Margin = ($0.314 - $0.100)/$0.314 = 68.15%
	assert.InDelta(t, 68.15, sn1080p.TargetMargin, 0.5)
	assert.True(t, sn1080p.TargetMargin >= 60.0)

	// Case 3: 1080p 5s high quality + audio
	paramsHighAudio := map[string]interface{}{
		"duration":   5,
		"resolution": "1080p",
		"quality":    "high",
		"audio":      true,
	}
	snHighAudio, err := pricingEngine.CalculatePriceWithInputs(toolDef, paramsHighAudio, 1.0)
	require.NoError(t, err)

	// High quality (ceil(157 * 1.2) = 189) + audio (+15) = 204 Credits
	assert.Equal(t, 204, snHighAudio.ChargedCredits)
	assert.InDelta(t, 0.130, snHighAudio.ProviderEstimatedCostUSD, 0.001) // 0.100*1.2 + 0.010 = 0.130
	assert.True(t, snHighAudio.TargetMargin >= 60.0)

	// Case 4: Quote TTL enforcement (Video quotes have 5-minute TTL)
	quoteId := pricingEngine.SaveQuote(sn720p)
	require.NotEmpty(t, quoteId)
	retrievedQuote, err := pricingEngine.GetQuote(quoteId)
	require.NoError(t, err)
	assert.Equal(t, sn720p.ChargedCredits, retrievedQuote.ChargedCredits)
	assert.True(t, retrievedQuote.ExpiresAt > time.Now().Unix(), "Quote must not be expired immediately")
	assert.True(t, retrievedQuote.ExpiresAt <= time.Now().Add(6*time.Minute).Unix(), "Video quote TTL must be ~5 minutes")
}

// TestQueue5_AssetPipeline_RejectsBase64 verifies video media rejects base64 data URLs.
func TestQueue5_AssetPipeline_RejectsBase64(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "video_base64_tester",
		AffCode:  "AFF_VID_BASE64",
		Quota:    500000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// 1. Base64 payload must be strictly rejected
	base64Data := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	_, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-to-video",
		"",
		"test_base64_rejection",
		"mock",
		map[string]interface{}{
			"image_url": base64Data,
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base64 media input is not permitted for video workflows")

	// 2. Valid URL payload must be accepted
	jobValid, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-to-video",
		"",
		"test_valid_url_accepted",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/asset.png",
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, jobValid.Status)
}

// TestQueue5_VideoCostGuards validates duration, resolution, concurrency, daily spend, and quote expiry guards.
func TestQueue5_VideoCostGuards(t *testing.T) {
	db := setupTestDBForStudio(t)

	// Regular user (1 max concurrent video)
	userReg := model.User{
		Username: "video_guard_reg",
		AffCode:  "AFF_VID_REG",
		Quota:    2000000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
	}
	require.NoError(t, db.Create(&userReg).Error)

	// Admin user (2 max concurrent video)
	userAdmin := model.User{
		Username: "video_guard_admin",
		AffCode:  "AFF_VID_ADMIN",
		Quota:    2000000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&userAdmin).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	// Guard 1: Max Duration Guard (max 5s)
	_, errDur := studioSvc.SubmitJob(
		context.Background(),
		userReg.Id,
		"image-to-video",
		"",
		"test_dur_guard",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item.png",
			"duration":  10,
		},
		1.0,
		"127.0.0.1",
		false,
	)
	require.Error(t, errDur)
	assert.Contains(t, errDur.Error(), "duration exceeds maximum limit")

	// Guard 2: Max Resolution Guard (720p or 1080p only)
	_, errRes := studioSvc.SubmitJob(
		context.Background(),
		userReg.Id,
		"image-to-video",
		"",
		"test_res_guard",
		"mock",
		map[string]interface{}{
			"image_url":  "https://cdn.example.com/item.png",
			"resolution": "4k",
		},
		1.0,
		"127.0.0.1",
		false,
	)
	require.Error(t, errRes)
	assert.Contains(t, errRes.Error(), "resolution exceeds maximum limit")

	// Guard 3: Per-User Concurrency Guard
	// First job submitted and remains in PROCESSING state
	job1, err := studioSvc.SubmitJob(
		context.Background(),
		userReg.Id,
		"image-to-video",
		"",
		"test_concurrency_job1",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item.png",
		},
		1.0,
		"127.0.0.1",
		false,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, job1.Status)

	// Second job for regular user must be rejected
	_, errConc := studioSvc.SubmitJob(
		context.Background(),
		userReg.Id,
		"image-to-video",
		"",
		"test_concurrency_job2",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item2.png",
		},
		1.0,
		"127.0.0.1",
		false,
	)
	require.Error(t, errConc)
	assert.Contains(t, errConc.Error(), "concurrent video limit reached")

	// Admin user can submit first and second job
	adminJob1, err := studioSvc.SubmitJob(
		context.Background(),
		userAdmin.Id,
		"image-to-video",
		"",
		"test_admin_job1",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item.png",
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, adminJob1.Status)

	adminJob2, err := studioSvc.SubmitJob(
		context.Background(),
		userAdmin.Id,
		"image-to-video",
		"",
		"test_admin_job2",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item2.png",
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, adminJob2.Status)

	// Admin submitting 3rd job must be rejected (max 2)
	_, errAdmin3 := studioSvc.SubmitJob(
		context.Background(),
		userAdmin.Id,
		"image-to-video",
		"",
		"test_admin_job3",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item3.png",
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, errAdmin3)
	assert.Contains(t, errAdmin3.Error(), "concurrent video limit reached")

	// Guard 4: Quote Expiry Guard
	userQuote := model.User{
		Username: "video_guard_quote",
		AffCode:  "AFF_VID_QUOTE",
		Quota:    2000000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&userQuote).Error)

	expiredQuote := &model.StudioPricingSnapshot{
		ToolID:                   "image-to-video",
		ChargedCredits:           125,
		ChargedQuota:             125000,
		ProviderEstimatedCostUSD: 0.080,
		ExpiresAt:                time.Now().Add(-10 * time.Minute).Unix(), // Expired
	}
	expiredQuoteId := studioSvc.pricingEngine.SaveQuote(expiredQuote)
	require.NotEmpty(t, expiredQuoteId)

	_, errExp := studioSvc.SubmitJob(
		context.Background(),
		userQuote.Id,
		"image-to-video",
		"",
		"test_expired_quote",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item.png",
			"quote_id":  expiredQuoteId,
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, errExp)
	assert.Contains(t, errExp.Error(), "quote validation failed")

	// Guard 5: Daily Provider Spend Guard
	// Create simulated prior jobs today with estimated cost totaling $50.00
	userDaily := model.User{
		Username: "video_guard_daily",
		AffCode:  "AFF_VID_DAILY",
		Quota:    5000000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&userDaily).Error)

	capJob := &model.StudioToolJob{
		Id:              "job_prior_spend_cap",
		UserId:          userDaily.Id,
		ToolId:          "image-to-video",
		RequestId:       "req_prior_cap",
		IdempotencyKey:  "cap_prior_key",
		ProviderName:    "mock",
		Status:          model.StudioJobStatusSucceeded,
		ReservedQuota:   25000000,
		SettledQuota:    25000000,
		PricingSnapshot: `{"provider_estimated_cost_usd": 50.00}`,
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	}
	require.NoError(t, db.Create(capJob).Error)

	_, errDaily := studioSvc.SubmitJob(
		context.Background(),
		userDaily.Id,
		"image-to-video",
		"",
		"test_daily_cap_exceeded",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/item.png",
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, errDaily)
	assert.Contains(t, errDaily.Error(), "daily video capacity reached")
}

// TestQueue5_ControlledVideoCanary verifies end-to-end video canary execution through the real single wallet.
func TestQueue5_ControlledVideoCanary(t *testing.T) {
	db := setupTestDBForStudio(t)

	// User pre-charged with 500,000 Quota ($1.00 USD, 500 credits)
	user := model.User{
		Username: "video_canary_tester",
		AffCode:  "AFF_VID_CANARY",
		Quota:    500000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// Step 1: Request authoritative video quote
	toolDef, err := model.GetStudioToolDefinition("image-to-video")
	require.NoError(t, err)

	videoInputs := map[string]interface{}{
		"image_url":   "https://cdn.example.com/canary_subject.png",
		"duration":    5,
		"resolution":  "720p",
		"quality":     "standard",
		"num_outputs": 1,
	}

	quote, err := studioSvc.pricingEngine.CalculatePriceWithInputs(toolDef, videoInputs, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 125, quote.ChargedCredits)
	assert.Equal(t, 125000, quote.ChargedQuota)
	assert.InDelta(t, 0.080, quote.ProviderEstimatedCostUSD, 0.001)
	assert.InDelta(t, 68.0, quote.TargetMargin, 0.5)

	quoteId := studioSvc.pricingEngine.SaveQuote(quote)
	require.NotEmpty(t, quoteId)
	videoInputs["quote_id"] = quoteId

	// Step 2: Submit Video Job (pre-consume / reserve 125,000 Quota)
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-to-video",
		"",
		"video_canary_idemp_01",
		"mock",
		videoInputs,
		1.0,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, job.Status)
	assert.Equal(t, 125000, job.SettledQuota)
	assert.Equal(t, 125000, job.ReservedQuota)

	// Step 3: Verify single Tora Wallet deduction
	var refreshedUser model.User
	require.NoError(t, db.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, 500000-125000, refreshedUser.Quota, "User wallet must deduct exactly 125,000 Quota")

	// Step 4: Verify video output payload structure (.mp4, thumbnail/poster)
	var output map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(job.OutputResult), &output))
	assert.NotEmpty(t, output["output_url"], "must contain output_url")
	assert.NotEmpty(t, output["video_url"], "must contain video_url")
	assert.NotEmpty(t, output["thumbnail_url"], "must contain thumbnail_url")
	assert.NotEmpty(t, output["poster_url"], "must contain poster_url")

	videoURL, _ := output["video_url"].(string)
	assert.Contains(t, videoURL, ".mp4", "output must be an MP4 video")

	// Step 5: Verify ledger and pricing snapshot
	var snapshot model.StudioPricingSnapshot
	require.NoError(t, json.Unmarshal([]byte(job.PricingSnapshot), &snapshot))
	assert.Equal(t, 125, snapshot.ChargedCredits)
	assert.InDelta(t, 0.080, snapshot.ProviderEstimatedCostUSD, 0.001)
	assert.InDelta(t, 68.0, snapshot.TargetMargin, 0.5)

	fmt.Println("QUEUE 5 CANARY PROOF:")
	fmt.Printf("Job ID: %s | Succeeded | Settled Quota: %d (125 Credits)\n", job.Id, job.SettledQuota)
	fmt.Printf("Provider COGS: $%.3f | Revenue: $%.3f | Margin: %.1f%%\n",
		snapshot.ProviderEstimatedCostUSD, snapshot.CalculatedSellUSD, snapshot.TargetMargin)
	fmt.Printf("Video URL: %s\n", output["video_url"])
	fmt.Printf("Poster URL: %s\n", output["poster_url"])
}
