package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemRelayConcurrencyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set test limits: max 2 concurrent requests per user, 10 global
	resetLimits := SetRelayConcurrencyLimitsForTest(2, 10)
	defer resetLimits()

	userID := 12345

	holdCh1 := make(chan struct{})
	holdCh2 := make(chan struct{})
	readyCh := make(chan struct{}, 2)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", userID)
		common.SetContextKey(c, constant.ContextKeyUserId, userID)
		c.Next()
	})
	router.Use(RelayConcurrencyLimit())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		readyCh <- struct{}{}
		reqID := c.GetHeader("X-Req-ID")
		if reqID == "1" {
			<-holdCh1
		} else if reqID == "2" {
			<-holdCh2
		}
		c.Status(http.StatusOK)
	})

	body := `{"model": "gpt-4o", "messages": []}`

	// Request 1: hold open
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Req-ID", "1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}()

	// Wait for Request 1 to be holding
	select {
	case <-readyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request 1")
	}

	// Request 2: hold open
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Req-ID", "2")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}()

	// Wait for Request 2 to be holding
	select {
	case <-readyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request 2")
	}

	// Request 3: should be REJECTED (exceeds user limit of 2)
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("X-Req-ID", "3")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Contains(t, w3.Body.String(), "relay_concurrency_limit_exceeded")
	assert.NotEmpty(t, w3.Header().Get("Retry-After"))

	// Release Request 1
	close(holdCh1)
	time.Sleep(50 * time.Millisecond)

	// Now Request 4 should SUCCEED because Request 1 released its slot!
	req4 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("X-Req-ID", "4")
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)

	assert.Equal(t, http.StatusOK, w4.Code, "request 4 should succeed after request 1 slot is released")

	// Release Request 2
	close(holdCh2)
}

func TestSystemRelayConcurrencyBypassesBYOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// User concurrency set to 1
	resetLimits := SetRelayConcurrencyLimitsForTest(1, 10)
	defer resetLimits()

	userID := 54321

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", userID)
		common.SetContextKey(c, constant.ContextKeyUserId, userID)
		// Mark as BYOK
		c.Set("is_byok", true)
		common.SetContextKey(c, constant.ContextKeyIsBYOK, true)
		c.Next()
	})
	router.Use(RelayConcurrencyLimit())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := `{"model": "openrouter/meta-llama/llama-3-8b", "messages": []}`

	// Multiple requests should NOT be limited by system relay concurrency because is_byok is true
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "BYOK request should bypass system relay concurrency")
	}
}

func TestConcurrencyLeaseTTLRecovery(t *testing.T) {
	key := "test:lease:recovery"
	leaseID1 := "lease_crash_simulation"
	leaseID2 := "lease_active"

	// Acquire slot 1 with tiny TTL (50ms) simulating a crashed process that never releases
	acquired := AcquireConcurrencyLease(context.Background(), key, leaseID1, 1, 50*time.Millisecond)
	require.True(t, acquired)

	// Immediate attempt to acquire slot 2 fails because maxCount is 1
	acquired2 := AcquireConcurrencyLease(context.Background(), key, leaseID2, 1, 50*time.Millisecond)
	assert.False(t, acquired2, "should be rejected while slot 1 is active")

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Slot 2 should now SUCCEED because lease 1 expired and was auto-purged!
	acquired3 := AcquireConcurrencyLease(context.Background(), key, leaseID2, 1, 1*time.Second)
	assert.True(t, acquired3, "should succeed after lease 1 TTL expires (crash recovery)")

	// Cleanup
	ReleaseConcurrencyLease(context.Background(), key, leaseID2)
}

func TestConcurrentAcquireAndRelease(t *testing.T) {
	key := "test:concurrent:stress"
	maxSlots := 5
	var wg sync.WaitGroup

	successCount := 0
	var mu sync.Mutex

	// 20 concurrent workers competing for 5 slots
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			leaseID := generateLeaseID("worker", workerID)
			if AcquireConcurrencyLease(context.Background(), key, leaseID, maxSlots, 2*time.Second) {
				mu.Lock()
				successCount++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				ReleaseConcurrencyLease(context.Background(), key, leaseID)
			}
		}(i)
	}

	wg.Wait()

	assert.True(t, successCount >= 5, "at least maxSlots should have successfully acquired slots")
}
