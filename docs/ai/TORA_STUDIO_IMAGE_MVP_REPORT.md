# Tora Studio Queue 3: Public Image MVP & Credit Conversion Report

```text
QUEUE 3 STATUS: IMAGE STUDIO MVP LIVE
```

**Date:** 2026-10-07  
**Branch:** `feat/formobile`  
**Commit Baseline:** `7f912c702` + Queue 3 Implementation  
**Environment:** Staging / Production Pre-release  
**Universal Wallet Policy:** ONE USER, ONE WALLET, ONE LEDGER  

---

## 1. Executive Summary

Tora Studio Queue 3 has successfully transitioned from an engineering prototype to a commercial public Image MVP with an integrated credit conversion and attribution funnel.

The initial public tools have been rigorously tested, priced, gated, and published with SEO landing pages and tracking telemetry. All video tools remain frozen until the dedicated video phase.

```
+----------------------------------------------------------------------------------------------------+
|                                    TORA STUDIO IMAGE MVP MATRIX                                    |
+------------------------------------+----------------+-------------------+--------------------------+
| Tool Slug                          | State          | Pricing (Credits) | Fal / Provider Route     |
+------------------------------------+----------------+-------------------+--------------------------+
| background-remove                  | ACTIVE         | 2                 | fal-ai/bria/rmbg-2.0     |
| image-upscale                      | ACTIVE         | 3                 | fal-ai/clarity-upscaler  |
| image-generate                     | ACTIVE         | 5                 | fal-ai/flux/schnell      |
| product-photo                      | BETA           | 6                 | fal-ai/bria/genfill      |
| object-erase                       | BETA           | 4                 | fal-ai/bria/genfill      |
| image-extend                       | BETA           | 4                 | fal-ai/bria/genfill      |
| image-to-video                     | COMING_SOON    | -                 | FROZEN (Non-public)      |
| text-to-video                      | COMING_SOON    | -                 | FROZEN (Non-public)      |
+------------------------------------+----------------+-------------------+--------------------------+
```

---

## 2. Tool Activation & Economics Verification

### A. Active & Beta Image Catalog
1. **`background-remove` (ACTIVE)**
   - **Endpoint:** `fal-ai/bria/rmbg-2.0`
   - **Unit Cost:** ~$0.010
   - **Tora Credits:** 2 (~$0.020 value)
   - **Gross Margin:** ~50%
2. **`image-upscale` (ACTIVE)**
   - **Endpoint:** `fal-ai/clarity-upscaler`
   - **Unit Cost:** ~$0.020
   - **Tora Credits:** 3 (~$0.030 value)
   - **Gross Margin:** ~33%
3. **`image-generate` (ACTIVE)**
   - **Endpoint:** `fal-ai/flux/schnell`
   - **Unit Cost:** ~$0.025
   - **Tora Credits:** 5 (~$0.050 value)
   - **Gross Margin:** ~50%
4. **`product-photo` (BETA)**
   - **Endpoint:** `fal-ai/bria/genfill`
   - **Unit Cost:** ~$0.030
   - **Tora Credits:** 6 (~$0.060 value)
   - **Gross Margin:** ~50%
5. **`object-erase` (BETA)**
   - **Endpoint:** `fal-ai/bria/genfill`
   - **Unit Cost:** ~$0.020
   - **Tora Credits:** 4 (~$0.040 value)
   - **Gross Margin:** ~50%
6. **`image-extend` (BETA)**
   - **Endpoint:** `fal-ai/bria/genfill`
   - **Unit Cost:** ~$0.020
   - **Tora Credits:** 4 (~$0.040 value)
   - **Gross Margin:** ~50%

### B. Video Tools Frozen
`image-to-video`, `text-to-video`, `lip-sync`, `face-swap`, `talking-avatar`, and `video-upscale` have been formally locked to `COMING_SOON`, `IsEnabled: false`, `IsPublic: false`. They are excluded from public directory lists and search indexes.

---

## 3. Tool Quality Assurance Matrix

The test suite in [`service/studio_image_mvp_test.go`](file:///Users/noppanan/new-api/service/studio_image_mvp_test.go) passed all QA vectors:

| QA Vector | Test Case | Status | Verified Behavior |
| :--- | :--- | :--- | :--- |
| **Small Payload** | `TestQueue3_ToolQA_SmallAndLargeInput` | **PASS** | Valid 500-byte PNG completes and reserves quota normally |
| **Large Payload** | `TestQueue3_ToolQA_SmallAndLargeInput` | **PASS** | Allowed up to 14.5MB; files exceeding 15MB are safely rejected with 413 |
| **Invalid MIME** | `TestQueue3_ToolQA_InvalidMIME_And_Security` | **PASS** | Windows PE executables, Linux ELF binaries, and scripts rejected |
| **Media Type Mismatch** | `TestQueue3_ToolQA_InvalidMIME_And_Security` | **PASS** | MP4 video uploads to Image tools rejected with 400 Bad Request |
| **SSRF Mitigation** | `TestQueue3_ToolQA_InvalidMIME_And_Security` | **PASS** | Loopback and cloud metadata endpoints (`169.254.169.254`) blocked |
| **Insufficient Credits** | `TestQueue3_ToolQA_InsufficientCredits` | **PASS** | Zero balance blocks job creation; 0 credits deducted |
| **Provider Validation** | `TestQueue3_ToolQA_ProviderValidationError_Refunds100Percent` | **PASS** | Upstream 422 triggers automatic 100% refund to user wallet |
| **Job Cancellation** | `TestQueue3_ToolQA_Cancel_And_Refund` | **PASS** | Cancelling queued/processing job restores 100% of reserved quota |
| **Idempotency** | `TestQueue3_ToolQA_DuplicateRequest_And_Retry` | **PASS** | Replayed client token returns original job without duplicate charge |
| **Remix Chain** | `TestQueue3_ToolQA_RemixChain` | **PASS** | Output of `image-generate` feeds cleanly into `background-remove` and `image-upscale` |
| **Conversion Funnel** | `TestQueue3_CreditConversionFunnel_E2E` | **PASS** | Full loop: Quote -> Block -> Purchase -> Return -> Requote -> User Confirms -> Run |
| **Admin Metrics** | `TestQueue3_AdminMetrics_TelemetryAggregation` | **PASS** | All telemetry aggregates compute accurately |

---

## 4. Credit Conversion Funnel & UX Invariants

### Funnel Steps
1. **User attempts tool action:** Client calls `/api/v1/studio/jobs/quote` to obtain a signed quote.
2. **Balance Check:** If credits < required price, [`InsufficientCreditModal`](file:///Users/noppanan/new-api/web/src/features/studio/components/InsufficientCreditModal.tsx) opens.
3. **Telemetry Emitted:** `studio_insufficient_credit` logged with `tool_id`, `credits_needed`, and `credits_balance`.
4. **Top-up CTA:** User clicks "Top Up Credits" -> `studio_buy_credit_click` logged.
5. **State Preservation:** Studio stores safe metadata in `sessionStorage` (`toolId`, `templateId`, `params`). **Zero raw base64 media** is stored in client storage.
6. **Top-up Completed:** User completes checkout on `/wallet`. A return toast appears pointing directly back to Studio.
7. **Return to Studio:** On mount, `StudioPlayground` detects return context and logs `studio_purchase_return`.
8. **Explicit Requote & User Confirmation:**
   - The UI automatically refreshes the quote.
   - A banner displays: *"Credits added! Confirm generation with your new balance."*
   - **STRICT INVARIANT:** System does **NOT** auto-generate on payment callback. The user must explicitly press the confirmation button.
9. **Final Execution:** User clicks generate -> `studio_generation_after_purchase` logged -> job executed.

---

## 5. Telemetry & Admin Observability

The admin telemetry endpoint (`GET /api/v1/studio/admin/telemetry`) aggregates real-time studio financial and conversion metrics:

```json
{
  "success": true,
  "data": {
    "studio_users": 1,
    "jobs": 2,
    "succeeded_total": 1,
    "failed_total": 1,
    "refunds": 1,
    "credits_spent": 5,
    "insufficient_credit_events": 1,
    "buy_credit_clicks": 1,
    "studio_originated_topups": 1,
    "generation_after_purchases": 1,
    "provider_cost": 0.025,
    "sell_value": 0.05,
    "gross_profit": 0.025,
    "gross_margin": 50.0
  }
}
```

### Privacy & Data Safety Guarantees
- Prompts, outputs, and user asset URLs are **never** logged to analytics or telemetry.
- Telemetry events only store anonymized counts, tool identifiers, and credit delta integers.

---

## 6. SEO Tool Landing Pages & Sitemap

Each activated tool features a dedicated server-rendered landing page at `/tools/:slug` with:
- Schema.org `WebApplication` and `SoftwareApplication` JSON-LD structured data.
- OpenGraph and Twitter card social meta tags.
- Detailed "How It Works" 3-step workflow guide.
- Common use cases, target audience, and Credit pricing transparency.
- Direct cross-links to related active tools.
- Primary CTA taking users directly into the pre-configured Playground.

All 6 routes are registered in the authoritative sitemap generator in [`service/news_seo.go`](file:///Users/noppanan/new-api/service/news_seo.go):
- `/tools/background-remove`
- `/tools/image-upscale`
- `/tools/image-generator`
- `/tools/product-photo`
- `/tools/object-eraser`
- `/tools/image-extend`

---

## 7. Operational Readiness Checklist

- [x] Go unit and integration tests passing (`service/studio_image_mvp_test.go`)
- [x] Full Go binary compilation passes cleanly (`go build -o /dev/null .`)
- [x] Web frontend builds with zero TypeScript/Vite errors (`npm run build`)
- [x] GORM boolean default tag fix applied (`IsEnabled`, `IsPublic`)
- [x] Credit conversion attribution funnel verified
- [x] No auto-generate after payment invariant verified
- [x] Privacy and security invariants verified
- [x] SEO tool landing pages and XML sitemap verified
