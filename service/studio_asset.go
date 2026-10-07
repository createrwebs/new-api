package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
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

	// ToraCanaryAssetFilename is the fixed, reproducible Tora synthetic canary asset filename.
	ToraCanaryAssetFilename = "tora_canary_synthetic_128x128.png"

	// ToraCanaryAssetSHA256 is the verified cryptographic digest of the synthetic canary asset.
	ToraCanaryAssetSHA256 = "0da9b6b7598b6e4934116253d113ad5ca6d6584868393a15617b320d51f0aa41"

	// ToraCanaryAssetBytes is the exact byte size of the synthetic 128x128 PNG image.
	ToraCanaryAssetBytes = 668
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
		if filename == ToraCanaryAssetFilename {
			continue
		}
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

// EnsureCanaryAsset ensures the fixed, reproducible Tora synthetic canary asset is persisted on disk.
func EnsureCanaryAsset() (string, string, error) {
	uploadDir := GetStudioUploadDir()
	filePath := filepath.Join(uploadDir, ToraCanaryAssetFilename)

	// Check if already exists and matches expected byte size
	if info, err := os.Stat(filePath); err == nil && info.Size() == ToraCanaryAssetBytes {
		return filePath, GetCanaryAssetURL(), nil
	}

	// Generate deterministic synthetic 128x128 PNG (solid white background, orange circle center)
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	orange := color.RGBA{249, 115, 22, 255}
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			dx := float64(x - 64)
			dy := float64(y - 64)
			if dx*dx+dy*dy <= 40*40 {
				img.Set(x, y, orange)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", err
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return "", "", err
	}

	return filePath, GetCanaryAssetURL(), nil
}

// GetCanaryAssetURL returns the public URL for the server-controlled synthetic canary image.
func GetCanaryAssetURL() string {
	serverURL := strings.TrimRight(os.Getenv("SERVER_URL"), "/")
	if serverURL == "" {
		serverURL = "https://www.toraapi.com"
	}
	return fmt.Sprintf("%s/api/studio/assets/%s", serverURL, ToraCanaryAssetFilename)
}

// IngestOutputAssetFromURL downloads a provider output URL, persists it locally with SHA-256, and registers ownership (Queue 2H Section 20).
func IngestOutputAssetFromURL(ctx context.Context, db *gorm.DB, userId int, jobId string, remoteURL string) (*model.StudioAsset, error) {
	if strings.TrimSpace(remoteURL) == "" {
		return nil, errors.New("remote URL cannot be empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating download request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed downloading remote asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote asset server returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading asset bytes: %w", err)
	}

	if len(data) == 0 {
		return nil, errors.New("downloaded asset is empty (0 bytes)")
	}

	if err := CheckStorageQuota(db, userId, int64(len(data))); err != nil {
		return nil, err
	}

	// Determine MIME type and file extension
	contentType := resp.Header.Get("Content-Type")
	mimeType := http.DetectContentType(data)
	if contentType != "" && strings.Contains(contentType, "image/") {
		mimeType = strings.Split(contentType, ";")[0]
	}

	ext := ".png"
	switch strings.ToLower(mimeType) {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	case "video/mp4":
		ext = ".mp4"
	}

	now := common.GetTimestamp()
	filename := fmt.Sprintf("out_%d_%s%s", now, common.GetUUID()[:8], ext)
	uploadDir := GetStudioUploadDir()
	filePath := filepath.Join(uploadDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed persisting local asset file: %w", err)
	}

	hashBytes := sha256.Sum256(data)
	sha256Hex := fmt.Sprintf("%x", hashBytes)

	serverURL := strings.TrimRight(os.Getenv("SERVER_URL"), "/")
	if serverURL == "" {
		serverURL = "https://www.toraapi.com"
	}
	localURL := fmt.Sprintf("%s/api/studio/assets/%s", serverURL, filename)

	asset := &model.StudioAsset{
		Id:                 fmt.Sprintf("ast_%d_%s", now, common.GetUUID()[:8]),
		UserId:             userId,
		JobId:              jobId,
		AssetType:          "output",
		MIMEType:           mimeType,
		FileSize:           int64(len(data)),
		SHA256:             sha256Hex,
		StorageURL:         localURL,
		AvailabilityStatus: "available",
		ExpiryAt:           now + int64(DefaultOutputAssetTTL.Seconds()),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if db != nil {
		if err := model.CreateStudioAsset(asset); err != nil {
			_ = os.Remove(filePath)
			return nil, fmt.Errorf("failed persisting asset to database: %w", err)
		}
	}

	return asset, nil
}
