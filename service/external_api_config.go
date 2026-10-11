package service

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
)

const (
	// DefaultFacebookGraphAPIVersion is Meta Graph API v26.0 (active 2026)
	DefaultFacebookGraphAPIVersion = "v26.0"
	// DefaultLinkedInAPIVersion is LinkedIn Marketing REST API 202609 (active 2026)
	DefaultLinkedInAPIVersion = "202609"
)

var (
	facebookVersionRegex = regexp.MustCompile(`^v[0-9]+(\.[0-9]+)?$`)
	linkedinVersionRegex = regexp.MustCompile(`^[0-9]{6}$`)
)

// GetFacebookGraphAPIVersion returns the validated Facebook Graph API version.
// Configurable via FACEBOOK_GRAPH_API_VERSION, defaulting to v26.0.
func GetFacebookGraphAPIVersion() string {
	val := strings.TrimSpace(os.Getenv("FACEBOOK_GRAPH_API_VERSION"))
	if val == "" {
		return DefaultFacebookGraphAPIVersion
	}
	if !facebookVersionRegex.MatchString(val) {
		common.SysLog(fmt.Sprintf("[ExternalAPIConfig] Invalid FACEBOOK_GRAPH_API_VERSION %q, falling back to default %s", val, DefaultFacebookGraphAPIVersion))
		return DefaultFacebookGraphAPIVersion
	}
	return val
}

// GetLinkedInAPIVersion returns the validated LinkedIn REST API version.
// Configurable via LINKEDIN_API_VERSION, defaulting to 202609 (YYYYMM format).
func GetLinkedInAPIVersion() string {
	val := strings.TrimSpace(os.Getenv("LINKEDIN_API_VERSION"))
	if val == "" {
		return DefaultLinkedInAPIVersion
	}
	if !linkedinVersionRegex.MatchString(val) {
		common.SysLog(fmt.Sprintf("[ExternalAPIConfig] Invalid LINKEDIN_API_VERSION %q, falling back to default %s", val, DefaultLinkedInAPIVersion))
		return DefaultLinkedInAPIVersion
	}
	return val
}

// ValidateExternalAPIVersions validates environment variable formats without logging credentials.
func ValidateExternalAPIVersions() error {
	fb := strings.TrimSpace(os.Getenv("FACEBOOK_GRAPH_API_VERSION"))
	if fb != "" && !facebookVersionRegex.MatchString(fb) {
		return fmt.Errorf("invalid FACEBOOK_GRAPH_API_VERSION %q (expected format like v26.0)", fb)
	}

	li := strings.TrimSpace(os.Getenv("LINKEDIN_API_VERSION"))
	if li != "" && !linkedinVersionRegex.MatchString(li) {
		return fmt.Errorf("invalid LINKEDIN_API_VERSION %q (expected YYYYMM format like 202609)", li)
	}

	return nil
}

// LogExternalAPIConfigSummary outputs a safe startup summary with zero sensitive credentials.
func LogExternalAPIConfigSummary(ctx context.Context) {
	fbVer := GetFacebookGraphAPIVersion()
	liVer := GetLinkedInAPIVersion()
	canonicalBase := common.GetCanonicalBaseURL()
	apiBase := common.GetApiBaseURL()
	gscSite := common.GetGSCSiteURL()

	logger.LogInfo(ctx, fmt.Sprintf(
		"[ExternalAPIConfig] Active config: FB_Graph_API=%s, LinkedIn_API=%s, Canonical_Origin=%s, API_Origin=%s, GSC_Site_URL=%s",
		fbVer, liVer, canonicalBase, apiBase, gscSite,
	))
}
