---
id: 7w8w5nsa5mbc807b3jb88psv
title: Native Go dispatch helper
status: ideation
score: "0.40"
source: handoff self-hosting gap
worktree:
started: 2026-05-30T19:18:28Z
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

## Problem statement

After the native-status flip (Stage 7, merged at `0d4d319`), `spacedock status` is native Go. The dispatch path is still Python: the FO assembles `Agent()` dispatch by piping JSON to `skills/commission/bin/claude-team build`, and dispatched ensigns bootstrap by running `claude-team show-stage-def` as a fetch command. A self-hosted handoff therefore still requires `python3` AND a `claude-team` script on PATH, even though the launcher itself is a single native binary. This entity removes the last Python from the dispatch path so a handoff uses ONE native binary for both status and dispatch.

The parity oracle is the PROJECT-VENDORED helper at `skills/commission/bin/claude-team` (1306 lines, NOT the plugin copy at `/Users/clkao/git/spacedock/skills/commission/bin/claude-team`). The two diverge: the plugin copy still uses `slug = splitext(basename(entity_path))` (folder-form → `index`), while the project-vendored copy carries the Stage-4 amendments — `entity_slug()` (folder-form `{slug}/index.md` → folder name) and `workflow_is_split_root()` (split-root CODE-only worktree isolation + state-checkout entity path). Native `build` must match the project-vendored copy.

## Decision 1 — Native subcommand surface (closes AC-3)

`build` is the ONLY subcommand reimplemented natively. It is the load-bearing handoff command on the critical path: every initial `Agent()` dispatch flows through it, and it is pure (stdin JSON + workflow README + entity file → stdout JSON + dispatch-file write), with no Claude Code runtime coupling.

`show-stage-def` is ALSO reimplemented natively — not because it is dispatch-critical, but because `build` emits a `claude-team show-stage-def ...` fetch command into every dispatch body, and that command must resolve to the native binary for the Python-free goal to hold (the recurring "claude-team not on PATH" failure noted in the 2026-05-30 debrief). It is pure (README `### {stage}` subsection extraction) and shares the `extractStageSubsection` parser with `build`'s own stage-subsection emission. Emitted fetch line becomes `spacedock dispatch show-stage-def --workflow-dir {wd} --stage {stage}`.

Scoped OUT of the native reimplementation (stay Python in the plugin `claude-team`, or are simply not on the self-hosted dispatch critical path):

- `context-budget` — reads Claude Code agent transcripts (`~/.claude/projects/*/{session}/subagents/agent-*.jsonl`) and team `config.json`. Deeply coupled to the Claude Code runtime's on-disk layout, advisory-only (gates ensign reuse), and never invoked by a dispatched ensign. Not on the dispatch critical path; reimplementing it buys nothing for the Python-free handoff goal and imports a large fragile surface (jsonl scanning, model-to-context mapping). Decision: scoped out.
- `spawn-standing` — emits an `Agent()` call spec for a standing-teammate mod and probes team `config.json` for already-alive members. Claude Code-runtime-coupled (Agent specs, team membership). Standing teammates are an optional team-mode feature, not part of the load-bearing handoff. Decision: scoped out.
- `list-standing` / `show-standing` — enumerate/render `_mods/*.md` standing-teammate declarations. Coupled to the standing-teammate feature (same scope-out rationale as `spawn-standing`). `build` itself calls `enumerate_declared_standing_teammates` to decide whether to emit the `show-standing` fetch line; the dev workflow (`docs/dev`) has NO `_mods/` directory, so this path is inert for the self-hosting handoff. Decision: native `build` reproduces the "emit the fetch line when standing mods exist" branch for parity (it must, to be byte-identical), but the `show-standing` SUBCOMMAND stays Python — i.e. native `build` may emit a fetch line for a command the native binary does not implement, which is acceptable because no `_mods/` exist in the self-hosted dispatch path. If a future workflow adds standing teammates under the native binary, `show-standing` (and the renamed fetch line) is a follow-up entity.

Rationale for the asymmetry: the self-hosting gap is specifically the `build` + `show-stage-def` round-trip. The standing-teammate and context-budget surfaces are team-mode niceties that (a) are runtime-coupled and (b) do not appear in the `docs/dev` self-hosted handoff path. Reimplementing them is YAGNI for this entity.

## Decision 2 — Build parity contract + golden test plan (closes AC-1)

Native `spacedock dispatch build` must be byte-identical to the project-vendored `claude-team build` on:
- the emitted JSON object: `schema_version`, `subagent_type`, `description`, `fetch_commands`, `dispatch_file_path`, `prompt`, `model`, and (team mode) `name`, `team_name`.
- the dispatch-file body written to `dispatch_file_path` (the 9-component prompt assembly, byte-for-byte).
- the stderr `[build] effective_model=...` advisory line and the `WARN: bare_mode ...` advisory.
- exit codes: 0 success; 1 for validation/IO errors; 2 for unsupported `schema_version`.

Behaviors that MUST be reproduced (the derivation surface):
1. **slug-not-stem** (`entity_slug`): folder-form `{slug}/index.md` → folder name; flat `{slug}.md` → filename stem. Drives `name`, branch, and `dispatch_file_path`.
2. **split-root state-checkout entity path** (`workflow_is_split_root`): when the README declares `state:`, a worktree stage isolates CODE only — the worktree CODE-instruction block uses the "for CODE" phrasing + the path-scoped state-commit clause, and the entity-read line + completion-signal ref point at the FO-passed `entity_path` UNCHANGED (no `.worktrees/` segment). Non-split-root worktree stages rewrite the entity path into the worktree (`os.path.relpath` + join).
3. **worktree stickiness** (Rule 4): route on the entity's stamped `worktree:` frontmatter field, not the stage's declared mode.
4. **effective_model precedence** (Rule 6 / model resolution): stage > defaults > null; validate against `sonnet|opus|haiku`.
5. **all 12 validation rules** with byte-identical error strings (required fields, schema_version, worktree-path rejection, checklist non-empty, entity/README readable, stage exists, name length/pattern, feedback-context, team-name).

Golden test plan (matching the existing `internal/status` harness pattern in `harness_test.go` + `golden_read_test.go`):
- **Mechanism-first (cheap, runs first):** a single fixture exercising the riskiest path — split-root + folder-form + worktree stage — diffed native-vs-oracle. This is the integration-level "smallest failing test first": if the slug-not-stem + split-root entity-path derivation matches here, the rest is breadth.
- **Cross-product fixtures (breadth):** flat `{slug}.md` × folder `{slug}/index.md`, split-root × single-root, worktree × non-worktree, team-mode × bare-mode. Plus negative fixtures for each validation rule (exit 1/2 + stderr string).
- **Oracle invocation:** drive the project-vendored `claude-team build` under `python3` (the `vendoredClaudeTeam` helper already exists in `skills/integration/dispatch_test.go`) and the native `spacedock dispatch build`, with a pinned `HOME=t.TempDir()` (hermetic team-probe) and pinned env, then byte-compare stdout JSON AND the dispatch-file body. Normalize nothing — `build` output is deterministic (no timestamps; `dispatch_file_path` is a fixed `/tmp/spacedock-dispatch/{name}.md` derived purely from inputs). The oracle skips-when-absent like the status harness (`oraclePath()` pattern), but the project-vendored copy is in-repo so it is always present.
- **Cost:** Go unit/golden tests; each fixture is a `t.TempDir()` git-init + README + entity write + two process runs. Seconds per fixture; no live workflow needed. Estimated ~12-18 fixtures.

## Decision 3 — Disjoint Go package boundary (closes AC-3 disjointness)

`build` needs three pure helpers that today live UNEXPORTED in `package status`: `parseFrontmatter`, `parseStagesWithDefaults`, `findGitRoot` (plus the folder-slug rule, which `internal/status/discover.go` already encodes for entity discovery). It also needs `extractStageSubsection` (README `### {stage}` parser), which does not yet exist in Go.

Same-package collision is a REAL risk, not hypothetical: commit `356c7e7` ("rename test helper contains -> containsSlug (resolve package-status redeclare)") records an actual `package status` redeclaration collision during status work. Adding `build` directly into `package status` reopens that risk.

Decision: implement `build` + `show-stage-def` in a NEW package `internal/dispatch`, routed from a NEW `cli.go` switch case `dispatch` (disjoint from the `status` case — the two never share a hunk). Command surface: `spacedock dispatch build --workflow-dir {wd}` (stdin JSON → stdout JSON) and `spacedock dispatch show-stage-def --workflow-dir {wd} --stage {stage}`.

To share the three pure helpers without duplication and without a disruptive mass-export of `package status`'s internals, the cleanest seam is to lift the pure, runtime-neutral parsers — `parseFrontmatter`, `parseStagesWithDefaults`, `findGitRoot`, and the folder-slug rule — into a small leaf package (proposed `internal/spec`) that BOTH `internal/status` and `internal/dispatch` import. This is the Go analog of the Python helper's `RUNTIME-NEUTRAL`-tagged borrowed surface (the Python `claude-team` imports exactly these from its sibling `status` script). The lift is a mechanical move (no logic change) and is itself a small, reviewable diff. `extractStageSubsection` is new and lives in `internal/dispatch` (or `internal/spec` if status ever needs it; YAGNI says `internal/dispatch` for now). The command name `dispatch` is chosen over `team`: the surface reimplemented is the dispatch path (`build` + `show-stage-def`), and `dispatch` reads truer than the Claude-Code-specific "team" framing — and the emitted fetch line `spacedock dispatch show-stage-def` is self-describing.

Note on the leaf-package move: it touches `package status` files (the helper definitions move out), so it IS a status-package edit. But it is a one-shot mechanical extraction landed in its own commit BEFORE the dispatch implementation, not an ongoing same-package coexistence. If the FO/captain prefers zero `package status` edits, the fallback is to export the three helpers in place (`ParseFrontmatter`, etc.) and import `internal/status` from `internal/dispatch` — also disjoint at the hunk level, at the cost of widening the `status` exported surface. Both keep `build` out of `package status`. Recommend the leaf-package lift; flag the export-in-place fallback for the gate.

## Refined acceptance criteria

AC-1, AC-2, AC-3 in the frontmatter `## Acceptance criteria` section stand as written and are end-state properties (byte-identical `build`; vendored refs invoke the native command; runtime-coupled subset explicitly scoped). This ideation refines them only by recording the scope decisions above:
- AC-3's "explicitly-scoped subset" is now decided: native = `build` + `show-stage-def`; scoped out = `context-budget`, `spawn-standing`, `list-standing`, `show-standing`.
- AC-2's "the native command" is now named: `spacedock dispatch build` and `spacedock dispatch show-stage-def`. The vendored FO runtime adapter's MANDATORY dispatch block (`skills/first-officer/references/claude-first-officer-runtime.md` lines 71-108) changes its pipe target from `{project_root}/skills/commission/bin/claude-team build --workflow-dir {wd}` to `spacedock dispatch build --workflow-dir {wd}`; the `build`-emitted fetch line (vendored `claude-team` line 408-411) changes `claude-team show-stage-def` → `spacedock dispatch show-stage-def`. `context-budget`/`spawn-standing`/`list-standing` references in the FO adapter are NOT dispatch-path and stay as-is (they invoke the scoped-out Python subcommands; this is acceptable because those features are team-mode-only and absent from the self-hosted `docs/dev` handoff — AC-2's "no `claude-team` reference remains in the DISPATCH PATH" is satisfied).

## Test plan

1. **Golden parity (AC-1)** — `internal/dispatch` golden/parity tests, mechanism-first then cross-product, per Decision 2. Gate: native `build` byte-identical to project-vendored `claude-team build` across flat/folder × split/single × worktree/non-worktree × team/bare. Go tests, seconds each.
2. **Native `show-stage-def` parity** — diff `spacedock dispatch show-stage-def` vs project-vendored `claude-team show-stage-def` across well-formed, decorated-heading, malformed-heading (ValueError string), and missing-stage fixtures. Go tests.
3. **Static skill tests (AC-2)** — extend `skills/integration/skill_text_test.go`: assert the FO runtime adapter's dispatch block pipes to `spacedock dispatch build` (not `claude-team build`); assert the `build`-emitted fetch line is `spacedock dispatch show-stage-def`; assert zero `skills/commission/bin/claude-team` reference remains in the dispatch-path prose of the vendored FO/ensign surface. Update the existing `dispatch_test.go` (currently drives the Python helper) to drive the native binary as the primary, keeping a Python-oracle parity arm.
4. **Ensign bootstrap (AC-2)** — confirm a dispatched ensign's first-action fetch command (`spacedock dispatch show-stage-def`) resolves against the native binary with the binary on PATH, closing the "claude-team not on PATH" gap. Verified by the static fetch-line assertion plus the native `show-stage-def` parity test; a live workflow smoke is NOT required because the fetch-line content is the testable claim.
5. **Baseline** — `go test ./...` and `go test ./... -race` stay green (246 tests baseline today).

## Stage Report: ideation

- DONE: Scope the native dispatch subcommand surface: which claude-team subcommands are reimplemented natively (build is load-bearing) vs kept as thin shims / scoped out (context-budget, list-standing, spawn-standing — coupled to Claude Code transcripts/Agent specs). Record and justify the decision.
  Decision 1 in body: native = `build` + `show-stage-def` (the dispatch round-trip); scoped out = `context-budget` (jsonl/transcript-coupled, advisory), `spawn-standing` (Agent-spec/team-coupled), `list-standing`/`show-standing` (standing-teammate feature, absent from the `docs/dev` self-hosted path). Asymmetry noted: native `build` reproduces the show-standing fetch-line branch for parity but the subcommand stays Python (inert — no `_mods/` in dev workflow).
- DONE: Define the byte-identical build parity contract + golden test plan vs the vendored Python claude-team build across flat/folder, split-root/single-root, worktree/non-worktree fixtures (slug-not-stem name/branch/dispatch-file derivation + split-root state-checkout entity path).
  Decision 2 + Test plan in body: byte-compare stdout JSON + dispatch-file body + stderr advisories + exit codes; mechanism-first fixture (split-root + folder + worktree) then cross-product; oracle is the PROJECT-VENDORED `claude-team` (not the plugin copy — they diverge on `entity_slug`/`workflow_is_split_root`); reuse the `internal/status` golden-harness pattern; deterministic output needs no normalization.
- DONE: Confirm a disjoint Go package boundary (e.g. internal/dispatch + a new subcommand) so implementation will not same-package merge-collide with status work; design the native command name (e.g. spacedock dispatch build).
  Decision 3 in body: new `internal/dispatch` package + new `dispatch` cli switch case (disjoint hunk from `status`); command names `spacedock dispatch build` and `spacedock dispatch show-stage-def`. Same-package collision risk validated by commit 356c7e7 (a real `package status` redeclare). Recommend lifting the 3 pure parsers (`parseFrontmatter`/`parseStagesWithDefaults`/`findGitRoot` + folder-slug rule) into a leaf `internal/spec` package both packages import (Go analog of the Python RUNTIME-NEUTRAL borrowed surface); export-in-place flagged as fallback for the gate.

### Summary

Fleshed out the native dispatch-helper task into problem statement, three scope/design decisions, refined ACs, and a test plan. Key findings the implementer must not miss: the parity oracle is the PROJECT-VENDORED `skills/commission/bin/claude-team` (carries the Stage-4 slug-not-stem + split-root amendments), NOT the plugin copy which still derives `index` for folder-form entities; Stage 7 is already merged so the sequencing caveat is satisfied; and `package status` has a documented prior same-package collision (356c7e7), making the disjoint-package boundary load-bearing rather than precautionary. Two open choices left for the gate: leaf-package lift (recommended) vs export-in-place for sharing the 3 pure parsers, and the `dispatch` vs `team` command-name framing (recommend `dispatch`). No frontmatter modified; ideation committed to main (non-worktree entity).
