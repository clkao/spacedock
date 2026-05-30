---
commissioned-by: spacedock@0.13.0-dev
entity-type: development_task
entity-label: Task
entity-label-plural: Tasks
id-style: slug
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
      gate: true
      fresh: true
      feedback-to: implementation
    - name: done
      terminal: true
---

# Spacedock v1 Development Workflow

This workflow tracks the compatibility-first Go launcher migration.

Runtime entities live in `.spacedock-state`, a per-workflow state checkout. During bootstrap, `.spacedock-state/README.md` may symlink to this README so current status tooling can operate against the state checkout directly.

No PR merge flow, mods, or lifecycle hooks are in scope for this bootstrap workflow.

## Stages

### `backlog`

Inputs:
- A proposed migration task or design checkpoint.

Outputs:
- A clear problem statement.
- Acceptance criteria.
- Stage-specific test gates.

Good:
- The entity names the exact compatibility behavior it protects.

Bad:
- The entity mixes launcher, status, and skill changes without a testable boundary.

### `ideation`

Inputs:
- A backlog entity.

Outputs:
- A proposed implementation shape.
- Edge cases and migration risks.
- A narrowed test list.

Good:
- The design preserves current Spacedock behavior before adding split-root behavior.

Bad:
- The design requires mods, PR flow, or external tracker writeback.

### `implementation`

Inputs:
- An approved design.

Outputs:
- Code or documentation changes in the assigned worktree.
- Focused tests for the behavior being changed.
- Entity-local reports or artifacts when useful.

Good:
- The implementation has small packages and stable CLI output.

Bad:
- The implementation changes skill instructions before the launcher behavior is tested.

### `validation`

Inputs:
- Completed implementation work and its report.

Outputs:
- Re-run evidence for every required test.
- Any regression notes.
- A pass/reject recommendation.

Good:
- Validation uses commands a future maintainer can run locally.

Bad:
- Validation cites only implementation claims without re-running tests.

### `done`

Inputs:
- Accepted validation.

Outputs:
- A final record suitable for archive.

Good:
- The entity can be archived without losing reports or artifacts.

Bad:
- The entity reaches done with missing test evidence.
