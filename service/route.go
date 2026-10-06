package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
)

var (
	routeMemoryCache     = make(map[int]*model.Route)
	routeMemoryCacheLock sync.RWMutex

	// channelCooldowns tracks transient cooldowns (e.g. on 429 Retry-After) without mutating DB status
	channelCooldowns     = make(map[int]time.Time)
	channelCooldownsLock sync.RWMutex
)

const (
	RouteCacheInvalidateTopic = "tora:cache:route:invalidate"
	DefaultCooldownDuration    = 15 * time.Second
)

func InitRouteSubscribers() {
	if !common.RedisEnabled {
		return
	}
	gopool.Go(func() {
		pubsub := common.RDB.Subscribe(context.Background(), RouteCacheInvalidateTopic)
		defer pubsub.Close()
		ch := pubsub.Channel()
		for msg := range ch {
			if msg != nil && msg.Payload != "" {
				routeMemoryCacheLock.Lock()
				// Invalidate all or specific ID
				routeMemoryCache = make(map[int]*model.Route)
				routeMemoryCacheLock.Unlock()
			}
		}
	})
}

// InvalidateRouteCache clears local memory cache and broadcasts to other nodes via Redis
func InvalidateRouteCache(id int) {
	routeMemoryCacheLock.Lock()
	if id > 0 {
		delete(routeMemoryCache, id)
	} else {
		routeMemoryCache = make(map[int]*model.Route)
	}
	routeMemoryCacheLock.Unlock()

	if common.RedisEnabled && common.RDB != nil {
		gopool.Go(func() {
			_ = common.RDB.Publish(context.Background(), RouteCacheInvalidateTopic, fmt.Sprintf("%d", id)).Err()
		})
	}
}

// CacheGetRoute retrieves a Route from memory cache or database
func CacheGetRoute(id int) (*model.Route, error) {
	if id <= 0 {
		return nil, model.ErrRouteNotFound
	}

	routeMemoryCacheLock.RLock()
	r, ok := routeMemoryCache[id]
	routeMemoryCacheLock.RUnlock()
	if ok && r != nil {
		return r, nil
	}

	route, err := model.GetRouteById(id)
	if err != nil {
		return nil, err
	}

	routeMemoryCacheLock.Lock()
	routeMemoryCache[id] = route
	routeMemoryCacheLock.Unlock()

	return route, nil
}

const (
	RouteChannelCooldownKeyPrefix = "tora:cooldown:channel:"
)

// SetChannelCooldown sets a temporary cooldown timestamp for a channel (both local memory and distributed Redis)
func SetChannelCooldown(channelId int, d time.Duration) {
	if channelId <= 0 {
		return
	}
	if d <= 0 {
		d = DefaultCooldownDuration
	}
	channelCooldownsLock.Lock()
	channelCooldowns[channelId] = time.Now().Add(d)
	channelCooldownsLock.Unlock()

	if common.RedisEnabled && common.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		key := fmt.Sprintf("%s%d", RouteChannelCooldownKeyPrefix, channelId)
		_ = common.RDB.Set(ctx, key, "1", d).Err()
	}
}

// IsChannelInCooldown checks if a channel is currently in temporary backoff
func IsChannelInCooldown(channelId int) bool {
	channelCooldownsLock.RLock()
	expiry, exists := channelCooldowns[channelId]
	channelCooldownsLock.RUnlock()
	if exists {
		if time.Now().Before(expiry) {
			return true
		}
		channelCooldownsLock.Lock()
		delete(channelCooldowns, channelId)
		channelCooldownsLock.Unlock()
	}

	if common.RedisEnabled && common.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		key := fmt.Sprintf("%s%d", RouteChannelCooldownKeyPrefix, channelId)
		val, err := common.RDB.Get(ctx, key).Result()
		if err == nil && val == "1" {
			ttl, err := common.RDB.TTL(ctx, key).Result()
			if err == nil && ttl > 0 {
				channelCooldownsLock.Lock()
				channelCooldowns[channelId] = time.Now().Add(ttl)
				channelCooldownsLock.Unlock()
			}
			return true
		}
	}
	return false
}

// RouteSnapshot captures an immutable view of a Route at request start (Section 10 consistency)
type RouteSnapshot struct {
	Id             int               `json:"id"`
	Name           string            `json:"name"`
	Slug           string            `json:"slug"`
	Kind           string            `json:"kind"`
	RoutingPolicy  string            `json:"routing_policy"`
	ChannelIds     []int             `json:"channel_ids"`
	CostMultiplier float64           `json:"cost_multiplier"`
	MinUserGroup   string            `json:"min_user_group"`
	Enabled        bool              `json:"enabled"`
	ModelMapping   map[string]string `json:"model_mapping"`
}

func SnapshotRoute(r *model.Route) *RouteSnapshot {
	if r == nil {
		return nil
	}
	chIds, _ := r.GetChannelIds()
	mapping, _ := r.GetModelMapping()
	return &RouteSnapshot{
		Id:             r.Id,
		Name:           r.Name,
		Slug:           r.Slug,
		Kind:           r.Kind,
		RoutingPolicy:  r.RoutingPolicy,
		ChannelIds:     chIds,
		CostMultiplier: r.CostMultiplier,
		MinUserGroup:   r.MinUserGroup,
		Enabled:        r.Enabled,
		ModelMapping:   mapping,
	}
}

func (s *RouteSnapshot) ToRoute() *model.Route {
	if s == nil {
		return nil
	}
	chIdsJson, _ := common.Marshal(s.ChannelIds)
	mappingJson, _ := common.Marshal(s.ModelMapping)
	return &model.Route{
		Id:             s.Id,
		Name:           s.Name,
		Slug:           s.Slug,
		Kind:           s.Kind,
		RoutingPolicy:  s.RoutingPolicy,
		ChannelIds:     string(chIdsJson),
		CostMultiplier: s.CostMultiplier,
		MinUserGroup:   s.MinUserGroup,
		Enabled:        s.Enabled,
		ModelMapping:   string(mappingJson),
	}
}

// RouteSelectionParams encapsulates the state of a route chain execution
type RouteSelectionParams struct {
	Token             *model.Token
	UserGroup         string
	ModelName         string
	RouteChain        []int
	RouteSnapshots    []*RouteSnapshot
	CurrentChainIndex int
	AttemptedChannels map[int]struct{}
	AttemptedRoutes   []int
	TraceLog          []RouteAttemptTrace
}

type RouteAttemptTrace struct {
	Attempt    int    `json:"attempt"`
	RouteId    int    `json:"route_id"`
	RouteName  string `json:"route_name"`
	ChannelId  int    `json:"channel_id"`
	StatusCode int    `json:"status_code"`
	Reason     string `json:"reason"`
	LatencyMs  int64  `json:"latency_ms"`
}

// ResolveModelForRoute checks if the route has model alias / rewriting configured
func ResolveModelForRoute(route *model.Route, requestedModel string) string {
	if route == nil {
		return requestedModel
	}
	mapping, err := route.GetModelMapping()
	if err == nil && len(mapping) > 0 {
		if rewritten, ok := mapping[requestedModel]; ok && rewritten != "" {
			return rewritten
		}
	}
	return requestedModel
}

// ValidateRouteEntitlement checks if the user's commercial tier allows accessing this route.
// Prime Invariant: Route != Entitlement Group. A route cannot elevate a user's entitlement.
func ValidateRouteEntitlement(route *model.Route, userGroup string) error {
	if route == nil {
		return errors.New("nil route")
	}
	if !route.Enabled {
		return fmt.Errorf("route %s (%d) is disabled", route.Name, route.Id)
	}
	if route.MinUserGroup != "" && route.MinUserGroup != "default" {
		// If route requires "pro", user must be in "pro" group
		if userGroup != route.MinUserGroup && userGroup != "root" && userGroup != "admin" {
			return fmt.Errorf("route %s requires %s entitlement, user has %s", route.Name, route.MinUserGroup, userGroup)
		}
	}
	return nil
}

// RouteSupportsModel checks if the route has at least one enabled channel supporting the requested (or mapped) model
// that is accessible to the caller's userGroup.
func RouteSupportsModel(route *model.Route, requestedModel string, userGroup string) bool {
	if route == nil || !route.Enabled {
		return false
	}
	channelIds, err := route.GetChannelIds()
	if err != nil || len(channelIds) == 0 {
		return false
	}
	effectiveModel := ResolveModelForRoute(route, requestedModel)
	for _, id := range channelIds {
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		if userGroup != "" && userGroup != "root" && userGroup != "admin" {
			groupAllowed := false
			for _, g := range ch.GetGroups() {
				if g == userGroup || g == "default" {
					groupAllowed = true
					break
				}
			}
			if !groupAllowed {
				continue
			}
		}
		for _, m := range ch.GetModels() {
			if m == effectiveModel || m == requestedModel {
				return true
			}
		}
	}
	return false
}

// SelectNextChannelInRoute resolves the next eligible channel for the given Route.
// It explicitly excludes already attempted channels, handles channel priority tiers safely,
// and respects weighted random distribution within the highest available priority tier.
func SelectNextChannelInRoute(
	route *model.Route,
	requestedModel string,
	userGroup string,
	attemptedChannels map[int]struct{},
) (*model.Channel, error) {
	if route == nil {
		return nil, errors.New("nil route")
	}

	channelIds, err := route.GetChannelIds()
	if err != nil {
		return nil, err
	}
	if len(channelIds) == 0 {
		return nil, fmt.Errorf("route %s has no configured channels", route.Name)
	}

	effectiveModel := ResolveModelForRoute(route, requestedModel)

	// Gather all enabled candidates that support the model, are not in cooldown, and haven't been attempted
	var candidates []*model.Channel
	for _, id := range channelIds {
		if _, attempted := attemptedChannels[id]; attempted {
			continue // Prevent repeating failed channel
		}
		if IsChannelInCooldown(id) {
			continue // Skip transient cooldown
		}
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil {
			continue
		}
		if ch.Status != common.ChannelStatusEnabled {
			continue
		}
		// Invariant: Channel group entitlement. Channel must be accessible to userGroup (unless admin/root)
		if userGroup != "" && userGroup != "root" && userGroup != "admin" {
			groupAllowed := false
			for _, g := range ch.GetGroups() {
				if g == userGroup || g == "default" {
					groupAllowed = true
					break
				}
			}
			if !groupAllowed {
				continue
			}
		}
		// Verify model support
		modelList := ch.GetModels()
		supported := false
		for _, m := range modelList {
			if m == effectiveModel || m == requestedModel {
				supported = true
				break
			}
		}
		if !supported {
			continue
		}
		candidates = append(candidates, ch)
	}

	if len(candidates) == 0 {
		return nil, nil // No remaining candidate channels in this route
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// Group by priority tiers
	priorityMap := make(map[int64][]*model.Channel)
	for _, ch := range candidates {
		p := ch.GetPriority()
		priorityMap[p] = append(priorityMap[p], ch)
	}

	var uniquePriorities []int64
	for p := range priorityMap {
		uniquePriorities = append(uniquePriorities, p)
	}
	sort.Slice(uniquePriorities, func(i, j int) bool {
		return uniquePriorities[i] > uniquePriorities[j]
	})

	// Top priority tier among available candidates
	topTierChannels := priorityMap[uniquePriorities[0]]
	if len(topTierChannels) == 1 {
		return topTierChannels[0], nil
	}

	// Weighted random selection within top tier
	totalWeight := 0
	for _, ch := range topTierChannels {
		w := ch.GetWeight()
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	randomWeight := rand.Intn(totalWeight)
	for _, ch := range topTierChannels {
		w := ch.GetWeight()
		if w <= 0 {
			w = 1
		}
		randomWeight -= w
		if randomWeight < 0 {
			return ch, nil
		}
	}

	return topTierChannels[0], nil
}

// CleanRouteCache is an alias for InvalidateRouteCache to support cache clearance in tests and operations.
func CleanRouteCache(id int) {
	InvalidateRouteCache(id)
}

// CalculateSettledQuota computes authoritative settled quota:
// SettledQuota = ceil(BaseQuota * ModelRatio * UserGroupRatio * RouteCostMultiplier)
func CalculateSettledQuota(baseQuota int, modelRatio float64, userGroupRatio float64, routeMultiplier float64) int {
	if routeMultiplier <= 0 {
		routeMultiplier = 1.0
	}
	dBase := decimal.NewFromInt(int64(baseQuota))
	dRatio := decimal.NewFromFloat(modelRatio).Mul(decimal.NewFromFloat(userGroupRatio)).Mul(decimal.NewFromFloat(routeMultiplier))
	dFinal := dBase.Mul(dRatio)
	return int(math.Ceil(dFinal.InexactFloat64()))
}
