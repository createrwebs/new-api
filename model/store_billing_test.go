package model

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupStoreBillingTestDB(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&User{},
		&SubscriptionPlan{},
		&UserSubscription{},
		&SubscriptionOrder{},
		&TopUp{},
		&StoreProductMapping{},
		&StoreSubscriptionBinding{},
		&StoreTransaction{},
		&Log{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
	})
}

func TestProcessStorePurchase_ValidInitialPurchase(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{
		Username: "store-user-1",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "aff-store-1",
	}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{
		Title:          "Pro Monthly",
		DurationUnit:   SubscriptionDurationMonth,
		DurationValue:  1,
		TotalAmount:    500000,
		UpgradeGroup:   "pro",
		DowngradeGroup: "default",
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{
		Platform:       StorePlatformApple,
		StoreProductId: "com.tora.pro.monthly",
		InternalPlanId: plan.Id,
		Environment:    "production",
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&mapping).Error)

	verified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "apple_tx_1001",
		StoreOriginalId:    "apple_orig_1001",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}

	result, err := ProcessStorePurchaseTx(user.Id, verified)
	require.NoError(t, err)
	assert.Equal(t, StoreStatusActive, result.Status)
	assert.Equal(t, plan.Id, result.PlanId)
	assert.False(t, result.AlreadyProcessed)
	assert.False(t, result.IsRenewal)

	// Verify User Group upgraded
	var updatedUser User
	require.NoError(t, DB.Where("id = ?", user.Id).First(&updatedUser).Error)
	assert.Equal(t, "pro", updatedUser.Group)

	// Verify UserSubscription created
	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", result.UserSubscriptionId).First(&sub).Error)
	assert.Equal(t, user.Id, sub.UserId)
	assert.Equal(t, plan.Id, sub.PlanId)
	assert.Equal(t, StoreStatusActive, sub.Status)
	assert.Equal(t, "default", sub.PrevUserGroup)
	assert.Equal(t, "pro", sub.UpgradeGroup)
	assert.Equal(t, int64(500000), sub.AmountTotal)

	// Verify StoreSubscriptionBinding
	var binding StoreSubscriptionBinding
	require.NoError(t, DB.Where("platform = ? AND store_original_id = ?", StorePlatformApple, "apple_orig_1001").First(&binding).Error)
	assert.Equal(t, user.Id, binding.UserId)
	assert.Equal(t, plan.Id, binding.InternalPlanId)
	assert.Equal(t, sub.Id, binding.ActiveUserSubId)
	assert.Equal(t, "apple_tx_1001", binding.LatestTransactionId)

	// Verify StoreTransaction record
	var storeTx StoreTransaction
	require.NoError(t, DB.Where("platform = ? AND store_transaction_id = ?", StorePlatformApple, "apple_tx_1001").First(&storeTx).Error)
	assert.Equal(t, user.Id, storeTx.UserId)
	assert.Equal(t, sub.Id, storeTx.UserSubscriptionId)

	// Verify SubscriptionOrder record
	var order SubscriptionOrder
	require.NoError(t, DB.Where("trade_no = ?", "STORE_APPLE_apple_tx_1001").First(&order).Error)
	assert.Equal(t, common.TopUpStatusSuccess, order.Status)
}

func TestProcessStorePurchase_ReplayIdempotency(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{
		Username: "store-replay-user",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "aff-replay-1",
	}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{
		Title:         "Pro Monthly",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   500000,
		UpgradeGroup:  "pro",
		Enabled:       true,
	}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{
		Platform:       StorePlatformGoogle,
		StoreProductId: "tora_pro_sub",
		InternalPlanId: plan.Id,
		Environment:    "production",
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&mapping).Error)

	verified := &VerifiedStorePurchase{
		Platform:                    StorePlatformGoogle,
		StoreTransactionId:          "GPA.1234-5678-9012-34567",
		StoreOriginalId:             "GPA.1234-5678-9012-34567",
		StoreProductId:              "tora_pro_sub",
		PurchaseTime:                now,
		ExpiresTime:                 now + 30*86400,
		Environment:                 "Production",
		ObfuscatedExternalAccountId: DeriveStoreAccountToken(user.Id, "Production"),
	}

	// First execution
	res1, err := ProcessStorePurchaseTx(user.Id, verified)
	require.NoError(t, err)
	assert.False(t, res1.AlreadyProcessed)

	// Replay 20 times
	for i := 0; i < 20; i++ {
		resN, err := ProcessStorePurchaseTx(user.Id, verified)
		require.NoError(t, err)
		assert.True(t, resN.AlreadyProcessed)
		assert.Equal(t, res1.UserSubscriptionId, resN.UserSubscriptionId)
		assert.Equal(t, StoreStatusActive, resN.Status)
	}

	// Verify only 1 UserSubscription row exists
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// Verify only 1 StoreTransaction exists
	var txCount int64
	require.NoError(t, DB.Model(&StoreTransaction{}).Where("store_transaction_id = ?", "GPA.1234-5678-9012-34567").Count(&txCount).Error)
	assert.Equal(t, int64(1), txCount)
}

func TestProcessStorePurchase_Concurrency(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{
		Username: "store-concurrent-user",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "aff-concurrent-1",
	}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{
		Title:         "Pro Monthly",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   500000,
		UpgradeGroup:  "pro",
		Enabled:       true,
	}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{
		Platform:       StorePlatformApple,
		StoreProductId: "com.tora.pro.monthly",
		InternalPlanId: plan.Id,
		Environment:    "production",
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&mapping).Error)

	verified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "concurrent_tx_999",
		StoreOriginalId:    "concurrent_orig_999",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}

	const concurrency = 10
	var wg sync.WaitGroup
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			_, errs[idx] = ProcessStorePurchaseTx(user.Id, verified)
		}()
	}
	wg.Wait()

	for _, err := range errs {
		assert.NoError(t, err)
	}

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", user.Id).Count(&subCount).Error)
	assert.Equal(t, int64(1), subCount)
}

func TestProcessStorePurchase_CrossAccountTheftRejection(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	userA := User{Username: "user-a", AffCode: "aff-user-a", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	userB := User{Username: "user-b", AffCode: "aff-user-b", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&userA).Error)
	require.NoError(t, DB.Create(&userB).Error)

	plan := SubscriptionPlan{
		Title:         "Pro Monthly",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   500000,
		UpgradeGroup:  "pro",
		Enabled:       true,
	}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{
		Platform:       StorePlatformApple,
		StoreProductId: "com.tora.pro.monthly",
		InternalPlanId: plan.Id,
		Environment:    "production",
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&mapping).Error)

	// User A legitimately purchases and binds original transaction
	verifiedA := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "apple_tx_user_a",
		StoreOriginalId:    "apple_orig_stolen",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(userA.Id, "Production"),
	}
	_, err := ProcessStorePurchaseTx(userA.Id, verifiedA)
	require.NoError(t, err)

	// User B attempts to claim the same original transaction ID with a renewal or captured proof
	verifiedB := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "apple_tx_user_b_stolen",
		StoreOriginalId:    "apple_orig_stolen",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(userB.Id, "Production"),
	}
	resB, errB := ProcessStorePurchaseTx(userB.Id, verifiedB)
	require.Error(t, errB)
	assert.ErrorIs(t, errB, ErrStoreTransactionBoundToAnotherAccount)
	assert.Nil(t, resB)

	// Verify User B received 0 entitlement
	var countB int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", userB.Id).Count(&countB).Error)
	assert.Equal(t, int64(0), countB)

	var userBRow User
	require.NoError(t, DB.Where("id = ?", userB.Id).First(&userBRow).Error)
	assert.Equal(t, "default", userBRow.Group)
}

func TestProcessStorePurchase_ExpiredTransactionRejected(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{Username: "user-exp", AffCode: "aff-exp", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{Title: "Pro", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{Platform: StorePlatformApple, StoreProductId: "com.tora.pro", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, DB.Create(&mapping).Error)

	expiredVerified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_expired",
		StoreOriginalId:    "orig_expired",
		StoreProductId:     "com.tora.pro",
		PurchaseTime:       now - 60*86400,
		ExpiresTime:        now - 100, // Expired
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}

	_, err := ProcessStorePurchaseTx(user.Id, expiredVerified)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStoreTransactionExpired)
}

func TestProcessStorePurchase_RevokedTransactionRejected(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{Username: "user-rev", AffCode: "aff-rev", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{Title: "Pro", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{Platform: StorePlatformApple, StoreProductId: "com.tora.pro", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, DB.Create(&mapping).Error)

	revokedVerified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_revoked",
		StoreOriginalId:    "orig_revoked",
		StoreProductId:     "com.tora.pro",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		RevocationTime:     now - 10,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}

	_, err := ProcessStorePurchaseTx(user.Id, revokedVerified)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStoreTransactionRevoked)
}

func TestProcessStorePurchase_RenewalExtendsExpiryAndResetsQuota(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{Username: "user-renewal", AffCode: "aff-renewal", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{
		Title:         "Pro Monthly",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   500000,
		UpgradeGroup:  "pro",
		Enabled:       true,
	}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{
		Platform:       StorePlatformApple,
		StoreProductId: "com.tora.pro.monthly",
		InternalPlanId: plan.Id,
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&mapping).Error)

	// 1. Initial purchase
	initialVerified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_period_1",
		StoreOriginalId:    "orig_chain_1",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}
	res1, err := ProcessStorePurchaseTx(user.Id, initialVerified)
	require.NoError(t, err)
	assert.False(t, res1.IsRenewal)

	// Simulate user consuming 250,000 units of quota
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", res1.UserSubscriptionId).Update("amount_used", 250000).Error)

	// 2. Renewal purchase arrives (different transaction ID, same original ID, extended expiry)
	newExpiry := now + 60*86400
	renewalVerified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_period_2",
		StoreOriginalId:    "orig_chain_1",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now + 30*86400,
		ExpiresTime:        newExpiry,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}
	res2, err := ProcessStorePurchaseTx(user.Id, renewalVerified)
	require.NoError(t, err)
	assert.True(t, res2.IsRenewal)
	assert.Equal(t, res1.UserSubscriptionId, res2.UserSubscriptionId)

	// Verify expiry extended and quota reset
	var updatedSub UserSubscription
	require.NoError(t, DB.Where("id = ?", res1.UserSubscriptionId).First(&updatedSub).Error)
	assert.Equal(t, newExpiry, updatedSub.EndTime)
	assert.Equal(t, int64(0), updatedSub.AmountUsed) // reset for new billing cycle

	// Replay renewal: must be idempotent and not change expiry or quota again
	resReplay, err := ProcessStorePurchaseTx(user.Id, renewalVerified)
	require.NoError(t, err)
	assert.True(t, resReplay.AlreadyProcessed)
}

func TestProcessStoreRevocation_ImmediateDowngrade(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{
		Username: "user-to-revoke",
		AffCode:  "aff-revoke",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{
		Title:          "Pro Monthly",
		DurationUnit:   SubscriptionDurationMonth,
		DurationValue:  1,
		TotalAmount:    500000,
		UpgradeGroup:   "pro",
		DowngradeGroup: "default",
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&plan).Error)

	mapping := StoreProductMapping{
		Platform:       StorePlatformApple,
		StoreProductId: "com.tora.pro.monthly",
		InternalPlanId: plan.Id,
		Enabled:        true,
	}
	require.NoError(t, DB.Create(&mapping).Error)

	verified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_refund_target",
		StoreOriginalId:    "orig_refund_target",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}
	res, err := ProcessStorePurchaseTx(user.Id, verified)
	require.NoError(t, err)

	// User is now Pro
	var activeUser User
	require.NoError(t, DB.Where("id = ?", user.Id).First(&activeUser).Error)
	assert.Equal(t, "pro", activeUser.Group)

	// Execute revocation
	err = ProcessStoreRevocationTx(StorePlatformApple, "orig_refund_target", now, "APPLE_CUSTOMER_REFUND")
	require.NoError(t, err)

	// Verify UserSubscription is cancelled and end_time set to now
	var revokedSub UserSubscription
	require.NoError(t, DB.Where("id = ?", res.UserSubscriptionId).First(&revokedSub).Error)
	assert.Equal(t, StoreStatusCancelled, revokedSub.Status)
	assert.True(t, revokedSub.EndTime <= now)

	// Verify User group downgraded back to default
	var downgradedUser User
	require.NoError(t, DB.Where("id = ?", user.Id).First(&downgradedUser).Error)
	assert.Equal(t, "default", downgradedUser.Group)

	// Verify Binding status is revoked
	var binding StoreSubscriptionBinding
	require.NoError(t, DB.Where("platform = ? AND store_original_id = ?", StorePlatformApple, "orig_refund_target").First(&binding).Error)
	assert.Equal(t, StoreStatusRevoked, binding.Status)
}

func TestProcessStorePurchase_UnmappedProductRejected(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{Username: "user-unmapped", AffCode: "aff-unmapped", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&user).Error)

	verified := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_unknown_prod",
		StoreOriginalId:    "orig_unknown_prod",
		StoreProductId:     "com.unknown.product",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    DeriveStoreAccountToken(user.Id, "Production"),
	}

	_, err := ProcessStorePurchaseTx(user.Id, verified)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStoreProductNotMapped)
}

func TestProcessStorePurchase_AccountTokenMismatch_FirstClaimantRejected(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	userA := User{Username: "legit-buyer-a", AffCode: "aff-legit-a", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	userB := User{Username: "attacker-b", AffCode: "aff-atk-b", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&userA).Error)
	require.NoError(t, DB.Create(&userB).Error)

	plan := SubscriptionPlan{Title: "Pro Monthly", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)
	mapping := StoreProductMapping{Platform: StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, DB.Create(&mapping).Error)

	// User A completed purchase on StoreKit with User A's derived appAccountToken
	tokenUserA := DeriveStoreAccountToken(userA.Id, "Production")
	verifiedUserAPurchase := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "apple_tx_fresh_101",
		StoreOriginalId:    "apple_orig_fresh_101",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    tokenUserA, // Cryptographically stamped with User A's token
	}

	// ATTACK SCENARIO: User B intercepts the transaction proof BEFORE User A can verify it with the backend.
	// User B is the FIRST claimant to call ProcessStorePurchaseTx!
	resB, errB := ProcessStorePurchaseTx(userB.Id, verifiedUserAPurchase)
	require.Error(t, errB)
	assert.ErrorIs(t, errB, ErrStoreAccountBindingMismatch, "User B must NOT become the first owner of User A's purchase!")
	assert.Nil(t, resB)

	// Verify no StoreSubscriptionBinding was created for User B or anyone else
	var countBindings int64
	require.NoError(t, DB.Model(&StoreSubscriptionBinding{}).Where("store_original_id = ?", "apple_orig_fresh_101").Count(&countBindings).Error)
	assert.Equal(t, int64(0), countBindings, "No binding should exist after failed attack")

	// Verify User B remained on default group
	var userBRow User
	require.NoError(t, DB.Where("id = ?", userB.Id).First(&userBRow).Error)
	assert.Equal(t, "default", userBRow.Group)

	// LEGITIMATE CLAIM: User A now submits their purchase proof
	resA, errA := ProcessStorePurchaseTx(userA.Id, verifiedUserAPurchase)
	require.NoError(t, errA, "User A must succeed in binding their own transaction")
	assert.Equal(t, StoreStatusActive, resA.Status)

	// Verify User A was upgraded to pro
	var userARow User
	require.NoError(t, DB.Where("id = ?", userA.Id).First(&userARow).Error)
	assert.Equal(t, "pro", userARow.Group)

	// Verify binding is now locked to User A
	var binding StoreSubscriptionBinding
	require.NoError(t, DB.Where("store_original_id = ?", "apple_orig_fresh_101").First(&binding).Error)
	assert.Equal(t, userA.Id, binding.UserId)
}

func TestProcessStorePurchase_StrictAccountTokenRequired(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{Username: "strict-user", AffCode: "aff-strict", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{Title: "Pro Monthly", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)
	mapping := StoreProductMapping{Platform: StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, DB.Create(&mapping).Error)

	// Untokenized purchase (empty AppAccountToken and empty ObfuscatedExternalAccountId)
	untokenized := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_no_token",
		StoreOriginalId:    "orig_no_token",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    "", // Missing!
	}

	// Under strict mode (StoreRequireAccountToken == true)
	_, err := ProcessStorePurchaseTx(user.Id, untokenized)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStoreAccountTokenRequired)
}

func TestProcessStorePurchase_PermissiveLegacyAccountTokenAllowed(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	// Switch to permissive mode temporarily
	prevStrict := setting.StoreRequireAccountToken
	setting.StoreRequireAccountToken = false
	defer func() { setting.StoreRequireAccountToken = prevStrict }()

	user := User{Username: "permissive-user", AffCode: "aff-perm", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{Title: "Pro Monthly", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)
	mapping := StoreProductMapping{Platform: StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, DB.Create(&mapping).Error)

	untokenized := &VerifiedStorePurchase{
		Platform:           StorePlatformApple,
		StoreTransactionId: "tx_legacy_allowed",
		StoreOriginalId:    "orig_legacy_allowed",
		StoreProductId:     "com.tora.pro.monthly",
		PurchaseTime:       now,
		ExpiresTime:        now + 30*86400,
		Environment:        "Production",
		AppAccountToken:    "", // Missing but permissive mode is active
	}

	res, err := ProcessStorePurchaseTx(user.Id, untokenized)
	require.NoError(t, err)
	assert.Equal(t, StoreStatusActive, res.Status)

	var binding StoreSubscriptionBinding
	require.NoError(t, DB.Where("store_original_id = ?", "orig_legacy_allowed").First(&binding).Error)
	assert.Equal(t, user.Id, binding.UserId)
}

func TestProcessStorePurchase_GoogleObfuscatedExternalAccountIdMatch(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	user := User{Username: "google-legit", AffCode: "aff-g-l", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&user).Error)

	plan := SubscriptionPlan{Title: "Pro Monthly", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)
	mapping := StoreProductMapping{Platform: StorePlatformGoogle, StoreProductId: "tora_pro_sub", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, DB.Create(&mapping).Error)

	token := DeriveStoreAccountToken(user.Id, "Production")
	googleVerified := &VerifiedStorePurchase{
		Platform:                    StorePlatformGoogle,
		StoreTransactionId:          "GPA.9999-8888-7777-66666",
		StoreOriginalId:             "GPA.9999-8888-7777-66666",
		StoreProductId:              "tora_pro_sub",
		PurchaseTime:                now,
		ExpiresTime:                 now + 30*86400,
		Environment:                 "Production",
		ObfuscatedExternalAccountId: token, // Matching Google Play token
	}

	res, err := ProcessStorePurchaseTx(user.Id, googleVerified)
	require.NoError(t, err)
	assert.Equal(t, StoreStatusActive, res.Status)
}

func TestProcessStorePurchase_GoogleObfuscatedExternalAccountIdMismatch(t *testing.T) {
	setupStoreBillingTestDB(t)
	now := time.Now().Unix()

	userA := User{Username: "google-victim", AffCode: "aff-g-v", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	userB := User{Username: "google-attacker", AffCode: "aff-g-a", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, DB.Create(&userA).Error)
	require.NoError(t, DB.Create(&userB).Error)

	plan := SubscriptionPlan{Title: "Pro Monthly", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)
	mapping := StoreProductMapping{Platform: StorePlatformGoogle, StoreProductId: "tora_pro_sub", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, DB.Create(&mapping).Error)

	// Token derived from User A
	tokenA := DeriveStoreAccountToken(userA.Id, "Production")
	googleVerifiedA := &VerifiedStorePurchase{
		Platform:                    StorePlatformGoogle,
		StoreTransactionId:          "GPA.1111-2222-3333-44444",
		StoreOriginalId:             "GPA.1111-2222-3333-44444",
		StoreProductId:              "tora_pro_sub",
		PurchaseTime:                now,
		ExpiresTime:                 now + 30*86400,
		Environment:                 "Production",
		ObfuscatedExternalAccountId: tokenA,
	}

	// User B submits User A's Google Play purchase
	resB, errB := ProcessStorePurchaseTx(userB.Id, googleVerifiedA)
	require.Error(t, errB)
	assert.ErrorIs(t, errB, ErrStoreAccountBindingMismatch, "Google Play purchase with User A's obfuscated account ID must reject User B!")
	assert.Nil(t, resB)
}

