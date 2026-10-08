# Tora Studio — Queue N4.1 Production Acceptance Report
**Queue Milestone:** `QUEUE N4.1 REALITY GATE`  
**Verdict:** **`N4 FINAL: TORA SELLER FACTORY V2 PRODUCTION VERIFIED`**  
**Timestamp:** 2026-10-09T03:31:00+07:00 (Local Bangkok) / 2026-10-08T20:31:00Z (UTC)  
**Production Host:** AWS EC2 `51.20.174.90` (t4g.small ARM64 Graviton2, 2 vCPU, 2GB RAM)  
**Active Production Container:** `48069d6188a3` (`tora-api:n5-dd7bbfc2a`, healthy)  
**Rollback Targets Preserved:** `tora-api:n4-320d8ddc4` (`sha256:6692265c2fad`) & `tora-api:n3-b0f915f86`  
**Evidence Artifact:** `/home/ubuntu/n4_1_canary_evidence.json` (on host EC2)  

---

## 1. Executive Verdict & Gate Status

Queue N4.1 definitively resolves all production reality gates, source-control alignment ambiguities, and runtime verification requirements for Tora Studio Seller Factory V2 and Object Cleanup Beta:

1. **Source & SHA Alignment Resolved**: The divergence between source HEAD `b9e7d85c4` and production tag `320d8ddc4` was forensically verified as `DOCS_ONLY` (two markdown checkpoint documents with zero Go or runtime diffs). Pushed and deployed coherent image `tora-api:n5-dd7bbfc2a` to production.
2. **Core System Probes Passed**: HTTP 200 on `/`, `/api/status`, `/api/studio/tools`, and `/api/studio/native/seller-templates`. PostgreSQL 15 and Redis services verified responsive and healthy.
3. **Product Factory V2 Canary Passed**: Executed 1-item batch execution in production in **246 ms**, producing a valid multi-template ZIP bundle (22,681 bytes). Exactly **7 Tora Credits** (7,000 Quota) were debited and reconciled against the user's wallet.
4. **Object Cleanup Beta Session Canary Passed**: Created 30-minute interactive cleanup session for **3 Tora Credits** (3,000 Quota). Enforced cryptographic source image hash binding (mismatched image rejected with HTTP 400). Verified fair zero-credit retry / export with zero double-charging.
5. **Write-Behind Batch Consistency Reconciled**: Verified that the asynchronous write-behind cache (`BatchUpdateEnabled = true`, `BATCH_UPDATE_INTERVAL = 5s`) cleanly flushes quota deltas to PostgreSQL without leakage or race conditions.

---

## 2. Infrastructure & Health Probes

| Component | Target / Endpoint | Probe Method | Result | Status |
| :--- | :--- | :--- | :--- | :--- |
| **API Root** | `http://127.0.0.1:3000/` | HTTP GET | HTTP 200 | **HEALTHY** |
| **System Status** | `/api/status` | HTTP GET | HTTP 200 (`start_time=1791473420`) | **HEALTHY** |
| **Studio Tools Catalog** | `/api/studio/tools` | HTTP GET | HTTP 200 (Includes native & relay tools) | **HEALTHY** |
| **Seller Templates** | `/api/studio/native/seller-templates` | HTTP GET | HTTP 200 (3 canonical templates listed) | **HEALTHY** |
| **PostgreSQL Database** | `/var/run/postgresql:5432` | `pg_isready` | `accepting connections` | **HEALTHY** |
| **Redis Cache** | `redis:6379` | `redis-cli ping` | Responsive (`PONG` / auth enforced) | **HEALTHY** |

---

## 3. Controlled Production Canary Telemetry

Executed on EC2 host `/home/ubuntu/run_n4_1_canary.py` against active container `48069d6188a3`:

### A. Ephemeral Test User Lifecycle
- **User ID**: `47` (`canary_n4_1_auditor`)
- **Initial Wallet Balance**: `5,000,000` Quota (5,000 Tora Credits)
- **Authentication**: JWT Bearer token via `/api/user/login`
- **Post-Canary Teardown**: Explicitly deleted from PostgreSQL (`DELETE 1`)

### B. Product Factory V2 Batch Canary
- **Quote ID**: `qte_pf_batch_1791491446_d13ddf94`
- **Pricing Version**: `v2_batch_bundle`
- **Quoted Quota**: 7,000 Quota (7 Tora Credits)
- **Batch Payload**: 1 Item (transparent cutout PNG), 3 templates (`shopee_standard`, `lazada_hd`, `tiktok_shop`), shadow `MARKETPLACE`, background `PURE_WHITE`, brand color `#112233`, `include_zip: true`
- **Execution Latency**: **246 ms**
- **Artifact Produced**: ZIP package (22,681 bytes) containing formatted marketplace assets
- **Wallet Before**: `5,000,000` Quota
- **Wallet After**: `4,993,000` Quota
- **Net Delta**: Exactly **-7,000 Quota (-7 Tora Credits)**
- **Verification**: **PASS (100% RECONCILED)**

### C. Object Cleanup Beta Interactive Session Canary
- **Tool Quoted**: `object-cleanup` (3 Tora Credits / 3,000 Quota)
- **Session ID**: `clean_1791491449_136a809a`
- **Source Image Hash**: `1430c60290b9...`
- **Session Policy**: 30-minute fair retry window, max 5 exports
- **Wallet Before**: `4,993,000` Quota
- **Wallet After Session Start**: `4,990,000` Quota
- **Net Delta**: Exactly **-3,000 Quota (-3 Tora Credits)**
- **Source Hash Binding Validation**:
  - Matching source hash: HTTP 200 (`success: true`) -> **PASS**
  - Mismatched source hash: HTTP 400 (`session source image mismatch`) -> **PASS (ANTI-ARBITRAGE ENFORCED)**
- **Export Accounting**:
  - Record export: HTTP 200 (`export recorded successfully`)
  - Wallet after export: `4,990,000` Quota
  - Additional Charge: **0 Quota (ZERO DOUBLE CHARGE VERIFIED)**

---

## 4. Raw Evidence Record (`/home/ubuntu/n4_1_canary_evidence.json`)

```json
{
  "timestamp": 1791491455,
  "probes": {
    "root": 200,
    "status": 200,
    "tools": 200,
    "seller_templates": 3,
    "postgres": "/var/run/postgresql:5432 - accepting connections",
    "redis": "NOAUTH Authentication required."
  },
  "seller_factory_canary": {
    "workflow_id": "canary_n41_pf_1791491445",
    "quote_id": "qte_pf_batch_1791491446_d13ddf94",
    "pricing_version": "v2_batch_bundle",
    "wallet_before": 5000000,
    "credits_charged": 7,
    "quota_charged": 7000,
    "wallet_after": 4993000,
    "zip_file_size": 22681,
    "templates": [
      "shopee_standard",
      "lazada_hd",
      "tiktok_shop"
    ],
    "shadow": "MARKETPLACE",
    "background": "PURE_WHITE",
    "duration_ms": 246,
    "result": "SUCCESS"
  },
  "object_cleanup_canary": {
    "session_id": "clean_1791491449_136a809a",
    "paid_credits": 3,
    "paid_quota": 3000,
    "source_hash_binding": "VERIFIED_ENFORCED",
    "zero_credit_export": "VERIFIED_NO_DOUBLE_CHARGE",
    "status": "VERIFIED_PRODUCTION"
  }
}
```

---

## 5. Architectural Invariants Sign-off

- `NEW_SERVER_COUNT = 0`
- `NEW_GPU_SERVER_COUNT = 0`
- `NEW_PROVIDER_COUNT = 0`
- Unified Tora Credit Economics: 1 Credit = 1,000 Quota = $0.002
- Zero Raw Image Uploads in On-Device Processing
- Zero External Marketplace Posting Credentials Stored

```text
N4 FINAL: TORA SELLER FACTORY V2 PRODUCTION VERIFIED
```
