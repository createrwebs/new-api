package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUserProviderRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	expectedRoutes := []struct {
		method      string
		routePath   string
		requestPath string
	}{
		{http.MethodGet, "/api/user/providers", "/api/user/providers"},
		{http.MethodPost, "/api/user/providers", "/api/user/providers"},
		{http.MethodPut, "/api/user/providers/:id", "/api/user/providers/1"},
		{http.MethodDelete, "/api/user/providers/:id", "/api/user/providers/1"},
		{http.MethodPost, "/api/user/providers/:id/test", "/api/user/providers/1/test"},
	}

	registeredRoutes := make(map[string]bool)
	for _, r := range engine.Routes() {
		registeredRoutes[fmt.Sprintf("%s %s", r.Method, r.Path)] = true
	}

	for _, tc := range expectedRoutes {
		routeKey := fmt.Sprintf("%s %s", tc.method, tc.routePath)
		assert.True(t, registeredRoutes[routeKey], "Route must be registered: %s", routeKey)

		// Unauthenticated request must receive 401 Unauthorized
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.requestPath, nil)
		engine.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusUnauthorized, recorder.Code, "Unauthenticated request to %s must return 401", tc.requestPath)
	}
}
