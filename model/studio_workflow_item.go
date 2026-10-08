package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrWorkflowJobNotFound  = errors.New("resumable workflow job not found")
	ErrWorkflowItemNotFound = errors.New("resumable workflow item not found")
)

type WorkflowItemState string

const (
	ItemStatePending         WorkflowItemState = "PENDING"
	ItemStateLocalComplete   WorkflowItemState = "LOCAL_COMPLETE"
	ItemStateServerComplete  WorkflowItemState = "SERVER_COMPLETE"
	ItemStateFailedRetryable WorkflowItemState = "FAILED_RETRYABLE"
	ItemStateFailedTerminal  WorkflowItemState = "FAILED_TERMINAL"
	ItemStateCancelled       WorkflowItemState = "CANCELLED"
)

// StudioWorkflowItem tracks item-level lifecycle within a resumable batch job.
type StudioWorkflowItem struct {
	Id           string            `json:"id" gorm:"primaryKey;type:varchar(64)"`
	JobId        string            `json:"job_id" gorm:"type:varchar(64);index;not null"`
	UserId       int               `json:"user_id" gorm:"index;not null"`
	ItemIndex    int               `json:"item_index" gorm:"not null"`
	OriginalName string            `json:"original_name" gorm:"size:255"`
	InputCutout  string            `json:"input_cutout" gorm:"type:text"`
	VariantsJSON string            `json:"variants_json" gorm:"type:text"` // Serialized rendered variants
	State        WorkflowItemState `json:"state" gorm:"type:varchar(32);default:'PENDING'"`
	RetryCount   int               `json:"retry_count" gorm:"default:0"`
	MaxRetries   int               `json:"max_retries" gorm:"default:3"`
	ErrorReason  string            `json:"error_reason" gorm:"type:text"`
	DurationMs   int64             `json:"duration_ms"`
	CreatedAt    int64             `json:"created_at"`
	UpdatedAt    int64             `json:"updated_at"`
}

func (w *StudioWorkflowItem) TableName() string {
	return "studio_workflow_items"
}

// ResumableWorkflowJob wraps a batch job with item resume capabilities.
type ResumableWorkflowJob struct {
	Id             string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId         int    `json:"user_id" gorm:"index;not null"`
	TotalItems     int    `json:"total_items"`
	CompletedItems int    `json:"completed_items" gorm:"default:0"`
	FailedItems    int    `json:"failed_items" gorm:"default:0"`
	Status         string `json:"status" gorm:"type:varchar(32);default:'CREATED'"` // CREATED, IN_PROGRESS, PARTIAL_COMPLETE, COMPLETED, ABORTED
	ZipAssetId     string `json:"zip_asset_id" gorm:"size:128"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

func (r *ResumableWorkflowJob) TableName() string {
	return "studio_resumable_jobs"
}

func CreateResumableJob(job *ResumableWorkflowJob, items []StudioWorkflowItem) error {
	now := time.Now().Unix()
	job.CreatedAt = now
	job.UpdatedAt = now
	job.Status = "IN_PROGRESS"

	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(job).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].JobId = job.Id
			items[i].UserId = job.UserId
			items[i].CreatedAt = now
			items[i].UpdatedAt = now
			if items[i].State == "" {
				items[i].State = ItemStatePending
			}
			if items[i].MaxRetries == 0 {
				items[i].MaxRetries = 3
			}
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func GetResumableJob(userId int, jobId string) (*ResumableWorkflowJob, []StudioWorkflowItem, error) {
	var job ResumableWorkflowJob
	err := DB.Where("id = ? AND user_id = ?", jobId, userId).First(&job).Error
	if err != nil {
		return nil, nil, ErrWorkflowJobNotFound
	}
	var items []StudioWorkflowItem
	err = DB.Where("job_id = ?", jobId).Order("item_index ASC").Find(&items).Error
	return &job, items, err
}

func UpdateWorkflowItemState(jobId string, itemIndex int, state WorkflowItemState, errorReason string, variantsJSON string, durationMs int64) error {
	now := time.Now().Unix()
	updates := map[string]interface{}{
		"state":        state,
		"error_reason": errorReason,
		"duration_ms":  durationMs,
		"updated_at":   now,
	}
	if variantsJSON != "" {
		updates["variants_json"] = variantsJSON
	}

	res := DB.Model(&StudioWorkflowItem{}).
		Where("job_id = ? AND item_index = ?", jobId, itemIndex).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrWorkflowItemNotFound
	}

	// Recalculate job counts
	var completedCount int64
	var failedCount int64
	DB.Model(&StudioWorkflowItem{}).Where("job_id = ? AND state = ?", jobId, ItemStateServerComplete).Count(&completedCount)
	DB.Model(&StudioWorkflowItem{}).Where("job_id = ? AND state IN (?)", jobId, []WorkflowItemState{ItemStateFailedRetryable, ItemStateFailedTerminal}).Count(&failedCount)

	return DB.Model(&ResumableWorkflowJob{}).Where("id = ?", jobId).Updates(map[string]interface{}{
		"completed_items": int(completedCount),
		"failed_items":    int(failedCount),
		"updated_at":      now,
	}).Error
}

func MarkJobCompleted(jobId string, zipAssetId string) error {
	now := time.Now().Unix()
	var job ResumableWorkflowJob
	if err := DB.Where("id = ?", jobId).First(&job).Error; err != nil {
		return ErrWorkflowJobNotFound
	}

	status := "COMPLETED"
	if job.FailedItems > 0 && job.CompletedItems > 0 {
		status = "PARTIAL_COMPLETE"
	} else if job.FailedItems > 0 && job.CompletedItems == 0 {
		status = "FAILED"
	}

	return DB.Model(&ResumableWorkflowJob{}).Where("id = ?", jobId).Updates(map[string]interface{}{
		"status":       status,
		"zip_asset_id": zipAssetId,
		"updated_at":   now,
	}).Error
}

func GetPendingOrRetryableItems(jobId string) ([]StudioWorkflowItem, error) {
	var items []StudioWorkflowItem
	err := DB.Where("job_id = ? AND state IN (?)", jobId, []WorkflowItemState{ItemStatePending, ItemStateLocalComplete, ItemStateFailedRetryable}).
		Order("item_index ASC").
		Find(&items).Error
	return items, err
}
