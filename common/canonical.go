package common

import (
	"os"
	"strings"
)

const (
	DefaultCanonicalBaseURL = "https://www.toraapi.com"
	DefaultApiBaseURL       = "https://www.toraapi.com"
	DefaultGSCSiteURL       = "sc-domain:toraapi.com"
)

// GetCanonicalBaseURL returns the public canonical web origin for crawlable content and SEO metadata.
// Configurable via CANONICAL_BASE_URL or PUBLIC_WEB_URL, defaulting to https://www.toraapi.com.
func GetCanonicalBaseURL() string {
	val := strings.TrimSpace(os.Getenv("CANONICAL_BASE_URL"))
	if val == "" {
		val = strings.TrimSpace(os.Getenv("PUBLIC_WEB_URL"))
	}
	if val == "" {
		return DefaultCanonicalBaseURL
	}
	return strings.TrimRight(val, "/")
}

// GetApiBaseURL returns the dedicated API endpoint origin.
// Configurable via API_BASE_URL, defaulting to https://www.toraapi.com.
func GetApiBaseURL() string {
	val := strings.TrimSpace(os.Getenv("API_BASE_URL"))
	if val == "" {
		return DefaultApiBaseURL
	}
	return strings.TrimRight(val, "/")
}

// GetGSCSiteURL returns the Google Search Console property URL or domain property identifier.
// Configurable via GSC_SITE_URL, defaulting to sc-domain:toraapi.com.
func GetGSCSiteURL() string {
	val := strings.TrimSpace(os.Getenv("GSC_SITE_URL"))
	if val == "" {
		return DefaultGSCSiteURL
	}
	return val
}

// GetGSCCredentialsFile returns the mounted path to the Google Service Account JSON
func GetGSCCredentialsFile() string {
	return strings.TrimSpace(os.Getenv("GSC_CREDENTIALS_FILE"))
}

// GetGSCCredentialsJSON returns raw Google Service Account JSON content
func GetGSCCredentialsJSON() string {
	return strings.TrimSpace(os.Getenv("GSC_CREDENTIALS_JSON"))
}

