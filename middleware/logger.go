package middleware

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const RouteTagKey = "route_tag"

func RouteTag(tag string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(RouteTagKey, tag)
		c.Next()
	}
}

func isSensitiveQueryParam(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "key", "api_key", "apikey", "api-key",
		"token", "access_token",
		"secret", "client_secret",
		"password", "passwd", "pwd",
		"private_key", "privatekey",
		"signature", "sig",
		"code", "state",
		"credential", "credentials",
		"authorization", "auth":
		return true
	default:
		return false
	}
}

// SanitizeLogPath redacts sensitive query parameters such as API keys, tokens,
// and private credentials from request paths before they are written to access logs.
func SanitizeLogPath(path string) string {
	// OAuth callbacks carry one-time codes and state in the query string.
	// Redact the log value only; the handler still needs the original query.
	if strings.HasPrefix(path, "/api/oauth/") || strings.HasPrefix(path, "/oauth/") {
		path, _, _ = strings.Cut(path, "?")
		return path
	}

	basePath, rawQuery, ok := strings.Cut(path, "?")
	if !ok || rawQuery == "" {
		return path
	}

	parts := strings.Split(rawQuery, "&")
	sanitizedParts := make([]string, len(parts))
	for i, part := range parts {
		rawKey, _, _ := strings.Cut(part, "=")
		unescapedKey, err := url.QueryUnescape(rawKey)
		if err != nil {
			unescapedKey = rawKey
		}
		if isSensitiveQueryParam(unescapedKey) {
			sanitizedParts[i] = rawKey + "=[REDACTED]"
		} else {
			sanitizedParts[i] = part
		}
	}

	return basePath + "?" + strings.Join(sanitizedParts, "&")
}

func SetUpLogger(server *gin.Engine) {
	server.Use(redactTaskArtifactAccessQuery())
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var requestID string
		if param.Keys != nil {
			requestID, _ = param.Keys[common.RequestIdKey].(string)
		}
		tag, _ := param.Keys[RouteTagKey].(string)
		if tag == "" {
			tag = "web"
		}
		path := SanitizeLogPath(param.Path)
		return fmt.Sprintf("[GIN] %s | %s | %s | %3d | %13v | %15s | %7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			tag,
			requestID,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			path,
		)
	}))
}
