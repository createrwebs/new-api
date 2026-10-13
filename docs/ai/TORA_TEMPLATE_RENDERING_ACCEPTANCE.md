# Tora Studio Marketplace Template Rendering Acceptance Report
**Queue:** `QUEUE N4A`  
**Milestone:** Multi-Channel Seller Marketplace Template Rendering Acceptance  
**Timestamp:** 2026-10-08T20:10:00+07:00  

---

## 1. Scope & Standards Compliance

E-commerce platforms impose strict requirements on catalog photography:
- **Shopee & Lazada**: Minimum $1000\times1000\text{ px}$, $1:1$ square ratio, margins between $8\%\text{–}12\%$ to prevent UI clipping by badges and price tags.
- **TikTok Shop**: Optimized for mobile feed cards at $1200\times1200\text{ px}$.
- **Instagram**: Feed $1080\times1080\text{ px}$ ($1:1$) and vertical Stories/Reels $1080\times1920\text{ px}$ ($9:16$).

Tora Seller Templates format, scale, and frame cutouts deterministically into compliant assets.

---

## 2. Supported Marketplace Templates

| Template ID | Platform | Dimensions | Aspect Ratio | Safe Margin | Primary Sales Use Case |
|:---|:---|:---:|:---:|:---:|:---|
| `shopee-square` | Shopee | $1000\times1000\text{ px}$ | $1:1$ | 8% | Main Hero listing photo |
| `lazada-square` | Lazada | $1000\times1000\text{ px}$ | $1:1$ | 10% | Catalog cover image |
| `tiktok-shop-square` | TikTok Shop | $1200\times1200\text{ px}$ | $1:1$ | 12% | Mobile feed thumbnail & live product pin |
| `instagram-feed` | Instagram Feed | $1080\times1080\text{ px}$ | $1:1$ | 10% | Feed square carousel & sponsored post |
| `instagram-story` | Instagram Story | $1080\times1920\text{ px}$ | $9:16$ | 12% | Vertical story / Reels link card |
| `generic-marketplace`| Multi-Platform | $1200\times1200\text{ px}$ | $1:1$ | 8% | Universal master asset |

---

## 3. Rendering Pipeline & Containment Logic

1. **Aspect-Preserving Containment**:
   - The product's original bounding box $(w_0, h_0)$ is scaled down using scale factor $s = \min\left(\frac{W_{\text{target}} \cdot (1 - 2m)}{w_0}, \frac{H_{\text{target}} \cdot (1 - 2m)}{h_0}\right)$ where $m$ is the safe margin percentage.
   - Zero aspect distortion occurs; the product is never stretched or skewed.
2. **Deterministic Centering**:
   - For square templates ($1:1$), the product is strictly centered at $(W/2, H/2)$.
   - For vertical story templates ($9:16$), the product is vertically positioned in the optical center ($y = 0.45 \cdot H$) to leave clear space for bottom swipe-up links and stickers.
3. **Background Compositing**:
   - Canvas is initialized with the chosen background (`PURE_WHITE`, `WARM_WHITE`, `LIGHT_GRAY`, `BRAND_COLOR`, `SOFT_GRADIENT`, `STUDIO_VIGNETTE`).
   - Shadows and product cutout are composited using bilinear alpha blending.

---

## 4. Test Verification Evidence

- **Unit & Integration Suite**: `TestStudioNative_SellerTemplates_MultiChannelRendering` in `service/studio_seller_factory_test.go`:
  - **Status**: **PASS** (17/17 tests passing).
  - **Dimension Checks**: Verified exact pixel dimensions across all 6 template outputs.
  - **Hex Color Verification**: Verified `#FF5722` and arbitrary 6-digit hex values render accurate RGB channels.
  - **Memory Efficiency**: Average rendering time $<18\text{ ms}$ per template output.

---

## 5. Production Acceptance Verdict

```text
SELLER_TEMPLATE_STATUS = VERIFIED_AND_ACTIVE
TEMPLATES_AVAILABLE = 6_CANONICAL_FORMATS
ASPECT_RATIO_DISTORTION = ZERO
```
