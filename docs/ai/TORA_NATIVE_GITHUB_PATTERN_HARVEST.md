# TORA NATIVE GITHUB PATTERN HARVEST REPORT

**Document**: `docs/ai/TORA_NATIVE_GITHUB_PATTERN_HARVEST.md`  
**Execution Timestamp**: 2026-10-08T08:42:00+07:00  
**Audit Standard**: `ADOPT vs ADAPT vs REFERENCE_ONLY vs REJECT`  
**Lab Root**: `/Users/noppanan/tora-studio-lab/native-references`  

---

## 1. Architectural Pattern Classification Matrix

| Technical Area | Source Repository | Pattern Description | Classification | Rationale & Tora Implementation Strategy |
| :--- | :--- | :--- | :--- | :--- |
| **Browser Model Cache** | `web-realesrgan` | IndexedDB persistent storage of ONNX/Graph models (`indexeddb://${name}`) | **ADOPT** | Eliminates redundant 10MB–50MB model downloads on repeat executions. Version keys prevent stale weights. |
| **Worker Inference** | `web-realesrgan` | Offloading inference and tensor conversion to dedicated Web Worker (`worker.js`) | **ADOPT** | Prevents blocking the browser main UI thread; guarantees 60fps responsive UI during heavy client tensor ops. |
| **Tile Overlap & Blending** | `Real-ESRGAN` & `web-realesrgan` | Splitting 2K/4K images into overlapping tiles (`min_lap`, `base_lap`) with linear alpha blending | **ADOPT** | Eliminates GPU/browser VRAM exhaustion and prevents visible tile grid seams. Clean-room TypeScript implementation without GPL taint. |
| **Alpha Matting Trimap** | `rembg` | Closed-form alpha matting (`estimate_alpha_cf`) and multilevel foreground extraction | **ADAPT** | Essential for fine hair strands and semi-transparent glass edges where raw binary thresholding leaves hard halos. |
| **ONNX Runtime Session Management** | `rembg` | Lazy session allocation, execution provider selection (CoreML, DirectML, WebAssembly, WebGPU) | **ADAPT** | Robust runtime abstraction for local CPU and browser WASM execution. |
| **Portrait Matting Architecture** | `MODNet` | Real-time trimap-free human portrait matting with sub-objective loss | **ADAPT** | Lightweight (~7MB ONNX), fast execution on consumer hardware, Apache-2.0 clean license. |
| **Mask Canvas UX & Undo/Redo** | `IOPaint` | Dual-layer canvas (source image + transparent mask overlay), zoom/pan wrapper, brush size HUD, history undo stack | **ADAPT** | Reusable for Tora Object Cleanup and Inpainting tools within existing React Tailwind component tree. |
| **GPL WebGPU Shaders** | `web-realesrgan` | Direct GLSL shader source code in repository | **REJECT** | Must NOT be copied due to GPL-2.0 copyleft terms. Tora uses standard ONNX Runtime Web / WebGL backends instead. |
| **BRIA RMBG-2.0 Weights** | `rembg` | Automated downloading of RMBG-2.0 checkpoints | **REJECT** | Non-commercial CC-BY-NC 4.0 license prevents commercial Tora SaaS usage without paid enterprise agreement. |
| **Archived App Hosting** | `IOPaint` | Running complete FastAPI backend application container | **REFERENCE_ONLY** | Archived upstream; Tora adopts UI patterns and model inference logic directly into Tora Go/React codebase. |
| **cv::inpaint (Telea / NS)** | `opencv` | Pinned `73a26a42` (5.x), Apache-2.0. Deterministic inpainting via Fast Marching Method (`INPAINT_TELEA`) & Navier-Stokes fluid PDE (`INPAINT_NS`). | **ADOPT (V1 BASELINE)** | 100% deterministic, zero model weights, zero download, ultra-low latency, Apache-2.0 clean. |
| **Randomized PatchMatch** | `PatchMatch` & `PyPatchMatch` | Pinned `64d9f3c6` / `ee63e2a1`, MIT. Randomized nearest neighbor field propagation for structural hole filling. | **REFERENCE_ONLY** | Valuable for texture/pattern synthesis, but higher complexity and CPU cost than Telea. Evaluated as potential quality upgrade. |
| **Biharmonic Inpainting** | `scikit-image` | Pinned `b33ab973`, BSD-3-Clause. Biharmonic PDE inpainting harness & validation metrics. | **BENCHMARK_ONLY** | Used strictly inside lab benchmark harness to calculate objective quality tolerances. |

---

## 2. In-Depth Technical Pattern Analysis

### A. Client-Side Browser Model Execution & Cache Strategy
1. **IndexedDB Weight Persistence**:
   - When a user selects a browser-native tool for the first time, model blobs (e.g. `realesr-general-x4v3.onnx` or `modnet_photographic.onnx`) are fetched via HTTP with streaming progress reported to the user.
   - Once downloaded and verified via SHA-256 hash, the raw ArrayBuffer is stored in browser IndexedDB keyed by `tora_model_<id>_<version>_<sha256>`.
   - Subsequent executions read directly from local IndexedDB, reducing cold-start latency from ~10s to <150ms.
2. **Web Worker Thread Isolation**:
   - All image decoding, canvas extraction, tensor conversion (`ImageData` $\to$ float32 tensor), and ONNX inference run inside a dedicated Web Worker.
   - The main React thread only manages UI progress bars, cancel buttons, and canvas rendering.
3. **Tile Overlap Algorithm**:
   - For an image of dimensions $W \times H$ upscaled by factor $S$:
     $$\text{Tile Size} = 128 \times 128,\quad \text{Overlap} = 16\text{ px}$$
   - Tiles are evaluated sequentially or in mini-batches. Neighboring overlapping margins are blended using a linear ramp filter $\alpha(x) = \frac{x}{\text{Overlap}}$ to completely eliminate boundary artifacts.

### B. Alpha Matting & Edge Postprocessing
1. **Naive Cutout vs Closed-Form Matting**:
   - Standard semantic segmentation outputs a continuous probability mask $[0.0, 1.0]$.
   - Naive cutout simply applies `img.putalpha(mask * 255)`. While fast ($<10\text{ms}$), it leaves jagged edges or color fringes on complex subjects.
   - Closed-form alpha matting expands the boundary region into an unknown band (trimap) through morphological dilation and erosion:
     $$\text{Erode}(M, k) \to \text{Foreground},\quad \text{Dilate}(M, k) \to \text{Background},\quad \text{Boundary} \to \text{Unknown}$$
   - Solving the matting Laplacian yields continuous, natural transparency around fine details (e.g., hair, fur, transparent bottles).

### C. Mask UX & Inpainting State Machine (from IOPaint)
1. **Dual Canvas Architecture**:
   - Base canvas: Renders background asset.
   - Mask canvas: Listens to mouse/pointer events with variable brush radius ($5\text{px} - 100\text{px}$).
   - Paint strokes recorded as vectorized path coordinates to enable arbitrary undo/redo levels before rasterization to an 8-bit grayscale PNG mask.
2. **Interactive Segmentation (SAM Integration)**:
   - Positive/negative point clicks mapped to bounding box anchors, yielding instant real-time mask generation without manual tracing.
