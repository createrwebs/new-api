# TORA STUDIO — FINAL ACCEPTANCE MATRIX (QUEUES 1–8)

**Document**: `docs/ai/TORA_STUDIO_FINAL_ACCEPTANCE_MATRIX.md`  
**Audit Standard**: `MOCK ≠ LIVE`, `TEST ≠ PRODUCTION`, `DOCUMENTATION ≠ IMPLEMENTATION`  
**Timestamp**: 2026-10-07T08:42:00+07:00  
**Repository**: `/Users/noppanan/new-api`  
**Branch**: `feat/formobile` (`e551ce467`)  
**Production Gateway**: `https://www.toraapi.com`  
**Overall Acceptance Status**: `FINAL STATUS: TORA STUDIO ACCEPTANCE PARTIAL — OPERATOR PROVIDER ACTION REQUIRED`

---

## 1. Master Acceptance Matrix

| QUEUE | REQUIREMENT | CLAIMED_STATUS | CODE_EVIDENCE | TEST_EVIDENCE | PRODUCTION_EVIDENCE | REAL_PROVIDER_EVIDENCE | BILLING_EVIDENCE | SECURITY_EVIDENCE | FINAL_VERDICT | REMEDIATION |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **QUEUE 1** | **Reference Harvest & Hardening**<br>7 reference repos, pinned SHAs, licenses, server quotes, pricing snapshots, variable costs, margin floor $\ge 60\%$, SSRF, idempotency, restart recovery. | VERIFIED | `TORA_STUDIO_PATTERN_HARVEST.md`, `/Users/noppanan/tora-studio-lab/references/` (7 repos with git SHAs), `service/studio_pricing.go`, `service/studio_security.go`. | `service/studio_canary_test.go` (`TestQueue2_PreCanaryGates` passes 7/7 gates), `service/studio_service_test.go` (`TestStudioPricing_*`). | Server quote endpoint `POST /api/studio/quote` active, 15-min TTL, dynamic calculation. | N/A (Architecture & hardening stage). | Strict margin floor $\ge 60\%$, ceiling rounding, immutable snapshot `StudioPricingSnapshot` with versioning. | SSRF blocked at dial time (`SafeHTTPClient`), magic bytes verified (JPEG, PNG, WebP, GIF, MP4, WebM). | **VERIFIED** | None required. All 7 harvest repos and hardening gates verified in code & tests. |
| **QUEUE 2** | **Real Provider Canary & Credit Loop**<br>Single Tora wallet pre-consume, provider dispatch, atomic settlement/refund, real network canary. | LIVE | `service/studio_fal.go`, `model/wallet_pre_consume.go`, `service/studio_service.go`. | `service/studio_canary_test.go`, `service/studio_service_test.go` (`TestStudioService_SubmitJob_Success`). | Endpoints `/api/v1/studio/jobs`, `/api/v1/studio/webhooks/fal` live on `https://www.toraapi.com`. | **OPERATOR_BLOCKED**: `FAL_KEY` not configured on local workstation or production container. Zero fake mock claims. | Single wallet invariant proven: `User.Quota` only, 0 separate Studio balance tables. `wallet_after = wallet_before - settled_quota`. | Idempotency key unique index on `studio_tool_jobs`, dual-resolution webhook/poll race safe. | **OPERATOR_BLOCKED** | Operator must provision `FAL_KEY` into production environment to execute the first paid canary. |
| **QUEUE 3** | **Image Studio MVP**<br>Public image tools (`background-remove`, `image-upscale`, `image-generate`), UI, SEO pages, credit modal, history, remix. | ACTIVE | `web/src/features/studio/`, `service/studio_seed.go`, `controller/studio.go`. | `service/studio_image_mvp_test.go` (12/12 unit tests pass), `npm run build` passes with 0 errors. | Routes `/studio`, `/tools/background-remove`, `/tools/image-upscaler` render with Flaq-grade UI. | **OPERATOR_BLOCKED**: Fal/Replicate keys missing; API dynamically reports `OPERATOR_BLOCKED` to UI. | Insufficient credit modal triggers Tora wallet flow, returns without auto-submit, refreshes quote. | LocalStorage state has 2-hour TTL, no raw base64 or video blobs stored in browser. | **PARTIALLY_VERIFIED** | Implementation and UI verified. Live provider invocation remains `OPERATOR_BLOCKED` until API keys added. |
| **QUEUE 4** | **Product Studio & E-Commerce**<br>Product photo workflow, 5 marketplace presets (Shopee, Lazada, IG 4:5, Story 9:16, TikTok), 8 visual styles, pack size (1 vs 4), multi-reference handling. | VERIFIED | `web/src/features/studio/components/StudioPlayground.tsx`, `service/studio_seed.go` (`tpl-prod-*`), `service/studio_pricing.go`. | `service/studio_product_photo_test.go` (9/9 tests pass). | Studio playground includes dedicated E-commerce preset HUD, aspect ratios, and visual template selector. | **OPERATOR_BLOCKED**: Live rendering requires upstream provider keys. | Pack size pricing: 1 pack (50 credits), 4-pack (200 credits). Margin $\ge 65\%$. | Direct logo diffusion prohibited; reference image URL validated against SSRF. | **PARTIALLY_VERIFIED** | Code, UI, presets, and billing fully verified. Live commercial provider canary blocked on `FAL_KEY`. |
| **QUEUE 5** | **Video Foundation & Image-to-Video**<br>Conservative video controls: max 5s, 720p/1080p, no base64 JSON, concurrency guard (1 regular, 2 admin), daily spend guard ($50/day), asset pipeline. | CLAIMED LIVE *(Corrected)* | `service/studio_service.go`, `service/studio_pricing.go`, `service/studio_asset.go`, `controller/studio.go`. | `service/studio_video_test.go` (8/8 tests pass), `service/studio_asset_test.go` (3/3 tests pass). | Video parameters and 5s duration limit enforced on server. | **OPERATOR_BLOCKED**: Previous claim of `IMAGE-TO-VIDEO LIVE` was based on test fixtures. Live Wan 2.2 call requires `FAL_KEY`. | Video quote TTL = 5 mins, 720p (125 credits), 1080p (157 credits), audio (+15 credits). Margin $\ge 68\%$. | Disk exhaustion guard: 500MB per-user quota, 5GB global quota, 24h asset TTL, automatic cleanup worker. | **PARTIALLY_VERIFIED** | Implementation, guards, and asset pipeline fully verified. Real video canary marked `OPERATOR_BLOCKED`. |
| **QUEUE 6** | **Creator Assistant & ToolPlan**<br>Natural language Thai intent resolution, ToolPlan DAG, authoritative per-step quotes, confirmation, step-level settlement/refund. | WORKFLOW VERIFIED | `service/studio_assistant.go`, `controller/studio_assistant.go`, `model/studio_workflow.go`. | `service/studio_assistant_test.go` (8/8 tests pass). | `/assistant` and `/studio` assistant tab route active in frontend. | **OPERATOR_BLOCKED**: Real multi-step upstream E2E pending live provider keys. | Step 1 & 2 succeed, Step 3 fails: Steps 1 & 2 remain charged, Step 3 refunded. Never calls provider directly. | Safety-gated high-risk tools (`face-swap`, `talking-avatar`) require explicit consent and cannot auto-execute. | **PARTIALLY_VERIFIED** | Implementation fully verified. Live multi-step execution classified as `REAL WORKFLOW E2E PENDING (OPERATOR_BLOCKED)`. |
| **QUEUE 7** | **Multi-Provider Routing**<br>Fal + Replicate adapters, Quality Tiers (`FAST`, `QUALITY`, `PREMIUM`), scoring formula, safe fallback without duplicate billing. | ROUTING VERIFIED | `service/studio_router.go`, `service/studio_replicate.go`, `service/studio_fal.go`. | `service/studio_router_test.go` (7/7 tests pass). | Server router selects provider dynamically; exposes quality tiers. | **OPERATOR_BLOCKED**: Neither `FAL_KEY` nor `REPLICATE_API_TOKEN` is provisioned. | Single Tora wallet deduction regardless of provider chosen; cost tracked per provider. | Safe fallback prohibited on ambiguous timeouts (`ErrProviderAmbiguous`) to eliminate double charges. | **PARTIALLY_VERIFIED** | Both adapters implemented and tested. Live multi-provider routing marked `OPERATOR_BLOCKED` until credentials supplied. |
| **QUEUE 8** | **Scale Economics & Self-Host Decision**<br>Rolling metrics calculation, full cost model (compute, idle, egress, storage, ops, SLA), open-source candidate mappings. | DATA MATURE *(Corrected)* | `service/studio_scale.go`, `service/studio_scale_test.go`. | `service/studio_scale_test.go` (6/6 tests pass). | Admin telemetry endpoints configured. | Real production usage volume is currently low/nascent. | Calculates sell value, gross profit, margin, and compares against true self-host cost. | Hard invariant enforced: `NEW_SERVER_COUNT = 0`, `NEW_GPU_SERVER_COUNT = 0`. | **SYSTEM IMPLEMENTED — DATA NOT MATURE** | Maintain current API routing. Gather $\ge 1,000$ real jobs/month before triggering any self-host migration. |

---

## 2. Invariant & Security Verification Checklist

| Security / Commercial Invariant | Verification Method | Code & Runtime Evidence | Verdict |
| :--- | :--- | :--- | :--- |
| **Single Tora Wallet (Zero Secondary Wallets)** | Code audit of `model/wallet_pre_consume.go` & `model/quota_reserve.go` | All reservations deduct from `User.Quota`. Studio-specific balance tables = 0. | **VERIFIED** |
| **Profitability Floor ($\ge 60\%$)** | `PricingEngine.ValidateProfitability` | Any quote yielding margin $<60\%$ is strictly bumped or rejected with `ErrMarginBelowFloor`. | **VERIFIED** |
| **Pricing Snapshot Immutability** | Audit of `StudioPricingSnapshot` | Preserves `pricing_version`, `quote_id`, `quoted_at`, `expires_at`, `provider`, `model`, `cogs`, `margin`, `sell_value`, `credits`. | **VERIFIED** |
| **Idempotency** | Database constraint on `studio_tool_jobs` | `idempotency_key` unique index prevents duplicate charges; returns existing job. | **VERIFIED** |
| **SSRF & Metadata Defense** | `service/studio_security.go` | Blocks `169.254.169.254`, `127.0.0.1`, RFC1918, IPv6 loops, and resolves IP at dial time to stop DNS rebinding. | **VERIFIED** |
| **Media Magic Bytes Validation** | `ValidateMagicBytes` | Validates file header for JPEG, PNG, WebP, GIF, MP4, WebM; rejects disguised executables. | **VERIFIED** |
| **Asset Storage & Disk Protection** | `service/studio_asset.go` | Max 500MB per user, max 5GB global, 24h input TTL, automated background cleanup worker. | **VERIFIED** |
| **No Video Base64 JSON** | `service/studio_service.go` | Rejects payloads starting with `data:` for video tools; requires dedicated upload endpoint. | **VERIFIED** |
| **Restart Recovery** | `ReconcileStaleJobs` | Orphaned `RESERVED` jobs are safely refunded; orphaned `PROCESSING` jobs polled without double-charge. | **VERIFIED** |
| **Webhook / Poll Race Safety** | `TestQueue2_PreCanaryGates` Gate 4 | Concurrent webhook delivery and client polling transitions state and settles wallet exactly once. | **VERIFIED** |
| **Zero New Servers Invariant** | Infrastructure inspection | `NEW_SERVER_COUNT = 0`, `NEW_GPU_SERVER_COUNT = 0`. No PyTorch/ComfyUI deployed to production. | **VERIFIED** |

---

## 3. Tool Truthfulness Matrix

| Tool ID | Display Name | Category | Base Credits | Quota Cost | Implementation Status | Live Upstream Provider | Public Catalog Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `background-remove` | ลบพื้นหลังอัจฉริยะ (BiRefNet) | utility | 10 | 10,000 | Code/Tests Verified | Fal / Replicate | `OPERATOR_BLOCKED` (Awaiting API Key) |
| `image-upscale` | ขยายภาพคมชัด 4K (Clarity) | utility | 18–25 | 18,000–25,000 | Code/Tests Verified | Fal / Replicate | `OPERATOR_BLOCKED` (Awaiting API Key) |
| `image-generate` | สร้างภาพ AI อัจฉริยะ (Flux.1) | image | 8–32 | 8,000–32,000 | Code/Tests Verified | Fal / Replicate | `OPERATOR_BLOCKED` (Awaiting API Key) |
| `product-photo` | สตูดิโอถ่ายภาพสินค้า (E-Commerce) | product | 50–200 | 50,000–200,000 | Code/Tests Verified | Fal | `OPERATOR_BLOCKED` (Awaiting API Key) |
| `image-to-video` | แปลงภาพเป็นวิดีโอ (Wan 2.2) | video | 125–157 | 125,000–157,000 | Code/Tests Verified | Fal / Wan 2.2 | `OPERATOR_BLOCKED` (Awaiting API Key) |
| `creator-assistant` | ผู้ช่วยสร้างสรรค์ (ToolPlan) | workflow | Dynamic | Dynamic | Code/Tests Verified | Internal DAG -> Tools | `OPERATOR_BLOCKED` (Routes to underlying tools) |
| `image-extend` | ขยายขอบเขตภาพ (Outpaint) | image | 30 | 30,000 | Code/Tests Verified | Fal | `BETA` / `OPERATOR_BLOCKED` |
| `object-erase` | ลบวัตถุและทำความสะอาดภาพ | utility | 30 | 30,000 | Code/Tests Verified | Fal | `BETA` / `OPERATOR_BLOCKED` |
| `text-to-video` | สร้างวิดีโอจากข้อความ | video | 125 | 125,000 | Seeded | N/A | `COMING_SOON` / `DISABLED` |
| `lip-sync` | ลิปซิงก์เสียงและขยับปาก | video | 75 | 75,000 | Seeded | N/A | `COMING_SOON` / `DISABLED` |
| `face-swap` | สลับใบหน้าระดับโปร | image | 40 | 40,000 | Seeded (Gated) | N/A | `COMING_SOON` / `DISABLED` |
| `talking-avatar` | อวตารผู้ประกาศข่าว | video | 100 | 100,000 | Seeded (Gated) | N/A | `COMING_SOON` / `DISABLED` |
| `video-upscale` | เพิ่มความคมชัดวิดีโอ | video | 100 | 100,000 | Seeded | N/A | `COMING_SOON` / `DISABLED` |
