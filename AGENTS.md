# Tora AI — Agent Operating Contract

This repository is operated as an autonomous engineering project. Agents may plan, implement, test, review, remediate, and continue to the next executable task without asking the operator after every micro-step.

## Read Order (mandatory at the start of every session)

1. `docs/ai/PROJECT_CONTEXT.md`
2. `docs/ai/PRODUCT_DECISIONS.md`
3. `docs/ai/SECURITY_INVARIANTS.md`
4. `docs/ai/CURRENT_STATE.md`
5. `docs/ai/ROADMAP.md`
6. `docs/ai/AUTONOMOUS_WORKFLOW.md`
7. `.agents/MASTER_ORCHESTRATOR_PROMPT.md`

Then inspect the actual source tree and git status before making changes.

## Source-of-Truth Hierarchy

When sources disagree, use this order:

1. Current source code and runtime behavior
2. Current automated tests
3. Current database/schema/config code
4. `docs/ai/*` project brain
5. Phase reports / historical audit artifacts
6. Upstream New-API documentation
7. Assumptions

Never implement a critical behavior from documentation alone if current source differs.

## Architecture Boundary

Tora AI architecture remains:

```text
Flutter mobile
    -> Tora/New-API directly
        -> managed system channels
        -> server-managed BYOK providers
        -> Apple / Google verification for native subscription
        -> existing New-API Stripe / TopUp stack for web/external billing
```

No BFF is introduced unless a concrete, reviewed requirement proves it is necessary.

## Autonomy

Agents are authorized to fix without operator approval:

- bugs
- failing tests
- race conditions
- parsing/state-machine defects
- missing validation
- lifecycle issues
- missing test coverage
- safe refactors
- code organization
- non-breaking security hardening within frozen policy
- documentation drift

Agents must stop and request operator approval before:

- changing price, quota, product tier, bundle/package ID, or production API hostname
- destructive production database operations
- rotating or exposing production secrets
- changing customer billing policy
- irreversible Store Console commercial actions
- releasing/publishing to production
- merging to protected branches if that can trigger production deployment
- weakening an existing security invariant to make tests pass

## Critical Review Rule

The author of critical code must not be the sole reviewer of that code.

Critical domains include:

- authentication
- authorization
- entitlement
- payment
- store verification
- webhooks
- quota / financial settlement
- BYOK credential handling
- secrets
- account/environment binding

Required flow:

```text
author -> independent reviewer -> tests -> adversarial reviewer -> remediation -> re-review
```

## Never Fake Completion

Do not label simulated, mocked, or unit-tested store billing as real store E2E.

Real Apple/Google completion requires actual store infrastructure and must be marked exactly as such.

## Session Completion

At the end of every successful work cycle:

1. Update `docs/ai/CURRENT_STATE.md`
2. Update `docs/ai/ROADMAP.md`
3. Record newly frozen decisions or unresolved risks
4. Run appropriate regression suites
5. Leave the repository in a coherent state
6. Select the next highest-priority executable task
7. Continue automatically unless an external/operator blocker is reached


The Master Orchestrator MUST NOT end an autonomous run while any safe
release-relevant task is marked EXECUTABLE_NOW or IN_PROGRESS.