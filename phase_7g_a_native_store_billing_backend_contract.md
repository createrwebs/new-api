# Phase 7G-A — Native Store Billing Backend Trust Boundary & Verification Contract

## Executive Summary

Phase 7G-A establishes the server-side trust boundary for native mobile subscription billing across Apple App Store (StoreKit 2) and Google Play Billing (Subscriptions v2). In alignment with strict financial security principles established in Phase 6, client-supplied claims of purchase success, plan tier, quota allocation, or subscription duration are treated as entirely untrusted.

The SaaSCover / Tora New-API backend serves as the sole authoritative arbiter of subscription entitlement. All native store purchases must submit raw cryptographic or store-issued proof (Apple JWS compact transaction token or Google Play purchase token) directly to dedicated backend verification endpoints. The backend independently validates the proof against store developer APIs, maps the verified store product to internal plans, checks for cross-account ownership conflicts, and atomically updates the user's subscription and group tier within isolated database transactions.

The implementation is verified with automated test suites covering replay protection, concurrency, cross-account theft rejection, renewal extension, quota reset, immediate refund revocation, and environment isolation. Because live production store verification requires real App Store Connect and Google Play Console credentials, the system safely disables verification and rejects requests with `STORE_SERVER_UNAVAILABLE` rather than trusting mock data when credentials are absent.

---

## Existing Subscription Architecture

An audit of [`model/subscription.go`](file:///Users/noppanan/new-api/model/subscription.go), [`controller/subscription.go`](file:///Users/noppanan/new-api/controller/subscription.go), and related services revealed the baseline subscription engine:

1. **Subscription Identity**:
   - Internal plans are defined by [`SubscriptionPlan`](file:///Users/noppanan/new-api/model/subscription.go#L146) (`id`, `title`, `duration_unit`, `duration_value`, `total_amount`, `upgrade_group`, `downgrade_group`, `quota_reset_period`).
   - Active user instances are represented by [`UserSubscription`](file:///Users/noppanan/new-api/model/subscription.go#L253) (`id`, `user_id`, `plan_id`, `amount_total`, `amount_used`, `start_time`, `end_time`, `status`, `upgrade_group`, `prev_user_group`, `downgrade_group`).
2. **Multi-Subscription Support**:
   - A user may possess multiple historical or concurrent active subscription rows. Active subscriptions are queried via `status = 'active' AND end_time > now`.
3. **Group Tier Elevation & Reversion**:
   - Upon subscription creation ([`CreateUserSubscriptionFromPlanTx`](file:///Users/noppanan/new-api/model/subscription.go#L484)), if `plan.UpgradeGroup` is set and differs from the user's current group, the user's prior group is snapshotted into `sub.PrevUserGroup` and `user.group` is updated to `plan.UpgradeGroup`.
   - Upon expiration, background daemon [`ExpireDueSubscriptions`](file:///Users/noppanan/new-api/model/subscription.go#L1148) (executed every minute by [`StartSubscriptionQuotaResetTask`](file:///Users/noppanan/new-api/service/subscription_reset_task.go#L29)) sets `status = 'expired'`. If no other active upgraded subscription exists, the user is downgraded back to `sub.DowngradeGroup` or `sub.PrevUserGroup`.
4. **Quota Accounting**:
   - `sub.AmountTotal` represents the plan's quota allowance for the billing period (0 = unlimited).
   - Chat/inference requests invoke [`PreConsumeUserSubscription`](file:///Users/noppanan/new-api/model/subscription.go#L1306) which atomically deducts quota by incrementing `sub.AmountUsed`.
   - When quota is exhausted, [`UserActiveSubscriptionsAllowWalletOverflow`](file:///Users/noppanan/new-api/model/subscription.go#L887) checks if fallback to wallet balance is permitted.

---

## Existing Financial Transaction Primitives

Phase 6 and existing payment implementations (Stripe, Creem, Epay, Waffo) established the following financial primitives that Phase 7G-A reuses:

1. **Idempotent Order Completion**:
   - Handled via [`CompleteSubscriptionOrder`](file:///Users/noppanan/new-api/model/subscription.go#L569) using `lockForUpdate(tx)` on `SubscriptionOrder.TradeNo`.
   - If an order has already transitioned to `common.TopUpStatusSuccess`, duplicate callbacks return `nil` immediately without double-crediting quota or extending duration.
2. **User Row Serialization**:
   - `lockForUpdate(tx).Select("id").Where("id = ?", order.UserId).First(&userRow)` serializes concurrent operations for the same user, preventing race conditions during purchase limit checks and group updates.
3. **Audit Logging & TopUp Records**:
   - Successful transactions create or update a corresponding [`TopUp`](file:///Users/noppanan/new-api/model/topup.go) row with matching `TradeNo` for unified balance accounting and write a detailed audit message via [`RecordLog`](file:///Users/noppanan/new-api/model/log.go).

---

## Store Trust Boundary

The trust boundary strictly segregates client and server responsibilities:

```text
┌────────────────────────────────┐
│      LumenFlow Mobile App      │
│  - Receives purchase proof     │
│  - Transmits raw proof only    │
│  - NEVER decides plan or quota │
└───────────────┬────────────────┘
                │ POST /api/subscription/{apple|google}/verify
                │ Payload: { signed_transaction_info } OR { purchase_token, product_id }
                ▼
┌────────────────────────────────────────────────────────┐
│            SaaSCover / Tora New-API Backend            │
│  1. Authoritative verification with Apple / Google     │
│  2. Cryptographic signature and cert chain checks      │
│  3. Environment isolation (Sandbox vs Production)      │
│  4. Server-side SKU -> Internal plan mapping lookup    │
│  5. Cross-account binding enforcement                  │
│  6. Atomic database transaction settlement             │
│  7. Idempotent record persistence (Replay protection)  │
└────────────────────────────────────────────────────────┘
```

Mobile requests containing `plan_id`, `quota`, `price`, `group`, or `duration` are ignored by the verification endpoints. Only verified store proof drives internal entitlement.

---

## Apple Verification Design

Integrated with **Apple StoreKit 2 & App Store Server API**:

1. **Client Proof**:
   - LumenFlow retrieves `Transaction.jwsRepresentation` (`signedTransactionInfo`) upon purchase or transaction update and transmits it to the backend.
2. **Server-Side Verification**:
   - Implemented in [`service.DefaultAppleVerifier`](file:///Users/noppanan/new-api/service/store_verifier.go#L62).
   - Decodes the JWS compact format (`header.payload.signature`).
   - If certificate chain (`x5c`) is present:
     - Extracts leaf certificate and verifies ES256 signature using the leaf public key.
     - Validates intermediate certificate chaining up to Apple Root CA.
   - Decodes transaction claims:
     - `transactionId`: Unique transaction ID.
     - `originalTransactionId`: Canonical subscription chain identifier.
     - `bundleId`: App bundle identifier.
     - `productId`: Store product SKU.
     - `purchaseDate`: Purchase timestamp in milliseconds.
     - `expiresDate`: Subscription expiration timestamp in milliseconds.
     - `revocationDate`: Non-zero if refunded or revoked.
     - `environment`: `"Production"` or `"Sandbox"`.
     - `appAccountToken`: Account UUID passed at purchase time.

---

## Google Verification Design

Integrated with **Google Play Billing (Subscriptions v2)**:

1. **Client Proof**:
   - LumenFlow retrieves `purchaseToken`, `productId`, and `orderId` from Google Play Billing and submits them to `POST /api/subscription/google/verify`.
2. **Server-Side Verification**:
   - Implemented in [`service.DefaultGoogleVerifier`](file:///Users/noppanan/new-api/service/store_verifier.go#L182).
   - Queries Google Play Developer API:
     `GET https://androidpublisher.googleapis.com/androidpublisher/v3/applications/{packageName}/purchases/subscriptionsv2/tokens/{token}`
   - Evaluates `subscriptionState`:
     - 1 (`PENDING`): Returns `STORE_VERIFICATION_PENDING`. Entitlement is withheld until payment completes.
     - 2 (`ACTIVE`): Verified active.
     - 6 (`CANCELED`): User cancelled auto-renew; active until `expiryTime`.
     - 7 (`EXPIRED`): Returns `STORE_SUBSCRIPTION_EXPIRED`.
   - Acknowledgment:
     - Google Play requires acknowledgment within 72 hours to prevent automatic refunds.
     - The New-API backend issues `POST ...:acknowledge` **only after** transactional entitlement settlement succeeds.

---

## User Binding & Cross-Account Theft Prevention

A major store billing threat is cross-account replay: User A purchases a subscription, then User B captures the receipt/token and submits it to claim entitlement on a second account.

### Solution: `StoreSubscriptionBinding` Table

Every store subscription chain is permanently bound to the first authenticated New-API account that presents it:

- Apple key: `original_transaction_id`
- Google key: `purchase_token` (or canonical `obfuscatedExternalAccountId`)
- Database constraint: `UNIQUE(platform, store_original_id)`

```text
User A submits Store Original ID "orig_123"
  → Binding created for User A (User A = Owner)

User B submits Store Original ID "orig_123"
  → Backend checks StoreSubscriptionBinding
  → Owner (User A) != Caller (User B)
  → REJECTED with 409 Conflict: STORE_TRANSACTION_ALREADY_BOUND
```

---

## SKU / Plan Mapping

Product identifiers are configured server-side in [`StoreProductMapping`](file:///Users/noppanan/new-api/model/store_billing.go#L39):

| Platform | Store Product ID | Store Base Plan ID | Internal Plan ID | Environment |
| :--- | :--- | :--- | :--- | :--- |
| `apple` | `com.saascover.tora.pro.monthly` | *none* | 1 (Pro Monthly) | `production` |
| `apple` | `com.saascover.tora.pro.yearly` | *none* | 2 (Pro Yearly) | `production` |
| `google` | `tora_pro_sub` | `pro-monthly` | 1 (Pro Monthly) | `production` |
| `google` | `tora_pro_sub` | `pro-yearly` | 2 (Pro Yearly) | `production` |

The server resolves the mapping via [`GetStoreProductMapping`](file:///Users/noppanan/new-api/model/store_billing.go#L159). If a product ID is unrecognized, the request is rejected with `STORE_PRODUCT_UNRECOGNIZED`.

---

## Environment Isolation

To prevent test purchases in Apple Sandbox or Google Test Track from granting real production access:

1. **Apple Sandbox Check**:
   - If JWS `environment == "Sandbox"` and `APPLE_ALLOW_SANDBOX != true`, the request is rejected with `STORE_ENVIRONMENT_MISMATCH`.
2. **Google Test Purchase Check**:
   - If `testPurchase` flag is set and `GOOGLE_ALLOW_TEST_PURCHASE != true`, the request is rejected with `STORE_ENVIRONMENT_MISMATCH`.
3. **Application Identity**:
   - Apple JWS `bundleId` must strictly equal `APPLE_BUNDLE_ID` (`STORE_BUNDLE_MISMATCH` on mismatch).
   - Google `packageName` must strictly equal `GOOGLE_PACKAGE_NAME`.

---

## State Machine & Transitions

```text
      [ Store Purchase Proof Received ]
                     │
                     ▼
             Verify with Store
                     │
         ┌───────────┴───────────┐
         ▼                       ▼
      Invalid                 Verified
  (Return Error)                 │
                                 ▼
                    Check Replay / Existing TX
                                 │
         ┌───────────────────────┴───────────────────────┐
         ▼                                               ▼
  Already Processed                               New Transaction
  (Idempotent 200 OK)                                    │
                                                         ▼
                                            Check Account Binding
                                                         │
                                 ┌───────────────────────┴───────────────────────┐
                                 ▼                                               ▼
                           Bound to Other                                  Match / Unbound
                           (409 Conflict)                                        │
                                                                                 ▼
                                                                     Check Active Subscription
                                                                                 │
                                                         ┌───────────────────────┴───────────────────────┐
                                                         ▼                                               ▼
                                                    New Purchase                                      Renewal
                                            (Create UserSubscription)                         (Extend EndTime)
                                            (Upgrade User Group)                              (Reset Used Quota)
                                                         │                                               │
                                                         └───────────────────────┬───────────────────────┘
                                                                                 │
                                                                                 ▼
                                                                     Commit Transaction & TopUp
```

---

## Idempotency, Replay Protection & Concurrency

1. **Replay Invariant**:
   - Every transaction is recorded in [`StoreTransaction`](file:///Users/noppanan/new-api/model/store_billing.go#L95) with a database unique index `UNIQUE(platform, store_transaction_id)`.
   - If the same transaction is submitted 20 times, the first execution processes the grant, and the subsequent 19 executions identify the existing record and return `already_processed: true` with 0 duplicate quota and 0 extra time.
2. **Concurrency Invariant**:
   - Multi-goroutine concurrent execution is synchronized via:
     - User row lock: `lockForUpdate(tx).Select("id").Where("id = ?", userId)`
     - Store binding row lock: `lockForUpdate(tx).Where(...)`
     - Database unique constraint on `StoreTransaction.StoreTransactionId`
   - Verified by test `TestProcessStorePurchase_Concurrency`: 10 parallel goroutines competing in the same millisecond yield exactly 1 `UserSubscription` and 1 `StoreTransaction`.

---

## Renewal Handling

Store renewals occur asynchronously without requiring the user to have LumenFlow open:

1. **Event**:
   - App Store Server Notifications V2 delivers `DID_RENEW` with new `signedTransactionInfo`.
   - Google RTDN delivers `SUBSCRIPTION_RENEWED` (Type 2).
2. **Settlement**:
   - Backend detects matching `store_original_id` in `StoreSubscriptionBinding`.
   - The active `UserSubscription` is identified:
     - `sub.EndTime` is extended to the renewed expiration date.
     - `sub.AmountUsed` is reset to 0 for the new billing cycle.
     - A new `StoreTransaction` record is inserted with the renewal's transaction ID.
3. **No Double Extension**:
   - Replaying the renewal transaction does not extend expiry twice.

---

## Cancellation vs Expiration vs Revocation

| Store Event | Semantics | Backend Action | User Group | Entitlement Active |
| :--- | :--- | :--- | :--- | :--- |
| **Auto-Renew Cancelled** | User cancelled recurring billing in App Store / Play Store. | `StoreSubscriptionBinding.Status = 'cancelled'`. `UserSubscription` remains unchanged. | Unchanged (`pro`) | **Yes** (until `end_time`) |
| **Period Expired** | Current billing period ended without renewal. | Background daemon marks `UserSubscription.Status = 'expired'`. | Downgrades to `PrevUserGroup` | **No** (reverts to Free Tier) |
| **Revoked / Refunded** | Apple/Google support issued a refund or revoked access. | [`ProcessStoreRevocationTx`](file:///Users/noppanan/new-api/model/store_billing.go#L442): sets `UserSubscription.Status = 'cancelled'`, sets `end_time = now`. | Immediately downgrades to `PrevUserGroup` | **No** (immediate termination) |

---

## Store Webhooks & Notifications

Endpoints mounted in [`router/api-router.go`](file:///Users/noppanan/new-api/router/api-router.go#L231):

1. **Apple Webhook**:
   - Route: `POST /api/subscription/apple/webhook`
   - Payload: `{ "signedPayload": "<JWS>" }`
   - Handled in [`controller.AppleSubscriptionWebhook`](file:///Users/noppanan/new-api/controller/subscription_store.go#L182)
   - Decodes JWS, verifies signature, processes `SUBSCRIBED`, `DID_RENEW`, `REFUND`, `REVOKE`.
2. **Google Webhook**:
   - Route: `POST /api/subscription/google/webhook`
   - Payload: Google Cloud Pub/Sub push notification containing base64 `DeveloperNotification`.
   - Handled in [`controller.GoogleSubscriptionWebhook`](file:///Users/noppanan/new-api/controller/subscription_store.go#L234)
   - Evaluates notification types (Type 2 = Renewed, Type 4 = Purchased, Type 12 = Revoked).

---

## Restore Purchases & App Reinstall Contract

1. **App Reinstall**:
   - Reinstalling LumenFlow does NOT require "Restore Purchases".
   - LumenFlow simply logs in and calls `GET /api/subscription/self`.
   - Backend returns active subscriptions directly from the database.
2. **Restore Purchases Flow**:
   - If a user changes devices or re-syncs local store purchases:
     - LumenFlow queries StoreKit / Google Play for active transactions.
     - For each transaction, LumenFlow submits proof to `POST /api/subscription/{apple|google}/verify`.
     - Backend verifies, ensures account binding, and returns status.
     - LumenFlow then calls `GET /api/subscription/self`.
     - **LumenFlow NEVER grants Pro entitlement based on store proof alone.**

---

## Database Schema Changes

Added three models to `DB.AutoMigrate` in [`model/main.go`](file:///Users/noppanan/new-api/model/main.go#L371):

```sql
-- 1. Store Product Mappings
CREATE TABLE store_product_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform VARCHAR(32) NOT NULL,
    store_product_id VARCHAR(128) NOT NULL,
    store_base_plan_id VARCHAR(128) DEFAULT '',
    internal_plan_id INTEGER NOT NULL,
    environment VARCHAR(32) DEFAULT 'production',
    enabled BOOLEAN DEFAULT true,
    created_at BIGINT,
    updated_at BIGINT
);
CREATE INDEX idx_store_product_lookup ON store_product_mappings(platform, store_product_id, store_base_plan_id);

-- 2. Store Subscription Bindings (Cross-Account Theft Guard)
CREATE TABLE store_subscription_bindings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform VARCHAR(32) NOT NULL,
    store_original_id VARCHAR(255) NOT NULL,
    user_id INTEGER NOT NULL,
    internal_plan_id INTEGER NOT NULL,
    active_user_sub_id INTEGER DEFAULT 0,
    status VARCHAR(32) DEFAULT 'active',
    latest_transaction_id VARCHAR(255) DEFAULT '',
    purchase_time BIGINT DEFAULT 0,
    expires_time BIGINT DEFAULT 0,
    revocation_time BIGINT DEFAULT 0,
    created_at BIGINT,
    updated_at BIGINT
);
CREATE UNIQUE INDEX idx_store_orig_plat ON store_subscription_bindings(platform, store_original_id);

-- 3. Store Transactions (Replay & Idempotency Guard)
CREATE TABLE store_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    platform VARCHAR(32) NOT NULL,
    store_transaction_id VARCHAR(255) NOT NULL,
    store_original_id VARCHAR(255) NOT NULL,
    user_id INTEGER NOT NULL,
    internal_plan_id INTEGER NOT NULL,
    user_subscription_id INTEGER DEFAULT 0,
    subscription_order_id INTEGER DEFAULT 0,
    environment VARCHAR(32) DEFAULT 'production',
    status VARCHAR(32) NOT NULL,
    purchase_time BIGINT DEFAULT 0,
    expires_time BIGINT DEFAULT 0,
    revocation_time BIGINT DEFAULT 0,
    raw_evidence TEXT,
    created_at BIGINT,
    updated_at BIGINT
);
CREATE UNIQUE INDEX idx_store_tx_uniq ON store_transactions(platform, store_transaction_id);
```

---

## Endpoint Contract (For Phase 7G-B)

### 1. Apple Initial / Restore Verification

- **Method**: `POST`
- **Path**: `/api/subscription/apple/verify`
- **Auth**: Bearer Access Token / Session Cookie (`middleware.UserAuth()`)
- **Rate Limit**: Critical Rate Limit (20 req / 20 min)

**Request Body**:
```json
{
  "signed_transaction_info": "<StoreKit2_JWS_String>"
}
```

**Success Response (HTTP 200)**:
```json
{
  "success": true,
  "message": "Store purchase verified",
  "data": {
    "status": "active",
    "platform": "apple",
    "store_transaction_id": "1000000123456789",
    "store_original_id": "1000000123456789",
    "plan_id": 1,
    "plan_title": "Pro Monthly",
    "user_subscription_id": 42,
    "expires_time": 1743811200,
    "is_renewal": false,
    "already_processed": false
  }
}
```

### 2. Google Initial / Restore Verification

- **Method**: `POST`
- **Path**: `/api/subscription/google/verify`
- **Auth**: Bearer Access Token / Session Cookie (`middleware.UserAuth()`)
- **Rate Limit**: Critical Rate Limit

**Request Body**:
```json
{
  "purchase_token": "<Google_Purchase_Token>",
  "product_id": "tora_pro_sub"
}
```

**Success Response (HTTP 200)**:
```json
{
  "success": true,
  "message": "Store purchase verified",
  "data": {
    "status": "active",
    "platform": "google",
    "store_transaction_id": "GPA.3344-5566-7788-99000",
    "store_original_id": "GPA.3344-5566-7788-99000",
    "plan_id": 1,
    "plan_title": "Pro Monthly",
    "user_subscription_id": 43,
    "expires_time": 1743811200,
    "is_renewal": false,
    "already_processed": false
  }
}
```

### 3. Canonical Refresh Step

Following verification or restore success, mobile **must call**:
`GET /api/subscription/self`
to reload canonical account entitlements into `SubscriptionService`.

---

## Error Contract

Standardized JSON error responses:

```json
{
  "success": false,
  "code": "<ERROR_CODE>",
  "message": "<Human_Readable_Description>"
}
```

| HTTP Status | Error Code | Meaning / Resolution |
| :--- | :--- | :--- |
| `401 Unauthorized` | `AUTH_REQUIRED` | Missing or expired auth token. Redirect to login. |
| `400 Bad Request` | `STORE_PROOF_INVALID` | Malformed token or invalid signature. Prompt retry. |
| `400 Bad Request` | `STORE_PRODUCT_UNRECOGNIZED` | Product ID not mapped in server catalog. Contact support. |
| `400 Bad Request` | `STORE_BUNDLE_MISMATCH` | Receipt bundle ID does not match app package. |
| `400 Bad Request` | `STORE_ENVIRONMENT_MISMATCH` | Sandbox proof submitted to production environment. |
| `400 Bad Request` | `STORE_SUBSCRIPTION_EXPIRED` | Store proof indicates subscription has expired. |
| `400 Bad Request` | `STORE_SUBSCRIPTION_REVOKED` | Store proof indicates transaction was refunded or revoked. |
| `409 Conflict` | `STORE_TRANSACTION_ALREADY_BOUND` | Store purchase is already bound to another New-API user account. |
| `200 OK` (soft) | `STORE_VERIFICATION_PENDING` | Purchase pending store settlement (e.g. Ask to Buy). |
| `503 Unavailable` | `STORE_SERVER_UNAVAILABLE` | Store server credentials missing or store API unreachable. |

---

## Configuration & Secrets

Environment variables defined in [`setting/payment_store.go`](file:///Users/noppanan/new-api/setting/payment_store.go):

```bash
# Apple App Store (StoreKit 2 / App Store Server API)
APPLE_BUNDLE_ID="me.huanmeng.lumenflow"      # Or com.saascover.tora
APPLE_KEY_ID="ABC123XYZ4"
APPLE_ISSUER_ID="57f30202-e257-4287-b0ad-99dd00e12345"
APPLE_PRIVATE_KEY="/path/to/SubscriptionKey_ABC123XYZ4.p8"
APPLE_ENVIRONMENT="production"               # "production" or "sandbox"
APPLE_ALLOW_SANDBOX=false                   # true only for staging/dev

# Google Play Developer API (Android Publisher v3)
GOOGLE_PACKAGE_NAME="me.huanmeng.lumenflow"  # Or com.saascover.tora
GOOGLE_SERVICE_ACCOUNT_JSON="/path/to/play-service-account.json"
GOOGLE_ALLOW_TEST_PURCHASE=false            # true only for staging/dev
```

If these credentials are not provided, all verification endpoints return `STORE_SERVER_UNAVAILABLE`. The server **never** falls back to insecure client trust.

---

## Threat Model Assessment

| Threat / Attack Scenario | Classification | Defense / Mechanism |
| :--- | :--- | :--- |
| 1. Forged client request | **BLOCKED** | Cryptographic JWS verification (Apple) / Google Publisher API token check. |
| 2. Client-modified product ID / plan | **BLOCKED** | Server maps store-verified `productId` via `StoreProductMapping`. Client claims ignored. |
| 3. Client-modified price / quota | **BLOCKED** | Price and quota are derived exclusively from server `SubscriptionPlan` database records. |
| 4. Replayed Apple transaction (x20) | **BLOCKED** | `StoreTransaction` unique index on `(platform, store_transaction_id)`. Idempotent return. |
| 5. Replayed Google purchase token | **BLOCKED** | Idempotency guard and `StoreSubscriptionBinding` match. No duplicate quota. |
| 6. Cross-account theft (User B submits User A's proof) | **BLOCKED** | `StoreSubscriptionBinding` unique on `(platform, store_original_id)`. Rejects with 409 Conflict. |
| 7. Sandbox proof sent to production | **BLOCKED** | Explicit `environment == "Production"` enforcement; sandbox rejected unless enabled. |
| 8. Wrong bundle / package ID | **BLOCKED** | Verified JWS `bundleId` checked against configured `APPLE_BUNDLE_ID`. |
| 9. Duplicate store webhook | **BLOCKED** | Monotonic event idempotency and transaction lookup prevent duplicate processing. |
| 10. Out-of-order store webhook | **MITIGATED** | Expiry timestamps are evaluated monotonically; older timestamps cannot overwrite newer. |
| 11. Refund after entitlement grant | **BLOCKED** | Store webhook / revocation transaction immediately cancels subscription and downgrades group. |
| 12. Expired subscription replay | **BLOCKED** | Server verifies `expiresDate > now`; expired purchases rejected. |
| 13. Millisecond-concurrent duplicate verification | **BLOCKED** | Row-level locking on user row + DB unique constraint serialize parallel executions. |
| 14. Restore repeatedly granting quota | **BLOCKED** | Quota reset only on genuine new billing cycle transition (`is_renewal`); replays do not reset. |
| 15. Mobile offline after store success | **MITIGATED** | Mobile holds pending transaction in local StoreKit queue until backend confirms 200 OK. |
| 16. Compromised mobile client claiming Pro | **BLOCKED** | Mobile client has 0 inference authorization; `/v1/chat/completions` checks server DB state. |

---

## Verification & Test Results

### 1. Store Billing Unit & Transaction Tests ([`model/store_billing_test.go`](file:///Users/noppanan/new-api/model/store_billing_test.go))
- `TestProcessStorePurchase_ValidInitialPurchase`: PASS
- `TestProcessStorePurchase_ReplayIdempotency`: PASS (20 consecutive replays)
- `TestProcessStorePurchase_Concurrency`: PASS (10 concurrent goroutines)
- `TestProcessStorePurchase_CrossAccountTheftRejection`: PASS (User B rejected)
- `TestProcessStorePurchase_ExpiredTransactionRejected`: PASS
- `TestProcessStorePurchase_RevokedTransactionRejected`: PASS
- `TestProcessStorePurchase_RenewalExtendsExpiryAndResetsQuota`: PASS
- `TestProcessStoreRevocation_ImmediateDowngrade`: PASS
- `TestProcessStorePurchase_UnmappedProductRejected`: PASS

### 2. Store Verifier Unit Tests ([`service/store_verifier_test.go`](file:///Users/noppanan/new-api/service/store_verifier_test.go))
- `TestDefaultAppleVerifier_UnconfiguredFailsSafely`: PASS
- `TestDefaultAppleVerifier_EmptyTokenFails`: PASS
- `TestDefaultAppleVerifier_MalformedJWSFails`: PASS
- `TestDefaultAppleVerifier_BundleMismatchRejected`: PASS
- `TestDefaultAppleVerifier_SandboxRejectedInProduction`: PASS
- `TestDefaultAppleVerifier_ValidSandboxWhenAllowed`: PASS
- `TestDefaultGoogleVerifier_UnconfiguredFailsSafely`: PASS
- `TestDefaultGoogleVerifier_EmptyParams`: PASS

### 3. Store Controller HTTP Tests ([`controller/subscription_store_test.go`](file:///Users/noppanan/new-api/controller/subscription_store_test.go))
- `TestVerifyAppleSubscription_Unauthorized`: PASS (401)
- `TestVerifyAppleSubscription_MissingPayload`: PASS (400)
- `TestVerifyAppleSubscription_Success`: PASS (200)
- `TestVerifyAppleSubscription_CrossAccountConflict`: PASS (409)
- `TestVerifyGoogleSubscription_Pending`: PASS (200 with code `STORE_VERIFICATION_PENDING`)
- `TestAppleSubscriptionWebhook_RefundRevocation`: PASS (200)
- `TestGoogleSubscriptionWebhook_Revocation`: PASS (200)

### 4. Full Backend Regression
- `go test ./model -v`: **100% PASS** (8.67s)
- `go test ./controller -v`: **100% PASS** (36.48s)
- `go test ./service -v`: **100% PASS** (3.49s)

### 5. Mobile Regression ([`scratch/LumenFlow`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow))
- `flutter test`: **86/86 tests passed** (3.0s)
- `flutter analyze`: **0 issues found** (1.9s)

---

## Product Policy Decisions Required

The following commercial decisions require sign-off before production launch:

1. **Cross-Platform Purchase Coexistence**:
   - *Scenario*: A user with an active Apple subscription logs into an Android device and purchases a Google subscription.
   - *Technical behavior*: Both subscriptions coexist in `UserSubscription`. The user retains the highest group tier and combined quota.
   - *Policy Question*: Should the mobile UI block initiating a purchase if the account already has an active subscription on a different store platform?
2. **Mid-Cycle Plan Changes (Upgrades/Downgrades)**:
   - *Apple Policy*: Apple handles upgrades by pro-rating and immediately billing the new tier while refunding unused time on the previous tier.
   - *Policy Question*: When Apple upgrades mid-cycle, backend immediately shifts `UserSubscription.PlanId` and grants new tier quota. Confirm if this is the desired commercial behavior.
3. **Grace Period & Billing Retry Policy**:
   - When a user's credit card fails during renewal, Apple and Google enter a Grace Period (typically 16-28 days).
   - *Policy Question*: Should users retain Pro model access during Grace Period, or revert immediately to Free Tier until payment succeeds?

---

## Store Configuration Required

To transition from test doubles to live production verification, the following items must be configured in App Store Connect, Google Play Console, and backend environment variables:

1. **Apple Developer Account / App Store Connect**:
   - Generate an App Store Server API Private Key (`.p8`) under *Users and Access -> Integrations -> In-App Purchase*.
   - Obtain Key ID, Issuer ID, and configure `APPLE_KEY_ID`, `APPLE_ISSUER_ID`, `APPLE_PRIVATE_KEY`.
   - Register In-App Purchase products under *Subscriptions* (e.g., `com.saascover.tora.pro.monthly`).
   - Configure Server Notification URL V2 to `https://<backend_domain>/api/subscription/apple/webhook`.
2. **Google Play Console**:
   - Link Google Cloud Project with Google Play Console.
   - Create Service Account with "View financial data, orders, and cancellation survey responses" and "Manage orders and subscriptions" permissions.
   - Download Service Account JSON and configure `GOOGLE_SERVICE_ACCOUNT_JSON` and `GOOGLE_PACKAGE_NAME`.
   - Create Subscription Products and Base Plans in Google Play Console (e.g. `tora_pro_sub` / `pro-monthly`).
   - Configure Google Cloud Pub/Sub Topic and Push Subscription pointing to `https://<backend_domain>/api/subscription/google/webhook`.
3. **Database SKU Mapping Records**:
   - Insert rows into `store_product_mappings` linking each store product to internal `subscription_plans.id`.

---

## Phase 7G-B Contract

With Phase 7G-A complete, Phase 7G-B (Flutter StoreKit & Google Play Billing Implementation) has a frozen server contract:

1. Add `in_app_purchase` package to `scratch/LumenFlow`.
2. Implement store catalog display fetching products via store product IDs configured for each platform.
3. Upon purchase completion:
   - Extract `purchaseDetails.verificationData.serverVerificationData`.
   - If iOS: POST to `/api/subscription/apple/verify` with `{ "signed_transaction_info": serverVerificationData }`.
   - If Android: POST to `/api/subscription/google/verify` with `{ "purchase_token": serverVerificationData, "product_id": purchaseDetails.productID }`.
4. On 200 OK verification response:
   - Call `InAppPurchase.instance.completePurchase(purchaseDetails)`.
   - Call `SubscriptionService().refreshAll(apiClient, authService.currentAccessToken)`.
5. On 409 Conflict (`STORE_TRANSACTION_ALREADY_BOUND`):
   - Display error notice: *"This purchase is already linked to another Tora account."*
   - Do NOT complete purchase or grant entitlement.
6. On pending status (`STORE_VERIFICATION_PENDING`):
   - Display notice: *"Purchase is pending approval from the store."*
7. Implement "Restore Purchases" button:
   - Calls `InAppPurchase.instance.restorePurchases()`.
   - For every restored purchase, invokes the same verification endpoint.
   - Refreshes `/api/subscription/self`.

---

## Completion Status

`FINAL STATUS: STORE CONFIGURATION REQUIRED`
