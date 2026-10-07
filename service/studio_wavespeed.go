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

// WaveSpeedAdapter implements MediaProtocolAdapter for WaveSpeed AI v3 API (Sections 9-14).
type WaveSpeedAdapter struct {
	client *http.Client
}

func NewWaveSpeedAdapter() *WaveSpeedAdapter {
	return &WaveSpeedAdapter{
		client: &http.Client{Timeout: 35 * time.Second},
	}
}

func (a *WaveSpeedAdapter) Protocol() string {
	return model.ProtocolWaveSpeedV3
}

func (a *WaveSpeedAdapter) getAPIKey(provider *model.StudioProviderConfig) string {
	envName := provider.SecretEnv
	if envName == "" {
		envName = "WAVESPEED_API_KEY"
	}
	return strings.TrimSpace(os.Getenv(envName))
}

func (a *WaveSpeedAdapter) getBaseURL(provider *model.StudioProviderConfig) string {
	base := strings.TrimRight(provider.BaseURL, "/")
	if base == "" {
		base = "https://api.wavespeed.ai/api/v3"
	}
	return base
}

func (a *WaveSpeedAdapter) ValidateConfiguration(provider *model.StudioProviderConfig) error {
	if a.getAPIKey(provider) == "" {
		return ErrProviderUnconfigured
	}
	return nil
}

// Probe checks connection and balance against WaveSpeed API (Section 9).
func (a *WaveSpeedAdapter) Probe(ctx context.Context, provider *model.StudioProviderConfig) (string, error) {
	key := a.getAPIKey(provider)
	if key == "" {
		return model.ProviderHealthDisabled, ErrProviderUnconfigured
	}

	url := fmt.Sprintf("%s/balance", a.getBaseURL(provider))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return model.ProviderHealthDisabled, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return model.ProviderHealthDegraded, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(strings.ToLower(string(body)), "balance") || strings.Contains(strings.ToLower(string(body)), "lock") {
			return model.ProviderHealthBillingBlocked, ErrProviderAccountLocked
		}
		return model.ProviderHealthAuthFailed, ErrProviderUnconfigured
	}

	if resp.StatusCode == http.StatusOK {
		return model.ProviderHealthActive, nil
	}

	return model.ProviderHealthDegraded, nil
}

// Quote evaluates provider pricing via WaveSpeed /model/price API with local caching (Section 10-12).
func (a *WaveSpeedAdapter) Quote(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (float64, error) {
	// 1. Check bounded price cache
	cacheKey := fmt.Sprintf("ws_price:%s:%s", route.ProviderModelId, input.AspectRatio)
	if cached, ok := GetPriceCache().Get(cacheKey); ok {
		return cached, nil
	}

	key := a.getAPIKey(provider)
	if key == "" || route.PricingStrategy != "DYNAMIC_API" {
		// Fallback to configured effective cost
		if route.EffectiveCostUSD > 0 {
			return route.EffectiveCostUSD, nil
		}
		return route.BaseCostUSD, nil
	}

	// 2. Query WaveSpeed dynamic price preview
	payloadMap, err := ApplyParameterMapping(input, route.InputMapping)
	if err != nil {
		payloadMap = make(map[string]interface{})
	}

	priceReqBody := map[string]interface{}{
		"model":  route.ProviderModelId,
		"inputs": payloadMap,
	}
	bodyBytes, _ := json.Marshal(priceReqBody)

	url := fmt.Sprintf("%s/model/price", a.getBaseURL(provider))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return route.EffectiveCostUSD, nil
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return route.EffectiveCostUSD, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return route.EffectiveCostUSD, nil
	}

	var priceResp struct {
		Code int `json:"code"`
		Data struct {
			BasePrice       float64 `json:"base_price"`
			DiscountedPrice float64 `json:"discounted_price"`
			DiscountRate    float64 `json:"discount_rate"`
			Currency        string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err == nil && priceResp.Data.DiscountedPrice > 0 {
		effectivePrice := priceResp.Data.DiscountedPrice
		// Cache for 5 minutes
		GetPriceCache().Set(cacheKey, effectivePrice, 5*time.Minute)
		return effectivePrice, nil
	}

	return route.EffectiveCostUSD, nil
}

// Submit launches an asynchronous task with WaveSpeed API (Section 13).
func (a *WaveSpeedAdapter) Submit(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	input *NormalizedMediaInput,
) (*NormalizedMediaOutput, error) {
	key := a.getAPIKey(provider)
	if key == "" {
		return nil, ErrProviderUnconfigured
	}

	payload, err := ApplyParameterMapping(input, route.InputMapping)
	if err != nil {
		return nil, fmt.Errorf("failed mapping input parameters: %w", err)
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s", a.getBaseURL(provider), strings.TrimPrefix(route.ProviderModelId, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
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
		if strings.Contains(strings.ToLower(bodyStr), "balance") || strings.Contains(strings.ToLower(bodyStr), "lock") {
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
			ID      string   `json:"id"`
			Status  string   `json:"status"` // "queued", "processing", "completed"
			Outputs []string `json:"outputs"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("malformed WaveSpeed response: %w", err)
	}

	if parsed.Data.ID == "" {
		return nil, fmt.Errorf("missing task ID in WaveSpeed response: %s", string(respBody))
	}

	status := "queued"
	if parsed.Data.Status == "completed" {
		status = "completed"
	} else if parsed.Data.Status == "processing" {
		status = "processing"
	}

	output := &NormalizedMediaOutput{
		ProviderJobID: parsed.Data.ID,
		Status:        status,
		RawResponse:   string(respBody),
		CostEstimated: route.EffectiveCostUSD,
	}

	for _, urlStr := range parsed.Data.Outputs {
		output.Assets = append(output.Assets, NormalizedAsset{
			Type: "image",
			URL:  urlStr,
		})
	}

	return output, nil
}

// Status queries predictions/{id}/result endpoint (Section 9).
func (a *WaveSpeedAdapter) Status(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) (*NormalizedMediaOutput, error) {
	key := a.getAPIKey(provider)
	if key == "" {
		return nil, ErrProviderUnconfigured
	}

	url := fmt.Sprintf("%s/predictions/%s/result", a.getBaseURL(provider), providerJobId)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, ErrProviderTransient
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: %s", ErrProviderPermanent, string(respBody))
	}

	var parsed struct {
		Code int `json:"code"`
		Data struct {
			ID      string   `json:"id"`
			Status  string   `json:"status"` // "queued", "processing", "completed", "failed"
			Outputs []string `json:"outputs"`
			Error   string   `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("malformed WaveSpeed result response: %w", err)
	}

	status := "processing"
	switch strings.ToLower(parsed.Data.Status) {
	case "completed", "succeeded", "success":
		status = "completed"
	case "failed", "error":
		status = "failed"
	case "queued", "waiting":
		status = "queued"
	}

	out := &NormalizedMediaOutput{
		ProviderJobID: providerJobId,
		Status:        status,
		RawResponse:   string(respBody),
		ErrorMessage:  parsed.Data.Error,
		CostEstimated: route.EffectiveCostUSD,
	}

	for _, urlStr := range parsed.Data.Outputs {
		out.Assets = append(out.Assets, NormalizedAsset{
			Type: "image",
			URL:  urlStr,
		})
	}

	return out, nil
}

// Cancel terminates a running prediction.
func (a *WaveSpeedAdapter) Cancel(
	ctx context.Context,
	provider *model.StudioProviderConfig,
	route *model.StudioModelRoute,
	providerJobId string,
) error {
	key := a.getAPIKey(provider)
	if key == "" {
		return nil
	}
	url := fmt.Sprintf("%s/predictions/%s/cancel", a.getBaseURL(provider), providerJobId)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	_, _ = a.client.Do(req)
	return nil
}

// FetchBalance queries account balance.
func (a *WaveSpeedAdapter) FetchBalance(
	ctx context.Context,
	provider *model.StudioProviderConfig,
) (float64, string, error) {
	key := a.getAPIKey(provider)
	if key == "" {
		return 0, "USD", ErrProviderUnconfigured
	}
	url := fmt.Sprintf("%s/balance", a.getBaseURL(provider))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "USD", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := a.client.Do(req)
	if err != nil {
		return 0, "USD", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "USD", fmt.Errorf("failed fetching balance: status %d", resp.StatusCode)
	}

	var balanceResp struct {
		Code int `json:"code"`
		Data struct {
			Balance  float64 `json:"balance"`
			Currency string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return 0, "USD", err
	}

	currency := balanceResp.Data.Currency
	if currency == "" {
		currency = "USD"
	}
	return balanceResp.Data.Balance, currency, nil
}
