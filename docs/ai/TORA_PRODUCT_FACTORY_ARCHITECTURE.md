# Tora Product Factory Architecture & Multi-Platform E-Commerce Packaging

**Identifier:** `TORA-ARCH-N3-PRODUCT-FACTORY`  
**Date:** 2026-10-08  
**Scope:** Hybrid On-Device to Cloud e-commerce automation pipeline for merchants and marketplace sellers.  

---

## 1. Product Factory Workflow & Value Proposition

Online merchants (selling on Shopee, Lazada, TikTok Shop, Instagram, and Amazon) face a tedious manual bottleneck:
1. Product photos have cluttered backgrounds (store aisles, desks, factory floors).
2. Each marketplace enforces distinct image dimension, background, and aspect ratio requirements.
3. Merchants currently spend 15–30 minutes per photo in Photoshop or pay expensive per-image software fees.

**Tora Product Factory** automates this entire pipeline into a single 1-click batch action:
- **Input**: 1 to 10 raw product photos taken directly with the smartphone camera.
- **Stage 1 (On-Device Neural Cutout)**: `u2netp` strips the cluttered background locally in `< 500ms`.
- **Stage 2 (Optional On-Device 2X Upscale)**: `realesrgan_2x` enhances detail and resolution.
- **Stage 3 (Cloud Deterministic Packaging)**: Cloud container generates pixel-perfect marketplace variants and bundles an All-In-One ZIP package with JSON inventory manifest.

---

## 2. Multi-Platform Marketplace Specification

The cloud packaging step (`POST /api/studio/native/product-pack`) produces 5 standardized e-commerce assets for every product:

| Asset Name | Target Platform | Canvas Dimensions | Aspect Ratio | Background Standard | Padding / Fit |
|---|---|---|---|---|---|
| `01_shopee_800x800.jpg` | **Shopee** | 800 x 800 px | 1:1 Square | Pure White (`#FFFFFF`) | 85% safe-box padding, e-commerce compliant |
| `02_lazada_1000x1000.jpg` | **Lazada** | 1000 x 1000 px | 1:1 Square | Pure White with subtle grounding shadow | 85% safe-box padding |
| `03_instagram_1080x1350.jpg` | **Instagram Feed** | 1080 x 1350 px | 4:5 Portrait | Modern soft neutral studio gradient | Centered product display |
| `04_story_tiktok_1080x1920.jpg` | **TikTok / IG Story** | 1080 x 1920 px | 9:16 Vertical | E-commerce story layout with top/bottom copy zone | Safe margin for UI overlays |
| `05_product_cutout_transparent.png` | **Master Asset** | Original Dimensions | Native | 100% Alpha Transparent | Preserves original cutout for custom branding |

### 2.1 Bundled All-In-One Package (`product_pack_<id>.zip`)
Includes:
- All 5 generated imagery assets.
- `manifest.json`: Machine-readable metadata recording product ID, variant URLs, SHA-256 digests, dimensions, and platform compliance badges.

---

## 3. Hybrid On-Device + Cloud Pipeline Diagram

```
+--------------------------------------------------------------------------------+
|                             USER'S MOBILE DEVICE                               |
|                                                                                |
|  [Raw Camera Photo] (Uncompressed 4000x3000)                                   |
|         |                                                                      |
|         v                                                                      |
|  +--------------------------------------------------+                          |
|  | STEP 1: On-Device Cutout (U2Net-P CoreML/NNAPI)  | ---> 2 Tora Credits      |
|  +--------------------------------------------------+                          |
|         |                                                                      |
|         v [Transparent Cutout PNG]                                             |
|         |                                                                      |
|  + - - -|- - - - - - - - - - - - - - - - - - - - - -+                          |
|  | OPTIONAL STEP 2: On-Device 2X Upscale (ESRGAN)   | ---> +3 Tora Credits     |
|  + - - -|- - - - - - - - - - - - - - - - - - - - - -+                          |
|         |                                                                      |
|         v [High-Res Cutout PNG]                                                |
+---------|----------------------------------------------------------------------+
          | (Only cutout uploaded; raw background NEVER leaves phone)
          v
+--------------------------------------------------------------------------------+
|                       TORA DETERMINISTIC SERVER (CLOUD)                        |
|                                                                                |
|  +--------------------------------------------------+                          |
|  | STEP 3: Multi-Platform Generator & ZIP Bundler   | ---> 5 Tora Credits      |
|  | (Shopee + Lazada + Instagram + TikTok + Manifest)|      (SUCCESS_SETTLEMENT)|
|  +--------------------------------------------------+                          |
|         |                                                                      |
|         v                                                                      |
|  [product_pack.zip + Pre-Signed CDN URLs]                                      |
+---------|----------------------------------------------------------------------+
          v
  Mobile User downloads individual presets or 1-Click ZIP Package
```

---

## 4. Architectural Verification

- **Batch Capability**: Tested with 1 to 10 photos concurrently.
- **Failure Isolation**: If item #3 fails, items #1 and #2 remain completed and accessible.
- **Zero GPU Overhead**: Step 3 executes via deterministic Go standard library image composition, incurring zero GPU cost for Tora infrastructure.
