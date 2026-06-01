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
  - Every task must produce a real, checkable change — code, a fixture, on-disk state, or instruction text whose effect a separate check can confirm — not just a document about itself. Each AC's "Verified by" must name something outside the task body that can fail: a test, a command's output or exit code, a file the change produces, or the resulting on-disk state. An AC whose only proof is reviewing the task's own prose ("verified by reviewing this task's decision section") can never fail, so it is not an acceptance criterion. If the task's only output is a decision with nothing shipped, it does not belong in this queue — record the decision in the roadmap instead. Cleanup and overhaul do qualify: the change is the new code plus passing tests.
  - When the design's soundness rests on an unverified mechanism — a parser round-trip, a runtime handoff, an on-disk format, a tool actually supporting a flag — try the riskiest unknown first: run the smallest end-to-end exercise of that path before committing to the rest of the plan, and record the result in the task body. Ask "what would invalidate the rest of this work if it broke?" — that goes first; pay the small bill first. The exercise is throwaway, but what it teaches seeds the implementation's first test. If nothing is unverified — the design only composes already-proven behavior — record "no spike needed: {the proven mechanisms it relies on}" so the determination is on the record rather than silent.
  - Test plans should state what verifies the implementation, estimated cost/complexity, and whether fixture, CLI, or live workflow tests are needed.
  - Plans should describe intended behavior at the level a future worker or validator needs to reason about it. Prefer observable behavior over implementation internals unless the task is specifically about that internal representation.
  - Choose proof at the same abstraction level as the claim, and prefer proof that exercises the behavior and observes the outcome — output bytes, exit code, resulting on-disk state, or a test feeding many inputs and asserting uniform handling: Go unit tests for parser and command behavior, golden fixtures for status output, behavior fixtures that drive the binary for command-level claims, and live workflow smoke tests only when runtime behavior is the claim. A substring search is not proof of behavior. Searching code asserts spelling (it false-passes on a renamed-but-equivalent branch and false-fails on a rename); searching instruction prose is weaker still — the document is not the behavior, so "the contract says to run the command" never proves the agent runs it. A static check is legitimate when it parses real artifacts and tests a relationship between real values (for example, that the plugin manifest's contract range brackets the binary's contract version), or when the claim itself is about the text — a presence check over instruction files proving they carry a required clause or stay free of a banned token is proof at the claim's own level. The line is: invariant over real values, or a property of the text when the text is the claim, is legitimate; a substring search standing in for a behavioral claim is not.
  - Prefer acceptance criteria a code gate can enforce — a guard in the binary, a test that fails on violation — over criteria the agent is merely instructed to follow. Where a behavior can be guarded by the binary or a failing test, the proof is that gate, not a sentence in a skill file. An AC whose only proof is "the instruction text says to do X" has a ceiling of wording-is-present and cannot stand on its own.
  - When captain feedback changes the target behavior, update the task body, acceptance criteria, and test plan together before re-validating.
  - For template or skill text changes: specific before/after wording, not just "change X".
- **Good:** Clearly scoped, behavior-first, actionable, addresses a real need, considers edge cases, avoids unnecessary runtime-internal modeling, and uses tests that prove the intended behavior directly
- **Bad:** Vague hand-waving, scope creep, solving problems that do not exist yet, no clear definition of done, acceptance criteria without a test plan, static prose tests for behavioral requirements, or tests that pass while missing the intended behavior
- **Staff review:** When the FO assesses ideation as complex, such as native status parity, split-root behavior, or skill integration, it should request an independent review before presenting the ideation gate. The review checks design soundness, test plan sufficiency, gaps, and that the riskiest unverified mechanism was exercised first (or that the task records an auditable "no spike needed" with the proven mechanisms it relies on). A design whose soundness rests on an unexercised, unverified mechanism is not ready for the gate.

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
- **Bad:** Rubber-stamping without testing, ignoring failing edge cases, validating against wrong criteria, accepting passing tests that encode stale prose, obsolete assumptions, or the wrong abstraction level, or accepting a substring search (over code or over instruction prose) as proof of a behavioral claim — proof of behavior must run the behavior; a static test passes only as an invariant over real parsed values, not as a spelling check
- **Spot-check principle:** Before committing to an expensive live workflow or compatibility run, do a cheap fixture or single-command spot-check to verify the infrastructure works end-to-end.

### `done`

A task reaches done when validation is complete and the captain approves the result. The task is closed with a verdict of PASSED or REJECTED.

- **Inputs:** The validation report with PASSED/REJECTED recommendation
- **Outputs:** Final verdict set in frontmatter, completed timestamp recorded
- **Good:** Clear resolution and lessons learned captured if relevant
- **Bad:** Closing without reading the validation report, overriding a REJECTED recommendation without reason, or reaching done with PASSED on a task whose deliverable is prose with nothing outside it that can fail (a design that concludes "do not build X" ships as a roadmap decision, not a PASSED dev-queue task)

## Workflow State

Workflow state is read from `.spacedock-state`. During the compatibility phase, use the current status script against the state checkout:

```bash
python3 /path/to/spacedock/skills/commission/bin/status --workflow-dir docs/dev/.spacedock-state
```

The target launcher command is:

```bash
spacedock status --workflow-dir docs/dev
```

To list the tasks ready for dispatch (the query the first officer runs each loop):

```bash
spacedock status --workflow-dir docs/dev --next
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

## Problem

{What is broken or missing, and why it matters. Ideation fills this in.}

## Proposed approach

{How the task intends to solve the problem. Ideation fills this in.}

## Out of scope

{What this task deliberately does not cover, so the boundary is explicit.}

## Acceptance criteria

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 - {End-state property.}**
Verified by: {test name / command output or exit code / file the change produces / resulting on-disk state — something outside this task body that a future reader can reproduce and that can fail.}

## Test plan

{What verifies the implementation, estimated cost/complexity, and whether fixture, CLI, or live workflow tests are needed.}
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
