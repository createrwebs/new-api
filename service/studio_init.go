package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

var (
	GlobalStudioService *StudioService
	studioOnce          sync.Once
)

// InitStudio initializes the Studio subsystem, registers providers, and seeds catalog.
func InitStudio(db *gorm.DB) error {
	var initErr error
	studioOnce.Do(func() {
		if db != nil {
			if err := model.EnsureStudioTables(db); err != nil {
				common.SysError(fmt.Sprintf("failed ensuring studio tables: %v", err))
				initErr = err
				return
			}
			if err := SeedStudioCatalog(db); err != nil {
				common.SysError(fmt.Sprintf("failed seeding studio catalog: %v", err))
				initErr = err
				return
			}
		}

		mockProvider := NewDeterministicMockProvider(MockModeInstantSuccess)
		falProvider := NewFalProvider()
		replicateProvider := NewReplicateProvider()

		GlobalStudioService = NewStudioService(mockProvider, falProvider, replicateProvider)
		if db != nil {
			StartStudioAssetCleanupWorker(db)
		}
		common.SysLog("Tora Studio V1 service initialized successfully")
	})

	return initErr
}

// GetStudioService returns the singleton StudioService, ensuring initialization.
func GetStudioService() *StudioService {
	if GlobalStudioService == nil {
		_ = InitStudio(model.DB)
	}
	return GlobalStudioService
}

// LogStudioStatus prints startup status for Studio providers.
func LogStudioStatus(ctx context.Context) {
	svc := GetStudioService()
	if svc == nil {
		return
	}
	fal := svc.GetProvider("fal")
	if fal != nil {
		if falP, ok := fal.(*FalProvider); ok {
			if err := falP.ValidateConfiguration(); err != nil {
				common.SysLog("[Studio] fal.ai provider status: OPERATOR_BLOCKED (API key not set)")
			} else {
				common.SysLog("[Studio] fal.ai provider status: ACTIVE (API key configured)")
			}
		}
	}
	replicate := svc.GetProvider("replicate")
	if replicate != nil {
		if repP, ok := replicate.(*ReplicateProvider); ok {
			if err := repP.ValidateConfiguration(); err != nil {
				common.SysLog("[Studio] replicate provider status: OPERATOR_BLOCKED (API token not set)")
			} else {
				common.SysLog("[Studio] replicate provider status: ACTIVE (API token configured)")
			}
		}
	}
	common.SysLog("[Studio] mock provider status: ACTIVE (Deterministic test double ready)")
}
