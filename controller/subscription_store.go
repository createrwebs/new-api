package controller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

// Sanitized mobile error codes
const (
	ErrCodeStoreVerificationPending     = "STORE_VERIFICATION_PENDING"
	ErrCodeStoreProofInvalid           = "STORE_PROOF_INVALID"
	ErrCodeStoreProductUnrecognized    = "STORE_PRODUCT_UNRECOGNIZED"
	ErrCodeStoreTransactionAlreadyBound = "STORE_TRANSACTION_ALREADY_BOUND"
	ErrCodeStoreSubscriptionExpired    = "STORE_SUBSCRIPTION_EXPIRED"
	ErrCodeStoreSubscriptionRevoked    = "STORE_SUBSCRIPTION_REVOKED"
	ErrCodeStoreServerUnavailable      = "STORE_SERVER_UNAVAILABLE"
	ErrCodeStoreEnvironmentMismatch    = "STORE_ENVIRONMENT_MISMATCH"
	ErrCodeStoreBundleMismatch         = "STORE_BUNDLE_MISMATCH"
	ErrCodeStoreAccountTokenMismatch   = "STORE_ACCOUNT_TOKEN_MISMATCH"
	ErrCodeStoreAccountTokenRequired   = "STORE_ACCOUNT_TOKEN_REQUIRED"
)

type AppleVerifyRequest struct {
	SignedTransactionInfo string `json:"signed_transaction_info"`
}

type GoogleVerifyRequest struct {
	PurchaseToken string `json:"purchase_token"`
	ProductId     string `json:"product_id"`
}

type AppleWebhookRequest struct {
	SignedPayload string `json:"signedPayload"`
}

type GooglePubSubPushRequest struct {
	Message struct {
		Data        string `json:"data"`
		MessageId   string `json:"messageId"`
		PublishTime string `json:"publishTime"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

func mapStoreError(err error) (string, int) {
	if err == nil {
		return "", http.StatusOK
	}
	switch {
	case errors.Is(err, model.ErrStoreServerNotConfigured):
		return ErrCodeStoreServerUnavailable, http.StatusServiceUnavailable
	case errors.Is(err, model.ErrStoreTransactionBoundToAnotherAccount):
		return ErrCodeStoreTransactionAlreadyBound, http.StatusConflict
	case errors.Is(err, model.ErrStoreProductNotMapped):
		return ErrCodeStoreProductUnrecognized, http.StatusBadRequest
	case errors.Is(err, model.ErrStoreBundleMismatch):
		return ErrCodeStoreBundleMismatch, http.StatusBadRequest
	case errors.Is(err, model.ErrStoreEnvironmentMismatch):
		return ErrCodeStoreEnvironmentMismatch, http.StatusBadRequest
	case errors.Is(err, model.ErrStoreTransactionExpired):
		return ErrCodeStoreSubscriptionExpired, http.StatusBadRequest
	case errors.Is(err, model.ErrStoreTransactionRevoked):
		return ErrCodeStoreSubscriptionRevoked, http.StatusBadRequest
	case errors.Is(err, model.ErrStoreAccountBindingMismatch):
		return ErrCodeStoreAccountTokenMismatch, http.StatusForbidden
	case errors.Is(err, model.ErrStoreAccountTokenRequired):
		return ErrCodeStoreAccountTokenRequired, http.StatusBadRequest
	case errors.Is(err, model.ErrStoreVerificationFailed):
		return ErrCodeStoreProofInvalid, http.StatusBadRequest
	default:
		return ErrCodeStoreProofInvalid, http.StatusBadRequest
	}
}

// GetStoreProductCatalog handles GET /api/subscription/store/products
func GetStoreProductCatalog(c *gin.Context) {
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))

	// Determine authoritative environment based on settings
	env := "production"
	if setting.AppleAllowSandbox || setting.GoogleAllowTestPurchase {
		env = "sandbox"
	}
	if reqEnv := strings.ToLower(strings.TrimSpace(c.Query("env"))); reqEnv != "" && (setting.AppleAllowSandbox || setting.GoogleAllowTestPurchase) {
		env = reqEnv
	}

	catalog, err := model.GetEnabledStoreProductCatalog(platform, env)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to load store product catalog",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    catalog,
	})
}

// VerifyAppleSubscription handles POST /api/subscription/apple/verify
func VerifyAppleSubscription(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"code":    "AUTH_REQUIRED",
			"message": "User authentication required",
		})
		return
	}

	var req AppleVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.SignedTransactionInfo) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"code":    ErrCodeStoreProofInvalid,
			"message": "Invalid request: signed_transaction_info required",
		})
		return
	}

	// 1. Authoritative store verification against Apple Root CA
	verified, err := service.GlobalAppleVerifier.VerifyTransaction(c.Request.Context(), req.SignedTransactionInfo)
	if err != nil {
		code, status := mapStoreError(err)
		c.JSON(status, gin.H{
			"success": false,
			"code":    code,
			"message": err.Error(),
		})
		return
	}

	// 2. Transactional settlement and entitlement grant
	result, err := model.ProcessStorePurchaseTx(userId, verified)
	if err != nil {
		code, status := mapStoreError(err)
		c.JSON(status, gin.H{
			"success": false,
			"code":    code,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Store purchase verified",
		"data":    result,
	})
}

// VerifyGoogleSubscription handles POST /api/subscription/google/verify
func VerifyGoogleSubscription(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"code":    "AUTH_REQUIRED",
			"message": "User authentication required",
		})
		return
	}

	var req GoogleVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.PurchaseToken) == "" || strings.TrimSpace(req.ProductId) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"code":    ErrCodeStoreProofInvalid,
			"message": "Invalid request: purchase_token and product_id required",
		})
		return
	}

	// 1. Authoritative store verification
	verified, err := service.GlobalGoogleVerifier.VerifySubscription(c.Request.Context(), req.ProductId, req.PurchaseToken)
	if err != nil {
		code, status := mapStoreError(err)
		c.JSON(status, gin.H{
			"success": false,
			"code":    code,
			"message": err.Error(),
		})
		return
	}

	if verified.IsPending {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"code":    ErrCodeStoreVerificationPending,
			"message": "Purchase is still pending store settlement",
		})
		return
	}

	// 2. Transactional settlement and entitlement grant
	result, err := model.ProcessStorePurchaseTx(userId, verified)
	if err != nil {
		code, status := mapStoreError(err)
		c.JSON(status, gin.H{
			"success": false,
			"code":    code,
			"message": err.Error(),
		})
		return
	}

	// 3. Acknowledge with Google
	_ = service.GlobalGoogleVerifier.AcknowledgeSubscription(c.Request.Context(), req.ProductId, req.PurchaseToken)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Store purchase verified",
		"data":    result,
	})
}

// AppleSubscriptionWebhook handles App Store Server Notifications V2
func AppleSubscriptionWebhook(c *gin.Context) {
	var req AppleWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.SignedPayload) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "signedPayload required"})
		return
	}

	// 1. Authoritative verification of outer notification JWS against Apple Root CA
	notif, err := service.GlobalAppleVerifier.VerifyNotification(c.Request.Context(), req.SignedPayload)
	if err != nil {
		logger.LogWarn(c.Request.Context(), "Apple webhook notification verification failed: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification verification failed", "message": err.Error()})
		return
	}

	if notif.Data.SignedTransactionInfo == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored_no_transaction"})
		return
	}

	// 2. Authoritative verification of inner transaction JWS against Apple Root CA
	verified, err := service.GlobalAppleVerifier.VerifyTransaction(c.Request.Context(), notif.Data.SignedTransactionInfo)
	if err != nil {
		logger.LogWarn(c.Request.Context(), "Apple webhook transaction verification failed: "+err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "transaction verification failed", "message": err.Error()})
		return
	}

	switch notif.NotificationType {
	case "REVOKE", "REFUND":
		_ = model.ProcessStoreRevocationTx(model.StorePlatformApple, verified.StoreOriginalId, verified.RevocationTime, notif.NotificationType)
	case "SUBSCRIBED", "DID_RENEW":
		// Find user from existing binding
		var binding model.StoreSubscriptionBinding
		if err := model.DB.Where("platform = ? AND store_original_id = ?", model.StorePlatformApple, verified.StoreOriginalId).First(&binding).Error; err == nil {
			_, _ = model.ProcessStorePurchaseTx(binding.UserId, verified)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GoogleSubscriptionWebhook handles Google RTDN Pub/Sub Push Notifications
func GoogleSubscriptionWebhook(c *gin.Context) {
	// 1. Verify Pub/Sub push token if configured
	if setting.GooglePubSubVerificationToken != "" {
		queryToken := c.Query("token")
		headerToken := c.GetHeader("X-Goog-PubSub-Verification-Token")
		if queryToken != setting.GooglePubSubVerificationToken && headerToken != setting.GooglePubSubVerificationToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid pubsub verification token"})
			return
		}
	}

	var req GooglePubSubPushRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Message.Data == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pubsub message data required"})
		return
	}

	dataBytes, err := base64.StdEncoding.DecodeString(req.Message.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode pubsub data"})
		return
	}

	var rtdn struct {
		Version                  string `json:"version"`
		PackageName              string `json:"packageName"`
		EventTimeMillis          int64  `json:"eventTimeMillis"`
		SubscriptionNotification struct {
			Version          string `json:"version"`
			NotificationType int    `json:"notificationType"`
			PurchaseToken    string `json:"purchaseToken"`
			SubscriptionId   string `json:"subscriptionId"`
		} `json:"subscriptionNotification"`
	}

	if err := json.Unmarshal(dataBytes, &rtdn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse rtdn"})
		return
	}

	// Validate package name if configured
	if setting.GooglePackageName != "" && rtdn.PackageName != "" && rtdn.PackageName != setting.GooglePackageName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package name mismatch"})
		return
	}

	sub := rtdn.SubscriptionNotification
	if sub.PurchaseToken == "" || sub.SubscriptionId == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ignored_no_subscription"})
		return
	}

	// Defense-in-depth: Authoritatively query Google Play Developer API before ANY database mutation
	verified, err := service.GlobalGoogleVerifier.VerifySubscription(c.Request.Context(), sub.SubscriptionId, sub.PurchaseToken)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("Google RTDN authoritative verification failed for sub %s: %v", sub.SubscriptionId, err))
		// Do not mutate database if Google Play API cannot verify the subscription
		c.JSON(http.StatusOK, gin.H{"status": "verification_failed", "error": err.Error()})
		return
	}

	// Notification types:
	// 2 = SUBSCRIPTION_RENEWED
	// 4 = SUBSCRIPTION_PURCHASED
	// 12 = SUBSCRIPTION_REVOKED
	// 13 = SUBSCRIPTION_EXPIRED
	if sub.NotificationType == 12 || verified.RevocationTime > 0 {
		revocationTime := verified.RevocationTime
		if revocationTime == 0 {
			revocationTime = rtdn.EventTimeMillis / 1000
		}
		_ = model.ProcessStoreRevocationTx(model.StorePlatformGoogle, sub.PurchaseToken, revocationTime, "GOOGLE_RTDN_REVOKED")
	} else if sub.NotificationType == 2 || sub.NotificationType == 4 || !verified.IsPending {
		// Find existing binding by purchase token OR linkedPurchaseToken chain
		var binding model.StoreSubscriptionBinding
		findQuery := model.DB.Where("platform = ? AND store_original_id = ?", model.StorePlatformGoogle, sub.PurchaseToken)
		if err := findQuery.First(&binding).Error; err != nil && verified.LinkedPurchaseToken != "" {
			_ = model.DB.Where("platform = ? AND store_original_id = ?", model.StorePlatformGoogle, verified.LinkedPurchaseToken).First(&binding).Error
		}

		if binding.UserId > 0 {
			_, _ = model.ProcessStorePurchaseTx(binding.UserId, verified)
			_ = service.GlobalGoogleVerifier.AcknowledgeSubscription(c.Request.Context(), sub.SubscriptionId, sub.PurchaseToken)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
