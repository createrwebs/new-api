package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ListSellerProfilesHandler handles GET /api/studio/seller/profiles
func ListSellerProfilesHandler(c *gin.Context) {
	userId := c.GetInt("id")
	profiles, err := model.GetSellerProfilesByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list profiles: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profiles})
}

// CreateSellerProfileHandler handles POST /api/studio/seller/profiles
func CreateSellerProfileHandler(c *gin.Context) {
	userId := c.GetInt("id")
	var profile model.SellerBrandProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid profile payload: " + err.Error()})
		return
	}
	profile.UserId = userId
	if profile.StoreName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "store_name is required"})
		return
	}

	if err := model.CreateSellerProfile(&profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to create profile: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

// UpdateSellerProfileHandler handles PUT /api/studio/seller/profiles/:id
func UpdateSellerProfileHandler(c *gin.Context) {
	userId := c.GetInt("id")
	profileId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid profile id"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid update payload: " + err.Error()})
		return
	}

	profile, err := model.UpdateSellerProfile(userId, profileId, updates)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

// DeleteSellerProfileHandler handles DELETE /api/studio/seller/profiles/:id
func DeleteSellerProfileHandler(c *gin.Context) {
	userId := c.GetInt("id")
	profileId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid profile id"})
		return
	}

	if err := model.DeleteSellerProfile(userId, profileId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "profile deleted successfully"})
}

// ListSellerPresetsHandler handles GET /api/studio/seller/presets
func ListSellerPresetsHandler(c *gin.Context) {
	userId := c.GetInt("id")
	presets, err := model.GetSellerPresetsByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list presets: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": presets})
}

// CreateSellerPresetHandler handles POST /api/studio/seller/presets
func CreateSellerPresetHandler(c *gin.Context) {
	userId := c.GetInt("id")
	var req struct {
		ProfileId         int      `json:"profile_id"`
		PresetName        string   `json:"preset_name"`
		Description       string   `json:"description"`
		SelectedTemplates []string `json:"selected_templates"`
		BgPreset          string   `json:"bg_preset"`
		ShadowPreset      string   `json:"shadow_preset"`
		BrandHex          string   `json:"brand_hex"`
		IncludeZip        bool     `json:"include_zip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid preset payload: " + err.Error()})
		return
	}
	if req.PresetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "preset_name is required"})
		return
	}

	tplBytes, _ := json.Marshal(req.SelectedTemplates)
	preset := model.SellerWorkflowPreset{
		UserId:            userId,
		ProfileId:         req.ProfileId,
		PresetName:        req.PresetName,
		Description:       req.Description,
		SelectedTemplates: string(tplBytes),
		BgPreset:          req.BgPreset,
		ShadowPreset:      req.ShadowPreset,
		BrandHex:          req.BrandHex,
		IncludeZip:        req.IncludeZip,
	}

	if err := model.CreateSellerPreset(&preset); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to create preset: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": preset})
}

// UpdateSellerPresetHandler handles PUT /api/studio/seller/presets/:id
func UpdateSellerPresetHandler(c *gin.Context) {
	userId := c.GetInt("id")
	presetId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid preset id"})
		return
	}

	var rawUpdates map[string]interface{}
	if err := c.ShouldBindJSON(&rawUpdates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid update payload: " + err.Error()})
		return
	}

	// Handle selected_templates if passed as array
	if templates, ok := rawUpdates["selected_templates"].([]interface{}); ok {
		tplBytes, _ := json.Marshal(templates)
		rawUpdates["selected_templates"] = string(tplBytes)
	}

	preset, err := model.UpdateSellerPreset(userId, presetId, rawUpdates)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": preset})
}

// DeleteSellerPresetHandler handles DELETE /api/studio/seller/presets/:id
func DeleteSellerPresetHandler(c *gin.Context) {
	userId := c.GetInt("id")
	presetId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid preset id"})
		return
	}

	if err := model.DeleteSellerPreset(userId, presetId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "preset deleted successfully"})
}

// GetRepeatLastPackHandler handles GET /api/studio/seller/repeat-last
func GetRepeatLastPackHandler(c *gin.Context) {
	userId := c.GetInt("id")
	rec, err := model.GetLastPackExecution(userId)
	if err != nil || rec == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "no previous product pack execution found for user",
		})
		return
	}

	var templates []string
	if rec.SelectedTemplates != "" {
		_ = json.Unmarshal([]byte(rec.SelectedTemplates), &templates)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"selected_templates": templates,
			"bg_preset":          rec.BgPreset,
			"shadow_preset":      rec.ShadowPreset,
			"brand_hex":          rec.BrandHex,
			"include_zip":        rec.IncludeZip,
			"item_count":         rec.ItemCount,
			"last_executed_at":   rec.LastExecutedAt,
		},
	})
}
