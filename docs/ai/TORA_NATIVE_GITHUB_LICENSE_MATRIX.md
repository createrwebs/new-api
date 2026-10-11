# TORA NATIVE GITHUB LICENSE FORENSICS MATRIX

**Document**: `docs/ai/TORA_NATIVE_GITHUB_LICENSE_MATRIX.md`  
**Execution Timestamp**: 2026-10-08T08:40:00+07:00  
**Audit Standard**: `CODE_LICENSE ≠ MODEL_WEIGHT_LICENSE ≠ DATASET_LICENSE`  
**Lab Root**: `/Users/noppanan/tora-studio-lab/native-references`  

---

## 1. Master Repository & Model License Matrix

| Repository | Pinned SHA | CODE_LICENSE | MODEL_WEIGHT_LICENSE | CHECKPOINT_LICENSE | DEPENDENCY_LICENSES | DATASET_RESTRICTIONS | ATTRIBUTION_REQUIRED | COMMERCIAL_USE | SAAS_USE | MODEL_REDISTRIBUTION | MODIFICATION | VERDICT |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **MODNet** | `28165a451e46` | Apache-2.0 | Apache-2.0 | Apache-2.0 | PyTorch (BSD), OpenCV (Apache-2.0), Pillow (HPND) | Trained on proprietary matting datasets; explicitly licensed Apache-2.0 by authors. | Yes (Apache-2.0 Notice) | **YES** | **YES** | **YES** | **YES** | **APPROVED_WITH_ATTRIBUTION** |
| **Real-ESRGAN** | `a4abfb2979a7` | BSD 3-Clause | BSD 3-Clause | BSD 3-Clause | PyTorch (BSD), BasicSR (Apache-2.0), torchvision (BSD) | DF2K, OST synthetic datasets. Author Xintao Wang distributes checkpoints under repo terms. | Yes (BSD Notice) | **YES** | **YES** | **YES** | **YES** | **APPROVED_WITH_ATTRIBUTION** |
| **rembg** (Harness) | `202e42649a84` | MIT | Model Specific (See Model Zoo) | Model Specific (See Model Zoo) | ONNX Runtime (MIT), Pillow (HPND), NumPy (BSD) | Harness contains no weights; downloads remote ONNX models. | Yes (MIT Notice) | **YES** (Harness code only) | **YES** | **YES** | **YES** | **ADAPT / HARNESS APPROVED** |
| **U2Net / U2NetP** (via rembg) | N/A (ONNX) | Apache-2.0 | Apache-2.0 / Academic ambiguous | Apache-2.0 | ONNX Runtime (MIT) | DUTS dataset used in training (academic origins). Upstream repo licenses weights Apache-2.0. | Yes | Permitted with attribution | Permitted with attribution | Permitted with attribution | Permitted | **APPROVED_WITH_ATTRIBUTION** |
| **transparent-background** (InSPyReNet) | `6860e05c4c57` | MIT | MIT / Academic Dataset | GitHub Release `.pth` | PyTorch (BSD), Albumentations (MIT), SwinTransformer (Apache-2.0) | Trained on DIS5K, DUTS-TR. Author licenses tool under MIT. | Yes (MIT Notice) | Conditional | Conditional | Conditional | Permitted | **REVIEW_REQUIRED** |
| **BiRefNet** | `ebcc0bc8ec7f` | MIT | **REVIEW_REQUIRED** | Mixed / Ambiguous | PyTorch (BSD), HuggingFace (Apache-2.0), Timm (Apache-2.0) | DIS5K, DUTS, HRSOD, P3M-10k. Academic dataset clauses create commercial ambiguity. | Yes | **UNCONFIRMED** | **UNCONFIRMED** | **UNCONFIRMED** | Permitted | **REVIEW_REQUIRED** |
| **BRIA RMBG-2.0** (Public Weights) | N/A | MIT / Apache (Architecture) | **CC-BY-NC 4.0 (Non-Commercial)** | **Non-Commercial** | Timm, PyTorch | BRIA proprietary high-res matting data. Commercial license requires enterprise paid contract with Bria.ai. | Yes | **PROHIBITED** without paid license | **PROHIBITED** without paid license | **PROHIBITED** | Permitted (Non-comm only) | **REJECTED (FOR COMMERCIAL SAAS)** |
| **web-realesrgan** | `1e3cf6331c80` | **GPL-2.0** | N/A (Frontend WebGL) | N/A | WebGPU / WebGL, TensorFlow.js / ONNX | N/A | Yes (Copyleft) | GPL-2.0 copyleft taint if linked | Copyleft implications | Copyleft implications | Copyleft implications | **REFERENCE_ONLY** |
| **IOPaint** | `61a759fb3f33` | Apache-2.0 | Model Specific (LaMa, StableDiffusion) | Archived upstream | FastAPI (MIT), PyTorch (BSD), Diffusers (Apache-2.0) | Upstream repository is archived. Individual inpainting models carry distinct licenses. | Yes | Permitted (code) | Permitted (code) | Permitted (code) | Permitted | **REFERENCE_ONLY** (Archived Upstream) |

---

## 2. Forensic License Findings & Governance Verdicts

### 1. BRIA RMBG-2.0 Gate
- **Status**: **STRICTLY REJECTED FOR UNLICENSED COMMERCIAL SAAS**
- **Rationale**: Bria.ai releases RMBG-1.4 and RMBG-2.0 weights on HuggingFace under Creative Commons Attribution-NonCommercial 4.0 International (`CC-BY-NC-4.0`).
- **Tora Invariant**: Tora Studio charges commercial Tora Credits. Using RMBG-2.0 without a dedicated Bria commercial enterprise contract violates licensing terms. Rembg's capability to download `briaai/RMBG-2.0` must NOT be enabled on commercial routes.

### 2. BiRefNet Checkpoint Audit
- **Status**: **REVIEW_REQUIRED — GATED FROM IMMEDIATE PRODUCTION DEPLOYMENT**
- **Rationale**: The code repository (`ZhengPeng7/BiRefNet`) is MIT. However, the pre-trained weights were trained across multiple academic datasets (`AIM-500`, `DIS-TR`, `DUTS-TR`, `HRSOD`, `P3M-10k`). Several of these source datasets carry academic-only research restrictions that taint the derived pre-trained checkpoint weights for commercial SaaS redistribution.
- **Action**: BiRefNet is retained in the lab for benchmarking and reference only. It will NOT be promoted to `ACTIVE` production until clean-room weights or explicit commercial clearances are established.

### 3. MODNet Portrait Matting
- **Status**: **APPROVED_WITH_ATTRIBUTION**
- **Rationale**: Authors (Zhang et al.) explicitly grant:
  > *"The code, models, and demos in this repository (excluding GIF files under the folder doc/gif) are released under the Apache-2.0 license."*
- **Tora Action**: Approved for portrait and human matting routes. Include Apache-2.0 notice in `THIRD_PARTY_NOTICES_NATIVE_TOOLS.md`.

### 4. Real-ESRGAN Super-Resolution
- **Status**: **APPROVED_WITH_ATTRIBUTION**
- **Rationale**: Code and official checkpoints (`RealESRGAN_x4plus`, `RealESRGAN_x2plus`, `realesr-general-x4v3`) are released under BSD 3-Clause by Xintao Wang. Binary redistribution and SaaS usage are explicitly permitted provided the copyright notice and disclaimer are retained.
- **Tora Action**: Approved for 2x and 4x image upscaling. Include BSD 3-Clause attribution.

### 5. web-realesrgan Copyleft Isolation
- **Status**: **REFERENCE_ONLY — ZERO CODE INGESTION**
- **Rationale**: Licensed under GNU General Public License Version 2 (`GPL-2.0`). Copying or statically linking its TypeScript/GLSL source into Tora Studio would trigger copyleft obligations over proprietary Tora code.
- **Tora Action**: Use strictly as an architectural reference for shader pipelines and WebGPU tile buffering. Tora native browser upscaler must be implemented independently using permissively licensed libraries (such as ONNX Runtime Web under MIT or custom WebGL/WebGPU shaders under MIT/Apache-2.0).

### 6. IOPaint Inpainting
- **Status**: **REFERENCE_ONLY — ARCHIVED UPSTREAM**
- **Rationale**: While licensed under Apache-2.0, the upstream repository has been officially archived by maintainer Sanster. Critical production systems must not take a hard dependency on an unmaintained codebase.
- **Tora Action**: Harvest UI/UX patterns (canvas brush, mask serialization, undo/redo state machine) into Tora-native React components.
