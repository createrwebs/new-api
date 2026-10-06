# Phase 7F — Subscription Entitlement, Quota & Account UX Report

## 1. Executive Summary

Phase 7F implements the read-side subscription, account entitlement, and quota usage user experience for the LumenFlow mobile application connected to the SaaSCover / Tora New-API backend (`feat/formobile`).

Following the strict scope requirements of Phase 7F:
- **Read-Side Only**: Surfaces verified server-authoritative subscription state (`GET /api/subscription/self`), plan catalog (`GET /api/subscription/plans`), and wallet quota usage (`GET /api/user/self`).
- **No Native Store Purchases**: Zero StoreKit, Google Play Billing, or RevenueCat code was introduced. Phase 7G will design and implement the native store purchase trust boundary separately.
- **Backend Read-Only**: Zero Go backend modifications were made. The existing backend routes and database schemas served as the strict source of truth.
- **Backend-Authoritative Trust Boundary**: Mobile acts strictly as a presentation layer. Mobile never treats cached subscription models or user groups as client-side authorization gates.
- **Rigorous Quota Semantics**: Resolved backend quota semantics: `user.Quota` is the remaining wallet balance (not total allowance), and `QuotaPerUnit = 500,000` units per $1.00 USD. The anti-pattern `quota - used_quota` was explicitly avoided.
- **Managed & BYOK Independence**: Server-managed BYOK remains fully functional even when a user has zero quota and no subscription. Managed quota limitations (HTTP 403 `insufficient_user_quota`, HTTP 429 `FREE_DAILY_LIMIT`) never trigger automatic fallback to BYOK, and BYOK failures never fall back to Managed mode.
- **Quality Gate**: 86 of 86 automated unit and mock integration tests pass (69 regression tests + 17 new Phase 7F tests). `flutter analyze` reports 0 issues.

---

## 2. Backend Contract Verification

Every backend endpoint used in Phase 7F was verified directly from the Go source code in `/Users/noppanan/new-api`:

### 2.1 `GET /api/subscription/self`
- **Controller**: [`controller/subscription.go`](file:///Users/noppanan/new-api/controller/subscription.go#L53-L75) (`GetSubscriptionSelf`)
- **Authentication**: `middleware.UserAuth()` (Bearer management JWT)
- **Response Shape**:
  ```json
  {
    "success": true,
    "message": "",
    "data": {
      "billing_preference": "subscription_first",
      "subscriptions": [
        {
          "subscription": {
            "id": 1,
            "user_id": 42,
            "plan_id": 2,
            "amount_total": 5000000,
            "amount_used": 1500000,
            "start_time": 1728000000,
            "end_time": 1730600000,
            "status": "active",
            "upgrade_group": "pro",
            "allow_wallet_overflow": true,
            "last_reset_time": 1728000000,
            "next_reset_time": 1730600000
          }
        }
      ],
      "all_subscriptions": [...]
    }
  }
  ```
- **Observations**: Each subscription item in `subscriptions` and `all_subscriptions` wraps `model.UserSubscription` inside a `"subscription"` JSON key via `model.SubscriptionSummary`. `amount_total` represents quota units allocated for the period (0 = unlimited). `amount_used` represents units consumed in the period.

### 2.2 `GET /api/subscription/plans`
- **Controller**: [`controller/subscription.go`](file:///Users/noppanan/new-api/controller/subscription.go#L32-L51) (`GetSubscriptionPlans`)
- **Authentication**: `middleware.UserAuth()`
- **Compliance Gate**: `operation_setting.IsPaymentComplianceConfirmed()`. If false, returns empty list `[]`.
- **Response Shape**:
  ```json
  {
    "success": true,
    "message": "",
    "data": [
      {
        "plan": {
          "id": 1,
          "title": "Pro Monthly",
          "subtitle": "5M tokens per month",
          "price_amount": 19.99,
          "currency": "USD",
          "duration_unit": "month",
          "duration_value": 1,
          "enabled": true,
          "sort_order": 10,
          "upgrade_group": "pro",
          "total_amount": 5000000,
          "quota_reset_period": "monthly"
        }
      }
    ]
  }
  ```

### 2.3 `GET /api/user/self`
- **Controller**: [`controller/user.go`](file:///Users/noppanan/new-api/controller/user.go#L464-L523) (`GetSelf` / `buildSelfUserData`)
- **Authentication**: `middleware.UserAuth()`
- **Verified Fields**:
  - `id`: User integer ID
  - `username`: Account handle
  - `display_name`: Display name
  - `email`: User email
  - `group`: Active user group string (e.g. `"default"`, `"pro"`, `"vip"`)
  - `quota`: Integer wallet balance quota units
  - `used_quota`: Integer lifetime consumed quota units
  - `request_count`: Total requests processed
  - `status`: 1 = enabled, 2 = disabled/banned

---

## 3. Files Changed

### New Files Created in `scratch/LumenFlow`:
1. [`lib/models/subscription_models.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/models/subscription_models.dart):
   - Typed models: `SubscriptionEntitlement`, `AccountUsage`, `SubscriptionPlanInfo`, `SubscriptionSelfResponse`.
   - Math helpers for remaining period quota, usage ratio, USD conversions, and date formatters.
2. [`lib/services/subscription_service.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/services/subscription_service.dart):
   - Singleton service managing entitlement lifecycle, account-scoped caching, environment-scoped caching, manual refresh, and Free Tier detection.
3. [`lib/screens/account_subscription_screen.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/screens/account_subscription_screen.dart):
   - Comprehensive account & subscription screen presenting account identity, active plan entitlement, progress bar, wallet balance, lifetime usage, available plan catalog, and billing portal status notes.
4. [`test/subscription_service_test.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/test/subscription_service_test.dart):
   - 17 unit and integration tests covering model math, service lifecycle, error mapping, multi-account isolation, and BYOK independence.

### Modified Files in `scratch/LumenFlow`:
1. [`lib/models/auth_models.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/models/auth_models.dart):
   - Added `requestCount` field to `AuthenticatedUser`.
2. [`lib/services/tora_api_client.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/services/tora_api_client.dart):
   - Added `getSubscriptionSelf({required String accessToken})` and `getSubscriptionPlans({required String accessToken})`.
3. [`lib/services/auth_service.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/services/auth_service.dart):
   - Added `getValidAccessToken()` and `refreshCurrentUser()`.
   - Hooked `SubscriptionService().clearCache()` into `switchEnvironment()`, `logout()`, and `_handleSessionLoss()`.
4. [`lib/screens/settings_screen.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/screens/settings_screen.dart):
   - Added `Subscription & Quota` navigation tile under `Tora Account`.
   - Enhanced user header tile to display plan name and wallet balance snippet.
5. [`lib/screens/chat_screen.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/screens/chat_screen.dart):
   - Added pre-send check verifying active managed model is present in `ToraInferenceClient().listModels()`. If plan expired and model is restricted, prompts user without deleting history or silently changing connection mode.
   - Enhanced `_handleStreamError` for `isQuotaExhausted` and `isDailyLimit` to display actionable dialog navigating to `AccountSubscriptionScreen`.

---

## 4. Subscription Domain Model

The domain models isolate mobile presentation from database schema internals:

### `SubscriptionEntitlement`
- `id`: int
- `userId`: int
- `planId`: int
- `planTitle`: String?
- `amountTotal`: int (quota allowance for the billing period; 0 = unlimited)
- `amountUsed`: int (quota units consumed in the billing period)
- `startTime`: int (epoch seconds)
- `endTime`: int (epoch seconds)
- `status`: String (`'active'`, `'expired'`, `'cancelled'`)
- `upgradeGroup`: String (`'pro'`, `'vip'`, etc.)
- `allowWalletOverflow`: bool
- `lastResetTime`: int
- `nextResetTime`: int
- **Calculated Properties**:
  - `isActive`: `status == 'active' && now < endTime`
  - `isExpired`: `status == 'expired' || now >= endTime`
  - `isUnlimited`: `amountTotal == 0`
  - `remainingQuota`: `(amountTotal > 0) ? max(0, amountTotal - amountUsed) : 0`
  - `usageRatio`: `(amountTotal > 0) ? (amountUsed / amountTotal).clamp(0.0, 1.0) : 0.0`

### `AccountUsage`
- `walletBalanceQuota`: int (from `user.quota`)
- `lifetimeUsedQuota`: int (from `user.used_quota`)
- `requestCount`: int (from `user.request_count`)
- `group`: String (from `user.group`)
- **Calculated Properties**:
  - `usdBalance`: `walletBalanceQuota / 500,000.0`
  - `usdLifetimeUsed`: `lifetimeUsedQuota / 500,000.0`
  - `isWalletZero`: `walletBalanceQuota <= 0`
  - `groupDisplayName`: Formatted badge label (e.g. `"Pro"`, `"VIP"`, `"Default"`)

### `SubscriptionPlanInfo`
- `id`: int
- `title`: String
- `subtitle`: String
- `priceAmount`: double
- `currency`: String
- `durationUnit`: String (`'month'`, `'year'`, etc.)
- `durationValue`: int
- `totalAmount`: int
- `upgradeGroup`: String
- `quotaResetPeriod`: String

---

## 5. SubscriptionService

`SubscriptionService` acts as the management-plane coordinator for all entitlement and usage data:
- **Singleton with Dependency Injection**: Default singleton constructor for UI widgets, and `SubscriptionService.custom({apiClient, authService})` for automated testing.
- **Cache Management**: Caches `SubscriptionSelfResponse`, `AccountUsage`, and `List<SubscriptionPlanInfo>` keyed by `account_id` and `backendBaseUrl`.
- **Automatic Invalidation**: Listens to `AuthService`. When `_authService.isAuthenticated` becomes false, or when account/environment switches, `clearCache()` scrubs all in-memory presentation models immediately.
- **Safe Fallback**: If network fails or server returns 500/401/403, logs safe error code and retains clean UI state without throwing unhandled exceptions.

---

## 6. Account Overview

The Account & Settings UX is structured as follows:

```text
SettingsScreen
 └── Tora Account Section
      ├── User Identity Tile: [Username] • [Plan Badge] • [Environment]
      │    └── (Tap) → AccountSubscriptionScreen
      ├── Subscription & Quota Tile: Plan: [Plan Name] • Balance: [$X.XX]
      │    └── (Tap) → AccountSubscriptionScreen
      ├── AI Connections (BYOK) Tile: Managed • OpenRouter • Custom
      │    └── (Tap) → ByokSettingsScreen
      └── Sign Out Tile (Destructive Action)
```

In `AccountSubscriptionScreen`:
1. **Account Identity**: Username, email (if set), access group, account status (Active/Disabled).
2. **Subscription Entitlement**: Plan name, status badge, validity expiration date, quota allowance progress bar (`used / total units`), remaining quota, and wallet overflow permission.
3. **Wallet Balance & Lifetime Usage**: USD balance and raw quota units, cumulative lifetime USD usage, total requests processed.
4. **Available Plans**: List of purchasable plans from backend catalog with price, duration, and quota allowance.
5. **Notice Card**: Explains Free Tier or Store billing availability without fabricating links.

---

## 7. Plan UX

Plan presentation preserves backend identity without fabricating imaginary commercial tiers:
- Backend group `pro` -> Display label: `Pro Plan`
- Backend group `vip` -> Display label: `VIP Plan`
- Backend group `default` (or empty) with zero quota -> Display label: `Free Tier`
- Backend group `default` with positive wallet quota -> Display label: `Standard`
- Active subscription with explicit `planTitle` -> Display label: `planTitle` (e.g. `Pro Monthly`)

---

## 8. Quota / Usage Semantics

During Phase 7F audit, backend quota semantics were verified in [`common/constants.go`](file:///Users/noppanan/new-api/common/constants.go#L22) and [`controller/relay.go`](file:///Users/noppanan/new-api/controller/relay.go):
- **Quota Unit Scaling**: `common.QuotaPerUnit = 500,000.0` ($0.002 / 1K tokens = $1.00 USD per 500,000 quota units).
- **`user.Quota` Meaning**: **`user.Quota` IS the current remaining wallet balance**. When relay inference settles, `user.Quota` decreases (`quota - amount`) and `user.UsedQuota` increases (`used_quota + amount`).
- **Anti-Pattern Avoided**: Computing `remaining = quota - used_quota` is **FATALLY FLAWED** because `quota` is already the remaining balance; subtracting `used_quota` would compute `remaining - lifetime_used`, which is meaningless or negative.
- **Subscription Quota Separated**: Subscription quota has its own matching numerator and denominator (`amount_used` and `amount_total` in `model.UserSubscription`). The progress bar is only computed when `amount_total > 0`.

---

## 9. Free Tier

Free Tier semantics were verified in [`middleware/free_tier.go`](file:///Users/noppanan/new-api/middleware/free_tier.go) and [`model/free_tier_quota.go`](file:///Users/noppanan/new-api/model/free_tier_quota.go):
- **Eligibility**: Non-admin (`user.Role < 10`), zero wallet balance (`user.Quota <= 0`), and zero active non-expired subscriptions.
- **Allowance**: 20 requests per Bangkok calendar day (`Asia/Bangkok` timezone, UTC+7).
- **Reset**: Every day at 00:00 Bangkok time.
- **Enforcement**: Middleware reserves requests in table `free_tier_daily_usages`. When exhausted, returns HTTP 429 with:
  ```json
  {
    "error": {
      "code": "FREE_DAILY_LIMIT",
      "message": "free tier daily request limit exceeded",
      "type": "rate_limit_error"
    }
  }
  ```
- **Backend Gap Documented**: Backend has no read endpoint exposing the daily used or remaining request count. Mobile displays the verified rule ("Daily allowance of 20 requests. Resets daily at midnight Bangkok time.") and explicitly avoids inventing a local client-side count (`BACKEND UX GAP — FREE TIER REMAINING COUNT`).

---

## 10. Managed Model Entitlement

Managed models remain strictly server-authoritative:
- Mobile fetches available models via `GET /v1/models` with Bearer relay token.
- Backend filters the catalog based on the user's current group (e.g. `default` vs `pro`).
- Mobile does NOT maintain a client-side hardcoded model allowlist.
- When subscription expires or group changes, `ToraInferenceClient().clearModelCache()` ensures the next catalog fetch reflects server-filtered availability.

---

## 11. BYOK Independence

Phase 7F preserves complete isolation and independence between Managed and BYOK modes:
- **No Subscription + BYOK**: A user with zero wallet balance and no subscription can configure and use OpenRouter or Custom BYOK without restriction.
- **Active Subscription + Managed**: A user with an active subscription can use Managed inference without configuring any BYOK credentials.
- **No Auto-Fallback**:
  - If Managed inference hits quota exhaustion (`insufficient_user_quota`) or daily limit (`FREE_DAILY_LIMIT`), mobile shows a dialog offering to view plans/usage. It **NEVER** silently falls back to BYOK.
  - If BYOK returns 401/404/rate-limit, mobile displays the BYOK provider error. It **NEVER** silently falls back to Managed mode.
- Changing connection mode always requires explicit user action.

---

## 12. Entitlement Errors

Centralized error handling maps backend status codes and error payloads into actionable UI states:

| Backend Status | Error Code / Body Signature | Domain Error Code | Mobile Presentation |
|---|---|---|---|
| HTTP 403 | `insufficient_user_quota` | `quotaExhausted` | "Quota exhausted. Please view your plan or contact support." + Dialog with "View Plan & Usage" button |
| HTTP 429 | `FREE_DAILY_LIMIT` | `dailyLimitReached` | "Daily free limit reached. Please upgrade your plan or try again tomorrow." + Dialog with "View Plan & Usage" button |
| HTTP 404 | `model_not_found` / `no available channel` | `modelUnavailable` | "The selected model is currently unavailable. Please select another model." |
| Pre-send Check | Model missing from `/v1/models` | `modelUnavailable` | "The model '<id>' is no longer available with your current plan." + Prompt to select model or switch connection |
| HTTP 401 | `AUTH_TOKEN_EXPIRED` | `unauthorized` | Re-authenticates via refresh token or prompts sign-in |

---

## 13. Expiration Behavior

When an active subscription expires:
1. Backend downgrades the user group (e.g. `pro` -> `default`).
2. Refreshing subscription updates entitlement status to `expired` and displays `Free Tier` or `Standard`.
3. `ToraInferenceClient().clearModelCache()` flushes cached models. Next `GET /v1/models` excludes Pro-only models.
4. If an existing conversation used a model that is now restricted:
   - **Conversation History Preserved**: All historical messages remain fully viewable.
   - **No Silent Modification**: The model is NOT silently changed behind the user's back.
   - **Pre-Send Verification**: On next send attempt, mobile checks model availability in `/v1/models`, intercepts the send, and presents a non-destructive dialog prompting the user to either select an available model or switch connection mode.

---

## 14. Billing Portal

Investigation in `/Users/noppanan/new-api`:
- Backend contains payment initiation endpoints (`/api/subscription/stripe/pay`, `/epay/pay`, `/creem/pay`, `/waffo-pancake/pay`).
- Backend contains **NO generic self-service customer billing portal URL endpoint** (e.g., Stripe Customer Portal session creation).
- Mobile displays: `BILLING PORTAL NOT AVAILABLE — Native in-app subscriptions (App Store & Google Play) are planned for Phase 7G. Web self-service billing portal is currently not configured.`
- Zero fabricated URLs or external query-parameter credential handoffs were introduced.

---

## 15. Account Isolation

Multi-account isolation was validated through automated tests:
- User A (Pro) logs in -> Entitlements and wallet balance are cached in `SubscriptionService`.
- User A logs out -> `AuthService.logout()` invokes `SubscriptionService().clearCache()`, scrubbing all cached DTOs, usage, and plans.
- User B (Free Tier) logs in -> `SubscriptionService` loads User B's profile.
- **Result**: User B never briefly or permanently sees User A's Pro plan, quota units, or model catalog.

---

## 16. Environment Isolation

Environment isolation is enforced by scoping cache keys to `AppEnvironment.backendBaseUrl`:
- When switching environment from Development to Staging via `AuthService.switchEnvironment()`:
  - Active inference streams are aborted.
  - Model catalog is cleared.
  - `SubscriptionService().clearCache()` immediately flushes all entitlement and usage data.
- Development subscription state never appears in Staging or Production.

---

## 17. Offline Behavior

When network is unavailable:
- Existing local conversation history remains fully readable (persisted in local SQLite database).
- `AccountSubscriptionScreen` displays the last known cached data with an explicit timestamp: `Updated: HH:mm:ss`. If no cached data exists, shows an unavailable indicator.
- Refresh actions fail gracefully with user-friendly notices without crashing.
- Offline state never permits local inference bypass.

---

## 18. Backend Gap Matrix

| Identified Gap | Classification | Impact & Mobile Handling |
|---|---|---|
| **Free Tier Remaining Count** | NON-BLOCKING UX GAP | Backend persists daily usage in `free_tier_daily_usages`, but exposes no read API for current used/remaining count. Mobile documents the 20-request limit and midnight Bangkok reset without fabricating a client counter. |
| **Billing Portal URL** | NON-BLOCKING UX GAP | Backend lacks a self-service customer portal session API. Mobile displays subscription state and plan info, and defers purchase execution to Phase 7G native store billing. |
| **Per-Model Quota Estimation** | FUTURE | Backend calculates pre-consumption server-side in `relay.PrepareRequestBilling`. Mobile does not attempt client-side price computation. |

---

## 19. Tests Added

17 automated tests were authored in [`test/subscription_service_test.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/test/subscription_service_test.dart):

1. `SubscriptionEntitlement parses active subscription and calculates remaining quota correctly`
2. `SubscriptionEntitlement handles expired status and past end_time`
3. `SubscriptionEntitlement handles unlimited quota correctly (amount_total = 0)`
4. `AccountUsage enforces verified backend quota semantics`
5. `AccountUsage handles zero quota safely without negative balance formatting`
6. `SubscriptionPlanInfo parses plan metadata and formats pricing`
7. `SubscriptionService loads active subscription and account usage successfully`
8. `SubscriptionService detects Free Tier accurately when user has zero quota and no subscription`
9. `SubscriptionService handles 401 Unauthorized safely and clears cache`
10. `SubscriptionService handles 500 server error and malformed JSON safely without crashing`
11. `Zero quota and inactive subscription does NOT disable BYOK provider selection`
12. `Managed quota exhaustion (insufficient_user_quota) maps accurately without automatic BYOK switch`
13. `Free daily limit (FREE_DAILY_LIMIT) maps accurately without automatic BYOK switch`
14. `BYOK error maps to byokCredentialInvalid without automatic Managed fallback`
15. `User A Pro logout and User B Free login never leaks User A entitlement`
16. `Environment switch immediately clears subscription cache and isolates data`
17. `Subscription expiration preserves historical conversation and avoids silent send`

---

## 20. Regression Results

Full test suite execution outcome:

```text
00:08 +86: All tests passed!
```

Breakdown:
- **Phase 7C / 7C-R**: Authentication, session refresh, token provisioning, multi-device isolation (all passed).
- **Phase 7D**: Tora managed chat, streaming, cancellation, error mapping (all passed).
- **Phase 7E**: BYOK provider CRUD, key rotation, connection mode switching (all passed).
- **Phase 7F**: Subscription models, quota semantics, Free Tier, account isolation (all passed).
- **Total**: 86 passed, 0 failed, 0 skipped.
- **Static Analysis**: `flutter analyze` -> `No issues found! (ran in 2.0s)`.

---

## 21. Manual E2E Debt

As documented in Phases 7C through 7E:
- **`MANAGED MANUAL E2E STILL NOT RUN — ENVIRONMENT UNAVAILABLE`**
- **`MANUAL BYOK E2E NOT RUN — ENVIRONMENT UNAVAILABLE`**
- **`SUBSCRIPTION READ E2E NOT RUN — ENVIRONMENT UNAVAILABLE`**

*Reason*: Local port 3000 hosts the Next.js web application rather than the compiled Go New-API backend binary, and live payment provider credentials are not provisioned in the local development sandbox. All verification is grounded on rigorous contract-level mock server tests and static analysis.

---

## 22. Known Limitations

1. **Native Purchases Deferred**:
   - In accordance with Phase 7F constraints, App Store (StoreKit) and Google Play Billing are not implemented. Users cannot purchase subscriptions within the app during Phase 7F.
2. **Free Tier Counter Granularity**:
   - Because the backend does not expose remaining free tier counts, the app displays the rule and resets rather than a live countdown.
3. **Web Portal Availability**:
   - Self-service web billing portal links are not exposed by the backend; subscription management is read-only in mobile settings.

---

## 23. Phase 7G Readiness

Phase 7F completes the read-side foundation required for native billing:
- Account identity, active subscription state, and quota usage are cleanly modeled and surfaced.
- The plan catalog (`GET /api/subscription/plans`) is parsed into strongly typed models ready for mapping to StoreKit / Google Play Billing product identifiers in Phase 7G.
- Trust boundaries are verified: backend remains the sole authority for granting entitlement and processing server-side receipts.

---

FINAL STATUS: READY FOR PHASE 7G — NATIVE STORE BILLING BACKEND CONTRACT
