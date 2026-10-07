package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// QualityTier represents user-facing logical quality selection.
type QualityTier string

const (
	QualityTierFast    QualityTier = "FAST"
	QualityTierQuality QualityTier = "QUALITY"
	QualityTierPremium QualityTier = "PREMIUM"
)

// RouteCandidate represents an evaluated provider option for a tool execution.
type RouteCandidate struct {
	ProviderName   string      `json:"provider_name"`
	ModelEndpoint  string      `json:"model_endpoint"`
	QualityTier    QualityTier `json:"quality_tier"`
	EstimatedCost  float64     `json:"estimated_cost_usd"`
	P50LatencyMs   int64       `json:"p50_latency_ms"`
	SuccessRate    float64     `json:"success_rate"`
	RoutingScore   float64     `json:"routing_score"`
	IsAvailable    bool        `json:"is_available"`
}

// StudioRoutingAnalytics aggregates live telemetry per provider and tool.
type StudioRoutingAnalytics struct {
	ToolID             string  `json:"tool_id"`
	ProviderName       string  `json:"provider_name"`
	TotalJobs          int64   `json:"total_jobs"`
	SuccessfulJobs     int64   `json:"successful_jobs"`
	FailedJobs         int64   `json:"failed_jobs"`
	SuccessRate        float64 `json:"success_rate"`
	AverageLatencyMs   int64   `json:"average_latency_ms"`
	TotalEstimatedCost float64 `json:"total_estimated_cost_usd"`
	TotalActualCost    float64 `json:"total_actual_cost_usd"`
	TotalRevenueUSD    float64 `json:"total_revenue_usd"`
	GrossMarginPercent float64 `json:"gross_margin_percent"`
}

// StudioRouter manages intelligent routing, scoring, quality tiers, and safe fallback.
type StudioRouter struct {
	mu           sync.RWMutex
	providers    map[string]StudioProvider
	failureStats map[string]int // key: provider_name -> consecutive failures
	latencies    map[string][]int64
}

func NewStudioRouter(providers map[string]StudioProvider) *StudioRouter {
	return &StudioRouter{
		providers:    providers,
		failureStats: make(map[string]int),
		latencies:    make(map[string][]int64),
	}
}

// SelectProvider determines the best primary provider and fallback list based on quality tier and health.
func (r *StudioRouter) SelectProvider(
	toolId string,
	tier QualityTier,
	inputParams map[string]interface{},
) (string, []string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if tier == "" {
		tier = QualityTierQuality
	}

	// Candidates mapping based on tool and tier
	var candidates []RouteCandidate

	switch toolId {
	case "background-remove":
		// Replicate rembg ($0.003) vs Fal birefnet ($0.005)
		if tier == QualityTierFast {
			candidates = []RouteCandidate{
				{ProviderName: "replicate", ModelEndpoint: "cjwbw/rembg", EstimatedCost: 0.003, QualityTier: QualityTierFast},
				{ProviderName: "fal", ModelEndpoint: "fal-ai/birefnet", EstimatedCost: 0.005, QualityTier: QualityTierQuality},
			}
		} else {
			candidates = []RouteCandidate{
				{ProviderName: "fal", ModelEndpoint: "fal-ai/birefnet", EstimatedCost: 0.005, QualityTier: QualityTierQuality},
				{ProviderName: "replicate", ModelEndpoint: "cjwbw/rembg", EstimatedCost: 0.003, QualityTier: QualityTierFast},
			}
		}

	case "image-upscale":
		// Replicate Real-ESRGAN ($0.005) vs Fal Clarity ($0.015)
		if tier == QualityTierFast {
			candidates = []RouteCandidate{
				{ProviderName: "replicate", ModelEndpoint: "nightmareai/real-esrgan", EstimatedCost: 0.005, QualityTier: QualityTierFast},
				{ProviderName: "fal", ModelEndpoint: "fal-ai/clarity-upscaler", EstimatedCost: 0.015, QualityTier: QualityTierPremium},
			}
		} else if tier == QualityTierPremium {
			candidates = []RouteCandidate{
				{ProviderName: "fal", ModelEndpoint: "fal-ai/clarity-upscaler", EstimatedCost: 0.015, QualityTier: QualityTierPremium},
				{ProviderName: "replicate", ModelEndpoint: "nightmareai/real-esrgan", EstimatedCost: 0.005, QualityTier: QualityTierFast},
			}
		} else {
			candidates = []RouteCandidate{
				{ProviderName: "fal", ModelEndpoint: "fal-ai/clarity-upscaler", EstimatedCost: 0.015, QualityTier: QualityTierQuality},
				{ProviderName: "replicate", ModelEndpoint: "nightmareai/real-esrgan", EstimatedCost: 0.005, QualityTier: QualityTierFast},
			}
		}

	case "image-generate":
		if tier == QualityTierFast {
			candidates = []RouteCandidate{
				{ProviderName: "fal", ModelEndpoint: "fal-ai/flux/schnell", EstimatedCost: 0.003, QualityTier: QualityTierFast},
				{ProviderName: "replicate", ModelEndpoint: "black-forest-labs/flux-schnell", EstimatedCost: 0.003, QualityTier: QualityTierFast},
			}
		} else {
			candidates = []RouteCandidate{
				{ProviderName: "fal", ModelEndpoint: "fal-ai/flux/dev", EstimatedCost: 0.025, QualityTier: QualityTierQuality},
				{ProviderName: "replicate", ModelEndpoint: "black-forest-labs/flux-dev", EstimatedCost: 0.025, QualityTier: QualityTierQuality},
			}
		}

	default:
		// Default to primary provider configured on tool or fal
		candidates = []RouteCandidate{
			{ProviderName: "fal", QualityTier: tier},
			{ProviderName: "replicate", QualityTier: tier},
		}
	}

	// Score candidates
	scored := make([]RouteCandidate, 0, len(candidates))
	for _, c := range candidates {
		p, exists := r.providers[c.ProviderName]
		if !exists {
			continue
		}

		avail := 1.0
		if c.ProviderName == "fal" {
			if f, ok := p.(*FalProvider); ok && f.ValidateConfiguration() != nil {
				avail = 0.0
			}
		} else if c.ProviderName == "replicate" {
			if rep, ok := p.(*ReplicateProvider); ok && rep.ValidateConfiguration() != nil {
				avail = 0.0
			}
		}

		consecFails := r.failureStats[c.ProviderName]
		successRate := 1.0 - (float64(consecFails) * 0.2)
		if successRate < 0.1 {
			successRate = 0.1
		}

		tierMatch := 0.5
		if c.QualityTier == tier {
			tierMatch = 1.0
		} else if tier == QualityTierFast && c.QualityTier == QualityTierQuality {
			tierMatch = 0.6
		} else if tier == QualityTierQuality && c.QualityTier == QualityTierFast {
			tierMatch = 0.4
		} else if tier == QualityTierPremium && c.QualityTier != QualityTierPremium {
			tierMatch = 0.2
		}

		// Composite Score: 0.35*avail + 0.35*tierMatch + 0.15*successRate + 0.15*costFactor
		costFactor := 1.0 - (c.EstimatedCost / 0.10)
		if costFactor < 0 {
			costFactor = 0
		}
		score := (0.35 * avail) + (0.35 * tierMatch) + (0.15 * successRate) + (0.15 * costFactor)

		c.IsAvailable = avail > 0
		c.SuccessRate = successRate * 100.0
		c.RoutingScore = score
		scored = append(scored, c)
	}

	if len(scored) == 0 {
		return "mock", []string{}, nil
	}

	// Sort candidates by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].RoutingScore > scored[j].RoutingScore
	})

	primary := scored[0].ProviderName
	var fallbacks []string
	for i := 1; i < len(scored); i++ {
		if scored[i].IsAvailable {
			fallbacks = append(fallbacks, scored[i].ProviderName)
		}
	}

	return primary, fallbacks, nil
}

// RecordJobResult updates health and latency stats for routing decisions.
func (r *StudioRouter) RecordJobResult(providerName string, success bool, latencyMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if success {
		r.failureStats[providerName] = 0
		if latencyMs > 0 {
			lats := r.latencies[providerName]
			if len(lats) > 100 {
				lats = lats[1:]
			}
			r.latencies[providerName] = append(lats, latencyMs)
		}
	} else {
		r.failureStats[providerName]++
	}
}

// ExecuteWithSafeFallback dispatches the job with strict safe-fallback rules.
func (r *StudioRouter) ExecuteWithSafeFallback(
	ctx context.Context,
	job *model.StudioToolJob,
	primaryProvider string,
	fallbackProviders []string,
) (*ProviderSubmitResult, string, error) {
	primary := r.providers[primaryProvider]
	if primary == nil {
		primary = r.providers["mock"]
		primaryProvider = "mock"
	}

	startTime := time.Now()
	res, err := primary.Submit(ctx, job)
	latency := time.Since(startTime).Milliseconds()

	if err == nil {
		r.RecordJobResult(primaryProvider, true, latency)
		return res, primaryProvider, nil
	}

	r.RecordJobResult(primaryProvider, false, latency)

	// SAFE FALLBACK INVARIANT:
	// If submission state is ambiguous (e.g. timeout after sending bytes),
	// WE MUST NOT FALL BACK! Reconcile first. Never send same generation to two providers.
	if errors.Is(err, ErrProviderAmbiguous) {
		return nil, primaryProvider, fmt.Errorf("%w: cannot safe-fallback due to ambiguous upstream state", err)
	}

	// Permanent rejection (e.g. invalid prompt/params or policy rejection): do NOT fall back.
	if errors.Is(err, ErrProviderPermanent) {
		return nil, primaryProvider, err
	}

	// Fallback is allowed ONLY when submission definitely did not begin (e.g. connection refused or unconfigured)
	for _, fallbackName := range fallbackProviders {
		fb := r.providers[fallbackName]
		if fb == nil {
			continue
		}

		fbStartTime := time.Now()
		fbRes, fbErr := fb.Submit(ctx, job)
		fbLatency := time.Since(fbStartTime).Milliseconds()

		if fbErr == nil {
			r.RecordJobResult(fallbackName, true, fbLatency)
			job.ProviderName = fallbackName
			return fbRes, fallbackName, nil
		}

		r.RecordJobResult(fallbackName, false, fbLatency)
		if errors.Is(fbErr, ErrProviderAmbiguous) {
			return nil, fallbackName, fmt.Errorf("%w: fallback encountered ambiguous submission", fbErr)
		}
	}

	return nil, primaryProvider, err
}

// GetRoutingAnalytics aggregates performance and gross margin metrics by tool and provider.
func (r *StudioRouter) GetRoutingAnalytics() ([]StudioRoutingAnalytics, error) {
	if model.DB == nil {
		return nil, errors.New("database unavailable")
	}

	var snapshots []model.StudioCostSnapshot
	if err := model.DB.Find(&snapshots).Error; err != nil {
		return nil, err
	}

	type key struct {
		tool     string
		provider string
	}

	grouped := make(map[key]*StudioRoutingAnalytics)

	for _, s := range snapshots {
		k := key{tool: s.ToolId, provider: s.ProviderName}
		item, exists := grouped[k]
		if !exists {
			item = &StudioRoutingAnalytics{
				ToolID:       s.ToolId,
				ProviderName: s.ProviderName,
			}
			grouped[k] = item
		}

		item.TotalJobs++
		item.SuccessfulJobs++
		item.TotalEstimatedCost += s.CostUSD
		item.TotalActualCost += s.CostUSD
		item.TotalRevenueUSD += s.ToraRevenueUSD
	}

	var results []StudioRoutingAnalytics
	for _, v := range grouped {
		if v.TotalJobs > 0 {
			v.SuccessRate = (float64(v.SuccessfulJobs) / float64(v.TotalJobs)) * 100.0
		}
		if v.TotalRevenueUSD > 0 {
			v.GrossMarginPercent = ((v.TotalRevenueUSD - v.TotalActualCost) / v.TotalRevenueUSD) * 100.0
		}
		results = append(results, *v)
	}

	return results, nil
}
