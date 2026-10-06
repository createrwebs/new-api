package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
)

// Store platforms
const (
	StorePlatformApple  = "apple"
	StorePlatformGoogle = "google"
)

// Store transaction / subscription statuses
const (
	StoreStatusActive    = "active"
	StoreStatusCancelled = "cancelled"
	StoreStatusExpired   = "expired"
	StoreStatusRevoked   = "revoked"
	StoreStatusRefunded  = "refunded"
	StoreStatusPending   = "pending"
)

// Store billing errors
var (
	ErrStoreProductNotMapped                 = errors.New("store product not mapped to internal plan")
	ErrStoreVerificationFailed               = errors.New("store verification failed")
	ErrStoreEnvironmentMismatch              = errors.New("store environment mismatch")
	ErrStoreBundleMismatch                   = errors.New("store bundle/package identifier mismatch")
	ErrStoreTransactionBoundToAnotherAccount = errors.New("store transaction bound to another account")
	ErrStoreTransactionExpired               = errors.New("store transaction is expired")
	ErrStoreTransactionRevoked               = errors.New("store transaction is revoked")
	ErrStoreServerNotConfigured              = errors.New("store server billing not configured")
	ErrStoreAccountBindingMismatch           = errors.New("store account token mismatch")
	ErrStoreAccountTokenRequired             = errors.New("store account token required for initial binding")
)

// StoreProductMapping maps Apple/Google store product identifiers to internal SubscriptionPlan IDs.
type StoreProductMapping struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Platform        string `json:"platform" gorm:"type:varchar(32);index:idx_store_product_lookup,priority:1;not null"`
	StoreProductId  string `json:"store_product_id" gorm:"type:varchar(128);index:idx_store_product_lookup,priority:2;not null"`
	StoreBasePlanId string `json:"store_base_plan_id" gorm:"type:varchar(128);index:idx_store_product_lookup,priority:3;default:''"`
	InternalPlanId  int    `json:"internal_plan_id" gorm:"index;not null"`
	Environment     string `json:"environment" gorm:"type:varchar(32);default:'production'"` // "production", "sandbox", "all"
	Enabled         bool   `json:"enabled" gorm:"default:true"`
	CreatedAt       int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt       int64  `json:"updated_at" gorm:"bigint"`
}

func (m *StoreProductMapping) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	m.CreatedAt = now
	m.UpdatedAt = now
	return nil
}

func (m *StoreProductMapping) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = common.GetTimestamp()
	return nil
}

// StoreSubscriptionBinding locks a recurring store subscription (original_transaction_id or purchase_token)
// to a specific internal New-API user ID, preventing cross-account reuse.
type StoreSubscriptionBinding struct {
	Id                  int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Platform            string `json:"platform" gorm:"type:varchar(32);index:idx_store_binding,priority:1;not null"`
	StoreOriginalId     string `json:"store_original_id" gorm:"type:varchar(255);uniqueIndex:idx_store_orig_plat,priority:2;not null"`
	UserId              int    `json:"user_id" gorm:"index;not null"`
	InternalPlanId      int    `json:"internal_plan_id" gorm:"not null"`
	ActiveUserSubId     int    `json:"active_user_sub_id" gorm:"index;default:0"`
	Status              string `json:"status" gorm:"type:varchar(32);default:'active'"`
	LatestTransactionId string `json:"latest_transaction_id" gorm:"type:varchar(255);default:''"`
	PurchaseTime        int64  `json:"purchase_time" gorm:"bigint;default:0"`
	ExpiresTime         int64  `json:"expires_time" gorm:"bigint;default:0"`
	RevocationTime      int64  `json:"revocation_time" gorm:"bigint;default:0"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint"`
}

func (b *StoreSubscriptionBinding) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	b.CreatedAt = now
	b.UpdatedAt = now
	return nil
}

func (b *StoreSubscriptionBinding) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = common.GetTimestamp()
	return nil
}

// StoreTransaction records individual purchase/renewal transactions for replay prevention and idempotency.
type StoreTransaction struct {
	Id                  int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Platform            string `json:"platform" gorm:"type:varchar(32);index:idx_store_tx_platform,priority:1;not null"`
	StoreTransactionId  string `json:"store_transaction_id" gorm:"type:varchar(255);uniqueIndex:idx_store_tx_uniq,priority:2;not null"`
	StoreOriginalId     string `json:"store_original_id" gorm:"type:varchar(255);index;not null"`
	UserId              int    `json:"user_id" gorm:"index;not null"`
	InternalPlanId      int    `json:"internal_plan_id" gorm:"not null"`
	UserSubscriptionId  int    `json:"user_subscription_id" gorm:"index;default:0"`
	SubscriptionOrderId int    `json:"subscription_order_id" gorm:"index;default:0"`
	Environment         string `json:"environment" gorm:"type:varchar(32);default:'production'"`
	Status              string `json:"status" gorm:"type:varchar(32);not null"` // verified, active, revoked, refunded
	PurchaseTime        int64  `json:"purchase_time" gorm:"bigint;default:0"`
	ExpiresTime         int64  `json:"expires_time" gorm:"bigint;default:0"`
	RevocationTime      int64  `json:"revocation_time" gorm:"bigint;default:0"`
	RawEvidence         string `json:"-" gorm:"type:text"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint"`
}

func (t *StoreTransaction) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	t.CreatedAt = now
	t.UpdatedAt = now
	return nil
}

func (t *StoreTransaction) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = common.GetTimestamp()
	return nil
}

// VerifiedStorePurchase contains verified evidence returned from an authoritative store verifier.
type VerifiedStorePurchase struct {
	Platform               string
	StoreTransactionId     string
	StoreOriginalId        string
	LinkedPurchaseToken    string // For Google token transitions (upgrade/replacement)
	StoreProductId         string
	StoreBasePlanId        string
	PurchaseTime           int64 // Unix epoch seconds
	ExpiresTime            int64 // Unix epoch seconds
	RevocationTime         int64 // Unix epoch seconds, > 0 if revoked
	Environment                 string
	AppAccountToken             string // Apple StoreKit 2 appAccountToken (RFC 4122 UUID)
	ObfuscatedExternalAccountId string // Google Play externalAccountIdentifiers.obfuscatedExternalAccountId
	RawEvidence                 string
	IsRenewal                   bool
	IsPending                   bool
	IsTestPurchase              bool
}

// GetStoreAccountToken returns the authoritative store account token embedded in the transaction (Apple or Google).
func (v *VerifiedStorePurchase) GetStoreAccountToken() string {
	if v == nil {
		return ""
	}
	if v.AppAccountToken != "" {
		return v.AppAccountToken
	}
	return v.ObfuscatedExternalAccountId
}

// StorePurchaseResult is returned after transactional settlement.
type StorePurchaseResult struct {
	Status              string `json:"status"` // "active", "pending", "already_processed", "revoked"
	Platform            string `json:"platform"`
	StoreTransactionId  string `json:"store_transaction_id"`
	StoreOriginalId     string `json:"store_original_id"`
	PlanId              int    `json:"plan_id"`
	PlanTitle           string `json:"plan_title"`
	UserSubscriptionId  int    `json:"user_subscription_id"`
	ExpiresTime         int64  `json:"expires_time"`
	IsRenewal           bool   `json:"is_renewal"`
	AlreadyProcessed    bool   `json:"already_processed"`
}

// GetStoreProductMapping finds the configured mapping for a store product.
func GetStoreProductMapping(platform string, storeProductId string, storeBasePlanId string, env string) (*StoreProductMapping, error) {
	if platform == "" || storeProductId == "" {
		return nil, ErrStoreProductNotMapped
	}
	envLower := strings.ToLower(strings.TrimSpace(env))
	var mapping StoreProductMapping
	query := DB.Where("platform = ? AND store_product_id = ? AND enabled = ?", platform, storeProductId, true)
	if storeBasePlanId != "" {
		query = query.Where("(store_base_plan_id = ? OR store_base_plan_id = '')", storeBasePlanId)
	}
	if envLower != "" {
		query = query.Where("(LOWER(environment) = ? OR LOWER(environment) = 'all')", envLower)
	}
	if err := query.Order("store_base_plan_id desc, id desc").First(&mapping).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStoreProductNotMapped
		}
		return nil, err
	}
	return &mapping, nil
}

// ProcessStorePurchaseTx atomically verifies, binds, settles, and updates entitlement for a store purchase.
// Replay-safe, concurrency-safe, multi-account binding enforced.
func ProcessStorePurchaseTx(userId int, verified *VerifiedStorePurchase) (*StorePurchaseResult, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if verified == nil {
		return nil, errors.New("verified purchase is nil")
	}
	if verified.StoreTransactionId == "" || verified.StoreOriginalId == "" {
		return nil, errors.New("missing store transaction identifier")
	}

	// 1. Resolve internal plan mapping
	mapping, err := GetStoreProductMapping(verified.Platform, verified.StoreProductId, verified.StoreBasePlanId, verified.Environment)
	if err != nil {
		return nil, ErrStoreProductNotMapped
	}
	plan, err := GetSubscriptionPlanById(mapping.InternalPlanId)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()

	// Check if already revoked
	if verified.RevocationTime > 0 {
		return nil, ErrStoreTransactionRevoked
	}

	// Check if already expired
	if verified.ExpiresTime > 0 && verified.ExpiresTime <= now {
		return nil, ErrStoreTransactionExpired
	}

	result := &StorePurchaseResult{
		Platform:           verified.Platform,
		StoreTransactionId: verified.StoreTransactionId,
		StoreOriginalId:    verified.StoreOriginalId,
		PlanId:             plan.Id,
		PlanTitle:          plan.Title,
		ExpiresTime:        verified.ExpiresTime,
	}

	var upgradeGroup string
	var groupChanged bool

	err = DB.Transaction(func(tx *gorm.DB) error {
		// A. Lock user row to serialize concurrent purchases for this user
		var userRow User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userId).First(&userRow).Error; err != nil {
			return err
		}

		// B. Check if this exact transaction was already processed (Replay Protection)
		var existingTx StoreTransaction
		if err := tx.Where("platform = ? AND store_transaction_id = ?", verified.Platform, verified.StoreTransactionId).First(&existingTx).Error; err == nil {
			// Already recorded! Idempotent return without double crediting.
			result.Status = existingTx.Status
			result.UserSubscriptionId = existingTx.UserSubscriptionId
			result.AlreadyProcessed = true
			result.ExpiresTime = existingTx.ExpiresTime
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// C. Cross-Account Theft Check: Check StoreSubscriptionBinding
		var binding StoreSubscriptionBinding
		bindingFound := false
		if err := lockForUpdate(tx).Where("platform = ? AND store_original_id = ?", verified.Platform, verified.StoreOriginalId).First(&binding).Error; err == nil {
			bindingFound = true
			if binding.UserId != userId {
				return ErrStoreTransactionBoundToAnotherAccount
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// C2. Token Transition Chain Preservation (Google linkedPurchaseToken)
		if !bindingFound && verified.LinkedPurchaseToken != "" {
			var linkedBinding StoreSubscriptionBinding
			if err := lockForUpdate(tx).Where("platform = ? AND store_original_id = ?", verified.Platform, verified.LinkedPurchaseToken).First(&linkedBinding).Error; err == nil {
				if linkedBinding.UserId != userId {
					return ErrStoreTransactionBoundToAnotherAccount
				}
				// User matches prior subscription in chain: preserve ownership and update token
				bindingFound = true
				binding = linkedBinding
				binding.StoreOriginalId = verified.StoreOriginalId
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		// C3. Native Account Binding Verification (Apple appAccountToken & Google obfuscatedExternalAccountId)
		storeToken := verified.GetStoreAccountToken()
		if storeToken != "" {
			// A store account token was stamped into the transaction.
			// It MUST cryptographically match the target user ID for this environment.
			targetUserId := userId
			if bindingFound {
				targetUserId = binding.UserId
			}
			if !ValidateStoreAccountToken(storeToken, targetUserId, verified.Environment) {
				return ErrStoreAccountBindingMismatch
			}
		} else if !bindingFound && setting.StoreRequireAccountToken {
			// New un-bound store transaction without an account token is rejected when strict binding is required
			return ErrStoreAccountTokenRequired
		}

		// D. Determine target expiration time
		endUnix := verified.ExpiresTime
		if endUnix <= 0 {
			calcEnd, err := calcPlanEndTime(time.Unix(verified.PurchaseTime, 0), plan)
			if err != nil {
				return err
			}
			endUnix = calcEnd
			result.ExpiresTime = endUnix
		}

		// E. Check if existing active subscription can be renewed/extended
		var activeSub UserSubscription
		isRenewal := false
		if bindingFound && binding.ActiveUserSubId > 0 {
			if err := lockForUpdate(tx).Where("id = ? AND status = ?", binding.ActiveUserSubId, StoreStatusActive).First(&activeSub).Error; err == nil {
				isRenewal = true
			}
		}

		var finalUserSubId int

		if isRenewal {
			// Renewal: Extend end time and reset quota for the new period
			result.IsRenewal = true
			if endUnix > activeSub.EndTime {
				activeSub.EndTime = endUnix
			}
			// Reset used quota for the renewed period
			activeSub.AmountUsed = 0
			activeSub.UpdatedAt = now
			if err := tx.Save(&activeSub).Error; err != nil {
				return err
			}
			finalUserSubId = activeSub.Id
		} else {
			// New Subscription instance
			upgradeGroupTarget := strings.TrimSpace(plan.UpgradeGroup)
			prevGroup := ""
			if upgradeGroupTarget != "" {
				currentGroup, err := getUserGroupByIdTx(tx, userId)
				if err != nil {
					return err
				}
				if currentGroup != upgradeGroupTarget {
					prevGroup = currentGroup
					if err := tx.Model(&User{}).Where("id = ?", userId).Update("group", upgradeGroupTarget).Error; err != nil {
						return err
					}
					upgradeGroup = upgradeGroupTarget
					groupChanged = true
				}
			}

			allowWalletOverflow := true
			if plan.AllowWalletOverflow != nil {
				allowWalletOverflow = *plan.AllowWalletOverflow
			}

			newSub := &UserSubscription{
				UserId:              userId,
				PlanId:              plan.Id,
				AmountTotal:         plan.TotalAmount,
				AmountUsed:          0,
				StartTime:           verified.PurchaseTime,
				EndTime:             endUnix,
				Status:              StoreStatusActive,
				Source:              verified.Platform,
				UpgradeGroup:        upgradeGroupTarget,
				PrevUserGroup:       prevGroup,
				DowngradeGroup:      strings.TrimSpace(plan.DowngradeGroup),
				AllowWalletOverflow: allowWalletOverflow,
				CreatedAt:           now,
				UpdatedAt:           now,
			}
			if err := tx.Create(newSub).Error; err != nil {
				return err
			}
			finalUserSubId = newSub.Id
		}

		result.UserSubscriptionId = finalUserSubId
		result.Status = StoreStatusActive

		// F. Create SubscriptionOrder record for unified audit and financial tracking
		orderTradeNo := fmt.Sprintf("STORE_%s_%s", strings.ToUpper(verified.Platform), verified.StoreTransactionId)
		order := &SubscriptionOrder{
			UserId:          userId,
			PlanId:          plan.Id,
			Money:           plan.PriceAmount,
			TradeNo:         orderTradeNo,
			PaymentMethod:   verified.Platform,
			PaymentProvider: verified.Platform,
			Status:          common.TopUpStatusSuccess,
			CreateTime:      verified.PurchaseTime,
			CompleteTime:    now,
			ProviderPayload: fmt.Sprintf("orig_id=%s;env=%s", verified.StoreOriginalId, verified.Environment),
		}
		if err := tx.Create(order).Error; err != nil {
			// In case order trade_no already exists
			return err
		}

		// Upsert TopUp record
		if err := upsertSubscriptionTopUpTx(tx, order); err != nil {
			return err
		}

		// G. Upsert StoreSubscriptionBinding
		if bindingFound {
			binding.ActiveUserSubId = finalUserSubId
			binding.LatestTransactionId = verified.StoreTransactionId
			binding.InternalPlanId = plan.Id
			binding.Status = StoreStatusActive
			binding.ExpiresTime = endUnix
			binding.UpdatedAt = now
			if err := tx.Save(&binding).Error; err != nil {
				return err
			}
		} else {
			binding = StoreSubscriptionBinding{
				Platform:            verified.Platform,
				StoreOriginalId:     verified.StoreOriginalId,
				UserId:              userId,
				InternalPlanId:      plan.Id,
				ActiveUserSubId:     finalUserSubId,
				Status:              StoreStatusActive,
				LatestTransactionId: verified.StoreTransactionId,
				PurchaseTime:        verified.PurchaseTime,
				ExpiresTime:         endUnix,
				CreatedAt:           now,
				UpdatedAt:           now,
			}
			if err := tx.Create(&binding).Error; err != nil {
				return err
			}
		}

		// H. Record StoreTransaction (Unique idempotency key)
		storeTx := &StoreTransaction{
			Platform:            verified.Platform,
			StoreTransactionId:  verified.StoreTransactionId,
			StoreOriginalId:     verified.StoreOriginalId,
			UserId:              userId,
			InternalPlanId:      plan.Id,
			UserSubscriptionId:  finalUserSubId,
			SubscriptionOrderId: order.Id,
			Environment:         verified.Environment,
			Status:              StoreStatusActive,
			PurchaseTime:        verified.PurchaseTime,
			ExpiresTime:         endUnix,
			RawEvidence:         verified.RawEvidence,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		if err := tx.Create(storeTx).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if groupChanged && upgradeGroup != "" {
		refreshSubscriptionUserGroupCache(userId, "store subscription purchase")
	}

	logMsg := fmt.Sprintf("%s 订阅验证成功，套餐: %s，交易号: %s", strings.ToUpper(verified.Platform), plan.Title, verified.StoreTransactionId)
	RecordLog(userId, LogTypeTopup, logMsg)

	return result, nil
}

// ProcessStoreRevocationTx atomically revokes a subscription (refund or store revocation),
// immediately terminating active access and downgrading the user group.
func ProcessStoreRevocationTx(platform string, storeOriginalId string, revocationTime int64, reason string) error {
	if platform == "" || storeOriginalId == "" {
		return errors.New("missing store subscription identifier")
	}
	now := common.GetTimestamp()
	if revocationTime <= 0 {
		revocationTime = now
	}

	var userId int
	var cacheGroup string

	err := DB.Transaction(func(tx *gorm.DB) error {
		var binding StoreSubscriptionBinding
		if err := lockForUpdate(tx).Where("platform = ? AND store_original_id = ?", platform, storeOriginalId).First(&binding).Error; err != nil {
			return err
		}
		userId = binding.UserId
		binding.Status = StoreStatusRevoked
		binding.RevocationTime = revocationTime
		binding.UpdatedAt = now
		if err := tx.Save(&binding).Error; err != nil {
			return err
		}

		// Revoke active user subscription immediately
		if binding.ActiveUserSubId > 0 {
			var sub UserSubscription
			if err := lockForUpdate(tx).Where("id = ?", binding.ActiveUserSubId).First(&sub).Error; err == nil {
				sub.Status = StoreStatusCancelled
				sub.EndTime = now
				sub.UpdatedAt = now
				if err := tx.Save(&sub).Error; err != nil {
					return err
				}
				target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
				if err != nil {
					return err
				}
				if target != "" {
					cacheGroup = target
				}
			}
		}

		// Update store transactions to revoked
		tx.Model(&StoreTransaction{}).
			Where("platform = ? AND store_original_id = ?", platform, storeOriginalId).
			Updates(map[string]any{
				"status":          StoreStatusRevoked,
				"revocation_time": revocationTime,
				"updated_at":      now,
			})

		return nil
	})

	if err != nil {
		return err
	}

	if cacheGroup != "" && userId > 0 {
		refreshSubscriptionUserGroupCache(userId, "store subscription revocation")
	}

	if userId > 0 {
		msg := fmt.Sprintf("%s 订阅已撤销/退款，原订阅标识: %s，原因: %s", strings.ToUpper(platform), storeOriginalId, reason)
		RecordLog(userId, LogTypeTopup, msg)
	}

	return nil
}

// StoreCatalogProduct represents public catalog presentation information for mobile.
type StoreCatalogProduct struct {
	Platform        string `json:"platform"`
	StoreProductId  string `json:"product_id"`
	StoreBasePlanId string `json:"base_plan_id,omitempty"`
	InternalPlanId  int    `json:"plan_id"`
	Title           string `json:"title"`
	DurationUnit    string `json:"duration_unit"`
	DurationValue   int    `json:"duration_value"`
	TotalAmount     int64  `json:"total_amount"`
}

// GetEnabledStoreProductCatalog returns only active, enabled store products matching the server environment.
func GetEnabledStoreProductCatalog(platform string, env string) ([]StoreCatalogProduct, error) {
	envLower := strings.ToLower(strings.TrimSpace(env))
	query := DB.Model(&StoreProductMapping{}).Where("enabled = ?", true)
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if envLower != "" {
		query = query.Where("(LOWER(environment) = ? OR LOWER(environment) = 'all')", envLower)
	}
	var mappings []StoreProductMapping
	if err := query.Order("platform asc, store_product_id asc").Find(&mappings).Error; err != nil {
		return nil, err
	}
	result := make([]StoreCatalogProduct, 0, len(mappings))
	for _, m := range mappings {
		plan, err := GetSubscriptionPlanById(m.InternalPlanId)
		if err != nil || !plan.Enabled {
			continue
		}
		result = append(result, StoreCatalogProduct{
			Platform:        m.Platform,
			StoreProductId:  m.StoreProductId,
			StoreBasePlanId: m.StoreBasePlanId,
			InternalPlanId:  m.InternalPlanId,
			Title:           plan.Title,
			DurationUnit:    plan.DurationUnit,
			DurationValue:   plan.DurationValue,
			TotalAmount:     plan.TotalAmount,
		})
	}
	return result, nil
}

// InitDefaultStoreProductMappings seeds approved Apple and Google store mappings for Pro Monthly.
func InitDefaultStoreProductMappings() error {
	plan, err := EnsureProMonthlySubscriptionPlan()
	if err != nil {
		return err
	}

	// 1. Apple mapping: com.saascover.tora.pro.monthly
	var appleMapping StoreProductMapping
	err = DB.Where("platform = ? AND store_product_id = ?", StorePlatformApple, "com.saascover.tora.pro.monthly").First(&appleMapping).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		appleMapping = StoreProductMapping{
			Platform:       StorePlatformApple,
			StoreProductId: "com.saascover.tora.pro.monthly",
			InternalPlanId: plan.Id,
			Environment:    "all",
			Enabled:        true,
		}
		if err := DB.Create(&appleMapping).Error; err != nil {
			return err
		}
		common.SysLog("seeded approved store product mapping: Apple com.saascover.tora.pro.monthly")
	} else if err != nil {
		return err
	}

	// 2. Google mapping: tora_pro / monthly
	var googleMapping StoreProductMapping
	err = DB.Where("platform = ? AND store_product_id = ? AND store_base_plan_id = ?", StorePlatformGoogle, "tora_pro", "monthly").First(&googleMapping).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		googleMapping = StoreProductMapping{
			Platform:        StorePlatformGoogle,
			StoreProductId:  "tora_pro",
			StoreBasePlanId: "monthly",
			InternalPlanId:  plan.Id,
			Environment:     "all",
			Enabled:         true,
		}
		if err := DB.Create(&googleMapping).Error; err != nil {
			return err
		}
		common.SysLog("seeded approved store product mapping: Google tora_pro / monthly")
	} else if err != nil {
		return err
	}

	return nil
}
