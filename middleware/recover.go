package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func RelayPanicRecover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := c.GetString(common.RequestIdKey)
				common.SysError(fmt.Sprintf("panic detected [request_id=%s]: %v\nstack:\n%s", requestID, err, string(debug.Stack())))
				msg := "An unexpected internal server error occurred. Please contact administrator."
				if requestID != "" {
					msg = fmt.Sprintf("An unexpected internal server error occurred. Please contact administrator with Request ID: %s", requestID)
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"message": msg,
						"type":    "internal_server_error",
						"code":    "internal_error",
					},
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
