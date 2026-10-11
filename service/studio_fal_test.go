package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFalProvider_Unconfigured_ReturnsOperatorBlocked(t *testing.T) {
	provider := &FalProvider{
		apiKey: "",
	}
	err := provider.ValidateConfiguration()
	assert.ErrorIs(t, err, ErrFalUnconfigured)

	// Submit fails early without HTTP call
	job := &model.StudioToolJob{ToolId: "background-remove"}
	_, err = provider.Submit(context.Background(), job)
	assert.ErrorIs(t, err, ErrFalUnconfigured)
}

func TestFalProvider_EndpointResolution(t *testing.T) {
	provider := &FalProvider{apiKey: "test_key"}

	cases := []struct {
		toolId   string
		expected string
	}{
		{"background-remove", "fal-ai/birefnet"},
		{"image-upscale", "fal-ai/clarity-upscaler"},
		{"image-generate", "fal-ai/flux/schnell"},
		{"product-photo", "fal-ai/product-photography"},
		{"image-extend", "fal-ai/flux-fill"},
		{"object-erase", "fal-ai/flux/dev/inpainting"},
		{"image-to-video", "wan-video/wan-2.2"},
	}

	for _, tc := range cases {
		job := &model.StudioToolJob{ToolId: tc.toolId}
		assert.Equal(t, tc.expected, provider.resolveEndpoint(job))
	}
}

func TestFalProvider_SubmitAndPoll_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		assert.Equal(t, "Key valid_fal_mock_key", r.Header.Get("Authorization"))

		switch r.URL.Path {
		case "/fal-ai/birefnet":
			// Submit response
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"request_id": "req_fal_mock_123", "status": "IN_QUEUE"}`))

		case "/fal-ai/birefnet/requests/req_fal_mock_123/status":
			// Poll response: completed
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "COMPLETED", "response_url": ""}`))

		case "/fal-ai/birefnet/requests/req_fal_mock_123":
			// Fetch result
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"image": {"url": "https://fal.media/files/lion/transparent.png"}}`))

		case "/fal-ai/birefnet/requests/req_fal_mock_123/cancel":
			w.WriteHeader(http.StatusOK)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := &FalProvider{
		client:     server.Client(),
		baseURL:    server.URL,
		apiKey:     "valid_fal_mock_key",
		costLedger: map[string]float64{"fal-ai/birefnet": 0.005},
	}

	// 1. Submit
	job := &model.StudioToolJob{
		ToolId:      "background-remove",
		InputParams: `{"image_url": "https://example.com/input.png"}`,
	}
	submitRes, err := provider.Submit(context.Background(), job)
	require.NoError(t, err)
	assert.Equal(t, "fal-ai/birefnet:req_fal_mock_123", submitRes.ProviderJobId)
	assert.Equal(t, "queued", submitRes.Status)

	// 2. Poll
	pollRes, err := provider.Poll(context.Background(), submitRes.ProviderJobId)
	require.NoError(t, err)
	assert.Equal(t, "completed", pollRes.Status)
	assert.Equal(t, "https://fal.media/files/lion/transparent.png", pollRes.OutputURL)

	// 3. Cancel
	err = provider.Cancel(context.Background(), submitRes.ProviderJobId)
	assert.NoError(t, err)
}

func TestFalProvider_EstimateCost(t *testing.T) {
	provider := NewFalProvider()
	assert.Equal(t, 0.005, provider.EstimateCost("fal-ai/birefnet"))
	assert.Equal(t, 0.080, provider.EstimateCost("wan-video/wan-2.2"))
	assert.Equal(t, 0.003, provider.EstimateCost("fal-ai/flux/schnell"))
	// Unknown defaults to 0.020
	assert.Equal(t, 0.020, provider.EstimateCost("unknown-model"))
}
