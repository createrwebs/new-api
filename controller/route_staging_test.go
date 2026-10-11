package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Section 4: Denial-of-Wallet & Adversarial Upstream Validation
func TestRouteStaging_DenialOfWallet_AdversarialScenarios(t *testing.T) {
	ctx := context.Background()

	// Case A: Connect failure before request body transmission -> Safe pre-execution failure -> Fallback ALLOWED
	connErr := types.NewError(fmt.Errorf("dial tcp 127.0.0.1:9999: connect: connection refused"), types.ErrorCodeDoRequestFailed)
	decision := service.ClassifyRouteError(ctx, connErr, false)
	assert.True(t, decision.AllowFallback, "Case A: Safe pre-execution error must allow fallback")
	assert.Equal(t, service.ClassSafePreExecutionFailure, decision.Class)

	// Case B: Read full request body, then close connection without response -> Ambiguous execution -> Fallback FORBIDDEN
	readResetErr := types.NewError(fmt.Errorf("read: connection reset by peer"), types.ErrorCodeDoRequestFailed)
	decision = service.ClassifyRouteError(ctx, readResetErr, false)
	assert.False(t, decision.AllowFallback, "Case B: Ambiguous connection reset must NOT fallback (Denial-of-Wallet guard)")
	assert.Equal(t, service.ClassAmbiguousExecutionFailure, decision.Class)

	// Case C: Read request, wait, then timeout awaiting headers -> Ambiguous execution -> Fallback FORBIDDEN
	timeoutErr := types.NewError(fmt.Errorf("net/http: Client.Timeout exceeded while awaiting headers"), types.ErrorCodeDoRequestFailed)
	decision = service.ClassifyRouteError(ctx, timeoutErr, false)
	assert.False(t, decision.AllowFallback, "Case C: Ambiguous read timeout must NOT fallback (Denial-of-Wallet guard)")
	assert.Equal(t, service.ClassAmbiguousExecutionFailure, decision.Class)

	// Case D: Return explicit 429 Rate Limit -> Fallback ALLOWED
	err429 := types.NewErrorWithStatusCode(fmt.Errorf("rate limit reached"), "rate_limit", 429)
	decision = service.ClassifyRouteError(ctx, err429, false)
	assert.True(t, decision.AllowFallback, "Case D: Explicit 429 must allow fallback")
	assert.Equal(t, service.ClassFallbackEligible, decision.Class)

	// Case E: Return explicit 503 Server Error -> Fallback ALLOWED
	err503 := types.NewErrorWithStatusCode(fmt.Errorf("upstream overloaded"), "service_unavailable", 503)
	decision = service.ClassifyRouteError(ctx, err503, false)
	assert.True(t, decision.AllowFallback, "Case E: Explicit 503 must allow fallback")
	assert.Equal(t, service.ClassFallbackEligible, decision.Class)

	// Case F: Begin response, emit tokens, then drop connection -> Stream committed -> Fallback FORBIDDEN
	decision = service.ClassifyRouteError(ctx, err503, true)
	assert.False(t, decision.AllowFallback, "Case F: Stream committed error must NEVER fallback")
	assert.Equal(t, service.ClassTerminalCommitted, decision.Class)
}

// Section 5: Real Stream Commitment Stack (CommitDetectingWriter)
func TestRouteStaging_RealStreamCommitment_FullStack(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Invariant: No client can receive bytes from Provider A followed by a fresh Provider B stream
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	writer := relay.NewCommitDetectingWriter(c.Writer)

	// 1. Headers set, no data flushed yet
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)
	assert.False(t, writer.IsCommitted(), "headers alone do not commit stream")
	assert.Empty(t, rec.Body.String())

	// 2. Upstream provider A fails before any event: Reset succeeds
	require.NoError(t, writer.Reset(), "reset succeeds before any data flushed")
	assert.False(t, writer.IsCommitted())

	// 3. Provider B emits first SSE event and flushes
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.WriteHeader(http.StatusOK)
	_, err := writer.WriteString("data: {\"choices\":[{\"delta\":{\"content\":\"Hello from B\"}}]}\n\n")
	require.NoError(t, err)
	writer.Flush()

	// Now committed!
	assert.True(t, writer.IsCommitted(), "flush with data commits stream")
	assert.Contains(t, rec.Body.String(), "Hello from B")

	// 4. Provider B drops mid-stream: Attempting reset to try Provider C MUST FAIL
	err = writer.Reset()
	assert.ErrorIs(t, err, relay.ErrAlreadyCommitted, "cannot failover after stream has committed to client")
}

// Section 7 & 17: Model Mapping Security & Request-Time Entitlement
func TestRouteStaging_ModelMappingSecurityAndEntitlement(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Route{}))

	// Setup: Route with model mapping gpt-4 -> gpt-4o and MinUserGroup: "pro"
	proRoute := &model.Route{
		Id:             201,
		Name:           "Pro Route",
		Slug:           "pro-route",
		Kind:           model.RouteKindLLM,
		MinUserGroup:   "pro",
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, proRoute.SetModelMapping(map[string]string{"gpt-4": "gpt-4o"}))
	require.NoError(t, proRoute.SetChannelIds([]int{1}))
	require.NoError(t, db.Create(proRoute).Error)
	service.CleanRouteCache(proRoute.Id)

	// 1. Free/default user attempts to access Pro route -> Rejected at execution time
	err := service.ValidateRouteEntitlement(proRoute, "default")
	assert.Error(t, err, "free user must be denied access to pro-restricted route")

	// 2. Pro user accesses Pro route -> Permitted
	err = service.ValidateRouteEntitlement(proRoute, "pro")
	assert.NoError(t, err, "pro user must be permitted on pro route")

	// 3. Model mapping resolution
	assert.Equal(t, "gpt-4o", service.ResolveModelForRoute(proRoute, "gpt-4"))
	assert.Equal(t, "unmapped-model", service.ResolveModelForRoute(proRoute, "unmapped-model"))

	// 4. Admin/root bypass
	assert.NoError(t, service.ValidateRouteEntitlement(proRoute, "admin"))
	assert.NoError(t, service.ValidateRouteEntitlement(proRoute, "root"))
}

// Section 8 & 9: Billing Exactness, Reservation Eligibility, and Refunds
func TestRouteStaging_BillingExactnessAndReservationEligibility(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Route{}, &model.Channel{}))

	// Channel 1: default group, supports gpt-4o
	ch1 := &model.Channel{
		Id:       10,
		Name:     "gpt-4o-default",
		Type:     1,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "gpt-4o",
		Priority: common.GetPointer(int64(100)),
		Weight:   common.GetPointer(uint(50)),
	}
	require.NoError(t, db.Create(ch1).Error)
	model.CacheUpdateChannel(ch1)

	// Route 1: Multiplier 1.0, LLM kind, default group, supports gpt-4o
	r1 := &model.Route{
		Id:             301,
		Name:           "Primary 1.0",
		Slug:           "primary-1",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, r1.SetChannelIds([]int{ch1.Id}))
	require.NoError(t, db.Create(r1).Error)

	// Route 2: Multiplier 0.8, LLM kind, default group, supports gpt-4o
	r2 := &model.Route{
		Id:             302,
		Name:           "Fallback 0.8",
		Slug:           "fallback-08",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 0.8,
		Enabled:        true,
	}
	require.NoError(t, r2.SetChannelIds([]int{ch1.Id}))
	require.NoError(t, db.Create(r2).Error)

	// Route 3: Multiplier 2.5, BUT requires "pro" entitlement (ineligible for default user)
	r3 := &model.Route{
		Id:             303,
		Name:           "Pro High Cost 2.5",
		Slug:           "pro-25",
		Kind:           model.RouteKindLLM,
		MinUserGroup:   "pro",
		CostMultiplier: 2.5,
		Enabled:        true,
	}
	require.NoError(t, r3.SetChannelIds([]int{ch1.Id}))
	require.NoError(t, db.Create(r3).Error)

	// Route 4: Multiplier 3.0, BUT only supports image models (ineligible for gpt-4o)
	chImage := &model.Channel{
		Id:       11,
		Name:     "dall-e-3",
		Type:     1,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "dall-e-3",
		Priority: common.GetPointer(int64(100)),
		Weight:   common.GetPointer(uint(50)),
	}
	require.NoError(t, db.Create(chImage).Error)
	model.CacheUpdateChannel(chImage)

	r4 := &model.Route{
		Id:             304,
		Name:           "Image Only 3.0",
		Slug:           "image-30",
		Kind:           model.RouteKindMedia,
		CostMultiplier: 3.0,
		Enabled:        true,
	}
	require.NoError(t, r4.SetChannelIds([]int{chImage.Id}))
	require.NoError(t, db.Create(r4).Error)

	service.CleanRouteCache(r1.Id)
	service.CleanRouteCache(r2.Id)
	service.CleanRouteCache(r3.Id)
	service.CleanRouteCache(r4.Id)

	// Verify Reservation Eligibility:
	// For default user requesting gpt-4o with chain [301, 302, 303, 304]:
	// 301 is eligible (mult 1.0)
	// 302 is eligible (mult 0.8)
	// 303 is INELIGIBLE (requires pro)
	// 304 is INELIGIBLE (model mismatch)
	// Therefore, maxMultiplier MUST BE 1.0, NOT 2.5 or 3.0!
	chain := []int{301, 302, 303, 304}
	userGroup := "default"
	modelName := "gpt-4o"

	maxMultiplier := 1.0
	var eligibleSnapshots []*service.RouteSnapshot
	for _, rId := range chain {
		r, err := service.CacheGetRoute(rId)
		require.NoError(t, err)
		if !r.Enabled || service.ValidateRouteEntitlement(r, userGroup) != nil {
			continue
		}
		if !service.RouteSupportsModel(r, modelName, userGroup) {
			continue
		}
		eligibleSnapshots = append(eligibleSnapshots, service.SnapshotRoute(r))
		if r.CostMultiplier > maxMultiplier {
			maxMultiplier = r.CostMultiplier
		}
	}

	assert.Equal(t, 2, len(eligibleSnapshots), "only Route 301 and 302 must be eligible")
	assert.Equal(t, 1.0, maxMultiplier, "reservation multiplier must NOT be inflated by ineligible routes")

	// Verify Exact Settlement:
	// If winning route is 302 (multiplier 0.8):
	baseQuota := 1000
	modelRatio := 1.0
	userGroupRatio := 1.0
	settledQuota := service.CalculateSettledQuota(baseQuota, modelRatio, userGroupRatio, eligibleSnapshots[1].CostMultiplier)
	assert.Equal(t, 800, settledQuota, "settlement must reflect winning route multiplier (1000 * 0.8 = 800)")

	// Unused reservation refund
	reservedQuota := int(float64(baseQuota) * maxMultiplier) // 1000
	refund := reservedQuota - settledQuota
	assert.Equal(t, 200, refund, "excess quota reserved must be refunded cleanly")
}

// Section 10: Route Snapshot Consistency (In-flight immutability)
func TestRouteStaging_RouteSnapshotConsistency(t *testing.T) {
	route := &model.Route{
		Id:             401,
		Name:           "Immutable Initial Route",
		Slug:           "snap-route",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, route.SetChannelIds([]int{1, 2}))
	require.NoError(t, route.SetModelMapping(map[string]string{"gpt-4": "gpt-4-0613"}))

	// Take snapshot at request start
	snap := service.SnapshotRoute(route)
	require.NotNil(t, snap)
	assert.Equal(t, 1.0, snap.CostMultiplier)
	assert.Equal(t, []int{1, 2}, snap.ChannelIds)

	// Simulate concurrent modification during request execution:
	// Admin changes multiplier to 2.5 and disables route
	route.CostMultiplier = 2.5
	route.Enabled = false
	_ = route.SetChannelIds([]int{99})

	// The in-flight snapshot MUST remain unchanged
	assert.Equal(t, 1.0, snap.CostMultiplier, "in-flight request must retain initial multiplier")
	assert.True(t, snap.Enabled, "in-flight request must retain initial enabled state")
	assert.Equal(t, []int{1, 2}, snap.ChannelIds, "in-flight request must retain initial channel pool")

	// Convert snapshot back to route for execution
	execRoute := snap.ToRoute()
	assert.Equal(t, 1.0, execRoute.CostMultiplier)
	assert.True(t, execRoute.Enabled)
	chIds, err := execRoute.GetChannelIds()
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, chIds)
}

// Section 12 & 13: Distributed Cooldown & Priority/Weight Tiers
func TestRouteStaging_DistributedCooldownAndPriorityTiers(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Route{}))

	// Channel A: Priority 100, Weight 80
	chA := &model.Channel{
		Id:       501,
		Name:     "Channel-A",
		Type:     1,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "gpt-4o",
		Priority: common.GetPointer(int64(100)),
		Weight:   common.GetPointer(uint(80)),
	}
	require.NoError(t, db.Create(chA).Error)
	model.CacheUpdateChannel(chA)

	// Channel B: Priority 100, Weight 20
	chB := &model.Channel{
		Id:       502,
		Name:     "Channel-B",
		Type:     1,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "gpt-4o",
		Priority: common.GetPointer(int64(100)),
		Weight:   common.GetPointer(uint(20)),
	}
	require.NoError(t, db.Create(chB).Error)
	model.CacheUpdateChannel(chB)

	// Channel C: Priority 50
	chC := &model.Channel{
		Id:       503,
		Name:     "Channel-C",
		Type:     1,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "gpt-4o",
		Priority: common.GetPointer(int64(50)),
		Weight:   common.GetPointer(uint(100)),
	}
	require.NoError(t, db.Create(chC).Error)
	model.CacheUpdateChannel(chC)

	route := &model.Route{
		Id:             550,
		Name:           "Tiered Route",
		Slug:           "tiered-route",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, route.SetChannelIds([]int{chA.Id, chB.Id, chC.Id}))
	require.NoError(t, db.Create(route).Error)
	service.CleanRouteCache(route.Id)

	// 1. Initial selection: Must pick from priority 100 (A or B), NEVER C
	attempted := make(map[int]struct{})
	selected, err := service.SelectNextChannelInRoute(route, "gpt-4o", "default", attempted)
	require.NoError(t, err)
	assert.Contains(t, []int{chA.Id, chB.Id}, selected.Id, "initial selection must be from top priority tier (100)")
	assert.NotEqual(t, chC.Id, selected.Id, "priority 50 channel C must not be selected while priority 100 is available")

	// 2. Simulate Channel A 429 rate limit -> enters 15s cooldown
	service.SetChannelCooldown(chA.Id, 15*time.Second)
	assert.True(t, service.IsChannelInCooldown(chA.Id), "channel A must be in cooldown")

	// Next selection: A is in cooldown, so B (priority 100) must be selected
	selected, err = service.SelectNextChannelInRoute(route, "gpt-4o", "default", attempted)
	require.NoError(t, err)
	assert.Equal(t, chB.Id, selected.Id, "channel B must be selected from priority 100")

	// 3. Mark B as attempted/failed
	attempted[chB.Id] = struct{}{}

	// Now priority 100 has NO available channels (A in cooldown, B attempted).
	// Priority tier 50 (Channel C) must become reachable without priority inversion!
	selected, err = service.SelectNextChannelInRoute(route, "gpt-4o", "default", attempted)
	require.NoError(t, err)
	assert.Equal(t, chC.Id, selected.Id, "channel C (priority 50) must become reachable when top tier exhausted")
}

// Section 14: Attempt Exhaustion & Deterministic Termination
func TestRouteStaging_AttemptExhaustion(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Route{}))

	r1 := &model.Route{
		Id:             601,
		Name:           "Route 1",
		Slug:           "route-1",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, r1.SetChannelIds([]int{101}))
	require.NoError(t, db.Create(r1).Error)

	r2 := &model.Route{
		Id:             602,
		Name:           "Route 2",
		Slug:           "route-2",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.2,
		Enabled:        true,
	}
	require.NoError(t, r2.SetChannelIds([]int{102}))
	require.NoError(t, db.Create(r2).Error)

	params := &service.RouteSelectionParams{
		UserGroup:         "default",
		ModelName:         "test-model",
		RouteChain:        []int{r1.Id, r2.Id},
		CurrentChainIndex: 0,
		AttemptedChannels: map[int]struct{}{101: {}, 102: {}}, // all channels already attempted
		TraceLog:          make([]service.RouteAttemptTrace, 0),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	info := &relaycommon.RelayInfo{OriginModelName: "test-model"}

	// When all channels in chain are exhausted, getChannelForRoute must return terminal error
	channel, route, newApiErr := getChannelForRoute(c, info, params)
	assert.Nil(t, channel)
	assert.Nil(t, route)
	require.NotNil(t, newApiErr)
	assert.Contains(t, newApiErr.Error(), "no available channels found across configured route chain")
	assert.True(t, types.IsSkipRetryError(newApiErr), "exhaustion error must skip retry")
}

// Section 15: BYOK Staging Isolation Regression
func TestRouteStaging_BYOKTerminalIsolation(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Route{}))

	// Managed route
	managedRoute := &model.Route{
		Id:             701,
		Name:           "Managed Route",
		Slug:           "managed-route",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, managedRoute.SetChannelIds([]int{1}))
	require.NoError(t, db.Create(managedRoute).Error)

	// BYOK route
	byokRoute := &model.Route{
		Id:             702,
		Name:           "BYOK Route",
		Slug:           "byok-route",
		Kind:           model.RouteKindBYOK,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, byokRoute.SetChannelIds([]int{2}))
	require.NoError(t, db.Create(byokRoute).Error)

	// BYOK request context
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("is_byok", true)
	info := &relaycommon.RelayInfo{OriginModelName: "gpt-4o", IsBYOK: true}

	params := &service.RouteSelectionParams{
		UserGroup:         "default",
		ModelName:         "gpt-4o",
		RouteChain:        []int{managedRoute.Id, byokRoute.Id},
		CurrentChainIndex: 0,
		AttemptedChannels: make(map[int]struct{}),
		TraceLog:          make([]service.RouteAttemptTrace, 0),
	}

	// Managed route MUST be skipped for BYOK request
	channel, route, _ := getChannelForRoute(c, info, params)
	assert.Nil(t, channel) // Channel 2 not seeded, but route 701 was skipped
	assert.Nil(t, route)
	assert.Equal(t, 2, params.CurrentChainIndex, "managed route must be skipped; byok route evaluated")
}

// Section 18 & 19: Direct Route-ID Tampering & Database Serialization
func TestRouteStaging_DirectRouteIDTamperingAndSerialization(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Route{}, &model.Token{}))

	token := &model.Token{
		Id:             801,
		UserId:         1,
		Name:           "Serialization Key",
		PrimaryRouteId: 10,
	}

	// 1. Serialization deduplication and filtering
	err := token.SetFallbackRouteIds([]int{10, 20, 20, -5, 0, 30})
	require.NoError(t, err)
	fallbacks, err := token.GetFallbackRouteIds()
	require.NoError(t, err)
	assert.Equal(t, []int{20, 30}, fallbacks, "primary route ID, duplicates, and non-positives must be filtered")

	// 2. Malformed JSON in FallbackRouteIds
	token.FallbackRouteIds = "{invalid-json}"
	chain := token.GetRouteChain()
	assert.Equal(t, []int{10}, chain, "malformed fallback JSON must gracefully fall back to primary route without crashing")

	// 3. Referential deletion protection (DeleteRouteSafe)
	route := &model.Route{
		Id:             901,
		Name:           "Protected Route",
		Slug:           "protected-route",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, db.Create(route).Error)

	boundToken := &model.Token{
		Id:             802,
		UserId:         1,
		Name:           "Bound Token",
		PrimaryRouteId: route.Id,
	}
	require.NoError(t, db.Create(boundToken).Error)

	// Attempting to delete route bound to active token MUST fail with ErrRouteReferencedByKeys
	err = model.DeleteRouteSafe(route.Id)
	assert.ErrorIs(t, err, model.ErrRouteReferencedByKeys)
}

// Section 22: Load & Concurrency Simulation (100 parallel selections)
func TestRouteStaging_LoadAndConcurrency(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Route{}))

	ch := &model.Channel{
		Id:       950,
		Name:     "concurrent-ch",
		Type:     1,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "gpt-4o",
		Priority: common.GetPointer(int64(100)),
		Weight:   common.GetPointer(uint(50)),
	}
	require.NoError(t, db.Create(ch).Error)
	model.CacheUpdateChannel(ch)

	route := &model.Route{
		Id:             951,
		Name:           "Concurrent Route",
		Slug:           "concurrent-route",
		Kind:           model.RouteKindLLM,
		CostMultiplier: 1.0,
		Enabled:        true,
	}
	require.NoError(t, route.SetChannelIds([]int{ch.Id}))
	require.NoError(t, db.Create(route).Error)
	service.CleanRouteCache(route.Id)

	var wg sync.WaitGroup
	var successCount atomic.Int64
	concurrentRequests := 100

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := service.CacheGetRoute(route.Id)
			if err == nil && r != nil {
				attempted := make(map[int]struct{})
				c, err := service.SelectNextChannelInRoute(r, "gpt-4o", "default", attempted)
				if err == nil && c != nil && c.Id == ch.Id {
					successCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(concurrentRequests), successCount.Load(), "all 100 concurrent route selections must succeed without race conditions")
}
