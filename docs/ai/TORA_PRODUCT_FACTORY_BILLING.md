# Tora Product Factory Unit Economics & Margin Audit

**Identifier:** `TORA-ECONOMICS-N3-PRODUCT-FACTORY`  
**Date:** 2026-10-08  
**Scope:** Unit economics, gross margin analysis, infrastructure COGS, and pricing structure for Tora Product Factory.  

---

## 1. Credit Pricing & COGS Breakdown

Tora 1 Credit = **$0.0020 USD** (Based on standard $10 = 5,000 Credits conversion).

### 1.1 Item Pipeline Breakdown: Without 2X Upscale

| Pipeline Step | Compute Location | Execution Class | Billing Policy | Tora Credits | Customer Price (USD) | Tora COGS (Server/Compute) | Gross Margin (%) |
|---|---|---|---|---|---|---|---|
| 1. Neural Cutout (`u2netp`) | User's Phone (CoreML/NNAPI) | `NATIVE_MOBILE` | `PREPAID_EXECUTION` | **2 Credits** | $0.0040 | **$0.0000** (Zero server cost) | **100.0%** |
| 2. Cloud Marketplace Pack | Tora Serverless CPU (Go) | `DETERMINISTIC_SERVER` | `SUCCESS_SETTLEMENT` | **5 Credits** | $0.0100 | **$0.0003** (30ms CPU + S3 IO) | **97.0%** |
| **Total (Base Pack)** | **Hybrid** | — | — | **7 Credits** | **$0.0140** | **$0.0003** | **97.8%** |

---

### 1.2 Item Pipeline Breakdown: With 2X Neural Upscale

| Pipeline Step | Compute Location | Execution Class | Billing Policy | Tora Credits | Customer Price (USD) | Tora COGS (Server/Compute) | Gross Margin (%) |
|---|---|---|---|---|---|---|---|
| 1. Neural Cutout (`u2netp`) | User's Phone | `NATIVE_MOBILE` | `PREPAID_EXECUTION` | **2 Credits** | $0.0040 | **$0.0000** | **100.0%** |
| 2. 2X Upscale (`realesrgan_2x`) | User's Phone (Tiled) | `NATIVE_MOBILE` | `PREPAID_EXECUTION` | **3 Credits** | $0.0060 | **$0.0000** | **100.0%** |
| 3. Cloud Marketplace Pack | Tora Serverless CPU | `DETERMINISTIC_SERVER` | `SUCCESS_SETTLEMENT` | **5 Credits** | $0.0100 | **$0.0003** | **97.0%** |
| **Total (Enhanced Pack)** | **Hybrid** | — | — | **10 Credits** | **$0.0200** | **$0.0003** | **98.5%** |

---

## 2. Competitive Market Comparison

How Tora Product Factory compares against market alternatives:

| Solution | Pricing per Product Image | Multi-Platform Formats | Speed / Latency | Privacy (Source Upload) | Tora Advantage |
|---|---|---|---|---|---|
| **Adobe Photoshop / Freelancer** | $1.00 – $5.00 / image | Manual resize | 24–48 hours | Full manual upload | **98% cheaper, instant** |
| **Photoroom / Pixelcut Pro** | $0.10 – $0.20 / image (or $15/mo) | Separate steps | 3–5 seconds | Uploads raw image to cloud | **10x cheaper, zero upload** |
| **Tora Product Factory (Base)** | **$0.014 / image** | **5 platforms in 1 ZIP** | **< 1.5s total** | **Zero raw photo upload** | **Unmatched margin & privacy** |
| **Tora Product Factory (2X)** | **$0.020 / image** | **5 platforms in 1 ZIP** | **< 2.5s total** | **Zero raw photo upload** | **Super-resolution included** |

---

## 3. Infrastructure Scalability & Capex Invariance

- **Zero GPU Procurement**: Because the heavy floating-point tensor arithmetic runs on the user's phone, Tora's cloud infrastructure only performs deterministic CPU bilinear resizing, canvas padding, and ZIP archiving.
- **Capacity**: A single $20/month t4g.small AWS EC2 instance can comfortably process over **50,000 product packs per day** without GPU saturation.
- **Gross Profit at Scale**:
  - 100,000 product packs processed = **$1,400 – $2,000 revenue**.
  - Infrastructure COGS = **~$30**.
  - Net Gross Margin = **> 98%**.
