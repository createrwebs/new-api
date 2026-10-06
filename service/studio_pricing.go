package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	// QuotaPerCredit defines internal quota units per 1 Tora Credit.
	// 500,000 Quota = 500 Tora Credits = $1.00 reference value.
	QuotaPerCredit = 1000

	// DefaultQuoteTTL defines the active lifespan of a pre-submission price quote (15 minutes).
	DefaultQuoteTTL = 15 * time.Minute

	// MinGrossMarginFloor defines the strict platform gross margin floor (>= 60%).
	MinGrossMarginFloor = 60.0
)

var (
	ErrMarginBelowFloor = errors.New("profitability guard rejected: calculated gross margin is below mandatory floor")
	ErrQuoteExpired     = errors.New("pricing quote has expired (15-minute TTL exceeded)")
	ErrQuoteNotFound    = errors.New("pricing quote not found")
)

type quoteCacheEntry struct {
	snapshot  model.StudioPricingSnapshot
	expiresAt time.Time
}

// PricingEngine calculates provider-neutral Tora Credit and Quota prices (Section 3 & 14).
type PricingEngine struct {
	defaultMargins map[string]float64
	quotesMu       sync.RWMutex
	quoteCache     map[string]quoteCacheEntry
}

func NewPricingEngine() *PricingEngine {
	return &PricingEngine{
		defaultMargins: map[string]float64{
			"utility": 65.0, // 60-75% margin for utility
			"image":   60.0, // 60-65% margin for premium image
			"video":   50.0, // 45-55% margin for video
			"product": 65.0, // 60-70% margin for creator workflow
		},
		quoteCache: make(map[string]quoteCacheEntry),
	}
}

// CalculatePrice derives Quota and Tora Credits from provider COGS and margin policy.
func (e *PricingEngine) CalculatePrice(category string, providerCostUSD float64, customMargin float64, planMultiplier float64) (creditCost int, quotaCost int, sellPriceUSD float64) {
	if providerCostUSD <= 0 {
		providerCostUSD = 0.005
	}

	margin := customMargin
	if margin <= 0 || margin >= 95 {
		if def, ok := e.defaultMargins[category]; ok {
			margin = def
		} else {
			margin = MinGrossMarginFloor
		}
	}

	// Floor enforcement
	if category != "video" && margin < MinGrossMarginFloor {
		margin = MinGrossMarginFloor
	}

	// sellPrice = cost / (1 - margin)
	sellPriceUSD = providerCostUSD / (1.0 - (margin / 100.0))

	// Base quota = sellPriceUSD * common.QuotaPerUnit (500,000 per USD)
	baseQuota := sellPriceUSD * common.QuotaPerUnit

	// Plan multiplier adjustment (Section 28)
	if planMultiplier <= 0 {
		planMultiplier = 1.0
	}
	adjustedQuota := baseQuota * planMultiplier

	// Convert to Tora Credits (strictly rounded up to nearest integer ceiling)
	calculatedCredits := adjustedQuota / float64(QuotaPerCredit)
	credits := e.ceilWithEpsilon(calculatedCredits)
	if credits < 1 {
		credits = 1
	}

	// Exact quota deduction = credits * QuotaPerCredit
	quotaCost = credits * QuotaPerCredit

	return credits, quotaCost, sellPriceUSD
}

// CalculatePriceWithInputs generates a verifiable StudioPricingSnapshot with dynamic inputs (duration, scale, outputs).
func (e *PricingEngine) CalculatePriceWithInputs(
	tool *model.StudioToolDefinition,
	params map[string]interface{},
	planMultiplier float64,
) (*model.StudioPricingSnapshot, error) {
	if tool == nil {
		return nil, errors.New("cannot calculate price for nil tool definition")
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	var costUSD float64
	var costBasis string
	var chargedCredits int

	baseCredits := tool.CreditCost
	if baseCredits < 1 {
		baseCredits = 10
	}

	switch tool.Id {
	case "image-to-video":
		duration := 5
		if d, ok := params["duration"].(float64); ok && d > 0 {
			duration = int(d)
		} else if d, ok := params["duration"].(int); ok && d > 0 {
			duration = d
		} else if d, ok := params["duration_sec"].(float64); ok && d > 0 {
			duration = int(d)
		} else if d, ok := params["duration_sec"].(int); ok && d > 0 {
			duration = d
		}
		if duration < 1 {
			duration = 5
		}
		// $0.016 / second ($0.080 for 5s standard)
		costUSD = float64(duration) * 0.016
		costBasis = "per_second"

		if duration > 5 {
			chargedCredits = int(math.Ceil(float64(baseCredits) * (float64(duration) / 5.0)))
		} else {
			chargedCredits = baseCredits
		}

	case "image-upscale":
		scale := 4
		if s, ok := params["scale"].(float64); ok && s > 0 {
			scale = int(s)
		} else if s, ok := params["scale"].(int); ok && s > 0 {
			scale = s
		} else if s, ok := params["scale_factor"].(float64); ok && s > 0 {
			scale = int(s)
		} else if s, ok := params["scale_factor"].(int); ok && s > 0 {
			scale = s
		}
		if scale <= 2 {
			costUSD = 0.010
			chargedCredits = int(math.Ceil(float64(baseCredits) * 0.72)) // e.g. 18 credits
		} else {
			costUSD = 0.015
			chargedCredits = baseCredits // 25 credits
		}
		costBasis = "per_image"

	case "image-generate":
		numOutputs := 1
		if n, ok := params["num_outputs"].(float64); ok && n > 0 {
			numOutputs = int(n)
		} else if n, ok := params["num_outputs"].(int); ok && n > 0 {
			numOutputs = n
		} else if n, ok := params["num_images"].(float64); ok && n > 0 {
			numOutputs = int(n)
		}
		if numOutputs < 1 {
			numOutputs = 1
		}
		costUSD = 0.003 * float64(numOutputs)
		costBasis = "per_image"
		chargedCredits = baseCredits * numOutputs

	case "product-photo":
		numOutputs := 1
		if n, ok := params["num_outputs"].(float64); ok && n > 0 {
			numOutputs = int(n)
		} else if n, ok := params["num_outputs"].(int); ok && n > 0 {
			numOutputs = n
		}
		if numOutputs < 1 {
			numOutputs = 1
		}
		costUSD = 0.035 * float64(numOutputs)
		costBasis = "per_image"
		chargedCredits = baseCredits * numOutputs

	case "image-extend":
		costUSD = 0.025
		costBasis = "per_image"
		chargedCredits = baseCredits

	case "object-erase":
		costUSD = 0.020
		costBasis = "per_image"
		chargedCredits = baseCredits

	default:
		if tool.Category == "video" {
			costUSD = 0.080
			costBasis = "per_second"
		} else {
			costUSD = 0.015
			costBasis = "flat"
		}
		chargedCredits = baseCredits
	}

	if planMultiplier <= 0 {
		planMultiplier = 1.0
	}
	if planMultiplier != 1.0 {
		chargedCredits = e.ceilWithEpsilon(float64(chargedCredits) * planMultiplier)
	}
	if chargedCredits < 1 {
		chargedCredits = 1
	}

	chargedQuota := chargedCredits * QuotaPerCredit
	revenueUSD := float64(chargedQuota) / common.QuotaPerUnit
	sellPriceUSD := revenueUSD
	calculatedCredits := float64(chargedCredits)

	targetMargin := tool.MarginPercent
	if revenueUSD > 0 {
		targetMargin = ((revenueUSD - costUSD) / revenueUSD) * 100.0
	}

	now := common.GetTimestamp()
	snapshot := &model.StudioPricingSnapshot{
		PricingVersion:          "v1.2",
		Provider:                tool.PrimaryProvider,
		ProviderModel:           tool.PrimaryModel,
		ProviderEstimatedCostUSD: costUSD,
		ProviderCostBasis:       costBasis,
		TargetMargin:            targetMargin,
		CalculatedSellUSD:       sellPriceUSD,
		CalculatedCredits:       calculatedCredits,
		ChargedCredits:          chargedCredits,
		ChargedQuota:            chargedQuota,
		PlanMultiplier:          planMultiplier,
		QuotedAt:                now,
		ExpiresAt:               now + int64(DefaultQuoteTTL.Seconds()),
	}

	return snapshot, nil
}

// ceilWithEpsilon converts a float to ceiling int, ignoring micro precision noise (e.g. 50.00000000000001 -> 50).
func (e *PricingEngine) ceilWithEpsilon(v float64) int {
	const epsilon = 1e-7
	rounded := math.Round(v)
	if math.Abs(v-rounded) < epsilon {
		return int(rounded)
	}
	return int(math.Ceil(v))
}

// ValidateProfitability verifies that the pricing snapshot satisfies gross margin invariants.
func (e *PricingEngine) ValidateProfitability(snapshot *model.StudioPricingSnapshot, minMargin float64) error {
	if snapshot == nil {
		return errors.New("pricing snapshot is nil")
	}

	if snapshot.ChargedQuota <= 0 {
		return errors.New("charged quota must be greater than zero")
	}

	revenueUSD := float64(snapshot.ChargedQuota) / common.QuotaPerUnit
	grossProfitUSD := revenueUSD - snapshot.ProviderEstimatedCostUSD
	if revenueUSD <= 0 || grossProfitUSD < 0 {
		return ErrMarginBelowFloor
	}

	actualMargin := (grossProfitUSD / revenueUSD) * 100.0
	// 0.05% tolerance for floating point representations
	if minMargin > 0 && actualMargin < (minMargin-0.05) {
		return fmt.Errorf("%w: actual %.2f%% < required %.2f%%", ErrMarginBelowFloor, actualMargin, minMargin)
	}

	return nil
}

// SaveQuote stores a pricing snapshot in memory with a 15-minute TTL.
func (e *PricingEngine) SaveQuote(snapshot *model.StudioPricingSnapshot) string {
	if snapshot == nil {
		return ""
	}

	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	quoteId := fmt.Sprintf("quote_%d_%s", common.GetTimestamp(), hex.EncodeToString(buf))

	e.quotesMu.Lock()
	defer e.quotesMu.Unlock()

	// Clean up expired entries if cache is growing
	if len(e.quoteCache) > 1000 {
		now := time.Now()
		for k, v := range e.quoteCache {
			if now.After(v.expiresAt) {
				delete(e.quoteCache, k)
			}
		}
	}

	e.quoteCache[quoteId] = quoteCacheEntry{
		snapshot:  *snapshot,
		expiresAt: time.Now().Add(DefaultQuoteTTL),
	}

	return quoteId
}

// GetQuote retrieves an unexpired pricing quote.
func (e *PricingEngine) GetQuote(quoteId string) (*model.StudioPricingSnapshot, error) {
	e.quotesMu.RLock()
	entry, exists := e.quoteCache[quoteId]
	e.quotesMu.RUnlock()

	if !exists {
		return nil, ErrQuoteNotFound
	}

	if time.Now().After(entry.expiresAt) {
		e.quotesMu.Lock()
		delete(e.quoteCache, quoteId)
		e.quotesMu.Unlock()
		return nil, ErrQuoteExpired
	}

	s := entry.snapshot
	return &s, nil
}

// FormatToraCredits converts raw quota units to user-facing Tora Credits.
func FormatToraCredits(quota int) int {
	return quota / QuotaPerCredit
}

// QuotaFromToraCredits converts user-facing Tora Credits to internal quota units.
func QuotaFromToraCredits(credits int) int {
	return credits * QuotaPerCredit
}

// EstimateToolCost calculates credits for a tool definition and user tier.
func (e *PricingEngine) EstimateToolCost(tool *model.StudioToolDefinition, planMultiplier float64) (int, int) {
	if tool == nil {
		return 10, 10000
	}

	credits := tool.CreditCost
	if planMultiplier > 0 && planMultiplier != 1.0 {
		credits = e.ceilWithEpsilon(float64(tool.CreditCost) * planMultiplier)
		if credits < 1 {
			credits = 1
		}
	}

	return credits, credits * QuotaPerCredit
}
