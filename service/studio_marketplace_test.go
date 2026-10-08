package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestMarketplaceComplianceRulesRegistry(t *testing.T) {
	rules := GetMarketplaceRules()
	if len(rules) < 4 {
		t.Fatalf("Expected at least 4 marketplace rules, got %d", len(rules))
	}

	platforms := map[string]bool{}
	for _, r := range rules {
		platforms[r.Platform] = true
	}

	expected := []string{"shopee", "lazada", "tiktok_shop", "instagram"}
	for _, exp := range expected {
		if !platforms[exp] {
			t.Errorf("Missing expected platform: %s", exp)
		}
	}
}

func TestValidateCompliance_ShopeeValid(t *testing.T) {
	input := model.ComplianceValidationInput{
		Width:            1000,
		Height:           1000,
		Format:           "jpg",
		FileSizeBytes:    800 * 1024, // 800KB
		IsPureWhiteBg:    true,
		ContainsWatermark: false,
	}

	res := ValidateCompliance(input, "shopee", "TH")
	if !res.Compliant {
		t.Fatalf("Expected compliant for valid Shopee input, got violations: %v", res.Violations)
	}
	if len(res.Violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(res.Violations))
	}
}

func TestValidateCompliance_ShopeeViolations(t *testing.T) {
	// Violation: dimension too small, wrong aspect ratio, invalid format, too large
	input := model.ComplianceValidationInput{
		Width:         400, // below 500
		Height:        600, // not 1:1
		Format:        "gif", // not allowed
		FileSizeBytes: 5 * 1024 * 1024, // exceeds 2MB
	}

	res := ValidateCompliance(input, "shopee", "TH")
	if res.Compliant {
		t.Fatalf("Expected non-compliant result")
	}
	if len(res.Violations) < 3 {
		t.Errorf("Expected at least 3 violations, got %d: %v", len(res.Violations), res.Violations)
	}
}

func TestValidateCompliance_TikTokShopRecommendedDistinction(t *testing.T) {
	// 800x800 meets minimum 600x600, but is below recommended 1200x1200
	input := model.ComplianceValidationInput{
		Width:         800,
		Height:        800,
		Format:        "jpg",
		FileSizeBytes: 1024 * 1024,
	}

	res := ValidateCompliance(input, "tiktok_shop", "TH")
	if !res.Compliant {
		t.Fatalf("Expected compliant, got violations: %v", res.Violations)
	}
	if len(res.Recommendations) == 0 {
		t.Errorf("Expected recommendation for 1200x1200px")
	}
}

func TestValidateCompliance_InstagramAspectRatios(t *testing.T) {
	// Instagram allows 1:1, 4:5, 9:16
	ratios := []struct {
		w, h int
		valid bool
	}{
		{1080, 1080, true}, // 1:1
		{1080, 1350, true}, // 4:5
		{1080, 1920, true}, // 9:16
		{1000, 500, false}, // 2:1
	}

	for _, tc := range ratios {
		input := model.ComplianceValidationInput{
			Width:  tc.w,
			Height: tc.h,
			Format: "jpg",
		}
		res := ValidateCompliance(input, "instagram", "GLOBAL")
		if res.Compliant != tc.valid {
			t.Errorf("Aspect ratio %d:%d expected compliant=%v, got %v (violations: %v)",
				tc.w, tc.h, tc.valid, res.Compliant, res.Violations)
		}
	}
}
