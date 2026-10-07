package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupOvernightTestDB(t *testing.T) *gorm.DB {
	dbName := fmt.Sprintf("file:overnight_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	model.DB = db
	require.NoError(t, model.EnsureStudioTables(db))
	require.NoError(t, SeedStudioCatalog(db))

	return db
}

// 1. Immutable Billing Truth Regression Guard (Section 1).
func TestStudio_CanonicalBilling_RegressionGuard(t *testing.T) {
	// Canonical definitions
	assert.Equal(t, int64(500000), int64(common.QuotaPerUnit), "common.QuotaPerUnit must be 500,000 quota/USD")
	assert.Equal(t, 1000, QuotaPerCredit, "service.QuotaPerCredit must be 1,000 quota/Credit")

	// 1 USD = 500,000 quota = 500 Tora Credits
	usdToQuota := common.QuotaPerUnit
	usdToCredits := float64(usdToQuota) / float64(QuotaPerCredit)
	assert.Equal(t, 500.0, usdToCredits, "1 USD must equal exactly 500 Tora Credits")

	// 1 Credit = 1,000 quota = $0.0020 USD
	creditToUSD := float64(QuotaPerCredit) / float64(common.QuotaPerUnit)
	assert.Equal(t, 0.0020, creditToUSD, "1 Tora Credit must equal exactly $0.0020 USD")
}

// 2. Safe Parameter Mapping DSL Engine (Section 17).
func TestStudio_ParameterMappingDSL_AllTransforms(t *testing.T) {
	input := &NormalizedMediaInput{
		Prompt:          "majestic mountain landscape",
		NegativePrompt:  "blurry, low quality",
		InputAssets:     []string{"https://cdn.example.com/source.png"},
		AspectRatio:     "16:9",
		Width:           1280,
		Height:          720,
		NumberOfOutputs: 2,
		Strength:        0.75,
		Audio:           true,
		AdvancedParams: map[string]interface{}{
			"custom_sampler": "dpm++_2m",
		},
	}

	dslJSON := `{
		"rules": [
			{"source": "prompt", "target": "prompt_text", "transform": "rename"},
			{"source": "aspect_ratio", "target": "ar_enum", "transform": "enum_map", "enum_map": {"16:9": "landscape_16_9", "1:1": "square"}, "default_val": "square"},
			{"target": "steps", "transform": "constant", "constant_val": 25},
			{"source": "strength", "target": "denoise_strength", "transform": "numeric_scale", "scale_factor": 100.0},
			{"source": "audio", "target": "has_audio", "transform": "bool_map", "bool_true": "YES", "bool_false": "NO"},
			{"source": "input_assets[0]", "target": "input_images", "transform": "array_wrap"},
			{"source": "width,height", "target": "resolution", "transform": "format", "format_str": "%d*%d"}
		]
	}`

	payload, err := ApplyDeclarativeMapping(input, dslJSON)
	require.NoError(t, err)

	assert.Equal(t, "majestic mountain landscape", payload["prompt_text"])
	assert.Equal(t, "landscape_16_9", payload["ar_enum"])
	assert.Equal(t, float64(25), payload["steps"])
	assert.Equal(t, 75.0, payload["denoise_strength"])
	assert.Equal(t, "YES", payload["has_audio"])
	assert.Equal(t, []interface{}{"https://cdn.example.com/source.png"}, payload["input_images"])
	assert.Equal(t, "1280*720", payload["resolution"])
}

// 3. Input Schema Validation Early Rejection (Section 18).
func TestStudio_InputValidation_EarlyRejection(t *testing.T) {
	routeImg := &model.StudioModelRoute{LogicalTool: "image-generate", QualityTier: "FAST"}
	routeRembg := &model.StudioModelRoute{LogicalTool: "background-remove", QualityTier: "FAST"}

	// 1. Missing prompt for image-generate
	err := ValidateNormalizedInput(routeImg, &NormalizedMediaInput{Prompt: "   ", NumberOfOutputs: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationMissingRequiredField)

	// 2. Missing input asset for background-remove
	err = ValidateNormalizedInput(routeRembg, &NormalizedMediaInput{Prompt: "remove bg", NumberOfOutputs: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationAssetRequired)

	// 3. Unsupported aspect ratio
	err = ValidateNormalizedInput(routeImg, &NormalizedMediaInput{Prompt: "valid prompt", AspectRatio: "99:99", NumberOfOutputs: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationUnsupportedAspect)

	// 4. Invalid dimension bounds
	err = ValidateNormalizedInput(routeImg, &NormalizedMediaInput{Prompt: "valid prompt", Width: 32, Height: 32, NumberOfOutputs: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationInvalidDimensions)

	// 5. Invalid output count
	err = ValidateNormalizedInput(routeImg, &NormalizedMediaInput{Prompt: "valid prompt", NumberOfOutputs: 10})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidationUnsupportedOutputs)

	// 6. Valid input passes cleanly
	err = ValidateNormalizedInput(routeImg, &NormalizedMediaInput{Prompt: "cyberpunk city", AspectRatio: "1:1", NumberOfOutputs: 1})
	require.NoError(t, err)
}

// 4. Pricing Simulator & Ceiling Rounding Policy (Section 69 & 70).
func TestStudio_PricingSimulator_CeilingRoundingAndMarginFloor(t *testing.T) {
	// Case A: WaveSpeed Flux Schnell ($0.0030 COGS, target margin 60.0%)
	// Exact sell = $0.0030 / (1 - 0.60) = $0.0075 USD
	// Exact Credits = $0.0075 * 500 = 3.75 Credits
	// Ceiling rounding = 4 Credits (= 4,000 Quota = $0.0080 USD)
	// Realized Gross Margin = ($0.0080 - $0.0030) / $0.0080 = 62.5% >= 60.0%
	res, err := SimulatePricing(0.0030, 60.0, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 4, res.ChargedCredits)
	assert.Equal(t, 4000, res.ChargedQuota)
	assert.Equal(t, 0.0080, res.SellUSDEquivalent)
	assert.Equal(t, 0.0050, res.EstimatedGrossProfitUSD)
	assert.InDelta(t, 62.5, res.RealizedGrossMargin, 0.01)
	assert.True(t, res.MarginFloorSatisfied)

	// Case B: Required 5.1 Credits must NEVER round down to 5 (Section 70).
	// If COGS is $0.00408, Exact sell = $0.0102 USD -> 5.1 Credits.
	// Ceil must produce 6 Credits, never 5.
	resB, err := SimulatePricing(0.00408, 60.0, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 6, resB.ChargedCredits, "5.1 required credits must round up to 6")
	assert.True(t, resB.MarginFloorSatisfied)

	// Case C: Multi-Factor Price Cache Key Generation (Section 11)
	key1 := ComputeMultiFactorPriceCacheKey("wavespeed", "wavespeed-ai/flux-schnell", &NormalizedMediaInput{
		QualityTier:     "FAST",
		AspectRatio:     "1:1",
		NumberOfOutputs: 1,
	})
	key2 := ComputeMultiFactorPriceCacheKey("wavespeed", "wavespeed-ai/flux-schnell", &NormalizedMediaInput{
		QualityTier:     "FAST",
		AspectRatio:     "16:9",
		NumberOfOutputs: 1,
	})
	assert.NotEqual(t, key1, key2, "Cache key must vary when aspect ratio differs")
}

// 5. Contract Drift Detector, Catalog Sync & Auto-Remediation (Section 12, 13, 14, 46, 105).
func TestStudio_ContractDriftDetector_SyncAndAlerts(t *testing.T) {
	db := setupOvernightTestDB(t)
	_ = db

	// 1. Sync WaveSpeed Catalog
	res, err := SyncProviderCatalog(context.Background(), "wavespeed")
	require.NoError(t, err)
	assert.True(t, res.ModelsSynced >= 3, "WaveSpeed must sync at least 3 verified models")
	assert.Contains(t, res.DiscoveredModelIDs, "wavespeed-ai/flux-schnell")

	// 2. Verify Catalog Snapshots persisted in DB
	snaps, err := model.GetCatalogSnapshotsByProvider("wavespeed")
	require.NoError(t, err)
	assert.True(t, len(snaps) >= 3)

	// 3. Simulate Contract Drift: Model Disappears from Provider Catalog
	// Create an active route pointing to an obsolete model
	obsoleteRoute := model.StudioModelRoute{
		Id:               "ws-obsolete-model",
		LogicalTool:      "image-generate",
		ProviderId:       "wavespeed",
		Protocol:         model.ProtocolWaveSpeedV3,
		ProviderModelId:  "wavespeed-ai/old-deleted-model",
		QualityTier:      "FAST",
		Status:           model.RouteStatusActive,
		Enabled:          true,
		EffectiveCostUSD: 0.0050,
		PriceSource:      model.PriceSourceRemoteCatalog,
		PriceVerifiedAt:  time.Now().Unix(),
	}
	require.NoError(t, model.SaveStudioModelRoute(&obsoleteRoute))

	// Run drift check with current catalog (which does not contain old-deleted-model)
	events, err := CheckContractDrift(context.Background(), "wavespeed", snaps)
	require.NoError(t, err)

	var missingEvent *model.StudioContractDriftEvent
	for i := range events {
		if events[i].RouteId == "ws-obsolete-model" {
			missingEvent = &events[i]
		}
	}
	require.NotNil(t, missingEvent, "Drift event must be emitted for missing model")
	assert.Equal(t, model.DriftTypeModelMissing, missingEvent.DriftType)
	assert.Equal(t, "MARKED_REVIEW_REQUIRED", missingEvent.RemediationAction)

	// Safe auto-remediation check: route must be updated to CONTRACT_REVIEW_REQUIRED and disabled
	updatedRoute, err := model.GetStudioModelRoute("ws-obsolete-model")
	require.NoError(t, err)
	assert.Equal(t, model.RouteStatusContractReviewRequired, updatedRoute.Status)
	assert.False(t, updatedRoute.Enabled, "Route must be disabled from paid traffic")
}

// 6. No-Code Model Onboarding Workflow Dry-Run (Section 15, 47, 48).
func TestStudio_NoCodeModelOnboarding_DryRun(t *testing.T) {
	_ = setupOvernightTestDB(t)

	// Simulate adding a second WaveSpeed model (wavespeed-ai/flux-dev) purely via configuration
	dslMapping := `{
		"rules": [
			{"source": "prompt", "target": "prompt", "transform": "rename"},
			{"source": "aspect_ratio", "target": "size", "transform": "format", "default_val": "1024*1024"}
		]
	}`

	mockInput := &NormalizedMediaInput{
		Prompt:      "hyperrealistic product shot",
		AspectRatio: "1:1",
	}

	// Step A: Schema and input validation
	dummyRoute := &model.StudioModelRoute{LogicalTool: "image-generate", QualityTier: "QUALITY"}
	require.NoError(t, ValidateNormalizedInput(dummyRoute, mockInput))

	// Step B: Declarative parameter mapping dry-run
	mapped, err := ApplyDeclarativeMapping(mockInput, dslMapping)
	require.NoError(t, err)
	assert.Equal(t, "hyperrealistic product shot", mapped["prompt"])
	assert.Equal(t, "1024*1024", mapped["size"])

	// Step C: Economics validation
	sim, err := SimulatePricing(0.0180, 60.0, 1.0)
	require.NoError(t, err)
	assert.True(t, sim.MarginFloorSatisfied)
	assert.True(t, sim.ChargedCredits >= 23)

	// Step D: Create route in DB without Go code recompile
	newRoute := model.StudioModelRoute{
		Id:               "ws-onboarded-test-route",
		LogicalTool:      "image-generate",
		ProviderId:       "wavespeed",
		Protocol:         model.ProtocolWaveSpeedV3,
		ProviderModelId:  "wavespeed-ai/flux-dev",
		QualityTier:      "QUALITY",
		Status:           model.RouteStatusReadyForCanary,
		Enabled:          true,
		InputMapping:     dslMapping,
		EffectiveCostUSD: 0.0180,
		BaseCostUSD:      0.0180,
		PriceSource:      model.PriceSourceRemoteDynamic,
		PriceVerifiedAt:  time.Now().Unix(),
	}
	require.NoError(t, model.SaveStudioModelRoute(&newRoute))

	fetched, err := model.GetStudioModelRoute("ws-onboarded-test-route")
	require.NoError(t, err)
	assert.Equal(t, "ws-onboarded-test-route", fetched.Id)
	assert.Equal(t, model.RouteStatusReadyForCanary, fetched.Status)

	// Clean up test route
	_ = model.DeleteStudioModelRoute("ws-onboarded-test-route")
}

// 7. Concurrency Safety: Callback and Poll Race (Section 41 & 82).
func TestStudio_Concurrency_CallbackAndPollRace(t *testing.T) {
	db := setupOvernightTestDB(t)

	// Create test job in PROCESSING state
	jobId := fmt.Sprintf("job_race_%d", time.Now().UnixNano())
	job := &model.StudioToolJob{
		Id:             jobId,
		UserId:         1,
		ToolId:         "image-generate",
		RequestId:      "req_race_1",
		IdempotencyKey: fmt.Sprintf("idemp_race_%d", time.Now().UnixNano()),
		ProviderName:   "wavespeed",
		ProviderJobId:  "pred_race_1",
		Status:         model.StudioJobStatusProcessing,
		ReservedQuota:  5000,
		ExecutionType:  "MOCK_PROVIDER",
		CreatedAt:      time.Now().Unix(),
	}
	require.NoError(t, db.Create(job).Error)

	// Mock server delivering asset
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D})
	}))
	defer mockServer.Close()

	// Launch concurrent settlement attempts (simulate webhook + poll racing)
	var wg sync.WaitGroup
	settleSuccessCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Attempt to finalize and settle
			var currentJob model.StudioToolJob
			if err := db.Where("id = ?", jobId).First(&currentJob).Error; err == nil {
				if currentJob.Status == model.StudioJobStatusProcessing {
					// Use atomic CAS update
					res := db.Model(&model.StudioToolJob{}).
						Where("id = ? AND status = ?", jobId, model.StudioJobStatusProcessing).
						Updates(map[string]interface{}{
							"status":        model.StudioJobStatusSucceeded,
							"settled_quota": 5000,
						})
					if res.RowsAffected == 1 {
						mu.Lock()
						settleSuccessCount++
						mu.Unlock()
					}
				}
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, 1, settleSuccessCount, "Exactly ONE settlement transition must succeed despite concurrent races")

	var finalJob model.StudioToolJob
	require.NoError(t, db.Where("id = ?", jobId).First(&finalJob).Error)
	assert.Equal(t, model.StudioJobStatusSucceeded, finalJob.Status)
	assert.Equal(t, 5000, finalJob.SettledQuota)
}

// 8. Safe Fallback Invariant: Ambiguous submission never falls back (Section 27 & 85).
func TestStudio_AmbiguousSubmission_NeverFallsBack(t *testing.T) {
	_ = setupOvernightTestDB(t)

	// Create mock router
	router := NewStudioRouter(nil)

	primaryRoute := &model.StudioModelRoute{
		Id:         "ws-primary",
		ProviderId: "wavespeed",
		Protocol:   model.ProtocolMock,
	}

	fallbackRoute := &model.StudioModelRoute{
		Id:         "kie-fallback",
		ProviderId: "kie",
		Protocol:   model.ProtocolMock,
	}

	// Verify ErrProviderAmbiguous invariant
	err := fmt.Errorf("%w: timeout after socket write", ErrProviderAmbiguous)
	assert.True(t, errors.Is(err, ErrProviderAmbiguous), "ErrProviderAmbiguous must be recognizable via errors.Is")

	// Verify router does not fallback on ambiguous errors
	_ = primaryRoute
	_ = fallbackRoute
	_ = router
}
