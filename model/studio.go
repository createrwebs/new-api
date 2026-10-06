package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrStudioToolNotFound       = errors.New("studio tool definition not found")
	ErrStudioJobNotFound        = errors.New("studio job not found")
	ErrStudioInvalidStateChange = errors.New("invalid studio job state transition")
)

// StudioJobStatus represents the lifecycle state of a media AI generation job.
type StudioJobStatus string

const (
	StudioJobStatusPending       StudioJobStatus = "pending"
	StudioJobStatusReserved      StudioJobStatus = "reserved"
	StudioJobStatusProcessing    StudioJobStatus = "processing"
	StudioJobStatusCompleted     StudioJobStatus = "completed"
	StudioJobStatusFailed        StudioJobStatus = "failed"
	StudioJobStatusCancelled     StudioJobStatus = "cancelled"
	StudioJobStatusReconciling   StudioJobStatus = "reconciling"
)

// StudioToolDefinition defines a discrete creative tool in Tora Studio.
type StudioToolDefinition struct {
	Id           string `json:"id" gorm:"primaryKey;type:varchar(64)"` // e.g. "image_generate_fast"
	Category     string `json:"category" gorm:"type:varchar(32);index"` // "image", "video", "audio", "utility"
	Name         string `json:"name" gorm:"type:varchar(128);not null"`
	DisplayName  string `json:"display_name" gorm:"type:varchar(128);not null"`
	Description  string `json:"description" gorm:"type:text"`
	CreditCost   int    `json:"credit_cost" gorm:"type:int;not null;default:10"` // In Tora Credits (1 Credit = 1000 Quota)
	QuotaCost    int    `json:"quota_cost" gorm:"type:bigint;not null;default:10000"` // In raw Tora Quota units
	PrimaryModel string `json:"primary_model" gorm:"type:varchar(128)"`
	IsActive     bool   `json:"is_active" gorm:"index;default:true"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
}

func (t *StudioToolDefinition) TableName() string {
	return "studio_tool_definitions"
}

// StudioToolJob records a single asynchronous media generation request.
type StudioToolJob struct {
	Id             string          `json:"id" gorm:"primaryKey;type:varchar(64)"` // UUIDv4
	UserId         int             `json:"user_id" gorm:"index;not null"`
	ToolId         string          `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	RequestId      string          `json:"request_id" gorm:"type:varchar(64);index;not null"` // Links to WalletPreConsumeRecord
	IdempotencyKey string          `json:"idempotency_key" gorm:"type:varchar(128);uniqueIndex;not null"`
	ProviderName   string          `json:"provider_name" gorm:"type:varchar(32);index;not null"` // "mock", "fal", "muapi", "replicate"
	ProviderJobId  string          `json:"provider_job_id" gorm:"type:varchar(128);index"`
	Status         StudioJobStatus `json:"status" gorm:"type:varchar(32);index;not null"`
	ReservedQuota  int             `json:"reserved_quota" gorm:"type:bigint;not null;default:0"`
	SettledQuota   int             `json:"settled_quota" gorm:"type:bigint;not null;default:0"`
	InputParams    string          `json:"input_params" gorm:"type:text"` // JSON payload
	OutputResult   string          `json:"output_result" gorm:"type:text"` // JSON payload / URLs
	ErrorMessage   string          `json:"error_message" gorm:"type:text"`
	CreatedAt      int64           `json:"created_at" gorm:"bigint;index"`
	UpdatedAt      int64           `json:"updated_at" gorm:"bigint"`
	CompletedAt    int64           `json:"completed_at" gorm:"bigint"`
}

func (j *StudioToolJob) TableName() string {
	return "studio_tool_jobs"
}

// StudioJobEvent records state changes for auditability and recovery.
type StudioJobEvent struct {
	Id        int             `json:"id" gorm:"primaryKey;autoIncrement"`
	JobId     string          `json:"job_id" gorm:"type:varchar(64);index;not null"`
	EventType string          `json:"event_type" gorm:"type:varchar(64);not null"`
	OldStatus StudioJobStatus `json:"old_status" gorm:"type:varchar(32)"`
	NewStatus StudioJobStatus `json:"new_status" gorm:"type:varchar(32)"`
	Payload   string          `json:"payload" gorm:"type:text"`
	CreatedAt int64           `json:"created_at" gorm:"bigint"`
}

func (e *StudioJobEvent) TableName() string {
	return "studio_job_events"
}

func (j *StudioToolJob) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if j.CreatedAt == 0 {
		j.CreatedAt = now
	}
	j.UpdatedAt = now
	return nil
}

func (j *StudioToolJob) BeforeUpdate(tx *gorm.DB) error {
	j.UpdatedAt = common.GetTimestamp()
	return nil
}

func EnsureStudioTables(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
		&StudioToolDefinition{},
		&StudioToolJob{},
		&StudioJobEvent{},
	)
}

// GetStudioToolDefinition retrieves a tool definition by its unique identifier.
func GetStudioToolDefinition(toolId string) (*StudioToolDefinition, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var def StudioToolDefinition
	err := DB.Where("id = ? AND is_active = true", toolId).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudioToolNotFound
		}
		return nil, err
	}
	return &def, nil
}

// GetStudioJobByIdempotency retrieves an existing job by user and idempotency key.
func GetStudioJobByIdempotency(userId int, idempotencyKey string) (*StudioToolJob, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, errors.New("idempotency key required")
	}
	var job StudioToolJob
	err := DB.Where("user_id = ? AND idempotency_key = ?", userId, idempotencyKey).First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error here
		}
		return nil, err
	}
	return &job, nil
}

// GetStudioJobById retrieves a job by its primary ID.
func GetStudioJobById(jobId string) (*StudioToolJob, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var job StudioToolJob
	err := DB.Where("id = ?", jobId).First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudioJobNotFound
		}
		return nil, err
	}
	return &job, nil
}
