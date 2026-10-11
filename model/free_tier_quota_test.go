package model

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreeTierCalendarDayUsesBangkokBoundary(t *testing.T) {
	beforeMidnight := time.Date(2026, 1, 1, 16, 59, 59, 0, time.UTC)
	afterMidnight := beforeMidnight.Add(2 * time.Second)

	beforeDay, err := FreeTierCalendarDay(beforeMidnight)
	require.NoError(t, err)
	afterDay, err := FreeTierCalendarDay(afterMidnight)
	require.NoError(t, err)

	assert.Equal(t, "2026-01-01", beforeDay)
	assert.Equal(t, "2026-01-02", afterDay)
}

func TestReserveFreeTierRequestStopsAtDailyLimit(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&User{}, &FreeTierDailyUsage{}))

	user := User{Username: "free-tier-" + t.Name(), Password: "unused", Role: 1, Status: 1, Group: "default", AuthVersion: 1}
	require.NoError(t, DB.Create(&user).Error)
	day := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)

	for range FreeTierDailyLimit {
		allowed, err := ReserveFreeTierRequest(user.Id, day)
		require.NoError(t, err)
		require.True(t, allowed)
	}
	allowed, err := ReserveFreeTierRequest(user.Id, day)
	require.NoError(t, err)
	assert.False(t, allowed)

	var usage FreeTierDailyUsage
	require.NoError(t, DB.Where("user_id = ?", user.Id).First(&usage).Error)
	assert.Equal(t, FreeTierDailyLimit, usage.Used)
}

func TestReserveFreeTierRequestConcurrentDoesNotExceedLimit(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&User{}, &FreeTierDailyUsage{}))

	user := User{Username: "free-tier-concurrent-" + t.Name(), Password: "unused", Role: 1, Status: 1, Group: "default", AuthVersion: 1}
	require.NoError(t, DB.Create(&user).Error)
	day := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)

	var wait sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0
	for range 50 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			allowed, err := ReserveFreeTierRequest(user.Id, day)
			if err != nil {
				return
			}
			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	wait.Wait()

	assert.Equal(t, int(FreeTierDailyLimit), allowedCount)
}
