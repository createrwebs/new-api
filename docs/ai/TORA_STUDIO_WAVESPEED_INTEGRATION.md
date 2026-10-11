# TORA STUDIO — WAVESPEED AI INTEGRATION SPECIFICATION
## Protocol `WAVESPEED_V3` & Dynamic Pricing Engine

> **Provider**: WaveSpeed AI  
> **Status**: SOFTWARE & CONTRACT RECONCILED — OPERATOR CREDENTIAL PENDING (`OPERATOR_BLOCKED` for live canary)  
> **Protocol Identifier**: `WAVESPEED_V3`  
> **Target Tool Coverage**: `background-remove`, `image-upscale`, `image-generate`, `product-photo`  
> **Authoritative Economic Standard**: `v2_canonical` (1 Tora Credit = 1,000 Quota = $0.0020 USD, $1.00 USD = 500 Tora Credits = 500,000 Quota)

---

### 1. Authoritative API Specification (Reconciled Queue 2G)

WaveSpeed AI provides high-throughput, low-cost media AI endpoints with dynamic pricing queries and asynchronous prediction lifecycle.

#### A. Base URL & Authentication
- **Base URL**: `https://api.wavespeed.ai/api/v3`
- **Header**: `Authorization: Bearer ${WAVESPEED_API_KEY}`
- **Content-Type**: `application/json`

#### B. Model Catalog Discovery (`GET /api/v3/models`)
- **Endpoint**: `GET https://api.wavespeed.ai/api/v3/models`
- **Purpose**: Programmatic discovery of available models, schemas, and supported parameters.

#### C. Dynamic Pricing Preview (`POST /api/v3/model/price`)
Allows pre-submission cost verification with account-level discounts:
- **Request**:
  ```json
  {
    "model": "wavespeed-ai/flux-schnell",
    "inputs": {
      "aspect_ratio": "16:9"
    }
  }
  ```
- **Response**:
  ```json
  {
    "code": 200,
    "data": {
      "base_price": 0.003,
      "discounted_price": 0.0025,
      "discount_rate": 0.1667,
      "currency": "USD"
    }
  }
  ```
- **Tora Accounting Rule**: Tora strictly charges and tracks margin against `discounted_price` (actual expected debit to Tora's account), never list price.
- **Cache Strategy**: Results are cached in `MediaPriceCache` for 5 minutes (keyed by `ws_price:{model_id}:{aspect_ratio}`) to minimize latency overhead.

#### D. Asynchronous Prediction Submission (`POST /api/v3/{model_id}`)
- **Endpoint**: `POST https://api.wavespeed.ai/api/v3/{model_id}` (e.g. `wavespeed-ai/flux-schnell` or `wavespeed-ai/birefnet`)
- **Note on Deprecated Paths**: Tora strictly prohibits submitting to obsolete `/api/v3/media/tasks` path. All inference submissions route directly to `/{model_id}`.
- **Request Payload**:
  ```json
  {
    "prompt": "Cyberpunk tiger in neo-Tokyo",
    "aspect_ratio": "16:9",
    "seed": 42
  }
  ```
- **Response**:
  ```json
  {
    "code": 200,
    "data": {
      "id": "ws_pred_9847192847",
      "status": "created",
      "urls": {
        "get": "/predictions/ws_pred_9847192847/result"
      }
    },
    "message": "success"
  }
  ```

#### E. Prediction Result Query (`GET /api/v3/predictions/{id}/result`)
- **Endpoint**: `GET https://api.wavespeed.ai/api/v3/predictions/{id}/result`
- **Response**:
  ```json
  {
    "code": 200,
    "data": {
      "id": "ws_pred_9847192847",
      "status": "completed",
      "outputs": [
        "https://cdn.wavespeed.ai/outputs/ws_pred_9847192847_0.png"
      ],
      "timings": {
        "inference": 1850
      },
      "error": ""
    }
  }
  ```

#### F. Account Balance Check (`GET /api/v3/balance`)
- **Endpoint**: `GET https://api.wavespeed.ai/api/v3/balance`
- **Response**:
  ```json
  {
    "code": 200,
    "data": {
      "balance": 15.50,
      "currency": "USD"
    }
  }
  ```

---

### 2. Audited WaveSpeed Model Routes in Tora Catalog (Canonical Economics)

All figures below reflect the authoritative Tora conversion ($1.00 USD = 500 Tora Credits = 500,000 Quota units):

| Route ID | Logical Tool | Provider Model ID | Tier | Status | COGS (USD) | Retail Credits | Canonical Quota | Sell Value (USD) | Gross Margin |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `ws-birefnet` | `background-remove` | `wavespeed-ai/birefnet` | FAST | `CREDENTIAL_REQUIRED` | $0.0040 | 10 Credits | 10,000 | $0.0200 | **80.0%** |
| `ws-upscaler` | `image-upscale` | `wavespeed-ai/image-upscaler` | FAST | `CREDENTIAL_REQUIRED` | $0.0100 | 25 Credits | 25,000 | $0.0500 | **80.0%** |
| `ws-flux-schnell` | `image-generate` | `wavespeed-ai/flux-schnell` | FAST | `CREDENTIAL_REQUIRED` | $0.0025 | 10 Credits | 10,000 | $0.0200 | **87.5%** |
| `ws-flux-dev` | `image-generate` | `wavespeed-ai/flux-dev` | QUALITY | `CREDENTIAL_REQUIRED` | $0.0180 | 30 Credits | 30,000 | $0.0600 | **70.0%** |
| `ws-product-flux` | `product-photo` | `wavespeed-ai/flux-dev` | QUALITY | `CREDENTIAL_REQUIRED` | $0.0180 | 40 Credits | 40,000 | $0.0800 | **77.5%** |

---

### 3. Implementation Verification

- Adapter implementation: `service/studio_wavespeed.go` (`WaveSpeedAdapter`)
- Registered in: `service/studio_init.go` (`ProtocolWaveSpeedV3`)
- Route contract test: `service/studio_canonical_billing_test.go` (`TestStudio_RouteContract_WaveSpeedAndKie`) — **PASS**
- Zero-code model addition test: `service/studio_generic_relay_test.go` (`TestStudio_Genericity_AddSecondWaveSpeedModelPurelyViaConfig`) — **PASS**
