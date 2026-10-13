# Phase 7E — Server-Managed BYOK Mobile Integration Report

**Date**: October 5, 2026  
**Repositories**:
- Mobile: `scratch/LumenFlow` (Branch: `main`)
- Backend: `/Users/noppanan/new-api` (Branch: `feat/formobile`)  
**Phase**: 7E — Server-Managed BYOK Mobile Integration  
**Final Status**: READY FOR PHASE 7F — SUBSCRIPTION ENTITLEMENT UX

---

## 1. Executive Summary

Phase 7E successfully implemented server-managed Bring-Your-Own-Key (BYOK) support inside the LumenFlow mobile application, integrating directly with the existing SaaSCover / Tora New-API backend (`/Users/noppanan/new-api` on branch `feat/formobile`).

Following the strict security and architectural invariants established across Phases 5B–5D, 6A–6H, and 7A–7D:
- **Zero Local Plaintext BYOK Key Storage**: The mobile app never persists plaintext BYOK secrets in `SharedPreferences`, SQLite (`ConversationDatabase`), or in-memory caches. Keys exist only ephemerally in function arguments during creation or rotation and are garbage-collected immediately upon HTTP dispatch.
- **Unified Inference Pipeline**: BYOK and Managed modes share the exact same `ToraInferenceClient` and `ToraProvider` pipeline. No separate BYOK HTTP clients were introduced.
- **Header & Secret Routing Separation**: All chat requests transmit strictly the device relay token (`Authorization: Bearer <device-relay-token>`). BYOK requests attach only the routing hint `X-Provider: openrouter` or `X-Provider: custom`. Zero BYOK secrets are ever sent in chat requests.
- **Strict No-Fallback Invariant**: A failure or deletion of a BYOK provider fails explicitly with a BYOK configuration error and never falls back silently to Managed mode, preventing unexpected depletion of user account balance/quota.
- **Single POST Attempt Invariant**: Chat completions retain an attempt count of exactly 1 (`attempt count == 1`), strictly forbidding automatic transient retries on POST requests.
- **Upstream 401 Discrimination**: Upstream BYOK credential errors (401) are mapped to `byokCredentialInvalid` and explicitly prohibited from triggering device relay token reprovisioning.
- **Backward Compatibility & Account Isolation**: Historical conversations retain their connection mode, provider identifier, and model name. Conversations remain isolated by `account_id` and environment.

The backend Go codebase in `/Users/noppanan/new-api` remained completely untouched. The Flutter test suite expanded from 52 to 69 tests with a 100% pass rate, and `flutter analyze` reports 0 issues.

---

## 2. Invariant Adherence Matrix

| Invariant | Specification Requirement | Implementation Verification | Status |
|---|---|---|:---:|
| **Zero Local Plaintext Storage** | BYOK secrets must never be written to disk, SQLite, or SharedPreferences. | Verified in `ByokService` and `test/byok_service_test.dart`: plaintexts are omitted from all local stores; only server-masked keys (`••••1234`) are retained. | **PASSED** |
| **Pipeline Unification** | Re-use `ToraInferenceClient` directly; do NOT build a separate BYOK inference client. | Implemented via `provider` parameter on `ToraInferenceClient.sendChatStreaming` and `sendChatSync`. | **PASSED** |
| **Routing Hint Separation** | Managed mode sends no `X-Provider` header. BYOK sends `X-Provider: openrouter` or `custom`. | Verified in `test/tora_inference_test.dart`: header is omitted for managed, present for BYOK. | **PASSED** |
| **Zero Secrets Over Wire in Chat** | Chat requests send only `Authorization: Bearer <device-relay-token>`. No secrets. | Verified via network interception assertions in `tora_inference_test.dart`. | **PASSED** |
| **Strict No-Fallback** | If BYOK provider fails or is deleted, fail explicitly. Never silently deplete managed quota. | Verified in `tora_inference_test.dart`: 404 throws `byokProviderNotFound` with `requestCount == 1`. | **PASSED** |
| **Single POST Attempt** | Chat completion requests must never be automatically retried across transient errors. | `requestCount == 1` enforced on 500, 502, network failure, and BYOK failure. | **PASSED** |
| **401 Discrimination** | Upstream 401 must not trigger device relay token reprovision loop. | Handled via `isByokCredentialInvalid` in `_withRelayToken` and `ToraInferenceException.fromResponse`. | **PASSED** |
| **Provider Limit Enforcement** | Respect backend limit of 1 OpenRouter and 1 Custom provider per user. | Enforced in `ByokService.createProvider` with `byok_duplicate_provider` error code. | **PASSED** |
| **Environment Partitioning** | BYOK caches and settings must be isolated per environment base URL. | `_cachedProvidersByEnv` strictly keyed by `AppEnvironment.backendBaseUrl` and cleared on logout. | **PASSED** |
| **Backend Immutability** | No modifications permitted to `/Users/noppanan/new-api`. | `git status` on `/Users/noppanan/new-api` confirms clean working tree on `feat/formobile`. | **PASSED** |

---

## 3. Mobile BYOK Architecture & System Flow

```text
                           LumenFlow Mobile Application
                                        │
                         Connection Selection in Chat
                                        │
                    ┌───────────────────┴───────────────────┐
                    ▼                                       ▼
            [Tora Managed]                         [Server-Managed BYOK]
                    │                                       │
            Selected Model                         Selected Model
            (e.g. gpt-4o)                          (e.g. anthropic/claude-3.5)
                    │                                       │
                    └───────────────────┬───────────────────┘
                                        │
                                        ▼
                              ToraInferenceClient
                    (Authorization: Bearer <device-relay-token>)
                    (If BYOK: X-Provider: openrouter | custom)
                    (If Managed: No X-Provider header)
                                        │
                                        ▼ POST /v1/chat/completions
                    ┌───────────────────────────────────────┐
                    │     SaaSCover / Tora New-API Backend   │
                    └───────────────────┬───────────────────┘
                                        │
                              BYOKRouter Middleware
                                        │
                    ┌───────────────────┴───────────────────┐
                    ▼                                       ▼
           [Managed Pipeline]                      [BYOK Pipeline]
           - User quota deducted                   - Zero quota deduction
           - System channel routing                - Decrypt server-stored key
                                                   - Route to OpenRouter / Custom
                                                            │
                                                            ▼
                                                   Upstream LLM Provider
```

### Components Implemented:
1. **`lib/models/byok_provider.dart`**: Domain entity representing server-managed providers, masked keys, validation, and connectivity test results.
2. **`lib/models/connection_mode.dart`**: Unified connection account model (`managed` vs `byok`) tracking routing metadata and display properties.
3. **`lib/services/byok_service.dart`**: Management service managing provider CRUD, connectivity testing, cache isolation, provider limit enforcement, and legacy key migration.
4. **`lib/screens/byok_settings_screen.dart`**: Management UI providing connection status, masked keys, connection testing, rotation, deletion dialogs, and legacy migration.
5. **`lib/screens/chat_screen.dart`**: Connection switching modal, provider metadata synchronization, and custom model entry dialog.

---

## 4. Backend Route Inventory Used

Mobile communicates with the backend across two strictly isolated planes:

### A. Management Plane (JWT Access Token)
- **Base Auth**: `Authorization: Bearer <access-token>`
- **Cookie**: `new_api_refresh=<cookie>`

| Method | Exact Path | Request Payload | Response Data | Purpose in Mobile |
|---|---|---|---|---|
| `GET` | `/api/user/providers` | None | `{"success": true, "data": [ByokProvider...]}` | List all server-managed providers for authenticated user. |
| `POST` | `/api/user/providers` | `{"provider", "name", "api_key", "base_url", "enabled"}` | `{"success": true, "data": ByokProvider}` | Register a new server-managed BYOK provider. |
| `PUT` | `/api/user/providers/:id` | `{"name"?, "api_key"?, "base_url"?, "enabled"?}` | `{"success": true, "data": ByokProvider}` | Update name/base_url or rotate API key. |
| `DELETE` | `/api/user/providers/:id` | None | `{"success": true, "message": "..."}` | Delete a BYOK provider from the server. |
| `POST` | `/api/user/providers/:id/test` | None | `{"success": true, "message": "..."}` | Test upstream connectivity using server-stored encrypted key. |

### B. Inference Plane (Device Relay Token)
- **Base Auth**: `Authorization: Bearer <device-relay-token>`
- **No Management Headers**: No JWT, session cookies, or plaintext user keys.

| Method | Exact Path | Extra Headers | Request Body | Purpose in Mobile |
|---|---|---|---|---|
| `GET` | `/v1/models` | None | None | Fetch available system/managed models. |
| `POST` | `/v1/chat/completions` | `X-Provider: openrouter` or `custom` (BYOK only) | Standard OpenAI Chat Completion JSON (`stream: true/false`, `model`, `messages`) | Perform chat inference without quota deduction. |

---

## 5. Upstream vs Relay Key Isolation Audit

An architectural audit was performed to verify complete separation between the device relay token and upstream BYOK keys:

1. **Relay Token Isolation**:
   - The device relay token (`sk-...`) is generated during authentication (Phase 7C-R) and stored in `FlutterSecureStorage`.
   - It is scoped strictly to the inference plane and has zero authority to query management endpoints (e.g. `/api/user/providers` returns 401 with relay token).
2. **Upstream BYOK Key Isolation**:
   - The mobile client never receives plaintext BYOK keys from the server.
   - When the user enters a BYOK key in `ByokSettingsScreen`, the plaintext key exists in memory only during the JSON encoding of `POST /api/user/providers` or `PUT /api/user/providers/:id`.
   - The backend encrypts the key using AES-GCM and stores ciphertext in PostgreSQL. The response payload returns only `api_key: "••••1234"`.
   - Mobile stores only this masked string (`maskedKey`).
3. **Chat Request Hygiene**:
   - During BYOK chat requests, mobile transmits `Authorization: Bearer <device-relay-token>` and `X-Provider: <provider>`.
   - No `x-api-key`, query parameter, or payload field containing the upstream secret is transmitted.
   - The backend `BYOKRouter` intercepts `X-Provider`, decrypts the user's stored upstream key in memory, and proxies the request to the upstream target.

---

## 6. BYOK Configuration UX & Screen Structure

A dedicated Cupertino management interface was implemented in `lib/screens/byok_settings_screen.dart` accessible via **Settings -> AI Connections (BYOK)**:

1. **Active Connection Section**:
   - Shows currently active inference connection (`Tora Managed` vs `OpenRouter (••••9876)`).
   - Clarifies billing impact: "Billed to Tora Account Quota" vs "Direct Upstream Billing".
2. **Legacy Migration Banner**:
   - Displayed conditionally if a legacy unmigrated local platform key (from LumenFlow's previous direct OpenRouter integration) is detected on the device.
   - Explains that keys can be migrated securely to the server with explicit user consent.
3. **System Providers Section**:
   - Allows one-tap switching back to `Tora Managed` mode.
4. **Server-Managed BYOK Providers List**:
   - Displays all registered providers with masked keys, provider type, and status indicator.
   - Shows capacity counter: `(X/2)` respecting backend composite limits.
   - Action sheet on tap:
     - **Test Connection**: Runs connectivity test against upstream.
     - **Set as Active Connection**: Switches active chat mode.
     - **Rotate API Key**: Opens modal to replace upstream key with zero prefilled values.
     - **Edit Configuration**: Modifies display name or custom base URL.
     - **Delete Provider**: Prompts confirmation explaining that existing conversations remain readable.
5. **Add Provider Modal**:
   - Segmented control between `OpenRouter` and `Custom OpenAI-compatible`.
   - Dynamically hides base URL field for OpenRouter (preset to standard endpoint).
   - Validates required fields before submitting to server.

---

## 7. BYOK Testing & Validation Flow

Connection testing is executed through `POST /api/user/providers/:id/test`:
1. **Backend Handling**:
   - The server makes an upstream test call using the stored encrypted credentials.
   - It validates upstream authentication and model endpoint accessibility.
2. **Mobile Sanitization**:
   - `ByokTestResult.fromResponse(json)` parses the backend result.
   - Errors are sanitized by `ToraInferenceException.fromResponse` and mapped to clean user-facing descriptions.
   - Internal stack traces, raw IP addresses, and encrypted blobs are never rendered in the UI.
3. **UI Feedback**:
   - A modal Cupertino dialog provides instantaneous visual confirmation (Success checkmark or sanitized failure message).

---

## 8. BYOK Key Rotation Flow

Key rotation is implemented in `updateProvider`:
1. **Zero Prefilled Secrets**:
   - The rotation modal presents an empty `CupertinoTextField` with `placeholder: 'Enter new API key (leave empty to keep current)'`.
   - Plaintext keys from previous inputs are never cached or populated into controllers.
2. **Atomic Upstream Update**:
   - If a new key is entered, mobile dispatches `PUT /api/user/providers/:id` with `{"api_key": "<new-key>"}`.
   - The server encrypts the new secret, invalidates server-side channel caches, and returns the newly masked key (e.g. `••••5678`).
3. **Local State Update**:
   - The in-memory cache is updated with the new masked representation.
   - If the rotated provider is currently active, `_activeConnection` updates its display label while keeping the conversation continuity intact.

---

## 9. Provider Deletion & Historical Conversation Safety

When a user deletes a BYOK provider:
1. **Server Cleanup**:
   - `DELETE /api/user/providers/:id` removes the provider and associated channel configuration.
2. **Active Connection Reset**:
   - If the deleted provider was the active connection, `ByokService` immediately resets `_activeConnection` to `ConnectionAccount.managed()`.
3. **Historical Chat Safety**:
   - Historical conversations stored in SQLite (`ConversationDatabase`) preserve their messages, timestamps, and model names.
   - Conversations are not deleted when a provider is removed.
4. **Pre-Send Validation**:
   - In `ChatScreen._sendMessage()`, when opening a conversation linked to a deleted or disabled provider, the app checks provider existence.
   - If the provider no longer exists, sending is blocked with an informative dialog:
     > *"This BYOK connection is no longer available because the provider was removed or disabled. Please switch to another connection in settings."*
   - Historical messages remain fully readable, scrollable, and exportable.

---

## 10. Connection Mode Architecture (Managed vs BYOK)

The app maintains a clear distinction between inference modes:

```dart
enum ConnectionMode {
  managed,
  byok,
}
```

- **Managed Mode**:
  - `ConnectionAccount.managed()`
  - Calls `/v1/chat/completions` with NO `X-Provider` header.
  - Backend routes request through managed channels and charges Tora account quota.
  - Model list queried dynamically from `/v1/models`.
- **BYOK Mode**:
  - `ConnectionAccount.fromByokProvider(provider)`
  - Calls `/v1/chat/completions` with `X-Provider: openrouter` or `X-Provider: custom`.
  - Backend routes request through user's private channel with zero quota charge.
  - Model selection offers provider-specific presets or custom model entry.

---

## 11. Inference Pipeline Unification & Header Contract

`ToraInferenceClient` acts as the single point of contact for inference:

```dart
// lib/services/tora_inference_client.dart
Future<Map<String, dynamic>> sendChatSync({
  required String model,
  required List<Map<String, dynamic>> messages,
  double? temperature,
  int? maxTokens,
  bool thinkingMode = false,
  String? provider, // <-- Phase 7E BYOK Routing Parameter
})

Stream<Map<String, dynamic>> sendChatStreaming({
  required String model,
  required List<Map<String, dynamic>> messages,
  double? temperature,
  int? maxTokens,
  bool thinkingMode = false,
  String? provider, // <-- Phase 7E BYOK Routing Parameter
})
```

### Exact Header Specification:
- When `provider == null`:
  - `Authorization: Bearer <device-relay-token>`
  - `X-Provider`: **OMITTED**
- When `provider == 'openrouter'`:
  - `Authorization: Bearer <device-relay-token>`
  - `X-Provider: openrouter`
- When `provider == 'custom'`:
  - `Authorization: Bearer <device-relay-token>`
  - `X-Provider: custom`

---

## 12. Model Discovery Gap Analysis & Solution

### Gap Identified:
`BACKEND GAP — BYOK MODEL DISCOVERY`  
The backend Go implementation of `/v1/models` is not intercepted by `BYOKRouter()` and returns only system-managed models. Furthermore, no endpoint exists at `/api/user/providers/:id/models` to discover models dynamically for an arbitrary user provider.

### Implemented Solution:
1. **OpenRouter Mode**:
   - `ToraProvider.fetchModels()` returns a curated list of top OpenRouter models:
     - `anthropic/claude-3.5-sonnet`
     - `deepseek/deepseek-r1`
     - `openai/gpt-4o`
     - `meta-llama/llama-3.3-70b-instruct`
     - `google/gemini-2.0-flash-001`
   - Plus, an action sheet option: *"Enter Custom Model ID..."* enables typing any valid OpenRouter model identifier (e.g. `anthropic/claude-3-opus`).
2. **Custom Provider Mode**:
   - Automatically provides the currently configured model or defaults to `gpt-4o`.
   - Offers *"Enter Custom Model ID..."* dialog allowing arbitrary model identifiers supported by the user's custom endpoint.
3. **Per-Conversation Model Persistence**:
   - Selected model identifiers are stored directly on the `Conversation` entity and saved to SQLite, ensuring model consistency across chat sessions.

---

## 13. Strict No-Fallback Implementation

Under no circumstances should an unavailable BYOK provider silently fall back to Managed mode, which would trigger billing against the user's Tora quota without their knowledge.

### Verification:
1. When a request to `/v1/chat/completions` with `X-Provider` fails with:
   - `404` (`byok_provider_not_found`)
   - `429` (`byok_rate_limit_exceeded`)
   - `429` (`byok_concurrency_limit_exceeded`)
   - `500` (`byok_decrypt_failed`)
2. `ToraInferenceException.fromResponse` maps the response to dedicated BYOK error codes.
3. The request terminates immediately. The client does NOT strip the `X-Provider` header or retry as Managed.
4. Verified in automated test: `No Managed fallback when BYOK provider returns 404 (request count == 1)`.

---

## 14. Upstream 401 vs Relay Token 401 Discrimination

Phase 7D implemented automatic token reprovisioning when a relay token expires or is deleted:
```text
401 Unauthorized -> Call RelayTokenProvisioner -> Retry once with new token
```

In Phase 7E, an upstream BYOK provider returning 401 (e.g. user provided an invalid OpenRouter key) must **NOT** trigger this reprovisioning loop:

### Resolution:
1. `ToraInferenceErrorCode.byokCredentialInvalid` is mapped when:
   - Status code is 401 AND body contains `byok_credential_invalid` or `upstream_credential_invalid`.
2. In `ToraInferenceClient._withRelayToken`:
   ```dart
   if (e.statusCode == 401 && !reprovisioned && !e.isByokCredentialInvalid) {
     // Reprovision relay token
   }
   ```
3. If `e.isByokCredentialInvalid == true`, the error propagates immediately to the UI prompting the user to update their BYOK API key in settings. The device relay token is left untouched.

---

## 15. Single POST Attempt Guarantee

To ensure total financial and quota safety:
- Automatic retries on `POST /v1/chat/completions` are strictly prohibited.
- Network timeouts, socket failures, 500 errors, and 502 Bad Gateway responses are surfaced to the user as retryable UI actions; the client library never performs silent automatic background retries.
- Verified in automated test: `Single POST attempt invariant: no automatic transient retry on BYOK failure`.

---

## 16. Legacy Platform Key Migration Strategy & Results

Prior to Phase 7E, LumenFlow stored direct provider keys (such as OpenRouter) locally inside `SettingsService` / `SharedPreferences`.

### Safe Migration Workflow:
1. **Passive Detection**:
   - `ByokService.checkLegacyPlatformKey('openrouter')` inspects local storage without altering it.
2. **User Consent**:
   - If an unmigrated key exists and OpenRouter is not yet configured on the server, a prominent migration banner is displayed in `ByokSettingsScreen`.
   - The user explicitly taps *"Migrate to Server-Managed BYOK"*.
3. **Safe Execution**:
   - `ByokService.migrateLegacyKey`:
     - Dispatches `createProvider` to store and encrypt the key on the server.
     - **Only on confirmed 200 response from backend**: Clears the plaintext key from local `SettingsService` storage.
     - If the network call fails, the local key is preserved to prevent data loss.

---

## 17. Automated Test Suite Results

The automated test suite in `scratch/LumenFlow` was executed via `flutter test`. All **69 out of 69** tests passed across all test files.

### Summary by Test File:
- `test/byok_service_test.dart`: 10 passed
- `test/tora_inference_test.dart`: 30 passed (including 7 new Phase 7E BYOK tests)
- `test/api_client_test.dart`: 5 passed
- `test/auth_service_test.dart`: 11 passed
- `test/relay_token_provisioner_test.dart`: 5 passed
- `test/multi_device_environment_test.dart`: 5 passed
- `test/auth_models_test.dart`: 3 passed

### Phase 7E Specific Tests:
1. `ByokService CRUD & Invariants Tests listProviders returns empty if not authenticated` — **PASSED**
2. `ByokService CRUD & Invariants Tests listProviders fetches from server and caches responses per environment` — **PASSED**
3. `ByokService CRUD & Invariants Tests createProvider rejects duplicate provider type (1 OpenRouter, 1 Custom limit)` — **PASSED**
4. `ByokService CRUD & Invariants Tests createProvider sends plaintext key ONLY over wire and NEVER stores it locally` — **PASSED**
5. `ByokService CRUD & Invariants Tests updateProvider rotates key on server and never stores new secret locally` — **PASSED**
6. `ByokService CRUD & Invariants Tests deleteProvider removes provider and resets active connection if active was deleted` — **PASSED**
7. `ByokService CRUD & Invariants Tests testProvider returns sanitized ByokTestResult` — **PASSED**
8. `ByokService CRUD & Invariants Tests onLogout clears caches and resets active connection to managed` — **PASSED**
9. `ByokService CRUD & Invariants Tests legacy platform key migration creates server provider and deletes local key` — **PASSED**
10. `ConnectionAccount Model Tests ConnectionAccount serialization and factory round-trip` — **PASSED**
11. `Phase 7E — Server-Managed BYOK Inference Tests BYOK OpenRouter attaches X-Provider: openrouter and zero secrets` — **PASSED**
12. `Phase 7E — Server-Managed BYOK Inference Tests BYOK Custom attaches X-Provider: custom` — **PASSED**
13. `Phase 7E — Server-Managed BYOK Inference Tests Managed mode explicitly omits X-Provider header` — **PASSED**
14. `Phase 7E — Server-Managed BYOK Inference Tests No Managed fallback when BYOK provider returns 404 (request count == 1)` — **PASSED**
15. `Phase 7E — Server-Managed BYOK Inference Tests Single POST attempt invariant: no automatic transient retry on BYOK failure` — **PASSED**
16. `Phase 7E — Server-Managed BYOK Inference Tests Upstream BYOK 401 maps to byokCredentialInvalid without reprovisioning relay token` — **PASSED**
17. `Phase 7E — Server-Managed BYOK Inference Tests Conversation persistence preserves connectionMode, providerType, providerId, and model` — **PASSED**

---

## 18. Static Analysis Results

Execution of `flutter analyze` across `scratch/LumenFlow`:

```bash
$ flutter analyze
Analyzing LumenFlow...
No issues found! (ran in 1.8s)
```

**0 warnings, 0 errors, 0 lints.**

---

## 19. Regression Verification

Verification of previous phase contracts:
- **Phase 7C / 7C-R Authentication & Tokens**:
  - Management JWT and cookie lifecycle untouched.
  - Multi-device installation ID and relay token isolation fully functional.
  - Account switching and environment switching cleanups verified.
- **Phase 7D Managed Chat & Streaming**:
  - Managed mode continues to send zero BYOK headers.
  - SSE streaming, reasoning stream emission, and request cancellation fully operational.
  - Conversation persistence continues to isolate by account ID.

---

## 20. Live E2E Verification Status

As documented in Phase 7C and 7D:
- **`MANAGED MANUAL E2E STILL NOT RUN — ENVIRONMENT UNAVAILABLE`**
- **`MANUAL BYOK E2E NOT RUN — ENVIRONMENT UNAVAILABLE`**

*Reason*: Local port 3000 runs the Next.js web application rather than the compiled Go New-API backend binary, and live upstream LLM credentials are not provisioned in the local development sandbox. All verification is grounded on rigorous contract-level mock server tests and static analysis.

---

## 21. Remaining Risks & Edge Cases

1. **Custom Endpoint Protocol Discrepancies**:
   - Custom BYOK endpoints rely on upstream servers adhering strictly to OpenAI chat completion JSON formatting. Non-standard JSON responses are caught safely by `ToraSseParser` and mapped to `upstreamError`.
2. **Upstream Rate Limiting**:
   - OpenRouter or custom providers may impose strict concurrency or token rate limits. These are surfaced to the user as clear BYOK error messages.
3. **Model Availability**:
   - Because the backend does not expose dynamic model enumeration for BYOK providers (`BACKEND GAP — BYOK MODEL DISCOVERY`), users must type their custom model ID if using models outside the preset list.

---

## 22. Next Steps / Phase 7F Preparation

Phase 7E establishes complete inference capability across both Managed and BYOK modes.

The next integration milestone is **Phase 7F — Subscription Entitlement UX**:
- Querying user subscription status and quota tier from backend management routes (`GET /api/user/self`, `GET /api/user/subscription`).
- Displaying current balance, quota usage, and subscription tier in user settings.
- Preparing entitlement gates without implementing StoreKit or Google Play Billing until payment backends are ready.

---

## 23. Release Gate & Final Status

All Phase 7E deliverables, invariants, test cases, and documentation are complete and verified.

FINAL STATUS: READY FOR PHASE 7F — SUBSCRIPTION ENTITLEMENT UX
