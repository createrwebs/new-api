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
	ErrNativeTicketNotFound      = errors.New("native execution ticket not found")
	ErrNativeTicketExpired       = errors.New("native execution ticket has expired")
	ErrNativeTicketAlreadySettled = errors.New("native execution ticket is already settled")
	ErrNativeTicketAlreadyRefunded = errors.New("native execution ticket is already refunded")
	ErrNativeTicketInvalidState   = errors.New("invalid native ticket state transition")
)

// NativeExecutionClass categorizes where and how the tool computation runs.
type NativeExecutionClass string

const (
	ExecutionClassNativeBrowser        NativeExecutionClass = "NATIVE_BROWSER"
	ExecutionClassNativeLocalCPU       NativeExecutionClass = "NATIVE_LOCAL_CPU"
	ExecutionClassNativeServerless     NativeExecutionClass = "NATIVE_SERVERLESS"
	ExecutionClassExternalRelay        NativeExecutionClass = "EXTERNAL_RELAY"
	ExecutionClassDeterministicProcess NativeExecutionClass = "DETERMINISTIC_PROCESSING"
)

// NativeTicketStatus manages the deterministic lifecycle of a client-side execution ticket.
type NativeTicketStatus string

const (
	TicketStatusReserved  NativeTicketStatus = "RESERVED"
	TicketStatusStarted   NativeTicketStatus = "STARTED"
	TicketStatusCompleted NativeTicketStatus = "COMPLETED"
	TicketStatusFailed    NativeTicketStatus = "FAILED"
	TicketStatusExpired   NativeTicketStatus = "EXPIRED"
	TicketStatusSettled   NativeTicketStatus = "SETTLED"
	TicketStatusRefunded  NativeTicketStatus = "REFUNDED"
)

// NativeExecutionTicket binds client-side inference to server-authoritative Tora quota reservation.
type NativeExecutionTicket struct {
	Id                  string               `json:"id" gorm:"primaryKey;type:varchar(64)"`
	TicketId            string               `json:"ticket_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	UserId              int                  `json:"user_id" gorm:"index;not null"`
	ToolId              string               `json:"tool_id" gorm:"type:varchar(64);index;not null"`
	ToolVersion         string               `json:"tool_version" gorm:"type:varchar(32);default:'v1.0.0'"`
	RouteVersion        string               `json:"route_version" gorm:"type:varchar(32);default:'v1_native'"`
	ExecutionClass      NativeExecutionClass `json:"execution_class" gorm:"type:varchar(32);not null"`
	ModelVersionHash    string               `json:"model_version_hash" gorm:"type:varchar(64);not null"`
	QuoteId             string               `json:"quote_id" gorm:"type:varchar(64);index"`
	RequestId           string               `json:"request_id" gorm:"type:varchar(128);uniqueIndex;not null"` // Binds to WalletPreConsumeRecord
	ReservedQuota       int                  `json:"reserved_quota" gorm:"type:bigint;not null"`
	ReservedCredits     int                  `json:"reserved_credits" gorm:"type:int;not null"`
	NormalizedInputHash string               `json:"normalized_input_hash" gorm:"type:varchar(64);not null"`
	Status              NativeTicketStatus   `json:"status" gorm:"type:varchar(32);index;default:'RESERVED'"`
	IssuedAt            int64                `json:"issued_at" gorm:"bigint"`
	ExpiresAt           int64                `json:"expires_at" gorm:"bigint;index"`
	CompletedAt         int64                `json:"completed_at" gorm:"bigint"`
	SettledAt           int64                `json:"settled_at" gorm:"bigint"`
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
	BatchSize    int    `json:"batch_size" gorm:"not null"` // 1, 10, 25, 50
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
	Id           string `json:"id" gorm:"primaryKey;type:varchar(64)"`
	BatchJobId   string `json:"batch_job_id" gorm:"type:varchar(64);index;not null"`
	ItemIndex    int    `json:"item_index" gorm:"not null"`
	InputAsset   string `json:"input_asset" gorm:"type:text"`
	OutputAsset  string `json:"output_asset" gorm:"type:text"`
	Status       string `json:"status" gorm:"type:varchar(32);default:'PENDING'"` // PENDING, PROCESSING, SUCCEEDED, FAILED
	ErrorReason  string `json:"error_reason" gorm:"type:text"`
	TicketId     string `json:"ticket_id" gorm:"type:varchar(64)"`
	DurationMs   int64  `json:"duration_ms" gorm:"bigint"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
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
