---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: slug
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: ideation
      initial: true
    - name: build
      worktree: true
      concurrency: 1
    - name: review
      gate: true
    - name: done
      terminal: true
---

# Suppression-reason Fixture Workflow

A frozen fixture pinning the #230 visibility surface: each entity is suppressed
from `--next` for a distinct, attributable reason so `next-suppressed-by` can be
asserted to distinguish worktree-set / concurrency-full / gate (and "" for a
dispatchable entity). build has concurrency 1 and is worktree-backed so a single
in-flight build saturates it; review is a gate; done is terminal.
