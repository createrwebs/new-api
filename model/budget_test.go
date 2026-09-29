package model

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
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

func TestReserveBudgetDoesNotOverflowLimit(t *testing.T) {
	setupBudgetTestDB(t)
	rule := createBudgetRule(t, BudgetScopeUser, 1, BudgetPeriodDaily, math.MaxInt64)
	now := time.Now()
	first, err := ReserveBudget(BudgetScopeUser, 1, math.MaxInt64-1, now)
	require.NoError(t, err)
	require.Len(t, first, 1)
	_, err = ReserveBudget(BudgetScopeUser, 1, 2, now)
	require.ErrorContains(t, err, "budget exceeded")
	used, err := GetBudgetUsage(rule.ID, first[0].PeriodStart)
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64-1), used)
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

func TestBudgetConcurrencyPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_FIXED_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_FIXED_POSTGRES_DSN is not configured")
	}
	previousDB, previousType := DB, common.MainDatabaseType()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(12)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&BudgetRule{}, &BudgetUsage{}))
	var version string
	require.NoError(t, db.Raw("select version()").Scan(&version).Error)
	t.Logf("database: %s", version)

	for _, tc := range []struct {
		name          string
		initial, jobs int64
		noUsageRow    bool
		wantSuccesses int
		wantUsed      int64
	}{
		{name: "80 used, two compete", initial: 80, jobs: 2, wantSuccesses: 1, wantUsed: 95},
		{name: "zero used, ten compete", initial: 0, jobs: 10, wantSuccesses: 6, wantUsed: 90},
		{name: "no usage row, ten compete", noUsageRow: true, jobs: 10, wantSuccesses: 6, wantUsed: 90},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			rule := createBudgetRule(t, BudgetScopeUser, now.UnixNano(), BudgetPeriodDaily, 100)
			t.Cleanup(func() {
				require.NoError(t, db.Where("budget_rule_id = ?", rule.ID).Delete(&BudgetUsage{}).Error)
				require.NoError(t, db.Delete(&rule).Error)
			})
			periodStart, err := GetBudgetPeriodStart(rule.Period, now)
			require.NoError(t, err)
			// Seed the row for the row-lock cases; the final case also
			// exercises concurrent creation of a previously missing row.
			if !tc.noUsageRow {
				require.NoError(t, db.Create(&BudgetUsage{BudgetRuleID: rule.ID, PeriodStart: periodStart, UsedQuota: tc.initial}).Error)
			}
			type outcome struct {
				reservations []BudgetReservation
				err          error
			}
			results := make(chan outcome, tc.jobs)
			start := make(chan struct{})
			var workers sync.WaitGroup
			for range tc.jobs {
				workers.Go(func() {
					<-start
					reservations, err := ReserveBudget(BudgetScopeUser, rule.ScopeID, 15, now)
					results <- outcome{reservations, err}
				})
			}
			close(start)
			workers.Wait()
			close(results)
			successes, exceeded := 0, 0
			for result := range results {
				if result.err == nil {
					successes++
					require.Equal(t, []BudgetReservation{{RuleID: rule.ID, PeriodStart: periodStart, Quota: 15}}, result.reservations)
					continue
				}
				require.ErrorContains(t, result.err, "budget exceeded")
				require.Empty(t, result.reservations)
				exceeded++
			}
			assert.Equal(t, tc.wantSuccesses, successes)
			assert.Equal(t, int(tc.jobs)-tc.wantSuccesses, exceeded)
			used, err := GetBudgetUsage(rule.ID, periodStart)
			require.NoError(t, err)
			assert.Equal(t, tc.initial+int64(successes)*15, used)
			assert.Equal(t, tc.wantUsed, used)
			assert.LessOrEqual(t, used, int64(100))
			t.Logf("successful reservations: %d; rejected: %d; final used_quota: %d", successes, exceeded, used)
		})
	}
}
