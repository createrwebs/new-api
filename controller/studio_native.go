package controller

import (
	"errors"
	"io"
	"net/http"
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

// CreateNativeTicketRequest represents request to authorize and reserve quota for client inference.
type CreateNativeTicketRequest struct {
	ToolId            string                     `json:"tool_id" binding:"required"`
	ExecutionClass    model.NativeExecutionClass `json:"execution_class"`
	Inputs            map[string]interface{}     `json:"inputs"`
	ClientDeviceClass string                     `json:"client_device_class"`
}

// CompleteNativeTicketRequest reports successful client-side inference and settles quota.
type CompleteNativeTicketRequest struct {
	TicketId          string `json:"ticket_id" binding:"required"`
	OutputAssetHash   string `json:"output_asset_hash"`
	ClientExecutionMs int64  `json:"client_execution_ms"`
	ClientDeviceClass string `json:"client_device_class"`
}

// RefundNativeTicketRequest reports client-side failure or cancellation to refund reserved quota.
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

	ticket, err := service.CreateNativeExecutionTicket(userId, req.ToolId, req.ExecutionClass, req.Inputs, req.ClientDeviceClass)
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
		"message": "execution ticket authorized and wallet reserved",
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
		} else if errors.Is(err, model.ErrNativeTicketAlreadySettled) {
			status = http.StatusOK // Idempotent
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error(), "data": ticket})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "execution settled successfully",
		"data":    ticket,
	})
}

// RefundNativeTicket handles POST /api/studio/native/refund.
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

	ticket, err := service.RefundNativeExecutionTicket(userId, req.TicketId, req.Reason)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNativeUnauthorized) {
			status = http.StatusForbidden
		} else if errors.Is(err, model.ErrNativeTicketNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, model.ErrNativeTicketAlreadySettled) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "execution ticket refunded",
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

	// Charge 5 Tora Credits (5000 Quota)
	packQuota := 5000
	hasQuota, err := model.TryReserveUserQuota(userId, packQuota)
	if err != nil || !hasQuota {
		c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "message": "insufficient wallet quota (5 Tora Credits required)"})
		return
	}

	packResult, err := service.GenerateMarketplaceProductPack(model.DB, userId, "", rawBytes)
	if err != nil {
		_ = model.IncreaseUserQuota(userId, packQuota, false) // refund
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed generating product pack: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "marketplace product pack generated successfully",
		"data":    packResult,
	})
}
