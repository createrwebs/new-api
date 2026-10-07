# TORA STUDIO — KIE.AI INTEGRATION SPECIFICATION & ARCHITECTURE
## Queue 2F: Protocol Adaptor for KIE Jobs API (`KIE_JOBS_V1`)

> **Platform**: Tora Studio Generic Media Relay Core  
> **Protocol**: `KIE_JOBS_V1`  
> **Auth Invariant**: `KIE_API_KEY` stored strictly in server environment; no plain-text credentials in DB or client requests.  
> **Zero-Code Model Invariant**: Adding a new KIE model requires inserting a row into `studio_model_routes`—zero Go code modifications.

---

### 1. Protocol Architecture & Invariants

KIE.ai operates as a task-based asynchronous media synthesis API. Tora's `KieAdapter` (`service/studio_kie.go`) registers under the protocol identifier `KIE_JOBS_V1` in `GlobalProtocolRegistry`.

- **Base URL**: `https://api.kie.ai` (configurable via `studio_provider_configs.base_url`)
- **Authentication**: `Authorization: Bearer ${KIE_API_KEY}`
- **Task Lifecycle**: `waiting` / `running` → `success` / `fail`
- **Dual Completion Strategy**: 
  1. Primary: Serverless Webhook Callbacks via `callBackUrl`
  2. Fallback: Adaptive Exponential Polling against `/api/v1/jobs/recordInfo`

---

### 2. Upstream REST Contract Specification

#### 2.1 Submit Task (`POST /api/v1/jobs/createTask`)

When Tora routes a job to KIE, the relay core normalizes input parameters via `StudioModelRoute.InputMapping` and injects Tora's webhook callback URL.

**Request**:
```http
POST /api/v1/jobs/createTask HTTP/1.1
Host: api.kie.ai
Authorization: Bearer <KIE_API_KEY>
Content-Type: application/json

{
  "model": "rembg",
  "callBackUrl": "https://www.toraapi.com/api/studio/webhook/kie",
  "params": {
    "image_url": "https://assets.toraapi.com/inputs/user1/test.png"
  }
}
```

**Success Response (`200 OK`)**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "taskId": "kie_task_89f02c4b81"
  }
}
```

**Error Response (`4xx / 5xx`)**:
```json
{
  "code": 400,
  "message": "insufficient_balance",
  "data": null
}
```
*Tora Circuit Breaker Handling*: If `code` or message indicates balance exhaustion, the provider status in `studio_provider_configs` transitions to `BILLING_BLOCKED`.

---

#### 2.2 Status Check & Polling (`GET /api/v1/jobs/recordInfo`)

Used by Tora background reconciler or when webhooks are delayed.

**Request**:
```http
GET /api/v1/jobs/recordInfo?taskId=kie_task_89f02c4b81 HTTP/1.1
Host: api.kie.ai
Authorization: Bearer <KIE_API_KEY>
```

**Response (`200 OK`)**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "taskId": "kie_task_89f02c4b81",
    "state": "success",
    "result": {
      "images": [
        "https://cdn.kie.ai/outputs/rembg_out_89f02c4b81.png"
      ]
    },
    "cost": 0.005,
    "failReason": ""
  }
}
```

**Terminal State Mapping in Tora**:
| KIE `state` | Tora Internal Status | Action |
|:---|:---|:---|
| `waiting`, `pending` | `QUEUED` | Continue polling / wait for webhook |
| `running`, `processing` | `PROCESSING` | Continue polling |
| `success`, `completed` | `SUCCEEDED` | Download asset, ingest to storage, settle wallet |
| `fail`, `failed` | `FAILED` | Settle fail, release quota refund |

---

#### 2.3 Webhook Ingress (`POST /api/studio/webhook/kie`)

Public webhook receiver endpoint in Tora Controller (`controller/studio.go`):

**Incoming Payload**:
```json
{
  "taskId": "kie_task_89f02c4b81",
  "state": "success",
  "result": {
    "images": [
      "https://cdn.kie.ai/outputs/rembg_out_89f02c4b81.png"
    ]
  },
  "cost": 0.005
}
```

**Idempotent Settlement Guarantee**:
1. Locates `StudioToolJob` by `provider_job_id = taskId`.
2. Verifies whether job is already in terminal state (`SUCCEEDED` or `FAILED`). If already terminal, returns HTTP `200 OK` with `{"status": "already_settled"}`.
3. If still `PROCESSING` / `QUEUED`:
   - Settles reserved quota in atomic DB transaction (`SettleUserWalletPreConsume`).
   - Ingests output media to Tora permanent storage.
   - Marks job `SUCCEEDED`.

---

### 3. Seeded Default KIE Model Routes

The following initial model routes are seeded in `service/studio_seed.go`:

| Logical Tool | Provider Model ID | Tier | Input Mapping | Base COGS | Retail Price (Tora Credits) |
|:---|:---|:---|:---|:---|:---|
| `background-remove` | `rembg` | `FAST` | `{"image_url": "image_url"}` | $0.005 | 20 Credits ($0.02) |
| `image-upscale` | `upscale-v1` | `QUALITY` | `{"image_url": "image_url", "scale": 4}` | $0.012 | 40 Credits ($0.04) |
| `image-generate` | `flux-schnell` | `FAST` | `{"prompt": "prompt", "aspect_ratio": "aspect_ratio"}` | $0.003 | 15 Credits ($0.015) |
| `image-generate` | `flux-dev` | `QUALITY` | `{"prompt": "prompt", "aspect_ratio": "aspect_ratio"}` | $0.020 | 60 Credits ($0.06) |

---

### 4. Zero-Code Model Addition Verification

As verified in `TestStudio_Genericity_AddSecondKieModelPurelyViaConfig`:
To onboard a new model (e.g. `ideogram-v2` or `kling-v1`), an administrator executes:

```sql
INSERT INTO studio_model_routes (
    id, logical_tool, provider_id, protocol, provider_model_id, quality_tier,
    enabled, input_mapping, output_mapping, capabilities,
    pricing_strategy, base_cost_usd, effective_cost_usd, priority,
    weight_health, weight_cost, weight_quality, weight_latency, min_margin
) VALUES (
    'route-kie-ideogram-v2', 'image-generate', 'kie', 'KIE_JOBS_V1', 'ideogram-v2', 'PREMIUM',
    1, '{"prompt":"prompt","style":"style"}', '{"result.images[0]":"image_url"}', '{"style_selection":true}',
    'FLAT', 0.040, 0.040, 90,
    0.3, 0.3, 0.3, 0.1, 0.60
);
```

**Result**:
- Tora immediately exposes `ideogram-v2` under `image-generate` with `PREMIUM` tier.
- No code recompilation or service restart required.
