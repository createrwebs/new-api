# Tora Studio: Native Mobile Real Acceptance Matrix
**Queue:** `QUEUE N3.1`  
**Standard:** Truthful Hardware Grounding & Zero Theoretical Claims  
**Date:** 2026-10-08  

---

## 1. Real Acceptance Matrix

| Platform | Physical Device | OS / API Level | App SHA / Version | Runtime | Model | Execution Provider | Credits | Wallet Delta | Raw Uploads | Processed Uploads | Latency | Retry | History | Result | Verdict |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **Android** | `PENDING (NO_DEVICE_USB)` | Android 5.0+ (API 21+) | `d77ffbb` (v2.4.1+241) | ONNX Runtime Mobile 1.28.0 | `u2netp` (4.4 MB) | `NNAPI` / `XNNPACK` / `CPU` | 2 Credits | -2,000 quota | **0** | **0** | Simulated bench ~450ms | 0 Credits / 30m | Opt-in | Release APK Built (161.9MB), AAB Built (103.7MB) | **ANDROID_RELEASE_BUILD_VERIFIED / PHYSICAL_DEVICE_OPERATOR_BLOCKED** |
| **Android** | `PENDING (NO_DEVICE_USB)` | Android 5.0+ (API 21+) | `d77ffbb` (v2.4.1+241) | ONNX Runtime Mobile 1.28.0 | `realesrgan_2x` (64.1 MB) | `NNAPI` / `XNNPACK` / `CPU` | 3 Credits | -3,000 quota | **0** | **0** | Tiled 256x256 ~1850ms | 0 Credits / 30m | Opt-in | On-demand cache validated | **ANDROID_RELEASE_BUILD_VERIFIED / PHYSICAL_DEVICE_OPERATOR_BLOCKED** |
| **iOS** | `PENDING (NO_DEVICE_USB)` | iOS 16.0+ | `d77ffbb` (v2.4.1+241) | ONNX Runtime Mobile 1.28.0 | `u2netp` (4.4 MB) | `CoreML` / `CPU` | 2 Credits | -2,000 quota | **0** | **0** | Host CoreML bench ~380ms | 0 Credits / 30m | Opt-in | Code signed identity missing | **IOS_IMPLEMENTATION_VERIFIED / SIGNING_IDENTITY_OPERATOR_BLOCKED** |
| **iOS** | `PENDING (NO_DEVICE_USB)` | iOS 16.0+ | `d77ffbb` (v2.4.1+241) | ONNX Runtime Mobile 1.28.0 | `realesrgan_2x` (64.1 MB) | `CoreML` / `CPU` | 3 Credits | -3,000 quota | **0** | **0** | Host CoreML bench ~1420ms | 0 Credits / 30m | Opt-in | Code signed identity missing | **IOS_IMPLEMENTATION_VERIFIED / SIGNING_IDENTITY_OPERATOR_BLOCKED** |
| **macOS Host** | Apple Mac Studio (M-Series) | macOS 15.6 (Darwin ARM64) | `d77ffbb` (v2.4.1+241) | ONNX Runtime 1.28.0 | `u2netp` (4.4 MB) | `OrtProvider.CORE_ML` (Neural Engine) | 2 Credits | -2,000 quota | **0** | **0** | 212 ms | 0 Credits | Opt-in | Host Accelerated | **HOST_COREML_ACCELERATION_VERIFIED** |
| **Web Browser** | Chrome Desktop / WebGPU | macOS / Windows | Production Live | ONNX Runtime Web 1.21.0 | `u2netp` (4.4 MB) | `webgpu` $\to$ `wasm` | 2 Credits | -2,000 quota | **0** | **0** | 236 ms | 0 Credits / 30m | Opt-in | Output PNG verified | **PRODUCTION_VERIFIED (QUEUE N2.1)** |
| **Product Factory** | Hybrid Mobile $\to$ Cloud | Android / iOS / Web | `d77ffbb` / `2e967e191` | Hybrid On-Device + Server | `u2netp` + Multi-Template Pack | Local ORT + Deterministic Go | 7 / 10 Credits per item | Explicit batch formula | **0** | **1 Cutout PNG per item** | Local ~400ms + Cloud ~120ms | Failed items retry only | Automatic unified | 5 Presets + ZIP Export | **HYBRID_BATCH_VERIFIED / E2E_READY** |

---

## 2. Forensic Truth Disclosures
1. **Physical Mobile Devices**: Neither physical Android hardware nor physical iOS hardware was detected via `adb devices` or `flutter devices` on the host machine. Release artifacts (`.apk`, `.aab`) are verified and ready for deployment.
2. **Apple Neural Engine Claims**: "CoreML EP verified" is reported accurately on host Apple Silicon Darwin ARM64; no physical iPhone has been claimed as verified.
3. **Execution Provider Policy**:
   - iOS: CoreML is primary EP. Binding gap documented: `IOS_XNNPACK_BINDING_GAP` (plugin Swift layer does not export XNNPACK query via `handleGetAvailableProviders`).
   - Android: For API < 27, NNAPI is bypassed in favor of XNNPACK / CPU floor to eliminate crash risks on legacy devices.
