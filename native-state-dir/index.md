---
id: nc85apg7333k7c594qam5m2n
title: Implement native state-dir support
status: ideation
score: "0.55"
source: bootstrap roadmap
worktree:
started: 2026-05-30T04:30:15Z
---

# Implement Native State-Dir Support

Teach native Go status to read workflow definition from the main README and mutable entities from the `state:` path.

## Acceptance Criteria

- `state: .spacedock-state` resolves relative to the workflow README directory.
- No README symlink is required inside `.spacedock-state`.
- Status reads stages and identity rules from the main README.
- Status reads and mutates entities under `.spacedock-state`.
- Archive moves happen under `.spacedock-state/_archive`.
- Workflow discovery finds the main README and ignores `.spacedock-state`.

## Test Gates

- `go test ./...`
- Split-root fixture with no `.spacedock-state/README.md`.
- `spacedock status --workflow-dir <workflow>` reads entities from the state checkout.
- `spacedock status --set <slug> status=implementation` changes only `.spacedock-state`.
- `spacedock status --archive <slug>` moves only `.spacedock-state` files.
- Discovery test proves `.spacedock-state` does not appear as a second workflow.

## Notes

Mods and PR guards remain out of scope. The first native split-root version should be boring.
