# TORA BILLING SYSTEM — CANONICAL CONVERSION SPECIFICATION
## Queue 2G: Authoritative Forensic Proof for Tora Credit ↔ Quota ↔ USD Exchange Rates

> **Document Status**: AUTHORITATIVE & BINDING  
> **Effective Version**: `v2_canonical`  
> **Key Governance Rule**: ONE TORA WALLET, ONE UNIT ECONOMY, ONE BILLING LEDGER. No subsystem (Studio, Chat, Relay) may invent a divergent credit or quota conversion.

---

### 1. Canonical Conversion Ratios

The Tora Platform operates on a single unified billing currency:

| Metric | Authoritative Value | Source of Truth |
| :--- | :--- | :--- |
| **AUTHORITATIVE_INTERNAL_QUOTA_PER_USD** | **`500,000`** quota / $1.00 USD | `common/constants.go:22` (`common.QuotaPerUnit`) |
| **AUTHORITATIVE_TORA_CREDITS_PER_USD** | **`500`** Credits / $1.00 USD | $500,000 \text{ Quota} / 1,000 \text{ QuotaPerCredit}$ |
| **AUTHORITATIVE_QUOTA_PER_TORA_CREDIT** | **`1,000`** quota / 1 Tora Credit | `service/studio_pricing.go:20` (`service.QuotaPerCredit`) |
| **AUTHORITATIVE_USD_EQUIVALENT_PER_TORA_CREDIT** | **`$0.0020`** USD / 1 Tora Credit | $\frac{1 \text{ USD}}{500 \text{ Credits}} = \frac{1,000 \text{ quota}}{500,000 \text{ quota/USD}}$ |

---

### 2. Forensic Codebase Trace & Evidence

#### 2.1 Core System Constant (`common/constants.go`)
```go
// common/constants.go line 22
var QuotaPerUnit = 500 * 1000.0 // $0.002 / 1K tokens -> 500,000 quota per $1.00 USD
```
- This constant has served as the bedrock of New-API / Tora since initial deployment.
- 500,000 internal quota units strictly represent $1.00 USD.

#### 2.2 Payment Top-up Fulfillment (`controller/topup.go`)
```go
// controller/topup.go line 194-202
func getTopUpQuota(amount int64) (int, error) {
    quota := decimal.NewFromInt(amount)
    if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
        quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
        quota = decimal.NewFromInt(quota.Div(quotaPerUnit).IntPart()).Mul(quotaPerUnit)
    } else {
        quota = quota.Mul(decimal.NewFromFloat(common.QuotaPerUnit))
    }
    return common.WalletQuotaFromDecimalStrict(quota)
}
```
When a user tops up $1.00 USD via Stripe, Waffo, Creem, or EPay:
$$\text{Purchased Quota} = 1 \times \text{common.QuotaPerUnit} = 500,000 \text{ quota units}.$$

#### 2.3 Stripe Top-up Crediting (`controller/topup_stripe.go`)
```go
// controller/topup_stripe.go line 424-432
func getStripeCreditedQuota(amount int64, group string) decimal.Decimal {
    topUpGroupRatio := common.GetTopupGroupRatio(group)
    if topUpGroupRatio == 0 {
        topUpGroupRatio = 1
    }
    return decimal.NewFromInt(amount).
        Mul(decimal.NewFromFloat(topUpGroupRatio)).
        Mul(decimal.NewFromFloat(common.QuotaPerUnit))
}
```
For standard users (`topUpGroupRatio = 1`), $1.00 USD directly credits 500,000 quota units.

#### 2.4 Frontend Currency and Quota Display (`web/src/lib/currency.ts`)
```typescript
// web/src/lib/currency.ts line 521-524
const { config } = getCurrencyDisplay()
const amountUSD = quota / config.quotaPerUnit // config.quotaPerUnit = 500,000
return formatCurrencyFromUSD(amountUSD, options)
```
The frontend UI divides user quota by 500,000 to display the user's balance in USD ($).

#### 2.5 Studio Pricing Engine (`service/studio_pricing.go`)
```go
// service/studio_pricing.go line 17-21
const (
    // QuotaPerCredit defines internal quota units per 1 Tora Credit.
    // 500,000 Quota = 500 Tora Credits = $1.00 reference value.
    QuotaPerCredit = 1000
    ...
)
```
```go
// service/studio_pricing.go line 85-96
baseQuota := sellPriceUSD * common.QuotaPerUnit // 500,000 per USD
...
calculatedCredits := adjustedQuota / float64(QuotaPerCredit) // QuotaPerCredit = 1,000
credits := e.ceilWithEpsilon(calculatedCredits)
```
If a customer is quoted 10 Tora Credits:
- Reserved Quota: $10 \times 1,000 = 10,000$ quota units.
- Retail Value in USD: $10,000 / 500,000 = \$0.020$ USD.

#### 2.6 Studio Accounting Settlement (`service/studio_service.go`)
```go
// service/studio_service.go line 513
revenueUSD := float64(job.SettledQuota) / common.QuotaPerUnit
```
Revenue recognized on job completion strictly divides settled quota by `common.QuotaPerUnit` (500,000).

---

### 3. Resolution of the Queue 2F Documentation Contradiction

#### 3.1 The Contradiction
In Queue 2F reports (`docs/ai/TORA_STUDIO_ROUTE_ECONOMICS.md` and `docs/ai/TORA_STUDIO_QUEUE_2F_REPORT.md`), an erroneous conversion was noted:
> "1 Tora Credit = 100 Quota units = $0.001 USD" (Erroneous assumption)

Furthermore, in `service/studio_router.go:183`:
```go
sellUSD := float64(retailQuota) / 1000000.0 // Erroneous divisor!
```
This assumed 1,000,000 quota per $1.00 USD, in direct conflict with `common.QuotaPerUnit` (500,000).

#### 3.2 The Authoritative Resolution
1. **Source of Truth Prevails**: `common.QuotaPerUnit` (500,000) and `service.QuotaPerCredit` (1,000) are the true system constants.
2. **Correct Tora Credit Value**:
   $$1 \text{ Tora Credit} = 1,000 \text{ Quota units} = \$0.0020 \text{ USD}.$$
   $$500 \text{ Tora Credits} = 500,000 \text{ Quota units} = \$1.00 \text{ USD}.$$
3. **Studio Router Correction**:
   `sellUSD := float64(retailQuota) / common.QuotaPerUnit`
4. **Historical Impact**:
   Zero real customer jobs were settled under the erroneous Queue 2F documentation ratios. All mock and internal jobs are preserved under audit logs, and future route pricing is canonicalized under `pricing_version: "v2_canonical"`.

---

### 4. Canonical Unit Conversion Table

| USD Value | Tora Credits | Internal Quota | Notes |
|:---|:---|:---|:---|
| **$0.002** | 1 Credit | 1,000 Quota | Single Tora Credit atom |
| **$0.010** | 5 Credits | 5,000 Quota | Micro utility operation |
| **$0.020** | 10 Credits | 10,000 Quota | Fast image gen / background remove |
| **$0.050** | 25 Credits | 25,000 Quota | 4K image upscaling |
| **$0.080** | 40 Credits | 40,000 Quota | Product photo studio generation |
| **$0.100** | 50 Credits | 50,000 Quota | High-definition generation |
| **$1.000** | 500 Credits | 500,000 Quota | $1 USD Top-up product |
| **$10.000** | 5,000 Credits | 5,000,000 Quota | $10 USD Top-up product |
