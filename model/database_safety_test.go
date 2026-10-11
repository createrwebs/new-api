package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCheckDatabaseSafety(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Case 1: SQLite with default settings passes
	t.Setenv("ALLOWED_DATABASE_NAME", "")
	t.Setenv("ENFORCE_LOCAL_DB_SAFETY", "")
	dbName, err := CheckDatabaseSafety(db, common.DatabaseTypeSQLite)
	assert.NoError(t, err)
	assert.Equal(t, common.SQLitePath, dbName)

	// Case 2: Matching allowed database name passes
	t.Setenv("ALLOWED_DATABASE_NAME", common.SQLitePath)
	dbName, err = CheckDatabaseSafety(db, common.DatabaseTypeSQLite)
	assert.NoError(t, err)
	assert.Equal(t, common.SQLitePath, dbName)

	// Case 3: Mismatched allowed database name fails
	t.Setenv("ALLOWED_DATABASE_NAME", "tora_production")
	_, err = CheckDatabaseSafety(db, common.DatabaseTypeSQLite)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE SAFETY GUARD TRIGGERED")

	// Case 4: Nil database fails
	_, err = CheckDatabaseSafety(nil, common.DatabaseTypeSQLite)
	assert.Error(t, err)
}
