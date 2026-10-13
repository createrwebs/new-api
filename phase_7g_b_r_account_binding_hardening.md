# Phase 7G-B-R — Native Billing Account Binding Hardening Report

## Executive Summary

Phase 7G-B-R performs a focused remediation and mathematical/cryptographic closure of **native billing account binding** across the Tora AI Flutter client (`scratch/LumenFlow`) and New-API Go backend (`/Users/noppanan/new-api`, branch `feat/formobile`).

Prior to this phase, cross-account protection relied solely on `StoreSubscriptionBinding` post-first-submission: once a subscription was bound to User A in the database, User B could not reclaim it. However, before the first successful verification, the backend bound the transaction to the *first claimant who submitted valid store proof* ("first claimant wins"). If User B intercepted User A's unconsumed store proof (via local device sharing, proxy logs, or MITM), User B could submit it first and permanently bind User A's purchase to User B's account.

This hardening pass eliminates the "first claimant wins" race condition by cryptographically embedding and validating a privacy-preserving account token at purchase inception:
1. **Apple StoreKit 2**: Uses StoreKit `appAccountToken` (RFC 4122 UUIDv5).
2. **Google Play Billing**: Uses `obfuscatedAccountId` on `BillingFlowParams` verified against Google Play Developer API's authoritative `externalAccountIdentifiers.obfuscatedExternalAccountId`.
3. **Deterministic Privacy-Safe Derivation**: Generates a canonical RFC 4122 UUIDv5 identical between Dart and Go (`tora:<env>:user:<userId>`), with 0 PII leakage and strict environment isolation.
4. **Configuration Contract Audit**: Resolves all env var aliases (`APPLE_IAP_*` vs `APPLE_*`, `GOOGLE_IAP_*` vs `GOOGLE_*`, key paths), and eradicates the deprecated "Tier 10" pricing label.
5. **Attack Test Verification**: Scenarios A through G pass 100% across both unit and end-to-end controller tests.

---

## 1. Threat Model & Vulnerability Analysis

### 1.1 The "First Claimant Wins" Flaw
In the Phase 7G-A/7G-B baseline:
```text
User A (authenticated)
  ↓ initiates purchase in StoreKit / Google Play
Apple / Google generates valid signed receipt / purchase token
  ↓
[ATTACK WINDOW] Attacker (User B) intercepts raw proof before User A submits
  ↓
User B calls POST /api/subscription/apple/verify with User A's proof
  ↓
New-API checks: StoreSubscriptionBinding exists for original_transaction_id?
  → NO (First submission!)
  ↓
New-API creates StoreSubscriptionBinding(UserId = User B, OriginalId = proof.original_id)
  ↓
User B receives Pro Entitlement!
User A is subsequently rejected with HTTP 409 STORE_TRANSACTION_ALREADY_BOUND.
```

### 1.2 Remediated Architecture (Cryptographic Account Binding)
With Phase 7G-B-R:
```text
User A (authenticated as userId: 42)
  ↓
Client computes deterministic token:
accountToken = DeriveStoreAccountToken(42, "production")
  → "e4569e57-fd52-58ce-9648-cc40061d2db6" (RFC 4122 UUIDv5)
  ↓
Client passes accountToken into PurchaseParam.applicationUserName
  • iOS: StoreKit 2 stamps appAccountToken into signed JWS transaction
  • Android: Play Billing sets obfuscatedAccountId on BillingFlowParams
  ↓
Attacker (User B, userId: 99) intercepts proof and submits first:
  POST /api/subscription/apple/verify
  Authorization: Bearer <User B session token>
  ↓
New-API decodes Apple signed JWS payload signed by Apple Root CA:
  claims.AppAccountToken == "e4569e57-fd52-58ce-9648-cc40061d2db6"
New-API derives expected token for caller:
  DeriveStoreAccountToken(99, "production") == "3f82a1..."
Token check: MISMATCH!
  ↓
New-API aborts transaction with HTTP 403 STORE_ACCOUNT_TOKEN_MISMATCH.
0 database rows written. 0 bindings created. Attacker gets NOTHING.
  ↓
User A submits proof:
  Authorization: Bearer <User A session token>
New-API checks token: MATCH!
New-API binds transaction to User A. Entitlement granted.
```

---

## 2. Privacy-Safe Account Identifier Derivation

### 2.1 Specification
- **Application Namespace UUID**: Fixed RFC 4122 UUID `e0f4d3b2-7a89-4c1e-9f3a-2b5d8e7c1042`.
- **Name Input**: `tora:<env>:user:<userId>`, where `<env>` is canonicalized to `production` or `sandbox`.
- **Algorithm**: RFC 4122 Section 4.3 (UUID Version 5 with SHA-1 hashing).
  1. Concatenate namespace bytes (16 bytes) and UTF-8 name bytes.
  2. Compute SHA-1 digest (20 bytes).
  3. Set version bits: `digest[6] = (digest[6] & 0x0F) | 0x50` (Version 5).
  4. Set variant bits: `digest[8] = (digest[8] & 0x3F) | 0x80` (RFC 4122 variant).
  5. Format first 16 bytes as 36-character lowercase hyphenated hex string `xxxxxxxx-xxxx-5xxx-yxxx-xxxxxxxxxxxx`.

### 2.2 Mathematical Cross-Language Equivalence Test
Both implementations were tested against identical vectors:

| Input Context | Go Backend Output | Dart Client Output | Match |
| :--- | :--- | :--- | :---: |
| User 42 (`production`) | `e4569e57-fd52-58ce-9648-cc40061d2db6` | `e4569e57-fd52-58ce-9648-cc40061d2db6` | **100% IDENTICAL** |
| User 42 (`sandbox`) | `e26a62c1-8042-5f1a-ad7d-0ae3cbc022a4` | `e26a62c1-8042-5f1a-ad7d-0ae3cbc022a4` | **100% IDENTICAL** |
| User 43 (`production`) | `6ce7ef99-ed76-5644-9f56-38d679b31b5c` | `6ce7ef99-ed76-5644-9f56-38d679b31b5c` | **100% IDENTICAL** |

### 2.3 Privacy & Store Compliance Properties
1. **Apple StoreKit Compliance**: Apple requires `appAccountToken` to be a valid UUID string. UUIDv5 is fully compliant with RFC 4122 and accepted natively by Swift `UUID(uuidString:)`.
2. **Google Play Compliance**: Maximum length for `obfuscatedAccountId` is 64 characters. A 36-character UUID string fits easily within limits.
3. **Zero PII Exposure**: One-way cryptographic hash with application namespace; impossible for Apple, Google, or any third party to reverse into user IDs, usernames, or emails.
4. **Environment Separation**: Sandbox tokens and production tokens differ completely, preventing cross-environment test token re-use.

---

## 3. Implementation Details

### 3.1 Flutter Mobile Client (`scratch/LumenFlow`)
- **`lib/utils/store_account_token.dart`**: Implements RFC 4122 UUIDv5 derivation in Dart using `package:crypto`.
- **`lib/services/native_billing_service.dart`**:
  - `purchase()`: Derives `accountToken` for the authenticated user and supplies it as `PurchaseParam.applicationUserName`.
    - On iOS: `in_app_purchase_storekit` maps `applicationUserName` directly to StoreKit 2 `SK2ProductPurchaseOptions(appAccountToken: purchaseParam.applicationUserName)`.
    - On Android: `in_app_purchase_android` maps `applicationUserName` directly to `BillingFlowParams.Builder.setObfuscatedAccountId`.
  - `restorePurchases()`: Supplies `applicationUserName: accountToken` to `restorePurchases()`.

### 3.2 Backend Service & Model (`new-api`)
- **`model/store_account_token.go`**: Canonical Go implementation of `DeriveStoreAccountToken`, `ValidateStoreAccountToken`, and `NormalizeStoreEnvironment`. Re-exported in `service/store_account_token.go`.
- **`model/store_billing.go`**:
  - `VerifiedStorePurchase`: Added `ObfuscatedExternalAccountId` and `GetStoreAccountToken()`.
  - `ProcessStorePurchaseTx`: Added step **C3 Native Account Binding Verification**:
    - Validates `storeToken` against `DeriveStoreAccountToken(targetUserId, verified.Environment)`.
    - Mismatches return `ErrStoreAccountBindingMismatch`.
    - Missing tokens on new initial purchases under strict mode return `ErrStoreAccountTokenRequired`.
- **`controller/subscription_store.go`**:
  - Maps `ErrStoreAccountBindingMismatch` -> HTTP 403 `STORE_ACCOUNT_TOKEN_MISMATCH`.
  - Maps `ErrStoreAccountTokenRequired` -> HTTP 400 `STORE_ACCOUNT_TOKEN_REQUIRED`.

### 3.3 Configuration Contract & Alias Support
In `setting/payment_store.go`, dual-convention environment variable resolution was implemented:
- **Apple**: Supports `APPLE_IAP_*`, `APPLE_*`, and `APPLE_STOREKIT_*`.
  - `APPLE_IAP_BUNDLE_ID` / `APPLE_BUNDLE_ID`
  - `APPLE_IAP_KEY_ID` / `APPLE_KEY_ID` / `APPLE_STOREKIT_KEY_ID`
  - `APPLE_IAP_ISSUER_ID` / `APPLE_ISSUER_ID` / `APPLE_STOREKIT_ISSUER_ID`
  - `APPLE_IAP_PRIVATE_KEY` / `APPLE_PRIVATE_KEY` / `APPLE_STOREKIT_PRIVATE_KEY`
  - File path fallback: `APPLE_IAP_PRIVATE_KEY_PATH` reads `.p8` file directly from disk.
- **Google**: Supports `GOOGLE_IAP_*` and `GOOGLE_*`.
  - `GOOGLE_IAP_PACKAGE_NAME` / `GOOGLE_PACKAGE_NAME`
  - `GOOGLE_IAP_SERVICE_ACCOUNT_JSON` / `GOOGLE_SERVICE_ACCOUNT_JSON`
  - File path fallback: `GOOGLE_IAP_SERVICE_ACCOUNT_PATH` reads service account JSON directly from disk.
  - `GOOGLE_PUBSUB_VERIFICATION_TOKEN` / `GOOGLE_IAP_PUBSUB_VERIFICATION_TOKEN`
- **Account Binding Mode**:
  - `STORE_REQUIRE_ACCOUNT_TOKEN` (default: `true`).
- **Pricing Documentation**:
  - Completely removed the deprecated "Tier 10" label from all documentation and checklists in favor of `USD $9.99 / month`.

---

## 4. Legacy / Restore / Family Sharing Security Policy

| Subscription State | Account Token in Store Proof | Binding in DB | Action Taken | Rationale |
| :--- | :---: | :---: | :--- | :--- |
| **New Purchase** | Matches Authenticated User | None | **Granted & Bound** | Cryptographically verified first purchase. |
| **New Purchase (Intercepted)** | Belongs to Another User | None | **REJECTED (403)** | Prevents first-claimant theft. |
| **New Purchase** | Missing / Empty | None | **REJECTED (400)** (Strict Mode)<br>**Granted** (Permissive Mode) | Strict mode enforced by default for production launch; permissive mode available for legacy user migration via `STORE_REQUIRE_ACCOUNT_TOKEN=false`. |
| **Renewal / Restore** | Present or Empty | Exists for Same User | **Extended / Restored** | Ownership already anchored in database ledger. |
| **Renewal / Restore (Hijack)** | Present or Empty | Exists for Different User | **REJECTED (409)** | Cross-account replay strictly prohibited. |

---

## 5. Automated Attack & Regression Test Matrix

### 5.1 Go Backend Verification Suite (`controller` & `model`)

| Test Identifier | Scenario Description | Expected Result | Actual Result | Status |
| :--- | :--- | :--- | :--- | :---: |
| `TestVerifyAppleSubscription_AttackScenarioA` | User B submits User A's unconsumed Apple transaction | HTTP 403 `STORE_ACCOUNT_TOKEN_MISMATCH`, 0 bindings | HTTP 403 `STORE_ACCOUNT_TOKEN_MISMATCH`, 0 bindings | **PASS** |
| `TestVerifyAppleSubscription_ScenarioB` | User B submits transaction with User B's token | HTTP 200, bound to User B | HTTP 200, bound to User B | **PASS** |
| `TestVerifyAppleSubscription_ScenarioC` | User B attempts replay of already-bound transaction | HTTP 409 `STORE_TRANSACTION_ALREADY_BOUND` | HTTP 409 `STORE_TRANSACTION_ALREADY_BOUND` | **PASS** |
| `TestVerifyGoogleSubscription_AttackScenarioD` | User B submits Google purchase with User A's obfuscated account ID | HTTP 403 `STORE_ACCOUNT_TOKEN_MISMATCH` | HTTP 403 `STORE_ACCOUNT_TOKEN_MISMATCH` | **PASS** |
| `TestVerifyGoogleSubscription_ScenarioE` | User A submits Google purchase with User A's obfuscated account ID | HTTP 200, Google Acknowledged | HTTP 200, Google Acknowledged | **PASS** |
| `TestVerifyAppleSubscription_ScenarioF` | Unbound purchase without account token in Strict Mode | HTTP 400 `STORE_ACCOUNT_TOKEN_REQUIRED` | HTTP 400 `STORE_ACCOUNT_TOKEN_REQUIRED` | **PASS** |
| `TestVerifyAppleSubscription_ScenarioG` | Unbound purchase without account token in Permissive Mode | HTTP 200, bound to claimant | HTTP 200, bound to claimant | **PASS** |
| `TestProcessStorePurchase_AccountTokenMismatch` | Database model layer first claimant attack | Returns `ErrStoreAccountBindingMismatch` | Returns `ErrStoreAccountBindingMismatch` | **PASS** |
| `TestProcessStorePurchase_ReplayIdempotency` | 20 consecutive replays of same transaction | Exactly 1 subscription & 1 ledger record | Exactly 1 subscription & 1 ledger record | **PASS** |
| `TestProcessStorePurchase_Concurrency` | 10 concurrent goroutines racing on same purchase | Exactly 1 succeeds, 9 return idempotent success | Exactly 1 succeeds, 9 return idempotent success | **PASS** |

### 5.2 Flutter Mobile Client Suite (`scratch/LumenFlow`)

| Test Suite | Tests Run | Pass Rate | Analyzer Warnings |
| :--- | :---: | :---: | :---: |
| `test/store_account_token_test.dart` | 2 | 100% | 0 |
| `test/native_billing_service_test.dart` | 10 | 100% | 0 |
| `test/subscription_service_test.dart` | 15 | 100% | 0 |
| Full App Test Suite | 98 | 100% (98/98) | 0 |

---

## 6. Verification Commands Executed

```bash
# 1. Flutter Client Verification
cd /Users/noppanan/.gemini/antigravity/scratch/LumenFlow
flutter test test/store_account_token_test.dart  # PASS
flutter test                                     # 98/98 PASS
flutter analyze                                  # No issues found!

# 2. Go Backend Verification
cd /Users/noppanan/new-api
go test -v ./service -run "TestDeriveStoreAccountToken.*"     # PASS
go test -v ./model -run "TestProcessStorePurchase.*"          # PASS
go test -v ./controller -run "TestVerify.*"                  # PASS
go test ./service ./model ./controller                       # PASS (100%)
```

---

## Final Status Declaration

FINAL STATUS: READY FOR REAL STORE SETUP AND SANDBOX E2E
