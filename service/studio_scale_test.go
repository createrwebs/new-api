package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueue8_RollingTelemetryCalculation(t *testing.T) {
	engine := NewScaleDecisionEngine()

	// 10 completed jobs, 2 failed jobs, across 3 distinct users
	now := time.Now().Unix()
	jobs := []model.StudioToolJob{
		{Id: "j1", UserId: 101, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 8}, // 2s
		{Id: "j2", UserId: 101, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 7}, // 3s
		{Id: "j3", UserId: 101, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 6}, // 4s
		{Id: "j4", UserId: 102, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 7}, // 3s
		{Id: "j5", UserId: 102, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 5}, // 5s
		{Id: "j6", UserId: 103, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 6}, // 4s
		{Id: "j7", UserId: 103, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 7}, // 3s
		{Id: "j8", UserId: 103, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 8}, // 2s
		{Id: "j9", UserId: 103, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 6}, // 4s
		{Id: "j10", UserId: 103, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusSucceeded, CreatedAt: now - 10, CompletedAt: now - 7}, // 3s
		{Id: "j11", UserId: 104, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusFailed, CreatedAt: now - 10},
		{Id: "j12", UserId: 105, ToolId: "background-remove", ProviderName: "fal", Status: model.StudioJobStatusFailed, CreatedAt: now - 10},
	}

	// Cost snapshots for the 10 completed jobs
	snapshots := []model.StudioCostSnapshot{
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
		{ToolId: "background-remove", ProviderName: "fal", CostUSD: 0.005, ToraRevenueUSD: 0.020},
	}

	telemetry := engine.ComputeRollingTelemetry("background-remove", "fal", jobs, snapshots, 30)

	// Invariants check:
	// Total jobs = 12 (10 success + 2 failed)
	// Jobs per day = 12 / 30 = 0.4
	// Jobs per month = 12
	assert.Equal(t, 0.4, telemetry.JobsPerDay)
	assert.Equal(t, int64(12), telemetry.JobsPerMonth)

	// Provider spend = 10 * 0.005 = $0.050
	assert.InDelta(t, 0.050, telemetry.ProviderSpendUSD, 0.0001)

	// Tora Sell Value = 10 * 0.020 = $0.200
	assert.InDelta(t, 0.200, telemetry.ToraSellValueUSD, 0.0001)

	// Gross Profit = $0.200 - $0.050 = $0.150
	assert.InDelta(t, 0.150, telemetry.GrossProfitUSD, 0.0001)

	// Gross Margin = (0.150 / 0.200) * 100 = 75.0%
	assert.InDelta(t, 75.0, telemetry.GrossMarginPercent, 0.01)

	// Success Rate = 10 / 12 = 83.33%
	assert.InDelta(t, 83.33, telemetry.SuccessRate, 0.1)

	// Refund Rate = 2 / 12 = 16.67%
	assert.InDelta(t, 16.67, telemetry.RefundRate, 0.1)

	// Repeat Users: Users 101 (3 jobs), 102 (2 jobs), 103 (5 jobs) = 3 repeat users out of 5 total unique users = 60.0%
	assert.InDelta(t, 60.0, telemetry.RepeatUseRate, 0.1)

	// Latency: P50 should be ~3000ms - 4000ms
	assert.True(t, telemetry.P50LatencyMs >= 2000 && telemetry.P50LatencyMs <= 4000)
}

func TestQueue8_CandidateStacks(t *testing.T) {
	engine := NewScaleDecisionEngine()

	// Check upscale candidate stack
	stack, exists := engine.GetCandidateStack("image-upscale")
	require.True(t, exists)
	assert.Equal(t, "Real-ESRGAN", stack.ModelFamily)
	assert.Equal(t, "BSD-3-Clause", stack.License)

	// Check background remove stack
	stack, exists = engine.GetCandidateStack("background-remove")
	require.True(t, exists)
	assert.Equal(t, "BiRefNet", stack.ModelFamily)
	assert.Equal(t, "MIT", stack.License)

	// Check video stack
	stack, exists = engine.GetCandidateStack("image-to-video")
	require.True(t, exists)
	assert.Equal(t, "Wan 2.1", stack.ModelFamily)
	assert.Equal(t, "Apache-2.0", stack.License)
}

func TestQueue8_DecisionEngine_KeepAPI(t *testing.T) {
	engine := NewScaleDecisionEngine()
	costModel := DefaultCostModel(0.44) // e.g. A4000 $0.44/hr

	// Normal healthy API usage ($200 spend/month, 75% gross margin, 99.5% success rate)
	telemetry := ToolRollingTelemetry{
		ToolID:             "background-remove",
		JobsPerDay:         50.0,
		JobsPerMonth:       1500,
		ProviderSpendUSD:   200.0,
		ToraSellValueUSD:   800.0,
		GrossProfitUSD:     600.0,
		GrossMarginPercent: 75.0,
		SuccessRate:        99.5,
	}

	result := engine.EvaluateTool(telemetry, costModel)
	assert.Equal(t, ClassKeepAPI, result.Classification)
	assert.False(t, result.MeetsTriggerRules)
	assert.Contains(t, result.Reason, "API-first remains optimal")
}

func TestQueue8_DecisionEngine_AddProvider(t *testing.T) {
	engine := NewScaleDecisionEngine()
	costModel := DefaultCostModel(0.44)

	// Spend is low ($350/mo), margin healthy (70%), but success rate degraded (91.0% < 95%)
	telemetry := ToolRollingTelemetry{
		ToolID:             "image-upscale",
		JobsPerDay:         40.0,
		JobsPerMonth:       1200,
		ProviderSpendUSD:   350.0,
		ToraSellValueUSD:   1166.0,
		GrossProfitUSD:     816.0,
		GrossMarginPercent: 70.0,
		SuccessRate:        91.0, // Flaky primary provider
	}

	result := engine.EvaluateTool(telemetry, costModel)
	assert.Equal(t, ClassAddProvider, result.Classification)
	assert.False(t, result.MeetsTriggerRules)
	assert.Contains(t, result.Reason, "prioritize multi-provider routing")
}

func TestQueue8_DecisionEngine_ServerlessGPUCandidate(t *testing.T) {
	engine := NewScaleDecisionEngine()
	// Hourly compute $0.44/hr * 720 = $316.80 + storage $20 + egress $30 + ops $550 = ~$916.80
	costModel := DefaultCostModel(0.44)

	// High spend ($3,000/mo > $1,000), compressed margin (35% < 50%), bursty jobs (300 jobs/day)
	// Savings = $3,000 - $916.80 = $2,083.20 (> 50% savings)
	telemetry := ToolRollingTelemetry{
		ToolID:             "image-generate",
		JobsPerDay:         300.0,
		JobsPerMonth:       9000,
		ProviderSpendUSD:   3000.0,
		ToraSellValueUSD:   4615.0,
		GrossProfitUSD:     1615.0,
		GrossMarginPercent: 35.0, // Compressed margin
		SuccessRate:        99.0,
	}

	result := engine.EvaluateTool(telemetry, costModel)
	assert.Equal(t, ClassServerlessGPUCandidate, result.Classification)
	assert.True(t, result.MeetsTriggerRules)
	assert.True(t, result.SavingsPercent > 50.0)
	assert.Contains(t, result.Reason, "Serverless Container GPU")
}

func TestQueue8_DecisionEngine_SelfHostCandidate(t *testing.T) {
	engine := NewScaleDecisionEngine()
	costModel := DefaultCostModel(0.44)

	// Massive spend ($8,000/mo), compressed margin (30%), extremely high sustained density (2,500 jobs/day)
	telemetry := ToolRollingTelemetry{
		ToolID:             "background-remove",
		JobsPerDay:         2500.0,
		JobsPerMonth:       75000,
		ProviderSpendUSD:   8000.0,
		ToraSellValueUSD:   11428.0,
		GrossProfitUSD:     3428.0,
		GrossMarginPercent: 30.0,
		SuccessRate:        99.2,
	}

	result := engine.EvaluateTool(telemetry, costModel)
	assert.Equal(t, ClassSelfHostCandidate, result.Classification)
	assert.True(t, result.MeetsTriggerRules)
	assert.True(t, result.SavingsPercent > 50.0)
	assert.Contains(t, result.Reason, "Sustained monthly spend exceeds $1,000")
}

func TestQueue8_TriggerRule_HighSpendButHighMargin_RemainsKeepAPI(t *testing.T) {
	engine := NewScaleDecisionEngine()
	costModel := DefaultCostModel(0.44)

	// High spend ($2,500/mo > $1,000), BUT high margin (75% > 50%)
	// TRIGGER RULE: Must NOT self-host if gross margin is already > 50%!
	telemetry := ToolRollingTelemetry{
		ToolID:             "product-photo",
		JobsPerDay:         100.0,
		JobsPerMonth:       3000,
		ProviderSpendUSD:   2500.0,
		ToraSellValueUSD:   10000.0,
		GrossProfitUSD:     7500.0,
		GrossMarginPercent: 75.0, // Strong profitability
		SuccessRate:        99.1,
	}

	result := engine.EvaluateTool(telemetry, costModel)
	// Because margin is >= 50%, trigger fails -> Must KEEP_API
	assert.Equal(t, ClassKeepAPI, result.Classification)
	assert.False(t, result.MeetsTriggerRules)
	assert.Contains(t, result.Reason, "API-first remains optimal")
}
