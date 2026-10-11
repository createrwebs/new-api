# TORA NATIVE REAL PRODUCTION MATRIX & ACCEPTANCE LEDGER
**Production Evidence, Execution Classes, Model Supply Chain & Billing Verification**
**Environment:** AWS EC2 Production (`saascover-api` / `51.20.174.90`) | **Branch:** `feat/formobile`
**Production Git Commit:** `b7ab06739` | **Docker Image:** `tora-api:n2-b7ab06739`
**Auditor:** Tora Web & Platform Engineering | **Date:** 2026-10-08

---

## 1. Truth in Labeling: Production Verification Stages

Every tool and capability in Tora Studio is tracked against nine mutually exclusive, non-collapsible verification stages:

1. `REFERENCE_HARVEST_VERIFIED`: Upstream open-source git repository cloned into isolated reference lab with SHA pinned.
2. `LICENSE_AUDIT_VERIFIED`: Legal audit confirms permissive commercial license (Apache-2.0, MIT, BSD-3-Clause).
3. `LOCAL_BENCHMARK_VERIFIED`: Offline benchmark executed on local dataset recording cold/warm latencies.
4. `CODE_IMPLEMENTED`: Server & client handlers, types, routes, and billing hooks committed to git.
5. `UNIT_TEST_VERIFIED`: Automated Go/TypeScript unit tests pass in isolation.
6. `REAL_BROWSER_VERIFIED`: Executed inside a real browser / runtime using ONNX Runtime Web.
7. `REAL_PUBLIC_PATH_VERIFIED`: Public reverse proxy, SSL/TLS, CDN headers, CORS, and model delivery endpoints verified live.
8. `REAL_CREDIT_CHARGE_VERIFIED`: Authoritative Tora Wallet (`users.quota`) debited and reconciled in production database.
9. `PRODUCTION_ACTIVE`: Formally promoted to public production catalog with live customer availability.

---

## 2. Tool Production Status Matrix

| Tool ID | Execution Class | Primary Model / Engine | Quota Cost (Credits) | Billing Policy | Verification Stage | Production Status |
|---|---|---|---|---|---|---|
| **`background-remove`** | `NATIVE_BROWSER` | `u2netp.onnx` (4.4 MB) | 2,000 (2 Credits) | `PREPAID_EXECUTION` | `REAL_CREDIT_CHARGE_VERIFIED` | 🚀 **BETA** |
| **`image-upscale-2x`** | `NATIVE_BROWSER` | `2x-realesrgan-x2plus.onnx` (64 MB) | 3,000 (3 Credits) | `PREPAID_EXECUTION` | `REAL_CREDIT_CHARGE_VERIFIED` | 🚀 **BETA** |
| **`product-pack`** | `DETERMINISTIC_SERVER` | Go Native Canvas + WebP/ZIP | 5,000 (5 Credits) | `SUCCESS_SETTLEMENT` | `REAL_CREDIT_CHARGE_VERIFIED` | 🚀 **ACTIVE** |
| **`image-generate (FAST)`** | `EXTERNAL_RELAY` | WaveSpeed `ws-flux-schnell` | 5,000 (5 Credits) | `SUCCESS_SETTLEMENT` | `REAL_CREDIT_CHARGE_VERIFIED` | 🚀 **ACTIVE** |
| **`portrait-matting`** | `NATIVE_BROWSER` | `modnet.onnx` (24.7 MB) | 2,000 (2 Credits) | `PREPAID_EXECUTION` | `LOCAL_BENCHMARK_VERIFIED` | ⏳ **STAGED** |
| **`image-upscale-4x`** | `NATIVE_BROWSER` | `RealESRGAN_x4plus.onnx` (64 MB) | 5,000 (5 Credits) | `PREPAID_EXECUTION` | `LOCAL_BENCHMARK_VERIFIED` | ⏳ **STAGED** |
| **`object-erase`** | `EXTERNAL_RELAY` | fal-ai `flux-dev-inpaint` | 30,000 (30 Credits) | `SUCCESS_SETTLEMENT` | `CODE_IMPLEMENTED` | 🚫 **DISABLED** |
| **`image-to-video`** | `EXTERNAL_RELAY` | fal-ai `wan-2.2` | 125,000 (125 Credits) | `SUCCESS_SETTLEMENT` | `CODE_IMPLEMENTED` | 🚫 **DISABLED** |

---

## 3. Production Model Artifact Supply Chain

Both open-weight ONNX model weights are served directly from the production application container with content-addressed caching:

| Model ID | Weight Filename | File Size | Exact SHA-256 Digest | Production Serving URL | Cache-Control Header | ETag Match | CORS Header |
|---|---|---|---|---|---|---|---|
| `u2netp` | `u2netp.onnx` | 4,574,861 B | `309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8` | `/api/studio/native/models/u2netp` | `public, max-age=31536000, immutable` | ✅ Matches | `Access-Control-Allow-Origin: *` |
| `realesrgan_2x` | `2x-realesrgan-x2plus.onnx` | 67,191,666 B | `c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483` | `/api/studio/native/models/realesrgan_2x` | `public, max-age=31536000, immutable` | ✅ Matches | `Access-Control-Allow-Origin: *` |

---

## 4. Production Billing V2 & Anti-Refund Abuse Ledger Audit

### A. Audit Subject
- **User ID**: `40`
- **Username**: `tora_canary_runner`
- **Starting Quota**: `50,000` (50.0 Tora Credits)
- **Ending Quota**: `38,000` (38.0 Tora Credits)
- **Net Quota Delta**: **`-12,000 Quota`** (-12.0 Tora Credits, reference value: $0.0240 USD)

### B. Transaction Sequence & Database Evidence

```
+-----------------------------------------------------------------------------------------------------------------------------------------+
| Step | Action               | Tool               | Execution Class      | Charged Quota | Wallet Balance | Ticket Status | Settlement Record ID                |
+-----------------------------------------------------------------------------------------------------------------------------------------+
| 0    | Initial State        | -                  | -                    | 0             | 50,000         | -             | -                                   |
| 1    | Issue Ticket 1       | background-remove  | NATIVE_BROWSER       | -2,000        | 48,000         | CHARGED       | req_native_tkt_1791439076_2d34ef03  |
| 2    | Complete Ticket 1    | background-remove  | NATIVE_BROWSER       | 0 (prepaid)   | 48,000         | COMPLETED     | settled (236ms inference, hash ok)  |
| 3    | Issue Ticket 2       | background-remove  | NATIVE_BROWSER       | -2,000        | 46,000         | CHARGED       | req_native_tkt_1791439078_0bc20487  |
| 4    | Malicious Refund     | background-remove  | NATIVE_BROWSER       | 0 (rejected)  | 46,000         | CHARGED       | REJECTED HTTP 400 (Abuse Blocked)   |
| 5    | Fair Retry           | background-remove  | NATIVE_BROWSER       | 0 (fair retry)| 46,000         | CHARGED       | Authorized retry window at 0 credit |
| 6    | Idempotent Replay    | background-remove  | NATIVE_BROWSER       | 0 (replayed)  | 46,000         | CHARGED       | Same ticket returned; 0 double charge|
| 7    | Issue Ticket 3       | image-upscale-2x   | NATIVE_BROWSER       | -3,000        | 43,000         | CHARGED       | req_native_tkt_1791439086_233095aa  |
| 8    | Complete Ticket 3    | image-upscale-2x   | NATIVE_BROWSER       | 0 (prepaid)   | 43,000         | COMPLETED     | settled (506ms inference, 2X exact) |
| 9    | Product Pack Export  | product-pack       | DETERMINISTIC_SERVER | -5,000        | 38,000         | SETTLED       | Generated 4 formats + 67.8 KB ZIP   |
+-----------------------------------------------------------------------------------------------------------------------------------------+
| TOTALS:                                                                 | -12,000 Quota | 38,000         | ZERO VARIANCE LEDGER INTEGRITY      |
+-----------------------------------------------------------------------------------------------------------------------------------------+
```

### C. Anti-Refund Abuse Security Proof
Under previous naive billing implementations, an attacker could create an execution ticket, download model weights, perform local inference, block the `/complete` webhook, and wait for automatic reconciliation to refund their quota.
In Queue N2.1:
1. `NATIVE_BROWSER` tools are strictly classified as `PREPAID_EXECUTION`. Quota is immediately and irrevocably debited and settled upon ticket issuance.
2. Malicious client refund request (`POST /api/studio/native/refund` on ticket `tkt_1791439078_0bc20487`) was **rejected with HTTP 400 Bad Request**:
   `"automatic refund is permitted only for pre-activation tickets; charged browser executions use fair retry"`.
3. Client suppression of `/complete` left the ticket in `CHARGED` state without any wallet refund. Zero credits leaked.
4. Client can safely retry inference on transient failures via `POST /api/studio/native/retry` within a 30-minute window at **0 additional credits**.
