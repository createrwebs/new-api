package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// GetMarketplaceRulesHandler handles GET /api/studio/marketplace/rules
func GetMarketplaceRulesHandler(c *gin.Context) {
	rules := service.GetMarketplaceRules()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rules,
	})
}

// ValidateMarketplaceComplianceHandler handles POST /api/studio/marketplace/validate
func ValidateMarketplaceComplianceHandler(c *gin.Context) {
	var req struct {
		Platform string                         `json:"platform" binding:"required"`
		Region   string                         `json:"region"`
		Asset    model.ComplianceValidationInput `json:"asset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request: " + err.Error()})
		return
	}

	result := service.ValidateCompliance(req.Asset, req.Platform, req.Region)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
