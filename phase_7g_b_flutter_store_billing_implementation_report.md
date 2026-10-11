# Phase 7G-B — Flutter Native Store Billing Implementation Report

## Executive Summary

Phase 7G-B completes the end-to-end integration of native in-app purchases (Apple StoreKit & Google Play Billing) in the **Tora AI** mobile application (`scratch/LumenFlow`), interfacing securely with the New-API backend (`/Users/noppanan/new-api`, branch `feat/formobile`).

This implementation preserves all foundational invariants:
- **Server-Authoritative Entitlement**: A completed native store transaction is never an entitlement. Entitlement is granted only when verified and settled by the New-API backend.
- **Safe Acknowledgement Order**: Transactions are acknowledged in the store (`completePurchase`) **only after** successful server verification or upon unrecoverable terminal conflict (409). On server outages (503/network), transactions remain unacknowledged, guaranteeing store redelivery.
- **Privacy & Token Hygiene**: Raw purchase tokens and signed transaction JWS are treated as high-value credentials, persisted only in secure hardware-backed storage (iOS Keychain / Android EncryptedSharedPreferences) when pending, and never logged or leaked.
- **Cross-Account & Environment Isolation**: Active purchase sessions snapshot user identity and environment. Receipts cannot cross account boundaries or leak between staging and production environments.

---

## 1. Architectural Components

```text
┌─────────────────────────────────────────────────────────────┐
│                 AccountSubscriptionScreen                   │
│   • Dynamic Store Pricing (ProductDetails.price)            │
│   • Pro Monthly Purchase Card                               │
│   • Restore Purchases Action & Dialog                       │
│   • Pending Offline Verification Banner                     │
│   • App Store / Google Play Subscription Management Link    │
│   • Required Legal Terms & Renewal Disclosures              │
└─────────────────────────────┬───────────────────────────────┘
                              │ Listens to state & initiates
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    NativeBillingService                     │
│   • Extends ChangeNotifier                                  │
│   • Single-flight InAppPurchase stream listener             │
│   • State Machine: idle -> purchasing -> backendPending...  │
│   • Session snapshot guards (_activePurchaseUserId/Env)     │
│   • Safe Acknowledgement Order Controller                   │
└─────────────┬───────────────────────────────┬───────────────┘
              │                               │
              ▼                               ▼
┌───────────────────────────┐   ┌───────────────────────────┐
│       ToraApiClient       │   │   SecureCredentialStore   │
│ • GET /store/products     │   │ • writePendingPurchase    │
│ • POST /apple/verify      │   │ • readPendingPurchase     │
│ • POST /google/verify     │   │ • clearPendingPurchase    │
└─────────────┬─────────────┘   └───────────────────────────┘
              │ TLS + Bearer Auth
              ▼
┌─────────────────────────────────────────────────────────────┐
│                    New-API Backend Server                   │
│ • Apple App Store Server API (JWS verification)             │
│ • Google Play Developer API (Purchase token verification)   │
│ • Transaction ledger & unique constraint deduplication      │
│ • Atomic entitlement update (group: 'pro', quota: 2,000,000)│
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Source Implementation Details

### 2.1 Dependencies & Platform Configuration
- **Package Added**: `in_app_purchase: ^3.2.3` and `in_app_purchase_platform_interface: ^1.4.0` in `scratch/LumenFlow/pubspec.yaml`.
- **iOS Configuration**:
  - Bundle ID: `com.saascover.tora`
  - In-App Purchase Capability enabled.
- **Android Configuration**:
  - Application ID: `com.saascover.tora`
  - Permission: `com.android.vending.BILLING` in `AndroidManifest.xml`.

### 2.2 Data Models (`lib/models/subscription_models.dart`)
- `StoreCatalogProduct`: Represents verified products from `GET /api/subscription/store/products`. Supports both primary and alias JSON keys (`store_product_id`, `product_id`).
- `StorePurchaseResult`: Result payload from backend verification endpoints (`/apple/verify`, `/google/verify`). Encapsulates settlement status (`active`, `already_processed`), original transaction ID, and plan ID.
- `PurchaseProcessState`: State machine enum (`idle`, `loadingProducts`, `ready`, `purchasing`, `storePending`, `backendVerificationPending`, `verified`, `failed`, `cancelled`, `restoring`).

### 2.3 Native Billing Service (`lib/services/native_billing_service.dart`)
- Singleton service with clean dependency injection constructors for testing (`NativeBillingService.custom`).
- Single-listener lifecycle over `InAppPurchase.instance.purchaseStream`.
- Dynamic product querying: Queries `_apiClient.getStoreProducts()` first to obtain permitted product IDs, then passes allowed IDs to `_iap.queryProductDetails()`.
- Offline recovery: `retryPendingVerification()` automatically re-attempts verification on app start or upon user trigger.

### 2.4 User Interface Integration (`lib/screens/account_subscription_screen.dart`)
- **Pro Monthly Card**: Displays localized store price formatted by the store (`ProductDetails.price`), feature highlights (2M units/mo, Managed models, BYOK included), and adaptive subscribe button.
- **Active Subscription View**: If the user has active Pro status, displays period quota usage bar, expiration date, and a "Manage Subscription in App Store / Google Play" button.
- **Restore Purchases**: Standalone button adhering to App Store Guideline 3.1.1.
- **Verification Pending Banner**: Amber alert banner with "Retry Verification Now" button displayed if network was interrupted after payment.
- **Store Terms Disclosures**: Complete legal disclosures regarding recurring billing, auto-renewal, and cancellation policy.

---

## 3. Verification & Test Outcomes

- **Mobile Unit & Integration Tests**: 10 tests in `test/native_billing_service_test.dart` passing 100%.
- **Full Flutter Test Suite**: 96/96 tests passing across `scratch/LumenFlow`.
- **Static Analysis**: `flutter analyze` completed with 0 issues / 0 warnings.
- **Backend Test Suite**: 100% pass across `model`, `service`, `controller` in `/Users/noppanan/new-api`.

---

## 4. Next Phase Readiness

The code is completely verified and functionally ready for real sandbox/production store testing once App Store Connect and Google Play Console credentials are provisioned by the operator.
