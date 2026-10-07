package model

import (
	"errors"
	"time"
)

// Provider health statuses
const (
	ProviderHealthActive         = "ACTIVE"
	ProviderHealthDegraded       = "DEGRADED"
	ProviderHealthDisabled       = "DISABLED"
	ProviderHealthBillingBlocked = "BILLING_BLOCKED"
	ProviderHealthAuthFailed     = "AUTH_FAILED"
	ProviderHealthRateLimited    = "RATE_LIMITED"
)

// Supported protocol identifiers
const (
	ProtocolWaveSpeedV3           = "WAVESPEED_V3"
	ProtocolKieJobsV1             = "KIE_JOBS_V1"
	ProtocolFalQueue              = "FAL_QUEUE"
	ProtocolReplicatePredictions  = "REPLICATE_PREDICTIONS"
	ProtocolMock                  = "MOCK"
)

// Route activation states (Queue 2G Section 22)
const (
	RouteStatusDraft              = "DRAFT"
	RouteStatusContractVerified   = "CONTRACT_VERIFIED"
	RouteStatusCredentialRequired = "CREDENTIAL_REQUIRED"
	RouteStatusReadyForCanary     = "READY_FOR_CANARY"
	RouteStatusActive             = "ACTIVE"
	RouteStatusDegraded           = "DEGRADED"
	RouteStatusBillingBlocked     = "BILLING_BLOCKED"
	RouteStatusDisabled           = "DISABLED"
)

// Provider price sources (Queue 2G Section 18)
const (
	PriceSourceRemoteDynamic  = "REMOTE_DYNAMIC"
	PriceSourceRemoteCatalog  = "REMOTE_CATALOG"
	PriceSourceManualVerified = "MANUAL_VERIFIED"
	PriceSourceUnknown        = "UNKNOWN"
)

// StudioProviderConfig defines a server-side media provider without storing secret values (Section 3).
type StudioProviderConfig struct {
	Id           string `json:"id" gorm:"primaryKey;type:varchar(32)"` // "wavespeed", "kie", "fal", "replicate", "mock"
	Name         string `json:"name" gorm:"type:varchar(64);not null"`
	BaseURL      string `json:"base_url" gorm:"type:varchar(255);not null"`
	SecretEnv    string `json:"secret_env" gorm:"type:varchar(64);not null"` // e.g. "WAVESPEED_API_KEY", "KIE_API_KEY", "FAL_KEY"
	AuthType     string `json:"auth_type" gorm:"type:varchar(32);default:'BEARER'"` // BEARER, KEY_HEADER, CUSTOM
	Protocol     string `json:"protocol" gorm:"type:varchar(32);not null"` // WAVESPEED_V3, KIE_JOBS_V1, FAL_QUEUE, REPLICATE_PREDICTIONS, MOCK
	Enabled      bool   `json:"enabled" gorm:"default:true"`
	Priority     int    `json:"priority" gorm:"default:1"`
	HealthStatus string `json:"health_status" gorm:"type:varchar(32);default:'ACTIVE'"` // ACTIVE, DEGRADED, DISABLED, BILLING_BLOCKED, AUTH_FAILED, RATE_LIMITED
	BalanceUSD   float64 `json:"balance_usd" gorm:"type:numeric(10,4);default:0.0"`
	LastBalanceAt int64  `json:"last_balance_at" gorm:"bigint;default:0"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
}

func (c *StudioProviderConfig) TableName() string {
	return "studio_provider_configs"
}

// StudioModelRoute represents a provider-neutral model route record (Section 4).
type StudioModelRoute struct {
	Id               string  `json:"id" gorm:"primaryKey;type:varchar(64)"` // e.g. "ws-birefnet", "kie-flux-schnell", "fal-birefnet"
	LogicalTool      string  `json:"logical_tool" gorm:"type:varchar(64);index;not null"` // "background-remove", "image-generate"
	ProviderId       string  `json:"provider_id" gorm:"type:varchar(32);index;not null"` // references StudioProviderConfig.Id
	Protocol         string  `json:"protocol" gorm:"type:varchar(32);not null"`
	ProviderModelId  string  `json:"provider_model_id" gorm:"type:varchar(128);not null"`
	QualityTier      string  `json:"quality_tier" gorm:"type:varchar(32);default:'QUALITY';index"` // FAST, QUALITY, PREMIUM
	Status           string  `json:"status" gorm:"type:varchar(32);default:'ACTIVE';index"` // DRAFT, CONTRACT_VERIFIED, CREDENTIAL_REQUIRED, READY_FOR_CANARY, ACTIVE, DEGRADED, BILLING_BLOCKED, DISABLED
	Enabled          bool    `json:"enabled" gorm:"default:true;index"`
	InputMapping     string  `json:"input_mapping" gorm:"type:text"` // JSON mapping rules
	OutputMapping    string  `json:"output_mapping" gorm:"type:text"` // JSON mapping rules
	Capabilities     string  `json:"capabilities" gorm:"type:text"` // JSON array e.g. ["aspect_ratio", "seed"]
	PricingStrategy  string  `json:"pricing_strategy" gorm:"type:varchar(32);default:'FIXED_COGS'"` // FIXED_COGS, DYNAMIC_API, PER_SECOND
	BaseCostUSD      float64 `json:"base_cost_usd" gorm:"type:numeric(10,4);default:0.005"`
	EffectiveCostUSD float64 `json:"effective_cost_usd" gorm:"type:numeric(10,4);default:0.005"`
	Currency         string  `json:"currency" gorm:"type:varchar(8);default:'USD'"`
	PriceSource      string  `json:"price_source" gorm:"type:varchar(32);default:'MANUAL_VERIFIED'"` // REMOTE_DYNAMIC, REMOTE_CATALOG, MANUAL_VERIFIED, UNKNOWN
	PriceSourceRef   string  `json:"price_source_ref" gorm:"type:varchar(255)"` // URL / Doc ref
	PriceVerifiedAt  int64   `json:"price_verified_at" gorm:"bigint;default:0"`
	PricingVersion   string  `json:"pricing_version" gorm:"type:varchar(32);default:'v2_canonical'"`
	Priority         int     `json:"priority" gorm:"type:int;default:1"`
	HealthWeight     float64 `json:"health_weight" gorm:"type:numeric(5,2);default:1.0"`
	CostWeight       float64 `json:"cost_weight" gorm:"type:numeric(5,2);default:1.0"`
	QualityWeight    float64 `json:"quality_weight" gorm:"type:numeric(5,2);default:1.0"`
	LatencyWeight    float64 `json:"latency_weight" gorm:"type:numeric(5,2);default:1.0"`
	MinMargin        float64 `json:"min_margin" gorm:"type:numeric(5,2);default:60.0"`
	ConsecutiveFails int     `json:"consecutive_fails" gorm:"type:int;default:0"`
	P50LatencyMs     int64   `json:"p50_latency_ms" gorm:"bigint;default:0"`
	P95LatencyMs     int64   `json:"p95_latency_ms" gorm:"bigint;default:0"`
	SuccessRate      float64 `json:"success_rate" gorm:"type:numeric(5,2);default:100.0"`
	RouteMetadata    string  `json:"route_metadata" gorm:"type:text"`
	CreatedAt        int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt        int64   `json:"updated_at" gorm:"bigint"`
}

func (r *StudioModelRoute) TableName() string {
	return "studio_model_routes"
}

// GetStudioProviderConfig fetches a provider config by ID.
func GetStudioProviderConfig(id string) (*StudioProviderConfig, error) {
	if DB == nil {
		return nil, errors.New("db is not initialized")
	}
	var cfg StudioProviderConfig
	err := DB.Where("id = ?", id).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetAllStudioProviderConfigs fetches all provider configs.
func GetAllStudioProviderConfigs() ([]StudioProviderConfig, error) {
	if DB == nil {
		return nil, nil
	}
	var configs []StudioProviderConfig
	err := DB.Order("priority ASC, id ASC").Find(&configs).Error
	return configs, err
}

// SaveStudioProviderConfig creates or updates a provider configuration.
func SaveStudioProviderConfig(cfg *StudioProviderConfig) error {
	if DB == nil {
		return errors.New("db is not initialized")
	}
	now := time.Now().Unix()
	if cfg.CreatedAt == 0 {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	return DB.Save(cfg).Error
}

// GetStudioModelRoutesByTool fetches all active routes for a logical tool.
func GetStudioModelRoutesByTool(toolId string) ([]StudioModelRoute, error) {
	if DB == nil {
		return nil, nil
	}
	var routes []StudioModelRoute
	err := DB.Where("logical_tool = ?", toolId).
		Order("priority ASC, effective_cost_usd ASC").
		Find(&routes).Error
	return routes, err
}

// GetAllStudioModelRoutes fetches all routes in the system.
func GetAllStudioModelRoutes() ([]StudioModelRoute, error) {
	if DB == nil {
		return nil, nil
	}
	var routes []StudioModelRoute
	err := DB.Order("logical_tool ASC, priority ASC").Find(&routes).Error
	return routes, err
}

// GetStudioModelRoute fetches a single route by ID.
func GetStudioModelRoute(id string) (*StudioModelRoute, error) {
	if DB == nil {
		return nil, errors.New("db is not initialized")
	}
	var route StudioModelRoute
	err := DB.Where("id = ?", id).First(&route).Error
	if err != nil {
		return nil, err
	}
	return &route, nil
}

// SaveStudioModelRoute creates or updates a route.
func SaveStudioModelRoute(route *StudioModelRoute) error {
	if DB == nil {
		return errors.New("db is not initialized")
	}
	now := time.Now().Unix()
	if route.CreatedAt == 0 {
		route.CreatedAt = now
	}
	route.UpdatedAt = now
	return DB.Save(route).Error
}

// UpdateRouteHealthStats updates runtime performance metrics for a route.
func UpdateRouteHealthStats(id string, latencyMs int64, success bool) error {
	if DB == nil {
		return nil
	}
	var route StudioModelRoute
	if err := DB.Where("id = ?", id).First(&route).Error; err != nil {
		return err
	}

	if success {
		route.ConsecutiveFails = 0
		if route.P50LatencyMs == 0 {
			route.P50LatencyMs = latencyMs
		} else {
			route.P50LatencyMs = (route.P50LatencyMs*4 + latencyMs) / 5
		}
		route.SuccessRate = (route.SuccessRate*9 + 100.0) / 10.0
	} else {
		route.ConsecutiveFails++
		route.SuccessRate = (route.SuccessRate * 9) / 10.0
	}
	route.UpdatedAt = time.Now().Unix()
	return DB.Save(&route).Error
}

// DeleteStudioModelRoute deletes a route by ID.
func DeleteStudioModelRoute(id string) error {
	if DB == nil {
		return errors.New("db is not initialized")
	}
	return DB.Where("id = ?", id).Delete(&StudioModelRoute{}).Error
}
