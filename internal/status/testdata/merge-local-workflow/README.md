---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: sequential
merge: local
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

# Merge-Local Fixture Workflow

A single-root fixture declaring `merge: local` and registering a `## Hook: merge`
mod so the terminal-transition merge-hook guard branch is actually reached. Pins
the guard-relaxation parity between the native runner and the Python oracle: under
`merge: local` a terminal transition with empty `pr` and cleared `mod-block` is
permitted without `--force`, while an in-flight `mod-block` still blocks.
