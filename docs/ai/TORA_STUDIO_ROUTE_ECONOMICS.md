# TORA STUDIO — ROUTE ECONOMICS & MARGIN POLICIES
## Queue 2G: Canonical Unit Economics, Pricing Governance & Margin Floor Enforcement

> **Authoritative Currency**: Tora Credits (1 Tora Credit = 1,000 Quota units = $0.0020 USD)  
> **Authoritative USD Anchor**: $1.00 USD = 500,000 Quota units = 500 Tora Credits (`common.QuotaPerUnit = 500,000.0`)  
> **Gross Margin Invariant**: Minimum Gross Margin $\ge 60\%$ on all customer-facing logical routes.  
> **Pricing Authorization**: Server-authoritative quote; client never determines price or sees upstream provider COGS.  
> **Status**: SUPERSEDES Queue 2F preliminary documentation.

---

### 1. Unified Ledger & Canonical Currency Conversions

Tora enforces the strict commercial invariant: **ONE USER, ONE TORA WALLET, ONE BILLING LEDGER**. All studio tools debit and credit `User.Quota` directly.

| Unit | USD Value | Quota Units | Tora Credits | Code Source of Truth |
|:---|:---|:---|:---|:---|
| **USD** | $1.00 | 500,000 | 500 | `common.QuotaPerUnit = 500_000.0` (`common/constants.go:22`) |
| **Tora Credit** | $0.0020 | 1,000 | 1 | `service.QuotaPerCredit = 1000` (`service/studio_pricing.go:20`) |
| **Base Quota Unit** | $0.000002 | 1 | 0.001 | Base database column `User.Quota` (`model/user.go`) |

#### Canonical Conversion Equations
$$\text{Charged Quota} = \text{Credits} \times \text{QuotaPerCredit} = \text{Credits} \times 1,000$$
$$\text{Sell USD} = \frac{\text{Charged Quota}}{\text{common.QuotaPerUnit}} = \frac{\text{Credits} \times 1,000}{500,000} = \text{Credits} \times 0.0020$$
$$\text{Gross Margin} = \frac{\text{Sell USD} - \text{Provider COGS USD}}{\text{Sell USD}} \ge 60\%$$

---

### 2. Multi-Provider Route Cost & Margin Matrix (Canonical)

The table below details seeded routes across WaveSpeedAI, KIE.ai, and fal.ai evaluated against retail prices defined in `StudioToolDefinition`:

| Logical Tool | Route ID | Provider | Quality Tier | Provider Model ID | Provider COGS (USD) | Retail Credits | Retail Price (USD) | Gross Margin (%) | Margin Status ($\ge 60\%$) |
|:---|:---|:---|:---|:---|:---|:---|:---|:---|:---|
| **Background Remove** | `ws-birefnet` | WaveSpeed | `FAST` | `wavespeed-ai/birefnet` | $0.0040 | 10 Credits | $0.0200 | **80.0%** | PASS |
| **Background Remove** | `kie-birefnet` | KIE | `FAST` | `rembg` | $0.0050 | 10 Credits | $0.0200 | **75.0%** | PASS |
| **Background Remove** | `fal-birefnet` | fal.ai | `QUALITY` | `fal-ai/birefnet` | $0.0050 | 10 Credits | $0.0200 | **75.0%** | PASS |
| **Image Upscale (4K)** | `ws-upscaler` | WaveSpeed | `FAST` | `wavespeed-ai/image-upscaler` | $0.0100 | 25 Credits | $0.0500 | **80.0%** | PASS |
| **Image Upscale (4K)** | `kie-upscale` | KIE | `QUALITY` | `upscale-v1` | $0.0120 | 25 Credits | $0.0500 | **76.0%** | PASS |
| **Image Upscale (4K)** | `fal-clarity` | fal.ai | `PREMIUM` | `fal-ai/clarity-upscaler` | $0.0150 | 25 Credits | $0.0500 | **70.0%** | PASS |
| **Image Generate (Fast)** | `ws-flux-schnell` | WaveSpeed | `FAST` | `wavespeed-ai/flux-schnell` | $0.0025 | 5 Credits | $0.0100 | **75.0%** | PASS |
| **Image Generate (Fast)** | `kie-flux-schnell` | KIE | `FAST` | `flux-schnell` | $0.0030 | 5 Credits | $0.0100 | **70.0%** | PASS |
| **Image Generate (Fast)** | `fal-flux-schnell` | fal.ai | `FAST` | `fal-ai/flux/schnell` | $0.0030 | 5 Credits | $0.0100 | **70.0%** | PASS |
| **Product Photo Studio** | `ws-product-flux` | WaveSpeed | `QUALITY` | `wavespeed-ai/flux-dev` | $0.0180 | 50 Credits | $0.1000 | **82.0%** | PASS |
| **Product Photo Studio** | `kie-product-flux` | KIE | `QUALITY` | `flux-dev` | $0.0200 | 50 Credits | $0.1000 | **80.0%** | PASS |

---

### 3. Provider Price Source Governance & Lifecycle

Tora rejects unverified prices from routing. Every route in `studio_model_routes` must specify a trusted `price_source`:

| Price Source | Code Constant | Verification Mechanism | Allowed in Paid Routing? |
|:---|:---|:---|:---:|
| `REMOTE_DYNAMIC` | `PriceSourceRemoteDynamic` | Queried upstream via API (e.g. WaveSpeed `POST /api/v3/model/price`) and cached for 5 min | YES |
| `REMOTE_CATALOG` | `PriceSourceRemoteCatalog` | Parsed from upstream catalog endpoint (e.g. WaveSpeed `GET /api/v3/models`) | YES |
| `MANUAL_VERIFIED` | `PriceSourceManualVerified` | Operator-verified against upstream public pricing docs with reference URL & timestamp | YES |
| `UNKNOWN` | `PriceSourceUnknown` | Unverified / default placeholder | **NO (Hard Block)** |

#### Route Lifecycle Statuses
1. `DRAFT`: Newly created route; not routable.
2. `CONTRACT_VERIFIED`: Protocol request/response payload shape verified against upstream specs.
3. `CREDENTIAL_REQUIRED`: Contract verified, but API key not yet loaded in environment.
4. `READY_FOR_CANARY`: API key loaded; eligible for canary verification.
5. `ACTIVE`: Live traffic enabled; passes health checks and margin checks.
6. `DEGRADED`: Transient failures observed; deprioritized by router.
7. `BILLING_BLOCKED`: Upstream returned balance exhaustion / payment required (e.g. HTTP 402/403). Skipped automatically.
8. `DISABLED`: Administratively disabled.

---

### 4. Dynamic Pricing Resolution & Circuit Breaker

#### 4.1 WaveSpeed Dynamic Quote Resolution
WaveSpeed's `POST /api/v3/model/price` endpoint returns pricing tiers:
```json
{
  "code": 200,
  "data": {
    "base_price": 0.005,
    "discounted_price": 0.0035,
    "currency": "USD"
  }
}
```
**Policy**: Tora uses `discounted_price` as the authoritative COGS basis. If `discounted_price` rises above retail equivalent or breaks the $60\%$ margin floor, `SelectRoute` automatically drops the route and switches to the next healthiest candidate.

#### 4.2 Margin Floor Enforcement (`min_margin`)
Each route in `studio_model_routes` defines `min_margin` (default `60.0%`).
In `service/studio_router.go`:
```go
sellUSD := float64(retailQuota) / common.QuotaPerUnit
margin := (sellUSD - effectiveCostUSD) / sellUSD
if margin < (route.MinMargin / 100.0) {
    // Route REJECTED: Violates minimum margin invariant
    continue
}
```
This guarantees zero loss-making executions even if downstream providers alter prices unannounced.

---

### 5. Historical Discrepancy Note (Queue 2F vs Queue 2G)

During Queue 2F, preliminary documentation informally stated `1 Credit = 100 Quota = $0.001 USD` and used an incorrect divisor (`/ 1,000,000.0`). 
In Queue 2G, forensic source analysis proved that:
1. `common.QuotaPerUnit = 500_000.0` has been the authoritative New-API baseline constant since repo creation.
2. `service.QuotaPerCredit = 1000` was established in Queue 1 (`service/studio_pricing.go`).
3. Therefore, $1.00 USD = 500 Tora Credits $\implies 1 \text{ Credit} = \$0.0020 \text{ USD}$.
4. The router logic and admin endpoints were corrected to divide by `common.QuotaPerUnit` (500,000.0).
5. All retail credit prices now produce healthy gross margins between **70.0% and 82.0%**, safely above the 60% floor.
