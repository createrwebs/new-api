package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

var (
	ErrProtocolNotSupported = errors.New("media protocol is not supported")
	ErrProviderUnconfigured = errors.New("media provider credential not configured")
	ErrProviderAccountLocked = errors.New("upstream provider account locked or balance exhausted")
	ErrInvalidModelRoute    = errors.New("invalid or disabled model route")
	ErrPriceUnavailable     = errors.New("upstream provider price unavailable")
)

// NormalizedMediaInput represents a provider-neutral generation/editing request (Section 6).
type NormalizedMediaInput struct {
	Prompt          string                 `json:"prompt,omitempty"`
	NegativePrompt  string                 `json:"negative_prompt,omitempty"`
	InputAssets     []string               `json:"input_assets,omitempty"`
	MaskAsset       string                 `json:"mask_asset,omitempty"`
	ReferenceAssets []string               `json:"reference_assets,omitempty"`
	AspectRatio     string                 `json:"aspect_ratio,omitempty"`
	Width           int                    `json:"width,omitempty"`
	Height          int                    `json:"height,omitempty"`
	Resolution      string                 `json:"resolution,omitempty"`
	Duration        int                    `json:"duration,omitempty"`
	FPS             int                    `json:"fps,omitempty"`
	Quality         string                 `json:"quality,omitempty"`
	NumberOfOutputs int                    `json:"number_of_outputs,omitempty"`
	Audio           bool                   `json:"audio,omitempty"`
	Seed            int64                  `json:"seed,omitempty"`
	Strength        float64                `json:"strength,omitempty"`
	QualityTier     string                 `json:"quality_tier,omitempty"`
	CallbackURL     string                 `json:"callback_url,omitempty"`
	AdvancedParams  map[string]interface{} `json:"advanced_params,omitempty"`
}

// NormalizedFromMap populates NormalizedMediaInput from generic map parameters.
func NormalizedFromMap(params map[string]interface{}) *NormalizedMediaInput {
	input := &NormalizedMediaInput{
		AdvancedParams: make(map[string]interface{}),
	}
	if params == nil {
		return input
	}

	for k, v := range params {
		switch strings.ToLower(k) {
		case "prompt":
			if s, ok := v.(string); ok {
				input.Prompt = s
			}
		case "negative_prompt":
			if s, ok := v.(string); ok {
				input.NegativePrompt = s
			}
		case "image_url", "image", "input_url":
			if s, ok := v.(string); ok && s != "" {
				input.InputAssets = []string{s}
			}
		case "input_assets":
			if arr, ok := v.([]string); ok {
				input.InputAssets = arr
			} else if arrInt, ok := v.([]interface{}); ok {
				for _, item := range arrInt {
					if s, ok := item.(string); ok && s != "" {
						input.InputAssets = append(input.InputAssets, s)
					}
				}
			}
		case "mask_url", "mask_asset":
			if s, ok := v.(string); ok {
				input.MaskAsset = s
			}
		case "reference_image_url", "reference_url":
			if s, ok := v.(string); ok && s != "" {
				input.ReferenceAssets = append(input.ReferenceAssets, s)
			}
		case "aspect_ratio":
			if s, ok := v.(string); ok {
				input.AspectRatio = s
			}
		case "width":
			if n, ok := toInt(v); ok {
				input.Width = n
			}
		case "height":
			if n, ok := toInt(v); ok {
				input.Height = n
			}
		case "resolution":
			if s, ok := v.(string); ok {
				input.Resolution = s
			}
		case "duration", "duration_sec":
			if n, ok := toInt(v); ok {
				input.Duration = n
			}
		case "fps":
			if n, ok := toInt(v); ok {
				input.FPS = n
			}
		case "number_of_outputs", "num_outputs", "pack_size":
			if n, ok := toInt(v); ok {
				input.NumberOfOutputs = n
			}
		case "seed":
			if n, ok := toInt64(v); ok {
				input.Seed = n
			}
		case "quality_tier":
			if s, ok := v.(string); ok {
				input.QualityTier = strings.ToUpper(s)
			}
		case "callback_url":
			if s, ok := v.(string); ok {
				input.CallbackURL = s
			}
		default:
			input.AdvancedParams[k] = v
		}
	}

	if input.NumberOfOutputs <= 0 {
		input.NumberOfOutputs = 1
	}

	return input
}

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	}
	return 0, false
}

func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int64:
		return val, true
	case float64:
		return int64(val), true
	}
	return 0, false
}

// NormalizedAsset represents an individual output file.
type NormalizedAsset struct {
	Type     string  `json:"type"` // "image", "video", "audio"
	URL      string  `json:"url"`
	MIME     string  `json:"mime,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

// NormalizedMediaOutput represents a canonical generation/task result (Section 8).
type NormalizedMediaOutput struct {
	ProviderJobID        string                 `json:"provider_job_id"`
	Status               string                 `json:"status"` // "queued", "processing", "completed", "failed"
	Assets               []NormalizedAsset      `json:"assets"`
	CostEstimated        float64                `json:"cost_estimated"`
	CostActual           *float64               `json:"cost_actual,omitempty"`
	ProviderMetadataSafe map[string]interface{} `json:"provider_metadata_safe,omitempty"`
	RawResponse          string                 `json:"raw_response,omitempty"`
	ErrorMessage         string                 `json:"error_message,omitempty"`
}

// PrimaryOutputURL helper returns the first asset URL or empty string.
func (o *NormalizedMediaOutput) PrimaryOutputURL() string {
	if len(o.Assets) > 0 {
		return o.Assets[0].URL
	}
	return ""
}

// AllOutputURLs helper returns all asset URLs.
func (o *NormalizedMediaOutput) AllOutputURLs() []string {
	var urls []string
	for _, a := range o.Assets {
		if a.URL != "" {
			urls = append(urls, a.URL)
		}
	}
	return urls
}

// MediaProtocolAdapter defines the protocol-level interface (Section 5).
type MediaProtocolAdapter interface {
	Protocol() string
	ValidateConfiguration(provider *model.StudioProviderConfig) error
	Probe(ctx context.Context, provider *model.StudioProviderConfig) (string, error)
	Quote(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, input *NormalizedMediaInput) (float64, error)
	Submit(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, input *NormalizedMediaInput) (*NormalizedMediaOutput, error)
	Status(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, providerJobId string) (*NormalizedMediaOutput, error)
	Cancel(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, providerJobId string) error
	FetchBalance(ctx context.Context, provider *model.StudioProviderConfig) (float64, string, error)
}

// GlobalProtocolRegistry manages registered media protocols.
type GlobalProtocolRegistry struct {
	mu       sync.RWMutex
	adapters map[string]MediaProtocolAdapter
}

var (
	defaultRegistry     *GlobalProtocolRegistry
	defaultRegistryOnce sync.Once
)

func GetProtocolRegistry() *GlobalProtocolRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = &GlobalProtocolRegistry{
			adapters: make(map[string]MediaProtocolAdapter),
		}
	})
	return defaultRegistry
}

func (r *GlobalProtocolRegistry) Register(adapter MediaProtocolAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.Protocol()] = adapter
}

func (r *GlobalProtocolRegistry) Get(protocol string) (MediaProtocolAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, exists := r.adapters[protocol]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProtocolNotSupported, protocol)
	}
	return adapter, nil
}

// ApplyParameterMapping maps NormalizedMediaInput into provider-specific payload using route.InputMapping (Section 7).
func ApplyParameterMapping(input *NormalizedMediaInput, mappingJSON string) (map[string]interface{}, error) {
	payload := make(map[string]interface{})

	var mapping map[string]string
	if strings.TrimSpace(mappingJSON) != "" {
		if err := json.Unmarshal([]byte(mappingJSON), &mapping); err != nil {
			return nil, fmt.Errorf("invalid input mapping JSON: %w", err)
		}
	}

	// Default 1:1 fallback mappings if mapping is empty
	if len(mapping) == 0 {
		if input.Prompt != "" {
			payload["prompt"] = input.Prompt
		}
		if input.NegativePrompt != "" {
			payload["negative_prompt"] = input.NegativePrompt
		}
		if len(input.InputAssets) > 0 {
			payload["image_url"] = input.InputAssets[0]
		}
		if input.AspectRatio != "" {
			payload["aspect_ratio"] = input.AspectRatio
		}
		if input.Seed > 0 {
			payload["seed"] = input.Seed
		}
		for k, v := range input.AdvancedParams {
			payload[k] = v
		}
		return payload, nil
	}

	// Apply declarative mapping
	for normKey, targetKey := range mapping {
		switch normKey {
		case "prompt":
			if input.Prompt != "" {
				payload[targetKey] = input.Prompt
			}
		case "negative_prompt":
			if input.NegativePrompt != "" {
				payload[targetKey] = input.NegativePrompt
			}
		case "image_url", "input_assets[0]":
			if len(input.InputAssets) > 0 {
				payload[targetKey] = input.InputAssets[0]
			}
		case "mask_url", "mask_asset":
			if input.MaskAsset != "" {
				payload[targetKey] = input.MaskAsset
			}
		case "aspect_ratio":
			if input.AspectRatio != "" {
				payload[targetKey] = input.AspectRatio
			}
		case "width":
			if input.Width > 0 {
				payload[targetKey] = input.Width
			}
		case "height":
			if input.Height > 0 {
				payload[targetKey] = input.Height
			}
		case "size":
			if input.Width > 0 && input.Height > 0 {
				payload[targetKey] = fmt.Sprintf("%d*%d", input.Width, input.Height)
			} else if input.AspectRatio == "1:1" {
				payload[targetKey] = "1024*1024"
			} else if input.AspectRatio == "16:9" {
				payload[targetKey] = "1280*720"
			} else if input.AspectRatio == "9:16" {
				payload[targetKey] = "720*1280"
			}
		case "duration":
			if input.Duration > 0 {
				payload[targetKey] = input.Duration
			}
		case "seed":
			if input.Seed > 0 {
				payload[targetKey] = input.Seed
			}
		case "num_outputs", "number_of_outputs":
			if input.NumberOfOutputs > 0 {
				payload[targetKey] = input.NumberOfOutputs
			}
		case "callback_url":
			if input.CallbackURL != "" {
				payload[targetKey] = input.CallbackURL
			}
		default:
			if v, ok := input.AdvancedParams[normKey]; ok {
				payload[targetKey] = v
			}
		}
	}

	return payload, nil
}

// MediaPriceCache provides short bounded in-memory caching for dynamic pricing lookups (Section 27).
type MediaPriceCache struct {
	mu    sync.RWMutex
	items map[string]priceCacheItem
}

type priceCacheItem struct {
	costUSD   float64
	expiresAt time.Time
}

var (
	globalPriceCache     *MediaPriceCache
	globalPriceCacheOnce sync.Once
)

func GetPriceCache() *MediaPriceCache {
	globalPriceCacheOnce.Do(func() {
		globalPriceCache = &MediaPriceCache{
			items: make(map[string]priceCacheItem),
		}
	})
	return globalPriceCache
}

func (c *MediaPriceCache) Get(key string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	if !found || time.Now().After(item.expiresAt) {
		return 0, false
	}
	return item.costUSD, true
}

func (c *MediaPriceCache) Set(key string, costUSD float64, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = priceCacheItem{
		costUSD:   costUSD,
		expiresAt: time.Now().Add(ttl),
	}
}
