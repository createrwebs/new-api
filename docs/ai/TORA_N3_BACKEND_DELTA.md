# Tora Studio Queue N3 Backend Delta Audit
**Commit Range:** `b7ab06739..b0f915f86`  
**Target Deployment Image:** `tora-api:n3-b0f915f86`  
**Baseline Production Image:** `tora-api:n2-b7ab06739`  
**Branch:** `feat/formobile`  
**Date:** 2026-10-08  

---

## 1. Executive Summary

This forensic delta audit examines all commits between production baseline `b7ab06739` and current source HEAD `b0f915f86`. It classifies each commit by architectural necessity, details runtime API and schema alterations, and demonstrates why upgrading the production image to `tora-api:n3-b0f915f86` is required to ensure 100% runtime compatibility with the LumenFlow mobile client and e-commerce Product Factory.

---

## 2. Commit Classification

| Commit Hash | Author Date | Commit Subject | Classification | Runtime Necessity |
|---|---|---|---|---|
| `d68d913e3` | 2026-10-08 12:59 | `docs(studio): add Queue N2.1 production canary report and real production matrix` | **DOCS_ONLY** | None |
| `2e2a70440` | 2026-10-08 13:41 | `feat(studio): add NATIVE_MOBILE execution class with PREPAID_EXECUTION and fair retry` | **MOBILE_REQUIRED** | **CRITICAL**: Enables `NATIVE_MOBILE` execution class in DB enum, tickets, and quotes |
| `84d672f51` | 2026-10-08 13:56 | `docs(studio): add comprehensive Queue N3 Native Mobile and Product Factory specifications` | **DOCS_ONLY** | None |
| `cec029fc2` | 2026-10-08 14:53 | `docs(studio): update Queue N3 report commit hashes` | **DOCS_ONLY** | None |
| `2e967e191` | 2026-10-08 18:27 | `feat(studio): add batch quote model, remote kill switch, and min app version gate` | **PRODUCT_FACTORY_REQUIRED / MOBILE_REQUIRED** | **CRITICAL**: Implements batch pricing (`PER_ITEM`, `PER_BATCH`, `BUNDLE`), 30-min fair retry for mobile, remote kill switches, and minimum mobile version gate |
| `f0cf06180` | 2026-10-08 18:37 | `fix(router): remove duplicate product-factory/quote route registration` | **ROUTER_INTEGRITY** | **CRITICAL**: Eliminates Gin route panic |
| `b0f915f86` | 2026-10-08 18:49 | `chore(docker): ignore data and logs in build context` | **BUILD_HYGIENE** | Prevents container disk bloat |

---

## 3. Detailed Runtime & Schema Impact

### 3.1 Commit `2e2a70440` (Mobile Execution Class)
- **Files Modified**: `model/studio_native.go`, `service/studio_native.go`, `service/studio_native_test.go`
- **Architectural Changes**:
  - Defined `ExecutionClassNativeMobile = "NATIVE_MOBILE"`.
  - Configured `BillingPolicyPrepaidExecution` for `NATIVE_MOBILE` (previously only `NATIVE_BROWSER` was treated as prepaid).
  - Ensured atomic wallet deduction prior to authorization token issuance on mobile.
  - Added unit test `TestStudioNative_NativeMobile_QuoteAndPrepaidExecution`.
- **Runtime Impact**: Old production image `b7ab06739` did not recognize `NATIVE_MOBILE` as a prepaid execution class, falling back to server success settlement which broke client ticket verification.

### 3.2 Commit `2e967e191` (Batch Pricing, Kill Switch & Version Gate)
- **Files Modified**: `model/studio_native.go`, `service/studio_native.go`, `controller/studio_native.go`, `router/api-router.go`, `service/studio_native_test.go`
- **Architectural Changes**:
  1. **Batch Price Scope (`model/studio_native.go`)**:
     - Added `ProductFactoryPriceScope` (`PER_ITEM`, `PER_BATCH`, `BUNDLE`).
     - Fixed `CreateNativeTicket` to set a 30-minute fair retry window (`RetryUntil = now + 1800`) for `NATIVE_MOBILE`.
  2. **Product Factory Batch Quote Service (`service/studio_native.go`)**:
     - Added `CalculateProductFactoryBatchQuote` supporting 1 to 10 photos.
     - Unambiguous pricing: 7 Credits/item (base) and 10 Credits/item (enhanced).
     - Explicit bundle discounts: 5% for 5–9 items, 10% for 10 items.
     - Pricing version: `v2_batch_bundle`.
  3. **Remote Kill Switch & Minimum App Version Gate (`service/studio_native.go`)**:
     - `MinimumMobileAppVersion = "2.4.0"` enforced. Rejects outdated clients with HTTP 426 (`Upgrade Required`).
     - Route kill switch registry: `native_mobile_u2netp`, `native_mobile_realesrgan2x`. Instantly stops unstable mobile routes server-side without releasing an app update.
  4. **API Route Registration (`router/api-router.go`, `controller/studio_native.go`)**:
     - Registered `POST /api/studio/native/product-factory/quote` for both public estimation and authenticated checkout.
- **Runtime Impact**: Resolves Product Factory pricing ambiguity completely; ensures client app updates are gated cleanly.

---

## 4. Database Schema Impact Assessment
- **Table Alterations**: **ZERO** destructive modifications.
- **Table Additions**: None. Table `studio_native_tickets` uses varchar columns (`execution_class`, `billing_policy`, `status`) which natively accept the new string constants (`NATIVE_MOBILE`, `BUNDLE`).
- **Data Migration**: Not required. 100% backward-compatible.

---

## 5. Deployment Verification Conclusion
Upgrading the production API container from `tora-api:n2-b7ab06739` to `tora-api:n3-2e967e191` is strictly required for Queue N3.1. Rollback image `tora-api:n2-b7ab06739` will be retained intact in the local Docker daemon.
