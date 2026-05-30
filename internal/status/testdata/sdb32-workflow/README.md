---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: sd-b32
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

# SD-B32 Fixture Workflow

A frozen fixture using id-style sd-b32 to pin --next-id parity. The next-id
candidate is SHA-derived from the workflow realpath, seed, actor, context, and a
pinned timestamp, so the harness pins all of that material for reproducibility.
