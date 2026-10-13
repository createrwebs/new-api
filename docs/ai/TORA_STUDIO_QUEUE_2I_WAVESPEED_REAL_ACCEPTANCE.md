# =====================================================================
# TORA STUDIO — QUEUE 2I.1 ACCEPTANCE REPORT
# FIRST GENERIC MEDIA RELAY MONEY-MOVING PROOF
# WAVESPEED LIVE CANARY & REAL WALLET SETTLEMENT
# =====================================================================

## 0. MANDATORY ACCEPTANCE FIELDS

- **CURRENT_HEAD**: `dfc787e830d970fd962cf2d27a4bd23e1a7645ec`
- **PARENT_HEAD**: `99b81d0e047caee1fc5bc15bce66810c9c368686`
- **PRODUCTION_IMAGE**: `tora-api:queue2i-dfc787e83`
- **PRODUCTION_IMAGE_DIGEST**: `sha256:60afbf3373418233ef925381164e4ead6f45cd97038bc6e0f369c935f89a1dc9`
- **PRODUCTION_GIT_SHA**: `dfc787e83`
- **DEPLOYED_AT**: `2026-10-07T21:51:44Z`
- **WAVESPEED_KEY_PRESENT**: `true` (Provisioned in `/home/ubuntu/new-api/.env` mode 600, outside Git)
- **WAVESPEED_AUTH_VERIFIED**: `true` (HTTP 200 on Pricing API & verified balance of $0.5224 USD)
- **MODEL_ID**: `wavespeed-ai/flux-schnell`
- **CONTRACT_VERIFIED_AT**: `2026-10-08T05:21:00+07:00`
- **PRICING_API_RESPONSE_CLASS**: `POST https://api.wavespeed.ai/api/v3/model/price` (Dynamic price structure: `base_price`, `discounted_price`, `estimated_cost`, `currency`)
- **PROVIDER_PRICE_SOURCE**: `REMOTE_DYNAMIC`
- **PROVIDER_ESTIMATED_COST**: `$0.0030 USD`
- **PROVIDER_ACTUAL_COST**: `$0.0030 USD`
- **PRICE_VARIANCE**: `0.00%`
- **TORA_CREDITS**: `5`
- **TORA_QUOTA**: `5,000`
- **TORA_SELL_VALUE**: `$0.0100 USD` (5 × $0.0020 USD)
- **ESTIMATED_MARGIN**: `70.0%`
- **ACTUAL_MARGIN**: `70.0%` (Gross Profit = $0.0070 USD per image; satisfies $\ge 60.0\%$ floor)
- **ROUTER_CANDIDATES**:
  1. `ws-flux-schnell`: `READY_FOR_CANARY` $\to$ `ACTIVE` (Eligible, selected, score: 6.94)
  2. `ws-flux-dev`: `CONTRACT_VERIFIED` (Eligible for quality tier, not selected for FAST)
  3. `fal-flux-schnell`: `BILLING_BLOCKED` (Ineligible: HTTP 403 User Locked / Exhausted balance)
  4. `kie-flux-schnell`: `DRAFT` (Ineligible: disabled)
- **ROUTER_SELECTED_ROUTE**: `ws-flux-schnell`
- **ROUTER_SELECTION_REASON**: `Selected route 'ws-flux-schnell' (WaveSpeed AI) with score 6.94 [cost: $0.0030, tier: FAST, priority: 1]`
- **REAL_QUOTE_ID**: `quote_studio_1791411938`
- **PRICING_VERSION**: `v1.2`
- **ROUTING_VERSION**: `v2_generic_relay`
- **CONTROLLED_USER_ID**: `40` (`tora_canary_runner`)
- **WALLET_BEFORE**: `100,000 quota` (100 Credits)
- **REAL_RESERVED_QUOTA**: `5,000 quota`
- **WALLET_AFTER_RESERVE**: `95,000 quota`
- **STUDIO_JOB_ID**: `job_1791411938_9db5d493`
- **WAVESPEED_PREDICTION_ID**: `a68bf4bb3dd948759d41b57e72aa0692`
- **REAL_RESULT_VERIFIED**: `TRUE (Valid JPEG decoded, 93,175 bytes, dimensions 1024x1024)`
- **ASSET_STORAGE_BACKEND**: `Amazon S3 / CloudFront CDN (cdn.cachegalaxy.com) + studio_assets DB`
- **ASSET_PROVIDER_URL**: `https://cdn.cachegalaxy.com/output/285cb8d0-bddf-4a4b-b2f0-d8d1ff86cbce-u2_245ed1dd-e293-4a4e-8e57-e56da22425ea.jpeg`
- **STUDIO_ASSET_ID**: `asset_1791411967_a68bf4bb`
- **ASSET_SHA256**: `5376f982816fe5d911ae801ead5125fe9ad0656c1a992b4e9a7850d9da083309`
- **ASSET_LIFECYCLE_STATE**: `JOB_OUTPUT`
- **SETTLED**: `5,000 quota`
- **WALLET_FINAL**: `95,000 quota`
- **NET_QUOTA_DELTA**: `5,000 quota` ($0.0100 USD equivalent)
- **IDEMPOTENCY_REPLAY**: `VERIFIED` (Replayed request with identical Idempotency-Key returned existing job `job_1791411938_9db5d493` with 0 duplicate predictions and 0 additional quota deductions)
- **CONCURRENT_DUPLICATE_PROTECTION**: `VERIFIED`
- **CALLBACK_POLL_RACE**: `TEST_VERIFIED`
- **SECRET_LEAK_SCAN**: `PASSED (0 leaks in logs, DB, or API responses)`
- **NORMAL_USER_PATH**: `CONCLUSIVELY_IDENTICAL`
- **PUBLIC_TOOL_STATUS**: `image-generate FAST = ACTIVE` (Exactly ONE tool promoted; all other 11 tools remain truthfully DISABLED)
- **FAL_STATUS**: `BILLING_BLOCKED`
- **KIE_STATUS**: `PREPARED / NOT LIVE`
- **TOTAL_PROVIDER_SPEND**: `$0.0030 USD` (Ceiling $\le \$0.02$ strictly respected)
- **NEW_SERVER_COUNT**: `0`
- **NEW_GPU_SERVER_COUNT**: `0`

---

## 1. FORENSIC VERDICT
```text
FINAL STATUS: TORA GENERIC MEDIA RELAY ONE PROVIDER LIVE — WAVESPEED CREDIT LOOP VERIFIED
```

### Forensic Evidence Summary
1. **Production Deployment & Code Alignment**:
   - `tora-api:queue2i-dfc787e83` is actively running on AWS EC2 (`51.20.174.90`).
   - Container health status: `Up (healthy)`.
   - PostgreSQL 15 and Redis 7 remained completely uninterrupted (0 restarts).

2. **Secret Ingestion & Upstream Authentication**:
   - `WAVESPEED_API_KEY` was securely provisioned into `/home/ubuntu/new-api/.env` (`chmod 600`).
   - Authenticated probe to `POST /api/v3/model/price` returned `HTTP 200 OK`.
   - Upstream balance confirmed: `$0.5224 USD`.

3. **Dynamic Upstream Quote & Margin Hard Gate**:
   - Dynamic Price API returned: `$0.0030 USD`.
   - Retail Sell Value (5 Tora Credits): `$0.0100 USD` (5,000 Quota).
   - Realized Gross Margin: **70.0%** ($\ge 60.0\%$ floor fully satisfied).

4. **Real Production Media Relay Canary Flow**:
   - User 40 (`tora_canary_runner`) Initial Quota: `100,000`.
   - Request dispatched to `POST /api/studio/jobs` with `tool_id = "image-generate"`, `quality_tier = "FAST"`.
   - Router selected `ws-flux-schnell`.
   - Wallet Pre-Consumed: `5,000 Quota` (Balance $\to$ `95,000`).
   - Upstream WaveSpeed Prediction: `a68bf4bb3dd948759d41b57e72aa0692`.
   - Status polled $\to$ `SUCCEEDED`.
   - Wallet Settled: `5,000 Quota`. Net quota delta = `5,000` Quota.
   - Genuine Output Image URL: `https://cdn.cachegalaxy.com/output/285cb8d0-bddf-4a4b-b2f0-d8d1ff86cbce-u2_245ed1dd-e293-4a4e-8e57-e56da22425ea.jpeg`.
   - Size: `93,175 bytes` | MIME: `image/jpeg` | Dimensions: `1024x1024`.
   - SHA-256: `5376f982816fe5d911ae801ead5125fe9ad0656c1a992b4e9a7850d9da083309`.
   - Asset persisted in `studio_assets` as `asset_1791411967_a68bf4bb` (`lifecycle_state = JOB_OUTPUT`).

5. **Idempotency Replay**:
   - Replay with identical `Idempotency-Key` returned `job_1791411938_9db5d493` with HTTP 201.
   - User quota after replay remained `95,000`. Zero duplicate predictions created.

6. **Public Tool Activation**:
   - `image-generate` FAST promoted to `ACTIVE`.
   - All other 11 Studio tools set to `DISABLED` until their respective canaries pass.
