# Phase 7C-R — Mobile Authentication Multi-Device & Environment Verification Report

## Executive Summary

Phase 7C-R performed a rigorous security audit, multi-device verification, and targeted remediation of the LumenFlow mobile authentication foundation prior to implementing chat integration (Phase 7D). 

The audit identified a critical multi-device concurrency vulnerability in the initial Phase 7C implementation:
- **Finding (Multi-Device Relay Token Collision)**: The initial `RelayTokenProvisioner` searched the backend server for a static token name (`"LumenFlow Mobile"`). When a user logged into their account from two distinct mobile devices (Device A and Device B), Device B discovered and reused Device A's server token. Consequently, when Device A logged out, it issued a server-side revocation (`DELETE /api/token/{id}`) that immediately severed Device B's inference capability without warning.

**Remediation Executed in Phase 7C-R**:
1. **Cryptographic Per-Installation Identity**: Implemented a privacy-safe, non-hardware installation identifier (`tora.installation_id`) generated using `Random.secure()` (32-character hex).
2. **Dedicated Token Naming**: Bound server token names to the installation ID (`LumenFlow Mobile [${shortId}]`, 27 characters, well within the backend's 50-character limit). Device A and Device B now receive completely separate server-side relay tokens.
3. **Targeted Revocation Invariant**: Logging out on Device A deletes only Device A's dedicated token (`DELETE /api/token/{id_A}`). Device B's inference key remains active and unaffected.
4. **Environment Credential Scoping**: Upgraded `SecureCredentialStore` to strictly isolate credentials per backend environment (`tora.<env>.*`), preventing token leakage when toggling between Development, Staging, and Production.
5. **Session Cookie & Origin Guard Verification**: Verified mobile client headers against the backend `SessionCookieOriginGuard` (`new-api/middleware/auth_origin.go`). The client transmits matching `Origin: ${scheme}://${host}` on sensitive auth endpoints, satisfying CSRF protection requirements.
6. **Multi-Account & Conversation Partitioning**: Verified that switching accounts on the same physical device isolates secure credentials and partitions local SQLite chat history by `account_id`.

All 29 automated unit and integration tests pass with 0 failures. Static analysis reports 0 issues. No backend production code was modified.

---

## Existing Relay Token Behavior (Pre-Remediation)

In the initial Phase 7C implementation, the token provisioning sequence operated as follows:

```text
User Login
  │
  ▼
GET /api/token/?p=0&size=100
  │
  ├── Token named "LumenFlow Mobile" found?
  │     ├── YES: POST /api/token/{id}/key (reveal key) -> Store locally
  │     └── NO:  POST /api/token/ (create "LumenFlow Mobile") -> POST key -> Store locally
  │
User Logout
  │
  ▼
DELETE /api/token/{id} (Server-side revocation)
```

This design assumed a single-device mental model.

---

## Multi-Device Finding (Root Cause Analysis)

When multiple mobile devices authenticate under the same user account, the static lookup created severe cross-device interference:

```text
User Alice
   │
   ├── Device A (Login)
   │     └── Creates Token ID 10 ("LumenFlow Mobile")
   │
   └── Device B (Login)
         └── Queries /api/token/ -> Finds Token ID 10 ("LumenFlow Mobile")
         └── Reuses Token ID 10
```

### Impact & Failure Modes:
1. **Cross-Device Revocation**: When Device A logs out, `AuthService.logout()` calls `RelayTokenProvisioner.revokeDeviceToken()`, issuing `DELETE /api/token/10`. Immediately, Device B's inference requests fail with `401 Unauthorized` (`ERR_INVALID_TOKEN`).
2. **Conflated Telemetry & Quota**: Device A and Device B shared identical request metering and token audit records in the backend database.
3. **Shared Blast Radius**: If Device A's secure storage was compromised or extracted, revoking Device A's token also revoked Device B.

---

## Remediation

The remediation guarantees that **every physical installation receives its own dedicated server-side relay token**:

```text
User Alice
   │
   ├── Installation A (inst_a1b2c3d4)
   │     └── Token: "LumenFlow Mobile [a1b2c3d4]" (ID: 101)
   │
   └── Installation B (inst_e5f6g7h8)
         └── Token: "LumenFlow Mobile [e5f6g7h8]" (ID: 102)
```

### Invariants Enforced:
1. **Search Precision**: When Device A logs in, it only searches for tokens matching its own specific label: `"LumenFlow Mobile [${shortId}]"`.
2. **Independent Revocation**: Logout on Device A issues `DELETE /api/token/101`. Token 102 on Device B remains active on the server.
3. **Idempotency**: Restarting the app on Device A reuses Token 101 without creating duplicate tokens.

---

## Installation Identity Implementation

The installation identity is generated and stored using strict privacy and security principles:

```dart
// lib/services/secure_credential_store.dart
Future<String> getOrCreateInstallationId() async {
  final existing = await _storage.read(key: _keyInstallationId);
  if (existing != null && existing.isNotEmpty) {
    return existing;
  }
  final random = Random.secure();
  final values = List<int>.generate(16, (i) => random.nextInt(256));
  final id = values.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  await _storage.write(key: _keyInstallationId, value: id);
  return id;
}
```

### Privacy & Compliance Guarantees:
- **Zero Hardware Fingerprinting**: No IMEI, MAC address, Android ID, IDFA, AAID, or hardware serial numbers are accessed or stored.
- **Cryptographic Entropy**: Generated via `Random.secure()` (128 bits of entropy).
- **Non-Credential Status**: The installation identifier is strictly a device label tag. It is never used as an authentication secret or bearer credential.
- **Backend Schema Compatibility**: The backend database constraint (`model/token.go`) enforces `Name varchar(50)`. The token label format `LumenFlow Mobile [${shortId}]` consumes exactly 27 characters (`16 + 1 + 1 + 8 + 1`), well within the limit.

---

## Token Lifecycle State Machine

```
┌────────────────────────────────────────────────────────┐
│                   App First Launch                     │
│        Generate and persist tora.installation_id       │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│                   User Login (/api)                    │
│     Store access_token, session_id, refresh_cookie     │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│              Relay Token Provisioning                  │
│       List tokens via GET /api/token/?p=0&size=100     │
└──────────────┬──────────────────────────┬──────────────┘
               │                          │
  Token exists for this installation?     │ No token for this installation
               │                          │
               ▼                          ▼
┌─────────────────────────────┐ ┌─────────────────────────────┐
│ POST /api/token/{id}/key    │ │ POST /api/token/            │
│ (Reveal existing secret key)│ │ (Create dedicated token)    │
│                             │ │ POST /api/token/{new_id}/key│
└──────────────┬──────────────┘ └──────────┬──────────────────┘
               │                           │
               └───────────┬───────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│      Store relay_token_key & ID in Secure Storage      │
│          Scoped under tora.<environment>.*             │
└──────────────────────────┬─────────────────────────────┘
                           │
             ┌─────────────┴─────────────┐
             ▼                           ▼
┌─────────────────────────────┐ ┌─────────────────────────────┐
│    Inference Requests       │ │         User Logout         │
│  Authorization: Bearer sk-..│ │  DELETE /api/token/{id}     │
│   Target: /v1/* endpoints   │ │  Wipe local env credentials │
└─────────────────────────────┘ └─────────────────────────────┘
```

---

## Account Switching on the Same Device

When User A logs out and User B logs in on the same physical device:
1. **User A Logout**: Device issues `DELETE /api/token/{id_A}` to server. Local credentials for User A are wiped from secure storage.
2. **User B Login**: The device's `installation_id` is preserved. `RelayTokenProvisioner` queries User B's tokens on the server for `LumenFlow Mobile [${shortId}]`.
3. **No Collision**: Because this installation has never logged in under User B, no matching token exists on User B's account. A new token (`id_B`) is provisioned under User B.
4. **Complete Isolation**: User B cannot access User A's token or management session.

---

## Conversation Isolation

LumenFlow local chat history (`ConversationDatabase` and `ConversationService`) has been partitioned to prevent data leakage between accounts:
- Every `Conversation` record contains an `accountId` property.
- For unauthenticated / guest usage, `accountId` defaults to `'local'`.
- Upon successful login, `AuthService` switches the active account ID to the authenticated user ID (`'usr_${userId}'`).
- All SQLite queries enforce `where: 'account_id = ?'`.
- When switching accounts, User B cannot read, modify, or delete User A's local conversation history.

---

## Canonical Backend Environment & Domain Verification

The backend repository (`/Users/noppanan/new-api`, branch `feat/formobile`) was inspected to establish canonical environment endpoints:

| Environment | Canonical Base URL | Source / Verification Method | Status |
|:---|:---|:---|:---|
| **Development (iOS / Desktop)** | `http://localhost:3000` | `.env.example`, `docker-compose.yml`, Go default port `:3000` | Verified |
| **Development (Android)** | `http://10.0.2.2:3000` | Android emulator loopback alias to host machine | Verified |
| **Staging** | `https://staging-api.tora.ai` | `phase_7b_mobile_backend_contract.md` (Not in repo infra) | Provisional |
| **Production** | `PRODUCTION URL REQUIRES OPERATOR CONFIRMATION` | Codebase is domain-agnostic reverse-proxy target | Awaiting Operator DNS |

### Production Domain Configuration
- The Go backend does not hardcode its public production hostname; it listens on `:3000` behind Nginx/Cloudflare.
- Candidate domains identified in documentation: `https://api.tora.ai`, `https://api.toraapi.com`.
- Mobile architecture supports compile-time override:
  ```bash
  flutter build apk --dart-define=APP_ENV=production --dart-define=BACKEND_BASE_URL=https://api.tora.ai
  ```
- In debug builds, the in-app environment selector allows runtime switching.

---

## Origin Guard & Cookie Contract Verification

The backend origin security middleware was audited in `new-api/middleware/auth_origin.go`:

```go
// SessionCookieOriginGuard checks Origin on sensitive session endpoints
func SessionCookieOriginGuard() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !common.IsSecureCookieMode() {
            c.Next()
            return
        }
        // Intercepts only refresh and logout
        if c.Request.URL.Path != "/api/user/auth/refresh" && c.Request.URL.Path != "/api/user/auth/logout" {
            c.Next()
            return
        }
        origin := c.Request.Header.Get("Origin")
        if !isAllowedSessionOrigin(origin, c.Request) {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid origin"})
            return
        }
        c.Next()
    }
}
```

### Mobile Contract Compliance:
1. **Target Endpoints**: Guard only intercepts `/api/user/auth/refresh` and `/api/user/auth/logout`.
2. **Allowed Origins**: `isAllowedSessionOrigin` allows requests where `Origin` equals `${requestScheme}://${req.Host}` or is listed in `SESSION_COOKIE_TRUSTED_URL`.
3. **Mobile Client Origin Generation**: `AppEnvironment.originHeader` generates exact matching scheme and host strings (e.g. `http://localhost:3000`, `https://api.tora.ai`), ensuring mobile requests pass the origin check when secure cookies are enabled.
4. **Dual Transport**: `ToraApiClient` transmits both the `Cookie: session=...` header and the `X-Session-Id` header, ensuring compatibility with all backend configurations.

---

## Environment Credential Isolation

To prevent credential leakage when developers or testers switch environments, `SecureCredentialStore` enforces namespaced storage keys:

| Scope | Key Pattern | Description |
|:---|:---|:---|
| **Global** | `tora.installation_id` | Privacy-safe installation UUID (shared across envs) |
| **Development** | `tora.development.access_token` | Management access token for local dev |
| **Development** | `tora.development.relay_token_key` | Inference key (`sk-...`) for local dev |
| **Staging** | `tora.staging.access_token` | Management access token for staging |
| **Staging** | `tora.staging.relay_token_key` | Inference key (`sk-...`) for staging |
| **Production** | `tora.production.access_token` | Production management access token |
| **Production** | `tora.production.relay_token_key` | Production inference key (`sk-...`) |

Switching environments in `AuthService` switches the active prefix. Credentials in Staging are never sent to Production or overwritten by Development sessions.

---

## Tests Added

Comprehensive automated tests were implemented in `test/multi_device_environment_test.dart` and existing test suites:

1. **Multi-Device Distinct Tokens**: Verifies that Device A and Device B logging into the same account create and receive distinct server tokens (`LumenFlow Mobile [aaaa1111]` vs `LumenFlow Mobile [bbbb4444]`).
2. **Targeted Revocation Safety**: Verifies that logout on Device A calls `deleteToken` on Token A only; Token B remains active on the server and in Device B's local storage.
3. **Idempotent Restart**: Verifies that Device A restarting reuses its existing enabled token without creating duplicate tokens on the server.
4. **Account Switch Isolation**: Verifies that switching accounts on the same device isolates credentials and conversation database partitions.
5. **Environment Scoping Isolation**: Verifies that Staging credentials cannot be read under Production mode and vice versa.
6. **Local Conversation Partitioning**: Verifies that SQL queries strictly filter chat history by `account_id`.

---

## Test Results

### Automated Suite (`flutter test`)
```text
00:01 +29: All tests passed!
```
- Total test files: 5
- Total test cases: 29
- Passing: 29
- Failing: 0
- Skipped: 0

### Static Code Analysis (`flutter analyze`)
```text
Analyzing LumenFlow...
No issues found! (ran in 1.7s)
```

### Backend Code Verification
```text
On branch feat/formobile
nothing to commit, working tree clean (untracked documentation reports only)
```
Zero production Go files were modified.

---

## Remaining Risks & Mitigation

| Risk | Severity | Mitigation |
|:---|:---|:---|
| **Production DNS Confirmation** | Low | Documented in `AppEnvironment`. Build pipeline uses `--dart-define=BACKEND_BASE_URL` once production domain is finalized by ops. |
| **App Uninstall Orphan Tokens** | Informational | If a user uninstalls the app without logging out, their server token remains until expired or deleted via Web console. Token quota limit prevents abuse. |
| **Name Field Length Boundary** | Informational | Backend enforces `varchar(50)`. Formatted string is exactly 27 characters. Well within safety bounds. |

---

## Phase 7D Readiness

The authentication, multi-device credential isolation, and environment foundations are fully verified and hardened:
- Management plane (`/api/*`) and Inference plane (`/v1/*`) credentials strictly separated.
- Multi-device concurrency safely handled with per-installation relay tokens.
- Hardware-backed secure storage configured with fallback protection.
- Conversation storage partitioned by user identity.

LumenFlow is now ready to begin Phase 7D (Chat Integration).

---

FINAL STATUS: READY FOR PHASE 7D — TORA CHAT INTEGRATION
