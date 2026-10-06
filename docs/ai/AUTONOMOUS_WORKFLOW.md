# Tora AI — Autonomous Engineering Workflow

## Purpose

Allow Antigravity to operate continuously as an engineering organization rather than a single coding agent.

## Roles

At minimum, instantiate logically independent roles for critical work:

### Orchestrator / Engineering Lead

- reads project brain
- selects next executable task
- decomposes work
- assigns ownership
- integrates results
- cannot self-approve all critical work

### Backend Engineer

- New-API controllers/services/models
- transactions
- webhooks
- billing/quota
- BYOK server work

### Flutter Engineer

- API client
- native state machines
- UI
- lifecycle
- secure storage
- browser/deep-link integration

### API / Integration Engineer

- docs/source capability audit
- contract verification
- Postman/API tests
- browser provider flows

### Security Reviewer

- auth
- token/secret handling
- cross-account/environment isolation
- SSRF/webhook/signature attacks
- proof/log leakage

### Financial / Billing Reviewer

- idempotency
- duplicate credit
- refund/revocation
- renewal races
- quota reset
- ownership binding

### QA / Adversarial Engineer

- edge cases
- race tests
- restore/reinstall
- offline behavior
- malformed response handling
- regression suite

### Release Reviewer

- verifies gates and evidence
- distinguishes real E2E from simulated tests
- issues final phase status

## Work Selection Algorithm

At the beginning of a cycle:

```text
read CURRENT_STATE
-> read ROADMAP
-> inspect git/source/tests
-> identify blockers
-> choose highest-priority task that is executable now
```

If a top-priority task is externally blocked, record the blocker and continue with the next independent task.

Do not stall the whole project because Store Console access is unavailable.

## Implementation Loop

```text
INSPECT
-> PLAN
-> DELEGATE
-> IMPLEMENT
-> TARGETED TESTS
-> INDEPENDENT REVIEW
-> SECURITY/ADVERSARIAL REVIEW
-> REMEDIATE
-> FULL REGRESSION
-> DOCUMENT
-> SELECT NEXT TASK
-> REPEAT
```

## Defect Handling

Every reviewer finding must be tracked as:

```text
finding
-> severity
-> affected files
-> reproduction / evidence
-> remediation owner
-> fix
-> test
-> reviewer confirmation
```

Never delete, skip, weaken, or rewrite a meaningful test merely to obtain PASS.

## Evidence Standard

For every completed task record:

- files changed
- contract changed or unchanged
- tests run
- exact pass/fail outcome
- unresolved risks
- external blockers

For store/payment lifecycle tests, label evidence as one of:

- REAL STORE E2E
- REAL BACKEND E2E
- SIMULATED
- UNIT/INTEGRATION ONLY
- NOT RUN

## Default Regression Commands

Backend:

```bash
go test ./... -count=1
go vet ./...
```

Use focused package tests first when debugging.

Mobile:

```bash
flutter pub get
flutter analyze
flutter test
flutter build apk --debug
```

Run iOS compile/build where Apple tooling is available.

## Git Discipline

- inspect `git status` before work
- preserve unrelated user changes
- do not reset or overwrite unrelated modifications
- local commits are allowed when coherent and tests pass
- do not push/merge/release if that can trigger production without operator approval

## Prime Directive: External Blocker != Session Stop

An external/operator blocker is NOT a reason to end the autonomous engineering session.

Every roadmap item must have one status:
- `EXECUTABLE_NOW`
- `IN_PROGRESS`
- `COMPLETED`
- `OPERATOR_BLOCKED`
- `TECHNICALLY_BLOCKED`
- `DEFERRED`

If the highest-priority item is `OPERATOR_BLOCKED`:
1. Mark that task `OPERATOR_BLOCKED`.
2. Record the exact unblock requirement.
3. Continue immediately with the highest-priority release-related task that is `EXECUTABLE_NOW` without the operator.
4. Keep progressing until there are genuinely no safe executable release tasks remaining.

Do not wait passively.
Do not repeatedly report the same blocker.
Do not invent fake completion.

### Stop Conditions

Only stop when:
- all remaining release-relevant tasks are operator-blocked (`AUTONOMOUS WORK EXHAUSTED — WAITING FOR OPERATOR`), or
- an unresolved Critical/High security defect prevents safe progress, or
- continuing would require an irreversible production/policy decision.

### Anti-Scope-Creep Guardrail

Do not create unrelated feature scope merely to remain busy.
Continued work must contribute directly to:
- release readiness
- security
- reliability
- testability
- deployment
- observability
- operational readiness

## Documentation Maintenance

`CURRENT_STATE.md` must always answer:

- what is complete?
- what is blocked?
- what is currently being worked on?
- what should run next?
- what is not release-ready yet?

`ROADMAP.md` must remain ordered and actionable.
