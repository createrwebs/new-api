package common

import (
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestVerificationCodeInMemoryLifecycleAndAntiReplay(t *testing.T) {
	oldRedis := RedisEnabled
	RedisEnabled = false
	defer func() { RedisEnabled = oldRedis }()

	email := "testuser@example.com"
	code := "123456"
	purpose := EmailVerificationPurpose

	RegisterVerificationCodeWithKey(email, code, purpose)

	// Wrong code fails
	assert.False(t, VerifyCodeWithKey(email, "654321", purpose))

	// Correct code succeeds (and is atomically consumed)
	assert.True(t, VerifyCodeWithKey(email, code, purpose))

	// Replay attack: verifying the SAME code again must FAIL
	assert.False(t, VerifyCodeWithKey(email, code, purpose), "consumed verification code must not be reusable")
}

func TestVerificationCodeInMemoryBruteForceProtection(t *testing.T) {
	oldRedis := RedisEnabled
	RedisEnabled = false
	defer func() { RedisEnabled = oldRedis }()

	email := "bruteforce@example.com"
	code := "correct-code"
	purpose := PasswordResetPurpose

	RegisterVerificationCodeWithKey(email, code, purpose)

	// Enter wrong code 5 times
	for i := 0; i < MaxVerificationAttempts; i++ {
		assert.False(t, VerifyCodeWithKey(email, "wrong-code", purpose))
	}

	// 6th attempt with CORRECT code should FAIL because max attempts was exceeded
	assert.False(t, VerifyCodeWithKey(email, code, purpose), "code must be invalidated after max failed attempts")
}

func TestVerificationCodeConcurrentConsumption(t *testing.T) {
	oldRedis := RedisEnabled
	RedisEnabled = false
	defer func() { RedisEnabled = oldRedis }()

	email := "concurrency@example.com"
	code := "race-code-777"
	purpose := EmailVerificationPurpose

	RegisterVerificationCodeWithKey(email, code, purpose)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// 10 concurrent requests attempting to consume the same code
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if VerifyCodeWithKey(email, code, purpose) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Exactly ONE request must succeed!
	assert.Equal(t, 1, successCount, "exactly one concurrent request must consume the code")
}

func TestVerificationCodeSafeFailureModeWhenRedisNil(t *testing.T) {
	oldRedis := RedisEnabled
	oldRDB := RDB
	RedisEnabled = true
	RDB = nil
	defer func() {
		RedisEnabled = oldRedis
		RDB = oldRDB
	}()

	email := "safe-failure@example.com"
	code := "123456"
	purpose := EmailVerificationPurpose

	// When Redis is enabled but RDB is nil, verify must safely reject (fail closed)
	assert.False(t, VerifyCodeWithKey(email, code, purpose), "must safely reject when redis client is unavailable")
}

func TestVerificationCodeRedisDistributedAndAntiReplay(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	oldRedis := RedisEnabled
	oldRDB := RDB
	RedisEnabled = true
	RDB = rdb
	defer func() {
		RedisEnabled = oldRedis
		RDB = oldRDB
	}()

	emailA := "userA@example.com"
	emailB := "userB@example.com"
	code := "redis-otp-123456"

	// 1. Register code under EmailVerificationPurpose (Node A writes to Redis)
	RegisterVerificationCodeWithKey(emailA, code, EmailVerificationPurpose)

	// 2. Scoping check: User B cannot consume User A's code
	assert.False(t, VerifyCodeWithKey(emailB, code, EmailVerificationPurpose), "cross-user consumption must fail")

	// 3. Purpose scoping check: PasswordResetPurpose cannot consume EmailVerification code
	assert.False(t, VerifyCodeWithKey(emailA, code, PasswordResetPurpose), "cross-purpose consumption must fail")

	// 4. Wrong code fails
	assert.False(t, VerifyCodeWithKey(emailA, "wrong-otp", EmailVerificationPurpose))

	// 5. Node B verifies and consumes the code atomically
	assert.True(t, VerifyCodeWithKey(emailA, code, EmailVerificationPurpose), "valid code must succeed")

	// 6. Anti-replay: second verification of the exact same code MUST fail
	assert.False(t, VerifyCodeWithKey(emailA, code, EmailVerificationPurpose), "consumed code must not be reusable")
}

func TestVerificationCodeRedisConcurrentRace(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	oldRedis := RedisEnabled
	oldRDB := RDB
	RedisEnabled = true
	RDB = rdb
	defer func() {
		RedisEnabled = oldRedis
		RDB = oldRDB
	}()

	email := "race-redis@example.com"
	code := "atomic-race-999"
	purpose := PasswordResetPurpose

	RegisterVerificationCodeWithKey(email, code, purpose)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if VerifyCodeWithKey(email, code, purpose) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Exactly one request succeeds in consuming the code from Redis
	assert.Equal(t, 1, successCount, "exactly one concurrent request must consume the code via Lua script")
}

func TestVerificationCodeRedisBruteForceLockout(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	oldRedis := RedisEnabled
	oldRDB := RDB
	RedisEnabled = true
	RDB = rdb
	defer func() {
		RedisEnabled = oldRedis
		RDB = oldRDB
	}()

	email := "lockout@example.com"
	code := "secret-code"
	purpose := EmailVerificationPurpose

	RegisterVerificationCodeWithKey(email, code, purpose)

	// Enter wrong code 5 times
	for i := 0; i < MaxVerificationAttempts; i++ {
		assert.False(t, VerifyCodeWithKey(email, "wrong", purpose))
	}

	// 6th attempt with correct code must fail because code was deleted due to brute force
	assert.False(t, VerifyCodeWithKey(email, code, purpose), "exceeding max attempts must invalidate code in Redis")
}
