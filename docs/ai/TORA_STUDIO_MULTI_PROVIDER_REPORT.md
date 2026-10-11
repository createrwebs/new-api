# TORA STUDIO QUEUE 7: MULTI-PROVIDER ROUTING & COST OPTIMIZATION

> **[SUPERSEDED NOTICE]**  
> **Status:** SUPERSEDED by `docs/ai/TORA_STUDIO_FINAL_ACCEPTANCE_REPORT.md`.  
> **Audit Finding:** The multi-provider routing architecture (Fal + Replicate adapters, Quality Tiers, scoring formula, and safe fallback) is completely implemented and tested. However, neither `FAL_KEY` nor `REPLICATE_API_TOKEN` is provisioned on the host. Under the strict acceptance policy, the true status is:  
> **`QUEUE 7 STATUS: CODE & TEST VERIFIED — LIVE MULTI-PROVIDER ADAPTERS OPERATOR_BLOCKED`**.

**Original Timestamp:** 2026-10-07T07:23:00+07:00  
**Repository:** `/Users/noppanan/new-api`  
**Branch:** `feat/formobile`

---

## 1. Executive Summary

Queue 7 establishes intelligent, multi-provider routing and cost optimization for Tora Studio. Rather than expanding provider counts blindly, Tora routes requests between **Fal.ai** and **Replicate** based on user-selected **Quality Tiers** (`FAST`, `QUALITY`, `PREMIUM`), real-time provider health, composite performance scoring, and strict safe-fallback invariants.

### Key Architectural Invariants Proven:
1. **Universal Single Wallet:** Quota reservation, settlement, and refunds occur strictly through the central Tora wallet (`model.PreConsumeUserWallet`, `model.SettleUserWalletPreConsume`, `model.RefundUserWalletPreConsume`).
2. **Provider Transparency:** The frontend and API expose user-friendly logical Quality Tiers (`FAST`, `QUALITY`, `PREMIUM`). Users never need to know internal provider details.
3. **Safe Fallback Invariant:** Fallback to a secondary provider is permitted **only** when connection drops before upstream processing begins (e.g., connection refused, offline host). Fallback is **strictly prohibited** on ambiguous timeouts (`ErrProviderAmbiguous`) or permanent policy errors (`ErrProviderPermanent`) to eliminate double-billing risks.

---

## 2. Provider Evaluation: Replicate vs MuAPI

| Evaluation Dimension | Replicate (Selected) | MuAPI |
| :--- | :--- | :--- |
| **API Specification** | Standard OpenAPI v1 REST (`/v1/predictions`) | Custom endpoint schema |
| **Prediction Lifecycle** | Deterministic: `starting` $\rightarrow$ `processing` $\rightarrow$ `succeeded` / `failed` | Asynchronous status queries |
| **Asset Outputs** | Direct immutable HTTPS CDN delivery | Variable hosted URLs |
| **Commercial Fit** | Established per-second billing with predictable ceilings | Tiered credits / proxy layer |
| **Target Workloads** | Rembg ($0.003), Real-ESRGAN ($0.005), Flux-schnell ($0.003) | Varied community endpoints |
| **Verdict** | **SELECTED** as secondary production provider | Held as tertiary candidate |

---

## 3. Quality Tiers & Candidate Routing Matrix

| Tool | Quality Tier | Primary Route | Fallback Route | Cost (USD) | Sell Price (USD) | Gross Margin |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **background-remove** | `FAST` | Replicate (`cjwbw/rembg`) | Fal (`fal-ai/birefnet`) | $0.003 | $0.020 (10 cr) | **85.0%** |
| **background-remove** | `QUALITY` | Fal (`fal-ai/birefnet`) | Replicate (`cjwbw/rembg`) | $0.005 | $0.020 (10 cr) | **75.0%** |
| **image-upscale** | `FAST` | Replicate (`nightmareai/real-esrgan`) | Fal (`fal-ai/clarity-upscaler`) | $0.005 | $0.030 (15 cr) | **83.3%** |
| **image-upscale** | `PREMIUM` | Fal (`fal-ai/clarity-upscaler`) | Replicate (`nightmareai/real-esrgan`) | $0.015 | $0.050 (25 cr) | **70.0%** |
| **image-generate** | `FAST` | Fal (`fal-ai/flux/schnell`) | Replicate (`flux-schnell`) | $0.003 | $0.016 (8 cr) | **81.25%** |
| **image-generate** | `QUALITY` | Fal (`fal-ai/flux/dev`) | Replicate (`flux-dev`) | $0.025 | $0.080 (40 cr) | **68.75%** |

---

## 4. Route Selection & Composite Scoring Formula

The router dynamically scores eligible candidates using four weighted telemetry signals:

$$\text{Score} = 0.35 \times \text{Availability} + 0.35 \times \text{TierMatch} + 0.15 \times \text{SuccessRate} + 0.15 \times \text{CostFactor}$$

Where:
- $\text{Availability} \in \{0.0, 1.0\}$: 0.0 if configuration validation fails or host is unreachable.
- $\text{TierMatch} \in [0.2, 1.0]$: 1.0 for exact tier match, penalized if provider is too slow/expensive for FAST tier or too low-res for PREMIUM.
- $\text{SuccessRate} \in [0.1, 1.0]$: Computed from consecutive job failures ($1.0 - 0.2 \times \text{fails}$).
- $\text{CostFactor} = \max(0, 1.0 - \frac{\text{Cost}_{\text{USD}}}{0.10})$.

---

## 5. Safe Fallback Proof & Double-Spend Protection

```
           Job Submission Request
                     │
         ┌───────────▼───────────┐
         │ Attempt Primary Route │
         └───────────┬───────────┘
                     │
          ┌──────────┴──────────┐
          │                     │
       Success               Failure
          │                     │
          ▼                     ▼
   [Settle Quota]      Is ErrProviderAmbiguous? ────────► YES ──► [HOLD RESERVATION / NO FALLBACK]
   (Normal Path)                │
                                NO
                                │
                       Is ErrProviderPermanent? ────────► YES ──► [REFUND / NO FALLBACK]
                                │
                                NO (Connection Refused / Offline)
                                │
                                ▼
                       ┌─────────────────┐
                       │ Safe Fallback   │
                       │ Secondary Route │
                       └─────────────────┘
```

### Verified Test Cases (`service/studio_router_test.go`):
1. `TestQueue7_QualityTier_RouteSelection`: Proves `FAST` picks Replicate ($0.003), `QUALITY` and `PREMIUM` pick Fal.
2. `TestQueue7_SafeFallback_ConnectionRefused`: Proves seamless handover to Replicate when Fal connection fails before sending bytes.
3. `TestQueue7_AmbiguousSubmission_BlocksFallback`: Proves zero fallback when primary times out ambiguously. Secondary provider calls = 0.
4. `TestQueue7_PermanentError_BlocksFallback`: Proves policy/schema rejections abort cleanly without redundant fallback attempts.
5. `TestQueue7_RoutingAnalytics`: Proves live aggregation of COGS, sell prices, and gross margins across providers.

---

## 6. Verification Status

```
=== RUN   TestQueue7_QualityTier_RouteSelection
--- PASS: TestQueue7_QualityTier_RouteSelection (0.00s)
=== RUN   TestQueue7_SafeFallback_ConnectionRefused
--- PASS: TestQueue7_SafeFallback_ConnectionRefused (0.00s)
=== RUN   TestQueue7_AmbiguousSubmission_BlocksFallback
--- PASS: TestQueue7_AmbiguousSubmission_BlocksFallback (0.00s)
=== RUN   TestQueue7_PermanentError_BlocksFallback
--- PASS: TestQueue7_PermanentError_BlocksFallback (0.00s)
=== RUN   TestQueue7_RoutingAnalytics
--- PASS: TestQueue7_RoutingAnalytics (0.01s)
PASS
ok  	github.com/QuantumNous/new-api/service	0.964s
```

**Status:** Completed and Verified.
