# TORA NATIVE PRODUCTION COST MODEL & DELIVERY ECONOMICS
**True Platform Cost Allocation: CDN Transfer, Storage, Compute & Margin Accounting**
**Author:** Tora Finance & Infrastructure Engineering | **Status:** PROVISIONAL MODEL | **Date:** 2026-10-08

---

## 1. Accounting Framework: Distinguishing COGS from Fully Allocated Costs

To avoid naive financial assumptions, Tora distinguishes between:
- **`PROVIDER_COGS`**: Payments made to external third-party API providers (e.g. WaveSpeed, fal, Replicate). For native client-side inference, `PROVIDER_COGS = $0.0000`.
- **`FULLY_ALLOCATED_COST`**: The total incremental cost to deliver the service, including model binary CDN egress, API server ingress/compute, persistent storage, and support.

> [!IMPORTANT]
> **Status**: `FULLY_ALLOCATED_COST = NOT YET MATURE`.
> While provider COGS is zero, platform gross margin cannot be classified as >99% until real CDN egress and retention telemetry is measured in live production.

---

## 2. Model Delivery Economics & CDN Egress

Model weights must be delivered to client browsers over HTTP. Assuming standard AWS CloudFront / Tier-1 CDN pricing of **\$0.085 per GB**:

| Model Asset | Uncompressed Binary | Brotli/Gzip Compressed Transfer | First-Load CDN Egress Cost | Amortized Egress (5 Runs per User) | Amortized Egress (20 Runs per User) |
|---|---|---|---|---|---|
| **`u2netp.onnx`** | 4.36 MB | **~3.9 MB** | **\$0.00033 USD** | **\$0.00007 USD** | **\$0.00002 USD** |
| **`modnet.onnx`** | 24.69 MB | **~22.1 MB** | **\$0.00188 USD** | **\$0.00038 USD** | **\$0.00009 USD** |
| **`realesrgan_2x.onnx`** | 64.08 MB | **~58.5 MB** | **\$0.00497 USD** | **\$0.00099 USD** | **\$0.00025 USD** |
| **`RealESRGAN_x4plus.onnx`** | 64.06 MB | **~58.4 MB** | **\$0.00496 USD** | **\$0.00099 USD** | **\$0.00025 USD** |

### Crucial Economic Dynamics:
1. **The 64MB Super-Resolution Cold Cost**:
   - For `realesrgan_2x`, charging a user 3 Tora Credits (\$0.0060 USD) means the very first download takes \$0.0050 in CDN transfer, leaving minimal margin on transaction #1.
   - However, because the binary is cached in `IndexedDB`, runs 2 through 100 incur **\$0.0000 in egress**, bringing the amortized platform margin to **85% – 95%** over the user's lifetime.
2. **The 4.4MB Background Removal Advantage**:
   - `u2netp` egress is negligible even on the very first transaction (\$0.0003 on a \$0.0040 charge, giving **> 91% gross margin on day one**).

---

## 3. Server-Side Deterministic Processing Cost (Marketplace Product Pack)

The Marketplace Product Pack runs deterministic pure-Go image transformations on Tora's existing AWS EC2 `t4g.small` (ARM64 Neoverse) host:
- **CPU Time**: 180 ms (wall time) per pack across 4 aspect ratios + transparent cutout.
- **Peak RAM Allocated**: ~45–65 MB during bilinear resizing and ZIP creation.
- **Server Compute Cost**: With a `t4g.small` priced at \$0.0168/hour:
  $$\text{Compute Cost per Pack} = \frac{0.18\text{ s}}{3600\text{ s}} \times \$0.0168 \approx \$0.00000084\text{ USD}$$
- **S3 / Disk Storage (Output Assets + ZIP)**:
  - Pack size: ~2.4 MB total files.
  - S3 Standard Storage (30-day lifecycle @ \$0.023/GB-month): ~\$0.000055 USD.
- **Total Server Incremental Cost**: **~ \$0.0001 USD per Pack**.
- **Customer Price**: 5 Tora Credits = **\$0.0100 USD**.
- **Estimated Real Contribution Margin**: **~ 98.9%**.

---

## 4. Resource Allocation & Host Safety Rules (Section 50)

To ensure Marketplace Product Pack jobs do not starve the primary API server (which also runs Chat, News autopilot, Redis, and Postgres connections on `51.20.174.90`):
1. **Maximum File Upload Size**: Hard limit of **20 MB** per raw image.
2. **Maximum Pixel Dimensions**: 5000 × 5000 pixels input bounding box limit.
3. **Conservative Batching**: Initial public release limits batch jobs to **1, 5, or 10 items**. 25–50 item batches are strictly disabled until dedicated worker processes are provisioned.
4. **Memory Guard**: Image buffers are processed sequentially in Go routines with explicit garbage collector reuse to prevent RSS memory spikes.
