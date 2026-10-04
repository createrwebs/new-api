package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// isFreeTierGenerationRequest limits Phase 4 free tier quota checks to the
// supported mobile generation entry points: POST /v1/chat/completions and
// POST /v1/responses.
func isFreeTierGenerationRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	path := strings.TrimSuffix(c.Request.URL.Path, "/")
	switch path {
	case "/v1/chat/completions", "/v1/responses":
		return true
	default:
		return false
	}
}

// FreeTierQuota enforces the account-level free request limit after API-token
// authentication and before channel distribution/upstream relay.
func FreeTierQuota() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !isFreeTierGenerationRequest(c) {
			c.Next()
			return
		}
		// BYOK requests use personal provider credentials and bypass server free tier limits.
		if c.GetBool("is_byok") || common.GetContextKeyBool(c, constant.ContextKeyIsBYOK) {
			c.Next()
			return
		}
		userID := c.GetInt("id")
		free, err := model.IsFreeTierUser(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "free tier quota unavailable",
					"type":    "server_error",
					"code":    "free_tier_unavailable",
				},
			})
			return
		}
		if !free {
			c.Next()
			return
		}

		allowed, err := model.ReserveFreeTierRequest(userID, time.Now())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "free tier quota unavailable",
					"type":    "server_error",
					"code":    "free_tier_unavailable",
				},
			})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "free tier daily request limit exceeded",
					"type":    "rate_limit_error",
					"code":    "FREE_DAILY_LIMIT",
				},
			})
			return
		}
		c.Next()
	}
}
