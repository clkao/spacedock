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
    - name: implementation
    - name: review
    - name: done
      terminal: true
---

# Symlink Profile Fixture Workflow

A frozen fixture workflow used to prove the `.spacedock-state/README.md -> ../README.md`
compatibility symlink. The runner is pointed at the state checkout alone and must
read this stages block through the symlink. id-style is slug so entities need no
numeric id and the slug is the effective id. Entities span distinct stages so the
default-table ordering by stage is unambiguous.
