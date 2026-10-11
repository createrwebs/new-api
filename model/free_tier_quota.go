package model

import (
	"errors"
	"time"
	_ "time/tzdata"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FreeTierDailyLimit = int64(20)
	FreeTierTimeZone   = "Asia/Bangkok"
)

var ErrFreeTierQuotaExceeded = errors.New("free tier daily request limit exceeded")

// FreeTierDailyUsage stores one durable request counter per user and Bangkok
// calendar day. The composite unique index is the quota identity boundary.
type FreeTierDailyUsage struct {
	ID          int64  `gorm:"primaryKey"`
	UserID      int    `gorm:"not null;uniqueIndex:idx_free_tier_daily_user_day,priority:1"`
	CalendarDay string `gorm:"type:varchar(10);not null;uniqueIndex:idx_free_tier_daily_user_day,priority:2"`
	Used        int64  `gorm:"not null;default:0"`
	CreatedAt   int64  `gorm:"autoCreateTime"`
	UpdatedAt   int64  `gorm:"autoUpdateTime"`
}

func FreeTierCalendarDay(now time.Time) (string, error) {
	location, err := time.LoadLocation(FreeTierTimeZone)
	if err != nil {
		return "", err
	}
	return now.In(location).Format("2006-01-02"), nil
}

func IsFreeTierUser(userID int) (bool, error) {
	if userID <= 0 {
		return false, errors.New("invalid user id")
	}
	user, err := GetUserById(userID, false)
	if err != nil {
		return false, err
	}
	if user.Role >= common.RoleAdminUser {
		return false, nil
	}
	if user.Quota > 0 {
		return false, nil
	}
	var count int64
	now := common.GetTimestamp()
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND end_time > ?", userID, "active", now).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// ReserveFreeTierRequest atomically admits one request for a user/day.
func ReserveFreeTierRequest(userID int, now time.Time) (bool, error) {
	if userID <= 0 {
		return false, errors.New("invalid user id")
	}
	day, err := FreeTierCalendarDay(now)
	if err != nil {
		return false, err
	}
	usage := &FreeTierDailyUsage{UserID: userID, CalendarDay: day}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(usage).Error; err != nil {
		return false, err
	}
	result := DB.Model(&FreeTierDailyUsage{}).
		Where("user_id = ? AND calendar_day = ? AND used < ?", userID, day, FreeTierDailyLimit).
		UpdateColumn("used", gorm.Expr("used + ?", 1))
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}
