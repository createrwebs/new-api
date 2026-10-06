package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSubscriptionPlanInitTestDB(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&SubscriptionPlan{},
		&StoreProductMapping{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
	})
}

func TestInitDefaultSubscriptionPlan_EmptyDB(t *testing.T) {
	setupSubscriptionPlanInitTestDB(t)

	err := InitDefaultSubscriptionPlan()
	require.NoError(t, err)

	var plans []SubscriptionPlan
	require.NoError(t, DB.Find(&plans).Error)
	require.Len(t, plans, 1)

	plan := plans[0]
	assert.Equal(t, "Pro Monthly", plan.Title)
	assert.Equal(t, 9.99, plan.PriceAmount)
	assert.Equal(t, "USD", plan.Currency)
	assert.Equal(t, "pro", plan.UpgradeGroup)
	assert.Equal(t, "default", plan.DowngradeGroup)
	assert.Equal(t, int64(2000000), plan.TotalAmount)
	assert.Equal(t, SubscriptionDurationMonth, plan.DurationUnit)
	assert.Equal(t, 1, plan.DurationValue)
	assert.Equal(t, SubscriptionResetMonthly, plan.QuotaResetPeriod)
	assert.True(t, plan.Enabled)
}

func TestInitDefaultSubscriptionPlan_Idempotent(t *testing.T) {
	setupSubscriptionPlanInitTestDB(t)

	require.NoError(t, InitDefaultSubscriptionPlan())
	require.NoError(t, InitDefaultSubscriptionPlan())

	var count int64
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestEnsureProMonthlySubscriptionPlan(t *testing.T) {
	setupSubscriptionPlanInitTestDB(t)

	plan1, err := EnsureProMonthlySubscriptionPlan()
	require.NoError(t, err)
	require.NotNil(t, plan1)
	assert.Equal(t, "Pro Monthly", plan1.Title)
	assert.Equal(t, "pro", plan1.UpgradeGroup)
	assert.Equal(t, int64(2000000), plan1.TotalAmount)

	plan2, err := EnsureProMonthlySubscriptionPlan()
	require.NoError(t, err)
	require.NotNil(t, plan2)
	assert.Equal(t, plan1.Id, plan2.Id)
}

func TestInitDefaultStoreProductMappings(t *testing.T) {
	setupSubscriptionPlanInitTestDB(t)

	require.NoError(t, InitDefaultSubscriptionPlan())
	require.NoError(t, InitDefaultStoreProductMappings())

	// Verify Apple mapping
	var appleMapping StoreProductMapping
	err := DB.Where("platform = ? AND store_product_id = ?", StorePlatformApple, "com.saascover.tora.pro.monthly").First(&appleMapping).Error
	require.NoError(t, err)
	assert.True(t, appleMapping.Enabled)
	assert.Equal(t, "all", appleMapping.Environment)

	// Verify Google mapping
	var googleMapping StoreProductMapping
	err = DB.Where("platform = ? AND store_product_id = ? AND store_base_plan_id = ?", StorePlatformGoogle, "tora_pro", "monthly").First(&googleMapping).Error
	require.NoError(t, err)
	assert.True(t, googleMapping.Enabled)
	assert.Equal(t, "all", googleMapping.Environment)

	// Test idempotency
	require.NoError(t, InitDefaultStoreProductMappings())
	var count int64
	require.NoError(t, DB.Model(&StoreProductMapping{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

