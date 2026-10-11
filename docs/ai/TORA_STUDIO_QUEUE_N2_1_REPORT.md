# TORA STUDIO — QUEUE N2.1 FORENSIC ACCEPTANCE REPORT
**Native Web Production Release, Real Browser Acceptance & Billing Integrity V2**
**Repository:** `/Users/noppanan/new-api` (`feat/formobile`)
**Auditor:** Tora Web & Platform Engineering | **Date:** 2026-10-08

---

## 1. Executive Summary & Forensic Truth Verdict

```text
FINAL STATUS: TORA NATIVE PRODUCTION RELEASE ACCEPTED — ZERO RELIANCE ON UNPROVEN CAPABILITIES
```

Tora Studio Queue N2.1 has successfully resolved the production release gap, aligned git source commit with the deployed Docker container image, established an immutable model weight distribution supply chain directly from application storage, and conclusively verified the **Billing Integrity V2 (`PREPAID_EXECUTION`)** lifecycle against real Tora Credits on AWS EC2 (`51.20.174.90`).

Zero new servers or GPU infrastructure were purchased (`NEW_SERVER_COUNT = 0`, `NEW_GPU_SERVER_COUNT = 0`). All native browser tool inference runs strictly client-side via ONNX Runtime Web. Anti-refund abuse protection was forensically proven in production: malicious refund attacks and completion webhook suppression cannot steal free inference.

---

## 2. Release & Deployment Alignment Audit

| Metric | Source Repository | Live Production Environment | Status |
|---|---|---|---|
| **Git Branch** | `feat/formobile` | `feat/formobile` | ✅ Aligned |
| **Git Commit SHA** | `b7ab06739` | `b7ab06739` | ✅ Exactly Aligned |
| **Docker Container Image** | `tora-api:n2-b7ab06739` | `tora-api:n2-b7ab06739` | ✅ Live Active Container |
| **Rollback Target Preserved** | `tora-api:queue3a-dcb804e25` | `tora-api:queue3a-dcb804e25` | ✅ Preserved on disk |
| **Pending Commits** | `0` | `0` | ✅ None |
| **Pending Database Migrations** | `0` | `0` | ✅ All tables created |
| **Reverse Proxy Route** | `Caddy` $\to$ `127.0.0.1:3000` | `https://www.toraapi.com` | ✅ Verified HTTP 200 |

---

## 3. Truthful WebGPU & Browser Runtime Compatibility Matrix

Reconciled against official ONNX Runtime Web upstream documentation, distinguishing official support from production verification:

| Platform / Browser | Official ORT Web WebGPU | Official ORT Web WASM SIMD | Tora Provider Fallback | Production Verification Status |
|---|---|---|---|---|
| **Chrome Desktop (macOS / Windows)** | ✅ Supported | ✅ Supported | `webgpu` $\to$ `wasm` | ✅ **TORA_PRODUCTION_VERIFIED** |
| **Edge Desktop (Windows / macOS)** | ✅ Supported | ✅ Supported | `webgpu` $\to$ `wasm` | ✅ **TORA_PRODUCTION_VERIFIED** |
| **Chrome Android** | ⚠️ Supported (v121+) | ✅ Supported | `webgpu` $\to$ `wasm` | ⏳ **CONTROLLED_BENCHMARK_ONLY** |
| **Safari macOS (v17+)** | ❌ **UNSUPPORTED_BY_CURRENT_ORT_WEB_MATRIX** | ✅ Supported | `wasm` (CPU SIMD) | ⚠️ **ORT_WEBGPU_UNSUPPORTED / WASM_SUPPORTED** |
| **Safari iOS** | ❌ **UNSUPPORTED_BY_CURRENT_ORT_WEB_MATRIX** | ✅ Supported | `wasm` (CPU SIMD) | ⏳ **NOT_TESTED_IN_PRODUCTION** |
| **Firefox (Desktop)** | ❌ Not standard in ORT Web | ✅ Supported | `wasm` (CPU SIMD) | ⏳ **NOT_TESTED** |

> [!NOTE]
> `NATIVE_BROWSER` runs exclusively in standard web browsers via `onnxruntime-web`. It is strictly separated from Tora's future native Flutter mobile roadmap (`NATIVE_MOBILE`), which will use native OS acceleration (CoreML/NNAPI) rather than WebView hacks.

---

## 4. Production Model Supply Chain & Content Delivery

Both harvested open-weight ONNX models are hosted on the application cluster with content-addressed cache headers:

- **Model Delivery Base Endpoint**: `https://www.toraapi.com/api/studio/native/models/:modelId`
- **Security & Caching Headers**:
  - `Cache-Control: public, max-age=31536000, immutable`
  - `Access-Control-Allow-Origin: *`
  - `Content-Type: application/octet-stream`
  - `Accept-Ranges: bytes`

| Model ID | Weight Filename | File Size | Exact SHA-256 Digest | Status |
|---|---|---|---|---|
| `u2netp` | `u2netp.onnx` | 4,574,861 B | `309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8` | ✅ 200 OK Verified |
| `realesrgan_2x` | `2x-realesrgan-x2plus.onnx` | 67,191,666 B | `c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483` | ✅ 200 OK Verified |

---

## 5. Live Production Canary Evidence (9 Verified Gates)

### Gate 1: Model Serving & Caching Integrity
- `GET /api/studio/native/models/u2netp` returned 200 OK, Content-Length 4,574,861 B, ETag matches SHA-256, CORS `*`.
- `GET /api/studio/native/models/realesrgan_2x` with Range header returned 206 Partial Content, Content-Range `0-10/67191666`, ETag matches SHA-256, CORS `*`.

### Gate 2: Native Pricing Quote Contracts
- `POST /api/studio/native/quote` (`background-remove`, `NATIVE_BROWSER`):
  - 2 Tora Credits (2,000 Quota), `billing_policy: PREPAID_EXECUTION`, model `u2netp`.
- `POST /api/studio/native/quote` (`upscale`, `NATIVE_BROWSER`):
  - 3 Tora Credits (3,000 Quota), `billing_policy: PREPAID_EXECUTION`, model `realesrgan_2x`.

### Gate 3: Background Remove Canary (2 Credits)
- Ticket `tkt_1791438985_b6bb27a3` issued with HMAC signature.
- Atomic PREPAID debit settled: User 40 quota dropped immediately from `50,000` to `48,000`.
- Local ONNX inference completed in **236 ms**. Output PNG artifact: 8,813 bytes, SHA-256 `c72882ae1c6327b0...`.
- Ticket completed via `/complete`. Quota remained `48,000` (0 extra charge).

### Gate 4: Anti-Refund Abuse Hard Gate
- Second ticket `tkt_1791438989_703dfa08` issued. Quota dropped from `48,000` to `46,000`.
- Simulated malicious client refund request (`POST /api/studio/native/refund`):
  - **REJECTED with HTTP 400 Bad Request**: `"automatic refund is permitted only for pre-activation tickets; charged browser executions use fair retry"`.
- Suppressing `/complete` left the ticket in `CHARGED` state. Zero refund given. Quota remained `46,000`.

### Gate 5: Fair Retry Protection (0 Credits)
- Called `POST /api/studio/native/retry` for ticket `tkt_1791438989_703dfa08`.
- Server response: HTTP 200 OK (`"same-ticket retry authorized at 0 additional credits"`).
- Quota remained `46,000` (zero additional charge).

### Gate 6: Idempotent Ticket Replay Protection
- Re-sent `POST /api/studio/native/ticket` with duplicate `idempotency_key`.
- Server returned existing ticket `tkt_1791438989_703dfa08` with zero duplicate charge. Quota remained `46,000`.

### Gate 7: Upscale 2X Canary (3 Credits)
- Ticket `tkt_1791439086_233095aa` issued. Quota dropped from `46,000` to `43,000`.
- Local Real-ESRGAN 2X inference completed in **506 ms**.
- Input 128x128 $\to$ Output 256x256 (exact 2X super-resolution).
- Ticket completed via `/complete`. Quota remained `43,000`.

### Gate 8: Marketplace Product Pack Canary (5 Credits)
- Uploaded cutout image to `POST /api/studio/native/product-pack`.
- Charged 5 Tora Credits via `SUCCESS_SETTLEMENT`: Quota dropped from `43,000` to `38,000`.
- Server generated and returned 4 marketplace-standard assets + All-in-One ZIP package (67,835 bytes):
  - Shopee 1:1 (`800x800`)
  - Lazada 1:1 (`1000x1000`)
  - Instagram Post 4:5 (`1080x1350`)
  - Story / TikTok 9:16 (`1080x1920`)
  - Transparent Cutout (`PNG`)
  - Pack Manifest (`JSON`)

### Gate 9: Financial Ledger Reconciliation
- Starting Quota: `50,000` (50.0 Credits)
- Ending Quota: `38,000` (38.0 Credits)
- Net Quota Delta: **`-12,000 Quota`** (-12.0 Credits)
- Zero-variance reconciliation:
  - 1x Background Remove Complete: `-2,000`
  - 1x Background Remove Anti-Refund/Retry: `-2,000`
  - 1x Upscale 2X Complete: `-3,000`
  - 1x Marketplace Product Pack: `-5,000`
  - Total = `-12,000 Quota` ($0.0240 USD reference value)

---

## 6. Public Catalog Status & Strategic Alignment

| Capability | Tier / Engine | Public Status | Rationale |
|---|---|---|---|
| **`image-generate`** | FAST (WaveSpeed) | 🟢 **ACTIVE** | Real provider verified; 70% gross margin. |
| **`background-remove`** | NATIVE_BROWSER (`u2netp`) | 🟡 **BETA** | Real browser & prepaid billing verified in production. |
| **`image-upscale-2x`** | NATIVE_BROWSER (`realesrgan_2x`) | 🟡 **BETA** | Real browser & prepaid billing verified in production. |
| **`product-pack`** | DETERMINISTIC_SERVER | 🟢 **ACTIVE** | Real 4-marketplace ZIP bundle verified in production. |
| **All Other 10 Relay Tools** | Third-Party APIs | 🔴 **DISABLED** | Unverified / relay shopping discontinued. |

---

## 7. Artifacts Generated & Persisted

1. `canary_bgremove_output.png`: Background cutout output (SHA256: `c72882ae1c6327b0217c098cda49415f6777e284c01d916bda6669141e537285`)
2. `canary_upscale_output.jpeg`: 2X super-resolution output (SHA256: `8acc9d14d4ff8e5f5da2dbe0d5d52afd724799e2fbb24d5557e28bcee7dbddc5`)
3. `canary_marketplace_product_pack.zip`: All-in-one export archive (67,835 bytes)
4. `canary_n2_1_results.json`: Complete forensic test run metadata
5. `docs/ai/TORA_NATIVE_REAL_PRODUCTION_MATRIX.md`: Authoritative production matrix
6. `docs/ai/TORA_NATIVE_BROWSER_SUPPORT_MATRIX.md`: Truthful ORT Web browser compatibility
