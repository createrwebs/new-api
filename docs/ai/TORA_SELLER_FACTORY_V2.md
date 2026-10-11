# Tora Studio Seller Factory V2 Specification & Architecture
**Queue:** `QUEUE N4A`  
**Milestone:** Seller Factory V2 — Complete E-Commerce Multi-Marketplace Pipeline  
**Timestamp:** 2026-10-08T20:10:00+07:00  

---

## 1. Value Proposition & Moat

Generic AI image generators hallucinate random product details, alter brand typography, and require complex prompt engineering. 
**Tora Seller Factory V2** delivers a deterministic, high-efficiency pipeline designed specifically for professional sellers across Shopee, Lazada, TikTok Shop, and Instagram:
- **Zero Hallucination**: Original product appearance and branding are 100% preserved.
- **On-Device Matting & Privacy**: Raw camera photos remain on the user's device (`RAW_SOURCE_UPLOAD_COUNT = 0`). Only the transparent cutout PNG is handed off.
- **Smart Shadows**: Physically grounded contact and drop shadows make floating products look naturally grounded on studio surfaces.
- **Multi-Marketplace Export**: One click generates marketplace-compliant hero images formatted and cropped for every major sales channel.
- **Batch Speed**: Processes up to 10 products per batch in resilient 3-item memory chunks, outputting a consolidated ZIP archive.

---

## 2. Pipeline Architecture

```text
[ Raw User Photo ]
        │
        ▼ (On-Device Mobile / Web Native Engine)
[ Background Removal (u2netp / BiRefNet) ]
        │
        ▼ (Optional Local Retouching)
[ Object Cleanup (Blemish & Cable Erase via Telea) ]
        │
        ▼ (Transparent Cutout PNG Only)
[ Handoff to Seller Factory V2 ]
        │
        ├──► 1. Smart Product Shadow Synthesis (SOFT_STUDIO, MARKETPLACE, FLOATING, GROUND_CONTACT)
        ├──► 2. Studio Background Injection (PURE_WHITE, WARM_WHITE, BRAND_COLOR, GRADIENT)
        └──► 3. Multi-Channel Template Framing (Shopee, Lazada, TikTok Shop, Instagram)
        │
        ▼
[ Consolidated Output ZIP & Individual Assets ]
```

---

## 3. Configuration & Options

### 3.1 Smart Shadows
- `SOFT_STUDIO`: Dual-layer Gaussian-blurred drop shadow simulating soft diffused ceiling lighting (offset Y: +18px, blur: 24px, opacity: 0.28).
- `MARKETPLACE`: Crisp, clean contact shadow adhering strictly to marketplace catalog guidelines (offset Y: +8px, blur: 12px, opacity: 0.35).
- `GROUND_CONTACT`: Ultra-tight contact occlusion shadow anchored directly underneath the product base (offset Y: +3px, blur: 4px, opacity: 0.60).
- `FLOATING`: Wide, dramatic dispersed shadow giving an elevated 3D appearance (offset Y: +32px, blur: 40px, opacity: 0.22).
- `NO_SHADOW`: Clean cut without shadow synthesis.

### 3.2 Studio Backgrounds
- `PURE_WHITE`: Strict #FFFFFF required by Google Shopping, Amazon, and Shopee Mall.
- `WARM_WHITE`: Warm luxury tone (#FBF9F5) ideal for cosmetics, apparel, and organic goods.
- `LIGHT_GRAY`: Neutral high-contrast gray (#F2F4F7) ideal for tech, gadgets, and electronics.
- `BRAND_COLOR`: Custom hexadecimal color hex code (e.g. `#1A365D`) matching seller brand guidelines.
- `SOFT_GRADIENT`: Subtle studio lighting gradient (from #F8F9FA to #E9ECEF).
- `STUDIO_VIGNETTE`: Professional spotlight vignette centering luminance on the product.
- `TRANSPARENT`: Alpha channel preserved for downstream compositing.

### 3.3 Marketplace Templates
1. **Shopee Hero**: $1000\times1000\text{ px}$ (1:1), 8% margin, high-contrast white.
2. **Lazada Main**: $1000\times1000\text{ px}$ (1:1), 10% margin, neutral background.
3. **TikTok Shop Product**: $1200\times1200\text{ px}$ (1:1), 12% margin optimized for mobile feed cards.
4. **Instagram Feed**: $1080\times1080\text{ px}$ (1:1), lifestyle studio layout.
5. **Instagram Story / Reels**: $1080\times1920\text{ px}$ (9:16), vertical full-bleed card with product centered in upper two-thirds.
6. **Generic Marketplace**: $1200\times1200\text{ px}$ (1:1), universal high-resolution master asset.

---

## 4. Batch Execution & Economics

- **Maximum Batch Size**: 10 items.
- **Chunk Execution Size**: 3 items concurrently (prevents memory spikes on server).
- **Partial Failure Isolation**: An invalid image in item #2 does not abort items #1, #3–#10.
- **Pricing Schedule**:
  - **1 Item**: 7 Credits ($0.0140) Base / 10 Credits ($0.0200) Enhanced (with 2X Upscale).
  - **5 Items**: 34 Credits ($0.0680) Base / 48 Credits ($0.0960) Enhanced (5% bundle discount).
  - **10 Items**: 63 Credits ($0.1260) Base / 90 Credits ($0.1800) Enhanced (10% bundle discount).
