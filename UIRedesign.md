We are going to redesign the entire New API web UI progressively, one page at a time.

Reference product:
https://flaq.ai

Goal:
Transform the existing New API web interface into a polished, modern AI SaaS product UI inspired by the visual quality, hierarchy, spacing, navigation patterns, and product feel of Flaq.ai.

IMPORTANT:
This is NOT a request to redesign any UI yet.

Your task in this phase is ONLY to inspect the existing frontend and create two planning/instruction files:

1. UI_REDESIGN_PLAN.md
2. .codex/skills/ui-redesign/SKILL.md

Do not modify existing application source files.
Do not modify React components.
Do not modify CSS.
Do not modify package.json.
Do not modify API code.
Do not modify backend code.
Do not modify database code.

==================================================
PHASE 1 — INSPECT THE EXISTING FRONTEND
==================================================

Thoroughly inspect the entire web frontend.

Start from:
- web/
- web/src/
- package.json
- routing configuration
- theme configuration
- global CSS
- shared components
- layouts
- pages
- hooks
- API/service clients
- state management
- authentication UI
- tables/forms/modals
- existing design tokens

Identify:

1. Frontend framework and architecture
2. Routing architecture
3. Layout architecture
4. UI/component library
5. Styling approach
6. Theme/dark-mode implementation
7. Shared components
8. Page-specific components
9. Reusable patterns
10. Duplicate UI patterns
11. Existing design tokens
12. Responsive behavior
13. Major technical debt affecting UI consistency

Build a complete inventory of existing routes/pages.

For every page identify:
- route
- purpose
- main components
- important interactions
- data sources/API calls
- shared components used
- page-specific components
- current UI problems
- estimated redesign complexity
- dependencies on shared layout/components

Do not guess.
Base the analysis on the actual source code.

==================================================
PHASE 2 — DEFINE THE TARGET DESIGN SYSTEM
==================================================

Study the visual language of Flaq.ai as a reference.

Do NOT copy:
- Flaq branding
- logo
- proprietary assets
- source code
- exact text
- proprietary illustrations
- exact page implementation

Instead extract general design principles such as:
- modern AI SaaS aesthetic
- information hierarchy
- spacing
- typography
- navigation
- cards
- controls
- tables
- forms
- status indicators
- empty states
- loading states
- responsive behavior

Then define a ToraAPI-specific design system.

The design system should cover:

- color tokens
- typography
- spacing scale
- border radius
- shadows
- borders
- surfaces
- buttons
- inputs
- selects
- tabs
- badges
- tables
- cards
- dialogs
- dropdowns
- tooltips
- alerts
- toast
- loading states
- empty states
- error states
- navigation
- sidebar
- header
- page headers
- content containers

Prioritize reuse and consistency.

==================================================
PHASE 3 — CREATE UI_REDESIGN_PLAN.md
==================================================

Create:

UI_REDESIGN_PLAN.md

It must contain:

# ToraAPI UI Redesign Plan

## 1. Objective

Explain the redesign goal.

## 2. Current Frontend Architecture

Describe the actual architecture discovered from the repository.

## 3. Current UI Inventory

Create a table containing:

| Route | Page | Complexity | Current Problems | Dependencies | Priority |

Include every important frontend route.

## 4. Target Design Direction

Describe the ToraAPI visual direction inspired by modern AI SaaS products and Flaq.ai.

## 5. Design System

Define the target design tokens and reusable UI patterns.

## 6. Shared Components

List components that should be created, refactored, or standardized.

For example:

- AppShell
- Sidebar
- Header
- PageHeader
- Section
- Card
- DataTable
- FilterBar
- SearchInput
- StatusBadge
- EmptyState
- LoadingState
- ConfirmDialog
- FormField
- PrimaryButton

Only include components that make sense based on the actual codebase.

## 7. Page Redesign Order

Create a dependency-aware implementation order.

Do NOT simply order pages alphabetically.

Foundation must come first.

Example structure:

Phase 0:
- Design tokens
- Theme
- App shell
- Sidebar
- Header
- shared components

Phase 1:
- Dashboard

Phase 2:
- Core administration pages

Phase 3:
- Budget / billing / management pages

Phase 4:
- Settings / authentication

Phase 5:
- Responsive and visual polish

Adapt this to the actual repository.

## 8. Per-Page Definition of Done

Define what must be checked before a page is considered complete.

Include:
- visual consistency
- existing functionality preserved
- API behavior preserved
- permissions preserved
- loading state
- empty state
- error state
- responsive behavior
- accessibility
- build
- lint/type checks

## 9. Rules

Explicitly state:

- Do not change backend behavior.
- Do not change API contracts.
- Do not change database schema.
- Do not change authentication behavior.
- Do not change business logic unless explicitly requested.
- Do not redesign multiple unrelated pages in one task.
- Reuse shared components.
- Avoid duplicated page-specific components.
- Do not introduce unnecessary dependencies.

## 10. Recommended Implementation Workflow

Describe the workflow:

inspect → plan → implement → validate → review → commit

==================================================
PHASE 4 — CREATE SKILL.md
==================================================

Create:

.codex/skills/ui-redesign/SKILL.md

This file is an operational instruction for future Codex UI redesign tasks.

It must contain concise but strong rules.

The skill should instruct Codex to:

1. Read UI_REDESIGN_PLAN.md first.
2. Inspect the target page before editing.
3. Inspect relevant shared components.
4. Understand existing data/API flow.
5. Preserve functionality.
6. Reuse the design system.
7. Avoid unnecessary refactoring.
8. Modify only the requested page and required shared components.
9. Never silently change backend behavior.
10. Validate build/type/lint.
11. Check responsive behavior.
12. Check loading/empty/error states.
13. Report changed files.
14. Report any remaining visual or technical issues.

Include a standard workflow:

## Workflow

### Step 1 — Understand
### Step 2 — Inspect
### Step 3 — Plan
### Step 4 — Implement
### Step 5 — Validate
### Step 6 — Report

Include a strict "Do Not" section.

==================================================
IMPORTANT OUTPUT REQUIREMENTS
==================================================

Before creating the files:

- Inspect the repository thoroughly.
- Do not invent routes or components.
- Do not assume the frontend architecture.
- Do not modify application source code.

After creating the two files:

1. Show a concise summary of what you discovered.
2. Show the number of routes/pages identified.
3. Show the recommended redesign phases.
4. Show exactly which files were created.
5. Confirm that no application source files were modified.

Do not implement any UI changes in this task.