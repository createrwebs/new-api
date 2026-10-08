# TORA NATIVE VS. RELAY ECONOMIC COMPARISON & COST MATRIX
**Strategic Financial Modeling: Client-Side Edge Inference vs. Cloud Relay Resale**
**Author:** Tora AI Finance & Architecture | **Status:** APPROVED | **Date:** 2026-10-08

---

## 1. Executive Summary

Historically, Tora Studio operated as a media relay reselling upstream provider compute (WaveSpeed, fal, KIE). While relay enables rapid time-to-market, it locks Tora into third-party gross margins of 50–75% and exposes the platform to provider rate limits, cold starts, and upstream outages.

By shifting utility workloads (Background Removal, Upscaling, Product Framing) to **Tora Native Browser/Deterministic processing**, Tora captures two immense structural advantages:
1. **Dramatic User Price Reductions (60% – 80% cheaper)**, driving high customer retention and virality.
2. **Gross Margin Expansion to > 97%**, completely decoupling revenue growth from GPU compute bills.

---

## 2. Unit Economic Comparison Matrix

All figures calculated using canonical Tora billing:
- $1\text{ USD} = 500,000\text{ Quota} = 500\text{ Tora Credits}$
- $1\text{ Tora Credit} = 1,000\text{ Quota} = \$0.0020\text{ USD}$

| Tool / Capability | Execution Route | Model Architecture | Provider COGS (USD) | Tora Price (Credits) | Tora Price (USD) | Platform Gross Margin (%) | Latency / UX |
|---|---|---|---|---|---|---|---|
| **Background Removal** | **Tora Native Browser** | `u2netp.onnx` (4.4MB) | **\$0.0000** | **2 Credits** | **\$0.0040** | **99.2%** | **85 ms** (Instant, Offline) |
| Background Removal | WaveSpeed Relay | `wavespeed-ai/image-background-remover` | \$0.0050 | 10 Credits | \$0.0200 | 75.0% | 1,800 ms (Network hop + queue) |
| Background Removal | fal.ai Relay | `birefnet` | \$0.0100 | 15 Credits | \$0.0300 | 66.7% | 2,400 ms |
| **Portrait Matting** | **Tora Native Browser** | `modnet.onnx` (24.7MB) | **\$0.0000** | **2 Credits** | **\$0.0040** | **99.2%** | **176 ms** (Fine Hair Matting) |
| **Image Upscale 2X** | **Tora Native Browser** | `realesrgan_2x.onnx` | **\$0.0000** | **3 Credits** | **\$0.0060** | **98.8%** | **330 ms** (Tiled WebGPU) |
| **Image Upscale 4X** | **Tora Native Browser** | `RealESRGAN_x4plus` | **\$0.0000** | **5 Credits** | **\$0.0100** | **98.5%** | **1,130 ms** (Tiled WebGPU) |
| Image Upscale 4X | WaveSpeed Relay | `wavespeed-ai/image-upscaler` | \$0.0080 | 25 Credits | \$0.0500 | 84.0% | 3,200 ms |
| **Product Pack (4 Sizes + Zip)** | **Tora Deterministic Pack** | Pure Go / In-Memory Canvas | **\$0.0001** (CPU/Storage) | **5 Credits** | **\$0.0100** | **99.0%** | **180 ms** (Deterministic, 0% hallucination) |
| Product Photo (Diffusion) | WaveSpeed Relay | `flux-schnell` + controlnet | \$0.0250 | 35 Credits | \$0.0700 | 64.3% | 7,500 ms (Nondeterministic text/logos) |

---

## 3. Scale Financial Simulation (100,000 Monthly Transactions)

Assume a workload of 100,000 monthly utility edits (70% background removal, 20% upscale, 10% product packs):

### Scenario A: 100% Upstream Relay
- **Gross Revenue**: 1,550,000 Credits = **\$3,100.00 USD**
- **Upstream Provider Bills (WaveSpeed/fal)**: **\$755.00 USD**
- **Hosting & Infrastructure**: **\$50.00 USD**
- **Net Contribution Margin**: **\$2,295.00 USD (74.0%)**

### Scenario B: Tora Native Execution (with WaveSpeed Fallback)
- **Gross Revenue**: 270,000 Credits = **\$540.00 USD** (At 75% lower price to users)
- **Upstream Provider Bills**: **\$0.00 USD** (Client WebGPU inference)
- **Model Storage & Bandwidth (CDN)**: **\$14.50 USD**
- **Net Contribution Margin**: **\$525.50 USD (97.3%)**
- **Customer Acquisition Advantage**: 4x–5x higher conversion due to 2–5 credit pricing and sub-second speed.

---

## 4. Strategic Recommendation

1. **Default to Native First**: Default all background removal and upscaling requests to `NATIVE_BROWSER` with `u2netp` and `realesrgan`.
2. **Preserve Relay as High-End Fallback**: Keep WaveSpeed active as an optional "Ultra Cloud Quality" tier when client hardware lacks WebGPU or when processing ultra-large complex scenes.
3. **Marketplace Positioning**: Market Tora Studio as the "Instant AI Toolbox" that runs locally on your phone without waiting in cloud queues.
