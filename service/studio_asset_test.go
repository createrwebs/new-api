package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestAssetDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.StudioAsset{}, &model.StudioToolJob{})
	require.NoError(t, err)

	return db
}

func TestStudioAsset_StorageQuotaGuards(t *testing.T) {
	db := setupTestAssetDB(t)

	// User 1 has 490MB of stored assets
	asset1 := model.StudioAsset{
		Id:                 "asset_user1_large",
		UserId:             1,
		FileSize:           490 * 1024 * 1024,
		AvailabilityStatus: "available",
		StorageURL:         "/api/v1/studio/assets/test1.png",
		CreatedAt:          time.Now().Unix(),
	}
	require.NoError(t, db.Create(&asset1).Error)

	// Attempt to upload 20MB -> exceeds 500MB user quota
	err := CheckStorageQuota(db, 1, 20*1024*1024)
	assert.ErrorIs(t, err, ErrUserStorageQuotaExceeded, "Should reject upload exceeding user quota limit")

	// Attempt to upload 5MB -> within 500MB user quota
	err = CheckStorageQuota(db, 1, 5*1024*1024)
	assert.NoError(t, err, "Should allow upload within user quota limit")
}

func TestStudioAsset_CleanupExpiredAssets(t *testing.T) {
	db := setupTestAssetDB(t)
	tmpDir, err := os.MkdirTemp("", "studio_test_upload_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origDir := os.Getenv("STUDIO_UPLOAD_DIR")
	os.Setenv("STUDIO_UPLOAD_DIR", tmpDir)
	defer os.Setenv("STUDIO_UPLOAD_DIR", origDir)

	now := common.GetTimestamp()

	// 1. Create dummy file on disk for expired asset
	expiredFile := filepath.Join(tmpDir, "expired.png")
	require.NoError(t, os.WriteFile(expiredFile, []byte("fake-png-data"), 0644))

	expiredAsset := model.StudioAsset{
		Id:                 "asset_expired_1",
		UserId:             1,
		StorageURL:         "/api/v1/studio/assets/expired.png",
		FileSize:           13,
		AvailabilityStatus: "available",
		ExpiryAt:           now - 3600, // expired 1 hour ago
		CreatedAt:          now - 86400,
	}
	require.NoError(t, db.Create(&expiredAsset).Error)

	// 2. Create dummy file on disk for active-job-referenced asset
	activeFile := filepath.Join(tmpDir, "active.png")
	require.NoError(t, os.WriteFile(activeFile, []byte("fake-active-data"), 0644))

	activeAsset := model.StudioAsset{
		Id:                 "asset_active_1",
		JobId:              "job_active_1",
		UserId:             1,
		StorageURL:         "/api/v1/studio/assets/active.png",
		FileSize:           16,
		AvailabilityStatus: "available",
		ExpiryAt:           now - 100, // expired timestamp, but job is still active!
		CreatedAt:          now - 3600,
	}
	require.NoError(t, db.Create(&activeAsset).Error)

	activeJob := model.StudioToolJob{
		Id:             "job_active_1",
		UserId:         1,
		ToolId:         "image-to-video",
		Status:         model.StudioJobStatusProcessing,
		InputParams:    fmt.Sprintf(`{"image_url":"%s"}`, activeAsset.StorageURL),
		IdempotencyKey: "test_active_key",
		CreatedAt:      now - 60,
	}
	require.NoError(t, db.Create(&activeJob).Error)

	// Run cleanup
	ctx := context.Background()
	cleaned, err := CleanupExpiredStudioAssets(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, 1, cleaned, "Should clean exactly 1 expired unreferenced asset")

	// Expired asset file should be removed from disk
	_, statErr := os.Stat(expiredFile)
	assert.True(t, os.IsNotExist(statErr), "Expired asset file must be deleted from disk")

	// Expired asset status in DB should be 'expired'
	var updatedExpired model.StudioAsset
	require.NoError(t, db.First(&updatedExpired, "id = ?", expiredAsset.Id).Error)
	assert.Equal(t, "expired", updatedExpired.AvailabilityStatus)

	// Active job asset file MUST NOT be removed from disk
	_, statActive := os.Stat(activeFile)
	assert.False(t, os.IsNotExist(statActive), "Active job asset file must NOT be deleted from disk")

	// Active job asset TTL should have been extended
	var updatedActive model.StudioAsset
	require.NoError(t, db.First(&updatedActive, "id = ?", activeAsset.Id).Error)
	assert.True(t, updatedActive.ExpiryAt > now, "Active job asset TTL must be extended")
	assert.Equal(t, "available", updatedActive.AvailabilityStatus)
}
