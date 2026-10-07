# =====================================================================
# TORA STUDIO — OVERNIGHT AUTONOMOUS MEDIA RELAY BASELINE
# FORENSIC BASELINE AUDIT
# =====================================================================

CURRENT_COMMIT: 230559f02
PRODUCTION_IMAGE: calciumion/new-api:latest (AWS EC2 Docker Compose)
DB_SCHEMA_VERSION: PostgreSQL GORM AutoMigrate (11 Studio tables)
CANONICAL_CREDIT_CONVERSION: 1 USD = 500,000 quota = 500 Tora Credits (1 Credit = 1,000 quota = $0.0020 USD)

PROVIDER_STATES:
  fal: AUTHENTICATED | REAL UPSTREAM REACHED | BILLING_BLOCKED (HTTP 403 User Locked: Exhausted balance)
  WaveSpeed: CONTRACT IMPLEMENTED | OPERATOR_BLOCKED (WAVESPEED_API_KEY absent from environment)
  KIE: CONTRACT IMPLEMENTED | OPERATOR_BLOCKED (KIE_API_KEY absent from environment)
  Replicate: FOUNDATION_ONLY (Activation disabled in this phase)

ACTIVE_ROUTES: []
DRAFT_ROUTES:
  - id: ws-product-flux
    provider: wavespeed
    model: wavespeed-ai/flux-dev
    reason: Composite product photo input mapping pending canary
  - id: kie-birefnet
    provider: kie
    model: rembg
    reason: Unverified generic model ID in KIE Market
  - id: kie-upscale
    provider: kie
    model: upscale-v1
    reason: Unverified generic model ID in KIE Market
  - id: kie-flux-schnell
    provider: kie
    model: flux-schnell
    reason: Unverified generic model ID in KIE Market (Queue 2I candidate: flux-2/flex-text-to-image)
  - id: kie-flux-dev
    provider: kie
    model: flux-dev
    reason: Unverified generic model ID in KIE Market (Queue 2I candidate: flux-2/pro-text-to-image)
  - id: kie-product-flux
    provider: kie
    model: flux-dev
    reason: Unverified generic model ID in KIE Market

CREDENTIAL_REQUIRED_ROUTES:
  - id: ws-flux-schnell
    provider: wavespeed
    model: wavespeed-ai/flux-schnell
    quality_tier: FAST
    cogs: 0.0030 USD
    status: READY_FOR_CANARY when WAVESPEED_API_KEY injected, else CREDENTIAL_REQUIRED
  - id: ws-upscaler
    provider: wavespeed
    model: wavespeed-ai/image-upscaler
    quality_tier: QUALITY
    cogs: 0.0100 USD
    status: CONTRACT_VERIFIED (awaits dedicated upscaler canary)
  - id: ws-flux-dev
    provider: wavespeed
    model: wavespeed-ai/flux-dev
    quality_tier: QUALITY
    cogs: 0.0180 USD
    status: CONTRACT_VERIFIED (awaits dedicated quality canary)

BILLING_BLOCKED_ROUTES:
  - id: fal-birefnet
    provider: fal
    model: fal-ai/birefnet
  - id: fal-clarity
    provider: fal
    model: fal-ai/clarity-upscaler
  - id: fal-flux-schnell
    provider: fal
    model: fal-ai/flux/schnell
  - id: fal-flux-dev
    provider: fal
    model: fal-ai/flux/dev

DISABLED_ROUTES:
  - id: ws-birefnet
    provider: wavespeed
    model: wavespeed-ai/birefnet
    reason: Model does not exist in WaveSpeed official catalog (uses Bria RMBG 2.0)

ACTIVE_PUBLIC_TOOLS: [] (Honest acceptance: 0 public tools promoted until live provider proof is achieved)
NEW_SERVER_COUNT: 0
NEW_GPU_SERVER_COUNT: 0

---

## Forensic Environment & Host Verification
- **Host Workstation**: macOS local dev environment (`new-api` Go workspace)
- **Production Host**: AWS EC2 with Docker Compose, PostgreSQL 15, Redis 7, Caddy reverse proxy
- **Credential Safety**:
  - `WAVESPEED_API_KEY_PRESENT`: `False`
  - `KIE_API_KEY_PRESENT`: `False`
  - `FAL_KEY_PRESENT`: `False` (in local shell, configured in production container)
  - No secrets logged, exposed, or committed.
- **Invariants**:
  - `NEW_SERVER_COUNT = 0`
  - `NEW_GPU_SERVER_COUNT = 0`
  - No local self-hosted models (ComfyUI, Wan, LTX, MuseTalk, CUDA, PyTorch strictly prohibited).
  - Single wallet invariant: `User.Quota` only.
