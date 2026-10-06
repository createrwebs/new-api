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
	require.NoError(t, SeedDefaultTools(db))

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
		"image_generate_fast",
		"idemp_key_success_1",
		"mock",
		map[string]interface{}{"prompt": "Modern AI architectural datacenter"},
	)
	require.NoError(t, err)
	require.NotNil(t, job)

	// 3. Assertions on Job State
	assert.Equal(t, model.StudioJobStatusCompleted, job.Status)
	assert.Equal(t, 5000, job.ReservedQuota)
	assert.Equal(t, 5000, job.SettledQuota)
	assert.Contains(t, job.OutputResult, "https://cdn.toraapi.com/mock-assets/")

	// 4. Assertions on Wallet Settlement
	remainingQuota, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 45000, remainingQuota, "User quota should be permanently reduced by settled amount")

	// 5. Assertions on WalletPreConsumeRecord
	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).First(&preRecord).Error)
	assert.Equal(t, "settled", preRecord.Status)
	assert.Equal(t, 5000, preRecord.PreConsumed)
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
		"image_generate_pro", // 30,000 Quota units
		"idemp_key_fail_1",
		"mock",
		map[string]interface{}{"prompt": "Trigger permanent upstream error"},
	)
	require.Error(t, err)
	require.NotNil(t, job)

	// Job should be marked failed
	assert.Equal(t, model.StudioJobStatusFailed, job.Status)
	assert.Equal(t, 30000, job.ReservedQuota)
	assert.Equal(t, 0, job.SettledQuota)

	// User quota should be 100% refunded back to 50,000
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
		"image_generate_fast",
		"idemp_key_broke_1",
		"mock",
		map[string]interface{}{"prompt": "Should fail before provider dispatch"},
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
		"image_generate_fast", // 5,000 Quota units
		sharedIdempKey,
		"mock",
		map[string]interface{}{"prompt": "Idempotent request test"},
	)
	require.NoError(t, err)
	require.NotNil(t, job1)
	assert.Equal(t, model.StudioJobStatusCompleted, job1.Status)

	// Check balance after first call
	bal1, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 45000, bal1)

	// Second submission with identical idempotency key
	job2, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"image_generate_fast",
		sharedIdempKey,
		"mock",
		map[string]interface{}{"prompt": "Idempotent request test"},
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
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeDelayedSuccess)
	studioSvc := NewStudioService(mockProvider)

	// 1. Submit async video generation (50,000 Quota units)
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"text_to_video_fast",
		"idemp_async_video_1",
		"mock",
		map[string]interface{}{"prompt": "Cinematic flying drone over Bangkok"},
	)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)
	assert.Equal(t, 50000, job.ReservedQuota)
	assert.Equal(t, 0, job.SettledQuota)

	// Quota is reserved (50,000 pending)
	balAfterSubmit, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, balAfterSubmit)

	// 2. Poll 1: Still processing
	jobPoll1, err := studioSvc.PollJob(context.Background(), job.Id)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusProcessing, jobPoll1.Status)
	assert.Equal(t, 0, jobPoll1.SettledQuota)

	// 3. Poll 2: Transitions to completed and settles
	jobPoll2, err := studioSvc.PollJob(context.Background(), job.Id)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusCompleted, jobPoll2.Status)
	assert.Equal(t, 50000, jobPoll2.SettledQuota)
	assert.Contains(t, jobPoll2.OutputResult, "delayed_output.png")

	// Final verification of settled record
	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).First(&preRecord).Error)
	assert.Equal(t, "settled", preRecord.Status)
}
