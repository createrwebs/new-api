# Tora AI — R10 Route & Fallback Routing Architecture Audit

**Status**: RESEARCH, SOURCE COMPARISON, ARCHITECTURE DESIGN, THREAT MODELING & TEST PLANNING  
**Repository Baseline**: `/Users/noppanan/new-api` (`feat/formobile`)  
**Target Identity**: Tora AI Platform & Mobile Ecosystem  
**Document**: `docs/ai/phase_7i_route_fallback_architecture_audit.md`  

---

## 1. Executive Summary

This architecture audit evaluates the requirements, current implementation, upstream capabilities, failure modes, and industry best practices for introducing a **first-class API-key route and ordered fallback routing system** into the Tora AI backend (New-API fork).

### Mission Objective
Design an intuitive, reliable, and financially safe routing engine where:
```text
API Key
  └── Primary Route
        └── Fallback Route #1
              └── Fallback Route #2
                    └── ...
```
Requests automatically fail over to subsequent routes upon encountering eligible upstream errors, while **strictly billing according to the route and channel actually utilized**, maintaining absolute separation between commercial entitlements and routing topologies, and upholding the zero-leak BYOK security invariant.

### Key Audit Conclusions
1. **Current Fork State**: `feat/formobile` inherits upstream New-API's rudimentary `group = "auto"` and `Token.AutoGroups` JSON array with `Token.CrossGroupRetry`. However, this mechanism **dangerously conflates commercial entitlement groups (`default`, `pro`) with routing policies**.
2. **Entitlement vs. Routing Conflict**: In Tora, groups represent billing quotas, subscription tiers, and rate ceilings. Forcing routing fallbacks through the group mechanism allows users to bypass quota restrictions, risks billing miscalculations, and causes priority inversion bugs.
3. **Upstream Deficiencies**: Upstream New-API lacks a first-class `Route` abstraction, suffers from priority index corruption when channels auto-disable, lacks per-model cooldowns, and can retry the same failed channel in identical priority tiers.
4. **Recommendation**: Implement a dedicated, first-class **Route Engine** with explicit Route entities, API-key route chains, strict pre-commitment streaming guards, authoritative per-route billing settlement, and complete isolation from BYOK execution paths.

---

## 2. Current Tora Source Audit (`feat/formobile`)

An exhaustive source audit of `/Users/noppanan/new-api` on branch `feat/formobile` reveals the exact current state of routing, selection, and billing primitives:

### 2.1 Token / API Key Model (`model/token.go`)
Inspection of lines 15–46 in `model/token.go`:
```go
type Token struct {
    Id                 int            `json:"id"`
    UserId             int            `json:"user_id" gorm:"index"`
    Key                string         `json:"key" gorm:"type:varchar(128);uniqueIndex"`
    KeyHash            string         `json:"-" gorm:"type:varchar(64);index"`
    Status             int            `json:"status" gorm:"default:1"`
    Name               string         `json:"name" gorm:"index"`
    CreatedTime        int64          `json:"created_time" gorm:"bigint"`
    AccessedTime       int64          `json:"accessed_time" gorm:"bigint"`
    ExpiredTime        int64          `json:"expired_time" gorm:"bigint;default:-1"`
    RemainQuota        int            `json:"remain_quota" gorm:"default:0"`
    UnlimitedQuota     bool           `json:"unlimited_quota"`
    ModelLimitsEnabled bool           `json:"model_limits_enabled"`
    ModelLimits        string         `json:"model_limits" gorm:"type:text"`
    AllowIps           *string        `json:"allow_ips" gorm:"default:''"`
    UsedQuota          int            `json:"used_quota" gorm:"default:0"`
    Group              string         `json:"group" gorm:"default:''"`
    CrossGroupRetry    bool           `json:"cross_group_retry"` // 跨分组重试，仅auto分组有效
    AutoGroups         string         `json:"-" gorm:"type:text"`
    DeletedAt          gorm.DeletedAt `gorm:"index"`
}
```
**Authoritative Findings**:
- `Token.CrossGroupRetry`: **PRESENT** (`bool`).
- `Token.AutoGroups`: **PRESENT** (`string`, serialized JSON array of group names via `GetAutoGroups()` / `SetAutoGroups()`).
- `Token.Group == "auto"`: **SUPPORTED** in business logic.
- `Token.AutoGroupOrder`: **NOT FOUND IN SOURCE**. Ordering is solely derived from the positional index in the `AutoGroups` JSON array.
- `Token.Subnet`: **NOT FOUND IN SOURCE** (IP restriction is handled via `AllowIps` newline-delimited string).

### 2.2 Token Authentication Middleware (`middleware/auth.go`)
- `TokenAuth()` (line 524) validates the key, checks expiration, status, IP limits (`token.GetIpLimits()`), and resolves the user's usable groups:
  ```go
  userUsableGroups := service.GetUserUsableGroups(userGroup)
  ```
- If `token.Group == "auto"`, it validates that every group listed in `token.AutoGroups` exists within `userUsableGroups`.
- Context variables set: `ContextKeyTokenGroup`, `ContextKeyUsingGroup`, `ContextKeyTokenCrossGroupRetry`, `ContextKeyModelLimits`.

### 2.3 Channel Cache & Architecture (`model/channel_cache.go`)
- **Data Structures**:
  - `group2model2channels`: `map[string]map[string][]int` (caches channel IDs by `[group][model]`).
  - `channelsIDM`: `map[int]*Channel` (in-memory lookup table of all channels).
- **Sorting**: Channels within `group2model2channels[group][model]` are sorted in descending order of priority at cache initialization (`InitChannelCache()`).
- **Priority Tiering**:
  - In `GetRandomSatisfiedChannel(group, model, retry, filters)`:
  - Extracts unique priority values into `sortedUniquePriorities` in descending order.
  - Resolves target priority by indexing: `targetPriority := sortedUniquePriorities[retry]`.
  - If `retry >= len(sortedUniquePriorities)`, it caps at the lowest priority tier: `retry = len(sortedUniquePriorities) - 1`.
- **Weighted Selection**:
  - Collects all channels matching `targetPriority`.
  - Calculates `sumWeight`. Applies a smoothing factor: if all weights are 0, effective weight = 100 per channel; if average weight < 10, multiplies weights by 100.
  - Draws channel using `rand.Intn(totalWeight)`.

### 2.4 Channel Selection & Auto Groups (`service/channel_select.go`)
- `CacheGetRandomSatisfiedChannel(param *RetryParam)`:
  - When `param.TokenGroup == "auto"`, it reads `startGroupIndex` from `ContextKeyAutoGroupIndex` (defaults to 0).
  - Iterates through groups sequentially: `for i := startGroupIndex; i < len(autoGroups); i++`.
  - Within each group, attempts to resolve a channel at `priorityRetry` (where `priorityRetry = param.GetRetry()` if in starting group, or `0` if moved to a subsequent group).
  - If a group has no channels for the requested model, it advances `ContextKeyAutoGroupIndex` to `i+1`, resets `param.SetRetry(0)`, and continues to the next group.
  - **Cross-Group Retry Mechanism**: If `crossGroupRetry == true` and `priorityRetry >= common.RetryTimes`, it sets `ContextKeyAutoGroupIndex` to `i+1` and resets `param.SetRetry(0)` for the *next* retry iteration.

### 2.5 Relay Loop & Error Handling (`controller/relay.go`)
- The main execution loop in `Relay(c *gin.Context, relayFormat types.RelayFormat)`:
  ```go
  for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
      channel, channelErr := getChannel(c, relayInfo, retryParam)
      // Execute upstream request...
      if newAPIError == nil {
          service.MarkRequestPolicySuccess(c, relayInfo.StreamStatus)
          return
      }
      decision := service.DecideRelayRetry(c, newAPIError, common.RetryTimes-retryParam.GetRetry())
      if c.GetBool("is_byok") || common.GetContextKeyBool(c, constant.ContextKeyIsBYOK) {
          break // BYOK is strictly terminal
      }
      if decision.Action != "retry" {
          break
      }
  }
  ```

### 2.6 Quota Billing Pipeline (`service/quota.go` & `service/billing.go`)
- **Pre-Charge**: `PreConsumeBilling()` initializes a `BillingSession` and calls `PreConsumeTokenQuota()`, which atomically decrements `token.remain_quota` via `model.TryReserveTokenQuota()`.
- **Settlement Formula**:
  $$\text{Quota} = \left(\text{InputTokens} + \text{OutputTokens} \times \text{CompletionRatio} + \text{AudioTokens} \times \dots\right) \times (\text{GroupRatio} \times \text{ModelRatio})$$
  *(Or if `UsePrice == true`: $\text{Quota} = \text{ModelPrice} \times \text{QuotaPerUnit} \times \text{GroupRatio}$)*
- **Post-Charge**: Decrements subscription quota or wallet quota via `model.DecreaseUserQuota()`, settles token balance via `model.DecreaseTokenQuota()`, and logs the actual `group_ratio` and `model_ratio` used.

---

## 3. Upstream New-API Comparison

### 3.1 Upstream Capabilities (`QuantumNous/new-api`)
A line-by-line comparison between upstream `main` and our `feat/formobile` branch:

| Feature / Primitive | Upstream `main` | Tora `feat/formobile` | Difference Analysis |
| :--- | :---: | :---: | :--- |
| `Token.Group` | `string` | `string` | Identical. |
| `Token.AutoGroups` | `string` (JSON) | `string` (JSON) | Identical. |
| `Token.CrossGroupRetry` | `bool` | `bool` | Identical. |
| Dedicated `Route` Entity | **None** | **None** | Neither has an explicit Route abstraction. |
| Per-Model Cooldown (`Retry-After`) | **None** | **None** | Both auto-disable channels globally; no backoff timer. |
| BYOK Execution Boundary | **None** | **Implemented** | Tora enforces strict BYOK isolation (`c.GetBool("is_byok")`). |
| Financial & Store Billing | **None** | **Implemented** | Tora contains store ledger, Apple/Google verifiers, subscription bindings. |
| Streaming Commitment Check | Implicit | Partial (`StreamStatus`) | Neither has explicit `c.Writer.Written()` guards before retry. |

### 3.2 Scoped Diff Assessment
- Upstream has **no routing capabilities** beyond the procedural `group = "auto"` loop in `service/channel_select.go`.
- Merging upstream wholesale would not solve the routing challenge and would overwrite Tora's security hardening, subscription ledger, and BYOK protections.

---

## 4. Known Upstream Failure Modes & Architectural Bugs

Research into upstream issues (e.g., #7659, #7660, #7640) and deep code analysis reveals four critical failure modes in the existing channel selection design:

### Bug 1: Priority Tier Mutation on Channel Auto-Disable
- **Mechanism**: When an upstream request fails with a 5xx or auto-ban error, `processChannelError` calls `DisableChannel(channel.Id)`. This synchronously or asynchronously updates `group2model2channels` by slicing out `channel.Id`.
- **Failure**: In `model/channel_cache.go`, `sortedUniquePriorities` is recomputed from the remaining channels. If the disabled channel was the sole occupant of Priority Tier 10, the priority slice contracts: `[10, 5, 1]` becomes `[5, 1]`.
- **Impact**: On the subsequent retry iteration (`retry = 1`), the selector targets index `1`, which is now Priority `1` instead of Priority `5`! The middle tier is skipped entirely.

### Bug 2: Repeated Selection of the Same Failed Channel
- **Mechanism**: If multiple channels share the same priority tier (e.g., three channels in Priority 10 with weights 50, 50, 50), and one channel returns a retryable 429 that does *not* trigger auto-ban:
- **Failure**: Because `GetRandomSatisfiedChannel` uses pure weighted pseudo-random selection without an exclusion list of attempted channels, the same failing channel can be selected on retry 1, retry 2, and retry 3.

### Bug 3: Cross-Group Retry Starvation
- **Mechanism**: In `service/channel_select.go`, `crossGroupRetry` only advances to the next group when `priorityRetry >= common.RetryTimes`.
- **Failure**: If Group A has 1 priority tier, and `RetryTimes` is 3, retries 0, 1, and 2 all execute against Group A's sole tier before Group B is ever evaluated. If Group A's provider is hard down, all retry attempts are wasted on Group A.

### Bug 4: Memory Cache vs. Redis Multi-Node Inconsistency
- **Mechanism**: Channel status updates use `CacheUpdateChannelStatus()` in local memory, while full synchronization occurs periodically (`SyncChannelCache` every N seconds).
- **Failure**: In a multi-replica deployment, node 1 disables a dead channel, but nodes 2 and 3 continue routing traffic to it until their next periodic sync tick, amplifying client error rates.

---

## 5. Open-Source Router Architecture Analysis

### 5.1 Neilch07/newapi-routing
- **Architecture**: A declarative CLI and DB management tool designed specifically to mitigate New-API's single-tier retry bugs.
- **Core Pattern**: Forces channels into strictly ordered priority tiers via YAML declarations.
- **Key Insight**: Explicitly disallows retrying HTTP 408 (Request Timeout). It treats timeouts as potentially accepted by the upstream LLM, preventing costly duplicate generation.
- **Limitation**: Does not alter runtime Go code; relies on database manipulation of existing New-API priority fields.

### 5.2 LiteLLM (`BerriAI/litellm`)
- **Architecture**: In-process `Router` maintaining deployment pools, model groups, and active fallback dictionaries.
- **Core Pattern**:
  - `model_group`: Logical alias (e.g., `gpt-4o`) mapping to an array of heterogeneous provider deployments.
  - `fallbacks`: Explicit mapping: `{"gpt-4o": ["claude-3-5-sonnet", "gemini-1.5-pro"]}`.
- **Cooldown & Health**: Automatically demotes a deployment for a configurable cooldown window (default 5s–60s) on 429 or 5xx, using Redis-backed cooldown keys.
- **Context-Aware Fallback**: Catches `ContextWindowExceededError` and routes to a larger context window deployment automatically.
- **Streaming Guard**: Verifies whether chunks have been yielded; aborts fallback if stream has started.

### 5.3 Portkey Gateway (`Portkey-AI/gateway`)
- **Architecture**: Edge-optimized gateway evaluating declarative JSON control trees.
- **Core Pattern**: Nested routing topologies:
  ```json
  {
    "strategy": "fallback",
    "targets": [
      { "strategy": "loadbalance", "targets": [ ... ] },
      { "provider": "anthropic", "model": "claude-3-5-sonnet" }
    ]
  }
  ```
- **Intelligent Streaming Timeout**: Tracks Time-to-First-Token (TTFT). The request timeout is reset once the first chunk is received, preventing premature cancellation of lengthy generations.

### 5.4 License & Intellectual Property Analysis
- **Neilch07/newapi-routing**: MIT License.
- **LiteLLM**: MIT License (core router).
- **Portkey Gateway**: MIT License.
*All architectural patterns, state-machine designs, and failure taxonomies can be legally adopted and adapted.*

---

## 6. Route != Entitlement Group (Architectural Axiom)

### The Core Architectural Violation in Upstream
In New-API, `group` was historically used for two entirely orthogonal concepts:
1. **User Entitlement & Billing Tier**: Which users have access to `pro` vs. `default`, what quota multiplier applies (`GroupRatio`), and which models are unlocked.
2. **Channel Routing Tag**: Which upstream provider credentials service the request.

Overloading `group` with routing produces fatal flaws:
- If a user on the `default` entitlement group has an API key that falls back to a channel in the `vip` group, the system either blocks the request (entitlement failure) or unintentionally elevates the user's commercial tier.
- Pricing becomes unpredictable: users are surprised when an unexpected group multiplier is applied mid-flight.

### Conceptual Separation Principle
Tora strictly enforces a 3-tier decoupling:

```text
┌────────────────────────────────────────────────────────┐
│ 1. User / Subscription Group (Entitlement & Quotas)    │
│    - Defines commercial tier (e.g., "default", "pro") │
│    - Governs quota allocation & maximum allowed spend  │
└──────────────────────────┬─────────────────────────────┘
                           │ Authenticates & Entitles
┌──────────────────────────▼─────────────────────────────┐
│ 2. Route & Route Chain (Upstream Routing Policy)       │
│    - First-class entity: "High-Performance", "Eco"     │
│    - Defines target channel pools, priorities, weights │
│    - Defines route-level billing adjustment/surcharge  │
└──────────────────────────┬─────────────────────────────┘
                           │ Selects & Dispatches
┌──────────────────────────▼─────────────────────────────┐
│ 3. Channel (Upstream Credential & Infrastructure)      │
│    - API Key, Base URL, Rate Limits, Provider Type     │
│    - Pure delivery mechanism                           │
└────────────────────────────────────────────────────────┘
```

---

## 7. Proposed Route Model & Data Architecture

To maintain performance, we avoid building a second disjoint routing layer. Instead, we introduce a **first-class `Route` entity that compiles into optimized channel constraint filters**.

### 7.1 Database Schema (`model/route.go`)

```sql
CREATE TABLE routes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(64) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    kind VARCHAR(32) NOT NULL DEFAULT 'managed', -- 'managed', 'custom', 'byok'
    routing_policy VARCHAR(32) NOT NULL DEFAULT 'priority', -- 'priority', 'weighted'
    channel_ids TEXT NOT NULL, -- JSON array of channel IDs: [1, 5, 12]
    channel_tags TEXT, -- JSON array of string tags
    model_mapping TEXT, -- Optional model alias/rewrite mapping (JSON)
    cost_multiplier DECIMAL(5,4) NOT NULL DEFAULT 1.0000, -- Route-specific multiplier
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at DATETIME
);

CREATE INDEX idx_routes_slug ON routes(slug);
CREATE INDEX idx_routes_enabled ON routes(enabled);
```

### 7.2 API Key Route Chain Schema (`model/token.go` Enhancement)

```sql
-- Migration for tokens table:
ALTER TABLE tokens ADD COLUMN primary_route_id INTEGER DEFAULT 0;
ALTER TABLE tokens ADD COLUMN fallback_route_ids TEXT DEFAULT ''; -- JSON array: [2, 5]
```

In Go struct:
```go
type Token struct {
    // ... existing fields ...
    PrimaryRouteId    int    `json:"primary_route_id" gorm:"default:0;index"`
    FallbackRouteIds  string `json:"fallback_route_ids" gorm:"type:text"` // JSON array of int IDs
}

func (t *Token) GetRouteChain() []int {
    chain := make([]int, 0)
    if t.PrimaryRouteId > 0 {
        chain = append(chain, t.PrimaryRouteId)
    }
    if t.FallbackRouteIds != "" {
        var fallbacks []int
        if err := common.UnmarshalJsonStr(t.FallbackRouteIds, &fallbacks); err == nil {
            for _, id := range fallbacks {
                // Prevent duplicate route IDs in chain
                if !slices.Contains(chain, id) {
                    chain = append(chain, id)
                }
            }
        }
    }
    return chain
}
```

### 7.3 Referential Integrity & Deletion Rules
- **Rule 1 (Referential Integrity Check)**: A Route cannot be hard-deleted if referenced as a `primary_route_id` by any active Token.
- **Rule 2 (Graceful Soft-Delete & Fallback Bypass)**: If a route is soft-deleted or marked `enabled = false`:
  - When evaluating the key's route chain at runtime, the selector skips disabled/deleted routes and immediately advances to the next fallback route in the chain.
  - If all routes in the chain are disabled/deleted, the request is rejected with `HTTP 503 (ErrorCode: "NO_ACTIVE_ROUTES_IN_CHAIN")`.

---

## 8. Model Access & Entitlement Guards

### 8.1 Model Coverage Discrepancies
When Primary Route supports `gpt-4o` but Fallback Route #1 only supports `claude-3-5-sonnet`:
- If the client requested `gpt-4o`, Fallback Route #1 **cannot satisfy the request**.
- The selector must inspect whether candidate channels in Fallback Route #1 support `gpt-4o`. If none do, it immediately skips Fallback Route #1 and evaluates Fallback Route #2.
- It must **NEVER** silently mutate the user's requested model to an unrelated model unless an explicit `model_mapping` rule is configured on that route.

### 8.2 Authorization Constraint Invariant
A fallback route must pass every authorization constraint that a direct request must satisfy:
1. **API Key Model Limits**: If the API key specifies `model_limits = ["gpt-4o"]`, no route in the chain can route to any other model.
2. **User Subscription Entitlement**: If the user is on the `default` plan, a fallback route containing Pro-only channels is rejected by the subscription entitlement filter.
3. **Tenant & Provider Isolation**: Managed routes can never cross into BYOK channels, and BYOK routes can never route to Managed infrastructure.

---

## 9. Failure Taxonomy: Safe vs. Terminal Failures

The router must reject naive "fallback on any error" logic. Errors must be partitioned into strict semantic classes:

```text
                               Upstream Error Occurs
                                         │
        ┌────────────────────────────────┴────────────────────────────────┐
        ▼                                                                 ▼
[Non-Retryable / Terminal]                                      [Eligible for Fallback]
  - 400 Bad Request                                               - 429 Rate Limit
  - 404 Model Not Found                                           - 503 Service Unavailable
  - 422 Unprocessable Entity                                      - 502 Bad Gateway
  - Local Auth / Quota Failure                                    - 504 Gateway Timeout
  - Context Window Exceeded                                       - Pre-Response Network Drop
  - Safety Rejection                                              - Upstream 401/403 (Invalid Key)
        │                                                                 │
        ▼                                                                 ▼
Return Immediate Error to Client                                Check Response Commitment
                                                                          │
                                                 ┌────────────────────────┴────────────────────────┐
                                                 ▼                                                 ▼
                                        [Uncommitted (0 bytes)]                           [Committed (> 0 bytes)]
                                                 │                                                 │
                                                 ▼                                                 ▼
                                      Execute Next Route in Chain                        Terminate Stream Gracefully
```

### Detailed Failure Classification Table

| Failure Type | HTTP Code / Trigger | Fallback Allowed? | Action & State Handling |
| :--- | :---: | :---: | :--- |
| **Upstream Rate Limit** | `HTTP 429` | **YES** | Increment route attempt; apply channel cooldown; advance route. |
| **Upstream Overload** | `HTTP 503`, `502` | **YES** | Advance to next channel/route. |
| **Gateway Timeout (Pre-Response)** | `HTTP 504` / Dial Timeout | **YES** | Only if 0 bytes received; advance route. |
| **Provider Auth Failure** | `HTTP 401`, `403` | **YES (Internal)** | Auto-ban failing upstream channel; failover to next channel/route. |
| **Malformed Request** | `HTTP 400` | **NO** | Terminal. Client error; retry would produce identical failure. |
| **Model Not Found** | `HTTP 404` | **NO** | Terminal. Upstream does not support model. |
| **Context Window Exceeded**| `HTTP 400` / Provider Error| **NO** | Terminal (unless explicit model-escalation route configured). |
| **Local Tora Auth Failure**| `HTTP 401` | **NO** | Terminal. Invalid Tora API key. |
| **Local Quota Exhausted** | `HTTP 402` / `403` | **NO** | Terminal. User or token balance depleted. |
| **Mid-Stream Network Drop** | Connection reset after tokens | **NO** | Terminal. Response already committed; cannot restart cleanly. |

---

## 10. Streaming Safety & Response Commitment

### 10.1 The Commitment Invariant
> **Invariant**: A client connection is committed the instant the HTTP response headers or the first byte of payload are flushed to the client socket. Once committed, no route fallback may occur.

### 10.2 Race Condition in Current Upstream
In standard New-API streaming handlers (`OaiStreamHandler`), `c.Writer.WriteHeader(200)` and `helper.FlushWriter(c)` are called as soon as the upstream connection is established. If the upstream provider yields a 500 error or disconnects after sending an initial chunk, the controller's outer loop cannot rewind the HTTP state. Attempting to start a fallback request writes duplicate headers, crashes the Gin context, and corrupts the client's SSE parser.

### 10.3 Safe Fallback Architecture for Streaming
To support transparent streaming fallback:
1. **Deferred Flush Wrapper**: Wrap `gin.ResponseWriter` with a `CommitDetectingWriter`:
   ```go
   type CommitDetectingWriter struct {
       gin.ResponseWriter
       committed   bool
       buffer      bytes.Buffer
       headerHold  bool
   }
   ```
2. **First-Token Gate**:
   - Upstream connect & headers received: buffer headers locally; do *not* flush to downstream client.
   - Wait for first valid SSE data event containing text tokens or tool calls.
   - If upstream fails or emits error *before* the first data token: discard buffer, abort channel, and **execute route fallback cleanly**.
   - As soon as the first token arrives: flush HTTP 200 headers, write first chunk, set `committed = true`. All subsequent errors become terminal.

---

## 11. Duplicate Billing & Idempotency Analysis

### 11.1 The Ambiguous Timeout Hazard
Consider the following sequence:
```text
Tora -> Provider A: POST /v1/chat/completions (Prompt: 5,000 tokens)
Provider A: Accepts request, starts inference processing
[Network Delay / Timeout: 15 seconds elapse]
Tora: HTTP client timeout triggers before response headers received
Tora: Decides to fallback -> Provider B: POST /v1/chat/completions
Provider B: Generates response successfully -> Returns to User
Provider A: Finishes inference, charges Tora's account balance!
```
**Consequence**: The customer receives one response, but Tora pays for two full completions.

### 11.2 Mitigation Policies
1. **Aggressive Connection Timeout vs. Conservative Read Timeout**:
   - Dial/Connect Timeout: 5 seconds (safe to retry if connection refused).
   - TLS Handshake Timeout: 5 seconds.
   - First-Byte Timeout: Configurable per route (e.g., 20s).
2. **Upstream Idempotency Keys**: Where supported (e.g., Anthropic `idempotency-key` header, Stripe), pass the client `request_id`.
3. **Non-Retryable Read Timeouts**: If a provider does not support idempotency keys and read timeout expires after headers, treat as non-retryable by default unless the route explicitly enables `speculative_fallback`.

---

## 12. Media & Asynchronous Tasks

### 12.1 Divergence from Chat Semantics
Asynchronous media jobs (DALL-E image generation, Midjourney tasks, Suno audio, Sora video) follow a two-phase submit-and-poll pattern:
1. **Submission Phase**: `POST /v1/images/generations` or `POST /mj/submit/imagine`.
2. **Polling Phase**: `GET /mj/task/{id}/fetch`.

### 12.2 Risk of Route Fallback on Task Submission
If the submission request times out after the upstream provider accepted the job:
- Falling back to a second image channel generates **two images** and incurs double the non-refundable credit cost.
- Polling routes cannot be swapped mid-flight because task IDs are strictly bound to the specific upstream channel instance (`channel_id`).

### 12.3 Recommendation
- **Phase R10 V1 Scope**: **Restrict Route Fallback to Synchronous Chat, Text, and Embedding APIs.**
- **Media / Async Task Fallback**: Defer to V2, requiring provider-side idempotency keys and pinned-channel polling semantics.

---

## 13. Authoritative Billing Formula & Route Composition

### 13.1 Avoiding Multiplier Explosion
Tora currently calculates quota consumption using:
$$\text{BaseQuota} = (\text{PromptTokens} \times \text{InputRatio}) + (\text{CompletionTokens} \times \text{OutputRatio})$$
$$\text{FinalQuota} = \text{BaseQuota} \times \text{ModelRatio} \times \text{GroupRatio}$$

If a Route also introduced an arbitrary multiplier, the calculation would become opaque and unpredictable to customers:
$$\text{Quota} \stackrel{?}{=} \text{BaseQuota} \times \text{ModelRatio} \times \text{GroupRatio} \times \text{RouteMultiplier}$$

### 13.2 Unified Billing Rule
1. **Settlement is Determined by the Winning Route**: The client is billed based on the route and channel that **successfully completed the request**, never the primary route that failed.
2. **Transparent Route Surcharge**:
   - Each Route defines an explicit `CostMultiplier` (default `1.0`).
   - The authoritative settlement formula is:
     $$\boxed{\text{SettledQuota} = \left\lceil \text{BaseQuota} \times \text{ModelRatio} \times \text{UserGroupRatio} \times \text{RouteCostMultiplier} \right\rceil}$$
3. **Pre-Consumption Ceiling**:
   - Pre-charge must reserve quota based on the **maximum possible cost** across the entire route chain:
     $$\text{PreChargeQuota} = \max_{r \in \text{RouteChain}} (\text{EstimatedQuota}_r)$$
   - Upon successful execution on Route $k$, the difference between $\text{PreChargeQuota}$ and $\text{SettledQuota}_k$ is immediately refunded to the user's balance.

---

## 14. BYOK Security Invariant

> **BYOK Boundary Rule**: A request initiated under BYOK mode must NEVER silently fallback to a Managed route or Managed provider channel under any circumstance.

### Enforcement Mechanism
In `controller/relay.go`:
```go
if c.GetBool("is_byok") || common.GetContextKeyBool(c, constant.ContextKeyIsBYOK) {
    // BYOK failure is terminal. Zero fallback to platform-funded infrastructure.
    break
}
```
- A BYOK API key can only configure route chains composed exclusively of user-owned BYOK providers.
- Admin and validation endpoints must reject any route chain that mixes `kind = "byok"` and `kind = "managed"` routes.

---

## 15. Observability & Audit Trail

Routing sequences must be completely transparent to administrators while preventing any credential or sensitive prompt leakage.

### 15.1 Structured Route Log Schema (`model/log.go` `other` metadata)
```json
{
  "routing": {
    "chain": [
      {
        "attempt": 0,
        "route_id": 1,
        "route_name": "Primary-Claude",
        "channel_id": 45,
        "status_code": 429,
        "latency_ms": 240,
        "failure_reason": "upstream_rate_limited"
      },
      {
        "attempt": 1,
        "route_id": 3,
        "route_name": "Fallback-DeepSeek",
        "channel_id": 82,
        "status_code": 200,
        "latency_ms": 1150,
        "outcome": "success"
      }
    ],
    "winning_route_id": 3,
    "winning_channel_id": 82,
    "fallback_triggered": true,
    "total_attempts": 2
  }
}
```

### 15.2 Masking & Redaction Rules
- **Prohibited in logs**: Upstream API keys, `Authorization` headers, BYOK encrypted secrets, user prompt text, completions.
- **Allowed in logs**: Route IDs, channel IDs, HTTP status codes, latency timings, token counts, error message classifications.

---

## 16. Circuit Breakers & Cooldown Engine

### 16.1 Flaw in Current Auto-Ban
Current New-API auto-ban is binary: a channel is either `Enabled (1)` or `AutoDisabled (3)`. Once auto-disabled, it requires manual admin re-enablement or an infrequent background sweep.

### 16.2 Dynamic Cooldown Specification
We implement a lightweight in-memory and Redis-backed **Channel Cooldown State**:
- On `HTTP 429` with `Retry-After` header: Cooldown channel for `min(RetryAfter, 60s)`.
- On transient `503`: Cooldown channel for 10 seconds.
- During cooldown, `GetRandomSatisfiedChannel` skips the channel without altering its database status or modifying priority slices.

---

## 17. Cache Invalidation & Multi-Node Synchronization

To ensure immediate consistency across multi-replica deployments:
1. **Redis Pub/Sub Invalidation Topic**: `tora:cache:route:invalidate`.
2. Whenever an administrator edits a Route or an API key updates its route chain:
   - Publishes message: `{"type": "route_update", "id": 12}`.
   - All worker nodes receive message and invalidate local memory cache immediately.
3. Fallback: 30-second TTL on in-memory route compilation structs.

---

## 18. UI / UX Design Specifications

### 18.1 API Key Management UX
In the Mobile App and Web Console (API Keys -> Create / Edit):

```text
┌──────────────────────────────────────────────────────────────┐
│ Routing Policy                                               │
│                                                              │
│ Primary Route *                                              │
│ [ High-Performance Claude 3.5 Sonnet (Direct)          ▼ ]   │
│                                                              │
│ Fallback Routes (Evaluated in order)                         │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ 1. DeepSeek V3 High-Speed [Managed]         [↑] [↓] [✕] │ │
│ ├──────────────────────────────────────────────────────────┤ │
│ │ 2. OpenAI GPT-4o Mini Backup                [↑] [↓] [✕] │ │
│ └──────────────────────────────────────────────────────────┘ │
│                                                              │
│ [ + Add Fallback Route ]                                     │
│                                                              │
│ ℹ Requests automatically fail over if the primary route is   │
│   rate-limited or down. You are billed based on the route    │
│   that successfully handles your request.                   │
└──────────────────────────────────────────────────────────────┘
```

**UX Validation Rules**:
- Primary Route is mandatory.
- Fallback routes are optional (0 to 5 allowed).
- No duplicate routes allowed in the chain (enforced in dropdown options).
- Real-time indicator showing whether all models in the API key's model limit are supported across the chain.

---

## 19. Admin Route Management UX

A dedicated Admin Console interface (`/admin/routes`):
- **List Routes**: Name, Slug, Channel Count, Policy, Multiplier, Status, Actions.
- **Create / Edit Route Modal**:
  - Route Name & Unique Slug.
  - Channel Binding: Select specific channels or channel tags.
  - Routing Strategy: Strict Priority Fallback vs. Weighted Random.
  - Cost Multiplier: Float (e.g., `1.00`, `1.20`).
  - Active Toggle.

---

## 20. Migration Strategy & Backward Compatibility

To guarantee zero downtime and 100% backward compatibility for existing users:
1. **Schema Non-Null Defaults**: `primary_route_id` defaults to `0`, `fallback_route_ids` defaults to `""`.
2. **Legacy Fallback Logic**:
   - If `token.primary_route_id == 0`: The router executes the existing legacy channel selection using `token.Group` and `token.AutoGroups`.
   - Existing API keys behave with 100% fidelity to their current configuration.
3. **Automatic Route Seeding**:
   - A migration script generates default system routes (`default-route`, `pro-route`) mapping to the legacy channel groups.
   - Admins can migrate existing tokens incrementally via an opt-in migration command.

---

## 21. Comprehensive Test Matrix

| ID | Test Scenario | Expected Behavior |
| :--- | :--- | :--- |
| **TC-01** | Primary route succeeds | Request terminates on attempt 0; billed at primary route rate. |
| **TC-02** | Primary 429 -> Fallback 1 succeeds | Seamless failover; attempt 1 succeeds; billed at Fallback 1 rate. |
| **TC-03** | Primary 503 -> Fallback 1 503 -> Fallback 2 succeeds | Two failovers; attempt 2 succeeds; billed at Fallback 2 rate. |
| **TC-04** | Primary 400 Bad Request | No fallback; immediate 400 returned to client; zero token quota billed. |
| **TC-05** | Local Tora 401 Unauthorized | No fallback; immediate 401 returned. |
| **TC-06** | User Quota Depleted | No fallback; immediate 403 / 402 returned. |
| **TC-07** | Fallback route lacks requested model | Fallback skipped automatically; advances to next route in chain. |
| **TC-08** | All routes in chain fail | Returns last upstream error code with `X-Tora-Route-Attempts: 3`. |
| **TC-09** | Channel auto-disabled during retry | Priority tier list does NOT corrupt; subsequent retries evaluate correctly. |
| **TC-10** | Concurrent requests on failing route | All concurrent requests failover independently without race conditions. |
| **TC-11** | Route disabled while in-flight | In-flight completes; subsequent requests bypass disabled route. |
| **TC-12** | Route deleted while referenced by key | Key skips deleted route; logs warning; uses next fallback or returns 503. |
| **TC-13** | Streaming fails before first token (0 bytes) | Transparent fallback to next route; headers deferred until first token. |
| **TC-14** | Streaming fails after token emitted | Fallback aborted; stream terminates with error chunk; no secondary call. |
| **TC-15** | Client cancels during failover | Context cancellation terminates loop immediately; quota refunded. |
| **TC-16** | BYOK request fails | Terminal failure; NEVER routes to Managed channels. |
| **TC-17** | Pro entitlement enforcement | Free user cannot reach Pro-only channel via fallback route. |
| **TC-18** | Route cost multiplier settlement | User charged exact multiplier corresponding to winning route. |

---

## 22. Independent Security & Threat Model Review

An adversarial security audit of this architecture identifies seven key threat vectors and their mitigations:

### 1. Entitlement Bypass via Fallback Injection
- **Threat**: An attacker on a `default` (Free) tier creates an API key with a Primary Route on `default` and a Fallback Route pointing to expensive `pro` models, intentionally triggering a 429 on Primary to gain unauthorized access to Pro infrastructure.
- **Mitigation**: The route compilation engine checks user entitlement on *every* candidate channel in *every* route in the chain. If a channel violates user group permissions, it is permanently pruned from the candidate set before selection.

### 2. Cheaper-Route Abuse & Billing Arbitrage
- **Threat**: A user configures an expensive route first, then a cheaper fallback, but relies on predictable quota pre-charge calculations to bypass minimum wallet balance checks.
- **Mitigation**: Quota pre-charge checks verify that the user has sufficient balance for the *most expensive* route in the chain.

### 3. Retry Amplification & Denial-of-Wallet
- **Threat**: A client intentionally sends complex prompts that cause upstream timeouts, causing Tora to fan out retries across 5 providers, charging the platform multiple times.
- **Mitigation**:
  - Maximum route chain length capped at 3 fallbacks (4 total attempts).
  - Pre-response timeout does not retry unless explicitly configured.
  - Overall request timeout deadline context spans the entire chain.

### 4. Admin vs. User Authorization Escalation
- **Threat**: Regular users calling `POST /api/routes` to create routes pointing to arbitrary private channels.
- **Mitigation**: Route CRUD endpoints are strictly restricted to `RoleAdmin` (`middleware.AdminAuth`). Regular users can only select from pre-approved public routes.

---

## 23. Implementation Plan & Staging

```text
Phase 1: Database & Core Models
  ├── Add Route model & migrations
  ├── Add primary_route_id and fallback_route_ids to Token
  └── Unit tests for Route compilation & chain serialization

Phase 2: Channel Selection & Cooldown Engine
  ├── Implement Route-aware candidate filter
  ├── Add in-memory channel cooldown tracking (Retry-After support)
  └── Fix priority tier mutation bug in channel_cache.go

Phase 3: Relay Controller & Streaming Protection
  ├── Implement CommitDetectingWriter in streaming relay
  ├── Integrate route chain iteration in controller/relay.go
  └── Enforce BYOK terminal boundary

Phase 4: Billing Settlement & Admin API
  ├── Implement winning-route settlement & refund reconciliation
  ├── Build Admin Route CRUD API (/api/routes)
  └── Build User Route Selection validation

Phase 5: UI & E2E Validation
  ├── Mobile / Web API Key route chain management UI
  ├── Admin Route management dashboard
  └── Full test suite execution across all 18 test matrix scenarios
```

---

## 24. Final Decision & Status

Following extensive source auditing of `feat/formobile`, comparative analysis with upstream `QuantumNous/new-api`, review of open-source router architectures (LiteLLM, Portkey, Newapi-routing), threat modeling, and billing verification:

**FINAL STATUS: READY TO IMPLEMENT FIRST-CLASS ROUTES**
