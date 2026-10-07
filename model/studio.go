package model

import (
	"errors"
	"fmt"
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
	IsEnabled        bool            `json:"is_enabled" gorm:"index"`
	IsPublic         bool            `json:"is_public" gorm:"index"`
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
	RouteId         string          `json:"route_id" gorm:"type:varchar(64);index"` // e.g. "ws-birefnet", "kie-flux-schnell"
	RoutingVersion  string          `json:"routing_version,omitempty" gorm:"type:varchar(32);default:'v1_deterministic'"`
	SelectionReason string          `json:"selection_reason" gorm:"type:text"` // Audit log of route selection decision
	ErrorMessage    string          `json:"error_message" gorm:"type:text"`
	RiskClass       string          `json:"risk_class" gorm:"type:varchar(32);default:'low'"`
	ClientIP        string          `json:"client_ip" gorm:"type:varchar(64)"`
	ExecutionType   string          `json:"execution_type" gorm:"type:varchar(32);default:'REAL_PROVIDER';index"` // "REAL_PROVIDER", "MOCK_PROVIDER", "INTERNAL_TEST"
	CreatedAt       int64           `json:"created_at" gorm:"bigint;index"`
	UpdatedAt       int64           `json:"updated_at" gorm:"bigint"`
	CompletedAt     int64           `json:"completed_at" gorm:"bigint"`
}

func (j *StudioToolJob) TableName() string {
	return "studio_tool_jobs"
}

// StudioPricingSnapshot records complete commercial audit metadata for a studio job (Section 14 & 34).
type StudioPricingSnapshot struct {
	QuoteID                  string   `json:"quote_id,omitempty"`
	ToolID                   string   `json:"tool_id,omitempty"`
	LogicalTool              string   `json:"logical_tool,omitempty"`
	QualityTier              string   `json:"quality_tier,omitempty"`
	PricingVersion           string   `json:"pricing_version"`
	Provider                 string   `json:"provider"`
	ProviderRoute            string   `json:"provider_route,omitempty"`
	ProviderModel            string   `json:"provider_model"`
	ProviderModelId          string   `json:"provider_model_id,omitempty"`
	ProviderPriceSource      string   `json:"provider_price_source,omitempty"`
	ProviderOriginalPrice    float64  `json:"provider_original_price,omitempty"`
	ProviderEffectivePrice   float64  `json:"provider_effective_price,omitempty"`
	ProviderDiscount         float64  `json:"provider_discount,omitempty"`
	Currency                 string   `json:"currency,omitempty"`
	InputPricingHash         string   `json:"input_pricing_hash,omitempty"`
	ProviderEstimatedCostUSD float64  `json:"provider_estimated_cost_usd"`
	ProviderCostBasis        string   `json:"provider_cost_basis"` // "per_image", "per_second", "flat"
	CostBasis                string   `json:"cost_basis,omitempty"`
	TargetMargin             float64  `json:"target_margin"`
	EstimatedMargin          float64  `json:"estimated_margin,omitempty"`
	CalculatedSellUSD        float64  `json:"calculated_sell_usd"`
	SellUSDEquivalent        float64  `json:"sell_usd_equivalent,omitempty"`
	CalculatedCredits        float64  `json:"calculated_credits"`
	ChargedCredits           int      `json:"charged_credits"`
	EstimatedCredits         int      `json:"estimated_credits,omitempty"`
	ChargedQuota             int      `json:"charged_quota"`
	PlanMultiplier           float64  `json:"plan_multiplier"`
	CandidateRoutes          []string `json:"candidate_routes,omitempty"`
	SelectionReason          string   `json:"selection_reason,omitempty"`
	RoutingVersion           string   `json:"routing_version,omitempty"`
	QuotedAt                 int64    `json:"quoted_at"`
	ExpiresAt                int64    `json:"expires_at"`
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
	AssetId            string `json:"asset_id,omitempty" gorm:"-"` // Alias
	UserId             int    `json:"user_id" gorm:"index;not null"`
	JobId              string `json:"job_id" gorm:"type:varchar(64);index"`
	AssetType          string `json:"asset_type" gorm:"type:varchar(32);not null"` // "input", "output"
	MIMEType           string `json:"mime_type" gorm:"type:varchar(64);not null"`
	MIME               string `json:"mime,omitempty" gorm:"-"`
	FileSize           int64  `json:"file_size" gorm:"bigint;not null"`
	Size               int64  `json:"size,omitempty" gorm:"-"`
	SHA256             string `json:"sha256,omitempty" gorm:"type:varchar(64)"`
	Width              int    `json:"width" gorm:"type:int;default:0"`
	Height             int    `json:"height" gorm:"type:int;default:0"`
	Duration           int    `json:"duration" gorm:"type:int;default:0"`
	AvailabilityStatus string `json:"availability_status" gorm:"type:varchar(32);default:'available'"`
	LifecycleState     string `json:"lifecycle_state,omitempty" gorm:"type:varchar(32);default:'JOB_OUTPUT';index"` // TEMPORARY_INPUT, JOB_INPUT, JOB_OUTPUT, PERSISTENT_USER_ASSET, EXPIRED, DELETED
	Status             string `json:"status,omitempty" gorm:"-"`
	StorageURL         string `json:"storage_url" gorm:"type:varchar(512);not null"`
	ExpiryAt           int64  `json:"expiry_at" gorm:"bigint;index"`
	ExpiresAt          int64  `json:"expires_at,omitempty" gorm:"-"`
	CreatedAt          int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"bigint"`
}

// Asset Lifecycle States (Section 55)
const (
	AssetLifecycleTemporaryInput      = "TEMPORARY_INPUT"
	AssetLifecycleJobInput            = "JOB_INPUT"
	AssetLifecycleJobOutput           = "JOB_OUTPUT"
	AssetLifecyclePersistentUserAsset = "PERSISTENT_USER_ASSET"
	AssetLifecycleExpired             = "EXPIRED"
	AssetLifecycleDeleted             = "DELETED"
)

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

// StudioConversionEvent tracks conversion funnel milestones (Queue 3).
type StudioConversionEvent struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"index;not null"`
	EventType string `json:"event_type" gorm:"type:varchar(64);index;not null"` // "insufficient_credit", "buy_credit_click", "purchase_return", "generation_after_purchase"
	ToolId    string `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	Credits   int    `json:"credits" gorm:"type:int;default:0"`
	SessionId string `json:"session_id" gorm:"type:varchar(64)"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index"`
}

func (c *StudioConversionEvent) TableName() string {
	return "studio_conversion_events"
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
		&StudioProviderConfig{},
		&StudioModelRoute{},
		&StudioToolJob{},
		&StudioJobEvent{},
		&StudioAsset{},
		&StudioCostSnapshot{},
		&StudioConversionEvent{},
		&StudioWorkflowPlan{},
		&StudioProviderCatalogSnapshot{},
		&StudioContractDriftEvent{},
		&StudioPricingDriftAlert{},
	)
}

// RecordStudioConversionEvent logs a conversion funnel step into the persistent database.
func RecordStudioConversionEvent(userId int, eventType string, toolId string, credits int, sessionId string) error {
	if DB == nil {
		return nil
	}
	event := &StudioConversionEvent{
		UserId:    userId,
		EventType: eventType,
		ToolId:    toolId,
		Credits:   credits,
		SessionId: sessionId,
		CreatedAt: common.GetTimestamp(),
	}
	return DB.Create(event).Error
}

// GetStudioToolDefinition retrieves a tool definition by its unique identifier or slug.
func GetStudioToolDefinition(toolIdOrSlug string) (*StudioToolDefinition, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	lookup := toolIdOrSlug
	if lookup == "object-eraser" {
		lookup = "object-erase"
	}
	var def StudioToolDefinition
	err := DB.Where("(id = ? OR slug = ? OR id = ? OR slug = ?) AND is_enabled = true", toolIdOrSlug, toolIdOrSlug, lookup, lookup).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudioToolNotFound
		}
		return nil, err
	}
	return &def, nil
}

// GetStudioToolDefinitionAnyStatus looks up a tool definition regardless of enabled status (e.g. for workflow quoting or admin).
func GetStudioToolDefinitionAnyStatus(toolIdOrSlug string) (*StudioToolDefinition, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	lookup := toolIdOrSlug
	if lookup == "object-eraser" {
		lookup = "object-erase"
	}
	var def StudioToolDefinition
	err := DB.Where("(id = ? OR slug = ? OR id = ? OR slug = ?)", toolIdOrSlug, toolIdOrSlug, lookup, lookup).First(&def).Error
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

// ListAllStudioTools retrieves all tool definitions regardless of enabled status for administration.
func ListAllStudioTools() ([]StudioToolDefinition, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var tools []StudioToolDefinition
	err := DB.Order("display_order ASC, id ASC").Find(&tools).Error
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

// GetStudioJobByProviderJobId retrieves a job by upstream provider job ID.
func GetStudioJobByProviderJobId(providerJobId string) (*StudioToolJob, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var job StudioToolJob
	err := DB.Where("provider_job_id = ? OR provider_job_id LIKE ?", providerJobId, "%:"+providerJobId).First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudioJobNotFound
		}
		return nil, err
	}
	return &job, nil
}

// GetStaleStudioJobs retrieves jobs in a specific state older than cutoff seconds for crash recovery.
func GetStaleStudioJobs(status StudioJobStatus, olderThanSeconds int64) ([]*StudioToolJob, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	cutoff := common.GetTimestamp() - olderThanSeconds
	var jobs []*StudioToolJob
	err := DB.Where("status = ? AND created_at <= ?", status, cutoff).Find(&jobs).Error
	return jobs, err
}

// CreateStudioAsset records a new media asset.
func CreateStudioAsset(asset *StudioAsset) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	if asset.Id == "" {
		asset.Id = fmt.Sprintf("asset_%d_%s", common.GetTimestamp(), common.GetUUID()[:8])
	}
	asset.AssetId = asset.Id
	asset.MIME = asset.MIMEType
	asset.Size = asset.FileSize
	asset.Status = asset.AvailabilityStatus
	asset.ExpiresAt = asset.ExpiryAt
	return DB.Create(asset).Error
}

// GetStudioAsset retrieves an asset by ID with tenant access control.
func GetStudioAsset(id string, userId int, isAdmin bool) (*StudioAsset, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var asset StudioAsset
	err := DB.Where("id = ?", id).First(&asset).Error
	if err != nil {
		return nil, err
	}
	if !isAdmin && asset.UserId != userId {
		return nil, ErrStudioForbiddenAccess
	}
	asset.AssetId = asset.Id
	asset.MIME = asset.MIMEType
	asset.Size = asset.FileSize
	asset.Status = asset.AvailabilityStatus
	asset.ExpiresAt = asset.ExpiryAt
	return &asset, nil
}

// ListStudioAssets retrieves paginated assets for a user.
func ListStudioAssets(userId int, page int, pageSize int) ([]*StudioAsset, int64, error) {
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
	DB.Model(&StudioAsset{}).Where("user_id = ?", userId).Count(&total)

	var assets []*StudioAsset
	err := DB.Where("user_id = ?", userId).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&assets).Error

	for _, a := range assets {
		a.AssetId = a.Id
		a.MIME = a.MIMEType
		a.Size = a.FileSize
		a.Status = a.AvailabilityStatus
		a.ExpiresAt = a.ExpiryAt
	}

	return assets, total, err
}
