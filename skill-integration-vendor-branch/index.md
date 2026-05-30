---
id: v2j735kcfmsg6yv30nqt8t61
title: Integrate skills with launcher
status: ideation
score: "0.75"
source: bootstrap roadmap
worktree:
started: 2026-05-30T04:30:15Z
---

# Integrate Skills With Launcher

Point first-officer and ensign skill instructions at the `spacedock` launcher while the launcher still uses the compatibility status path.

## Acceptance Criteria

- First-officer instructions call `spacedock status`, not plugin-private script paths.
- Ensign dispatch receives entity paths under `.spacedock-state` for workflow records.
- Skill tests cover command text and path handoff.
- A pilot workflow can list, mutate, and archive an entity through the launcher.
- No PR merge mod, lifecycle mod, or mod-hook behavior is introduced.

## Test Gates

- `go test ./...`
- Static skill tests for command text.
- Smoke test for list, `--set`, and archive through `spacedock status`.
- Regression check that no skill text references `skills/commission/bin/status`.

## Notes

If `claude-team` cannot operate through the symlink profile, file a separate helper task before changing the skill contract.
