---
commissioned-by: spacedock@test
mission: Fixture README with a malformed stage heading for AC-4 sad-path
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
    - name: done
      terminal: true
---

# Malformed Stage Heading Workflow

The `ideation` stage heading below is intentionally malformed — it places
trailing prose before the stage-name token, so `extract_stage_subsection`
must surface a parse error rather than silently failing.

## Stages

### Important note about ideation

A task moves to ideation when a pilot starts fleshing out the idea.
This heading mentions `ideation` as a substring but it is NOT the
first content token, so `extract_stage_subsection` must raise.
