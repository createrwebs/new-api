package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
)

const (
	FalDefaultBaseURL = "https://queue.fal.run"
)

var (
	ErrFalUnconfigured    = errors.New("fal.ai API key is not configured (OPERATOR_BLOCKED)")
	ErrFalInvalidEndpoint = errors.New("unsupported fal.ai model endpoint")
)

// FalProvider implements StudioProvider for fal.ai generative media endpoints.
type FalProvider struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	costLedger map[string]float64
}

func NewFalProvider() *FalProvider {
	apiKey := strings.TrimSpace(os.Getenv("FAL_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("FAL_API_KEY"))
	}

	return &FalProvider{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: FalDefaultBaseURL,
		apiKey:  apiKey,
		costLedger: map[string]float64{
			"fal-ai/flux/schnell":        0.003,
			"fal-ai/flux/dev":            0.025,
			"fal-ai/birefnet":            0.005,
			"fal-ai/clarity-upscaler":    0.015,
			"fal-ai/product-photography": 0.035,
			"fal-ai/flux-fill":           0.025,
			"fal-ai/flux/dev/inpainting": 0.020,
			"wan-video/wan-2.2":          0.080,
			"fal-ai/ltx-video":           0.030,
		},
	}
}

func (f *FalProvider) Name() string {
	return "fal"
}

// ValidateConfiguration checks if the provider is ready or blocked by operator.
func (f *FalProvider) ValidateConfiguration() error {
	if f.apiKey == "" {
		return ErrFalUnconfigured
	}
	return nil
}

// EstimateCost returns the expected provider COGS in USD for a given tool or model.
func (f *FalProvider) EstimateCost(modelEndpoint string) float64 {
	if cost, ok := f.costLedger[modelEndpoint]; ok {
		return cost
	}
	return 0.020 // fallback conservative default
}

func (f *FalProvider) resolveEndpoint(job *model.StudioToolJob) string {
	switch job.ToolId {
	case "background-remove":
		return "fal-ai/birefnet"
	case "image-upscale":
		return "fal-ai/clarity-upscaler"
	case "image-generate":
		return "fal-ai/flux/schnell"
	case "product-photo":
		return "fal-ai/product-photography"
	case "image-extend":
		return "fal-ai/flux-fill"
	case "object-erase":
		return "fal-ai/flux/dev/inpainting"
	case "image-to-video":
		return "wan-video/wan-2.2"
	default:
		return "fal-ai/flux/schnell"
	}
}

func (f *FalProvider) Submit(ctx context.Context, job *model.StudioToolJob) (*ProviderSubmitResult, error) {
	if err := f.ValidateConfiguration(); err != nil {
		return nil, err
	}

	endpoint := f.resolveEndpoint(job)
	submitURL := fmt.Sprintf("%s/%s", f.baseURL, endpoint)

	payloadBytes := []byte(job.InputParams)
	if len(payloadBytes) == 0 {
		payloadBytes = []byte("{}")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		if ctx.Err() != nil || strings.Contains(err.Error(), "timeout") {
			return nil, ErrProviderAmbiguous
		}
		return nil, ErrProviderTransient
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrFalUnconfigured
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrProviderTransient
	}

	if resp.StatusCode >= 500 {
		return nil, ErrProviderTransient
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: %s", ErrProviderPermanent, string(respBody))
	}

	var parsed struct {
		RequestID   string `json:"request_id"`
		Status      string `json:"status"`
		StatusURL   string `json:"status_url"`
		ResponseURL string `json:"response_url"`
		CancelURL   string `json:"cancel_url"`
	}
	_ = json.Unmarshal(respBody, &parsed)

	providerJobId := parsed.RequestID
	if providerJobId != "" {
		providerJobId = fmt.Sprintf("%s:%s", endpoint, parsed.RequestID)
	} else {
		providerJobId = fmt.Sprintf("fal_%d", time.Now().UnixNano())
	}

	status := "queued"
	if parsed.Status == "COMPLETED" || parsed.Status == "OK" {
		status = "completed"
	}

	return &ProviderSubmitResult{
		ProviderJobId: providerJobId,
		Status:        status,
		RawResponse:   string(respBody),
	}, nil
}

func (f *FalProvider) Poll(ctx context.Context, providerJobId string) (*ProviderPollResult, error) {
	if err := f.ValidateConfiguration(); err != nil {
		return nil, err
	}

	modelId := "fal-ai/birefnet"
	reqId := providerJobId
	if strings.Contains(providerJobId, ":") {
		parts := strings.SplitN(providerJobId, ":", 2)
		modelId = parts[0]
		reqId = parts[1]
	}

	// Status endpoint: GET https://queue.fal.run/{model_id}/requests/{request_id}/status
	statusURL := fmt.Sprintf("%s/%s/requests/%s/status", f.baseURL, modelId, reqId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, ErrProviderTransient
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status %d", ErrProviderPermanent, resp.StatusCode)
	}

	var statusResp struct {
		Status      string `json:"status"` // IN_QUEUE, IN_PROGRESS, COMPLETED, OK
		ResponseURL string `json:"response_url"`
		Error       string `json:"error"`
	}
	_ = json.Unmarshal(respBody, &statusResp)

	normalizedStatus := strings.ToUpper(statusResp.Status)
	switch normalizedStatus {
	case "COMPLETED", "OK":
		outputURL, allURLs := f.fetchOutputURLs(ctx, statusResp.ResponseURL, modelId, reqId)
		return &ProviderPollResult{
			Status:     "completed",
			Progress:   100,
			OutputURL:  outputURL,
			OutputURLs: allURLs,
		}, nil

	case "IN_PROGRESS":
		return &ProviderPollResult{
			Status:   "processing",
			Progress: 50,
		}, nil

	case "IN_QUEUE":
		return &ProviderPollResult{
			Status:   "queued",
			Progress: 10,
		}, nil

	default:
		if statusResp.Error != "" {
			return &ProviderPollResult{
				Status:       "failed",
				ErrorMessage: statusResp.Error,
			}, nil
		}
		return &ProviderPollResult{
			Status:   "processing",
			Progress: 30,
		}, nil
	}
}

func (f *FalProvider) fetchOutputURLs(ctx context.Context, responseURL string, modelId string, reqId string) (string, []string) {
	if responseURL == "" {
		responseURL = fmt.Sprintf("%s/%s/requests/%s", f.baseURL, modelId, reqId)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, responseURL, nil)
	if err != nil {
		return "", nil
	}
	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var out struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
		Image struct {
			URL string `json:"url"`
		} `json:"image"`
		Video struct {
			URL string `json:"video"`
		} `json:"video"`
		Payload struct {
			Image struct {
				URL string `json:"url"`
			} `json:"image"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
			Video struct {
				URL string `json:"video"`
			} `json:"video"`
		} `json:"payload"`
	}
	_ = json.Unmarshal(body, &out)

	var allURLs []string
	for _, img := range out.Payload.Images {
		if img.URL != "" {
			allURLs = append(allURLs, img.URL)
		}
	}
	if len(allURLs) == 0 {
		for _, img := range out.Images {
			if img.URL != "" {
				allURLs = append(allURLs, img.URL)
			}
		}
	}
	if len(allURLs) == 0 && out.Payload.Image.URL != "" {
		allURLs = append(allURLs, out.Payload.Image.URL)
	}
	if len(allURLs) == 0 && out.Image.URL != "" {
		allURLs = append(allURLs, out.Image.URL)
	}
	if len(allURLs) == 0 && out.Payload.Video.URL != "" {
		allURLs = append(allURLs, out.Payload.Video.URL)
	}
	if len(allURLs) == 0 && out.Video.URL != "" {
		allURLs = append(allURLs, out.Video.URL)
	}

	primary := ""
	if len(allURLs) > 0 {
		primary = allURLs[0]
	}
	return primary, allURLs
}

func (f *FalProvider) Cancel(ctx context.Context, providerJobId string) error {
	if err := f.ValidateConfiguration(); err != nil {
		return err
	}

	modelId := "fal-ai/birefnet"
	reqId := providerJobId
	if strings.Contains(providerJobId, ":") {
		parts := strings.SplitN(providerJobId, ":", 2)
		modelId = parts[0]
		reqId = parts[1]
	}

	cancelURL := fmt.Sprintf("%s/%s/requests/%s/cancel", f.baseURL, modelId, reqId)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, cancelURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
