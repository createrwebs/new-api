package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Section 37: Mandatory regression test verifying billing consistency across all subsystems.
func TestStudio_BillingConsistency_CanonicalConversion(t *testing.T) {
	// 1. Assert Authoritative System Constants
	assert.Equal(t, 500_000.0, common.QuotaPerUnit, "AUTHORITATIVE_INTERNAL_QUOTA_PER_USD must strictly be 500,000")
	assert.Equal(t, 1000, QuotaPerCredit, "AUTHORITATIVE_QUOTA_PER_TORA_CREDIT must strictly be 1,000")

	// 2. Derive Authoritative Units
	quotaPerUSD := common.QuotaPerUnit
	quotaPerCredit := float64(QuotaPerCredit)
	creditsPerUSD := quotaPerUSD / quotaPerCredit
	usdPerCredit := quotaPerCredit / quotaPerUSD

	assert.Equal(t, 500.0, creditsPerUSD, "AUTHORITATIVE_TORA_CREDITS_PER_USD must be 500 Credits per $1.00 USD")
	assert.InDelta(t, 0.0020, usdPerCredit, 0.00001, "AUTHORITATIVE_USD_EQUIVALENT_PER_TORA_CREDIT must be $0.0020 USD")

	// 3. Test Pricing Engine Consistency
	engine := NewPricingEngine()

	// Convert 10 credits -> 10,000 quota
	assert.Equal(t, 10000, engine.CreditsToQuota(10))
	assert.Equal(t, 10, engine.QuotaToCredits(10000))

	// If a tool costs $0.0040 USD provider COGS with 60% target margin:
	// Sell USD = 0.0040 / (1 - 0.60) = $0.0100 USD.
	// Base quota = 0.0100 * 500,000 = 5,000 quota units.
	// Credits = 5,000 / 1,000 = 5 Tora Credits.
	credits, quota, sellUSD := engine.CalculatePrice("utility", 0.0040, 60.0, 1.0)
	assert.Equal(t, 5, credits, "Sell price of $0.010 USD must equal 5 Tora Credits")
	assert.Equal(t, 5000, quota, "5 Tora Credits must equal 5,000 internal quota units")
	assert.InDelta(t, 0.0100, sellUSD, 0.0001)

	// Revenue recognition check (from service/studio_service.go:513)
	recognizedRevenueUSD := float64(quota) / common.QuotaPerUnit
	assert.InDelta(t, 0.0100, recognizedRevenueUSD, 0.0001, "Recognized revenue must equal sell price")

	// 4. Assert Queue 2F errant assumption fails
	errantQuotaPerCredit := 100
	assert.NotEqual(t, errantQuotaPerCredit, QuotaPerCredit, "System must reject legacy 100 quota per credit assumption")
}

// Section 38: Mandatory route contract test verifying WaveSpeed and KIE endpoint contracts.
func TestStudio_RouteContract_WaveSpeedAndKie(t *testing.T) {
	// A. WaveSpeed Endpoint Contract Test
	var recordedWsPaths []string
	var recordedWsMethods []string
	var recordedWsAuth string

	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedWsPaths = append(recordedWsPaths, r.URL.Path)
		recordedWsMethods = append(recordedWsMethods, r.Method)
		recordedWsAuth = r.Header.Get("Authorization")

		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "birefnet") {
			// POST /api/v3/{model_id}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"id":     "ws_test_task_123",
					"status": "created",
					"urls": map[string]string{
						"get": "/predictions/ws_test_task_123/result",
					},
				},
			})
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "predictions") {
			// GET /api/v3/predictions/{id}/result
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"id":      "ws_test_task_123",
					"status":  "completed",
					"outputs": []string{"https://cdn.wavespeed.ai/transparent.png"},
				},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer wsServer.Close()

	os.Setenv("TEST_WS_KEY", "real-ws-secret-xyz")
	defer os.Unsetenv("TEST_WS_KEY")

	wsProvider := &model.StudioProviderConfig{
		Id:        "wavespeed",
		Name:      "WaveSpeed",
		BaseURL:   wsServer.URL,
		SecretEnv: "TEST_WS_KEY",
		AuthType:  "BEARER",
		Protocol:  model.ProtocolWaveSpeedV3,
		Enabled:   true,
	}

	wsRoute := &model.StudioModelRoute{
		Id:              "ws-birefnet",
		LogicalTool:     "background-remove",
		ProviderId:      "wavespeed",
		Protocol:        model.ProtocolWaveSpeedV3,
		ProviderModelId: "wavespeed-ai/birefnet",
		Status:          model.RouteStatusActive,
		Enabled:         true,
		InputMapping:    `{"image_url":"image_url"}`,
	}

	wsAdapter := NewWaveSpeedAdapter()
	normInput := &NormalizedMediaInput{
		InputAssets: []string{"https://assets.toraapi.com/input.png"},
	}

	// Submit task
	submitOut, err := wsAdapter.Submit(context.Background(), wsProvider, wsRoute, normInput)
	require.NoError(t, err)
	assert.Equal(t, "ws_test_task_123", submitOut.ProviderJobID)
	assert.Equal(t, "Bearer real-ws-secret-xyz", recordedWsAuth)

	// Specifically verify that POST /api/v3/{model_id} was called and NOT /api/v3/media/tasks
	require.NotEmpty(t, recordedWsPaths)
	assert.Equal(t, "/wavespeed-ai/birefnet", recordedWsPaths[0], "WaveSpeed submission must hit /{model_id}")
	assert.False(t, strings.Contains(recordedWsPaths[0], "media/tasks"), "Must NEVER hit obsolete /api/v3/media/tasks path")

	// Status check
	statusOut, err := wsAdapter.Status(context.Background(), wsProvider, wsRoute, "ws_test_task_123")
	require.NoError(t, err)
	assert.Equal(t, "completed", statusOut.Status)
	assert.Equal(t, "/predictions/ws_test_task_123/result", recordedWsPaths[1], "WaveSpeed status query must hit predictions/{id}/result")

	// B. KIE Endpoint Contract Test
	var recordedKiePath string
	var recordedKieBody map[string]interface{}

	kieServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedKiePath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&recordedKieBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"taskId": "kie_task_abc_456",
			},
		})
	}))
	defer kieServer.Close()

	os.Setenv("TEST_KIE_KEY", "real-kie-secret-789")
	defer os.Unsetenv("TEST_KIE_KEY")

	kieProvider := &model.StudioProviderConfig{
		Id:        "kie",
		Name:      "KIE",
		BaseURL:   kieServer.URL,
		SecretEnv: "TEST_KIE_KEY",
		AuthType:  "BEARER",
		Protocol:  model.ProtocolKieJobsV1,
		Enabled:   true,
	}

	kieRoute := &model.StudioModelRoute{
		Id:              "kie-rembg",
		LogicalTool:     "background-remove",
		ProviderId:      "kie",
		Protocol:        model.ProtocolKieJobsV1,
		ProviderModelId: "rembg",
		Status:          model.RouteStatusActive,
		Enabled:         true,
		InputMapping:    `{"image_url":"image_url"}`,
	}

	kieAdapter := NewKieAdapter()
	normKieInput := &NormalizedMediaInput{
		InputAssets: []string{"https://assets.toraapi.com/input.png"},
		CallbackURL: "https://www.toraapi.com/api/studio/webhook/kie",
	}

	kieSubmitOut, err := kieAdapter.Submit(context.Background(), kieProvider, kieRoute, normKieInput)
	require.NoError(t, err)
	assert.Equal(t, "kie_task_abc_456", kieSubmitOut.ProviderJobID)
	assert.Equal(t, "/api/v1/jobs/createTask", recordedKiePath)
	assert.Equal(t, "rembg", recordedKieBody["model"])
	assert.Equal(t, "https://www.toraapi.com/api/studio/webhook/kie", recordedKieBody["callBackUrl"])
	assert.NotNil(t, recordedKieBody["input"], "KIE payload must contain input mapping")
}

// Section 18 & 22: Tests governance rejection of UNKNOWN price sources and inactive route states.
func TestStudio_Governance_RejectsUnknownPriceSourceAndInactiveStates(t *testing.T) {
	dsn := fmt.Sprintf("file:test_gov_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	require.NoError(t, model.EnsureStudioTables(db))

	providerCfg := &model.StudioProviderConfig{
		Id:           "mock-provider",
		Name:         "Mock Provider",
		BaseURL:      "http://localhost",
		SecretEnv:    "MOCK_KEY",
		Protocol:     model.ProtocolMock,
		Enabled:      true,
		HealthStatus: model.ProviderHealthActive,
	}
	require.NoError(t, model.SaveStudioProviderConfig(providerCfg))
	os.Setenv("MOCK_KEY", "dummy")
	defer os.Unsetenv("MOCK_KEY")

	// 1. Route with PriceSource: UNKNOWN
	unknownPriceRoute := &model.StudioModelRoute{
		Id:               "route-unknown-price",
		LogicalTool:      "test-gov-tool",
		ProviderId:       "mock-provider",
		Protocol:         model.ProtocolMock,
		ProviderModelId:  "model-unknown",
		QualityTier:      "FAST",
		Status:           model.RouteStatusActive,
		Enabled:          true,
		PriceSource:      model.PriceSourceUnknown, // UNKNOWN!
		BaseCostUSD:      0.005,
		EffectiveCostUSD: 0.005,
		MinMargin:        60.0,
		Priority:         1,
		SuccessRate:      100.0,
	}
	require.NoError(t, model.SaveStudioModelRoute(unknownPriceRoute))

	router := NewStudioRouter(nil)
	primary, _, _, _, candidates, err := router.SelectRoute(context.Background(), "test-gov-tool", QualityTierFast, nil, 10000)
	assert.Error(t, err)
	assert.Nil(t, primary)
	require.Len(t, candidates, 1)
	assert.False(t, candidates[0].IsEligible)
	assert.Contains(t, candidates[0].SkipReason, "price source UNKNOWN rejected by governance")

	// 2. Route with Status: DRAFT or BILLING_BLOCKED
	unknownPriceRoute.PriceSource = model.PriceSourceManualVerified
	unknownPriceRoute.Status = model.RouteStatusDraft
	require.NoError(t, model.SaveStudioModelRoute(unknownPriceRoute))

	primary, _, _, _, candidates, err = router.SelectRoute(context.Background(), "test-gov-tool", QualityTierFast, nil, 10000)
	assert.Error(t, err)
	assert.Nil(t, primary)
	require.Len(t, candidates, 1)
	assert.False(t, candidates[0].IsEligible)
	assert.Contains(t, candidates[0].SkipReason, "route status draft")
}
