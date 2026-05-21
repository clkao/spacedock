---
commissioned-by: spacedock@test
mission: Fixture workflow with no standing teammates for fetch-on-demand tests
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

# Fetch-on-Demand Test Workflow (no standing teammates)

Sibling of `fetch_on_demand_workflow/` with no `_mods/` directory. Used to
assert that `cmd_build` omits the `show-standing` fetch line when no
declared standing teammates exist.

## Stages

### `backlog`

Initial holding state.

### `ideation`

Flesh out the task.

### `implementation`

A task moves to implementation once its design is approved.

- **Inputs:** The fleshed-out task body
- **Outputs:** The deliverable committed
- **Good:** Minimal changes
- **Bad:** Over-engineering

### `done`

Terminal state.
