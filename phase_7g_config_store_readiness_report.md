# Phase 7G-CONFIG — Native Store Configuration & Sandbox Readiness Report

## Executive Summary

Phase 7G-CONFIG conducts an operational and configuration audit of the native store billing infrastructure across the **LumenFlow** Flutter mobile codebase (`scratch/LumenFlow`) and the **New-API** backend (`/Users/noppanan/new-api`, branch `feat/formobile`).

While the backend billing architecture, cryptographic trust boundary, webhook defense-in-depth, replay protection, and catalog endpoints are fully sealed (verified with 100% test pass rates), **native store configuration cannot proceed until the operator makes two foundational decisions**:
1. **Production App Identity**: The mobile client currently retains upstream author identifiers (`me.huanmeng.lumenFlow` / `me.huanmeng.lumenflow`). Publishing or registering store records under upstream author identifiers is strictly forbidden.
2. **Product & Pricing Policy**: The backend database contains zero commercial subscription plans. No store products or base plans exist in App Store Connect or Google Play Console.

In accordance with Phase 7G-CONFIG safety directives:
- **Zero fake production SKUs** have been created.
- **Zero client-side billing bypasses** (`grantPro()`) have been introduced.
- **Zero secrets** have been committed (strict `.gitignore` patterns added to both repositories).
- **Zero unapproved identifier changes** were made.

---

## Mobile App Identity

A detailed inspection of the current mobile native project configuration reveals:

### iOS Configuration (`ios/Runner.xcodeproj/project.pbxproj`, `ios/Runner/Info.plist`)
- **Bundle Identifier**: `me.huanmeng.lumenFlow`
- **RunnerTests Identifier**: `me.huanmeng.lumenFlow.RunnerTests`
- **Minimum iOS Deployment Target**: `iOS 13.0` (`IPHONEOS_DEPLOYMENT_TARGET = 13.0`)
- **Signing / Team**: `DEVELOPMENT_TEAM` is unset (No Apple Developer team configured)
- **Product Name**: `lumen_flow` (`PRODUCT_NAME = "$(TARGET_NAME)"`)
- **Display Name**: `Lumen Flow` (`CFBundleDisplayName = "Lumen Flow"`)
- **Entitlements**: None present (No `In-App Purchase` capability provisioned)

### Android Configuration (`android/app/build.gradle`, `android/app/src/main/AndroidManifest.xml`)
- **Application ID**: `me.huanmeng.lumenflow`
- **Namespace**: `me.huanmeng.lumenflow`
- **Minimum SDK**: `flutter.minSdkVersion` (API 21 / Android 5.0)
- **Target SDK**: `flutter.targetSdkVersion` (API 34/35)
- **Compile SDK**: `flutter.compileSdkVersion`
- **Signing Configuration**: `signingConfigs.release` configured to read `android/key.properties` (with debug fallback for local runs)
- **Application Label**: `LumenFlow` (`android:label="@string/app_name"`)

### Identity Evaluation Table

| Platform | Current Identifier | Production Ready? | Action Required |
| :--- | :--- | :---: | :--- |
| **iOS** | `me.huanmeng.lumenFlow` | **NO** | Operator must specify production Apple Bundle ID and assign Developer Team |
| **Android** | `me.huanmeng.lumenflow` | **NO** | Operator must specify production Android Application ID and provide release keystore |

---

## Upstream Identifier Audit

The mobile project was audited for traces of the upstream open-source project (`HuanMeng-official/LumenFlow`):

| Occurrence Location | Identifier / Reference | Classification | Remediation Requirement |
| :--- | :--- | :---: | :--- |
| `ios/Runner.xcodeproj/project.pbxproj` | `me.huanmeng.lumenFlow` | **MUST CHANGE** | Update to production Apple Bundle ID once chosen |
| `macos/Runner/Configs/AppInfo.xcconfig` | `me.huanmeng.lumenFlow` | **MUST CHANGE** | Synchronize with production bundle identifier |
| `linux/CMakeLists.txt` | `me.huanmeng.lumen_flow` | **MUST CHANGE** | Synchronize with production package identifier |
| `android/app/build.gradle` | `applicationId = "me.huanmeng.lumenflow"` | **MUST CHANGE** | Update to production Android Application ID once chosen |
| `android/app/build.gradle` | `namespace = "me.huanmeng.lumenflow"` | **MUST CHANGE** | Update Android package namespace |
| `android/app/src/main/java/.../MainActivity.java` | `package me.huanmeng.lumenflow` | **MUST CHANGE** | Refactor Java package structure |
| `android/app/src/main/kotlin/.../*.kt` | `package me.huanmeng.lumenflow` | **MUST CHANGE** | Refactor Kotlin package structure |
| `android/app/src/main/kotlin/.../LiveUpdateManager.kt`| `CHANNEL = "me.huanmeng.lumenflow/live_update"` | **MUST CHANGE** | Update native method channel prefix |
| `lib/services/live_update_service.dart` | `MethodChannel('me.huanmeng.lumenflow/live_update')` | **MUST CHANGE** | Update Dart method channel name |
| `lib/screens/about_screen.dart` | `https://github.com/HuanMeng-official` | **MUST CHANGE** | Update or replace about/credits upstream links |
| `lib/screens/credits_screen.dart` | `assets/image/huanmeng.jpg` | **NOT RELEVANT** | Internal credits avatar asset (optional UI cleanup) |
| `pubspec.yaml` | `name: lumen_flow` | **KEEP** | Internal Dart package name; does not leak to stores |
| `ios/Runner/Info.plist` | `CFBundleURLTypes` (URL schemes) | **NOT RELEVANT** | No custom URL schemes currently declared |
| `android/app/src/main/AndroidManifest.xml` | Deep link schemes | **NOT RELEVANT** | No custom intent-filters declared |

> [!CRITICAL]
> Under no circumstances may an application be submitted to the Apple App Store or Google Play Store under `me.huanmeng.*`.

---

## OPERATOR DECISION REQUIRED — PRODUCTION APP IDENTIFIERS

Before any store application records or In-App Purchase products can be created, the operator must select and approve the authoritative production identities:

### Apple
```text
<production bundle ID> (e.g. com.saascover.tora or com.tora.ai.app)
```

### Android
```text
<production package/application ID> (e.g. com.saascover.tora or com.tora.ai.app)
```

---

## Internal Subscription Plans

An inspection of the backend database schema and models in `/Users/noppanan/new-api` reveals:

- **Model**: `model.SubscriptionPlan` in [`model/subscription.go`](file:///Users/noppanan/new-api/model/subscription.go)
- **Active Records in Database (`data/one-api.db`)**: **0 plans configured**.
- **User Groups in Database**: Only the `"default"` user group exists.
- **Top-Up Ratio Presets (`common/topup-ratio.go`)**: `"default"`, `"vip"`, `"svip"`.

### Internal Plan Eligibility Matrix

| Internal Plan | Code / Title | Group | Period | Quota Semantics | Eligible for Native Store? |
| :---: | :---: | :---: | :---: | :---: | :---: |
| *None* | *None* | *None* | *None* | *None* | **NO PLANS IN DATABASE** |

---

## PRODUCT POLICY DECISION REQUIRED

No commercial tiers or pricing models have been seeded into the production database. Before native store SKUs can be configured, the product team must establish:
1. **Tier Names & Entitlements**: E.g., `Pro`, `Ultra` (or internal groups like `pro`, `vip`).
2. **Quota Allowance**: E.g., `5,000,000` quota units ($10.00 equivalent at 500k units/USD) per month.
3. **Billing Cadence**: Monthly, Annual, or both.
4. **Target Retail Pricing**: USD retail prices for each tier/cadence.

---

## Apple App Store Connect Configuration Checklist

Current configuration readiness for Apple StoreKit 2:

| Apple Requirement | Status | Operator Action |
| :--- | :---: | :--- |
| **Developer Account** | **UNKNOWN** | Ensure Apple Developer Program membership is active |
| **Paid Applications Agreement**| **UNKNOWN** | Accept Paid Apps Agreement & submit Tax/Banking info in App Store Connect |
| **Bundle ID** | **NOT REGISTERED** | Register production bundle ID under Certificates, Identifiers & Profiles |
| **App Record** | **NOT CREATED** | Create app record in App Store Connect using production bundle ID |
| **Subscription Group** | **NOT CREATED** | Create a Subscription Group (e.g., `Tora Subscriptions`) |
| **Products** | **NOT CREATED** | Create Auto-Renewable Subscription products with pricing & localization |
| **Sandbox Tester Account** | **NOT CONFIGURED** | Create at least one Sandbox Apple ID in App Store Connect Users and Access |
| **App Store Server API Key** | **MISSING** | Generate API Key in App Store Connect (Users and Access → Integrations → In-App Purchase) |
| **Notification URL** | **NOT CONFIGURED** | Enter production webhook URL in App Store Connect App Information |

---

## Apple Products

No Apple In-App Purchase subscription products currently exist in App Store Connect.

| Product ID | Subscription Group | Duration | Mapped Internal Plan | Environment | Status |
| :---: | :---: | :---: | :---: | :---: | :---: |
| *None* | *None* | *None* | *None* | *None* | **AWAITING OPERATOR CREATION** |

*(Note: Strings such as `com.saascover.tora.pro.monthly` in prior tests were test fixtures and must not be assumed without real creation).*

---

## Apple Backend Configuration

Status of environment variables in `/Users/noppanan/new-api`:

| Variable | Current Status | Operator Action |
| :--- | :---: | :--- |
| `APPLE_BUNDLE_ID` | `""` (**MISSING**) | Set to registered Apple production bundle ID |
| `APPLE_KEY_ID` | `""` (**MISSING**) | Set to 10-character Key ID from App Store Connect |
| `APPLE_ISSUER_ID` | `""` (**MISSING**) | Set to Issuer UUID from App Store Connect |
| `APPLE_PRIVATE_KEY` | `""` (**MISSING**) | Provide `.p8` private key contents securely via secret manager |
| `APPLE_ENVIRONMENT` | `"production"` | Active environment (`production` or `sandbox`) |
| `APPLE_ALLOW_SANDBOX` | `false` | Enable only in development/staging |

---

## Apple Notification Configuration

- **Mounted Endpoint**: `POST /api/subscription/apple/webhook`
- **Handler**: `controller.AppleSubscriptionWebhook`
- **Security**: Cryptographically verified against embedded Apple Root CA (`service.GlobalAppleVerifier.VerifyNotification`)
- **Required Production URL Format**:
  ```text
  https://<production-api-host>/api/subscription/apple/webhook
  ```

### OPERATOR ACTION REQUIRED — PRODUCTION API HOSTNAME

Candidate domains identified in project documentation:
- `https://api.tora.ai`
- `https://api.toraapi.com`

The operator must confirm the live production HTTPS hostname so the App Store Server Notifications V2 URL can be registered.

---

## Google Play Console Configuration Checklist

Current configuration readiness for Google Play Billing:

| Google Requirement | Status | Operator Action |
| :--- | :---: | :--- |
| **Google Play Developer Account** | **UNKNOWN** | Verify Google Play Developer Console registration |
| **Play Console App Record** | **NOT CREATED** | Create application record in Google Play Console |
| **Production Package Name** | **NOT REGISTERED**| Reserve production package name |
| **Initial Build Upload** | **NOT UPLOADED** | Upload initial AAB with `com.android.vending.BILLING` to Internal Testing |
| **Subscription Products** | **NOT CREATED** | Create subscription product in Play Console (Monetize → Subscriptions) |
| **Base Plans** | **NOT CREATED** | Create auto-renewing base plans (e.g. monthly, annual) |
| **Service Account & API** | **NOT CONFIGURED** | Create GCP service account, enable Android Publisher API, link to Play Console |
| **License Tester** | **NOT CONFIGURED** | Add test email addresses to Setup → License Testing |
| **Cloud Pub/Sub Topic** | **NOT CREATED** | Create GCP Pub/Sub topic for RTDN (e.g. `play-subscription-notifications`) |
| **Pub/Sub IAM Permissions** | **NOT GRANTED** | Grant `google-play-developer-notifications@system.gserviceaccount.com` Publisher role |
| **Push Subscription** | **NOT CREATED** | Create Push subscription pointing to backend webhook with verification token |
| **RTDN Enabled** | **NOT ENABLED** | Enter topic name in Play Console Monetization Setup |

---

## Google Products and Base Plans

No Google Play subscription products or base plans currently exist in Play Console.

| Subscription Product ID | Base Plan ID | Mapped Internal Plan | Billing Period | Environment | Status |
| :---: | :---: | :---: | :---: | :---: | :---: |
| *None* | *None* | *None* | *None* | *None* | **AWAITING OPERATOR CREATION** |

*(Note: Strings such as `tora_pro_sub` and `pro-monthly` in prior tests were test fixtures and must not be assumed without real creation).*

---

## Google Backend Configuration

Status of environment variables in `/Users/noppanan/new-api`:

| Variable | Current Status | Operator Action |
| :--- | :---: | :--- |
| `GOOGLE_PACKAGE_NAME` | `""` (**MISSING**) | Must match the Android application's production `applicationId` |
| `GOOGLE_SERVICE_ACCOUNT_JSON` | `""` (**MISSING**) | GCP Service Account credentials JSON with AndroidPublisher v3 permissions |
| `GOOGLE_PUBSUB_VERIFICATION_TOKEN`| `""` (**MISSING**) | Secure shared secret configured on Pub/Sub push subscription |
| `GOOGLE_ALLOW_TEST_PURCHASE` | `false` | Enable only in development/staging |

> [!IMPORTANT]
> The value of `GOOGLE_PACKAGE_NAME` on the backend must strictly match the mobile client's production `applicationId`. Any mismatch triggers `STORE_BUNDLE_MISMATCH` rejection.

---

## Google RTDN Configuration

The Google Real-Time Developer Notifications pipeline requires:

```text
Google Play Subscription Event
          ↓
Real-Time Developer Notifications (RTDN)
          ↓
Google Cloud Pub/Sub Topic: projects/<gcp-project>/topics/play-billing
          ↓
Pub/Sub Push Subscription (with ?token=<GOOGLE_PUBSUB_VERIFICATION_TOKEN>)
          ↓
POST https://<production-api-host>/api/subscription/google/webhook
          ↓
New-API: Authenticate Push Token → Authoritative Google API Query → ProcessStorePurchaseTx
```

- **Mounted Endpoint**: `POST /api/subscription/google/webhook`
- **Security Check**: Enforces `setting.GooglePubSubVerificationToken` and authoritatively queries `AndroidPublisher v3` (`purchases.subscriptionsv2.get`) before any database record is mutated.

---

## Store Product Mappings

Database table: `store_product_mappings` (schema defined in [`model/store_billing.go`](file:///Users/noppanan/new-api/model/store_billing.go)).

- **Current State**: Exactly **0** rows.
- No dummy or example mappings have been injected into the database.
- Once the operator creates real internal plans and real store SKUs, mapping rows must be inserted with:
  - `platform`: `'apple'` or `'google'`
  - `store_product_id`: Exact product ID from App Store Connect / Play Console
  - `store_base_plan_id`: Base plan ID for Google (or empty for Apple)
  - `internal_plan_id`: Integer ID of the active `subscription_plans` row
  - `environment`: `'production'`, `'sandbox'`, or `'all'`
  - `enabled`: `true`

---

## Mapping Environment Separation

Environment isolation is enforced by the backend at both query and execution levels:
1. `GetStoreProductMapping` queries `(LOWER(environment) = ? OR LOWER(environment) = 'all')`.
2. When `APPLE_ALLOW_SANDBOX = false`, any Apple receipt bearing `environment == "Sandbox"` is rejected with `400 STORE_ENVIRONMENT_MISMATCH`.
3. When `GOOGLE_ALLOW_TEST_PURCHASE = false`, test license purchases are rejected with `400 STORE_ENVIRONMENT_MISMATCH`.
4. The client **cannot** select or override the target environment.

---

## Product Catalog Verification

- **Endpoint**: `GET /api/subscription/store/products` (authenticated)
- **Behavior Verified via Unit Tests**:
  - `TestGetStoreProductCatalog_Success`: Returns only enabled products mapped to active plans.
  - `TestGetStoreProductCatalog_FilterPlatform`: Accurately filters by `platform=apple` or `platform=google`.
  - **No Secrets**: Contains only public display metadata (`product_id`, `base_plan_id`, `title`, `duration_unit`, `duration_value`, `total_amount`).
- **Live Readiness**: Currently returns `{"success": true, "data": []}` because zero mappings exist in the database, safely preventing client errors.

---

## Store Price Source

The architectural distribution of billing metadata is frozen:

| Metadata Attribute | Authoritative Source | Presentation in LumenFlow |
| :--- | :--- | :--- |
| **Available Products / SKUs** | Backend Catalog (`/store/products`) | Fetched dynamically on paywall load |
| **Internal Entitlement Mapping** | Backend Database (`StoreProductMapping`) | Hidden from client; resolved on verify |
| **Quota Allocation / Plan Limits**| Backend Subscription Plan | Presented from backend plan model |
| **Retail Price Amount** | Native Store (App Store / Google Play) | Extracted from store `ProductDetails` |
| **Currency Symbol / Code** | Native Store (App Store / Google Play) | Formatted via store locale / currency |
| **Subscription Duration String** | Native Store / Backend Plan | Localized store subscription period |

> [!WARNING]
> Flutter Phase 7G-B must never hardcode prices or display the backend's internal monetary amounts. The native store remains the single source of truth for retail prices, currencies, and tax disclosures.

---

## Secret Deployment

A secret security audit was conducted across both repositories:
- **Scan Results**:
  - No `.p8` private keys found.
  - No Google service account JSON files found.
  - No leaked API private keys found.
- **Accidental Commit Mitigation**:
  - Added `*.p8`, `*service-account*.json`, `*service_account*.json` to `.gitignore` in both `/Users/noppanan/new-api` and `scratch/LumenFlow`.
  - Documented variable names in `.env.example` without values.
- **Recommended Deployment Mechanism**:
  - In containerized production (Docker / Kubernetes), secrets must be injected via secure environment variables or mounted Kubernetes Secret volumes.
  - Never store credentials in source control or Docker images.

---

## Sandbox / Test Readiness

### Apple Sandbox Readiness
- **Status**: **NOT READY**
- **Missing Requirements**:
  1. Approved production Bundle ID.
  2. Registered App Store Connect app record & subscription group.
  3. Created Apple auto-renewable subscription products.
  4. Created Sandbox Apple ID tester.
  5. Backend environment variables (`APPLE_KEY_ID`, `APPLE_ISSUER_ID`, `APPLE_PRIVATE_KEY`, `APPLE_BUNDLE_ID`).
  6. Resolved production API hostname for webhooks.

### Google Test Purchase Readiness
- **Status**: **NOT READY**
- **Missing Requirements**:
  1. Approved production Android Application ID.
  2. Google Play Console app record & initial AAB upload to testing track.
  3. Created subscription product & base plan in Play Console.
  4. Configured License Tester email in Play Console.
  5. Backend environment variables (`GOOGLE_PACKAGE_NAME`, `GOOGLE_SERVICE_ACCOUNT_JSON`, `GOOGLE_PUBSUB_VERIFICATION_TOKEN`).
  6. GCP Pub/Sub topic and push subscription configured.

---

## Local Development Rules

For local development where store infrastructure is absent:
1. **No Paid Access Bypasses**: Never add `if (isDev) grantPro()` or local entitlement grants.
2. **Automated Unit Testing**: Tests use `SetAppleVerifierForTest` and `SetGoogleVerifierForTest` with mock implementations and synthetic ECDSA certificates.
3. **Safe Server Fallback**: If store credentials are missing, the server returns `503 STORE_SERVER_UNAVAILABLE`. All standard non-store APIs (`GET /api/subscription/self`, `GET /api/subscription/plans`, chat streaming, BYOK) remain 100% operational.

---

## Frozen Phase 7G-B Inputs

Phase 7G-B (Flutter StoreKit / Play Billing UI & integration) will consume:

### Backend Endpoints
- **Product Catalog**: `GET /api/subscription/store/products`
- **Apple Verify**: `POST /api/subscription/apple/verify` (`{"signed_transaction_info": "..."}`)
- **Google Verify**: `POST /api/subscription/google/verify` (`{"product_id": "...", "purchase_token": "..."}`)
- **Current Subscription**: `GET /api/subscription/self`

### Normalized Error Contract
- `STORE_VERIFICATION_PENDING`: Purchase awaiting bank/store settlement (display pending banner).
- `STORE_PROOF_INVALID`: Receipt signature failed or corrupted.
- `STORE_PRODUCT_UNRECOGNIZED`: SKU not mapped to an active backend plan.
- `STORE_BUNDLE_MISMATCH`: App identity mismatch.
- `STORE_ENVIRONMENT_MISMATCH`: Sandbox proof submitted to production.
- `STORE_SUBSCRIPTION_EXPIRED`: Subscription is in the past.
- `STORE_SUBSCRIPTION_REVOKED`: Subscription cancelled/refunded.
- `STORE_TRANSACTION_ALREADY_BOUND`: Cross-account conflict (subscription linked to another user).
- `STORE_SERVER_UNAVAILABLE`: Store verification credentials missing on server.

---

## Operator Actions Required

The following manual operator steps must be completed before store billing can become operational:

```markdown
### 1. App Identity & Policy (IMMEDIATE BLOCKER)
- [ ] Decide production Apple Bundle ID (e.g., com.saascover.tora)
- [ ] Decide production Android Application ID (e.g., com.saascover.tora)
- [ ] Decide commercial subscription tiers, quotas, and retail pricing
- [ ] Insert internal SubscriptionPlan rows into database

### 2. Apple App Store Setup
- [ ] Register production Bundle ID in Apple Developer Portal
- [ ] Create App Record in App Store Connect
- [ ] Accept Paid Applications Agreement & enter banking details
- [ ] Create Subscription Group and Auto-Renewable Subscription products
- [ ] Generate In-App Purchase API Key (.p8) and record Key ID / Issuer ID
- [ ] Create Sandbox Tester Apple ID
- [ ] Set App Store Server Notifications V2 URL to https://<production-host>/api/subscription/apple/webhook

### 3. Google Play Setup
- [ ] Create Application in Google Play Console with matching Application ID
- [ ] Build & upload initial signed AAB to Internal Testing track
- [ ] Create Subscription product and Base Plans
- [ ] Create GCP Service Account with AndroidPublisher permissions and download JSON key
- [ ] Add license tester email addresses
- [ ] Create Cloud Pub/Sub topic and Push subscription pointing to https://<production-host>/api/subscription/google/webhook?token=<secret_token>
- [ ] Enable RTDN in Play Console

### 4. Backend Deployment
- [ ] Configure Apple environment variables on server (APPLE_KEY_ID, APPLE_ISSUER_ID, APPLE_PRIVATE_KEY, APPLE_BUNDLE_ID)
- [ ] Configure Google environment variables on server (GOOGLE_PACKAGE_NAME, GOOGLE_SERVICE_ACCOUNT_JSON, GOOGLE_PUBSUB_VERIFICATION_TOKEN)
- [ ] Insert StoreProductMapping records connecting real store SKUs to internal plan IDs
```

---

## Regression Results

- **Backend (`new-api`)**:
  - `go vet ./model ./service ./controller`: **PASS (0 issues)**
  - `go test ./service -count=1`: **PASS**
  - `go test ./model -count=1`: **PASS**
  - `go test ./controller -count=1`: **PASS**
- **Mobile (`LumenFlow`)**:
  - `flutter analyze`: **PASS (No issues found)**
  - `flutter test`: **PASS (86/86 tests passing)**

---

## Remaining Blockers

1. **Production App Identity**: Upstream author identifier (`me.huanmeng.lumenflow`) cannot be registered in production stores.
2. **Product Policy**: Zero subscription plans exist in the backend database.
3. **Store Credentials**: Operator has not created or provided Apple / Google developer console credentials.

---

## FINAL STATUS: PRODUCTION APP IDENTITY REQUIRED
