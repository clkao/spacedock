---
commissioned-by: spacedock@0.13.0-dev
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: sd-b32
state: .spacedock-state
stages:
  defaults:
    worktree: false
    concurrency: 1
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

# Build Spacedock v1 - Go Launcher Workflow

Spacedock v1 is the Go launcher and compatibility bridge for the next Spacedock command surface. This workflow tracks design and implementation tasks from initial concepts through validated, shippable behavior.

Runtime entities live in `.spacedock-state`, a per-workflow state checkout. During bootstrap, `.spacedock-state/README.md` may symlink to this README so current status tooling can operate against the state checkout directly.

No PR merge flow, mods, or lifecycle hooks are in scope for this bootstrap workflow.

## File Naming

Each task is a folder or markdown file named `{slug}` or `{slug}.md` - lowercase, hyphens, no spaces. Use folder-form entities when reports or artifacts may accumulate beside the task. Example: `native-go-status/index.md`.

## Schema

Every task file has YAML frontmatter. Fields are documented below; see **Task Template** for a copy-paste starter.

### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique 24-character Spacedock Base32 ID because this workflow uses `sd-b32` |
| `title` | string | Human-readable task name |
| `status` | enum | One of: backlog, ideation, implementation, validation, done |
| `source` | string | Where this task came from |
| `started` | ISO 8601 | When active work began |
| `completed` | ISO 8601 | When the task reached terminal status |
| `verdict` | enum | PASSED or REJECTED - set at final stage |
| `score` | number | Priority score, 0.0-1.0 (optional). Workflows can upgrade to a multi-dimension rubric in their README. |
| `worktree` | string | Worktree path while a dispatched agent is active, empty otherwise |
| `issue` | string | Optional external ticket reference, such as `ENG-123`, `kata:task-abc123`, or `owner/repo#42` |

## Stages

### `backlog`

A task enters backlog when it is first proposed. It has a seed description but no design work has been done yet.

- **Inputs:** None - this is the initial state
- **Outputs:** A seed task file with title, source, brief description, acceptance criteria, and stage-specific test gates
- **Good:** Clear enough to understand what the task is about and what proof future stages must provide
- **Bad:** Mixing launcher, status, skill integration, and tracker work without a testable boundary

### `ideation`

A task moves to ideation when a pilot starts fleshing out the idea: clarify the problem, explore approaches, and produce a concrete description of what "done" looks like.

- **Inputs:** The seed description and any relevant context, including existing code, user feedback, related tasks, and current Spacedock behavior
- **Outputs:** A fleshed-out task body with problem statement, proposed approach, acceptance criteria, and a test plan
  - Acceptance criteria must include how each criterion will be tested.
  - Acceptance criteria are **entity-level** - they describe properties of the finished task, not stage actions. Items that describe stage work belong in the stage report's checklist.
  - If an AC item reads as an imperative verb phrase, rewrite it as the end-state property it produces.
  - Test plans should state what verifies the implementation, estimated cost/complexity, and whether fixture, CLI, or live workflow tests are needed.
  - Plans should describe intended behavior at the level a future worker or validator needs to reason about it. Prefer observable behavior over implementation internals unless the task is specifically about that internal representation.
  - Choose proof at the same abstraction level as the claim: Go unit tests for parser and command behavior, golden fixtures for status output, static skill tests for instruction text, and live workflow smoke tests only when runtime behavior is the claim.
  - When captain feedback changes the target behavior, update the task body, acceptance criteria, and test plan together before re-validating.
  - For template or skill text changes: specific before/after wording, not just "change X".
- **Good:** Clearly scoped, behavior-first, actionable, addresses a real need, considers edge cases, avoids unnecessary runtime-internal modeling, and uses tests that prove the intended behavior directly
- **Bad:** Vague hand-waving, scope creep, solving problems that do not exist yet, no clear definition of done, acceptance criteria without a test plan, static prose tests for behavioral requirements, or tests that pass while missing the intended behavior
- **Staff review:** When the FO assesses ideation as complex, such as native status parity, split-root behavior, or skill integration, it should request an independent review before presenting the ideation gate. The review checks design soundness, test plan sufficiency, and gaps.

### `implementation`

A task moves to implementation once its design is approved. The work here is to produce the deliverable: write code, generate fixtures, update skill instructions, or make whatever changes the task describes. Implementation is complete when the deliverable exists and is ready for independent verification.

- **Inputs:** The fleshed-out task body from ideation with approach and acceptance criteria
- **Outputs:** The deliverable committed to the relevant repo or state checkout, with a summary of what was produced and where
- **Good:** Minimal changes that satisfy acceptance criteria, clean Go packages, stable CLI output, tests where appropriate, and a self-contained deliverable
- **Bad:** Over-engineering, unrelated refactoring, skipping tests, ignoring edge cases identified in ideation, or leaving the deliverable incomplete for validation to finish

### `validation`

A task moves to validation after implementation is complete. The work here is to verify the deliverable meets the acceptance criteria defined in ideation. The validator checks what was produced - it does not produce the deliverable itself.

- **Inputs:** The implementation summary and the acceptance criteria from the task body
- **Outputs:**
  - Run applicable tests from the Testing Resources section and report results.
  - Verify each acceptance criterion with evidence.
  - Pull every `**AC-N**` item from the entity body's `## Acceptance criteria` section; reproduce the evidence cited in each "Verified by" clause; flag any AC without evidence.
  - Check that the task body, acceptance criteria, implementation, and tests reflect the latest captain feedback.
  - Reject when tests pass but prove an obsolete, over-specified, or wrong target behavior.
  - A PASSED/REJECTED recommendation.
- **Good:** Thorough testing against acceptance criteria, clear evidence of pass/fail, honest assessment, and validation that tests prove the current intended behavior
- **Bad:** Rubber-stamping without testing, ignoring failing edge cases, validating against wrong criteria, or accepting passing tests that encode stale prose, obsolete assumptions, or the wrong abstraction level
- **Spot-check principle:** Before committing to an expensive live workflow or compatibility run, do a cheap fixture or single-command spot-check to verify the infrastructure works end-to-end.

### `done`

A task reaches done when validation is complete and the captain approves the result. The task is closed with a verdict of PASSED or REJECTED.

- **Inputs:** The validation report with PASSED/REJECTED recommendation
- **Outputs:** Final verdict set in frontmatter, completed timestamp recorded
- **Good:** Clear resolution and lessons learned captured if relevant
- **Bad:** Closing without reading the validation report or overriding a REJECTED recommendation without reason

## Workflow State

Workflow state is read from `.spacedock-state`. During the compatibility phase, use the current status script against the state checkout:

```bash
python3 /path/to/spacedock/skills/commission/bin/status --workflow-dir docs/dev/.spacedock-state
```

The target launcher command is:

```bash
spacedock status --workflow-dir docs/dev
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
issue:
---

Brief description of this task and what it aims to achieve.

## Acceptance criteria

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 - {End-state property.}**
Verified by: {grep / test name / file path / command a future reader can reproduce.}
```

## Testing Resources

Validation pilots should use these when verifying implementation work:

| Resource | Command or Path | Covers |
|----------|-----------------|--------|
| Go unit suite | `go test ./...` | CLI routing, parser behavior, status implementation, fixtures |
| Race-enabled Go suite | `go test ./... -race` | Concurrency hazards in Go code when relevant |
| Launcher help smoke test | `go run ./cmd/spacedock --help` | Basic command entrypoint behavior |
| Launcher version smoke test | `go run ./cmd/spacedock --version` | Basic version output behavior |
| Current status validator | `python3 /path/to/status --workflow-dir docs/dev/.spacedock-state --validate` | Compatibility with current Spacedock entity contract |
| Current status table | `python3 /path/to/status --workflow-dir docs/dev/.spacedock-state` | Compatibility with current status output during symlink phase |
| State behavior extension | `docs/specs/state-behavior-extension.md` | Split-root state semantics and external tracker bridge principles |
| Bootstrap roadmap | `docs/roadmap/bootstrap-roadmap.md` | Stage-specific required tests |

Validators should pick the smallest test surface that proves the claim. Use Go unit tests for package behavior, golden fixtures for stable command output, status-script comparisons for compatibility claims, and live workflow smoke tests only when the runtime integration itself is the claim.

## Commit Discipline

- Commit state changes at dispatch and archive boundaries.
- Commit task body updates when substantive.
- Keep main repo changes and `.spacedock-state` changes in their respective git repositories.
