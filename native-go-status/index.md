---
id:
title: Implement native Go status parity
status: backlog
score: "0.65"
source: bootstrap roadmap
worktree:
---

# Implement Native Go Status Parity

Replace the delegated status path with a native Go implementation that matches current behavior before adding new state-dir semantics.

## Acceptance Criteria

- Go frontmatter parser matches the current line-oriented parser.
- Go stage parser matches current README stage behavior.
- Entity discovery supports flat files and folder-form `index.md`.
- Status output and mutations match current behavior for compatibility fixtures.
- Validation failures remain clear and stable.

## Test Gates

- `go test ./...`
- Parser tests for empty fields, quoted fields, nested-line ignoring, and missing frontmatter.
- Stage parser tests for defaults, gates, terminal stages, worktree flags, and feedback targets.
- Golden parity for default status, `--archived`, `--next`, `--where`, `--fields`, `--all-fields`, `--next-id`, `--resolve`, and `--short-id`.
- Mutation tests for `--set`, timestamp fill, and `--archive`.
- Validation tests for duplicate IDs, invalid IDs, flat/folder conflicts, unknown statuses, and archive guards.

## Notes

Do not implement `state:` split-root support in this stage. Keep the replacement focused.
