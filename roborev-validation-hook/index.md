---
id: ng20gc9xdakkhk35e82385gm
title: Incremental commit-review hook for the validation stage (roborev-style)
status: backlog
source: session 1 debrief — validation flywheel
score: "0.20"
worktree:
---

Wire an incremental, per-commit code review onto the worktree code branch during/after implementation, feeding the validation gate (shift-left). Division of labor: the incremental reviewer covers code-quality / structure / subtle-bug hunting; the validation ensign keeps its unique job — behavioral parity against the oracle. Inspired by roborev's post-commit + session-reuse model.

## Acceptance criteria

**AC-1 - Per-commit review runs asynchronously on the worktree code branch and its findings are available at the validation gate.**
Verified by: a dispatched implementation that produces N commits yields N review findings (or a consolidated set) retrievable when validation runs; the FO gate review surfaces them alongside the parity report.

**AC-2 - The review watches the CODE branch (worktree), not the `.spacedock-state` checkout, and adds zero wall-clock to implementation (async).**
Verified by: the hook targets the worktree branch; implementation completion is not blocked on review completion.

**AC-3 - Sandbox / environment handling is documented and works in this environment.**
Verified by: a documented path for the daemon-backed tool (roborev needs a writable `~/.roborev` — relocate `HOME`), and a fallback for the no-daemon case; claude-code as the review agent (the only one healthy here).

## Test gates

- `go test ./...` (for any launcher glue)
- A pilot: dispatch a multi-commit implementation, confirm findings reach the gate; sandbox/HOME handling verified.

## Notes

From the session-1 debrief. Decision needed in ideation: build a thin native hook vs wrap roborev (`roborev` is updated to v0.56 but its daemon is sandbox-blocked here — `~/.roborev` is OS-denied; `HOME` relocation works). Lower priority than the self-hosting entities; genuinely flywheel for review quality but not on the critical path.
