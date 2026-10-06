package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type mockAppleVerifier struct {
	verifyFunc      func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error)
	verifyNotifFunc func(ctx context.Context, payload string) (*service.AppleNotificationPayload, error)
}

func (m *mockAppleVerifier) VerifyTransaction(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, token)
	}
	return nil, model.ErrStoreVerificationFailed
}

func (m *mockAppleVerifier) VerifyNotification(ctx context.Context, payload string) (*service.AppleNotificationPayload, error) {
	if m.verifyNotifFunc != nil {
		return m.verifyNotifFunc(ctx, payload)
	}
	return nil, model.ErrStoreVerificationFailed
}

type mockGoogleVerifier struct {
	verifyFunc func(ctx context.Context, productId string, token string) (*model.VerifiedStorePurchase, error)
	ackFunc    func(ctx context.Context, productId string, token string) error
}

func (m *mockGoogleVerifier) VerifySubscription(ctx context.Context, productId string, token string) (*model.VerifiedStorePurchase, error) {
	return m.verifyFunc(ctx, productId, token)
}

func (m *mockGoogleVerifier) AcknowledgeSubscription(ctx context.Context, productId string, token string) error {
	if m.ackFunc != nil {
		return m.ackFunc(ctx, productId, token)
	}
	return nil
}

func setupStoreControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	_ = i18n.Init()
	model.InitCol()
	previousRedis := common.RedisEnabled
	previousMemory := common.MemoryCacheEnabled
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(
		&model.User{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.SubscriptionOrder{},
		&model.TopUp{},
		&model.StoreProductMapping{},
		&model.StoreSubscriptionBinding{},
		&model.StoreTransaction{},
		&model.Log{},
	))
	model.DB, model.LOG_DB = database, database
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		common.RedisEnabled = previousRedis
		common.MemoryCacheEnabled = previousMemory
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousType, previousLogType)
		_ = sqlDB.Close()
	})
	return database
}

func TestVerifyAppleSubscription_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBufferString(`{}`))
	// No user ID in context

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestVerifyAppleSubscription_MissingPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", 100)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBufferString(`{}`))

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, ErrCodeStoreProofInvalid, resp["code"])
}

func TestVerifyAppleSubscription_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	user := model.User{Username: "apple-user", AffCode: "aff-apple-u", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&user).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)

	mapping := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	now := time.Now().Unix()
	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_mock_apple_1",
				StoreOriginalId:    "orig_mock_apple_1",
				StoreProductId:     "com.tora.pro.monthly",
				PurchaseTime:       now,
				ExpiresTime:        now + 30*86400,
				Environment:        "Production",
				AppAccountToken:    service.DeriveStoreAccountToken(user.Id, "Production"),
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", user.Id)
	body, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "mock-jws-token"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]any)
	assert.Equal(t, model.StoreStatusActive, data["status"])
	assert.Equal(t, float64(plan.Id), data["plan_id"])
}

func TestVerifyAppleSubscription_CrossAccountConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	userA := model.User{Username: "user-legit", AffCode: "aff-legit", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	userB := model.User{Username: "user-thief", AffCode: "aff-thief", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&userA).Error)
	require.NoError(t, model.DB.Create(&userB).Error)

	plan := model.SubscriptionPlan{Title: "Pro", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.tora.pro", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	// User A owns the original transaction
	binding := model.StoreSubscriptionBinding{
		Platform:        model.StorePlatformApple,
		StoreOriginalId: "orig_shared",
		UserId:          userA.Id,
		InternalPlanId:  plan.Id,
		Status:          model.StoreStatusActive,
	}
	require.NoError(t, model.DB.Create(&binding).Error)

	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_shared_2",
				StoreOriginalId:    "orig_shared",
				StoreProductId:     "com.tora.pro",
				PurchaseTime:       time.Now().Unix(),
				ExpiresTime:        time.Now().Unix() + 30*86400,
				Environment:        "Production",
				AppAccountToken:    service.DeriveStoreAccountToken(userA.Id, "Production"),
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	// User B submits
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", userB.Id)
	body, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "token"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, ErrCodeStoreTransactionAlreadyBound, resp["code"])
}

func TestVerifyGoogleSubscription_Pending(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	user := model.User{Username: "google-pending-u", AffCode: "aff-g-p", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(&user).Error)

	prevVerifier := service.GlobalGoogleVerifier
	service.SetGoogleVerifierForTest(&mockGoogleVerifier{
		verifyFunc: func(ctx context.Context, productId string, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformGoogle,
				StoreTransactionId: "g_tx_pending",
				StoreOriginalId:    "g_orig_pending",
				StoreProductId:     productId,
				IsPending:          true,
			}, nil
		},
	})
	t.Cleanup(func() { service.SetGoogleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", user.Id)
	body, _ := json.Marshal(GoogleVerifyRequest{PurchaseToken: "tok_pend", ProductId: "prod_pend"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/google/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyGoogleSubscription(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	assert.Equal(t, ErrCodeStoreVerificationPending, resp["code"])
}

func TestAppleSubscriptionWebhook_RefundRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	user := model.User{Username: "webhook-u", AffCode: "aff-wh-u", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "pro"}
	require.NoError(t, model.DB.Create(&user).Error)

	sub := model.UserSubscription{UserId: user.Id, PlanId: 1, Status: model.StoreStatusActive, EndTime: now + 86400, PrevUserGroup: "default", UpgradeGroup: "pro"}
	require.NoError(t, model.DB.Create(&sub).Error)

	binding := model.StoreSubscriptionBinding{
		Platform:        model.StorePlatformApple,
		StoreOriginalId: "orig_wh_refund",
		UserId:          user.Id,
		ActiveUserSubId: sub.Id,
		Status:          model.StoreStatusActive,
	}
	require.NoError(t, model.DB.Create(&binding).Error)

	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyNotifFunc: func(ctx context.Context, payload string) (*service.AppleNotificationPayload, error) {
			notif := &service.AppleNotificationPayload{
				NotificationType: "REFUND",
			}
			notif.Data.SignedTransactionInfo = "mock-tx-token"
			return notif, nil
		},
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_wh_refund",
				StoreOriginalId:    "orig_wh_refund",
				StoreProductId:     "com.tora.pro",
				RevocationTime:     now,
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqBody, _ := json.Marshal(AppleWebhookRequest{SignedPayload: "valid-jws-payload"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/webhook", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	AppleSubscriptionWebhook(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify UserSubscription was cancelled
	var revokedSub model.UserSubscription
	require.NoError(t, model.DB.Where("id = ?", sub.Id).First(&revokedSub).Error)
	assert.Equal(t, model.StoreStatusCancelled, revokedSub.Status)

	// Verify User group downgraded
	var updatedUser model.User
	require.NoError(t, model.DB.Where("id = ?", user.Id).First(&updatedUser).Error)
	assert.Equal(t, "default", updatedUser.Group)
}

func TestAppleSubscriptionWebhook_InvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyNotifFunc: func(ctx context.Context, payload string) (*service.AppleNotificationPayload, error) {
			return nil, model.ErrStoreVerificationFailed
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqBody, _ := json.Marshal(AppleWebhookRequest{SignedPayload: "tampered-payload"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/webhook", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	AppleSubscriptionWebhook(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGoogleSubscriptionWebhook_Revocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	user := model.User{Username: "google-wh-u", AffCode: "aff-gwh-u", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "pro"}
	require.NoError(t, model.DB.Create(&user).Error)

	sub := model.UserSubscription{UserId: user.Id, PlanId: 1, Status: model.StoreStatusActive, EndTime: now + 86400, PrevUserGroup: "default", UpgradeGroup: "pro"}
	require.NoError(t, model.DB.Create(&sub).Error)

	binding := model.StoreSubscriptionBinding{
		Platform:        model.StorePlatformGoogle,
		StoreOriginalId: "google_tok_123",
		UserId:          user.Id,
		ActiveUserSubId: sub.Id,
		Status:          model.StoreStatusActive,
	}
	require.NoError(t, model.DB.Create(&binding).Error)

	prevVerifier := service.GlobalGoogleVerifier
	service.SetGoogleVerifierForTest(&mockGoogleVerifier{
		verifyFunc: func(ctx context.Context, productId string, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformGoogle,
				StoreTransactionId: "google_tok_123",
				StoreOriginalId:    "google_tok_123",
				StoreProductId:     productId,
				RevocationTime:     now,
			}, nil
		},
	})
	t.Cleanup(func() { service.SetGoogleVerifierForTest(prevVerifier) })

	rtdn := map[string]any{
		"version":         "1.0",
		"packageName":     "me.huanmeng.lumenflow",
		"eventTimeMillis": now * 1000,
		"subscriptionNotification": map[string]any{
			"version":          "1.0",
			"notificationType": 12, // REVOKED
			"purchaseToken":    "google_tok_123",
			"subscriptionId":   "tora_pro_sub",
		},
	}
	rtdnJSON, _ := json.Marshal(rtdn)
	pubsubPayload := GooglePubSubPushRequest{}
	pubsubPayload.Message.Data = base64.StdEncoding.EncodeToString(rtdnJSON)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqBody, _ := json.Marshal(pubsubPayload)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/google/webhook", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	GoogleSubscriptionWebhook(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify UserSubscription was cancelled
	var revokedSub model.UserSubscription
	require.NoError(t, model.DB.Where("id = ?", sub.Id).First(&revokedSub).Error)
	assert.Equal(t, model.StoreStatusCancelled, revokedSub.Status)

	// Verify User group downgraded
	var updatedUser model.User
	require.NoError(t, model.DB.Where("id = ?", user.Id).First(&updatedUser).Error)
	assert.Equal(t, "default", updatedUser.Group)
}

func TestGoogleSubscriptionWebhook_PubSubTokenUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	prevToken := setting.GooglePubSubVerificationToken
	setting.GooglePubSubVerificationToken = "secret-pubsub-token"
	t.Cleanup(func() { setting.GooglePubSubVerificationToken = prevToken })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/google/webhook?token=wrong-token", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	GoogleSubscriptionWebhook(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGoogleSubscriptionWebhook_DefenseInDepthRejectsUnverified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	user := model.User{Username: "google-victim", AffCode: "aff-gwh-v", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "pro"}
	require.NoError(t, model.DB.Create(&user).Error)

	sub := model.UserSubscription{UserId: user.Id, PlanId: 1, Status: model.StoreStatusActive, EndTime: now + 86400, PrevUserGroup: "default", UpgradeGroup: "pro"}
	require.NoError(t, model.DB.Create(&sub).Error)

	binding := model.StoreSubscriptionBinding{
		Platform:        model.StorePlatformGoogle,
		StoreOriginalId: "google_tok_victim",
		UserId:          user.Id,
		ActiveUserSubId: sub.Id,
		Status:          model.StoreStatusActive,
	}
	require.NoError(t, model.DB.Create(&binding).Error)

	// Google verifier fails authoritative check
	prevVerifier := service.GlobalGoogleVerifier
	service.SetGoogleVerifierForTest(&mockGoogleVerifier{
		verifyFunc: func(ctx context.Context, productId string, token string) (*model.VerifiedStorePurchase, error) {
			return nil, model.ErrStoreVerificationFailed
		},
	})
	t.Cleanup(func() { service.SetGoogleVerifierForTest(prevVerifier) })

	// Spoofed RTDN attempting to revoke victim's subscription
	rtdn := map[string]any{
		"version":         "1.0",
		"packageName":     "me.huanmeng.lumenflow",
		"eventTimeMillis": now * 1000,
		"subscriptionNotification": map[string]any{
			"version":          "1.0",
			"notificationType": 12,
			"purchaseToken":    "google_tok_victim",
			"subscriptionId":   "tora_pro_sub",
		},
	}
	rtdnJSON, _ := json.Marshal(rtdn)
	pubsubPayload := GooglePubSubPushRequest{}
	pubsubPayload.Message.Data = base64.StdEncoding.EncodeToString(rtdnJSON)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqBody, _ := json.Marshal(pubsubPayload)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/google/webhook", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	GoogleSubscriptionWebhook(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// Ensure victim's subscription is UNTOUCHED (still Active)
	var activeSub model.UserSubscription
	require.NoError(t, model.DB.Where("id = ?", sub.Id).First(&activeSub).Error)
	assert.Equal(t, model.StoreStatusActive, activeSub.Status)

	var checkUser model.User
	require.NoError(t, model.DB.Where("id = ?", user.Id).First(&checkUser).Error)
	assert.Equal(t, "pro", checkUser.Group)
}

func TestGetStoreProductCatalog_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	planPro := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, Enabled: true}
	require.NoError(t, model.DB.Create(&planPro).Error)

	mappingApple := model.StoreProductMapping{
		Platform:        model.StorePlatformApple,
		StoreProductId:  "com.saascover.tora.pro.monthly",
		InternalPlanId:  planPro.Id,
		Environment:     "production",
		Enabled:         true,
	}
	require.NoError(t, model.DB.Create(&mappingApple).Error)

	mappingGoogle := model.StoreProductMapping{
		Platform:        model.StorePlatformGoogle,
		StoreProductId:  "tora_pro_sub",
		StoreBasePlanId: "pro-monthly",
		InternalPlanId:  planPro.Id,
		Environment:     "production",
		Enabled:         true,
	}
	require.NoError(t, model.DB.Create(&mappingGoogle).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/subscription/store/products", nil)

	GetStoreProductCatalog(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                         `json:"success"`
		Data    []model.StoreCatalogProduct `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data, 2)
}

func TestGetStoreProductCatalog_FilterPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	planPro := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, Enabled: true}
	require.NoError(t, model.DB.Create(&planPro).Error)

	mappingApple := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.saascover.tora.pro.monthly", InternalPlanId: planPro.Id, Environment: "production", Enabled: true}
	require.NoError(t, model.DB.Create(&mappingApple).Error)
	mappingGoogle := model.StoreProductMapping{Platform: model.StorePlatformGoogle, StoreProductId: "tora_pro_sub", StoreBasePlanId: "pro-monthly", InternalPlanId: planPro.Id, Environment: "production", Enabled: true}
	require.NoError(t, model.DB.Create(&mappingGoogle).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/subscription/store/products?platform=apple", nil)

	GetStoreProductCatalog(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                         `json:"success"`
		Data    []model.StoreCatalogProduct `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "apple", resp.Data[0].Platform)
	assert.Equal(t, "com.saascover.tora.pro.monthly", resp.Data[0].StoreProductId)
}

// -----------------------------------------------------------------------------
// Phase 7G-B-R Native Billing Account Binding Attack & Hardening Test Suite
// -----------------------------------------------------------------------------

// Scenario A: User B intercepts User A's unconsumed Apple transaction with User A's appAccountToken.
// User B attempts to become the first claimant owner.
// MUST BE REJECTED with 403 STORE_ACCOUNT_TOKEN_MISMATCH.
// Then User A submits -> MUST BE ACCEPTED and bound to User A.
func TestVerifyAppleSubscription_AttackScenarioA_FirstClaimantTheftBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	userA := model.User{Username: "victim-a", AffCode: "aff-va", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	userB := model.User{Username: "thief-b", AffCode: "aff-tb", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&userA).Error)
	require.NoError(t, model.DB.Create(&userB).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	// User A purchased on device; Apple JWS contains User A's appAccountToken
	tokenUserA := service.DeriveStoreAccountToken(userA.Id, "Production")
	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_intercepted_001",
				StoreOriginalId:    "orig_intercepted_001",
				StoreProductId:     "com.tora.pro.monthly",
				PurchaseTime:       now,
				ExpiresTime:        now + 30*86400,
				Environment:        "Production",
				AppAccountToken:    tokenUserA, // Stamped by StoreKit with User A's identity
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	// ATTACK: User B calls verify first!
	wB := httptest.NewRecorder()
	cB, _ := gin.CreateTestContext(wB)
	cB.Set("id", userB.Id)
	bodyB, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "intercepted-jws-token"})
	cB.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(bodyB))
	cB.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(cB)
	assert.Equal(t, http.StatusForbidden, wB.Code)

	var respB map[string]any
	require.NoError(t, json.Unmarshal(wB.Body.Bytes(), &respB))
	assert.False(t, respB["success"].(bool))
	assert.Equal(t, ErrCodeStoreAccountTokenMismatch, respB["code"])

	// Verify no binding exists
	var countBindings int64
	require.NoError(t, model.DB.Model(&model.StoreSubscriptionBinding{}).Where("store_original_id = ?", "orig_intercepted_001").Count(&countBindings).Error)
	assert.Equal(t, int64(0), countBindings)

	// Verify User B received 0 entitlement
	var userBRow model.User
	require.NoError(t, model.DB.Where("id = ?", userB.Id).First(&userBRow).Error)
	assert.Equal(t, "default", userBRow.Group)

	// LEGITIMATE CLAIM: User A verifies their own purchase
	wA := httptest.NewRecorder()
	cA, _ := gin.CreateTestContext(wA)
	cA.Set("id", userA.Id)
	bodyA, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "legit-jws-token"})
	cA.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(bodyA))
	cA.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(cA)
	assert.Equal(t, http.StatusOK, wA.Code)

	var respA map[string]any
	require.NoError(t, json.Unmarshal(wA.Body.Bytes(), &respA))
	assert.True(t, respA["success"].(bool))

	// Verify User A upgraded to pro and bound to transaction
	var userARow model.User
	require.NoError(t, model.DB.Where("id = ?", userA.Id).First(&userARow).Error)
	assert.Equal(t, "pro", userARow.Group)

	var binding model.StoreSubscriptionBinding
	require.NoError(t, model.DB.Where("store_original_id = ?", "orig_intercepted_001").First(&binding).Error)
	assert.Equal(t, userA.Id, binding.UserId)
}

// Scenario B: User B submits valid Apple transaction with User B's own appAccountToken.
// MUST BE ACCEPTED and bound to User B.
func TestVerifyAppleSubscription_ScenarioB_ValidUserBClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	userB := model.User{Username: "legit-b", AffCode: "aff-lb", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&userB).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	tokenUserB := service.DeriveStoreAccountToken(userB.Id, "Production")
	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_valid_b",
				StoreOriginalId:    "orig_valid_b",
				StoreProductId:     "com.tora.pro.monthly",
				PurchaseTime:       now,
				ExpiresTime:        now + 30*86400,
				Environment:        "Production",
				AppAccountToken:    tokenUserB,
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", userB.Id)
	body, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "user-b-jws"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	var binding model.StoreSubscriptionBinding
	require.NoError(t, model.DB.Where("store_original_id = ?", "orig_valid_b").First(&binding).Error)
	assert.Equal(t, userB.Id, binding.UserId)
}

// Scenario C: User A buys and binds. User B attempts restore/replay.
// MUST BE REJECTED with 409 STORE_TRANSACTION_ALREADY_BOUND.
func TestVerifyAppleSubscription_ScenarioC_ReplayAfterBindingBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	userA := model.User{Username: "owner-a", AffCode: "aff-oa", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	userB := model.User{Username: "replay-b", AffCode: "aff-rb", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&userA).Error)
	require.NoError(t, model.DB.Create(&userB).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	// User A has already bound this subscription
	bindingA := model.StoreSubscriptionBinding{
		Platform:        model.StorePlatformApple,
		StoreOriginalId: "orig_locked_to_a",
		UserId:          userA.Id,
		InternalPlanId:  plan.Id,
		Status:          model.StoreStatusActive,
	}
	require.NoError(t, model.DB.Create(&bindingA).Error)

	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_locked_to_a",
				StoreOriginalId:    "orig_locked_to_a",
				StoreProductId:     "com.tora.pro.monthly",
				PurchaseTime:       now,
				ExpiresTime:        now + 30*86400,
				Environment:        "Production",
				AppAccountToken:    service.DeriveStoreAccountToken(userA.Id, "Production"),
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", userB.Id) // User B tries to claim
	body, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "jws"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	assert.Equal(t, ErrCodeStoreTransactionAlreadyBound, resp["code"])
}

// Scenario D: Google purchase with mismatched obfuscatedExternalAccountId.
// MUST BE REJECTED with 403 STORE_ACCOUNT_TOKEN_MISMATCH.
func TestVerifyGoogleSubscription_AttackScenarioD_AccountMismatchBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	userA := model.User{Username: "google-victim", AffCode: "aff-gv", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	userB := model.User{Username: "google-thief", AffCode: "aff-gt", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&userA).Error)
	require.NoError(t, model.DB.Create(&userB).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformGoogle, StoreProductId: "tora_pro_sub", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	// Google Play API response contains User A's obfuscated account ID
	tokenUserA := service.DeriveStoreAccountToken(userA.Id, "Production")
	prevVerifier := service.GlobalGoogleVerifier
	service.SetGoogleVerifierForTest(&mockGoogleVerifier{
		verifyFunc: func(ctx context.Context, productId string, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:                    model.StorePlatformGoogle,
				StoreTransactionId:          "GPA.5555-4444-3333-22222",
				StoreOriginalId:             "GPA.5555-4444-3333-22222",
				StoreProductId:              productId,
				PurchaseTime:                now,
				ExpiresTime:                 now + 30*86400,
				Environment:                 "Production",
				ObfuscatedExternalAccountId: tokenUserA, // Stamped with User A
			}, nil
		},
	})
	t.Cleanup(func() { service.SetGoogleVerifierForTest(prevVerifier) })

	// User B submits User A's Google Play purchase
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", userB.Id)
	body, _ := json.Marshal(GoogleVerifyRequest{PurchaseToken: "stolen-google-tok", ProductId: "tora_pro_sub"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/google/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyGoogleSubscription(c)
	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	assert.Equal(t, ErrCodeStoreAccountTokenMismatch, resp["code"])
}

// Scenario E: Google purchase with matching obfuscatedExternalAccountId.
// MUST BE ACCEPTED and acknowledged.
func TestVerifyGoogleSubscription_ScenarioE_ValidAccountMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	user := model.User{Username: "google-legit-user", AffCode: "aff-glu", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&user).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformGoogle, StoreProductId: "tora_pro_sub", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	tokenUser := service.DeriveStoreAccountToken(user.Id, "Production")
	ackCalled := false
	prevVerifier := service.GlobalGoogleVerifier
	service.SetGoogleVerifierForTest(&mockGoogleVerifier{
		verifyFunc: func(ctx context.Context, productId string, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:                    model.StorePlatformGoogle,
				StoreTransactionId:          "GPA.7777-8888-9999-00000",
				StoreOriginalId:             "GPA.7777-8888-9999-00000",
				StoreProductId:              productId,
				PurchaseTime:                now,
				ExpiresTime:                 now + 30*86400,
				Environment:                 "Production",
				ObfuscatedExternalAccountId: tokenUser,
			}, nil
		},
		ackFunc: func(ctx context.Context, productId string, token string) error {
			ackCalled = true
			return nil
		},
	})
	t.Cleanup(func() { service.SetGoogleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", user.Id)
	body, _ := json.Marshal(GoogleVerifyRequest{PurchaseToken: "valid-tok", ProductId: "tora_pro_sub"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/google/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyGoogleSubscription(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, ackCalled, "Google subscription must be acknowledged on success")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
}

// Scenario F: Missing account token on new purchase when StoreRequireAccountToken is true.
// MUST BE REJECTED with 400 STORE_ACCOUNT_TOKEN_REQUIRED.
func TestVerifyAppleSubscription_ScenarioF_StrictAccountTokenRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	now := time.Now().Unix()
	user := model.User{Username: "strict-user", AffCode: "aff-su", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&user).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_strict_no_token",
				StoreOriginalId:    "orig_strict_no_token",
				StoreProductId:     "com.tora.pro.monthly",
				PurchaseTime:       now,
				ExpiresTime:        now + 30*86400,
				Environment:        "Production",
				AppAccountToken:    "", // Missing!
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", user.Id)
	body, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "no-token-jws"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	assert.Equal(t, ErrCodeStoreAccountTokenRequired, resp["code"])
}

// Scenario G: Missing account token on legacy purchase when StoreRequireAccountToken is false.
// MUST BE ACCEPTED (permissive mode for legacy migration).
func TestVerifyAppleSubscription_ScenarioG_PermissiveModeAllowsLegacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupStoreControllerTestDB(t)

	prevStrict := setting.StoreRequireAccountToken
	setting.StoreRequireAccountToken = false
	defer func() { setting.StoreRequireAccountToken = prevStrict }()

	now := time.Now().Unix()
	user := model.User{Username: "perm-user", AffCode: "aff-pu", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&user).Error)

	plan := model.SubscriptionPlan{Title: "Pro Monthly", DurationUnit: model.SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 500000, UpgradeGroup: "pro", Enabled: true}
	require.NoError(t, model.DB.Create(&plan).Error)
	mapping := model.StoreProductMapping{Platform: model.StorePlatformApple, StoreProductId: "com.tora.pro.monthly", InternalPlanId: plan.Id, Enabled: true}
	require.NoError(t, model.DB.Create(&mapping).Error)

	prevVerifier := service.GlobalAppleVerifier
	service.SetAppleVerifierForTest(&mockAppleVerifier{
		verifyFunc: func(ctx context.Context, token string) (*model.VerifiedStorePurchase, error) {
			return &model.VerifiedStorePurchase{
				Platform:           model.StorePlatformApple,
				StoreTransactionId: "tx_perm_legacy",
				StoreOriginalId:    "orig_perm_legacy",
				StoreProductId:     "com.tora.pro.monthly",
				PurchaseTime:       now,
				ExpiresTime:        now + 30*86400,
				Environment:        "Production",
				AppAccountToken:    "", // Missing, but permissive mode is active
			}, nil
		},
	})
	t.Cleanup(func() { service.SetAppleVerifierForTest(prevVerifier) })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", user.Id)
	body, _ := json.Marshal(AppleVerifyRequest{SignedTransactionInfo: "legacy-jws"})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/apple/verify", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	VerifyAppleSubscription(c)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	var binding model.StoreSubscriptionBinding
	require.NoError(t, model.DB.Where("store_original_id = ?", "orig_perm_legacy").First(&binding).Error)
	assert.Equal(t, user.Id, binding.UserId)
}
