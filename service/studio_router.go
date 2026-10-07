package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// QualityTier represents user-facing logical quality selection (Section 22).
type QualityTier string

const (
	QualityTierFast    QualityTier = "FAST"
	QualityTierQuality QualityTier = "QUALITY"
	QualityTierPremium QualityTier = "PREMIUM"
)

// RouteCandidate represents an evaluated provider option for a tool execution.
type RouteCandidate struct {
	ProviderName  string      `json:"provider_name"`
	ModelEndpoint string      `json:"model_endpoint"`
	QualityTier   QualityTier `json:"quality_tier"`
	EstimatedCost float64     `json:"estimated_cost_usd"`
	P50LatencyMs  int64       `json:"p50_latency_ms"`
	SuccessRate   float64     `json:"success_rate"`
	RoutingScore  float64     `json:"routing_score"`
	IsAvailable   bool        `json:"is_available"`
}

// RouteCandidateResult holds the evaluated candidate route, provider, and score (Section 21 & 34).
type RouteCandidateResult struct {
	Route            *model.StudioModelRoute     `json:"route"`
	ProviderConfig   *model.StudioProviderConfig `json:"provider_config"`
	EffectiveCostUSD float64                     `json:"effective_cost_usd"`
	Score            float64                     `json:"score"`
	Reason           string                      `json:"reason"`
	IsEligible       bool                        `json:"is_eligible"`
	SkipReason       string                      `json:"skip_reason,omitempty"`
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

// SelectRoute evaluates all configured DB routes for a tool, evaluates dynamic quotes,
// applies circuit breakers, enforces profitability, and deterministically scores routes (Sections 21-25 & 28).
func (r *StudioRouter) SelectRoute(
	ctx context.Context,
	toolId string,
	tier QualityTier,
	normInput *NormalizedMediaInput,
	retailQuota int,
) (*model.StudioModelRoute, *model.StudioProviderConfig, []*model.StudioModelRoute, string, []RouteCandidateResult, error) {
	if tier == "" {
		tier = QualityTierQuality
	}

	// 1. Fetch configured routes for logical tool from database
	routes, err := model.GetStudioModelRoutesByTool(toolId)
	if err != nil || len(routes) == 0 {
		return nil, nil, nil, "", nil, errors.New("no configured routes for tool")
	}

	var candidates []RouteCandidateResult
	var skippedReasons []string

	for i := range routes {
		route := routes[i]
		cand := RouteCandidateResult{
			Route:            &route,
			EffectiveCostUSD: route.EffectiveCostUSD,
			IsEligible:       true,
		}

		// A. Check Route Status (Queue 2G Section 22 / Queue 2H)
		if route.Status == model.RouteStatusDisabled {
			cand.IsEligible = false
			cand.SkipReason = "route disabled"
			candidates = append(candidates, cand)
			continue
		}
		if route.Status == model.RouteStatusDraft {
			cand.IsEligible = false
			cand.SkipReason = "route status draft (not executable)"
			candidates = append(candidates, cand)
			continue
		}
		if route.Status == model.RouteStatusBillingBlocked {
			cand.IsEligible = false
			cand.SkipReason = "route billing blocked (exhausted balance)"
			candidates = append(candidates, cand)
			continue
		}
		if route.Status == model.RouteStatusCredentialRequired {
			cand.IsEligible = false
			cand.SkipReason = "route credential required (unconfigured)"
			candidates = append(candidates, cand)
			continue
		}

		// A2. Check Route enabled flag
		if !route.Enabled {
			cand.IsEligible = false
			cand.SkipReason = "route disabled"
			candidates = append(candidates, cand)
			continue
		}
		if route.Status == model.RouteStatusContractVerified {
			cand.IsEligible = false
			cand.SkipReason = "route contract verified only (canary verification required)"
			candidates = append(candidates, cand)
			continue
		}

		// A3. Provider Price Source Model Governance (Queue 2G Section 18)
		if route.PriceSource == model.PriceSourceUnknown || route.PriceSource == "UNKNOWN" {
			cand.IsEligible = false
			cand.SkipReason = "price source UNKNOWN rejected by governance"
			candidates = append(candidates, cand)
			continue
		}

		// B. Fetch Provider Config and check circuit breaker (Section 25)
		providerCfg, cfgErr := model.GetStudioProviderConfig(route.ProviderId)
		if cfgErr != nil || providerCfg == nil {
			cand.IsEligible = false
			cand.SkipReason = fmt.Sprintf("provider config '%s' not found", route.ProviderId)
			candidates = append(candidates, cand)
			continue
		}
		cand.ProviderConfig = providerCfg

		if !providerCfg.Enabled {
			cand.IsEligible = false
			cand.SkipReason = "provider disabled"
			candidates = append(candidates, cand)
			continue
		}

		if providerCfg.HealthStatus == model.ProviderHealthBillingBlocked {
			cand.IsEligible = false
			cand.SkipReason = "provider billing blocked (exhausted balance)"
			skippedReasons = append(skippedReasons, fmt.Sprintf("%s billing blocked", route.Id))
			candidates = append(candidates, cand)
			continue
		}

		if providerCfg.HealthStatus == model.ProviderHealthDisabled || providerCfg.HealthStatus == model.ProviderHealthAuthFailed {
			cand.IsEligible = false
			cand.SkipReason = fmt.Sprintf("provider health: %s", providerCfg.HealthStatus)
			candidates = append(candidates, cand)
			continue
		}

		// C. Route-level Circuit Breaker: consecutive failures >= 5 (Section 24)
		if route.ConsecutiveFails >= 5 {
			cand.IsEligible = false
			cand.SkipReason = fmt.Sprintf("circuit breaker tripped (consecutive_fails: %d)", route.ConsecutiveFails)
			candidates = append(candidates, cand)
			continue
		}

		// C2. Validate Normalized Input parameters against route constraints (Section 18)
		if normInput != nil {
			if valInputErr := ValidateNormalizedInput(&route, normInput); valInputErr != nil {
				cand.IsEligible = false
				cand.SkipReason = fmt.Sprintf("input validation failed: %v", valInputErr)
				candidates = append(candidates, cand)
				continue
			}
		}

		// D. Validate Protocol Adapter registration & credentials
		adapter, adErr := GetProtocolRegistry().Get(route.Protocol)
		if adErr != nil {
			cand.IsEligible = false
			cand.SkipReason = fmt.Sprintf("adapter error: %v", adErr)
			candidates = append(candidates, cand)
			continue
		}

		if valErr := adapter.ValidateConfiguration(providerCfg); valErr != nil {
			cand.IsEligible = false
			cand.SkipReason = fmt.Sprintf("credentials unconfigured: %v", valErr)
			candidates = append(candidates, cand)
			continue
		}

		// E. Dynamic Quote Lookup if DYNAMIC_API configured (Section 10 & 27)
		costUSD := route.EffectiveCostUSD
		if route.PricingStrategy == "DYNAMIC_API" && normInput != nil {
			q, qErr := adapter.Quote(ctx, providerCfg, &route, normInput)
			if qErr != nil {
				cand.IsEligible = false
				cand.SkipReason = fmt.Sprintf("pricing unavailable: %v", qErr)
				candidates = append(candidates, cand)
				continue
			}
			if q > 0 {
				costUSD = q
			}
		}
		cand.EffectiveCostUSD = costUSD

		// F. Profitability Guard (Queue 2G: Authoritative QuotaPerUnit conversion)
		if retailQuota > 0 && route.MinMargin > 0 {
			sellUSD := float64(retailQuota) / common.QuotaPerUnit // Canonical conversion: 500,000 Quota = $1.00 USD
			if sellUSD > 0 {
				grossMargin := ((sellUSD - costUSD) / sellUSD) * 100.0
				if grossMargin < route.MinMargin {
					cand.IsEligible = false
					cand.SkipReason = fmt.Sprintf("profitability violation: gross margin %.1f%% below floor %.1f%%", grossMargin, route.MinMargin)
					candidates = append(candidates, cand)
					continue
				}
			}
		}

		// G. Tier Matching (Section 22)
		tierMatch := 0.5
		if route.QualityTier == string(tier) {
			tierMatch = 1.0
		} else if tier == QualityTierFast && route.QualityTier == "QUALITY" {
			tierMatch = 0.6
		} else if tier == QualityTierQuality && route.QualityTier == "FAST" {
			tierMatch = 0.5
		} else if tier == QualityTierPremium && route.QualityTier != "PREMIUM" {
			tierMatch = 0.2
		}

		// H. Health & Latency Factor (Section 24)
		successRate := route.SuccessRate / 100.0
		if successRate <= 0 {
			successRate = 1.0
		}
		if route.ConsecutiveFails > 0 {
			successRate -= float64(route.ConsecutiveFails) * 0.15
			if successRate < 0.1 {
				successRate = 0.1
			}
		}

		// I. Cost Factor (Section 23: Cost-first within equivalent quality)
		costFactor := 1.0 - (costUSD / 0.10)
		if costFactor < 0 {
			costFactor = 0
		}

		// J. Priority Factor
		priFactor := 1.0 / float64(max(1, route.Priority))

		// Composite deterministic score
		qWeight := route.QualityWeight
		if qWeight <= 0 {
			qWeight = 1.0
		}
		hWeight := route.HealthWeight
		if hWeight <= 0 {
			hWeight = 1.0
		}
		cWeight := route.CostWeight
		if cWeight <= 0 {
			cWeight = 1.0
		}

		score := (tierMatch * qWeight * 2.0) +
			(successRate * hWeight * 2.0) +
			(costFactor * cWeight * 2.0) +
			priFactor

		cand.Score = score
		cand.Reason = fmt.Sprintf("tier: %s, match: %.2f, cost: $%.4f, success: %.1f%%, pri: %d",
			route.QualityTier, tierMatch, costUSD, route.SuccessRate, route.Priority)
		candidates = append(candidates, cand)
	}

	// Filter eligible candidates
	var eligible []RouteCandidateResult
	for _, c := range candidates {
		if c.IsEligible {
			eligible = append(eligible, c)
		}
	}

	if len(eligible) == 0 {
		return nil, nil, nil, "", candidates, errors.New("no eligible routes available after health, credential, and profitability checks")
	}

	// Sort eligible candidates by Score descending
	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].Score > eligible[j].Score
	})

	primary := eligible[0].Route
	primaryProvider := eligible[0].ProviderConfig
	var fallbacks []*model.StudioModelRoute
	for i := 1; i < len(eligible); i++ {
		fallbacks = append(fallbacks, eligible[i].Route)
	}

	reason := fmt.Sprintf("Selected route '%s' (%s) with score %.2f [cost: $%.4f, tier: %s, priority: %d]",
		primary.Id, primaryProvider.Name, eligible[0].Score, eligible[0].EffectiveCostUSD, primary.QualityTier, primary.Priority)

	if len(skippedReasons) > 0 {
		reason = fmt.Sprintf("%s (skipped: %s)", reason, strings.Join(skippedReasons, ", "))
	}

	return primary, primaryProvider, fallbacks, reason, candidates, nil
}

// ExecuteRouteWithSafeFallback dispatches the generation request through the generic protocol adapter,
// strictly enforcing Section 26: Ambiguous submissions NEVER fall back.
func (r *StudioRouter) ExecuteRouteWithSafeFallback(
	ctx context.Context,
	job *model.StudioToolJob,
	normInput *NormalizedMediaInput,
	primaryRoute *model.StudioModelRoute,
	fallbackRoutes []*model.StudioModelRoute,
) (*NormalizedMediaOutput, *model.StudioModelRoute, error) {
	providerCfg, err := model.GetStudioProviderConfig(primaryRoute.ProviderId)
	if err != nil || providerCfg == nil {
		return nil, primaryRoute, fmt.Errorf("provider config '%s' not found: %w", primaryRoute.ProviderId, err)
	}

	adapter, err := GetProtocolRegistry().Get(primaryRoute.Protocol)
	if err != nil {
		return nil, primaryRoute, err
	}

	startTime := time.Now()
	out, err := adapter.Submit(ctx, providerCfg, primaryRoute, normInput)
	latencyMs := time.Since(startTime).Milliseconds()

	if err == nil {
		_ = model.UpdateRouteHealthStats(primaryRoute.Id, latencyMs, true)
		return out, primaryRoute, nil
	}

	_ = model.UpdateRouteHealthStats(primaryRoute.Id, latencyMs, false)

	// SAFE FALLBACK INVARIANT (Section 26):
	// Ambiguous state (timeout after bytes dispatched) MUST NOT fall back!
	if errors.Is(err, ErrProviderAmbiguous) {
		return nil, primaryRoute, fmt.Errorf("%w: ambiguous submission to '%s'; safe fallback blocked to avoid double execution", err, primaryRoute.Id)
	}

	// Permanent rejection (e.g. invalid prompt/params or policy rejection): do NOT fall back.
	if errors.Is(err, ErrProviderPermanent) {
		return nil, primaryRoute, err
	}

	// Mark billing blocked if provider balance exhausted
	if errors.Is(err, ErrProviderAccountLocked) {
		providerCfg.HealthStatus = model.ProviderHealthBillingBlocked
		_ = model.SaveStudioProviderConfig(providerCfg)
	}

	// Safe Fallback loop (only for clean pre-submission failures)
	for _, fbRoute := range fallbackRoutes {
		fbProvider, fbCfgErr := model.GetStudioProviderConfig(fbRoute.ProviderId)
		if fbCfgErr != nil || fbProvider == nil || !fbProvider.Enabled ||
			fbProvider.HealthStatus == model.ProviderHealthBillingBlocked ||
			fbProvider.HealthStatus == model.ProviderHealthDisabled {
			continue
		}

		fbAdapter, fbAdErr := GetProtocolRegistry().Get(fbRoute.Protocol)
		if fbAdErr != nil {
			continue
		}

		fbStartTime := time.Now()
		fbOut, fbErr := fbAdapter.Submit(ctx, fbProvider, fbRoute, normInput)
		fbLatencyMs := time.Since(fbStartTime).Milliseconds()

		if fbErr == nil {
			_ = model.UpdateRouteHealthStats(fbRoute.Id, fbLatencyMs, true)
			return fbOut, fbRoute, nil
		}

		_ = model.UpdateRouteHealthStats(fbRoute.Id, fbLatencyMs, false)
		if errors.Is(fbErr, ErrProviderAmbiguous) {
			return nil, fbRoute, fmt.Errorf("%w: fallback to '%s' encountered ambiguous submission", fbErr, fbRoute.Id)
		}
	}

	return nil, primaryRoute, err
}

// SelectProvider determines the best primary provider and fallback list based on quality tier and health.
// Maintains backward compatibility with legacy providers while honoring new route configs when available.
func (r *StudioRouter) SelectProvider(
	toolId string,
	tier QualityTier,
	inputParams map[string]interface{},
) (string, []string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try DB routes first
	normInput := NormalizedFromMap(inputParams)
	primaryRoute, _, fallbackRoutes, _, _, err := r.SelectRoute(context.Background(), toolId, tier, normInput, 0)
	if err == nil && primaryRoute != nil {
		primary := primaryRoute.ProviderId
		var fallbacks []string
		for _, fb := range fallbackRoutes {
			fallbacks = append(fallbacks, fb.ProviderId)
		}
		return primary, fallbacks, nil
	}

	if tier == "" {
		tier = QualityTierQuality
	}

	// Fallback to legacy candidates mapping based on tool and tier
	var candidates []RouteCandidate

	switch toolId {
	case "background-remove":
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

// ExecuteWithSafeFallback dispatches the job with strict safe-fallback rules for legacy StudioProvider.
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

	if errors.Is(err, ErrProviderAmbiguous) {
		return nil, primaryProvider, fmt.Errorf("%w: cannot safe-fallback due to ambiguous upstream state", err)
	}

	if errors.Is(err, ErrProviderPermanent) {
		return nil, primaryProvider, err
	}

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
