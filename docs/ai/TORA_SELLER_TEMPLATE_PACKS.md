# Tora Studio: Seller Template Packs Specification
**Queue:** `QUEUE N4 PREFLIGHT`  
**Focus:** Config-Driven Multi-Marketplace Presets (Southeast Asia & Global)  
**Date:** 2026-10-08  

---

## 1. Overview

Tora Product Factory standardizes multi-platform e-commerce image exports through structured, data-driven templates. Instead of requiring manual cropping and canvas adjustments, a single product photograph is automatically formatted into all certified marketplace specifications in a single pass.

---

## 2. Certified Marketplace Template Schema

```json
{
  "template_version": "1.0.0",
  "templates": [
    {
      "platform_id": "shopee_standard",
      "platform_name": "Shopee Marketplace",
      "canvas_width": 800,
      "canvas_height": 800,
      "aspect_ratio": "1:1",
      "safe_margin_percent": 15,
      "subject_target_coverage": 0.75,
      "background": "pure_white",
      "shadow": "marketplace",
      "export_format": "jpeg",
      "quality": 92
    },
    {
      "platform_id": "lazada_standard",
      "platform_name": "Lazada Official Store",
      "canvas_width": 1000,
      "canvas_height": 1000,
      "aspect_ratio": "1:1",
      "safe_margin_percent": 12,
      "subject_target_coverage": 0.80,
      "background": "pure_white",
      "shadow": "marketplace",
      "export_format": "jpeg",
      "quality": 95
    },
    {
      "platform_id": "tiktok_shop",
      "platform_name": "TikTok Shop / Feed",
      "canvas_width": 1080,
      "canvas_height": 1080,
      "aspect_ratio": "1:1",
      "safe_margin_percent": 10,
      "subject_target_coverage": 0.82,
      "background": "warm_white",
      "shadow": "soft_studio",
      "export_format": "jpeg",
      "quality": 92
    },
    {
      "platform_id": "instagram_feed",
      "platform_name": "Instagram Feed Portrait",
      "canvas_width": 1080,
      "canvas_height": 1350,
      "aspect_ratio": "4:5",
      "safe_margin_percent": 12,
      "subject_target_coverage": 0.78,
      "background": "studio_vignette",
      "shadow": "soft_studio",
      "export_format": "jpeg",
      "quality": 95
    },
    {
      "platform_id": "story_reels_tiktok",
      "platform_name": "TikTok / IG Story Fullscreen",
      "canvas_width": 1080,
      "canvas_height": 1920,
      "aspect_ratio": "9:16",
      "safe_margin_percent": 25,
      "subject_target_coverage": 0.65,
      "background": "warm_white",
      "shadow": "floating",
      "export_format": "jpeg",
      "quality": 90
    },
    {
      "platform_id": "master_cutout",
      "platform_name": "Master Transparent PNG",
      "canvas_width": 0,
      "canvas_height": 0,
      "aspect_ratio": "original",
      "safe_margin_percent": 0,
      "subject_target_coverage": 1.0,
      "background": "transparent",
      "shadow": "no_shadow",
      "export_format": "png",
      "quality": 100
    }
  ]
}
```

---

## 3. Template Execution Pipeline
1. **Centering & Fit**: The segmented subject is positioned at $(W/2, H/2)$ with aspect ratio preserved.
2. **Safe Margin Bounding**: Subject dimensions are scaled so bounding box does not exceed $(1 - 2 \times \text{safe\_margin}) \times \min(W, H)$.
3. **Shadow Layering**: The selected shadow preset is positioned under the base contact points.
4. **All-in-One Packaging**: All rendered outputs are archived into `product_pack_<timestamp>.zip` alongside `manifest.json`.
