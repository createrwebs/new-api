package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueue6_AssistantPlan_ThaiIntentResolution tests natural language resolution into a 3-step ToolPlan.
func TestQueue6_AssistantPlan_ThaiIntentResolution(t *testing.T) {
	_ = setupTestDBForStudio(t)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)
	assistantEngine := studioSvc.GetAssistantEngine()

	userId := 1
	thaiPrompt := "ทำรูปสินค้านี้เป็นโฆษณา Instagram แล้วทำวิดีโอ 5 วิ"
	initialImage := "https://cdn.example.com/raw_shoe.png"

	plan, err := assistantEngine.PlanFromPrompt(context.Background(), userId, thaiPrompt, initialImage)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, model.WorkflowStatusDraft, plan.Status)
	assert.Equal(t, thaiPrompt, plan.GoalPrompt)

	var steps []model.StudioWorkflowStep
	require.NoError(t, json.Unmarshal([]byte(plan.StepsJSON), &steps))
	require.Len(t, steps, 3, "Workflow must resolve to exactly 3 steps")

	// Step 1: background-remove
	assert.Equal(t, 1, steps[0].StepIndex)
	assert.Equal(t, "background-remove", steps[0].LogicalTool)
	assert.Empty(t, steps[0].DependsOn)
	assert.Equal(t, initialImage, steps[0].Parameters["image_url"])
	assert.NotEmpty(t, steps[0].QuoteID)
	assert.Equal(t, 10, steps[0].EstimatedCredits) // 10 Credits for background-remove

	// Step 2: product-photo with Instagram 4:5 template
	assert.Equal(t, 2, steps[1].StepIndex)
	assert.Equal(t, "product-photo", steps[1].LogicalTool)
	assert.Equal(t, "tpl-prod-instagram-4-5", steps[1].TemplateId)
	assert.Equal(t, []int{1}, steps[1].DependsOn, "Step 2 must depend on Step 1")
	assert.NotEmpty(t, steps[1].QuoteID)
	assert.Equal(t, 50, steps[1].EstimatedCredits) // 50 Credits for product-photo

	// Step 3: image-to-video with 5s duration
	assert.Equal(t, 3, steps[2].StepIndex)
	assert.Equal(t, "image-to-video", steps[2].LogicalTool)
	assert.EqualValues(t, 5, steps[2].Parameters["duration"])
	assert.Equal(t, []int{2}, steps[2].DependsOn, "Step 3 must depend on Step 2")
	assert.NotEmpty(t, steps[2].QuoteID)
	assert.Equal(t, 125, steps[2].EstimatedCredits) // 125 Credits for 5s 720p image-to-video

	// Total Authoritative Quote: 10 + 50 + 125 = 185 Credits
	expectedTotalCredits := 10 + 50 + 125
	assert.Equal(t, expectedTotalCredits, plan.TotalEstimatedCredits)
	assert.Equal(t, expectedTotalCredits*QuotaPerCredit, plan.TotalEstimatedQuota)
}

// TestQueue6_WorkflowBilling_StepLevelExecutionAndSafety tests step-level reserve/settle.
func TestQueue6_WorkflowBilling_StepLevelExecutionAndSafety(t *testing.T) {
	db := setupTestDBForStudio(t)

	// User has 300,000 Quota (300 credits)
	user := model.User{
		Username: "workflow_user_success",
		AffCode:  "AFF_WF_SUCCESS",
		Quota:    300000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.PlanFromPrompt(
		context.Background(),
		user.Id,
		"ทำรูปสินค้านี้เป็นโฆษณา Instagram แล้วทำวิดีโอ 5 วิ",
		"https://cdn.example.com/raw_shoe.png",
	)
	require.NoError(t, err)

	// Confirm and execute workflow
	completedPlan, err := assistantEngine.ConfirmAndExecuteWorkflow(
		context.Background(),
		plan.Id,
		user.Id,
		false,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.WorkflowStatusCompleted, completedPlan.Status)
	assert.Equal(t, 185, completedPlan.TotalSettledCredits)
	assert.Equal(t, 185000, completedPlan.TotalSettledQuota)

	// Verify all 3 steps succeeded and chained outputs
	_, steps, err := assistantEngine.GetWorkflowPlan(plan.Id)
	require.NoError(t, err)
	require.Len(t, steps, 3)

	for _, s := range steps {
		assert.Equal(t, model.StepStatusSuccess, s.Status)
		assert.NotEmpty(t, s.JobID)
		assert.NotEmpty(t, s.OutputResult)
	}

	// Verify User Wallet Deduction in universal single wallet: 300,000 - 185,000 = 115,000 Quota
	var refreshedUser model.User
	require.NoError(t, db.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, 300000-185000, refreshedUser.Quota, "User quota must deduct exactly 185,000 units")
}

// FailingStepMockProvider fails specifically for a given tool.
type FailingStepMockProvider struct {
	DeterministicMockProvider
	failTool string
}

func (f *FailingStepMockProvider) Submit(ctx context.Context, job *model.StudioToolJob) (*ProviderSubmitResult, error) {
	if job.ToolId == f.failTool {
		return nil, errors.New("simulated upstream provider failure on step")
	}
	return f.DeterministicMockProvider.Submit(ctx, job)
}

// TestQueue6_WorkflowBilling_PartialFailure_NeverRefundsSuccessfulSteps verifies failure safety.
func TestQueue6_WorkflowBilling_PartialFailure_NeverRefundsSuccessfulSteps(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "workflow_user_partial_fail",
		AffCode:  "AFF_WF_PARTIAL",
		Quota:    300000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&user).Error)

	// Fail specifically on image-to-video (Step 3)
	failProvider := &FailingStepMockProvider{
		DeterministicMockProvider: *NewDeterministicMockProvider(MockModeInstantSuccess),
		failTool:                  "image-to-video",
	}

	studioSvc := NewStudioService(failProvider)
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.PlanFromPrompt(
		context.Background(),
		user.Id,
		"ทำรูปสินค้านี้เป็นโฆษณา Instagram แล้วทำวิดีโอ 5 วิ",
		"https://cdn.example.com/raw_shoe.png",
	)
	require.NoError(t, err)

	// Execute workflow: Step 1 and 2 succeed, Step 3 fails
	execPlan, err := assistantEngine.ConfirmAndExecuteWorkflow(
		context.Background(),
		plan.Id,
		user.Id,
		false,
		"127.0.0.1",
		true,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow step 3 (image-to-video) failed")
	assert.Equal(t, model.WorkflowStatusPartialFailed, execPlan.Status)

	// Retrieve steps state
	_, steps, err := assistantEngine.GetWorkflowPlan(plan.Id)
	require.NoError(t, err)
	assert.Equal(t, model.StepStatusSuccess, steps[0].Status, "Step 1 must remain SUCCEEDED")
	assert.Equal(t, model.StepStatusSuccess, steps[1].Status, "Step 2 must remain SUCCEEDED")
	assert.Equal(t, model.StepStatusFailed, steps[2].Status, "Step 3 must be FAILED")

	// CRITICAL INVARIANT: Steps 1 & 2 quota (10 + 50 = 60 Credits = 60,000 Quota) remains settled!
	// Step 3 (125 Credits) was safely refunded.
	// User remaining quota: 300,000 - 60,000 = 240,000 Quota
	var refreshedUser model.User
	require.NoError(t, db.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, 240000, refreshedUser.Quota, "Successful steps 1 and 2 must NOT be refunded; only failed step refunded")
	assert.Equal(t, 60, execPlan.TotalSettledCredits)

	// Now Retry Step 3 with a working provider!
	workingProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc.RegisterProvider(workingProvider)

	retriedPlan, err := assistantEngine.RetryWorkflowStep(
		context.Background(),
		plan.Id,
		3,
		user.Id,
		"127.0.0.1",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, model.WorkflowStatusCompleted, retriedPlan.Status)
	assert.Equal(t, 185, retriedPlan.TotalSettledCredits)

	// Verify user balance after retrying Step 3: 240,000 - 125,000 = 115,000 Quota
	require.NoError(t, db.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, 115000, refreshedUser.Quota)
}

// TestQueue6_InsufficientCredit_UX_Data tests required/available/missing calculation.
func TestQueue6_InsufficientCredit_UX_Data(t *testing.T) {
	db := setupTestDBForStudio(t)

	// User only has 50 Credits (50,000 Quota)
	user := model.User{
		Username: "workflow_user_poor",
		AffCode:  "AFF_WF_POOR",
		Quota:    50000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.PlanFromPrompt(
		context.Background(),
		user.Id,
		"ทำรูปสินค้านี้เป็นโฆษณา Instagram แล้วทำวิดีโอ 5 วิ",
		"https://cdn.example.com/raw_shoe.png",
	)
	require.NoError(t, err)
	assert.Equal(t, 185, plan.TotalEstimatedCredits)

	// Attempting execution fails with structured WorkflowInsufficientCreditError
	_, errExec := assistantEngine.ConfirmAndExecuteWorkflow(
		context.Background(),
		plan.Id,
		user.Id,
		false,
		"127.0.0.1",
		true,
	)
	require.Error(t, errExec)

	var insErr *WorkflowInsufficientCreditError
	require.True(t, errors.As(errExec, &insErr))
	assert.Equal(t, 185, insErr.RequiredCredits)
	assert.Equal(t, 50, insErr.AvailableCredits)
	assert.Equal(t, 135, insErr.MissingCredits)

	// Verify wallet was NOT touched
	var refreshedUser model.User
	require.NoError(t, db.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, 50000, refreshedUser.Quota)
}

// TestQueue6_Safety_FaceSwapConsentRequired verifies identity workflows require consent.
func TestQueue6_Safety_FaceSwapConsentRequired(t *testing.T) {
	db := setupTestDBForStudio(t)

	user := model.User{
		Username: "workflow_user_faceswap",
		AffCode:  "AFF_WF_FACESWAP",
		Quota:    500000,
		Status:   common.UserStatusEnabled,
		Role:     common.RoleAdminUser,
	}
	require.NoError(t, db.Create(&user).Error)

	mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
	studioSvc := NewStudioService(mockProvider)
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.PlanFromPrompt(
		context.Background(),
		user.Id,
		"ช่วยเปลี่ยนหน้าในรูปนี้เป็นหน้าของฉัน",
		"https://cdn.example.com/portrait.png",
	)
	require.NoError(t, err)
	assert.True(t, plan.RequiresConsent, "Face swap workflow must require consent")

	// Attempt to execute without consent must be rejected
	_, errWithoutConsent := assistantEngine.ConfirmAndExecuteWorkflow(
		context.Background(),
		plan.Id,
		user.Id,
		false, // Consent NOT confirmed
		"127.0.0.1",
		true,
	)
	require.Error(t, errWithoutConsent)
	assert.True(t, errors.Is(errWithoutConsent, ErrConsentRequired))

	// Executing with consent confirmed succeeds
	completedPlan, errWithConsent := assistantEngine.ConfirmAndExecuteWorkflow(
		context.Background(),
		plan.Id,
		user.Id,
		true, // Consent confirmed!
		"127.0.0.1",
		true,
	)
	require.NoError(t, errWithConsent)
	assert.Equal(t, model.WorkflowStatusCompleted, completedPlan.Status)
}
