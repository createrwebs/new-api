package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFailingProvider simulates different provider failure modes
type mockFailingProvider struct {
	name         string
	submitErr    error
	submitResult *ProviderSubmitResult
	submitCalls  int
}

func (m *mockFailingProvider) Name() string { return m.name }
func (m *mockFailingProvider) Submit(ctx context.Context, job *model.StudioToolJob) (*ProviderSubmitResult, error) {
	m.submitCalls++
	if m.submitErr != nil {
		return nil, m.submitErr
	}
	return m.submitResult, nil
}
func (m *mockFailingProvider) Poll(ctx context.Context, providerJobId string) (*ProviderPollResult, error) {
	return &ProviderPollResult{Status: "completed"}, nil
}
func (m *mockFailingProvider) Cancel(ctx context.Context, providerJobId string) error {
	return nil
}
func (m *mockFailingProvider) ValidateConfiguration() error {
	return nil
}

func TestQueue7_QualityTier_RouteSelection(t *testing.T) {
	providers := map[string]StudioProvider{
		"fal":       NewDeterministicMockProvider(MockModeInstantSuccess),
		"replicate": NewDeterministicMockProvider(MockModeInstantSuccess),
	}
	router := NewStudioRouter(providers)

	// 1. Background Remove
	// FAST -> Replicate (cheaper $0.003), fallback Fal ($0.005)
	primary, fallbacks, err := router.SelectProvider("background-remove", QualityTierFast, nil)
	require.NoError(t, err)
	assert.Equal(t, "replicate", primary)
	assert.Contains(t, fallbacks, "fal")

	// QUALITY -> Fal (higher precision BiRefNet $0.005), fallback Replicate ($0.003)
	primary, fallbacks, err = router.SelectProvider("background-remove", QualityTierQuality, nil)
	require.NoError(t, err)
	assert.Equal(t, "fal", primary)
	assert.Contains(t, fallbacks, "replicate")

	// 2. Image Upscale
	// FAST -> Replicate (Real-ESRGAN $0.005)
	primary, fallbacks, err = router.SelectProvider("image-upscale", QualityTierFast, nil)
	require.NoError(t, err)
	assert.Equal(t, "replicate", primary)
	assert.Contains(t, fallbacks, "fal")

	// PREMIUM -> Fal (Clarity Upscaler $0.015)
	primary, fallbacks, err = router.SelectProvider("image-upscale", QualityTierPremium, nil)
	require.NoError(t, err)
	assert.Equal(t, "fal", primary)
	assert.Contains(t, fallbacks, "replicate")
}

func TestQueue7_SafeFallback_ConnectionRefused(t *testing.T) {
	// Primary fails immediately before upstream processing begins (e.g. dial tcp connection refused)
	primaryMock := &mockFailingProvider{
		name:      "fal",
		submitErr: errors.New("dial tcp 127.0.0.1:443: connect: connection refused"),
	}
	fallbackMock := &mockFailingProvider{
		name: "replicate",
		submitResult: &ProviderSubmitResult{
			ProviderJobId: "rep_safe_fallback_123",
			Status:        "completed",
		},
	}

	providers := map[string]StudioProvider{
		"fal":       primaryMock,
		"replicate": fallbackMock,
	}
	router := NewStudioRouter(providers)

	job := &model.StudioToolJob{
		Id:           "job_safe_fallback_test",
		ToolId:       "background-remove",
		ProviderName: "fal",
	}

	res, chosenProvider, err := router.ExecuteWithSafeFallback(context.Background(), job, "fal", []string{"replicate"})
	require.NoError(t, err)
	assert.Equal(t, "replicate", chosenProvider)
	assert.Equal(t, "replicate", job.ProviderName)
	assert.NotNil(t, res)
	assert.Equal(t, "rep_safe_fallback_123", res.ProviderJobId)

	assert.Equal(t, 1, primaryMock.submitCalls, "Primary should have been attempted once")
	assert.Equal(t, 1, fallbackMock.submitCalls, "Fallback should have taken over and succeeded")
}

func TestQueue7_AmbiguousSubmission_BlocksFallback(t *testing.T) {
	// Primary timed out or encountered ambiguous status (bytes may have been sent)
	primaryMock := &mockFailingProvider{
		name:      "fal",
		submitErr: fmt.Errorf("timeout awaiting upstream response: %w", ErrProviderAmbiguous),
	}
	fallbackMock := &mockFailingProvider{
		name: "replicate",
		submitResult: &ProviderSubmitResult{
			ProviderJobId: "rep_should_never_run",
			Status:        "completed",
		},
	}

	providers := map[string]StudioProvider{
		"fal":       primaryMock,
		"replicate": fallbackMock,
	}
	router := NewStudioRouter(providers)

	job := &model.StudioToolJob{
		Id:           "job_ambiguous_test",
		ToolId:       "image-upscale",
		ProviderName: "fal",
	}

	res, chosenProvider, err := router.ExecuteWithSafeFallback(context.Background(), job, "fal", []string{"replicate"})
	// INVARIANT: Fallback MUST NOT execute on ambiguous upstream error to avoid double-spend!
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProviderAmbiguous))
	assert.Nil(t, res)
	assert.Equal(t, "fal", chosenProvider)
	assert.Equal(t, 1, primaryMock.submitCalls)
	assert.Equal(t, 0, fallbackMock.submitCalls, "CRITICAL: Fallback must NOT be called on ambiguous submission!")
}

func TestQueue7_PermanentError_BlocksFallback(t *testing.T) {
	// Content policy or permanent parameter error
	primaryMock := &mockFailingProvider{
		name:      "fal",
		submitErr: fmt.Errorf("prompt contains blocked keywords: %w", ErrProviderPermanent),
	}
	fallbackMock := &mockFailingProvider{
		name: "replicate",
		submitResult: &ProviderSubmitResult{
			ProviderJobId: "rep_should_never_run",
			Status:        "completed",
		},
	}

	providers := map[string]StudioProvider{
		"fal":       primaryMock,
		"replicate": fallbackMock,
	}
	router := NewStudioRouter(providers)

	job := &model.StudioToolJob{
		Id:           "job_perm_test",
		ToolId:       "image-generate",
		ProviderName: "fal",
	}

	res, _, err := router.ExecuteWithSafeFallback(context.Background(), job, "fal", []string{"replicate"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProviderPermanent))
	assert.Nil(t, res)
	assert.Equal(t, 0, fallbackMock.submitCalls, "Permanent error must not fallback to alternative provider")
}

func TestQueue7_RoutingAnalytics(t *testing.T) {
	_ = setupTestDBForStudio(t)

	// Seed cost snapshots for Fal and Replicate
	now := time.Now().Unix()
	s1 := model.StudioCostSnapshot{
		JobId:          "job_analytics_1",
		ToolId:         "background-remove",
		ProviderName:   "fal",
		CostUSD:        0.005,
		ToraRevenueUSD: 0.020, // 10 Credits = $0.020
		SnapshotAt:     now,
	}
	s2 := model.StudioCostSnapshot{
		JobId:          "job_analytics_2",
		ToolId:         "background-remove",
		ProviderName:   "replicate",
		CostUSD:        0.003,
		ToraRevenueUSD: 0.020, // 10 Credits = $0.020
		SnapshotAt:     now,
	}

	require.NoError(t, model.DB.Create(&s1).Error)
	require.NoError(t, model.DB.Create(&s2).Error)

	router := NewStudioRouter(map[string]StudioProvider{})
	analytics, err := router.GetRoutingAnalytics()
	require.NoError(t, err)
	assert.NotEmpty(t, analytics)

	for _, a := range analytics {
		if a.ToolID == "background-remove" {
			if a.ProviderName == "fal" {
				assert.Equal(t, float64(0.005), a.TotalActualCost)
				assert.Equal(t, float64(0.020), a.TotalRevenueUSD)
				// Margin = (0.020 - 0.005) / 0.020 = 75%
				assert.InDelta(t, 75.0, a.GrossMarginPercent, 0.1)
			} else if a.ProviderName == "replicate" {
				assert.Equal(t, float64(0.003), a.TotalActualCost)
				assert.Equal(t, float64(0.020), a.TotalRevenueUSD)
				// Margin = (0.020 - 0.003) / 0.020 = 85%
				assert.InDelta(t, 85.0, a.GrossMarginPercent, 0.1)
			}
		}
	}
}
