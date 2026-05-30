---
id: zrep3rt8a21y7k3szbs3ysyn
title: Retest without README symlink
status: ideation
score: "0.45"
source: bootstrap roadmap
worktree:
started: 2026-05-30T04:30:16Z
---

# Retest Without README Symlink

Remove the compatibility symlink from the state checkout and prove the native split-root path is complete.

## Acceptance Criteria

- `.spacedock-state/README.md` is not needed.
- Pilot workflow commands work from the main workflow directory.
- Main repo status does not show runtime state churn.
- State checkout status shows runtime entity changes.
- Reports and artifacts remain with folder-form entities through archive.

## Test Gates

- `go test ./...`
- Delete `.spacedock-state/README.md` in the split-root fixture.
- Rerun status, `--next`, `--validate`, `--set`, and `--archive`.
- Verify main repo `git status --short` excludes `.spacedock-state`.
- Verify state checkout `git status --short` shows entity mutations when expected.

## Notes

This is the migration gate from compatibility profile to native profile.
