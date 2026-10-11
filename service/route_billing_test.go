package service_test

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestRouteBillingFormula(t *testing.T) {
	// Invariant 0.4: Authoritative Billing Formula:
	// SettledQuota = ceil(BaseQuota * ModelRatio * UserGroupRatio * WinningRouteCostMultiplier)
	promptTokens := 1000
	completionTokens := 500
	completionRatio := 2.0 // completion tokens count 2x

	baseQuota := float64(promptTokens) + float64(completionTokens)*completionRatio // 1000 + 1000 = 2000
	modelRatio := 1.5
	userGroupRatio := 1.0
	winningRouteMultiplier := 1.25

	// Authoritative formula calculation
	expectedSettled := math.Ceil(baseQuota * modelRatio * userGroupRatio * winningRouteMultiplier) // ceil(2000 * 1.5 * 1.0 * 1.25) = ceil(3750) = 3750

	// Test PriceData Decimal calculation
	priceData := types.PriceData{
		ModelRatio:      modelRatio,
		CompletionRatio: completionRatio,
	}
	priceData.AddOtherRatio("route_multiplier", winningRouteMultiplier)

	dBase := decimal.NewFromFloat(baseQuota)
	dRatio := decimal.NewFromFloat(modelRatio).Mul(decimal.NewFromFloat(userGroupRatio))
	dCalculated := dBase.Mul(dRatio)
	dFinal := priceData.ApplyOtherRatiosToDecimal(dCalculated)

	actualQuota := int(math.Ceil(dFinal.InexactFloat64()))
	assert.Equal(t, int(expectedSettled), actualQuota, "decimal calculation must exactly match authoritative formula")

	// Test that route multiplier does not get applied twice
	assert.Equal(t, winningRouteMultiplier, priceData.OtherRatioMultiplier())
}

func TestConservativePreConsumptionCeiling(t *testing.T) {
	// Requirement 18:
	// reserve = max(estimated cost of every eligible route in chain)
	baseQuotaToPreConsume := 1000

	routeMultipliers := []float64{1.0, 1.25, 1.50} // Primary: 1.0, Fallback 1: 1.25, Fallback 2: 1.50

	maxMultiplier := 1.0
	for _, m := range routeMultipliers {
		if m > maxMultiplier {
			maxMultiplier = m
		}
	}
	assert.Equal(t, 1.50, maxMultiplier)

	reservedQuota := int(math.Ceil(float64(baseQuotaToPreConsume) * maxMultiplier))
	assert.Equal(t, 1500, reservedQuota)

	// Simulate winning route is Fallback 1 (multiplier 1.25)
	actualWinningQuota := int(math.Ceil(float64(baseQuotaToPreConsume) * 1.25))
	assert.Equal(t, 1250, actualWinningQuota)

	// Difference to refund
	refundDelta := reservedQuota - actualWinningQuota
	assert.Equal(t, 250, refundDelta, "difference between max reserved and actual winning cost must be refunded")
}
