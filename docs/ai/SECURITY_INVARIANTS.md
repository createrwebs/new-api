# Tora AI — Security Invariants

These are non-negotiable unless a new security review explicitly replaces them.

## General

1. Mobile is untrusted.
2. Server is authoritative for identity, entitlement, billing, quota, provider ownership, and settlement.
3. Secrets are never committed to source control.
4. Sensitive credentials and raw payment proofs must not be logged.
5. Environment boundaries (production/staging/etc.) must be preserved.

## Authentication

- Management and inference credentials remain separated.
- Relay tokens are per-installation and environment-scoped.
- Logout on one device must not revoke unrelated device credentials unless policy explicitly requires it.
- Cross-account callbacks must never be silently rebound to the current user.

## Managed Chat

- Managed mode must contain zero BYOK/provider hints.
- Chat POST is not automatically retried in a way that can double bill.
- Stream cancellation must cancel the real upstream request context.

## BYOK

- Receive plaintext key only for create/rotate/test use.
- Encrypt server-side with AES-256-GCM.
- Persist ciphertext only.
- Never return stored plaintext.
- Never persist plaintext in logs/cache.
- BYOK failure must not silently fall back to managed credentials.
- SSRF / DNS rebinding / redirect protections remain enforced.

## Native Store Billing

A store-side purchase event is never entitlement by itself.

Required order:

```text
store proof
-> backend verification against authoritative store data
-> account/environment/product validation
-> transactional settlement
-> `/api/subscription/self`
-> mobile reflects entitlement
```

Replay, duplicate callbacks, concurrent verification, webhook retries, and restore flows must be idempotent and financially safe.

Cross-account ownership is protected by both backend binding records and native account-binding signals for new purchases.

### Account Binding Hardening

Phase 7G-B-R introduced store account tokens:

- Apple: `appAccountToken`
- Google: obfuscated account ID / authoritative external account ID

The current implementation reportedly derives a deterministic UUIDv5 from `tora:<env>:user:<userId>`.

**Security note:** this derivation can be suitable as a stable binding identifier, but it must not be described as a secret or as strongly non-enumerable privacy protection when the namespace and numeric user IDs are predictable. Treat it as an account-binding identifier, not a credential. If stronger privacy is required, prefer an opaque random per-account UUID or a server-issued stable binding identifier.

Do not weaken strict account-token enforcement for new purchases merely to make legacy tests easier.

## Payment / Stripe

- Reuse existing New-API Stripe / TopUp stack before creating a new billing system.
- Stripe webhook endpoints are server-only.
- Webhook authenticity must be verified before mutation.
- Payment completion must be idempotent.
- QR display, checkout creation, or client-side success is not proof of settlement.
- PromptPay is one-time/top-up unless the actual provider contract proves otherwise.

## Web / Browser Fallback

- HTTPS only in production.
- Use trusted-domain allowlists.
- Do not inject long-lived auth tokens into URLs or JavaScript.
- Avoid unrestricted JavaScript bridges.
- Validate return/deep-link origins and schemes.
- Isolate/clear web sessions appropriately on logout/account/environment changes.

## Review Gate

Critical/High findings and release-blocking Medium findings must be remediated before release-candidate status.
