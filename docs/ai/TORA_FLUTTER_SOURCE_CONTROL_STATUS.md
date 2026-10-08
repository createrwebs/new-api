# Tora Studio Flutter Source Control & Durability Audit
**Queue:** `QUEUE N4A`  
**Timestamp:** 2026-10-08T19:53:00+07:00  
**Workspace:** `/Users/noppanan/.gemini/antigravity/scratch/LumenFlow`  
**Current HEAD:** `d34fac0`  
**Local Branches:** `main`, `tora/main`  

---

## 1. Remote Inventory & Governance Audit

### Remote Configuration
```text
origin  https://github.com/HuanMeng-official/LumenFlow.git (fetch)
origin  https://github.com/HuanMeng-official/LumenFlow.git (push)
```

### Governance Verdict
* **Upstream Protection**: The `origin` remote points to the third-party upstream open-source repository (`HuanMeng-official/LumenFlow.git`).
* **Push Policy**: Proprietary Tora Studio mobile integration (Native Mobile inference, Billing V2 client, Product Factory V2 batch engine, Object Cleanup mask editor, Tora Credit wallet integration) must **NOT** be pushed directly to upstream LumenFlow.
* **Tora Remote Status**: No authoritative Tora-owned remote (e.g. `github.com/createrwebs/tora-mobile` or similar) is currently configured in git remotes.
* **Formal Status**: **`TORA_FLUTTER_REMOTE_REQUIRED`** (Operator action required to provision a dedicated private repository for Tora Mobile).

---

## 2. Durable Local Backup Artifacts

Because the scratch directory must never be the sole copy of engineering work, complete durable backups have been generated and archived in `/Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/`:

| Artifact Type | File Path | SHA-256 Checksum | Size |
|---|---|---|---|
| **Git Bundle (Complete DAG)** | `/Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/lumenflow-tora-d34fac0.bundle` | `dafc7704f2b321f4ed20ad814cf6a37613f5194bafb589371f4fcb739c9de97c` | 9.4 MB |
| **Patch 1 (Mobile & Product Factory)** | `/Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/patches/0001-feat-studio-add-Tora-Native-Mobile-and-Product-Facto.patch` | `3ac3c2940bdf86b587d004e94036919ec5f8db0c25592a34051dccbefa0aba21` | 33.2 KB |
| **Patch 2 (Object Cleanup Mask Editor)** | `/Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/patches/0002-feat-studio-add-Object-Cleanup-interactive-mask-edit.patch` | `4b2232abdfd899d36b489d7d32e3e6923c94f7f274f32230d1590cafaad05735` | 27.6 KB |
| **Checksum Manifest** | `/Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/CHECKSUMS.txt` | Verified | 546 B |

### Restoration Command
Any developer or CI pipeline can restore the exact Tora mobile repository without access to scratch via:
```bash
git clone /Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/lumenflow-tora-d34fac0.bundle -b tora/main lumenflow-restored
```

---

## 3. Commit Provenance Preservation

Upstream commit provenance is 100% preserved. The Tora mobile feature commits sit directly on top of upstream HEAD (`bd496fd`):

```text
d34fac0 (HEAD -> main, tora/main) feat(studio): add Object Cleanup interactive mask editor, session client, and coverage validation
d77ffbb feat(studio): add Tora Native Mobile and Product Factory on-device inference with Billing V2
bd496fd (origin/main, origin/HEAD) chore: 生成器健壮性处理
a74ccc0 feat: 导入导出校验与提示
dc156ee chore: 修正拼写错误
5a6e9f0 feat: 导入XML严格匹配标签
d9b1036 fix: 构建时间改由构建脚本注入
813e966 chore: 新的构建脚本
f75a31f chore: 调整默认窗口大小
2298418 feat: 适配deepseek-flash识图
7f3c048 refactor & fix: 抽取Provider共享基类, 修复错误响应解析
```

All LumenFlow MIT licenses and original copyrights remain intact in root `LICENSE` and headers.

---

## 4. Operator Action: Remote Provisioning Template (`TORA_FLUTTER_REMOTE_REQUIRED`)

When an authoritative Tora repository is provisioned on GitHub, the operator should execute the following sequence:

```bash
cd /Users/noppanan/.gemini/antigravity/scratch/LumenFlow

# 1. Add authoritative private Tora remote
git remote add tora git@github.com:createrwebs/tora-mobile.git

# 2. Push current feature branch
git push -u tora tora/main:main

# 3. Verify remote HEAD matches d34fac0
git ls-remote tora refs/heads/main
```

Until this remote is configured, the local bundle `/Users/noppanan/tora-studio-lab/backups/flutter-lumenflow/lumenflow-tora-d34fac0.bundle` and associated patches serve as the authoritative off-scratch backup.


