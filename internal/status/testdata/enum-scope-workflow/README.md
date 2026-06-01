---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: slug
stages:
  defaults:
    worktree: false
    concurrency: 1
  states:
    - name: backlog
      initial: true
    - name: done
      terminal: true
---

# Enumeration Scope Fixture Workflow

A frozen fixture pinning the #207 enumeration-scope rule: placement, not the
`status` frontmatter value, decides whether an entity is enumerated in the
active scope (top-level) or the archived scope (`_archive/`). Two entities carry
the same `status` so a scope difference can only come from placement. id-style is
slug so the active + archived effective ids stay distinct (no duplicate-id
validation collision across scopes).
