package service

import (
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	// QuotaPerCredit defines internal quota units per 1 Tora Credit.
	// 500,000 Quota = 500 Tora Credits = $1.00 reference value.
	QuotaPerCredit = 1000
)

// PricingEngine calculates provider-neutral Tora Credit and Quota prices (Section 3).
type PricingEngine struct {
	defaultMargins map[string]float64
}

func NewPricingEngine() *PricingEngine {
	return &PricingEngine{
		defaultMargins: map[string]float64{
			"utility": 50.0, // 40-60% margin for utility
			"image":   60.0, // 50-65% margin for premium image
			"video":   45.0, // 35-55% margin for video
			"product": 65.0, // 50-70% margin for creator workflow
		},
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
			margin = 55.0
		}
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

	// Convert to Tora Credits (rounded up to nearest integer)
	credits := int(math.Ceil(adjustedQuota / float64(QuotaPerCredit)))
	if credits < 1 {
		credits = 1
	}

	// Exact quota deduction = credits * QuotaPerCredit
	quotaCost = credits * QuotaPerCredit

	return credits, quotaCost, sellPriceUSD
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
		credits = int(math.Ceil(float64(tool.CreditCost) * planMultiplier))
		if credits < 1 {
			credits = 1
		}
	}

	return credits, credits * QuotaPerCredit
}
