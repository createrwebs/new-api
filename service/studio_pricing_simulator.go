package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
)

var (
	ErrSimulatorInvalidCost   = errors.New("provider cost must be greater than zero")
	ErrSimulatorInvalidMargin = errors.New("target margin must be between 0 and 99.9%")
)

// PricingSimulationResult contains calculated economics with strict ceiling rounding (Section 69 & 70).
type PricingSimulationResult struct {
	ProviderCostUSD         float64 `json:"provider_cost_usd"`
	TargetMarginPercent     float64 `json:"target_margin_percent"`
	PlanMultiplier          float64 `json:"plan_multiplier"`
	ExactRequiredSellUSD    float64 `json:"exact_required_sell_usd"`
	ExactRequiredCredits    float64 `json:"exact_required_credits"`
	ChargedCredits          int     `json:"charged_credits"`
	ChargedQuota            int     `json:"charged_quota"`
	SellUSDEquivalent       float64 `json:"sell_usd_equivalent"`
	EstimatedGrossProfitUSD float64 `json:"estimated_gross_profit_usd"`
	RealizedGrossMargin     float64 `json:"realized_gross_margin"`
	MarginFloorSatisfied    bool    `json:"margin_floor_satisfied"`
}

// SimulatePricing calculates retail Tora Credits guaranteeing the margin floor (Section 69 & 70).
func SimulatePricing(providerCostUSD float64, targetMarginPercent float64, planMultiplier float64) (*PricingSimulationResult, error) {
	if providerCostUSD <= 0 {
		return nil, ErrSimulatorInvalidCost
	}
	if targetMarginPercent < 0 || targetMarginPercent >= 100.0 {
		return nil, ErrSimulatorInvalidMargin
	}
	if planMultiplier <= 0 {
		planMultiplier = 1.0
	}

	// 1. Calculate minimum required retail sell USD to meet margin floor:
	// SellUSD = ProviderCost / (1 - Margin/100)
	marginDecimal := targetMarginPercent / 100.0
	exactSellUSD := providerCostUSD / (1.0 - marginDecimal)

	// 2. Convert to exact Tora Credits using canonical conversion (1 USD = 500 Credits, 1 Credit = $0.0020 USD)
	// Credits = exactSellUSD / 0.0020 = exactSellUSD * 500
	canonicalCreditsPerUSD := float64(common.QuotaPerUnit) / float64(QuotaPerCredit) // 500,000 / 1,000 = 500
	exactCredits := exactSellUSD * canonicalCreditsPerUSD

	// Apply plan multiplier
	adjustedCredits := exactCredits * planMultiplier

	// 3. Strict Ceiling Rounding Policy (Section 70):
	// "If target margin is a floor: never round down below the required sell price.
	// Example: required 5.1 Credits must not charge 5 if that violates margin floor."
	chargedCredits := int(math.Ceil(adjustedCredits))
	if chargedCredits < 1 {
		chargedCredits = 1
	}

	// 4. Calculate final ledger quota and realized economics
	chargedQuota := chargedCredits * QuotaPerCredit
	sellUSDEquivalent := float64(chargedQuota) / float64(common.QuotaPerUnit)
	grossProfitUSD := sellUSDEquivalent - providerCostUSD
	realizedMargin := (grossProfitUSD / sellUSDEquivalent) * 100.0

	return &PricingSimulationResult{
		ProviderCostUSD:         providerCostUSD,
		TargetMarginPercent:     targetMarginPercent,
		PlanMultiplier:          planMultiplier,
		ExactRequiredSellUSD:    exactSellUSD,
		ExactRequiredCredits:    exactCredits,
		ChargedCredits:          chargedCredits,
		ChargedQuota:            chargedQuota,
		SellUSDEquivalent:       sellUSDEquivalent,
		EstimatedGrossProfitUSD: grossProfitUSD,
		RealizedGrossMargin:     realizedMargin,
		MarginFloorSatisfied:    realizedMargin >= targetMarginPercent,
	}, nil
}

// ComputeMultiFactorPriceCacheKey builds a multi-dimensional cache key for dynamic pricing (Section 11).
func ComputeMultiFactorPriceCacheKey(providerId string, modelId string, input *NormalizedMediaInput) string {
	if input == nil {
		return fmt.Sprintf("quote:%s:%s:default", providerId, modelId)
	}

	// Multi-factor parameters that affect upstream pricing:
	// quality tier, width, height, aspect ratio, duration, number of outputs, audio
	sig := fmt.Sprintf("tier=%s;w=%d;h=%d;ar=%s;dur=%d;out=%d;aud=%t",
		input.QualityTier,
		input.Width,
		input.Height,
		input.AspectRatio,
		input.Duration,
		input.NumberOfOutputs,
		input.Audio,
	)

	h := md5.Sum([]byte(sig))
	hashStr := hex.EncodeToString(h[:8]) // 16-hex char compact hash

	return fmt.Sprintf("quote:%s:%s:%s", providerId, modelId, hashStr)
}
