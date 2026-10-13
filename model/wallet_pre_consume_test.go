package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDBForWalletPreConsume(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:test_wallet_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
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

	require.NoError(t, db.AutoMigrate(&User{}, &WalletPreConsumeRecord{}))
	return db
}

func TestWalletPreConsume_LifecycleAndReconciliation(t *testing.T) {
	db := setupTestDBForWalletPreConsume(t)

	// Create test user with quota = 100,000 ($2.00)
	user := User{
		Username: "test_reconcile_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(&user).Error)

	reqID := "test-req-6c05-1"

	// 1. PreConsume 30,000
	err := PreConsumeUserWallet(reqID, user.Id, 30000)
	require.NoError(t, err)

	quota, err := GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 70000, quota, "User quota should be reduced by pre-consumed amount")

	var record WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", reqID).First(&record).Error)
	assert.Equal(t, 30000, record.PreConsumed)
	assert.Equal(t, "pending", record.Status)

	// 2. Incremental pre-consume on same request ID (e.g. image overrides)
	err = PreConsumeUserWallet(reqID, user.Id, 20000)
	require.NoError(t, err)

	quota, err = GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, quota, "User quota should be reduced further")

	require.NoError(t, db.Where("request_id = ?", reqID).First(&record).Error)
	assert.Equal(t, 50000, record.PreConsumed, "Pending record should have accumulated amount")

	// 3. Settle
	err = SettleUserWalletPreConsume(reqID)
	require.NoError(t, err)

	require.NoError(t, db.Where("request_id = ?", reqID).First(&record).Error)
	assert.Equal(t, "settled", record.Status)

	// 4. Reconciliation must ignore settled records
	reconciled, err := ReconcileOrphanedWalletPreConsumes(0)
	require.NoError(t, err)
	assert.Equal(t, 0, reconciled)

	// 5. Test Orphaned Pre-Consume Recovery (Crash simulation)
	abandonedReqID := "test-req-crash-6c05"
	err = PreConsumeUserWallet(abandonedReqID, user.Id, 25000)
	require.NoError(t, err)

	quota, err = GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 25000, quota)

	// Backdate the created_at/updated_at timestamp to simulate 15 minutes old
	require.NoError(t, db.Model(&WalletPreConsumeRecord{}).
		Where("request_id = ?", abandonedReqID).
		Updates(map[string]any{
			"created_at": common.GetTimestamp() - 900,
			"updated_at": common.GetTimestamp() - 900,
		}).Error)

	// Run reconciliation for records older than 600s (10 min)
	reconciled, err = ReconcileOrphanedWalletPreConsumes(600)
	require.NoError(t, err)
	assert.Equal(t, 1, reconciled, "Should reconcile exactly 1 orphaned record")

	quota, err = GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, quota, "Orphaned quota should be fully restored to user wallet")

	var reconciledRec WalletPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", abandonedReqID).First(&reconciledRec).Error)
	assert.Equal(t, "refunded", reconciledRec.Status)

	// 6. Test Idempotent Refund
	refundReqID := "test-req-refund-6c05"
	err = PreConsumeUserWallet(refundReqID, user.Id, 10000)
	require.NoError(t, err)

	quota, err = GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 40000, quota)

	// First refund
	err = RefundUserWalletPreConsume(refundReqID)
	require.NoError(t, err)

	quota, err = GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, quota, "Refund should restore wallet quota")

	// Repeated refund must be idempotent and not add extra balance
	err = RefundUserWalletPreConsume(refundReqID)
	require.NoError(t, err)

	quota, err = GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, 50000, quota, "Repeated refund must not double-credit")
}
