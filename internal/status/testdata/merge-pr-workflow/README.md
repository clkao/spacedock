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

# Merge-PR Fixture Workflow

The default-policy (`merge:` key absent ⇒ `pr`) sibling of merge-local-workflow.
Registers the same `## Hook: merge` mod so the terminal-guard merge-hook branch is
reachable, pinning that an absent `merge:` key is byte-identical to today: a
terminal transition with empty `pr` and empty `mod-block` still refuses without
`--force`.
