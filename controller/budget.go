package controller

import (
	"errors"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type budgetRuleRequest struct {
	ScopeType  string `json:"scope_type"`
	ScopeID    int64  `json:"scope_id"`
	Period     string `json:"period"`
	LimitQuota int64  `json:"limit_quota"`
	Enabled    *bool  `json:"enabled"`
}

type budgetRuleResponse struct {
	ID             int64     `json:"id"`
	ScopeType      string    `json:"scope_type"`
	ScopeID        int64     `json:"scope_id"`
	Period         string    `json:"period"`
	LimitQuota     int64     `json:"limit_quota"`
	Enabled        bool      `json:"enabled"`
	PeriodStart    time.Time `json:"period_start"`
	UsedQuota      int64     `json:"used_quota"`
	RemainingQuota int64     `json:"remaining_quota"`
}

func budgetRuleView(rule model.BudgetRule, now time.Time) (budgetRuleResponse, error) {
	start, err := model.GetBudgetPeriodStart(rule.Period, now)
	if err != nil {
		return budgetRuleResponse{}, err
	}
	used, err := model.GetBudgetUsage(rule.ID, start)
	if err != nil {
		return budgetRuleResponse{}, err
	}
	return budgetRuleResponse{ID: rule.ID, ScopeType: rule.ScopeType, ScopeID: rule.ScopeID,
		Period: rule.Period, LimitQuota: rule.LimitQuota, Enabled: rule.Enabled,
		PeriodStart: start, UsedQuota: used, RemainingQuota: max(0, rule.LimitQuota-used)}, nil
}

func AdminListBudgetRules(c *gin.Context) {
	var rules []model.BudgetRule
	if err := model.DB.Order("id desc").Find(&rules).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]budgetRuleResponse, 0, len(rules))
	now := time.Now()
	for _, rule := range rules {
		item, err := budgetRuleView(rule, now)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		result = append(result, item)
	}
	common.ApiSuccess(c, result)
}

func AdminGetBudgetUsage(c *gin.Context) {
	AdminListBudgetRules(c)
}

func saveAdminBudgetRule(c *gin.Context, id int64) {
	var req budgetRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid budget rule request")
		return
	}
	if req.ScopeType != model.BudgetScopeUser && req.ScopeType != model.BudgetScopeToken {
		common.ApiErrorMsg(c, "scope_type must be user or token")
		return
	}
	if req.ScopeID <= 0 {
		common.ApiErrorMsg(c, "scope_id must be positive")
		return
	}
	if req.Period != model.BudgetPeriodDaily && req.Period != model.BudgetPeriodMonthly {
		common.ApiErrorMsg(c, "period must be daily or monthly")
		return
	}
	if req.LimitQuota <= 0 {
		common.ApiErrorMsg(c, "limit_quota must be positive")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := model.BudgetRule{ID: id, ScopeType: req.ScopeType, ScopeID: req.ScopeID, Period: req.Period, LimitQuota: req.LimitQuota, Enabled: enabled}
	if err := model.SaveBudgetRule(&rule); err != nil {
		switch {
		case errors.Is(err, model.ErrBudgetRuleDuplicate), errors.Is(err, model.ErrBudgetScopeNotFound), errors.Is(err, model.ErrBudgetRuleImmutable), errors.Is(err, gorm.ErrRecordNotFound):
			common.ApiError(c, err)
		default:
			common.SysLog("failed to save budget rule: " + err.Error())
			common.ApiErrorMsg(c, "failed to save budget rule")
		}
		return
	}
	view, err := budgetRuleView(rule, time.Now())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, view)
}

func AdminCreateBudgetRule(c *gin.Context) {
	saveAdminBudgetRule(c, 0)
}

func AdminUpdateBudgetRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid budget rule id")
		return
	}
	saveAdminBudgetRule(c, id)
}

// Deletion disables a rule rather than discarding its usage history or
// interfering with reservations that are still being settled or released.
func AdminDeleteBudgetRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid budget rule id")
		return
	}
	result := model.DB.Model(&model.BudgetRule{}).Where("id = ?", id).Update("enabled", false)
	if result.Error != nil {
		common.ApiError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		var rule model.BudgetRule
		if err := model.DB.First(&rule, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "budget rule not found")
			} else {
				common.ApiError(c, err)
			}
			return
		}
	}
	common.ApiSuccess(c, nil)
}
