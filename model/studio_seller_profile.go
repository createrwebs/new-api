package model

import (
	"errors"
	"time"
)

var (
	ErrSellerProfileNotFound = errors.New("seller brand profile not found")
)

// SellerBrandProfile stores reusable seller identity, branding colors, and channel defaults.
type SellerBrandProfile struct {
	Id               int     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId           int     `json:"user_id" gorm:"index;not null"`
	StoreName        string  `json:"store_name" gorm:"size:255;not null"`
	PrimaryChannels  string  `json:"primary_channels" gorm:"size:255"` // comma-separated e.g. "shopee,lazada,tiktok_shop"
	BrandHex         string  `json:"brand_hex" gorm:"size:32"`
	SecondaryHex     string  `json:"secondary_hex" gorm:"size:32"`
	PreferredBg      string  `json:"preferred_bg" gorm:"size:64;default:'PURE_WHITE'"`
	PreferredShadow  string  `json:"preferred_shadow" gorm:"size:64;default:'MARKETPLACE'"`
	DefaultMarginPct float64 `json:"default_margin_pct" gorm:"default:0.10"`
	IsDefault        bool    `json:"is_default" gorm:"default:false"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

func CreateSellerProfile(profile *SellerBrandProfile) error {
	now := time.Now().Unix()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	if profile.IsDefault {
		_ = DB.Model(&SellerBrandProfile{}).Where("user_id = ?", profile.UserId).Update("is_default", false)
	}
	return DB.Create(profile).Error
}

func GetSellerProfilesByUserId(userId int) ([]SellerBrandProfile, error) {
	var profiles []SellerBrandProfile
	err := DB.Where("user_id = ?", userId).Order("is_default DESC, id DESC").Find(&profiles).Error
	return profiles, err
}

func GetSellerProfileById(userId int, profileId int) (*SellerBrandProfile, error) {
	var profile SellerBrandProfile
	err := DB.Where("id = ? AND user_id = ?", profileId, userId).First(&profile).Error
	if err != nil {
		return nil, ErrSellerProfileNotFound
	}
	return &profile, nil
}

func UpdateSellerProfile(userId int, profileId int, updates map[string]interface{}) (*SellerBrandProfile, error) {
	profile, err := GetSellerProfileById(userId, profileId)
	if err != nil {
		return nil, err
	}
	updates["updated_at"] = time.Now().Unix()
	if isDef, ok := updates["is_default"].(bool); ok && isDef {
		_ = DB.Model(&SellerBrandProfile{}).Where("user_id = ?", userId).Update("is_default", false)
	}
	err = DB.Model(profile).Updates(updates).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func DeleteSellerProfile(userId int, profileId int) error {
	res := DB.Where("id = ? AND user_id = ?", profileId, userId).Delete(&SellerBrandProfile{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSellerProfileNotFound
	}
	return nil
}
