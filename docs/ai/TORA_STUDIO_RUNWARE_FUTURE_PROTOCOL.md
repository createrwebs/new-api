# =====================================================================
# TORA STUDIO — RUNWARE FUTURE PROTOCOL ARCHITECTURAL RESEARCH
# PROTOCOL SHAPE, DYNAMIC COST RETURN, MODEL CATALOG & RELAY FIT
# =====================================================================

> **Status**: RESEARCH ONLY — ZERO PRODUCTION CHANGES  
> **Target**: Future Provider Protocol Candidate #4 (Post-WaveSpeed, Post-KIE, Post-Fal)  
> **Date**: 2026-10-08

---

## 1. Executive Summary

Runware (runware.ai) offers high-throughput, low-latency generative media inference via a WebSocket-first and HTTP REST architecture. This document evaluates Runware's architectural compatibility with the Tora Generic Media Relay Core (`MediaProtocolAdapter`).

---

## 2. Protocol Shape & Authentication

### 2.1 Communication Protocol
- **Transport**: Supports both WebSocket (`wss://ws-api.runware.ai/v1`) and HTTP REST (`POST https://api.runware.ai/v1`).
- **Connection Model for Relay**:
  - While WebSocket offers sub-second connection reuse for high-frequency interactive apps, Tora Media Relay operates as an asynchronous task broker.
  - The HTTP REST endpoint (`POST https://api.runware.ai/v1`) fits cleanly into `MediaProtocolAdapter.Submit` without requiring persistent socket daemon infrastructure.
- **Authentication**:
  - Header: `Authorization: Bearer <RUNWARE_API_KEY>` or API Key in request body `[{"taskType": "authentication", "apiKey": "..."}]`.
  - Secret Environment Name: `RUNWARE_API_KEY`.

---

## 3. Request / Response Contract

### 3.1 Task Execution Request (Batch Array Contract)
Runware accepts an array of tasks in a single JSON payload:
```json
[
  {
    "taskType": "imageInference",
    "taskUUID": "uuid-v4-generated-by-client",
    "positivePrompt": "high quality studio portrait",
    "negativePrompt": "blurry, low quality",
    "width": 1024,
    "height": 1024,
    "model": "runware:100@1",
    "steps": 28,
    "numberResults": 1,
    "outputType": "URL",
    "outputFormat": "WEBP"
  }
]
```

### 3.2 Response Structure & Cost Return
Runware returns immediate image URLs along with exact execution cost:
```json
{
  "data": [
    {
      "taskType": "imageInference",
      "taskUUID": "uuid-v4-generated-by-client",
      "imageUUID": "img-01j...",
      "imageURL": "https://temp-media.runware.ai/...",
      "cost": 0.0018
    }
  ]
}
```

### 3.3 Dynamic Cost Return Alignment with Tora Relay
- Runware natively returns `"cost": 0.0018` in the execution response.
- This maps directly into Tora's `NormalizedMediaOutput.CostActual` (`*float64`).
- Enables immediate, high-fidelity margin tracking without post-hoc billing reconciliation.

---

## 4. Model Catalog & Pricing Benchmarks

| Capability | Model Identifier | Typical Cost (USD) | Latency (p50) | Tora Margin at 5 Credits ($0.010 USD) |
|:---|:---|:---|:---|:---|
| **Flux.1 Schnell** | `runware:101@1` | **$0.0018** | ~0.8s | **82.0%** |
| **Flux.1 Dev** | `runware:100@1` | **$0.0120** | ~3.5s | **76.0%** (at 25 Credits / $0.050 USD) |
| **SDXL Turbo** | `runware:20@1` | **$0.0009** | ~0.4s | **91.0%** |
| **Background Remove** | `runware:rembg@1` | **$0.0020** | ~0.5s | **80.0%** (at 10 Credits / $0.020 USD) |

---

## 5. Fit with Tora Generic Media Relay

### 5.1 Protocol Identifier
- `ProtocolRunwareV1 = "RUNWARE_V1"`

### 5.2 Declarative Parameter Mapping DSL Fit
Runware parameter names align cleanly with Tora DSL:
- `prompt` -> `positivePrompt` (Transform: `rename`)
- `negative_prompt` -> `negativePrompt` (Transform: `rename`)
- `aspect_ratio` -> `width,height` (Transform: `format` or `default`)
- `output_format` -> `"WEBP"` (Transform: `constant`)

### 5.3 Verdict
Runware represents an exceptionally high-margin candidate (COGS $0.0018 for Flux Schnell yields an 82% margin vs WaveSpeed's 70%). However, following the overnight instructions:
- **Runware will NOT be activated tonight.**
- It remains cataloged as the future Provider #4 candidate once WaveSpeed and KIE live loops are established in production.
