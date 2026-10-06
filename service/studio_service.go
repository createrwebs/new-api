package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	providers map[string]StudioProvider
}

func NewStudioService(providers ...StudioProvider) *StudioService {
	pMap := make(map[string]StudioProvider)
	for _, p := range providers {
		pMap[p.Name()] = p
	}
	return &StudioService{
		providers: pMap,
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

// SubmitJob handles the end-to-end atomic reservation, provider dispatch, and outcome reconciliation.
func (s *StudioService) SubmitJob(
	ctx context.Context,
	userId int,
	toolId string,
	idempotencyKey string,
	providerName string,
	inputParams map[string]interface{},
) (*model.StudioToolJob, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, errors.New("idempotency_key is required")
	}

	// 1. Check idempotency: Return existing job if duplicate submission
	existingJob, err := model.GetStudioJobByIdempotency(userId, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed checking idempotency: %w", err)
	}
	if existingJob != nil {
		return existingJob, nil
	}

	// 2. Lookup Tool Definition to determine quota price
	toolDef, err := model.GetStudioToolDefinition(toolId)
	if err != nil {
		return nil, fmt.Errorf("invalid tool requested: %w", err)
	}

	// 3. Select Provider
	if providerName == "" {
		providerName = "mock"
	}
	provider := s.GetProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("provider %s is not registered or unavailable", providerName)
	}

	// 4. Generate durable request ID and job ID
	jobId := fmt.Sprintf("job_%d_%s", common.GetTimestamp(), common.GetUUID()[:8])
	requestId := fmt.Sprintf("req_studio_%s", jobId)
	inputParamsJSON, _ := json.Marshal(inputParams)

	// 5. Pre-Consume / Reserve Quota in single Tora Wallet atomically
	quotaToReserve := toolDef.QuotaCost
	if quotaToReserve > 0 {
		err := model.PreConsumeUserWallet(requestId, userId, quotaToReserve)
		if err != nil {
			if errors.Is(err, model.ErrWalletQuotaInsufficient) {
				return nil, ErrInsufficientQuota
			}
			return nil, fmt.Errorf("quota reservation failed: %w", err)
		}
	}

	// 6. Record Job in Reserved State
	job := &model.StudioToolJob{
		Id:             jobId,
		UserId:         userId,
		ToolId:         toolId,
		RequestId:      requestId,
		IdempotencyKey: idempotencyKey,
		ProviderName:   providerName,
		Status:         model.StudioJobStatusReserved,
		ReservedQuota:  quotaToReserve,
		InputParams:    string(inputParamsJSON),
		CreatedAt:      common.GetTimestamp(),
		UpdatedAt:      common.GetTimestamp(),
	}

	if model.DB != nil {
		if err := model.DB.Create(job).Error; err != nil {
			// Rollback quota reservation if database insert fails
			if quotaToReserve > 0 {
				_ = model.RefundUserWalletPreConsume(requestId)
			}
			return nil, fmt.Errorf("failed persisting studio job: %w", err)
		}
	}

	// 7. Dispatch to External Provider
	submitResult, err := provider.Submit(ctx, job)
	if err != nil {
		if errors.Is(err, ErrProviderAmbiguous) {
			// Ambiguous timeout: do not refund yet, require background reconciliation
			job.Status = model.StudioJobStatusReconciling
			job.ErrorMessage = "Provider submission timed out; pending background reconciliation"
			if model.DB != nil {
				_ = model.DB.Save(job)
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
		}
		return job, err
	}

	// 8. Handle Provider Submit Result
	job.ProviderJobId = submitResult.ProviderJobId

	if submitResult.Status == "completed" {
		// Immediate synchronous completion: Settle reserved quota
		if quotaToReserve > 0 {
			if err := model.SettleUserWalletPreConsume(requestId); err != nil {
				common.SysError(fmt.Sprintf("failed settling wallet pre-consume for job %s: %v", jobId, err))
			}
		}
		job.Status = model.StudioJobStatusCompleted
		job.SettledQuota = quotaToReserve
		job.OutputResult = fmt.Sprintf(`{"output_url": "%s"}`, submitResult.OutputURL)
		job.CompletedAt = common.GetTimestamp()
	} else {
		// Asynchronous / queued generation
		job.Status = model.StudioJobStatusProcessing
	}

	if model.DB != nil {
		_ = model.DB.Save(job)
	}

	return job, nil
}

// PollJob inspects background job progress and settles/refunds upon terminal state.
func (s *StudioService) PollJob(ctx context.Context, jobId string) (*model.StudioToolJob, error) {
	job, err := model.GetStudioJobById(jobId)
	if err != nil {
		return nil, err
	}

	if job.Status == model.StudioJobStatusCompleted ||
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

	switch pollResult.Status {
	case "completed":
		if job.ReservedQuota > 0 && job.SettledQuota == 0 {
			_ = model.SettleUserWalletPreConsume(job.RequestId)
			job.SettledQuota = job.ReservedQuota
		}
		job.Status = model.StudioJobStatusCompleted
		job.OutputResult = fmt.Sprintf(`{"output_url": "%s"}`, pollResult.OutputURL)
		job.CompletedAt = common.GetTimestamp()

	case "failed":
		if job.ReservedQuota > 0 && job.SettledQuota == 0 {
			_ = model.RefundUserWalletPreConsume(job.RequestId)
		}
		job.Status = model.StudioJobStatusFailed
		job.ErrorMessage = pollResult.ErrorMessage

	case "processing", "queued":
		job.Status = model.StudioJobStatusProcessing
	}

	if model.DB != nil {
		_ = model.DB.Save(job)
	}

	return job, nil
}

// SeedDefaultTools populates the 10 MVP studio tools if not present.
func SeedDefaultTools(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	defaultTools := []model.StudioToolDefinition{
		{
			Id:           "image_generate_fast",
			Category:     "image",
			Name:         "Image Generate (Fast)",
			DisplayName:  "สร้างภาพแบบรวดเร็ว (Flux Schnell)",
			Description:  "สร้างภาพด้วยโมเดล Flux.1 Schnell ความเร็วสูง เหมาะสำหรับการดราฟต์ไอเดีย",
			CreditCost:   5,
			QuotaCost:    5000,
			PrimaryModel: "fal-ai/flux/schnell",
			IsActive:     true,
		},
		{
			Id:           "image_generate_pro",
			Category:     "image",
			Name:         "Image Generate (Pro)",
			DisplayName:  "สร้างภาพคุณภาพสูง (Flux Dev)",
			Description:  "สร้างภาพความละเอียดสูงด้วย Flux.1 Dev รายละเอียดคมชัด เหมาะสำหรับชิ้นงานจริง",
			CreditCost:   30,
			QuotaCost:    30000,
			PrimaryModel: "fal-ai/flux/dev",
			IsActive:     true,
		},
		{
			Id:           "background_remove",
			Category:     "utility",
			Name:         "Background Remove",
			DisplayName:  "ลบพื้นหลังอัจฉริยะ (BiRefNet)",
			Description:  "ลบพื้นหลังตัดขอบคมชัดด้วยอัลกอริทึม BiRefNet รองรับภาพสินค้าความละเอียด 4K",
			CreditCost:   10,
			QuotaCost:    10000,
			PrimaryModel: "fal-ai/birefnet",
			IsActive:     true,
		},
		{
			Id:           "image_upscale_4k",
			Category:     "utility",
			Name:         "Image Upscale (4K)",
			DisplayName:  "ขยายภาพคมชัด 4K (Clarity Upscaler)",
			Description:  "เพิ่มความละเอียดภาพขึ้น 4 เท่า พร้อมฟื้นฟูรายละเอียดและ Texture ของวัตถุ",
			CreditCost:   25,
			QuotaCost:    25000,
			PrimaryModel: "fal-ai/clarity-upscaler",
			IsActive:     true,
		},
		{
			Id:           "product_photo_studio",
			Category:     "image",
			Name:         "Product Photo Studio",
			DisplayName:  "สตูดิโอถ่ายภาพสินค้าโฆษณา",
			Description:  "วางสินค้าลงบนฉากจัดแสงระดับสตูดิโออัตโนมัติ สำหรับยิงแอด E-commerce",
			CreditCost:   50,
			QuotaCost:    50000,
			PrimaryModel: "fal-ai/product-photography",
			IsActive:     true,
		},
		{
			Id:           "text_to_video_fast",
			Category:     "video",
			Name:         "Text to Video (Fast)",
			DisplayName:  "สร้างวิดีโอจากข้อความ (LTX-Video)",
			Description:  "สร้างวิดีโอ 5 วินาทีความเร็วสูงด้วยโมเดล LTX-Video",
			CreditCost:   50,
			QuotaCost:    50000,
			PrimaryModel: "fal-ai/ltx-video",
			IsActive:     true,
		},
		{
			Id:           "video_generate_pro",
			Category:     "video",
			Name:         "Video Generate (Pro)",
			DisplayName:  "สร้างวิดีโอระดับมืออาชีพ (Wan 2.2)",
			Description:  "สร้างวิดีโอความละเอียดสูงสมจริงด้วย Wan 2.2 รองรับทั้ง Text-to-Video และ Image-to-Video",
			CreditCost:   125,
			QuotaCost:    125000,
			PrimaryModel: "wan-video/wan-2.2",
			IsActive:     true,
		},
	}

	for _, t := range defaultTools {
		var count int64
		db.Model(&model.StudioToolDefinition{}).Where("id = ?", t.Id).Count(&count)
		if count == 0 {
			now := common.GetTimestamp()
			t.CreatedAt = now
			t.UpdatedAt = now
			if err := db.Create(&t).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
