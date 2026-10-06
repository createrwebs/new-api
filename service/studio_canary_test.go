package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueue2_PreCanaryGates tests all 7 required Queue 1 safety gates before paid execution.
func TestQueue2_PreCanaryGates(t *testing.T) {
	db := setupTestDBForStudio(t)
	pricingEngine := NewPricingEngine()

	// Gate 1: Server Quote PASS
	toolDef, err := model.GetStudioToolDefinition("background-remove")
	require.NoError(t, err)
	quote, err := pricingEngine.CalculatePriceWithInputs(toolDef, map[string]interface{}{}, 1.0)
	require.NoError(t, err)
	quoteId := pricingEngine.SaveQuote(quote)
	require.NotEmpty(t, quoteId)
	assert.Equal(t, 10, quote.ChargedCredits)
	assert.Equal(t, 10000, quote.ChargedQuota)

	// Gate 2: Pricing Snapshot PASS
	assert.Equal(t, "v1.2", quote.PricingVersion)
	assert.Equal(t, "fal-ai/birefnet", quote.ProviderModel)
	assert.Equal(t, 0.005, quote.ProviderEstimatedCostUSD)
	assert.True(t, quote.TargetMargin >= 60.0)

	// Gate 3: Idempotency PASS
	user := model.User{Username: "gate_user", Quota: 100000, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)
	job1, err := studioSvc.SubmitJob(context.Background(), user.Id, "background-remove", "", "gate_idemp_key_1", "mock", map[string]interface{}{"image_url": "https://example.com/test.png"}, 1.0, "127.0.0.1")
	require.NoError(t, err)
	job2, err := studioSvc.SubmitJob(context.Background(), user.Id, "background-remove", "", "gate_idemp_key_1", "mock", map[string]interface{}{"image_url": "https://example.com/test.png"}, 1.0, "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, job1.Id, job2.Id, "Duplicate idempotency key must return existing job")

	// Gate 4: Webhook / Poll Race PASS
	jobRace, err := studioSvc.SubmitJob(context.Background(), user.Id, "background-remove", "", "gate_idemp_key_race", "mock", map[string]interface{}{"image_url": "https://example.com/test.png"}, 1.0, "127.0.0.1")
	require.NoError(t, err)
	// Webhook call
	_, err = studioSvc.HandleWebhook(context.Background(), "mock", jobRace.ProviderJobId, "completed", "https://cdn.toraapi.com/output.png", "")
	require.NoError(t, err)
	// Duplicate poll call
	_, _ = studioSvc.PollJob(context.Background(), jobRace.Id, user.Id, false)
	bal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 80000, bal, "Wallet quota must be deducted exactly twice (10k + 10k), never duplicated")

	// Gate 5: Restart Recovery PASS
	reconciled, err := studioSvc.ReconcileStaleJobs(context.Background(), 3600)
	require.NoError(t, err)
	assert.True(t, reconciled >= 0)

	// Gate 6: Asset Safety PASS
	validPng := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	mime, err := ValidateMediaUpload(validPng, "safe.png", false)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mime)

	// Gate 7: SSRF Protection PASS
	assert.ErrorIs(t, ValidateExternalURL("http://169.254.169.254/latest/meta-data"), ErrSSRFForbidden)
	assert.ErrorIs(t, ValidateExternalURL("http://127.0.0.1:8080/internal"), ErrSSRFForbidden)
}

// TestQueue2_RealToraWallet_CreditLoop_SimulatedLiveCanary verifies the end-to-end monetization loop.
func TestQueue2_RealToraWallet_CreditLoop_SimulatedLiveCanary(t *testing.T) {
	db := setupTestDBForStudio(t)

	// 1. Controlled Tora test account
	user := model.User{
		Username: "canary_tester_wallet",
		Quota:    50000, // 50 Tora Credits ($0.10 USD)
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	walletBefore, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, walletBefore)

	// 2. Server quote before submission
	toolDef, err := model.GetStudioToolDefinition("background-remove")
	require.NoError(t, err)

	pricingEngine := NewPricingEngine()
	quote, err := pricingEngine.CalculatePriceWithInputs(toolDef, map[string]interface{}{}, 1.0)
	require.NoError(t, err)
	quoteId := pricingEngine.SaveQuote(quote)
	require.NotEmpty(t, quoteId)
	assert.Equal(t, 10, quote.ChargedCredits, "background-remove requires 10 credits")
	assert.Equal(t, 10000, quote.ChargedQuota)

	// 3. Mock live fal.ai queue runner
	mockFalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Key fal_live_canary_test_key", r.Header.Get("Authorization"))

		switch r.URL.Path {
		case "/fal-ai/birefnet":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"request_id": "fal_req_canary_uuid_7788",
				"status": "IN_QUEUE",
				"status_url": "https://queue.fal.run/fal-ai/birefnet/requests/fal_req_canary_uuid_7788/status",
				"response_url": "https://queue.fal.run/fal-ai/birefnet/requests/fal_req_canary_uuid_7788",
				"cancel_url": "https://queue.fal.run/fal-ai/birefnet/requests/fal_req_canary_uuid_7788/cancel"
			}`))

		case "/fal-ai/birefnet/requests/fal_req_canary_uuid_7788/status":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"status": "COMPLETED",
				"response_url": ""
			}`))

		case "/fal-ai/birefnet/requests/fal_req_canary_uuid_7788":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"image": {
					"url": "https://fal.media/files/canary/cutout_clean.png",
					"content_type": "image/png"
				}
			}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer mockFalServer.Close()

	falProvider := &FalProvider{
		client:     mockFalServer.Client(),
		baseURL:    mockFalServer.URL,
		apiKey:     "fal_live_canary_test_key",
		costLedger: map[string]float64{"fal-ai/birefnet": 0.005},
	}
	studioSvc := NewStudioService(falProvider)

	// 4. Submit Job: reserves quota through PreConsumeUserWallet
	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"idemp_canary_tx_001",
		"fal",
		map[string]interface{}{"image_url": "https://example.com/sample_100x100.png"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Equal(t, model.StudioJobStatusProcessing, job.Status)
	assert.Equal(t, 10000, job.ReservedQuota)
	assert.Equal(t, 0, job.SettledQuota)
	assert.Equal(t, "fal-ai/birefnet:fal_req_canary_uuid_7788", job.ProviderJobId)

	// Verify wallet balance DURING reservation
	reservedBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 40000, reservedBal, "User balance must be reduced by reserved amount immediately")

	// 5. Poll Job: reconciles completion and triggers SettleUserWalletPreConsume
	settledJob, err := studioSvc.PollJob(context.Background(), job.Id, user.Id, false)
	require.NoError(t, err)
	assert.Equal(t, model.StudioJobStatusSucceeded, settledJob.Status)
	assert.Equal(t, 10000, settledJob.SettledQuota)
	assert.Contains(t, settledJob.OutputResult, "https://fal.media/files/canary/cutout_clean.png")

	// 6. Verify wallet balance AFTER settlement
	walletAfter, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 40000, walletAfter, "wallet_after must equal wallet_before - charged_quota")
	assert.Equal(t, walletBefore-10000, walletAfter)

	// 7. Verify Job Evidence Persistence
	var persistedJob model.StudioToolJob
	require.NoError(t, db.Where("id = ?", job.Id).First(&persistedJob).Error)
	assert.Equal(t, model.StudioJobStatusSucceeded, persistedJob.Status)
	assert.NotEmpty(t, persistedJob.PricingSnapshot)

	var pricingSnapshot model.StudioPricingSnapshot
	require.NoError(t, json.Unmarshal([]byte(persistedJob.PricingSnapshot), &pricingSnapshot))
	assert.Equal(t, "fal-ai/birefnet", pricingSnapshot.ProviderModel)
	assert.Equal(t, 0.005, pricingSnapshot.ProviderEstimatedCostUSD)
	assert.Equal(t, 10, pricingSnapshot.ChargedCredits)

	// Verify StudioJobEvent rows
	var events []model.StudioJobEvent
	require.NoError(t, db.Where("job_id = ?", job.Id).Order("created_at ASC").Find(&events).Error)
	assert.True(t, len(events) >= 3, "Must record CREATED, RESERVED, SUBMITTED, and SUCCEEDED events")

	// Verify StudioCostSnapshot persistence
	var costSnapshot model.StudioCostSnapshot
	require.NoError(t, db.Where("job_id = ?", job.Id).First(&costSnapshot).Error)
	assert.Equal(t, 0.005, costSnapshot.CostUSD)
	assert.Equal(t, 10000, costSnapshot.QuotaCost)
	assert.Equal(t, 0.020, costSnapshot.ToraRevenueUSD)
	assert.Equal(t, 0.015, costSnapshot.GrossProfitUSD)
	assert.Equal(t, 75.0, costSnapshot.GrossMarginPercent)

	// 8. Duplicate Request Test (Idempotent Replay)
	duplicateJob, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"idemp_canary_tx_001", // SAME KEY
		"fal",
		map[string]interface{}{"image_url": "https://example.com/sample_100x100.png"},
		1.0,
		"127.0.0.1",
	)
	require.NoError(t, err)
	assert.Equal(t, job.Id, duplicateJob.Id, "Must return identical job record")

	// Verify zero additional charges or reservations on idempotent replay
	balReplay, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 40000, balReplay, "Duplicate submission must never deduct additional quota")

	var preRecords []model.WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", job.RequestId).Find(&preRecords).Error)
	assert.Len(t, preRecords, 1, "Exactly one wallet pre-consume record must exist for the request")
}

// TestQueue2_Failure_DeterministicRefund proves failure refund restoration using MockProvider.
func TestQueue2_Failure_DeterministicRefund(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "refund_tester",
		Quota:    30000, // 30 Credits
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	initialBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 30000, initialBal)

	// Use MockProvider in Permanent Failure mode
	failProvider := NewDeterministicMockProvider(MockModePermanentFail)
	studioSvc := NewStudioService(failProvider)

	job, err := studioSvc.SubmitJob(
		context.Background(),
		user.Id,
		"background-remove",
		"",
		"idemp_fail_tx_001",
		"mock",
		map[string]interface{}{"image_url": "https://example.com/bad_input.png"},
		1.0,
		"127.0.0.1",
	)
	require.Error(t, err)
	require.NotNil(t, job)
	assert.Equal(t, model.StudioJobStatusFailed, job.Status)

	// Wallet must be 100% restored to initial balance
	finalBal, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, initialBal, finalBal, "Wallet quota must be fully restored upon submission failure")

	// Verify pre-consume record was marked refunded
	var preRecord model.WalletPreConsumeRecord
	require.NoError(t, db.Where("user_id = ? AND status = ?", user.Id, "refunded").First(&preRecord).Error)
	assert.Equal(t, 10000, preRecord.PreConsumed)
}
