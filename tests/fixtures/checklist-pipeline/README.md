---
mission: Checklist protocol e2e
commissioned-by: spacedock@test
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
    - name: work
    - name: done
      terminal: true
---

# Checklist Protocol E2E Workflow

A minimal workflow for verifying that the first-officer dispatch prompt carries a
completion checklist and that the ensign accounts for that checklist in a Stage
Report.

## File Naming

Kebab-case slug: `my-task.md`

## Schema

```yaml
---
id: "001"
title: Short description
status: backlog
score: 0.50
source: test
started:
completed:
verdict:
worktree:
---
```

## Stages

### backlog

The initial holding stage.

- **Inputs:** A task description
- **Outputs:** The task exists with status backlog

### work

The task is actively being worked on.

- **Inputs:** A task in backlog
- **Outputs:** A concrete deliverable plus evidence captured in the entity body
- **Good:** AC are satisfied and verifiable from the entity file
- **Bad:** Missing deliverable, missing evidence, or missing stage report protocol

### done

Terminal stage.

## Commit Discipline

Prefix commits with the stage name: `work: did the thing`

