---
id: bwck187yrng4rgxeyar1kfzz
title: NAME_PATTERN rejects stage names with underscores
status: backlog
source: captain (CL)
started:
completed:
verdict:
score:
worktree:
---

`claude-team build` derives `Agent()` names as `{worker_key}-{slug}-{stage}` and validates them against `NAME_PATTERN = ^[a-z0-9][a-z0-9-]*[a-z0-9]$` (`skills/commission/bin/claude-team:37`, used at line 317). Workflows whose `stages.states[].name` contains underscores cannot be dispatched — the derived name carries the underscore through and fails the regex with `derived name '...' contains invalid characters`.

Observed examples of stage names that trip this: `in_progress`, `live_full_approved`, `live_probation`.

The README's stage-name field is operator-authored prose. Spacedock's own README uses hyphens by convention, but nothing in commission or status enforces that convention, so workflows authored with snake_case stage names look valid at commission time and then break only at the first dispatch.

Possible resolution directions (decide in ideation):
- Allow underscores in `NAME_PATTERN` (and verify Claude Code's own agent-name validation accepts them).
- Reject underscore stage names at commission / `status --validate` time with a clear error pointing at the offending stage.
- Normalize underscores to hyphens when deriving the agent name (risks collisions across stages that differ only by `_` vs `-`).

Whichever path we pick, the failure must move from "first dispatch surprises the captain" to either "stage name is accepted everywhere" or "commission/validate refuses the stage name with an actionable message".
