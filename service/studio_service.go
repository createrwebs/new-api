package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

var (
	ErrInsufficientQuota = errors.New("insufficient Tora Credits / wallet quota to initiate media job")
)

// StudioService orchestrates job submission, quota reservation, provider execution, and atomic settlement.
type StudioService struct {
	providers     map[string]StudioProvider
	pricingEngine *PricingEngine
}

func NewStudioService(providers ...StudioProvider) *StudioService {
	pMap := make(map[string]StudioProvider)
	for _, p := range providers {
		pMap[p.Name()] = p
	}
	return &StudioService{
		providers:     pMap,
		pricingEngine: NewPricingEngine(),
	}
}

func (s *StudioService) RegisterProvider(provider StudioProvider) {
	if s.providers == nil {
		s.providers = make(map[string]StudioProvider)
	}
	s.providers[provider.Name()] = provider
}

func (s *StudioService) GetProvider(name string) StudioProvider {
	if s.providers == nil {
		return nil
	}
	return s.providers[name]
}

func (s *StudioService) GetPricingEngine() *PricingEngine {
	if s.pricingEngine == nil {
		s.pricingEngine = NewPricingEngine()
	}
	return s.pricingEngine
}

// recordEvent persists an audit event for state transitions.
func (s *StudioService) recordEvent(db *gorm.DB, jobId string, eventType string, oldStatus model.StudioJobStatus, newStatus model.StudioJobStatus, payload string) {
	if db == nil {
		return
	}
	event := model.StudioJobEvent{
		JobId:     jobId,
		EventType: eventType,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Payload:   payload,
		CreatedAt: common.GetTimestamp(),
	}
	_ = db.Create(&event)
}

// SubmitJob handles the end-to-end atomic reservation, provider dispatch, and outcome reconciliation.
func (s *StudioService) SubmitJob(
	ctx context.Context,
	userId int,
	toolId string,
	templateId string,
	idempotencyKey string,
	providerName string,
	inputParams map[string]interface{},
	planMultiplier float64,
	clientIP string,
	isAdmin ...bool,
) (*model.StudioToolJob, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, errors.New("idempotency_key is required")
	}

	// 1. Check idempotency: Return existing job if duplicate submission (Section 16)
	existingJob, err := model.GetStudioJobByIdempotency(userId, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed checking idempotency: %w", err)
	}
	if existingJob != nil {
		return existingJob, nil
	}

	// 2. Lookup Tool Definition to determine quota price (Section 7)
	toolDef, err := model.GetStudioToolDefinition(toolId)
	if err != nil {
		return nil, fmt.Errorf("invalid tool requested: %w", err)
	}

	// 3. Select Provider & guard mock usage in production
	userIsAdmin := false
	if len(isAdmin) > 0 {
		userIsAdmin = isAdmin[0]
	}

	if providerName == "" {
		providerName = toolDef.PrimaryProvider
	}
	if providerName == "" {
		providerName = "mock"
	}

	if providerName == "mock" && !userIsAdmin {
		// Mock provider is strictly prohibited for non-admin requests in release mode
		if !common.DebugEnabled && os.Getenv("GIN_MODE") == "release" {
			return nil, errors.New("mock provider is disabled in production environment for non-admin requests")
		}
	}

	provider := s.GetProvider(providerName)
	if provider == nil {
		// Fallback to mock only if registered and permitted
		if !userIsAdmin && !common.DebugEnabled && os.Getenv("GIN_MODE") == "release" {
			return nil, fmt.Errorf("provider %s is not registered or unavailable", providerName)
		}
		provider = s.GetProvider("mock")
		if provider == nil {
			return nil, fmt.Errorf("provider %s is not registered or unavailable", providerName)
		}
		providerName = "mock"
	}

	// 4. Calculate dynamic pricing snapshot & enforce profitability guard (Section 14 & 35)
	pricingSnapshot, err := s.pricingEngine.CalculatePriceWithInputs(toolDef, inputParams, planMultiplier)
	if err != nil {
		return nil, fmt.Errorf("failed calculating pricing snapshot: %w", err)
	}

	minMargin := toolDef.MarginPercent
	if minMargin <= 0 {
		minMargin = MinGrossMarginFloor
	}
	if err := s.pricingEngine.ValidateProfitability(pricingSnapshot, minMargin); err != nil {
		return nil, err
	}

	quotaToReserve := pricingSnapshot.ChargedQuota
	pricingSnapshotJSON, _ := json.Marshal(pricingSnapshot)

	// 5. Generate durable request ID and job ID
	jobId := fmt.Sprintf("job_%d_%s", common.GetTimestamp(), common.GetUUID()[:8])
	requestId := fmt.Sprintf("req_studio_%s", jobId)
	inputParamsJSON, _ := json.Marshal(inputParams)

	// 6. Pre-Consume / Reserve Quota in single Tora Wallet atomically (Section 14)
	if quotaToReserve > 0 {
		err := model.PreConsumeUserWallet(requestId, userId, quotaToReserve)
		if err != nil {
			if errors.Is(err, model.ErrWalletQuotaInsufficient) {
				return nil, ErrInsufficientQuota
			}
			return nil, fmt.Errorf("quota reservation failed: %w", err)
		}
	}

	// 7. Record Job in RESERVED State
	job := &model.StudioToolJob{
		Id:              jobId,
		UserId:          userId,
		ToolId:          toolId,
		TemplateId:      templateId,
		RequestId:       requestId,
		IdempotencyKey:  idempotencyKey,
		ProviderName:    providerName,
		Status:          model.StudioJobStatusReserved,
		ReservedQuota:   quotaToReserve,
		InputParams:     string(inputParamsJSON),
		PricingSnapshot: string(pricingSnapshotJSON),
		RiskClass:       toolDef.RiskClass,
		ClientIP:        clientIP,
		CreatedAt:       common.GetTimestamp(),
		UpdatedAt:       common.GetTimestamp(),
	}

	if model.DB != nil {
		if err := model.DB.Create(job).Error; err != nil {
			// Rollback quota reservation if database insert fails
			if quotaToReserve > 0 {
				_ = model.RefundUserWalletPreConsume(requestId)
			}
			return nil, fmt.Errorf("failed persisting studio job: %w", err)
		}
		s.recordEvent(model.DB, jobId, "WALLET_RESERVED", model.StudioJobStatusCreated, model.StudioJobStatusReserved, "")
	}

	// 8. Dispatch to External Provider (transition to SUBMITTING)
	job.Status = model.StudioJobStatusSubmitting
	submitResult, err := provider.Submit(ctx, job)
	if err != nil {
		if errors.Is(err, ErrProviderAmbiguous) {
			// Section 15: Ambiguous timeout: do NOT refund yet, mark AMBIGUOUS_SUBMISSION
			job.Status = model.StudioJobStatusAmbiguousSubmission
			job.ErrorMessage = "Provider submission timed out; pending background reconciliation"
			if model.DB != nil {
				_ = model.DB.Save(job)
				s.recordEvent(model.DB, jobId, "SUBMISSION_AMBIGUOUS", model.StudioJobStatusSubmitting, model.StudioJobStatusAmbiguousSubmission, err.Error())
			}
			return job, nil
		}

		// Permanent / definite failure: execute full atomic refund
		if quotaToReserve > 0 {
			_ = model.RefundUserWalletPreConsume(requestId)
		}
		job.Status = model.StudioJobStatusFailed
		job.ErrorMessage = err.Error()
		if model.DB != nil {
			_ = model.DB.Save(job)
			s.recordEvent(model.DB, jobId, "WALLET_REFUNDED_FAILURE", model.StudioJobStatusSubmitting, model.StudioJobStatusFailed, err.Error())
		}
		return job, err
	}

	// 9. Handle Provider Submit Result
	job.ProviderJobId = submitResult.ProviderJobId

	if submitResult.Status == "completed" {
		// Immediate synchronous completion: Settle reserved quota
		if quotaToReserve > 0 {
			if err := model.SettleUserWalletPreConsume(requestId); err != nil {
				common.SysError(fmt.Sprintf("failed settling wallet pre-consume for job %s: %v", jobId, err))
			}
		}
		job.Status = model.StudioJobStatusSucceeded
		job.SettledQuota = quotaToReserve
		job.OutputResult = fmt.Sprintf(`{"output_url": "%s"}`, submitResult.OutputURL)
		job.CompletedAt = common.GetTimestamp()

		if model.DB != nil {
			s.recordEvent(model.DB, jobId, "JOB_SUCCEEDED", model.StudioJobStatusSubmitting, model.StudioJobStatusSucceeded, submitResult.OutputURL)
			s.recordCostSnapshot(model.DB, job, toolDef, providerName)
		}
	} else {
		// Asynchronous / queued generation
		job.Status = model.StudioJobStatusProcessing
		if model.DB != nil {
			s.recordEvent(model.DB, jobId, "JOB_PROCESSING", model.StudioJobStatusSubmitting, model.StudioJobStatusProcessing, submitResult.ProviderJobId)
		}
	}

	if model.DB != nil {
		_ = model.DB.Save(job)
	}

	return job, nil
}

// recordCostSnapshot captures financial telemetry for margin analytics (Section 19, 34, 35).
func (s *StudioService) recordCostSnapshot(db *gorm.DB, job *model.StudioToolJob, toolDef *model.StudioToolDefinition, providerName string) {
	if db == nil || toolDef == nil {
		return
	}

	costUSD := 0.005
	var snapshot model.StudioPricingSnapshot
	if job.PricingSnapshot != "" && json.Unmarshal([]byte(job.PricingSnapshot), &snapshot) == nil && snapshot.ProviderEstimatedCostUSD > 0 {
		costUSD = snapshot.ProviderEstimatedCostUSD
	} else if p, ok := s.providers[providerName]; ok {
		if fal, isFal := p.(*FalProvider); isFal {
			costUSD = fal.EstimateCost(toolDef.PrimaryModel)
		}
	}

	revenueUSD := float64(job.SettledQuota) / common.QuotaPerUnit
	marginUSD := revenueUSD - costUSD
	marginPercent := 0.0
	if revenueUSD > 0 {
		marginPercent = (marginUSD / revenueUSD) * 100.0
	}

	costRecord := model.StudioCostSnapshot{
		JobId:              job.Id,
		ToolId:             job.ToolId,
		ProviderName:       providerName,
		ProviderJobId:      job.ProviderJobId,
		CostUSD:            costUSD,
		QuotaCost:          job.SettledQuota,
		ToraRevenueUSD:     revenueUSD,
		GrossProfitUSD:     marginUSD,
		GrossMarginPercent: marginPercent,
		MarginUSD:          marginUSD,
		MarginPercent:      marginPercent,
		SnapshotAt:         common.GetTimestamp(),
	}
	_ = db.Create(&costRecord)
}

// PollJob inspects background job progress and settles/refunds upon terminal state.
func (s *StudioService) PollJob(ctx context.Context, jobId string, userId int, isAdmin bool) (*model.StudioToolJob, error) {
	job, err := model.GetStudioJobById(jobId, userId, isAdmin)
	if err != nil {
		return nil, err
	}

	if job.Status == model.StudioJobStatusSucceeded ||
		job.Status == model.StudioJobStatusFailed ||
		job.Status == model.StudioJobStatusCancelled {
		return job, nil
	}

	provider := s.GetProvider(job.ProviderName)
	if provider == nil {
		return nil, fmt.Errorf("provider %s unavailable for polling", job.ProviderName)
	}

	pollResult, err := provider.Poll(ctx, job.ProviderJobId)
	if err != nil {
		return job, err
	}

	oldStatus := job.Status
	switch pollResult.Status {
	case "completed":
		if job.ReservedQuota > 0 && job.SettledQuota == 0 {
			_ = model.SettleUserWalletPreConsume(job.RequestId)
			job.SettledQuota = job.ReservedQuota
		}
		job.Status = model.StudioJobStatusSucceeded
		job.OutputResult = fmt.Sprintf(`{"output_url": "%s"}`, pollResult.OutputURL)
		job.CompletedAt = common.GetTimestamp()

		if model.DB != nil {
			s.recordEvent(model.DB, jobId, "JOB_SUCCEEDED_POLL", oldStatus, model.StudioJobStatusSucceeded, pollResult.OutputURL)
			toolDef, _ := model.GetStudioToolDefinition(job.ToolId)
			s.recordCostSnapshot(model.DB, job, toolDef, job.ProviderName)
		}

	case "failed":
		if job.ReservedQuota > 0 && job.SettledQuota == 0 {
			_ = model.RefundUserWalletPreConsume(job.RequestId)
		}
		job.Status = model.StudioJobStatusFailed
		job.ErrorMessage = pollResult.ErrorMessage

		if model.DB != nil {
			s.recordEvent(model.DB, jobId, "JOB_FAILED_POLL", oldStatus, model.StudioJobStatusFailed, pollResult.ErrorMessage)
		}

	case "processing", "queued":
		job.Status = model.StudioJobStatusProcessing
	}

	if model.DB != nil {
		_ = model.DB.Save(job)
	}

	return job, nil
}

// CancelJob cancels an active job and refunds reserved quota if applicable.
func (s *StudioService) CancelJob(ctx context.Context, jobId string, userId int, isAdmin bool) (*model.StudioToolJob, error) {
	job, err := model.GetStudioJobById(jobId, userId, isAdmin)
	if err != nil {
		return nil, err
	}

	if job.Status == model.StudioJobStatusSucceeded || job.Status == model.StudioJobStatusFailed || job.Status == model.StudioJobStatusCancelled {
		return job, nil // Already in terminal state
	}

	oldStatus := job.Status
	if provider := s.GetProvider(job.ProviderName); provider != nil {
		_ = provider.Cancel(ctx, job.ProviderJobId)
	}

	// Refund reserved quota upon cancellation
	if job.ReservedQuota > 0 && job.SettledQuota == 0 {
		_ = model.RefundUserWalletPreConsume(job.RequestId)
	}

	job.Status = model.StudioJobStatusCancelled
	job.UpdatedAt = common.GetTimestamp()

	if model.DB != nil {
		_ = model.DB.Save(job)
		s.recordEvent(model.DB, jobId, "JOB_CANCELLED", oldStatus, model.StudioJobStatusCancelled, "")
	}

	return job, nil
}
