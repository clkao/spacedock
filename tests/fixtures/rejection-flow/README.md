---
commissioned-by: spacedock@test
mission: Rejection flow test
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
    - name: implementation
      worktree: true
    - name: validation
      worktree: true
      fresh: true
      feedback-to: implementation
    - name: done
      terminal: true
---

# Rejection Flow Test Workflow

A minimal workflow for testing that the first officer detects validation rejections and dispatches an implementer to fix them.

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

- **Inputs:** None — this is the initial state
- **Outputs:** A seed task file with title, source, and brief description

### implementation

Write the code described in the task.

- **Inputs:** A task description with acceptance criteria
- **Outputs:** Working code committed to the worktree
- **Good:** Code that satisfies the acceptance criteria
- **Bad:** Code that does not match the described behavior

### validation

Verify the implementation meets the acceptance criteria.

- **Inputs:** The implementation and acceptance criteria from the task body
- **Outputs:** A PASSED/REJECTED recommendation with evidence
- **Good:** Thorough testing, clear evidence of pass/fail
- **Bad:** Rubber-stamping without testing

### done

Terminal stage. The task is complete.

## Testing Resources

| Resource | Path | Covers |
|----------|------|--------|
| Add test | `tests/test_add.py` | Verifies the add function returns correct sums |

## Commit Discipline

Prefix commits with the stage name: `implementation: did the thing`

## Task Template

```markdown
---
id: "{id}"
title: "{title}"
status: backlog
score: 0.50
source: test
started:
completed:
verdict:
worktree:
---

{description}
```
