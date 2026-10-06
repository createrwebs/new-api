# TORA AI STUDIO — OPEN SOURCE REPOSITORY AUDIT MATRIX
## ARCHITECTURAL EVALUATION, LICENSING, HARDWARE FOOTPRINT & PRODUCTION RECOMMENDATIONS

> **Target Platform**: Tora Studio Media AI Suite  
> **Infrastructure Invariant (Section 45)**: `NEW_SERVER_COUNT = 0`, `NEW_GPU_SERVER_COUNT = 0`.  
> **Production Policy**: Zero PyTorch/CUDA workloads on current production EC2 (`51.20.174.90`). All open-source repositories audited herein are evaluated for **Reference Only** or **Future Self-Hosted GPU Cluster**, with initial production MVP routing strictly via high-availability managed API providers (MuAPI, fal.ai, Replicate).

---

### 1. Comprehensive Repository Audit Table

| Repository | Pinned Commit SHA | Primary Language / Runtime | Code License | Model Weights License | GPU / Hardware Requirement | CPU Viable? | Maintenance State | Commercial Risk Level | Tora Role & Recommendation |
| :--- | :--- | :--- | :--- | :--- | :--- | :---: | :---: | :---: | :--- |
| **Comfy-Org / ComfyUI** | `7a5dad695fe1cae25efcb2550530fb20ef68da3d` | Python 3.11+ / PyTorch | GPL-3.0 | Variable (Checkpoint dependent) | 12GB–24GB+ VRAM (NVIDIA CUDA) | No | Highly Active | Medium (GPL-3.0 copyleft) | `REFERENCE_ONLY`<br/>Node graph orchestration model |
| **xinntao / Real-ESRGAN** | `a4abfb2979a7bbff3f69f58f58ae324608821e27` | Python / C++ (NCNN Vulkan) | BSD 3-Clause | BSD 3-Clause (DIV2K / Flickr2K) | 4GB–8GB VRAM | Yes (NCNN 2–5s/tile) | Stable / Maintenance | Low (Permissive) | `FUTURE_SELF_HOST`<br/>Candidate for dedicated worker |
| **danielgatis / rembg** | `202e42649a8492a7c49f808de36608a7d1cbbfe3` | Python / ONNX Runtime | MIT | Apache 2.0 (U2-Net / BiRefNet) | 2GB–4GB VRAM | Yes (1–2s on 4 vCPU) | Highly Active | Low (MIT / Apache) | `FUTURE_SELF_HOST`<br/>Can run as lightweight CPU sidecar |
| **TencentARC / GFPGAN** | `7552a7791caad982045a7bbe5634bbf1cd5c8679` | Python / PyTorch | Apache 2.0 | Non-Commercial (FFHQ Dataset) | 6GB–8GB VRAM | No | Stable / Low updates | **HIGH** (Non-commercial weights) | `NOT_RECOMMENDED`<br/>Commercial licensing risk |
| **TencentARC / PhotoMaker** | `060b4fcb10b76a4554edf565d6106b7e36c968f0` | Python / Diffusers | Apache 2.0 | CC-BY-NC 4.0 (Non-Commercial) | 16GB–24GB VRAM (SDXL) | No | Active Research | **HIGH** (Non-commercial weights) | `NOT_RECOMMENDED`<br/>Use managed commercial providers |
| **TMElyralab / MuseTalk** | `0a89dec45a0192b824e3cf4daf96c239440c5ed8` | Python / PyTorch / Whisper | Apache 2.0 | Research Only (HDTF Dataset) | 16GB–24GB VRAM (Real-time 30fps) | No | Active | Medium (Facial dataset rights) | `REFERENCE_ONLY`<br/>Lip-sync architecture reference |
| **Lightricks / LTX-Video** | `4b2d053057623ddd4d0a1d3e9cd28890e9ef487f` | Python / PyTorch / Diffusers | Apache 2.0 | Apache 2.0 (Permissive) | 12GB–24GB VRAM (High speed) | No | Highly Active | Low (Apache 2.0) | `FUTURE_SELF_HOST`<br/>Initial MVP via fal.ai |
| **Wan-Video / Wan2.1 & 2.2**| `1ea34ff48f87168174e12956e200b1d908b1c5ff` | Python / PyTorch / FlashAttn-2 | Apache 2.0 | Apache 2.0 (1.3B & 14B) | 16GB–48GB VRAM | No | Cutting-edge / Active | Low (Apache 2.0) | `FUTURE_SELF_HOST`<br/>Initial MVP via fal.ai |
| **facefusion / facefusion** | `72470819a0373be3388b3929c8f8f311f418fc3c` | Python / ONNX / TensorRT | AGPL-3.0 | Variable / InsightFace restrictions | 8GB–16GB VRAM | No | Highly Active | **CRITICAL** (AGPL + Deepfake risk)| `NOT_RECOMMENDED`<br/>Avoid direct hosting |
| **gitroomhq / agent-media** | `817f28477ca80a0e1be8270ec9616a1b8bafcd78` | TypeScript / Node.js | MIT | N/A (API Orchestrator) | 0 (No GPU required) | Yes (CLI/MCP) | Active | Low (MIT) | `REFERENCE_ONLY`<br/>Agent UGC generation pattern |
| **gitroomhq / postiz-app** | `22c034188092be11576190a05a82563a53efed9a` | TypeScript / React / NestJS | AGPL-3.0 | N/A (Scheduling Platform) | 0 (No GPU required) | Yes (Standard Web) | Highly Active | Medium (AGPL-3.0) | `REFERENCE_ONLY`<br/>Asset management reference |
| **thatseoagent / mcp** | `c35bd83f9b03a5ea055254f2706b0fd58a6fd843` | TypeScript / Python | MIT | N/A (MCP Server) | 0 (No GPU required) | Yes | Active | Low (MIT) | `REFERENCE_ONLY`<br/>MCP server tooling pattern |

---

### 2. Deep Architectural Insights & Synthesis

#### A. The Commercial Licensing Minefield (GFPGAN, PhotoMaker, FaceFusion)
Many highly popular image/video repositories on GitHub feature permissive code licenses (Apache 2.0, MIT) but are paired with model weights trained on academic non-commercial datasets:
- **GFPGAN** is trained on the FFHQ (Flickr-Faces-HQ) dataset, which explicitly prohibits commercial use.
- **PhotoMaker** weights are published under CC-BY-NC 4.0.
- **FaceFusion** utilizes AGPL-3.0 licensing combined with sensitive biometric face swap technologies that create severe regulatory and platform liabilities.

**Tora Architectural Decision**: Tora AI will NOT self-host or distribute weights with non-commercial restrictions. In the initial Studio MVP, all facial enhancement, identity preservation, and portrait styling will be routed through commercial API providers (fal.ai, MuAPI, Replicate) that offer commercial indemnification and compliant models (e.g. BiRefNet, Flux LoRA, SDXL Commercial).

#### B. The True Cost of Self-Hosting Video (Wan2.1/2.2 & LTX-Video)
- Video models like **Wan2.2 (14B)** require substantial GPU VRAM: minimum 24GB for quantized 4-bit, and 48GB–80GB (NVIDIA A100/H100) for full fp16 generation.
- Running a single dedicated A100 (80GB) on AWS/Lambda/RunPod costs approximately **$1.80 – $3.50 per hour** ($1,300 – $2,500/month flat).
- At early traffic volume (<1,000 video generations/day), renting dedicated GPUs produces a negative margin.
- **Tora Economic Decision**: By routing video generation to **fal.ai** (`fal-ai/wan-t2v` at ~$0.04 - $0.08 per 5s clip), Tora incurs **zero idle cost**, 100% pay-per-generation pricing, and preserves healthy gross margins (65%+).

#### C. CPU / Lightweight Self-Hosting Candidates for Phase 12
When Tora's transaction volume justifies self-hosting to reduce COGS:
1. **Background Removal (`danielgatis/rembg`)**: Can be deployed via ONNX Runtime on standard AMD64/ARM64 CPU instances with ~1.5s latency per image.
2. **Image Upscaling (`Real-ESRGAN-ncnn-vulkan`)**: Can run on small GPU nodes (T4 or L4) at sub-cent costs per 4K upscale.
