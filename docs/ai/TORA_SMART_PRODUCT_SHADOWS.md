# Tora Studio: Smart Deterministic Product Shadows & Backgrounds
**Queue:** `QUEUE N4 PREFLIGHT`  
**Focus:** Zero-GPU High-Margin Studio Quality Geometry & Lighting  
**Date:** 2026-10-08  

---

## 1. Executive Summary

Generative AI diffusion is unnecessarily slow, expensive, and unpredictable for standard e-commerce product staging. **Smart Product Shadows** and **Studio Backgrounds** are engineered as **100% deterministic mathematical pipelines**, running locally in client memory or in sub-millisecond CPU time on the server.

---

## 2. Deterministic Contact Shadow Algorithm

Given an alpha-isolated foreground product cutout ($W \times H$ RGBA PNG):

```
1. BOUNDING BOX & BASE EXTRACTION:
   Scan alpha mask to extract bounding box [minX, minY, maxX, maxY].
   Identify bottom-most foreground pixels within bottom 10% vertical zone:
   Base Center = ((minX + maxX)/2, maxY)
   Base Width  = (maxX - minX) * 0.85

2. GEOMETRIC SHADOW PROJECTION:
   Construct contact ellipse centered at (Base Center.X, maxY + verticalOffset):
   Semi-major axis a = (Base Width / 2) * scaleFactor
   Semi-minor axis b = a * 0.22 (flattened perspective ratio)

3. GRADIENT FILL & GAUSSIAN CONVOLUTION:
   Apply dual-layer radial gradient:
   - Inner Core: #111111 at 45% opacity (ambient occlusion contact)
   - Outer Penumbra: #444444 fading to #000000 at 0% opacity
   Convolve with 2D Gaussian blur (sigma = 8px to 24px per preset).

4. ALPHA COMPOSITION:
   Background Color Layer 
       + Convolved Shadow Layer 
       + Original Subject Foreground Layer
   = Professional Studio Product Shot.
```

---

## 3. Product Shadow Presets

| Preset Name | Shadow Geometry | Blur Sigma | Opacity | Vertical Offset | Best Use Case |
|---|---|---|---|---|---|
| **Soft Studio** | Dual-layer wide elliptical penumbra | 16 px | 28% | +4 px | Cosmetics, jewelry, electronics |
| **Marketplace** | Tight, crisp contact shadow directly below base | 6 px | 45% | +1 px | Shopee, Lazada standard white catalog |
| **Floating** | Detached, softened diffused ellipse | 28 px | 20% | +24 px | Sneakers, drones, tech accessories |
| **No Shadow** | Pure transparent or flat composite | 0 px | 0% | 0 px | Raw cutout catalog compliance |

---

## 4. Deterministic Background Presets

| Preset ID | Visual Style | Hex / Gradient Definition | Marketplace Compliance |
|---|---|---|---|
| `pure_white` | Standard E-Commerce White | `#FFFFFF` flat fill | Shopee / Lazada / Amazon 100% compliant |
| `warm_white` | Modern Lifestyle Neutral | `#FAF9F6` with subtle center glow | Instagram / Shopify storefronts |
| `light_gray` | Tech Studio Minimal | `#F2F4F7` flat fill | Consumer electronics, hardware |
| `studio_vignette` | Soft Radial Studio Gradient | Radial: Center `#FFFFFF` $\to$ Edge `#E5E7EB` | Premium hero presentation |
| `brand_color` | Custom User Hex Fill | User-defined `#RRGGBB` hex code | Brand matching, social banners |
