package common

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type verificationValue struct {
	code     string
	time     time.Time
	attempts int
}

const (
	EmailVerificationPurpose = "v"
	PasswordResetPurpose     = "r"
	MaxVerificationAttempts  = 5
)

var verificationMutex sync.Mutex
var verificationMap map[string]verificationValue
var verificationMapMaxSize = 100
var VerificationValidMinutes = 10

func redisVerificationKey(purpose, key string) string {
	return fmt.Sprintf("verification:%s:%s", purpose, strings.ToLower(strings.TrimSpace(key)))
}

func redisVerificationAttemptsKey(purpose, key string) string {
	return fmt.Sprintf("verification:attempts:%s:%s", purpose, strings.ToLower(strings.TrimSpace(key)))
}

func GenerateVerificationCode(length int) string {
	code := uuid.New().String()
	code = strings.Replace(code, "-", "", -1)
	if length == 0 {
		return code
	}
	return code[:length]
}

func RegisterVerificationCodeWithKey(key string, code string, purpose string) {
	key = strings.ToLower(strings.TrimSpace(key))
	code = strings.TrimSpace(code)
	if key == "" || code == "" {
		return
	}

	if RedisEnabled {
		if RDB == nil {
			SysError("redis is enabled but RDB is nil")
			return
		}
		ctx := context.Background()
		ttl := time.Duration(VerificationValidMinutes) * time.Minute
		pipe := RDB.Pipeline()
		pipe.Set(ctx, redisVerificationKey(purpose, key), code, ttl)
		pipe.Del(ctx, redisVerificationAttemptsKey(purpose, key))
		_, err := pipe.Exec(ctx)
		if err != nil {
			SysError(fmt.Sprintf("failed to save verification code to redis: %v", err))
		}
		return
	}

	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap[purpose+key] = verificationValue{
		code:     code,
		time:     time.Now(),
		attempts: 0,
	}
	if len(verificationMap) > verificationMapMaxSize {
		removeExpiredPairs()
	}
}

func VerifyCodeWithKey(key string, code string, purpose string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	code = strings.TrimSpace(code)
	if key == "" || code == "" {
		return false
	}

	if RedisEnabled {
		if RDB == nil {
			SysError("redis is enabled but RDB is nil")
			return false // safe failure mode
		}
		ctx := context.Background()
		luaScript := `
			local codeKey = KEYS[1]
			local attemptsKey = KEYS[2]
			local maxAttempts = tonumber(ARGV[2])
			local currentAttempts = tonumber(redis.call('GET', attemptsKey) or '0')

			if currentAttempts >= maxAttempts then
				redis.call('DEL', codeKey)
				redis.call('DEL', attemptsKey)
				return 0
			end

			local stored = redis.call('GET', codeKey)
			if not stored then
				return 0
			end

			if stored == ARGV[1] then
				redis.call('DEL', codeKey)
				redis.call('DEL', attemptsKey)
				return 1
			else
				redis.call('INCR', attemptsKey)
				redis.call('EXPIRE', attemptsKey, 600)
				return 0
			end
		`
		result, err := RDB.Eval(ctx, luaScript, []string{
			redisVerificationKey(purpose, key),
			redisVerificationAttemptsKey(purpose, key),
		}, code, MaxVerificationAttempts).Int()
		if err != nil {
			SysError(fmt.Sprintf("failed to verify code with redis: %v", err))
			return false // safe failure mode
		}
		return result == 1
	}

	// In-memory fallback
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	fullKey := purpose + key
	value, exists := verificationMap[fullKey]
	now := time.Now()
	if !exists || int(now.Sub(value.time).Seconds()) >= VerificationValidMinutes*60 {
		return false
	}
	if value.attempts >= MaxVerificationAttempts {
		delete(verificationMap, fullKey)
		return false
	}
	if value.code != code {
		value.attempts++
		verificationMap[fullKey] = value
		return false
	}

	// Atomic consumption on match
	delete(verificationMap, fullKey)
	return true
}

func DeleteKey(key string, purpose string) {
	key = strings.ToLower(strings.TrimSpace(key))
	if RedisEnabled {
		if RDB != nil {
			ctx := context.Background()
			RDB.Del(ctx, redisVerificationKey(purpose, key), redisVerificationAttemptsKey(purpose, key))
		}
		return
	}
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	delete(verificationMap, purpose+key)
}

func removeExpiredPairs() {
	now := time.Now()
	for key, val := range verificationMap {
		if int(now.Sub(val.time).Seconds()) >= VerificationValidMinutes*60 {
			delete(verificationMap, key)
		}
	}
}

func init() {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()
	verificationMap = make(map[string]verificationValue)
}
