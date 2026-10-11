package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestRelayDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:test_relay_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	require.NoError(t, model.EnsureStudioTables(db))
	require.NoError(t, SeedStudioCatalog(db))

	// Register adapters
	registry := GetProtocolRegistry()
	registry.Register(NewWaveSpeedAdapter())
	registry.Register(NewKieAdapter())
	registry.Register(NewFalQueueAdapter(NewFalProvider()))
	registry.Register(NewReplicatePredictionsAdapter(NewReplicateProvider()))
	registry.Register(NewMockProtocolAdapter(NewDeterministicMockProvider(MockModeInstantSuccess)))

	// Set test credentials so adapter.ValidateConfiguration passes
	os.Setenv("WAVESPEED_API_KEY", "test-ws-key")
	os.Setenv("KIE_API_KEY", "test-kie-key")
	os.Setenv("FAL_KEY", "test-fal-key")
	os.Setenv("REPLICATE_API_TOKEN", "test-rep-token")

	// Ensure Wallet tables
	_ = db.AutoMigrate(&model.WalletPreConsumeRecord{}, &model.User{})

	// In test DB, activate WaveSpeed and KIE routes because test credentials are provided
	_ = db.Model(&model.StudioModelRoute{}).Where("provider_id IN ?", []string{"wavespeed", "kie"}).Update("status", model.RouteStatusActive)

	return db
}

// 1. Section 37: Proves adding a second WaveSpeed model requires ZERO new Go provider classes.
func TestStudio_Genericity_AddSecondWaveSpeedModelPurelyViaConfig(t *testing.T) {
	db := setupTestRelayDB(t)

	// Mock WaveSpeed Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-ws-key", r.Header.Get("Authorization"))
		if r.URL.Path == "/wavespeed-ai/flux-ultra" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"id":      "ws_task_flux_ultra_999",
					"status":  "completed",
					"outputs": []string{"https://cdn.wavespeed.ai/output_ultra.png"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	os.Setenv("WAVESPEED_TEST_KEY", "test-ws-key")
	defer os.Unsetenv("WAVESPEED_TEST_KEY")

	// 1. Insert Provider Config
	providerCfg := &model.StudioProviderConfig{
		Id:           "wavespeed-test",
		Name:         "WaveSpeed AI Test",
		BaseURL:      server.URL,
		SecretEnv:    "WAVESPEED_TEST_KEY",
		AuthType:     "BEARER",
		Protocol:     model.ProtocolWaveSpeedV3,
		Enabled:      true,
		Priority:     1,
		HealthStatus: model.ProviderHealthActive,
	}
	require.NoError(t, model.SaveStudioProviderConfig(providerCfg))

	// 2. Add Second Model via pure DB configuration (ZERO Go Code additions!)
	secondModelRoute := &model.StudioModelRoute{
		Id:               "ws-flux-ultra",
		LogicalTool:      "image-generate",
		ProviderId:       "wavespeed-test",
		Protocol:         model.ProtocolWaveSpeedV3,
		ProviderModelId:  "wavespeed-ai/flux-ultra", // New model ID
		QualityTier:      "PREMIUM",
		Enabled:          true,
		PricingStrategy:  "FIXED_COGS",
		BaseCostUSD:      0.035,
		EffectiveCostUSD: 0.035,
		Priority:         1,
		InputMapping:     `{"prompt":"prompt","aspect_ratio":"aspect_ratio"}`,
	}
	require.NoError(t, model.SaveStudioModelRoute(secondModelRoute))

	// 3. Verify registry dispatches using existing generic adapter
	adapter, err := GetProtocolRegistry().Get(secondModelRoute.Protocol)
	require.NoError(t, err)
	require.NotNil(t, adapter)

	input := &NormalizedMediaInput{
		Prompt:      "A hyperrealistic photo of a tiger in snow",
		AspectRatio: "16:9",
	}

	out, err := adapter.Submit(context.Background(), providerCfg, secondModelRoute, input)
	require.NoError(t, err)
	assert.Equal(t, "ws_task_flux_ultra_999", out.ProviderJobID)
	assert.Equal(t, "completed", out.Status)
	assert.Equal(t, "https://cdn.wavespeed.ai/output_ultra.png", out.PrimaryOutputURL())
	_ = db
}

// 2. Section 37: Proves adding a second KIE model requires ZERO new Go provider classes.
func TestStudio_Genericity_AddSecondKieModelPurelyViaConfig(t *testing.T) {
	db := setupTestRelayDB(t)

	// Mock KIE Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-kie-key", r.Header.Get("Authorization"))
		if r.URL.Path == "/api/v1/jobs/createTask" {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "ideogram-v2", body["model"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"taskId": "kie_task_ideogram_777",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	os.Setenv("KIE_TEST_KEY", "test-kie-key")
	defer os.Unsetenv("KIE_TEST_KEY")

	// 1. Insert Provider Config
	providerCfg := &model.StudioProviderConfig{
		Id:           "kie-test",
		Name:         "KIE.ai Test",
		BaseURL:      server.URL,
		SecretEnv:    "KIE_TEST_KEY",
		AuthType:     "BEARER",
		Protocol:     model.ProtocolKieJobsV1,
		Enabled:      true,
		Priority:     2,
		HealthStatus: model.ProviderHealthActive,
	}
	require.NoError(t, model.SaveStudioProviderConfig(providerCfg))

	// 2. Add Second KIE Model via pure DB configuration (ZERO Go Code additions!)
	secondModelRoute := &model.StudioModelRoute{
		Id:               "kie-ideogram-v2",
		LogicalTool:      "image-generate",
		ProviderId:       "kie-test",
		Protocol:         model.ProtocolKieJobsV1,
		ProviderModelId:  "ideogram-v2", // Newly added model
		QualityTier:      "QUALITY",
		Enabled:          true,
		PricingStrategy:  "FIXED_COGS",
		BaseCostUSD:      0.020,
		EffectiveCostUSD: 0.020,
		Priority:         1,
		InputMapping:     `{"prompt":"prompt"}`,
	}
	require.NoError(t, model.SaveStudioModelRoute(secondModelRoute))

	adapter, err := GetProtocolRegistry().Get(secondModelRoute.Protocol)
	require.NoError(t, err)

	input := &NormalizedMediaInput{
		Prompt: "Typography poster with text 'TORA AI'",
	}

	out, err := adapter.Submit(context.Background(), providerCfg, secondModelRoute, input)
	require.NoError(t, err)
	assert.Equal(t, "kie_task_ideogram_777", out.ProviderJobID)
	assert.Equal(t, "queued", out.Status)
	_ = db
}

// 3. Section 37: Proves a logical tool can have multiple routes.
func TestStudio_Router_LogicalToolHasMultipleRoutes(t *testing.T) {
	_ = setupTestRelayDB(t)

	routes, err := model.GetStudioModelRoutesByTool("background-remove")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(routes), 3, "background-remove should have at least 3 configured routes (WaveSpeed, KIE, Fal)")
}

// 4. Section 37: Proves a route can be disabled and is skipped by router.
func TestStudio_Router_DisabledRouteIsSkipped(t *testing.T) {
	_ = setupTestRelayDB(t)

	// Disable WaveSpeed route
	route, err := model.GetStudioModelRoute("ws-birefnet")
	require.NoError(t, err)
	route.Enabled = false
	require.NoError(t, model.SaveStudioModelRoute(route))

	router := NewStudioRouter(nil)
	primary, _, _, _, candidates, err := router.SelectRoute(context.Background(), "background-remove", QualityTierFast, nil, 10000)

	// ws-birefnet must be in candidates but marked ineligible
	var wsCand *RouteCandidateResult
	for i := range candidates {
		if candidates[i].Route.Id == "ws-birefnet" {
			wsCand = &candidates[i]
			break
		}
	}
	require.NotNil(t, wsCand)
	assert.False(t, wsCand.IsEligible)
	assert.Equal(t, "route disabled", wsCand.SkipReason)

	if primary != nil {
		assert.NotEqual(t, "ws-birefnet", primary.Id)
	}
}

// 5. Section 37 & 35: Proves billing-blocked provider (Fal) is skipped safely without error.
func TestStudio_Router_BillingBlockedProviderSkippedSafely(t *testing.T) {
	_ = setupTestRelayDB(t)

	// Ensure Fal provider is configured with BILLING_BLOCKED
	falCfg, err := model.GetStudioProviderConfig("fal")
	require.NoError(t, err)
	assert.Equal(t, model.ProviderHealthBillingBlocked, falCfg.HealthStatus)

	router := NewStudioRouter(nil)
	// Select route for background-remove where fal-birefnet exists
	primary, _, _, reason, candidates, _ := router.SelectRoute(context.Background(), "background-remove", QualityTierQuality, nil, 10000)

	var falCand *RouteCandidateResult
	for i := range candidates {
		if candidates[i].Route.Id == "fal-birefnet" {
			falCand = &candidates[i]
			break
		}
	}
	require.NotNil(t, falCand)
	assert.False(t, falCand.IsEligible)
	assert.Contains(t, falCand.SkipReason, "billing blocked")

	if primary != nil {
		assert.NotEqual(t, "fal-birefnet", primary.Id)
	}
	_ = reason
}

// 6. Section 37 & 23: Proves price-based route selection prefers lower cost route.
func TestStudio_Router_PriceBasedRouteSelection(t *testing.T) {
	_ = setupTestRelayDB(t)

	// Configure two mock routes with identical quality tier and priority, but different costs
	mockCheap := &model.StudioModelRoute{
		Id:               "mock-cheap",
		LogicalTool:      "test-pricing-tool",
		ProviderId:       "mock",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "mock-cheap-v1",
		QualityTier:      "QUALITY",
		Enabled:          true,
		PricingStrategy:  "FIXED_COGS",
		BaseCostUSD:      0.002,
		EffectiveCostUSD: 0.002,
		Priority:         1,
		SuccessRate:      100.0,
	}
	mockExpensive := &model.StudioModelRoute{
		Id:               "mock-expensive",
		LogicalTool:      "test-pricing-tool",
		ProviderId:       "mock",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "mock-exp-v1",
		QualityTier:      "QUALITY",
		Enabled:          true,
		PricingStrategy:  "FIXED_COGS",
		BaseCostUSD:      0.015,
		EffectiveCostUSD: 0.015,
		Priority:         1,
		SuccessRate:      100.0,
	}
	require.NoError(t, model.SaveStudioModelRoute(mockCheap))
	require.NoError(t, model.SaveStudioModelRoute(mockExpensive))

	router := NewStudioRouter(nil)
	primary, _, fallbacks, _, _, err := router.SelectRoute(context.Background(), "test-pricing-tool", QualityTierQuality, nil, 50000)
	require.NoError(t, err)
	require.NotNil(t, primary)

	assert.Equal(t, "mock-cheap", primary.Id, "Cheaper route must be selected as primary when tiers are equal")
	require.Len(t, fallbacks, 1)
	assert.Equal(t, "mock-expensive", fallbacks[0].Id)
}

// 7. Section 37 & 22: Proves quality tier route selection matches requested tier.
func TestStudio_Router_QualityTierRouteSelection(t *testing.T) {
	_ = setupTestRelayDB(t)

	mockFast := &model.StudioModelRoute{
		Id:               "mock-fast-model",
		LogicalTool:      "test-tier-tool",
		ProviderId:       "mock",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "fast-v1",
		QualityTier:      "FAST",
		Enabled:          true,
		PricingStrategy:  "FIXED_COGS",
		BaseCostUSD:      0.005,
		EffectiveCostUSD: 0.005,
		Priority:         1,
		SuccessRate:      100.0,
	}
	mockPremium := &model.StudioModelRoute{
		Id:               "mock-premium-model",
		LogicalTool:      "test-tier-tool",
		ProviderId:       "mock",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "premium-v1",
		QualityTier:      "PREMIUM",
		Enabled:          true,
		PricingStrategy:  "FIXED_COGS",
		BaseCostUSD:      0.006,
		EffectiveCostUSD: 0.006,
		Priority:         1,
		SuccessRate:      100.0,
	}
	require.NoError(t, model.SaveStudioModelRoute(mockFast))
	require.NoError(t, model.SaveStudioModelRoute(mockPremium))

	router := NewStudioRouter(nil)

	// User requests FAST
	primaryFast, _, _, _, _, err := router.SelectRoute(context.Background(), "test-tier-tool", QualityTierFast, nil, 50000)
	require.NoError(t, err)
	assert.Equal(t, "mock-fast-model", primaryFast.Id)

	// User requests PREMIUM
	primaryPrem, _, _, _, _, err := router.SelectRoute(context.Background(), "test-tier-tool", QualityTierPremium, nil, 50000)
	require.NoError(t, err)
	assert.Equal(t, "mock-premium-model", primaryPrem.Id)
}

// 8. Section 37 & 24: Proves health-based route selection penalizes failing routes and trips circuit breaker.
func TestStudio_Router_HealthBasedRouteSelection(t *testing.T) {
	_ = setupTestRelayDB(t)

	brokenRoute := &model.StudioModelRoute{
		Id:               "mock-broken-route",
		LogicalTool:      "test-health-tool",
		ProviderId:       "mock",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "broken-v1",
		QualityTier:      "QUALITY",
		Enabled:          true,
		BaseCostUSD:      0.001,
		EffectiveCostUSD: 0.001,
		Priority:         1,
		ConsecutiveFails: 5, // Circuit breaker tripped!
		SuccessRate:      20.0,
	}
	healthyRoute := &model.StudioModelRoute{
		Id:               "mock-healthy-route",
		LogicalTool:      "test-health-tool",
		ProviderId:       "mock",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "healthy-v1",
		QualityTier:      "QUALITY",
		Enabled:          true,
		BaseCostUSD:      0.005,
		EffectiveCostUSD: 0.005,
		Priority:         1,
		ConsecutiveFails: 0,
		SuccessRate:      100.0,
	}
	require.NoError(t, model.SaveStudioModelRoute(brokenRoute))
	require.NoError(t, model.SaveStudioModelRoute(healthyRoute))

	router := NewStudioRouter(nil)
	primary, _, _, _, candidates, err := router.SelectRoute(context.Background(), "test-health-tool", QualityTierQuality, nil, 50000)
	require.NoError(t, err)
	require.NotNil(t, primary)

	assert.Equal(t, "mock-healthy-route", primary.Id)

	var brokenCand *RouteCandidateResult
	for i := range candidates {
		if candidates[i].Route.Id == "mock-broken-route" {
			brokenCand = &candidates[i]
			break
		}
	}
	require.NotNil(t, brokenCand)
	assert.False(t, brokenCand.IsEligible)
	assert.Contains(t, brokenCand.SkipReason, "circuit breaker tripped")
}

// 9. Section 37 & 26: Proves ambiguous first submission strictly prevents fallback.
type AmbiguousFailingAdapter struct {
	MediaProtocolAdapter
	submitCalls int
}

func (a *AmbiguousFailingAdapter) Protocol() string { return "AMBIGUOUS_TEST" }
func (a *AmbiguousFailingAdapter) ValidateConfiguration(provider *model.StudioProviderConfig) error {
	return nil
}
func (a *AmbiguousFailingAdapter) Submit(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, input *NormalizedMediaInput) (*NormalizedMediaOutput, error) {
	a.submitCalls++
	return nil, fmt.Errorf("%w: downstream timeout after HTTP payload sent", ErrProviderAmbiguous)
}

func TestStudio_Router_AmbiguousFirstSubmissionPreventsFallback(t *testing.T) {
	_ = setupTestRelayDB(t)

	ambAdapter := &AmbiguousFailingAdapter{}
	GetProtocolRegistry().Register(ambAdapter)

	primaryRoute := &model.StudioModelRoute{
		Id:          "route-ambiguous",
		LogicalTool: "test-ambiguous",
		ProviderId:  "mock",
		Protocol:    "AMBIGUOUS_TEST",
	}
	fallbackRoute := &model.StudioModelRoute{
		Id:          "route-fallback-should-not-run",
		LogicalTool: "test-ambiguous",
		ProviderId:  "mock",
		Protocol:    model.ProtocolMock,
	}

	job := &model.StudioToolJob{
		Id: "job_amb_test_1",
	}

	router := NewStudioRouter(nil)
	out, winningRoute, err := router.ExecuteRouteWithSafeFallback(
		context.Background(),
		job,
		&NormalizedMediaInput{},
		primaryRoute,
		[]*model.StudioModelRoute{fallbackRoute},
	)

	assert.Nil(t, out)
	assert.Equal(t, primaryRoute.Id, winningRoute.Id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProviderAmbiguous), "Must return ErrProviderAmbiguous")
	assert.Equal(t, 1, ambAdapter.submitCalls, "Must NOT dispatch to fallback route!")
}

// 10. Section 37 & 17: Proves duplicate callback settles quota exactly once.
func TestStudio_Webhook_DuplicateCallbackSettlesOnce(t *testing.T) {
	db := setupTestRelayDB(t)

	// Create user with initial quota
	user := &model.User{
		Id:    101,
		Quota: 100000,
	}
	_ = db.Create(user)

	jobId := "job_idemp_callback_1"
	reqId := "req_idemp_callback_1"
	reservedQuota := 20000

	// Pre-consume
	err := model.PreConsumeUserWallet(reqId, user.Id, reservedQuota)
	require.NoError(t, err)

	job := &model.StudioToolJob{
		Id:            jobId,
		UserId:        user.Id,
		ToolId:        "image-generate",
		RequestId:     reqId,
		ProviderJobId: "kie_task_callback_xyz",
		ProviderName:  "kie",
		Status:        model.StudioJobStatusProcessing,
		ReservedQuota: reservedQuota,
		SettledQuota:  0,
	}
	require.NoError(t, db.Create(job).Error)

	svc := GetStudioService()

	// First Webhook Callback
	firstJob, err := svc.HandleWebhook(context.Background(), "kie", "kie_task_callback_xyz", "success", "https://cdn.kie.ai/output.png", "")
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, firstJob.Status)
	assert.Equal(t, reservedQuota, firstJob.SettledQuota)

	var record model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", reqId).First(&record).Error)
	assert.Equal(t, "settled", strings.ToLower(record.Status))

	// Duplicate Webhook Callback
	dupJob, err := svc.HandleWebhook(context.Background(), "kie", "kie_task_callback_xyz", "success", "https://cdn.kie.ai/output.png", "")
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, dupJob.Status)
	assert.Equal(t, reservedQuota, dupJob.SettledQuota)

	// Verify wallet record was settled only once
	var count int64
	db.Model(&model.StudioJobEvent{}).Where("job_id = ? AND event_type = ?", jobId, "JOB_SUCCEEDED_WEBHOOK").Count(&count)
	assert.Equal(t, int64(1), count, "Event must be recorded only once")
}

// 11. Section 28: Profitability guard prevents loss-making routes.
func TestStudio_ProfitabilityGuard_RejectsLossMakingRoute(t *testing.T) {
	_ = setupTestRelayDB(t)

	// Expensive route ($0.050) vs retail quota (10,000 Quota = 10 Credits = $0.010)
	lossMakingRoute := &model.StudioModelRoute{
		Id:               "mock-loss-maker",
		LogicalTool:      "test-margin-tool",
		ProviderId:       "mock",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "expensive-gpu-v1",
		QualityTier:      "QUALITY",
		Status:           model.RouteStatusActive,
		Enabled:          true,
		PricingStrategy:  "FIXED_COGS",
		BaseCostUSD:      0.050,
		EffectiveCostUSD: 0.050,
		MinMargin:        60.0,
		Priority:         1,
		SuccessRate:      100.0,
	}
	require.NoError(t, model.SaveStudioModelRoute(lossMakingRoute))

	router := NewStudioRouter(nil)
	primary, _, _, _, candidates, err := router.SelectRoute(context.Background(), "test-margin-tool", QualityTierQuality, nil, 10000)

	assert.Error(t, err)
	assert.Nil(t, primary)
	require.Len(t, candidates, 1)
	assert.False(t, candidates[0].IsEligible)
	assert.Contains(t, candidates[0].SkipReason, "profitability violation")
}
