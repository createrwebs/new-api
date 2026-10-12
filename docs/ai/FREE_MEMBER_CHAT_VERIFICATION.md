# Free member chat — verification, 2026-10-12

Status: **LOCAL BUILD PASS — LIVE ACTIVATION BLOCKED**.

Worktree: `/Users/noppanan/new-api`, branch `main`, base HEAD `580583674a51b778c5daf90f2a6124ac4d30a13c` with uncommitted changes. No commit, push, database mutation, admin configuration write or deployment performed.

## Delivered

- Durable operator requirement in `PRODUCT_DECISIONS.md`: new web members can use managed Thai chat without buying a plan, topping up or supplying a provider key.
- Native disabled channel payload `FREE_MEMBER_CHAT_CHANNEL.json` and activation/rollback instructions `FREE_MEMBER_CHAT_RUNBOOK.md`; preserve unrelated model prices and use version-checked per-model pricing updates.
- Existing free account counter now also protects web playground. One counter across web chat, chat API and Responses API; 20 admitted attempts/day, Bangkok boundary, existing paid/admin/BYOK exceptions preserved.
- Frontend compilation repairs: unused imports, duplicate Studio video flag, unsupported Button `asChild` replaced with the existing native Button/input ref and router `render` composition, valid wallet route, and attribution event type aligned with existing backend telemetry. Existing status colors preserved. No new dependency/service.

## Commands actually run

All Go tests below used isolated test SQLite; none contacted a production DB.

| Command / check | Observed result |
| --- | --- |
| `rtk proxy bun run build:check` in `web` before repairs | FAIL: unused imports, duplicate variable, Button composition, attribution event types and wallet route |
| Same command after repairs | PASS: TypeScript + Rsbuild production build, built in 4.67s; output `web/dist`, reported total 66,776.4 kB / gzip 20,332.9 kB |
| `rtk proxy go test ./middleware -run 'TestFreeTierQuota(Shared\|Playground)' -count=1` before guard fix | FAIL reproduced: web bypassed the allowance, 204 instead of 429 in both directions |
| `rtk proxy go test ./middleware ./router ./model -run '(FreeTier\|Playground\|BYOKBypasses.*FreeTier)' -count=1` after fix | PASS all three packages |
| `rtk proxy go test -race ./middleware ./model -run '(FreeTier\|BYOKBypasses.*FreeTier)' -count=1` | PASS both packages; linker emitted macOS LC_DYSYMTAB warnings, no race findings |
| `rtk proxy go test ./middleware ./router ./model -count=1` | Middleware/model PASS; router FAIL in existing `TestAccessTokenRouteRulesCoverEveryDashboardRoute` |
| Original HEAD `router/relay-router.go` supplied via Go `-overlay`, same route-coverage test | FAIL again, 88 missing static-route declarations. Confirms the new playground guard did not introduce this release-gate failure. No production auth policy modified to make it pass |
| `rtk proxy bun run test src/features/studio/__tests__ src/features/wallet/__tests__/copy.test.tsx` in `web` | PASS: 4 files, 10 tests. New navigation fixture repaired to use a real DRAFT plan before final pass |
| Targeted `rtk proxy bunx oxlint -c .oxlintrc.json` on changed frontend files and new tests | PASS with 3 pre-existing no-explicit-any warnings |
| `rtk proxy bunx oxfmt --write src/features/studio/__tests__` | PASS. Formatting check on the 8 changed production frontend files remains FAIL; original HEAD copies of all 8 also fail. Broad formatting rewrite deferred to preserve a small diff |
| Existing homepage layout test checked by frontend development agent | 4 baseline failures on stale labels/category-grid/headline expectations; not relabelled PASS or rewritten to fit current marketing |
| `rtk git diff --check` | PASS |
| Channel JSON parse/bounds/mapping assertions | PASS: HTTPS origin, type 1, default group, status 2, mapping covers both logical models |
| `rtk proxy env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOWORK=off go build -o bin/new-api-free-chat-linux-arm64 .` | PASS: statically linked ELF ARM aarch64; Linux runtime execution NOT RUN |

The independent reviewer separately ran 7 free-tier middleware tests, the real-router authenticated-session exhaustion test, and 3 paid/admin/BYOK exception tests: all PASS. No new blocking defect found in the guard, channel payload or runbook. This is development/source review with isolated local tests, not public-site QA acceptance.

## Real upstream probes

No member/provider credentials used. Both requests used one user message `ตอบคำว่า สวัสดี เท่านั้น`, `max_tokens: 16`, `stream: true`, bounded by curl timeouts; no automatic retries.

| Probe | Observed result |
| --- | --- |
| `rtk proxy curl --max-time 15 -i https://self-begun-upgrading-retreat.trycloudflare.com/v1/models` | HTTP/2 200; Typhoon :latest, tora-qwen :latest, qwen2.5:14b |
| `rtk proxy curl --silent --show-error --max-time 45 -i https://self-begun-upgrading-retreat.trycloudflare.com/v1/chat/completions -H 'Content-Type: application/json' -d '{"model":"scb10x/llama3.1-typhoon2-8b-instruct:latest","messages":[{"role":"user","content":"ตอบคำว่า สวัสดี เท่านั้น"}],"max_tokens":16,"stream":true}'` | HTTP/2 200, text/event-stream; real content สวัสดี, finish_reason stop, [DONE] |
| Same request with `tora-qwen:latest`, `--max-time 60` | HTTP/2 200, text/event-stream; real content สวัสดี, finish_reason stop, [DONE] |
| `rtk proxy curl --max-time 15 -I https://toraapi.com` and `https://www.toraapi.com/api/status` | curl exit 35 (TLS connection failed on this network) |
| HTTP toraapi.com and `http://172.198.56.245:11434/api/tags` | HTTP 503 with AXONS Access Blocked policy page, not an origin health result |

This confirms real upstream replies, not streaming latency, concurrency capacity, model quality, Thai-only guarantees, throughput of 4.2 words/s, hardware/RAM/swap size or Kaggle GPU inference. Cloudflare Quick Tunnels remain documented as testing-only/unsupported for SSE despite complete probe responses; see the official source in the runbook.

## Build artifact

Path: `/Users/noppanan/new-api/bin/new-api-free-chat-linux-arm64`

Size: 161,109,744 bytes.

SHA-256: `b21216d591e5f700925f04b96b35df146d3baac576ac6e6e3c8f1563c6e1aa3e`.

Linux ARM64 was chosen from the project's documented EC2 release topology, not a live host inspection in this turn. Verify current production architecture before deployment. Runtime image still needs the project's CA certificates/timezone/fonts/licenses; a successful cross-build does not prove a deployed container.

## Remaining gates / exact next frontier

1. Remediate the independently reproduced baseline PAT route-coverage failure and run its regression; do not weaken auth rules or skip the test to obtain PASS. Local diagnostic command: `rtk proxy go test ./router -run '^TestAccessTokenRouteRulesCoverEveryDashboardRoute$' -count=1`.
2. Coordinator must use an approved network/deployment/admin session that can reach toraapi.com. No credential-file/SSH-key/production-env search from this task, no network-policy bypass, and no TLS-verification disabling.
3. Replace the temporary, unauthenticated upstream with a stable HTTPS origin that supports SSE and restricts callers; tighten the user-reported Any-source NSG 11434 rule. Do not expose Ollama directly as member authentication.
4. Apply the native channel and zero-price/pricing/preconsume settings following the runbook; global free-preconsume setting requires reviewing existing zero-price models and preserving its previous value.
5. Run signup → zero-wallet member login → both models → completed web chat → wallet remains 0 → shared limit / anonymous / paid/BYOK regression on the live site in a separate QA conversation.

Production deployment, channel/pricing activation and public new-member chat: **NOT RUN / BLOCKED**. No working member URL claimed from builds, unit tests or direct upstream replies.
