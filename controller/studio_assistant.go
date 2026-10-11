package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type AssistantPlanRequest struct {
	Prompt          string `json:"prompt" binding:"required"`
	InitialImageURL string `json:"initial_image_url"`
}

type AssistantConfirmRequest struct {
	ConsentConfirmed bool `json:"consent_confirmed"`
}

type AssistantRetryRequest struct {
	StepIndex int `json:"step_index" binding:"required"`
}

// CreateAssistantPlan resolves natural language into an authoritative ToolPlan.
func CreateAssistantPlan(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "authentication required",
		})
		return
	}

	var req AssistantPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: prompt is required",
		})
		return
	}

	studioSvc := service.GetStudioService()
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.PlanFromPrompt(c.Request.Context(), userId, req.Prompt, req.InitialImageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "failed to generate toolplan: " + err.Error(),
		})
		return
	}

	var steps []model.StudioWorkflowStep
	_ = json.Unmarshal([]byte(plan.StepsJSON), &steps)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "toolplan generated successfully",
		"data": gin.H{
			"plan":  plan,
			"steps": steps,
		},
	})
}

// GetAssistantPlan retrieves an existing ToolPlan and steps.
func GetAssistantPlan(c *gin.Context) {
	userId := c.GetInt("id")
	role := c.GetInt("role")
	userIsAdmin := role == common.RoleAdminUser || role == common.RoleRootUser
	planId := strings.TrimSpace(c.Param("id"))

	if planId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "workflow plan id is required",
		})
		return
	}

	studioSvc := service.GetStudioService()
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, steps, err := assistantEngine.GetWorkflowPlan(planId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "workflow plan not found",
		})
		return
	}

	if plan.UserId != userId && !userIsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "forbidden: plan belongs to another user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"plan":  plan,
			"steps": steps,
		},
	})
}

// ConfirmAssistantPlan authorizes and executes the ToolPlan sequentially.
func ConfirmAssistantPlan(c *gin.Context) {
	userId := c.GetInt("id")
	role := c.GetInt("role")
	userIsAdmin := role == common.RoleAdminUser || role == common.RoleRootUser
	planId := strings.TrimSpace(c.Param("id"))

	var req AssistantConfirmRequest
	_ = c.ShouldBindJSON(&req)

	studioSvc := service.GetStudioService()
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.ConfirmAndExecuteWorkflow(
		c.Request.Context(),
		planId,
		userId,
		req.ConsentConfirmed,
		c.ClientIP(),
		userIsAdmin,
	)

	if err != nil {
		var insErr *service.WorkflowInsufficientCreditError
		if errors.As(err, &insErr) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"success": false,
				"error_code": "INSUFFICIENT_CREDITS",
				"message": insErr.Error(),
				"data": gin.H{
					"required_credits":  insErr.RequiredCredits,
					"available_credits": insErr.AvailableCredits,
					"missing_credits":   insErr.MissingCredits,
				},
			})
			return
		}

		if errors.Is(err, service.ErrConsentRequired) {
			c.JSON(http.StatusPreconditionRequired, gin.H{
				"success":    false,
				"error_code": "CONSENT_REQUIRED",
				"message":    err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    plan,
		})
		return
	}

	var steps []model.StudioWorkflowStep
	_ = json.Unmarshal([]byte(plan.StepsJSON), &steps)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "workflow executed successfully",
		"data": gin.H{
			"plan":  plan,
			"steps": steps,
		},
	})
}

// RetryAssistantPlanStep retries only a specific failed step in the ToolPlan.
func RetryAssistantPlanStep(c *gin.Context) {
	userId := c.GetInt("id")
	role := c.GetInt("role")
	userIsAdmin := role == common.RoleAdminUser || role == common.RoleRootUser
	planId := strings.TrimSpace(c.Param("id"))

	var req AssistantRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "step_index is required",
		})
		return
	}

	studioSvc := service.GetStudioService()
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.RetryWorkflowStep(
		c.Request.Context(),
		planId,
		req.StepIndex,
		userId,
		c.ClientIP(),
		userIsAdmin,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    plan,
		})
		return
	}

	var steps []model.StudioWorkflowStep
	_ = json.Unmarshal([]byte(plan.StepsJSON), &steps)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "step retried successfully",
		"data": gin.H{
			"plan":  plan,
			"steps": steps,
		},
	})
}

// RequoteAssistantPlan refreshes quotes for a ToolPlan before execution.
func RequoteAssistantPlan(c *gin.Context) {
	planId := strings.TrimSpace(c.Param("id"))

	studioSvc := service.GetStudioService()
	assistantEngine := studioSvc.GetAssistantEngine()

	plan, err := assistantEngine.RequoteWorkflow(planId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	var steps []model.StudioWorkflowStep
	_ = json.Unmarshal([]byte(plan.StepsJSON), &steps)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "workflow requoted successfully",
		"data": gin.H{
			"plan":  plan,
			"steps": steps,
		},
	})
}
