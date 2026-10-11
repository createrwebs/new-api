package service

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

// ToolScaleClassification represents the infrastructure decision state for a tool.
type ToolScaleClassification string

const (
	ClassKeepAPI                ToolScaleClassification = "KEEP_API"
	ClassAddProvider            ToolScaleClassification = "ADD_PROVIDER"
	ClassServerlessGPUCandidate ToolScaleClassification = "SERVERLESS_GPU_CANDIDATE"
	ClassSelfHostCandidate      ToolScaleClassification = "SELF_HOST_CANDIDATE"
	ClassNotEconomic            ToolScaleClassification = "NOT_ECONOMIC"
)

// CandidateStack defines the open-source self-host alternative for a tool.
type CandidateStack struct {
	ToolID         string  `json:"tool_id"`
	StackName      string  `json:"stack_name"`
	ModelFamily    string  `json:"model_family"`
	License        string  `json:"license"`
	RequiredVRAM   string  `json:"required_vram"`
	TargetGPU      string  `json:"target_gpu"`
	HourlyGPUCost  float64 `json:"hourly_gpu_cost_usd"`
}

// SelfHostCostModel accounts for true infrastructure and operational overheads.
type SelfHostCostModel struct {
	HourlyComputeUSD   float64 `json:"hourly_compute_usd"`    // e.g. RunPod / Lambda A4000/A5000 ($0.44 - $0.79/hr)
	MonthlyIdleHours   float64 `json:"monthly_idle_hours"`    // Off-peak idle time allowance (e.g. 240 hrs)
	MonthlyEgressUSD   float64 `json:"monthly_egress_usd"`    // Media egress bandwidth
	MonthlyStorageUSD  float64 `json:"monthly_storage_usd"`   // Persistent model / weights volume
	MaintenanceHours   float64 `json:"maintenance_hours"`     // Dev/Ops hours per month for patching/deploying
	OpsHourlyRateUSD   float64 `json:"ops_hourly_rate_usd"`   // Internal ops cost basis ($50.00/hr)
	IncidentBurdenUSD  float64 `json:"incident_burden_usd"`   // Buffer for on-call incidents, failovers, downtime
	ColdStartPenaltyMs int64   `json:"cold_start_penalty_ms"` // Typical cold start delay
	SLAAvailabilityPct float64 `json:"sla_availability_pct"`  // Expected cluster SLA (e.g. 98.5% vs 99.95% API)
}

// DefaultCostModel returns realistic operational parameters for serverless/dedicated GPU hosting.
func DefaultCostModel(hourlyCompute float64) SelfHostCostModel {
	return SelfHostCostModel{
		HourlyComputeUSD:   hourlyCompute,
		MonthlyIdleHours:   180.0,
		MonthlyEgressUSD:   30.0,
		MonthlyStorageUSD:  20.0,
		MaintenanceHours:   8.0,
		OpsHourlyRateUSD:   50.0, // $400/mo ops cost
		IncidentBurdenUSD:  150.0,
		ColdStartPenaltyMs: 12000,
		SLAAvailabilityPct: 99.2,
	}
}

// ToolRollingTelemetry aggregates performance, monetization, and reliability over a rolling period.
type ToolRollingTelemetry struct {
	ToolID             string  `json:"tool_id"`
	ProviderName       string  `json:"provider_name"`
	JobsPerDay         float64 `json:"jobs_per_day"`
	JobsPerMonth       int64   `json:"jobs_per_month"`
	ProviderSpendUSD   float64 `json:"provider_spend_usd"`
	ToraSellValueUSD   float64 `json:"tora_sell_value_usd"`
	GrossProfitUSD     float64 `json:"gross_profit_usd"`
	GrossMarginPercent float64 `json:"gross_margin_percent"`
	P50LatencyMs       int64   `json:"p50_latency_ms"`
	P95LatencyMs       int64   `json:"p95_latency_ms"`
	SuccessRate        float64 `json:"success_rate"`
	RefundRate         float64 `json:"refund_rate"`
	RepeatUseRate      float64 `json:"repeat_use_rate"`
}

// ScaleEvaluationResult contains the decision engine verdict and economic justification.
type ScaleEvaluationResult struct {
	ToolID             string                  `json:"tool_id"`
	Classification     ToolScaleClassification `json:"classification"`
	CurrentAPISpend    float64                 `json:"current_api_spend_usd"`
	SelfHostCostUSD    float64                 `json:"self_host_cost_usd"`
	ProjectedSavingsUSD float64                `json:"projected_savings_usd"`
	SavingsPercent     float64                 `json:"savings_percent"`
	MeetsTriggerRules  bool                    `json:"meets_trigger_rules"`
	Reason             string                  `json:"reason"`
}

// ScaleDecisionEngine evaluates when to remain API-first vs when to graduate to self-hosting.
type ScaleDecisionEngine struct {
	candidateStacks map[string]CandidateStack
}

func NewScaleDecisionEngine() *ScaleDecisionEngine {
	return &ScaleDecisionEngine{
		candidateStacks: map[string]CandidateStack{
			"background-remove": {
				ToolID:        "background-remove",
				StackName:     "rembg / BiRefNet container",
				ModelFamily:   "BiRefNet",
				License:       "MIT",
				RequiredVRAM:  "4GB",
				TargetGPU:     "T4 / RTX 3060",
				HourlyGPUCost: 0.22,
			},
			"image-upscale": {
				ToolID:        "image-upscale",
				StackName:     "Real-ESRGAN / Compact",
				ModelFamily:   "Real-ESRGAN",
				License:       "BSD-3-Clause",
				RequiredVRAM:  "8GB",
				TargetGPU:     "RTX 3090 / A4000",
				HourlyGPUCost: 0.44,
			},
			"image-generate": {
				ToolID:        "image-generate",
				StackName:     "ComfyUI / Flux-schnell",
				ModelFamily:   "Flux",
				License:       "Apache-2.0",
				RequiredVRAM:  "16GB - 24GB",
				TargetGPU:     "A5000 / RTX 4090",
				HourlyGPUCost: 0.79,
			},
			"product-photo": {
				ToolID:        "product-photo",
				StackName:     "ComfyUI Pipeline + ControlNet",
				ModelFamily:   "SDXL / Flux Inpaint",
				License:       "OpenRAIL",
				RequiredVRAM:  "24GB",
				TargetGPU:     "RTX 4090 / A5000",
				HourlyGPUCost: 0.79,
			},
			"image-to-video": {
				ToolID:        "image-to-video",
				StackName:     "Wan 2.1 / LTX-Video",
				ModelFamily:   "Wan 2.1",
				License:       "Apache-2.0",
				RequiredVRAM:  "48GB - 80GB",
				TargetGPU:     "A100 80GB / H100",
				HourlyGPUCost: 2.19,
			},
			"lip-sync": {
				ToolID:        "lip-sync",
				StackName:     "MuseTalk Serverless",
				ModelFamily:   "MuseTalk",
				License:       "Research/Commercial Pending",
				RequiredVRAM:  "16GB",
				TargetGPU:     "A4000",
				HourlyGPUCost: 0.44,
			},
			"face-swap": {
				ToolID:        "face-swap",
				StackName:     "FaceFusion Pipeline",
				ModelFamily:   "FaceFusion",
				License:       "GPL-3.0 (Requires Compliance Review)",
				RequiredVRAM:  "16GB",
				TargetGPU:     "RTX 3090 / A4000",
				HourlyGPUCost: 0.44,
			},
		},
	}
}

// GetCandidateStack returns reference open-source stack information.
func (e *ScaleDecisionEngine) GetCandidateStack(toolId string) (CandidateStack, bool) {
	c, exists := e.candidateStacks[toolId]
	return c, exists
}

// ComputeRollingTelemetry calculates 30-day rolling operational metrics.
func (e *ScaleDecisionEngine) ComputeRollingTelemetry(
	toolId string,
	providerName string,
	jobs []model.StudioToolJob,
	snapshots []model.StudioCostSnapshot,
	windowDays int,
) ToolRollingTelemetry {
	if windowDays <= 0 {
		windowDays = 30
	}

	var totalSpendUSD float64
	var totalRevenueUSD float64
	for _, s := range snapshots {
		if s.ToolId == toolId && (providerName == "" || s.ProviderName == providerName) {
			totalSpendUSD += s.CostUSD
			totalRevenueUSD += s.ToraRevenueUSD
		}
	}

	var successfulJobs int64
	var failedJobs int64
	var latencies []int64
	userJobCounts := make(map[int]int)

	for _, j := range jobs {
		if j.ToolId == toolId && (providerName == "" || j.ProviderName == providerName) {
			userJobCounts[j.UserId]++
			if j.Status == model.StudioJobStatusSucceeded {
				successfulJobs++
			} else if j.Status == model.StudioJobStatusFailed {
				failedJobs++
			}

			if j.CompletedAt > 0 && j.CreatedAt > 0 && j.CompletedAt >= j.CreatedAt {
				latMs := (j.CompletedAt - j.CreatedAt) * 1000
				latencies = append(latencies, latMs)
			}
		}
	}

	totalJobs := successfulJobs + failedJobs
	jobsPerDay := float64(totalJobs) / float64(windowDays)
	jobsPerMonth := int64(jobsPerDay * 30.0)

	grossProfit := totalRevenueUSD - totalSpendUSD
	grossMargin := 0.0
	if totalRevenueUSD > 0 {
		grossMargin = (grossProfit / totalRevenueUSD) * 100.0
	}

	successRate := 100.0
	refundRate := 0.0
	if totalJobs > 0 {
		successRate = (float64(successfulJobs) / float64(totalJobs)) * 100.0
		refundRate = (float64(failedJobs) / float64(totalJobs)) * 100.0
	}

	// Calculate Repeat Use Rate (users with >= 2 jobs / total unique users)
	uniqueUsers := len(userJobCounts)
	repeatUsers := 0
	for _, count := range userJobCounts {
		if count >= 2 {
			repeatUsers++
		}
	}
	repeatUseRate := 0.0
	if uniqueUsers > 0 {
		repeatUseRate = (float64(repeatUsers) / float64(uniqueUsers)) * 100.0
	}

	// Calculate P50 and P95 latency
	var p50, p95 int64
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		p50Idx := int(math.Floor(float64(len(latencies)) * 0.50))
		p95Idx := int(math.Floor(float64(len(latencies)) * 0.95))
		if p50Idx >= len(latencies) {
			p50Idx = len(latencies) - 1
		}
		if p95Idx >= len(latencies) {
			p95Idx = len(latencies) - 1
		}
		p50 = latencies[p50Idx]
		p95 = latencies[p95Idx]
	}

	return ToolRollingTelemetry{
		ToolID:             toolId,
		ProviderName:       providerName,
		JobsPerDay:         jobsPerDay,
		JobsPerMonth:       jobsPerMonth,
		ProviderSpendUSD:   totalSpendUSD,
		ToraSellValueUSD:   totalRevenueUSD,
		GrossProfitUSD:     grossProfit,
		GrossMarginPercent: grossMargin,
		P50LatencyMs:       p50,
		P95LatencyMs:       p95,
		SuccessRate:        successRate,
		RefundRate:         refundRate,
		RepeatUseRate:      repeatUseRate,
	}
}

// ComputeRollingTelemetryFromDB reads directly from GORM tables.
func (e *ScaleDecisionEngine) ComputeRollingTelemetryFromDB(db *gorm.DB, toolId string, windowDays int) (ToolRollingTelemetry, error) {
	if db == nil {
		return ToolRollingTelemetry{}, errors.New("database unavailable")
	}
	if windowDays <= 0 {
		windowDays = 30
	}

	since := time.Now().AddDate(0, 0, -windowDays).Unix()

	var jobs []model.StudioToolJob
	if err := db.Where("tool_id = ? AND created_at >= ?", toolId, since).Find(&jobs).Error; err != nil {
		return ToolRollingTelemetry{}, err
	}

	var snapshots []model.StudioCostSnapshot
	if err := db.Where("tool_id = ? AND snapshot_at >= ?", toolId, since).Find(&snapshots).Error; err != nil {
		return ToolRollingTelemetry{}, err
	}

	return e.ComputeRollingTelemetry(toolId, "", jobs, snapshots, windowDays), nil
}

// EvaluateTool executes the formal decision engine and trigger rules.
//
// TRIGGER RULE:
// Do not consider self-host until:
// 1. Tool spend > $1,000/month
// 2. AND Margin < 50%
// 3. AND Self-host savings > 50%
// 4. AND Maintenance cost < Savings
// Otherwise:
// KEEP API-FIRST.
func (e *ScaleDecisionEngine) EvaluateTool(
	telemetry ToolRollingTelemetry,
	costModel SelfHostCostModel,
) ScaleEvaluationResult {
	// Rule 0: If gross margin is negative, economics are broken
	if telemetry.ToraSellValueUSD > 0 && telemetry.GrossMarginPercent < 0 {
		return ScaleEvaluationResult{
			ToolID:          telemetry.ToolID,
			Classification:  ClassNotEconomic,
			CurrentAPISpend: telemetry.ProviderSpendUSD,
			Reason:          "Negative gross margin; sell price does not cover API COGS",
		}
	}

	// 1. Compute Full Self-Host Monthly Cost
	// 720 hours base active month, factoring in idle, storage, egress, ops, and incident buffer
	activeComputeCost := 720.0 * costModel.HourlyComputeUSD
	totalOpsMaintenance := (costModel.MaintenanceHours * costModel.OpsHourlyRateUSD) + costModel.IncidentBurdenUSD
	totalSelfHostCost := activeComputeCost + costModel.MonthlyEgressUSD + costModel.MonthlyStorageUSD + totalOpsMaintenance

	projectedSavingsUSD := telemetry.ProviderSpendUSD - totalSelfHostCost
	savingsPercent := 0.0
	if telemetry.ProviderSpendUSD > 0 {
		savingsPercent = (projectedSavingsUSD / telemetry.ProviderSpendUSD) * 100.0
	}

	// 2. Evaluate Trigger Rules
	condHighSpend := telemetry.ProviderSpendUSD > 1000.0
	condLowMargin := telemetry.GrossMarginPercent < 50.0
	condMeaningfulSavings := savingsPercent > 50.0
	condOpsJustified := totalOpsMaintenance < projectedSavingsUSD

	meetsTrigger := condHighSpend && condLowMargin && condMeaningfulSavings && condOpsJustified

	if meetsTrigger {
		// If traffic is highly dense (> 1000 jobs/day), dedicated cluster is viable
		if telemetry.JobsPerDay > 1000.0 {
			return ScaleEvaluationResult{
				ToolID:              telemetry.ToolID,
				Classification:      ClassSelfHostCandidate,
				CurrentAPISpend:     telemetry.ProviderSpendUSD,
				SelfHostCostUSD:     totalSelfHostCost,
				ProjectedSavingsUSD: projectedSavingsUSD,
				SavingsPercent:      savingsPercent,
				MeetsTriggerRules:   true,
				Reason:              "Sustained monthly spend exceeds $1,000, margin below 50%, and net self-host savings exceed 50% after ops costs",
			}
		}

		// Otherwise bursty/moderate traffic -> Serverless Container GPU
		return ScaleEvaluationResult{
			ToolID:              telemetry.ToolID,
			Classification:      ClassServerlessGPUCandidate,
			CurrentAPISpend:     telemetry.ProviderSpendUSD,
			SelfHostCostUSD:     totalSelfHostCost,
			ProjectedSavingsUSD: projectedSavingsUSD,
			SavingsPercent:      savingsPercent,
			MeetsTriggerRules:   true,
			Reason:              "Trigger conditions met, but bursty demand favors Serverless Container GPU (e.g. RunPod serverless / Modal) before dedicated clusters",
		}
	}

	// 3. Check Reliability / Multi-Provider Need
	if telemetry.SuccessRate < 95.0 && telemetry.JobsPerMonth >= 50 {
		return ScaleEvaluationResult{
			ToolID:              telemetry.ToolID,
			Classification:      ClassAddProvider,
			CurrentAPISpend:     telemetry.ProviderSpendUSD,
			SelfHostCostUSD:     totalSelfHostCost,
			ProjectedSavingsUSD: projectedSavingsUSD,
			SavingsPercent:      savingsPercent,
			MeetsTriggerRules:   false,
			Reason:              "Provider success rate is below 95%; prioritize multi-provider routing fallback over self-hosting",
		}
	}

	// 4. Default: KEEP_API
	reason := "API-first remains optimal: provider spend ($" +
		formatFloat(telemetry.ProviderSpendUSD) + "/mo) is below $1,000 threshold, gross margin (" +
		formatFloat(telemetry.GrossMarginPercent) + "%) is healthy, and self-hosting introduces unneeded ops overhead"

	return ScaleEvaluationResult{
		ToolID:              telemetry.ToolID,
		Classification:      ClassKeepAPI,
		CurrentAPISpend:     telemetry.ProviderSpendUSD,
		SelfHostCostUSD:     totalSelfHostCost,
		ProjectedSavingsUSD: projectedSavingsUSD,
		SavingsPercent:      savingsPercent,
		MeetsTriggerRules:   false,
		Reason:              reason,
	}
}

func formatFloat(v float64) string {
	return mathRoundString(v)
}

func mathRoundString(val float64) string {
	// Round to 2 decimal places
	rounded := math.Round(val*100) / 100
	var buf [32]byte
	b := buf[:0]
	return string(appendFloat(b, rounded))
}

func appendFloat(dst []byte, val float64) []byte {
	// Simple float formatting helper
	if val < 0 {
		dst = append(dst, '-')
		val = -val
	}
	intPart := int64(val)
	fracPart := int64(math.Round((val - float64(intPart)) * 100))
	if fracPart >= 100 {
		intPart++
		fracPart -= 100
	}
	dst = appendInt(dst, intPart)
	dst = append(dst, '.')
	if fracPart < 10 {
		dst = append(dst, '0')
	}
	dst = appendInt(dst, fracPart)
	return dst
}

func appendInt(dst []byte, n int64) []byte {
	if n == 0 {
		return append(dst, '0')
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + (n % 10))
		n /= 10
	}
	return append(dst, digits[i:]...)
}
