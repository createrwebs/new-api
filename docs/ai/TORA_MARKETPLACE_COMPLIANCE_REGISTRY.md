# Tora Studio — Marketplace Compliance Registry
**File:** `docs/ai/TORA_MARKETPLACE_COMPLIANCE_REGISTRY.md`  
**Version:** `2026.1`  
**Audit Date:** 2026-10-08  
**Scope:** Southeast Asia (TH, SEA) & Global E-Commerce Export Standards  

---

## 1. Architectural Philosophy

Tora Studio separates marketplace e-commerce formatting into two strictly delineated tiers:
1. **Platform Requirements (`PLATFORM_REQUIREMENTS`)**: Hard constraints mandated by marketplace channels (Shopee, Lazada, TikTok Shop, Instagram). Violations of these constraints cause listing rejections, image upload failures, or policy strikes.
2. **Tora Recommended Presets (`TORA_RECOMMENDED_PRESETS`)**: Visual optimization guidelines designed by Tora Studio to maximize product conversion, click-through rates, and high-DPI display fidelity without violating platform constraints.

> [!IMPORTANT]
> **Zero Credential / Direct Publishing Invariant**:  
> In Queue N5, Tora Studio does NOT store merchant marketplace API keys, seller login passwords, or execute automated posting to seller centers. All compliance enforcement is executed deterministically on exported assets prior to packaging into `.zip` or asset downloads.

---

## 2. Authoritative Channel Specifications

### A. Shopee Thailand (`shopee` / `TH`)
- **Channel**: Shopee Thailand Standard & Shopee Mall
- **Platform Constraints**:
  - Minimum Resolution: `500x500` px
  - Maximum Resolution: `2000x2000` px
  - Mandatory Aspect Ratio: `1:1` (Square)
  - Allowed Formats: `JPG`, `JPEG`, `PNG`
  - Max File Size: `2.0 MB`
  - Cover Image Policy: Pure white (`#FFFFFF`) background mandatory for Shopee Mall; no watermarks, phone numbers, or border overlays on cover.
- **Tora Recommended Preset**:
  - Resolution: `1000x1000` px
  - Format: `JPG` (Quality 92)
  - Padding: `10%` balanced safe margin
  - Background: `PURE_WHITE` (`#FFFFFF`)
  - Drop Shadow: `MARKETPLACE` (grounded contact shadow)

### B. Lazada Thailand (`lazada` / `TH`)
- **Channel**: Lazada Thailand & LazMall
- **Platform Constraints**:
  - Minimum Resolution: `330x330` px
  - Maximum Resolution: `5000x5000` px
  - Mandatory Aspect Ratio: `1:1` (Square)
  - Allowed Formats: `JPG`, `JPEG`, `PNG`, `WEBP`
  - Max File Size: `3.0 MB`
  - Cover Image Policy: Pure white background required for primary catalog listing; product must occupy $\ge 80\%$ of canvas area.
- **Tora Recommended Preset**:
  - Resolution: `1000x1000` px
  - Format: `JPG` (Quality 92)
  - Padding: `10%` balanced margin
  - Background: `PURE_WHITE` (`#FFFFFF`)
  - Drop Shadow: `MARKETPLACE` (grounded contact shadow)

### C. TikTok Shop Thailand (`tiktok_shop` / `TH`)
- **Channel**: TikTok Shop E-Commerce
- **Platform Constraints**:
  - Minimum Resolution: `600x600` px
  - Maximum Resolution: `2000x2000` px
  - Mandatory Aspect Ratio: `1:1` (Square)
  - Allowed Formats: `JPG`, `JPEG`, `PNG`
  - Max File Size: `5.0 MB`
  - Cover Image Policy: Prohibits misleading badges (fake "No. 1" tags), external contact information, or QR codes.
- **Tora Recommended Preset**:
  - Resolution: `1200x1200` px (Higher resolution optimized for retina smartphone feed viewing)
  - Format: `JPG` (Quality 94)
  - Padding: `10%` margin
  - Background: `PURE_WHITE` (`#FFFFFF`)
  - Drop Shadow: `SOFT_STUDIO` (subtle ambient shadow)

### D. Instagram Global (`instagram` / `GLOBAL`)
- **Channel**: Instagram Feed, Carousel & Stories
- **Platform Constraints**:
  - Minimum Resolution: `320x320` px
  - Maximum Resolution: `1920x1920` px
  - Allowed Aspect Ratios: `1:1` (Square), `4:5` (Portrait Feed), `9:16` (Story/Reel)
  - Allowed Formats: `JPG`, `JPEG`, `PNG`
  - Max File Size: `8.0 MB`
- **Tora Recommended Presets**:
  - Feed: `1080x1080` px (1:1), `JPG` Quality 95, `WARM_WHITE` (`#F7F7F7`), `SOFT_STUDIO` shadow
  - Story: `1080x1920` px (9:16), `JPG` Quality 95, balanced vertical centering

---

## 3. Endpoints & Developer Integration

### Inspection Endpoint
`GET /api/studio/marketplace/rules`
Returns the complete canonical registry of platform rules, policy notes, and recommended presets.

### Pre-Validation Endpoint
`POST /api/studio/marketplace/validate`
```json
{
  "platform": "shopee",
  "region": "TH",
  "asset": {
    "width": 1000,
    "height": 1000,
    "format": "jpg",
    "file_size_bytes": 750000,
    "is_pure_white_bg": true,
    "contains_watermark": false
  }
}
```

Response:
```json
{
  "success": true,
  "data": {
    "compliant": true,
    "platform": "shopee",
    "region": "TH",
    "violations": [],
    "warnings": [],
    "recommendations": []
  }
}
```
