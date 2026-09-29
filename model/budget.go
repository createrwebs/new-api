package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	BudgetScopeUser  = "user"
	BudgetScopeToken = "token"

	BudgetPeriodDaily   = "daily"
	BudgetPeriodMonthly = "monthly"
)

type BudgetRule struct {
	ID         int64  `gorm:"primaryKey"`
	ScopeType  string `gorm:"size:32;not null;index:idx_budget_rule_scope,priority:1"`
	ScopeID    int64  `gorm:"not null;index:idx_budget_rule_scope,priority:2"`
	Period     string `gorm:"size:16;not null;index:idx_budget_rule_scope,priority:3"`
	LimitQuota int64  `gorm:"not null"`
	Enabled    bool   `gorm:"not null;default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type BudgetUsage struct {
	ID           int64     `gorm:"primaryKey"`
	BudgetRuleID int64     `gorm:"not null;uniqueIndex:idx_budget_usage_period,priority:1"`
	PeriodStart  time.Time `gorm:"not null;uniqueIndex:idx_budget_usage_period,priority:2"`
	UsedQuota    int64     `gorm:"not null;default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
type BudgetReservation struct {
	RuleID      int64
	PeriodStart time.Time
	Quota       int64
}

func GetEnabledBudgetRules(scopeType string, scopeID int64) ([]BudgetRule, error) {
	var rules []BudgetRule

	err := DB.
		Where("scope_type = ? AND scope_id = ? AND enabled = ?", scopeType, scopeID, true).
		Find(&rules).Error

	return rules, err
}

func GetBudgetPeriodStart(period string, now time.Time) (time.Time, error) {
	now = now.In(time.Local)

	switch period {
	case BudgetPeriodDaily:
		return time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			0, 0, 0, 0,
			now.Location(),
		), nil

	case BudgetPeriodMonthly:
		return time.Date(
			now.Year(),
			now.Month(),
			1,
			0, 0, 0, 0,
			now.Location(),
		), nil

	default:
		return time.Time{}, fmt.Errorf("unsupported budget period: %s", period)
	}
}

func GetBudgetUsage(ruleID int64, periodStart time.Time) (int64, error) {
	var usage BudgetUsage

	err := DB.
		Where("budget_rule_id = ? AND period_start = ?", ruleID, periodStart).
		First(&usage).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}

	return usage.UsedQuota, nil
}
func CheckBudget(scopeType string, scopeID int64, requestedQuota int64, now time.Time) error {
	rules, err := GetEnabledBudgetRules(scopeType, scopeID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		periodStart, err := GetBudgetPeriodStart(rule.Period, now)
		if err != nil {
			return err
		}

		usedQuota, err := GetBudgetUsage(rule.ID, periodStart)
		if err != nil {
			return err
		}

		if usedQuota+requestedQuota > rule.LimitQuota {
			return fmt.Errorf(
				"%s budget exceeded: used=%d requested=%d limit=%d period=%s",
				scopeType,
				usedQuota,
				requestedQuota,
				rule.LimitQuota,
				rule.Period,
			)
		}
	}

	return nil
}
func AddBudgetUsage(scopeType string, scopeID int64, quota int64, now time.Time) error {
	if quota <= 0 {
		return nil
	}

	rules, err := GetEnabledBudgetRules(scopeType, scopeID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		periodStart, err := GetBudgetPeriodStart(rule.Period, now)
		if err != nil {
			return err
		}

		usage := BudgetUsage{
			BudgetRuleID: rule.ID,
			PeriodStart:  periodStart,
			UsedQuota:    quota,
		}

		err = DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "budget_rule_id"},
				{Name: "period_start"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"used_quota": gorm.Expr("used_quota + ?", quota),
				"updated_at": now,
			}),
		}).Create(&usage).Error

		if err != nil {
			return err
		}
	}

	return nil
}
func ReserveBudget(
	scopeType string,
	scopeID int64,
	requestedQuota int64,
	now time.Time,
) ([]BudgetReservation, error) {
	if requestedQuota <= 0 {
		return nil, nil
	}

	rules, err := GetEnabledBudgetRules(scopeType, scopeID)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, nil
	}

	reservations := make([]BudgetReservation, 0, len(rules))

	err = DB.Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			periodStart, err := GetBudgetPeriodStart(rule.Period, now)
			if err != nil {
				return err
			}

			usage := BudgetUsage{
				BudgetRuleID: rule.ID,
				PeriodStart:  periodStart,
				UsedQuota:    0,
			}

			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "budget_rule_id"},
					{Name: "period_start"},
				},
				DoNothing: true,
			}).Create(&usage).Error; err != nil {
				return err
			}

			var current BudgetUsage
			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where(
					"budget_rule_id = ? AND period_start = ?",
					rule.ID,
					periodStart,
				).
				First(&current).Error; err != nil {
				return err
			}

			if current.UsedQuota+requestedQuota > rule.LimitQuota {
				return fmt.Errorf(
					"%s budget exceeded: used=%d requested=%d limit=%d period=%s",
					scopeType,
					current.UsedQuota,
					requestedQuota,
					rule.LimitQuota,
					rule.Period,
				)
			}

			if err := tx.Model(&BudgetUsage{}).
				Where("id = ?", current.ID).
				UpdateColumn(
					"used_quota",
					gorm.Expr("used_quota + ?", requestedQuota),
				).Error; err != nil {
				return err
			}

			reservations = append(reservations, BudgetReservation{
				RuleID:      rule.ID,
				PeriodStart: periodStart,
				Quota:       requestedQuota,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return reservations, nil
}
func ReleaseBudget(reservations []BudgetReservation) error {
	if len(reservations) == 0 {
		return nil
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, r := range reservations {
			if r.Quota <= 0 {
				continue
			}

			if err := tx.Model(&BudgetUsage{}).
				Where(
					"budget_rule_id = ? AND period_start = ?",
					r.RuleID,
					r.PeriodStart,
				).
				UpdateColumn(
					"used_quota",
					gorm.Expr(
						"CASE WHEN used_quota >= ? THEN used_quota - ? ELSE 0 END",
						r.Quota,
						r.Quota,
					),
				).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
func SettleBudget(
	reservations []BudgetReservation,
	actualQuota int64,
) error {
	if len(reservations) == 0 {
		return nil
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		for _, r := range reservations {
			delta := actualQuota - r.Quota

			if delta == 0 {
				continue
			}

			if delta > 0 {
				if err := tx.Model(&BudgetUsage{}).
					Where(
						"budget_rule_id = ? AND period_start = ?",
						r.RuleID,
						r.PeriodStart,
					).
					UpdateColumn(
						"used_quota",
						gorm.Expr("used_quota + ?", delta),
					).Error; err != nil {
					return err
				}

				continue
			}

			refund := -delta

			if err := tx.Model(&BudgetUsage{}).
				Where(
					"budget_rule_id = ? AND period_start = ?",
					r.RuleID,
					r.PeriodStart,
				).
				UpdateColumn(
					"used_quota",
					gorm.Expr(
						"CASE WHEN used_quota >= ? THEN used_quota - ? ELSE 0 END",
						refund,
						refund,
					),
				).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
