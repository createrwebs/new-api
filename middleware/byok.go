package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

var (
	byokRateLimitNum      = common.GetEnvOrDefault("BYOK_RATE_LIMIT_NUM", 60)
	byokRateLimitDuration = int64(common.GetEnvOrDefault("BYOK_RATE_LIMIT_DURATION", 60))
	byokMaxConcurrency    = common.GetEnvOrDefault("BYOK_MAX_CONCURRENCY", 5)

	byokConcurrencyTracker = struct {
		mu     sync.Mutex
		active map[int]int
	}{
		active: make(map[int]int),
	}
)

func SetBYOKRateLimitsForTest(num int, duration int64, concurrency int) func() {
	prevNum, prevDur, prevConc := byokRateLimitNum, byokRateLimitDuration, byokMaxConcurrency
	byokRateLimitNum, byokRateLimitDuration, byokMaxConcurrency = num, duration, concurrency
	return func() {
		byokRateLimitNum, byokRateLimitDuration, byokMaxConcurrency = prevNum, prevDur, prevConc
	}
}

func checkBYOKRateLimit(c *gin.Context, userID int) bool {
	if byokRateLimitNum <= 0 || byokRateLimitDuration <= 0 {
		return true
	}
	if common.RedisEnabled && common.RDB != nil {
		allowed, _, ttlSeconds, err := redisFixedWindowTake(
			c.Request.Context(),
			redisUserRateLimitKey("BYOK", userID),
			byokRateLimitNum,
			byokRateLimitDuration,
		)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("BYOK rate limit redis check failed for user %d: %v", userID, err))
			return true
		}
		if !allowed {
			writeBYOKRateLimited(c, ttlSeconds)
			return false
		}
		return true
	}

	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	key := fmt.Sprintf("byok:user:%d", userID)
	if !inMemoryRateLimiter.Request(key, byokRateLimitNum, byokRateLimitDuration) {
		writeBYOKRateLimited(c, byokRateLimitDuration)
		return false
	}
	return true
}

func writeBYOKRateLimited(c *gin.Context, retryAfterSeconds int64) {
	if retryAfterSeconds > 0 {
		c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
	}
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{
			"message": "BYOK request rate limit exceeded. Please wait before sending more requests.",
			"type":    "requests",
			"code":    "byok_rate_limit_exceeded",
		},
	})
}

func acquireBYOKConcurrency(c *gin.Context, userID int) bool {
	if byokMaxConcurrency <= 0 {
		return true
	}
	byokConcurrencyTracker.mu.Lock()
	defer byokConcurrencyTracker.mu.Unlock()

	current := byokConcurrencyTracker.active[userID]
	if current >= byokMaxConcurrency {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("Too many concurrent BYOK requests. Maximum allowed is %d.", byokMaxConcurrency),
				"type":    "requests",
				"code":    "byok_concurrency_limit_exceeded",
			},
		})
		return false
	}
	byokConcurrencyTracker.active[userID] = current + 1
	return true
}

func releaseBYOKConcurrency(userID int) {
	if byokMaxConcurrency <= 0 {
		return
	}
	byokConcurrencyTracker.mu.Lock()
	defer byokConcurrencyTracker.mu.Unlock()

	current := byokConcurrencyTracker.active[userID]
	if current <= 1 {
		delete(byokConcurrencyTracker.active, userID)
	} else {
		byokConcurrencyTracker.active[userID] = current - 1
	}
}

type byokRequestInfo struct {
	isBYOK     bool
	provider   string
	hasPrefix  bool
	cleanModel string
	rawModel   string
}

// detectBYOKRequest detects if the incoming request specifies a BYOK provider
// via HTTP header, query parameter, JSON body provider property, or model prefix ("openrouter/" or "custom/").
func detectBYOKRequest(c *gin.Context) byokRequestInfo {
	info := byokRequestInfo{}
	if c.Request == nil || c.Request.Method != http.MethodPost {
		return info
	}

	// 1. Check HTTP headers
	provider := strings.TrimSpace(c.GetHeader("X-Provider"))
	if provider == "" {
		provider = strings.TrimSpace(c.GetHeader("X-BYOK-Provider"))
	}
	if provider == "" {
		provider = strings.TrimSpace(c.GetHeader("X-BYOK"))
	}

	// 2. Check query parameter
	if provider == "" {
		provider = strings.TrimSpace(c.Query("provider"))
	}

	// 3. Inspect JSON body if available
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/json") {
		storage, err := common.GetBodyStorage(c)
		if err == nil && storage != nil {
			bodyBytes, bErr := storage.Bytes()
			if bErr == nil && len(bodyBytes) > 0 && gjson.ValidBytes(bodyBytes) {
				if provider == "" {
					if pVal := gjson.GetBytes(bodyBytes, "provider"); pVal.Exists() && pVal.Type == gjson.String {
						provider = strings.TrimSpace(pVal.String())
					}
				}
				if mVal := gjson.GetBytes(bodyBytes, "model"); mVal.Exists() && mVal.Type == gjson.String {
					rawModel := strings.TrimSpace(mVal.String())
					info.rawModel = rawModel
					lowerModel := strings.ToLower(rawModel)
					if strings.HasPrefix(lowerModel, "openrouter/") {
						provider = model.ProviderOpenRouter
						info.hasPrefix = true
						info.cleanModel = rawModel[len("openrouter/"):]
					} else if strings.HasPrefix(lowerModel, "custom/") {
						provider = model.ProviderCustom
						info.hasPrefix = true
						info.cleanModel = rawModel[len("custom/"):]
					} else {
						info.cleanModel = rawModel
					}
				}
			}
		}
	}

	if provider != "" {
		info.isBYOK = true
		info.provider = strings.ToLower(provider)
	}

	return info
}

// BYOKRouter inspects requests to determine if they target user BYOK credentials,
// decrypts the personal API key server-side, binds the virtual channel metadata,
// and skips system channel distribution and free tier quota limits.
func BYOKRouter() gin.HandlerFunc {
	return func(c *gin.Context) {
		info := detectBYOKRequest(c)
		if !info.isBYOK {
			c.Next()
			return
		}

		if info.provider != model.ProviderOpenRouter && info.provider != model.ProviderCustom {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": fmt.Sprintf("invalid BYOK provider: '%s', supported providers are 'openrouter' and 'custom'", info.provider),
					"type":    "invalid_request_error",
					"code":    "byok_invalid_provider",
				},
			})
			return
		}

		userID := c.GetInt("id")
		if userID == 0 {
			userID = common.GetContextKeyInt(c, constant.ContextKeyUserId)
		}
		if userID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "authentication required for BYOK relay",
					"type":    "invalid_request_error",
					"code":    "byok_unauthorized",
				},
			})
			return
		}

		userProvider, err := model.GetUserProviderByType(userID, info.provider)
		if err != nil || userProvider == nil || !userProvider.Enabled {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": fmt.Sprintf("BYOK provider '%s' is not configured or is disabled for this user", info.provider),
					"type":    "invalid_request_error",
					"code":    "byok_provider_not_found",
				},
			})
			return
		}

		// Per-user BYOK abuse protection (rate limit & concurrency limit)
		if !checkBYOKRateLimit(c, userID) {
			return
		}
		if !acquireBYOKConcurrency(c, userID) {
			return
		}
		defer releaseBYOKConcurrency(userID)

		apiKey, err := userProvider.DecryptAPIKey()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "failed to decrypt BYOK provider credentials",
					"type":    "server_error",
					"code":    "byok_decrypt_failed",
				},
			})
			return
		}

		var channelType int
		var baseURL string
		if info.provider == model.ProviderOpenRouter {
			channelType = constant.ChannelTypeOpenRouter
			baseURL = strings.TrimRight(userProvider.BaseURL, "/")
			if baseURL == "" {
				baseURL = "https://openrouter.ai/api"
			} else {
				if strings.HasSuffix(baseURL, "/v1") {
					baseURL = strings.TrimSuffix(baseURL, "/v1")
				}
				if !strings.HasSuffix(baseURL, "/api") {
					baseURL = baseURL + "/api"
				}
			}
		} else {
			channelType = constant.ChannelTypeOpenAI
			baseURL = strings.TrimRight(userProvider.BaseURL, "/")
			if strings.HasSuffix(baseURL, "/v1") {
				baseURL = strings.TrimSuffix(baseURL, "/v1")
			}
		}

		cleanModel := info.cleanModel
		if cleanModel == "" {
			cleanModel = info.rawModel
		}

		if info.hasPrefix && cleanModel != "" {
			if rewriteErr := rewriteTaskPluginJSONModel(c, cleanModel); rewriteErr != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": gin.H{
						"message": "failed to normalize model name",
						"type":    "invalid_request_error",
						"code":    "invalid_request",
					},
				})
				return
			}
		}

		autoBanInt := 0
		c.Set("is_byok", true)
		c.Set("byok_provider", info.provider)
		c.Set("channel_id", -1)
		c.Set("channel_type", channelType)
		c.Set("channel_name", "BYOK-"+userProvider.Name)
		c.Set("auto_ban", false)
		if cleanModel != "" {
			c.Set("original_model", cleanModel)
		}

		common.SetContextKey(c, constant.ContextKeyIsBYOK, true)
		common.SetContextKey(c, constant.ContextKeyBYOKProvider, info.provider)
		common.SetContextKey(c, constant.ContextKeyChannelId, -1)
		common.SetContextKey(c, constant.ContextKeyChannelName, "BYOK-"+userProvider.Name)
		common.SetContextKey(c, constant.ContextKeyChannelType, channelType)
		common.SetContextKey(c, constant.ContextKeyChannelKey, apiKey)
		common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, baseURL)
		if cleanModel != "" {
			common.SetContextKey(c, constant.ContextKeyOriginalModel, cleanModel)
		}
		common.SetContextKey(c, constant.ContextKeyChannelAutoBan, &autoBanInt)
		common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())

		c.Next()
	}
}
