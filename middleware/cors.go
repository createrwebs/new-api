package middleware

import (
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS configures Cross-Origin Resource Sharing.
// To prevent cross-origin ambient credential abuse (Finding 6E-05):
// - By default, AllowAllOrigins is true but AllowCredentials is false (returning Access-Control-Allow-Origin: *).
// - When CORS_ALLOWED_ORIGINS is configured (comma-separated), AllowOrigins is set to the whitelist and AllowCredentials is enabled.
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}

	allowedOriginsEnv := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if allowedOriginsEnv != "" {
		rawOrigins := strings.Split(allowedOriginsEnv, ",")
		var origins []string
		for _, o := range rawOrigins {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
		if len(origins) > 0 {
			config.AllowAllOrigins = false
			config.AllowOrigins = origins
			config.AllowCredentials = true
			return cors.New(config)
		}
	}

	config.AllowAllOrigins = true
	config.AllowCredentials = false
	return cors.New(config)
}

// Version optionally injects the X-New-Api-Version header if explicitly enabled.
// In production, version header is suppressed by default to prevent software fingerprinting (Finding 6E-06).
func Version() gin.HandlerFunc {
	expose := strings.EqualFold(os.Getenv("EXPOSE_VERSION_HEADER"), "true") ||
		os.Getenv("EXPOSE_VERSION_HEADER") == "1" ||
		common.DebugEnabled
	return func(c *gin.Context) {
		if expose {
			c.Header("X-New-Api-Version", common.Version)
		}
		c.Next()
	}
}

// SecurityHeaders adds standard defensive HTTP response headers (Finding 6E-06):
// - X-Content-Type-Options: nosniff (prevents MIME type confusion attacks)
// - X-Frame-Options: SAMEORIGIN (prevents clickjacking attacks)
// - Referrer-Policy: strict-origin-when-cross-origin (prevents URL leakage in referrers)
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
