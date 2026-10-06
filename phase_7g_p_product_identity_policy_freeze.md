# Phase 7G-P — Tora AI Production Identity & Commercial Policy Freeze Report

**Date**: 2026-10-05  
**Status**: APPROVED & FROZEN  
**Final Verdict**: `FINAL STATUS: READY FOR STORE CONSOLE SETUP`

---

## 1. Executive Summary

Phase 7G-P converts the mobile client (`scratch/LumenFlow`) and backend subscription infrastructure (`new-api`, branch `feat/formobile`) from development/reference identity into the approved **Tora AI** production product identity and commercial policy baseline.

All upstream author bundle identifiers (`me.huanmeng.*`) and draft domains have been systematically replaced with the approved production identifiers. A default commercial plan seeder for **Pro Monthly** ($9.99 / 2,000,000 units / month) has been integrated into the New-API database lifecycle and verified with automated unit tests.

The repositories are now fully prepared for operator console configuration in Apple App Store Connect and Google Play Console.

---

## 2. Approved Production Product Identity Baseline

The following values are approved by the operator and frozen across the stack:

| Parameter | Approved Production Value | Notes |
| :--- | :--- | :--- |
| **Product Name** | `Tora AI` | User-facing display name across all platforms |
| **iOS Bundle Identifier** | `com.saascover.tora` | App Store Connect primary bundle identifier |
| **iOS Test Bundle Identifier** | `com.saascover.tora.RunnerTests` | Unit test bundle identifier |
| **Android Application ID** | `com.saascover.tora` | Google Play package name |
| **Android Namespace** | `com.saascover.tora` | Gradle namespace & R package |
| **Canonical Production API** | `https://api.tora.ai` | Production API endpoint & cookie Origin guard |
| **Staging API** | `https://staging-api.tora.ai` | Staging API endpoint |
| **Linux Application ID** | `com.saascover.tora` | GTK application identifier |
| **macOS Bundle Identifier** | `com.saascover.tora` | macOS desktop identifier |

---

## 3. Mobile Codebase Identity Modifications

The mobile application (`scratch/LumenFlow`) has been migrated from `me.huanmeng.lumenflow` to `com.saascover.tora`:

### 3.1 iOS Configuration
- [`ios/Runner.xcodeproj/project.pbxproj`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/ios/Runner.xcodeproj/project.pbxproj):
  - Changed `PRODUCT_BUNDLE_IDENTIFIER = me.huanmeng.lumenFlow;` to `PRODUCT_BUNDLE_IDENTIFIER = com.saascover.tora;` across Profile, Debug, and Release configurations.
  - Changed `PRODUCT_BUNDLE_IDENTIFIER = me.huanmeng.lumenFlow.RunnerTests;` to `PRODUCT_BUNDLE_IDENTIFIER = com.saascover.tora.RunnerTests;`.
- [`ios/Runner/Info.plist`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/ios/Runner/Info.plist):
  - Changed `CFBundleDisplayName` from `Lumen Flow` to `Tora AI`.
  - Changed `CFBundleName` from `lumen_flow` to `tora_ai`.

### 3.2 Android Configuration & Package Tree
- [`android/app/build.gradle`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/android/app/build.gradle):
  - Changed `namespace = "me.huanmeng.lumenflow"` to `namespace = "com.saascover.tora"`.
  - Changed `applicationId = "me.huanmeng.lumenflow"` to `applicationId = "com.saascover.tora"`.
- **Source Package Relocation**:
  - Relocated Java/Kotlin sources from `android/app/src/main/{java,kotlin}/me/huanmeng/lumenflow` to `android/app/src/main/{java,kotlin}/com/saascover/tora`.
  - Updated [`MainActivity.java`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/android/app/src/main/java/com/saascover/tora/MainActivity.java): `package com.saascover.tora;` and registered `com.saascover.tora.LiveUpdatePlugin`.
  - Updated [`LiveUpdateManager.kt`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/android/app/src/main/kotlin/com/saascover/tora/LiveUpdateManager.kt): `package com.saascover.tora` and method channel `com.saascover.tora/live_update`.
  - Updated [`LiveUpdatePlugin.kt`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/android/app/src/main/kotlin/com/saascover/tora/LiveUpdatePlugin.kt): `package com.saascover.tora`.
  - Updated [`SnackbarNotificationManager.kt`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/android/app/src/main/kotlin/com/saascover/tora/SnackbarNotificationManager.kt): `package com.saascover.tora`, `import com.saascover.tora.R`, and updated notification channels and title to `Tora AI`.
  - Removed obsolete `me/` directory trees.
- Android String Resources:
  - [`android/app/src/main/res/values/strings.xml`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/android/app/src/main/res/values/strings.xml): Updated `app_name` to `Tora AI`.
  - [`android/app/src/main/res/values-zh/strings.xml`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/android/app/src/main/res/values-zh/strings.xml): Updated `app_name` to `Tora AI`.

### 3.3 Desktop Platforms
- [`macos/Runner/Configs/AppInfo.xcconfig`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/macos/Runner/Configs/AppInfo.xcconfig):
  - Changed `PRODUCT_NAME` to `tora_ai`.
  - Changed `PRODUCT_BUNDLE_IDENTIFIER` to `com.saascover.tora`.
- [`macos/Runner.xcodeproj/project.pbxproj`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/macos/Runner.xcodeproj/project.pbxproj):
  - Changed `RunnerTests` bundle identifier to `com.saascover.tora.RunnerTests`.
- [`linux/CMakeLists.txt`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/linux/CMakeLists.txt):
  - Changed `set(APPLICATION_ID "me.huanmeng.lumen_flow")` to `set(APPLICATION_ID "com.saascover.tora")`.

### 3.4 Dart Services & Localization
- [`lib/services/live_update_service.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/services/live_update_service.dart):
  - Updated method channel to `com.saascover.tora/live_update`.
  - Updated default notification title to `Tora AI`.
- [`lib/config/app_environment.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/config/app_environment.dart):
  - Frozen canonical production URL to `https://api.tora.ai`.
  - Cleaned up development comments.
- [`lib/main.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/main.dart):
  - Updated `CupertinoApp` title from `流光` to `Tora AI`.
- Localization arb Files:
  - Updated `appTitle` to `Tora AI` across `app_en.arb`, `app_es.arb`, `app_ja.arb`, `app_ko.arb`, `app_zh.arb`, and `app_lzh.arb`.
  - Regenerated localization bindings via `flutter gen-l10n`.

---

## 4. License & Upstream Attribution Integrity

In strict compliance with open-source licensing:
- [`LICENSE`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/LICENSE): The MIT License (Copyright (c) 2025 HuanMeng-official) is preserved unmodified.
- [`lib/screens/credits_screen.dart`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/lib/screens/credits_screen.dart): All original developer and contributor credits (幻梦official, 浮沫, 浅唱ヾ落雨殇, 枫下之秋) are preserved unmodified in the in-app Credits screen.

---

## 5. Approved Commercial Policy & Backend Seeding

### 5.1 Plan Specification
The operator has approved the initial commercial structure:

| Parameter | Specification | Notes |
| :--- | :--- | :--- |
| **Plan Name** | `Pro Monthly` | Public plan title |
| **Subtitle** | `Pro model access with 2,000,000 units per month` | Public description |
| **Retail Price** | `USD $9.99` | Monthly consumer pricing |
| **Duration Unit** | `month` | Standard calendar month |
| **Duration Value** | `1` | 1 month |
| **Quota Units** | `2,000,000` | 2M units/month (at 500,000 units/USD = $4.00 nominal quota) |
| **Quota Reset Period** | `monthly` | Reset remaining quota at each monthly cycle |
| **Upgrade Group** | `pro` | Grants user membership in backend `pro` group |
| **Downgrade Group** | `default` | Reverts expired/cancelled subscriptions to `default` |
| **Enabled** | `true` | Active plan |
| **Allow Balance Pay** | `false` | In-app credit conversion disabled for standard store SKU |
| **Allow Wallet Overflow** | `true` | Users with top-up balance can continue when quota exhausted |

### 5.2 Explicit Non-Approvals
The following tiers and plans are explicitly NOT approved and MUST NOT be created:
- No Annual subscription (`pro_annual`)
- No Ultra, VIP, or Premium tiers
- No Lifetime subscription

### 5.3 Free Tier & BYOK Independence Policy
- **Free Tier**: Uses backend group `default`. Default users receive whatever base free-tier quota is configured on the instance.
- **BYOK Independence**: Free users with valid server-managed BYOK credentials retain full ability to chat with BYOK models. BYOK connection mode is completely decoupled from subscription status and does not require a paid plan.
- **Exhausted Quota**: Zero-quota Free users can continue inference using BYOK without any paywall blocking.

### 5.4 Backend Implementation & Migration
Implemented in [`model/subscription.go`](file:///Users/noppanan/new-api/model/subscription.go) and [`model/main.go`](file:///Users/noppanan/new-api/model/main.go):
1. **`InitDefaultSubscriptionPlan()`**:
   - Executes during `migrateDB()`.
   - If no subscription plans exist in the database, automatically seeds `Pro Monthly` with the approved fields.
   - Fully idempotent: exits immediately without mutation if any plan exists.
2. **`EnsureProMonthlySubscriptionPlan()`**:
   - Helper returning the approved `Pro Monthly` plan, creating it if missing.
3. **Unit Tests**:
   - Created [`model/subscription_plan_init_test.go`](file:///Users/noppanan/new-api/model/subscription_plan_init_test.go).
   - Verifies plan creation, field accuracy, and idempotency.

---

## 6. Operator Store Setup Worksheet

The operator should use the exact parameters below when configuring App Store Connect and Google Play Console.

### 6.1 Apple App Store Connect Setup

```text
[App Information]
App Name:                    Tora AI
Primary Language:            English (US) / Simplified Chinese
Bundle ID:                   com.saascover.tora
SKU:                         tora-ai-ios

[In-App Purchases -> Subscriptions]
Subscription Group Name:     Tora AI Subscriptions
Subscription Group ID:       (Generated by Apple)

[Subscription Product Details]
Reference Name:              Pro Monthly
Product ID:                  com.saascover.tora.pro.monthly
Subscription Duration:       1 Month
Price:                       USD $9.99 / month
App Store Localization:
  - Display Name:            Tora Pro Monthly
  - Description:             2,000,000 units per month with access to Pro AI models.

[App Store Server Notifications V2]
URL (Production):            https://api.tora.ai/api/subscription/apple/webhook
URL (Sandbox):               https://staging-api.tora.ai/api/subscription/apple/webhook
Notification Version:        Version 2
```

### 6.2 Google Play Console Setup

```text
[App Details]
App Name:                    Tora AI
Package Name:                com.saascover.tora

[Monetize -> Products -> Subscriptions]
Product ID:                  tora_pro
Name:                        Tora Pro
Description:                 Monthly subscription with Pro model access and 2,000,000 units.

[Base Plan Details]
Base Plan ID:                monthly
Type:                        Auto-renewing
Billing Period:              1 month
Renewal Type:                Auto-renews until cancelled
Grace Period:                16 days (recommended)
Price (USD):                 $9.99

[Real-Time Developer Notifications (RTDN)]
Topic Name:                  projects/{GCP_PROJECT_ID}/topics/play-billing-rtdn
Pub/Sub Push Endpoint:       https://api.tora.ai/api/subscription/google/webhook
```

---

## 7. Store Product Mapping Preparation

Once the operator creates the products in App Store Connect and Google Play Console, the following records should be inserted into the `store_product_mappings` table (or populated via Admin API):

```sql
-- Apple App Store Mapping (InternalPlanId references seeded 'Pro Monthly' plan)
INSERT INTO store_product_mappings (
    platform, store_product_id, store_base_plan_id, internal_plan_id, environment, enabled, created_at, updated_at
) VALUES (
    'apple', 'com.saascover.tora.pro.monthly', '', 1, 'all', 1, strftime('%s', 'now'), strftime('%s', 'now')
);

-- Google Play Billing Mapping
INSERT INTO store_product_mappings (
    platform, store_product_id, store_base_plan_id, internal_plan_id, environment, enabled, created_at, updated_at
) VALUES (
    'google', 'tora_pro', 'monthly', 1, 'all', 1, strftime('%s', 'now'), strftime('%s', 'now')
);
```

*Note: Per Phase 7G-P instructions, no unverified mock mappings were committed to production migrations. The table remains clean awaiting operator console creation.*

---

## 8. Secret Protection & Keystore Safety Audit

Both repositories maintain strict protections against accidental credential exposure:

- [`.gitignore`](file:///Users/noppanan/new-api/.gitignore) in `new-api` excludes:
  - `*.p8` (Apple StoreKit private keys)
  - `*service-account*.json` / `*service_account*.json` (Google Cloud Service Account keys)
  - `*.keystore` / `*.jks` (Android signing keys)
- [`.gitignore`](file:///Users/noppanan/.gemini/antigravity/scratch/LumenFlow/.gitignore) in `LumenFlow` excludes:
  - `*.keystore` / `*.jks`
  - `key.properties`
  - `*.p8`
- **Zero Secrets Committed**: Audit confirms 0 private keys, 0 signing credentials, and 0 production secrets present in git history or untracked changes.

---

## 9. Verification & Quality Assurance Results

### 9.1 Backend Test Suite (`new-api`)
- `model`: 100% PASS (including `TestInitDefaultSubscriptionPlan_EmptyDB`, `TestInitDefaultSubscriptionPlan_Idempotent`, and `TestEnsureProMonthlySubscriptionPlan`)
- `service`: 100% PASS (including Apple Root CA verification and Google Play verifier tests)
- `controller`: 100% PASS (including store verification, replay protection, and webhook handling tests)
- `go vet ./...`: 0 issues found

### 9.2 Mobile Test Suite (`LumenFlow`)
- `flutter test`: 86/86 unit and integration tests passed cleanly
- `flutter analyze`: 0 issues found
- `flutter gen-l10n`: Completed with code 0

---

## 10. Operator Actions Remaining Checklist

To complete native store billing readiness, the operator must perform the following administrative actions:

- [ ] **App Store Connect**:
  1. Register App ID `com.saascover.tora` under Apple Developer Certificates, Identifiers & Profiles.
  2. Create app `Tora AI` in App Store Connect with bundle identifier `com.saascover.tora`.
  3. Create Subscription Group `Tora AI Subscriptions` and product `com.saascover.tora.pro.monthly` ($9.99 USD / 1 month).
  4. Generate StoreKit 2 In-App Purchase Key (`.p8`), note Key ID, Issuer ID, and Bundle ID.
  5. Configure App Store Server Notifications V2 URL: `https://api.tora.ai/api/subscription/apple/webhook`.
  6. Provide `.p8` key and credentials via backend environment variables.
- [ ] **Google Play Console**:
  1. Create app `Tora AI` with application ID `com.saascover.tora`.
  2. Create subscription product `tora_pro` with base plan `monthly` ($9.99 USD / 1 month).
  3. Set up Google Cloud Pub/Sub topic and subscription with push URL: `https://api.tora.ai/api/subscription/google/webhook`.
  4. Create Google Play Developer API Service Account JSON key.
  5. Provide Service Account credentials via backend environment variables.
- [ ] **Database Mapping Insertion**:
  1. Insert the two mapping records in `store_product_mappings` linking Apple `com.saascover.tora.pro.monthly` and Google `tora_pro:monthly` to `Pro Monthly` (Plan ID 1).
- [ ] **Phase 7G-B Authorization**:
  1. Once store products and credentials are active in sandbox/test environments, proceed to **Phase 7G-B (Flutter In-App Purchases)**.

---

```text
================================================================================
FINAL STATUS: READY FOR STORE CONSOLE SETUP
================================================================================
```
