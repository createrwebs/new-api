# Phase 7B — LumenFlow ↔ Tora/New-API Mobile Integration Contract

**Document Version:** 1.0.0  
**Date:** October 5, 2026  
**Status:** Frozen Contract Specification  
**Mobile Base:** LumenFlow (Flutter 3.x / Dart 3.11, MIT License, Cupertino UI)  
**Backend Baseline:** SaaSCover / Tora New-API (`/Users/noppanan/new-api` @ `feat/formobile`, commit `8d68eaa8b5dda6c647d1e88278d1f667a8e0d0e5`)  

---

## 1. Executive Summary

Phase 7A audited the mobile codebases and concluded that **LumenFlow** (`https://github.com/HuanMeng-official/LumenFlow.git`) is the optimal production foundation. It provides an elegant Cupertino interface, a clean `HttpProviderBase` with exponential backoff, robust SSE streaming parsing with reasoning deltas, and local SQLite conversation persistence. Phase 7A also established that **no Backend-for-Frontend (BFF) is required**: LumenFlow can communicate directly with the existing SaaSCover / Tora New-API backend.

Phase 7B audits the exact backend source code on branch `feat/formobile` and defines the authoritative, frozen integration contract between LumenFlow and the backend.

### Key Conclusions of the Contract Audit
1. **Direct Backend Integration:** LumenFlow connects directly to New-API over TLS 1.3. No intermediary proxy or gateway is needed.
2. **Dual Authentication Mechanics:**
   - **Management Layer (`/api/*`):** Authenticated via Dashboard Session JWT (`access_token`, 15-minute TTL) returned by `POST /api/user/login`, refreshed via `POST /api/user/auth/refresh` with an HttpOnly cookie or secure session header.
   - **Inference Relay Layer (`/v1/*`):** Authenticated via a dedicated mobile API relay key (`sk-...`, stored in the `tokens` table) managed via `/api/token`.
3. **Strict BYOK Privacy & Security:** Plaintext provider API keys are **never** returned over the network. The backend AES-256-GCM encryption is completely write-only (`APIKeyEncrypted` has `json:"-"`). Mobile passes `X-Provider: openrouter` or `X-Provider: custom` to trigger server-side credential resolution, concurrency gating, and quota bypassing.
4. **Subscription-Driven Entitlements:** Model access is dynamically driven by the user's `group` (`default`, `pro`, `vip`), which is upgraded upon subscription activation and automatically downgraded upon expiration.
5. **No HTTP 402:** The backend does **not** return HTTP 402 for quota exhaustion. Insufficient balance returns `HTTP 403 Forbidden` (`code: "insufficient_user_quota"`), and daily free limits return `HTTP 429 Too Many Requests` (`code: "FREE_DAILY_LIMIT"`).
6. **Store Billing Separation:** Native Apple StoreKit 2 and Google Play Billing receipt validation do not yet exist in `new-api`. Phase 7C–7F can proceed immediately with web subscription status and billing portal deep links; Phase 7G will add server receipt verification.

---

## 2. Backend Route Inventory

Every route required by mobile has been verified directly in `router/api-router.go` and `router/relay-router.go`:

| Domain | Method | Exact Path | Middleware | Auth Required | Request Schema | Response Schema | Purpose |
|---|---|---|---|---|---|---|---|
| **Auth** | `POST` | `/api/user/login` | `CriticalRateLimit`, `DisableCache`, `TurnstileCheck` | None (Public) | `{"username": "...", "password": "..."}` | `{"success": true, "data": {"access_token": "...", "session": {...}, "user": {...}}}` | User login & session creation |
| **Auth** | `POST` | `/api/user/register` | `CriticalRateLimit`, `TurnstileCheck` | None (Public) | `{"username": "...", "password": "...", "email": "..."}` | `{"success": true, "message": "..."}` | New user registration |
| **Auth** | `POST` | `/api/user/auth/refresh` | `SessionCookieOriginGuard`, `CriticalRateLimit`, `DisableCache` | Session Cookie / Header | Empty body | `{"success": true, "data": {"access_token": "...", "session": {...}}}` | Refresh 15-minute access token |
| **Auth** | `POST` | `/api/user/auth/logout` | `SessionCookieOriginGuard`, `CriticalRateLimit`, `DisableCache` | `UserAuth()` | Empty body | `{"success": true, "data": {"revoked_sid": "..."}}` | Revoke session & clear cookie |
| **User** | `GET` | `/api/user/self` | `DisableCache`, `UserAuth()` | `access_token` | None | `{"success": true, "data": {"id": 1, "username": "...", "quota": 500000, "group": "pro"}}` | Fetch user profile & quota |
| **User** | `DELETE` | `/api/user/self` | `DisableCache`, `UserAuth()` | `access_token` | None | `{"success": true, "message": "..."}` | User self-deletion (GDPR/AppStore) |
| **Tokens** | `GET` | `/api/token/` | `UserAuth()`, `TokenOperationAudit()` | `access_token` | Query params: `p`, `page_size` | `{"success": true, "data": [{"id": 1, "name": "...", "key": "sk-****"}]}` | List user relay tokens |
| **Tokens** | `POST` | `/api/token/` | `UserAuth()`, `TokenOperationAudit()` | `access_token` | `{"name": "LumenFlow-Mobile", "unlimited_quota": true}` | `{"success": true, "message": ""}` | Create mobile relay token |
| **Tokens** | `POST` | `/api/token/:id/key` | `CriticalRateLimit`, `DisableCache`, `UserAuth()` | `access_token` | None | `{"success": true, "data": {"key": "sk-..."}}` | Retrieve full unmasked relay key |
| **BYOK** | `GET` | `/api/user/providers` | `DisableCache`, `UserAuth()` | `access_token` | None | `{"success": true, "data": [{"id": 1, "provider": "openrouter", "api_key": "sk-or-****"}]}` | List configured BYOK providers |
| **BYOK** | `POST` | `/api/user/providers` | `CriticalRateLimit`, `DisableCache`, `UserAuth()` | `access_token` | `{"provider": "openrouter", "name": "MyOR", "api_key": "sk-..."}` | `{"success": true, "data": {"id": 1, "api_key": "sk-****"}}` | Store encrypted BYOK provider |
| **BYOK** | `PUT` | `/api/user/providers/:id` | `CriticalRateLimit`, `DisableCache`, `UserAuth()` | `access_token` | `{"name": "...", "api_key": "sk-...", "enabled": true}` | `{"success": true, "data": {...}}` | Update BYOK provider |
| **BYOK** | `DELETE` | `/api/user/providers/:id` | `CriticalRateLimit`, `DisableCache`, `UserAuth()` | `access_token` | None | `{"success": true, "data": {"id": 1}}` | Remove BYOK provider |
| **BYOK** | `POST` | `/api/user/providers/:id/test` | `CriticalRateLimit`, `DisableCache`, `UserAuth()` | `access_token` | None | `{"success": true, "data": {"message": "..."}}` | Test provider connectivity |
| **Models** | `GET` | `/v1/models` | `RouteTag`, `TokenAuth()` | `sk-...` | None | `{"object": "list", "data": [{"id": "gpt-4o", "object": "model"}]}` | Available models for relay key |
| **Models** | `GET` | `/api/user/models` | `DisableCache`, `UserAuth()` | `access_token` | Optional query `?group=auto` | `{"success": true, "data": ["gpt-4o", "claude-3-5-sonnet"]}` | Entitled models for user group |
| **Chat** | `POST` | `/v1/chat/completions` | `TokenAuth()`, `BYOKRouter()`, `RelayConcurrencyLimit()`, `FreeTierQuota()`, `Distribute()` | `sk-...` | OpenAI Chat Completion Request Schema | SSE Stream or JSON Completion Response | Text & reasoning generation |
| **Sub** | `GET` | `/api/subscription/plans` | `UserAuth()` | `access_token` | None | `{"success": true, "data": [{"plan": {"id": 1, "name": "Pro"}}]}` | List available subscription plans |
| **Sub** | `GET` | `/api/subscription/self` | `UserAuth()` | `access_token` | None | `{"success": true, "data": {"billing_preference": "subscription_first", "subscriptions": [...]}}` | User active subscription state |

---

## 3. Authentication Contract

```text
               LumenFlow Mobile                       Tora / New-API
                     │                                      │
                     │  POST /api/user/login                │
                     │  {"username": "...", "password": ""} │
                     ├─────────────────────────────────────►│
                     │                                      │ Authenticate credentials
                     │                                      │ Issue Session & JWT
                     │  HTTP 200 OK                         │ Set-Cookie: new_api_refresh
                     │  { access_token: "...", user: {...}} │
                     │◄─────────────────────────────────────┤
                     │                                      │
                     │  Store in flutter_secure_storage     │
                     │  (access_token, refresh_cookie)      │
                     │                                      │
                     │  [If device sk- token missing]       │
                     │  GET /api/token/ -> POST /api/token/ │
                     ├─────────────────────────────────────►│
                     │◄─────────────────────────────────────┤
                     │  Store sk-... in secure storage      │
                     │                                      │
                     │  Subsequent Management API:          │
                     │  GET /api/user/self                  │
                     │  Authorization: Bearer <access_token>│
                     ├─────────────────────────────────────►│
                     │                                      │
                     │  Subsequent Chat Completion:         │
                     │  POST /v1/chat/completions           │
                     │  Authorization: Bearer <sk_key>      │
                     ├─────────────────────────────────────►│
```

### Detailed Token Lifecycle
1. **Access Token (`access_token`):**
   - **Type:** JSON Web Token (JWT), HS256 signed with HMAC derived from `common.SessionSecret`.
   - **Lifetime:** Exactly 15 minutes (`service.AccessTokenTTL = 15 * time.Minute`).
   - **Claims:** Issuer: `new-api`, Audience: `new-api-dashboard`, Subject: `UserID`, `sid`: Session ID, `uv`: User Auth Version, `sv`: Session Version.
   - **Transmission:** `Authorization: Bearer <access_token>`.
2. **Refresh Mechanism:**
   - **Trigger:** When any `/api/*` call returns `401 AUTH_TOKEN_EXPIRED` or when the token is within 60 seconds of `access_expires_at`.
   - **Endpoint:** `POST /api/user/auth/refresh`.
   - **Headers:** `X-Auth-Session: <session_id>` and `Origin: <backend_origin>` (to satisfy `middleware.SessionCookieOriginGuard()` in production HTTPS).
   - **Cookie:** `new_api_refresh=<refresh_token>` (handled natively by the HTTP client cookie jar).
   - **Rotation:** The server issues a new refresh token and new access token on each refresh.
3. **Revocation & Invalidation:**
   - **Logout:** `POST /api/user/auth/logout` revokes the specific session in Redis/PostgreSQL and clears the cookie.
   - **Password Change:** Bumps `user.auth_version`, instantly invalidating all active JWTs and refresh tokens across all devices.
   - **User Ban/Disable:** Checked live via user cache on every single request; disabled users receive `401 AUTH_USER_Banned` immediately.
4. **Relay API Key (`sk-...`):**
   - Used for `/v1/chat/completions` and `/v1/models`.
   - Created with `UnlimitedQuota: true` (or tied directly to account wallet quota).
   - Does not expire in 15 minutes, allowing uninterrupted streaming chat sessions even during prolonged backgrounding.

---

## 4. Mobile Secure Storage Contract

Data stored on the mobile device must strictly adhere to the following tiering:

| Data Element | Storage Location | Encryption Mechanism | Lifecycle / Purge Rule |
|---|---|---|---|
| **Dashboard Access Token** (`access_token`) | `flutter_secure_storage` | iOS Keychain / Android Keystore AES-256 | Purged on logout or 401 session revocation |
| **Refresh Token / Cookie** | `flutter_secure_storage` | iOS Keychain / Android Keystore AES-256 | Purged on logout |
| **Relay Key** (`sk-...`) | `flutter_secure_storage` | iOS Keychain / Android Keystore AES-256 | Purged on logout |
| **User ID & Username** | `SharedPreferences` | Plaintext OS sandbox | Overwritten on login, cleared on logout |
| **Active Model Selection** | `SharedPreferences` | Plaintext OS sandbox | Persists across app launches |
| **UI Theme, Language, Temperature** | `SharedPreferences` | Plaintext OS sandbox | User preferences |
| **Conversations & Messages** | SQLite (`sqflite`) | Sandbox DB file | Retained across logout; wiped on account deletion |
| **Attachments (cached images/docs)** | App Documents Directory | Sandbox file system | Cleared via conversation deletion |
| **Plaintext BYOK Keys** | **NEVER PERSIST** | **FORBIDDEN** | Ephemeral only during input form submission |
| **Subscription Authority** | **NEVER PERSIST** | **FORBIDDEN** | Always queried live from `/api/subscription/self` |
| **Quota / Balance Authority** | **NEVER PERSIST** | **FORBIDDEN** | Always queried live from `/api/user/self` |

---

## 5. BYOK Contract

### Routes and Signatures
- **List Providers:** `GET /api/user/providers`
- **Create Provider:** `POST /api/user/providers`
  - Body:
    ```json
    {
      "provider": "openrouter",
      "name": "My OpenRouter",
      "base_url": "",
      "api_key": "sk-or-v1-abcdef123456...",
      "enabled": true
    }
    ```
  - `provider` must be `"openrouter"` or `"custom"`.
  - For `"openrouter"`, `base_url` defaults to `https://openrouter.ai/api/v1`.
  - For `"custom"`, `base_url` is required, must start with `http://` or `https://`, and must not target internal private networks (RFC 1918 / loopback).
- **Update Provider:** `PUT /api/user/providers/:id`
  - Body: `{"name": "...", "base_url": "...", "api_key": "...", "enabled": true}`. If `api_key` is omitted or empty, the existing key is retained.
- **Delete Provider:** `DELETE /api/user/providers/:id`
- **Test Provider:** `POST /api/user/providers/:id/test`

### Masked Keys & Write-Only Guarantee
When mobile lists or creates providers, the server returns:
```json
{
  "id": 12,
  "user_id": 45,
  "provider": "openrouter",
  "name": "My OpenRouter",
  "base_url": "https://openrouter.ai/api/v1",
  "api_key": "sk-or-v1-****56",
  "enabled": true,
  "created_at": 1728100000,
  "updated_at": 1728100000
}
```
- In `model/user_provider.go`, `APIKeyEncrypted` is tagged with `json:"-"`.
- The plaintext API key is encrypted immediately upon submission using **AES-256-GCM** with Additional Authenticated Data (`user_provider:<user_id>:<provider>`).
- The backend **never** sends the plaintext key back to the client.

### Connection Testing & SSRF Boundaries
- The test endpoint executes `DefaultProviderTester.Test()`.
- Outbound requests are subject to strict SSRF filters (`service.NewBYOKHTTPClient`), enforcing:
  - Private IP blocking (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8, 169.254.0.0/16).
  - DNS rebinding validation before dial.
  - Strict 7-second timeout.
  - Disabled redirect following.

---

## 6. Managed Mode Contract

When using Tora's managed service (system-provided models):
1. **Request Execution:**
   - Mobile sends `POST /v1/chat/completions` with `Authorization: Bearer <sk_key>`.
   - Omit `X-Provider`, `X-BYOK-Provider`, and `X-BYOK` headers.
   - Do not prefix the model name with `openrouter/` or `custom/`.
2. **Channel Selection & Distribution:**
   - `middleware.BYOKRouter()` determines `isBYOK == false`.
   - `middleware.FreeTierQuota()` checks if the user is on the free tier; if exceeded, rejects with `429 FREE_DAILY_LIMIT`.
   - `middleware.Distribute()` resolves the model to an active system channel based on user group weighting and health priority.
3. **Billing & Quota Deduction:**
   - Handled via `service.BillingSession`.
   - Quota is pre-reserved based on model token rates.
   - On completion (or SSE finish), actual tokens are calculated from upstream `usage` or counted via tokenizer, and the wallet or subscription quota is finalized atomically.

---

## 7. Model Contract

### Model Listing
Mobile calls `GET /v1/models` with `Authorization: Bearer <sk_key>`.

**Response Format:**
```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1626777600,
      "owned_by": "openai"
    },
    {
      "id": "claude-3-5-sonnet-20241022",
      "object": "model",
      "created": 1626777600,
      "owned_by": "anthropic"
    }
  ]
}
```

### Entitlement Filtering
- The model list is pre-filtered by the backend based on:
  1. The user's active `group` (`default`, `pro`, `vip`).
  2. Channels enabled for that group.
  3. Token-specific model restrictions (`token_model_limit`).
- Mobile does not need to guess which models the user is entitled to run; the returned list represents exactly what the current user can invoke.
- When BYOK mode is toggled in the UI, mobile can either show models supported by OpenRouter (e.g. `openrouter/auto`, `deepseek/deepseek-r1`) or query OpenRouter's public model catalog.

---

## 8. Chat Contract

### Endpoint
`POST /v1/chat/completions`

### Headers
```http
POST /v1/chat/completions HTTP/1.1
Host: api.tora.ai
Authorization: Bearer sk-mobile-token-abc123
Content-Type: application/json
Accept: text/event-stream
X-Provider: openrouter  <!-- Omit for Managed Mode -->
```

### Request Payload
```json
{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful AI assistant."
    },
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Analyze this chart:"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
          }
        }
      ]
    }
  ],
  "stream": true,
  "temperature": 0.7,
  "max_tokens": 4096
}
```

---

## 9. Streaming Contract (SSE)

### Protocol Mechanics
- Standard Server-Sent Events (SSE) over HTTP/1.1 or HTTP/2.
- Content-Type: `text/event-stream; charset=utf-8`.
- Chunks separated by `\n\n`.
- Stream termination indicated by `data: [DONE]\n\n`.

### Event Payload Structure
Each event line starts with `data: ` followed by a JSON object:
```json
data: {"id":"chatcmpl-9x123","object":"chat.completion.chunk","created":1728100000,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}
```

### Reasoning Delta Support
For reasoning models (DeepSeek-R1, OpenAI o1/o3, Qwen-Thinking):
```json
data: {"id":"chatcmpl-9x124","choices":[{"index":0,"delta":{"reasoning_content":"Step 1: Calculate..."},"finish_reason":null}]}
```
LumenFlow's existing SSE parser (`OtherProvider` lines 195-200) natively checks `delta['reasoning']` and `delta['reasoning_content']`, emitting `{ 'type': 'reasoning', 'content': ... }` events directly into the UI thinking block.

---

## 10. Cancellation Contract

```text
User taps [Stop] in ChatScreen
           │
           ▼
_streamSubscription.cancel()
           │
           ▼
http.Client.close() / StreamedResponse abort
           │
           ▼ (TCP RST / Socket FIN sent over TLS)
Tora / New-API Backend
           │
           ▼
c.Request.Context().Done() fires
           │
           ├─► Release BYOK Concurrency Lease (releaseBYOKConcurrency)
           ├─► Terminate upstream HTTP/TLS client connection
           └─► Settle partial quota in BillingSession
```

1. **Client Action:** When the user taps the Stop button, LumenFlow calls `_streamSubscription.cancel()` and closes the HTTP client.
2. **Transport Teardown:** The OS closes the TCP socket / TLS session.
3. **Server-Side Handling:** The Go Gin handler detects `c.Request.Context().Done()`. It immediately cancels the outbound request context to the upstream LLM provider, aborting further upstream token generation and cost.
4. **Lease Cleanup:** If in BYOK mode, `defer releaseBYOKConcurrency(userID, leaseID)` executes, ensuring the user's concurrency slot is released immediately.

---

## 11. Error Contract

All relay and API errors are mapped to predictable structures:

| HTTP Status | Error Type / Code | Description | Mobile User Feedback |
|---|---|---|---|
| **401 Unauthorized** | `AUTH_UNAUTHORIZED` / `invalid_token` | Missing, expired, or invalid credentials | Prompt re-login; purge invalid token |
| **401 Unauthorized** | `AUTH_TOKEN_EXPIRED` | 15-minute access token expired | Trigger silent `RefreshAuth` in background |
| **401 Unauthorized** | `AUTH_USER_BANNED` | User account disabled/banned | Display account banned dialog; log out |
| **403 Forbidden** | `insufficient_user_quota` | User wallet or subscription quota exhausted | Prompt user to recharge or upgrade plan |
| **403 Forbidden** | `AUTH_ORIGIN_FORBIDDEN` | Missing/mismatched Origin header on refresh | Add configured Origin header to mobile client |
| **403 Forbidden** | `ACCESS_TOKEN_SCOPE_DENIED` | Personal Access Token lacks route scope | Use proper dashboard session token |
| **404 Not Found** | `byok_provider_not_found` | Selected BYOK provider not configured/disabled | Prompt user to configure provider in Settings |
| **429 Too Many Requests** | `FREE_DAILY_LIMIT` | Daily free tier request limit reached | Prompt user to upgrade to Pro plan |
| **429 Too Many Requests** | `byok_rate_limit_exceeded` | Exceeded 60 BYOK requests per minute | Display countdown banner based on `Retry-After` |
| **429 Too Many Requests** | `byok_concurrency_limit_exceeded`| Exceeded 5 concurrent BYOK requests | "Another request is currently generating..." |
| **502 Bad Gateway** | `upstream_error` | Upstream provider outage or gateway timeout | "Provider temporary error. Tap to retry." |
| **504 Gateway Timeout** | `timeout` | Upstream model response exceeded 600s | "Request timed out. Please try again." |

### JSON Error Envelope Formats
- **Relay Endpoints (`/v1/*`):**
  ```json
  {
    "error": {
      "message": "用户额度不足, 剩余额度: 0.000000",
      "type": "new_api_error",
      "param": "",
      "code": "insufficient_user_quota"
    }
  }
  ```
- **Management Endpoints (`/api/*`):**
  ```json
  {
    "success": false,
    "code": "AUTH_SESSION_REVOKED",
    "message": "用户未登录"
  }
  ```

---

## 12. Subscription Contract

### Available Plans Catalog
- **Endpoint:** `GET /api/subscription/plans`
- **Response:**
  ```json
  {
    "success": true,
    "data": [
      {
        "plan": {
          "id": 1,
          "name": "Pro Monthly",
          "price": 19.99,
          "currency": "USD",
          "billing_cycle": "month",
          "upgrade_group": "pro",
          "features": ["All Flagship Models", "High Concurrency", "Priority Routing"]
        }
      }
    ]
  }
  ```

### Active User Subscription
- **Endpoint:** `GET /api/subscription/self`
- **Response:**
  ```json
  {
    "success": true,
    "data": {
      "billing_preference": "subscription_first",
      "subscriptions": [
        {
          "id": 88,
          "plan_id": 1,
          "plan_name": "Pro Monthly",
          "status": "active",
          "expires_at": 1730764800,
          "amount_total": 5000000,
          "amount_used": 124000
        }
      ]
    }
  }
  ```

### Group Upgrade and Downgrade Dynamics
- Active subscription upgrades the user to `upgrade_group` (e.g. `"pro"`).
- Expiration triggers an automatic background cron in New-API that reverts the user to `prev_user_group` or `"default"`, dynamically adjusting model availability without requiring mobile client intervention.

---

## 13. Store Billing Boundary

1. **Current Reality:** `new-api` contains webhook handlers for Stripe, Epay, Creem, and Waffo. It does **not** contain Apple StoreKit 2 (`/in-app-purchase/verify`) or Google Play Developer API verification endpoints.
2. **Interim Strategy (Phases 7C–7F):**
   - Mobile shows current subscription status and quotas queried from `/api/subscription/self`.
   - "Upgrade Plan" buttons open an external secure checkout webview / Safari View Controller (`SFSafariViewController` / Chrome Custom Tabs) targeting the web portal billing session.
3. **Phase 7G Implementation:**
   - Add backend endpoints `POST /api/subscription/apple/verify` and `POST /api/subscription/google/verify` with server-side signature validation and App Store Server Notifications v2 before offering native in-app purchases.

---

## 14. Attachments & Multimodal Contract

| Attachment Type | LumenFlow Implementation | New-API Support | Protocol Format | Status |
|---|---|---|---|---|
| **Images (PNG, JPEG, WebP, GIF)** | Encodes to Base64 Data URL | Fully supported in relay | `{"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}}` | **Supported End-to-End** |
| **Text Documents (.txt, .md, .csv)** | Local text extraction (up to 10MB) | Appended as text prompt | `{"type": "text", "text": "Attachment: data.csv\n..."}` | **Supported End-to-End** |
| **PDF Documents** | Local text extraction or image rasterization | Depends on target model | Text extracted or sent as images | **Mobile Adaptation Required** |
| **Audio (.mp3, .wav)** | Records local audio | Supported via `/v1/audio/*` | Separate audio endpoint | **Phase 7F Future Work** |
| **Video Files** | Base64 URL format | Supported by Gemini models | `{"type": "video_url", ...}` | **Phase 7F Future Work** |

- Total attachment payload limit enforced by LumenFlow: **50 MB**.
- Single file base64 encoding threshold: **25 MB**.

---

## 15. Conversation Persistence & Offline Contract

1. **Local Ownership:** Conversations, prompt presets, and message histories remain **strictly local** inside the SQLite database (`conversation_database.dart`).
2. **Rationale:**
   - New-API is an inference and billing relay gateway; it does not persist user chat histories.
   - Local persistence ensures instantaneous chat loading, full offline search and viewing capability, and absolute privacy for conversations.
3. **Offline Handling:**
   - When offline, LumenFlow allows reviewing past conversations and drafting messages.
   - Attempting to send a message when offline immediately catches `SocketException` and displays a non-blocking toast: *"No network connection. Message saved as draft."*

---

## 16. Multi-Device Scope Contract

| Data Domain | Cross-Device Sync | Persistence Location |
|---|---|---|
| **Account Identity & Profile** | **YES** | Backend PostgreSQL |
| **Subscription Plan & Entitlements** | **YES** | Backend PostgreSQL |
| **Wallet Quota & Balance** | **YES** | Backend PostgreSQL / Redis |
| **BYOK Provider Configurations** | **YES** | Backend PostgreSQL (`user_providers`) |
| **Conversations & Message Threads** | **NO (Initial Release)** | Local Device SQLite |
| **UI Preferences (Theme, Font, Accent)** | **NO** | Local Device `SharedPreferences` |

---

## 17. Environment Configuration Contract

Mobile configuration must be decoupled from code using build flavors or compile-time Dart defines (`--dart-define`):

```dart
// lib/config/app_config.dart
class AppConfig {
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'https://api.tora.ai',
  );
  
  static const String appEnv = String.fromEnvironment(
    'APP_ENV',
    defaultValue: 'production',
  );
  
  static const bool enableLogging = bool.fromEnvironment(
    'ENABLE_LOGGING',
    defaultValue: false,
  );
}
```

### Environment Mapping
- **Development:** `http://10.0.2.2:3000` (Android Emulator) / `http://localhost:3000` (iOS Sim).
- **Staging:** `https://staging-api.tora.ai`.
- **Production:** `https://api.tora.ai`.

---

## 18. Retry Policy & Client Idempotency Contract

### Safe vs. Dangerous Retries
LumenFlow's `HttpProviderBase` includes automatic exponential backoff. This policy must be restricted as follows:

| Operation | HTTP Method | Automatic Retry Allowed? | Retry Constraints |
|---|---|---|---|
| **Model Listing** | `GET` | **YES** | Max 3 retries (1s, 2s, 4s backoff) on 5xx or network errors |
| **User Profile / Quota** | `GET` | **YES** | Max 3 retries on network drop |
| **Subscription Status** | `GET` | **YES** | Max 3 retries |
| **BYOK Provider Listing** | `GET` | **YES** | Max 3 retries |
| **BYOK Provider Create/Update** | `POST` / `PUT` | **NO** | User must manually tap retry to avoid duplicates |
| **BYOK Provider Deletion** | `DELETE` | **YES** | Idempotent operation |
| **Chat Completions (Streaming)** | `POST` | **CONDITIONAL** | Only retry if socket fails **before** the first byte/chunk is received. Never retry once tokens have started streaming (to prevent duplicate billing). |
| **Subscription Checkout** | `POST` | **NO** | Financial operation; never auto-retry |

---

## 19. Privacy Boundary Contract

Integrating LumenFlow with Tora introduces a defined privacy boundary that must be presented in the mobile Terms of Service and Privacy Policy:

1. **Prompts & Completions:**
   - **Managed Mode:** Prompts traverse Tora New-API to upstream AI providers (OpenAI, Anthropic, Google). New-API logs token usage and metadata; prompt text is not permanently retained unless system debug logging is explicitly enabled.
   - **BYOK Mode:** Prompts traverse New-API directly to the user's personal provider (e.g. OpenRouter). Credentials are decrypted only in memory during transit.
2. **BYOK Credentials:** Plaintext keys are encrypted with AES-256-GCM before database write and never logged.
3. **Local Conversations:** Stored strictly on device storage; never backed up to Tora servers without explicit future user opt-in.

---

## 20. Deprecated Mobile Paths Contract

In **Tora Account Mode**, direct provider clients are bypassed in favor of `ToraProvider`:

| Component / Class | Classification | Rationale |
|---|---|---|
| `OpenAIProvider` (`lib/providers/openai_provider.dart`) | **KEEP FOR STANDALONE MODE** | Preserved if user toggles off Tora Account mode |
| `ClaudeProvider` (`lib/providers/claude_provider.dart`) | **KEEP FOR STANDALONE MODE** | Preserved for standalone mode |
| `GeminiProvider` (`lib/providers/gemini_provider.dart`) | **KEEP FOR STANDALONE MODE** | Preserved for standalone mode |
| `DeepSeekProvider`, `MiniMaxProvider`, etc. | **KEEP FOR STANDALONE MODE** | Preserved for standalone mode |
| Direct SharedPreferences Plaintext API Keys | **REPLACE** | Must use `flutter_secure_storage` for Tora credentials |
| Hardcoded Model Dropdown Lists | **REPLACE** | Must populate dynamically from `GET /v1/models` |

---

## 21. Reusable Mobile Components Matrix

| Component | File Path | Reuse Level | Notes |
|---|---|---|---|
| **ChatScreen UI** | `lib/screens/chat_screen.dart` | **100% REUSE** | Message bubbling, auto-scroll, stop button, attachment picker |
| **Message Bubble & Rendering** | `lib/widgets/message_bubble.dart` | **100% REUSE** | Markdown, LaTeX math, code highlighting, reasoning block |
| **Conversation Database** | `lib/services/conversation_database.dart`| **100% REUSE** | SQLite schema for conversations and messages |
| **Attachment Processing** | `lib/services/file_service.dart` | **100% REUSE** | Image compression, base64 data URL formatting |
| **SSE Event Decoder** | `lib/providers/other_provider.dart` | **95% REUSE** | Port buffer accumulation logic into `ToraProvider` |
| **Cupertino Theme & Widgets** | `lib/utils/app_theme.dart` | **100% REUSE** | iOS design system, dark mode support |
| **Localization (l10n)** | `lib/l10n/*.arb` | **95% REUSE** | Add new strings for Tora login, quota, and BYOK |
| **Settings Navigation Shell** | `lib/screens/settings_screen.dart` | **90% REUSE** | Add Tora Account & Subscription sections |

---

## 22. Backend Gap Matrix

| Identified Gap | Why Required | Blocker for Mobile? | Backend Change Required? | Resolution Phase |
|---|---|---|---|---|
| **No In-App Purchase Receipt Validation** | App Store / Google Play native purchase verification | **NO** (Phases 7C–7F can use web checkout; required only for 7G) | **YES** (Add `/api/subscription/apple/verify` & `/google/verify`) | **Phase 7G** |
| **SessionCookieOriginGuard Rejection on Native Client** | Mobile `http` requests omit `Origin` header | **NO** (Mobile client simply provides `Origin: <host>` header in request) | **NO** (Handled via mobile client config) | **Phase 7C** |
| **No Dedicated Mobile sk- Generation Route** | Mobile needs an `sk-...` relay key to call `/v1/chat/completions` | **NO** (Mobile calls `GET /api/token/` or `POST /api/token/` upon first login) | **NO** (Existing `/api/token/` routes fully support this) | **Phase 7C** |
| **Lack of Cloud Conversation Sync** | Synchronizing chat history between devices | **NO** (Conversations remain local for initial product release) | **NO** (Explicitly deferred) | **Future Phase** |

---

## 23. Frozen Architecture Specification

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        LumenFlow Flutter Client                        │
│                                                                        │
│   ┌──────────────────────┐  ┌────────────────┐  ┌──────────────────┐   │
│   │   ChatScreen & UI    │  │ SQLite Database│  │  Secure Storage  │   │
│   │(Cupertino, Markdown) │  │ (Local History)│  │(Keychain/Keystore│   │
│   └──────────┬───────────┘  └────────────────┘  └────────┬─────────┘   │
│              │                                           │             │
│              ▼                                           ▼             │
│     ┌───────────────────┐                     ┌──────────────────┐     │
│     │   ToraProvider    │◄────────────────────┤   AuthService    │     │
│     │ (OpenAI Protocol) │                     │ (Token Lifecycle)│     │
│     └────────┬──────────┘                     └──────────┬───────┘     │
└──────────────┼───────────────────────────────────────────┼─────────────┘
               │                                           │
    Authorization: Bearer sk-...                Authorization: Bearer <jwt>
    X-Provider: openrouter (optional)           Cookie: new_api_refresh
    POST /v1/chat/completions                   POST /api/user/login
    GET  /v1/models                             GET  /api/user/self
               │                                GET  /api/subscription/self
               │                                GET  /api/user/providers
               │                                           │
               ▼                                           ▼
┌────────────────────────────────────────────────────────────────────────┐
│                         Tora / New-API Backend                         │
│                                                                        │
│    ┌───────────────────────────┐       ┌───────────────────────────┐   │
│    │     Relay Router /v1      │       │     Management API /api   │   │
│    │  (middleware.TokenAuth)   │       │   (middleware.UserAuth)   │   │
│    └─────────────┬─────────────┘       └─────────────┬─────────────┘   │
│                  │                                   │                 │
│                  ▼                                   ▼                 │
│         middleware.BYOKRouter()             Controllers & Services     │
│          ┌───────┴───────┐                   ├── User / Auth Session   │
│          │               │                   ├── BYOK Provider CRUD    │
│       [BYOK]         [Managed]               └── Subscription Service  │
│          │               │                                             │
│    Decrypt Key     Distribute()                                        │
│          │         FreeTier & Quota                                    │
│          │               │                                             │
│          └───────┬───────┘                                             │
│                  ▼                                                     │
│        relay.Relay() Engine                                            │
└──────────────────┬─────────────────────────────────────────────────────┘
                   │
                   ▼
         Upstream AI Providers
    (OpenAI, Anthropic, OpenRouter)
```

---

## 24. Implementation Roadmap (Phases 7C – 7H)

```text
Phase 7C: Auth Foundation  ──►  Phase 7D: Tora Chat  ──►  Phase 7E: BYOK Mgmt
      │                               │                         │
      ▼                               ▼                         ▼
Phase 7F: Subscriptions   ──►  Phase 7G: Store Billing ──► Phase 7H: Staging E2E
```

### Phase 7C — Authentication Foundation
- **Scope:**
  - Add `flutter_secure_storage` dependency to `pubspec.yaml`.
  - Implement `AuthService` (login, register, refresh, logout, session state).
  - Configure HTTP interceptor to handle 401s and inject `Origin` / `Authorization` headers.
  - Implement Login Screen and Register Screen matching LumenFlow's Cupertino design.
- **Exit Criteria:** User can log in, stay logged in across app restarts, refresh tokens silently, and log out cleanly.

### Phase 7D — Tora Chat Integration
- **Scope:**
  - Implement `ToraProvider` extending `HttpProviderBase`.
  - Connect dynamic model catalog fetching via `GET /v1/models`.
  - Wire streaming chat completions with reasoning content deltas.
  - Implement robust socket abort on user stream cancellation.
- **Exit Criteria:** User can select an entitled model, send prompts, receive streaming responses with reasoning blocks, and cancel generation cleanly.

### Phase 7E — BYOK Management
- **Scope:**
  - Build BYOK Settings screen in Cupertino style.
  - Implement provider listing, adding, updating, and deleting via `/api/user/providers`.
  - Implement provider connection test button calling `/api/user/providers/:id/test`.
  - Add BYOK toggle / selector in the main chat screen injecting `X-Provider` header.
- **Exit Criteria:** User can add an OpenRouter key, test connection, select OpenRouter models, and chat with quota bypassing verified.

### Phase 7F — Subscription & Quota UI
- **Scope:**
  - Build Subscription & Wallet screen displaying active plan, quota balance, and renewal date.
  - Handle `HTTP 403 insufficient_user_quota` and `HTTP 429 FREE_DAILY_LIMIT` by presenting an Upgrade modal.
  - Deep-link to web checkout portal for plan upgrades.
- **Exit Criteria:** Quota balances reflect live backend state; quota exhaustion prompts user to upgrade cleanly.

### Phase 7G — Store Billing (IAP)
- **Scope:**
  - Implement backend verification endpoints for Apple App Store StoreKit 2 and Google Play Billing.
  - Integrate `in_app_purchase` Flutter plugin.
  - End-to-end receipt validation and entitlement activation.
- **Exit Criteria:** In-app purchase in sandbox environment successfully upgrades user group and quota on the backend.

### Phase 7H — Staging End-to-End Verification
- **Scope:**
  - Full smoke and regression testing on physical iOS and Android devices against staging backend.
  - Network disconnection and reconnection recovery testing.
  - Verification of zero data leakage and proper token revocation on password change.
- **Exit Criteria:** Ready for App Store and Google Play release submission.

---

## 25. Answers to All 25 Mandatory Questions

### 1. What exact login endpoint should LumenFlow call?
`POST /api/user/login` (Content-Type: `application/json`, payload: `{"username": "<username>", "password": "<password>"}`).

### 2. What credential should it persist?
The session `access_token` (JWT, 15-minute TTL), the `new_api_refresh` cookie/token, and the device relay API key (`sk-...`).

### 3. Where should that credential be stored?
In hardware-backed secure storage via `flutter_secure_storage` (iOS Keychain with `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly` / Android Keystore encrypted preferences).

### 4. What exact logout behavior is required?
Call `POST /api/user/auth/logout` with `Authorization: Bearer <access_token>` and `X-Auth-Session: <session_id>`, then purge all stored tokens from `flutter_secure_storage` and navigate to the unauthenticated view.

### 5. What exact provider CRUD endpoints exist?
- `GET /api/user/providers`
- `POST /api/user/providers`
- `PUT /api/user/providers/:id`
- `DELETE /api/user/providers/:id`

### 6. What exact provider test endpoint exists?
`POST /api/user/providers/:id/test` (executes server-side SSRF-protected probe to upstream provider).

### 7. Does backend ever return stored BYOK plaintext?
**NO.** Plaintext keys are strictly write-only. Database field `APIKeyEncrypted` is tagged with `json:"-"`. The server only returns masked strings (`sk-or-****56`).

### 8. What canonical mechanism should mobile use to select BYOK?
Pass the HTTP header `X-Provider: openrouter` or `X-Provider: custom` on `POST /v1/chat/completions`.

### 9. How should mobile select managed mode?
Omit all `X-Provider` headers, omit the `"provider"` field from the JSON body, and use clean model identifiers (without `openrouter/` prefix).

### 10. What exact model-list endpoint should mobile use?
`GET /v1/models` using `Authorization: Bearer <sk_key>` (or `GET /api/user/models` using `Authorization: Bearer <access_token>`).

### 11. Is model availability already entitlement-aware?
**YES.** The backend filters models via `service.GetGroupsEnabledModels()` matching the user's active `group` (`default`, `pro`, `vip`), which is automatically updated by the subscription state machine.

### 12. What exact chat endpoint should mobile use?
`POST /v1/chat/completions` (OpenAI-compatible streaming and non-streaming relay endpoint).

### 13. Is LumenFlow SSE parsing fully compatible?
**YES.** LumenFlow's SSE parser buffers events on `\n\n`, handles `data: [DONE]`, and extracts both standard `delta.content` and reasoning deltas (`delta.reasoning` / `delta.reasoning_content`).

### 14. How should cancellation work?
Client calls `_streamSubscription.cancel()` and closes the HTTP client socket; the backend Gin context detects `c.Request.Context().Done()`, aborts upstream requests, and releases concurrency locks.

### 15. What are exact quota/subscription errors?
- Wallet/Quota Exhausted: `HTTP 403 Forbidden` (`code: "insufficient_user_quota"`).
- Subscription Quota Exhausted: `HTTP 403 Forbidden` (`code: "insufficient_user_quota"`).
- Free Daily Limit: `HTTP 429 Too Many Requests` (`code: "FREE_DAILY_LIMIT"`).
- Token Invalid/Exhausted: `HTTP 401 Unauthorized` (`code: "invalid_token"`).

### 16. Is HTTP 402 actually used?
**NO.** HTTP 402 is not used anywhere in the codebase; insufficient quota returns `HTTP 403`, and daily limits return `HTTP 429`.

### 17. Which endpoint gives subscription state?
`GET /api/subscription/self` (returns `billing_preference`, `subscriptions`, and `all_subscriptions`).

### 18. Which endpoint gives remaining quota?
`GET /api/user/self` (returns `data.quota` and `data.used_quota`).

### 19. Is a store receipt validation endpoint present?
**NO.** In-app purchase receipt verification for Apple StoreKit or Google Play Billing does not currently exist on the backend.

### 20. Does Phase 7 require backend changes before mobile implementation?
**NO.** Phases 7C through 7F can be implemented immediately with existing backend routes. Only Phase 7G (Store Billing) requires backend receipt validation.

### 21. Which direct provider clients should be deprecated?
Direct client classes in `lib/providers/` (`OpenAIProvider`, `ClaudeProvider`, etc.) should be deprecated in favor of `ToraProvider` when running in Tora Account mode.

### 22. Which LumenFlow components can be reused unchanged?
`ChatScreen`, `ConversationDatabase` (SQLite), `MessageBubble`, `AvatarWidget`, `FileService`, `AppLocalizations`, and the core Cupertino theme system.

### 23. Should conversations remain local for first release?
**YES.** Keeping conversations in local SQLite ensures optimal performance, full offline usability, and maximum user privacy.

### 24. What account state should sync across devices?
User profile, subscription plan, wallet quota, and configured BYOK providers. (Conversations remain device-local).

### 25. What is the final mobile/backend architecture?
A direct, TLS 1.3-encrypted pair: LumenFlow mobile client communicating with New-API's `/api/*` for account/BYOK/subscription management, and `/v1/*` for high-performance inference routing through `BYOKRouter()` and `Distribute()`.

---

## 26. Final Status

```text
FINAL STATUS: CONTRACT READY FOR IMPLEMENTATION
```
