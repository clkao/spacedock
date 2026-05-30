---
id: jsacmxpkcwp3vg2t60yjdc4y
title: Agent-facing output modes — JSON, field projection, quiet (token-frugal)
status: backlog
source: session 1 debrief — rtk-inspired
score: "0.38"
worktree:
---

Give `spacedock status` agent-facing output modes so first officers and ensigns parse state cheaply and robustly instead of scraping a padded table. Inspired by rtk (Rust Token Killer): the agent surface should be token-frugal and structured. FLYWHEEL: the FO reads status on every boot / dispatchable check / gate across a long autonomous sprint, so per-read cost compounds.

## Acceptance criteria

**AC-1 - `spacedock status --json` emits valid, stable JSON for the read commands (default table, --next, --archived, --where, --resolve, --boot sections) that an agent can parse without table glyphs/padding.**
Verified by: Go tests assert the JSON shape per command and that it round-trips the same fields as the table; documented schema.

**AC-2 - `--fields a,b,c` projects only the requested columns (table and JSON), so an agent can request the minimal set.**
Verified by: tests that `--fields id,status` emits only those keys/columns in both modes.

**AC-3 - `--quiet` suppresses non-essential chrome for hook/script use (e.g. mutation narration reduced to a machine line), without losing exit-code semantics.**
Verified by: tests on `--set`/`--archive --quiet` output + exit codes.

**AC-4 - Output is proxy-survivable / structured.**
Verified by: a note + test that `--json` output is byte-stable and free of decorative formatting that a token-optimizing proxy (e.g. rtk) would mangle; the FO contract is updated to prefer `--json`/`--fields` for its own reads.

## Test gates

- `go test ./...`
- `--json` schema tests per read command; `--fields` projection tests; `--quiet` mutation tests; FO-contract update to consume `--json`.

## Notes

rtk this session mangled `git log` output and summarized `go test` — a tool meant to be agent-driven should emit structured output that survives proxying, or ship an rtk profile. Highest-leverage of the ergonomic items because the FO itself is the heaviest reader; worth pulling early in the sprint.
