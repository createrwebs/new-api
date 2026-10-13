# =====================================================================
# TORA STUDIO — QUEUE 2J PREFLIGHT
# KIE PROVIDER #2 READINESS & ACTIVATION SPECIFICATION
# =====================================================================

## 1. KIE_MODEL_CANDIDATE
- **Model Name**: FLUX.1 Flex Text-to-Image
- **Provider**: KIE.ai (Market API)
- **Candidate Justification**: High fidelity, reliable upstream availability in KIE model catalog, suitable as secondary provider for `image-generate` tool.

---

## 2. LOGICAL_TOOL
- **Logical Tool ID**: `image-generate`
- **Target Quality Tier**: `QUALITY` (or secondary fallback for `FAST`)
- **Capability**: Text-to-Image generation

---

## 3. OFFICIAL_MODEL_ID
- **Official Model Slug**: `flux-2/flex-text-to-image`
- **Endpoint Pattern**: `POST https://api.kie.ai/api/v1/jobs`
- **Polling Endpoint**: `GET https://api.kie.ai/api/v1/jobs/{job_id}`
- **Authentication**: `Authorization: Bearer <KIE_API_KEY>`

---

## 4. INPUT_SCHEMA
```json
{
  "model": "flux-2/flex-text-to-image",
  "input": {
    "prompt": "string (required, max 2000 chars)",
    "aspect_ratio": "1:1 | 16:9 | 9:16 | 4:3 | 3:4",
    "num_outputs": 1,
    "seed": 0
  }
}
```

---

## 5. OUTPUT_SCHEMA
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "job_kie_12345678",
    "status": "completed",
    "outputs": [
      "https://cdn.kie.ai/outputs/job_kie_12345678/result.jpg"
    ],
    "usage": {
      "cost_usd": 0.0040
    }
  }
}
```

---

## 6. COST_SOURCE
- **Source**: KIE Market Official Pricing Tier
- **Strategy**: Dynamic quote lookup via model pricing endpoint or verified catalog snapshot.

---

## 7. ESTIMATED_COGS
- **Provider Base Cost**: **$0.0040 USD** per generation (1024x1024, 1 image)

---

## 8. PROPOSED_TORA_CREDITS
- **Proposed Price**: **6 Tora Credits**
- **Canonical Quota Equivalent**: 6,000 Quota
- **Retail Sell USD Equivalent**:
  $$6 \times \$0.0020 = \$0.0120\text{ USD}$$

---

## 9. ESTIMATED_MARGIN
$$\text{Gross Profit} = \$0.0120 - \$0.0040 = \$0.0080\text{ USD}$$
$$\text{Gross Margin} = \frac{\$0.0080}{\$0.0120} = 66.67\%$$
- **Verdict**: Complies with and exceeds the mandatory $\ge 60.0\%$ gross margin floor.

---

## 10. CALLBACK_SECURITY
- Asynchronous notifications support `X-KIE-Signature` HMAC verification.
- SSRF-protected callback URLs.
- Poller failover fallback using bounded exponential backoff.

---

## 11. ROUTE_STATE
- **Current Database Route ID**: `kie-flex-text` (or `kie-flux-schnell` migrated to `flux-2/flex-text-to-image`)
- **Current State**: `CONTRACT_VERIFIED` / `CREDENTIAL_REQUIRED`
- **Activation Gate**: Transitions to `READY_FOR_CANARY` once `KIE_API_KEY` is present, and to `ACTIVE` upon single real-money canary passage.

---

## 12. KEY_REQUIRED
- **Environment Variable Name**: `KIE_API_KEY`
- **Status**: Not currently set (`OPERATOR_BLOCKED`).
