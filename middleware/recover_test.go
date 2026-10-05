package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRelayPanicRecoverSanitizesResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const panicSecretCanary = "PANIC_SECRET_CANARY_database_connection_password_12345"

	router := gin.New()
	router.Use(RequestId())
	router.Use(RelayPanicRecover())
	router.GET("/panic", func(c *gin.Context) {
		panic(panicSecretCanary)
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		router.ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// The client message must NOT contain the sensitive panic object
	assert.NotContains(t, resp.Error.Message, panicSecretCanary)
	assert.Contains(t, resp.Error.Message, "An unexpected internal server error occurred")
	assert.Contains(t, resp.Error.Message, "Request ID:")
	assert.Equal(t, "internal_server_error", resp.Error.Type)
}
