---
commissioned-by: spacedock@test
mission: Fixture workflow for fetch-on-demand dispatch tests (entity 0x9)
entity-label: task
entity-label-plural: tasks
id-style: sequential
stages:
  defaults:
    worktree: false
    fresh: false
    gate: false
    concurrency: 1
  states:
    - name: backlog
      initial: true
    - name: ideation
    - name: implementation
    - name: done
      terminal: true
---

# Fetch-on-Demand Test Workflow

A fixture workflow used by `tests/test_claude_team.py` and
`tests/test_fetch_on_demand_dispatch.py` to exercise the helper's
fetch-on-demand dispatch shape. Stage definitions below mirror the real
`docs/plans/README.md` so size measurements match production scale.

## Stages

### `backlog`

A task enters backlog when it is first proposed. It has a seed description but no design work has been done yet.

- **Inputs:** None — this is the initial state
- **Outputs:** A seed task file with title, source, and brief description
- **Good:** Clear enough to understand what the task is about
- **Bad:** N/A — backlog is a holding state, not an action

### `ideation`

A task moves to ideation when a pilot starts fleshing out the idea: clarify the problem, explore approaches, and produce a concrete description of what "done" looks like.

- **Inputs:** The seed description and any relevant context (existing code, user feedback, related tasks)
- **Outputs:** A fleshed-out task body with problem statement, proposed approach, acceptance criteria, and a test plan
  - Acceptance criteria must include how each criterion will be tested
  - Acceptance criteria are **entity-level** — they describe properties of the finished task (end-state facts a future reader can verify), not stage actions. Items that describe stage work ("run X 3 times", "produce analysis Y") belong in the stage report's checklist, not in the AC list. If an AC item reads as an imperative verb phrase ("Run …", "Produce …", "Capture …"), rewrite it as the end-state property it produces ("Test X passes reliably", "Analysis Y concludes with cited evidence", "File Z contains string W").
  - Test plan: what tests verify the implementation, estimated cost/complexity, whether E2E tests are needed
  - Plans should describe intended behavior at the level a future worker or validator needs to reason about it. Prefer observable behavior over implementation internals unless the task is specifically about that internal representation.
  - Choose proof at the same abstraction level as the claim: static checks for durable doc/contract structure, parser or transcript fixtures for orchestration behavior, and live E2E only when real runtime behavior is the claim.
  - When captain feedback changes the target behavior, update the task body, acceptance criteria, and test plan together before re-validating.
  - For template changes: specific before/after wording, not just "change X"
- **Good:** Clearly scoped, behavior-first, actionable, addresses a real need, considers edge cases, avoids unnecessary runtime-internal modeling, and uses tests that prove the intended behavior directly
- **Bad:** Vague hand-waving, scope creep, solving problems that don't exist yet, no clear definition of done, acceptance criteria without a test plan, static prose tests for behavioral requirements, or tests that pass while missing the current intended behavior
- **Staff review:** When the FO assesses ideation as complex (touches scaffolding, requires E2E tests, or score >= 0.8), it spawns a fresh independent reviewer subagent before presenting at the ideation gate. The reviewer checks design soundness, test plan sufficiency, and gaps. The captain sees both the ideation and the reviewer's assessment.

### `implementation`

A task moves to implementation once its design is approved. The work here is to produce the deliverable: write code, run experiments, generate analysis, or make whatever changes the task describes. Implementation is complete when the deliverable exists and is ready for independent verification.

- **Inputs:** The fleshed-out task body from ideation with approach and acceptance criteria
- **Outputs:** The deliverable committed to the repo (code, experiment results, analysis, test suites — whatever the task specifies), with a summary of what was produced and where
- **Good:** Minimal changes that satisfy acceptance criteria, clean code, tests where appropriate, deliverable is self-contained and verifiable
- **Bad:** Over-engineering, unrelated refactoring, skipping tests, ignoring edge cases identified in ideation, leaving the deliverable incomplete for validation to finish

### `done`

Terminal state for completed tasks.
