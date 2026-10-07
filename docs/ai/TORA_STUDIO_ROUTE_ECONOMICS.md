# TORA STUDIO — ROUTE ECONOMICS & MARGIN POLICIES
## Queue 2F: Media Relay Unit Economics, Pricing Models & Margin Floor Governance

> **Authoritative Currency**: Tora Credits (1 Tora Credit = 100 Quota units = $0.001 USD)  
> **Gross Margin Invariant**: Minimum Gross Margin $\ge 60\%$ on all customer-facing logical routes.  
> **Pricing Authorization**: Server-authoritative dynamic quote; client never determines price or sees upstream provider COGS.

---

### 1. Unified Ledger & Currency Conversions

Tora enforces the strict invariant: **ONE USER, ONE TORA WALLET, ONE BILLING LEDGER**. All studio tools debit and credit `User.Quota` directly.

| Unit | USD Value | Quota Units | Tora Credits |
|:---|:---|:---|:---|
| **USD** | $1.00 | 100,000 | 1,000 |
| **Tora Credit** | $0.001 | 100 | 1 |
| **Base Quota Unit** | $0.00001 | 1 | 0.01 |

**Settlement Equation**:
$$\text{Charged Quota} = \text{Credits} \times 100$$
$$\text{Gross Margin} = \frac{\text{Retail USD} - \text{Provider COGS USD}}{\text{Retail USD}} \ge 60\%$$

---

### 2. Multi-Provider Route Cost & Margin Matrix

The table below details current configured routes across WaveSpeedAI, KIE.ai, and fal.ai:

| Logical Tool | Route ID | Provider | Quality Tier | Provider Model ID | Provider COGS (USD) | Retail Credits | Retail Price (USD) | Gross Margin (%) |
|:---|:---|:---|:---|:---|:---|:---|:---|:---|
| **Background Remove** | `route-wavespeed-birefnet` | WaveSpeed | `FAST` | `wavespeed-ai/birefnet` | $0.0040 | 15 Credits | $0.0150 | **73.3%** |
| **Background Remove** | `route-kie-rembg` | KIE | `FAST` | `rembg` | $0.0050 | 20 Credits | $0.0200 | **75.0%** |
| **Background Remove** | `route-fal-birefnet` | fal.ai | `FAST` | `fal-ai/birefnet` | $0.0050 | 20 Credits | $0.0200 | **75.0%** |
| **Image Upscale** | `route-wavespeed-upscaler` | WaveSpeed | `QUALITY` | `wavespeed-ai/image-upscaler` | $0.0100 | 35 Credits | $0.0350 | **71.4%** |
| **Image Upscale** | `route-kie-upscale` | KIE | `QUALITY` | `upscale-v1` | $0.0120 | 40 Credits | $0.0400 | **70.0%** |
| **Image Upscale** | `route-fal-clarity` | fal.ai | `QUALITY` | `fal-ai/clarity-upscaler` | $0.0150 | 50 Credits | $0.0500 | **70.0%** |
| **Image Generate** | `route-wavespeed-flux-schnell` | WaveSpeed | `FAST` | `wavespeed-ai/flux-schnell` | $0.0025 | 10 Credits | $0.0100 | **75.0%** |
| **Image Generate** | `route-kie-flux-schnell` | KIE | `FAST` | `flux-schnell` | $0.0030 | 15 Credits | $0.0150 | **80.0%** |
| **Image Generate** | `route-wavespeed-flux-dev` | WaveSpeed | `QUALITY` | `wavespeed-ai/flux-dev` | $0.0180 | 50 Credits | $0.0500 | **64.0%** |
| **Image Generate** | `route-kie-flux-dev` | KIE | `QUALITY` | `flux-dev` | $0.0200 | 60 Credits | $0.0600 | **66.7%** |
| **Image Generate** | `route-fal-flux-schnell` | fal.ai | `FAST` | `fal-ai/flux/schnell` | $0.0030 | 15 Credits | $0.0150 | **80.0%** |
| **Product Photo** | `route-wavespeed-product-photo`| WaveSpeed | `PREMIUM` | `wavespeed-ai/product-photo` | $0.0250 | 80 Credits | $0.0800 | **68.8%** |

---

### 3. Dynamic Pricing Resolution & Circuit Breaker

#### 3.1 WaveSpeed Dynamic Quote Parsing
WaveSpeed's `/model/price` returns tiered prices:
```json
{
  "base_price": 0.005,
  "discounted_price": 0.0035,
  "currency": "USD"
}
```
**Policy**: Tora uses `discounted_price` as the authoritative COGS basis. If `discounted_price` rises above retail equivalent or breaks the $60\%$ margin floor, `SelectRoute` automatically drops the route and switches to the next healthiest candidate.

#### 3.2 Margin Floor Enforcement (`min_margin`)
Each route in `studio_model_routes` defines `min_margin` (default `0.60`).
In `service/studio_router.go`:
```go
margin := (retailUSD - effectiveCostUSD) / retailUSD
if margin < route.MinMargin {
    // Route REJECTED due to negative or sub-threshold margin
    continue
}
```
This guarantees zero loss-making executions even if downstream providers alter prices unannounced.

---

### 4. Operational Controls & Routing Weights

Routing decisions are governed by a multi-factor score:
$$\text{Score} = (W_{\text{health}} \times S_{\text{health}}) + (W_{\text{cost}} \times S_{\text{cost}}) + (W_{\text{quality}} \times S_{\text{quality}}) + (W_{\text{latency}} \times S_{\text{latency}})$$

Default weights:
- $W_{\text{health}} = 0.40$
- $W_{\text{cost}} = 0.30$
- $W_{\text{quality}} = 0.20$
- $W_{\text{latency}} = 0.10$

Operations can modify these weights in real time via `PUT /api/admin/studio/routes/:id` without recompiling the application.
