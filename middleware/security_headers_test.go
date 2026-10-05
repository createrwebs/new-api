package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "SAMEORIGIN", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
}

func TestVersionHeaderSuppression(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Default suppressed", func(t *testing.T) {
		orig := os.Getenv("EXPOSE_VERSION_HEADER")
		origDebug := common.DebugEnabled
		common.DebugEnabled = false
		os.Unsetenv("EXPOSE_VERSION_HEADER")
		defer func() {
			if orig != "" {
				os.Setenv("EXPOSE_VERSION_HEADER", orig)
			}
			common.DebugEnabled = origDebug
		}()

		router := gin.New()
		router.Use(Version())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("X-New-Api-Version"))
	})

	t.Run("Explicitly enabled", func(t *testing.T) {
		t.Setenv("EXPOSE_VERSION_HEADER", "true")

		router := gin.New()
		router.Use(Version())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, common.Version, rec.Header().Get("X-New-Api-Version"))
	})
}

func TestCORSNoAmbientCredentialsByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orig := os.Getenv("CORS_ALLOWED_ORIGINS")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")
	defer func() {
		if orig != "" {
			os.Setenv("CORS_ALLOWED_ORIGINS", orig)
		}
	}()

	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Request with an untrusted Origin
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Must NOT reflect evil.com with Allow-Credentials: true
	assert.NotEqual(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSWhitelistedOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://trusted-dashboard.com,https://api.trusted.com")

	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Trusted origin receives credentials
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://trusted-dashboard.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://trusted-dashboard.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))

	// Untrusted origin is rejected
	reqEvil := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqEvil.Header.Set("Origin", "https://evil.com")
	recEvil := httptest.NewRecorder()
	router.ServeHTTP(recEvil, reqEvil)

	assert.Empty(t, recEvil.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEqual(t, "true", recEvil.Header().Get("Access-Control-Allow-Credentials"))
}
