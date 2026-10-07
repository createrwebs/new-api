package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	// DefaultInputAssetTTL defines the lifecycle for temporary input uploads (24 hours).
	DefaultInputAssetTTL = 24 * time.Hour

	// DefaultOutputAssetTTL defines the lifecycle for persistent output assets (30 days).
	DefaultOutputAssetTTL = 30 * 24 * time.Hour

	// MaxGlobalStudioStorageBytes defines the global storage ceiling (5 GB) to prevent disk exhaustion.
	MaxGlobalStudioStorageBytes = int64(5 * 1024 * 1024 * 1024)

	// MaxUserStudioStorageBytes defines per-user active storage ceiling (500 MB).
	MaxUserStudioStorageBytes = int64(500 * 1024 * 1024)
)

var (
	ErrGlobalStorageQuotaExceeded = errors.New("platform storage capacity limit reached; automatic cleanup in progress, please retry shortly")
	ErrUserStorageQuotaExceeded   = errors.New("user storage quota exceeded (500MB limit); please remove older assets before uploading new media")
	assetCleanupOnce              sync.Once
)

// GetStudioUploadDir returns the authoritative local storage directory for media assets.
func GetStudioUploadDir() string {
	if customDir := strings.TrimSpace(os.Getenv("STUDIO_UPLOAD_DIR")); customDir != "" {
		_ = os.MkdirAll(customDir, 0755)
		return customDir
	}
	defaultDir := "./data/upload/studio"
	_ = os.MkdirAll(defaultDir, 0755)
	return defaultDir
}

// CheckStorageQuota verifies that the incoming upload will not exhaust user or global disk limits.
func CheckStorageQuota(db *gorm.DB, userId int, incomingBytes int64) error {
	if db == nil {
		return nil
	}

	// 1. Check User-level storage quota
	var userTotalBytes int64
	err := db.Model(&model.StudioAsset{}).
		Where("user_id = ? AND availability_status = ?", userId, "available").
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&userTotalBytes).Error
	if err == nil {
		if userTotalBytes+incomingBytes > MaxUserStudioStorageBytes {
			return ErrUserStorageQuotaExceeded
		}
	}

	// 2. Check Global storage quota across all active studio assets
	var globalTotalBytes int64
	err = db.Model(&model.StudioAsset{}).
		Where("availability_status = ?", "available").
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&globalTotalBytes).Error
	if err == nil {
		if globalTotalBytes+incomingBytes > MaxGlobalStudioStorageBytes {
			return ErrGlobalStorageQuotaExceeded
		}
	}

	return nil
}

// CleanupExpiredStudioAssets purges expired files from disk and marks them EXPIRED in the ledger.
// It strictly preserves assets currently referenced by active jobs or history.
func CleanupExpiredStudioAssets(ctx context.Context, db *gorm.DB) (int, error) {
	if db == nil {
		return 0, nil
	}

	now := common.GetTimestamp()
	var candidates []model.StudioAsset
	err := db.Where("availability_status = ? AND expiry_at > 0 AND expiry_at <= ?", "available", now).
		Limit(100).
		Find(&candidates).Error
	if err != nil {
		return 0, err
	}

	if len(candidates) == 0 {
		return 0, nil
	}

	// Active job statuses where assets must NOT be deleted
	activeStatuses := []string{
		string(model.StudioJobStatusReserved),
		string(model.StudioJobStatusSubmitting),
		string(model.StudioJobStatusQueued),
		string(model.StudioJobStatusProcessing),
	}

	uploadDir := GetStudioUploadDir()
	deletedCount := 0

	for _, asset := range candidates {
		// Guard: Do not delete if referenced by an active/pending job
		var activeJobCount int64
		err := db.Model(&model.StudioToolJob{}).
			Where("(id = ? OR input_params LIKE ? OR output_result LIKE ?) AND status IN (?)",
				asset.JobId,
				"%"+asset.Id+"%",
				"%"+asset.Id+"%",
				activeStatuses,
			).Count(&activeJobCount).Error

		if err == nil && activeJobCount > 0 {
			// Asset is still needed by an in-flight job; extend TTL by 1 hour
			_ = db.Model(&asset).Update("expiry_at", now+3600)
			continue
		}

		// Remove file from disk
		filename := filepath.Base(asset.StorageURL)
		filePath := filepath.Join(uploadDir, filename)
		if _, statErr := os.Stat(filePath); statErr == nil {
			_ = os.Remove(filePath)
		}

		// Update database status to 'expired'
		_ = db.Model(&asset).Updates(map[string]interface{}{
			"availability_status": "expired",
			"updated_at":          now,
		})
		deletedCount++
	}

	return deletedCount, nil
}

// StartStudioAssetCleanupWorker runs a periodic background worker to clean up expired media.
func StartStudioAssetCleanupWorker(db *gorm.DB) {
	assetCleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				if db != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					cleaned, err := CleanupExpiredStudioAssets(ctx, db)
					if err != nil {
						common.SysError(fmt.Sprintf("[StudioAsset] cleanup error: %v", err))
					} else if cleaned > 0 {
						common.SysLog(fmt.Sprintf("[StudioAsset] cleaned up %d expired assets", cleaned))
					}
					cancel()
				}
			}
		}()
	})
}
