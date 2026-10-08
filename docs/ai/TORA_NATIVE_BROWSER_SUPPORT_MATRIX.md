# TORA NATIVE BROWSER SUPPORT MATRIX & PERFORMANCE BREAKDOWN
**Client Environment Compatibility, Hardware Acceleration & Latency Telemetry**
**Author:** Tora Web & Platform Engineering | **Status:** UPDATED & AUDITED | **Date:** 2026-10-08

---

## 1. Important Platform Distinction: Web vs. Mobile Native

> [!IMPORTANT]
> **Flutter is NOT `NATIVE_BROWSER`**:
> The `NATIVE_BROWSER` execution class runs strictly in web environments via `onnxruntime-web` (WebGPU / WASM SIMD). It must NEVER be conflated with Tora's future native Flutter mobile roadmap (`NATIVE_MOBILE`), which will use platform channels and native operating system acceleration (CoreML on iOS, NNAPI on Android). No WebView inference should be shoved into Flutter merely to claim "mobile-native".

---

## 2. Browser Environment Support Matrix

| Browser & OS | Official ORT Web WebGPU EP | Official ORT Web WASM EP | Tora Execution Provider Order | Tora Production Verified Status | Notes / Limitations |
|---|---|---|---|---|---|
| **Chrome / Chromium (macOS / Windows)** | ✅ Officially Supported (v113+) | ✅ Officially Supported | `webgpu` → `wasm` | ✅ **TORA_PRODUCTION_VERIFIED (WebGPU)** | Primary recommended browser. Tested on Apple Silicon and Windows 11. |
| **Microsoft Edge (Windows / macOS)** | ✅ Officially Supported (v113+) | ✅ Officially Supported | `webgpu` → `wasm` | ✅ **TORA_PRODUCTION_VERIFIED (WebGPU)** | Chromium-based parity with Chrome. |
| **Chrome Android** | ⚠️ Supported (v121+ / Vulkan) | ✅ Officially Supported | `webgpu` → `wasm` | ⏳ **CONTROLLED_BENCHMARK_ONLY** | Tested in local mobile harness; pending public production device field run. |
| **Safari macOS (v17+)** | ❌ **UNSUPPORTED_BY_CURRENT_ORT_WEB_MATRIX** | ✅ Officially Supported | `wasm` (CPU SIMD) | ⚠️ **ORT_WEBGPU_UNSUPPORTED / WASM_SUPPORTED** | WebKit WebGPU is not officially supported by ORT Web. Must use WASM fallback. |
| **Safari iOS (iPhone / iPad)** | ❌ **UNSUPPORTED_BY_CURRENT_ORT_WEB_MATRIX** | ✅ Officially Supported | `wasm` (CPU SIMD) | ⏳ **NOT_TESTED_IN_PRODUCTION** | ORT Web WebGPU disabled. WASM single-thread/SIMD viable for lightweight models only. |
| **Mozilla Firefox (v120+)** | ❌ Not standard in ORT Web | ✅ Officially Supported | `wasm` (CPU SIMD) | ⏳ **FIREFOX_WASM = NOT_TESTED** | WebGPU flag-only; official ORT execution is WASM SIMD. |
| **Legacy Browsers (No WASM SIMD)** | ❌ Unavailable | ❌ Too slow | None | 🚫 **UNSUPPORTED** | Blocked at client capability preflight; no credits charged. |

---

## 3. End-to-End Latency Breakdown (Controlled Local Benchmark)
*Note: The following metrics reflect **`CONTROLLED_BROWSER_BENCHMARK`** on a developer workstation (MacBook Pro M-series, Chrome Desktop WebGPU). Production telemetry is recorded separately in `TORA_NATIVE_REAL_PRODUCTION_MATRIX.md`.*

### A. Background Remove (`u2netp.onnx` — 4.36 MB)
- **Model Download (Cold, Broadband 100Mbps)**: ~380 ms
- **Web Crypto SHA-256 Digest Verification**: ~18 ms
- **WebGPU Session Creation & Pipeline Compilation**: ~140 ms
- **Cold First Inference**: 1,367 ms
- **Warm Inference (Repeat Runs)**: **84.7 ms**
- **Bilinear Canvas Post-Processing & Rendering**: ~12 ms
- **Total First-Run Wait Time**: **~1,917 ms**
- **Total Warm Wait Time (Cached in IndexedDB)**: **~105 ms**

### B. Portrait Matting (`modnet.onnx` — 24.69 MB)
- **Model Download (Cold)**: ~1,250 ms
- **Web Crypto SHA-256 Digest Verification**: ~72 ms
- **WebGPU Session Creation**: ~280 ms
- **Cold First Inference**: 785 ms
- **Warm Inference**: **176.9 ms**
- **Post-Processing (Alpha Blending)**: ~25 ms
- **Total First-Run Wait Time**: **~2,412 ms**
- **Total Warm Wait Time**: **~210 ms**

### C. Image Upscale 2X (`realesrgan_2x.onnx` — 64.08 MB)
- **Model Download (Cold)**: ~3,400 ms
- **Web Crypto SHA-256 Digest Verification**: ~165 ms
- **Session & Tiled Pipeline Initialization**: ~410 ms
- **Cold First Inference (4 Tiles)**: 2,334 ms
- **Warm Inference (4 Tiles @ 128x128 with Overlap)**: **330.2 ms**
- **Tile Seam Linear Alpha Stitching**: ~35 ms
- **Total First-Run Wait Time**: **~6,344 ms**
- **Total Warm Wait Time**: **~385 ms**

### D. Image Upscale 4X (`RealESRGAN_x4plus.onnx` — 64.06 MB)
- **Model Download (Cold)**: ~3,400 ms
- **Cold First Inference (16 Tiles)**: 2,148 ms
- **Warm Inference (16 Tiles)**: **1,130.5 ms**
- **Total Warm Wait Time**: **~1,220 ms**

---

## 4. Key Performance Insights

1. **IndexedDB is Mandatory**: Once a model binary is cached in IndexedDB (`indexeddb://tora_models_v1`), the 380ms–3400ms network download is eliminated completely on subsequent visits.
2. **Sub-200ms Clarification**: Only `u2netp` and `modnet` warm runs achieve sub-200ms latency. Multi-tile super-resolution requires between 330ms and 1,130ms warm latency.
3. **Safety Caps on Input Dimensions**:
   - WebGPU max 2D texture dimensions are typically 8192 × 8192 or 16384 × 16384.
   - To prevent tab crashes and out-of-memory errors on shared unified RAM, Tora caps native browser inputs to **4096 × 4096 pixels**. Larger inputs must be downscaled or routed to server compute.
