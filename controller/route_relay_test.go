package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteRelay_ClassifierDecisions(t *testing.T) {
	ctx := context.Background()

	// 1. Primary 429 -> Fallback eligible
	err429 := types.NewErrorWithStatusCode(errors.New("rate limit reached"), "rate_limit", 429)
	decision := service.ClassifyRouteError(ctx, err429, false)
	assert.True(t, decision.AllowFallback)
	assert.Equal(t, service.ClassFallbackEligible, decision.Class)

	// 2. Primary 503 -> Fallback eligible
	err503 := types.NewErrorWithStatusCode(errors.New("upstream unavailable"), "service_unavailable", 503)
	decision = service.ClassifyRouteError(ctx, err503, false)
	assert.True(t, decision.AllowFallback)
	assert.Equal(t, service.ClassFallbackEligible, decision.Class)

	// 3. Client 400 Bad Request -> Terminal Client (No fallback)
	err400 := types.NewErrorWithStatusCode(errors.New("malformed body"), "bad_request", 400)
	decision = service.ClassifyRouteError(ctx, err400, false)
	assert.False(t, decision.AllowFallback)
	assert.Equal(t, service.ClassTerminalClient, decision.Class)

	// 4. Client 404 Model Not Found -> Terminal Client (No fallback)
	err404 := types.NewErrorWithStatusCode(errors.New("model not found"), "model_not_found", 404)
	decision = service.ClassifyRouteError(ctx, err404, false)
	assert.False(t, decision.AllowFallback)
	assert.Equal(t, service.ClassTerminalClient, decision.Class)

	// 5. Streaming committed: Even 429 or 503 MUST NOT fallback once stream committed
	decision = service.ClassifyRouteError(ctx, err429, true)
	assert.False(t, decision.AllowFallback, "fallback strictly forbidden once response committed downstream")
	assert.Equal(t, service.ClassTerminalCommitted, decision.Class)

	decision = service.ClassifyRouteError(ctx, err503, true)
	assert.False(t, decision.AllowFallback, "fallback strictly forbidden once response committed downstream")
	assert.Equal(t, service.ClassTerminalCommitted, decision.Class)

	// 6. Client cancellation -> Terminal (No fallback)
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	decision = service.ClassifyRouteError(cancelCtx, err503, false)
	assert.False(t, decision.AllowFallback, "client cancellation must terminate without fallback")
	assert.Equal(t, service.ClassClientCancelled, decision.Class)
}

func TestRouteRelay_TokenRouteChainResolution(t *testing.T) {
	// Verify that a token with Primary and Fallback routes resolves deterministically
	token := &model.Token{
		Id:             100,
		Name:           "Routing Key",
		PrimaryRouteId: 1,
	}
	err := token.SetFallbackRouteIds([]int{2, 3})
	require.NoError(t, err)

	chain := token.GetRouteChain()
	assert.Equal(t, []int{1, 2, 3}, chain)

	// Legacy token without primary route
	legacyToken := &model.Token{
		Id:             101,
		Name:           "Legacy Key",
		PrimaryRouteId: 0,
		Group:          "default",
	}
	assert.False(t, legacyToken.IsRouteEnabled())
	assert.Empty(t, legacyToken.GetRouteChain())
}

func TestRouteRelay_AdminAPIRouteCRUD(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Route{}))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("role", 100) // Admin
		c.Set("id", 1)
		c.Set("group", "default")
		c.Next()
	})
	r.GET("/api/routes", GetAllRoutes)
	r.GET("/api/routes/:id", GetRoute)
	r.POST("/api/routes", CreateRoute)
	r.PUT("/api/routes", UpdateRoute)
	r.DELETE("/api/routes/:id", DeleteRoute)
	r.GET("/api/routes/available", GetAvailableRoutes)

	// 1. Create Route
	newRoute := model.Route{
		Name:           "Primary Managed Route",
		Slug:           "primary-managed",
		Kind:           model.RouteKindLLM,
		RoutingPolicy:  model.RoutePolicyPriority,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	body, _ := json.Marshal(newRoute)
	createReq := httptest.NewRequest(http.MethodPost, "/api/routes", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, createReq)
	assert.Equal(t, http.StatusOK, wCreate.Code)

	// 2. GET /api/routes
	req := httptest.NewRequest(http.MethodGet, "/api/routes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. GET /api/routes/available
	reqAvail := httptest.NewRequest(http.MethodGet, "/api/routes/available", nil)
	wAvail := httptest.NewRecorder()
	r.ServeHTTP(wAvail, reqAvail)
	assert.Equal(t, http.StatusOK, wAvail.Code)
}
