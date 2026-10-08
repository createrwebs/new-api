# TORA STUDIO — IMAGE ACTIVATION MATRIX (QUEUE 3A)

**Document**: `docs/ai/TORA_STUDIO_IMAGE_ACTIVATION_MATRIX.md`  
**Audit Standard**: `MOCK ≠ LIVE`, `TEST ≠ PRODUCTION`, `DOCUMENTATION ≠ IMPLEMENTATION`  
**Generated At**: 2026-10-08T08:15:00+07:00  
**Repository**: `/Users/noppanan/new-api`  
**Branch**: `feat/formobile`  
**Production Gateway**: `https://www.toraapi.com`  
**Active Live Provider**: WaveSpeedAI (`WAVESPEED_API_KEY`)  

---

## 1. Real Image Toolbox Activation Matrix

| Tool | Route | Provider Model | Contract | Dynamic Price | Tora Credits | Quota Cost | Retail USD | Margin | Real Canary Job | Upstream Job ID | Wallet Deduction | Asset Evidence | Idempotency | UI Status | Public Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`image-generate` (FAST)** | `ws-flux-schnell` | `wavespeed-ai/flux-schnell` | `POST /api/v3/predictions`<br>`{"prompt": "...", "size": "1024*1024"}` | $0.0030 USD | 5 Credits | 5,000 | $0.0100 USD | **70.0%** | `job_1791419088_99a8b1c2` | `a68bf4bb3dd948759d41b57e72aa0692` | Pre-consume 5,000 Quota<br>Settled 5,000 Quota | JPEG, 93,175 bytes, 1024x1024<br>SHA256: `5376f982...` | Verified: 201 Created, 0 dup charge | Integrated (`/studio`, `/tools/image-generator`) | **ACTIVE** |
| **`image-upscale`** | `ws-upscaler` | `wavespeed-ai/image-upscaler` | `POST /api/v3/predictions`<br>`{"image": "...", "target_resolution": "4k"}` | $0.0100 USD | 25 Credits | 25,000 | $0.0500 USD | **80.0%** | `job_1791420961_70c6a5be` | `490568d2d1f6474ea28388ca36dd5602` | Pre-consume 25,000 Quota<br>Settled 25,000 Quota | JPEG, 1,013,538 bytes, 4096x4096<br>SHA256: `0be0f90a...` | Verified: 201 Created, 0 dup charge | Integrated (`/studio`, `/tools/image-upscaler`) | **ACTIVE** |
| **`background-remove`** | `ws-birefnet` | `wavespeed-ai/image-background-remover` | `POST /api/v3/predictions`<br>`{"image": "..."}` | $0.0040 USD | 10 Credits | 10,000 | $0.0200 USD | **80.0%** | `job_1791421084_ce23bcda` | `c179ece3c0a14d0eba5e04c55c484dbd` | Pre-consume 10,000 Quota<br>Settled 10,000 Quota | PNG Truecolor Alpha, 1,837 bytes, 128x128<br>SHA256: `0d650fdc...` | Verified: 201 Created, 0 dup charge | Integrated (`/studio`, `/tools/background-remove`) | **ACTIVE** |
| **`product-photo`** | `ws-product-flux` | `wavespeed-ai/flux-kontext-dev` | `POST /api/v3/predictions`<br>`{"image": "...", "prompt": "..."}` | $0.0250 USD | 50 Credits | 50,000 | $0.1000 USD | **75.0%** | `job_1791421293_d7c12149` | `2705db93da98431d8789576e7c6dbe66` | Pre-consume 50,000 Quota<br>Settled 50,000 Quota | JPEG, 171,008 bytes, 1024x1024<br>SHA256: `848ef599...` | Verified: 201 Created, 0 dup charge | Integrated (`/studio`, `/tools/product-photo`) | **ACTIVE** |

---

## 2. Inactive / Gated Tools Catalog Truth

| Tool | Category | Planned Route | Provider Model | Status | Rationale |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `image-generate` (QUALITY) | image | `ws-flux-dev` | `wavespeed-ai/flux-dev` | `CONTRACT_VERIFIED` | Route verified in WaveSpeed catalog; awaits dedicated quality tier promotion canary. |
| `image-extend` | editing | `fal-flux-fill` | `fal-ai/flux-fill` | `DISABLED` | fal is `BILLING_BLOCKED`; pending WaveSpeed or KIE inpainting model onboarding. |
| `object-erase` | editing | `ws-image-eraser` | `wavespeed-ai/image-eraser` | `DISABLED` | Discovered in WaveSpeed catalog ($0.025 USD); awaits inpainting canary. |
| `image-to-video` | video | `fal-wan-2.2` | `wan-video/wan-2.2` | `DISABLED` | Video execution strictly prohibited in Queue 3A per governance. |
| `text-to-video` | video | - | - | `COMING_SOON` | Video execution strictly prohibited. |
| `lip-sync` | video | - | - | `COMING_SOON` | Video execution strictly prohibited. |
| `face-swap` | utility | - | - | `COMING_SOON` | Safety-gated high risk tool. |
| `talking-avatar` | video | - | - | `COMING_SOON` | Video execution strictly prohibited. |
| `video-upscale` | video | - | - | `COMING_SOON` | Video execution strictly prohibited. |

---

## 3. Financial Invariant Proof

- **Canonical Exchange Rate**:
  $$1\text{ USD} = 500,000\text{ Quota} = 500\text{ Tora Credits}$$
  $$1\text{ Tora Credit} = 1,000\text{ Quota} = \$0.0020\text{ USD}$$
- **Gross Margin Formula**:
  $$\text{Margin} = \frac{\text{Retail USD} - \text{COGS USD}}{\text{Retail USD}} \times 100\%$$
- **Minimum Gross Margin Floor**: $\ge 60.0\%$
  - `image-generate` (FAST): $(0.0100 - 0.0030) / 0.0100 = 70.0\%$ (Passes floor)
  - `image-upscale`: $(0.0500 - 0.0100) / 0.0500 = 80.0\%$ (Passes floor)
  - `background-remove`: $(0.0200 - 0.0040) / 0.0200 = 80.0\%$ (Passes floor)
  - `product-photo`: $(0.1000 - 0.0250) / 0.1000 = 75.0\%$ (Passes floor)
- **Real Provider Spend Accounting**:
  - Starting WaveSpeed balance: `$0.5194 USD`
  - Balance after upscale: `$0.5094 USD` (cost: `$0.0100 USD`)
  - Balance after bg-remove: `$0.5054 USD` (cost: `$0.0040 USD`)
  - Balance after product-photo: `$0.4804 USD` (cost: `$0.0250 USD`)
  - **Total Session Spend**: **$0.0390 USD** (Strictly within the $\le \$0.15\text{ USD}$ cap).
