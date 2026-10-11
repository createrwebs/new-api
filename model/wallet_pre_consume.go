package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

var ErrWalletQuotaInsufficient = errors.New("wallet quota insufficient")

// WalletPreConsumeRecord tracks durable pending wallet reservations to prevent
// unrecoverable quota loss on unexpected server crashes.
type WalletPreConsumeRecord struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RequestId   string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	UserId      int    `json:"user_id" gorm:"index"`
	PreConsumed int    `json:"pre_consumed" gorm:"type:bigint;not null;default:0"`
	Status      string `json:"status" gorm:"type:varchar(32);index"` // pending, settled, refunded
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;index"`
}

func (r *WalletPreConsumeRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *WalletPreConsumeRecord) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func ensureWalletPreConsumeTable(db *gorm.DB) {
	if db == nil {
		return
	}
	if !db.Migrator().HasTable(&WalletPreConsumeRecord{}) {
		_ = db.AutoMigrate(&WalletPreConsumeRecord{})
	}
}

// PreConsumeUserWallet atomically reserves quota and records a durable pending record.
func PreConsumeUserWallet(requestId string, userId int, amount int) error {
	if amount <= 0 {
		return nil
	}
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is required")
	}
	ensureWalletPreConsumeTable(DB)
	reserved, err := TryReserveUserQuota(userId, amount)
	if err != nil {
		return err
	}
	if !reserved {
		return ErrWalletQuotaInsufficient
	}

	var existing []WalletPreConsumeRecord
	if err := DB.Where("request_id = ? AND status = ?", requestId, "pending").Limit(1).Find(&existing).Error; err == nil && len(existing) > 0 {
		if updateErr := DB.Model(&existing[0]).Update("pre_consumed", gorm.Expr("pre_consumed + ?", amount)).Error; updateErr != nil {
			_ = IncreaseUserQuota(userId, amount, false)
			return updateErr
		}
		return nil
	}

	record := &WalletPreConsumeRecord{
		RequestId:   requestId,
		UserId:      userId,
		PreConsumed: amount,
		Status:      "pending",
		CreatedAt:   common.GetTimestamp(),
		UpdatedAt:   common.GetTimestamp(),
	}
	if err := DB.Create(record).Error; err != nil {
		// Compensate reserved quota if recording fails
		_ = IncreaseUserQuota(userId, amount, false)
		return err
	}
	return nil
}

// SettleUserWalletPreConsume transitions a pending reservation to settled.
func SettleUserWalletPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return nil
	}
	ensureWalletPreConsumeTable(DB)
	return DB.Model(&WalletPreConsumeRecord{}).
		Where("request_id = ? AND status = ?", requestId, "pending").
		Update("status", "settled").Error
}

// RefundUserWalletPreConsume idempotently refunds a pending reservation.
func RefundUserWalletPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	ensureWalletPreConsumeTable(DB)
	var userId int
	var preConsumed int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var records []WalletPreConsumeRecord
		if err := lockForUpdate(tx).
			Where("request_id = ?", requestId).Limit(1).Find(&records).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		record := records[0]
		if record.Status != "pending" {
			return nil
		}
		record.Status = "refunded"
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
		if record.PreConsumed > 0 {
			if err := IncreaseUserQuotaTx(tx, record.UserId, record.PreConsumed); err != nil {
				return err
			}
		}
		userId = record.UserId
		preConsumed = record.PreConsumed
		return nil
	})
	if err != nil {
		return err
	}
	if preConsumed > 0 {
		gopool.Go(func() {
			if err := cacheIncrUserQuota(userId, int64(preConsumed)); err != nil {
				common.SysLog("failed to increase user quota cache on refund: " + err.Error())
			}
		})
	}
	return nil
}

// ReconcileOrphanedWalletPreConsumes finds abandoned pending reservations after a crash/timeout
// and refunds them back to the respective user wallets.
func ReconcileOrphanedWalletPreConsumes(olderThanSeconds int64) (int, error) {
	if olderThanSeconds <= 0 {
		olderThanSeconds = 600 // 10 minutes default
	}
	cutoff := common.GetTimestamp() - olderThanSeconds
	var records []WalletPreConsumeRecord
	if err := DB.Where("status = ? AND created_at < ?", "pending", cutoff).
		Limit(100).
		Find(&records).Error; err != nil {
		return 0, err
	}
	reconciled := 0
	for _, rec := range records {
		if err := RefundUserWalletPreConsume(rec.RequestId); err == nil {
			reconciled++
		}
	}
	return reconciled, nil
}

// CleanupWalletPreConsumeRecords removes old settled/refunded records to keep table compact.
func CleanupWalletPreConsumeRecords(olderThanSeconds int64) (int64, error) {
	if olderThanSeconds <= 0 {
		olderThanSeconds = 7 * 24 * 3600
	}
	cutoff := common.GetTimestamp() - olderThanSeconds
	res := DB.Where("status IN ? AND updated_at < ?", []string{"settled", "refunded"}, cutoff).
		Delete(&WalletPreConsumeRecord{})
	return res.RowsAffected, res.Error
}
