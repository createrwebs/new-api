# TORA STUDIO — QUEUE 3A REAL IMAGE EXPANSION FORENSIC REPORT

**Document**: `docs/ai/TORA_STUDIO_QUEUE_3A_REAL_IMAGE_EXPANSION.md`  
**Standard**: `MOCK ≠ LIVE`, `TEST ≠ PRODUCTION`, `DOCUMENTATION ≠ IMPLEMENTATION`  
**Execution Timestamp**: 2026-10-08T08:15:00+07:00  
**Target Repository**: `/Users/noppanan/new-api`  
**Git Branch**: `feat/formobile`  
**Production Host**: AWS EC2 `51.20.174.90` (alias `saascover-api`)  
**Production Gateway**: `https://www.toraapi.com`  

---

## 1. Executive Summary

Queue 3A expands Tora Studio from a single live tool (`image-generate` FAST) into a four-tool real image toolbox powered by the verified **WaveSpeed** provider.

### Core Acceptance Achievements:
1. **Three Additional Real Tools Verified**:
   - `image-upscale` (WaveSpeed Image Upscaler 4K) -> **ACTIVE**
   - `background-remove` (WaveSpeed Image Background Remover) -> **ACTIVE**
   - `product-photo` (WaveSpeed Flux Kontext Dev) -> **ACTIVE**
2. **Zero Mock Substitutions**: Every activation was grounded in a live paid prediction submitted to `https://api.wavespeed.ai/api/v3/predictions`.
3. **Single Tora Wallet Integrity**:
   - Pre-consumption from single `User.Quota` wallet table.
   - Successful settlement upon output generation.
   - Idempotent replay checks returning HTTP 201 without double charging or duplicate provider predictions.
4. **Economic Margin Floor Protection**:
   - All 4 active tools operate between **70.0% and 80.0%** gross margin, comfortably above the mandatory $\ge 60.0\%$ floor.
5. **Real Provider Spend Within Budget**:
   - Total provider cost incurred across all 3 canaries: **$0.0390 USD**, safely within the strict $\le \$0.1500\text{ USD}$ threshold.

---

## 2. Forensic Canary Details by Tool

### Tool #1: `image-upscale`
- **Logical Tool**: `image-upscale`
- **Route ID**: `ws-upscaler`
- **Provider**: `wavespeed`
- **Provider Model ID**: `wavespeed-ai/image-upscaler`
- **Dynamic Quote (COGS)**: `$0.0100 USD` (obtained via `POST /api/v3/model/price`)
- **Retail Economics**:
  - Sell Price: 25 Tora Credits = 25,000 Quota = `$0.0500 USD`
  - Gross Margin: **80.0%** ($(\$0.0500 - \$0.0100) / \$0.0500$)
- **Canary Job Execution**:
  - Tora Job ID: `job_1791420961_70c6a5be`
  - WaveSpeed Job ID: `490568d2d1f6474ea28388ca36dd5602`
  - Input: 128x128 test image URL
  - Parameters: `{"image": "...", "target_resolution": "4k"}`
- **Wallet Ledger**:
  - Pre-consumed: `25,000 Quota` ($95,000 \to 70,000$)
  - Settled: `25,000 Quota` (final balance: $70,000$)
- **Asset Ingestion & Forensic Verification**:
  - Output URL: `https://cdn.cachegalaxy.com/output/311d0a62-40ab-48d4-b5ee-6bd29204435f-u2_d1898347-1d06-42b8-afbe-b605131f5c63.jpeg`
  - MIME Type: `image/jpeg`
  - Content Length: 1,013,538 bytes
  - Resolution: **4096 x 4096** (successfully upscaled from 128x128)
  - SHA-256: `0be0f90a5c24ead149975b88b7ec0e3d9dcff157cefce1faa05954dc9f85b138`
  - Database Record: `studio_assets` row `asset_1791421008_490568d2`
- **Idempotency Proof**:
  - Replayed identical `idempotency_key`
  - Result: HTTP 201 Created with cached job payload, 0 new upstream calls, 0 quota deductions.

---

### Tool #2: `background-remove`
- **Logical Tool**: `background-remove`
- **Route ID**: `ws-birefnet`
- **Provider**: `wavespeed`
- **Provider Model ID**: `wavespeed-ai/image-background-remover`
- **Dynamic Quote (COGS)**: `$0.0040 USD` (obtained via `POST /api/v3/model/price`)
- **Retail Economics**:
  - Sell Price: 10 Tora Credits = 10,000 Quota = `$0.0200 USD`
  - Gross Margin: **80.0%** ($(\$0.0200 - \$0.0040) / \$0.0200$)
- **Canary Job Execution**:
  - Tora Job ID: `job_1791421084_ce23bcda`
  - WaveSpeed Job ID: `c179ece3c0a14d0eba5e04c55c484dbd`
  - Input: 128x128 test image URL
  - Parameters: `{"image": "..."}`
- **Wallet Ledger**:
  - Pre-consumed: `10,000 Quota` ($70,000 \to 60,000$)
  - Settled: `10,000 Quota` (final balance: $60,000$)
- **Asset Ingestion & Forensic Verification**:
  - Output URL: `https://cdn.cachegalaxy.com/output/7b2e3134-7f4c-4e92-bf86-966805bd5755-u1_109db10e-3878-4769-b205-38293f1e3690.png`
  - MIME Type: `image/png`
  - Content Length: 1,837 bytes
  - Resolution: **128 x 128**
  - Transparency Verification: Verified Truecolor Alpha (`color_type=6`, `has_alpha=True`), confirming clean alpha cutout.
  - SHA-256: `0d650fdca16562ed4334af1d7423d15945022907b1a44aaa1af787fc4f417e5a`
  - Database Record: `studio_assets` row `asset_1791421122_c179ece3`
- **Idempotency Proof**:
  - Replayed identical `idempotency_key`
  - Result: HTTP 201 Created with existing job payload, 0 duplicate charges.

---

### Tool #3: `product-photo`
- **Logical Tool**: `product-photo`
- **Route ID**: `ws-product-flux`
- **Provider**: `wavespeed`
- **Provider Model ID**: `wavespeed-ai/flux-kontext-dev`
- **Dynamic Quote (COGS)**: `$0.0250 USD` (obtained via `POST /api/v3/model/price`)
- **Retail Economics**:
  - Sell Price: 50 Tora Credits = 50,000 Quota = `$0.1000 USD`
  - Gross Margin: **75.0%** ($(\$0.1000 - \$0.0250) / \$0.1000$)
- **Canary Job Execution**:
  - Tora Job ID: `job_1791421293_d7c12149`
  - WaveSpeed Job ID: `2705db93da98431d8789576e7c6dbe66`
  - Input Product Asset: `https://www.toraapi.com/api/studio/assets/tora_canary_product_bottle_512x512.png` (512x512 green cosmetic dropper bottle PNG)
  - Parameters: `{"image": "...", "prompt": "Professional commercial product photography of this cosmetic dropper bottle standing on an organic stone pedestal..."}`
- **Wallet Ledger**:
  - Pre-consumed: `50,000 Quota` ($60,000 \to 10,000$)
  - Settled: `50,000 Quota` (final balance: $10,000$)
- **Asset Ingestion & Forensic Verification**:
  - Output URL: `https://cdn.cachegalaxy.com/output/8af9a9de-8661-47df-bfb0-47a7c0ab5d49-u2_19787d41-1acd-42c4-8f4e-f0f6f66850d7.jpeg`
  - MIME Type: `image/jpeg`
  - Content Length: 171,008 bytes
  - Resolution: **1024 x 1024**
  - SHA-256: `848ef599c6ce9ad6ea04a47087d61a436490876e6f0878820d4b5401bab59e4d`
  - Visual Fidelity Check: Green bottle geometry and dropper cap accurately preserved; photorealistic studio lighting and natural shadows placed on stone podium.
  - Database Record: `studio_assets` row `asset_1791421322_2705db93`
- **Idempotency Proof**:
  - Replayed identical `idempotency_key`
  - Result: HTTP 201 Created with existing job payload, 0 duplicate charges.

---

## 3. Financial & Budget Reconciliations

| Step | Operation | Upstream Cost | Remaining Balance | Margin |
| :--- | :--- | :--- | :--- | :--- |
| 0 | Pre-Session Balance Check | - | **$0.5194 USD** | - |
| 1 | `image-upscale` Canary | $0.0100 USD | $0.5094 USD | 80.0% |
| 2 | `background-remove` Canary | $0.0040 USD | $0.5054 USD | 80.0% |
| 3 | `product-photo` Canary | $0.0250 USD | $0.4804 USD | 75.0% |
| **Total** | **Queue 3A Real Executions** | **$0.0390 USD** | **$0.4804 USD** | **77.5% Avg** |

Total real provider expenditure for Queue 3A is **$0.0390 USD**, which complies with the hard constraint of remaining under $\$0.15\text{ USD}$.

---

## 4. Catalog Truth Status in Production

Following these successful canaries, the public tool definitions in `studio_tool_definitions` reflect authoritative reality:
- `image-generate` (FAST): **ACTIVE**
- `image-upscale`: **ACTIVE**
- `background-remove`: **ACTIVE**
- `product-photo`: **ACTIVE**
- All 8 remaining tools: truthfully maintained as **DISABLED** / **COMING_SOON**.
