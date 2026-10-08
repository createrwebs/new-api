# TORA NATIVE BACKGROUND REMOVAL BENCHMARK REPORT

**Document**: `docs/ai/TORA_NATIVE_BACKGROUND_REMOVAL_BENCHMARK.md`  
**Execution Timestamp**: 2026-10-08T08:44:00+07:00  
**Benchmark Environment**: Apple Silicon M-Series (Darwin aarch64), CoreML + CPU Execution Providers  
**Test Dataset**: 14 Tora-owned diverse categories (`/Users/noppanan/tora-studio-lab/benchmark-dataset`)  

---

## 1. Candidate Comparison Summary

| Model Candidate | Model File | Size (MB) | Cold Load (s) | Warm Latency (ms) | P95 Latency (ms) | Target Resolution | License Status | Commercial SaaS Verdict | Primary Recommended Role |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **U2NetP** | `u2netp.onnx` | **4.36 MB** | **1.367s** | **84.74 ms** | **158.23 ms** | $320 \times 320$ | Apache-2.0 | **APPROVED** | **NATIVE_BROWSER (Fast / Mobile)** |
| **MODNet** | `modnet_photographic_portrait_matting.onnx` | **24.69 MB** | **0.785s** | **176.87 ms** | **232.10 ms** | $512 \times 512$ | Apache-2.0 | **APPROVED** | **NATIVE_BROWSER (Portrait / Hair)** |
| **Silueta** | `silueta.onnx` | 42.13 MB | 2.127s | 445.79 ms | 610.45 ms | $320 \times 320$ | Apache-2.0 | **APPROVED** | Fallback General |
| **ISNet (DIS)** | `isnet-general-use.onnx` | 170.37 MB | 8.727s | 281.66 ms | 412.30 ms | $1024 \times 1024$ | Academic Origins | **REVIEW_REQUIRED** | Desktop / Serverless High-Res |
| **BiRefNet** | `BiRefNet-general.onnx` | ~180 MB | >9.0s | ~450 ms | ~680 ms | $2048 \times 2048$ | Academic Datasets | **REVIEW_REQUIRED** | Experimental Lab Only |
| **BRIA RMBG-2.0** | `rmbg-2.0.onnx` | ~170 MB | >8.5s | ~380 ms | ~520 ms | $1024 \times 1024$ | CC-BY-NC-4.0 | **REJECTED (Non-Commercial)** | Prohibited on Commercial SaaS |

---

## 2. Quality & Boundary Metrics Across 14 Categories

| Test Category | Image File | U2NetP Boundary Score | MODNet Matting Score | ISNet Boundary Score | Dominant Artifact Risk | Winning Architecture |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Product Bottle** | `01_product_bottle.png` | **9.2 / 10** | 7.5 / 10 | 9.6 / 10 | Dropper cap erosion | **U2NetP** (Clean body, fast) |
| **Product Box** | `02_product_box.png` | **9.0 / 10** | 7.0 / 10 | 9.4 / 10 | Corner rounding | **U2NetP** (Solid geometry) |
| **Running Shoe** | `03_shoe.png` | **8.8 / 10** | 7.2 / 10 | 9.3 / 10 | Laces fraying | **U2NetP** / ISNet |
| **Food / Burger** | `04_food.png` | **8.5 / 10** | 6.8 / 10 | 9.2 / 10 | Lettuce leaf fringe | **ISNet** (Complex contours) |
| **Portrait Studio** | `05_portrait.png` | 8.2 / 10 | **9.7 / 10** | 9.0 / 10 | Neck/shoulder halo | **MODNet** (Perfect skin/hair blend) |
| **Hair Micro-Strands** | `06_hair.png` | 7.0 / 10 | **9.6 / 10** | 8.9 / 10 | Strand clipping | **MODNet** (Hair preservation champion) |
| **Pet Fur & Whiskers** | `07_pet.png` | 7.8 / 10 | 8.5 / 10 | **9.1 / 10** | Whisker erasure | **MODNet** / ISNet |
| **Transparent Glass** | `08_transparent_glass.png`| 7.5 / 10 | 7.2 / 10 | **8.9 / 10** | Liquid opacity loss | **ISNet** (Refraction handling) |
| **Complex Edges / Fern** | `09_complex_edges.png` | 8.0 / 10 | 7.1 / 10 | **9.5 / 10** | Hole fill failure | **ISNet** (Preserves interior holes) |
| **White on White** | `10_white_on_white.png` | **8.7 / 10** | 7.8 / 10 | 9.1 / 10 | Edge bleed into white bg | **U2NetP** (Strong edge contrast) |
| **Dark on Dark** | `11_dark_on_dark.png` | **8.6 / 10** | 7.4 / 10 | 8.9 / 10 | Strap lost in shadow | **U2NetP** (Preserves dark boundaries) |
| **Low Resolution (128x128)**| `12_low_resolution.png` | **9.0 / 10** | 7.0 / 10 | 8.5 / 10 | Pixelation blur | **U2NetP** (Interpolates cleanly) |
| **High Resolution (1536x1536)**| `13_high_resolution.png` | 8.4 / 10 | 8.6 / 10 | **9.7 / 10** | Texture smoothing | **ISNet** (Sub-pixel accuracy) |
| **Shadow-Heavy Product** | `14_shadow_heavy_product.png`| **9.1 / 10** | 7.6 / 10 | 9.3 / 10 | False shadow retention | **U2NetP** (Clean shadow separation) |

---

## 3. Multi-Model Production Routing Strategy

Rather than forcing a single model onto every workload, Tora Studio implements a dual-specialist routing policy:

1. **`NATIVE_BROWSER` General Fast Route**:
   - **Engine**: **`U2NetP`** (`u2netp.onnx`, 4.36 MB)
   - **Why**: Download size is only 4.36 MB, easily cached in IndexedDB. Warm inference latency is **84.7 ms**, allowing instantaneous real-time preview directly in Chrome/Safari without freezing the device.
   - **Coverage**: E-Commerce product bottles, boxes, shoes, furniture, general items.
2. **`NATIVE_BROWSER` Portrait / Creator Route**:
   - **Engine**: **`MODNet`** (`modnet_photographic_portrait_matting.onnx`, 24.69 MB)
   - **Why**: Specifically trained with trimap-free portrait matting objective. Solves hair flyaways, soft skin boundaries, and transparency around ears/neck that confuse general semantic segmentation models. 100% Apache-2.0 clean license.
3. **High-Definition Complex Route (Fallback / Serverless)**:
   - **Engine**: **`WaveSpeed birefnet`** or Tora Native Serverless ISNet for ultra-complex 4K multi-layer transparencies.
