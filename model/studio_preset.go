package model

import (
	"errors"
	"time"
)

var (
	ErrSellerPresetNotFound = errors.New("seller workflow preset not found")
)

// SellerWorkflowPreset stores reusable workflow configurations for product factory batches.
// INVARIANT: Presets persist ONLY configuration parameters, NEVER price quotes or payment commitments.
type SellerWorkflowPreset struct {
	Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId            int    `json:"user_id" gorm:"index;not null"`
	ProfileId         int    `json:"profile_id" gorm:"index;default:0"`
	PresetName        string `json:"preset_name" gorm:"size:255;not null"`
	Description       string `json:"description" gorm:"size:512"`
	SelectedTemplates string `json:"selected_templates" gorm:"type:text"` // JSON encoded string of template IDs
	BgPreset          string `json:"bg_preset" gorm:"size:64;default:'PURE_WHITE'"`
	ShadowPreset      string `json:"shadow_preset" gorm:"size:64;default:'MARKETPLACE'"`
	BrandHex          string `json:"brand_hex" gorm:"size:32"`
	IncludeZip        bool   `json:"include_zip" gorm:"default:true"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

// SellerLastPackExecution stores the most recent successful Product Factory execution config for 1-click repetition.
type SellerLastPackExecution struct {
	Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId            int    `json:"user_id" gorm:"uniqueIndex;not null"`
	SelectedTemplates string `json:"selected_templates" gorm:"type:text"`
	BgPreset          string `json:"bg_preset" gorm:"size:64"`
	ShadowPreset      string `json:"shadow_preset" gorm:"size:64"`
	BrandHex          string `json:"brand_hex" gorm:"size:32"`
	IncludeZip        bool   `json:"include_zip"`
	ItemCount         int    `json:"item_count"`
	LastExecutedAt    int64  `json:"last_executed_at"`
}

func CreateSellerPreset(preset *SellerWorkflowPreset) error {
	now := time.Now().Unix()
	preset.CreatedAt = now
	preset.UpdatedAt = now
	return DB.Create(preset).Error
}

func GetSellerPresetsByUserId(userId int) ([]SellerWorkflowPreset, error) {
	var presets []SellerWorkflowPreset
	err := DB.Where("user_id = ?", userId).Order("id DESC").Find(&presets).Error
	return presets, err
}

func GetSellerPresetById(userId int, presetId int) (*SellerWorkflowPreset, error) {
	var preset SellerWorkflowPreset
	err := DB.Where("id = ? AND user_id = ?", presetId, userId).First(&preset).Error
	if err != nil {
		return nil, ErrSellerPresetNotFound
	}
	return &preset, nil
}

func UpdateSellerPreset(userId int, presetId int, updates map[string]interface{}) (*SellerWorkflowPreset, error) {
	preset, err := GetSellerPresetById(userId, presetId)
	if err != nil {
		return nil, err
	}
	updates["updated_at"] = time.Now().Unix()
	err = DB.Model(preset).Updates(updates).Error
	if err != nil {
		return nil, err
	}
	return preset, nil
}

func DeleteSellerPreset(userId int, presetId int) error {
	res := DB.Where("id = ? AND user_id = ?", presetId, userId).Delete(&SellerWorkflowPreset{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSellerPresetNotFound
	}
	return nil
}

func SaveLastPackExecution(userId int, selectedTemplates string, bgPreset, shadowPreset, brandHex string, includeZip bool, itemCount int) error {
	now := time.Now().Unix()
	var rec SellerLastPackExecution
	err := DB.Where("user_id = ?", userId).First(&rec).Error
	if err != nil {
		rec = SellerLastPackExecution{
			UserId:            userId,
			SelectedTemplates: selectedTemplates,
			BgPreset:          bgPreset,
			ShadowPreset:      shadowPreset,
			BrandHex:          brandHex,
			IncludeZip:        includeZip,
			ItemCount:         itemCount,
			LastExecutedAt:    now,
		}
		return DB.Create(&rec).Error
	}
	return DB.Model(&rec).Updates(map[string]interface{}{
		"selected_templates": selectedTemplates,
		"bg_preset":          bgPreset,
		"shadow_preset":      shadowPreset,
		"brand_hex":          brandHex,
		"include_zip":        includeZip,
		"item_count":         itemCount,
		"last_executed_at":   now,
	}).Error
}

func GetLastPackExecution(userId int) (*SellerLastPackExecution, error) {
	var rec SellerLastPackExecution
	err := DB.Where("user_id = ?", userId).First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
