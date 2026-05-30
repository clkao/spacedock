---
id: bdxxg3yr1ys4nsmwk8x65j7b
title: Prove symlink state profile
status: ideation
score: "0.85"
source: bootstrap roadmap
worktree:
started: 2026-05-30T04:30:15Z
---

# Prove Symlink State Profile

Prove that a per-workflow `.spacedock-state` checkout can run current status behavior when its README symlinks back to the workflow README.

## Acceptance Criteria

- A fixture workflow has `README.md` in the main workflow directory.
- `.spacedock-state/README.md` symlinks to `../README.md`.
- Active entities live directly under `.spacedock-state`.
- Folder-form entities can contain `reports/` and `artifacts/`.
- `_archive` is inside `.spacedock-state`.
- No mods or PR behavior is required.

## Test Gates

- `go test ./...`
- Integration fixture creates the symlink layout in a temporary repo.
- `spacedock status --workflow-dir <workflow>/.spacedock-state` renders active entities.
- `spacedock status --archive <slug>` moves the folder entity to `.spacedock-state/_archive`.
- Entity-local `reports/` and `artifacts/` are not discovered as entities.

## Notes

This is the bridge that lets current tooling work before native split-root support exists.
