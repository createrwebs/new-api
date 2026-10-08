package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// QuoteNativeToolRequest represents request payload for native pricing quotes.
type QuoteNativeToolRequest struct {
	ToolId         string                     `json:"tool_id" binding:"required"`
	ExecutionClass model.NativeExecutionClass `json:"execution_class"`
}

// CreateNativeTicketRequest represents request to authorize and provision an execution ticket.
type CreateNativeTicketRequest struct {
	ToolId            string                     `json:"tool_id" binding:"required"`
	ExecutionClass    model.NativeExecutionClass `json:"execution_class"`
	Inputs            map[string]interface{}     `json:"inputs"`
	ClientDeviceClass string                     `json:"client_device_class"`
	IdempotencyKey    string                     `json:"idempotency_key"`
}

// CompleteNativeTicketRequest reports client-side inference completion with output digest.
type CompleteNativeTicketRequest struct {
	TicketId          string `json:"ticket_id" binding:"required"`
	OutputAssetHash   string `json:"output_asset_hash"`
	ClientExecutionMs int64  `json:"client_execution_ms"`
	ClientDeviceClass string `json:"client_device_class"`
}

// FailNativeTicketRequest reports client-side runtime error.
type FailNativeTicketRequest struct {
	TicketId string `json:"ticket_id" binding:"required"`
	Reason   string `json:"reason"`
}

// RetryNativeTicketRequest requests same-ticket fair retry authorization.
type RetryNativeTicketRequest struct {
	TicketId string `json:"ticket_id" binding:"required"`
}

// RefundNativeTicketRequest reports pre-activation failure to release reserved quota.
type RefundNativeTicketRequest struct {
	TicketId string `json:"ticket_id" binding:"required"`
	Reason   string `json:"reason"`
}

// QuoteNativeTool handles POST /api/studio/native/quote.
func QuoteNativeTool(c *gin.Context) {
	var req QuoteNativeToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request parameters: " + err.Error()})
		return
	}

	quote, err := service.GetNativeToolQuote(req.ToolId, req.ExecutionClass)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    quote,
	})
}

// QuoteProductFactoryBatch handles POST /api/studio/native/product-factory/quote (Section 33).
func QuoteProductFactoryBatch(c *gin.Context) {
	var req service.ProductFactoryBatchQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request parameters: " + err.Error()})
		return
	}

	quote, err := service.CalculateProductFactoryBatchQuote(req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrAppVersionUnsupported) {
			status = http.StatusUpgradeRequired
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    quote,
	})
}

// CreateNativeTicket handles POST /api/studio/native/ticket.
func CreateNativeTicket(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authentication required"})
		return
	}

	var req CreateNativeTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request parameters: " + err.Error()})
		return
	}

	// Resolve Idempotency Key from header or body
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	}

	ticket, err := service.CreateNativeExecutionTicket(userId, req.ToolId, req.ExecutionClass, req.Inputs, req.ClientDeviceClass, idempotencyKey)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, model.ErrWalletQuotaInsufficient) || strings.Contains(err.Error(), "insufficient") {
			status = http.StatusPaymentRequired
		} else if errors.Is(err, service.ErrNativeToolNotSupported) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "execution ticket authorized and wallet debited",
		"data":    ticket,
	})
}

// CompleteNativeTicket handles POST /api/studio/native/complete.
func CompleteNativeTicket(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authentication required"})
		return
	}

	var req CompleteNativeTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request parameters: " + err.Error()})
		return
	}

	ticket, err := service.CompleteNativeExecutionTicket(userId, req.TicketId, req.OutputAssetHash, req.ClientExecutionMs, req.ClientDeviceClass)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNativeUnauthorized) {
			status = http.StatusForbidden
		} else if errors.Is(err, model.ErrNativeTicketNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, model.ErrNativeTicketExpired) {
			status = http.StatusGone
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error(), "data": ticket})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "execution completed and recorded successfully",
		"data":    ticket,
	})
}

// FailNativeTicket handles POST /api/studio/native/fail (Reports client runtime error).
func FailNativeTicket(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authentication required"})
		return
	}

	var req FailNativeTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request parameters: " + err.Error()})
		return
	}

	ticket, err := service.FailNativeExecutionTicket(userId, req.TicketId, req.Reason)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNativeUnauthorized) {
			status = http.StatusForbidden
		} else if errors.Is(err, model.ErrNativeTicketNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "client failure recorded; same-ticket retry is permitted within retry window",
		"data":    ticket,
	})
}

// RetryNativeTicket handles POST /api/studio/native/retry (Fair retry at 0 additional Credits).
func RetryNativeTicket(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authentication required"})
		return
	}

	var req RetryNativeTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request parameters: " + err.Error()})
		return
	}

	ticket, err := service.RetryNativeExecutionTicket(userId, req.TicketId)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNativeUnauthorized) {
			status = http.StatusForbidden
		} else if errors.Is(err, model.ErrNativeTicketNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, model.ErrNativeTicketRetryExceeded) {
			status = http.StatusGone
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "same-ticket retry authorized at 0 additional credits",
		"data":    ticket,
	})
}

// RefundNativeTicket handles POST /api/studio/native/refund (Pre-activation failures only).
func RefundNativeTicket(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authentication required"})
		return
	}

	var req RefundNativeTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request parameters: " + err.Error()})
		return
	}

	ticket, err := service.RefundPreExecutionTicket(userId, req.TicketId, req.Reason)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNativeUnauthorized) {
			status = http.StatusForbidden
		} else if errors.Is(err, model.ErrNativeTicketNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, service.ErrPreExecutionRefundOnly) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "pre-activation reservation refunded",
		"data":    ticket,
	})
}

// GenerateProductPackRequest payload for server-side deterministic processing.
type GenerateProductPackRequest struct {
	ImageURL string `json:"image_url"`
}

// GenerateMarketplaceProductPack handles POST /api/studio/native/product-pack.
func GenerateMarketplaceProductPack(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authentication required"})
		return
	}

	var rawBytes []byte

	// Check if multi-part form upload
	file, err := c.FormFile("file")
	if err == nil {
		f, openErr := file.Open()
		if openErr == nil {
			defer f.Close()
			rawBytes, _ = io.ReadAll(f)
		}
	}

	// If no form file, check JSON payload or download
	if len(rawBytes) == 0 {
		var req GenerateProductPackRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.ImageURL != "" {
			reqHTTP, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, req.ImageURL, nil)
			if err == nil {
				resp, err := service.SafeHTTPClient().Do(reqHTTP)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					rawBytes, _ = io.ReadAll(resp.Body)
				}
			}
		}
	}

	if len(rawBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "no valid image file or image_url provided"})
		return
	}

	// Guard maximum image upload size (20MB) to protect host memory (Section 50)
	if len(rawBytes) > 20*1024*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "image file exceeds maximum safe limit of 20MB"})
		return
	}

	// Charge 5 Tora Credits (5000 Quota) via SUCCESS_SETTLEMENT
	packQuota := 5000
	hasQuota, err := model.TryReserveUserQuota(userId, packQuota)
	if err != nil || !hasQuota {
		c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "message": "insufficient wallet quota (5 Tora Credits required)"})
		return
	}

	packResult, err := service.GenerateMarketplaceProductPack(model.DB, userId, "", rawBytes)
	if err != nil {
		_ = model.IncreaseUserQuota(userId, packQuota, false) // refund on server processing failure
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed generating product pack: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "marketplace product pack generated successfully",
		"data":    packResult,
	})
}

// ServeNativeModel serves open-weight model artifacts with content-addressed cache headers.
func ServeNativeModel(c *gin.Context) {
	modelId := c.Param("modelId")
	meta, ok := service.NativeModels[modelId]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "model not recognized in manifest"})
		return
	}

	modelsDir := os.Getenv("STUDIO_MODELS_DIR")
	if modelsDir == "" {
		modelsDir = "/data/models"
	}
	modelPath := filepath.Join(modelsDir, meta.Filename)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		// Fallback check in current working directory / models
		modelPath = filepath.Join("data", "models", meta.Filename)
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "model artifact not found on server"})
			return
		}
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("ETag", `"`+meta.SHA256+`"`)
	c.Header("Access-Control-Allow-Origin", "*")
	c.File(modelPath)
}

// StartObjectCleanupSessionRequest represents input to start an interactive cleanup session.
type StartObjectCleanupSessionRequest struct {
	SourceHash        string `json:"source_hash" binding:"required"`
	ClientDeviceClass string `json:"client_device_class"`
}

// ValidateObjectCleanupSessionRequest checks if session is valid for subsequent export or edit.
type ValidateObjectCleanupSessionRequest struct {
	SessionId  string `json:"session_id" binding:"required"`
	SourceHash string `json:"source_hash"`
}

// RecordObjectCleanupExportRequest records an export within a session.
type RecordObjectCleanupExportRequest struct {
	SessionId string `json:"session_id" binding:"required"`
}

// StartObjectCleanupSessionHandler handles POST /api/studio/native/object-cleanup/session.
func StartObjectCleanupSessionHandler(c *gin.Context) {
	userId := c.GetInt("id")
	var req StartObjectCleanupSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request: " + err.Error()})
		return
	}

	session, ticket, err := service.StartObjectCleanupSession(userId, req.SourceHash, req.ClientDeviceClass)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "object cleanup interactive session started successfully",
		"data": gin.H{
			"session": session,
			"ticket":  ticket,
		},
	})
}

// ValidateObjectCleanupSessionHandler handles POST /api/studio/native/object-cleanup/session/validate.
func ValidateObjectCleanupSessionHandler(c *gin.Context) {
	userId := c.GetInt("id")
	var req ValidateObjectCleanupSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request: " + err.Error()})
		return
	}

	session, err := service.ValidateObjectCleanupSession(userId, req.SessionId, req.SourceHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "session is valid",
		"data":    session,
	})
}

// RecordObjectCleanupExportHandler handles POST /api/studio/native/object-cleanup/session/export.
func RecordObjectCleanupExportHandler(c *gin.Context) {
	userId := c.GetInt("id")
	var req RecordObjectCleanupExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request: " + err.Error()})
		return
	}

	if err := service.RecordObjectCleanupExport(userId, req.SessionId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "export recorded successfully",
	})
}

// GetSellerTemplatesHandler handles GET /api/studio/native/seller-templates.
func GetSellerTemplatesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"templates": service.CanonicalSellerTemplates,
			"shadow_presets": []string{
				string(service.ShadowSoftStudio),
				string(service.ShadowMarketplace),
				string(service.ShadowFloating),
				string(service.ShadowGroundContact),
				string(service.ShadowNone),
			},
			"bg_presets": []string{
				string(service.BgPureWhite),
				string(service.BgWarmWhite),
				string(service.BgLightGray),
				string(service.BgBrandColor),
				string(service.BgSoftGradient),
				string(service.BgStudioVignette),
				string(service.BgTransparent),
			},
		},
	})
}

// ExecuteProductFactoryV2BatchHandler handles POST /api/studio/native/product-factory/v2/batch.
func ExecuteProductFactoryV2BatchHandler(c *gin.Context) {
	userId := c.GetInt("id")
	var req service.ProductFactoryV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid batch request: " + err.Error()})
		return
	}

	itemCount := len(req.Items)
	if itemCount == 0 || itemCount > 25 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "batch size must be between 1 and 25 items"})
		return
	}

	// Calculate quota using authoritative batch quote
	batchQuote, err := service.CalculateProductFactoryBatchQuote(service.ProductFactoryBatchQuoteRequest{
		InputCount: itemCount,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "failed calculating batch quote: " + err.Error()})
		return
	}

	// Settle quota via SUCCESS_SETTLEMENT escrow
	hasQuota, err := model.TryReserveUserQuota(userId, batchQuote.TotalQuota)
	if err != nil || !hasQuota {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"success": false,
			"message": fmt.Sprintf("insufficient wallet quota (%d Tora Credits required for %d items)", batchQuote.TotalCredits, itemCount),
		})
		return
	}

	result, err := service.GenerateSellerFactoryV2Batch(userId, req)
	if err != nil {
		_ = model.IncreaseUserQuota(userId, batchQuote.TotalQuota, false) // refund on server execution failure
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed executing seller factory batch: " + err.Error()})
		return
	}

	// Persist last pack execution configuration for 1-click repetition
	tplBytes, _ := json.Marshal(req.SelectedTemplates)
	_ = model.SaveLastPackExecution(
		userId,
		string(tplBytes),
		string(req.BgPreset),
		string(req.ShadowPreset),
		req.BrandHex,
		req.IncludeZip,
		len(req.Items),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "seller factory v2 batch executed successfully",
		"data": gin.H{
			"result":        result,
			"batch_quote":   batchQuote,
			"credits_spent": batchQuote.TotalCredits,
		},
	})
}
