# Phase 7G-B — Security & Abuse Audit: Native Mobile Store Billing

## Executive Summary

This security and abuse review evaluates the native mobile store billing architecture implemented for **Tora AI** across the Flutter client (`scratch/LumenFlow`) and the New-API backend (`/Users/noppanan/new-api`, branch `feat/formobile`).

**Core Security Invariant:**
> Client execution of a native in-app purchase (Apple StoreKit / Google Play Billing) is **NEVER** trusted by the system as authorization for subscription entitlement. The mobile client remains untrusted; entitlement is activated strictly upon backend verification against Apple App Store Server API / Google Play Developer API and transactional settlement into New-API database tables.

---

## 1. Threat Modeling & Attack Surface Analysis

| Threat ID | Threat Vector | Risk Level | Mitigation Architecture | Verification Status |
|:---|:---|:---:|:---|:---:|
| **T-01** | Forged / Manipulated Store Receipts | **CRITICAL** | Client-side signatures are ignored. Server independently verifies raw JWS / purchase token with Apple/Google servers using production credentials. | **VERIFIED** |
| **T-02** | Replay Attacks (Submitting same receipt twice) | **HIGH** | Unique constraint on `store_transactions.original_transaction_id` + `transaction_id`. Server returns idempotent response (`already_processed`) without double-crediting. | **VERIFIED** |
| **T-03** | Cross-Account Receipt Theft | **CRITICAL** | Backend binds `original_transaction_id` permanently to first `user_id`. Attempt to verify on another user triggers HTTP 409 `STORE_TRANSACTION_ALREADY_BOUND`. Mobile sanitizes error to prevent user enumeration. | **VERIFIED** |
| **T-04** | Client Memory / State Manipulation (`isPro=true`) | **HIGH** | Client UI and inference clients read entitlement status strictly from authoritative backend responses (`GET /api/subscription/self`). No local boolean flags govern server-side access. | **VERIFIED** |
| **T-05** | Credential / Proof Leakage | **HIGH** | In-app purchase verification data is transmitted exclusively over TLS with Bearer auth; tokens are never logged or stored in plain-text storage (persisted exclusively in iOS Keychain / Android EncryptedSharedPreferences). | **VERIFIED** |
| **T-06** | Cross-Environment Proof Contamination | **MEDIUM** | `NativeBillingService` snapshots `AppEnvironment.currentEnvironment`. Submissions across production/staging boundary are rejected with `ENVIRONMENT_MISMATCH`. | **VERIFIED** |
| **T-07** | Account Switch Race Condition | **HIGH** | Active purchase session snapshots `_activePurchaseUserId`. Stream delivery arriving after user switch is rejected with `ACCOUNT_MISMATCH`. | **VERIFIED** |
| **T-08** | Cold Start Unprompted Receipt Injection | **HIGH** | If `_activePurchaseUserId == null`, transactions cannot be claimed by current user unless stored pending purchase matches user ID, or user explicitly initiates "Restore Purchases". | **VERIFIED** |

---

## 2. In-Depth Defense Verification

### 2.1 Server-Authoritative Trust Boundary

```text
┌───────────────────────┐
│ Apple App Store /     │
│ Google Play Billing   │
└──────────┬────────────┘
           │ Native Purchase Completed
           ▼
┌───────────────────────┐
│ Tora AI Client        │ (Untrusted Zone)
│ - Holds raw proof     │
│ - isPro cannot be set │
└──────────┬────────────┘
           │ POST /api/subscription/{apple|google}/verify
           ▼
┌───────────────────────┐
│ New-API Server        │ (Authoritative Trust Boundary)
│ - Validates JWT/token │
│ - Inquires Store APIs │
│ - DB Transaction Lock │
│ - Ledger Settlement   │
└──────────┬────────────┘
           │ 200 OK + active status
           ▼
┌───────────────────────┐
│ Tora AI Client        │
│ - completePurchase()  │
│ - Refresh Entitlement │
└───────────────────────┘
```

1. **Client Never Grants Entitlement Locally**: The Flutter client contains no code path setting local pro status upon `PurchaseStatus.purchased`. It transitions to `PurchaseProcessState.backendVerificationPending` and calls the backend.
2. **Safe Acknowledgement Order**: `InAppPurchase.completePurchase()` is invoked **strictly after** the backend confirms settlement (`result.isSuccessful`) or on terminal non-recoverable error (`STORE_TRANSACTION_ALREADY_BOUND`). On network or server failure (503/timeout), `completePurchase()` is intentionally **not called**, ensuring platform store retries transaction delivery.

### 2.2 Replay Protection & Idempotency

- In the New-API backend (`model/store_billing.go`):
  ```sql
  CREATE UNIQUE INDEX idx_store_txn_unique ON store_transactions (platform, store_transaction_id);
  CREATE UNIQUE INDEX idx_store_sub_orig_id ON store_subscriptions (platform, original_transaction_id);
  ```
- Any re-submission of an existing transaction ID is detected via database unique constraint violation or query lookup:
  - If identical transaction was already settled, backend returns `already_processed` status with HTTP 200.
  - No new quota or period extension is granted for duplicate submissions.

### 2.3 Cross-Account Isolation & User Anonymity

- **Cross-Account Attack Vector**: User B obtains User A's receipt (e.g. on a shared device) and attempts to submit it to claim Pro status on User B's account.
- **Backend Defense**:
  - `store_subscriptions` records `user_id`. If `sub.UserID != requestingUserID`, backend returns HTTP 409 with error code `STORE_TRANSACTION_ALREADY_BOUND`.
- **Client Defense**:
  - Mobile client sanitizes the error response:
    ```dart
    case 'STORE_TRANSACTION_ALREADY_BOUND':
      _setError(
        'This store subscription is already linked to another Tora AI account.',
        code,
      );
    ```
  - The client **never** prints or renders the other account's `user_id`, username, or email.
- **Race Condition Guard**:
  - When User A clicks "Subscribe", `_activePurchaseUserId` is snapshotted as `User A.id`.
  - If User A logs out and User B logs in before the store callback resolves, `_verifyAndSettlePurchase` compares `_activePurchaseUserId` with `currentUserId`. Because `101 != 102`, verification is aborted with `ACCOUNT_MISMATCH`.
  - Unprompted store deliveries on cold start with no active session or matching pending record return `SESSION_REQUIRED`, directing the user to explicitly tap "Restore Purchases".

### 2.4 Token Secrecy & Storage Hardening

- All mobile state persistence uses `SecureCredentialStore`:
  - **iOS**: Apple Keychain Services with `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`.
  - **Android**: Android Keystore with `EncryptedSharedPreferences` (AES-256-GCM + RSA-OAEP).
- Pending purchases containing unverified receipts are stored in hardware-backed secure storage (`writePendingPurchase`) and purged immediately upon settlement (`clearPendingPurchase`).
- Zero receipt tokens, JWS strings, or session secrets are written to log files, crash reports, or console outputs in release builds.

---

## 3. Findings & Remediations During Audit

During Phase 7G-B implementation and testing, two critical edge cases were discovered and resolved:

1. **Unprompted Receipt Injection Vulnerability**:
   - *Discovery*: If `_activePurchaseUserId` was null (e.g., service re-instantiated), an unprompted purchase callback could have verified under whichever user was currently authenticated.
   - *Remediation*: Added explicit verification that `expectedUserId != null` and matches `currentUserId`, requiring cold-start users to tap "Restore Purchases" to claim unacknowledged store transactions.
2. **503 Server Failure Acknowledgement Leak**:
   - *Discovery*: A 404/500 routing mismatch in early tests caused terminal error fallback that called `completePurchase`.
   - *Remediation*: Standardized route matching to `/api/subscription/apple/verify` and `/api/subscription/google/verify` and verified that 5xx errors or network exceptions strictly bypass `completePurchase()`.

---

## 4. Audit Sign-Off

- **Security Architecture**: **APPROVED**
- **Abuse Prevention**: **APPROVED**
- **Cross-Account Protection**: **APPROVED**
- **Token Hygiene**: **APPROVED**
