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
