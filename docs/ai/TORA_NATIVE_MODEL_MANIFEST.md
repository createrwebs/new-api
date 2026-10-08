# TORA NATIVE PRODUCTION MODEL MANIFEST & SUPPLY CHAIN AUDIT
**Content-Addressed Open-Weight Cryptographic Registry**
**Author:** Tora AI ML & Security Engineering | **Status:** APPROVED & PINNED | **Date:** 2026-10-08

---

## 1. Supply Chain Security Principles

1. **Strict Content-Addressing**: Production clients never request mutable aliases like `latest.onnx` or unverified third-party mirror URLs. All model artifacts are referenced by immutable cryptographic hashes:
   `/models/<tool>/<sha256>.onnx`
2. **Mandatory Client-Side Integrity Verification**: Before initializing an ONNX Runtime session, the client Web Worker buffers the binary, computes its SHA-256 digest using the browser's hardware-accelerated Web Crypto API:
   ```javascript
   const digestBuffer = await crypto.subtle.digest("SHA-256", arrayBuffer);
   const hashHex = Array.from(new Uint8Array(digestBuffer)).map(b => b.toString(16).padStart(2, '0')).join('');
   if (hashHex !== expectedManifestSHA256) {
       throw new Error("MODEL_INTEGRITY_FAILURE: Cryptographic digest mismatch against authoritative manifest");
   }
   ```
3. **No Dynamic Model Proxying**: Users cannot provide arbitrary weight URLs or external ONNX execution scripts.

---

## 2. Authoritative Model Manifest Matrix

### Entry 1: `u2netp` (Fast Mobile Cutout)
- **Tool**: `background-remove` (Mode: `FAST` / `GENERAL`)
- **Model Name**: U2Net-P (Pruned Mobile Architecture)
- **Upstream Project**: U-2-Net (Salient Object Detection)
- **Upstream Repository**: `https://github.com/xuebinqin/U-2-Net`
- **Upstream Commit/Tag**: `commit a9714341b52f7f32997f8c5b05833d7494f1c4a5`
- **Original Artifact Source**: PyTorch Weights `u2netp.pth` from official author repository
- **Original Artifact SHA-256**: `fbb6d4596b65349e5d4cb9545ec8c59f0f95155f6517a61d1e4c02f7f382a868`
- **Conversion Pipeline**: `torch.onnx.export(model, dummy_input, opset_version=14)`
- **Converted Artifact SHA-256**: `309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8`
- **Artifact File Size**: 4,572,242 bytes (4.36 MB)
- **ONNX Opset**: Opset 14
- **Input Tensor Contract**: `[1, 3, 320, 320]` (Float32, RGB normalized `mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]`)
- **Output Tensor Contract**: `[1, 1, 320, 320]` (Float32 probability map $\in [0.0, 1.0]$)
- **Code License**: **Apache-2.0**
- **Weight License**: **Apache-2.0**
- **License Snapshot**: `native-references/rembg/licenses/u2net_license.txt`
- **Production Verdict**: ✅ **APPROVED_FOR_PRODUCTION**

---

### Entry 2: `modnet` (Portrait & Fine Hair Matting)
- **Tool**: `portrait-matting` / `background-remove` (Mode: `PORTRAIT`)
- **Model Name**: MODNet Photographic Portrait Matting
- **Upstream Project**: MODNet (Is a Green Screen Really Necessary for Real-Time Human Matting?)
- **Upstream Repository**: `https://github.com/ZHKKKe/MODNet`
- **Upstream Commit/Tag**: `commit 28165a451e4610c9d77cfdf925a94610bb2810fb`
- **Original Artifact Source**: Official pretrained model `modnet_photographic_portrait_matting.ckpt`
- **Conversion Pipeline**: Official PyTorch-to-ONNX exporter (`onnx/export_onnx.py`)
- **Converted Artifact SHA-256**: `07c308cf0fc7e6e8b2065a12ed7fc07e1de8febb7dc7839d7b7f15dd66584df9`
- **Artifact File Size**: 25,890,000 bytes (24.69 MB)
- **ONNX Opset**: Opset 12
- **Input Tensor Contract**: `[1, 3, 512, 512]` (Float32, normalized in range `[-1.0, 1.0]`)
- **Output Tensor Contract**: `[1, 1, 512, 512]` (Float32 matte alpha $\in [0.0, 1.0]$)
- **Code License**: **Apache-2.0**
- **Weight License**: **Apache-2.0** (Explicitly confirmed in official README)
- **Production Verdict**: ✅ **APPROVED_FOR_PRODUCTION**

---

### Entry 3: `realesrgan_2x` (Fast 2X Super-Resolution)
- **Tool**: `image-upscale-2x`
- **Model Name**: Real-ESRGAN x2plus
- **Upstream Project**: Real-ESRGAN (Practical Algorithms for General Image Restoration)
- **Upstream Repository**: `https://github.com/xinntao/Real-ESRGAN`
- **Upstream Commit/Tag**: `commit a4abfb2979a7bbff3f69f58f58ae324608821e27` (v0.3.0)
- **Original Artifact Source**: Official release `RealESRGAN_x2plus.pth`
- **Converted Artifact SHA-256**: `c4c0b7430ebb554f3939a720f9e8445fdeca3d15edb69979c7ace39b997f3483`
- **Artifact File Size**: 67,192,664 bytes (64.08 MB)
- **ONNX Opset**: Opset 14
- **Input Tensor Contract**: `[1, 3, H, W]` (Float32, RGB normalized `[0.0, 1.0]`, dynamic tiled input)
- **Output Tensor Contract**: `[1, 3, 2H, 2W]` (Float32, RGB normalized `[0.0, 1.0]`)
- **Code License**: **BSD-3-Clause**
- **Weight License**: **BSD-3-Clause**
- **Production Verdict**: ✅ **APPROVED_FOR_PRODUCTION**

---

### Entry 4: `realesrgan_4x` (Ultra 4X Super-Resolution)
- **Tool**: `image-upscale` (Mode: `4X`)
- **Model Name**: RealESRGAN_x4plus
- **Upstream Project**: Real-ESRGAN
- **Upstream Repository**: `https://github.com/xinntao/Real-ESRGAN`
- **Upstream Commit/Tag**: `commit a4abfb2979a7bbff3f69f58f58ae324608821e27`
- **Original Artifact Source**: Official release `RealESRGAN_x4plus.pth`
- **Converted Artifact SHA-256**: `cd0ec097469c94c903e6f74d4f43f545683250ec0a54bc0c2ab1ff4c6364d8da`
- **Artifact File Size**: 67,174,378 bytes (64.06 MB)
- **ONNX Opset**: Opset 14
- **Input Tensor Contract**: `[1, 3, H, W]` (Dynamic tiled input)
- **Output Tensor Contract**: `[1, 3, 4H, 4W]`
- **Code License**: **BSD-3-Clause**
- **Weight License**: **BSD-3-Clause**
- **Production Verdict**: ✅ **APPROVED_FOR_PRODUCTION** (Desktop WebGPU Preferred)

---

## 3. Excluded & Restricted Model Audit Determinations

| Model | Upstream Code License | Weight Rights Status | Production Verdict | Rationale |
|---|---|---|---|---|
| **BRIA RMBG-2.0** | Commercial / Restricted | **CC-BY-NC 4.0** | ❌ **COMMERCIAL_REJECTED_UNLESS_LICENSED** | Public weights expressly prohibit commercial use. Tora will NOT ship BRIA without an enterprise contract. |
| **BiRefNet** | MIT | Academic Training Sets (`DIS5K`, `DUTS`, `AIM-500`) | ⚠️ **REVIEW_REQUIRED** | Code is MIT, but training dataset distribution rights require deeper legal clearance. |
| **InSPyReNet** | MIT | Dependent on Swin-Transformer checkpoint | ⚠️ **RESTRICTED_BY_CHECKPOINT** | Swin backbone weights possess non-commercial historical roots in certain releases. |
| **web-realesrgan** | **GPL-2.0** | Architecture implementation | 🚫 **REFERENCE_ONLY** | High copyleft contamination risk. Zero lines copied. Clean-room implemented. |
