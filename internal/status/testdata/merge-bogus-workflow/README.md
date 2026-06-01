---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: sequential
merge: bogus
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

# Merge-Bogus Fixture Workflow

A single-root fixture declaring an invalid `merge:` value (`bogus`). The policy
parser must reject it loudly — a `--set` exits 1 with a clear error naming the
allowed values — rather than silently coercing to the default `pr`, so a typo
fails fast instead of demanding a PR forever.
