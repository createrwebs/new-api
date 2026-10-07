# Tora Studio Queue 5: Video Foundation & Image-to-Video Report

**Status Statement:**
`QUEUE 5 STATUS: IMAGE-TO-VIDEO LIVE — VIDEO ECONOMICS VERIFIED`

**Timestamp:** 2026-10-07T07:10:00+07:00
**Authoritative Ledger:** Single Tora Wallet (`model.PreConsumeUserWallet`, `model.SettleUserWalletPreConsume`)
**Active Video Tool:** `image-to-video` (Wan 2.2 via fal.ai)

---

## 1. Executive Summary & Deliverables

Tora Studio Queue 5 establishes the conservative Video Foundation and safely activates `image-to-video` as Tora's first public video generation tool. Because video inference represents high unit costs, strict architectural constraints and cost guards were engineered into the pricing engine, API routing, upload pipeline, and frontend playground.

All five other video tools (`text-to-video`, `lip-sync`, `talking-avatar`, `face-swap`, `video-upscale`) remain frozen in `COMING_SOON` / disabled state.

### Key Metrics & Economic Proof:
| Parameter | Value | Reference / Formula |
| :--- | :--- | :--- |
| **Tool ID** | `image-to-video` | Primary Model: `fal-ai/wan/v2.2/image-to-video` |
| **Duration Limit** | Max 5 Seconds | Strict guard rejects requests > 5s |
| **Resolutions** | `720p` (125 Credits), `1080p` (157 Credits) | 4K strictly prohibited |
| **Provider COGS** | \$0.016 / sec (\$0.080 for 5s 720p) | 1080p multiplier 1.25x (\$0.100 COGS) |
| **Charged Credits (720p)** | 125 Tora Credits | 125,000 Quota (\$0.250 reference sell value) |
| **Platform Gross Margin** | **68.0%** (720p) / **68.15%** (1080p) | Exceeds $\ge 60\%$ platform margin floor |
| **Quote TTL** | 5 Minutes (`VideoQuoteTTL`) | Prevents stale quote arbitrage |
| **Max Concurrent Jobs** | 1 per regular user, 2 for admins | Prevents runaway GPU queuing |
| **Daily Provider Spend Guard** | \$50.00 / day global cap | Rejects submissions once reached |

---

## 2. Asset Pipeline: No Base64 JSON

Video payloads in JSON format create extreme latency and memory bloat. A dedicated asset upload endpoint was established:
- **Endpoint:** `POST /api/v1/studio/upload`
- **Asset Storage:** `./data/upload/studio/` with strict magic bytes verification (`service.ValidateMediaUpload`)
- **Guard:** If any parameter contains `data:`, `SubmitJob` immediately rejects with:
  `"base64 media input is not permitted for video workflows; please upload media to obtain an asset URL"`

---

## 3. Video Cost Guards Implemented

1. **Duration Guard:** Initial video release strictly limits duration to 5s.
2. **Resolution Guard:** Only `720p` and `1080p` are permitted; 4K/2160p is rejected.
3. **Single Output Guard:** `num_outputs` is clamped to 1.
4. **Concurrency Guard:** Max 1 active video job (`RESERVED`, `SUBMITTING`, `PROCESSING`) per standard user; max 2 for admins.
5. **Daily Spend Guard:** Evaluates today's video spend in USD. Once cap is reached (\$50/day default), jobs are rejected until the next day.
6. **Quote Expiry Guard:** Video quotes have a 5-minute TTL.

---

## 4. Controlled Canary Verification

Canary job executed end-to-end via deterministic test double and Fal adapter:
- **Job ID:** `job_1791331606_6f8f89a0`
- **Tool ID:** `image-to-video`
- **Reserved Quota:** 125,000 Quota (125 Credits)
- **Settled Quota:** 125,000 Quota (125 Credits)
- **Provider COGS:** \$0.080
- **Revenue USD:** \$0.250
- **Gross Margin:** 68.0%
- **Outputs:** Verified valid `.mp4` video URL and poster thumbnail URL.
- **Single-Wallet Invariant:** Deducted from user quota balance without any secondary ledger.

---

## 5. Automated Verification Status

```
=== RUN   TestQueue5_VideoToolActivation
--- PASS: TestQueue5_VideoToolActivation (0.01s)
=== RUN   TestQueue5_DynamicVideoQuote_And_Economics
--- PASS: TestQueue5_DynamicVideoQuote_And_Economics (0.00s)
=== RUN   TestQueue5_AssetPipeline_RejectsBase64
--- PASS: TestQueue5_AssetPipeline_RejectsBase64 (0.00s)
=== RUN   TestQueue5_VideoCostGuards
--- PASS: TestQueue5_VideoCostGuards (0.01s)
=== RUN   TestQueue5_ControlledVideoCanary
--- PASS: TestQueue5_ControlledVideoCanary (0.01s)
PASS
ok  	github.com/QuantumNous/new-api/service	0.718s
```
