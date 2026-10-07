package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWaveSpeedAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:test_wsaudit_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	require.NoError(t, model.EnsureStudioTables(db))
	require.NoError(t, SeedStudioCatalog(db))

	_ = db.AutoMigrate(&model.WalletPreConsumeRecord{}, &model.User{}, &model.StudioAsset{})
	return db
}

// 1. Queue 2H Section 6: Proves unverified WaveSpeed routes are disabled/draft.
func TestStudio_WaveSpeed_RouteAudit_ReconciledCatalog(t *testing.T) {
	db := setupWaveSpeedAuditTestDB(t)

	var wsBirefnet model.StudioModelRoute
	require.NoError(t, db.Where("id = ?", "ws-birefnet").First(&wsBirefnet).Error)
	assert.Equal(t, model.RouteStatusDisabled, wsBirefnet.Status, "ws-birefnet must be DISABLED per Queue 2H Section 6")
	assert.False(t, wsBirefnet.Enabled, "ws-birefnet must be disabled")

	var wsUpscaler model.StudioModelRoute
	require.NoError(t, db.Where("id = ?", "ws-upscaler").First(&wsUpscaler).Error)
	assert.Equal(t, model.RouteStatusContractVerified, wsUpscaler.Status, "ws-upscaler must be CONTRACT_VERIFIED")
	assert.True(t, wsUpscaler.Enabled)
	assert.Equal(t, "wavespeed-ai/image-upscaler", wsUpscaler.ProviderModelId)

	var wsFluxSchnell model.StudioModelRoute
	require.NoError(t, db.Where("id = ?", "ws-flux-schnell").First(&wsFluxSchnell).Error)
	assert.Equal(t, "wavespeed-ai/flux-schnell", wsFluxSchnell.ProviderModelId)
	assert.True(t, wsFluxSchnell.Enabled)
	assert.Equal(t, 0.003, wsFluxSchnell.BaseCostUSD, "Base cost must reflect 0.003 baseline per Queue 2H Section 1")

	var wsProductFlux model.StudioModelRoute
	require.NoError(t, db.Where("id = ?", "ws-product-flux").First(&wsProductFlux).Error)
	assert.Equal(t, model.RouteStatusDraft, wsProductFlux.Status, "ws-product-flux must remain in DRAFT pending canary")
	assert.False(t, wsProductFlux.Enabled)
}

// 2. Queue 2H Section 29 & 30: Proves unverified KIE routes are downgraded to DRAFT.
func TestStudio_Kie_RouteAudit_AllGenericDowngradedToDraft(t *testing.T) {
	db := setupWaveSpeedAuditTestDB(t)

	kieRouteIds := []string{"kie-birefnet", "kie-upscale", "kie-flux-schnell", "kie-flux-dev", "kie-product-flux"}
	for _, id := range kieRouteIds {
		var route model.StudioModelRoute
		require.NoError(t, db.Where("id = ?", id).First(&route).Error)
		assert.Equal(t, model.RouteStatusDraft, route.Status, "Route %s must be DRAFT per Queue 2H Section 29", id)
		assert.False(t, route.Enabled, "Route %s must be disabled until verified", id)
	}
}

// 3. Queue 2H Section 8: Tests WaveSpeed dynamic pricing API contract and quoting.
func TestStudio_WaveSpeed_DynamicPricing_APIContract(t *testing.T) {
	var requestedModelId string
	var requestedInputs map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/model/price", r.URL.Path)
		assert.Equal(t, "Bearer test-ws-audit-key", r.Header.Get("Authorization"))

		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if m, ok := reqBody["model_id"].(string); ok {
			requestedModelId = m
		}
		if inp, ok := reqBody["inputs"].(map[string]interface{}); ok {
			requestedInputs = inp
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"base_price":       0.005,
				"discounted_price": 0.003,
				"currency":         "USD",
			},
		})
	}))
	defer server.Close()

	os.Setenv("WAVESPEED_AUDIT_KEY", "test-ws-audit-key")
	defer os.Unsetenv("WAVESPEED_AUDIT_KEY")

	adapter := NewWaveSpeedAdapter()
	providerCfg := &model.StudioProviderConfig{
		Id:        "wavespeed",
		BaseURL:   server.URL + "/api/v3",
		SecretEnv: "WAVESPEED_AUDIT_KEY",
	}
	route := &model.StudioModelRoute{
		Id:              "ws-flux-schnell",
		ProviderModelId: "wavespeed-ai/flux-schnell",
		PricingStrategy: "DYNAMIC_API",
		InputMapping:    `{"prompt":"prompt","aspect_ratio":"aspect_ratio","size":"size"}`,
	}
	input := &NormalizedMediaInput{
		Prompt:      "Abstract studio lighting geometry",
		AspectRatio: "1:1",
	}

	price, err := adapter.Quote(context.Background(), providerCfg, route, input)
	require.NoError(t, err)
	assert.Equal(t, 0.003, price)
	assert.Equal(t, "wavespeed-ai/flux-schnell", requestedModelId)
	assert.Equal(t, "Abstract studio lighting geometry", requestedInputs["prompt"])
	assert.Equal(t, "1024*1024", requestedInputs["size"])
}

// 4. Queue 2H Section 9: Tests fallback static cost with freshness TTL and safety multiplier.
func TestStudio_WaveSpeed_PricingFallback_FreshnessTTLAndSafetyMultiplier(t *testing.T) {
	adapter := NewWaveSpeedAdapter()
	providerCfg := &model.StudioProviderConfig{
		Id:        "wavespeed",
		BaseURL:   "http://127.0.0.1:59999", // Unreachable server
		SecretEnv: "NON_EXISTENT_KEY",
	}

	now := time.Now().Unix()

	// A. Fresh verified price: within 7 days -> returns 1.20x multiplier
	freshRoute := &model.StudioModelRoute{
		Id:               "ws-fresh",
		ProviderModelId:  "wavespeed-ai/flux-schnell",
		PricingStrategy:  "DYNAMIC_API",
		EffectiveCostUSD: 0.003,
		PriceVerifiedAt:  now - 3600, // 1 hour ago
	}
	fallbackCost, err := adapter.Quote(context.Background(), providerCfg, freshRoute, &NormalizedMediaInput{})
	require.NoError(t, err)
	assert.InDelta(t, 0.003*1.20, fallbackCost, 0.0001, "Fresh fallback must apply 1.20x safety multiplier")

	// B. Stale price: older than 7 days -> fails with ErrPriceUnavailable
	staleRoute := &model.StudioModelRoute{
		Id:               "ws-stale",
		ProviderModelId:  "wavespeed-ai/flux-schnell",
		PricingStrategy:  "DYNAMIC_API",
		EffectiveCostUSD: 0.003,
		PriceVerifiedAt:  now - (8 * 24 * 3600), // 8 days ago
	}
	_, err = adapter.Quote(context.Background(), providerCfg, staleRoute, &NormalizedMediaInput{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPriceUnavailable), "Stale route must return ErrPriceUnavailable")
}

// 5. Queue 2H Section 10, 11, 14: Router selection for image-generate FAST.
func TestStudio_Router_WaveSpeedFluxSchnellSelectionAndMargins(t *testing.T) {
	db := setupWaveSpeedAuditTestDB(t)

	// Set simulated WaveSpeed key for canary test
	os.Setenv("WAVESPEED_API_KEY", "test-canary-key")
	defer os.Unsetenv("WAVESPEED_API_KEY")

	// Ensure ws-flux-schnell is READY_FOR_CANARY in DB
	require.NoError(t, db.Model(&model.StudioModelRoute{}).Where("id = ?", "ws-flux-schnell").Updates(map[string]interface{}{
		"status":            model.RouteStatusReadyForCanary,
		"enabled":           true,
		"price_source":      model.PriceSourceRemoteDynamic,
		"price_verified_at": time.Now().Unix(),
		"effective_cost_usd": 0.003,
	}).Error)

	registry := GetProtocolRegistry()
	registry.Register(NewWaveSpeedAdapter())

	router := NewStudioRouter(nil)
	retailQuota := 5000 // 5 Tora Credits = $0.010 USD sell price
	primary, primaryProvider, fallbacks, reason, candidates, err := router.SelectRoute(
		context.Background(),
		"image-generate",
		QualityTierFast,
		&NormalizedMediaInput{Prompt: "abstract geometry", AspectRatio: "1:1"},
		retailQuota,
	)

	require.NoError(t, err)
	require.NotNil(t, primary)
	assert.Equal(t, "ws-flux-schnell", primary.Id, "WaveSpeed Flux Schnell must be selected")
	assert.Equal(t, "wavespeed", primaryProvider.Id)
	assert.Contains(t, reason, "ws-flux-schnell")
	_ = fallbacks

	// Verify other candidate statuses and skip reasons
	var falCand, kieCand *RouteCandidateResult
	for i := range candidates {
		if candidates[i].Route.Id == "fal-flux-schnell" {
			falCand = &candidates[i]
		}
		if candidates[i].Route.Id == "kie-flux-schnell" {
			kieCand = &candidates[i]
		}
	}

	require.NotNil(t, falCand)
	assert.False(t, falCand.IsEligible)
	assert.Contains(t, falCand.SkipReason, "billing blocked", "fal must be skipped due to BILLING_BLOCKED")

	require.NotNil(t, kieCand)
	assert.False(t, kieCand.IsEligible)
	assert.Contains(t, kieCand.SkipReason, "draft", "kie must be skipped due to DRAFT status")

	// Verify Gross Margin
	sellUSD := float64(retailQuota) / common.QuotaPerUnit // 5000 / 500,000 = $0.010 USD
	costUSD := primary.EffectiveCostUSD                   // $0.003 USD
	grossMargin := ((sellUSD - costUSD) / sellUSD) * 100.0
	assert.InDelta(t, 70.0, grossMargin, 0.01, "Gross Margin must be exactly 70.0%")
	assert.True(t, grossMargin >= 60.0, "Gross margin must pass >= 60% floor")
}

// 6. Queue 2H Section 20: Asset pipeline local ingestion with SHA-256 and ownership.
func TestStudio_Asset_IngestOutputAssetFromURL(t *testing.T) {
	db := setupWaveSpeedAuditTestDB(t)

	// Mock remote provider asset server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0x00, 0x00, 0x0d})
	}))
	defer server.Close()

	asset, err := IngestOutputAssetFromURL(context.Background(), db, 42, "job_test_ingest_1", server.URL+"/output.png")
	require.NoError(t, err)
	require.NotNil(t, asset)

	assert.Equal(t, 42, asset.UserId)
	assert.Equal(t, "job_test_ingest_1", asset.JobId)
	assert.Equal(t, "output", asset.AssetType)
	assert.Equal(t, "image/png", asset.MIMEType)
	assert.NotEmpty(t, asset.SHA256)
	assert.NotEmpty(t, asset.StorageURL)

	// Verify file was saved locally
	filename := filepath.Base(asset.StorageURL)
	filePath := filepath.Join(GetStudioUploadDir(), filename)
	defer os.Remove(filePath)
	_, statErr := os.Stat(filePath)
	assert.NoError(t, statErr, "Asset file must exist on disk")
}
