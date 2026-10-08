# Tora Studio Product Factory V2 Billing & Economic Model
**Queue:** `OVERNIGHT QUEUE N4`  
**Milestone:** Mixed-Mode Step Billing, Bundle Pricing & Quota Integrity  
**Timestamp:** 2026-10-08T20:31:00+07:00  

---

## 1. Unified Currency & Quota Conversion

All Tora Studio capabilities—Web, Mobile, Native, Server, and Relay—operate exclusively on canonical **Tora Credits**:

$$\text{1 Tora Credit} = 1,000 \text{ Quota Units} = \$0.0020 \text{ Reference Sell Value}$$

- Stored in standard `users.quota` column in PostgreSQL.
- No secondary or platform-specific wallets (e.g. no "Mobile Credits" or "Seller Tokens").

---

## 2. Server-Authoritative Batch Pricing Schedule (`v2_batch_bundle`)

The batch quote engine calculates quotes on the server via `POST /api/studio/native/product-factory/quote`. Frontends are strictly forbidden from performing client-side pricing multiplication.

### Batch Pricing Matrix:

| Batch Size | Pipeline Mode | Included Steps | Gross Credits | Bundle Discount | Net Customer Credits | Quota Deducted | USD Reference Value | Estimated COGS | Contribution Margin |
|:---:|:---|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **1 Item** | **Base** | Cutout (Local) + 6 Templates (Server) | 7 | 0% (0) | **7 Credits** | 7,000 | $0.0140 | $0.0003 | **97.8%** |
| **1 Item** | **Enhanced** | Cutout + 2X Upscale (Local) + Templates | 10 | 0% (0) | **10 Credits** | 10,000 | $0.0200 | $0.0003 | **98.5%** |
| **5 Items** | **Base** | 5 $\times$ [Cutout + Templates] | 35 | 5% (-1) | **34 Credits** | 34,000 | $0.0680 | $0.0015 | **97.8%** |
| **5 Items** | **Enhanced** | 5 $\times$ [Cutout + Upscale + Templates] | 50 | 5% (-2) | **48 Credits** | 48,000 | $0.0960 | $0.0015 | **98.4%** |
| **10 Items** | **Base** | 10 $\times$ [Cutout + Templates] | 70 | 10% (-7) | **63 Credits** | 63,000 | $0.1260 | $0.0030 | **97.6%** |
| **10 Items** | **Enhanced** | 10 $\times$ [Cutout + Upscale + Templates] | 100 | 10% (-10) | **90 Credits** | 90,000 | $0.1800 | $0.0030 | **98.3%** |

*Note: Smart Shadows and 7 Studio Background Presets are included as intrinsic value within the Product Factory workflow at 0 additional micro-charge.*

---

## 3. Mixed Billing Execution Semantics

The Product Factory executes a hybrid pipeline combining on-device privacy with server batch compositing. To handle partial completion safely, a dual-layer billing model is enforced:

```text
[ Batch Workflow Started ]
           │
           ├──► LOCAL STEPS (Cutout, Upscale, Cleanup)
           │    • Policy: PREPAID_EXECUTION
           │    • Charge: Pre-deducted before releasing on-device model ticket
           │    • Fair Retry: 30-minute retry window at 0 additional credits
           │    • Refund Policy: No automatic refund once local entitlement is used
           │
           └──► SERVER STEPS (Compositing, Templates, ZIP Packaging)
                • Policy: SUCCESS_SETTLEMENT / RESERVATION
                • Reservation: Quota reserved during batch initialization
                • Settlement: Captured when server ZIP is generated
                • Failure Isolation: If item #4 fails, items #1-#3 settle normally;
                  unprocessed server quota for item #4 is refunded.
```

---

## 4. Object Cleanup Session Add-On

When an artist or seller selects optional **Object Cleanup**:
1. An interactive cleanup ticket is provisioned for **3 Tora Credits** (3,000 Quota).
2. The ticket is immutably bound to the specific item's image `source_hash`.
3. The merchant receives 30 minutes of interactive mask adjustments, live preview rendering, and up to 5 final exports at 0 additional charge.
4. Attempting to apply the same session ticket to a different source image is rejected with HTTP 400 (`source_hash mismatch`).

---

## 5. Quote Expiry & Buy Credits Flow

1. **Quote TTL**: All Product Factory quotes carry an `expires_at` timestamp with a **5-minute TTL**. Once expired, the frontend must request a fresh quote before confirmation.
2. **Insufficient Quota Handling**:
   - If a merchant's `users.quota` is insufficient to cover the quote, the API returns HTTP 402 (`INSUFFICIENT_QUOTA`).
   - The UI redirects the merchant to the Stripe / PromptPay checkout flow.
   - Upon successful payment, the merchant is returned to the exact preserved Product Factory configuration with a freshly refreshed quote.
   - Execution **never** starts automatically post-purchase without explicit merchant reconfirmation.
