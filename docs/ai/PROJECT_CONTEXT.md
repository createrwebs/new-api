# Tora AI — Project Context

## Product

**Tora AI** is a Flutter mobile AI client backed directly by a modified New-API/Tora backend.

Primary mobile base:

- LumenFlow (Flutter/Dart, MIT)
- local working tree historically referenced as `scratch/LumenFlow`

Backend:

- `/Users/noppanan/new-api`
- branch historically referenced as `feat/formobile`

Oriveo was used only as architectural reference because of AGPL licensing; its code/assets are not to be copied into Tora AI.

## Core Product Modes

### Managed

The backend selects system/provider channels, applies entitlement, quota, and billing rules, and sends inference to upstream AI providers.

### BYOK

The user provides a provider key once. The backend encrypts it server-side and never returns stored plaintext. Inference still flows through Tora/New-API.

BYOK is independent from paid subscription. A free/default account may use BYOK subject to BYOK controls.

## Mobile / Backend Trust Model

The mobile app is never authoritative for:

- user identity
- role
- provider ownership
- BYOK billing state
- subscription entitlement
- quota
- pricing
- billing mode
- payment settlement

Conversations are local-first for the initial release and partitioned by backend user ID and environment. Account/subscription/BYOK state is server-backed.

## Auth & Credential Planes

Management and inference credentials are intentionally separated.

Management plane:

- `/api/*`
- access session + refresh/session mechanism

Inference plane:

- `/v1/*`
- persistent device relay token (`sk-...`)

Mobile credentials are stored only in secure storage.

## Inference Contract

Managed inference:

- `GET /v1/models`
- `POST /v1/chat/completions`
- no BYOK/provider hints

BYOK inference:

- same inference pipeline
- canonical provider routing via `X-Provider`
- no fallback to managed credentials if BYOK fails

POST chat must not be automatically retried in a way that could double bill. GET model catalog requests may use safe retries.

## BYOK Security Baseline

Server-side BYOK protections include:

- AES-256-GCM encryption at rest
- request-scoped decryption
- no stored plaintext return path
- SSRF protection
- DNS rebinding defense
- redirect restrictions
- rate limiting
- concurrency limits
- ownership isolation
- deletion cleanup

Required production secrets include stable, identical high-entropy values across nodes:

- `SESSION_SECRET`
- `CRYPTO_SECRET`
- `BYOK_ENCRYPTION_KEY`

## Subscription & Billing Context

There are two billing families:

### Native mobile subscription

- Apple StoreKit / App Store Server verification
- Google Play Billing / Android Publisher verification
- server-authoritative entitlement only

### Existing New-API web/external billing

New-API already contains TopUp / Stripe payment infrastructure and Stripe webhook support. Tora AI should discover and reuse the existing implementation before creating new payment code.

For Thailand, the intended launch approach is:

- Stripe card for web/external recurring billing where applicable
- Stripe PromptPay for one-time/top-up flows if existing Tora/New-API configuration supports it
- TrueMoney direct integration deferred unless existing code or real demand justifies it

For native store-distributed digital subscriptions, preserve Apple/Google native billing rather than replacing it with a Stripe WebView.

## API-First / Web-Fallback Strategy

Tora AI should maximize safe reuse of New-API:

```text
Native API where valuable
-> system/in-app browser for hosted provider flows
-> trusted Tora web fallback when native integration has little value
```

Do not blindly expose every New-API endpoint to the consumer app.

Admin/server-only capabilities remain excluded from normal mobile UX.

External providers such as Stripe/OAuth should prefer a system or in-app browser over a raw embedded WebView. Embedded WebView is a fallback for trusted Tora-owned pages only and must use a strict allowlist.

Never put management tokens, refresh tokens, relay tokens, BYOK keys, receipts, purchase tokens, or long-lived auth credentials in URLs.
