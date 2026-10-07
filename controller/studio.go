package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// --- Public Catalog Endpoints (Section 7 & 8) ---

// GetStudioTools returns all active, public tools for the /studio catalog.
func GetStudioTools(c *gin.Context) {
	tools, err := model.ListPublicStudioTools()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to list studio tools: " + err.Error(),
		})
		return
	}

	studioSvc := service.GetStudioService()
	falProvider := studioSvc.GetProvider("fal")
	falBlocked := false
	if falProvider != nil {
		if fal, ok := falProvider.(*service.FalProvider); ok {
			if fal.ValidateConfiguration() != nil {
				falBlocked = true
			}
		}
	}

	replicateProvider := studioSvc.GetProvider("replicate")
	replicateBlocked := false
	if replicateProvider != nil {
		if rep, ok := replicateProvider.(*service.ReplicateProvider); ok {
			if rep.ValidateConfiguration() != nil {
				replicateBlocked = true
			}
		}
	}

	// Update live tool state if provider is blocked
	for i := range tools {
		if tools[i].PrimaryProvider == "fal" && falBlocked {
			tools[i].Status = model.StudioToolStateOperatorBlocked
		} else if tools[i].PrimaryProvider == "replicate" && replicateBlocked {
			tools[i].Status = model.StudioToolStateOperatorBlocked
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tools,
	})
}

// GetStudioToolBySlug returns details of a single tool.
func GetStudioToolBySlug(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "tool slug required"})
		return
	}

	tool, err := model.GetStudioToolDefinition(slug)
	if err != nil {
		if errors.Is(err, model.ErrStudioToolNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "tool not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	studioSvc := service.GetStudioService()
	falBlocked := false
	if falProvider := studioSvc.GetProvider("fal"); falProvider != nil {
		if fal, ok := falProvider.(*service.FalProvider); ok && fal.ValidateConfiguration() != nil {
			falBlocked = true
		}
	}
	replicateBlocked := false
	if repProvider := studioSvc.GetProvider("replicate"); repProvider != nil {
		if rep, ok := repProvider.(*service.ReplicateProvider); ok && rep.ValidateConfiguration() != nil {
			replicateBlocked = true
		}
	}
	if tool.PrimaryProvider == "fal" && falBlocked {
		tool.Status = model.StudioToolStateOperatorBlocked
	} else if tool.PrimaryProvider == "replicate" && replicateBlocked {
		tool.Status = model.StudioToolStateOperatorBlocked
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tool,
	})
}

// GetStudioTemplates returns curated templates for a tool or category.
func GetStudioTemplates(c *gin.Context) {
	toolId := strings.TrimSpace(c.Query("tool_id"))
	category := strings.TrimSpace(c.Query("category"))

	templates, err := model.ListStudioTemplates(toolId, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed listing templates: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    templates,
	})
}

// --- Public Catalog & Quote Endpoints (Section 7, 8 & 14) ---

type QuoteStudioJobRequest struct {
	ToolId      string                 `json:"tool_id" binding:"required"`
	TemplateId  string                 `json:"template_id"`
	InputParams map[string]interface{} `json:"input_params"`
}

// QuoteStudioJob returns a verifiable, 15-minute TTL pricing quote prior to submission (Section 14).
func QuoteStudioJob(c *gin.Context) {
	var req QuoteStudioJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid quote request payload: " + err.Error(),
		})
		return
	}

	toolDef, err := model.GetStudioToolDefinition(req.ToolId)
	if err != nil {
		if errors.Is(err, model.ErrStudioToolNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "tool not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	studioSvc := service.GetStudioService()
	pricingEngine := studioSvc.GetPricingEngine()

	planMultiplier := 1.0
	// If user is authenticated, check custom tier if available
	if userId := c.GetInt("id"); userId > 0 {
		planMultiplier = 1.0
	}

	snapshot, err := pricingEngine.CalculatePriceWithInputs(toolDef, req.InputParams, planMultiplier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	quoteId := pricingEngine.SaveQuote(snapshot)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quote_id":                    quoteId,
			"tool_id":                     toolDef.Id,
			"tool_name":                   toolDef.DisplayName,
			"pricing_version":             snapshot.PricingVersion,
			"provider":                    snapshot.Provider,
			"provider_route":              snapshot.Provider,
			"provider_model":              snapshot.ProviderModel,
			"provider_estimated_cost_usd": snapshot.ProviderEstimatedCostUSD,
			"provider_cost_basis":         snapshot.ProviderCostBasis,
			"cost_basis":                  snapshot.ProviderCostBasis,
			"target_margin":               snapshot.TargetMargin,
			"calculated_sell_usd":         snapshot.CalculatedSellUSD,
			"sell_usd_equivalent":         snapshot.CalculatedSellUSD,
			"calculated_credits":          snapshot.CalculatedCredits,
			"charged_credits":             snapshot.ChargedCredits,
			"estimated_credits":           snapshot.ChargedCredits,
			"charged_quota":               snapshot.ChargedQuota,
			"plan_multiplier":             snapshot.PlanMultiplier,
			"quoted_at":                   snapshot.QuotedAt,
			"expires_at":                  snapshot.ExpiresAt,
		},
	})
}

// StudioWebhook handles provider completion webhooks with dual-resolution safety.
func StudioWebhook(c *gin.Context) {
	provider := strings.TrimSpace(c.Param("provider"))
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "missing provider parameter"})
		return
	}

	var payload struct {
		RequestId     string `json:"request_id"`
		ProviderJobId string `json:"provider_job_id"`
		Status        string `json:"status"` // "COMPLETED", "FAILED", "OK", "ERROR"
		OutputURL     string `json:"output_url"`
		Error         string `json:"error"`
		Payload       struct {
			Image struct {
				URL string `json:"url"`
			} `json:"image"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
			Video struct {
				URL string `json:"video"`
			} `json:"video"`
		} `json:"payload"`
		Image         struct {
			URL string `json:"url"`
		} `json:"image"`
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
		Video struct {
			URL string `json:"video"`
		} `json:"video"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid webhook payload: " + err.Error()})
		return
	}

	jobId := payload.ProviderJobId
	if jobId == "" {
		jobId = payload.RequestId
	}
	if jobId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "missing provider job / request id"})
		return
	}

	outputURL := payload.OutputURL
	if outputURL == "" {
		if len(payload.Payload.Images) > 0 && payload.Payload.Images[0].URL != "" {
			outputURL = payload.Payload.Images[0].URL
		} else if payload.Payload.Image.URL != "" {
			outputURL = payload.Payload.Image.URL
		} else if payload.Payload.Video.URL != "" {
			outputURL = payload.Payload.Video.URL
		} else if len(payload.Images) > 0 && payload.Images[0].URL != "" {
			outputURL = payload.Images[0].URL
		} else if payload.Image.URL != "" {
			outputURL = payload.Image.URL
		} else if payload.Video.URL != "" {
			outputURL = payload.Video.URL
		}
	}

	studioSvc := service.GetStudioService()
	job, err := studioSvc.HandleWebhook(c.Request.Context(), provider, jobId, payload.Status, outputURL, payload.Error)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "webhook processed successfully",
		"data":    job,
	})
}

// --- Authenticated User Endpoints (Section 19 & 20) ---

type CreateStudioJobRequest struct {
	ToolId         string                 `json:"tool_id" binding:"required"`
	TemplateId     string                 `json:"template_id"`
	IdempotencyKey string                 `json:"idempotency_key"`
	Provider       string                 `json:"provider"`
	InputParams    map[string]interface{} `json:"input_params" binding:"required"`
}

// CreateStudioJob initiates an atomic reservation and provider execution.
func CreateStudioJob(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}

	var req CreateStudioJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request payload: " + err.Error()})
		return
	}

	// Resolve idempotency key
	idempKey := strings.TrimSpace(req.IdempotencyKey)
	if idempKey == "" {
		idempKey = strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	}
	if idempKey == "" {
		idempKey = common.GetUUID()
	}

	// Security Defense: Validate input parameters (SSRF, video base64, size limits)
	for key, val := range req.InputParams {
		if strVal, ok := val.(string); ok {
			// Disable video base64 uploads (Section 21)
			if strings.HasPrefix(strVal, "data:video/") {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": service.ErrBase64Video.Error(),
				})
				return
			}

			// Validate maximum base64 payload size (20MB encoded ~ 15MB binary)
			if strings.HasPrefix(strVal, "data:image/") && len(strVal) > 20*1024*1024 {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": service.ErrFileSizeExceeded.Error(),
				})
				return
			}

			// SSRF Defense: validate any external URLs (Section 30)
			if strings.HasSuffix(key, "_url") || key == "url" || strings.HasPrefix(strVal, "http://") || strings.HasPrefix(strVal, "https://") {
				if strings.HasPrefix(strVal, "http") {
					if err := service.ValidateExternalURL(strVal); err != nil {
						c.JSON(http.StatusBadRequest, gin.H{
							"success": false,
							"message": fmt.Sprintf("invalid %s: %s", key, err.Error()),
						})
						return
					}
				}
			}
		}
	}

	studioSvc := service.GetStudioService()
	planMultiplier := 1.0 // Standard rate
	isAdmin := c.GetInt("role") >= common.RoleAdminUser

	job, err := studioSvc.SubmitJob(
		c.Request.Context(),
		userId,
		req.ToolId,
		req.TemplateId,
		idempKey,
		req.Provider,
		req.InputParams,
		planMultiplier,
		c.ClientIP(),
		isAdmin,
	)

	if err != nil {
		if errors.Is(err, service.ErrInsufficientQuota) {
			// Section 25: Insufficient Credit UX with exact breakdown
			toolDef, _ := model.GetStudioToolDefinition(req.ToolId)
			requiredCredits := 10
			if toolDef != nil {
				requiredCredits = toolDef.CreditCost
			}
			userQuota, _ := model.GetUserQuota(userId, true)
			currentCredits := service.FormatToraCredits(userQuota)
			missingCredits := requiredCredits - currentCredits
			if missingCredits < 0 {
				missingCredits = 0
			}

			c.JSON(http.StatusPaymentRequired, gin.H{
				"success":    false,
				"error_code": "INSUFFICIENT_CREDITS",
				"message":    "เครดิต Tora Credits ไม่เพียงพอสำหรับการสร้างงานนี้",
				"data": gin.H{
					"required_credits": requiredCredits,
					"current_credits":  currentCredits,
					"missing_credits":   missingCredits,
					"tool_id":          req.ToolId,
					"template_id":      req.TemplateId,
				},
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "job submitted successfully",
		"data":    job,
	})
}

// GetStudioJobDetail retrieves status and results with IDOR verification (Section 20).
func GetStudioJobDetail(c *gin.Context) {
	userId := c.GetInt("id")
	jobId := strings.TrimSpace(c.Param("id"))
	if jobId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "job ID required"})
		return
	}

	isAdmin := c.GetInt("role") >= common.RoleAdminUser
	studioSvc := service.GetStudioService()

	job, err := studioSvc.PollJob(c.Request.Context(), jobId, userId, isAdmin)
	if err != nil {
		if errors.Is(err, model.ErrStudioForbiddenAccess) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "access denied: you do not own this job"})
			return
		}
		if errors.Is(err, model.ErrStudioJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    job,
	})
}

// ListStudioUserJobs returns paginated jobs for the current user.
func ListStudioUserJobs(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	jobs, total, err := model.ListUserStudioJobs(userId, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"jobs":      jobs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// CancelStudioJob cancels an active job and refunds reservation.
func CancelStudioJob(c *gin.Context) {
	userId := c.GetInt("id")
	jobId := strings.TrimSpace(c.Param("id"))
	isAdmin := c.GetInt("role") >= common.RoleAdminUser

	studioSvc := service.GetStudioService()
	job, err := studioSvc.CancelJob(c.Request.Context(), jobId, userId, isAdmin)
	if err != nil {
		if errors.Is(err, model.ErrStudioForbiddenAccess) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "job cancelled",
		"data":    job,
	})
}

// --- Attribution & Conversion Funnel Endpoints (Queue 3) ---

type StudioAttributionPayload struct {
	EventType string `json:"event_type" binding:"required"` // "insufficient_credit", "buy_credit_click", "purchase_return", "generation_after_purchase"
	ToolId    string `json:"tool_id" binding:"required"`
	Credits   int    `json:"credits"`
	SessionId string `json:"session_id"`
}

// RecordStudioAttribution stores non-sensitive conversion milestones.
func RecordStudioAttribution(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}

	var req StudioAttributionPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid attribution payload: " + err.Error()})
		return
	}

	_ = model.RecordStudioConversionEvent(userId, req.EventType, req.ToolId, req.Credits, req.SessionId)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// --- Admin Studio Telemetry Endpoint (Section 34 & 35) ---

// GetStudioAdminTelemetry aggregates business and operational metrics.
func GetStudioAdminTelemetry(c *gin.Context) {
	if model.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "database not initialized"})
		return
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()

	var jobsToday int64
	var succeededToday int64
	var failedToday int64
	var totalJobs int64
	var succeededTotal int64
	var failedTotal int64
	var studioUsers int64
	var refundsCount int64

	model.DB.Model(&model.StudioToolJob{}).Where("created_at >= ?", startOfDay).Count(&jobsToday)
	model.DB.Model(&model.StudioToolJob{}).Where("created_at >= ? AND status = ?", startOfDay, model.StudioJobStatusSucceeded).Count(&succeededToday)
	model.DB.Model(&model.StudioToolJob{}).Where("created_at >= ? AND status = ?", startOfDay, model.StudioJobStatusFailed).Count(&failedToday)

	model.DB.Model(&model.StudioToolJob{}).Count(&totalJobs)
	model.DB.Model(&model.StudioToolJob{}).Where("status = ?", model.StudioJobStatusSucceeded).Count(&succeededTotal)
	model.DB.Model(&model.StudioToolJob{}).Where("status = ?", model.StudioJobStatusFailed).Count(&failedTotal)
	model.DB.Model(&model.StudioToolJob{}).Distinct("user_id").Count(&studioUsers)
	model.DB.Model(&model.StudioJobEvent{}).Where("event_type LIKE ?", "%REFUND%").Count(&refundsCount)

	var creditsConsumed int64
	var totalCostUSD float64

	type CostAgg struct {
		TotalQuota int64
		TotalCost  float64
	}
	var agg CostAgg
	_ = model.DB.Model(&model.StudioCostSnapshot{}).
		Select("COALESCE(SUM(quota_cost), 0) as total_quota, COALESCE(SUM(cost_usd), 0) as total_cost").
		Scan(&agg)

	creditsConsumed = agg.TotalQuota / service.QuotaPerCredit
	totalCostUSD = agg.TotalCost
	revenueUSD := float64(agg.TotalQuota) / common.QuotaPerUnit
	grossProfitUSD := revenueUSD - totalCostUSD
	marginPercent := 0.0
	if revenueUSD > 0 {
		marginPercent = (grossProfitUSD / revenueUSD) * 100.0
	}

	// Conversion funnel metrics
	var insufficientCreditEvents int64
	var buyCreditClicks int64
	var purchaseReturns int64
	var generationAfterPurchases int64

	model.DB.Model(&model.StudioConversionEvent{}).Where("event_type = ?", "insufficient_credit").Count(&insufficientCreditEvents)
	model.DB.Model(&model.StudioConversionEvent{}).Where("event_type = ?", "buy_credit_click").Count(&buyCreditClicks)
	model.DB.Model(&model.StudioConversionEvent{}).Where("event_type = ?", "purchase_return").Count(&purchaseReturns)
	model.DB.Model(&model.StudioConversionEvent{}).Where("event_type = ?", "generation_after_purchase").Count(&generationAfterPurchases)

	// Product Studio specific funnel metrics (Queue 4)
	var prodVisitors int64
	var prodGenerateClicks int64
	var prodInsufficientCredit int64
	var prodBuyCreditClicks int64
	var prodSuccessfulTopups int64
	var prodGenerationAfterTopup int64
	var prodRepeatGenerations int64

	model.DB.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND (event_type = ? OR event_type = ?)", "product-photo", "tool_visit", "tool_view").Count(&prodVisitors)
	model.DB.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "generate_click").Count(&prodGenerateClicks)
	model.DB.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "insufficient_credit").Count(&prodInsufficientCredit)
	model.DB.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "buy_credit_click").Count(&prodBuyCreditClicks)
	model.DB.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND (event_type = ? OR event_type = ?)", "product-photo", "purchase_return", "successful_topup").Count(&prodSuccessfulTopups)
	model.DB.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND (event_type = ? OR event_type = ?)", "product-photo", "generation_after_purchase", "generation_after_topup").Count(&prodGenerationAfterTopup)
	model.DB.Model(&model.StudioConversionEvent{}).Where("tool_id = ? AND event_type = ?", "product-photo", "repeat_generation").Count(&prodRepeatGenerations)

	falConfigured := osGetEnv("FAL_KEY") != "" || osGetEnv("FAL_API_KEY") != ""
	providerStatus := gin.H{
		"fal": gin.H{
			"name":        "fal.ai",
			"status":      boolToStatus(falConfigured),
			"canary_tool": "background-remove",
		},
		"mock": gin.H{
			"name":   "deterministic_mock",
			"status": "ACTIVE",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"studio_users":                 studioUsers,
			"jobs":                         totalJobs,
			"jobs_today":                   jobsToday,
			"succeeded_today":              succeededToday,
			"failed_today":                 failedToday,
			"succeeded_total":              succeededTotal,
			"failed_total":                 failedTotal,
			"refunds":                      refundsCount,
			"credits_spent":                creditsConsumed,
			"credits_consumed":             creditsConsumed,
			"insufficient_credit_events":   insufficientCreditEvents,
			"buy_credit_clicks":            buyCreditClicks,
			"studio_originated_topups":     purchaseReturns,
			"generation_after_purchases":   generationAfterPurchases,
			"product_studio": gin.H{
				"tool_visitors":              prodVisitors,
				"generate_clicks":            prodGenerateClicks,
				"insufficient_credit":        prodInsufficientCredit,
				"buy_credit_clicks":          prodBuyCreditClicks,
				"successful_topups":          prodSuccessfulTopups,
				"generation_after_topup":     prodGenerationAfterTopup,
				"repeat_generations":         prodRepeatGenerations,
			},
			"provider_cost":                totalCostUSD,
			"sell_value":                   revenueUSD,
			"gross_profit":                 grossProfitUSD,
			"gross_margin":                 marginPercent,
			"estimated_revenue_usd":        revenueUSD,
			"estimated_cost_usd":           totalCostUSD,
			"estimated_profit_usd":         grossProfitUSD,
			"gross_margin_percent":         marginPercent,
			"providers":                    providerStatus,
		},
	})
}

func boolToStatus(b bool) string {
	if b {
		return "ACTIVE"
	}
	return "OPERATOR_BLOCKED"
}

func osGetEnv(k string) string {
	return strings.TrimSpace(os.Getenv(k))
}

// --- SSR Public Landing Pages (Queue 3 SEO & Content) ---

type ToolUseCase struct {
	Title string
	Desc  string
}

type ToolSEOMetadata struct {
	UseCases     []ToolUseCase
	RelatedSlugs []string
}

var toolSEORegistry = map[string]ToolSEOMetadata{
	"background-remove": {
		UseCases: []ToolUseCase{
			{"ภาพสินค้า E-commerce & Marketplace", "ตัดพื้นหลังสินค้าเป็นสีขาวหรือโปร่งใสทันที พร้อมลงขายบน Shopee, Lazada และ TikTok Shop"},
			{"ภาพถ่ายบุคคล & ทำรูปติดบัตร", "แยกเส้นผมและขอบเสื้อผ้าอย่างแม่นยำด้วยโมเดล BiRefNet สำหรับรูปโปรไฟล์และเอกสารทางการ"},
			{"งานกราฟิกดีไซน์ & สื่อโฆษณา", "ไดคัทวัตถุอย่างรวดเร็ว เพื่อนำไปจัดวางบนแบนเนอร์และภาพกราฟิกโปรโมชัน"},
		},
		RelatedSlugs: []string{"image-upscale", "product-photo", "image-generator"},
	},
	"image-upscale": {
		UseCases: []ToolUseCase{
			{"ขยายภาพความละเอียดสูงสำหรับงานพิมพ์", "อัปสเกลภาพขึ้น 4 เท่าแบบ 4K โดยไม่แตก รักษาเส้นสายคมชัดระดับสตูดิโอ"},
			{"กู้คืนภาพถ่ายเก่าและภาพความละเอียดต่ำ", "ฟื้นฟูรายละเอียดพื้นผิวและ Texture อย่างเป็นธรรมชาติด้วย Clarity Upscaler"},
			{"เพิ่มความคมชัดสำหรับงานเว็บไซต์และจอเรตินา", "ยกระดับภาพกราฟิกให้ดูพรีเมียมบนหน้าจอแสดงผลความละเอียดสูงทุกขนาด"},
		},
		RelatedSlugs: []string{"background-remove", "image-generator", "image-extend"},
	},
	"image-generator": {
		UseCases: []ToolUseCase{
			{"สร้างภาพคอนเทนต์สำหรับโซเชียลมีเดีย", "สร้างสรรค์ผลงานภาพเสมือนจริงด้วย Flux.1 Schnell จากคำบรรยายภาษาไทยและอังกฤษ"},
			{"ออกแบบ Concept Art & Moodboard", "ระดมความคิดและทดลองสไตล์ศิลปะ ตัวละคร หรือทิวทัศน์ได้อย่างรวดเร็ว"},
			{"สื่อการตลาดและภาพประกอบโฆษณา", "ได้ภาพที่มีเอกลักษณ์เฉพาะตัว ไม่ซ้ำใคร ช่วยเพิ่มการมีส่วนร่วมและยอดคลิก"},
		},
		RelatedSlugs: []string{"image-upscale", "background-remove", "image-extend"},
	},
	"image-generate": {
		UseCases: []ToolUseCase{
			{"สร้างภาพคอนเทนต์สำหรับโซเชียลมีเดีย", "สร้างสรรค์ผลงานภาพเสมือนจริงด้วย Flux.1 Schnell จากคำบรรยายภาษาไทยและอังกฤษ"},
			{"ออกแบบ Concept Art & Moodboard", "ระดมความคิดและทดลองสไตล์ศิลปะ ตัวละคร หรือทิวทัศน์ได้อย่างรวดเร็ว"},
			{"สื่อการตลาดและภาพประกอบโฆษณา", "ได้ภาพที่มีเอกลักษณ์เฉพาะตัว ไม่ซ้ำใคร ช่วยเพิ่มการมีส่วนร่วมและยอดคลิก"},
		},
		RelatedSlugs: []string{"image-upscale", "background-remove", "image-extend"},
	},
	"product-photo": {
		UseCases: []ToolUseCase{
			{"Shopee & Lazada ร้านค้าออนไลน์ (1:1 Square)", "จัดฉากขาว สตูดิโอพรีเมียม หรือมินิมอล ถูกต้องตามเกณฑ์ภาพหน้าปก Shopee/Lazada ดึงดูด CTR บนหน้า Search Feed"},
			{"TikTok Shop & วิดีโอปักตะกร้า (9:16 Vertical)", "สร้างภาพปกสินค้าแนวตั้ง 9:16 ดึงดูดสายตา หยุดนิ้วโป้งลูกค้าภายใน 1 วินาทีแรก เพิ่มอัตราการคลิกสั่งซื้อ"},
			{"Instagram Merchants & แคตตาล็อก (4:5 Feed & Story)", "จัดฉากถ่ายภาพสินค้าพร้อม Mood & Tone สไตล์แมกกาซีน เพิ่มความน่าเชื่อถือให้แบรนด์และกระตุ้นยอดขายบนโซเชียล"},
			{"SME & พ่อค้าแม่ค้าออนไลน์ ประหยัดต้นทุน 90%", "ถ่ายรูปสินค้าจากมือถือ อัปโหลด จัดฉากแสงระดับโปรได้ใน 10 วินาที โดยไม่ต้องจ้างสตูดิโอถ่ายภาพหลักหมื่นบาท"},
		},
		RelatedSlugs: []string{"background-remove", "image-upscale", "object-eraser"},
	},
	"object-eraser": {
		UseCases: []ToolUseCase{
			{"ลบลายน้ำ วันที่ และข้อความบนรูปภาพ", "ลบข้อความที่ไม่ต้องการออกอย่างเรียบเนียน เติมเต็มพื้นหลังให้กลมกลืน"},
			{"ลบคนและสิ่งแปลกปลอมในภาพถ่าย", "แก้ไขรูปภาพท่องเที่ยวและภาพครอบครัว กำจัด Photobomb ได้อย่างสมบูรณ์แบบ"},
			{"รีทัชรูปสินค้าและแก้ไขจุดบกพร่อง", "ลบฝุ่น ริ้วรอย หรือรอยเปื้อนบนตัวสินค้าเพื่อความสวยงามขั้นสูงสุด"},
		},
		RelatedSlugs: []string{"background-remove", "image-upscale", "image-extend"},
	},
	"object-erase": {
		UseCases: []ToolUseCase{
			{"ลบลายน้ำ วันที่ และข้อความบนรูปภาพ", "ลบข้อความที่ไม่ต้องการออกอย่างเรียบเนียน เติมเต็มพื้นหลังให้กลมกลืน"},
			{"ลบคนและสิ่งแปลกปลอมในภาพถ่าย", "แก้ไขรูปภาพท่องเที่ยวและภาพครอบครัว กำจัด Photobomb ได้อย่างสมบูรณ์แบบ"},
			{"รีทัชรูปสินค้าและแก้ไขจุดบกพร่อง", "ลบฝุ่น ริ้วรอย หรือรอยเปื้อนบนตัวสินค้าเพื่อความสวยงามขั้นสูงสุด"},
		},
		RelatedSlugs: []string{"background-remove", "image-upscale", "image-extend"},
	},
	"image-extend": {
		UseCases: []ToolUseCase{
			{"ขยายภาพเป็นสัดส่วน Story 9:16 หรือ Feed", "ปรับเปลี่ยนอัตราส่วนภาพแนวนอนเป็นแนวตั้งสำหรับ Instagram Reels และ TikTok"},
			{"เติมเต็มฉากหลังสำหรับแบนเนอร์เว็บไซต์", "ขยายขอบภาพออกด้านข้างอย่างไร้รอยต่อ เติมเต็มบรรยากาศรอบข้างอย่างสมจริง"},
			{"จัดองค์ประกอบภาพใหม่เพื่อเพิ่มพื้นที่ข้อความ", "เพิ่มพื้นที่ว่างบนภาพสำหรับใส่พาดหัวโฆษณาและโลโก้ได้อย่างสวยงาม"},
		},
		RelatedSlugs: []string{"image-generator", "image-upscale", "background-remove"},
	},
	"image-to-video": {
		UseCases: []ToolUseCase{
			{"ขยับภาพสินค้าให้น่าสนใจบน TikTok & Shopee Video", "แปลงภาพนิ่งสินค้าให้มีมูฟเมนต์เป็นธรรมชาติ ช่วยหยุดสายตาลูกค้าและเพิ่มยอดขายอย่างก้าวกระโดด"},
			{"เปลี่ยนภาพนิ่งโปรโมชันเป็นวิดีโอ 5 วินาทีหยุดสายตา", "สร้างคลิปวิดีโอสั้นระดับภาพยนตร์ด้วยโมเดล Wan 2.2 จากภาพถ่ายเพียงใบเดียว"},
			{"Motion Graphics สำหรับโฆษณาโซเชียลมีเดีย", "สร้างความน่าตื่นตาตื่นใจให้คอนเทนต์ Facebook, Instagram Reels และ YouTube Shorts ได้อย่างง่ายดาย"},
		},
		RelatedSlugs: []string{"product-photo", "image-generate", "image-upscale"},
	},
}

// RenderStudioToolLandingPage serves indexable, crawlable HTML pages for high-value tools.
func RenderStudioToolLandingPage(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	canonicalBase := common.GetCanonicalBaseURL()

	tool, err := model.GetStudioToolDefinition(slug)
	if err != nil {
		c.Redirect(http.StatusFound, "/studio")
		return
	}

	meta, hasMeta := toolSEORegistry[slug]
	if !hasMeta {
		meta, hasMeta = toolSEORegistry[tool.Slug]
	}
	if !hasMeta {
		meta, _ = toolSEORegistry[tool.Id]
	}

	canonicalURL := fmt.Sprintf("%s/tools/%s", canonicalBase, tool.Slug)

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="th" class="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>` + tool.DisplayName + ` | Tora AI Studio</title>
  <meta name="description" content="` + tool.Description + ` ใช้ Tora Credits เพียง ` + strconv.Itoa(tool.CreditCost) + ` เครดิต (฿` + fmt.Sprintf("%.2f", float64(tool.CreditCost)*0.07) + `)">
  <link rel="canonical" href="` + canonicalURL + `">
  <meta property="og:type" content="website">
  <meta property="og:title" content="` + tool.DisplayName + ` — Tora AI Studio">
  <meta property="og:description" content="` + tool.Description + `">
  <meta property="og:url" content="` + canonicalURL + `">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="` + tool.DisplayName + ` — Tora AI Studio">
  <meta name="twitter:description" content="` + tool.Description + `">
  <script src="https://cdn.tailwindcss.com"></script>
  <script type="application/ld+json">
  {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    "name": "` + tool.DisplayName + `",
    "applicationCategory": "MultimediaApplication",
    "operatingSystem": "Web",
    "offers": {
      "@type": "Offer",
      "price": "` + fmt.Sprintf("%.2f", float64(tool.CreditCost)*0.07) + `",
      "priceCurrency": "THB"
    },
    "description": "` + tool.Description + `"
  }
  </script>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen flex flex-col font-sans">
  <nav class="border-b border-slate-800 bg-slate-900/80 backdrop-blur px-6 py-4 flex items-center justify-between">
    <div class="flex items-center space-x-3">
      <a href="/" class="text-xl font-bold text-white tracking-wider">TORA<span class="text-blue-500">.AI</span></a>
      <span class="text-xs bg-blue-500/10 text-blue-400 px-2.5 py-0.5 rounded-full border border-blue-500/20 font-semibold">STUDIO</span>
    </div>
    <div class="flex items-center space-x-4">
      <a href="/studio" class="text-sm text-slate-300 hover:text-white transition">คลังเครื่องมือทั้งหมด</a>
      <a href="/pricing" class="text-sm text-slate-300 hover:text-white transition">เติมเครดิต</a>
      <a href="/studio?tool=` + tool.Id + `&tab=playground" class="text-sm bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded-lg font-medium shadow-lg shadow-blue-500/20 transition">เริ่มใช้งานทันที</a>
    </div>
  </nav>

  <main class="flex-1 max-w-5xl mx-auto px-6 py-12">
    <!-- Hero Section -->
    <div class="text-center max-w-3xl mx-auto mb-14">
      <div class="inline-flex items-center space-x-2 bg-slate-900 border border-slate-800 px-3 py-1 rounded-full text-xs text-slate-400 mb-4">
        <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
        <span>หมวดหมู่: ` + strings.ToUpper(tool.Category) + `</span>
      </div>
      <h1 class="text-4xl md:text-5xl font-extrabold text-white mb-4 tracking-tight">` + tool.DisplayName + `</h1>
      <p class="text-lg text-slate-300 mb-6 leading-relaxed">` + tool.Description + `</p>
      <div class="flex flex-wrap items-center justify-center gap-4">
        <a href="/studio?tool=` + tool.Id + `&tab=playground" class="bg-blue-600 hover:bg-blue-500 text-white px-8 py-3.5 rounded-xl font-semibold shadow-xl shadow-blue-600/25 transition">
          เริ่มใช้งานใน Tora Studio
        </a>
        <div class="bg-slate-900 border border-slate-800 px-5 py-3 rounded-xl text-sm text-slate-300 flex items-center space-x-2">
          <span>อัตราค่าบริการ:</span>
          <strong class="text-blue-400 font-bold">` + strconv.Itoa(tool.CreditCost) + ` Tora Credits</strong>
          <span class="text-xs text-slate-500">(≈ ฿` + fmt.Sprintf("%.2f", float64(tool.CreditCost)*0.07) + `)</span>
        </div>
      </div>
    </div>

    <!-- Use Cases Section -->
    <div class="mb-16">
      <h2 class="text-2xl font-bold text-white mb-6 text-center">ตัวอย่างการนำไปใช้งานจริง (Use Cases)</h2>
      <div class="grid md:grid-cols-3 gap-6">`)

	if len(meta.UseCases) > 0 {
		for i, uc := range meta.UseCases {
			badgeNum := strconv.Itoa(i + 1)
			sb.WriteString(`
        <div class="bg-slate-900/60 border border-slate-800/80 rounded-2xl p-6 hover:border-slate-700 transition">
          <div class="w-9 h-9 rounded-xl bg-blue-500/10 flex items-center justify-center text-blue-400 mb-4 font-bold text-sm">` + badgeNum + `</div>
          <h3 class="text-base font-bold text-white mb-2">` + uc.Title + `</h3>
          <p class="text-sm text-slate-400 leading-relaxed">` + uc.Desc + `</p>
        </div>`)
		}
	} else {
		sb.WriteString(`
        <div class="bg-slate-900/60 border border-slate-800/80 rounded-2xl p-6">
          <div class="w-9 h-9 rounded-xl bg-blue-500/10 flex items-center justify-center text-blue-400 mb-4 font-bold text-sm">1</div>
          <h3 class="text-base font-bold text-white mb-2">มาตรฐานระดับสตูดิโอ</h3>
          <p class="text-sm text-slate-400">ประมวลผลด้วยโมเดล AI ล้ำสมัย ให้ความแม่นยำสูง</p>
        </div>`)
	}

	sb.WriteString(`
      </div>
    </div>

    <!-- How It Works Section -->
    <div class="mb-16 bg-slate-900/40 border border-slate-800 rounded-3xl p-8 md:p-10">
      <h2 class="text-2xl font-bold text-white mb-8 text-center">ขั้นตอนการทำงาน (How It Works)</h2>
      <div class="grid md:grid-cols-3 gap-8 text-center">
        <div>
          <div class="w-12 h-12 mx-auto rounded-2xl bg-blue-600/20 text-blue-400 flex items-center justify-center font-bold text-lg mb-4">1</div>
          <h4 class="font-bold text-white mb-2">1. เลือกเครื่องมือ & ใส่ข้อมูล</h4>
          <p class="text-xs text-slate-400 leading-relaxed">อัปโหลดรูปภาพต้นฉบับ หรือใส่คำสั่ง Prompt ตามที่คุณต้องการสร้างสรรค์</p>
        </div>
        <div>
          <div class="w-12 h-12 mx-auto rounded-2xl bg-purple-600/20 text-purple-400 flex items-center justify-center font-bold text-lg mb-4">2</div>
          <h4 class="font-bold text-white mb-2">2. เช็คราคาล่วงหน้า</h4>
          <p class="text-xs text-slate-400 leading-relaxed">ระบบคำนวณ Tora Credits แบบโปร่งใส รับประกันไม่มีค่าใช้จ่ายแอบแฝง</p>
        </div>
        <div>
          <div class="w-12 h-12 mx-auto rounded-2xl bg-emerald-600/20 text-emerald-400 flex items-center justify-center font-bold text-lg mb-4">3</div>
          <h4 class="font-bold text-white mb-2">3. รับผลงานความละเอียดสูง</h4>
          <p class="text-xs text-slate-400 leading-relaxed">AI ประมวลผลและส่งมอบผลงานคุณภาพสตูดิโอ สามารถนำไปใช้งานต่อได้ทันที</p>
        </div>
      </div>
    </div>`)

	if tool.Id == "product-photo" || tool.Slug == "product-photo" {
		sb.WriteString(`
    <!-- Marketplace Presets Section -->
    <div class="mb-16">
      <h2 class="text-2xl font-bold text-white mb-3 text-center">สัดส่วนและพรีเซ็ตพร้อมใช้สำหรับทุก Marketplace</h2>
      <p class="text-sm text-slate-400 text-center mb-8 max-w-2xl mx-auto">ปรับขนาดและคอมโพสิชันอัตโนมัติให้ตรงตามมาตรฐานของแต่ละแพลตฟอร์ม ไม่ต้องครอปรูปเอง</p>
      <div class="grid grid-cols-2 md:grid-cols-5 gap-4">
        <div class="bg-slate-900/60 border border-slate-800 rounded-xl p-4 text-center">
          <div class="text-xs font-bold text-orange-400 bg-orange-500/10 py-1 px-2 rounded-md mb-2 inline-block">1:1 Square</div>
          <h4 class="font-semibold text-white text-sm">Shopee 1:1</h4>
          <p class="text-[11px] text-slate-400 mt-1">รูปหน้าปกสินค้าหลัก พร้อมวางกรอบโปร</p>
        </div>
        <div class="bg-slate-900/60 border border-slate-800 rounded-xl p-4 text-center">
          <div class="text-xs font-bold text-blue-400 bg-blue-500/10 py-1 px-2 rounded-md mb-2 inline-block">1:1 Square</div>
          <h4 class="font-semibold text-white text-sm">Lazada 1:1</h4>
          <p class="text-[11px] text-slate-400 mt-1">คมชัด โดดเด่นบนผลการค้นหา</p>
        </div>
        <div class="bg-slate-900/60 border border-slate-800 rounded-xl p-4 text-center">
          <div class="text-xs font-bold text-pink-400 bg-pink-500/10 py-1 px-2 rounded-md mb-2 inline-block">4:5 Vertical</div>
          <h4 class="font-semibold text-white text-sm">Instagram 4:5</h4>
          <p class="text-[11px] text-slate-400 mt-1">เต็มฟีดมือถือ เพิ่มยอดคลิกและบันทึก</p>
        </div>
        <div class="bg-slate-900/60 border border-slate-800 rounded-xl p-4 text-center">
          <div class="text-xs font-bold text-purple-400 bg-purple-500/10 py-1 px-2 rounded-md mb-2 inline-block">9:16 Fullscreen</div>
          <h4 class="font-semibold text-white text-sm">Story 9:16</h4>
          <p class="text-[11px] text-slate-400 mt-1">เว้นพื้นที่บน-ล่าง ใส่สติกเกอร์และราคา</p>
        </div>
        <div class="bg-slate-900/60 border border-slate-800 rounded-xl p-4 text-center">
          <div class="text-xs font-bold text-emerald-400 bg-emerald-500/10 py-1 px-2 rounded-md mb-2 inline-block">9:16 Vertical</div>
          <h4 class="font-semibold text-white text-sm">TikTok Shop 9:16</h4>
          <p class="text-[11px] text-slate-400 mt-1">ปกคลิปปักตะกร้า หยุดนิ้วใน 1 วินาที</p>
        </div>
      </div>
    </div>

    <!-- Connected Evergreen E-Commerce Guides -->
    <div class="mb-16 bg-gradient-to-r from-slate-900 via-slate-900/90 to-slate-900 border border-slate-800 rounded-3xl p-8">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h3 class="text-xl font-bold text-white">คู่มือและเทคนิคการถ่ายภาพสินค้า E-Commerce</h3>
          <p class="text-xs text-slate-400 mt-1">บทความแนะนำจากห้องข่าว Tora AI เพื่อช่วยผู้ประกอบการไทยเพิ่มยอดขาย</p>
        </div>
        <a href="/news" class="text-xs text-blue-400 hover:text-blue-300 font-medium">ดูบทความทั้งหมด &rarr;</a>
      </div>
      <div class="grid md:grid-cols-3 gap-4">
        <a href="/news" class="block bg-slate-950/70 border border-slate-800/80 rounded-xl p-4 hover:border-slate-700 transition">
          <span class="text-[10px] text-orange-400 font-semibold uppercase tracking-wider">Shopee & Lazada</span>
          <h5 class="text-sm font-semibold text-white mt-1 mb-1">เทคนิคทำภาพปกสินค้าให้ CTR พุ่ง 3 เท่าบน Shopee</h5>
          <p class="text-xs text-slate-400">กฎพื้นหลังขาวและคอมโพสิชันที่ระบบแนะนำชื่นชอบ</p>
        </a>
        <a href="/news" class="block bg-slate-950/70 border border-slate-800/80 rounded-xl p-4 hover:border-slate-700 transition">
          <span class="text-[10px] text-emerald-400 font-semibold uppercase tracking-wider">TikTok Shop</span>
          <h5 class="text-sm font-semibold text-white mt-1 mb-1">สัดส่วน 9:16 ปักตะกร้าอย่างไรให้ยอดสั่งซื้อไหลมาเทมา</h5>
          <p class="text-xs text-slate-400">การจัดแสงและ Mood ให้เข้ากับกลุ่มลูกค้า Gen Z</p>
        </a>
        <a href="/news" class="block bg-slate-950/70 border border-slate-800/80 rounded-xl p-4 hover:border-slate-700 transition">
          <span class="text-[10px] text-purple-400 font-semibold uppercase tracking-wider">SME Guide 2026</span>
          <h5 class="text-sm font-semibold text-white mt-1 mb-1">ลดต้นทุนถ่ายภาพสินค้า 90% ด้วย Tora AI Studio</h5>
          <p class="text-xs text-slate-400">สร้างภาพ 100 แบบในราคาไม่ถึง 200 บาทสำหรับธุรกิจเริ่มต้น</p>
        </a>
      </div>
    </div>`)
	}

	sb.WriteString(`
    <!-- Related Tools Section -->
    <div class="mb-16">
      <h2 class="text-2xl font-bold text-white mb-6 text-center">เครื่องมือที่เกี่ยวข้อง (Related Tools)</h2>
      <div class="grid md:grid-cols-3 gap-6">`)

	for _, relSlug := range meta.RelatedSlugs {
		relTool, err := model.GetStudioToolDefinition(relSlug)
		if err == nil && relTool != nil {
			sb.WriteString(`
        <a href="/tools/` + relTool.Slug + `" class="block bg-slate-900/60 border border-slate-800/80 rounded-2xl p-6 hover:border-blue-500/50 hover:bg-slate-900 transition group">
          <div class="flex items-center justify-between mb-3">
            <span class="text-xs font-semibold text-blue-400 bg-blue-500/10 px-2 py-0.5 rounded-full">` + strings.ToUpper(relTool.Category) + `</span>
            <span class="text-xs text-slate-500">` + strconv.Itoa(relTool.CreditCost) + ` Cr</span>
          </div>
          <h3 class="text-base font-bold text-white mb-1 group-hover:text-blue-400 transition">` + relTool.DisplayName + `</h3>
          <p class="text-xs text-slate-400 line-clamp-2">` + relTool.Description + `</p>
        </a>`)
		}
	}

	sb.WriteString(`
      </div>
    </div>

    <!-- Bottom CTA -->
    <div class="text-center bg-gradient-to-r from-blue-900/30 to-purple-900/30 border border-blue-500/20 rounded-3xl p-10">
      <h2 class="text-2xl md:text-3xl font-extrabold text-white mb-3">พร้อมเริ่มสร้างผลงานระดับสตูดิโอแล้วหรือยัง?</h2>
      <p class="text-sm text-slate-300 max-w-xl mx-auto mb-6">ใช้งาน Tora Credits ร่วมกันได้ทั้ง Chat, API และ Creator Studio เติมเงินเพียงกระเป๋าเดียว สะดวกและคุ้มค่าที่สุด</p>
      <a href="/studio?tool=` + tool.Id + `&tab=playground" class="inline-block bg-blue-600 hover:bg-blue-500 text-white px-8 py-3.5 rounded-xl font-semibold shadow-xl shadow-blue-600/30 transition">
        ทดลองใช้งาน ` + tool.DisplayName + ` ทันที
      </a>
    </div>
  </main>

  <footer class="border-t border-slate-900 bg-slate-950 py-8 text-center text-xs text-slate-500">
    <p>&copy; 2026 Tora AI. All rights reserved. <a href="/privacy-policy" class="hover:text-slate-400">นโยบายความเป็นส่วนตัว</a> | <a href="/user-agreement" class="hover:text-slate-400">ข้อกำหนดการใช้งาน</a></p>
  </footer>
</body>
</html>`)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, sb.String())
}

// UploadStudioAsset handles safe media upload for Studio pipelines (Queue 5 Asset Pipeline).
func UploadStudioAsset(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "file is required in multipart form-data: " + err.Error()})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed reading file: " + err.Error()})
		return
	}

	// Validate magic bytes and size (videos and images allowed)
	mimeType, err := service.ValidateMediaUpload(data, header.Filename, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "media validation failed: " + err.Error()})
		return
	}

	// Check disk quota exhaustion guards (Section 17 & 18)
	if err := service.CheckStorageQuota(model.DB, userId, int64(len(data))); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	assetId := fmt.Sprintf("asset_%d_%s", common.GetTimestamp(), common.GetUUID()[:8])
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		if strings.HasPrefix(mimeType, "image/") {
			ext = ".png"
		} else {
			ext = ".mp4"
		}
	}
	savedFilename := fmt.Sprintf("%s%s", assetId, ext)

	// Save file to authoritative studio upload directory
	uploadDir := service.GetStudioUploadDir()
	filePath := filepath.Join(uploadDir, savedFilename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed saving asset: " + err.Error()})
		return
	}

	assetURL := fmt.Sprintf("/api/v1/studio/assets/%s", savedFilename)
	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	if host != "" {
		assetURL = fmt.Sprintf("%s://%s/api/v1/studio/assets/%s", scheme, host, savedFilename)
	}

	now := common.GetTimestamp()
	asset := &model.StudioAsset{
		Id:                 assetId,
		UserId:             userId,
		AssetType:          "input",
		StorageURL:         assetURL,
		FileSize:           int64(len(data)),
		MIMEType:           mimeType,
		AvailabilityStatus: "available",
		ExpiryAt:           now + int64(service.DefaultInputAssetTTL.Seconds()),
		CreatedAt:          now,
	}
	if model.DB != nil {
		_ = model.DB.Create(asset)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"asset_id":  assetId,
			"url":       assetURL,
			"file_name": header.Filename,
			"mime_type": mimeType,
			"file_size": len(data),
		},
	})
}

// ServeStudioAsset serves uploaded studio assets safely.
func ServeStudioAsset(c *gin.Context) {
	filename := c.Param("filename")
	cleanFilename := filepath.Base(filename)
	filePath := filepath.Join(service.GetStudioUploadDir(), cleanFilename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "asset not found"})
		return
	}

	c.File(filePath)
}
