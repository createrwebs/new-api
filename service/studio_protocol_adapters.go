package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// FalQueueAdapter integrates Fal.ai under the generic MediaProtocolAdapter interface (Section 35).
type FalQueueAdapter struct {
	fal *FalProvider
}

func NewFalQueueAdapter(fal *FalProvider) *FalQueueAdapter {
	if fal == nil {
		fal = NewFalProvider()
	}
	return &FalQueueAdapter{fal: fal}
}

func (a *FalQueueAdapter) Protocol() string {
	return model.ProtocolFalQueue
}

func (a *FalQueueAdapter) ValidateConfiguration(provider *model.StudioProviderConfig) error {
	return a.fal.ValidateConfiguration()
}

func (a *FalQueueAdapter) Probe(ctx context.Context, provider *model.StudioProviderConfig) (string, error) {
	status, err := a.fal.ProbeConnection(ctx)
	if err != nil {
		return model.ProviderHealthDisabled, err
	}
	if status == "FAL_CONNECTED" {
		return model.ProviderHealthActive, nil
	}
	return model.ProviderHealthBillingBlocked, nil
}

func (a *FalQueueAdapter) Quote(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (float64, error) {
	if route.EffectiveCostUSD > 0 {
		return route.EffectiveCostUSD, nil
	}
	return a.fal.EstimateCost(route.ProviderModelId), nil
}

func (a *FalQueueAdapter) Submit(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (*NormalizedMediaOutput, error) {
	inputParams, err := ApplyParameterMapping(input, route.InputMapping)
	if err != nil {
		return nil, err
	}

	dummyJob := &model.StudioToolJob{
		ToolId:   route.LogicalTool,
		InputParams: func() string {
			// json encode dummy
			return ""
		}(),
	}
	_ = dummyJob

	// Use FalProvider native Submit
	job := &model.StudioToolJob{
		ToolId: route.LogicalTool,
	}
	// Inject mapped params into job for FalProvider
	if url, ok := inputParams["image_url"].(string); ok {
		job.InputParams = fmt.Sprintf(`{"image_url":"%s"}`, url)
	}

	res, err := a.fal.Submit(ctx, job)
	if err != nil {
		if strings.Contains(err.Error(), "balance") || strings.Contains(err.Error(), "locked") {
			return nil, ErrProviderAccountLocked
		}
		return nil, err
	}

	out := &NormalizedMediaOutput{
		ProviderJobID: res.ProviderJobId,
		Status:        res.Status,
		RawResponse:   res.RawResponse,
		CostEstimated: route.EffectiveCostUSD,
	}
	if res.OutputURL != "" {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: res.OutputURL})
	}
	for _, u := range res.OutputURLs {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: u})
	}

	return out, nil
}

func (a *FalQueueAdapter) Status(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) (*NormalizedMediaOutput, error) {
	res, err := a.fal.Poll(ctx, providerJobId)
	if err != nil {
		return nil, err
	}

	out := &NormalizedMediaOutput{
		ProviderJobID: providerJobId,
		Status:        res.Status,
		ErrorMessage:  res.ErrorMessage,
		CostEstimated: route.EffectiveCostUSD,
	}
	if res.OutputURL != "" {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: res.OutputURL})
	}
	for _, u := range res.OutputURLs {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: u})
	}
	return out, nil
}

func (a *FalQueueAdapter) Cancel(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) error {
	return a.fal.Cancel(ctx, providerJobId)
}

func (a *FalQueueAdapter) FetchBalance(
	ctx context.Context,
	provider *model.StudioProviderConfig,
) (float64, string, error) {
	return 0, "USD", nil
}

// ReplicatePredictionsAdapter wraps ReplicateProvider.
type ReplicatePredictionsAdapter struct {
	replicate *ReplicateProvider
}

func NewReplicatePredictionsAdapter(rep *ReplicateProvider) *ReplicatePredictionsAdapter {
	if rep == nil {
		rep = NewReplicateProvider()
	}
	return &ReplicatePredictionsAdapter{replicate: rep}
}

func (a *ReplicatePredictionsAdapter) Protocol() string {
	return model.ProtocolReplicatePredictions
}

func (a *ReplicatePredictionsAdapter) ValidateConfiguration(provider *model.StudioProviderConfig) error {
	key := os.Getenv("REPLICATE_API_TOKEN")
	if key == "" {
		return ErrProviderUnconfigured
	}
	return nil
}

func (a *ReplicatePredictionsAdapter) Probe(ctx context.Context, provider *model.StudioProviderConfig) (string, error) {
	key := os.Getenv("REPLICATE_API_TOKEN")
	if key == "" {
		return model.ProviderHealthDisabled, ErrProviderUnconfigured
	}
	return model.ProviderHealthActive, nil
}

func (a *ReplicatePredictionsAdapter) Quote(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (float64, error) {
	if route.EffectiveCostUSD > 0 {
		return route.EffectiveCostUSD, nil
	}
	return route.BaseCostUSD, nil
}

func (a *ReplicatePredictionsAdapter) Submit(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (*NormalizedMediaOutput, error) {
	job := &model.StudioToolJob{
		ToolId: route.LogicalTool,
	}
	res, err := a.replicate.Submit(ctx, job)
	if err != nil {
		return nil, err
	}
	out := &NormalizedMediaOutput{
		ProviderJobID: res.ProviderJobId,
		Status:        res.Status,
		RawResponse:   res.RawResponse,
		CostEstimated: route.EffectiveCostUSD,
	}
	if res.OutputURL != "" {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: res.OutputURL})
	}
	return out, nil
}

func (a *ReplicatePredictionsAdapter) Status(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) (*NormalizedMediaOutput, error) {
	res, err := a.replicate.Poll(ctx, providerJobId)
	if err != nil {
		return nil, err
	}
	out := &NormalizedMediaOutput{
		ProviderJobID: providerJobId,
		Status:        res.Status,
		ErrorMessage:  res.ErrorMessage,
		CostEstimated: route.EffectiveCostUSD,
	}
	if res.OutputURL != "" {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: res.OutputURL})
	}
	return out, nil
}

func (a *ReplicatePredictionsAdapter) Cancel(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) error {
	return a.replicate.Cancel(ctx, providerJobId)
}

func (a *ReplicatePredictionsAdapter) FetchBalance(
	ctx context.Context,
	provider *model.StudioProviderConfig,
) (float64, string, error) {
	return 0, "USD", nil
}

// MockProtocolAdapter implements MediaProtocolAdapter for deterministic testing.
type MockProtocolAdapter struct {
	mock *DeterministicMockProvider
}

func NewMockProtocolAdapter(mock *DeterministicMockProvider) *MockProtocolAdapter {
	if mock == nil {
		mock = NewDeterministicMockProvider(MockModeInstantSuccess)
	}
	return &MockProtocolAdapter{mock: mock}
}

func (a *MockProtocolAdapter) Protocol() string {
	return model.ProtocolMock
}

func (a *MockProtocolAdapter) ValidateConfiguration(provider *model.StudioProviderConfig) error {
	return nil
}

func (a *MockProtocolAdapter) Probe(ctx context.Context, provider *model.StudioProviderConfig) (string, error) {
	return model.ProviderHealthActive, nil
}

func (a *MockProtocolAdapter) Quote(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (float64, error) {
	return route.EffectiveCostUSD, nil
}

func (a *MockProtocolAdapter) Submit(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (*NormalizedMediaOutput, error) {
	job := &model.StudioToolJob{
		ToolId: route.LogicalTool,
	}
	res, err := a.mock.Submit(ctx, job)
	if err != nil {
		return nil, err
	}
	out := &NormalizedMediaOutput{
		ProviderJobID: res.ProviderJobId,
		Status:        res.Status,
		CostEstimated: route.EffectiveCostUSD,
	}
	if res.OutputURL != "" {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: res.OutputURL})
	}
	return out, nil
}

func (a *MockProtocolAdapter) Status(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) (*NormalizedMediaOutput, error) {
	res, err := a.mock.Poll(ctx, providerJobId)
	if err != nil {
		return nil, err
	}
	out := &NormalizedMediaOutput{
		ProviderJobID: providerJobId,
		Status:        res.Status,
		ErrorMessage:  res.ErrorMessage,
		CostEstimated: route.EffectiveCostUSD,
	}
	if res.OutputURL != "" {
		out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: res.OutputURL})
	}
	return out, nil
}

func (a *MockProtocolAdapter) Cancel(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) error {
	return nil
}

func (a *MockProtocolAdapter) FetchBalance(
	ctx context.Context,
	provider *model.StudioProviderConfig,
) (float64, string, error) {
	return 9999.0, "USD", nil
}
