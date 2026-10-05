package model

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDBForPhase6F(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:test_phase6f_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	oldDB := DB
	DB = db
	t.Cleanup(func() {
		DB = oldDB
		_ = sqlDB.Close()
	})

	require.NoError(t, db.AutoMigrate(
		&User{},
		&TopUp{},
		&SubscriptionPlan{},
		&UserSubscription{},
		&SubscriptionOrder{},
		&SubscriptionPreConsumeRecord{},
		&WalletPreConsumeRecord{},
		&Channel{},
	))
	return db
}

// 6F-01: Broken Transaction Boundary in Subscription Pre-Consume Refund
func Test6F01_SubscriptionPreConsumeRefund_TransactionBoundary(t *testing.T) {
	db := setupTestDBForPhase6F(t)

	user := User{
		Username: "sub_refund_user",
		Status:   common.UserStatusEnabled,
		Quota:    0,
	}
	require.NoError(t, db.Create(&user).Error)

	sub := UserSubscription{
		UserId:      user.Id,
		PlanId:      1,
		Status:      "active",
		AmountTotal: 100000,
		AmountUsed:  50000,
		StartTime:   time.Now().Unix(),
		EndTime:     time.Now().Add(24 * time.Hour).Unix(),
	}
	require.NoError(t, db.Create(&sub).Error)

	reqID := "req-sub-refund-01"
	record := SubscriptionPreConsumeRecord{
		RequestId:          reqID,
		UserId:             user.Id,
		UserSubscriptionId: sub.Id,
		PreConsumed:        50000,
		Status:             "consumed",
	}
	require.NoError(t, db.Create(&record).Error)

	// Single refund
	err := RefundSubscriptionPreConsume(reqID)
	require.NoError(t, err)

	var updatedSub UserSubscription
	require.NoError(t, db.First(&updatedSub, sub.Id).Error)
	assert.Equal(t, int64(0), updatedSub.AmountUsed, "AmountUsed should be decremented back to 0")

	var updatedRec SubscriptionPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", reqID).First(&updatedRec).Error)
	assert.Equal(t, "refunded", updatedRec.Status)

	// Idempotent second refund
	err = RefundSubscriptionPreConsume(reqID)
	require.NoError(t, err)

	require.NoError(t, db.First(&updatedSub, sub.Id).Error)
	assert.Equal(t, int64(0), updatedSub.AmountUsed, "AmountUsed must not be reduced below 0 on repeated refund")

	// Concurrent refund test
	reqID2 := "req-sub-refund-02"
	updatedSub.AmountUsed = 50000
	require.NoError(t, db.Save(&updatedSub).Error)

	record2 := SubscriptionPreConsumeRecord{
		RequestId:          reqID2,
		UserId:             user.Id,
		UserSubscriptionId: sub.Id,
		PreConsumed:        50000,
		Status:             "consumed",
	}
	require.NoError(t, db.Create(&record2).Error)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = RefundSubscriptionPreConsume(reqID2)
		}()
	}
	wg.Wait()

	require.NoError(t, db.First(&updatedSub, sub.Id).Error)
	assert.Equal(t, int64(0), updatedSub.AmountUsed, "Concurrent refunds must exactly refund once")
}

// 6F-02: Non-Transactional Race in Stripe Delayed Payment Failure
func Test6F02_StripeDelayedPaymentFailure_Transactional(t *testing.T) {
	db := setupTestDBForPhase6F(t)

	user := User{Username: "stripe_fail_user", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)

	topUp := TopUp{
		UserId:          user.Id,
		Amount:          10,
		Money:           10.0,
		TradeNo:         "trade-stripe-race-01",
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(&topUp).Error)

	// 1. Success transition
	err := Recharge(topUp.TradeNo, "cus_123", "127.0.0.1")
	require.NoError(t, err)

	var settledTopUp TopUp
	require.NoError(t, db.Where("trade_no = ?", topUp.TradeNo).First(&settledTopUp).Error)
	assert.Equal(t, common.TopUpStatusSuccess, settledTopUp.Status)

	// 2. Delayed payment failure arriving after success MUST fail and NOT overwrite success
	err = UpdatePendingTopUpStatus(topUp.TradeNo, PaymentProviderStripe, common.TopUpStatusFailed)
	assert.ErrorIs(t, err, ErrTopUpStatusInvalid, "Should reject failure transition on settled order")

	require.NoError(t, db.Where("trade_no = ?", topUp.TradeNo).First(&settledTopUp).Error)
	assert.Equal(t, common.TopUpStatusSuccess, settledTopUp.Status, "Status must remain Success")

	// 3. Subscription order delayed failure test
	subOrder := SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          1,
		TradeNo:         "sub-trade-stripe-01",
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(&subOrder).Error)

	err = UpdatePendingSubscriptionOrderStatus(subOrder.TradeNo, PaymentProviderStripe, common.TopUpStatusFailed)
	require.NoError(t, err)

	var failedOrder SubscriptionOrder
	require.NoError(t, db.Where("trade_no = ?", subOrder.TradeNo).First(&failedOrder).Error)
	assert.Equal(t, common.TopUpStatusFailed, failedOrder.Status)

	// Second attempt on already failed order
	err = UpdatePendingSubscriptionOrderStatus(subOrder.TradeNo, PaymentProviderStripe, common.TopUpStatusFailed)
	assert.ErrorIs(t, err, ErrSubscriptionOrderStatusInvalid)
}

// 6F-03: AffQuota Transfer Quota Cache Desync & Full Struct Clobbering
func Test6F03_TransferAffQuotaToQuota_AtomicAndCache(t *testing.T) {
	db := setupTestDBForPhase6F(t)

	unit := int(common.QuotaPerUnit) // e.g. 500,000
	user := User{
		Username: "aff_user_01",
		AffQuota: 5 * unit,
		Quota:    1 * unit,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	// Transfer 1 unit
	err := user.TransferAffQuotaToQuota(unit)
	require.NoError(t, err)

	var refreshedUser User
	require.NoError(t, db.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, 4*unit, refreshedUser.AffQuota)
	assert.Equal(t, 2*unit, refreshedUser.Quota)
	assert.Equal(t, 4*unit, user.AffQuota)
	assert.Equal(t, 2*unit, user.Quota)

	// Concurrent transfers: 4 goroutines each transferring 1 unit
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u := User{Id: user.Id}
			_ = u.TransferAffQuotaToQuota(unit)
		}()
	}
	wg.Wait()

	require.NoError(t, db.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, 0, refreshedUser.AffQuota, "All remaining 4 units transferred across 4 concurrent requests")
	assert.Equal(t, 6*unit, refreshedUser.Quota, "Total wallet quota should equal 1 initial + 5 transferred units")

	// Attempting to transfer with insufficient aff quota
	err = refreshedUser.TransferAffQuotaToQuota(unit)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不足")
}

// 6F-04: Non-Atomic Wallet Pre-Consume Refund
func Test6F04_WalletPreConsumeRefund_Atomic(t *testing.T) {
	db := setupTestDBForPhase6F(t)

	user := User{
		Username: "wallet_atomic_refund_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	reqID := "req-atomic-wallet-refund"
	err := PreConsumeUserWallet(reqID, user.Id, 30000)
	require.NoError(t, err)

	quota, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 70000, quota)

	// Concurrently refund the same reservation 10 times
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = RefundUserWalletPreConsume(reqID)
		}()
	}
	wg.Wait()

	// Quota must be exactly 100,000, never 100k + extra
	quota, err = GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 100000, quota, "Refund must atomically restore quota exactly once")

	var record WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", reqID).First(&record).Error)
	assert.Equal(t, "refunded", record.Status)
}

// 6F-05: Multi-Key Polling Cursor Freeze
func Test6F05_MultiKeyModePolling_CursorAdvancement(t *testing.T) {
	db := setupTestDBForPhase6F(t)

	oldMemCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemCache
	})

	channel := Channel{
		Id:     99,
		Type:   constant.ChannelTypeOpenAI,
		Name:   "multi-key-test-channel",
		Key:    "key0\nkey1\nkey2",
		Status: common.ChannelStatusEnabled,
		ChannelInfo: ChannelInfo{
			IsMultiKey:           true,
			MultiKeyMode:         constant.MultiKeyModePolling,
			MultiKeyPollingIndex: 0,
		},
	}
	require.NoError(t, db.Create(&channel).Error)
	CacheUpdateChannel(&channel)

	// Sequential calls to GetNextEnabledKey must cycle through keys in order
	expectedKeys := []string{"key0", "key1", "key2", "key0", "key1", "key2"}
	for i, expected := range expectedKeys {
		k, idx, apiErr := channel.GetNextEnabledKey()
		require.Nil(t, apiErr, "call %d failed", i)
		assert.Equal(t, expected, k, "call %d expected %s, got %s", i, expected, k)
		assert.Equal(t, i%3, idx, "call %d index mismatch", i)
	}
}

// 6F-06: Webhook Idempotency Contract Mismatch
func Test6F06_WebhookIdempotency_Contract(t *testing.T) {
	db := setupTestDBForPhase6F(t)

	user := User{Username: "webhook_user_01", Status: common.UserStatusEnabled, Quota: 0}
	require.NoError(t, db.Create(&user).Error)

	// 1. Stripe Recharge idempotency
	topUpStripe := TopUp{
		UserId:          user.Id,
		Amount:          10,
		Money:           10.0,
		TradeNo:         "trade-stripe-idemp-01",
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(&topUpStripe).Error)

	// First webhook delivery
	err := Recharge(topUpStripe.TradeNo, "cus_1", "127.0.0.1")
	require.NoError(t, err)

	q1, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Greater(t, q1, 0)

	// Duplicate webhook delivery must return nil (HTTP 200 OK for payment gateway)
	err = Recharge(topUpStripe.TradeNo, "cus_1", "127.0.0.1")
	require.NoError(t, err, "Duplicate Stripe webhook must return nil without error")

	q2, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, q1, q2, "Duplicate webhook must not double-credit")

	// 2. Creem Recharge idempotency
	topUpCreem := TopUp{
		UserId:          user.Id,
		Amount:          50000,
		Money:           1.0,
		TradeNo:         "trade-creem-idemp-01",
		PaymentProvider: PaymentProviderCreem,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(&topUpCreem).Error)

	err = RechargeCreem(topUpCreem.TradeNo, "a@b.com", "User", "127.0.0.1")
	require.NoError(t, err)

	q3, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)

	err = RechargeCreem(topUpCreem.TradeNo, "a@b.com", "User", "127.0.0.1")
	require.NoError(t, err, "Duplicate Creem webhook must return nil without error")

	q4, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, q3, q4, "Duplicate Creem webhook must not double-credit")

	// 3. ManualCompleteTopUp idempotency
	topUpAdmin := TopUp{
		UserId:          user.Id,
		Amount:          10,
		Money:           10.0,
		TradeNo:         "trade-admin-idemp-01",
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, db.Create(&topUpAdmin).Error)

	err = ManualCompleteTopUp(topUpAdmin.TradeNo, "127.0.0.1")
	require.NoError(t, err)

	q5, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)

	err = ManualCompleteTopUp(topUpAdmin.TradeNo, "127.0.0.1")
	require.NoError(t, err, "Duplicate ManualCompleteTopUp must return nil")

	q6, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, q5, q6, "Duplicate AdminCompleteTopUp must not double-credit")
}
