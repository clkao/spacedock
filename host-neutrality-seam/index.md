---
id: 0mxzm82mpjbv0tjd51pv2t5j
title: Host-neutrality seam — relocate ~/.claude team-state reads behind an injected probe (zs prerequisite)
status: ideation
source: zs claude-runtime-segregation decomposition (CL 2026-05-31 "zs - split"; staff review M-3) — the riskiest-first prerequisite
started: 2026-05-31T20:45:00Z
completed:
verdict:
score: "0.34"
worktree:
issue:
---

The prerequisite carved out of `claude-runtime-segregation` (zs) per the staff-review decomposition (zs
`### Decision 4`). It establishes the `internal/claudeteam` package and a host-supplied **team-state
probe** seam, relocating the three pre-existing `~/.claude` reads that already violate zs's AC-3
host-neutrality invariant — landing FIRST so the five native subcommands plug into a proven seam
rather than stacking six workstreams behind one integration point. "Validate the riskiest path first."

## The three pre-existing leaks (FO-confirmed against source)

- `internal/status/boot.go:131-163` `probeTeamState` — reads `~/.claude/teams/*/config.json` mtimes,
  called by `gatherBoot:206`, feeds the boot `TEAM_STATE` JSON (`teamPresent`/`teamHint`).
- `internal/dispatch/helpers.go:137-162` `recentTeamEvidence` — reads `~/.claude/teams` via
  `os.UserHomeDir()`, called by `build.go:115` to gate a bare-mode WARN.
- `internal/dispatch/build.go:116-120` — the bare-mode WARN string literal naming
  `~/.claude/teams/*/config.json`.

## Approach

Create `internal/claudeteam`. Move the two team-state reads (and the WARN text) into it behind a
host-supplied probe (a func/interface) injected into `gatherBoot` and `runBuild`; the Claude package
fills the probe, the Codex/bare path leaves it nil. The generic `internal/dispatch` + `internal/status`
source then carries no `.claude` literal.

## Acceptance criteria

**AC-P1 — The AC-3 code-side oracle goes RED→GREEN.**
End state: a `go/parser` test over `internal/dispatch` + `internal/status` source finds no
`~/.claude` path-join / `os.UserHomeDir`-rooted team/transcript read literals. It FAILS today on the
three leaks above and PASSES after relocation.
Verified by: the code-side structural test (ship it RED first, then green after the move).

**AC-P2 — Boot `TEAM_STATE` and build bare-mode-WARN behavior are preserved across the relocation.**
End state: with the probe injected (Claude) the boot JSON `TEAM_STATE` fields and the build bare-mode
WARN are byte-identical to today's output on a fixture `~/.claude` tree; with the probe nil (Codex/bare)
the outputs differ ONLY in `TEAM_STATE: present` and the WARN's presence — no structural or other-field
divergence.
Verified by: **the seam-parity SPIKE the staff re-review requires (run in ideation, before
implementation)** — instrument boot/build on a Codex-like nil-probe fixture and a Claude real-probe
fixture and assert the diff is confined to those two surfaces; then a regression test pinning it.

**AC-P3 — The five runtime-coupled subcommands still resolve via Python until zs-main lands.**
End state: this entity does NOT reimplement context-budget/list-standing/show-standing/spawn-standing
or the build `_mods` branch — it only establishes the package + probe seam; those still route through
the vendored Python oracle.
Verified by: the existing parity harness still green; no new native subcommand handlers added here.

## Test plan

The seam-parity spike (AC-P2) is the riskiest-mechanism check and runs in ideation. Implementation:
the code-side `go/parser` oracle (AC-P1), the boot/build parity regression (AC-P2), and a no-new-native-
subcommands assertion (AC-P3). `go test ./...` green. Worktree-backed implementation (CODE-only under
split-root; entity body + reports stay in the state checkout).

## Blocks

`claude-runtime-segregation` (zs) — its implementation starts after this seam lands (zs `### Decision 4`).
Coordinate the `first-officer-shared-core.md` edits with zs (zs owns the Standing-Teammates rewrite +
reuse-condition prose; this entity touches only the code seam + the boot/build paths).
