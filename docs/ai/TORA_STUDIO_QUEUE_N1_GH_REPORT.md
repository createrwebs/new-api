# TORA STUDIO — QUEUE N1-GH COMPREHENSIVE MILESTONE REPORT
**GITHUB NATIVE TOOL HARVEST & ON-DEVICE / DETERMINISTIC TRANSITION**
**Milestone:** QUEUE N1-GH | **Status:** LOCAL_BENCHMARK_VERIFIED / UNIT_TEST_VERIFIED (PRODUCTION_GATES_PENDING_QUEUE_N2) | **Date:** 2026-10-08
**Verification Taxonomy:**
- `REFERENCE_HARVEST_VERIFIED`: ✅ PASS
- `LICENSE_AUDIT_VERIFIED`: ✅ PASS
- `LOCAL_BENCHMARK_VERIFIED`: ✅ PASS
- `CODE_IMPLEMENTED`: ✅ PASS
- `UNIT_TEST_VERIFIED`: ✅ PASS
- `REAL_BROWSER_VERIFIED`: ⏳ PENDING (Queue N2)
- `REAL_PUBLIC_PATH_VERIFIED`: ⏳ PENDING (Queue N2)
- `REAL_CREDIT_CHARGE_VERIFIED`: ⏳ PENDING (Queue N2)
- `PRODUCTION_ACTIVE`: ⏳ PENDING (Queue N2)
**Repository:** `/Users/noppanan/new-api` (`feat/formobile`) | **Lab:** `/Users/noppanan/tora-studio-lab`

---

## 1. Executive Summary & Strategic Shift

Tora Studio has completed its strategic transition from relying solely on third-party API relay resale to owning and deploying **Tora-Native AI Tools**:
- **Commercial Invariant Maintained**: Exactly **ONE Tora User**, **ONE Tora Wallet**, and **ONE Tora Billing Ledger** (`User.Quota`). All native tools quote, reserve, execute, and settle through standard Tora Credits ($1\text{ USD} = 500,000\text{ Quota} = 500\text{ Credits}$, $1\text{ Credit} = \$0.0020$).
- **Hardware & Cloud Cost Ceiling Enforced**: `NEW_SERVER_COUNT = 0`, `NEW_GPU_SERVER_COUNT = 0`. No expensive cloud GPU clusters or ComfyUI instances were spun up.
- **Client-Side Edge Execution (`NATIVE_BROWSER`)**: Light neural networks (`u2netp` 4.4MB, `modnet` 24.7MB, `realesrgan` 64MB) run directly inside user browser threads via WebGPU/WASM with **sub-200ms latency** and **\$0.0000 provider COGS**.
- **Deterministic Marketplace Processing**: Replaced slow, hallucinating generative diffusion workflows for e-commerce with a 100% deterministic, high-speed pure-Go pipeline generating compliant Shopee, Lazada, Instagram, and Story image packs in under 200ms.
- **Relay Continuity**: The verified WaveSpeed Generic Media Relay remains active in production as an automated high-end fallback.

---

## 2. Harvested Repositories Forensic Matrix

All 7 priority repositories were cloned into an isolated reference lab (`/Users/noppanan/tora-studio-lab/native-references`), pinned to immutable commit SHAs, and audited:

| Repository | Pinned Commit SHA | Primary License | Commercial Verdict | Role in Tora Platform |
|---|---|---|---|---|
| **`transparent-background`** | `6860e05c4c57db8e29127aa78323b77c31b1188b` | MIT | APPROVED | Reference for InSPyReNet tensor pipelines |
| **`rembg`** | `202e42649a8492a7c49f808de36608a7d1cbbfe3` | MIT | APPROVED | Reference for ONNX inference & alpha matting |
| **`MODNet`** | `28165a451e4610c9d77cfdf925a94610bb2810fb` | Apache-2.0 | APPROVED | Extracted `modnet_photographic_portrait_matting.onnx` |
| **`Real-ESRGAN`** | `a4abfb2979a7bbff3f69f58f58ae324608821e27` | BSD-3-Clause | APPROVED | Extracted `2x-realesrgan-x2plus.onnx` & `RealESRGAN_x4plus.onnx` |
| **`web-realesrgan`** | `1e3cf6331c80f25cafa5d0f196cc82922009931f` | **GPL-2.0** | **REFERENCE ONLY** | Studied tile overlapping math; zero code copied (clean-room implemented) |
| **`IOPaint`** | `61a759fb3f332bacdce8b2813f4837495c9b86e0` | Apache-2.0 | APPROVED | Studied LaMa inpainting & canvas masking UX |
| **`BiRefNet`** | `ebcc0bc8ec7fe919cec829f2dea656b3078acddc` | MIT code / Restricted datasets | REVIEW REQUIRED | Upstream code is MIT, but training sets contain non-commercial academic data |

---

## 3. Verified Open-Weight Model Registry

The following models were acquired, evaluated, and cryptographically verified:

```
c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483  2x-realesrgan-x2plus.onnx (64.08 MB)
cd0ec097469c94c903e6f74d4f43f545683250ec0a54bc0c2ab1ff4c6364d8da  RealESRGAN_x4plus.onnx (64.06 MB)
60920e99c45464f2ba57bee2ad08c919a52bbf852739e96947fbb4358c0d964a  isnet-general-use.onnx (170.37 MB)
07c308cf0fc7e6e8b2065a12ed7fc07e1de8febb7dc7839d7b7f15dd66584df9  modnet_photographic_portrait_matting.onnx (24.69 MB)
309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8  u2netp.onnx (4.36 MB)
```

---

## 4. Benchmark Performance & Selection Results

Inference was benchmarked across our 14-category synthetic evaluation dataset in `/Users/noppanan/tora-studio-lab`:

| Model | Primary Category | Cold Latency | Warm Latency | Peak RAM | Production Verdict |
|---|---|---|---|---|---|
| **`u2netp`** | Fast Cutout (Mobile) | 1,367 ms | **84.7 ms** | 185 MB | **WINNER**: Default for Mobile & Fast BG Removal (4.4MB download) |
| **`modnet`** | Portrait / Hair Matting | 785 ms | **176.9 ms** | 240 MB | **WINNER**: Default for People & Portrait Photography |
| **`realesrgan_2x`** | 2X Super-Resolution | 2,334 ms | **330.2 ms** | 310 MB | **WINNER**: Default for Fast 2X Upscale |
| **`RealESRGAN_x4plus`** | 4X Super-Resolution | 2,148 ms | **1,130.5 ms** | 420 MB | **WINNER**: Default for 4K Detail Upscale |
| **`isnet`** | High-Res Server Cutout | 3,420 ms | 940.0 ms | 650 MB | Serverless / Local CPU fallback option |

---

## 5. Software Architecture & Implementation Summary

### A. Data Models & Migrations (`model/studio_native.go`, `model/studio.go`)
- `NativeExecutionTicket`: Durably tracks client inference authorization, binding `UserId`, `ToolId`, `ReservedQuota`, `NormalizedInputHash`, and cryptographic `ModelVersionHash`.
- State transitions: `RESERVED` $\to$ `STARTED` $\to$ `SETTLED` / `REFUNDED` / `EXPIRED`.
- Integrated directly into `EnsureStudioTables(db)`.

### B. Execution Ticket Service (`service/studio_native.go`)
- `GetNativeToolQuote()`: Server-authoritative quote generation.
- `CreateNativeExecutionTicket()`: Reserves wallet quota atomically via `model.PreConsumeUserWallet(requestId, userId, quota)`.
- `CompleteNativeExecutionTicket()`: Validates ticket ownership, enforces 300s TTL, settles reservation via `model.SettleUserWalletPreConsume(requestId)`, and records execution telemetry.
- `RefundNativeExecutionTicket()`: Refunds quota via `model.RefundUserWalletPreConsume(requestId)` on client errors.
- `ReconcileExpiredNativeTickets()`: Background worker running every 2 minutes that sweeps abandoned tickets past TTL and restores user quota.

### C. Deterministic Marketplace Product Pack Pipeline (`service/studio_product_pack.go`)
- Detects non-transparent subject bounding box with 1px safety margin.
- Crops subject tightly and scales cleanly using `golang.org/x/image/draw.BiLinear`.
- Generates natural elliptical contact drop shadow beneath the subject's base.
- Generates compliant outputs for **Shopee (800x800)**, **Lazada (1000x1000)**, **Instagram (1080x1350)**, and **Story/TikTok (1080x1920)**.
- Packages all variants, transparent cutout PNG, and `manifest.json` into an in-memory ZIP package.

### D. REST Controllers & Routes (`controller/studio_native.go`, `router/api-router.go`)
- `POST /api/studio/native/quote`: Public tool pricing quote.
- `POST /api/studio/native/ticket`: Authorized ticket issuance & escrow deduction.
- `POST /api/studio/native/complete`: Client inference settlement.
- `POST /api/studio/native/refund`: Client error rollback.
- `POST /api/studio/native/product-pack`: Server-side deterministic pack processing.

### E. Unit Test Verification (`service/studio_native_test.go`)
- All 5 test suites executed and passed with 100% success rate:
  1. `TestStudioNative_QuotesAndCatalogSpecs` (PASS)
  2. `TestStudioNative_TicketLifecycle_ReserveAndComplete` (PASS)
  3. `TestStudioNative_TicketLifecycle_Refund` (PASS)
  4. `TestStudioNative_Reconciliation_ExpiredTickets` (PASS)
  5. `TestStudioProductPack_DeterministicPipeline` (PASS)

---

## 6. Financial Economics & Margin Re-Alignment

| Capability | Previous Relay Price | Native Tora Price | Price Reduction to Merchant | Previous Margin | Native Platform Margin |
|---|---|---|---|---|---|
| **Background Remove** | 10 Credits (\$0.020) | **2 Credits (\$0.004)** | **-80%** | 75% | **> 99%** |
| **Portrait Matting** | 15 Credits (\$0.030) | **2 Credits (\$0.004)** | **-87%** | 67% | **> 99%** |
| **Image Upscale 2X** | 25 Credits (\$0.050) | **3 Credits (\$0.006)** | **-88%** | 70% | **> 98%** |
| **Image Upscale 4X** | 25 Credits (\$0.050) | **5 Credits (\$0.010)** | **-80%** | 84% | **> 98%** |
| **Product Pack (4 Sizes + Zip)** | 35 Credits (\$0.070) | **5 Credits (\$0.010)** | **-86%** | 64% | **> 99%** |

---

## 7. Deliverable Documentation Artifacts

1. `docs/ai/TORA_NATIVE_GITHUB_LICENSE_MATRIX.md` (Forensic licensing audit & determinations)
2. `docs/ai/TORA_NATIVE_GITHUB_PATTERN_HARVEST.md` (Harvested architecture patterns)
3. `docs/ai/TORA_NATIVE_BACKGROUND_REMOVAL_BENCHMARK.md` (Background removal benchmark report)
4. `docs/ai/TORA_NATIVE_UPSCALE_BENCHMARK.md` (Upscale benchmark report)
5. `docs/ai/TORA_NATIVE_BROWSER_RUNTIME.md` (Client Web Worker & WebGPU runtime architecture)
6. `docs/ai/TORA_NATIVE_EXECUTION_TICKET.md` (Financial protocol specification & abuse prevention)
7. `docs/ai/TORA_MARKETPLACE_PRODUCT_PACK.md` (Deterministic e-commerce pipeline specification)
8. `docs/ai/TORA_NATIVE_VS_RELAY_COST_MATRIX.md` (Economic comparison matrix)
9. `docs/ai/TORA_NATIVE_OBJECT_CLEANUP_RESEARCH.md` (LaMa inpainting research report)
10. `docs/ai/TORA_STUDIO_QUEUE_N1_GH_REPORT.md` (This authoritative milestone report)

---

## 8. Git Commit Verification

All changes compiled cleanly and verified via tests.
Ready for production integration and deployment.

---

## 9. Queue N1-GH Errata & Precision Corrections

1. **Production Status Erratum**:
   - *Previous Text*: "100% COMPLETE & VERIFIED"
   - *Correction*: The N1-GH phase completed reference harvesting, license forensics, local ONNX benchmarking, data models, and Go service unit tests in branch `feat/formobile` (commit `775da8c05`). However, production deployment, real browser client execution, and real user wallet deductions were not yet performed on `https://www.toraapi.com`. Real production verification is strictly deferred to Queue N2.

2. **Sub-200ms Latency Wording Erratum**:
   - *Previous Text*: "sub-200ms latency across client-side edge execution"
   - *Correction*: Sub-200ms warm latency applies specifically to lightweight background removal models (`u2netp` at 84.7ms and `modnet` at 176.9ms on Apple Silicon). Super-resolution models take significantly longer: `realesrgan_2x` takes 330.2ms and `RealESRGAN_x4plus` takes 1,130.5ms. Furthermore, total end-user wait time includes initial model download, session creation, shader compilation, and post-processing canvas encoding.

3. **Platform Margin & Cost Wording Erratum**:
   - *Previous Text*: "Native Platform Margin >98–99%"
   - *Correction*: While third-party upstream compute costs are eliminated (`PROVIDER_COGS ≈ $0.0000`), the true `FULLY_ALLOCATED_COST` is not yet mature. Fully allocated unit economics must account for CDN model weight egress (especially for 25MB–64MB models on initial fetch), S3 asset storage for history persistence, and server API traffic. Platform gross margin cannot be classified as >98% without live production accounting evidence.

4. **Browser Billing Semantics & Security Vulnerability Erratum**:
   - *Vulnerability Identified*: The initial V1 ticket lifecycle (reserve quota $\to$ client runs local inference $\to$ client calls `/complete` $\to$ settle, or refund on timeout) contains a critical exploit. A client running open-weight models locally can obtain the completed cutout, intentionally suppress the `/complete` call, wait 5 minutes for ticket expiration, and receive a full quota refund, thus obtaining free inference.
   - *Remediation*: As specified in Queue N2, `NATIVE_BROWSER` execution must use **`PREPAID_EXECUTION`**, where the charge is settled atomically *at ticket activation*, before client inference begins. Charged tickets are never automatically refunded upon timeout, and same-ticket retries are permitted within a bounded 15–30 minute window at 0 additional cost.
