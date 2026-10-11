package model

import (
	"fmt"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// CheckDatabaseSafety verifies that the connected database conforms to safety invariants.
// 1. If ALLOWED_DATABASE_NAME is non-empty, the active database MUST match it exactly.
// 2. If ENFORCE_LOCAL_DB_SAFETY is true, connecting to protected production ("new-api")
//    or live staging ("tora_staging") databases is strictly forbidden and aborts.
func CheckDatabaseSafety(db *gorm.DB, dbType common.DatabaseType) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database connection is nil")
	}

	var currentDB string
	switch dbType {
	case common.DatabaseTypePostgreSQL:
		row := db.Raw("SELECT current_database()").Row()
		if err := row.Scan(&currentDB); err != nil {
			return "", fmt.Errorf("failed to query current_database(): %w", err)
		}
	case common.DatabaseTypeMySQL:
		row := db.Raw("SELECT DATABASE()").Row()
		if err := row.Scan(&currentDB); err != nil {
			return "", fmt.Errorf("failed to query DATABASE(): %w", err)
		}
	case common.DatabaseTypeSQLite:
		currentDB = common.SQLitePath
	default:
		return "", nil
	}

	allowedDB := os.Getenv("ALLOWED_DATABASE_NAME")
	if allowedDB != "" && !strings.EqualFold(currentDB, allowedDB) {
		return currentDB, fmt.Errorf("FATAL: DATABASE SAFETY GUARD TRIGGERED: active database is %q, but ALLOWED_DATABASE_NAME requires %q", currentDB, allowedDB)
	}

	// Production database "new-api" is permanently protected against test/staging access.
	if strings.EqualFold(currentDB, "new-api") && os.Getenv("ALLOW_PRODUCTION_DB") != "true" {
		return currentDB, fmt.Errorf("FATAL: DATABASE SAFETY GUARD TRIGGERED: attempted connection to protected production database %q without explicit override. Aborting immediately!", currentDB)
	}

	// When ENFORCE_LOCAL_DB_SAFETY is true, both production and live staging are forbidden.
	if os.Getenv("ENFORCE_LOCAL_DB_SAFETY") == "true" {
		if strings.EqualFold(currentDB, "new-api") || strings.EqualFold(currentDB, "tora_staging") {
			return currentDB, fmt.Errorf("FATAL: DATABASE SAFETY GUARD TRIGGERED: attempted connection to protected database %q under local safety guard mode. Aborting immediately!", currentDB)
		}
	}

	common.SysLog(fmt.Sprintf("DATABASE_ENGINE=%s DATABASE_NAME=%s (safety guard verified)", dbType, currentDB))
	return currentDB, nil
}
