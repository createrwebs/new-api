package model

// PlatformRequirement defines hard platform constraints enforced by marketplace portals.
type PlatformRequirement struct {
	MinWidth                  int      `json:"min_width"`
	MaxWidth                  int      `json:"max_width"`
	MinHeight                 int      `json:"min_height"`
	MaxHeight                 int      `json:"max_height"`
	AllowedAspectRatios       []string `json:"allowed_aspect_ratios"`
	AllowedFormats            []string `json:"allowed_formats"` // "jpg", "jpeg", "png", "webp"
	MaxFileSizeBytes          int64    `json:"max_file_size_bytes"`
	PureWhiteRequiredForCover bool     `json:"pure_white_required_for_cover"`
	NoWatermarkPolicy         bool     `json:"no_watermark_policy"`
	MinSafeMarginPct          float64  `json:"min_safe_margin_pct"`
}

// ToraRecommendedPreset defines optimized export parameters recommended by Tora Studio.
type ToraRecommendedPreset struct {
	RecommendedWidth        int     `json:"recommended_width"`
	RecommendedHeight       int     `json:"recommended_height"`
	RecommendedAspectRatio  string  `json:"recommended_aspect_ratio"`
	RecommendedFormat       string  `json:"recommended_format"`
	RecommendedQuality      int     `json:"recommended_quality"`
	RecommendedBgPreset     string  `json:"recommended_bg_preset"`
	RecommendedShadowPreset string  `json:"recommended_shadow_preset"`
	RecommendedPaddingPct   float64 `json:"recommended_padding_pct"`
}

// MarketplaceRule defines authoritative channel compliance specifications.
type MarketplaceRule struct {
	Platform       string                `json:"platform"` // "shopee", "lazada", "tiktok_shop", "instagram"
	Region         string                `json:"region"`   // "TH", "SEA", "GLOBAL"
	ChannelName    string                `json:"channel_name"`
	Version        string                `json:"version"`
	LastAuditedAt  string                `json:"last_audited_at"`
	Requirements   PlatformRequirement   `json:"requirements"`
	Recommendations ToraRecommendedPreset `json:"recommendations"`
	PolicyNotes    []string              `json:"policy_notes"`
}

// ComplianceValidationInput represents asset metadata to validate.
type ComplianceValidationInput struct {
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	Format           string `json:"format"`
	FileSizeBytes    int64  `json:"file_size_bytes,omitempty"`
	IsPureWhiteBg    bool   `json:"is_pure_white_bg,omitempty"`
	ContainsWatermark bool  `json:"contains_watermark,omitempty"`
}

// ComplianceResult details validation status and diagnostics.
type ComplianceResult struct {
	Compliant       bool     `json:"compliant"`
	Platform        string   `json:"platform"`
	Region          string   `json:"region"`
	Violations      []string `json:"violations"`
	Warnings        []string `json:"warnings"`
	Recommendations []string `json:"recommendations"`
}
