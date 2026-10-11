# Tora Studio: High-Volume Batch Architecture & Chunking Plan
**Queue:** `QUEUE N4 PREFLIGHT`  
**Milestone:** Scaling from 10 to 25/50 Product Items via Client Chunking  
**Date:** 2026-10-08  

---

## 1. Executive Summary & Policy Invariant

In accordance with strict system stability rules:
- **Current Hard Limit**: 10 items per batch is retained in Queue N3.1.
- **Scaling Milestone**: Expanding to 25 and 50 items will only occur in Queue N4 following load evidence.
- **Server Safety Guarantee**: Zero heavy PyTorch or GPU models will run on the t4g.small production instance. The server only performs lightweight 2D image composition (Go `image/draw`), keeping CPU utilization below 15%.

---

## 2. Chunked Execution Architecture

Attempting to process 25 to 50 high-resolution camera photos (12MP–48MP) simultaneously would exhaust mobile RAM and trigger OS Low Memory Killer (LMK) terminations.

```
USER SELECTION (25 or 50 Photos)
                  │
                  ▼
         CLIENT-SIDE CHUNKING
       (Window Size = 3 items)
                  │
┌─────────────────┴─────────────────┐
│ Chunk 1 (Items 1-3):               │
│ - On-Device Cutout (Sequential)   │
│ - Peak RAM < 120 MB               │
│ - Upload Cutout PNGs to Server    │
│ - Receive Pack URLs               │
└─────────────────┬─────────────────┘
                  │ (Evict raw bitmaps from RAM)
┌─────────────────▼─────────────────┐
│ Chunk 2 (Items 4-6):               │
│ - Repeat on-device cutout         │
│ - Update overall progress UI      │
└─────────────────┬─────────────────┘
                  │
                  ▼
         CONSOLIDATED EXPORT
       - Single Master ZIP Archive
       - Unified History Record
```

---

## 3. Server-Side Bounded Queue Protection
1. **Concurrency Cap**: Maximum 4 concurrent composite jobs per client session.
2. **Backpressure**: If server queue exceeds 50 pending composites, client is signaled to pause before dispatching next chunk.
3. **Partial Failure Isolation**: If item #17 encounters decoding errors, remaining 49 items finish and download successfully.
