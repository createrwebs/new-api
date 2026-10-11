# Tora Studio Overnight Autonomous Queue N5 Baseline Report
**Queue:** `OVERNIGHT QUEUE N5`  
**Milestone:** Checkpoint Zero Baseline  
**Timestamp:** 2026-10-08T22:05:00+07:00  

---

## 1. Baseline State Inventory

```text
BACKEND_HEAD = b9e7d85c4
PRODUCTION_SHA = 320d8ddc4 (Runtime aligned with b9e7d85c4; diff is DOCS_ONLY)
PRODUCTION_IMAGE = tora-api:n4-320d8ddc4
FLUTTER_HEAD = d34fac0
FLUTTER_REMOTE_STATUS = TORA_FLUTTER_REMOTE_REQUIRED (Local bundle: /Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/lumenflow-tora-d34fac0.bundle)
ACTIVE_TOOLS = image-generate, background-remove, image-upscale, product-pack, product-factory
BETA_TOOLS = object-cleanup (BETA_NATIVE)
BATCH_MAX = 10
JOB_CONCURRENCY = 3 (Chunk Execution Size)
PRODUCT_FACTORY_PRICING_VERSION = v2_batch_bundle
SELLER_TEMPLATE_VERSIONS = v2.0 (Shopee, Lazada, TikTok Shop, Instagram Feed, Instagram Story, Generic Marketplace)
PHYSICAL_ANDROID_AVAILABLE = NO (PHYSICAL_DEVICE_PENDING)
PHYSICAL_IOS_AVAILABLE = NO (PHYSICAL_DEVICE_PENDING)
NEW_SERVER_COUNT = 0
NEW_GPU_SERVER_COUNT = 0
NEW_PROVIDER_COUNT = 0
```

---

## 2. Invariants & Scope Boundaries for Queue N5

1. **North Star**: Build the repeatable **Seller Work System** (Seller Profiles, Workflow Presets, Repeat Last Pack, Marketplace Compliance Registry, Resumable Jobs).
2. **Canonical Currency**: $1\text{ Credit} = 1,000\text{ Quota} = \$0.002\text{ USD reference value}$. No auxiliary wallets.
3. **No Provider Shopping & No GPU**: Zero external AI providers, zero server purchases, zero commercial license violations.
4. **Target Hardware Truth**: All performance and batch capacity claims must be measured and validated on the target AWS EC2 `t4g.small` production instance, not projected solely from host Apple Silicon.
