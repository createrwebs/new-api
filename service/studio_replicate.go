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
	ReplicateDefaultBaseURL = "https://api.replicate.com/v1"
)

var (
	ErrReplicateUnconfigured = errors.New("replicate API token is not configured (OPERATOR_BLOCKED)")
)

// ReplicateProvider implements StudioProvider for Replicate generative endpoints.
type ReplicateProvider struct {
	client     *http.Client
	baseURL    string
	apiToken   string
	costLedger map[string]float64
}

func NewReplicateProvider() *ReplicateProvider {
	apiToken := strings.TrimSpace(os.Getenv("REPLICATE_API_TOKEN"))
	if apiToken == "" {
		apiToken = strings.TrimSpace(os.Getenv("REPLICATE_KEY"))
	}

	return &ReplicateProvider{
		client:   &http.Client{Timeout: 30 * time.Second},
		baseURL:  ReplicateDefaultBaseURL,
		apiToken: apiToken,
		costLedger: map[string]float64{
			"nightmareai/real-esrgan":         0.005,
			"cjwbw/rembg":                     0.003,
			"black-forest-labs/flux-schnell": 0.003,
			"black-forest-labs/flux-dev":     0.025,
			"wan-video/wan-2.1-1.3b":          0.060,
		},
	}
}

func (r *ReplicateProvider) Name() string {
	return "replicate"
}

func (r *ReplicateProvider) ValidateConfiguration() error {
	if r.apiToken == "" {
		return ErrReplicateUnconfigured
	}
	return nil
}

func (r *ReplicateProvider) ResolveModelEndpoint(toolId string, inputParams map[string]interface{}) (string, float64) {
	switch toolId {
	case "background-remove":
		return "cjwbw/rembg", r.costLedger["cjwbw/rembg"]
	case "image-upscale":
		return "nightmareai/real-esrgan", r.costLedger["nightmareai/real-esrgan"]
	case "image-generate":
		if q, ok := inputParams["quality"].(string); ok && (q == "hd" || q == "ultra") {
			return "black-forest-labs/flux-dev", r.costLedger["black-forest-labs/flux-dev"]
		}
		return "black-forest-labs/flux-schnell", r.costLedger["black-forest-labs/flux-schnell"]
	default:
		return "black-forest-labs/flux-schnell", 0.003
	}
}

func (r *ReplicateProvider) Submit(ctx context.Context, job *model.StudioToolJob) (*ProviderSubmitResult, error) {
	if err := r.ValidateConfiguration(); err != nil {
		return nil, err
	}

	var inputParams map[string]interface{}
	if job.InputParams != "" {
		_ = json.Unmarshal([]byte(job.InputParams), &inputParams)
	}

	modelSlug, _ := r.ResolveModelEndpoint(job.ToolId, inputParams)
	targetURL := fmt.Sprintf("%s/models/%s/predictions", r.baseURL, modelSlug)

	payload := map[string]interface{}{
		"input": inputParams,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling replicate payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed creating replicate request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		// Connection timeout or network error before submission
		return nil, fmt.Errorf("%w: %v", ErrProviderTransient, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: replicate authentication failed (status %d)", ErrProviderPermanent, resp.StatusCode)
	}

	if resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("%w: replicate payload rejected (status %d): %s", ErrProviderPermanent, resp.StatusCode, string(respBody))
	}

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: replicate server error (status %d)", ErrProviderTransient, resp.StatusCode)
	}

	var repResp struct {
		Id     string      `json:"id"`
		Status string      `json:"status"`
		Output interface{} `json:"output"`
		Error  string      `json:"error"`
	}

	if err := json.Unmarshal(respBody, &repResp); err != nil {
		return nil, fmt.Errorf("%w: malformed replicate response", ErrProviderAmbiguous)
	}

	if repResp.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrProviderPermanent, repResp.Error)
	}

	var outputURL string
	var outputURLs []string

	if repResp.Status == "succeeded" && repResp.Output != nil {
		switch out := repResp.Output.(type) {
		case string:
			outputURL = out
			outputURLs = []string{out}
		case []interface{}:
			for _, item := range out {
				if s, ok := item.(string); ok && s != "" {
					outputURLs = append(outputURLs, s)
				}
			}
			if len(outputURLs) > 0 {
				outputURL = outputURLs[0]
			}
		}
	}

	status := "processing"
	if repResp.Status == "succeeded" {
		status = "completed"
	} else if repResp.Status == "failed" || repResp.Status == "canceled" {
		status = "failed"
	}

	return &ProviderSubmitResult{
		ProviderJobId: repResp.Id,
		Status:        status,
		OutputURL:     outputURL,
		OutputURLs:    outputURLs,
		RawResponse:   string(respBody),
	}, nil
}

func (r *ReplicateProvider) Poll(ctx context.Context, providerJobId string) (*ProviderPollResult, error) {
	if err := r.ValidateConfiguration(); err != nil {
		return nil, err
	}

	pollURL := fmt.Sprintf("%s/predictions/%s", r.baseURL, providerJobId)
	req, err := http.NewRequestWithContext(ctx, "GET", pollURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+r.apiToken)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: polling request failed", ErrProviderTransient)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var repResp struct {
		Id     string      `json:"id"`
		Status string      `json:"status"` // "starting", "processing", "succeeded", "failed", "canceled"
		Output interface{} `json:"output"`
		Error  string      `json:"error"`
	}

	if err := json.Unmarshal(respBody, &repResp); err != nil {
		return nil, fmt.Errorf("%w: failed unmarshaling poll response", ErrProviderAmbiguous)
	}

	result := &ProviderPollResult{}

	switch repResp.Status {
	case "succeeded":
		result.Status = "completed"
		result.Progress = 100
		if repResp.Output != nil {
			switch out := repResp.Output.(type) {
			case string:
				result.OutputURL = out
				result.OutputURLs = []string{out}
			case []interface{}:
				for _, item := range out {
					if s, ok := item.(string); ok && s != "" {
						result.OutputURLs = append(result.OutputURLs, s)
					}
				}
				if len(result.OutputURLs) > 0 {
					result.OutputURL = result.OutputURLs[0]
				}
			}
		}

	case "failed":
		result.Status = "failed"
		result.ErrorMessage = repResp.Error
		if result.ErrorMessage == "" {
			result.ErrorMessage = "replicate prediction failed"
		}

	case "canceled":
		result.Status = "failed"
		result.ErrorMessage = "replicate prediction canceled"

	case "starting":
		result.Status = "queued"
		result.Progress = 10

	default: // "processing"
		result.Status = "processing"
		result.Progress = 50
	}

	return result, nil
}

func (r *ReplicateProvider) Cancel(ctx context.Context, providerJobId string) error {
	if err := r.ValidateConfiguration(); err != nil {
		return err
	}

	cancelURL := fmt.Sprintf("%s/predictions/%s/cancel", r.baseURL, providerJobId)
	req, err := http.NewRequestWithContext(ctx, "POST", cancelURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+r.apiToken)

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
