---
id: 7w8w5nsa5mbc807b3jb88psv
title: Native Go dispatch helper
status: backlog
score: "0.40"
source: handoff self-hosting gap
worktree:
---

Reimplement the `claude-team` dispatch helper — currently a vendored Python script at `skills/commission/bin/claude-team` — as a native Go surface of the `spacedock` launcher, so the first-officer handoff uses a SINGLE native binary for BOTH status and dispatch and the dispatch path has no Python dependency. Raised during the handoff-prompt review: after the native-status flip the launcher is native Go, but `claude-team build` is still Python, so a self-hosted handoff still shells out to Python (and assumes `claude-team` on PATH).

This is NOT part of Stage 7 (symlink removal/retest) and was NOT in the original bootstrap roadmap — Stage 4 only VENDORED and amended the Python `claude-team` (slug-not-stem + split-root entity path), it never reimplemented it. This entity closes the remaining self-hosting gap.

## Acceptance criteria

**AC-1 - Native dispatch `build` is byte-identical to the vendored Python `claude-team build`.**
Verified by: golden parity tests feeding the same input JSON to both and diffing the emitted spec (`subagent_type`, `name`, `model`, `prompt`, `dispatch_file_path`) AND the generated dispatch-file body, across flat `{slug}.md` and folder-form `{slug}/index.md` entities, split-root and single-root, and worktree vs non-worktree stages — i.e. the slug-not-stem name/branch/dispatch-file derivation and the split-root state-checkout entity path both match.

**AC-2 - The vendored FO/ensign references invoke the native dispatch command, not the Python `claude-team`.**
Verified by: static skill tests — the vendored refs call the native command (e.g. `spacedock dispatch build` or `spacedock team build`); no `skills/commission/bin/claude-team` reference remains in the dispatch path of the vendored skill surface; dispatched ensigns bootstrap from the vendored ensign reference (closing the last plugin dependency).

**AC-3 - context-budget / list-standing / spawn-standing: parity or an explicitly-scoped subset.**
Verified by: parity tests for whichever subcommands ideation scopes in; OR an explicit ideation decision recording which `claude-team` subcommands are reimplemented vs deferred. NOTE for ideation: `build` is the load-bearing handoff command; `context-budget` reads Claude Code agent transcripts and `spawn-standing` emits Agent() call specs — both are coupled to the Claude Code runtime and may stay thin shims or be scoped out of the native reimplementation.

## Test gates

- `go test ./...`
- Golden parity: native dispatch `build` vs vendored Python `claude-team build` across flat / folder / split-root / single-root / worktree fixtures.
- Static skill tests: vendored refs call the native dispatch command; zero Python `claude-team` references in the dispatch path.

## Notes

Goal is a clean, plugin-free, Python-free self-hosted handoff: one `spacedock` binary for status + dispatch, the vendored FO/ensign references loaded as the contract, and an install step putting `spacedock` on PATH. Ideation should scope the subcommand surface and decide the runtime-coupled commands (context-budget/spawn-standing). Independent of Stage 7's symlink work (disjoint surface), but sequence after Stage 7 to avoid same-package merge collisions unless ideation confirms a disjoint package (`internal/dispatch` + a new `cmd`/subcommand).
