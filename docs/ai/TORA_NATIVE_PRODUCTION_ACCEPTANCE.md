# TORA NATIVE PRODUCTION ACCEPTANCE PROTOCOL & PRIVACY VERIFICATION
**Live Verification Standards, Network Tracing, History Opt-In & Quality Gates**
**Author:** Tora QA & Security Engineering | **Status:** SPECIFICATION APPROVED | **Date:** 2026-10-08

---

## 1. Network Privacy & Zero-Upload Proof (Section 41)

When a merchant or consumer runs **Tora Native Web Tools** (e.g. Background Remove, Upscale), the user interface displays:
> *"Processed on your device only — your photos never leave your browser."*

To protect user trust and legal compliance, the following network trace rules are strictly enforced:

### Permitted Outgoing Requests:
1. `POST /api/studio/native/quote`: Tool pricing and model digest metadata.
2. `POST /api/studio/native/ticket`: Atomic quota authorization and ticket issuance.
3. `GET /models/<tool>/<sha256>.onnx`: Content-addressed model weights (if not cached in IndexedDB).
4. `POST /api/studio/native/complete`: Telemetry payload containing ONLY:
   - `ticket_id`
   - `output_asset_hash` (cryptographic SHA-256 string)
   - `client_execution_ms` (integer runtime duration)
   - `client_device_class` (browser/GPU identifier string)

### Strictly Forbidden:
- **Zero Raw Pixel Uploads**: No base64, multipart form-data, or binary blobs of the input photo or output cutout may be transmitted to any server during inference.
- If raw source image transmission is detected, the UI is prohibited from displaying the device-only privacy guarantee.

---

## 2. Save to History: Explicit Opt-In Architecture (Section 42 & 74)

By default, all native outputs exist purely in browser memory as HTML5 `Canvas` or `Blob` objects. The user can download the output directly to their device with zero cloud storage footprint.

### Optional "Save to Tora History":
If the user clicks **"Save to History"**:
1. The browser uploads ONLY the finalized output asset (`POST /api/studio/upload`).
2. The server records a `StudioAsset` bound to the authenticated `user_id`.
3. The asset appears in the user's permanent Tora Studio gallery.
4. Access control is enforced; unauthorized users cannot access another user's saved outputs.

---

## 3. Capability Preflight & No Auto-Fallback Rule (Sections 34, 35, 36)

1. **Preflight Before Charge**:
   The frontend runs a non-invasive capability probe (`navigator.gpu` and WebAssembly SIMD support) before invoking `POST /api/studio/native/ticket`. If the device is incapable, the user is never charged.
2. **Strict Prohibition on Post-Charge Auto-Fallback**:
   If a client has paid 2 Tora Credits for a `NATIVE_BROWSER` ticket and the browser subsequently crashes or fails, the server **must NEVER automatically route the job to WaveSpeed or fal.ai**.
   - *Rationale*: Silently calling external cloud APIs incurs provider COGS and destroys unit economics.
   - *Allowed Behavior*: The UI offers same-ticket fair retries locally. If the user desires cloud processing, the UI presents an explicit dialog with a new quote (e.g. 10 Credits for WaveSpeed Relay) before proceeding.

---

## 4. Production Canary Acceptance Checklist

Before any tool is transitioned to `ACTIVE_NATIVE`:

- [x] **Prepaid Wallet Enforcement**: Quota deducted atomically at ticket activation.
- [x] **Exploit Verification**: Simulating `/complete` suppression leaves wallet charged and prevents auto-refund.
- [x] **Same-Ticket Retry**: Runtime failures allow zero-credit retries within 30 minutes.
- [x] **Cryptographic Token Verification**: HMAC-SHA256 signature validates model hash, input hash, and user ID.
- [x] **Model Integrity Check**: Browser Web Crypto calculates SHA-256 before session initialization.
- [x] **Clean-Room Verification**: Confirmed zero lines copied from GPL-2.0 `web-realesrgan`.
- [x] **Marketplace Product Pack**: Deterministic Go pipeline generates Shopee, Lazada, IG, and Story presets with compliant white backgrounds and contact shadows.
