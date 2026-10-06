# TORA AI STUDIO — UNIT ECONOMICS & PRICING MODEL
## MEDIA AI TOOL MARGINS, PROVIDER COGS & TORA CREDIT CONVERSION

> **Platform Anchor**: Tora Unified Wallet & Billing Ledger  
> **Currency Invariant**: `1.00 USD = 500,000 Quota Units` (`QuotaPerUnit = 500,000`)  
> **Exchange Anchor**: ~35 THB / USD (1,000 Quota Units ≈ 0.07 THB)  
> **Target Gross Margin**: 60.0% – 75.0% across all media AI generation tools

---

### 1. Single Wallet Credit Conversion Formula

To eliminate multiple confusing tokens or balances for the user:
```
1 Tora Credit = 1,000 Quota Units (≈ 0.002 USD ≈ 0.07 THB)
500 Tora Credits = 500,000 Quota Units = 1.00 USD (≈ 35.00 THB)
```

The user sees **"Tora Credits"** in the Studio interface, while backend settlement executes atomically against `user.Quota` in standard Tora units (`model.Quota`).

---

### 2. Comprehensive 10-Tool MVP Unit Economics Table

| # | Studio Tool | Provider Candidate | Estimated Provider COGS (USD) | Provider COGS (Quota Units) | Target Gross Margin (%) | Proposed Sell Price (USD) | Proposed Studio Price (Tora Credits) | Equivalent User Quota Deduction | Approx THB Price |
| :-: | :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| 1 | **Image Generate (Fast / Flux Schnell)** | fal.ai | $0.0030 | 1,500 | **70.0%** | $0.0100 | **5 Credits** | 5,000 | ฿0.35 |
| 2 | **Image Generate (Pro / Flux Dev)** | fal.ai | $0.0250 | 12,500 | **60.0%** | $0.0625 | **30 Credits** | 30,000 | ฿2.19 |
| 3 | **Background Remove (BiRefNet)** | fal.ai | $0.0050 | 2,500 | **75.0%** | $0.0200 | **10 Credits** | 10,000 | ฿0.70 |
| 4 | **Image Upscale (4K Clarity)** | fal.ai | $0.0150 | 7,500 | **70.0%** | $0.0500 | **25 Credits** | 25,000 | ฿1.75 |
| 5 | **Object Erase / Inpaint** | fal.ai | $0.0200 | 10,000 | **66.7%** | $0.0600 | **30 Credits** | 30,000 | ฿2.10 |
| 6 | **Product Photo Studio (Relighting)** | fal.ai / MuAPI | $0.0350 | 17,500 | **65.0%** | $0.1000 | **50 Credits** | 50,000 | ฿3.50 |
| 7 | **Portrait Enhance (Face Restore)** | fal.ai | $0.0100 | 5,000 | **66.7%** | $0.0300 | **15 Credits** | 15,000 | ฿1.05 |
| 8 | **Text-to-Video (Fast / LTX-Video 5s)**| fal.ai | $0.0300 | 15,000 | **70.0%** | $0.1000 | **50 Credits** | 50,000 | ฿3.50 |
| 9 | **Image/Text-to-Video (Pro / Wan 2.2 5s)**| MuAPI / fal.ai | $0.0800 | 40,000 | **68.0%** | $0.2500 | **125 Credits** | 125,000 | ฿8.75 |
| 10 | **Lip Sync Video (10s audio)** | fal.ai / MuAPI | $0.0500 | 25,000 | **66.7%** | $0.1500 | **75 Credits** | 75,000 | ฿5.25 |

---

### 3. Blended Margin Analysis: Typical Creator Workflow

Consider a common Thai e-commerce creator workflow:
> *"ถ่ายรูปสินค้า -> ลบพื้นหลัง -> สร้างฉากโฆษณาสินค้า -> ทำวิดีโอ 5 วินาทีสำหรับยิงแอด TikTok"*

#### Workflow Breakdown:
1. **Background Remove**: 10 Credits (Provider cost: $0.005, Revenue: $0.020)
2. **Product Photo Studio**: 50 Credits (Provider cost: $0.035, Revenue: $0.100)
3. **Image-to-Video (5s Wan 2.2)**: 125 Credits (Provider cost: $0.080, Revenue: $0.250)
4. **Total Credits Consumed**: **185 Credits** (185,000 Quota Units)
5. **Total Revenue from User**: **$0.37 USD** (~฿12.95 THB)
6. **Total Provider COGS**: **$0.12 USD** (~฿4.20 THB)
7. **Gross Profit per Workflow Run**: **$0.25 USD** (~฿8.75 THB)
8. **Blended Gross Margin**: **67.6%**

---

### 4. Subscription Multipliers & Volume Discount Rules

1. **Pay-As-You-Go / Top-Up Wallet**: Standard 1.0x credit rate.
2. **Pro / Developer Subscription Tier ($29/mo)**:
   - Includes 15,000 Tora Credits / month ($30 value)
   - 10% credit discount on overage generations (effective 0.9x rate).
3. **Enterprise / Agency Tier ($99/mo)**:
   - Includes 60,000 Tora Credits / month ($120 value)
   - 20% credit discount on overage generations (effective 0.8x rate).
   - Dedicated concurrency lanes and priority queue routing.
