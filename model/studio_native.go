package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrNativeTicketNotFound        = errors.New("native execution ticket not found")
	ErrNativeTicketExpired         = errors.New("native execution ticket has expired")
	ErrNativeTicketAlreadyCharged  = errors.New("native execution ticket is already charged")
	ErrNativeTicketAlreadySettled  = errors.New("native execution ticket is already settled")
	ErrNativeTicketAlreadyRefunded = errors.New("native execution ticket is already refunded")
	ErrNativeTicketInvalidState    = errors.New("invalid native ticket state transition")
	ErrNativeTicketRetryExceeded   = errors.New("native execution ticket retry window has expired")
)

// NativeExecutionClass categorizes where and how the tool computation runs.
type NativeExecutionClass string

const (
	ExecutionClassNativeBrowser        NativeExecutionClass = "NATIVE_BROWSER"
	ExecutionClassNativeServer         NativeExecutionClass = "NATIVE_SERVER"
	ExecutionClassNativeServerless     NativeExecutionClass = "NATIVE_SERVERLESS"
	ExecutionClassExternalRelay        NativeExecutionClass = "EXTERNAL_RELAY"
	ExecutionClassDeterministicServer  NativeExecutionClass = "DETERMINISTIC_SERVER"

	// Legacy aliases
	ExecutionClassNativeLocalCPU       NativeExecutionClass = "NATIVE_SERVER"
	ExecutionClassDeterministicProcess NativeExecutionClass = "DETERMINISTIC_SERVER"
)

// NativeBillingPolicy defines financial commitment semantics per execution class (Section 3).
type NativeBillingPolicy string

const (
	BillingPolicyPrepaidExecution        NativeBillingPolicy = "PREPAID_EXECUTION"         // NATIVE_BROWSER: Charged at activation
	BillingPolicySuccessSettlement       NativeBillingPolicy = "SUCCESS_SETTLEMENT"        // DETERMINISTIC_SERVER, NATIVE_SERVER, NATIVE_SERVERLESS
	BillingPolicyAmbiguousReconciliation NativeBillingPolicy = "AMBIGUOUS_RECONCILIATION" // EXTERNAL_RELAY
)

// NativeTicketStatus manages the deterministic lifecycle of an execution ticket (Section 5).
type NativeTicketStatus string

const (
	TicketStatusQuoted          NativeTicketStatus = "QUOTED"
	TicketStatusReserved        NativeTicketStatus = "RESERVED"
	TicketStatusActivating      NativeTicketStatus = "ACTIVATING"
	TicketStatusCharged         NativeTicketStatus = "CHARGED"
	TicketStatusStarted         NativeTicketStatus = "STARTED"
	TicketStatusCompleted       NativeTicketStatus = "COMPLETED"
	TicketStatusFailedClient    NativeTicketStatus = "FAILED_CLIENT"
	TicketStatusExpired         NativeTicketStatus = "EXPIRED"
	TicketStatusSupportRefunded NativeTicketStatus = "SUPPORT_REFUNDED"

	// Legacy / Compatibility aliases
	TicketStatusSettled  NativeTicketStatus = "CHARGED"
	TicketStatusRefunded NativeTicketStatus = "SUPPORT_REFUNDED"
	TicketStatusFailed   NativeTicketStatus = "FAILED_CLIENT"
)

// NativeExecutionTicket binds client-side or server execution to authoritative Tora quota.
type NativeExecutionTicket struct {
	Id                  string               `json:"id" gorm:"primaryKey;type:varchar(64)"`
	TicketId            string               `json:"ticket_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	UserId              int                  `json:"user_id" gorm:"index;not null"`
	ToolId              string               `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	ToolVersion         string               `json:"tool_version" gorm:"type:varchar(32);default:'v1.0.0'"`
	RouteVersion        string               `json:"route_version" gorm:"type:varchar(32);default:'v1_native'"`
	ExecutionClass      NativeExecutionClass `json:"execution_class" gorm:"type:varchar(32);not null"`
	BillingPolicy       NativeBillingPolicy  `json:"billing_policy" gorm:"type:varchar(32);not null;default:'PREPAID_EXECUTION'"`
	ModelId             string               `json:"model_id" gorm:"type:varchar(64)"`
	ModelVersion        string               `json:"model_version" gorm:"type:varchar(32)"`
	ModelVersionHash    string               `json:"model_version_hash" gorm:"type:varchar(64);not null"`
	QuoteId             string               `json:"quote_id" gorm:"type:varchar(64);index"`
	RequestId           string               `json:"request_id" gorm:"type:varchar(128);uniqueIndex;not null"` // Binds to WalletPreConsumeRecord
	IdempotencyKey      string               `json:"idempotency_key,omitempty" gorm:"type:varchar(128);index"`
	ReservedQuota       int                  `json:"reserved_quota" gorm:"type:bigint;not null"`
	ReservedCredits     int                  `json:"reserved_credits" gorm:"type:int;not null"`
	ChargedQuota        int                  `json:"charged_quota" gorm:"type:bigint;default:0"`
	ChargedCredits      int                  `json:"charged_credits" gorm:"type:int;default:0"`
	NormalizedInputHash string               `json:"normalized_input_hash" gorm:"type:varchar(64);not null"`
	Status              NativeTicketStatus   `json:"status" gorm:"type:varchar(32);index;default:'RESERVED'"`
	AuthToken           string               `json:"auth_token,omitempty" gorm:"type:varchar(256)"`
	Nonce               string               `json:"nonce" gorm:"type:varchar(64)"`
	IssuedAt            int64                `json:"issued_at" gorm:"bigint"`
	ExpiresAt           int64                `json:"expires_at" gorm:"bigint;index"`
	ChargedAt           int64                `json:"charged_at" gorm:"bigint"`
	SettledAt           int64                `json:"settled_at" gorm:"bigint"` // Alias for ChargedAt in prepaid
	RetryUntil          int64                `json:"retry_until" gorm:"bigint;index"`
	RetryCount          int                  `json:"retry_count" gorm:"default:0"`
	CompletedAt         int64                `json:"completed_at" gorm:"bigint"`
	RefundedAt          int64                `json:"refunded_at" gorm:"bigint"`
	ClientExecutionMs   int64                `json:"client_execution_ms" gorm:"bigint;default:0"`
	ClientDeviceClass   string               `json:"client_device_class" gorm:"type:varchar(64)"`
	OutputAssetHash     string               `json:"output_asset_hash" gorm:"type:varchar(64)"`
	ErrorReason         string               `json:"error_reason" gorm:"type:text"`
	CreatedAt           int64                `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64                `json:"updated_at" gorm:"bigint"`
}

func (t *NativeExecutionTicket) TableName() string {
	return "studio_native_tickets"
}

// ComputeNormalizedInputHash produces a deterministic SHA-256 hash of tool parameters.
func ComputeNormalizedInputHash(inputs map[string]interface{}) string {
	b, _ := json.Marshal(inputs)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// CreateNativeTicket persists a newly authorized execution ticket.
func CreateNativeTicket(ticket *NativeExecutionTicket) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	now := common.GetTimestamp()
	if ticket.Id == "" {
		ticket.Id = fmt.Sprintf("tkt_%d_%s", now, common.GetUUID()[:8])
	}
	ticket.TicketId = ticket.Id
	ticket.IssuedAt = now
	ticket.CreatedAt = now
	ticket.UpdatedAt = now
	if ticket.ExpiresAt == 0 {
		ticket.ExpiresAt = now + 300 // 5-minute standard TTL
	}
	if ticket.RetryUntil == 0 && ticket.ExecutionClass == ExecutionClassNativeBrowser {
		ticket.RetryUntil = now + 1800 // 30-minute fair retry window
	}
	return DB.Create(ticket).Error
}

// GetNativeTicket retrieves a ticket by ticket_id.
func GetNativeTicket(ticketId string) (*NativeExecutionTicket, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	var ticket NativeExecutionTicket
	err := DB.Where("ticket_id = ? OR id = ?", ticketId, ticketId).First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNativeTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// GetNativeTicketByIdempotencyKey retrieves an existing ticket by idempotency key.
func GetNativeTicketByIdempotencyKey(userId int, idempotencyKey string) (*NativeExecutionTicket, error) {
	if DB == nil || idempotencyKey == "" {
		return nil, ErrNativeTicketNotFound
	}
	var ticket NativeExecutionTicket
	err := DB.Where("user_id = ? AND idempotency_key = ?", userId, idempotencyKey).First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNativeTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// UpdateNativeTicketStatus transitions the state of a ticket with concurrency protection.
func UpdateNativeTicketStatus(ticketId string, newStatus NativeTicketStatus, updates map[string]interface{}) error {
	if DB == nil {
		return errors.New("database not initialized")
	}
	if updates == nil {
		updates = make(map[string]interface{})
	}
	updates["status"] = newStatus
	updates["updated_at"] = common.GetTimestamp()

	res := DB.Model(&NativeExecutionTicket{}).
		Where("(ticket_id = ? OR id = ?)", ticketId, ticketId).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNativeTicketNotFound
	}
	return nil
}

// StudioBatchJob tracks a multi-item batch execution (Section 25).
type StudioBatchJob struct {
	Id           string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId       int    `json:"user_id" gorm:"index;not null"`
	ToolId       string `json:"tool_id" gorm:"type:varchar(64);not null"`
	BatchSize    int    `json:"batch_size" gorm:"not null"` // 1, 5, 10
	TotalQuota   int    `json:"total_quota" gorm:"type:bigint;not null"`
	TotalCredits int    `json:"total_credits" gorm:"not null"`
	SuccessCount int    `json:"success_count" gorm:"default:0"`
	FailedCount  int    `json:"failed_count" gorm:"default:0"`
	Status       string `json:"status" gorm:"type:varchar(32);default:'PROCESSING'"` // PROCESSING, COMPLETED, PARTIAL_SUCCESS, FAILED
	ZipAssetId   string `json:"zip_asset_id" gorm:"type:varchar(64)"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
}

func (b *StudioBatchJob) TableName() string {
	return "studio_batch_jobs"
}

// StudioBatchItem tracks individual item progress in a batch job.
type StudioBatchItem struct {
	Id          string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	BatchJobId  string `json:"batch_job_id" gorm:"type:varchar(64);index;not null"`
	ItemIndex   int    `json:"item_index" gorm:"not null"`
	InputAsset  string `json:"input_asset" gorm:"type:text"`
	OutputAsset string `json:"output_asset" gorm:"type:text"`
	Status      string `json:"status" gorm:"type:varchar(32);default:'PENDING'"` // PENDING, PROCESSING, SUCCEEDED, FAILED
	ErrorReason string `json:"error_reason" gorm:"type:text"`
	TicketId    string `json:"ticket_id" gorm:"type:varchar(64)"`
	DurationMs  int64  `json:"duration_ms" gorm:"bigint"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
}

func (bi *StudioBatchItem) TableName() string {
	return "studio_batch_items"
}

// EnsureNativeStudioTables migrates all native studio tables.
func EnsureNativeStudioTables(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.AutoMigrate(
		&NativeExecutionTicket{},
		&StudioBatchJob{},
		&StudioBatchItem{},
	)
}
