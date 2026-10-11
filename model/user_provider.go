package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ProviderOpenRouter = "openrouter"
	ProviderCustom     = "custom"

	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
)

var (
	ErrProviderNotFound       = errors.New("provider not found")
	ErrProviderDuplicate      = errors.New("provider already configured for this user")
	ErrProviderInvalidType    = errors.New("invalid provider type, must be 'openrouter' or 'custom'")
	ErrProviderNameEmpty      = errors.New("provider name cannot be empty")
	ErrProviderAPIKeyEmpty    = errors.New("provider api key cannot be empty")
	ErrProviderBaseURLEmpty   = errors.New("base_url is required for custom provider")
	ErrProviderBaseURLInvalid = errors.New("base_url must be a valid http or https URL")
)

type UserProvider struct {
	Id              int64  `json:"id" gorm:"primaryKey"`
	UserId          int    `json:"user_id" gorm:"not null;uniqueIndex:idx_user_provider,priority:1;index"`
	Provider        string `json:"provider" gorm:"type:varchar(32);not null;uniqueIndex:idx_user_provider,priority:2"`
	Name            string `json:"name" gorm:"type:varchar(64);not null"`
	BaseURL         string `json:"base_url" gorm:"type:varchar(255)"`
	APIKeyEncrypted string `json:"-" gorm:"type:text;not null"`
	APIKeyMasked    string `json:"api_key" gorm:"type:varchar(16);not null"`
	Enabled         bool   `json:"enabled" gorm:"not null"`
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (UserProvider) TableName() string {
	return "user_providers"
}

// AAD returns the contextual authentication binding used by AES-256-GCM.
func (p *UserProvider) AAD() string {
	return fmt.Sprintf("user_provider:%d:%s", p.UserId, p.Provider)
}

// SetAPIKey encrypts the plaintext API key using AES-256-GCM and generates a masked key.
// The plaintext key is never stored in memory beyond this method.
func (p *UserProvider) SetAPIKey(plaintext string) error {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return ErrProviderAPIKeyEmpty
	}
	encrypted, err := common.EncryptBYOKSecret(plaintext, p.AAD())
	if err != nil {
		return err
	}
	p.APIKeyEncrypted = encrypted
	p.APIKeyMasked = common.MaskAPIKey(plaintext)
	return nil
}

// DecryptAPIKey returns the decrypted API key for upstream relay or testing.
func (p *UserProvider) DecryptAPIKey() (string, error) {
	if p.APIKeyEncrypted == "" {
		return "", ErrProviderAPIKeyEmpty
	}
	return common.DecryptBYOKSecret(p.APIKeyEncrypted, p.AAD())
}

// NormalizeBaseURL trims trailing slashes, validates the URL scheme and host,
// and enforces strict SSRF protection against private/loopback/metadata destinations.
func NormalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return "", ErrProviderBaseURLEmpty
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", ErrProviderBaseURLInvalid
	}
	if err := common.BYOKSSRFProtection.ValidateURL(raw); err != nil {
		return "", fmt.Errorf("%w: %v", ErrProviderBaseURLInvalid, err)
	}
	return raw, nil
}

// ValidateUserProviderInput validates provider fields before persistence.
func ValidateUserProviderInput(provider, name, baseURL string) (normalizedBaseURL string, err error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != ProviderOpenRouter && provider != ProviderCustom {
		return "", ErrProviderInvalidType
	}
	if strings.TrimSpace(name) == "" {
		return "", ErrProviderNameEmpty
	}
	if provider == ProviderCustom {
		cleanURL, err := NormalizeBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		return cleanURL, nil
	}
	if strings.TrimSpace(baseURL) == "" {
		return DefaultOpenRouterBaseURL, nil
	}
	return NormalizeBaseURL(baseURL)
}

// GetUserProviders returns all BYOK providers configured for the specified user.
func GetUserProviders(userID int) ([]*UserProvider, error) {
	var providers []*UserProvider
	err := DB.Where("user_id = ?", userID).Order("id asc").Find(&providers).Error
	return providers, err
}

// GetUserProviderByID retrieves a provider scoped strictly to the authenticated user.
func GetUserProviderByID(userID int, id int64) (*UserProvider, error) {
	var provider UserProvider
	err := DB.Where("id = ? AND user_id = ?", id, userID).First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return &provider, nil
}

// GetUserProviderByType retrieves a user's provider by type (e.g. "openrouter", "custom").
func GetUserProviderByType(userID int, providerType string) (*UserProvider, error) {
	var provider UserProvider
	err := DB.Where("user_id = ? AND provider = ?", userID, providerType).First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return &provider, nil
}

// CreateUserProvider persists a new UserProvider ensuring uniqueness per (user_id, provider).
func CreateUserProvider(p *UserProvider) error {
	var count int64
	if err := DB.Model(&UserProvider{}).
		Where("user_id = ? AND provider = ?", p.UserId, p.Provider).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrProviderDuplicate
	}
	return DB.Omit("id").Create(p).Error
}

// UpdateUserProvider saves updates to an existing UserProvider strictly scoped to user_id.
func UpdateUserProvider(p *UserProvider) error {
	result := DB.Model(&UserProvider{}).
		Where("id = ? AND user_id = ?", p.Id, p.UserId).
		Updates(map[string]any{
			"name":              p.Name,
			"base_url":          p.BaseURL,
			"api_key_encrypted": p.APIKeyEncrypted,
			"api_key_masked":    p.APIKeyMasked,
			"enabled":           p.Enabled,
			"updated_at":        common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}

// DeleteUserProvider removes a provider scoped strictly to user_id.
func DeleteUserProvider(userID int, id int64) error {
	result := DB.Where("id = ? AND user_id = ?", id, userID).Delete(&UserProvider{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}
