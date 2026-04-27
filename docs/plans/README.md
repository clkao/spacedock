---
commissioned-by: spacedock@0.10.2
entity-type: entity
entity-label: task
entity-label-plural: tasks
id-style: sequential
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
      gate: true
    - name: ideation
      gate: true
    - name: implementation
      worktree: true
    - name: validation
      worktree: true
      fresh: true
      feedback-to: implementation
      gate: true
    - name: done
      terminal: true
---

# Design and Build Spacedock - Plain Text Workflow for Agents

Spacedock is a Claude Code plugin that turns directories of markdown files into structured workflows operated by AI agents. This workflow tracks the design and implementation tasks for building Spacedock itself — from initial concepts through validated, shippable features.

## File Naming

Each task is a markdown file named `{slug}.md` — lowercase, hyphens, no spaces. Example: `pilot-worktree-isolation.md`.

## Schema

Every task file has YAML frontmatter. Fields are documented below; see **Task Template** for a copy-paste starter.

### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier, format determined by id-style in README frontmatter |
| `title` | string | Human-readable task name |
| `status` | enum | One of: backlog, ideation, implementation, validation, done |
| `source` | string | Where this task came from |
| `started` | ISO 8601 | When active work began |
| `completed` | ISO 8601 | When the task reached terminal status |
| `verdict` | enum | PASSED or REJECTED — set at final stage |
| `score` | number | Priority score, 0.0–1.0 (optional). Workflows can upgrade to a multi-dimension rubric in their README. |
| `worktree` | string | Worktree path while a dispatched agent is active, empty otherwise |
| `issue` | string | GitHub issue reference (e.g., `#42` or `owner/repo#42`). Optional cross-reference, set manually. |
| `pr` | string | GitHub PR reference (e.g., `#57` or `owner/repo#57`). Set when a PR is created for this entity's worktree branch. |
| `mod-block` | string | Pending mod-declared blocking action, format `{lifecycle_point}:{mod_name}`. Empty when no block is active. |

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

### `validation`

A task moves to validation after implementation is complete. The work here is to verify the deliverable meets the acceptance criteria defined in ideation. The validator checks what was produced — it does not produce the deliverable itself.

- **Inputs:** The implementation summary and the acceptance criteria from the task body
- **Outputs:**
  - Run applicable tests from the Testing Resources section and report results
    - Use `tests/README.md` to choose the right harness and entrypoint before running tests
    - Prefer the stable repo-level entrypoints when they fit the task: `make test-static` for the offline suite, `make test-live-claude` / `make test-live-codex` for tier-aware live runs, and `make test-e2e TEST=... RUNTIME=...` for single-file runtime-specific E2E checks
  - Verify each acceptance criterion with evidence
  - Pull every `**AC-N**` item from the entity body's `## Acceptance criteria` section; reproduce the evidence cited in each "Verified by" clause; flag any AC without evidence. Validation's job is cross-check, not re-derive.
  - Check that the task body, acceptance criteria, implementation, and tests reflect the latest captain feedback.
  - Reject when tests pass but prove an obsolete, over-specified, or wrong target behavior.
  - A PASSED/REJECTED recommendation
- **Good:** Thorough testing against acceptance criteria, clear evidence of pass/fail, honest assessment, and validation that the tests prove the current intended behavior
- **Bad:** Rubber-stamping without actually testing, ignoring failing edge cases, validating against wrong criteria, or accepting passing tests that encode stale prose, obsolete assumptions, or the wrong abstraction level
- **Spot-check principle:** Before committing to an expensive multi-run experiment or long test suite, do a cheap single-run spot-check (ideally on a smaller/cheaper model) to verify the infrastructure works end-to-end. Fix broken plumbing before burning budget on real runs.

### `done`

A task reaches done when validation is complete and CL approves the result. The task is closed with a verdict of PASSED or REJECTED.

- **Inputs:** The validation report with PASSED/REJECTED recommendation
- **Outputs:** Final verdict set in frontmatter, completed timestamp recorded
- **Good:** Clear resolution, lessons learned captured if relevant
- **Bad:** Closing without reading the validation report, overriding a REJECTED recommendation without reason

## Workflow State

View the workflow overview:

```bash
skills/commission/bin/status docs/plans
```

Output columns: ID, SLUG, STATUS, TITLE, SCORE, SOURCE.

Include archived tasks with `--archived`:

```bash
skills/commission/bin/status docs/plans --archived
```

Find dispatchable tasks ready for their next stage:

```bash
skills/commission/bin/status docs/plans --next
```

Find tasks in a specific stage:

```bash
grep -l "status: ideation" docs/plans/*.md
```

## Task Template

```yaml
---
id:
title: Task name here
status: backlog
source:
started:
completed:
verdict:
score:
worktree:
---

Brief description of this task and what it aims to achieve.

## Acceptance criteria

Each AC names a property of the finished entity (not a stage action) and how it is verified.

**AC-1 — {End-state property.}**
Verified by: {grep / test name / file path / command a future reader can reproduce.}
```

## Testing Resources

Validation pilots should use these when verifying implementation work:

| Resource | Path | Covers |
|----------|------|--------|
| Commission test harness | `scripts/test-harness.md` | Batch-mode commission invocation, generated file validation, guardrail checks |
| Rejection flow E2E test | `tests/test_rejection_flow.py` | Validation rejection detection, validator-to-implementer relay dispatch |
| Scaffolding guardrail E2E test | `tests/test_scaffolding_guardrail.py` | Scaffolding change guardrail, issue filing guardrail |
| Merge hook guardrail E2E test | `tests/test_merge_hook_guardrail.py` | Merge hook fires before local merge, no-mods fallback |
| Repo edit guardrail E2E test | `tests/test_repo_edit_guardrail.py` | FO write scope guardrail, code/test/mod edit rejection |
| Test authoring and execution guide | `tests/README.md` | Test infrastructure, stable entrypoints, CLI conventions, fixtures, and harness selection |

The test harness documents how to run `claude -p` with `--plugin-dir` for non-interactive commission testing, plus structural and guardrail assertions against the generated output. Use it for any task that changes `skills/commission/SKILL.md` or the first-officer template.

Validators should treat `tests/README.md` as the authoritative guide for selecting the right test surface and command shape. Not every validation run should use the same entrypoint: some tasks need the offline repo suite, some need runtime-specific E2E coverage, and some need both.

The stable repo-level offline suite is:

```bash
make test-static
```

Live E2E checks go through the pytest two-tier wrappers:

```bash
make test-live-claude                                # serial tier, then parallel tier
make test-live-codex
make test-live-claude-opus                           # same shape, --model opus --effort low
make test-e2e TEST=tests/test_gate_guardrail.py RUNTIME=codex   # single-file override
```

- `test-live-{claude,codex}` run the serial tier (`-m "... and serial" -x`) first, then the parallel tier (`-m "... and not serial" -n $LIVE_{CLAUDE,CODEX}_WORKERS`) regardless of the serial tier outcome. Overall result is the logical AND of both exit codes.
- `test-e2e` replaces the old `test-e2e-commission` target: pass `TEST=tests/test_commission.py` for the same effect.
- Use direct `uv run pytest …` invocations only when the test guide calls for a more specific command than the stable wrappers provide.

### Running E2E tests

Tests run under pytest. When running from inside a Claude Code session, unset `CLAUDECODE` first (Claude refuses to launch as a subprocess when this variable is set):

    unset CLAUDECODE && uv run pytest tests/test_output_format.py --runtime claude -v

This applies to every test invocation, including those behind the `make test-static` / `make test-live-*` / `make test-e2e` wrappers.

## Commit Discipline

- Commit status changes at dispatch and merge boundaries
- Commit task body updates when substantive
