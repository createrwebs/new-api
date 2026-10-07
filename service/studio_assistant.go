package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var (
	ErrWorkflowNotFound   = errors.New("workflow plan not found")
	ErrConsentRequired    = errors.New("explicit consent is required for identity-altering workflows before execution")
	ErrWorkflowNotRunning = errors.New("workflow is not in executable state")
)

// WorkflowInsufficientCreditError represents a structured insufficient balance response.
type WorkflowInsufficientCreditError struct {
	RequiredCredits  int `json:"required_credits"`
	AvailableCredits int `json:"available_credits"`
	MissingCredits   int `json:"missing_credits"`
}

func (e *WorkflowInsufficientCreditError) Error() string {
	return fmt.Sprintf("insufficient credits: required %d Credits, available %d Credits, missing %d Credits",
		e.RequiredCredits, e.AvailableCredits, e.MissingCredits)
}

// StudioAssistantEngine resolves conversational intents into verifiable ToolPlans.
type StudioAssistantEngine struct {
	studioSvc     *StudioService
	pricingEngine *PricingEngine
	mu            sync.RWMutex
}

func NewStudioAssistantEngine(studioSvc *StudioService) *StudioAssistantEngine {
	return &StudioAssistantEngine{
		studioSvc:     studioSvc,
		pricingEngine: studioSvc.pricingEngine,
	}
}

// PlanFromPrompt analyzes natural language outcome and resolves an authoritative multi-step ToolPlan.
func (a *StudioAssistantEngine) PlanFromPrompt(
	ctx context.Context,
	userId int,
	prompt string,
	initialImageURL string,
) (*model.StudioWorkflowPlan, error) {
	cleanPrompt := strings.TrimSpace(prompt)
	if cleanPrompt == "" {
		return nil, errors.New("prompt cannot be empty")
	}

	lowerPrompt := strings.ToLower(cleanPrompt)

	// Step 1: Detect Intent & Build Logical Steps
	var steps []model.StudioWorkflowStep
	stepCounter := 1

	// Intent Check: Product Photo / E-Commerce
	hasProductPhoto := strings.Contains(lowerPrompt, "รูปสินค้า") ||
		strings.Contains(lowerPrompt, "โฆษณา") ||
		strings.Contains(lowerPrompt, "สินค้า") ||
		strings.Contains(lowerPrompt, "ขายของ") ||
		strings.Contains(lowerPrompt, "product photo") ||
		strings.Contains(lowerPrompt, "commercial") ||
		strings.Contains(lowerPrompt, "shopee") ||
		strings.Contains(lowerPrompt, "lazada") ||
		strings.Contains(lowerPrompt, "instagram") ||
		strings.Contains(lowerPrompt, "ig") ||
		strings.Contains(lowerPrompt, "tiktok")

	// Intent Check: Background Removal (Queue 6: raw product photos standardly isolate subject first)
	skipBgRemove := strings.Contains(lowerPrompt, "ไม่ลบพื้นหลัง") || strings.Contains(lowerPrompt, "no bg remove") || strings.Contains(lowerPrompt, "keep background")
	hasBgRemove := !skipBgRemove && (strings.Contains(lowerPrompt, "ลบพื้นหลัง") ||
		strings.Contains(lowerPrompt, "ตัดฉากหลัง") ||
		strings.Contains(lowerPrompt, "พื้นหลังขาว") ||
		strings.Contains(lowerPrompt, "remove background") ||
		strings.Contains(lowerPrompt, "cutout") ||
		hasProductPhoto)

	// Intent Check: Video
	hasVideo := strings.Contains(lowerPrompt, "วิดีโอ") ||
		strings.Contains(lowerPrompt, "วีดีโอ") ||
		strings.Contains(lowerPrompt, "video") ||
		strings.Contains(lowerPrompt, "animate") ||
		strings.Contains(lowerPrompt, "ภาพเคลื่อนไหว") ||
		strings.Contains(lowerPrompt, "5 วิ") ||
		strings.Contains(lowerPrompt, "5s")

	// Intent Check: Upscale
	hasUpscale := strings.Contains(lowerPrompt, "ขยายภาพ") ||
		strings.Contains(lowerPrompt, "ชัดขึ้น") ||
		strings.Contains(lowerPrompt, "upscale")

	// Intent Check: Face Swap (Safety Critical)
	hasFaceSwap := strings.Contains(lowerPrompt, "เปลี่ยนหน้า") ||
		strings.Contains(lowerPrompt, "สลับหน้า") ||
		strings.Contains(lowerPrompt, "face swap")

	// Default fallback if no specific tool detected: image-generate
	if !hasBgRemove && !hasProductPhoto && !hasVideo && !hasUpscale && !hasFaceSwap {
		hasProductPhoto = true
	}

	// 1. Resolve Background Remove Step
	if hasBgRemove {
		params := map[string]interface{}{}
		if initialImageURL != "" {
			params["image_url"] = initialImageURL
		}
		steps = append(steps, model.StudioWorkflowStep{
			StepIndex:       stepCounter,
			LogicalTool:     "background-remove",
			ToolDisplayName: "ลบพื้นหลัง (Background Removal)",
			Parameters:      params,
			DependsOn:       []int{},
			Status:          model.StepStatusPending,
		})
		stepCounter++
	}

	// 2. Resolve Product Photo Step
	if hasProductPhoto {
		templateId := "tpl-prod-white-studio"
		templateName := "White Studio 1:1"
		aspectRatio := "1:1"

		if strings.Contains(lowerPrompt, "instagram") || strings.Contains(lowerPrompt, "ig") || strings.Contains(lowerPrompt, "4:5") {
			templateId = "tpl-prod-instagram-4-5"
			templateName = "Instagram Post 4:5"
			aspectRatio = "4:5"
		} else if strings.Contains(lowerPrompt, "tiktok") || strings.Contains(lowerPrompt, "story") || strings.Contains(lowerPrompt, "9:16") {
			templateId = "tpl-prod-tiktok-9-16"
			templateName = "TikTok / Story 9:16"
			aspectRatio = "9:16"
		} else if strings.Contains(lowerPrompt, "shopee") {
			templateId = "tpl-prod-shopee-1-1"
			templateName = "Shopee Marketplace 1:1"
			aspectRatio = "1:1"
		} else if strings.Contains(lowerPrompt, "lazada") {
			templateId = "tpl-prod-lazada-1-1"
			templateName = "Lazada Marketplace 1:1"
			aspectRatio = "1:1"
		}

		params := map[string]interface{}{
			"aspect_ratio": aspectRatio,
			"template_id":  templateId,
			"pack_size":    1,
		}

		dependsOn := []int{}
		if hasBgRemove {
			dependsOn = append(dependsOn, 1) // Depends on background-remove step output
		} else if initialImageURL != "" {
			params["image_url"] = initialImageURL
		}

		steps = append(steps, model.StudioWorkflowStep{
			StepIndex:       stepCounter,
			LogicalTool:     "product-photo",
			ToolDisplayName: fmt.Sprintf("รูปสินค้า (%s)", templateName),
			TemplateId:      templateId,
			TemplateName:    templateName,
			Parameters:      params,
			DependsOn:       dependsOn,
			Status:          model.StepStatusPending,
		})
		stepCounter++
	}

	// 3. Resolve Video Step
	if hasVideo {
		duration := 5
		if strings.Contains(lowerPrompt, "3 วิ") || strings.Contains(lowerPrompt, "3s") || strings.Contains(lowerPrompt, "3 sec") {
			duration = 3
		}

		params := map[string]interface{}{
			"duration":    duration,
			"resolution":  "720p",
			"quality":     "standard",
			"num_outputs": 1,
		}

		dependsOn := []int{}
		if len(steps) > 0 {
			// Depends on immediate preceding step output
			dependsOn = append(dependsOn, steps[len(steps)-1].StepIndex)
		} else if initialImageURL != "" {
			params["image_url"] = initialImageURL
		}

		steps = append(steps, model.StudioWorkflowStep{
			StepIndex:       stepCounter,
			LogicalTool:     "image-to-video",
			ToolDisplayName: fmt.Sprintf("วิดีโอสั้น %d วินาที (Image to Video)", duration),
			Parameters:      params,
			DependsOn:       dependsOn,
			Status:          model.StepStatusPending,
		})
		stepCounter++
	}

	// 4. Resolve Upscale Step if requested
	if hasUpscale {
		params := map[string]interface{}{
			"scale": 2,
		}
		dependsOn := []int{}
		if len(steps) > 0 {
			dependsOn = append(dependsOn, steps[len(steps)-1].StepIndex)
		} else if initialImageURL != "" {
			params["image_url"] = initialImageURL
		}

		steps = append(steps, model.StudioWorkflowStep{
			StepIndex:       stepCounter,
			LogicalTool:     "image-upscale",
			ToolDisplayName: "ขยายความละเอียด 2x (Image Upscale)",
			Parameters:      params,
			DependsOn:       dependsOn,
			Status:          model.StepStatusPending,
		})
		stepCounter++
	}

	// Safety Guard: Face Swap Requires Explicit Rights & Consent
	requiresConsent := false
	if hasFaceSwap {
		requiresConsent = true
		params := map[string]interface{}{}
		if initialImageURL != "" {
			params["image_url"] = initialImageURL
		}
		steps = append(steps, model.StudioWorkflowStep{
			StepIndex:       stepCounter,
			LogicalTool:     "face-swap",
			ToolDisplayName: "สลับใบหน้า (Face Swap - Requires Consent)",
			Parameters:      params,
			DependsOn:       []int{},
			Status:          model.StepStatusPending,
		})
		stepCounter++
	}

	// Step 2: Authoritative Server Quoting for Each Step
	totalCredits := 0
	totalQuota := 0

	for i := range steps {
		toolDef, err := model.GetStudioToolDefinition(steps[i].LogicalTool)
		if err != nil {
			toolDef, err = model.GetStudioToolDefinitionAnyStatus(steps[i].LogicalTool)
		}
		if err != nil {
			return nil, fmt.Errorf("tool definition not found for %s: %w", steps[i].LogicalTool, err)
		}

		quote, err := a.pricingEngine.CalculatePriceWithInputs(toolDef, steps[i].Parameters, 1.0)
		if err != nil {
			return nil, fmt.Errorf("failed calculating quote for step %d (%s): %w", steps[i].StepIndex, steps[i].LogicalTool, err)
		}

		quoteId := a.pricingEngine.SaveQuote(quote)
		steps[i].QuoteID = quoteId
		steps[i].EstimatedCredits = quote.ChargedCredits
		steps[i].EstimatedQuota = quote.ChargedQuota

		totalCredits += quote.ChargedCredits
		totalQuota += quote.ChargedQuota
	}

	stepsBytes, _ := json.Marshal(steps)
	planId := fmt.Sprintf("wf_%d_%s", common.GetTimestamp(), common.GetUUID()[:8])
	now := common.GetTimestamp()

	plan := &model.StudioWorkflowPlan{
		Id:                    planId,
		UserId:                userId,
		GoalPrompt:            cleanPrompt,
		Status:                model.WorkflowStatusDraft,
		StepsJSON:             string(stepsBytes),
		TotalEstimatedCredits: totalCredits,
		TotalEstimatedQuota:   totalQuota,
		RequiresConsent:       requiresConsent,
		ConsentConfirmed:      false,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if model.DB != nil {
		if err := model.DB.Create(plan).Error; err != nil {
			return nil, fmt.Errorf("failed persisting workflow plan: %w", err)
		}
	}

	return plan, nil
}

// GetWorkflowPlan retrieves an existing workflow plan.
func (a *StudioAssistantEngine) GetWorkflowPlan(planId string) (*model.StudioWorkflowPlan, []model.StudioWorkflowStep, error) {
	var plan model.StudioWorkflowPlan
	if model.DB == nil {
		return nil, nil, errors.New("database unavailable")
	}

	if err := model.DB.Where("id = ?", planId).First(&plan).Error; err != nil {
		return nil, nil, ErrWorkflowNotFound
	}

	var steps []model.StudioWorkflowStep
	if err := json.Unmarshal([]byte(plan.StepsJSON), &steps); err != nil {
		return nil, nil, fmt.Errorf("failed deserializing workflow steps: %w", err)
	}

	return &plan, steps, nil
}

// RequoteWorkflow refreshes pricing quotes for a workflow plan before execution.
func (a *StudioAssistantEngine) RequoteWorkflow(planId string) (*model.StudioWorkflowPlan, error) {
	plan, steps, err := a.GetWorkflowPlan(planId)
	if err != nil {
		return nil, err
	}

	totalCredits := 0
	totalQuota := 0

	for i := range steps {
		toolDef, err := model.GetStudioToolDefinition(steps[i].LogicalTool)
		if err != nil {
			toolDef, err = model.GetStudioToolDefinitionAnyStatus(steps[i].LogicalTool)
		}
		if err != nil {
			return nil, err
		}

		quote, err := a.pricingEngine.CalculatePriceWithInputs(toolDef, steps[i].Parameters, 1.0)
		if err != nil {
			return nil, err
		}

		quoteId := a.pricingEngine.SaveQuote(quote)
		steps[i].QuoteID = quoteId
		steps[i].EstimatedCredits = quote.ChargedCredits
		steps[i].EstimatedQuota = quote.ChargedQuota

		totalCredits += quote.ChargedCredits
		totalQuota += quote.ChargedQuota
	}

	stepsBytes, _ := json.Marshal(steps)
	plan.StepsJSON = string(stepsBytes)
	plan.TotalEstimatedCredits = totalCredits
	plan.TotalEstimatedQuota = totalQuota
	plan.UpdatedAt = common.GetTimestamp()

	if err := model.DB.Save(plan).Error; err != nil {
		return nil, err
	}

	return plan, nil
}

// ConfirmAndExecuteWorkflow sequentially executes the confirmed ToolPlan.
func (a *StudioAssistantEngine) ConfirmAndExecuteWorkflow(
	ctx context.Context,
	planId string,
	userId int,
	consentConfirmed bool,
	clientIP string,
	userIsAdmin bool,
) (*model.StudioWorkflowPlan, error) {
	plan, steps, err := a.GetWorkflowPlan(planId)
	if err != nil {
		return nil, err
	}

	if plan.UserId != userId && !userIsAdmin {
		return nil, model.ErrStudioForbiddenAccess
	}

	// 1. Safety Check: Face Swap Consent
	if plan.RequiresConsent {
		if !consentConfirmed && !plan.ConsentConfirmed {
			return nil, ErrConsentRequired
		}
		plan.ConsentConfirmed = true
	}

	// 2. Wallet Pre-Check
	var user model.User
	if err := model.DB.First(&user, userId).Error; err != nil {
		return nil, fmt.Errorf("failed fetching user wallet: %w", err)
	}

	availableCredits := user.Quota / QuotaPerCredit
	if user.Quota < plan.TotalEstimatedQuota {
		missing := plan.TotalEstimatedCredits - availableCredits
		if missing < 0 {
			missing = 0
		}
		return nil, &WorkflowInsufficientCreditError{
			RequiredCredits:  plan.TotalEstimatedCredits,
			AvailableCredits: availableCredits,
			MissingCredits:   missing,
		}
	}

	plan.Status = model.WorkflowStatusRunning
	_ = model.DB.Save(plan)

	// Step-by-Step Sequential Execution (Step-level reserve/settle)
	var lastOutputURL string
	outputsByStep := make(map[int]string)

	for i := range steps {
		step := &steps[i]

		// Resolve input assets from dependencies
		if len(step.DependsOn) > 0 {
			parentStepIdx := step.DependsOn[len(step.DependsOn)-1]
			parentOutput := outputsByStep[parentStepIdx]
			if parentOutput == "" {
				parentOutput = lastOutputURL
			}
			if parentOutput != "" {
				step.Parameters["image_url"] = parentOutput
			}
		}

		// Inject quote_id for quote expiration verification
		if step.QuoteID != "" {
			step.Parameters["quote_id"] = step.QuoteID
		}

		stepIdempKey := fmt.Sprintf("%s_step_%d", plan.Id, step.StepIndex)
		step.Status = model.StepStatusRunning

		// Execute step via StudioService (handles reserve, provider submit, settle/refund)
		job, err := a.studioSvc.SubmitJob(
			ctx,
			userId,
			step.LogicalTool,
			step.TemplateId,
			stepIdempKey,
			"", // Default provider routing
			step.Parameters,
			1.0,
			clientIP,
			userIsAdmin,
		)

		if err != nil {
			step.Status = model.StepStatusFailed
			step.ErrorMessage = err.Error()

			// Mark remaining steps as skipped
			for j := i + 1; j < len(steps); j++ {
				steps[j].Status = model.StepStatusSkipped
			}

			plan.Status = model.WorkflowStatusPartialFailed
			stepsBytes, _ := json.Marshal(steps)
			plan.StepsJSON = string(stepsBytes)
			plan.UpdatedAt = common.GetTimestamp()
			_ = model.DB.Save(plan)

			return plan, fmt.Errorf("workflow step %d (%s) failed: %w", step.StepIndex, step.LogicalTool, err)
		}

		// Step succeeded: extract output URL
		step.Status = model.StepStatusSuccess
		step.JobID = job.Id
		step.OutputResult = job.OutputResult

		var outMap map[string]interface{}
		if json.Unmarshal([]byte(job.OutputResult), &outMap) == nil {
			if u, ok := outMap["output_url"].(string); ok && u != "" {
				lastOutputURL = u
				outputsByStep[step.StepIndex] = u
			}
		}

		plan.TotalSettledCredits += step.EstimatedCredits
		plan.TotalSettledQuota += step.EstimatedQuota
	}

	plan.Status = model.WorkflowStatusCompleted
	stepsBytes, _ := json.Marshal(steps)
	plan.StepsJSON = string(stepsBytes)
	plan.UpdatedAt = common.GetTimestamp()
	_ = model.DB.Save(plan)

	return plan, nil
}

// RetryWorkflowStep retries only a specific failed step in a ToolPlan.
func (a *StudioAssistantEngine) RetryWorkflowStep(
	ctx context.Context,
	planId string,
	stepIndex int,
	userId int,
	clientIP string,
	userIsAdmin bool,
) (*model.StudioWorkflowPlan, error) {
	plan, steps, err := a.GetWorkflowPlan(planId)
	if err != nil {
		return nil, err
	}

	if plan.UserId != userId && !userIsAdmin {
		return nil, model.ErrStudioForbiddenAccess
	}

	var targetStep *model.StudioWorkflowStep
	var targetIdx int
	for i := range steps {
		if steps[i].StepIndex == stepIndex {
			targetStep = &steps[i]
			targetIdx = i
			break
		}
	}

	if targetStep == nil {
		return nil, fmt.Errorf("step index %d not found in plan", stepIndex)
	}

	// Resolve inputs from prior dependencies
	if len(targetStep.DependsOn) > 0 {
		parentStepIdx := targetStep.DependsOn[len(targetStep.DependsOn)-1]
		for _, prev := range steps {
			if prev.StepIndex == parentStepIdx && prev.Status == model.StepStatusSuccess {
				var outMap map[string]interface{}
				if json.Unmarshal([]byte(prev.OutputResult), &outMap) == nil {
					if u, ok := outMap["output_url"].(string); ok && u != "" {
						targetStep.Parameters["image_url"] = u
					}
				}
			}
		}
	}

	retryIdempKey := fmt.Sprintf("%s_step_%d_retry_%d", plan.Id, targetStep.StepIndex, common.GetTimestamp())
	targetStep.Status = model.StepStatusRunning
	targetStep.ErrorMessage = ""

	job, err := a.studioSvc.SubmitJob(
		ctx,
		userId,
		targetStep.LogicalTool,
		targetStep.TemplateId,
		retryIdempKey,
		"",
		targetStep.Parameters,
		1.0,
		clientIP,
		userIsAdmin,
	)

	if err != nil {
		targetStep.Status = model.StepStatusFailed
		targetStep.ErrorMessage = err.Error()
		stepsBytes, _ := json.Marshal(steps)
		plan.StepsJSON = string(stepsBytes)
		plan.UpdatedAt = common.GetTimestamp()
		_ = model.DB.Save(plan)
		return plan, fmt.Errorf("step %d retry failed: %w", targetStep.StepIndex, err)
	}

	targetStep.Status = model.StepStatusSuccess
	targetStep.JobID = job.Id
	targetStep.OutputResult = job.OutputResult
	plan.TotalSettledCredits += targetStep.EstimatedCredits
	plan.TotalSettledQuota += targetStep.EstimatedQuota

	// If this was the last step or all steps are now succeeded
	allDone := true
	for _, s := range steps {
		if s.Status != model.StepStatusSuccess {
			allDone = false
			break
		}
	}
	if allDone {
		plan.Status = model.WorkflowStatusCompleted
	} else {
		// Resume subsequent steps if any were skipped
		var lastURL string
		var outMap map[string]interface{}
		if json.Unmarshal([]byte(job.OutputResult), &outMap) == nil {
			if u, ok := outMap["output_url"].(string); ok {
				lastURL = u
			}
		}

		for j := targetIdx + 1; j < len(steps); j++ {
			nextStep := &steps[j]
			if lastURL != "" {
				nextStep.Parameters["image_url"] = lastURL
			}
			nextIdemp := fmt.Sprintf("%s_step_%d_resume_%d", plan.Id, nextStep.StepIndex, common.GetTimestamp())
			nextJob, nextErr := a.studioSvc.SubmitJob(ctx, userId, nextStep.LogicalTool, nextStep.TemplateId, nextIdemp, "", nextStep.Parameters, 1.0, clientIP, userIsAdmin)
			if nextErr != nil {
				nextStep.Status = model.StepStatusFailed
				nextStep.ErrorMessage = nextErr.Error()
				plan.Status = model.WorkflowStatusPartialFailed
				break
			}
			nextStep.Status = model.StepStatusSuccess
			nextStep.JobID = nextJob.Id
			nextStep.OutputResult = nextJob.OutputResult
			plan.TotalSettledCredits += nextStep.EstimatedCredits
			plan.TotalSettledQuota += nextStep.EstimatedQuota

			if json.Unmarshal([]byte(nextJob.OutputResult), &outMap) == nil {
				if u, ok := outMap["output_url"].(string); ok {
					lastURL = u
				}
			}
		}

		allDoneAfterResume := true
		for _, s := range steps {
			if s.Status != model.StepStatusSuccess {
				allDoneAfterResume = false
				break
			}
		}
		if allDoneAfterResume {
			plan.Status = model.WorkflowStatusCompleted
		}
	}

	stepsBytes, _ := json.Marshal(steps)
	plan.StepsJSON = string(stepsBytes)
	plan.UpdatedAt = common.GetTimestamp()
	_ = model.DB.Save(plan)

	return plan, nil
}
