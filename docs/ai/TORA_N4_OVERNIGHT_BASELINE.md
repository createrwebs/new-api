# Tora Studio Overnight Autonomous Queue N4 Baseline Report
**Queue:** `OVERNIGHT QUEUE N4`  
**Milestone:** Checkpoint Zero Baseline  
**Timestamp:** 2026-10-08T20:30:00+07:00  

---

## 1. Checkpoint Zero State

```text
BACKEND_HEAD = bd7373089
PRODUCTION_SHA = 1d79f475b
PRODUCTION_IMAGE = tora-api:n4a-1d79f475b
FLUTTER_HEAD = d34fac0
FLUTTER_DIRTY_STATE = MODIFIED_LOCAL_SCRATCH (Untracked playstore docs and cmake files; core source committed)
FLUTTER_REMOTES = origin https://github.com/HuanMeng-official/LumenFlow.git (Tora private remote pending)
ACTIVE_TOOLS = image-generate, background-remove, image-upscale, product-pack, product-factory
BETA_TOOLS = object-cleanup (BETA_NATIVE)
PRODUCT_FACTORY_VERSION = v2_seller_factory
CURRENT_BATCH_LIMIT = 10
CURRENT_PRICING_VERSION = v2_batch_bundle
CURRENT_NATIVE_MODEL_VERSIONS = u2netp_onnx_v1, realesrgan_2x_onnx_v1, telea_fmm_deterministic_v1
ANDROID_DEVICE_AVAILABLE = NO (PHYSICAL_DEVICE_PENDING)
IOS_DEVICE_AVAILABLE = NO (PHYSICAL_DEVICE_PENDING)
NEW_SERVER_COUNT = 0
NEW_GPU_SERVER_COUNT = 0
NEW_PROVIDER_COUNT = 0
```

---

## 2. Invariant Rules for Overnight Engineering Run

1. **Duration & Execution**: Continuous, substantive engineering across all workstreams. No busy waiting, no simulated time passing.
2. **One Tora Credit System**:
   - $1\text{ Credit} = 1,000\text{ Quota} = \$0.002\text{ reference value}$.
   - Unified `users.quota` across Web, Mobile, Native, Deterministic, Relay, and Product Factory.
   - Zero secondary wallets or token schemes.
3. **Hard Infrastructure Constraints**:
   - `NEW_SERVER_COUNT = 0`
   - `NEW_GPU_SERVER_COUNT = 0`
   - `NEW_PROVIDER_COUNT = 0`
4. **Legal / Neural Weight Constraints**:
   - LaMa and MAT remain **strictly rejected** from production.
   - Telea Fast Marching Method and pure Go/Dart deterministic algorithms remain authoritative.
5. **Batch Constraints**:
   - Public maximum batch size is capped at 10 items.
   - Chunk execution size is 3 items.
