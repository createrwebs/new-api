package service

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

var (
	ErrMarketplaceNotFound = errors.New("marketplace rule not found for specified platform and region")
)

// CanonicalMarketplaceRules defines official marketplace compliance standards for Southeast Asia & Global e-commerce.
var CanonicalMarketplaceRules = []model.MarketplaceRule{
	{
		Platform:      "shopee",
		Region:        "TH",
		ChannelName:   "Shopee Thailand (Mall & Standard)",
		Version:       "2026.1",
		LastAuditedAt: "2026-10-08",
		Requirements: model.PlatformRequirement{
			MinWidth:                  500,
			MaxWidth:                  2000,
			MinHeight:                 500,
			MaxHeight:                 2000,
			AllowedAspectRatios:       []string{"1:1"},
			AllowedFormats:            []string{"jpg", "jpeg", "png"},
			MaxFileSizeBytes:          2 * 1024 * 1024, // 2MB
			PureWhiteRequiredForCover: true,
			NoWatermarkPolicy:         true,
			MinSafeMarginPct:          0.08,
		},
		Recommendations: model.ToraRecommendedPreset{
			RecommendedWidth:        1000,
			RecommendedHeight:       1000,
			RecommendedAspectRatio:  "1:1",
			RecommendedFormat:       "jpg",
			RecommendedQuality:      92,
			RecommendedBgPreset:     "PURE_WHITE",
			RecommendedShadowPreset: "MARKETPLACE",
			RecommendedPaddingPct:   0.10,
		},
		PolicyNotes: []string{
			"Cover image must feature clear product on solid background (pure white #FFFFFF required for Shopee Mall).",
			"Promotional text, border frames, and seller watermarks on cover image can cause search ranking suppression.",
			"Minimum resolution 500x500px, 1000x1000px strongly recommended for desktop zoom inspection.",
		},
	},
	{
		Platform:      "lazada",
		Region:        "TH",
		ChannelName:   "Lazada Thailand (LazMall & Marketplace)",
		Version:       "2026.1",
		LastAuditedAt: "2026-10-08",
		Requirements: model.PlatformRequirement{
			MinWidth:                  330,
			MaxWidth:                  5000,
			MinHeight:                 330,
			MaxHeight:                 5000,
			AllowedAspectRatios:       []string{"1:1"},
			AllowedFormats:            []string{"jpg", "jpeg", "png", "webp"},
			MaxFileSizeBytes:          3 * 1024 * 1024, // 3MB
			PureWhiteRequiredForCover: true,
			NoWatermarkPolicy:         true,
			MinSafeMarginPct:          0.08,
		},
		Recommendations: model.ToraRecommendedPreset{
			RecommendedWidth:        1000,
			RecommendedHeight:       1000,
			RecommendedAspectRatio:  "1:1",
			RecommendedFormat:       "jpg",
			RecommendedQuality:      92,
			RecommendedBgPreset:     "PURE_WHITE",
			RecommendedShadowPreset: "MARKETPLACE",
			RecommendedPaddingPct:   0.10,
		},
		PolicyNotes: []string{
			"Main image background must be pure white (RGB 255, 255, 255).",
			"Product must occupy at least 80% of canvas area with balanced margins.",
			"Watermarks, logos not belonging to brand owner, and contact information are prohibited.",
		},
	},
	{
		Platform:      "tiktok_shop",
		Region:        "TH",
		ChannelName:   "TikTok Shop Thailand",
		Version:       "2026.1",
		LastAuditedAt: "2026-10-08",
		Requirements: model.PlatformRequirement{
			MinWidth:                  600,
			MaxWidth:                  2000,
			MinHeight:                 600,
			MaxHeight:                 2000,
			AllowedAspectRatios:       []string{"1:1"},
			AllowedFormats:            []string{"jpg", "jpeg", "png"},
			MaxFileSizeBytes:          5 * 1024 * 1024, // 5MB
			PureWhiteRequiredForCover: false,
			NoWatermarkPolicy:         true,
			MinSafeMarginPct:          0.10,
		},
		Recommendations: model.ToraRecommendedPreset{
			RecommendedWidth:        1200,
			RecommendedHeight:       1200,
			RecommendedAspectRatio:  "1:1",
			RecommendedFormat:       "jpg",
			RecommendedQuality:      94,
			RecommendedBgPreset:     "PURE_WHITE",
			RecommendedShadowPreset: "SOFT_STUDIO",
			RecommendedPaddingPct:   0.10,
		},
		PolicyNotes: []string{
			"1:1 aspect ratio mandatory for main catalog thumbnail.",
			"Official recommendation is 1200x1200px for optimal mobile feed rendering.",
			"No misleading badges (e.g. fake 'Best Seller' or external contact QR codes).",
		},
	},
	{
		Platform:      "instagram",
		Region:        "GLOBAL",
		ChannelName:   "Instagram (Feed & Story)",
		Version:       "2026.1",
		LastAuditedAt: "2026-10-08",
		Requirements: model.PlatformRequirement{
			MinWidth:                  320,
			MaxWidth:                  1920,
			MinHeight:                 320,
			MaxHeight:                 1920,
			AllowedAspectRatios:       []string{"1:1", "4:5", "9:16"},
			AllowedFormats:            []string{"jpg", "jpeg", "png"},
			MaxFileSizeBytes:          8 * 1024 * 1024, // 8MB
			PureWhiteRequiredForCover: false,
			NoWatermarkPolicy:         false,
			MinSafeMarginPct:          0.05,
		},
		Recommendations: model.ToraRecommendedPreset{
			RecommendedWidth:        1080,
			RecommendedHeight:       1080,
			RecommendedAspectRatio:  "1:1",
			RecommendedFormat:       "jpg",
			RecommendedQuality:      95,
			RecommendedBgPreset:     "WARM_WHITE",
			RecommendedShadowPreset: "SOFT_STUDIO",
			RecommendedPaddingPct:   0.12,
		},
		PolicyNotes: []string{
			"Square feed (1080x1080), Portrait feed (1080x1350, 4:5), Story/Reel (1080x1920, 9:16).",
			"High-contrast presentation with realistic soft studio drop shadows delivers superior engagement.",
		},
	},
}

// GetMarketplaceRules returns all registered compliance rules.
func GetMarketplaceRules() []model.MarketplaceRule {
	return CanonicalMarketplaceRules
}

// GetMarketplaceRule finds a specific rule by platform and optional region.
func GetMarketplaceRule(platform, region string) (*model.MarketplaceRule, error) {
	normPlatform := strings.ToLower(strings.TrimSpace(platform))
	normRegion := strings.ToUpper(strings.TrimSpace(region))

	for _, rule := range CanonicalMarketplaceRules {
		if strings.ToLower(rule.Platform) == normPlatform {
			if normRegion == "" || strings.ToUpper(rule.Region) == normRegion || rule.Region == "GLOBAL" {
				return &rule, nil
			}
		}
	}
	return nil, ErrMarketplaceNotFound
}

// ValidateCompliance tests whether an image asset complies with a platform's specifications.
func ValidateCompliance(input model.ComplianceValidationInput, platform, region string) model.ComplianceResult {
	rule, err := GetMarketplaceRule(platform, region)
	if err != nil {
		return model.ComplianceResult{
			Compliant:  false,
			Platform:   platform,
			Region:     region,
			Violations: []string{fmt.Sprintf("Unknown marketplace platform '%s' (region: '%s')", platform, region)},
		}
	}

	result := model.ComplianceResult{
		Compliant:       true,
		Platform:        rule.Platform,
		Region:          rule.Region,
		Violations:      make([]string, 0),
		Warnings:        make([]string, 0),
		Recommendations: make([]string, 0),
	}

	req := rule.Requirements
	rec := rule.Recommendations

	// 1. Dimension constraints
	if input.Width < req.MinWidth {
		result.Violations = append(result.Violations, fmt.Sprintf("Width %dpx is below platform minimum of %dpx", input.Width, req.MinWidth))
	}
	if input.Width > req.MaxWidth {
		result.Violations = append(result.Violations, fmt.Sprintf("Width %dpx exceeds platform maximum of %dpx", input.Width, req.MaxWidth))
	}
	if input.Height < req.MinHeight {
		result.Violations = append(result.Violations, fmt.Sprintf("Height %dpx is below platform minimum of %dpx", input.Height, req.MinHeight))
	}
	if input.Height > req.MaxHeight {
		result.Violations = append(result.Violations, fmt.Sprintf("Height %dpx exceeds platform maximum of %dpx", input.Height, req.MaxHeight))
	}

	// 2. Aspect Ratio constraints
	actualRatio := float64(input.Width) / float64(input.Height)
	ratioMatched := false
	for _, expectedRatioStr := range req.AllowedAspectRatios {
		var expectedRatio float64
		switch expectedRatioStr {
		case "1:1":
			expectedRatio = 1.0
		case "4:5":
			expectedRatio = 0.8
		case "9:16":
			expectedRatio = 9.0 / 16.0
		default:
			expectedRatio = 1.0
		}
		if math.Abs(actualRatio-expectedRatio) < 0.02 {
			ratioMatched = true
			break
		}
	}
	if !ratioMatched {
		result.Violations = append(result.Violations, fmt.Sprintf("Aspect ratio %.2f does not match allowed ratios %v", actualRatio, req.AllowedAspectRatios))
	}

	// 3. Format constraints
	normFormat := strings.ToLower(strings.TrimPrefix(input.Format, "."))
	formatAllowed := false
	for _, f := range req.AllowedFormats {
		if normFormat == f {
			formatAllowed = true
			break
		}
	}
	if !formatAllowed {
		result.Violations = append(result.Violations, fmt.Sprintf("Format '%s' is not allowed. Supported formats: %v", input.Format, req.AllowedFormats))
	}

	// 4. File Size constraints (if provided)
	if input.FileSizeBytes > 0 && input.FileSizeBytes > req.MaxFileSizeBytes {
		result.Violations = append(result.Violations, fmt.Sprintf("File size %.2f MB exceeds platform maximum of %.2f MB",
			float64(input.FileSizeBytes)/(1024*1024), float64(req.MaxFileSizeBytes)/(1024*1024)))
	}

	// 5. Pure white background requirement
	if req.PureWhiteRequiredForCover && !input.IsPureWhiteBg {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s recommends pure white (#FFFFFF) background for primary cover images", rule.ChannelName))
	}

	// 6. Watermark policy
	if req.NoWatermarkPolicy && input.ContainsWatermark {
		result.Warnings = append(result.Warnings, "Platform prohibits seller promotional watermarks, badges, or contact QR codes on cover images")
	}

	// 7. Recommendations vs Requirements distinction
	if input.Width < rec.RecommendedWidth || input.Height < rec.RecommendedHeight {
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Recommended resolution is %dx%d (current: %dx%d) for crisp rendering on high-DPI displays",
			rec.RecommendedWidth, rec.RecommendedHeight, input.Width, input.Height))
	}

	if len(result.Violations) > 0 {
		result.Compliant = false
	}

	return result
}
