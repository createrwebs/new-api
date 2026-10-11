# Third-Party Notices — Tora Native Tools & Reference Lab

This document acknowledges third-party open-source projects, algorithmic references, and libraries studied, adapted, or benchmarked for Tora Studio Native Tools.

---

## 1. OpenCV (Open Source Computer Vision Library)
- **Component**: `cv::inpaint` (Fast Marching Method Telea, Navier-Stokes inpainting)
- **Source Repository**: `https://github.com/opencv/opencv`
- **Pinned Commit / Branch**: `73a26a423163d0284a41e997717b00933978b256` (`5.x` / `5.1.0-dev`)
- **License**: Apache License 2.0
- **Copyright**: Copyright (C) 2000-2026, Intel Corporation, OpenCV Foundation, all rights reserved.
- **Usage in Tora**: Clean-room algorithmic reference for deterministic Fast Marching Method (`DeterministicTeleaInpaint`) and benchmark baseline.

---

## 2. PatchMatch (Randomized Correspondence Algorithm)
- **Component**: Core PatchMatch randomized nearest-neighbor field algorithm
- **Source Repositories**:
  - `https://github.com/younesse-cv/PatchMatch` (Commit `64d9f3c6ee3e419f39bbab75e527094e4b7e20db`)
  - `https://github.com/vacancy/PyPatchMatch` (Commit `ee63e2a10338c52ce5479368122cc0367c28e59c`)
- **License**: MIT License
- **Copyright**: Copyright (c) 2016-2024 Younesse, Vacancy
- **Usage in Tora**: Reference lab study for structural patch synthesis. Isolated from production codebase.

---

## 3. scikit-image
- **Component**: Image processing algorithms and inpainting metrics
- **Source Repository**: `https://github.com/scikit-image/scikit-image`
- **Pinned Commit**: `b33ab973a08498266362efa6cb29c0caf6af912f` (`main`)
- **License**: Modified BSD (3-Clause) License
- **Copyright**: Copyright (C) 2011-2026, the scikit-image team. All rights reserved.
- **Usage in Tora**: Benchmark harness reference metrics.

---

## 4. Microsoft ONNX Runtime
- **Component**: ONNX Runtime Engine (Mobile and Web execution providers)
- **Source Repository**: `https://github.com/microsoft/onnxruntime`
- **License**: MIT License
- **Copyright**: Copyright (c) Microsoft Corporation. All rights reserved.
- **Usage in Tora**: On-device inference acceleration for background matting and super-resolution.

---

## 5. Non-Commercial Neural Weights Policy
- **LaMa (Resolution-robust Large Mask Inpainting)**: **STRICTLY REJECTED**. Trained on Places2 non-commercial dataset and released under non-commercial research license. Not used or bundled in Tora commercial software.
- **MAT (Mask-Aware Transformer)**: **STRICTLY REJECTED**. Non-commercial research license. Not used or bundled in Tora commercial software.
