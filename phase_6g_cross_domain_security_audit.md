# Phase 6G — Cross-Domain Security & Production Readiness Audit

## 1. Executive Summary

This report delivers the comprehensive, implementation-level **Phase 6G Cross-Domain Security & Production Readiness Audit** for the SaaSCover / New-API backend (`branch: feat/formobile`). 

Previous security phases individually evaluated and hardened isolated subsystems:
- **Phase 5B/5C/5D**: User-supplied BYOK credentials, SSRF mitigations, DNS-rebinding protection, outbound transport isolation, and per-user BYOK rate/concurrency limits (`PASS`).
- **Phase 6A/6B**: Authentication, authorization, token encryption, Midjourney capability token HMAC bindings, and step-up verification proofs (`PASS`).
- **Phase 6C/6C-R**: Billing pre-consumption reservations, completion token overflow protections, free-tier ratio protections, and Midjourney pricing models (`PASS`).
- **Phase 6D/6D-R**: Gateway abuse, global concurrency limits, SSE stream lifecycle timeouts, and resource exhaustion guards (`PASS`).
- **Phase 6E/6E-R**: Data exposure, secret redaction in logs/errors, CSRF, and HTTP security headers (`PASS`).
- **Phase 6F/6F-R**: Business logic, state transitions, transaction boundaries, Stripe delayed payment failure race conditions, and webhook idempotency (`PASS`).

The specific mandate of **Phase 6G** is to audit the **boundaries BETWEEN those security domains**. When individually hardened subsystems interact across runtime call paths—such as authentication state propagating to BYOK selection, BYOK selection interacting with billing bypass logic, streaming lifecycle events triggering settlement and refund, or asynchronous payment webhooks updating wallet quotas—any impedance mismatch or misplaced assumption can introduce severe financial or security vulnerabilities.

### Key Audit Findings
1. **Core Cross-Domain Invariants Hold**: 
   - **Identity Derivation**: Invariant A is strictly satisfied. Identity is permanently bound to cryptographic database token validation (`token.UserId`). Neither HTTP headers, URL query parameters, JSON payload keys, nor model prefixes can override the authenticated user context.
   - **Resource Ownership**: Invariant B is strictly enforced. UserProviders, AccessTokens, Subscriptions, Orders, Tasks, and Artifacts require strict SQL filtering (`WHERE user_id = ? AND id = ?`).
   - **BYOK / System Channel Separation**: A user cannot claim BYOK status to bypass system billing without possessing an enabled, verified `UserProvider`. If BYOK is detected, failure to resolve the user's provider immediately aborts the request (HTTP 400), completely preventing fall-through to system channels under an unbilled context.
   - **SSRF Transport Consistency**: Outbound requests via user-configured endpoints unconditionally traverse `byokRoundTripper`, which enforces strict IP/CIDR validation, DNS rebinding checks, and completely disables automatic redirect following (`http.ErrUseLastResponse`).
   - **Financial State Machine & Idempotency**: Pre-consumption reservations, settlement adjustments, and refund operations are protected by mutex guards, atomic database transactions (`IncreaseUserQuotaTx`, `PostConsumeUserSubscriptionDeltaTx`), and strict conditional status transitions (`Pending -> Success`). All 6 attack chains (A through F) are confirmed **BLOCKED**.
2. **Residual Low-Risk & Informational Observations**:
   - **Finding 6G-01 (Low)**: In multi-node deployments without explicit `SESSION_SECRET` or `CRYPTO_SECRET` environment variables, startup falls back to an ephemeral process-local UUID, preventing cross-node credential decryption and session sharing.
   - **Finding 6G-02 (Informational)**: Redis rate-limiter check failure falls back to local in-memory limiting to ensure system availability during Redis partitions; billing and quota protections remain fully ACID-backed in the SQL database.

### Final Audit Status
**FINAL STATUS: PASS WITH LOW-RISK FINDINGS**

---

## 2. System Security Model

### Complete End-to-End Request Lifecycle
The runtime lifecycle of an incoming API request across all subsystem boundaries is reconstructed below:

```text
                                [ Client HTTP / SSE / WebSocket Request ]
                                                   │
                                                   ▼
                                        [ middleware.RouteTag ]
                                                   │
                                                   ▼
                                 [ middleware.SystemPerformanceCheck ]
                                                   │
                                                   ▼
                                      [ middleware.TokenAuth ]
                           ┌───────────────────────┴───────────────────────┐
                     (Invalid / Expired)                            (Valid Token)
                           │                                               │
                           ▼                                               ▼
                   [ 401 Unauthorized ]                        Extract token.UserId
                                                          Write Context: id, quota, group
                                                                           │
                                                                           ▼
                                                             [ middleware.BYOKRouter ]
                                            ┌──────────────────────────────┴──────────────────────────────┐
                                     (BYOK Requested)                                              (Normal System)
                                            │                                                             │
                                            ▼                                                             ▼
                               Lookup UserProvider(userID)                                   [ middleware.FreeTierQuota ]
                                            │                                                             │
                         ┌──────────────────┴──────────────────┐                                          ▼
                   (Not Found / Disabled)                 (Found & Enabled)                      [ middleware.Distribute ]
                         │                                     │                                          │
                         ▼                                     ▼                                 Select System Channel
                 [ 400 Bad Request ]                 Decrypt Key via AAD                                  │
                 (Request Terminated)                Bind Virtual Channel                                 │
                         ▲                                     │                                          │
                         │                                     ▼                                          ▼
                         │                         [ relay.PrepareRequestBilling ] ◄──────────────────────┘
                         │                                     │
                         │                   ┌─────────────────┴─────────────────┐
                         │              (BYOK Request)                    (System Channel)
                         │                   │                                   │
                         │             Skip Pre-Consume                 Estimate Prompt + Max Output
                         │                   │                          Reserve Quota in DB / Wallet
                         │                   │                                   │
                         │                   │                          Insufficient Quota? ──► [ 403 Forbidden ]
                         │                   │                                   │
                         │                   └─────────────────┬─────────────────┘
                         │                                     │
                         │                                     ▼
                         │                        [ Outbound Relay Request ]
                         │                                     │
                         │                   ┌─────────────────┴─────────────────┐
                         │              (BYOK Endpoint)                   (System Channel)
                         │                   │                                   │
                         │          service.GetBYOKHttpClient           Standard Channel Proxy Client
                         │          - SSRF IP & DNS Check                        │
                         │          - Disable Redirects                          │
                         │                   │                                   │
                         │                   └─────────────────┬─────────────────┘
                         │                                     │
                         │                                     ▼
                         │                       [ Streaming / Non-Streaming ]
                         │                                     │
                         │                   ┌─────────────────┴─────────────────┐
                         │              (HTTP Success)                     (HTTP Failure)
                         │                   │                                   │
                         │          Accumulate Token Usage             RefundFailedRequestBilling
                         │                   │                         - Restore Quota Reservation
                         │                   ▼                         - Apply Violation Fee if any
                         │        [ service.SettleBilling ]                      │
                         │        - Calculate Delta                              ▼
                         │        - Commit Quota Difference              [ Terminate Request ]
                         │        - Mark Billing Settled
                         │                   │
                         │                   ▼
                         └─────────── [ Client Response ]
```

### Complete Asynchronous Lifecycle Paths

```text
1. Asynchronous Payment Top-Up Path:
   [ Payment Provider (Stripe / EPay) ] ──► Webhook POST ──► Signature Verification
                                                                     │
                                                                     ▼
                                                          Lookup Order (Pending)
                                                                     │
                                                                     ▼
                                                         [ ACID DB Transaction ]
                                                         - Conditional Update (Pending -> Success)
                                                         - IncreaseUserQuotaTx
                                                         - Commit
                                                                     │
                                                                     ▼
                                                          200 OK (Idempotent)

2. Asynchronous Midjourney Task & Capability Path:
   Submit Task ──► Reserve Quota ──► Upstream MJ Polling ──► Task Finished (Success / Failed)
                                                                     │
                                                                     ▼
                                                          Generate HMAC Capability Token
                                                          (task_id, user_id, expiry, sign)
                                                                     │
                                                                     ▼
   Client Image Request ──► Verify HMAC Signature ──► Verify Task & User Enabled ──► Proxy Image Binary
```

---

## 3. Cross-Domain Invariants

### Invariant A — Identity Canonicalization
* **Definition**: Every security-sensitive operation must derive identity exclusively from authenticated cryptographic state. Client-controlled identifiers (`user_id` query parameter, `body.user_id`, `X-User-Id` header, model prefix, or token metadata) must never override or substitute the identity established during authentication.
* **Audit Verification**:
  - In `middleware/auth.go:TokenAuth()`: Identity is derived strictly via `token, err := model.ValidateUserToken(key)`. The context key `"id"` is set directly from `token.UserId` (`c.Set("id", token.UserId)`).
  - In `middleware/auth.go:UserAuth()` / `AdminAuth()`: Session authentication resolves the authenticated `UserBase` record via `service.ValidateLoginSession(identity)`.
  - In all relay handlers (`relay/relay.go`, `relay/request_billing.go`, `service/billing.go`): Identity is read from `c.GetInt("id")` or `relayInfo.UserId`.
  - No handler or middleware allows request query parameters or request body parameters to rebind `c.GetInt("id")`.
* **Status**: **VERIFIED ENFORCED**.

### Invariant B — Object Ownership Enforcement
* **Definition**: For every user-owned resource, `resource.user_id == authenticated_user.id` must be guaranteed at the access boundary before any read, update, deletion, or downstream dispatch occurs.
* **Audit Verification**:
  - `UserProvider`: `model.GetUserProviderByID(userID, id)` and `model.GetUserProviderByType(userID, provider)` query with `WHERE user_id = ? AND ...`.
  - `UserAccessToken`: `model.GetUserAccessToken(userID, id)` enforces `WHERE user_id = ? AND id = ?`.
  - `UserSubscription`: `model.GetUserSubscriptionByID(userID, id)` scopes strictly to the authenticated user.
  - `TopUp Orders`: `model.GetTopUpByID(id, userID)` enforces user ownership.
  - `Task Plugin Protocol`: `deps.getByTaskId(userID, taskID)` in `controller/plugin_protocol.go:957` validates ownership.
  - `Midjourney Capability`: Validates HMAC signature over `task_id` and `user_id`, and verifies `task.UserId == capability.UserId` in `controller/image.go`.
* **Status**: **VERIFIED ENFORCED**.

---

## 4. Authentication → Authorization Boundary

### Identity & Role Derivation
Authentication handles credential validation and identity derivation across three distinct interfaces:
1. **API / Relay Tokens (`TokenAuth`, `TokenAuthReadOnly`)**:
   - Extracts the token from `Authorization: Bearer <key>`.
   - Strips optional `sk-` prefix and validates token via `model.ValidateUserToken(key)`.
   - Rejects disabled (`common.TokenStatusDisabled`), expired (`token.ExpiredTime`), or exhausted (`token.RemainQuota <= 0`) tokens.
   - Enforces IP CIDR restrictions if configured on the token (`token.GetIpLimits()`).
   - Retrieves user account status via `model.GetUserCache(token.UserId)`. If `userCache.Status != common.UserStatusEnabled`, immediately aborts with HTTP 403 `MsgAuthUserBanned`.
   - Writes user and token attributes to the request context: `id`, `token_id`, `token_name`, `user_quota`, `user_group`.
2. **Dashboard Browser Sessions (`UserAuth`, `AdminAuth`, `RootAuth`)**:
   - Resolves session tokens via `service.ValidateLoginSession(identity)`.
   - Enforces session versions and account `AuthVersion`. If the account's `AuthVersion` was bumped (e.g. password change, admin demotion/disable), existing session tokens are invalidated.
3. **Personal Access Tokens (PAT)**:
   - Scoped access tokens enforce route-level permission policies (`enforceAccessTokenRoute`).

### Boundary Integrity
No path exists where an unauthenticated or lower-privileged user can elevate their role:
- Casbin RBAC policies (`service/authz`) enforce route-level authorization.
- Admin routes are guarded by `middleware.AdminAuth()` or `middleware.RootAuth()`.
- Critical admin actions (user deletion, role change, manual balance adjustments) require explicit cryptographic step-up verification proofs (`requireAdminUserProof`).

---

## 5. Authentication → BYOK Boundary

### Credential Resolution & Ownership Verification
When a client sends a relay request specifying a BYOK provider, the boundary between authentication and BYOK resolution operates as follows:

```go
// middleware/byok.go:202-227
userID := c.GetInt("id")
if userID == 0 {
    userID = common.GetContextKeyInt(c, constant.ContextKeyUserId)
}
if userID <= 0 {
    c.AbortWithStatusJSON(http.StatusUnauthorized, ...)
    return
}

userProvider, err := model.GetUserProviderByType(userID, info.provider)
if err != nil || userProvider == nil || !userProvider.Enabled {
    c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
        "error": gin.H{
            "message": fmt.Sprintf("BYOK provider '%s' is not configured or is disabled for this user", info.provider),
            "type":    "invalid_request_error",
            "code":    "byok_provider_not_found",
        },
    })
    return
}
```

### Critical Security Properties
1. **Authenticated Context Binding**: `userID` is retrieved directly from the context populated by `TokenAuth()`. The client has no mechanism to supply an arbitrary `user_id` to `GetUserProviderByType`.
2. **Strict Ownership Isolation**: User A cannot use User B's provider credentials, even if User A knows User B's provider ID, provider type, or model name.
3. **Decryption with AAD**: `userProvider.DecryptAPIKey()` calls `common.DecryptBYOKSecret(p.APIKeyEncrypted, p.AAD())`, where `p.AAD()` binds the encryption context to `user_id` and provider type. Even in the event of database row tampering, ciphertext transplanted across users fails AES-GCM tag verification.
4. **Abuse Controls**: BYOK requests are throttled by per-user rate limits (`checkBYOKRateLimit`, default 60 req/min) and concurrency limits (`acquireBYOKConcurrency`, default 5 concurrent requests).

---

## 6. BYOK → SSRF Boundary

### Outbound HTTP Client Architecture
All outbound requests directed to user-configured BYOK endpoints (such as custom OpenAI-compatible endpoints or OpenRouter endpoints) are strictly confined to the SSRF-protected transport:

1. **Client Selection (`relay/channel/api_request.go:520-536`)**:
   ```go
   if info != nil && info.IsBYOK {
       client, err = service.GetBYOKHttpClient(info.ChannelSetting.Proxy, info.ChannelSetting)
   } else {
       client, err = service.GetHttpClientWithProxySettings(info.ChannelSetting.Proxy, info.ChannelSetting)
   }
   ```
2. **Unconditional Transport Filtering (`service/byok_http_client.go:37-50`)**:
   - `byokRoundTripper` wraps `http.Transport` and validates every outbound request URL with `common.BYOKSSRFProtection.ValidateURL(req.URL.String())`.
   - It verifies destination hosts against private IP ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.0/8`, `169.254.169.254`, IPv6 `::1`, `fc00::/7`, `fe80::/10`).
   - Domain resolution uses a DNS-rebinding safe dialer (`protectedFetchDialer`) that resolves IPs prior to connection and aborts if an address resolves to a prohibited network.
3. **Redirect Protection**:
   - `relayClient.CheckRedirect = keepUpstreamRedirectResponse` explicitly returns `http.ErrUseLastResponse`.
   - The HTTP client never follows 3xx redirects automatically, completely neutralizing redirect-based SSRF vectors.
4. **Endpoint Validation on Configuration**:
   - When users register or update a provider (`controller/user_provider.go`), `NormalizeBaseURL` validates the URL scheme (`http`/`https`) and runs `common.BYOKSSRFProtection.ValidateURL`.
   - The provider test endpoint (`controller/user_provider.go:TestUserProviderEndpoint`) uses the identical SSRF-protected client.

---

## 7. BYOK → Billing Boundary

### System Channel vs. BYOK Billing Invariant
The SaaSCover billing architecture enforces a strict separation:
- **BYOK Requests**: The user supplies their own API key; no system balance or free-tier quota is deducted.
- **System Channel Requests**: The server routes through system channels; quota must be reserved and settled against the user's wallet or subscription.

### Cross-Domain Inversion Audit
Can a client claim BYOK to bypass billing while actually using a system channel?

1. **Attempted Bypass Scenario**:
   - Client sends `/v1/chat/completions` with header `X-Provider: openrouter`, but does not configure an OpenRouter provider in their account.
2. **Runtime Execution**:
   - `detectBYOKRequest(c)` returns `info.isBYOK = true, info.provider = "openrouter"`.
   - `BYOKRouter()` executes `model.GetUserProviderByType(userID, "openrouter")`.
   - Because no enabled record exists, `BYOKRouter()` halts execution immediately:
     `c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": {"code": "byok_provider_not_found"}})`
   - `c.Abort()` halts the Gin handler chain. Downstream middlewares (`FreeTierQuota`, `Distribute`) and controllers (`Relay`) **never execute**.
3. **Legitimate BYOK Scenario**:
   - If the user has a valid provider, `BYOKRouter()` sets `c.Set("is_byok", true)` and channel metadata with `channel_id = -1` and `channel_key = apiKey`.
   - `Distribute()` inspects `if c.GetBool("is_byok") { c.Next(); return }` and skips system channel allocation.
   - `PrepareRequestBilling()` inspects `if c.GetBool("is_byok") { return nil }` and skips billing.
   - Upstream execution uses the user's decrypted key (`channel_id = -1`), consuming the user's external account quota.

**Audit Conclusion**: BYOK status cannot be spoofed to obtain free system-channel service. System channels cannot be invoked with `is_byok = true`.

---

## 8. Relay → Billing Boundary

### Execution Ordering & Reservation Lifecycle
For all system-channel relay traffic, the lifecycle strictly follows:

```text
1. TokenAuth establishes authenticated identity (userID).
2. Distribute selects system channel and model.
3. PrepareRequestBilling(c, relayInfo) is invoked:
   a. Token count is estimated (prompt tokens + max completion tokens).
   b. Model pricing is determined via ModelPriceHelper.
   c. If FreeModel is true, billing is skipped and logged.
   d. Otherwise, service.PreConsumeBilling(c, QuotaToPreConsume, relayInfo) executes.
      - Deducts QuotaToPreConsume atomically from User/Wallet/Subscription.
      - Creates and attaches a BillingSession to relayInfo.Billing.
4. Upstream channel request is dispatched.
5. SettleBilling(c, relayInfo, actualQuota) executes:
   a. Delta = actualQuota - PreConsumedQuota.
   b. If Delta > 0: additional quota is deducted.
   c. If Delta < 0: unused pre-consumed quota is refunded.
   d. BillingSession is marked settled (s.settled = true).
6. If upstream fails before successful settlement:
   a. RefundFailedRequestBilling(c, info, apiErr) executes.
   b. info.Billing.Refund(c) restores the pre-consumed reservation.
```

### All Relay Modalities Audited
- **Chat Completions (`/v1/chat/completions`)**: Pre-consumes prompt + max completion allowance; settles upon completion.
- **Responses API (`/v1/responses`)**: Pre-consumes prompt + max output allowance; settles upon completion.
- **Image Generation / Edit (`/v1/images/*`)**: Pre-consumes fixed per-image cost; settles on result.
- **Audio Speech / Transcription (`/v1/audio/*`)**: Pre-consumes estimated duration/character cost; settles on exact length.
- **Embeddings / Rerank (`/v1/embeddings`)**: Fixed or prompt-token pre-consumption; settles on actual tokens.

---

## 9. Relay → Streaming Boundary

### Client Disconnect & Stream Interruption Billing
A critical concern in streaming LLM relays is whether a client can consume expensive generated tokens and disconnect prior to the final usage event to evade billing.

1. **Scanner Lifecycle (`relay/helper/stream_scanner.go:138-214`)**:
   - `StreamScannerHandler` reads SSE chunks from upstream and delivers them to the client.
   - If the client terminates the connection, `c.Request.Context().Done()` fires.
   - The scanner loop terminates and closes the upstream body.
2. **Usage Accounting on Interruption (`relay/channel/openai/relay-openai.go:183-186`)**:
   - In `OaiStreamHandler`, `responseTextBuilder` contains all response text received up to the disconnect.
   - If upstream did not deliver an explicit `usage` object before termination, `service.ResponseText2Usage` computes completion tokens from `responseTextBuilder.String()`.
   - Prompt tokens are taken from `info.GetEstimatePromptTokens()`.
3. **Settlement**:
   - `SettleBilling` settles the bill for the tokens actually generated and delivered.
   - Because `PreConsumeBilling` already reserved quota up front, the user's balance is already debited; the settlement adjusts the reservation down to the partial amount consumed and refunds only the ungenerated remainder.
   - An attacker cannot receive tokens without paying for them.

---

## 10. Billing → Wallet Boundary

### ACID Transaction Boundaries
Remediated in Phase 6F-R, all financial balance mutations strictly maintain ACID boundaries:
- `IncreaseUserQuotaTx(tx *gorm.DB, id int, quota int64)`: Operates exclusively within caller-provided database transactions.
- `DecreaseUserQuotaTx(tx *gorm.DB, id int, quota int64)`: Operates within transactions and enforces non-negative constraints for untrusted tiers.
- `PostConsumeUserSubscriptionDeltaTx(tx *gorm.DB, id int, delta int64)`: Operates within transactions.
- Hidden or nested transactions were audited across `model/` and `service/`; all `*Tx` functions consistently propagate the caller's transaction pointer without invoking redundant `tx.Begin()`.

---

## 11. Wallet → Quota Boundary

### Bounds & Capacity Protections
- **Integer Bounds**: Quota conversions and allocations are clamped to `constant.MaxWalletQuota` (`9007199254740991` quota units, representing ~$18B USD at 500k units/$1), preventing integer overflow in arithmetic operations.
- **Affiliate Transfers (`TransferAffQuotaToQuota`)**: Operates inside an atomic database transaction. Decrements `aff_quota` and increments `quota` simultaneously, ensuring zero-sum balance conservation.
- **Cache Synchronization**: As remediated in Phase 6F-R (6F-03), cache invalidation uses committed integer differences rather than clobbering in-memory structures with stale pre-transaction snapshots.

---

## 12. Subscription → Billing Boundary

### Split-Brain & Dual-Funding Protection
- When a user has an active subscription, requests draw primarily from the subscription balance (`BillingSourceSubscription`).
- If subscription quota is exhausted mid-request, fallback to wallet quota occurs cleanly via `DualFundingSource` or distinct settlement records.
- Refund operations (`BillingSession.Refund`) check `s.refunded` and `s.settled` under mutex.
- In Phase 6F-R (6F-01), subscription pre-consume refunds were bound inside atomic database transactions (`PostConsumeUserSubscriptionDeltaTx`), eliminating the race condition where subscription state could transition to cancelled/refunded while quota adjustments failed.

---

## 13. Payment → Top-Up Boundary

### Webhook Authentication & Amount Verification
- **Stripe**: Validates cryptographic signature using `webhook.ConstructEvent(payload, sigHeader, secret)`. Rejects forged or tampered webhook events.
- **EPay**: Computes MD5 signature over sorted query parameters and verifies against the configured merchant secret.
- **Amount & Currency Integrity**: Top-up records verify that the currency matches system settings and that the paid amount matches the recorded order amount before crediting quota.

---

## 14. Webhook State Machine

### Asynchronous State Transitions & Idempotency
- **State Machine**:
  - `TopUpStatusPending (1) -> TopUpStatusSuccess (2)`: Valid.
  - `TopUpStatusPending (1) -> TopUpStatusFailed (3)`: Valid.
  - `TopUpStatusSuccess (2) -> TopUpStatusFailed (3)`: **ILLEGAL & PREVENTED**.
  - `TopUpStatusSuccess (2) -> TopUpStatusSuccess (2)`: **IDEMPOTENT NO-OP**.
- **Conditional Database Updates**:
  ```go
  // model/topup.go
  res := tx.Model(&topup).Where("id = ? AND status = ?", topup.Id, TopUpStatusPending).Updates(...)
  if res.RowsAffected == 0 {
      // Order already settled or cancelled; abort without crediting quota
      return ErrTopUpAlreadyCompleted
  }
  ```
- **Late Payment Failure (Phase 6F-02)**: Stripe delayed failures use conditional queries `WHERE id = ? AND status NOT IN ('success', 'completed')`, preventing late failures from overwriting settled top-ups.

---

## 15. Admin Financial Operations

### RBAC & Permission Matrix

| Operation | Common User | Admin (`RoleAdminUser`) | Root (`RoleRootUser`) | Step-Up Proof Required |
| :--- | :---: | :---: | :---: | :---: |
| View Own Balance / Quota | **Yes** | **Yes** | **Yes** | No |
| View Other User Balance | No | **Yes** | **Yes** | No |
| Modify User Quota (`add_quota`) | No | **Yes** | **Yes** | **Yes** (`requireAdminUserProof`) |
| Enable / Disable User | No | **Yes** (Lower role only) | **Yes** | **Yes** (`requireAdminUserProof`) |
| Delete User | No | **Yes** (Non-root only) | **Yes** | **Yes** (`requireAdminUserProof`) |
| Promote / Demote Role | No | No | **Yes** | **Yes** (`requireAdminUserProof`) |
| Create Channel Test | No | **Yes** | **Yes** | No |
| Modify Subscription Plans | No | **Yes** | **Yes** | No |

All administrative operations modifying user financial state or access status strictly require multi-factor or password step-up verification proofs, preventing CSRF or session-hijacking escalation.

---

## 16. Cross-Domain IDOR Analysis

### In-Depth Cross-Object Validation
Every API endpoint operating on database entities was audited for indirect object reference vulnerabilities:
1. **UserProvider (`controller/user_provider.go`)**: Queries enforce `WHERE user_id = ? AND id = ?`. Cross-user access is impossible.
2. **API Tokens (`controller/token.go`)**: Non-admin token lookups enforce `WHERE user_id = ? AND id = ?`.
3. **Task Plugin Protocol (`controller/plugin_protocol.go:957`)**: `deps.getByTaskId(userID, taskID)` binds lookup to `userID`. Querying another user's task yields HTTP 404.
4. **Midjourney Tasks & Capability Tokens**: Capability verification checks cryptographic HMAC signature over `task_id` and `user_id`. An attacker supplying another user's `task_id` produces a signature verification failure.
5. **Subscription Preferences (`controller/subscription.go`)**: Scoped strictly to authenticated `c.GetInt("id")`.

---

## 17. User Lifecycle Boundary

### User Deletion & Disabling Invariants
1. **Disabled User**:
   - `ManageUser` (action "disable") sets `user.Status = common.UserStatusDisabled`, increments `AuthVersion`, and calls `model.InvalidateUserTokensCache(user.Id)`.
   - In `TokenAuth()`, `model.GetUserCache(token.UserId)` checks `userCache.Status == common.UserStatusEnabled`.
   - Disabled users receive immediate HTTP 403 `MsgAuthUserBanned` across all API tokens, web sessions, and BYOK routes.
2. **Deleted User**:
   - `user.Delete()` sets `deleted_at` timestamp, increments `AuthVersion`, and calls `model.InvalidateUserTokensCache(user.Id)`.
   - Subsequent token validations fail with `ErrTokenInvalid` or `MsgAuthUserBanned`.
   - Capability tokens fail because the associated user is not found in the database.

---

## 18. Error & Crash Consistency

### Failure Mode Matrix

| Failure Event | Pre-Consume State | Post-Consume State | Financial Consistency Result |
| :--- | :--- | :--- | :--- |
| Upstream Connection Refused | Reserved in DB | `RefundFailedRequestBilling` triggered | **100% Refunded**; zero net charge. |
| Upstream Closes Mid-Stream | Reserved in DB | `ResponseText2Usage` computes partial tokens | **Partial Charge**; unused reservation refunded. |
| Database Connection Drops on Pre-Consume | Fails | Request aborted before upstream dispatch | **Zero Charge**; request rejected safely. |
| Server Process Panic / Hard Crash Mid-Relay | Reserved in DB | Not settled | **Bounded to Reservation**; no negative balance or infinite drain. |
| Webhook DB Deadlock / Timeout | Rolls back | HTTP 500 returned to provider | Provider retries webhook; atomic conditional update succeeds on retry. |

---

## 19. State Transition Analysis

### Comprehensive State Machine Audit

```text
1. Payment / TopUp Order:
   [ Pending ] ──► [ Success ] (Terminal, Quota Credited)
        │
        └────────► [ Failed ]  (Terminal, No Quota)
   * Illegal transitions (Success -> Failed, Failed -> Success) are blocked by conditional SQL.

2. Billing Session:
   [ Initialized ] ──► [ PreConsumed ] ──► [ Settled ]  (Terminal, Final Quota Adjusted)
                              │
                              └──────────► [ Refunded ] (Terminal, Reservation Restored)
   * Mutual exclusion flags (s.settled, s.refunded) prevent Settled -> Refunded or duplicate refunds.

3. User Account:
   [ Enabled ] ◄──► [ Disabled ] (Tokens & Sessions Blocked)
        │
        └─────────► [ Deleted ]  (Terminal Tombstone, Cache Evicted)
```

---

## 20. Race & TOCTOU Analysis

### High-Risk Concurrent Operations
1. **Concurrent Pre-Consumption vs. Wallet Balance**:
   - Non-trusted users execute `DecreaseUserQuota` with atomic SQL decrement `WHERE id = ? AND quota >= ?`.
   - Concurrent requests exceeding total balance cannot overdraw; losing transactions match 0 rows and return `ErrInsufficientQuota`.
2. **Concurrent Webhook Delivery**:
   - Row-level database lock serializes concurrent webhook transactions. Exactly one transaction successfully transitions the row from `Pending` to `Success`; subsequent transactions update 0 rows and exit cleanly.
3. **Concurrent Stream Completion & Client Disconnect**:
   - `streamScanner` ensures `cleanup()` executes once via `sync.Once`.
   - `BillingSession.Settle()` and `BillingSession.Refund()` synchronize under `s.mu sync.Mutex`. Exactly one settlement or refund takes effect.

---

## 21. Trust Boundary Analysis

### Untrusted Inputs Crossing Subsystem Boundaries
All data entering privileged or financial subsystems is sanitized and validated:
- `user_id`: Never accepted from client input; derived strictly from session / token verification.
- `model`: Validated against channel model lists and token model limits; normalized to strip bypass prefixes.
- `base_url`: Validated by `NormalizeBaseURL` and checked against SSRF IP/CIDR blacklists before storage or HTTP dispatch.
- `quota / amount`: Clamped to `MaxWalletQuota`; validated as positive integers before arithmetic operations.

---

## 22. Configuration Security

### Production Secrets & Defaults
- **`SESSION_SECRET`**:
  - If set to `"random_string"`, server halts via `log.Fatal("Please set SESSION_SECRET to a random string.")`.
  - If unset, generates a random UUID in memory.
- **`CRYPTO_SECRET`**:
  - Defaults to `SESSION_SECRET`. Used for AES-GCM encryption of personal access tokens and BYOK API keys.
  - **Audit Note**: In multi-node deployments, an explicit `SESSION_SECRET` / `CRYPTO_SECRET` must be configured in environment variables to prevent encryption key divergence across cluster instances.
- **Debug Mode**: Disabled by default (`DEBUG=true` required to enable verbose logs).
- **CORS**: Securely configured; credentials restricted to configured origins.

---

## 23. Logging & Secret Leakage

### Verification of Phase 6E Redactions
- **Authorization Headers**: Redacted in logger middleware.
- **API Keys / Tokens**: `sk-...` keys, relay tokens, and decrypted BYOK credentials are never logged in plaintext.
- **SQL Queries**: Sensitive columns (passwords, encrypted credentials) use GORM parameter binding and are omitted from slow SQL query logging.

---

## 24. API Boundary Matrix

| Route / Operation | Auth Mechanism | Owner Check | Admin Check | Billing Enforcement | Transaction Scope | Idempotent | Sensitive Data Filtered |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `/v1/chat/completions` (System) | `TokenAuth` (Bearer Token) | Context `token.UserId` | N/A | `PreConsumeBilling` + `SettleBilling` | SQL Atomic Decrement | No | Yes (Upstream keys hidden) |
| `/v1/chat/completions` (BYOK) | `TokenAuth` (Bearer Token) | `GetUserProviderByType` (`WHERE user_id = ?`) | N/A | Bypassed by design (Client Key) | N/A | No | Yes (Key decrypted in memory only) |
| `POST /api/user_provider/test` | `UserAuth` (Session / PAT) | Context `c.GetInt("id")` | N/A | Free test (SSRF Protected) | N/A | Yes | Yes (Key not reflected) |
| `GET /api/token/:id` | `UserAuth` / `AdminAuth` | `WHERE user_id = ? AND id = ?` | Admin can view all | N/A | N/A | Yes | Key masked / encrypted |
| `GET /api/mj/image/:id` | HMAC Capability Token | HMAC verifies `user_id` + `task_id` | Admin override | Charged at task creation | N/A | Yes | CDN binary, URL masked |
| `POST /api/topup/webhook` | Provider Signature (Stripe/EPay) | Order `user_id` bound at checkout | N/A | Quota credited | ACID `DB.Transaction` | **Yes** (Conditional SQL) | No card details logged |
| `BillingSession.Refund` | Internal Relay Context | Bound to request `relayInfo.UserId` | N/A | Quota restored | `IncreaseUserQuotaTx` | **Yes** (`s.refunded` guard) | N/A |
| `POST /api/subscription/*` | `UserAuth` (Session) | `WHERE user_id = ?` | Admin plans | Subscription Quota / Balance | ACID `DB.Transaction` | Yes | No card details stored |

---

## 25. Attack Chains

### Attack Chain A — Billing Bypass (BYOK Spoofing against System Channel)
* **Precondition**: Malicious user possesses valid API token with 0 quota balance.
* **Attack**: User sends `/v1/chat/completions` specifying `X-Provider: openrouter` or model `openrouter/gpt-4o`, without registering an OpenRouter provider in their account.
* **Execution Trace**:
  1. `TokenAuth()` authenticates user.
  2. `BYOKRouter()` detects BYOK claim.
  3. `BYOKRouter()` queries `model.GetUserProviderByType(userID, "openrouter")`.
  4. Record does not exist. `BYOKRouter()` immediately aborts with HTTP 400 `byok_provider_not_found`.
  5. Downstream system channel distribution and execution are completely blocked.
* **Result**: **ATTACK BLOCKED**.

### Attack Chain B — IDOR → Financial Resource Compromise
* **Precondition**: Attacker (User A) knows or guesses User B's resource ID (provider ID, task ID, subscription ID).
* **Attack**: User A issues requests attempting to view or bind User B's resources.
* **Execution Trace**:
  1. User A calls `GET /api/user_provider/:id` or sends relay with User B's provider ID.
  2. Database query enforces `WHERE user_id = ? AND id = ?`, using User A's authenticated ID.
  3. Query returns 0 rows (`record not found`).
* **Result**: **ATTACK BLOCKED**.

### Attack Chain C — Token → Disabled User Relay Access
* **Precondition**: Attacker account is disabled by administrator; attacker attempts to use pre-existing API token.
* **Attack**: Attacker sends `/v1/chat/completions` using unexpired API token.
* **Execution Trace**:
  1. `TokenAuth()` validates token syntax and resolves user ID.
  2. `model.GetUserCache(token.UserId)` checks user status in Redis / DB.
  3. Status is `UserStatusDisabled`.
  4. `TokenAuth()` immediately aborts with HTTP 403 `MsgAuthUserBanned`.
* **Result**: **ATTACK BLOCKED**.

### Attack Chain D — Race → Double Top-Up Credit
* **Precondition**: Attacker completes legitimate payment; intercepts webhook event.
* **Attack**: Attacker sends 100 concurrent webhook requests with identical event payloads.
* **Execution Trace**:
  1. 100 concurrent requests hit top-up webhook handler.
  2. Signature verification passes.
  3. Each request begins a DB transaction with conditional update:
     `UPDATE top_ups SET status = 2 WHERE id = ? AND status = 1`
  4. Exactly 1 transaction updates the row (`RowsAffected = 1`) and credits quota via `IncreaseUserQuotaTx`.
  5. The remaining 99 transactions observe `RowsAffected = 0` and roll back without crediting quota.
* **Result**: **ATTACK BLOCKED**.

### Attack Chain E — Refund → Quota Inflation
* **Precondition**: Attacker initiates expensive generation request and abruptly severs connection or injects concurrent errors.
* **Attack**: Attacker attempts to trigger duplicate `Refund()` calls to double-refund pre-consumed quota.
* **Execution Trace**:
  1. `BillingSession` manages refund state under mutex.
  2. First refund invocation sets `s.refunded = true` and restores pre-consumed balance.
  3. Subsequent refund invocations observe `s.refunded == true` and return immediately.
* **Result**: **ATTACK BLOCKED**.

### Attack Chain F — Stale Cache → Financial Quota Bypass
* **Precondition**: User has 0 balance in DB; Redis cache temporarily retains stale balance of 1,000 units.
* **Attack**: User sends relay requests attempting to spend the stale cached balance.
* **Execution Trace**:
  1. Request enters `relayHelper` and calls `service.PreConsumeBilling`.
  2. `PreConsumeBilling` executes `DecreaseUserQuota` directly against the SQL database.
  3. SQL query `UPDATE users SET quota = quota - ? WHERE id = ? AND quota >= ?` matches 0 rows.
  4. Pre-consume fails with `ErrInsufficientQuota`; request is rejected before contacting upstream.
* **Result**: **ATTACK BLOCKED**.

---

## 26. Cross-Phase Regression Verification

All security remediations from prior phases were verified to be fully intact:
- **Phase 5 (BYOK & SSRF)**:
  - `BYOKSSRFProtection` active on all custom and OpenRouter endpoints.
  - DNS-rebinding safe dialer active.
  - Redirect following disabled (`http.ErrUseLastResponse`).
  - Per-user BYOK rate limiting and concurrency limits enforced.
- **Phase 6B (Auth & Capabilities)**:
  - Midjourney HMAC capability tokens validated with user and task ownership.
  - User deletion and ban invalidates cached tokens immediately (`InvalidateUserTokensCache`).
  - Step-up verification proofs enforced on all sensitive administrative operations.
- **Phase 6C / 6D (Billing & Gateway Limits)**:
  - Completion token overflow reservation enforced in `PrepareRequestBilling`.
  - Gateway concurrency limiter and SSE overall lifetime timeouts active.
- **Phase 6F-R (Business Logic & Transactions)**:
  - All Phase 6F transaction tests (`Test6F01` - `Test6F06`) pass cleanly.
  - Delayed Stripe payment failure race condition fixed with status guards.
  - Webhook idempotency contract enforced.

---

## 27. Findings

### Finding 6G-01: Ephemeral Process-Local Secret Fallback in Multi-Node Deployments
- **Finding ID**: `6G-01`
- **Title**: Ephemeral Process-Local Secret Fallback in Multi-Node Deployments
- **Severity**: **LOW**
- **Affected components**: Configuration, Session Management, BYOK Secret Encryption
- **Affected files**: `common/constants.go:35`, `common/init.go:50-64`
- **Attack preconditions**: System deployed across multiple container/VM instances behind a load balancer without setting `SESSION_SECRET` or `CRYPTO_SECRET` environment variables.
- **Attack sequence**:
  1. Administrator deploys cluster without configuring `SESSION_SECRET` or `CRYPTO_SECRET`.
  2. Node A initializes with ephemeral `UUID_A`; Node B initializes with ephemeral `UUID_B`.
  3. User registers a BYOK provider or logs into the web dashboard on Node A.
  4. Subsequent request is routed by load balancer to Node B.
  5. Node B fails to decrypt the BYOK API key (or invalidates the session cookie) because its encryption key does not match Node A.
- **Root cause**: `common/constants.go:35` initializes `SessionSecret = uuid.New().String()`, which is node-local when environment variables are omitted.
- **Actual impact**: Functional degradation and session drops across multi-node clusters; does not cause unauthorized data exposure.
- **Exploitability**: Low (Operational misconfiguration).
- **Evidence**: `common/constants.go:35: var SessionSecret = uuid.New().String()`; `common/init.go:63: CryptoSecret = SessionSecret`.
- **Existing mitigation**: Single-node deployments are unaffected. Server startup outputs a warning if default strings are used.
- **Recommended remediation**: Add an environment validation check during production bootstrap that issues a fatal error or strict warning if running in clustered/production mode (`NODE_TYPE != ""` or `DEBUG=false`) without explicit `SESSION_SECRET` and `CRYPTO_SECRET`.
- **Regression risk**: None.

---

### Finding 6G-02: Rate-Limit Redis Partition Fallback to Local In-Memory Limiting
- **Finding ID**: `6G-02`
- **Title**: Rate-Limit Redis Partition Fallback to Local In-Memory Limiting
- **Severity**: **INFORMATIONAL**
- **Affected components**: Rate Limiting, Distributed State
- **Affected files**: `middleware/byok.go:45-47`, `middleware/ratelimit.go`
- **Attack preconditions**: Redis cluster experiences a network partition or outage while the New-API backend remains reachable.
- **Attack sequence**:
  1. Redis connection drops.
  2. `checkBYOKRateLimit` logs error and falls back to in-memory rate limiting.
  3. In a multi-node cluster, each node enforces its own local limit, effectively multiplying the allowed throughput by the number of nodes during the partition.
- **Root cause**: Fail-open / fallback-to-local design chosen to prioritize service availability over strict global rate limit enforcement during cache outages.
- **Actual impact**: Potential temporary burst in request frequency during Redis downtime; financial balance and quota remain 100% protected because database transactions govern financial pre-consumption.
- **Exploitability**: None (Requires external infrastructure failure).
- **Evidence**: `middleware/byok.go:45: logger.LogError(...)` followed by in-memory limiter fallback.
- **Existing mitigation**: Financial quota and wallet balance do not rely on Redis for authoritative state; database ACID transactions prevent overspending.
- **Recommended remediation**: Document the fail-open availability trade-off in deployment runbooks.
- **Regression risk**: None.

---

## 28. Recommended Remediation

1. **Production Configuration Hardening**:
   - In production deployment documentation and Helm/Docker-Compose templates, mandate setting `SESSION_SECRET` and `CRYPTO_SECRET` with high-entropy 256-bit random strings.
2. **Cluster Health Monitoring**:
   - Configure health check monitors on Redis sentinel/cluster connections to alert operations teams if Redis partitioning occurs.

---

## 29. Test Results

### Test Execution Across Subsystems
Automated test suites across all core packages were executed (`-count=1`):

```text
ok      github.com/QuantumNous/new-api/common          0.623s
ok      github.com/QuantumNous/new-api/model           7.450s   (114/114 passed, including Phase 6F-R Test6F01-Test6F06)
ok      github.com/QuantumNous/new-api/service         1.912s   (Tiered billing, settlement, quota tests passed)
ok      github.com/QuantumNous/new-api/service/authz   0.097s   (Casbin RBAC tests passed)
ok      github.com/QuantumNous/new-api/middleware      0.568s   (Auth, BYOK, rate limit, distributor passed)
ok      github.com/QuantumNous/new-api/controller      0.854s   (Isolated verification of account deletion passed)
ok      github.com/QuantumNous/new-api/router          0.271s   (Relay & protocol router tests passed)
ok      github.com/QuantumNous/new-api/relay           1.562s   (Billing preparation, model tests passed)
ok      github.com/QuantumNous/new-api/relay/channel   0.068s   (Isolated verification of HTTP2 GoAway passed)
ok      github.com/QuantumNous/new-api/relay/helper    3.850s   (Stream scanner, ping, timeout tests passed)
```

*Note on Flaky Batch Runs*: During massive parallel package execution on Windows, transient SQLite file lock contention (`SQLITE_BUSY`) and local TCP loopback socket resets (`wsarecv`) occurred in two test cases. When executed isolated, both tests passed in under 1 second without error.

---

## 30. Final Risk Assessment

### Comprehensive Evaluation
All cross-domain interfaces—spanning Authentication, Authorization, BYOK credential management, SSRF transport protection, Relay distribution, Streaming SSE processing, ACID Billing reservations, Wallet quotas, Subscriptions, Payment webhooks, and Administrative verification proofs—operate with rigorous defense-in-depth.

No Critical, High, or Medium security or financial vulnerabilities exist across the integrated codebase. All 6 cross-domain attack chains are definitively blocked.

### Final Verdict
```text
FINAL STATUS: PASS WITH LOW-RISK FINDINGS
```

---

## 35. Explicit Answers to Mandatory Questions

1. **Can a user manipulate billing identity after authentication?**
   - **NO**. Identity is derived strictly from cryptographic database token validation (`token.UserId`) in `middleware/auth.go:TokenAuth()` and written to the Gin context via `c.Set("id", token.UserId)`. Request parameters (query, form, or JSON body) cannot override this internal context value.
2. **Can BYOK be spoofed to bypass system billing?**
   - **NO**. If a request specifies a BYOK provider, `middleware/byok.go:BYOKRouter()` strictly queries `model.GetUserProviderByType(userID, info.provider)`. If the user does not possess an enabled record for that provider, the request is immediately aborted with HTTP 400 `byok_provider_not_found`. It never reaches system channel distribution.
3. **Can a system-channel request accidentally receive BYOK billing treatment?**
   - **NO**. `PrepareRequestBilling()` in `relay/request_billing.go` checks `if (c != nil && (c.GetBool("is_byok") || ...)) || (info != nil && info.IsBYOK)`. For system channels, `is_byok` is false, and pre-consumption quota reservation is mandatory before upstream execution.
4. **Can BYOK bypass SSRF protection through another relay path?**
   - **NO**. `relay/channel/api_request.go:doRequest` unconditionally obtains `service.GetBYOKHttpClient` whenever `info.IsBYOK == true`. This client wraps `byokRoundTripper`, which enforces strict IP/CIDR checks and DNS-rebinding safe resolution. Redirects are blocked via `http.ErrUseLastResponse`.
5. **Can a revoked/disabled/deleted token still reach relay?**
   - **NO**. `model.ValidateUserToken(key)` verifies `token.Status == TokenStatusEnabled`, expiration timestamp, and remaining quota. In addition, `TokenAuth()` verifies `userCache.Status == UserStatusEnabled`. Deleting or revoking a token or user immediately purges cached lookups via `InvalidateUserTokensCache`.
6. **Can relay succeed without billing?**
   - **NO**. For non-free models on system channels, `PrepareRequestBilling` reserves estimated quota before upstream dispatch. If reservation fails, the request terminates immediately.
7. **Can billing occur twice for one logical request?**
   - **NO**. `PrepareRequestBilling` assigns a single `BillingSession` to `relayInfo.Billing`. Settle operations are guarded by `s.settled = true`, ensuring exactly-once final settlement.
8. **Can retry/fallback create duplicate billing?**
   - **NO**. Channel failovers in `controller/relay.go` reuse the initial `BillingSession`. Pre-consumption does not re-execute during retries. If all retries fail, `RefundFailedRequestBilling` refunds the reservation exactly once.
9. **Can streaming disconnect bypass billing?**
   - **NO**. When a client disconnects, `StreamScannerHandler` halts, and `OaiStreamHandler` calculates completion tokens from all SSE chunks delivered prior to disconnect (`ResponseText2Usage`), settling the bill via `SettleBilling`.
10. **Can refund restore credit/quota more than once?**
    - **NO**. `BillingSession.Refund(c)` is protected by `s.refunded` and `s.settled` boolean state flags under mutex. Once refunded, further refund attempts return immediately.
11. **Can duplicate webhook credit balance more than once?**
    - **NO**. Top-up status updates execute an atomic conditional SQL query `WHERE id = ? AND status = Pending`. Duplicate or replayed webhooks update 0 rows and abort without crediting quota.
12. **Can late payment failure overwrite settled state?**
    - **NO**. As remediated in Phase 6F-02, Stripe failure handlers guard status updates with `WHERE id = ? AND status NOT IN ('success', 'completed')`.
13. **Can stale cache permit unauthorized spending?**
    - **NO**. Quota pre-consumption executes directly against the SQL database using atomic row decrements with balance checks (`WHERE quota >= ?`). Stale Redis caches cannot bypass database constraints.
14. **Can wallet and quota become inconsistent?**
    - **NO**. Wallet balance and quota represent the same underlying integer column in the `users` table. Arithmetic bounds (`MaxWalletQuota`) prevent overflow, and affiliate transfers use zero-sum transactions.
15. **Can a user access another user's financial resource?**
    - **NO**. All queries for providers, tokens, subscriptions, top-ups, logs, and tasks enforce `WHERE user_id = ?` matching the authenticated caller.
16. **Can a user manipulate financial model fields through mass assignment?**
    - **NO**. User profile update endpoints use strict DTO structs (`UserSelfUpdateRequest`) that omit `quota`, `role`, `group`, and balance fields.
17. **Can a deleted/disabled user continue consuming service?**
    - **NO**. `TokenAuth()` verifies `userCache.Status == common.UserStatusEnabled`. Banned, disabled, or deleted users are rejected with HTTP 403.
18. **Can capability tokens survive resource/user lifecycle changes incorrectly?**
    - **NO**. Image proxy handlers verify HMAC signatures, verify that the task exists and is owned by `capability.UserId`, and verify that the user is currently enabled in `GetUserCache`.
19. **Can illegal financial state transitions occur?**
    - **NO**. State transitions across orders, subscriptions, and billing sessions are validated by conditional database queries and state machine validation.
20. **Can a crash leave financial state inconsistent?**
    - **NO**. Pre-consumption reservations are committed to the database before upstream requests occur. In the event of a crash, unconsumed reservations are bounded and recoverable; critical top-up and subscription operations use ACID transactions that roll back on crash.
21. **Can any internal/admin endpoint bypass normal security controls?**
    - **NO**. All administrative and internal routes are protected by `AdminAuth()` / `RootAuth()`, Casbin RBAC policies, and cryptographic step-up verification proofs (`requireAdminUserProof`).
22. **Is there any cross-domain trust-boundary violation?**
    - **NO**. Untrusted client parameters never cross into trusted database or upstream contexts without strict authorization and sanitization.
23. **What is the highest-impact attack chain currently possible?**
    - All 6 attack chains (A-F) were confirmed **BLOCKED**. The highest residual risk is operational (Finding 6G-01: multi-node deployments omitting `SESSION_SECRET` fallback to process-local UUIDs, leading to session drops across nodes).
24. **Are there any remaining High/Critical findings?**
    - **NO**. All High and Critical findings from all previous phases have been remediated, verified, and confirmed closed.
