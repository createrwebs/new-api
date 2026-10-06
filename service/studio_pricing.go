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

	// 1. Output count dimension
	numOutputs := 1
	if n, ok := params["number_of_outputs"].(float64); ok && n > 0 {
		numOutputs = int(n)
	} else if n, ok := params["number_of_outputs"].(int); ok && n > 0 {
		numOutputs = n
	} else if n, ok := params["num_outputs"].(float64); ok && n > 0 {
		numOutputs = int(n)
	} else if n, ok := params["num_outputs"].(int); ok && n > 0 {
		numOutputs = n
	} else if n, ok := params["num_images"].(float64); ok && n > 0 {
		numOutputs = int(n)
	}
	if numOutputs < 1 {
		numOutputs = 1
	}

	// 2. Duration, FPS & Audio dimensions for Video
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

	fps := 30
	if f, ok := params["fps"].(float64); ok && f > 0 {
		fps = int(f)
	} else if f, ok := params["fps"].(int); ok && f > 0 {
		fps = f
	}

	hasAudio := false
	if a, ok := params["audio"].(bool); ok {
		hasAudio = a
	}

	// 3. Resolution, Megapixels & Scale dimensions
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

	if resStr, ok := params["resolution"].(string); ok {
		if resStr == "2k" || resStr == "720p" || resStr == "1080p" {
			if scale > 2 && tool.Id == "image-upscale" {
				scale = 2
			}
		} else if resStr == "4k" || resStr == "ultra" {
			scale = 4
		}
	}
	if mp, ok := params["megapixels"].(float64); ok && mp < 2.0 && tool.Id == "image-upscale" {
		scale = 2
	}

	// 4. Quality & Model override dimensions
	quality, _ := params["quality"].(string)
	primaryModel := tool.PrimaryModel
	if modelOverride, ok := params["model"].(string); ok && modelOverride != "" {
		primaryModel = modelOverride
	}

	baseCredits := tool.CreditCost
	if baseCredits < 1 {
		baseCredits = 10
	}

	switch tool.Id {
	case "image-to-video", "video-generate", "text-to-video":
		// $0.016 / second ($0.080 for 5s standard)
		costUSD = float64(duration) * 0.016
		if fps >= 60 {
			costUSD *= 1.25 // 25% higher compute for high framerate
		}
		if hasAudio {
			costUSD += 0.010 // Voice / SFX track synthesis
		}
		costBasis = "per_second"

		if duration > 5 {
			chargedCredits = int(math.Ceil(float64(baseCredits) * (float64(duration) / 5.0)))
		} else {
			chargedCredits = baseCredits
		}
		if fps >= 60 {
			chargedCredits = int(math.Ceil(float64(chargedCredits) * 1.25))
		}
		if hasAudio {
			chargedCredits += 15
		}

	case "image-upscale":
		if scale <= 2 {
			costUSD = 0.010
			chargedCredits = int(math.Ceil(float64(baseCredits) * 0.72)) // e.g. 18 credits
		} else {
			costUSD = 0.015
			chargedCredits = baseCredits // 25 credits
		}
		costBasis = "per_image"

	case "image-generate":
		if quality == "ultra" || quality == "hd" {
			costUSD = 0.025 * float64(numOutputs)
			chargedCredits = 32 * numOutputs // 32 credits = $0.064 USD -> 60.9% margin over $0.025 COGS
			primaryModel = "fal-ai/flux/dev"
		} else {
			costUSD = 0.003 * float64(numOutputs)
			chargedCredits = baseCredits * numOutputs
		}
		costBasis = "per_image"

	case "product-photo":
		costUSD = 0.035 * float64(numOutputs)
		costBasis = "per_image"
		chargedCredits = baseCredits * numOutputs

	case "background-remove":
		costUSD = 0.005 * float64(numOutputs)
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
			costUSD = float64(duration) * 0.016
			if fps >= 60 {
				costUSD *= 1.25
			}
			if hasAudio {
				costUSD += 0.010
			}
			costBasis = "per_second"
			if duration > 5 {
				chargedCredits = int(math.Ceil(float64(baseCredits) * (float64(duration) / 5.0)))
			} else {
				chargedCredits = baseCredits
			}
			if fps >= 60 {
				chargedCredits = int(math.Ceil(float64(chargedCredits) * 1.25))
			}
			if hasAudio {
				chargedCredits += 15
			}
		} else {
			costUSD = 0.015 * float64(numOutputs)
			costBasis = "flat"
			chargedCredits = baseCredits * numOutputs
		}
	}

	if planMultiplier <= 0 {
		planMultiplier = 1.0
	}
	if planMultiplier != 1.0 {
		chargedCredits = e.ceilWithEpsilon(float64(chargedCredits) * planMultiplier)
	}

	// Strict Margin Floor Guarantee (Queue 1: Margin >= 60%)
	minMargin := tool.MarginPercent / 100.0
	if minMargin < 0.60 {
		minMargin = 0.60
	}
	if costUSD > 0 && minMargin < 1.0 {
		requiredSellUSD := costUSD / (1.0 - minMargin)
		minCredits := e.ceilWithEpsilon(requiredSellUSD * (common.QuotaPerUnit / float64(QuotaPerCredit)))
		if chargedCredits < minCredits {
			chargedCredits = minCredits
		}
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
		ProviderRoute:           tool.PrimaryProvider,
		ProviderModel:           primaryModel,
		ProviderEstimatedCostUSD: costUSD,
		ProviderCostBasis:       costBasis,
		CostBasis:               costBasis,
		TargetMargin:            targetMargin,
		CalculatedSellUSD:       sellPriceUSD,
		SellUSDEquivalent:       sellPriceUSD,
		CalculatedCredits:       calculatedCredits,
		ChargedCredits:          chargedCredits,
		EstimatedCredits:        chargedCredits,
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

	snapshot.QuoteID = quoteId

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
