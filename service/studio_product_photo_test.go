package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueue4_ProductPhoto_PresetsAndTemplates validates all 5 marketplace presets
// and 8 visual templates seeded in catalog for Product Studio (Queue 4).
func TestQueue4_ProductPhoto_PresetsAndTemplates(t *testing.T) {
	_ = setupTestDBForStudio(t)

	// 1. Verify product-photo tool status
	tool, err := model.GetStudioToolDefinition("product-photo")
	require.NoError(t, err, "product-photo tool must exist in catalog")
	assert.Equal(t, model.StudioToolStateActive, tool.Status, "product-photo must be ACTIVE in Queue 4")
	assert.True(t, tool.IsEnabled && tool.IsPublic, "product-photo must be enabled and public")
	assert.True(t, tool.MarginPercent >= 60.0, "product-photo margin must meet >= 60%% floor")

	// 2. Verify 5 Marketplace Presets
	expectedPresets := []string{
		"tpl-prod-shopee-1-1",
		"tpl-prod-lazada-1-1",
		"tpl-prod-instagram-4-5",
		"tpl-prod-story-9-16",
		"tpl-prod-tiktok-9-16",
	}

	marketplaceTemplates, err := model.ListStudioTemplates("product-photo", "marketplace")
	require.NoError(t, err)
	assert.Len(t, marketplaceTemplates, len(expectedPresets), "Must have exactly 5 marketplace presets")

	presetMap := make(map[string]model.StudioToolTemplate)
	for _, p := range marketplaceTemplates {
		presetMap[p.Id] = p
		assert.Equal(t, "product-photo", p.ToolId)
		assert.Equal(t, "marketplace", p.Category)
		assert.NotEmpty(t, p.PresetPrompt)
		assert.NotEmpty(t, p.PresetAspectRatio)
		assert.True(t, p.IsActive)
	}

	for _, expectedId := range expectedPresets {
		_, exists := presetMap[expectedId]
		assert.True(t, exists, "Marketplace preset %s must exist", expectedId)
	}

	// 3. Verify 8 Visual Templates
	expectedVisualStyles := []string{
		"tpl-prod-white-studio",
		"tpl-prod-luxury-black",
		"tpl-prod-minimal-beige",
		"tpl-prod-kitchen",
		"tpl-prod-food",
		"tpl-prod-cosmetics",
		"tpl-prod-fashion",
		"tpl-prod-outdoor",
	}

	visualTemplates, err := model.ListStudioTemplates("product-photo", "visual_style")
	require.NoError(t, err)
	assert.Len(t, visualTemplates, len(expectedVisualStyles), "Must have exactly 8 visual templates")

	visualMap := make(map[string]model.StudioToolTemplate)
	for _, v := range visualTemplates {
		visualMap[v.Id] = v
		assert.Equal(t, "product-photo", v.ToolId)
		assert.Equal(t, "visual_style", v.Category)
		assert.NotEmpty(t, v.PresetPrompt)
		assert.NotEmpty(t, v.PresetAspectRatio)
		assert.True(t, v.IsActive)
	}

	for _, expectedId := range expectedVisualStyles {
		_, exists := visualMap[expectedId]
		assert.True(t, exists, "Visual template %s must exist", expectedId)
	}
}

// TestQueue4_ProductPack_QuoteAndPricing verifies single (50 Cr) vs 4-pack (200 Cr)
// quotes, provider COGS proportionality, and margin floor preservation.
func TestQueue4_ProductPack_QuoteAndPricing(t *testing.T) {
	_ = setupTestDBForStudio(t)
	pricingEngine := NewPricingEngine()

	tool, err := model.GetStudioToolDefinition("product-photo")
	require.NoError(t, err)

	// 1. Single Result Pack Quote (50 Credits, $0.035 COGS, ~65% target margin)
	singleInputs := map[string]interface{}{
		"pack_size": 1,
	}
	quoteSingle, err := pricingEngine.CalculatePriceWithInputs(tool, singleInputs, 1.0)
	require.NoError(t, err)
	assert.Equal(t, float64(50), quoteSingle.CalculatedCredits, "Single output must cost 50 Credits")
	assert.Equal(t, int(50000), quoteSingle.ChargedQuota, "Single output must charge 50,000 Quota units ($0.10)")
	assert.InDelta(t, 0.035, quoteSingle.ProviderEstimatedCostUSD, 0.001, "Single output COGS must be ~$0.035")
	assert.True(t, quoteSingle.TargetMargin >= 60.0, "Single output margin must be >= 60%%")

	// 2. 4-Result Pack Quote (200 Credits, $0.140 COGS, ~65% target margin)
	pack4Inputs := map[string]interface{}{
		"pack_size": 4,
	}
	quotePack4, err := pricingEngine.CalculatePriceWithInputs(tool, pack4Inputs, 1.0)
	require.NoError(t, err)
	assert.Equal(t, float64(200), quotePack4.CalculatedCredits, "4-pack must cost 200 Credits")
	assert.Equal(t, int(200000), quotePack4.ChargedQuota, "4-pack must charge 200,000 Quota units ($0.40)")
	assert.InDelta(t, 0.140, quotePack4.ProviderEstimatedCostUSD, 0.001, "4-pack COGS must be ~$0.140")
	assert.True(t, quotePack4.TargetMargin >= 60.0, "4-pack margin must be >= 60%%")

	// 3. Margin Floor Enforcement
	err = pricingEngine.ValidateProfitability(quoteSingle, 60.0)
	assert.NoError(t, err)
	err = pricingEngine.ValidateProfitability(quotePack4, 60.0)
	assert.NoError(t, err)

	// 4. Default pack size when omitted
	quoteDefault, err := pricingEngine.CalculatePriceWithInputs(tool, map[string]interface{}{}, 1.0)
	require.NoError(t, err)
	assert.Equal(t, float64(50), quoteDefault.CalculatedCredits, "Default pack size must cost 50 Credits")
	assert.Equal(t, int(50000), quoteDefault.ChargedQuota)
}

// TestQueue4_MultiReference_Validation tests that valid external reference backgrounds
// are accepted, SSRF/private IPs are rejected, and raw logo diffusion is strictly guarded.
func TestQueue4_MultiReference_Validation(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "seller_tester",
		Quota:    500000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// 1. Valid Reference Background Image: Accepted
	validRefJob, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo",
		"tpl-prod-shopee-1-1",
		"ref_valid_01",
		"mock",
		map[string]interface{}{
			"image_url":           "https://1.1.1.1/product.png",
			"reference_image_url": "https://1.1.1.1/background.jpg",
			"pack_size":           1,
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, validRefJob.Status)

	// 2. Malicious SSRF Reference Image (Cloud Metadata IP): Rejected
	_, errSSRF := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo",
		"tpl-prod-shopee-1-1",
		"ref_ssrf_01",
		"mock",
		map[string]interface{}{
			"image_url":           "https://cdn.example.com/product.png",
			"reference_image_url": "http://169.254.169.254/latest/meta-data/",
			"pack_size":           1,
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, errSSRF)
	assert.Contains(t, errSSRF.Error(), "invalid reference background URL")

	// 3. Private Network Reference Image (10.0.0.1): Rejected
	_, errPrivate := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo",
		"tpl-prod-shopee-1-1",
		"ref_private_01",
		"mock",
		map[string]interface{}{
			"image_url":      "https://cdn.example.com/product.png",
			"background_url": "http://10.0.0.1/intranet_bg.jpg",
			"pack_size":      1,
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, errPrivate)
	assert.Contains(t, errPrivate.Error(), "invalid reference background URL")

	// 4. Raw Logo Diffusion Injection: Strictly Rejected with Advisory
	_, errLogo := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo",
		"tpl-prod-shopee-1-1",
		"ref_logo_01",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/product.png",
			"logo":      "brand_nike.png",
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, errLogo)
	assert.Contains(t, errLogo.Error(), "unreliable combination: direct in-model logo diffusion degrades brand typography")

	// 5. Raw Logo URL Injection: Strictly Rejected
	_, errLogoURL := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo",
		"tpl-prod-shopee-1-1",
		"ref_logourl_01",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/product.png",
			"logo_url":  "https://cdn.example.com/logo.png",
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.Error(t, errLogoURL)
	assert.Contains(t, errLogoURL.Error(), "unreliable combination: direct in-model logo diffusion degrades brand typography")
}

// TestQueue4_DeterministicExecution_SingleAndPack4 tests end-to-end execution
// of single vs 4-pack jobs with output variants serialization and quota deduction.
func TestQueue4_DeterministicExecution_SingleAndPack4(t *testing.T) {
	db := setupTestDBForStudio(t)

	// User starts with 300,000 Quota ($0.60, 300 credits)
	user := model.User{
		Username: "pack_tester",
		Quota:    300000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// Step 1: Submit Single Job (50 Credits = 50,000 Quota)
	jobSingle, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo",
		"tpl-prod-shopee-1-1",
		"exec_pack_single_01",
		"mock",
		map[string]interface{}{
			"image_url": "https://cdn.example.com/cosmetic.png",
			"pack_size": 1,
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, jobSingle.Status)
	assert.Equal(t, int(50000), jobSingle.SettledQuota)

	// Verify remaining quota after single job: 300,000 - 50,000 = 250,000
	var refreshedUser1 model.User
	require.NoError(t, db.First(&refreshedUser1, user.Id).Error)
	assert.Equal(t, int(250000), refreshedUser1.Quota)

	// Verify single output JSON format
	var singleOutput map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(jobSingle.OutputResult), &singleOutput))
	assert.NotEmpty(t, singleOutput["output_url"])

	// Step 2: Submit 4-Pack Job (200 Credits = 200,000 Quota)
	jobPack4, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo",
		"tpl-prod-tiktok-9-16",
		"exec_pack_4_01",
		"mock",
		map[string]interface{}{
			"image_url": "https://1.1.1.1/cosmetic.png",
			"pack_size": 4,
		},
		1.0,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, jobPack4.Status)
	assert.Equal(t, int(200000), jobPack4.SettledQuota)

	// Verify remaining quota after 4-pack job: 250,000 - 200,000 = 50,000
	var refreshedUser2 model.User
	require.NoError(t, db.First(&refreshedUser2, user.Id).Error)
	assert.Equal(t, int(50000), refreshedUser2.Quota)

	// Verify 4 variants serialized in OutputResult
	var pack4Output map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(jobPack4.OutputResult), &pack4Output))
	assert.NotEmpty(t, pack4Output["output_url"])

	urls, ok := pack4Output["output_urls"].([]interface{})
	require.True(t, ok, "output_urls must be an array in 4-pack result")
	assert.Len(t, urls, 4, "Must contain exactly 4 distinct output URLs")

	variants, ok := pack4Output["variants"].([]interface{})
	require.True(t, ok, "variants must be an array in 4-pack result")
	assert.Len(t, variants, 4, "Must contain exactly 4 distinct variants")
}

// TestQueue4_ConversionFunnel_Telemetry tests tracking and aggregation
// of Product Studio conversion funnel metrics.
func TestQueue4_ConversionFunnel_Telemetry(t *testing.T) {
	db := setupTestDBForStudio(t)

	// Create sample conversion funnel events for product-photo
	events := []model.StudioConversionEvent{
		{ToolId: "product-photo", EventType: "tool_visit", UserId: 101, CreatedAt: common.GetTimestamp()},
		{ToolId: "product-photo", EventType: "tool_visit", UserId: 102, CreatedAt: common.GetTimestamp()},
		{ToolId: "product-photo", EventType: "generate_click", UserId: 101, CreatedAt: common.GetTimestamp()},
		{ToolId: "product-photo", EventType: "insufficient_credit", UserId: 101, CreatedAt: common.GetTimestamp()},
		{ToolId: "product-photo", EventType: "buy_credit_click", UserId: 101, CreatedAt: common.GetTimestamp()},
		{ToolId: "product-photo", EventType: "successful_topup", UserId: 101, CreatedAt: common.GetTimestamp()},
		{ToolId: "product-photo", EventType: "generation_after_topup", UserId: 101, CreatedAt: common.GetTimestamp()},
		{ToolId: "product-photo", EventType: "repeat_generation", UserId: 101, CreatedAt: common.GetTimestamp()},
	}

	for _, ev := range events {
		require.NoError(t, db.Create(&ev).Error)
	}

	var prodVisitors int64
	var prodGenerateClicks int64
	var prodInsufficientCredit int64
	var prodBuyCreditClicks int64
	var prodSuccessfulTopups int64
	var prodGenerationAfterTopup int64
	var prodRepeatGenerations int64

	db.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND (event_type = ? OR event_type = ?)", "product-photo", "tool_visit", "tool_view").Count(&prodVisitors)
	db.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "generate_click").Count(&prodGenerateClicks)
	db.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "insufficient_credit").Count(&prodInsufficientCredit)
	db.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "buy_credit_click").Count(&prodBuyCreditClicks)
	db.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND (event_type = ? OR event_type = ?)", "product-photo", "purchase_return", "successful_topup").Count(&prodSuccessfulTopups)
	db.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND (event_type = ? OR event_type = ?)", "product-photo", "generation_after_purchase", "generation_after_topup").Count(&prodGenerationAfterTopup)
	db.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "repeat_generation").Count(&prodRepeatGenerations)

	assert.Equal(t, int64(2), prodVisitors)
	assert.Equal(t, int64(1), prodGenerateClicks)
	assert.Equal(t, int64(1), prodInsufficientCredit)
	assert.Equal(t, int64(1), prodBuyCreditClicks)
	assert.Equal(t, int64(1), prodSuccessfulTopups)
	assert.Equal(t, int64(1), prodGenerationAfterTopup)
	assert.Equal(t, int64(1), prodRepeatGenerations)
}
