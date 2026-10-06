package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDBForStudio(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:test_studio_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	oldDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
		_ = sqlDB.Close()
	})

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.WalletPreConsumeRecord{},
	))
	require.NoError(t, model.EnsureStudioTables(db))
	require.NoError(t, SeedStudioCatalog(db))

	return db
}

func TestStudioService_InstantSuccess_SettlesQuota(t *testing.T) {
	db := setupTestDBForStudio(t)

	// 1. Create user with 50,000 Quota units (approx $0.10)
	user := model.User{
		Username: "studio_creator_1",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// 2. Submit fast generation job (5,000 Quota units = 5 Tora Credits)
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"tpl-img-cinematic",
		"idemp_key_success_1",
		"mock",
		map[string]interface{}{"prompt": "Modern AI architectural datacenter"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	require.NotNil(t, job)

	// 3. Assertions on Job State (Section 13)
	assert.Equal(t, model.StudioJobStatusSucceeded, job.Status)
	assert.Equal(t, 5000, job.ReservedQuota)
	assert.Equal(t, 5000, job.SettledQuota)
	assert.Contains(t, job.OutputResult, "https://cdn.toraapi.com/mock-assets/")

	// 4. Assertions on Wallet Settlement (Section 14)
	remainingQuota, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 45000, remainingQuota, "User quota should be permanently reduced by settled amount")

	// 5. Assertions on WalletPreConsumeRecord
	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).First(&preRecord).Error)
	assert.Equal(t, "settled", preRecord.Status)
	assert.Equal(t, 5000, preRecord.PreConsumed)

	// 6. Assertions on Cost Snapshot (Section 35)
	var snapshot model.StudioCostSnapshot
	require.NoError(t, db.Where("job_id = ?", job.Id).First(&snapshot).Error)
	assert.Equal(t, 5000, snapshot.QuotaCost)
	assert.True(t, snapshot.MarginPercent > 0)
}

func TestStudioService_PermanentFail_RefundsQuota(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "studio_creator_2",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModePermanentFail)
	studioSvc := NewStudioService(mockProvider)

	// Submit job that will fail upstream permanently
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"product-photo", // 50,000 Quota units
		"tpl-prod-white-studio",
		"idemp_key_fail_1",
		"mock",
		map[string]interface{}{"prompt": "Trigger permanent upstream error"},
		1.0,
		"127.0.0.1",
	)
	require.Error(t, err)
	require.NotNil(t, job)

	// Job should be marked failed
	assert.Equal(t, model.StudioJobStatusFailed, job.Status)
	assert.Equal(t, 50000, job.ReservedQuota)
	assert.Equal(t, 0, job.SettledQuota)

	// User quota should be 100% refunded back to 50,000 (Section 14)
	remainingQuota, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, remainingQuota, "User quota must be restored upon permanent provider failure")

	// Pre-consume record must be refunded
	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).First(&preRecord).Error)
	assert.Equal(t, "refunded", preRecord.Status)
}

func TestStudioService_InsufficientQuota_EarlyRejection(t *testing.T) {
	db := setupTestDBForStudio(t)

	// User has only 2,000 Quota units
	user := model.User{
		Username: "studio_broke_user",
		Quota:    2000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// Tool costs 5,000 Quota units
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		"idemp_key_broke_1",
		"mock",
		map[string]interface{}{"prompt": "Should fail before provider dispatch"},
		1.0,
		"127.0.0.1",
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInsufficientQuota))
	assert.Nil(t, job)

	// Provider should never have been invoked
	assert.Equal(t, 0, mockProvider.SubmitCalls, "Provider must not be called when quota is insufficient")

	// User balance should remain completely untouched
	remainingQuota, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 2000, remainingQuota)
}

func TestStudioService_Idempotency_PreventsDoubleCharge(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "studio_idemp_user",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	const sharedIdempKey = "idemp_unique_tx_999"

	// First submission
	job1, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate", // 5,000 Quota units
		"",
		sharedIdempKey,
		"mock",
		map[string]interface{}{"prompt": "Idempotent request test"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	require.NotNil(t, job1)
	assert.Equal(t, model.StudioJobStatusSucceeded, job1.Status)

	// Check balance after first call
	bal1, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 45000, bal1)

	// Second submission with identical idempotency key (Section 16)
	job2, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		sharedIdempKey,
		"mock",
		map[string]interface{}{"prompt": "Idempotent request test"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	require.NotNil(t, job2)
	assert.Equal(t, job1.Id, job2.Id, "Duplicate submission must return exact same job")

	// Provider should have been called only once
	assert.Equal(t, 1, mockProvider.SubmitCalls)

	// User balance must NOT be deducted again
	bal2, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 45000, bal2, "User balance must not be deducted on duplicate idempotent call")
}

func TestStudioService_DelayedSuccess_PollSettles(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "studio_async_user",
		Quota:    150000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	// Enable image-to-video in test fixture
	require.NoError(t, db.Model(&model.StudioToolDefinition{}).Where("id = ?", "image-to-video").Updates(map[string]interface{}{"is_enabled": true, "is_public": true}).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	// 1. Submit async video generation (125,000 Quota units)
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-to-video",
		"tpl-vid-product-ad-5s",
		"idemp_async_video_1",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/shoe.png"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)
	assert.Equal(t, 125000, job.ReservedQuota)
	assert.Equal(t, 0, job.SettledQuota)

	// Quota is reserved (125,000 pending)
	balAfterSubmit, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 25000, balAfterSubmit)

	// 2. Poll 1: Still processing
	jobPoll1, err := studioSvc.PollJob(context.Background(), job.Id, user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, jobPoll1.Status)
	assert.Equal(t, 0, jobPoll1.SettledQuota)

	// 3. Poll 2: Transitions to succeeded and settles
	jobPoll2, err := studioSvc.PollJob(context.Background(), job.Id, user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, jobPoll2.Status)
	assert.Equal(t, 125000, jobPoll2.SettledQuota)
	assert.Contains(t, jobPoll2.OutputResult, "delayed_output.png")

	// Final verification of settled record
	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).First(&preRecord).Error)
	assert.Equal(t, "settled", preRecord.Status)
}

func TestStudioService_AmbiguousSubmission_HoldsReservation(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "studio_ambiguous_user",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeAmbiguousFail)
	studioSvc := NewStudioService(mockProvider)

	// Section 15: Submit job with ambiguous upstream timeout
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		"idemp_ambiguous_1",
		"mock",
		map[string]interface{}{"prompt": "Test timeout"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err) // Returns job in ambiguous state without immediate error
	require.NotNil(t, job)

	// Status must be AMBIGUOUS_SUBMISSION, NOT refunded yet
	assert.Equal(t, model.StudioJobStatusAmbiguousSubmission, job.Status)
	assert.Equal(t, 5000, job.ReservedQuota)
	assert.Equal(t, 0, job.SettledQuota)

	// Quota remains reserved (45,000 user balance) to prevent double generation or theft
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 45000, bal)

	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).First(&preRecord).Error)
	assert.Equal(t, "pending", preRecord.Status)
}

func TestStudioService_CancelJob_RefundsQuota(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "studio_cancel_user",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		"idemp_cancel_1",
		"mock",
		map[string]interface{}{"prompt": "Test cancellation"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)

	// Cancel active job
	cancelledJob, err := studioSvc.CancelJob(context.Background(), job.Id, user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusCancelled, cancelledJob.Status)

	// Quota is 100% refunded back to 50,000
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, bal)

	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).First(&preRecord).Error)
	assert.Equal(t, "refunded", preRecord.Status)
}

func TestStudioSecurity_ValidateExternalURL_BlocksSSRF(t *testing.T) {
	// Section 30: Test SSRF blocklist
	assert.ErrorIs(t, ValidateExternalURL("http://127.0.0.1:8080/image.png"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://localhost:3000/photo.jpg"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://169.254.169.254/latest/meta-data"), ErrSSRFForbidden)
	assert.Error(t, ValidateExternalURL("ftp://malicious.com/file.png"))
}

func TestStudioSecurity_ValidateMagicBytes_RejectsExecutable(t *testing.T) {
	// Valid PNG header
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	mime, err := ValidateMagicBytes(pngBytes)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mime)

	// Malicious ELF executable
	elfBytes := []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err = ValidateMagicBytes(elfBytes)
	assert.ErrorIs(t, err, ErrInvalidFileType)
}

func TestStudioService_IDOR_AccessControl(t *testing.T) {
	db := setupTestDBForStudio(t)

	user1 := model.User{Username: "user_owner", AffCode: "AFF_OWNER", Quota: 50000, Status: common.UserStatusEnabled}
	user2 := model.User{Username: "user_attacker", AffCode: "AFF_ATTACKER", Quota: 50000, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user1).Error)
	require.NoError(t, db.Create(&user2).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user1.Id,
		"image-generate",
		"",
		"idemp_idor_1",
		"mock",
		map[string]interface{}{"prompt": "Confidential portrait"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)

	// User 2 attempts to query User 1's job (Section 20: IDOR test)
	_, err = studioSvc.PollJob(context.Background(), job.Id, user2.Id, false)
	assert.ErrorIs(t, err, model.ErrStudioForbiddenAccess)

	// User 2 attempts to cancel User 1's job
	_, err = studioSvc.CancelJob(context.Background(), job.Id, user2.Id, false)
	assert.ErrorIs(t, err, model.ErrStudioForbiddenAccess)

	// Admin is allowed to view
	adminJob, err := studioSvc.PollJob(context.Background(), job.Id, user2.Id, true)
	require.NoError(t, err)
	assert.Equal(t, job.Id, adminJob.Id)
}

func TestStudioPricing_QuoteGenerationAndTTL(t *testing.T) {
	pe := NewPricingEngine()
	toolDef := &model.StudioToolDefinition{
		Id:              "image-to-video",
		PrimaryProvider: "fal",
		PrimaryModel:    "wan-video/wan-2.2",
		CreditCost:      125,
		QuotaCost:       125000,
		Category:        "video",
		MarginPercent:   68.0,
	}

	// 1. Generate Quote for 10-second video (duration scaled)
	params := map[string]interface{}{"duration": 10}
	snapshot, err := pe.CalculatePriceWithInputs(toolDef, params, 1.0)
	require.NoError(t, err)
	require.NotNil(t, snapshot)

	// 10s video = 2x base cost = 250 credits
	assert.Equal(t, 250, snapshot.ChargedCredits)
	assert.Equal(t, 250000, snapshot.ChargedQuota)
	assert.Equal(t, "per_second", snapshot.ProviderCostBasis)
	assert.Equal(t, 0.160, snapshot.ProviderEstimatedCostUSD)
	assert.True(t, snapshot.TargetMargin >= 60.0)

	// 2. Save quote and retrieve
	quoteId := pe.SaveQuote(snapshot)
	require.NotEmpty(t, quoteId)

	retrieved, err := pe.GetQuote(quoteId)
	require.NoError(t, err)
	assert.Equal(t, snapshot.ChargedCredits, retrieved.ChargedCredits)
	assert.Equal(t, snapshot.ProviderModel, retrieved.ProviderModel)

	// 3. Unknown quote returns ErrQuoteNotFound
	_, err = pe.GetQuote("quote_non_existent")
	assert.ErrorIs(t, err, ErrQuoteNotFound)

	// 4. Force expiration and verify ErrQuoteExpired
	pe.quotesMu.Lock()
	entry := pe.quoteCache[quoteId]
	entry.expiresAt = time.Now().Add(-1 * time.Minute)
	pe.quoteCache[quoteId] = entry
	pe.quotesMu.Unlock()

	_, err = pe.GetQuote(quoteId)
	assert.ErrorIs(t, err, ErrQuoteExpired)
}

func TestStudioPricing_ProfitabilityGuardFailsClosed(t *testing.T) {
	pe := NewPricingEngine()

	// 1. Snapshot with insufficient margin (charged quota too low)
	unprofitableSnapshot := &model.StudioPricingSnapshot{
		ProviderEstimatedCostUSD: 0.100, // Costs 10 cents
		ChargedQuota:             10000, // 10,000 Quota = 2 cents revenue -> Gross profit is negative!
	}
	err := pe.ValidateProfitability(unprofitableSnapshot, 60.0)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMarginBelowFloor)

	// 2. Snapshot below minimum floor (e.g. 50% margin when 60% required)
	// Cost = $0.05, Revenue = $0.08 -> Margin = 37.5%
	lowMarginSnapshot := &model.StudioPricingSnapshot{
		ProviderEstimatedCostUSD: 0.050,
		ChargedQuota:             40000, // $0.080 revenue
	}
	err = pe.ValidateProfitability(lowMarginSnapshot, 60.0)
	assert.Error(t, err)

	// 3. Snapshot with healthy 65% margin passes
	// Cost = $0.035, ChargedQuota = 50,000 ($0.100 revenue) -> Margin = 65%
	healthySnapshot := &model.StudioPricingSnapshot{
		ProviderEstimatedCostUSD: 0.035,
		ChargedQuota:             50000,
	}
	err = pe.ValidateProfitability(healthySnapshot, 60.0)
	assert.NoError(t, err)
}

func TestStudioPricing_CeilRoundingPreservesMargin(t *testing.T) {
	pe := NewPricingEngine()

	// Arbitrary fractional cost: $0.0133 at 60% target margin
	// sellPrice = 0.0133 / 0.4 = 0.03325
	// baseQuota = 0.03325 * 500,000 = 16,625 Quota units
	// calculatedCredits = 16.625
	// chargedCredits = ceil(16.625) = 17 Credits = 17,000 Quota units
	credits, quota, sellUSD := pe.CalculatePrice("utility", 0.0133, 60.0, 1.0)
	assert.Equal(t, 17, credits)
	assert.Equal(t, 17000, quota)
	assert.InDelta(t, 0.03325, sellUSD, 0.0001)

	// Calculate actual realized gross margin from charged quota
	revenueUSD := float64(quota) / common.QuotaPerUnit // 17000 / 500000 = 0.034
	grossProfitUSD := revenueUSD - 0.0133
	actualMargin := (grossProfitUSD / revenueUSD) * 100.0

	// Actual margin is 60.88%, preserving the >= 60.0% floor!
	assert.True(t, actualMargin >= 60.0, "Ceil rounding must mathematically guarantee margin >= target")
}

func TestStudioSecurity_SSRF_AdvancedBlocklist(t *testing.T) {
	// AWS / GCP Metadata
	assert.ErrorIs(t, ValidateExternalURL("http://169.254.169.254/latest/meta-data"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://169.254.170.2/v2/metadata"), ErrSSRFForbidden)

	// Alibaba Cloud Metadata
	assert.ErrorIs(t, ValidateExternalURL("http://100.100.100.200/latest/meta-data"), ErrSSRFForbidden)

	// RFC 1918 Private ranges
	assert.ErrorIs(t, ValidateExternalURL("http://10.254.1.1/internal-asset.png"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://192.168.1.50:80/photo.jpg"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://172.16.0.5/image.png"), ErrSSRFForbidden)

	// Loopback and IPv4-mapped IPv6
	assert.ErrorIs(t, ValidateExternalURL("http://127.0.0.1:9090/test"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://localhost:8080/image"), ErrSSRFForbidden)

	// Dangerous internal ports
	assert.ErrorIs(t, ValidateExternalURL("http://example.com:22/ssh"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://example.com:5432/db"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://example.com:6379/redis"), ErrSSRFForbidden)
}

func TestStudioSecurity_ValidateMediaUpload(t *testing.T) {
	// Valid PNG upload within 15MB
	validPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	mime, err := ValidateMediaUpload(validPNG, "image.png", false)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mime)

	// Oversized image payload (>15MB)
	oversized := make([]byte, 16*1024*1024)
	copy(oversized[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	_, err = ValidateMediaUpload(oversized, "huge.png", false)
	assert.ErrorIs(t, err, ErrFileSizeExceeded)

	// Video header when video is not allowed
	mp4Header := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}
	_, err = ValidateMediaUpload(mp4Header, "clip.mp4", false)
	assert.ErrorIs(t, err, ErrVideoNotAllowed)

	// Video header when video IS allowed
	mime, err = ValidateMediaUpload(mp4Header, "clip.mp4", true)
	require.NoError(t, err)
	assert.Equal(t, "video/mp4", mime)
}

func TestStudioService_ConcurrentSettlement_RaceSafe(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{Username: "studio_race_user", Quota: 100000, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		"idemp_race_1",
		"mock",
		map[string]interface{}{"prompt": "Race condition verification"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)

	// Advance mock provider state to succeeded
	mockProvider.JobStatusMap[job.ProviderJobId] = "completed"

	// Simulate concurrent webhook / poll race
	done := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, _ = studioSvc.PollJob(context.Background(), job.Id, user.Id, false)
			done <- true
		}()
	}
	<-done
	<-done

	// Verify wallet balance: should only be settled ONCE (deducting 5,000, leaving 95,000)
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 95000, bal, "Concurrent settlement must never double-deduct wallet quota")
}

func TestStudioService_HandleWebhook_SuccessAndIdempotency(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{Username: "webhook_user_1", Quota: 60000, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		"idemp_webhook_test_1",
		"mock",
		map[string]interface{}{"prompt": "Cyberpunk Bangkok street"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)
	assert.Equal(t, 5000, job.ReservedQuota)
	assert.Equal(t, 0, job.SettledQuota)

	// Trigger incoming webhook with status "completed"
	webhookOutput := "https://cdn.toraapi.com/outputs/cyberpunk_bangkok.png"
	updatedJob, err := studioSvc.HandleWebhook(context.Background(), "mock", job.ProviderJobId, "completed", webhookOutput, "")
	require.NoError(t, err)
	require.NotNil(t, updatedJob)

	assert.Equal(t, model.StudioJobStatusSucceeded, updatedJob.Status)
	assert.Equal(t, 5000, updatedJob.SettledQuota)
	assert.Contains(t, updatedJob.OutputResult, webhookOutput)

	// Check wallet balance
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 55000, bal, "User quota must be permanently deducted on settlement")

	// Call webhook a 2nd time to prove idempotency
	replayedJob, err := studioSvc.HandleWebhook(context.Background(), "mock", job.ProviderJobId, "completed", webhookOutput, "")
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, replayedJob.Status)

	bal2, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 55000, bal2, "Duplicate webhook must not double-charge")
}

func TestStudioService_HandleWebhook_FailureRefund(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{Username: "webhook_user_fail", Quota: 40000, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image-generate",
		"",
		"idemp_webhook_fail_1",
		"mock",
		map[string]interface{}{"prompt": "Error trigger"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)

	// Trigger incoming webhook with status "failed"
	updatedJob, err := studioSvc.HandleWebhook(context.Background(), "mock", job.ProviderJobId, "failed", "", "Provider GPU out of memory")
	require.NoError(t, err)
	require.NotNil(t, updatedJob)

	assert.Equal(t, model.StudioJobStatusFailed, updatedJob.Status)
	assert.Equal(t, "Provider GPU out of memory", updatedJob.ErrorMessage)
	assert.Equal(t, 0, updatedJob.SettledQuota)

	// Balance must be fully restored/refunded
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 40000, bal, "Quota must be fully refunded upon provider failure webhook")
}

func TestStudioService_ReconcileStaleJobs_CrashRecovery(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{Username: "stale_recovery_user", Quota: 50000, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)

	// 1. Simulate stranded RESERVED job (e.g. server crashed after wallet pre-consume before dispatching)
	preReqId := fmt.Sprintf("req_crash_%d", time.Now().UnixNano())
	err := model.PreConsumeUserWallet(preReqId, user.Id, 10000)
	require.NoError(t, err)

	oldTimestamp := common.GetTimestamp() - 7200 // 2 hours ago
	strandedJob := model.StudioToolJob{
		Id:             "job_stranded_reserved_1",
		UserId:         user.Id,
		ToolId:         "image-generate",
		RequestId:      preReqId,
		IdempotencyKey: "idemp_crash_1",
		ProviderName:   "mock",
		Status:         model.StudioJobStatusReserved,
		ReservedQuota:  10000,
		SettledQuota:   0,
		CreatedAt:      oldTimestamp,
		UpdatedAt:      oldTimestamp,
	}
	require.NoError(t, db.Create(&strandedJob).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)

	// Run crash recovery for jobs older than 1800s (30m)
	reconciled, err := studioSvc.ReconcileStaleJobs(context.Background(), 1800)
	require.NoError(t, err)
	assert.True(t, reconciled >= 1)

	// Verify stranded job is now FAILED and reservation refunded
	var recoveredJob model.StudioToolJob
	require.NoError(t, db.Where("id = ?", strandedJob.Id).First(&recoveredJob).Error)
	assert.Equal(t, model.StudioJobStatusFailed, recoveredJob.Status)
	assert.Contains(t, recoveredJob.ErrorMessage, "safely refunded")

	// User quota should be back to 50,000
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, bal, "Quota should be refunded for stranded crash job")
}

func TestStudioPricing_CalculatePriceWithInputs_MultiVariable(t *testing.T) {
	engine := NewPricingEngine()

	videoTool := &model.StudioToolDefinition{
		Id:              "video-generate",
		PrimaryProvider: "fal",
		PrimaryModel:    "fal-ai/kling-video/v1/standard/text-to-video",
		Category:        "video",
		CreditCost:      100, // 100 credits = $0.20 USD (meets 60% margin over $0.080)
		MarginPercent:   60.0,
	}

	// 1. Standard 5s @ 30fps without audio
	p1, err := engine.CalculatePriceWithInputs(videoTool, map[string]interface{}{
		"duration": 5,
		"fps":      30,
	}, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 100, p1.ChargedCredits)
	assert.True(t, p1.CalculatedSellUSD >= p1.ProviderEstimatedCostUSD)

	// 2. High framerate 60fps with audio synthesis
	p2, err := engine.CalculatePriceWithInputs(videoTool, map[string]interface{}{
		"duration": 5,
		"fps":      60,
		"audio":    true,
	}, 1.0)
	require.NoError(t, err)
	assert.True(t, p2.ChargedCredits > p1.ChargedCredits, "60fps with audio must cost more than standard 30fps")
	assert.True(t, p2.ProviderEstimatedCostUSD > p1.ProviderEstimatedCostUSD)

	// Margin must still satisfy >= 60%
	margin := (p2.CalculatedSellUSD - p2.ProviderEstimatedCostUSD) / p2.CalculatedSellUSD
	assert.True(t, margin >= 0.60, "Gross margin for multi-variable compute must remain >= 60%")

	// 3. Multi-output image generation
	imgTool := &model.StudioToolDefinition{
		Id:              "image-generate",
		PrimaryProvider: "fal",
		PrimaryModel:    "fal-ai/flux/schnell",
		Category:        "image",
		CreditCost:      5,
		MarginPercent:   60.0,
	}
	pImg, err := engine.CalculatePriceWithInputs(imgTool, map[string]interface{}{
		"num_outputs": 4,
		"quality":     "ultra",
	}, 1.0)
	require.NoError(t, err)
	assert.Equal(t, 128, pImg.ChargedCredits, "4 ultra images should cost 32 * 4 = 128 credits")
	assert.Equal(t, 0.10, pImg.ProviderEstimatedCostUSD, "4 ultra images should cost 4 * $0.025 = $0.10")
}

func TestStudioAsset_CRUDAndAccessControl(t *testing.T) {
	_ = setupTestDBForStudio(t)

	asset := &model.StudioAsset{
		Id:                 "asset_test_user1_pic",
		UserId:             101,
		JobId:              "job_xyz_1",
		AssetType:          "output",
		MIMEType:           "image/png",
		FileSize:           2048576,
		Width:              1024,
		Height:             1024,
		AvailabilityStatus: "available",
		StorageURL:         "https://cdn.toraapi.com/assets/pic.png",
		ExpiryAt:           common.GetTimestamp() + 86400*30,
		CreatedAt:          common.GetTimestamp(),
	}

	err := model.CreateStudioAsset(asset)
	require.NoError(t, err)

	// Retrieve by owner
	fetched, err := model.GetStudioAsset(asset.Id, 101, false)
	require.NoError(t, err)
	assert.Equal(t, asset.Id, fetched.Id)
	assert.Equal(t, asset.MIMEType, fetched.MIME)
	assert.Equal(t, asset.FileSize, fetched.Size)

	// Another user forbidden
	_, err = model.GetStudioAsset(asset.Id, 102, false)
	assert.ErrorIs(t, err, model.ErrStudioForbiddenAccess)

	// Admin can access
	adminFetched, err := model.GetStudioAsset(asset.Id, 999, true)
	require.NoError(t, err)
	assert.Equal(t, asset.Id, adminFetched.Id)

	// List user assets
	assets, total, err := model.ListStudioAssets(101, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, assets, 1)
}


