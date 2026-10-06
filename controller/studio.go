package controller

import (
	"errors"
	"fmt"
	"net/http"
	"os"
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

	// Update live tool state if provider is blocked
	for i := range tools {
		if tools[i].PrimaryProvider == "fal" && falBlocked {
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
			"provider_model":              snapshot.ProviderModel,
			"provider_estimated_cost_usd": snapshot.ProviderEstimatedCostUSD,
			"provider_cost_basis":         snapshot.ProviderCostBasis,
			"target_margin":               snapshot.TargetMargin,
			"calculated_sell_usd":         snapshot.CalculatedSellUSD,
			"calculated_credits":          snapshot.CalculatedCredits,
			"charged_credits":             snapshot.ChargedCredits,
			"charged_quota":               snapshot.ChargedQuota,
			"plan_multiplier":             snapshot.PlanMultiplier,
			"quoted_at":                   snapshot.QuotedAt,
			"expires_at":                  snapshot.ExpiresAt,
		},
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

	model.DB.Model(&model.StudioToolJob{}).Where("created_at >= ?", startOfDay).Count(&jobsToday)
	model.DB.Model(&model.StudioToolJob{}).Where("created_at >= ? AND status = ?", startOfDay, model.StudioJobStatusSucceeded).Count(&succeededToday)
	model.DB.Model(&model.StudioToolJob{}).Where("created_at >= ? AND status = ?", startOfDay, model.StudioJobStatusFailed).Count(&failedToday)

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
			"jobs_today":            jobsToday,
			"succeeded_today":       succeededToday,
			"failed_today":          failedToday,
			"credits_consumed":      creditsConsumed,
			"estimated_revenue_usd": revenueUSD,
			"estimated_cost_usd":    totalCostUSD,
			"estimated_profit_usd":  grossProfitUSD,
			"gross_margin_percent":  marginPercent,
			"providers":             providerStatus,
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

// --- SSR Public Landing Pages (Section 23) ---

// RenderStudioToolLandingPage serves indexable, crawlable HTML pages for high-value tools.
func RenderStudioToolLandingPage(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	canonicalBase := common.GetCanonicalBaseURL()

	tool, err := model.GetStudioToolDefinition(slug)
	if err != nil {
		c.Redirect(http.StatusFound, "/studio")
		return
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
      <a href="/playground?tool=` + tool.Id + `" class="text-sm bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded-lg font-medium shadow-lg shadow-blue-500/20 transition">เริ่มใช้งานทันที</a>
    </div>
  </nav>

  <main class="flex-1 max-w-5xl mx-auto px-6 py-12">
    <div class="text-center max-w-3xl mx-auto mb-12">
      <div class="inline-flex items-center space-x-2 bg-slate-900 border border-slate-800 px-3 py-1 rounded-full text-xs text-slate-400 mb-4">
        <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
        <span>หมวดหมู่: ` + strings.ToUpper(tool.Category) + `</span>
      </div>
      <h1 class="text-4xl md:text-5xl font-extrabold text-white mb-4 tracking-tight">` + tool.DisplayName + `</h1>
      <p class="text-lg text-slate-300 mb-6 leading-relaxed">` + tool.Description + `</p>
      <div class="flex flex-wrap items-center justify-center gap-4">
        <a href="/playground?tool=` + tool.Id + `" class="bg-blue-600 hover:bg-blue-500 text-white px-8 py-3.5 rounded-xl font-semibold shadow-xl shadow-blue-600/25 transition">
          ทดลองใช้งานใน Playground
        </a>
        <div class="bg-slate-900 border border-slate-800 px-5 py-3 rounded-xl text-sm text-slate-300 flex items-center space-x-2">
          <span>อัตราค่าบริการ:</span>
          <strong class="text-blue-400 font-bold">` + strconv.Itoa(tool.CreditCost) + ` Tora Credits</strong>
          <span class="text-xs text-slate-500">(≈ ฿` + fmt.Sprintf("%.2f", float64(tool.CreditCost)*0.07) + `)</span>
        </div>
      </div>
    </div>

    <div class="grid md:grid-cols-3 gap-6 mb-16">
      <div class="bg-slate-900/60 border border-slate-800/80 rounded-2xl p-6">
        <div class="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center text-blue-400 mb-4 font-bold">1</div>
        <h3 class="text-lg font-bold text-white mb-2">มาตรฐานระดับสากล</h3>
        <p class="text-sm text-slate-400">ประมวลผลด้วยโมเดล ` + tool.PrimaryModel + ` คุณภาพเทียบเท่าสตูดิโอระดับโลก</p>
      </div>
      <div class="bg-slate-900/60 border border-slate-800/80 rounded-2xl p-6">
        <div class="w-10 h-10 rounded-xl bg-purple-500/10 flex items-center justify-center text-purple-400 mb-4 font-bold">2</div>
        <h3 class="text-lg font-bold text-white mb-2">กระเป๋าเงินเดียว (One Wallet)</h3>
        <p class="text-sm text-slate-400">ใช้เครดิต Tora ร่วมกับ Chat และ API ได้ทันที ไม่ต้องเติมเงินแยกกระเป๋า</p>
      </div>
      <div class="bg-slate-900/60 border border-slate-800/80 rounded-2xl p-6">
        <div class="w-10 h-10 rounded-xl bg-emerald-500/10 flex items-center justify-center text-emerald-400 mb-4 font-bold">3</div>
        <h3 class="text-lg font-bold text-white mb-2">ปลอดภัย & คืนเงินอัตโนมัติ</h3>
        <p class="text-sm text-slate-400">หากเกิดข้อผิดพลาดในการประมวลผล ระบบจะคืนเครดิตเข้ากระเป๋าเต็มจำนวนทันที</p>
      </div>
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
