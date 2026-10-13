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


**Pull requests:** When creating a pull request:

- First compare the current git user (`git config user.name` / `git config user.email`) with the repository's historical core developers, such as the recurring top authors in `git log`. Do not change git config.
- If the current git user is not one of those historical core developers, explicitly state in the PR body that the code was AI-generated or AI-assisted.
- When the pull request is created for the project owner, use the ordinary human PR template: `.github/PULL_REQUEST_TEMPLATE.md` for Chinese requests or `.github/PULL_REQUEST_TEMPLATE/en.md` for English requests. Project-owner pull requests MUST NOT use `.agents/github/PR.md` unless the owner explicitly asks for it.
- For all other agent-created pull requests, fill `.agents/github/PR.md` as the entire PR body. Do not use the ordinary human PR templates unless the project owner explicitly requests one. In User request, quote the user's request to the agent as faithfully as possible; do not rewrite or summarize it. Keep the body short and factual; do not paste unfiltered AI-generated text in the PR body or in later comments. Verification must be commands actually run and observed results, not only a statement that `go build` or tests passed. If any required condition is not met, tell the user and do not open the PR.


<!-- BEGIN AWS Agent Toolkit rules -->
# AWS Guidance for the new AWS experience

This user has signed up for the new AWS experience. This experience lets you sign into AWS using a social provider and requires the following additional context.

Where this guidance conflicts with the project's own instructions, the project's instructions take precedence.

## Context

### Terminology:

- Say "project" instead of "account" — a project contains an AWS account and settings for sharing with other collaborators
- Say "team member" instead of "IAM user" — users are invited by email, not created or federated in IAM
- Say "AWS Settings" when referring to management tasks at [settings.aws.com](https://settings.aws.com/) (project management, billing, team members, spend limits). Users view their actual AWS resources in the AWS Management Console.
- Say "selected Region" when referring to the user's Region — not "home Region"
- The user has a managed IAM experience. This includes a managed service control policies (SCP) and resource control policies (RCP) that govern the use of AWS. They will still need to use IAM to create policies to let services work with each other. If there are questions about the SCPs or RCPs, go to the documentation at https://docs.aws.amazon.com/accounts/latest/reference/scps-and-rcps-for-projects.html

### Constraints:

- All projects share a single AWS Region determined by the user's contact address. Resources cannot be created in other Regions
- When developing:
  - MUST create all Regional resources in the project's assigned Region
  - You CAN create AWS WAF and Cloudwatch Logs resources in us-east-1 when there are global resources (like a global WAF instance) that require a connection to dependencies in us-east-1. You should not use these for any other reason, because resources in the selected Region will provide lower cost (due to no cross-Region traffic), increased availability (due to no cross-Region traffic), and easier manageability (due to not needing to look in another Region). When you need to do an inventory of resources, you need to look in both the selected Region and us-east-1 for Cloudwatch Logs or WAF resources.
  - MUST NOT attempt to create Lambda, API Gateway, or other Regional resources in any other Region
  - MUST direct users to confirm their Region in AWS Settings > View all projects > Overview > Additional Info > Region. If the user cannot confirm their Region, check in ~/.aws/config
  - MUST NOT use Lambda@Edge — excluded from both Lambda and CloudFront
  - MUST NOT use CloudFormation StackSets — no multi-account or multi-Region deployments
  - MUST NOT attempt cross-Region actions — no cross-Region replication for DynamoDB/S3/RDS, no multi-Region KMS keys
  - MUST NOT use Route 53 cross-Region routing — geolocation, latency-based, and failover routing policies are not available
  - CloudFront is a global service and its actions ARE allowed in `us-east-1`. A user can create a CloudFront distribution pointing to their project-region Lambda function URL or API Gateway. However, Lambda and API Gateway themselves MUST NOT be created in `us-east-1` — they must be in the project Region.
  - Reduced availability in `eu-north-1` specifically: Amazon Rekognition, Amazon Textract, Amazon Personalize, AWS App Runner are not available in that Region.
- IAM permissions for human access are managed by AWS. Don't assign roles to team members unless absolutely necessary
- The user may have a spend limit if they are on the paid plan. The limit that pauses their project if it's exceeded. If resources suddenly become inaccessible, ask if they have a spend limit configured. Only project owners can modify a spend limit.
- When developing:
  - MUST ask about spend limit status if the user reports sudden "Access Denied" errors on operations that previously worked
  - MUST direct users to check spend status in AWS Settings > Billing
  - MUST check if a user has upgraded their account to the paid plan
  - MUST ask the user if they want to clean up the successfully created resources or keep them to reduce cost
- The user sets up billing, creates spend limits, and retrieves and pays invoices in AWS Settings. The user creates budgets and optimizes their costs in the AWS Billing and Cost Management console
- Not all AWS services are available. If a service isn't working, do the following:
  1. Run the command `aws freetier get-account-plan-state`
  2. If accountPlanType": "FREE", check the [Free Tier supported services list](https://docs.aws.amazon.com/accounts/latest/reference/supported-services-sign-up-new.html#supported-services-free-tier) next,
  3. If accountPlanType": "PAID", check the [Paid Tier supported services list](https://docs.aws.amazon.com/accounts/latest/reference/supported-services-sign-up-new.html#supported-services-paid-plan).
  4. If neither list shows the service, check the [Not supported for this experience list](https://docs.aws.amazon.com/accounts/latest/reference/supported-services-sign-up-new.html#unsupported-services). The user will need to activate advanced features to access this service.
- Users can activate advanced AWS services and capabilities for their account.
- Before starting a task, check whether a relevant AWS skill is available. Load the skill with retrieve_skill and prefer its guidance over general knowledge.

### Help level

- help_level (required): LOW, MEDIUM, or HIGH. While a user is building, you MUST ask the user: "How much guidance would you like from me? Low (I only flag security risks), medium (I ask a couple of clarifying questions if something seems off), or high (I explain what I'm doing, suggest alternatives, and flag best practices)."

You CAN update this rule file to save a user's help_level.

Constraints for each level:

**LOW:**

- MUST follow all constraints in this context file
- MUST execute the user’s request without modification
- MUST NOT ask clarifying questions unless the action would create a security vulnerability
- MUST NOT suggest alternatives or improvements

**MEDIUM:**

- MUST execute the user's request
- MAY ask up to two clarifying questions per task if the request has an ambiguity or a potential issue
- MUST NOT repeat a question or suggestion the user has already dismissed
- MUST NOT explain trade-offs or alternatives unless the user asks

**HIGH:**

- MUST explain what each step does and why before executing it
- MUST suggest alternatives when a better approach exists
- MUST flag best practices and explain trade-offs
- MUST still execute the user's choice if they disagree with a suggestion
<!-- END AWS Agent Toolkit rules -->
=======
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

