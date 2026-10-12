package controller

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// authenticateHuobaoToken validates the bearer token specifically for Huobao identity & credits verification.
// Invariant A0: Separate key/account validity from quota.
// A token with exhausted or zero quota is VALID for identity, allowing users to log in, view existing projects,
// and configure direct BYOK keys.
// However, disabled tokens, expired tokens, deleted tokens, disallowed IPs, and banned user accounts MUST be rejected.
func authenticateHuobaoToken(c *gin.Context) (*model.Token, *model.User, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, nil, errors.New("authorization header is required")
	}
	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, nil, errors.New("invalid authorization header format, Bearer token required")
	}
	rawKey := parts[1]
	key := strings.TrimPrefix(rawKey, "sk-")
	keyParts := strings.Split(key, "-")
	key = keyParts[0]

	token, err := model.GetTokenByKey(key, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("token invalid or not found")
		}
		return nil, nil, fmt.Errorf("database error: %w", err)
	}

	if token.DeletedAt.Valid {
		return nil, nil, errors.New("token is deleted")
	}

	if token.Status == common.TokenStatusDisabled {
		return nil, nil, errors.New("token is disabled")
	}

	if token.Status == common.TokenStatusExpired {
		return nil, nil, errors.New("token is expired")
	}
	if token.ExpiredTime != -1 && token.ExpiredTime < common.GetTimestamp() {
		return nil, nil, errors.New("token has expired")
	}

	// IP restrictions policy
	allowIps := token.GetIpLimits()
	if len(allowIps) > 0 {
		clientIp := c.ClientIP()
		ip := net.ParseIP(clientIp)
		if ip == nil || !common.IsIpInCIDRList(ip, allowIps) {
			return nil, nil, errors.New("client IP not permitted by token access policy")
		}
	}

	// User account verification
	user, err := model.GetUserById(token.UserId, false)
	if err != nil {
		return nil, nil, fmt.Errorf("user lookup error: %w", err)
	}
	if user.Status != common.UserStatusEnabled {
		return nil, nil, errors.New("user account is disabled or banned")
	}

	return token, user, nil
}

// GetHuobaoIdentity returns the stable verified user identity from a Tora API key.
// GET /api/huobao/identity
func GetHuobaoIdentity(c *gin.Context) {
	token, user, err := authenticateHuobaoToken(c)
	if err != nil {
		status := http.StatusUnauthorized
		if strings.Contains(err.Error(), "disabled or banned") || strings.Contains(err.Error(), "client IP not permitted") {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Username
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":         token.UserId,
			"token_id":        token.Id,
			"username":        user.Username,
			"display_name":    displayName,
			"account_status":  user.Status,
			"token_status":    token.Status,
			"expires_at":      token.ExpiredTime,
			"quota":           token.RemainQuota,
			"unlimited_quota": token.UnlimitedQuota,
			"user_quota":      user.Quota,
		},
	})
}

// GetHuobaoCredits returns the real spendable credit calculation for Huobao.
// GET /api/huobao/credits and GET /v1/huobao/credits
func GetHuobaoCredits(c *gin.Context) {
	token, user, err := authenticateHuobaoToken(c)
	if err != nil {
		status := http.StatusUnauthorized
		if strings.Contains(err.Error(), "disabled or banned") || strings.Contains(err.Error(), "client IP not permitted") {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	spendableQuota := user.Quota
	if !token.UnlimitedQuota {
		if token.RemainQuota < spendableQuota {
			spendableQuota = token.RemainQuota
		}
	}
	if spendableQuota < 0 {
		spendableQuota = 0
	}

	quotaPerCredit := service.QuotaPerCredit
	if quotaPerCredit <= 0 {
		quotaPerCredit = 1000
	}

	spendableCredits := float64(spendableQuota) / float64(quotaPerCredit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":           token.UserId,
			"token_id":          token.Id,
			"quota_per_credit":  quotaPerCredit,
			"user_quota":        user.Quota,
			"token_quota":       token.RemainQuota,
			"unlimited_quota":   token.UnlimitedQuota,
			"spendable_quota":   spendableQuota,
			"spendable_credits": spendableCredits,
		},
	})
}

// GetVideoQuotaPerSecond returns the configured quota rate per second for a given model and resolution.
func GetVideoQuotaPerSecond(modelName, resolution string) int {
	normRes := strings.ToUpper(strings.TrimSpace(resolution))
	normModel := strings.ToLower(strings.TrimSpace(modelName))

	baseRate := 2000 // default 720P = 2 credits/sec = 2000 quota/sec
	if strings.Contains(normRes, "480") {
		baseRate = 1000
	} else if strings.Contains(normRes, "1080") {
		baseRate = 4000
	} else if strings.Contains(normRes, "720") {
		baseRate = 2000
	}

	if strings.Contains(normModel, "prime") || strings.Contains(normModel, "pro") {
		baseRate = int(float64(baseRate) * 1.5)
	}

	return baseRate
}

type VideoQuoteRequest struct {
	Model           string `json:"model"`
	Resolution      string `json:"resolution"`
	DurationSeconds int    `json:"duration_seconds"`
}

type VideoReserveRequest struct {
	RequestId          string `json:"request_id"`
	Model              string `json:"model"`
	Resolution         string `json:"resolution"`
	MaxDurationSeconds int    `json:"max_duration_seconds"`
}

type VideoSettleRequest struct {
	RequestId             string `json:"request_id"`
	Success               bool   `json:"success"`
	ActualDurationSeconds int    `json:"actual_duration_seconds"`
	ErrorMessage          string `json:"error_message"`
}

// QuoteHuobaoVideoTask returns pre-submission quote for managed video generation.
// POST /api/huobao/video/quote
func QuoteHuobaoVideoTask(c *gin.Context) {
	token, user, err := authenticateHuobaoToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}

	var req VideoQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request body: " + err.Error()})
		return
	}

	duration := req.DurationSeconds
	if duration <= 0 {
		duration = 5
	}
	rate := GetVideoQuotaPerSecond(req.Model, req.Resolution)
	estimatedQuota := duration * rate

	spendableQuota := user.Quota
	if !token.UnlimitedQuota && token.RemainQuota < spendableQuota {
		spendableQuota = token.RemainQuota
	}
	if spendableQuota < 0 {
		spendableQuota = 0
	}

	quotaPerCredit := service.QuotaPerCredit
	if quotaPerCredit <= 0 {
		quotaPerCredit = 1000
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model":             req.Model,
			"resolution":        req.Resolution,
			"duration_seconds":  duration,
			"quota_per_second":  rate,
			"credit_per_second": float64(rate) / float64(quotaPerCredit),
			"estimated_quota":   estimatedQuota,
			"estimated_credits": float64(estimatedQuota) / float64(quotaPerCredit),
			"spendable_quota":   spendableQuota,
			"can_afford":        spendableQuota >= estimatedQuota,
		},
	})
}

// ReserveHuobaoVideoTask atomically pre-consumes wallet quota before dispatching video generation.
// POST /api/huobao/video/reserve
func ReserveHuobaoVideoTask(c *gin.Context) {
	token, user, err := authenticateHuobaoToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}

	var req VideoReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request body: " + err.Error()})
		return
	}

	reqId := strings.TrimSpace(req.RequestId)
	if reqId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "request_id is required"})
		return
	}

	maxDuration := req.MaxDurationSeconds
	if maxDuration <= 0 {
		maxDuration = 10
	}
	rate := GetVideoQuotaPerSecond(req.Model, req.Resolution)
	requiredQuota := maxDuration * rate

	spendableQuota := user.Quota
	if !token.UnlimitedQuota && token.RemainQuota < spendableQuota {
		spendableQuota = token.RemainQuota
	}
	if spendableQuota < requiredQuota {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"success": false,
			"message": "insufficient wallet or token quota to reserve video task",
			"data": gin.H{
				"required_quota":  requiredQuota,
				"spendable_quota": spendableQuota,
			},
		})
		return
	}

	// Atomically reserve quota in durable pre-consume record
	if err := model.PreConsumeUserWallet(reqId, user.Id, requiredQuota); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "message": "failed to reserve quota: " + err.Error()})
		return
	}

	quotaPerCredit := service.QuotaPerCredit
	if quotaPerCredit <= 0 {
		quotaPerCredit = 1000
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"request_id":           reqId,
			"model":                req.Model,
			"resolution":           req.Resolution,
			"max_duration_seconds": maxDuration,
			"quota_per_second":     rate,
			"reserved_quota":       requiredQuota,
			"reserved_credits":     float64(requiredQuota) / float64(quotaPerCredit),
			"status":               "pending",
		},
	})
}

// SettleHuobaoVideoTask settles exact successful output seconds or refunds on failure.
// POST /api/huobao/video/settle
func SettleHuobaoVideoTask(c *gin.Context) {
	_, _, err := authenticateHuobaoToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}

	var req VideoSettleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request body: " + err.Error()})
		return
	}

	reqId := strings.TrimSpace(req.RequestId)
	if reqId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "request_id is required"})
		return
	}

	quotaPerCredit := service.QuotaPerCredit
	if quotaPerCredit <= 0 {
		quotaPerCredit = 1000
	}

	if !req.Success {
		// Task failed: refund entire reservation, 0 video fee charged
		if err := model.RefundUserWalletPreConsume(reqId); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to refund reservation: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"request_id":    reqId,
				"settled_quota": 0,
				"status":        "refunded",
				"reason":        req.ErrorMessage,
			},
		})
		return
	}

	// Task succeeded: calculate actual quota based on actual output seconds
	var existing []model.WalletPreConsumeRecord
	if err := model.DB.Where("request_id = ?", reqId).Limit(1).Find(&existing).Error; err != nil || len(existing) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "reservation record not found"})
		return
	}

	record := existing[0]
	actualDuration := req.ActualDurationSeconds
	if actualDuration <= 0 {
		actualDuration = 5 // fallback standard
	}

	rate := 2000 // default 720P rate
	actualQuota := actualDuration * rate
	if record.PreConsumed > 0 && actualQuota > record.PreConsumed {
		actualQuota = record.PreConsumed // cannot exceed upper bound reservation
	}

	if err := model.SettleUserWalletPreConsumeWithActual(reqId, actualQuota); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to settle reservation: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"request_id":              reqId,
			"actual_duration_seconds": actualDuration,
			"settled_quota":           actualQuota,
			"settled_credits":         float64(actualQuota) / float64(quotaPerCredit),
			"status":                  "settled",
		},
	})
}

