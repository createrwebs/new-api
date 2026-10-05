package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

var (
	relayMaxUserConcurrency   = common.GetEnvOrDefault("RELAY_MAX_USER_CONCURRENCY", 20)
	relayMaxGlobalConcurrency = common.GetEnvOrDefault("RELAY_MAX_GLOBAL_CONCURRENCY", 500)
)

const (
	redisAcquireLeaseScript = `
local key = KEYS[1]
local leaseID = ARGV[1]
local now = tonumber(ARGV[2])
local expireAt = tonumber(ARGV[3])
local maxConcurrency = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

-- 1. Remove expired leases
redis.call('ZREMRANGEBYSCORE', key, '-inf', now)

-- 2. Count active leases
local count = redis.call('ZCARD', key)
if count < maxConcurrency then
  redis.call('ZADD', key, expireAt, leaseID)
  redis.call('EXPIRE', key, ttl)
  return 1
else
  return 0
end
`

	redisReleaseLeaseScript = `
local key = KEYS[1]
local leaseID = ARGV[1]

redis.call('ZREM', key, leaseID)
local count = redis.call('ZCARD', key)
if count == 0 then
  redis.call('DEL', key)
end
return 1
`
)

type inMemoryLeaseTracker struct {
	mu     sync.Mutex
	leases map[string]map[string]time.Time // key -> leaseID -> expireAt
}

var memoryTracker = &inMemoryLeaseTracker{
	leases: make(map[string]map[string]time.Time),
}

func (t *inMemoryLeaseTracker) acquire(key string, leaseID string, maxConcurrency int, ttl time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	activeMap, exists := t.leases[key]
	if !exists {
		activeMap = make(map[string]time.Time)
		t.leases[key] = activeMap
	}

	// Purge expired
	for id, expireAt := range activeMap {
		if now.After(expireAt) {
			delete(activeMap, id)
		}
	}

	if len(activeMap) >= maxConcurrency {
		return false
	}

	activeMap[leaseID] = now.Add(ttl)
	return true
}

func (t *inMemoryLeaseTracker) release(key string, leaseID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if activeMap, exists := t.leases[key]; exists {
		delete(activeMap, leaseID)
		if len(activeMap) == 0 {
			delete(t.leases, key)
		}
	}
}

// AcquireConcurrencyLease attempts to acquire a concurrency slot under the given key.
// If Redis is enabled and accessible, it uses an atomic ZSET lease.
// If Redis is disabled or encounters an error, it falls back to the in-memory tracker.
func AcquireConcurrencyLease(ctx context.Context, key string, leaseID string, maxConcurrency int, ttl time.Duration) bool {
	if maxConcurrency <= 0 {
		return true
	}

	if common.RedisEnabled && common.RDB != nil {
		nowSec := time.Now().Unix()
		expireSec := nowSec + int64(ttl.Seconds())
		ttlSec := int64(ttl.Seconds())
		if ttlSec <= 0 {
			ttlSec = 600
		}

		res, err := common.RDB.Eval(ctx, redisAcquireLeaseScript, []string{key}, leaseID, nowSec, expireSec, maxConcurrency, ttlSec).Int()
		if err == nil {
			return res == 1
		}
		// Redis error: log and fallback to memoryTracker
		common.SysError(fmt.Sprintf("redis acquire concurrency lease error for key %s: %v, falling back to memory tracker", key, err))
	}

	return memoryTracker.acquire(key, leaseID, maxConcurrency, ttl)
}

// ReleaseConcurrencyLease releases a previously acquired concurrency lease.
func ReleaseConcurrencyLease(ctx context.Context, key string, leaseID string) {
	if common.RedisEnabled && common.RDB != nil {
		_ = common.RDB.Eval(ctx, redisReleaseLeaseScript, []string{key}, leaseID).Err()
	}
	memoryTracker.release(key, leaseID)
}

// SetRelayConcurrencyLimitsForTest allows tests to override concurrency limits.
func SetRelayConcurrencyLimitsForTest(userLimit, globalLimit int) func() {
	prevUser, prevGlobal := relayMaxUserConcurrency, relayMaxGlobalConcurrency
	relayMaxUserConcurrency, relayMaxGlobalConcurrency = userLimit, globalLimit
	return func() {
		relayMaxUserConcurrency, relayMaxGlobalConcurrency = prevUser, prevGlobal
	}
}

func generateLeaseID(prefix string, id int) string {
	return fmt.Sprintf("%s_%d_%d_%d", prefix, id, time.Now().UnixNano(), rand.Int63())
}

// RelayConcurrencyLimit enforces concurrency limits on system relay traffic.
// It bypasses BYOK requests (which enforce BYOK_MAX_CONCURRENCY in BYOKRouter).
func RelayConcurrencyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Skip if BYOK to prevent double limiting
		if c.GetBool("is_byok") || common.GetContextKeyBool(c, constant.ContextKeyIsBYOK) {
			c.Next()
			return
		}

		// 2. Identify user
		userID := c.GetInt("id")
		if userID == 0 {
			userID = common.GetContextKeyInt(c, constant.ContextKeyUserId)
		}
		if userID <= 0 {
			// If not authenticated yet or internal route, proceed
			c.Next()
			return
		}

		timeoutSeconds := common.RelayTimeout
		if timeoutSeconds <= 0 {
			timeoutSeconds = 600
		}
		ttl := time.Duration(timeoutSeconds) * time.Second

		leaseID := generateLeaseID("relay", userID)

		// 3. Global safety concurrency check
		globalKey := "concurrency:relay:global"
		if !AcquireConcurrencyLease(c.Request.Context(), globalKey, leaseID, relayMaxGlobalConcurrency, ttl) {
			writeRelayConcurrencyLimited(c, "Server is currently at maximum relay capacity. Please retry shortly.")
			return
		}
		defer ReleaseConcurrencyLease(context.Background(), globalKey, leaseID)

		// 4. Per-user concurrency check
		userKey := fmt.Sprintf("concurrency:relay:user:%d", userID)
		if !AcquireConcurrencyLease(c.Request.Context(), userKey, leaseID, relayMaxUserConcurrency, ttl) {
			writeRelayConcurrencyLimited(c, fmt.Sprintf("Too many concurrent requests. Maximum allowed is %d.", relayMaxUserConcurrency))
			return
		}
		defer ReleaseConcurrencyLease(context.Background(), userKey, leaseID)

		c.Next()
	}
}

func writeRelayConcurrencyLimited(c *gin.Context, message string) {
	c.Header("Retry-After", "2")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "requests",
			"code":    "relay_concurrency_limit_exceeded",
		},
	})
}
