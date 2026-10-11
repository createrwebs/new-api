# Tora Native Mobile Model Manifest & Supply Chain Integrity

**Identifier:** `TORA-MANIFEST-N3-MODELS`  
**Date:** 2026-10-08  
**Scope:** Pinned cryptographic model weights, distribution architecture, and licensing for on-device mobile inference.  

---

## 1. Authoritative Model Supply Chain

All models used in Tora Native Mobile are pinned to exact cryptographic SHA-256 digests. Any binary byte alteration triggers an immediate `CHECKSUM_MISMATCH` rejection.

| Model ID | Formal Model Name | Task | Framework / Format | SHA-256 Digest | Size (Bytes) | License | Asset Strategy |
|---|---|---|---|---|---|---|---|
| `u2netp` | U2Net-P Fast Mobile Cutout | Background Removal | ONNX (opset 12) | `309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8` | 4,572,242 (~4.4 MB) | Apache-2.0 | **Bundled in App Assets** (`assets/models/u2netp.onnx`) |
| `realesrgan_2x` | Real-ESRGAN x2plus | 2X Neural Upscaling | ONNX (opset 13) | `c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483` | 67,192,664 (~64.1 MB) | BSD-3-Clause | **On-Demand Cache** (Streamed from `/api/studio/native/models/realesrgan_2x`) |
| `modnet` | MODNet Photographic Portrait | Portrait Matting | ONNX (opset 12) | `07c308cf0fc7e6e8b2065a12ed7fc07e1de8febb7dc7839d7b7f15dd66584df9` | 25,890,000 (~24.7 MB) | Apache-2.0 | On-Demand Cache |
| `realesrgan_4x` | Real-ESRGAN x4plus | 4X Neural Upscaling | ONNX (opset 13) | `cd0ec097469c94c903e6f74d4f43f545683250ec0a54bc0c2ab1ff4c6364d8da` | 67,174,378 (~64.1 MB) | BSD-3-Clause | On-Demand Cache |

---

## 2. Distribution Strategy & Cache Architecture

### 2.1 Bundled Model (`u2netp.onnx`)
- **Size**: 4.4 MB.
- **Location**: Included directly in app bundle under `assets/models/u2netp.onnx`.
- **User Experience**: Users have **zero initial download delay** for background removal. First-run cutout executes immediately upon installation, even without cellular/WiFi connectivity.
- **Integrity**: Hash validated during extraction to application support directory.

### 2.2 On-Demand Streaming Cache (`realesrgan_2x`)
- **Size**: 64.1 MB.
- **Location**: Cached in `getApplicationSupportDirectory() / tora_models / 2x-realesrgan-x2plus.onnx`.
- **User Experience**: Light base APK download size. When the user enables 2X upscale, the weight file is streamed once with a percentage progress bar, verified with streaming SHA-256 computation, and stored permanently in app sandbox storage.
- **Eviction / Integrity Policy**: If the local file fails SHA-256 check (e.g. storage corruption), it is deleted and re-downloaded automatically.

---

## 3. License Audit & Commercial Rights

- **Apache-2.0 (`u2netp`, `modnet`)**: Permissive commercial use, modification, and binary distribution permitted. Patent grant included.
- **BSD-3-Clause (`realesrgan_2x`, `realesrgan_4x`)**: Commercial use and binary redistribution permitted with notice retention.
- **Zero GPL / Copyleft**: No viral copyleft dependencies are linked or compiled into the mobile binary.
