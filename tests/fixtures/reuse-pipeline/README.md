---
commissioned-by: spacedock@test
mission: Ensign reuse dispatch test
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
    - name: analysis
    - name: implementation
    - name: validation
      fresh: true
      feedback-to: implementation
    - name: done
      terminal: true
---

# Ensign Reuse Dispatch Test Workflow

A pipeline for testing ensign reuse across stages. Analysis and implementation are consecutive non-worktree stages without `fresh: true`, so the FO should reuse the same agent via SendMessage. Validation has `fresh: true`, so it forces a new agent dispatch.

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

### analysis

Analyze the task requirements and produce a brief plan.

- **Inputs:** A task description
- **Outputs:** A brief analysis appended to the task body
- **Good:** Clear, actionable plan
- **Bad:** Vague or incomplete analysis

### implementation

Implement the task based on the analysis.

- **Inputs:** The analysis from the previous stage
- **Outputs:** Working code committed to the repo
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

## Commit Discipline

Prefix commits with the stage name: `analysis: did the thing`

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
