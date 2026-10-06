# Tora AI — Observability & Logging Security Audit

This document records the results of the observability and logging audit for Tora AI across backend and mobile services.

## Audit Scope
- Management Plane API (`/api/*`)
- Inference Plane API (`/v1/*`)
- Native Store Verification (`/api/subscription/*`)
- BYOK Vault & Encryption Operations (`/api/user/providers/*`)
- Mobile API Client (`ToraApiClient`)
- Mobile Inference Client (`ToraInferenceClient`)
- Mobile Native Billing Service (`NativeBillingService`)

---

## 1. Zero-Leakage Invariants Verified

| Surface | Logging Target | Protection Mechanism | Status |
| :--- | :--- | :--- | :--- |
| **Store Purchase Verification** | `signed_transaction_info`, `purchase_token` | Excluded from server logs; only error classification codes returned | **VERIFIED** |
| **Store Account Tokens** | SHA-256 hash input (`userId`, `storeAccountId`) | Only 32-byte hexadecimal hash digest is transmitted or compared; raw token never logged | **VERIFIED** |
| **BYOK Credentials** | `api_key` | Plaintext key is encrypted in memory via AES-256-GCM; only masked string (`****cdef`) returned | **VERIFIED** |
| **Mobile API Client** | `Authorization` headers, `new_api_refresh` cookies | `_log(method, path, status, code)` only logs path and HTTP status code | **VERIFIED** |
| **Mobile Inference Client** | Prompts, completions, streaming tokens, relay keys | Zero prompt/completion logging; sockets cleanly terminate on disconnect | **VERIFIED** |
| **Native Purchase Engine** | StoreKit transaction JWS, Play Billing purchase tokens | Zero logging of receipt payload strings in debug or release builds | **VERIFIED** |

---

## 2. Audit Conclusion

All logging pathways adhere to strict zero-leakage security boundaries. No credentials, tokens, receipts, or personal payload data are written to standard logs, telemetry, or debug streams.
