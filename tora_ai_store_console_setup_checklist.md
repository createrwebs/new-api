# Tora AI — Store Console Setup & Verification Checklist

**Document**: `tora_ai_store_console_setup_checklist.md`  
**Target Identity**:  
- iOS Bundle ID: `com.saascover.tora`  
- Android Application ID: `com.saascover.tora`  
- Canonical Production API: `https://api.tora.ai`  
- Staging API: `https://staging-api.tora.ai`  
**Commercial Baseline**:  
- Plan: `Pro Monthly`  
- Price: USD $9.99 / month  
- Quota: 2,000,000 units / month (group `pro`)  

This checklist tracks external operator setup actions required in Apple App Store Connect, Google Play Console, and Google Cloud Platform. Complete each item to unblock live sandbox and production verification.

---

## 1. Apple App Store Connect

- [ ] **Action 1.1: Register App ID & Capabilities**
  - **Where**: Apple Developer Portal -> Certificates, Identifiers & Profiles -> Identifiers -> App IDs
  - **Expected Value**: `com.saascover.tora` with Capability: "In-App Purchase" enabled.
  - **How to verify**: Verify App ID is listed with prefix and explicit ID `com.saascover.tora`.
  - **Blocking which test**: iOS App Store build signing & StoreKit product discovery.

- [ ] **Action 1.2: Create App Record in App Store Connect**
  - **Where**: App Store Connect -> Apps -> Add App (+)
  - **Expected Value**: Name: `Tora AI`, Bundle ID: `com.saascover.tora`, SKU: `tora-ai-ios`, Primary Language: English (US).
  - **How to verify**: App dashboard opens in App Store Connect.
  - **Blocking which test**: StoreKit 2 transaction verification & sandbox testing.

- [ ] **Action 1.3: Create Subscription Group**
  - **Where**: App Store Connect -> Apps -> Tora AI -> Monetization -> Subscriptions -> Subscription Groups (+)
  - **Expected Value**: Group Name: `Tora AI Subscriptions`.
  - **How to verify**: Group appears with 0 subscriptions.
  - **Blocking which test**: Auto-renewing subscription creation.

- [ ] **Action 1.4: Create Pro Monthly Auto-Renewable Subscription**
  - **Where**: Under `Tora AI Subscriptions` -> (+) Create Subscription
  - **Expected Value**: 
    - Reference Name: `Pro Monthly`
    - Product ID: `com.saascover.tora.pro.monthly`
    - Duration: `1 Month`
    - Price: `USD $9.99 / month`
    - Localization: English: "Tora Pro Monthly" / "2,000,000 units per month with access to Pro AI models."
  - **How to verify**: Product ID status is "Ready to Submit" or "Approved".
  - **Blocking which test**: Native iOS `in_app_purchase` live catalog query and purchase flow.

- [ ] **Action 1.5: Configure App Store Server Notifications V2**
  - **Where**: App Store Connect -> Apps -> Tora AI -> App Information -> App Store Server Notifications
  - **Expected Value**:
    - Version: Version 2
    - Production Server URL: `https://api.tora.ai/api/subscription/apple/webhook`
    - Sandbox Server URL: `https://staging-api.tora.ai/api/subscription/apple/webhook`
  - **How to verify**: Click "Send Test Notification" and verify HTTP 200 response in backend logs.
  - **Blocking which test**: Real-time renewal, revocation, and refund reconciliation.

- [ ] **Action 1.6: Generate StoreKit 2 Private Key (.p8)**
  - **Where**: App Store Connect -> Users and Access -> Integrations -> In-App Purchase
  - **Expected Value**: Active Key with "In-App Purchase" role. Download `.p8` file. Record Key ID and Issuer ID.
  - **How to verify**: Set `APPLE_STOREKIT_KEY_ID`, `APPLE_STOREKIT_ISSUER_ID`, `APPLE_BUNDLE_ID=com.saascover.tora`, and `APPLE_STOREKIT_PRIVATE_KEY` on backend; call verification endpoint.
  - **Blocking which test**: Live backend JWS verification with Apple Root CA chain.

- [ ] **Action 1.7: Create Sandbox Tester Account**
  - **Where**: App Store Connect -> Users and Access -> Sandbox -> Testers (+)
  - **Expected Value**: Clean Apple ID not linked to production purchase history.
  - **How to verify**: Sign into Sandbox Apple ID on test iOS device under Settings -> App Store -> Sandbox Account.
  - **Blocking which test**: Phase 7G-C manual Apple Sandbox E2E purchase.

---

## 2. Google Play Console & Google Cloud Platform

- [ ] **Action 2.1: Create App in Google Play Console**
  - **Where**: Google Play Console -> All Apps -> Create App
  - **Expected Value**: App Name: `Tora AI`, Default Language: English (US), App, Free. Package name assigned as `com.saascover.tora`.
  - **How to verify**: Dashboard created with package `com.saascover.tora`.
  - **Blocking which test**: Play Billing discovery and license testing.

- [ ] **Action 2.2: Enable Google Play Developer API**
  - **Where**: Google Cloud Console -> APIs & Services -> Enable APIs -> "Google Play Android Developer API".
  - **Expected Value**: Enabled on linked GCP project.
  - **How to verify**: Status is "Enabled" with metrics active.
  - **Blocking which test**: Backend server-to-server purchase token verification.

- [ ] **Action 2.3: Create Service Account & Grant Permissions**
  - **Where**: GCP Console -> IAM & Admin -> Service Accounts (+) -> Create key (JSON).
  - **Expected Value**: Service account invited into Play Console with "View financial data, orders, and cancellation survey responses" and "Manage orders and subscriptions".
  - **How to verify**: Backend configured with `GOOGLE_SERVICE_ACCOUNT_JSON` successfully queries test purchases.
  - **Blocking which test**: Authoritative backend verification against `androidpublisher.purchases.subscriptionsv2.get`.

- [ ] **Action 2.4: Create Subscription Product & Base Plan**
  - **Where**: Play Console -> Tora AI -> Monetize -> Products -> Subscriptions
  - **Expected Value**:
    - Subscription ID: `tora_pro`
    - Base Plan ID: `monthly`
    - Type: Auto-renewing
    - Billing period: 1 month
    - Price: USD $9.99
    - Status: Active
  - **How to verify**: Base plan state is "Active".
  - **Blocking which test**: Android `in_app_purchase` live SKU query and purchase.

- [ ] **Action 2.5: Configure Google Cloud Pub/Sub RTDN**
  - **Where**: GCP Console -> Cloud Pub/Sub -> Topics -> Create Topic: `play-billing-rtdn`.
  - **Expected Value**:
    - Grant Publisher role to `google-play-developer-notifications@system.gserviceaccount.com`.
    - Create Subscription with Push Delivery URL: `https://api.tora.ai/api/subscription/google/webhook?token={SECRET_TOKEN}`.
  - **How to verify**: Play Console -> Monetization setup -> Real-time developer notifications -> Link topic name. Click "Send test notification".
  - **Blocking which test**: Google RTDN renewal, revocation, and expiration handling.

- [ ] **Action 2.6: Configure License Tester**
  - **Where**: Play Console -> Setup -> License testing -> Add test Gmail accounts.
  - **Expected Value**: License test response set to "RESPOND_NORMALLY" (test card always approves).
  - **How to verify**: Purchase inside internal test track triggers test purchase card.
  - **Blocking which test**: Phase 7G-C manual Google Play internal track E2E.

- [ ] **Action 2.7: Android Production Signing Keystore**
  - **Where**: Local Secure Vault / Password Manager
  - **Expected Value**: `android/key.properties` pointing to release keystore (`.jks` / `.keystore`):
    ```properties
    storePassword=YOUR_STORE_PASSWORD
    keyPassword=YOUR_KEY_PASSWORD
    keyAlias=YOUR_KEY_ALIAS
    storeFile=/path/to/upload-keystore.jks
    ```
  - **How to verify**: Run `flutter build appbundle --release` and verify package signs with production upload certificate.
  - **Blocking which test**: Google Play Console release track submission / internal app sharing.

---

## 3. Database Mapping Activation (Post-Console Setup)

After completing the console actions above, execute the following SQL in New-API database:

```sql
-- Link Apple StoreKit product to internal plan Pro Monthly (ID: 1)
INSERT INTO store_product_mappings (
    platform, store_product_id, store_base_plan_id, internal_plan_id, environment, enabled, created_at, updated_at
) VALUES (
    'apple', 'com.saascover.tora.pro.monthly', '', 1, 'all', 1, strftime('%s', 'now'), strftime('%s', 'now')
);

-- Link Google Play Billing product & base plan to internal plan Pro Monthly (ID: 1)
INSERT INTO store_product_mappings (
    platform, store_product_id, store_base_plan_id, internal_plan_id, environment, enabled, created_at, updated_at
) VALUES (
    'google', 'tora_pro', 'monthly', 1, 'all', 1, strftime('%s', 'now'), strftime('%s', 'now')
);
```

---

## 4. Current Status
- **Engineering Status**: 100% COMPLETE & VERIFIED.
  - Full-Stack Managed & BYOK E2E passed (100%).
  - Android APK & AAB release builds verified.
  - iOS release compilation verified.
  - All 117 mobile test specs & backend unit tests passing.
- **Console Status**: `OPERATOR_BLOCKED` pending external credentials and Store Console setup actions above.
- **E2E Status**: Live Store Sandbox / Test Track verification (R8) will commence immediately upon operator completion of this checklist.
