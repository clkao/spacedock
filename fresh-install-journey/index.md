---
id: 5an0s2dqb0nh18n7nn6x07vx
title: Land launcher on next + document the fresh-install user journey
status: ideation
source: sprint — Ship the Launcher slice D (captain, 2026-05-30)
started: 2026-05-31T03:20:03Z
completed:
verdict:
score: "0.26"
worktree:
issue:
---

Integrate the launcher slice on the `next` branch and document the end-to-end fresh-install journey.

## Target
- Add origin `git@github.com:clkao/spacedock.git` to spacedock-v1; land the launcher + formula + release config on a `next` branch (NOT main).
- README documents: `brew tap spacedock-dev/homebrew-tap && brew install spacedock`, the safehouse prerequisite + install hint, and the `spacedock claude` / `spacedock codex` usage.
- Fresh-install user-journey doc: clean Mac → `brew install` → plugin install → `spacedock claude` launches the FO through safehouse.
- `go test ./...` green on next.

## Dependencies
- Integrates A, A′, B, C. Sequence LAST.
- The origin push (`next` to `clkao/spacedock`) is a captain repo action per the sprint notes; this entity stages the branch + docs.

## Acceptance criteria (provisional — ideation hardens each)

**AC-1 — next builds green.** `go test ./...` exits 0 on the `next` branch carrying the launcher.

**AC-2 — install docs present + accurate.** README documents the brew tap + install + safehouse prereq; the journey doc walks a fresh Mac end to end against the real command surface.
