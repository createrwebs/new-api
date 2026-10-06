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
	ErrStudioForbiddenAccess    = errors.New("forbidden: job does not belong to authenticated user")
)

// StudioJobStatus represents the lifecycle state of a media AI generation job (Section 13).
type StudioJobStatus string

const (
	StudioJobStatusCreated             StudioJobStatus = "CREATED"
	StudioJobStatusReserved            StudioJobStatus = "RESERVED"
	StudioJobStatusSubmitting          StudioJobStatus = "SUBMITTING"
	StudioJobStatusSubmitted           StudioJobStatus = "SUBMITTED"
	StudioJobStatusQueued              StudioJobStatus = "QUEUED"
	StudioJobStatusProcessing          StudioJobStatus = "PROCESSING"
	StudioJobStatusSucceeded           StudioJobStatus = "SUCCEEDED"
	StudioJobStatusFailed              StudioJobStatus = "FAILED"
	StudioJobStatusCancelled           StudioJobStatus = "CANCELLED"
	StudioJobStatusNeedsReview         StudioJobStatus = "NEEDS_REVIEW"
	StudioJobStatusAmbiguousSubmission StudioJobStatus = "AMBIGUOUS_SUBMISSION"
)

// StudioToolState defines tool availability status in UI (Section 21).
type StudioToolState string

const (
	StudioToolStateActive          StudioToolState = "ACTIVE"
	StudioToolStateBeta            StudioToolState = "BETA"
	StudioToolStateComingSoon      StudioToolState = "COMING_SOON"
	StudioToolStateOperatorBlocked StudioToolState = "OPERATOR_BLOCKED"
)

// StudioToolDefinition defines a discrete creative tool in Tora Studio (Section 7).
type StudioToolDefinition struct {
	Id               string          `json:"id" gorm:"primaryKey;type:varchar(64)"` // e.g. "image-generate"
	Slug             string          `json:"slug" gorm:"type:varchar(64);uniqueIndex;not null"` // e.g. "image-generator"
	Name             string          `json:"name" gorm:"type:varchar(128);not null"`
	DisplayName      string          `json:"display_name" gorm:"type:varchar(128);not null"`
	Description      string          `json:"description" gorm:"type:text"`
	Category         string          `json:"category" gorm:"type:varchar(32);index"` // "image", "video", "product", "utility"
	InputSchema      string          `json:"input_schema" gorm:"type:text"` // JSON Schema for client form generation
	OutputType       string          `json:"output_type" gorm:"type:varchar(32);not null"` // "image/png", "video/mp4"
	AllowedMIMETypes string          `json:"allowed_mime_types" gorm:"type:varchar(255)"` // "image/jpeg,image/png,image/webp"
	MaxUploadSize    int64           `json:"max_upload_size" gorm:"bigint;default:52428800"` // 50MB default
	IsEnabled        bool            `json:"is_enabled" gorm:"index;default:true"`
	IsPublic         bool            `json:"is_public" gorm:"index;default:true"`
	CreditCost       int             `json:"credit_cost" gorm:"type:int;not null;default:10"` // In Tora Credits (1 Credit = 1000 Quota)
	QuotaCost        int             `json:"quota_cost" gorm:"type:bigint;not null;default:10000"` // In raw Tora Quota units
	RiskClass        string          `json:"risk_class" gorm:"type:varchar(32);default:'low'"` // "low", "moderate", "high"
	DisplayOrder     int             `json:"display_order" gorm:"type:int;default:0"`
	PrimaryProvider  string          `json:"primary_provider" gorm:"type:varchar(32);default:'mock'"`
	PrimaryModel     string          `json:"primary_model" gorm:"type:varchar(128)"`
	MarginPercent    float64         `json:"margin_percent" gorm:"type:numeric(5,2);default:60.00"`
	Status           StudioToolState `json:"status" gorm:"type:varchar(32);default:'ACTIVE'"`
	CreatedAt        int64           `json:"created_at" gorm:"bigint"`
	UpdatedAt        int64           `json:"updated_at" gorm:"bigint"`
}

func (t *StudioToolDefinition) TableName() string {
	return "studio_tool_definitions"
}

// StudioToolTemplate provides curated preset prompts, styles, and dimensions (Section 8).
type StudioToolTemplate struct {
	Id                   string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	ToolId               string `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	Slug                 string `json:"slug" gorm:"type:varchar(64);index;not null"`
	Name                 string `json:"name" gorm:"type:varchar(128);not null"`
	DisplayName          string `json:"display_name" gorm:"type:varchar(128);not null"`
	Description          string `json:"description" gorm:"type:text"`
	Category             string `json:"category" gorm:"type:varchar(32);index"` // "product", "portrait", "motion"
	PresetPrompt         string `json:"preset_prompt" gorm:"type:text"`
	PresetNegativePrompt string `json:"preset_negative_prompt" gorm:"type:text"`
	PresetAspectRatio    string `json:"preset_aspect_ratio" gorm:"type:varchar(16);default:'1:1'"`
	PresetParams         string `json:"preset_params" gorm:"type:text"` // JSON
	ThumbnailURL         string `json:"thumbnail_url" gorm:"type:varchar(512)"`
	DisplayOrder         int    `json:"display_order" gorm:"type:int;default:0"`
	IsActive             bool   `json:"is_active" gorm:"index;default:true"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt            int64  `json:"updated_at" gorm:"bigint"`
}

func (t *StudioToolTemplate) TableName() string {
	return "studio_tool_templates"
}

// StudioProviderRoute manages dynamic routing and fallback between providers.
type StudioProviderRoute struct {
	Id            int     `json:"id" gorm:"primaryKey;autoIncrement"`
	ToolId        string  `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	ProviderName  string  `json:"provider_name" gorm:"type:varchar(32);index;not null"`
	ModelEndpoint string  `json:"model_endpoint" gorm:"type:varchar(128);not null"`
	Priority      int     `json:"priority" gorm:"type:int;default:1"` // 1 = primary, 2 = secondary
	IsActive      bool    `json:"is_active" gorm:"default:true"`
	MaxConcurrent int     `json:"max_concurrent" gorm:"default:10"`
	LastLatencyMs int64   `json:"last_latency_ms" gorm:"bigint;default:0"`
	ErrorRate     float64 `json:"error_rate" gorm:"type:numeric(5,2);default:0.0"`
	CreatedAt     int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt     int64   `json:"updated_at" gorm:"bigint"`
}

func (r *StudioProviderRoute) TableName() string {
	return "studio_provider_routes"
}

// StudioToolJob records a single asynchronous media generation request.
type StudioToolJob struct {
	Id             string          `json:"id" gorm:"primaryKey;type:varchar(64)"` // UUIDv4
	UserId         int             `json:"user_id" gorm:"index;not null"`
	ToolId         string          `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	TemplateId     string          `json:"template_id" gorm:"type:varchar(64);index"`
	RequestId      string          `json:"request_id" gorm:"type:varchar(64);index;not null"` // Links to WalletPreConsumeRecord
	IdempotencyKey string          `json:"idempotency_key" gorm:"type:varchar(128);uniqueIndex;not null"`
	ProviderName   string          `json:"provider_name" gorm:"type:varchar(32);index;not null"` // "mock", "fal", "muapi"
	ProviderJobId  string          `json:"provider_job_id" gorm:"type:varchar(128);index"`
	Status         StudioJobStatus `json:"status" gorm:"type:varchar(32);index;not null"`
	ReservedQuota  int             `json:"reserved_quota" gorm:"type:bigint;not null;default:0"`
	SettledQuota   int             `json:"settled_quota" gorm:"type:bigint;not null;default:0"`
	InputParams     string          `json:"input_params" gorm:"type:text"` // JSON payload
	OutputResult    string          `json:"output_result" gorm:"type:text"` // JSON payload / URLs
	PricingSnapshot string          `json:"pricing_snapshot" gorm:"type:text"` // JSON of StudioPricingSnapshot (Section 14)
	ErrorMessage    string          `json:"error_message" gorm:"type:text"`
	RiskClass       string          `json:"risk_class" gorm:"type:varchar(32);default:'low'"`
	ClientIP        string          `json:"client_ip" gorm:"type:varchar(64)"`
	CreatedAt       int64           `json:"created_at" gorm:"bigint;index"`
	UpdatedAt       int64           `json:"updated_at" gorm:"bigint"`
	CompletedAt     int64           `json:"completed_at" gorm:"bigint"`
}

func (j *StudioToolJob) TableName() string {
	return "studio_tool_jobs"
}

// StudioPricingSnapshot records complete commercial audit metadata for a studio job (Section 14).
type StudioPricingSnapshot struct {
	PricingVersion          string  `json:"pricing_version"`
	Provider                string  `json:"provider"`
	ProviderModel           string  `json:"provider_model"`
	ProviderEstimatedCostUSD float64 `json:"provider_estimated_cost_usd"`
	ProviderCostBasis       string  `json:"provider_cost_basis"` // "per_image", "per_second", "flat"
	TargetMargin            float64 `json:"target_margin"`
	CalculatedSellUSD       float64 `json:"calculated_sell_usd"`
	CalculatedCredits       float64 `json:"calculated_credits"`
	ChargedCredits          int     `json:"charged_credits"`
	ChargedQuota            int     `json:"charged_quota"`
	PlanMultiplier          float64 `json:"plan_multiplier"`
	QuotedAt                int64   `json:"quoted_at"`
	ExpiresAt               int64   `json:"expires_at"`
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

// StudioAsset tracks uploaded input and generated output media assets (Section 21 & 24).
type StudioAsset struct {
	Id                 string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId             int    `json:"user_id" gorm:"index;not null"`
	JobId              string `json:"job_id" gorm:"type:varchar(64);index"`
	AssetType          string `json:"asset_type" gorm:"type:varchar(32);not null"` // "input", "output"
	MIMEType           string `json:"mime_type" gorm:"type:varchar(64);not null"`
	FileSize           int64  `json:"file_size" gorm:"bigint;not null"`
	Width              int    `json:"width" gorm:"type:int;default:0"`
	Height             int    `json:"height" gorm:"type:int;default:0"`
	Duration           int    `json:"duration" gorm:"type:int;default:0"`
	AvailabilityStatus string `json:"availability_status" gorm:"type:varchar(32);default:'available'"`
	StorageURL         string `json:"storage_url" gorm:"type:varchar(512);not null"`
	ExpiryAt           int64  `json:"expiry_at" gorm:"bigint;index"`
	CreatedAt          int64  `json:"created_at" gorm:"bigint"`
}

func (a *StudioAsset) TableName() string {
	return "studio_assets"
}

// StudioCostSnapshot captures provider COGS for margin telemetry (Section 19 & 35).
type StudioCostSnapshot struct {
	Id                 int     `json:"id" gorm:"primaryKey;autoIncrement"`
	JobId              string  `json:"job_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	ToolId             string  `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	ProviderName       string  `json:"provider_name" gorm:"type:varchar(32);not null"`
	ProviderJobId      string  `json:"provider_job_id" gorm:"type:varchar(128)"`
	CostUSD            float64 `json:"cost_usd" gorm:"type:numeric(8,4);not null"`
	QuotaCost          int     `json:"quota_cost" gorm:"type:bigint;not null"`
	ToraRevenueUSD     float64 `json:"tora_revenue_usd" gorm:"type:numeric(8,4);default:0"`
	GrossProfitUSD     float64 `json:"gross_profit_usd" gorm:"type:numeric(8,4);default:0"`
	GrossMarginPercent float64 `json:"gross_margin_percent" gorm:"type:numeric(5,2);default:0"`
	MarginUSD          float64 `json:"margin_usd" gorm:"type:numeric(8,4);not null"`
	MarginPercent      float64 `json:"margin_percent" gorm:"type:numeric(5,2);not null"`
	SnapshotAt         int64   `json:"snapshot_at" gorm:"bigint;index"`
}

func (s *StudioCostSnapshot) TableName() string {
	return "studio_cost_snapshots"
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
		&StudioToolTemplate{},
		&StudioProviderRoute{},
		&StudioToolJob{},
		&StudioJobEvent{},
		&StudioAsset{},
		&StudioCostSnapshot{},
	)
}

// GetStudioToolDefinition retrieves a tool definition by its unique identifier or slug.
func GetStudioToolDefinition(toolIdOrSlug string) (*StudioToolDefinition, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var def StudioToolDefinition
	err := DB.Where("(id = ? OR slug = ?) AND is_enabled = true", toolIdOrSlug, toolIdOrSlug).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudioToolNotFound
		}
		return nil, err
	}
	return &def, nil
}

// ListPublicStudioTools retrieves all enabled public tools ordered by display order.
func ListPublicStudioTools() ([]StudioToolDefinition, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var tools []StudioToolDefinition
	err := DB.Where("is_enabled = ? AND is_public = ?", true, true).
		Order("display_order ASC, id ASC").
		Find(&tools).Error
	return tools, err
}

// ListStudioTemplates retrieves templates for a specific tool or category.
func ListStudioTemplates(toolId string, category string) ([]StudioToolTemplate, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	query := DB.Where("is_active = ?", true)
	if strings.TrimSpace(toolId) != "" {
		query = query.Where("tool_id = ?", toolId)
	}
	if strings.TrimSpace(category) != "" {
		query = query.Where("category = ?", category)
	}
	var templates []StudioToolTemplate
	err := query.Order("display_order ASC, id ASC").Find(&templates).Error
	return templates, err
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
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// GetStudioJobById retrieves a job by primary ID with user ownership protection (Section 20).
func GetStudioJobById(jobId string, userId int, isAdmin bool) (*StudioToolJob, error) {
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
	if !isAdmin && job.UserId != userId {
		return nil, ErrStudioForbiddenAccess
	}
	return &job, nil
}

// ListUserStudioJobs retrieves paginated jobs owned by the user.
func ListUserStudioJobs(userId int, page int, pageSize int) ([]StudioToolJob, int64, error) {
	if DB == nil {
		return nil, 0, errors.New("database not initialized")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	DB.Model(&StudioToolJob{}).Where("user_id = ?", userId).Count(&total)

	var jobs []StudioToolJob
	err := DB.Where("user_id = ?", userId).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&jobs).Error

	return jobs, total, err
}
