# Phase 7G-A-R — Native Store Billing Contract Closure & Production Configuration Readiness

## Executive Summary

Phase 7G-A established the server-side architectural boundary for native store subscriptions (Apple App Store StoreKit 2 and Google Play Billing). Phase 7G-A-R executes the **contract closure pass**, resolving all remaining cryptographic, authenticity, transition chain, and catalog contracts before Flutter client-side StoreKit / Google Play Billing integration begins in Phase 7G-B.

All cryptographic trust chains, webhook authenticity barriers, token replacement chain mappings, and public catalog endpoints are implemented, tested, and verified against both `new-api` Go backend suites and `LumenFlow` Flutter suites (86/86 tests passing, 0 analyzer issues).

---

## 1. Architectural Invariant & Trust Boundary

Entitlement authority resides **exclusively** on the SaaSCover / Tora New-API server:
1. The Flutter mobile client is **untrusted**: it acts solely as a carrier of signed store proofs.
2. The server independently verifies proofs directly against Apple Root CA and Google Play Developer APIs.
3. The server independently maps verified store products to internal subscription plans and tiers.
4. The server atomically updates subscription records, resets quotas, and adjusts user groups inside serializable database transactions.
5. In the absence of production credentials, the server fails safely with `503 Service Unavailable` (`STORE_SERVER_UNAVAILABLE`), preventing unverified or simulated purchases from breaching production data.

```text
  ┌──────────────────────────────────────────────────────────────┐
  │                 Mobile Device (Untrusted)                    │
  │  StoreKit 2 / Google Play Billing                            │
  │  Receives purchase proof / token                             │
  └──────────────────────────────┬───────────────────────────────┘
                                 │ POST /api/subscription/{apple,google}/verify
                                 ▼
  ┌──────────────────────────────────────────────────────────────┐
  │               New-API Backend (Authoritative)                │
  │                                                              │
  │ 1. Cryptographic Trust Check                                 │
  │    - Apple: x5c chain verified to embedded Apple Root CA     │
  │    - Google: Authorized call to AndroidPublisher v3          │
  │                                                              │
  │ 2. Environment & Application Identity Check                  │
  │    - Bundle ID / Package Name verification                   │
  │    - Sandbox / Test purchase isolation                       │
  │                                                              │
  │ 3. Row-Locked Atomic Transaction (ProcessStorePurchaseTx)    │
  │    - Replay protection (Idempotent 200 OK)                   │
  │    - Cross-account binding enforcement (409 Conflict)        │
  │    - Token transition chain preservation (LinkedToken)       │
  │    - Active subscription & quota update                      │
  └──────────────────────────────────────────────────────────────┘
```

---

## 2. Cryptographic Trust Closure

### 2.1 Apple StoreKit 2 JWS Trust Chain
- **Vulnerability Closed**: In the preliminary pass, signature decoding verified ECDSA public keys extracted from JWS headers without verifying that the certificate chained to an authoritative Apple Root CA. An attacker could craft a self-signed certificate and forge valid claims.
- **Implementation**:
  - Embedded official Apple Root CA certificates into `service/store_verifier.go`:
    - `Apple Root CA - G3` (ECC P-384 root: primary trust anchor for StoreKit 2 and App Store Server Notifications V2)
    - `Apple Root CA - G2` (RSA 4096-bit root)
    - `Apple Root CA` (RSA 2048-bit root)
  - `verifyAppleJWSChain` decodes `x5c[0]` (leaf certificate) and `x5c[1:]` (intermediate certificates) and enforces standard X.509 path validation (`leafCert.Verify`) anchored to the Apple Root CA pool.
  - ES256 JWS cryptographic signature is verified against the validated leaf certificate public key with claims validation disabled at the parser layer (`jwt.WithoutClaimsValidation()`) to allow domain-level claim checks.
  - Any token missing `x5c`, possessing self-signed/untrusted certificates, or with tampered payloads/signatures is rejected with `ErrStoreVerificationFailed`.

### 2.2 Apple App Store Server Notifications V2 Authenticity
- Outer JWS notifications (`signedPayload`) delivered to `POST /api/subscription/apple/webhook` are cryptographically verified via `AppleVerifier.VerifyNotification`:
  - Outer header certificate chain validated against Apple Root CA.
  - Outer ES256 signature verified against validated leaf certificate.
  - Notification `environment` and `bundleId` checked against server configuration.
  - Inner `signedTransactionInfo` JWS independently validated against Apple Root CA via `AppleVerifier.VerifyTransaction`.
  - Notifications with untrusted signatures return `400 Bad Request`.

### 2.3 Google RTDN Authenticity & Defense-in-Depth
- `POST /api/subscription/google/webhook` accepts Cloud Pub/Sub push messages.
- **Authentication**:
  - Validates `GOOGLE_PUBSUB_VERIFICATION_TOKEN` via URL query param (`?token=...`) or `X-Goog-PubSub-Verification-Token` header.
  - Unauthenticated push attempts return `401 Unauthorized`.
  - Package name checked against `GOOGLE_PACKAGE_NAME`.
- **Defense-in-Depth Against Revocation Spoofing**:
  - The webhook handler **never** relies solely on the RTDN event payload to mutate database state.
  - For every notification (including `notificationType == 12` REVOKED), the server authoritatively queries Google Play Developer API (`GlobalGoogleVerifier.VerifySubscription`) before executing any revocation or renewal transaction.
  - If Google Play API fails to verify the subscription, database mutation is rejected and skipped.

---

## 3. Google Subscription Chain Identity (Token Transitions)

Google Play Subscriptions v2 issues a **new `purchaseToken`** during plan changes (upgrades, downgrades, cross-grades) and certain renewal cycles, providing the prior token in `linkedPurchaseToken`.

- **Cross-Account Attack Vector**: Without transition awareness, an attacker could attempt to bind a new `purchaseToken` derived from another user's base subscription, or an upgrade could be falsely flagged as `409 Conflict: STORE_TRANSACTION_ALREADY_BOUND`.
- **Implementation in `ProcessStorePurchaseTx`**:
  - If `verified.LinkedPurchaseToken != ""` and no binding exists for the new token:
    1. Look up existing `StoreSubscriptionBinding` using `linkedPurchaseToken`.
    2. If found and belongs to `userId`:
       - Seamlessly update the binding's `store_original_id` to the new token.
       - Proceed with entitlement grant and quota adjustment.
    3. If found and belongs to `different_user_id`:
       - Reject immediately with `ErrStoreTransactionBoundToAnotherAccount` (`409 Conflict`).
- **Implementation in `GoogleSubscriptionWebhook`**:
  - When finding the existing subscriber for renewal or upgrade events, searches by `sub.PurchaseToken`, and if absent, searches by `verified.LinkedPurchaseToken`.

---

## 4. Product Catalog API Specification

To prevent hardcoding Store SKUs in the Flutter mobile application, the backend exposes the active product catalog.

### Endpoint
`GET /api/subscription/store/products`

### Authentication
User authenticated (`Bearer <jwt_token>`).

### Query Parameters
| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `platform` | string | Optional | Filter by platform: `apple` or `google`. If omitted, returns all. |
| `env` | string | Optional | Only allowed in sandbox/test environments (`sandbox` / `production`). |

### Response Schema (`200 OK`)
```json
{
  "success": true,
  "data": [
    {
      "platform": "apple",
      "product_id": "com.saascover.tora.pro.monthly",
      "plan_id": 1,
      "title": "Pro Monthly",
      "duration_unit": "month",
      "duration_value": 1,
      "total_amount": 500000
    },
    {
      "platform": "google",
      "product_id": "tora_pro_sub",
      "base_plan_id": "pro-monthly",
      "plan_id": 1,
      "title": "Pro Monthly",
      "duration_unit": "month",
      "duration_value": 1,
      "total_amount": 500000
    }
  ]
}
```

### Security Properties
- **Zero Secrets**: Contains no API keys, private keys, issuer IDs, service accounts, or webhook tokens.
- **Environment Filtering**: Server automatically filters products by its active runtime environment (`production` vs `sandbox`).
- **Status Gating**: Only returns mappings where both `mapping.enabled == true` and `subscription_plan.enabled == true`.

---

## 5. Frozen Client/Server Billing API Contract for Phase 7G-B

The following contracts are frozen and ready for Phase 7G-B mobile implementation:

### 5.1 Verification Endpoints

#### Apple StoreKit 2 Verification
- **Path**: `POST /api/subscription/apple/verify`
- **Headers**: `Authorization: Bearer <user_token>`, `Content-Type: application/json`
- **Request Body**:
  ```json
  {
    "signed_transaction_info": "<StoreKit2_JWS_String>"
  }
  ```

#### Google Play Billing Verification
- **Path**: `POST /api/subscription/google/verify`
- **Headers**: `Authorization: Bearer <user_token>`, `Content-Type: application/json`
- **Request Body**:
  ```json
  {
    "product_id": "tora_pro_sub",
    "purchase_token": "<Google_Purchase_Token>"
  }
  ```

### 5.2 Server Error Response Codes
All store billing endpoints return structured error codes mapped to HTTP status codes:

| Error Code | HTTP Status | Meaning | Mobile Action |
| :--- | :--- | :--- | :--- |
| `STORE_VERIFICATION_PENDING` | 200 OK (`success: false`) | Google Play purchase pending cash/bank settlement | Display pending banner; re-verify on app resume |
| `STORE_PROOF_INVALID` | 400 Bad Request | JWS signature invalid, untrusted certificate, or corrupt token | Alert user; do not retry corrupted proof |
| `STORE_PRODUCT_UNRECOGNIZED` | 400 Bad Request | Product ID not configured in database mapping | Report error; contact support |
| `STORE_BUNDLE_MISMATCH` | 400 Bad Request | Bundle ID / Package name does not match app | Configuration error |
| `STORE_ENVIRONMENT_MISMATCH` | 400 Bad Request | Sandbox proof submitted to production environment | Reject sandbox receipt |
| `STORE_SUBSCRIPTION_EXPIRED` | 400 Bad Request | Store proof expiration date is in the past | Prompt user to renew in app store |
| `STORE_SUBSCRIPTION_REVOKED` | 400 Bad Request | Store proof has been revoked or refunded | Cancel local entitlement |
| `STORE_TRANSACTION_ALREADY_BOUND`| 409 Conflict | Subscription belongs to another Tora account | Display "Subscription already linked to another account" dialog |
| `STORE_SERVER_UNAVAILABLE` | 503 Service Unavailable | Server store verification credentials unconfigured | Display temporary outage message; retry later |

### 5.3 Idempotency & Restore Semantics
- **Replay / Duplicate Submission**: Resubmitting an already processed transaction ID returns `200 OK` with `already_processed: true` without double-crediting quota or modifying expiration.
- **Restore Purchases**: Mobile iterates through StoreKit 2 `Transaction.currentEntitlements` or Google Play `queryPurchasesAsync` and sends active proofs to the verify endpoint. If the user owns the binding, the current subscription state is returned and local UI unlocks.

---

## 6. Zero Hardcoded SKU Audit

An automated scan of the entire codebase was conducted across `model/`, `service/`, `controller/`, `setting/`, and `router/`:
- **Result**: Exactly **0** hardcoded SKU strings exist in production code.
- All product identifiers are managed dynamically via the `store_product_mappings` table.
- Test suites utilize localized mock fixtures in test files only.

---

## 7. Production Configuration Readiness Inventory

The following configuration variables are required for live production operation. Currently, production secrets are not set in the local development environment:

| Platform | Environment Variable | Current Value / Status | Description | Action Required |
| :--- | :--- | :--- | :--- | :--- |
| **Apple** | `APPLE_BUNDLE_ID` | `""` (**NOT CONFIGURED**) | App bundle ID (e.g. `me.huanmeng.lumenflow` or `com.saascover.tora`) | **OPERATOR ACTION REQUIRED** |
| **Apple** | `APPLE_KEY_ID` | `""` (**NOT CONFIGURED**) | App Store Connect API Key ID (10 chars) | **OPERATOR ACTION REQUIRED** |
| **Apple** | `APPLE_ISSUER_ID` | `""` (**NOT CONFIGURED**) | App Store Connect Issuer UUID | **OPERATOR ACTION REQUIRED** |
| **Apple** | `APPLE_PRIVATE_KEY` | `""` (**NOT CONFIGURED**) | App Store Connect Private Key (`.p8` content) | **OPERATOR ACTION REQUIRED** |
| **Apple** | `APPLE_ENVIRONMENT` | `"production"` | Active Apple environment (`production` or `sandbox`) | Configured |
| **Apple** | `APPLE_ALLOW_SANDBOX` | `false` | Allow Sandbox receipts in dev | Optional |
| **Google**| `GOOGLE_PACKAGE_NAME` | `""` (**NOT CONFIGURED**) | Android application package name (e.g. `me.huanmeng.lumenflow`) | **OPERATOR ACTION REQUIRED** |
| **Google**| `GOOGLE_SERVICE_ACCOUNT_JSON` | `""` (**NOT CONFIGURED**) | Google Cloud Service Account JSON for AndroidPublisher v3 | **OPERATOR ACTION REQUIRED** |
| **Google**| `GOOGLE_PUBSUB_VERIFICATION_TOKEN` | `""` (**NOT CONFIGURED**) | Secret token for Google RTDN Cloud Pub/Sub push webhook | **OPERATOR ACTION REQUIRED** |
| **Google**| `GOOGLE_ALLOW_TEST_PURCHASE` | `false` | Allow Google Play license test purchases in dev | Optional |

> [!WARNING]
> Both Apple and Google store verification fail safely with `503 STORE_SERVER_UNAVAILABLE` when their respective production credentials are unset. This ensures no unverified transactions can bypass authentication or entitlement settlement.

---

## 8. Verification Results

### Backend (`new-api` Go Test Suites)
- `go test ./service -v`: **12/12 PASS** (All cryptographic tests, X.509 chain verification, tamper detection, notification verification, and fail-safe checks passed).
- `go test ./model -run "Test.*Store.*" -v`: **8/8 PASS** (All replay protection, atomic row-locking, cross-account theft rejection, and renewal tests passed).
- `go test ./controller -run "Test.*Store.*|TestApple.*|TestGoogle.*|TestGetStore.*" -v`: **10/10 PASS** (All controller endpoints, catalog queries, Apple webhook verification, and Google RTDN defense-in-depth tests passed).
- Full suite `go test ./model ./service ./controller`: **ALL PASSED (0 failures)**.

### Mobile (`LumenFlow` Flutter Test Suites)
- `flutter test`: **86/86 PASS** (100% test passing rate, zero regression across auth, inference, BYOK, and entitlement suites).
- `flutter analyze`: **No issues found!** (0 errors, 0 warnings, 0 lints).

---

## FINAL STATUS: STORE CONFIGURATION REQUIRED

The Phase 7G-A-R contract closure pass is complete. The backend trust boundary is cryptographically sealed, webhook authenticity is enforced, token transition chains are preserved, and public catalog endpoints are exposed. Production secrets remain unconfigured pending live operator provisioning. Phase 7G-B mobile integration may proceed.
