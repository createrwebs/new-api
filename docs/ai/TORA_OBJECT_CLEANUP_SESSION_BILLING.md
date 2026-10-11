# Tora Studio Object Cleanup Interactive Session Billing Architecture
**Queue:** `QUEUE N4A`  
**Milestone:** Interactive Mask Editing & Session Billing Specification  
**Timestamp:** 2026-10-08T20:10:00+07:00  

---

## 1. Problem Statement: Brush-Stroke vs Session Billing

In conventional interactive photo retouching, billing per brush-stroke or per inpainting preview creates an adversarial user experience:
- An artist or e-commerce merchant making 15 incremental stroke corrections would be charged 15 separate times.
- Users hesitate to refine masks out of fear of credit consumption.
- If an automated inpainting step produces an imperfect edge, charging for the retry causes customer dissatisfaction and refund disputes.

**The Solution: Tora Interactive Cleanup Session Architecture.**
Users purchase a bounded **Interactive Cleanup Session** for **3 Tora Credits** ($0.006 reference value).

---

## 2. Session Lifecycle & Economics

| Attribute | Canonical Value | Rationale |
|:---|:---|:---|
| **Base Price** | **3 Tora Credits** (3,000 Quota) | Equivalent to 2X Upscale; maintains unified Tora Credits ($0.002/credit). |
| **Active Session Window** | **30 Minutes** (`expires_at = now + 30m`) | Ample time for iterative retouching, masking, and previewing. |
| **Max High-Res Exports** | **5 Exports** per session ticket | Allows exporting at different resolutions or slight stroke variants. |
| **Interactive Previews** | **Unlimited** (Client-side / Native device) | On-device previewing incurs zero server cost. |
| **Image Binding** | Strictly bound to `source_hash` (SHA-256) | Prevents reusing one ticket across multiple product photos. |
| **Fair Zero-Credit Retry** | Built-in | Retrying or exporting another variant within 30 minutes incurs 0 additional credits. |

---

## 3. Endpoints & API Contract

### 3.1 Start Interactive Session (`POST /api/studio/native/object-cleanup/session`)
- **Request Body**:
  ```json
  {
    "source_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "width": 1024,
    "height": 1024,
    "image_format": "jpeg"
  }
  ```
- **Billing Action**: Deducts 3,000 quota (3 Credits) from `users.quota`.
- **Response Body**:
  ```json
  {
    "session_id": "clean-session-9f3a1b2c4e",
    "ticket_id": "TKT-CLEAN-587123",
    "source_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "credits_deducted": 3,
    "remaining_quota": 997000,
    "expires_at": "2026-10-08T20:40:00Z",
    "max_exports": 5,
    "exports_used": 0
  }
  ```

### 3.2 Validate Mask Quality (`POST /api/studio/native/object-cleanup/session/validate`)
- **Purpose**: Validates mask metrics prior to processing.
- **Constraints**:
  - Rejects empty masks ($0\text{ px}$ covered $\to$ `EMPTY_MASK`).
  - Flags and rejects masks covering $>50\%$ of image area (`MASK_OVERSIZED`).
  - Informs user if mask exceeds optimal small-object threshold ($>15\%$ coverage).

### 3.3 Export Inpainted Result (`POST /api/studio/native/object-cleanup/session/export`)
- **Request Body**:
  ```json
  {
    "session_id": "clean-session-9f3a1b2c4e",
    "source_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "mask_data": "data:image/png;base64,...",
    "algorithm": "TELEA"
  }
  ```
- **Verification**:
  - Validates `session_id` exists in database and is not expired (`expires_at > now`).
  - Verifies `source_hash` matches session ticket exactly.
  - Verifies `exports_used < max_exports`. Increments `exports_used`.
- **Cost**: **0 Additional Credits**.

---

## 4. Security & Abuse Prevention

1. **Source Hash Tampering Protection**:
   - The session ticket binds immutably to the `source_hash`.
   - If an attacker sends a valid session ID with a different image's SHA-256, the server rejects the request with HTTP 400 (`source_hash mismatch with session ticket`).
2. **Quota Race Protection**:
   - Quota pre-deduction occurs inside an atomic GORM database transaction.
   - Sessions expired past 30 minutes are marked `EXPIRED` and cannot be revived.
3. **Export Rate-Limiting**:
   - Hard cap of 5 exports per ticket prevents using an interactive session as an inpainting relay for other tasks.
