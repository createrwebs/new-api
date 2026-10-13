package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSellerTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:test_seller_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	origDB := DB
	t.Cleanup(func() {
		DB = origDB
	})
	DB = db
	require.NoError(t, db.AutoMigrate(
		&SellerBrandProfile{},
		&SellerWorkflowPreset{},
		&SellerLastPackExecution{},
	))
	return db
}

func TestSellerBrandProfile_Lifecycle(t *testing.T) {
	setupSellerTestDB(t)

	userId := 101

	// 1. Create Profile 1 (default)
	p1 := &SellerBrandProfile{
		UserId:          userId,
		StoreName:       "Siam Crafts Official",
		PrimaryChannels: "shopee,lazada,tiktok_shop",
		BrandHex:        "#FF5722",
		SecondaryHex:    "#FFFFFF",
		PreferredBg:     "PURE_WHITE",
		PreferredShadow: "MARKETPLACE",
		IsDefault:       true,
	}
	err := CreateSellerProfile(p1)
	require.NoError(t, err)
	assert.True(t, p1.Id > 0)
	assert.True(t, p1.IsDefault)

	// 2. Create Profile 2 (make default, should clear p1 default)
	p2 := &SellerBrandProfile{
		UserId:          userId,
		StoreName:       "Bangkok Apparel",
		PrimaryChannels: "instagram",
		BrandHex:        "#333333",
		PreferredBg:     "WARM_WHITE",
		PreferredShadow: "SOFT_STUDIO",
		IsDefault:       true,
	}
	err = CreateSellerProfile(p2)
	require.NoError(t, err)

	// Verify p1 is no longer default
	p1Refetched, err := GetSellerProfileById(userId, p1.Id)
	require.NoError(t, err)
	assert.False(t, p1Refetched.IsDefault)

	p2Refetched, err := GetSellerProfileById(userId, p2.Id)
	require.NoError(t, err)
	assert.True(t, p2Refetched.IsDefault)

	// 3. List profiles
	profiles, err := GetSellerProfilesByUserId(userId)
	require.NoError(t, err)
	assert.Equal(t, 2, len(profiles))
	assert.Equal(t, p2.Id, profiles[0].Id) // p2 is default, sorted first

	// 4. Update profile
	updated, err := UpdateSellerProfile(userId, p1.Id, map[string]interface{}{
		"brand_hex": "#00AA55",
	})
	require.NoError(t, err)
	assert.Equal(t, "#00AA55", updated.BrandHex)

	// 5. Delete profile
	err = DeleteSellerProfile(userId, p1.Id)
	require.NoError(t, err)

	_, err = GetSellerProfileById(userId, p1.Id)
	assert.ErrorIs(t, err, ErrSellerProfileNotFound)
}

func TestSellerWorkflowPreset_Lifecycle(t *testing.T) {
	setupSellerTestDB(t)

	userId := 102

	// 1. Create preset
	preset := &SellerWorkflowPreset{
		UserId:            userId,
		PresetName:        "Weekly Shopee Mall Launch",
		Description:       "Clean pure white background with grounded shadow",
		SelectedTemplates: "[\"shopee_standard\",\"lazada_hd\"]",
		BgPreset:          "PURE_WHITE",
		ShadowPreset:      "MARKETPLACE",
		BrandHex:          "#FF5722",
		IncludeZip:        true,
	}
	err := CreateSellerPreset(preset)
	require.NoError(t, err)
	assert.True(t, preset.Id > 0)

	// 2. Fetch preset
	fetched, err := GetSellerPresetById(userId, preset.Id)
	require.NoError(t, err)
	assert.Equal(t, "Weekly Shopee Mall Launch", fetched.PresetName)

	// 3. Update preset
	updated, err := UpdateSellerPreset(userId, preset.Id, map[string]interface{}{
		"preset_name": "Updated Shopee & Lazada Launch",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Shopee & Lazada Launch", updated.PresetName)

	// 4. Delete preset
	err = DeleteSellerPreset(userId, preset.Id)
	require.NoError(t, err)

	_, err = GetSellerPresetById(userId, preset.Id)
	assert.ErrorIs(t, err, ErrSellerPresetNotFound)
}

func TestSellerLastPackExecution_Repeat(t *testing.T) {
	setupSellerTestDB(t)

	userId := 103

	// 1. Initially no last execution
	last, err := GetLastPackExecution(userId)
	assert.Error(t, err)
	assert.Nil(t, last)

	// 2. Save first execution
	err = SaveLastPackExecution(userId, "[\"shopee_standard\"]", "PURE_WHITE", "MARKETPLACE", "#112233", true, 3)
	require.NoError(t, err)

	last, err = GetLastPackExecution(userId)
	require.NoError(t, err)
	assert.Equal(t, 3, last.ItemCount)
	assert.Equal(t, "PURE_WHITE", last.BgPreset)
	assert.Equal(t, "#112233", last.BrandHex)

	// 3. Update execution (repeat overwrite)
	time.Sleep(10 * time.Millisecond)
	err = SaveLastPackExecution(userId, "[\"shopee_standard\",\"tiktok_shop\"]", "WARM_WHITE", "SOFT_STUDIO", "#990000", false, 5)
	require.NoError(t, err)

	lastUpdated, err := GetLastPackExecution(userId)
	require.NoError(t, err)
	assert.Equal(t, 5, lastUpdated.ItemCount)
	assert.Equal(t, "WARM_WHITE", lastUpdated.BgPreset)
	assert.Equal(t, "#990000", lastUpdated.BrandHex)
	assert.False(t, lastUpdated.IncludeZip)
}
