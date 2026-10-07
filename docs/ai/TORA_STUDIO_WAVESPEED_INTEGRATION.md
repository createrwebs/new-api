# TORA STUDIO — WAVESPEED AI INTEGRATION SPECIFICATION
## Protocol `WAVESPEED_V3` & Dynamic Pricing Engine

> **Provider**: WaveSpeed AI  
> **Status**: SOFTWARE READY — OPERATOR CREDENTIAL PENDING (`OPERATOR_BLOCKED` for live canary)  
> **Protocol Identifier**: `WAVESPEED_V3`  
> **Target Tool Coverage**: `background-remove`, `image-upscale`, `image-generate`, `product-photo`

---

### 1. Authoritative API Specification (2026-10-07)

WaveSpeed AI provides high-throughput, low-cost media AI endpoints with dynamic pricing queries and asynchronous prediction lifecycle.

#### A. Base URL & Authentication
- **Base URL**: `https://api.wavespeed.ai/api/v3`
- **Header**: `Authorization: Bearer ${WAVESPEED_API_KEY}`
- **Content-Type**: `application/json`

#### B. Dynamic Pricing Preview (`POST /api/v3/model/price`)
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

#### C. Asynchronous Prediction Submission (`POST /api/v3/{model_id}`)
- **Endpoint**: `POST https://api.wavespeed.ai/api/v3/{model_id}` (e.g. `wavespeed-ai/flux-schnell` or `wavespeed-ai/birefnet`)
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
      "status": "queued"
    },
    "message": "success"
  }
  ```

#### D. Prediction Result Query (`GET /api/v3/predictions/{id}/result`)
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
      "error": ""
    }
  }
  ```

#### E. Account Balance Check (`GET /api/v3/balance`)
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

### 2. Initial WaveSpeed Model Routes in Tora Catalog

| Route ID | Logical Tool | Provider Model ID | Tier | Pricing Strategy | Base COGS | Effective COGS | Tora Sell Credits | Est. Margin |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `ws-birefnet` | `background-remove` | `wavespeed-ai/birefnet` | FAST | DYNAMIC_API | $0.002 | $0.002 | 10 Credits ($0.010) | **80.0%** |
| `ws-real-esrgan` | `image-upscale` | `wavespeed-ai/real-esrgan` | FAST | DYNAMIC_API | $0.003 | $0.003 | 25 Credits ($0.025) | **88.0%** |
| `ws-flux-schnell` | `image-generate` | `wavespeed-ai/flux-schnell` | FAST | DYNAMIC_API | $0.003 | $0.003 | 5 Credits ($0.005) | **40.0%** |
| `ws-flux-dev` | `image-generate` | `wavespeed-ai/flux-dev` | QUALITY | DYNAMIC_API | $0.020 | $0.020 | 50 Credits ($0.050) | **60.0%** |
| `ws-product-flux` | `product-photo` | `wavespeed-ai/flux-dev` | QUALITY | DYNAMIC_API | $0.020 | $0.020 | 50 Credits ($0.050) | **60.0%** |

---

### 3. Implementation Verification

- Adapter implementation: `service/studio_wavespeed.go` (`WaveSpeedAdapter`)
- Registered in: `service/studio_init.go` (`ProtocolWaveSpeedV3`)
- Pure config model addition test: `service/studio_generic_relay_test.go` (`TestStudio_Genericity_AddSecondWaveSpeedModelPurelyViaConfig`) — **PASS**
