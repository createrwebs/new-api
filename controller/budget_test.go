package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type adminBudgetTestResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestAdminBudgetRules(t *testing.T) {
	for _, tc := range []struct{ kind, env string }{
		{"sqlite", ""}, {"mysql", "TEST_FIXED_MYSQL_DSN"}, {"postgres", "TEST_FIXED_POSTGRES_DSN"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			if tc.env != "" && os.Getenv(tc.env) == "" {
				t.Skip(tc.env + " is not configured")
			}
			testAdminBudgetRules(t, tc.kind, os.Getenv(tc.env))
		})
	}
}

func testAdminBudgetRules(t *testing.T, kind, dsn string) {
	var db *gorm.DB
	if kind == "sqlite" {
		setupAccessTokenAudit(t)
		db = model.DB
	} else {
		db, _ = newAuditTestDatabase(t, kind, dsn)
		previousDB, previousLogDB := model.DB, model.LOG_DB
		previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
		previousRedis, previousMaster := common.RedisEnabled, common.IsMasterNode
		model.DB, model.LOG_DB = db, db
		common.RedisEnabled, common.IsMasterNode = false, true
		if kind == "mysql" {
			common.SetDatabaseTypes(common.DatabaseTypeMySQL, common.DatabaseTypeMySQL)
		} else {
			common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypePostgreSQL)
		}
		t.Cleanup(func() {
			model.DB, model.LOG_DB = previousDB, previousLogDB
			common.SetDatabaseTypes(previousMain, previousLog)
			common.RedisEnabled, common.IsMasterNode = previousRedis, previousMaster
		})
		require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Log{}, &model.AuditLog{}, &model.CasbinRule{}, &model.AuthzRole{}))
		require.NoError(t, authz.Init(db))
	}
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.BudgetRule{}, &model.BudgetUsage{}))
	var version string
	query := "select version()"
	if kind == "sqlite" {
		query = "select sqlite_version()"
	}
	require.NoError(t, db.Raw(query).Scan(&version).Error)
	t.Logf("database: %s", version)
	credential := "budget-admin-credential"
	admin := model.User{Username: "budget-admin", AffCode: "budget-admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AccessToken: &credential}
	require.NoError(t, db.Create(&admin).Error)
	user := model.User{Username: "budget-user", AffCode: "budget-user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "secret-budget-token", Status: common.TokenStatusEnabled}
	require.NoError(t, db.Create(&token).Error)
	memberCredential := "budget-member-credential"
	require.NoError(t, db.Model(&user).Update("access_token", memberCredential).Error)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	route := router.Group("/api/budget")
	route.Use(middleware.AdminAuth())
	route.GET("/rules", AdminListBudgetRules)
	route.POST("/rules", AdminCreateBudgetRule)
	route.PUT("/rules/:id", AdminUpdateBudgetRule)
	route.DELETE("/rules/:id", AdminDeleteBudgetRule)
	route.GET("/usage", AdminGetBudgetUsage)
	request := func(method, path, body, auth string) (int, adminBudgetTestResponse) {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+auth)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		var response adminBudgetTestResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.NotContains(t, recorder.Body.String(), token.Key)
		return recorder.Code, response
	}
	create := fmt.Sprintf(`{"scope_type":"user","scope_id":%d,"period":"daily","limit_quota":100}`, user.Id)
	code, response := request(http.MethodPost, "/api/budget/rules", create, credential)
	require.Equal(t, http.StatusOK, code)
	require.True(t, response.Success, response.Message)
	var created budgetRuleResponse
	require.NoError(t, common.Unmarshal(response.Data, &created))
	assert.Equal(t, int64(100), created.LimitQuota)
	var rule model.BudgetRule
	require.NoError(t, db.Where("scope_type = ? AND scope_id = ?", model.BudgetScopeUser, user.Id).First(&rule).Error)
	assert.True(t, rule.Enabled)
	if kind != "sqlite" {
		// Concurrent administrators must not create two copies of a new rule.
		start := make(chan struct{})
		results := make(chan error, 2)
		var workers sync.WaitGroup
		for range 2 {
			workers.Go(func() {
				<-start
				results <- model.SaveBudgetRule(&model.BudgetRule{ScopeType: model.BudgetScopeUser, ScopeID: int64(user.Id), Period: model.BudgetPeriodMonthly, LimitQuota: 80, Enabled: true})
			})
		}
		close(start)
		workers.Wait()
		close(results)
		created, duplicates := 0, 0
		for err := range results {
			switch {
			case err == nil:
				created++
			case errors.Is(err, model.ErrBudgetRuleDuplicate):
				duplicates++
			default:
				t.Fatalf("concurrent rule creation: %v", err)
			}
		}
		assert.Equal(t, 1, created)
		assert.Equal(t, 1, duplicates)
		var count int64
		require.NoError(t, db.Model(&model.BudgetRule{}).Where("scope_type = ? AND scope_id = ? AND period = ?", model.BudgetScopeUser, user.Id, model.BudgetPeriodMonthly).Count(&count).Error)
		assert.Equal(t, int64(1), count)
	}
	for _, tc := range []struct{ name, body, message string }{
		{"duplicate", create, "already exists"},
		{"invalid scope", fmt.Sprintf(`{"scope_type":"team","scope_id":%d,"period":"daily","limit_quota":100}`, user.Id), "scope_type"},
		{"invalid period", fmt.Sprintf(`{"scope_type":"user","scope_id":%d,"period":"weekly","limit_quota":100}`, user.Id), "period"},
		{"invalid limit", fmt.Sprintf(`{"scope_type":"user","scope_id":%d,"period":"monthly","limit_quota":0}`, user.Id), "limit_quota"},
		{"invalid id", `{"scope_type":"user","scope_id":0,"period":"daily","limit_quota":100}`, "scope_id"},
		{"missing user", `{"scope_type":"user","scope_id":999999,"period":"daily","limit_quota":100}`, "not found"},
		{"missing token", `{"scope_type":"token","scope_id":999999,"period":"daily","limit_quota":100}`, "not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rejected := request(http.MethodPost, "/api/budget/rules", tc.body, credential)
			assert.False(t, rejected.Success)
			assert.Contains(t, rejected.Message, tc.message)
		})
	}
	_, response = request(http.MethodPost, "/api/budget/rules", fmt.Sprintf(`{"scope_type":"token","scope_id":%d,"period":"monthly","limit_quota":90}`, token.Id), credential)
	require.True(t, response.Success, response.Message)
	_, response = request(http.MethodPost, "/api/budget/rules", fmt.Sprintf(`{"scope_type":"token","scope_id":%d,"period":"daily","limit_quota":20,"enabled":false}`, token.Id), credential)
	require.True(t, response.Success, response.Message)
	var disabled budgetRuleResponse
	require.NoError(t, common.Unmarshal(response.Data, &disabled))
	assert.False(t, disabled.Enabled)
	var savedDisabled model.BudgetRule
	require.NoError(t, db.First(&savedDisabled, disabled.ID).Error)
	assert.False(t, savedDisabled.Enabled)
	_, response = request(http.MethodPost, "/api/budget/rules", fmt.Sprintf(`{"scope_type":"token","scope_id":%d,"period":"daily","limit_quota":20}`, token.Id), credential)
	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "already exists")
	period, err := model.GetBudgetPeriodStart(model.BudgetPeriodDaily, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.BudgetUsage{BudgetRuleID: rule.ID, PeriodStart: period, UsedQuota: 40}).Error)
	_, response = request(http.MethodGet, "/api/budget/rules", "", credential)
	require.True(t, response.Success, response.Message)
	var listed []budgetRuleResponse
	require.NoError(t, common.Unmarshal(response.Data, &listed))
	if kind == "sqlite" {
		require.Len(t, listed, 3)
	} else {
		require.Len(t, listed, 4)
	}
	for _, item := range listed {
		if item.ID == rule.ID {
			assert.Equal(t, int64(40), item.UsedQuota)
			assert.Equal(t, int64(60), item.RemainingQuota)
			assert.True(t, item.PeriodStart.Equal(period))
		} else if item.ID == disabled.ID {
			assert.False(t, item.Enabled)
			assert.Equal(t, int64(20), item.RemainingQuota)
		} else if item.ScopeType == model.BudgetScopeUser {
			assert.Equal(t, int64(80), item.RemainingQuota)
		} else {
			assert.Equal(t, int64(0), item.UsedQuota)
			assert.Equal(t, int64(90), item.RemainingQuota)
		}
	}
	_, response = request(http.MethodGet, "/api/budget/usage", "", credential)
	require.True(t, response.Success, response.Message)
	var usage []budgetRuleResponse
	require.NoError(t, common.Unmarshal(response.Data, &usage))
	assert.Len(t, usage, len(listed))
	for _, item := range usage {
		if item.ID == rule.ID {
			assert.Equal(t, period, item.PeriodStart)
			assert.Equal(t, int64(40), item.UsedQuota)
			assert.Equal(t, int64(100), item.LimitQuota)
			assert.Equal(t, int64(60), item.RemainingQuota)
		}
	}
	_, response = request(http.MethodPut, fmt.Sprintf("/api/budget/rules/%d", rule.ID), fmt.Sprintf(`{"scope_type":"user","scope_id":%d,"period":"daily","limit_quota":30,"enabled":false}`, user.Id), credential)
	require.True(t, response.Success, response.Message)
	require.NoError(t, db.First(&rule, rule.ID).Error)
	assert.Equal(t, int64(30), rule.LimitQuota)
	assert.False(t, rule.Enabled)
	_, response = request(http.MethodGet, "/api/budget/usage", "", credential)
	require.True(t, response.Success, response.Message)
	require.NoError(t, common.Unmarshal(response.Data, &usage))
	for _, item := range usage {
		if item.ID == rule.ID {
			assert.Equal(t, int64(0), item.RemainingQuota)
		}
	}
	_, response = request(http.MethodPut, fmt.Sprintf("/api/budget/rules/%d", rule.ID), fmt.Sprintf(`{"scope_type":"token","scope_id":%d,"period":"daily","limit_quota":30}`, token.Id), credential)
	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "cannot be changed")
	_, response = request(http.MethodPut, fmt.Sprintf("/api/budget/rules/%d", disabled.ID), fmt.Sprintf(`{"scope_type":"token","scope_id":%d,"period":"monthly","limit_quota":30}`, token.Id), credential)
	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "cannot be changed")
	_, response = request(http.MethodPut, "/api/budget/rules/999999", create, credential)
	assert.False(t, response.Success)
	_, response = request(http.MethodDelete, "/api/budget/rules/999999", "", credential)
	assert.False(t, response.Success)
	_, response = request(http.MethodDelete, fmt.Sprintf("/api/budget/rules/%d", rule.ID), "", credential)
	require.True(t, response.Success, response.Message)
	_, response = request(http.MethodDelete, fmt.Sprintf("/api/budget/rules/%d", rule.ID), "", credential)
	require.True(t, response.Success, response.Message)
	require.NoError(t, db.First(&rule, rule.ID).Error)
	assert.False(t, rule.Enabled)
	var count int64
	require.NoError(t, db.Model(&model.BudgetUsage{}).Where("budget_rule_id = ?", rule.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	_, response = request(http.MethodPut, fmt.Sprintf("/api/budget/rules/%d", rule.ID), fmt.Sprintf(`{"scope_type":"user","scope_id":%d,"period":"daily","limit_quota":60,"enabled":true}`, user.Id), credential)
	require.True(t, response.Success, response.Message)
	require.NoError(t, db.First(&rule, rule.ID).Error)
	assert.True(t, rule.Enabled)
	assert.Equal(t, int64(60), rule.LimitQuota)
	_, response = request(http.MethodDelete, fmt.Sprintf("/api/budget/rules/%d", rule.ID), "", credential)
	require.True(t, response.Success, response.Message)
	for _, endpoint := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/budget/rules", ""},
		{http.MethodGet, "/api/budget/usage", ""},
		{http.MethodPost, "/api/budget/rules", create},
		{http.MethodPut, fmt.Sprintf("/api/budget/rules/%d", rule.ID), create},
		{http.MethodDelete, fmt.Sprintf("/api/budget/rules/%d", rule.ID), ""},
	} {
		status, _ := request(endpoint.method, endpoint.path, endpoint.body, memberCredential)
		assert.Equal(t, http.StatusForbidden, status, endpoint.path)
	}
}
