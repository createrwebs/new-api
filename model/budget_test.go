package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBudgetTestDB(t *testing.T) {
	t.Helper()

	previousDB := DB

	dsn := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared",
		strings.ReplaceAll(t.Name(), "/", "_"),
	)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	DB = db

	require.NoError(t, DB.AutoMigrate(
		&BudgetRule{},
		&BudgetUsage{},
	))

	sqlDB, err := db.DB()
	require.NoError(t, err)

	t.Cleanup(func() {
		DB = previousDB
		_ = sqlDB.Close()
	})
}

func createBudgetRule(
	t *testing.T,
	scopeType string,
	scopeID int64,
	period string,
	limit int64,
) BudgetRule {
	t.Helper()

	rule := BudgetRule{
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		Period:     period,
		LimitQuota: limit,
		Enabled:    true,
	}

	require.NoError(t, DB.Create(&rule).Error)

	return rule
}

func TestReserveBudgetWithinLimit(t *testing.T) {
	setupBudgetTestDB(t)

	createBudgetRule(
		t,
		BudgetScopeUser,
		1,
		BudgetPeriodDaily,
		100,
	)

	now := time.Date(
		2026, 9, 29,
		10, 0, 0, 0,
		time.Local,
	)

	reservations, err := ReserveBudget(
		BudgetScopeUser,
		1,
		60,
		now,
	)

	require.NoError(t, err)
	require.Len(t, reservations, 1)

	periodStart, err := GetBudgetPeriodStart(
		BudgetPeriodDaily,
		now,
	)
	require.NoError(t, err)

	used, err := GetBudgetUsage(
		reservations[0].RuleID,
		periodStart,
	)
	require.NoError(t, err)

	assert.EqualValues(t, 60, used)
}

func TestReserveBudgetRejectsOverLimit(t *testing.T) {
	setupBudgetTestDB(t)

	createBudgetRule(
		t,
		BudgetScopeUser,
		1,
		BudgetPeriodDaily,
		100,
	)

	now := time.Date(
		2026, 9, 29,
		10, 0, 0, 0,
		time.Local,
	)

	first, err := ReserveBudget(
		BudgetScopeUser,
		1,
		80,
		now,
	)
	require.NoError(t, err)
	require.Len(t, first, 1)

	_, err = ReserveBudget(
		BudgetScopeUser,
		1,
		30,
		now,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "budget exceeded")

	periodStart, err := GetBudgetPeriodStart(
		BudgetPeriodDaily,
		now,
	)
	require.NoError(t, err)

	used, err := GetBudgetUsage(
		first[0].RuleID,
		periodStart,
	)
	require.NoError(t, err)

	// failed reservation must not change usage
	assert.EqualValues(t, 80, used)
}

func TestReleaseBudgetReturnsReservedQuota(t *testing.T) {
	setupBudgetTestDB(t)

	createBudgetRule(
		t,
		BudgetScopeToken,
		10,
		BudgetPeriodDaily,
		100,
	)

	now := time.Date(
		2026, 9, 29,
		10, 0, 0, 0,
		time.Local,
	)

	reservations, err := ReserveBudget(
		BudgetScopeToken,
		10,
		70,
		now,
	)
	require.NoError(t, err)
	require.Len(t, reservations, 1)

	require.NoError(t, ReleaseBudget(reservations))

	periodStart, err := GetBudgetPeriodStart(
		BudgetPeriodDaily,
		now,
	)
	require.NoError(t, err)

	used, err := GetBudgetUsage(
		reservations[0].RuleID,
		periodStart,
	)
	require.NoError(t, err)

	assert.EqualValues(t, 0, used)
}

func TestSettleBudgetAdjustsToActualQuota(t *testing.T) {
	setupBudgetTestDB(t)

	createBudgetRule(
		t,
		BudgetScopeUser,
		1,
		BudgetPeriodDaily,
		200,
	)

	now := time.Date(
		2026, 9, 29,
		10, 0, 0, 0,
		time.Local,
	)

	reservations, err := ReserveBudget(
		BudgetScopeUser,
		1,
		100,
		now,
	)
	require.NoError(t, err)
	require.Len(t, reservations, 1)

	// estimated 100, actual 60
	require.NoError(t, SettleBudget(
		reservations,
		60,
	))

	periodStart, err := GetBudgetPeriodStart(
		BudgetPeriodDaily,
		now,
	)
	require.NoError(t, err)

	used, err := GetBudgetUsage(
		reservations[0].RuleID,
		periodStart,
	)
	require.NoError(t, err)

	assert.EqualValues(t, 60, used)
}

func TestSettleBudgetIncreasesWhenActualExceedsReservation(t *testing.T) {
	setupBudgetTestDB(t)

	createBudgetRule(
		t,
		BudgetScopeUser,
		1,
		BudgetPeriodMonthly,
		200,
	)

	now := time.Date(
		2026, 9, 29,
		10, 0, 0, 0,
		time.Local,
	)

	reservations, err := ReserveBudget(
		BudgetScopeUser,
		1,
		50,
		now,
	)
	require.NoError(t, err)

	require.NoError(t, SettleBudget(
		reservations,
		80,
	))

	periodStart, err := GetBudgetPeriodStart(
		BudgetPeriodMonthly,
		now,
	)
	require.NoError(t, err)

	used, err := GetBudgetUsage(
		reservations[0].RuleID,
		periodStart,
	)
	require.NoError(t, err)

	assert.EqualValues(t, 80, used)
}

func TestDailyAndMonthlyBudgetsBothReserve(t *testing.T) {
	setupBudgetTestDB(t)

	createBudgetRule(
		t,
		BudgetScopeUser,
		1,
		BudgetPeriodDaily,
		100,
	)

	createBudgetRule(
		t,
		BudgetScopeUser,
		1,
		BudgetPeriodMonthly,
		1000,
	)

	now := time.Date(
		2026, 9, 29,
		10, 0, 0, 0,
		time.Local,
	)

	reservations, err := ReserveBudget(
		BudgetScopeUser,
		1,
		40,
		now,
	)

	require.NoError(t, err)
	assert.Len(t, reservations, 2)

	for _, reservation := range reservations {
		used, err := GetBudgetUsage(
			reservation.RuleID,
			reservation.PeriodStart,
		)
		require.NoError(t, err)

		assert.EqualValues(t, 40, used)
	}
}
