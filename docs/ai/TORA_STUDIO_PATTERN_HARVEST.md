# TORA AI — STUDIO REFERENCE HARVEST & PATTERN ANALYSIS
**Document**: `docs/ai/TORA_STUDIO_PATTERN_HARVEST.md`  
**Date**: 2026-10-07  
**Status**: APPROVED & APPLIED  
**Lab Directory**: `/Users/noppanan/tora-studio-lab`

---

## 1. Cloned Reference Repositories & Pinned SHAs

All reference repositories were cloned into the isolated lab directory `/Users/noppanan/tora-studio-lab/references/` outside the Tora new-api git tree.

| Repository | Upstream URL | Pinned Commit SHA | Last Activity Date | Primary Language / Runtime | Code License |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **gitroomhq/agent-media** | `https://github.com/gitroomhq/agent-media.git` | `817f28477ca80a0e1be8270ec9616a1b8bafcd78` | 2026-10-05 | Markdown / JSON / Node.js plugins | Apache 2.0 |
| **SamurAIGPT/muapi-cli** | `https://github.com/SamurAIGPT/muapi-cli.git` | `c4057aab8006e75ebf41a86cc5aa27901f30bf6e` | 2026-10-01 | Python 3.10+ / CLI / MCP | MIT |
| **two-71/studio** | `https://github.com/two-71/studio.git` | `e2f24f414f0cc6401e269dd98c7f6e07be8d0d98` | 2026-08-19 | TypeScript / React / Drizzle / Next.js | MIT |
| **SamurAIGPT/amazon-product-studio** | `https://github.com/SamurAIGPT/amazon-product-studio.git` | `30c25c1893f7f1f5823c21acbdb9396933a7f6bf` | 2026-09-02 | JavaScript / Next.js / Prisma | MIT |
| **Comfy-Org/ComfyUI** | `https://github.com/Comfy-Org/ComfyUI.git` | `6a8dcf514bc02a29d1b20257cb0fc9bfed3223e1` | 2026-10-06 | Python 3.10+ / PyTorch | GPLv3 |
| **gitroomhq/postiz-app** | `https://github.com/gitroomhq/postiz-app.git` | `22c034188092be11576190a05a82563a53efed9a` | 2026-10-06 | TypeScript / NestJS / Prisma / BullMQ | AGPLv3 |
| **wide-trace/open-higgsfield** | `https://github.com/wide-trace/open-higgsfield.git` | `b16a0efe4d7e2707b56f8ccb02387fd2a9d2eddf` | 2026-09-17 | TypeScript / Next.js 16 / React 19 | Private / Unlicensed |

---

## 2. Intellectual Property & License Safety Taxonomy

| Repository | Code License | Model Weight License | Dependency Licenses | Safety Classification | Tora Compliance Directive |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **agent-media** | Apache 2.0 | N/A (API wrapper) | Permissive npm | `SAFE_REUSE_WITH_ATTRIBUTION` | Architectural quote/asset flow patterns safe to adopt; attribute in NOTICE. |
| **muapi-cli** | MIT | N/A (API client) | httpx, click, rich | `SAFE_REUSE_WITH_ATTRIBUTION` | Schema introspection & normalization ideas safe to adopt; write native Go. |
| **two-71/studio** | MIT | Third-party LoRAs (SAFetensors) | Drizzle, Zod, AI SDK | `SAFE_REUSE_WITH_ATTRIBUTION` | Tool/Model config architecture & billing interface safe to adapt. |
| **amazon-product-studio** | MIT | N/A (API client) | Next.js, Prisma, Tailwind | `SAFE_REUSE_WITH_ATTRIBUTION` | E-commerce reference images & aspect ratio presets safe to adapt. |
| **ComfyUI** | GPLv3 | Community checkpoints/models | PyTorch, torchvision | `REFERENCE_ONLY` | **DO NOT COPY CODE**. Learn DAG execution and workflow JSON abstraction only. |
| **postiz-app** | AGPLv3 | N/A (Social platform) | NestJS, Redis, BullMQ | `REFERENCE_ONLY` | **DO NOT COPY AGPL CODE**. Learn provider retry & async dispatch mechanics only. |
| **open-higgsfield** | None (Private) | Hosted endpoints | TanStack, Vercel Blob | `REFERENCE_ONLY` | **DO NOT COPY CODE**. Learn IndexedDB client media caching and viewer UX. |

---

## 3. Pattern Harvest Matrix (22 Core Dimensions)

| Dimension | Pattern Description | Best Reference | Decision | Rationale & Tora Implementation Strategy |
| :--- | :--- | :--- | :--- | :--- |
| **1. Tool Registry** | Static metadata + dynamic capabilities | two-71/studio | **ADAPT** | Keep PostgreSQL-backed `StudioToolDefinition` in Go; add versioning and variable pricing inputs. |
| **2. Provider Catalog** | Separated logical tools from provider model IDs | agent-media & two-71 | **ADOPT** | Never expose provider model strings (`fal-ai/birefnet`) to frontend. Frontend uses `tool_id="background-remove"`. |
| **3. Templates** | Curated preset prompts, ratios, and negative prompts | amazon-product-studio | **ADAPT** | Expand `StudioToolTemplate` with e-commerce presets (Shopee 1:1, TikTok 9:16, Clean White, Luxury). |
| **4. Presets** | One-click aspect ratio and marketplace dimensions | amazon-product-studio | **ADOPT** | Add aspect ratio cards with visual icons (1:1, 9:16, 16:9, 4:5) in Studio UI. |
| **5. Parameter Schema** | JSON Schema validation per tool | agent-media | **ADOPT** | Strict JSON Schema validation on server before reserving wallet quota. |
| **6. Quote Before Run** | Server-authoritative quote endpoint with TTL | agent-media | **ADOPT** | Introduce `POST /api/studio/quote`. Client cannot invent or compute prices. |
| **7. Balance / Credits** | Single wallet pre-charge & settlement | Tora Commercial Invariant | **ADOPT** | STRICT INVARIANT: 1 Credit = 1,000 Quota. No secondary wallet. Balance checked before quote confirmation. |
| **8. Job State Machine** | Multi-phase state machine with audit events | Tora / two-71 | **ADAPT** | Maintain: CREATED -> RESERVED -> SUBMITTING -> SUBMITTED -> PROCESSING -> SUCCEEDED / FAILED / REFUNDED. |
| **9. Async Processing** | Non-blocking provider submission with task runner | two-71 / Postiz | **ADOPT** | Background worker polls provider queue or accepts webhooks without blocking HTTP handler. |
| **10. Webhooks** | Signed webhook callback URL for provider completion | amazon-product-studio | **ADAPT** | Dual-resolution: background polling runs alongside incoming webhooks with concurrency guard. |
| **11. Polling** | Exponential backoff polling with jitter | open-higgsfield | **ADAPT** | Poll provider queue with backoff (1s, 2s, 4s...) up to 300s timeout. |
| **12. Cancellation** | Cancel in-flight job if not yet committed by provider | two-71 | **ADAPT** | Support job cancel if provider supports cancellation; auto-refund reserved quota immediately. |
| **13. Gallery** | Personal generation history with filtering | two-71 & open-higgsfield | **ADAPT** | Studio History tab shows user's past generations, prompt, tool, timestamp, and download button. |
| **14. History** | Infinite scroll or paginated cards of generations | open-higgsfield | **ADOPT** | Virtualized / paginated cards with retry/remix buttons. |
| **15. Remix** | Pre-populate tool form with past generation's parameters | open-higgsfield | **ADOPT** | "Remix" button loads past job parameters into Composer without re-uploading. |
| **16. Upload UX** | Drag-and-drop with client validation & preview | agent-media | **ADOPT** | Validate image size (<15MB), MIME type, and magic bytes before uploading. Show thumbnail preview. |
| **17. Result UX** | Before/After slider comparison for utilities | amazon-product-studio | **ADAPT** | For background-remove and upscale, render interactive split before/after visual comparison. |
| **18. Agent Workflows** | Structured ToolPlan with multi-step DAG | agent-media | **ADAPT** | Prepare future `ToolPlan` (e.g. Remove BG -> Product Studio -> Video), quoting each step authoritatively. |
| **19. Chained Jobs** | Output of step N feeds input of step N+1 | ComfyUI / agent-media | **ADAPT** | Asset ID references passed downstream; avoid client roundtrip of media bytes. |
| **20. Mobile UX** | Responsive layout, bottom sheets, touch targets | open-higgsfield | **ADAPT** | Composer adapts to mobile viewport; sticky bottom action bar with clear credit display. |
| **21. Cost Reporting** | Dedicated table recording COGS and gross margin | Tora Economics | **ADOPT** | Record `StudioCostSnapshot` with Provider Cost, Charged Revenue, Margin USD, and Margin Percent. |
| **22. Asset Expiry** | Provider URLs expire in 24-48 hours | agent-media | **ADOPT** | Persist `expires_at` on assets; frontend warns user to download or export before expiration. |

---

## 4. Deep-Dive Repository Case Studies

### A. GitroomHQ Agent-Media
* **Key Finding**: Pure API-first agent wrapper. Enforces `quote` before generation. Generates vertical UGC video with strict duration-based pricing.
* **Asset Upload Flow**: Provides an isolated temporary upload link that expires in 24 hours. Prevents raw base64 from polluting conversation transcripts and payload sizes.
* **Tora Mapping**:
  * Agent quote -> `POST /api/studio/quote`
  * Upload panel -> `StudioAsset` storage endpoint with 24-hour expiration
  * Agent tools -> Tora `StudioToolDefinition` catalog

### B. Two-71 Studio
* **Key Finding**: Clean TypeScript architecture separating `StudioConfig`, `BillingProvider`, `StorageAdapter`, and `StudioModel`.
* **Billing Seam**: `BillingProvider` interface with `charge()`, `refund()`, `costFor()`, `getBalance()`. Over-quota requests immediately trigger native insufficient-funds error.
* **Tora Mapping**:
  * Tora adopts identical interface separation in Go: `BillingProvider` maps to `PreConsumeUserWallet` and `SettleUserWalletPreConsume`.
  * Preserves single-wallet invariant without any guest or dual-ledger leakage.

### C. Amazon Product Studio (MuAPI)
* **Key Finding**: Specializes in e-commerce product photos. Takes multi-reference images (`images_list` up to 14 images) to preserve product packaging while altering background scene.
* **Dual Resolution**: Submits with webhook URL, but runs client-side / server-side polling simultaneously. Handles concurrency gracefully with status check.
* **Tora Mapping**:
  * Product Photo tool in Tora will support multi-reference inputs and presets tailored for Thai e-commerce (Shopee, Lazada, TikTok Shop).
  * Webhook + Polling race prevention: First worker to transition to `SUCCEEDED` settles wallet quota atomically; second worker no-ops.

### D. MuAPI CLI
* **Key Finding**: Live OpenAPI specification introspection cached locally for 1 hour. Normalized types for image/video inputs across heterogeneous models.
* **Tora Mapping**:
  * Tora's logical tools remain provider-neutral. Logical tools expose canonical fields (`prompt`, `image_url`, `aspect_ratio`), which the adapter translates to provider-specific payloads.

### E. ComfyUI
* **Key Finding**: Node-based DAG workflow engine. Caching by node input hash.
* **Tora Mapping**:
  * Tora rejects deploying ComfyUI or self-hosting heavy GPU clusters at this stage (zero server expansion).
  * Adapts the DAG concept into future sequential `ToolPlan` orchestration.

### F. Postiz
* **Key Finding**: Enterprise-grade async job queue with rate limit cooldown and exponential backoff retry.
* **Tora Mapping**:
  * Adapts retry backoff strategy for provider status polling.
  * AGPL license respected: zero code copied.

### G. Open-Higgsfield
* **Key Finding**: Modern Next.js 16 / React 19 generation frontend with IndexedDB (`idb.ts`) client storage.
* **Tora Mapping**:
  * Eliminates localStorage media storage: Tora Studio stores only safe metadata (asset ID, tool, prompt) in localStorage with bounded TTL, avoiding quota exhaustion and privacy leaks.

---

## 5. Architectural Gap Analysis: Tora Studio vs. References

| Capability | Current Tora V1 | Reference Benchmark | Gap Level | Planned Hardening Action |
| :--- | :--- | :--- | :--- | :--- |
| **Credit Rounding** | Truncation / standard float | Minimum margin ceiling | High | Use `math.Ceil` to next integer Credit; enforce 60% gross margin floor. |
| **Pre-Submission Quote** | Frontend calculated | Server-authoritative quote | Critical | Implement `POST /api/studio/quote` with 15-minute TTL. |
| **Pricing Snapshot** | Partial in CostSnapshot | Full snapshot on Job record | High | Embed `PricingSnapshot` directly in `StudioToolJob` input/audit. |
| **Variable Cost** | Flat per-tool cost | Parameter-aware cost | Medium | Calculate price based on duration, resolution, quality. |
| **Asset Pipeline** | Inline base64 | Asset ID / Pre-authenticated URL | High | Limit base64 to <15MB images; disable video base64 entirely. |
| **LocalStorage Safety** | Could store pending data | IndexedDB or metadata only | High | Strip base64 and sensitive data from localStorage pending state. |
| **Object Erase Branding** | "watermark-remove" | "Object Eraser / Cleanup" | Medium | Update UI title, slug, and descriptions; add ownership check. |
| **SSRF Security** | Hostname check | Redirect & DNS rebinding guard | High | Validate every redirect hop, block AWS metadata `169.254.169.254`. |
| **Double Submission** | Idempotency key exists | Strict ACID lock & idempotent return | Medium | Return existing job on duplicate key without re-reserving quota. |
