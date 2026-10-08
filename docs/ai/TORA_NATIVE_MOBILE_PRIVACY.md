# Tora Native Mobile Privacy & Zero Data Egress Architecture

**Identifier:** `TORA-PRIVACY-N3-MOBILE`  
**Date:** 2026-10-08  
**Scope:** Privacy verification, local image processing boundaries, and metadata telemetry minimization for mobile devices.  

---

## 1. Zero-Egress Privacy Invariant

In standard cloud AI photo editing applications, the user's personal photographs (including faces, private documents, private product prototypes) are transmitted over the internet to remote GPU servers.

**Tora Native Mobile Invariant:**
> **Source photographs never leave the user's mobile device.** All neural network preprocessing, tensor normalization, weight execution, and alpha matting run entirely on local silicon.

---

## 2. Network Payload Audit: What Travels Across the Wire

### 2.1 Ticket Creation (`POST /api/studio/native/ticket`)
The client sends **only metadata and cryptographic hashes**:
```json
{
  "tool_id": "background-remove",
  "execution_class": "NATIVE_MOBILE",
  "inputs": {
    "tool": "background-remove",
    "width": 1920,
    "height": 1080,
    "input_sha256": "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca7"
  },
  "client_device_class": "ios_coreml_neural_engine",
  "idempotency_key": "idem_1700000000_abc"
}
```
- **Image Bytes Transmitted**: `0 bytes`.
- **Private Data Leak**: `0 bytes`.

### 2.2 Completion Report (`POST /api/studio/native/complete`)
Upon finishing local inference, the client reports **only settlement proof**:
```json
{
  "ticket_id": "tkt_1700000000_mobile01",
  "output_asset_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "client_execution_ms": 412,
  "client_device_class": "ios_coreml_neural_engine"
}
```
- **Image Bytes Transmitted**: `0 bytes`.

### 2.3 Product Factory Handoff
In the hybrid Product Factory mode:
- The user's original background (which might contain personal bedroom, office, or factory environment) is **stripped locally on-device**.
- Only the clean, transparent object cutout (`cutout.png`) is transmitted to `/api/studio/native/product-pack` for multi-platform canvas resizing.
- Raw background context is permanently discarded on-device.

---

## 3. Regulatory Compliance & Enterprise Readiness

- **GDPR / CCPA**: Complies by design since biometric and environmental pixels are never ingested by server infrastructure.
- **Data Sovereignty**: Meets government and enterprise confidentiality guidelines where external upload of trade or employee photos is restricted.
