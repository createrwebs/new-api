# Tora AI — Runtime Configuration Preflight Matrix

This document defines the authoritative runtime configuration contract and environment variable preflight matrix across Local, Staging, and Production environments.

## 1. Environment Matrix Overview

| Variable | Description | Local / E2E | Staging | Production |
| :--- | :--- | :--- | :--- | :--- |
| `PORT` | Listening HTTP port | `3005` (E2E) / `3000` | `3000` | `3000` |
| `SERVER_ADDRESS` | Public canonical API base URL | `http://localhost:3005` | `https://staging-api.tora.ai` | `https://api.tora.ai` |
| `GIN_MODE` | Gin framework mode | `release` | `release` | `release` |
| `SQL_DSN` | Relational database DSN | `""` (uses SQLite) | `postgres://user:pass@host/db` | `postgres://user:pass@host/db` |
| `SQLITE_PATH` | Path to SQLite DB | `data/e2e/tora-e2e.db` | *Unset* | *Unset* |
| `REDIS_CONN_STRING` | Redis cache connection string | `""` (in-memory) | `redis://host:6379/0` | `redis://host:6379/0` |
| `MEMORY_CACHE_ENABLED` | Channel / option memory cache | `false` (E2E) | `true` | `true` |
| `SESSION_SECRET` | Web session encryption key | Deterministic 32B | High-entropy 32B+ | High-entropy 32B+ |
| `CRYPTO_SECRET` | General payload cryptographic secret | Deterministic 32B | High-entropy 32B+ | High-entropy 32B+ |
| `BYOK_ENCRYPTION_KEY` | AES-256-GCM master BYOK key | 64-hex test key | 64-hex staging key | 64-hex production key |
| `SESSION_COOKIE_SECURE` | HttpOnly + Secure cookie flag | `false` | `true` | `true` |
| `CRITICAL_RATE_LIMIT_ENABLE` | Auth & sensitive route rate limits | `false` (E2E) | `true` | `true` |
| `GLOBAL_API_RATE_LIMIT` | Global requests / min | `10000` | `1000` | `1000` |
| `GLOBAL_WEB_RATE_LIMIT` | Global web requests / min | `10000` | `1000` | `1000` |
| `APPLE_BUNDLE_ID` | Apple iOS App Bundle ID | `com.saascover.tora` | `com.saascover.tora` | `com.saascover.tora` |
| `APPLE_KEY_ID` | App Store Connect API Key ID | *Optional* | `XXXXXXXXXX` | `XXXXXXXXXX` |
| `APPLE_ISSUER_ID` | App Store Connect Issuer UUID | *Optional* | `uuid-v4` | `uuid-v4` |
| `APPLE_PRIVATE_KEY_PATH` | Path to StoreKit 2 Private Key (`.p8`) | *Optional* | `/secrets/apple.p8` | `/secrets/apple.p8` |
| `APPLE_ALLOW_SANDBOX` | Allow Apple Sandbox verification | `true` | `true` | `false` |
| `GOOGLE_PACKAGE_NAME` | Android Application ID | `com.saascover.tora` | `com.saascover.tora` | `com.saascover.tora` |
| `GOOGLE_SERVICE_ACCOUNT_JSON_PATH` | Google Service Account JSON | *Optional* | `/secrets/google.json` | `/secrets/google.json` |
| `GOOGLE_ALLOW_TEST_PURCHASE` | Allow Google Play test purchases | `true` | `true` | `false` |
| `PRO_MONTHLY_PRODUCT_ID` | Internal / Apple Product ID | `com.saascover.tora.pro.monthly` | `com.saascover.tora.pro.monthly` | `com.saascover.tora.pro.monthly` |
| `PRO_MONTHLY_GOOGLE_PRODUCT_ID` | Google Play Subscription Base Plan | `tora_pro` | `tora_pro` | `tora_pro` |
| `PRO_MONTHLY_QUOTA_DELTA` | Monthly credited quota units | `2000000` | `2000000` | `2000000` |
| `PRO_MONTHLY_GROUP` | Model routing user group | `pro` | `pro` | `pro` |
| `STRIPE_API_SECRET` | Stripe secret key (`sk_...`) | *Optional* | `sk_test_...` | `sk_live_...` |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook secret (`whsec_...`) | *Optional* | `whsec_test_...` | `whsec_live_...` |
| `STORE_VERIFICATION_SIMULATION` | Allow simulated store verification | `true` | `false` | `false` |

---

## 2. Security Invariants Enforced by Preflight

1. **Zero Secret Leakage**:
   - `APPLE_PRIVATE_KEY_PATH` and `GOOGLE_SERVICE_ACCOUNT_JSON_PATH` contain filesystem paths to secret files, never raw secret strings in command lines.
   - Secret file paths must never be logged or echoed in HTTP responses.
2. **Production Sandbox Prohibition**:
   - In Production (`SERVER_ADDRESS="https://api.tora.ai"`), `APPLE_ALLOW_SANDBOX` and `GOOGLE_ALLOW_TEST_PURCHASE` MUST be `false`. Sandbox transactions will be rejected with HTTP 400 (`STORE_ENVIRONMENT_MISMATCH`).
3. **Session Cookie Isolation**:
   - In Staging and Production, `SESSION_COOKIE_SECURE=true` ensures refresh token cookies are never transmitted over unencrypted HTTP.
4. **Authoritative Backend Pricing**:
   - Client applications never specify quotas or amounts. The backend independently maps `com.saascover.tora.pro.monthly` and `tora_pro` to 2,000,000 units and group `pro`.
