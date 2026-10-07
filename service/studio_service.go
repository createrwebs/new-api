package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

var (
	ErrInsufficientQuota = errors.New("insufficient Tora Credits / wallet quota to initiate media job")
)

// StudioService orchestrates job submission, quota reservation, provider execution, and atomic settlement.
type StudioService struct {
	providers       map[string]StudioProvider
	pricingEngine   *PricingEngine
	assistantEngine *StudioAssistantEngine
	router          *StudioRouter
}

func NewStudioService(providers ...StudioProvider) *StudioService {
	pMap := make(map[string]StudioProvider)
	for _, p := range providers {
		pMap[p.Name()] = p
	}
	svc := &StudioService{
		providers:     pMap,
		pricingEngine: NewPricingEngine(),
	}
	svc.assistantEngine = NewStudioAssistantEngine(svc)
	svc.router = NewStudioRouter(svc.providers)
	return svc
}

func (s *StudioService) RegisterProvider(provider StudioProvider) {
	if s.providers == nil {
		s.providers = make(map[string]StudioProvider)
	}
	s.providers[provider.Name()] = provider
	s.router = NewStudioRouter(s.providers)
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

func (s *StudioService) GetAssistantEngine() *StudioAssistantEngine {
	if s.assistantEngine == nil {
		s.assistantEngine = NewStudioAssistantEngine(s)
	}
	return s.assistantEngine
}

func (s *StudioService) GetRouter() *StudioRouter {
	if s.router == nil {
		s.router = NewStudioRouter(s.providers)
	}
	return s.router
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

	// 2. User Role & Admin check
	userIsAdmin := false
	if len(isAdmin) > 0 {
		userIsAdmin = isAdmin[0]
	}

	// 3. Lookup Tool Definition to determine quota price (Section 7)
	toolDef, err := model.GetStudioToolDefinition(toolId)
	if err != nil {
		if userIsAdmin {
			toolDef, err = model.GetStudioToolDefinitionAnyStatus(toolId)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid tool requested: %w", err)
		}
	}

	// Multi-reference validation (Queue 4: Product Studio)
	if toolId == "product-photo" && inputParams != nil {
		if _, hasLogo := inputParams["logo"]; hasLogo {
			return nil, errors.New("unreliable combination: direct in-model logo diffusion degrades brand typography; use post-composite vector overlay instead")
		}
		if _, hasLogoURL := inputParams["logo_url"]; hasLogoURL {
			return nil, errors.New("unreliable combination: direct in-model logo diffusion degrades brand typography; use post-composite vector overlay instead")
		}
		if refURL, ok := inputParams["reference_image_url"].(string); ok && refURL != "" {
			if err := ValidateExternalURL(refURL); err != nil {
				return nil, fmt.Errorf("invalid reference background URL: %w", err)
			}
		}
		if bgURL, ok := inputParams["background_url"].(string); ok && bgURL != "" {
			if err := ValidateExternalURL(bgURL); err != nil {
				return nil, fmt.Errorf("invalid reference background URL: %w", err)
			}
		}
	}

	isVideoJob := toolDef.Category == "video" || toolId == "image-to-video"

	// Video-specific input & cost guards (Queue 5: Video Foundation)
	if isVideoJob {
		// Guard 1: Asset Pipeline - Reject base64 JSON
		if inputParams != nil {
			for _, v := range inputParams {
				if s, ok := v.(string); ok && strings.HasPrefix(s, "data:") {
					return nil, errors.New("base64 media input is not permitted for video workflows; please upload media to obtain an asset URL")
				}
			}
		}

		// Guard 2: Maximum Duration Guard (Initial video release: max 5s)
		duration := 5
		if d, ok := inputParams["duration"].(float64); ok && d > 0 {
			duration = int(d)
		} else if d, ok := inputParams["duration"].(int); ok && d > 0 {
			duration = d
		} else if d, ok := inputParams["duration_sec"].(float64); ok && d > 0 {
			duration = int(d)
		} else if d, ok := inputParams["duration_sec"].(int); ok && d > 0 {
			duration = d
		}
		if duration > 5 {
			return nil, errors.New("duration exceeds maximum limit: initial video release supports maximum 5 seconds")
		}

		// Guard 3: Maximum Resolution Guard (Initial video release: 720p or 1080p)
		if res, ok := inputParams["resolution"].(string); ok && res != "" {
			lowerRes := strings.ToLower(strings.TrimSpace(res))
			if lowerRes != "720p" && lowerRes != "1080p" {
				return nil, errors.New("resolution exceeds maximum limit: allowed resolutions are 720p and 1080p")
			}
		}

		// Guard 4: Per-User Concurrent Video Limit Guard (1 for regular user, 2 for admin)
		maxConcurrentVideo := 1
		if userIsAdmin {
			maxConcurrentVideo = 2
		}
		if model.DB != nil {
			var activeCount int64
			videoToolIds := []string{"image-to-video", "video-generate", "text-to-video", "lip-sync", "talking-avatar", "video-upscale"}
			model.DB.Model(&model.StudioToolJob{}).
				Where("user_id = ? AND tool_id IN (?) AND status IN (?)",
					userId,
					videoToolIds,
					[]string{
						string(model.StudioJobStatusReserved),
						string(model.StudioJobStatusSubmitting),
						string(model.StudioJobStatusProcessing),
					},
				).Count(&activeCount)
			if int(activeCount) >= maxConcurrentVideo {
				return nil, errors.New("concurrent video limit reached: you already have an active video job in progress; please wait for it to complete")
			}

			// Guard 5: Daily Provider Spend Guard
			dailyLimitUSD := 50.0 // Default $50/day
			if customLimitStr := os.Getenv("STUDIO_DAILY_VIDEO_SPEND_LIMIT_USD"); customLimitStr != "" {
				if parsedLimit, err := strconv.ParseFloat(customLimitStr, 64); err == nil && parsedLimit > 0 {
					dailyLimitUSD = parsedLimit
				}
			}
			todayStart := time.Now().Truncate(24 * time.Hour).Unix()
			var currentDailySpend float64
			var videoJobsToday []model.StudioToolJob
			model.DB.Model(&model.StudioToolJob{}).
				Where("tool_id IN (?) AND created_at >= ? AND status != ?",
					videoToolIds,
					todayStart,
					string(model.StudioJobStatusFailed),
				).Find(&videoJobsToday)
			for _, vj := range videoJobsToday {
				if vj.PricingSnapshot != "" {
					var sn model.StudioPricingSnapshot
					if json.Unmarshal([]byte(vj.PricingSnapshot), &sn) == nil && sn.ProviderEstimatedCostUSD > 0 {
						currentDailySpend += sn.ProviderEstimatedCostUSD
					}
				}
			}
			if currentDailySpend >= dailyLimitUSD {
				return nil, errors.New("daily video capacity reached: daily provider spend guard threshold has been reached; please try again tomorrow or contact support")
			}
		}

		// Guard 6: Quote Expiry Guard (if quote_id provided)
		if quoteId, ok := inputParams["quote_id"].(string); ok && strings.TrimSpace(quoteId) != "" {
			quoteSnapshot, err := s.pricingEngine.GetQuote(quoteId)
			if err != nil {
				return nil, fmt.Errorf("quote validation failed: %w", err)
			}
			if quoteSnapshot.ToolID != "" && quoteSnapshot.ToolID != toolId {
				return nil, errors.New("quote tool mismatch")
			}
		}
	}

	// 4. Select Provider & guard mock usage in production
	var fallbackProviders []string
	if providerName == "" || providerName == "auto" {
		tier := QualityTierQuality
		if t, ok := inputParams["quality_tier"].(string); ok && t != "" {
			tier = QualityTier(strings.ToUpper(t))
		}
		var routeErr error
		providerName, fallbackProviders, routeErr = s.GetRouter().SelectProvider(toolId, tier, inputParams)
		if routeErr != nil || providerName == "" {
			providerName = toolDef.PrimaryProvider
		}
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

	// 8. Dispatch to External Provider (transition to SUBMITTING) with safe fallback
	job.Status = model.StudioJobStatusSubmitting
	submitResult, chosenProvider, err := s.GetRouter().ExecuteWithSafeFallback(ctx, job, providerName, fallbackProviders)
	if chosenProvider != "" && chosenProvider != job.ProviderName {
		job.ProviderName = chosenProvider
	}
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
		if isVideoJob || strings.HasSuffix(strings.ToLower(submitResult.OutputURL), ".mp4") {
			posterURL := strings.Replace(submitResult.OutputURL, ".mp4", "_poster.jpg", 1)
			job.OutputResult = fmt.Sprintf(`{"output_url": "%s", "video_url": "%s", "thumbnail_url": "%s", "poster_url": "%s"}`,
				submitResult.OutputURL, submitResult.OutputURL, posterURL, posterURL)
		} else if len(submitResult.OutputURLs) > 1 {
			urlsJSON, _ := json.Marshal(submitResult.OutputURLs)
			job.OutputResult = fmt.Sprintf(`{"output_url": "%s", "output_urls": %s, "variants": %s}`, submitResult.OutputURL, string(urlsJSON), string(urlsJSON))
		} else {
			job.OutputResult = fmt.Sprintf(`{"output_url": "%s"}`, submitResult.OutputURL)
		}
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
		if job.ToolId == "image-to-video" || strings.HasSuffix(job.ToolId, "-video") || strings.HasSuffix(strings.ToLower(pollResult.OutputURL), ".mp4") {
			posterURL := strings.Replace(pollResult.OutputURL, ".mp4", "_poster.jpg", 1)
			job.OutputResult = fmt.Sprintf(`{"output_url": "%s", "video_url": "%s", "thumbnail_url": "%s", "poster_url": "%s"}`,
				pollResult.OutputURL, pollResult.OutputURL, posterURL, posterURL)
		} else if len(pollResult.OutputURLs) > 1 {
			urlsJSON, _ := json.Marshal(pollResult.OutputURLs)
			job.OutputResult = fmt.Sprintf(`{"output_url": "%s", "output_urls": %s, "variants": %s}`, pollResult.OutputURL, string(urlsJSON), string(urlsJSON))
		} else {
			job.OutputResult = fmt.Sprintf(`{"output_url": "%s"}`, pollResult.OutputURL)
		}
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

// HandleWebhook reconciles an incoming asynchronous webhook callback with atomic settlement.
func (s *StudioService) HandleWebhook(ctx context.Context, providerName string, providerJobId string, status string, outputURL string, errorMsg string) (*model.StudioToolJob, error) {
	job, err := model.GetStudioJobByProviderJobId(providerJobId)
	if err != nil {
		return nil, fmt.Errorf("job not found for provider_job_id %s: %w", providerJobId, err)
	}

	// Idempotent short-circuit: if job is already in terminal state, do not re-process
	if job.Status == model.StudioJobStatusSucceeded ||
		job.Status == model.StudioJobStatusFailed ||
		job.Status == model.StudioJobStatusCancelled {
		return job, nil
	}

	normalizedStatus := strings.ToLower(status)
	oldStatus := job.Status

	if normalizedStatus == "completed" || normalizedStatus == "succeeded" || normalizedStatus == "ok" {
		// Atomic check-and-settle: only settle if not already settled
		if job.ReservedQuota > 0 && job.SettledQuota == 0 {
			if err := model.SettleUserWalletPreConsume(job.RequestId); err != nil {
				common.SysError(fmt.Sprintf("failed settling pre-consume on webhook for job %s: %v", job.Id, err))
			}
			job.SettledQuota = job.ReservedQuota
		}
		job.Status = model.StudioJobStatusSucceeded
		job.OutputResult = fmt.Sprintf(`{"output_url": "%s"}`, outputURL)
		job.CompletedAt = common.GetTimestamp()

		if model.DB != nil {
			s.recordEvent(model.DB, job.Id, "JOB_SUCCEEDED_WEBHOOK", oldStatus, model.StudioJobStatusSucceeded, outputURL)
			toolDef, _ := model.GetStudioToolDefinition(job.ToolId)
			s.recordCostSnapshot(model.DB, job, toolDef, providerName)
			_ = model.DB.Save(job)
		}
	} else if normalizedStatus == "failed" || normalizedStatus == "error" {
		if job.ReservedQuota > 0 && job.SettledQuota == 0 {
			_ = model.RefundUserWalletPreConsume(job.RequestId)
		}
		job.Status = model.StudioJobStatusFailed
		job.ErrorMessage = errorMsg

		if model.DB != nil {
			s.recordEvent(model.DB, job.Id, "JOB_FAILED_WEBHOOK", oldStatus, model.StudioJobStatusFailed, errorMsg)
			_ = model.DB.Save(job)
		}
	}

	return job, nil
}

// ReconcileStaleJobs recovers orphaned jobs after process restarts without double charging or early refunds.
func (s *StudioService) ReconcileStaleJobs(ctx context.Context, olderThanSeconds int64) (int, error) {
	if model.DB == nil {
		return 0, nil
	}

	reconciled := 0

	// 1. Recover stranded RESERVED jobs: if a job was RESERVED > olderThanSeconds ago and never dispatched (process crashed), refund safely
	staleReserved, err := model.GetStaleStudioJobs(model.StudioJobStatusReserved, olderThanSeconds)
	if err == nil {
		for _, job := range staleReserved {
			if job.ReservedQuota > 0 && job.SettledQuota == 0 {
				_ = model.RefundUserWalletPreConsume(job.RequestId)
			}
			job.Status = model.StudioJobStatusFailed
			job.ErrorMessage = "Process crashed prior to provider dispatch; reservation safely refunded"
			_ = model.DB.Save(job)
			s.recordEvent(model.DB, job.Id, "RESTART_RECOVERY_REFUND", model.StudioJobStatusReserved, model.StudioJobStatusFailed, "server restarted")
			reconciled++
		}
	}

	// 2. Recover stranded SUBMITTING / PROCESSING jobs: query upstream provider via Poll
	staleProcessing, err := model.GetStaleStudioJobs(model.StudioJobStatusProcessing, olderThanSeconds)
	if err == nil {
		for _, job := range staleProcessing {
			if job.ProviderJobId != "" {
				updatedJob, err := s.PollJob(ctx, job.Id, job.UserId, true)
				if err == nil && (updatedJob.Status == model.StudioJobStatusSucceeded || updatedJob.Status == model.StudioJobStatusFailed) {
					reconciled++
				}
			}
		}
	}

	return reconciled, nil
}
