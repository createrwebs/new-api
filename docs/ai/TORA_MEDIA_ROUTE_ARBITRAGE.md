# =====================================================================
# TORA STUDIO — MULTI-PROVIDER MEDIA ROUTE ARBITRAGE
# FORENSIC COST COMPARISON, CONTRACT CONFIDENCE & MARGIN FLOOR GOVERNANCE
# =====================================================================

> **Monetary Anchor**: 
> - Canonical conversion: $1\text{ USD} = 500,000\text{ Quota} = 500\text{ Tora Credits}$
> - Credit equivalence: $1\text{ Tora Credit} = 1,000\text{ Quota} = \$0.0020\text{ USD}$
> - Margin invariant: Minimum Gross Margin $\ge 60.0\%$ on all customer-facing routes.

---

## 1. Route Arbitrage Matrix Across Providers

| Logical Tool | Quality Tier | Provider | Route ID | Verified Upstream Model ID | Cost Basis | Upstream COGS (USD) | Retail Credits | Retail Price (USD) | Gross Profit (USD) | Gross Margin (%) | Contract Confidence | Operational Status |
|:---|:---|:---|:---|:---|:---|:---|:---|:---|:---|:---|:---|:---|
| **image-generate** | `FAST` | **WaveSpeedAI** | `ws-flux-schnell` | `wavespeed-ai/flux-schnell` | Per run | **$0.0030** | 5 Credits | $0.0100 | **$0.0070** | **70.0%** | **VERIFIED** | `READY_FOR_CANARY`* |
| **image-generate** | `FAST` | **KIE.ai** | `kie-flux-schnell` | `flux-2/flex-text-to-image` | Per run | **$0.0035** | 5 Credits | $0.0100 | **$0.0065** | **65.0%** | **VERIFIED** (Queue 2I Nominee) | `CONTRACT_PENDING` |
| **image-generate** | `FAST` | **fal.ai** | `fal-flux-schnell` | `fal-ai/flux/schnell` | Per megapixel | **$0.0035** | 5 Credits | $0.0100 | **$0.0065** | **65.0%** | **VERIFIED** | `BILLING_BLOCKED` |
| **image-generate** | `QUALITY` | **WaveSpeedAI** | `ws-flux-dev` | `wavespeed-ai/flux-dev` | Per run | **$0.0180** | 25 Credits | $0.0500 | **$0.0320** | **64.0%** | **VERIFIED** | `CONTRACT_VERIFIED` |
| **image-generate** | `QUALITY` | **KIE.ai** | `kie-flux-dev` | `flux-2/pro-text-to-image` | Per run | **$0.0200** | 30 Credits | $0.0600 | **$0.0400** | **66.7%** | **VERIFIED** | `CONTRACT_PENDING` |
| **image-generate** | `QUALITY` | **fal.ai** | `fal-flux-dev` | `fal-ai/flux/dev` | Per megapixel | **$0.0250** | 35 Credits | $0.0700 | **$0.0450** | **64.3%** | **VERIFIED** | `BILLING_BLOCKED` |
| **image-upscale** | `QUALITY` | **WaveSpeedAI** | `ws-upscaler` | `wavespeed-ai/image-upscaler` | Per image | **$0.0100** | 15 Credits | $0.0300 | **$0.0200** | **66.7%** | **VERIFIED** | `CONTRACT_VERIFIED` |
| **image-upscale** | `PREMIUM` | **fal.ai** | `fal-clarity` | `fal-ai/clarity-upscaler` | Per image | **$0.0150** | 20 Credits | $0.0400 | **$0.0250** | **62.5%** | **VERIFIED** | `BILLING_BLOCKED` |
| **background-remove**| `FAST` | **fal.ai** | `fal-birefnet` | `fal-ai/birefnet` | Per image | **$0.0050** | 10 Credits | $0.0200 | **$0.0150** | **75.0%** | **VERIFIED** | `BILLING_BLOCKED` |
| **background-remove**| `FAST` | **WaveSpeedAI** | `ws-birefnet` | *(Bria RMBG 2.0)* | Per image | **TBD** | - | - | - | - | **UNVERIFIED** | **DISABLED** |
| **product-photo** | `QUALITY` | **WaveSpeedAI** | `ws-product-flux`| `wavespeed-ai/flux-dev` | Per run | **$0.0180** | 30 Credits | $0.0600 | **$0.0420** | **70.0%** | **VERIFIED** | `DRAFT` |

*\*Evaluates to `READY_FOR_CANARY` when `WAVESPEED_API_KEY` is present, or `CREDENTIAL_REQUIRED` when absent.*

---

## 2. Deep Arbitrage Findings by Logical Tool

### 2.1 Image Generate FAST
- **Arbitrage Winner**: **WaveSpeedAI** (`wavespeed-ai/flux-schnell`) at **$0.0030 USD** per run.
- **Gross Margin**: **70.0%** at 5 Tora Credits ($0.0100 USD retail).
- **Secondary Candidate**: **KIE.ai** (`flux-2/flex-text-to-image`) at **$0.0035 USD** (+16.7% cost over WaveSpeed).
- **Fal Status**: Fal charges $0.0035/megapixel but is currently `BILLING_BLOCKED` (HTTP 403 Exhausted Balance).
- **Strategic Recommendation**: WaveSpeed serves as primary router candidate; KIE serves as immediate hot standby once Queue 2I canary executes.

### 2.2 Image Generate QUALITY
- **Arbitrage Winner**: **WaveSpeedAI** (`wavespeed-ai/flux-dev`) at **$0.0180 USD**.
- **Gross Margin**: **64.0%** at 25 Tora Credits ($0.0500 USD retail).
- **Secondary Candidate**: **KIE.ai** (`flux-2/pro-text-to-image`) at **$0.0200 USD** (+11.1% cost over WaveSpeed).
- **Fal Status**: Fal charges $0.0250 USD (+38.9% cost over WaveSpeed).

### 2.3 Image Upscale
- **Arbitrage Winner**: **WaveSpeedAI** (`wavespeed-ai/image-upscaler`) at **$0.0100 USD**.
- **Gross Margin**: **66.7%** at 15 Tora Credits ($0.0300 USD retail).
- **Alternative**: Fal Clarity Upscaler at $0.0150 USD (+50.0% cost over WaveSpeed).

### 2.4 Background Removal
- **Catalog Forensic Reality**: WaveSpeed catalog does **NOT** contain BiRefNet (uses Bria RMBG 2.0 / Bria Fibo).
- `ws-birefnet` is intentionally marked **`DISABLED`** to avoid invalid upstream dispatches.
- KIE generic name `rembg` is unverified and marked **`DRAFT`**.
- Fal BiRefNet is verified at $0.0050 USD but `BILLING_BLOCKED`.
- **Strategic Policy**: Background Remove tool remains unpromoted until a verified Bria RMBG or KIE model is fully onboarded and proven.

---

## 3. Router Scoring & Selection Determinism

The router calculates a deterministic composite score:
$$\text{Score} = (W_{\text{quality}} \times S_{\text{tier}}) + (W_{\text{health}} \times S_{\text{health}}) + (W_{\text{cost}} \times S_{\text{cost}}) + \frac{1}{\text{Priority}}$$

### Simulation: `image-generate` FAST
1. Candidate `ws-flux-schnell`:
   - Status: `READY_FOR_CANARY` / `CREDENTIAL_REQUIRED`
   - Cost: $0.0030 USD
   - Margin: 70.0% $\ge 60.0\%$ (Pass)
   - Health: 100%
   - **Score: 5.70 -> Selected as Primary**
2. Candidate `kie-flux-schnell`:
   - Status: `DRAFT` (Skipped: `route status draft (not executable)`)
3. Candidate `fal-flux-schnell`:
   - Status: `BILLING_BLOCKED` (Skipped: `route billing blocked (exhausted balance)`)

---

## 4. Operational Recommendations for Operator
1. Inject `WAVESPEED_API_KEY` to unlock the primary route (`ws-flux-schnell`) with 70.0% gross margin.
2. In Queue 2I, onboard verified KIE model `flux-2/flex-text-to-image` using the No-Code Model Onboarding API to establish true multi-provider live redundancy.
