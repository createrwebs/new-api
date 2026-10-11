# Tora AI — New-API Full Capability Audit & Mobile Classification Matrix

## Executive Summary

This audit performs an exhaustive, source-grounded inspection of the modified **New-API / Tora backend** (`/Users/noppanan/new-api`, branch `feat/formobile`) and its OpenAPI specification (`docs/openapi/api.json`).

Per the **Source-of-Truth Hierarchy** defined in `AGENTS.md`:
1. **Current source code and runtime behavior strictly supersede documentation.**
2. Endpoints are never exposed to the consumer application merely because they exist in upstream documentation.
3. Every endpoint is classified into exactly one primary category:
   - `NATIVE`: High-value consumer mobile workflow implemented directly in Flutter.
   - `WEB_FALLBACK`: External provider, OAuth, captcha-gated, or hosted flow opened in system/in-app browser.
   - `SERVER_ONLY`: Internal infrastructure, webhooks, or server-to-server callbacks.
   - `ADMIN_ONLY`: System administration, channel routing, user management, and provider catalog controls.
   - `NOT_NEEDED`: Desktop-only, batch developer tooling, or affiliate features irrelevant to core Tora AI mobile.
   - `DEPRECATED`: Abandoned or obsolete upstream routes preserved only for backward compatibility.

---

## 1. Capability Classification Matrix

### 1.1 Public & Discovery Plane (`/api/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/api/status` | GET | `controller.GetStatus` | Public, GlobalAPIRateLimit | `NATIVE` | Server health, version, build, and feature flags. Used for connectivity checks. |
| `/api/notice` | GET | `controller.GetNotice` | Public | `NATIVE` | System bulletin / announcement banner displayed in app. |
| `/api/user-agreement` | GET | `controller.GetUserAgreement` | Public | `NATIVE` | Terms of service text for in-app legal review. |
| `/api/privacy-policy` | GET | `controller.GetPrivacyPolicy` | Public | `NATIVE` | Privacy policy text required for App Store and Play Store compliance. |
| `/api/about` | GET | `controller.GetAbout` | Public | `NATIVE` | About app, version, and server details. |
| `/api/home_page_content`| GET | `controller.GetHomePageContent`| Public | `WEB_FALLBACK`| Raw HTML/Markdown landing page content. |
| `/api/pricing` | GET | `controller.GetPricing` | HeaderNavModuleAuth | `WEB_FALLBACK`| Public web pricing table. Mobile uses `/api/subscription/plans`. |
| `/api/setup` | GET/POST | `controller.GetSetup` / `PostSetup` | Public (Initial setup) | `ADMIN_ONLY` | Initial instance initialization wizard. Admin only. |
| `/api/status/test` | GET | `controller.TestStatus` | `AdminAuth()` | `ADMIN_ONLY` | Admin diagnostic status check. |
| `/api/uptime/status` | GET | `controller.GetUptimeKumaStatus` | Public | `NOT_NEEDED` | Uptime Kuma monitoring widget. |
| `/api/perf-metrics/*` | GET | `controller.GetPerfMetrics*` | Pricing Module Auth | `NOT_NEEDED` | Web dashboard performance metrics. |
| `/api/rankings` | GET | `controller.GetRankings` | Rankings Module Auth | `NOT_NEEDED` | Public model performance leaderboard. |
| `/api/verification` | GET | `controller.SendEmailVerification`| EmailVerificationRateLimit, TurnstileCheck | `WEB_FALLBACK` | Requires Cloudflare Turnstile captcha. Best handled via web verification. |
| `/api/reset_password` | GET | `controller.SendPasswordResetEmail`| CriticalRateLimit, TurnstileCheck | `WEB_FALLBACK` | Turnstile-protected email reset trigger. |
| `/api/user/reset` | POST | `controller.ResetPassword` | CriticalRateLimit | `WEB_FALLBACK` | Password reset completion page. |

---

### 1.2 Authentication & Session Plane (`/api/user/*`, `/api/oauth/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/api/user/login` | POST | `controller.Login` | CriticalRateLimit, TurnstileCheck* | `NATIVE` | **Already implemented.** Primary mobile credentials auth with refresh cookie handling. |
| `/api/user/register` | POST | `controller.Register` | CriticalRateLimit, TurnstileCheck* | `NATIVE` | **Already implemented.** Mobile registration. |
| `/api/user/auth/refresh` | POST | `controller.RefreshAuth` | SessionCookieOriginGuard | `NATIVE` | **Already implemented.** Background session token refresh via `new_api_refresh` cookie. |
| `/api/user/auth/logout` | POST | `controller.AuthLogout` | SessionCookieOriginGuard | `NATIVE` | **Already implemented.** Session destruction and cookie clearance. |
| `/api/user/login/encryption-key` | GET | `controller.GetPasswordEncryptionKey` | DisableCache | `NATIVE` | Client-side password pre-hashing public key. |
| `/api/user/login/2fa` | POST | `controller.Verify2FALogin` | CriticalRateLimit | `NATIVE` | Multi-factor TOTP verification during login flow. |
| `/api/user/login/verify` | POST | `controller.VerifyLogin` | CriticalRateLimit | `NATIVE` | Email verification code login step. |
| `/api/user/sessions` | GET | `controller.GetLoginSessions` | `UserAuth()` | `NATIVE` | Lists active login sessions for multi-device management. |
| `/api/user/sessions/:sid`| DELETE | `controller.DeleteLoginSession` | `UserAuth()` | `NATIVE` | Terminates a specific remote session. |
| `/api/user/sessions/revoke-others` | POST | `controller.RevokeOtherLoginSessions` | `UserAuth()` | `NATIVE` | Revokes all sessions except current device. |
| `/api/oauth/state` | POST | `controller.GenerateOAuthCode` | TryUserAuth | `WEB_FALLBACK` | Generates secure OAuth state for external providers. |
| `/api/oauth/:provider` | GET | `controller.HandleOAuth` | TryUserAuth | `WEB_FALLBACK` | Standard OAuth (GitHub, Google, Discord). Opens in external browser. |
| `/api/user/passkey/*` | ANY | Passkey Handlers | `UserAuth()`, WebAuthn | `WEB_FALLBACK` | FIDO2 / WebAuthn browser ceremony. |
| `/api/user/2fa/*` | ANY | 2FA Handlers | `UserAuth()`, SecurityVerification | `WEB_FALLBACK` | TOTP setup, backup codes generation. Security-sensitive, requires step-up auth. |

---

### 1.3 User Self-Service & Credential Plane (`/api/user/self`, `/api/token/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/api/user/self` | GET | `controller.GetSelf` | `UserAuth()` | `NATIVE` | **Already implemented.** User profile, numeric ID, quota, group, and balance. |
| `/api/user/self` | PUT | `controller.UpdateSelf` | `UserAuth()`, CriticalRateLimit | `NATIVE` | Updates user display name or password. |
| `/api/user/self` | DELETE | `controller.DeleteSelf` | `UserAuth()` | `WEB_FALLBACK` | Account deletion with confirmation. Requires step-up auth. |
| `/api/user/groups` | GET | `controller.GetUserGroups` | Public / User | `NATIVE` | Available user groups and descriptions. |
| `/api/token/` | GET | `controller.GetAllTokens` | `UserAuth()` | `NATIVE` | **Already implemented.** Lists relay tokens (`p=1&page_size=20`). |
| `/api/token/` | POST | `controller.AddToken` | `UserAuth()` | `NATIVE` | **Already implemented.** Provisions dedicated per-installation relay token. |
| `/api/token/:id/key` | POST | `controller.GetTokenKey` | `UserAuth()`, CriticalRateLimit | `NATIVE` | **Already implemented.** Unmasks raw `sk-...` relay key for inference client. |
| `/api/token/:id` | DELETE | `controller.DeleteToken` | `UserAuth()` | `NATIVE` | **Already implemented.** Revokes mobile relay token on reset. |
| `/api/token/:id` | GET/PUT | `controller.GetToken` / `UpdateToken` | `UserAuth()` | `NOT_NEEDED` | Token editing (rename, quota caps). Handled automatically by provisioner. |
| `/api/token/batch*` | POST | `controller.DeleteTokenBatch` | `UserAuth()` | `NOT_NEEDED` | Bulk developer dashboard token management. |
| `/api/user/access_tokens/*` | ANY | Access Token Handlers | `UserAuth()` | `NOT_NEEDED` | Management plane scoped developer tokens (Web dashboard feature). |

---

### 1.4 BYOK Management Plane (`/api/user/providers/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/api/user/providers` | GET | `controller.ListUserProviders` | `UserAuth()`, DisableCache | `NATIVE` | **Already implemented.** Lists configured BYOK providers (masked credentials). |
| `/api/user/providers` | POST | `controller.CreateUserProvider` | `UserAuth()`, CriticalRateLimit | `NATIVE` | **Already implemented.** Securely adds AES-256-GCM encrypted provider key. |
| `/api/user/providers/:id` | PUT | `controller.UpdateUserProvider` | `UserAuth()`, CriticalRateLimit | `NATIVE` | **Already implemented.** Rotates provider API key or updates base URL. |
| `/api/user/providers/:id` | DELETE | `controller.DeleteUserProvider` | `UserAuth()` | `NATIVE` | **Already implemented.** Removes BYOK provider and purges encrypted secrets. |
| `/api/user/providers/:id/test`| POST | `controller.TestUserProvider` | `UserAuth()`, CriticalRateLimit | `NATIVE` | **Already implemented.** Verifies provider key with upstream AI service. |

---

### 1.5 Subscription & Quota Plane (`/api/subscription/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/api/subscription/plans` | GET | `controller.GetSubscriptionPlans` | `UserAuth()` | `NATIVE` | **Already implemented.** Lists commercial subscription catalog. |
| `/api/subscription/self` | GET | `controller.GetSubscriptionSelf` | `UserAuth()` | `NATIVE` | **Already implemented.** Active entitlement, plan title, expiry, and group. |
| `/api/subscription/store/products` | GET | `controller.GetStoreProductCatalog` | `UserAuth()` | `NATIVE` | **Already implemented.** Native store product metadata (Apple/Google). |
| `/api/subscription/apple/verify` | POST | `controller.VerifyAppleSubscription` | `UserAuth()`, CriticalRateLimit | `NATIVE` | **Already implemented.** StoreKit 2 JWS verification & settlement. |
| `/api/subscription/google/verify` | POST | `controller.VerifyGoogleSubscription` | `UserAuth()`, CriticalRateLimit | `NATIVE` | **Already implemented.** Google Play Billing token verification & settlement. |
| `/api/subscription/balance/pay` | POST | `controller.SubscriptionRequestBalancePay` | `UserAuth()`, CriticalRateLimit | `NATIVE` | Purchases subscription plan directly using account wallet quota. |
| `/api/subscription/self/preference` | PUT | `controller.UpdateSubscriptionPreference` | `UserAuth()` | `NATIVE` | Toggles subscription preferences. |
| `/api/subscription/stripe/pay` | POST | `controller.SubscriptionRequestStripePay` | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | Web Stripe subscription checkout (system/in-app browser). |
| `/api/subscription/epay/pay` | POST | `controller.SubscriptionRequestEpay` | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | Chinese Epay subscription gateway. |
| `/api/subscription/creem/pay` | POST | `controller.SubscriptionRequestCreemPay` | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | Creem payment gateway. |
| `/api/subscription/waffo-pancake/pay`| POST | `controller.SubscriptionRequestWaffoPancakePay` | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | Waffo Pancake gateway. |

---

### 1.6 Top-Up & Payment Plane (`/api/user/topup/*`, `/api/user/pay`, `/api/user/stripe/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/api/user/topup` | POST | `controller.TopUp` | `UserAuth()`, PaymentCompliance | `NATIVE` | **High-Value Candidate.** Redeems voucher / gift card redemption code for quota balance. |
| `/api/user/topup/info` | GET | `controller.GetTopUpInfo` | `UserAuth()` | `NATIVE` | **High-Value Candidate.** Returns enabled payment methods, min topup, rates, and discount brackets. |
| `/api/user/topup/self` | GET | `controller.GetUserTopUps` | `UserAuth()` | `NATIVE` | **High-Value Candidate.** Returns paginated transaction/top-up history for current user. |
| `/api/user/stripe/amount` | POST | `controller.RequestStripeAmount` | `UserAuth()` | `NATIVE` | Calculates exact Stripe billing charge for a requested quota amount. |
| `/api/user/stripe/pay` | POST | `controller.RequestStripePay` | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | **High-Value Candidate.** Generates Stripe Checkout Session URL; launches in browser. |
| `/api/user/pay` | POST | `controller.RequestEpay` | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | Epay top-up gateway. |
| `/api/user/amount` | POST | `controller.RequestAmount` | `UserAuth()` | `WEB_FALLBACK` | Epay amount calculation. |
| `/api/user/creem/pay` | POST | `controller.RequestCreemPay` | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | Creem top-up gateway. |
| `/api/user/waffo/*` | POST | Waffo Handlers | `UserAuth()`, CriticalRateLimit | `WEB_FALLBACK` | Waffo international gateway. |
| `/api/user/checkin` | GET/POST | `controller.GetCheckinStatus` / `DoCheckin` | `UserAuth()`, TurnstileCheck | `WEB_FALLBACK` | Daily check-in reward (Turnstile-protected). |
| `/api/user/aff` | GET | `controller.GetAffCode` | `UserAuth()` | `NOT_NEEDED` | Affiliate referral code. |
| `/api/user/aff_transfer` | POST | `controller.TransferAffQuota` | `UserAuth()`, UserCriticalRateLimit | `NOT_NEEDED` | Affiliate commission transfer. |

---

### 1.7 Usage, Logs & Analytics Plane (`/api/log/*`, `/api/data/*`, `/api/usage/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/api/log/self/stat` | GET | `controller.GetLogsSelfStat` | `UserAuth()` | `NATIVE` | **High-Value Candidate.** Summary statistics: total quota consumed, RPM, TPM. |
| `/api/log/self` | GET | `controller.GetUserLogs` | `UserAuth()` | `NATIVE` | Detailed request history: model, token count, latency, quota cost. |
| `/api/data/self` | GET | `controller.GetUserQuotaDates` | `UserAuth()` | `NATIVE` | Daily quota consumption breakdown (up to 30 days) for charting. |
| `/api/usage/token/` | GET | `controller.GetTokenUsage` | `TokenAuthReadOnly()` | `NATIVE` | Token-specific usage inspection. |
| `/api/log/self/search` | GET | `controller.SearchUserLogs` | `UserAuth()` | `DEPRECATED` | Explicitly marked `Deprecated: 已废弃` in backend source. |
| `/api/log/token` | GET | `controller.GetLogByKey` | `TokenAuthReadOnly()` | `NOT_NEEDED` | Token-level raw log querying. |

---

### 1.8 Inference Plane (`/v1/*`, `/v1beta/*`)

| Endpoint | Method | Source Handler | Auth / Middleware | Classification | Rationale & Mobile Strategy |
| :--- | :---: | :--- | :--- | :---: | :--- |
| `/v1/models` | GET | `controller.ListModels` | `TokenAuth()` | `NATIVE` | **Already implemented.** Model catalog discovery using relay token. |
| `/v1/chat/completions` | POST | `controller.Relay` | `TokenAuth()`, BYOKRouter, FreeTierQuota | `NATIVE` | **Already implemented.** Streaming chat completions with reasoning deltas & genuine cancellation. |
| `/v1/responses` | GET | `controller.ResponsesWebSocket` | `TokenAuth()` | `NATIVE` | Realtime Responses WebSocket API (advanced future capability). |
| `/v1/realtime` | GET | `controller.Relay` (Realtime) | `TokenAuth()` | `NATIVE` | OpenAI Realtime voice/multimodal API (future capability). |
| `/pg/chat/completions` | POST | `controller.Playground` | `UserAuth()`, Distribute | `NOT_NEEDED` | Web dashboard playground tester. |

---

### 1.9 Server-Only Webhooks & Callbacks (`SERVER_ONLY`)

All webhook endpoints MUST remain strictly server-to-server. They are never called by the mobile client:

| Endpoint | Method | Verification Mechanism | Classification | Security Policy |
| :--- | :---: | :--- | :---: | :--- |
| `/api/stripe/webhook` | POST | `Stripe-Signature` HMAC-SHA256 | `SERVER_ONLY` | Verified with `setting.StripeWebhookSecret`. Mobile must never call directly. |
| `/api/subscription/apple/webhook` | POST | Apple App Store Server Notifications V2 JWS | `SERVER_ONLY` | Verified with Apple Root CA x5c cert chain. Server authoritative. |
| `/api/subscription/google/webhook` | POST | Google Cloud Pub/Sub RTDN Token / JWT | `SERVER_ONLY` | Verified with Pub/Sub verification token. Server authoritative. |
| `/api/subscription/epay/notify` | POST/GET| MD5 signature check | `SERVER_ONLY` | Epay asynchronous payment notification. |
| `/api/subscription/epay/return` | GET/POST| MD5 signature check | `SERVER_ONLY` | Epay synchronous browser return. |
| `/api/creem/webhook` | POST | Creem signature verification | `SERVER_ONLY` | Third-party payment callback. |
| `/api/waffo/webhook` | POST | Waffo signature verification | `SERVER_ONLY` | Third-party payment callback. |
| `/api/waffo-pancake/webhook/:env` | POST | Waffo Pancake signature | `SERVER_ONLY` | Third-party payment callback. |

---

### 1.10 Admin & System Management Plane (`ADMIN_ONLY`)

All administration routes are restricted to administrators (`AdminAuth`) or root operators (`RootAuth`) and MUST NEVER be exposed in consumer mobile UI:

| Route Group | Source Handlers | Auth Middleware | Classification | Description |
| :--- | :--- | :--- | :---: | :--- |
| `/api/user/` (Admin) | `GetAllUsers`, `CreateUser`, `ManageUser`, etc. | `AdminAuth()` | `ADMIN_ONLY` | Multi-user administration and account suspension. |
| `/api/channel/*` | `GetAllChannels`, `AddChannel`, `TestChannel`, etc. | `AdminAuth()` | `ADMIN_ONLY` | Upstream AI channel management, API keys, and models. |
| `/api/subscription/admin/*`| `AdminListSubscriptionPlans`, `AdminBindSubscription` | `AdminAuth()` | `ADMIN_ONLY` | Subscription plan pricing, manual grants, and overrides. |
| `/api/budget/*` | `AdminListBudgetRules`, `AdminCreateBudgetRule` | `AdminAuth()` | `ADMIN_ONLY` | Enterprise quota budgeting rules. |
| `/api/option/*` | `GetOptions`, `UpdateOption`, `UpdateRequestPolicy` | `RootAuth()` | `ADMIN_ONLY` | Server runtime parameters, encryption secrets, model ratios. |
| `/api/redemption/*` | `GetAllRedemptions`, `AddRedemption`, etc. | `AdminAuth()` | `ADMIN_ONLY` | Voucher / gift card code generation and lifecycle management. |
| `/api/models/*` | `SyncUpstreamModels`, `CreateModelMeta`, etc. | `AdminAuth()` | `ADMIN_ONLY` | Global model pricing and upstream metadata synchronization. |
| `/api/deployments/*` | `CreateDeployment`, `ListDeploymentContainers` | `AdminAuth()` | `ADMIN_ONLY` | GPU container and model deployment infrastructure. |
| `/api/vendors/*` | `CreateVendorMeta`, `ApplyVendorOperation` | `AdminAuth()` | `ADMIN_ONLY` | Model vendor management. |
| `/api/plugin/task/*` | `UploadTaskPlugin`, `ActivateTaskPlugin` | `RootAuth()` | `ADMIN_ONLY` | Server-side task plugin management. |
| `/api/system-task/*` | `ListSystemTasks`, `CreateLogCleanupSystemTask` | `RootAuth()` | `ADMIN_ONLY` | Server background jobs and maintenance tasks. |
| `/api/system-info/*` | `ListSystemInstances`, `DeleteStaleSystemInstance` | `RootAuth()` | `ADMIN_ONLY` | Multi-instance cluster node discovery and health. |
| `/api/performance/*` | `GetPerformanceStats`, `ForceGC`, `ClearDiskCache`| `RootAuth()` | `ADMIN_ONLY` | Server memory profiling, GC triggers, and cache management. |
| `/api/ratio_sync/*` | `FetchUpstreamRatios` | `RootAuth()` | `ADMIN_ONLY` | Upstream pricing ratio synchronizers. |
| `/api/authz/*` | `GetPermissionCatalog` | `AdminAuth()` | `ADMIN_ONLY` | RBAC role-based authorization rules. |

---

## 2. High-Value Mobile Integrations Identified

Based on this comprehensive audit, four immediately viable, high-value consumer integrations have been identified for Flutter implementation without backend redesign:

1. **Voucher / Redemption Code Top-Up (`POST /api/user/topup`)**:
   - Allows users to enter promotional vouchers or gift cards directly in the app.
   - 100% backend ready. Uses `controller.TopUp` and `model.Redeem`.
2. **Top-Up Rates & Wallet Transaction History (`GET /api/user/topup/info`, `GET /api/user/topup/self`)**:
   - Provides users with transparent visibility of current billing rates and past top-up records.
   - 100% backend ready. Uses `controller.GetTopUpInfo` and `controller.GetUserTopUps`.
3. **External Stripe Checkout Launcher (`POST /api/user/stripe/pay`)**:
   - Allows users desiring external web/card top-up to initiate a hosted Stripe checkout session in the system browser with secure return callbacks.
   - 100% backend ready.
4. **Usage Analytics & Consumption Statistics (`GET /api/log/self/stat`, `GET /api/data/self`)**:
   - Displays real-time quota usage, request rates (RPM/TPM), and daily consumption charts.
   - 100% backend ready.
