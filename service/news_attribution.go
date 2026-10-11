package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	ConversionOrganicLanding    = "organic_landing"
	ConversionSocialLanding     = "social_landing"
	ConversionSignup            = "signup"
	ConversionApiKeyCreated     = "api_key_created"
	ConversionFirstInference    = "first_successful_inference"
	ConversionTopup             = "topup"
	ConversionSubscription      = "subscription"
)

// BuildAttributionURL generates a canonical URL tagged with UTM parameters and content tracking
func BuildAttributionURL(baseURL, source, medium, campaign, contentId string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := u.Query()
	if source != "" {
		q.Set("utm_source", strings.ToLower(source))
	}
	if medium != "" {
		q.Set("utm_medium", strings.ToLower(medium))
	}
	if campaign != "" {
		q.Set("utm_campaign", strings.ToLower(campaign))
	}
	if contentId != "" {
		q.Set("utm_content", contentId)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// HashClientIdentity creates a privacy-preserving one-way pseudonym from IP and User Agent using keyed HMAC-SHA256
func HashClientIdentity(ip, userAgent string) string {
	key := os.Getenv("ATTRIBUTION_HASH_KEY")
	if key == "" {
		key = "tora_attribution_secure_hmac_key_2026"
	}
	mac := hmac.New(sha256.New, []byte(key))
	// Zero raw IPs stored, normalized inputs
	normalizedIP := strings.TrimSpace(ip)
	normalizedUA := strings.TrimSpace(userAgent)
	mac.Write([]byte(normalizedIP + "|" + normalizedUA))
	hash := hex.EncodeToString(mac.Sum(nil))
	if len(hash) > 32 {
		return hash[:32]
	}
	return hash
}

// RecordConversion records a funnel progression event linked to content attribution
func RecordConversion(
	eventType string,
	contentId string,
	utmSource string,
	utmMedium string,
	utmCampaign string,
	clientIP string,
	userAgent string,
	userId int,
	revenueTHB float64,
	metaJSON string,
) error {
	visitorHash := HashClientIdentity(clientIP, userAgent)

	evt := &model.NewsConversionEvent{
		EventType:    eventType,
		ContentId:    contentId,
		UtmSource:    utmSource,
		UtmMedium:    utmMedium,
		UtmCampaign:  utmCampaign,
		VisitorHash:  visitorHash,
		UserId:       userId,
		RevenueTHB:   revenueTHB,
		MetadataJSON: metaJSON,
		CreatedAt:    common.GetTimestamp(),
	}

	return model.CreateNewsConversionEvent(evt)
}
