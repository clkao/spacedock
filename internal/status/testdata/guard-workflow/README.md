---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: sequential
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
    - name: done
      terminal: true
---

# Sequential Fixture Workflow

A frozen fixture workflow used to pin status read/mutation parity. Entities have
distinct stages and scores so table ordering by (stage_order, -score) is
unambiguous; one entity has an empty score to lock the "empty sorts last" rule.
