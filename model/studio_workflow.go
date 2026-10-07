package model

import (
	"gorm.io/gorm"
)

// StudioWorkflowStatus defines the lifecycle states of a multi-step ToolPlan.
type StudioWorkflowStatus string

const (
	WorkflowStatusDraft         StudioWorkflowStatus = "DRAFT"
	WorkflowStatusConfirmed     StudioWorkflowStatus = "CONFIRMED"
	WorkflowStatusRunning       StudioWorkflowStatus = "RUNNING"
	WorkflowStatusCompleted     StudioWorkflowStatus = "COMPLETED"
	WorkflowStatusPartialFailed StudioWorkflowStatus = "PARTIAL_FAILED"
	WorkflowStatusFailed        StudioWorkflowStatus = "FAILED"
	WorkflowStatusCancelled     StudioWorkflowStatus = "CANCELLED"
)

// StudioStepStatus defines the state of an individual step in a ToolPlan.
type StudioStepStatus string

const (
	StepStatusPending  StudioStepStatus = "PENDING"
	StepStatusReserved StudioStepStatus = "RESERVED"
	StepStatusRunning  StudioStepStatus = "RUNNING"
	StepStatusSuccess  StudioStepStatus = "SUCCEEDED"
	StepStatusFailed   StudioStepStatus = "FAILED"
	StepStatusSkipped  StudioStepStatus = "SKIPPED"
)

// StudioWorkflowStep represents a single discrete tool operation in a ToolPlan.
type StudioWorkflowStep struct {
	StepIndex        int                    `json:"step_index"`
	LogicalTool      string                 `json:"logical_tool"` // e.g. "background-remove", "product-photo", "image-to-video"
	ToolDisplayName  string                 `json:"tool_display_name"`
	TemplateId       string                 `json:"template_id,omitempty"`
	TemplateName     string                 `json:"template_name,omitempty"`
	Parameters       map[string]interface{} `json:"parameters"`
	DependsOn        []int                  `json:"depends_on"` // Step indices providing input assets
	InputAssetRefs   []string               `json:"input_asset_refs,omitempty"`
	QuoteID          string                 `json:"quote_id"`
	EstimatedCredits int                    `json:"estimated_credits"`
	EstimatedQuota   int                    `json:"estimated_quota"`
	Status           StudioStepStatus       `json:"status"`
	JobID            string                 `json:"job_id,omitempty"`
	OutputResult     string                 `json:"output_result,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
}

// StudioWorkflowPlan represents a multi-step creative generation plan.
type StudioWorkflowPlan struct {
	Id                    string               `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserId                int                  `json:"user_id" gorm:"index;not null"`
	GoalPrompt            string               `json:"goal_prompt" gorm:"type:text;not null"`
	Status                StudioWorkflowStatus `json:"status" gorm:"type:varchar(32);default:'DRAFT'"`
	StepsJSON             string               `json:"steps_json" gorm:"type:text"` // Serialized []StudioWorkflowStep
	TotalEstimatedCredits int                  `json:"total_estimated_credits" gorm:"type:int;default:0"`
	TotalEstimatedQuota   int                  `json:"total_estimated_quota" gorm:"type:bigint;default:0"`
	TotalSettledCredits   int                  `json:"total_settled_credits" gorm:"type:int;default:0"`
	TotalSettledQuota     int                  `json:"total_settled_quota" gorm:"type:bigint;default:0"`
	RequiresConsent       bool                 `json:"requires_consent" gorm:"default:false"`
	ConsentConfirmed      bool                 `json:"consent_confirmed" gorm:"default:false"`
	CreatedAt             int64                `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64                `json:"updated_at" gorm:"bigint"`
}

func (w *StudioWorkflowPlan) TableName() string {
	return "studio_workflow_plans"
}

// MigrateWorkflowPlan creates the database table if it doesn't exist.
func MigrateWorkflowPlan(db *gorm.DB) error {
	return db.AutoMigrate(&StudioWorkflowPlan{})
}
