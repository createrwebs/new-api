package controller

import (
	"errors"
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
