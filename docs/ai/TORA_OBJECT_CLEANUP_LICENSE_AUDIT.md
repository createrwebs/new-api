# Tora Studio: Object Cleanup Model & Repository License Audit
**Queue:** `QUEUE N3.1 / N4 PREFLIGHT`  
**Focus:** Commercial Feasibility, Intellectual Property Safety & Clean-Room Inpainting  
**Date:** 2026-10-08  

---

## 1. Executive Summary

This audit establishes a strict commercial compliance boundary for the future **Object Cleanup** (object erasure/inpainting) capability slated for Queue N4. In accordance with Tora's legal and financial security invariants, all candidate models, weights, and repositories are evaluated against Apache-2.0, MIT, and commercial training data provenance.

---

## 2. Model & Repository Candidate Audit

| Candidate Repository / Model | Upstream Code License | Weight / Model Checkpoint License | Training Dataset Terms | Commercial Tora Status | Rationale |
|---|---|---|---|---|---|
| **Sanster/IOPaint** (formerly Lama-Cleaner) | Apache-2.0 (Archived) | Varies per bundled model | Multi-source | **REFERENCE ONLY** | Upstream GUI and runner code is Apache-2.0, but each underlying weight carries its own terms. Model weights cannot inherit repo license. |
| **LaMa** (`advimman/lama`) | Apache-2.0 | **NON-COMMERCIAL RESEARCH ONLY** | Places2 (CC BY-NC 4.0) | **REJECTED (COMMERCIAL HARD GATE)** | Upstream weights are trained on Places2 / CC BY-NC data and explicitly released for non-commercial research purposes only. Commercial exploitation poses copyright liability. |
| **MAT** (`fenglinglwb/MAT` Mask-Aware Transformer) | Research Only | Research Only | Places2 / CelebA (NC) | **REJECTED / RESEARCH_ONLY** | Author repository explicitly restricts usage to research/academic purposes. Commercial deployment prohibited. |
| **MIGAN** / **AOT-GAN** | Non-Commercial | Non-Commercial | Places Challenge (NC) | **REJECTED** | Non-commercial academic license. |
| **Fast Inpainting (OpenCV Telea / Navier-Stokes)** | BSD-3-Clause | Pure Algorithm (No Weights) | N/A | **COMMERCIAL SAFE (TIER 0)** | Deterministic pixel-diffusion inpainting; zero AI weights; ultra-fast for tiny scratches and logos. |
| **Clean-Room ONNX Inpainting (CC0 / Apache Trained)** | Apache-2.0 | Commercial-Clean ONNX | Public Domain / CC0 | **UNDER EVALUATION FOR N4** | Requires explicit verifiable chain of custody on training datasets before release. |

---

## 3. Findings & Hard Gates

### 3.1 The "LaMa Trap"
Many open-source tools (such as IOPaint/Lama-Cleaner) are distributed under permissive top-level licenses (Apache-2.0 or MIT). However, downloading and executing default `big-lama.pt` or `lama.onnx` binds the user to the original checkpoint license of the `advimman/lama` research team. That checkpoint was trained on the Places2 dataset, which is restricted to non-commercial research under Creative Commons Attribution-NonCommercial 4.0 (CC BY-NC 4.0).
**Verdict**: `LaMa` weights are **REJECTED** for commercial Tora Studio execution.

### 3.2 The MAT Warning
Mask-Aware Transformer (MAT) demonstrates superior large-hole fill capabilities in academic benchmarks, but its repository contains an explicit non-commercial restriction.
**Verdict**: `MAT` is **REJECTED / RESEARCH_ONLY**.

---

## 4. Recommended Tora Studio N4 Strategy

To deliver immediate customer value without legal risk or expensive GPU server purchases:

1. **Decouple Mask Creation from Inpainting**:
   - Deliver a high-performance, client-side mask editing canvas (brush, eraser, lasso, feather, zoom/pan).
   - The user workflow is immediately operational for marking unwanted objects.
2. **Tier 0 Deterministic Patching**:
   - Fast, non-AI algorithm (patch-match / exemplar-based synthesis) for small text, timestamps, and dust marks. 100% commercial-safe, instant execution, zero cloud COGS.
3. **Tier 1 Verified Commercial AI Model**:
   - Only activate neural inpainting checkpoints trained exclusively on public-domain / licensed imagery with explicit commercial rights.
