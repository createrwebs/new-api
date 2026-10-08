# Tora Studio: Object Cleanup UI & Mask Architecture
**Queue:** `QUEUE N4 PREFLIGHT`  
**Milestone:** Client-Side Mask Creation & Inpainting Pipeline Decoupling  
**Date:** 2026-10-08  

---

## 1. Architectural Philosophy: Decoupled Masking

To avoid tying user experience to heavy, legally questionable, or expensive server GPU models, **Object Cleanup** is strictly decoupled into two independent layers:

```
┌────────────────────────────────────────────────────────┐
│               LAYER 1: MASK EDITOR CANVAS              │
│  (100% Client-Side Interactive Painting & Geometry)    │
│  - Brush / Eraser with Dynamic Radius & Feathering     │
│  - Infinite Zoom & Smooth 2D Panning (InteractiveViewer)│
│  - Multi-Level Undo / Redo Command Stack                │
│  - High-Contrast Semi-Transparent Mask Overlay          │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼ Binary Alpha Mask (0/255 PNG)
┌────────────────────────────────────────────────────────┐
│            LAYER 2: INPAINTING EXECUTION ENGINE        │
│  - Tier 0: Fast Local PatchMatch (Text/Logo removal)   │
│  - Tier 1: Commercial-Safe ONNX Inpainting             │
│  - Optional: Remote Serverless Diffusion Inpaint        │
└────────────────────────────────────────────────────────┘
```

---

## 2. Mask Editor UI Component Specifications

### 2.1 Interaction Controls
- **Brush Tool**: Variable diameter (8px to 128px), hardness slider, soft feather boundary.
- **Eraser Tool**: Inverts brush mask to restore accidentally marked pixels.
- **Pan / Zoom Controls**: Two-finger pinch-to-zoom (mobile) and scroll-wheel/drag (web) without painting unintended strokes.
- **Undo / Redo Buffer**: In-memory stroke vector history (up to 30 steps) allowing single-tap restoration.
- **Before / After Comparison**: Press-and-hold split-slider to toggle original image versus masked view.
- **Reset**: One-click purge of all mask strokes.

### 2.2 Output Contract
- **Format**: Single-channel 8-bit grayscale PNG or RGBA PNG where Alpha channel encodes marked mask (255 = erase target, 0 = keep).
- **Dimensions**: Strictly matches original input photo resolution (no downscaling during mask export).
