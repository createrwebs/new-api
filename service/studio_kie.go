package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// KieAdapter implements MediaProtocolAdapter for KIE.ai Jobs v1 API (Sections 15-19).
type KieAdapter struct {
	client *http.Client
}

func NewKieAdapter() *KieAdapter {
	return &KieAdapter{
		client: &http.Client{Timeout: 35 * time.Second},
	}
}

func (k *KieAdapter) Protocol() string {
	return model.ProtocolKieJobsV1
}

func (k *KieAdapter) getAPIKey(provider *model.StudioProviderConfig) string {
	envName := provider.SecretEnv
	if envName == "" {
		envName = "KIE_API_KEY"
	}
	return strings.TrimSpace(os.Getenv(envName))
}

func (k *KieAdapter) getBaseURL(provider *model.StudioProviderConfig) string {
	base := strings.TrimRight(provider.BaseURL, "/")
	if base == "" {
		base = "https://api.kie.ai"
	}
	return base
}

func (k *KieAdapter) ValidateConfiguration(provider *model.StudioProviderConfig) error {
	if k.getAPIKey(provider) == "" {
		return ErrProviderUnconfigured
	}
	return nil
}

// Probe checks API connectivity with KIE.ai (Section 15).
func (k *KieAdapter) Probe(ctx context.Context, provider *model.StudioProviderConfig) (string, error) {
	key := k.getAPIKey(provider)
	if key == "" {
		return model.ProviderHealthDisabled, ErrProviderUnconfigured
	}

	url := fmt.Sprintf("%s/api/v1/jobs/recordInfo?taskId=probe_probe", k.getBaseURL(provider))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return model.ProviderHealthDisabled, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := k.client.Do(req)
	if err != nil {
		return model.ProviderHealthDegraded, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(strings.ToLower(string(body)), "balance") || strings.Contains(strings.ToLower(string(body)), "credit") {
			return model.ProviderHealthBillingBlocked, ErrProviderAccountLocked
		}
		return model.ProviderHealthAuthFailed, ErrProviderUnconfigured
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		return model.ProviderHealthActive, nil
	}

	return model.ProviderHealthDegraded, nil
}

// Quote returns the configured effective cost for the model route.
func (k *KieAdapter) Quote(
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

// Submit creates a task via POST /api/v1/jobs/createTask (Section 16).
func (k *KieAdapter) Submit(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (*NormalizedMediaOutput, error) {
	key := k.getAPIKey(provider)
	if key == "" {
		return nil, ErrProviderUnconfigured
	}

	mappedInput, err := ApplyParameterMapping(input, route.InputMapping)
	if err != nil {
		return nil, fmt.Errorf("failed mapping KIE inputs: %w", err)
	}

	taskPayload := map[string]interface{}{
		"model": route.ProviderModelId,
		"input": mappedInput,
	}

	if input.CallbackURL != "" {
		taskPayload["callBackUrl"] = input.CallbackURL
	}

	bodyBytes, err := json.Marshal(taskPayload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling KIE payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/jobs/createTask", k.getBaseURL(provider))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		if ctx.Err() != nil || strings.Contains(err.Error(), "timeout") {
			return nil, ErrProviderAmbiguous
		}
		return nil, ErrProviderTransient
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		bodyStr := string(respBody)
		if strings.Contains(strings.ToLower(bodyStr), "balance") || strings.Contains(strings.ToLower(bodyStr), "credit") || strings.Contains(strings.ToLower(bodyStr), "lock") {
			return nil, ErrProviderAccountLocked
		}
		return nil, ErrProviderUnconfigured
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
		Code int `json:"code"`
		Data struct {
			TaskId string `json:"taskId"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("malformed KIE response: %w", err)
	}

	if parsed.Data.TaskId == "" {
		return nil, fmt.Errorf("missing taskId in KIE response: %s", string(respBody))
	}

	return &NormalizedMediaOutput{
		ProviderJobID: parsed.Data.TaskId,
		Status:        "queued",
		RawResponse:   string(respBody),
		CostEstimated: route.EffectiveCostUSD,
	}, nil
}

// Status queries /api/v1/jobs/recordInfo?taskId=... (Section 15).
func (k *KieAdapter) Status(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) (*NormalizedMediaOutput, error) {
	key := k.getAPIKey(provider)
	if key == "" {
		return nil, ErrProviderUnconfigured
	}

	url := fmt.Sprintf("%s/api/v1/jobs/recordInfo?taskId=%s", k.getBaseURL(provider), providerJobId)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, ErrProviderTransient
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status %d", ErrProviderPermanent, resp.StatusCode)
	}

	var parsed struct {
		Code int `json:"code"`
		Data struct {
			TaskId   string      `json:"taskId"`
			State    string      `json:"state"` // "waiting", "queuing", "generating", "success", "fail"
			Result   interface{} `json:"result"`
			ErrorMsg string      `json:"errorMsg"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("malformed KIE record response: %w", err)
	}

	status := "processing"
	switch strings.ToLower(parsed.Data.State) {
	case "success", "completed":
		status = "completed"
	case "fail", "failed", "error":
		status = "failed"
	case "waiting", "queuing":
		status = "queued"
	case "generating":
		status = "processing"
	}

	out := &NormalizedMediaOutput{
		ProviderJobID: providerJobId,
		Status:        status,
		RawResponse:   string(respBody),
		ErrorMessage:  parsed.Data.ErrorMsg,
		CostEstimated: route.EffectiveCostUSD,
	}

	// Parse result items (can be array of URLs or array of objects with "url")
	if arr, ok := parsed.Data.Result.([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok && s != "" {
				out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: s})
			} else if obj, ok := item.(map[string]interface{}); ok {
				if u, ok := obj["url"].(string); ok && u != "" {
					out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: u})
				}
			}
		}
	} else if obj, ok := parsed.Data.Result.(map[string]interface{}); ok {
		if u, ok := obj["url"].(string); ok && u != "" {
			out.Assets = append(out.Assets, NormalizedAsset{Type: "image", URL: u})
		}
	}

	return out, nil
}

// Cancel terminates a running KIE task.
func (k *KieAdapter) Cancel(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) error {
	return nil // KIE currently auto-expires unfinished jobs
}

// FetchBalance placeholder for KIE.
func (k *KieAdapter) FetchBalance(
	ctx context.Context,
	provider *model.StudioProviderConfig,
) (float64, string, error) {
	return 0, "USD", nil
}
