# TORA STUDIO — GENERIC MEDIA RELAY CORE ARCHITECTURE
## Meta-Router for Media AI APIs (Queue 2F Specification)

> **Platform**: Tora AI Media Studio (`/studio`)  
> **Status**: IMPLEMENTED & TESTED  
> **Repository**: `github.com/QuantumNous/new-api` (branch `feat/formobile`)  
> **Infrastructure Invariant**: `NEW_SERVER_COUNT = 0`, `NEW_GPU_SERVER_COUNT = 0` (Existing EC2 / Compose / PostgreSQL)  
> **Commercial Invariant**: `ONE USER, ONE TORA WALLET, ONE BILLING LEDGER` (`User.Quota` only)

---

### 1. Architectural Philosophy: Meta-Router vs Single-Provider Lock-in

Tora Studio is not hardcoded to any single upstream media API (such as fal.ai, WaveSpeed, or KIE.ai). Instead, Tora Studio operates as a **Meta-Router for Media AI APIs**:

```
                       [ TORA USER REQUEST ]
                                 │
                   (Tool, Template, Quality Tier)
                                 │
                                 ▼
                    [ NORMALIZED MEDIA INPUT ]
             (prompt, assets, aspect_ratio, seed, tier)
                                 │
                                 ▼
                     [ GENERIC MEDIA ROUTER ]
            ┌────────────────────┼────────────────────┐
            │                    │                    │
            ▼                    ▼                    ▼
     [ Route Candidate 1 ] [ Route Candidate 2 ] [ Route Candidate 3 ]
     WaveSpeed AI (V3)     KIE.ai (Jobs V1)       Fal.ai (Queue V1)
            │                    │                    │
            ▼                    ▼                    ▼
     [ Price Engine ]     [ Circuit Breaker ]  [ Margin Floor ]
      Dynamic Quote        Health / Balance     Min Margin >= 60%
            │                    │                    │
            └────────────────────┼────────────────────┘
                                 │
                     [ DETERMINISTIC SCORING ]
             Tier Match + Health - Cost - Latency + Pri
                                 │
                                 ▼
                      [ ATOMIC WALLET LOCK ]
                     PreConsumeUserWallet()
                                 │
                                 ▼
                     [ SAFE FALLBACK DISPATCH ]
               (Ambiguous timeout -> NEVER fall back)
                                 │
                                 ▼
                    [ OUTPUT NORMALIZATION ]
               (Unified provider_job_id, assets[])
                                 │
                                 ▼
                     [ WALLET SETTLEMENT ]
                     SettleUserWalletPreConsume()
```

### 2. Separation of Concerns: Code vs Data

- **Protocol is CODE (`MediaProtocolAdapter` interface)**:
  - Wire format, HTTP headers, authentication handshake, submission payload structure, status polling/webhook format, balance fetching.
  - Examples: `WAVESPEED_V3`, `KIE_JOBS_V1`, `FAL_QUEUE`, `REPLICATE_PREDICTIONS`, `MOCK`.
- **Provider is DATA (`StudioProviderConfig` table)**:
  - Base URL, Secret environment variable name (`secret_env`), default priority, live health status.
- **Model Route is DATA (`StudioModelRoute` table)**:
  - Logical tool binding, model ID, quality tier, input/output mappings, pricing strategy, weights, margin floor.
- **Adding a new model under an existing protocol requires ZERO Go code changes.** It is purely an insert into `studio_model_routes`.

---

### 3. Core Data Contracts

#### A. StudioProviderConfig (`studio_provider_configs`)
```sql
CREATE TABLE studio_provider_configs (
    id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    base_url VARCHAR(255) NOT NULL,
    secret_env VARCHAR(64) NOT NULL, -- e.g. WAVESPEED_API_KEY (never raw secret)
    auth_type VARCHAR(32) DEFAULT 'BEARER',
    protocol VARCHAR(32) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    priority INT DEFAULT 1,
    health_status VARCHAR(32) DEFAULT 'ACTIVE',
    balance_usd NUMERIC(10,4) DEFAULT 0.0,
    last_balance_at BIGINT DEFAULT 0,
    created_at BIGINT,
    updated_at BIGINT
);
```

#### B. StudioModelRoute (`studio_model_routes`)
```sql
CREATE TABLE studio_model_routes (
    id VARCHAR(64) PRIMARY KEY,
    logical_tool VARCHAR(64) NOT NULL,
    provider_id VARCHAR(32) NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    provider_model_id VARCHAR(128) NOT NULL,
    quality_tier VARCHAR(32) DEFAULT 'QUALITY',
    enabled BOOLEAN DEFAULT true,
    input_mapping TEXT,
    output_mapping TEXT,
    capabilities TEXT,
    pricing_strategy VARCHAR(32) DEFAULT 'FIXED_COGS',
    base_cost_usd NUMERIC(10,4) DEFAULT 0.005,
    effective_cost_usd NUMERIC(10,4) DEFAULT 0.005,
    currency VARCHAR(8) DEFAULT 'USD',
    priority INT DEFAULT 1,
    health_weight NUMERIC(5,2) DEFAULT 1.0,
    cost_weight NUMERIC(5,2) DEFAULT 1.0,
    quality_weight NUMERIC(5,2) DEFAULT 1.0,
    latency_weight NUMERIC(5,2) DEFAULT 1.0,
    min_margin NUMERIC(5,2) DEFAULT 60.0,
    consecutive_fails INT DEFAULT 0,
    p50_latency_ms BIGINT DEFAULT 0,
    p95_latency_ms BIGINT DEFAULT 0,
    success_rate NUMERIC(5,2) DEFAULT 100.0,
    created_at BIGINT,
    updated_at BIGINT
);
```

---

### 4. Normalized Media Models

#### NormalizedMediaInput
Represents any creative generation or transformation request without leaking provider-specific parameter names:
```go
type NormalizedMediaInput struct {
    Prompt          string                 `json:"prompt,omitempty"`
    NegativePrompt  string                 `json:"negative_prompt,omitempty"`
    InputAssets     []string               `json:"input_assets,omitempty"`
    MaskAsset       string                 `json:"mask_asset,omitempty"`
    ReferenceAssets []string               `json:"reference_assets,omitempty"`
    AspectRatio     string                 `json:"aspect_ratio,omitempty"`
    Width           int                    `json:"width,omitempty"`
    Height          int                    `json:"height,omitempty"`
    Resolution      string                 `json:"resolution,omitempty"`
    Duration        int                    `json:"duration,omitempty"`
    FPS             int                    `json:"fps,omitempty"`
    Quality         string                 `json:"quality,omitempty"`
    NumberOfOutputs int                    `json:"number_of_outputs,omitempty"`
    Audio           bool                   `json:"audio,omitempty"`
    Seed            int64                  `json:"seed,omitempty"`
    Strength        float64                `json:"strength,omitempty"`
    QualityTier     string                 `json:"quality_tier,omitempty"`
    CallbackURL     string                 `json:"callback_url,omitempty"`
    AdvancedParams  map[string]interface{} `json:"advanced_params,omitempty"`
}
```

#### NormalizedMediaOutput
Canonical result representation for all media formats:
```go
type NormalizedMediaOutput struct {
    ProviderJobID        string                 `json:"provider_job_id"`
    Status               string                 `json:"status"` // queued, processing, completed, failed
    Assets               []NormalizedAsset      `json:"assets"`
    CostEstimated        float64                `json:"cost_estimated"`
    CostActual           *float64               `json:"cost_actual,omitempty"`
    ProviderMetadataSafe map[string]interface{} `json:"provider_metadata_safe,omitempty"`
    RawResponse          string                 `json:"raw_response,omitempty"`
    ErrorMessage         string                 `json:"error_message,omitempty"`
}
```

---

### 5. MediaProtocolAdapter Interface

Every protocol adapter implements:
```go
type MediaProtocolAdapter interface {
    Protocol() string
    ValidateConfiguration(provider *model.StudioProviderConfig) error
    Probe(ctx context.Context, provider *model.StudioProviderConfig) (string, error)
    Quote(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, input *NormalizedMediaInput) (float64, error)
    Submit(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, input *NormalizedMediaInput) (*NormalizedMediaOutput, error)
    Status(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, providerJobId string) (*NormalizedMediaOutput, error)
    Cancel(ctx context.Context, provider *model.StudioProviderConfig, route *model.StudioModelRoute, providerJobId string) error
    FetchBalance(ctx context.Context, provider *model.StudioProviderConfig) (float64, string, error)
}
```

Registered Adapters:
1. `WAVESPEED_V3` (`service/studio_wavespeed.go`)
2. `KIE_JOBS_V1` (`service/studio_kie.go`)
3. `FAL_QUEUE` (`service/studio_protocol_adapters.go`)
4. `REPLICATE_PREDICTIONS` (`service/studio_protocol_adapters.go`)
5. `MOCK` (`service/studio_protocol_adapters.go`)

---

### 6. Router Scorer & Circuit Breaker Policy

#### Route Scoring Algorithm
```
Score = (tierMatch * route.QualityWeight * 2.0)
      + (successRate * route.HealthWeight * 2.0)
      + (costFactor * route.CostWeight * 2.0)
      + priorityBonus
```
Where:
- `tierMatch`: 1.0 (exact tier match), 0.6 (FAST requested, QUALITY available), 0.5 (QUALITY requested, FAST available), 0.2 (PREMIUM requested, other available).
- `successRate`: `route.SuccessRate / 100.0 - (route.ConsecutiveFails * 0.15)`.
- `costFactor`: `1.0 - (effectiveCostUSD / 0.10)`.
- `priorityBonus`: `1.0 / route.Priority`.

#### Circuit Breakers & Ineligibility Filters
1. **Disabled Route**: `route.Enabled == false` -> Skip.
2. **Disabled Provider**: `provider.Enabled == false` -> Skip.
3. **Billing Blocked Provider**: `provider.HealthStatus == "BILLING_BLOCKED"` -> Skip safely (e.g. Fal).
4. **Auth Failure**: `provider.HealthStatus == "AUTH_FAILED"` -> Skip safely.
5. **Route Flapping**: `route.ConsecutiveFails >= 5` -> Skip (tripped circuit breaker).
6. **Profitability Floor**: If `((sellUSD - costUSD) / sellUSD) * 100.0 < route.MinMargin` -> Skip (loss prevention).

---

### 7. Safe Fallback & Invariant Guarantees

#### The Ambiguous Submission Rule (Section 26)
- If downstream returns `ErrProviderAmbiguous` (e.g., HTTP request timeout after bytes were pushed over the socket), **WE MUST NOT FALL BACK TO A SECOND PROVIDER**.
- Calling a second provider while the first might be running would risk double billing or duplicate asset generation.
- The job status is set to `AMBIGUOUS_SUBMISSION` and left for background reconciliation.
- Fallback is ONLY permitted on clean pre-submission failures:
  - Missing credentials (`ErrProviderUnconfigured`)
  - Provider locked / exhausted balance (`ErrProviderAccountLocked`)
  - TCP connection refused before request headers sent.

---

### 8. Secret Vault & SSRF Security

1. **Secret Registry**: The database stores ONLY the environment variable name (`WAVESPEED_API_KEY`, `KIE_API_KEY`, `FAL_KEY`). Raw secret tokens are never persisted in the DB or returned in admin API responses.
2. **SSRF Prevention**: Base URLs are strictly locked to approved provider configurations (`https://api.wavespeed.ai`, `https://api.kie.ai`, `https://queue.fal.run`). Public users can NEVER supply `base_url`, `model_endpoint`, or custom headers.
