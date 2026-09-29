package service

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestBudgetBillingLifecycle(t *testing.T) {
	for _, dialect := range []struct {
		name common.DatabaseType
		env  string
	}{
		{common.DatabaseTypeSQLite, ""},
		{common.DatabaseTypeMySQL, "TEST_FIXED_MYSQL_DSN"},
		{common.DatabaseTypePostgreSQL, "TEST_FIXED_POSTGRES_DSN"},
	} {
		t.Run(string(dialect.name), func(t *testing.T) {
			var driver gorm.Dialector = sqlite.Open(":memory:")
			if dialect.env != "" {
				dsn := os.Getenv(dialect.env)
				if dsn == "" {
					t.Skip(dialect.env + " is not configured")
				}
				if dialect.name == common.DatabaseTypeMySQL {
					driver = mysql.Open(dsn)
				} else {
					driver = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
				}
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			oldDB, oldType := model.DB, common.MainDatabaseType()
			model.DB = db
			common.SetDatabaseTypes(dialect.name, common.LogDatabaseType())
			t.Cleanup(func() { model.DB = oldDB; common.SetDatabaseTypes(oldType, common.LogDatabaseType()) })
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.BudgetRule{}, &model.BudgetUsage{}))
			require.NoError(t, db.AutoMigrate(&model.BudgetRule{}, &model.BudgetUsage{}))
			versionQuery := "select version()"
			if dialect.name == common.DatabaseTypeSQLite {
				versionQuery = "select sqlite_version()"
			}
			var version string
			require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
			t.Logf("database: %s", version)

			for _, tc := range []struct {
				name                  string
				userLimit, tokenLimit int64
				wallet, tokenQuota    int
				playground            bool
				pre, actual           int
				refund, topUp         bool
				topUpError            bool
				wantError             types.ErrorCode
				wantUser, wantToken   int64
			}{
				{name: "both reserve and settle down once", userLimit: 200, tokenLimit: 200, wallet: 500, tokenQuota: 500, pre: 100, actual: 60, wantUser: 60, wantToken: 60},
				{name: "settle up", userLimit: 200, tokenLimit: 200, wallet: 500, tokenQuota: 500, pre: 50, actual: 80, wantUser: 80, wantToken: 80},
				{name: "equal settlement", userLimit: 200, tokenLimit: 200, wallet: 500, tokenQuota: 500, pre: 100, actual: 100, wantUser: 100, wantToken: 100},
				{name: "zero estimate settles actual", userLimit: 200, tokenLimit: 200, wallet: 500, tokenQuota: 500, pre: 0, actual: 40, wantUser: 40, wantToken: 40},
				{name: "refund once", userLimit: 200, tokenLimit: 200, wallet: 500, tokenQuota: 500, pre: 100, refund: true},
				{name: "token budget failure rolls back user", userLimit: 200, tokenLimit: 40, wallet: 500, tokenQuota: 500, pre: 100, wantError: types.ErrorCodeInsufficientUserQuota},
				{name: "session wallet failure rolls back budgets", userLimit: 200, tokenLimit: 200, wallet: 10, tokenQuota: 500, pre: 100, wantError: types.ErrorCodeInsufficientUserQuota},
				{name: "session token failure rolls back budgets", userLimit: 200, tokenLimit: 200, wallet: 500, tokenQuota: 10, pre: 100, wantError: types.ErrorCodePreConsumeTokenQuotaFailed},
				{name: "playground skips token budget", userLimit: 200, tokenLimit: 10, wallet: 500, tokenQuota: 0, playground: true, pre: 100, actual: 60, wantUser: 60},
				{name: "additional reservation settles once", userLimit: 200, tokenLimit: 200, wallet: 500, tokenQuota: 500, pre: 50, topUp: true, actual: 80, wantUser: 80, wantToken: 80},
				{name: "additional token budget failure rolls back top up", userLimit: 200, tokenLimit: 60, wallet: 500, tokenQuota: 500, pre: 50, topUp: true, topUpError: true, actual: 50, wantUser: 50, wantToken: 50},
				{name: "no rules keeps wallet and token behavior", wallet: 500, tokenQuota: 500, pre: 100, actual: 60},
			} {
				t.Run(tc.name, func(t *testing.T) {
					user := model.User{Username: fmt.Sprintf("budget_%s", tc.name), Quota: tc.wallet, Status: common.UserStatusEnabled}
					require.NoError(t, db.Create(&user).Error)
					token := model.Token{UserId: user.Id, Key: fmt.Sprintf("budget-%d", user.Id), Name: "budget", RemainQuota: tc.tokenQuota, Status: common.TokenStatusEnabled}
					require.NoError(t, db.Create(&token).Error)
					t.Cleanup(func() {
						require.NoError(t, db.Where("budget_rule_id IN (SELECT id FROM budget_rules WHERE scope_type = ? AND scope_id = ?) OR budget_rule_id IN (SELECT id FROM budget_rules WHERE scope_type = ? AND scope_id = ?)", model.BudgetScopeUser, user.Id, model.BudgetScopeToken, token.Id).Delete(&model.BudgetUsage{}).Error)
						require.NoError(t, db.Where("scope_type = ? AND scope_id = ? OR scope_type = ? AND scope_id = ?", model.BudgetScopeUser, user.Id, model.BudgetScopeToken, token.Id).Delete(&model.BudgetRule{}).Error)
						require.NoError(t, db.Unscoped().Delete(&token).Error)
						require.NoError(t, db.Unscoped().Delete(&user).Error)
					})
					var rules []model.BudgetRule
					for _, scope := range []struct {
						name  string
						id    int64
						limit int64
					}{
						{model.BudgetScopeUser, int64(user.Id), tc.userLimit},
						{model.BudgetScopeToken, int64(token.Id), tc.tokenLimit},
					} {
						if scope.limit == 0 {
							continue
						}
						rule := model.BudgetRule{ScopeType: scope.name, ScopeID: scope.id, Period: model.BudgetPeriodDaily, LimitQuota: scope.limit, Enabled: true}
						require.NoError(t, db.Create(&rule).Error)
						rules = append(rules, rule)
					}
					check := func(wants map[string]int64) {
						t.Helper()
						for _, rule := range rules {
							start, err := model.GetBudgetPeriodStart(rule.Period, time.Now())
							require.NoError(t, err)
							used, err := model.GetBudgetUsage(rule.ID, start)
							require.NoError(t, err)
							assert.Equal(t, wants[rule.ScopeType], used, rule.ScopeType)
						}
					}
					ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
					ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
					info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id, TokenKey: token.Key, IsPlayground: tc.playground, ForcePreConsume: true, UserSetting: dto.UserSetting{BillingPreference: "wallet_only"}}
					apiErr := PreConsumeBilling(ctx, tc.pre, info)
					if tc.wantError != "" {
						require.NotNil(t, apiErr)
						assert.Equal(t, tc.wantError, apiErr.GetErrorCode())
						assert.Nil(t, info.Billing)
						check(map[string]int64{})
						return
					}
					require.Nil(t, apiErr)
					require.NotNil(t, info.Billing)
					check(map[string]int64{model.BudgetScopeUser: int64(tc.pre), model.BudgetScopeToken: func() int64 {
						if tc.playground {
							return 0
						}
						return int64(tc.pre)
					}()})
					if tc.topUp {
						reserveErr := info.Billing.Reserve(100)
						if tc.topUpError {
							require.Error(t, reserveErr)
							assert.Equal(t, tc.pre, info.Billing.GetPreConsumedQuota())
							check(map[string]int64{model.BudgetScopeUser: int64(tc.pre), model.BudgetScopeToken: int64(tc.pre)})
						} else {
							require.NoError(t, reserveErr)
							check(map[string]int64{model.BudgetScopeUser: 100, model.BudgetScopeToken: 100})
						}
					}
					if tc.refund {
						finished := make(chan struct{}, 1)
						const callback = "budget_refund_observed"
						require.NoError(t, db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register(callback, func(tx *gorm.DB) {
							if tx.Statement.Table == "budget_usages" && tx.Error == nil {
								select {
								case finished <- struct{}{}:
								default:
								}
							}
						}))
						t.Cleanup(func() { require.NoError(t, db.Callback().Update().Remove(callback)) })
						info.Billing.Refund(ctx)
						info.Billing.Refund(ctx)
						select {
						case <-finished:
						case <-time.After(5 * time.Second):
							t.Fatal("budget refund did not complete")
						}
						check(map[string]int64{})
						return
					}
					require.NoError(t, info.Billing.Settle(tc.actual))
					require.NoError(t, info.Billing.Settle(tc.actual))
					check(map[string]int64{model.BudgetScopeUser: tc.wantUser, model.BudgetScopeToken: tc.wantToken})
					wallet, err := model.GetUserQuota(user.Id, true)
					require.NoError(t, err)
					assert.Equal(t, tc.wallet-tc.actual, wallet)
					var savedToken model.Token
					require.NoError(t, db.First(&savedToken, token.Id).Error)
					if tc.playground {
						assert.Equal(t, tc.tokenQuota, savedToken.RemainQuota)
					} else {
						assert.Equal(t, tc.tokenQuota-tc.actual, savedToken.RemainQuota)
					}
				})
			}
		})
	}
}

func TestBudgetTrustedUpstreamFailuresReleaseReservations(t *testing.T) {
	oldDB, oldType := model.DB, common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	quotaSettings := operation_setting.GetQuotaSetting()
	oldTrustQuota := quotaSettings.TrustQuotaUSD
	quotaSettings.TrustQuotaUSD = 10
	t.Cleanup(func() {
		model.DB = oldDB
		common.SetMainDatabaseType(oldType)
		quotaSettings.TrustQuotaUSD = oldTrustQuota
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.BudgetRule{}, &model.BudgetUsage{}))
	user := model.User{Username: "trusted-failure", Quota: 20_000_000, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "trusted-failure-token", RemainQuota: 20_000_000, UsedQuota: 29_545, Status: common.TokenStatusEnabled}
	require.NoError(t, db.Create(&token).Error)
	userRule := model.BudgetRule{ScopeType: model.BudgetScopeUser, ScopeID: int64(user.Id), Period: model.BudgetPeriodDaily, LimitQuota: 100_000, Enabled: true}
	tokenRule := model.BudgetRule{ScopeType: model.BudgetScopeToken, ScopeID: int64(token.Id), Period: model.BudgetPeriodDaily, LimitQuota: 100_000, Enabled: true}
	require.NoError(t, db.Create(&userRule).Error)
	require.NoError(t, db.Create(&tokenRule).Error)
	now := time.Now()
	period, err := model.GetBudgetPeriodStart(model.BudgetPeriodDaily, now)
	require.NoError(t, err)
	for _, rule := range []model.BudgetRule{userRule, tokenRule} {
		require.NoError(t, db.Create(&model.BudgetUsage{BudgetRuleID: rule.ID, PeriodStart: period, UsedQuota: 19_700}).Error)
	}
	for attempt := range 2 {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
		ctx.Set("token_quota", 20_000_000)
		info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id, TokenKey: token.Key, UserSetting: dto.UserSetting{BillingPreference: "wallet_only"}}
		require.Nil(t, PreConsumeBilling(ctx, 30, info))
		for _, rule := range []model.BudgetRule{userRule, tokenRule} {
			used, err := model.GetBudgetUsage(rule.ID, period)
			require.NoError(t, err)
			assert.Equal(t, int64(19_730), used)
		}
		require.Zero(t, info.Billing.GetPreConsumedQuota())
		assert.True(t, info.Billing.NeedsRefund())
		// The upstream rejects the request; refund is invoked through the
		// same BillingSession interface as the relay failure wrapper.
		info.Billing.Refund(ctx)
		info.Billing.Refund(ctx)
		require.Eventually(t, func() bool {
			for _, rule := range []model.BudgetRule{userRule, tokenRule} {
				used, err := model.GetBudgetUsage(rule.ID, period)
				if err != nil || used != 19_700 {
					return false
				}
			}
			return true
		}, 5*time.Second, 10*time.Millisecond, "failure %d left budget reserved", attempt+1)
		assert.False(t, info.Billing.NeedsRefund())
		require.NoError(t, info.Billing.Settle(30), "a refunded reservation must not settle")
		for _, rule := range []model.BudgetRule{userRule, tokenRule} {
			used, err := model.GetBudgetUsage(rule.ID, period)
			require.NoError(t, err)
			assert.Equal(t, int64(19_700), used, "failure %d", attempt+1)
		}
		var savedToken model.Token
		require.NoError(t, db.First(&savedToken, token.Id).Error)
		assert.Equal(t, 29_545, savedToken.UsedQuota)
		wallet, err := model.GetUserQuota(user.Id, true)
		require.NoError(t, err)
		assert.Equal(t, user.Quota, wallet)
	}
	// A trusted request which does succeed must reconcile its reservation only
	// once, even if the caller repeats settlement or later tries to refund.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	ctx.Set("token_quota", 20_000_000)
	info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id, TokenKey: token.Key, UserSetting: dto.UserSetting{BillingPreference: "wallet_only"}}
	require.Nil(t, PreConsumeBilling(ctx, 30, info))
	require.NoError(t, info.Billing.Settle(30))
	require.NoError(t, info.Billing.Settle(30))
	info.Billing.Refund(ctx)
	for _, rule := range []model.BudgetRule{userRule, tokenRule} {
		used, err := model.GetBudgetUsage(rule.ID, period)
		require.NoError(t, err)
		assert.Equal(t, int64(19_730), used)
	}
	wallet, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	assert.Equal(t, user.Quota-30, wallet)
	var savedToken model.Token
	require.NoError(t, db.First(&savedToken, token.Id).Error)
	assert.Equal(t, 29_575, savedToken.UsedQuota)
}
