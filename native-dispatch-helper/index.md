---
id: 7w8w5nsa5mbc807b3jb88psv
title: Native Go dispatch helper
status: implementation
score: "0.40"
source: handoff self-hosting gap
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-native-dispatch-helper
started: 2026-05-30T19:18:28Z
---

Reimplement the `claude-team` dispatch helper — currently a vendored Python script at `skills/commission/bin/claude-team` — as a native Go surface of the `spacedock` launcher, so the first-officer handoff uses a SINGLE native binary for BOTH status and dispatch and the dispatch path has no Python dependency. Raised during the handoff-prompt review: after the native-status flip the launcher is native Go, but `claude-team build` is still Python, so a self-hosted handoff still shells out to Python (and assumes `claude-team` on PATH).

This is NOT part of Stage 7 (symlink removal/retest) and was NOT in the original bootstrap roadmap — Stage 4 only VENDORED and amended the Python `claude-team` (slug-not-stem + split-root entity path), it never reimplemented it. This entity closes the remaining self-hosting gap.

## Acceptance criteria

**AC-1 - Native dispatch `build` is semantically equivalent to the vendored Python `claude-team build` on non-`_mods` workflows.**
Verified by: golden parity tests that feed the SAME input JSON to the project-vendored Python oracle and to `spacedock dispatch build`, then byte-compare three channels separately — (a) the emitted stdout JSON with `fetch_commands` carved out, (b) the dispatch-file body with the single `show-stage-def` fetch line carved out, (c) exit code — and assert the ONE rewritten line: every `claude-team show-stage-def ...` becomes `spacedock dispatch show-stage-def ...` byte-for-byte in both the JSON `fetch_commands` array and the body's `### Fetch commands` block. The compared bytes are identical EXCEPT that rewritten fetch line. Fixtures span flat `{slug}.md` × folder `{slug}/index.md`, split-root × single-root, worktree × non-worktree, team × bare. Equivalence (not byte-identity) is the contract because (i) the fetch line is deliberately rewritten and (ii) two error paths carry Python's `str(e)` (see AC-1b). NO `_mods/` fixtures are in scope — the standing-teammate fetch-line branch is the sibling entity `claude-runtime-segregation`'s concern.

**AC-1b - The two `str(e)`-bearing error paths are structurally equivalent, not byte-equal; every other error byte-string is byte-identical.**
Verified by: negative parity tests. For the 18 deterministic error byte-strings enumerated in Decision 2 (Rules 1-12 + model-enum + heading + name), assert the native stderr equals the oracle stderr byte-for-byte AND the exit code matches (1 for validation/IO, 2 for unsupported schema_version). For the 2 `str(e)` paths — invalid-JSON-on-stdin and dispatch_file_write_failed — assert exit code parity (1) and a byte-identical stderr PREFIX (`error: invalid JSON on stdin: ` and `dispatch_file_write_failed: {path}: ` respectively), not the trailing interpreter-version-specific message text. The spike proved the invalid-JSON tail varies by Python version (3.14 emits `Illegal trailing comma before end of object`), so a full byte-compare there would be a Python-version-coupled test.

**AC-2 - The vendored FO/ensign dispatch path invokes the native dispatch command; zero `claude-team` reference remains in the dispatch path.**
Verified by: a static-text predicate over the vendored skill surface (extending `skills/integration/skill_text_test.go`) that is true iff BOTH hold — (a) the FO runtime adapter's MANDATORY dispatch block pipes to `spacedock dispatch build --workflow-dir {workflow_dir}` (NOT `{project_root}/skills/commission/bin/claude-team build`), and (b) every fetch line `spacedock dispatch build` emits resolves to `spacedock dispatch show-stage-def` (asserted by the AC-1 body parity, which observes the emitted bytes). The `context-budget` / `list-standing` / `spawn-standing` references in the FO adapter MAY retain `claude-team` — those subcommands are the sibling entity `claude-runtime-segregation`'s concern and are absent from the self-hosted `docs/dev` dispatch path (no `_mods/`). The predicate scopes its "zero claude-team" assertion to the dispatch block + emitted fetch lines, not the whole file.

**AC-3 - The native subcommand surface is exactly `build` + `show-stage-def`; everything else is explicitly deferred to the sibling entity.**
Verified by: the ideation record (Decision 1) naming native = `build` + `show-stage-def` and deferred = `context-budget` + `list-standing` + `show-standing` + `spawn-standing` + the `_mods`/standing fetch-line branch, all MOVED to sibling `claude-runtime-segregation` (zse4a3ds0x19gpdcjh7anhgs); plus a behavioral guard — a parity fixture confirming `spacedock dispatch` with an unknown/deferred subcommand (e.g. `context-budget`) exits non-zero with a usage diagnostic rather than silently no-op'ing, so the deferral is observable, not merely asserted in prose.

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

REVISED per Cycle 1: a captain-directed scope split moves the entire `_mods`/standing surface to a sibling entity. This entity's native surface is exactly two subcommands, both pure and both on the self-hosted dispatch round-trip:

- `build` — the load-bearing handoff command. Every initial `Agent()` dispatch flows through it; it is pure (stdin JSON + workflow README + entity file → stdout JSON + dispatch-file write) with no Claude Code runtime coupling. **Scoped to NON-`_mods` workflows.** The self-hosted target workflow `docs/dev` has no `_mods/` directory, so native `build` needs NO `parse_mod_metadata`, NO `enumerate_declared_standing_teammates`, and emits NO `show-standing` fetch line. This keeps the native port lean: the only fetch line `build` emits is the `show-stage-def` line.
- `show-stage-def` — reimplemented natively because `build` emits a `... show-stage-def ...` fetch line into every dispatch body and that command must resolve to the native binary for the Python-free goal to hold (the recurring "claude-team not on PATH" failure in the 2026-05-30 debrief). It is pure (README `### {stage}` subsection extraction) and shares the `extractStageSubsection` parser with nothing else native, since `build` no longer emits a stage subsection inline (the v2 file-pointer body references it via the fetch line). Emitted fetch line: `spacedock dispatch show-stage-def --workflow-dir {wd} --stage {stage}`.

**Deferred to sibling entity `claude-runtime-segregation` (zse4a3ds0x19gpdcjh7anhgs), NOT reimplemented here:**

- The `_mods`/standing surface in its entirety: `build`'s show-standing fetch-line branch (the `enumerate_declared_standing_teammates` → emit-`show-standing`-line logic at oracle lines 478-482), and the `show-standing` / `list-standing` / `spawn-standing` subcommands. All are coupled to the standing-teammate feature and require `parse_mod_metadata`.
- `context-budget` — reads Claude Code agent transcripts (`~/.claude/projects/*/{session}/subagents/agent-*.jsonl`) and team `config.json`. Deeply coupled to the Claude Code runtime's on-disk layout, advisory-only (gates ensign reuse), never invoked by a dispatched ensign. The sibling owns it.

Why deferral rather than reproducing-the-branch-for-parity (the Cycle-0 plan): scoping `build` to non-`_mods` workflows means there is no `_mods/` path to reproduce. AC-1 parity is tested ONLY on non-`_mods` workflows, where native and oracle agree without `build` containing any standing logic. The sibling entity closes the `_mods` parity gap on workflows that have `_mods/`. This resolves the Cycle-0 Decision-1-vs-AC-2 conflict (native `build` emitting a fetch line for a subcommand it does not implement) by DEFERRAL — the conflict cannot arise when the branch is absent. The native binary will reject the deferred subcommands with a usage diagnostic (AC-3 behavioral guard), making the deferral observable.

## Decision 2 — Build parity contract + golden test plan (closes AC-1, AC-1b)

GROUNDED IN THE SPIKE. A spike ran the project-vendored oracle on model-precedence (stage/defaults/null), malformed-stdin, the riskiest derivation (split-root+folder+worktree), bare-mode, non-split-root-worktree, feedback+scope, and all error fixtures, capturing exact stdout/stderr/exit bytes (artifacts under `/tmp/spike-build/`). The contract below is what those bytes prove, not what the source reads.

### Three-channel comparison (the test helper MUST split stdout/stderr)

The existing `runBuild` helper in `skills/integration/dispatch_test.go` uses `CombinedOutput()` — which CANNOT byte-compare stdout and stderr separately and would interleave the `[build] effective_model` / `WARN` stderr lines into the JSON. The native parity harness must capture stdout and stderr into separate `bytes.Buffer`s (the `internal/status` harness already does this in `runOracle`/`runLauncher`). Three channels, compared independently:

1. **stdout JSON.** Observed key order is insertion order: `schema_version, subagent_type, description, fetch_commands, dispatch_file_path, prompt, model`, then `name, team_name` present ONLY in team mode (absent keys in bare mode, not null). Byte-level facts the native emitter MUST match, each a spike finding:
   - **Trailing newline.** Oracle output ends `}\n` (Python `print`). Go `json.MarshalIndent` does not append `\n`; the emitter must (or use `json.Encoder`, which does).
   - **No HTML escaping.** Python `json.dumps` emits `<`, `>`, `&` literally; Go's default `json.Marshal` escapes them to `<`/`>`/`&`. The emitter MUST use `json.Encoder` with `SetEscapeHTML(false)`. This is load-bearing: `feedback_context`/`scope_notes`/`checklist` values routinely carry prose with `<`/`>`/`&` (the FO passes reviewer findings verbatim). A parity fixture MUST include a `<`/`>`/`&`-bearing field to lock this.
   - **`null` literal** for absent model (`"model": null`), produced by a `*string` field or equivalent.
   - **`indent=2`**, two-space.
   The parity assertion carves `fetch_commands` OUT of the JSON compare and asserts it separately (it is the one rewritten channel — see below).
2. **dispatch-file body** written to `dispatch_file_path`. Byte-identical EXCEPT the single `### Fetch commands` line, which is rewritten (see below). The body is the prompt assembly: first-action block, header, conditional worktree block (split-root "for CODE" phrasing + path-scoped state-commit clause vs non-split-root "Your working directory is" + plain branch clause), entity-read line (split-root → FO-passed path unchanged; non-split-root worktree → path rewritten into the worktree via `relpath`+join), conditional feedback block, conditional scope_notes, checklist+summary, the `### Fetch commands` block, and (team mode only) the `### Completion Signal` block. The spike confirmed every one of these branches byte-for-byte.
3. **exit code + stderr.** 0 success; 1 validation/IO; 2 unsupported schema_version. The two stderr advisory channels: `[build] effective_model={m} (from {source}) → Agent model={m}\n` (the arrow is U+2192 `→`, UTF-8 `e2 86 92`; emitted only when a model is resolved — the null case emits empty stderr), and the `WARN: bare_mode ...` line (full text in the oracle at lines 237-243). Both reproduced byte-for-byte.

### The ONE rewritten line (the only intentional divergence)

`build` emits exactly one fetch line. Oracle: `claude-team show-stage-def --workflow-dir {wd} --stage {stage}` (shlex-quoted args). Native: `spacedock dispatch show-stage-def --workflow-dir {wd} --stage {stage}`. The parity test asserts the native fetch line equals the oracle's with `claude-team` → `spacedock dispatch` substituted, in BOTH the JSON `fetch_commands[0]` and the body's `### Fetch commands` block. Note shlex-quoting: paths with spaces get single-quoted by the oracle; the native emitter must reproduce POSIX shell quoting equivalently (Go has no stdlib `shlex.quote`, so the implementer ports the minimal rule — quote when the arg contains shell-unsafe chars). The dev fixtures use space-free paths, but a space-bearing-path fixture MUST lock the quoting rule.

### Error byte-string enumeration (closes AC-1b)

The spike enumerated the distinct error strings. **18 are deterministic and MUST be byte-identical** (Rules 1-12 plus model-enum, no-stages-block, malformed-heading, name-length, name-pattern):

1. `error: stdin must be a JSON object` (exit 1)
2. `error: missing required field '{field}'` (exit 1) — also fires for present-but-null fields
3. `error: unsupported input schema_version {sv}, schema_version: 2 required` (exit 2)
4. `error: entity_path must be a project-root absolute path; got worktree path '{p}'. Pass the project-root location (e.g. '/repo/docs/plans/{slug}.md'), not the worktree copy. The helper derives the worktree read target internally.` (exit 1) — fires for both `/.worktrees/` substring and `.worktrees/` prefix
5. `error: checklist must not be empty` (exit 1) — ALSO the message when `checklist` is a non-list (the spike confirmed both collapse to this one string)
6. `error: entity file not readable at '{p}'` (exit 1)
7. `error: workflow README not found at '{p}'` (exit 1)
8. `error: no stages block found in {readme}` (exit 1)
9. `error: stage '{stage}' not found in {readme}` (exit 1)
10. `error: invalid model for stages.states[{idx}].model: '{m}' — must be one of: sonnet, opus, haiku` (exit 1)
11. `error: invalid model for stages.defaults.model: '{m}' — must be one of: sonnet, opus, haiku` (exit 1)
12. `error: worktree path '{p}' does not exist` (exit 1)
13. `error: worktree stage '{stage}' but entity has no worktree path` (exit 1)
14. `error: dispatching to feedback target stage '{stage}' but feedback_context is missing` (exit 1)
15. `error: team mode requires team_name` (exit 1)
16. `error: derived name '{name}' exceeds 200 characters` (exit 1)
17. `error: derived name '{name}' contains invalid characters: stage name '{stage}' must match ^[a-z0-9][a-z0-9-]*[a-z0-9]$ (kebab-case lowercase letters, digits, and hyphens only). Run \`status --validate\` against the workflow to surface the same stage-name error upstream of dispatch.` (exit 1)
18. `error: stage heading at line {n} mentions '{stage}' but does not parse as a stage heading: '{raw}'. The stage name must be the first content token of the heading after stripping Markdown decoration (backticks, *, _, ~) and treating '(' and '[' as token terminators.` (exit 1)

Note (10)/(11) embed the em-dash `—` (U+2014). Note the model-validation order: the spike confirms stage-model is validated before defaults-model, and BOTH before stickiness/worktree resolution.

**2 paths are `str(e)`-bearing and tested as structurally equivalent (exit + stderr PREFIX), NOT byte-equal:**

- **invalid-JSON-on-stdin:** `error: invalid JSON on stdin: {str(JSONDecodeError)}` (exit 1). The spike proved the tail is Python-version-coupled (3.14: `Expecting property name enclosed in double quotes: line 1 column 2 (char 1)`, `Extra data: ...`, `Illegal trailing comma before end of object: ...`). Native Go `encoding/json` produces entirely different tails. Test: exit 1 + stderr begins with `error: invalid JSON on stdin: `.
- **dispatch_file_write_failed:** `dispatch_file_write_failed: {path}: {str(OSError)}` (exit 1, NO `error: ` prefix — distinct shape). Rare IO path; the spike could not benignly trigger it. Test: exit 1 + stderr begins with `dispatch_file_write_failed: {path}: ` (the path is deterministic; the OSError tail is not).

### Behaviors the derivation surface MUST reproduce (all spike-confirmed)

1. **slug-not-stem** (`entity_slug`): folder-form `{slug}/index.md` → folder name (spike: `skill-launcher`, never `index`); flat `{slug}.md` → filename stem. Drives `name`, branch, and `dispatch_file_path`.
2. **split-root state-checkout entity path** (`workflow_is_split_root` = README declares `state:`): worktree stage isolates CODE only — body uses the "for CODE" phrasing + the path-scoped state-commit clause; the entity-read line and completion-signal ref point at the FO-passed `entity_path` UNCHANGED (no `.worktrees/` segment). Non-split-root worktree stages rewrite the entity path into the worktree (`relpath`+join → `{worktree}/{rel}`, spike-confirmed).
3. **worktree stickiness** (Rule 4): route on the entity's stamped `worktree:` frontmatter field, not the stage's declared mode.
4. **effective_model precedence**: stage > defaults > null (spike: opus from stage beats haiku defaults; haiku from defaults when stage unset; null when neither). Validate declared values against `sonnet|opus|haiku`.

### Golden test plan

Matches the `internal/status` harness pattern (`harness_test.go` split-channel runners). Determinism: `build` output has no timestamps and `dispatch_file_path` is a fixed `/tmp/spacedock-dispatch/{name}.md` derived purely from inputs, so NO normalization is needed — raw byte compare.

- **Mechanism-first (cheap, runs first):** the riskiest fixture — split-root + folder-form + worktree — diffed native-vs-oracle across all three channels. The spike already captured these oracle bytes; if the native slug-not-stem + split-root entity-path derivation matches here, the rest is breadth.
- **Cross-product fixtures (breadth):** flat × folder, split-root × single-root, worktree × non-worktree, team × bare. Plus: a model-precedence triple (stage/defaults/null — locks the `[build] effective_model` stderr channel + the `null` JSON literal), a bare-mode fixture (locks the `WARN bare_mode` stderr + absent `name`/`team_name` keys), a `<`/`>`/`&`-bearing field fixture (locks no-HTML-escape), a space-bearing-path fixture (locks shlex quoting), and a feedback+scope fixture (locks those two conditional blocks). Plus the 18 negative byte-identical error fixtures and the 2 structural-prefix error fixtures.
- **Oracle invocation:** drive the project-vendored `claude-team build` under `python3` (reuse/adapt `vendoredClaudeTeam`) and native `spacedock dispatch build`, both with pinned `HOME=t.TempDir()` (hermetic team-probe) and pinned env, separate stdout/stderr buffers. The oracle is in-repo so it is always present (no skip-when-absent needed for the project-vendored copy).
- **Cost:** Go unit/golden tests; each fixture is a `t.TempDir()` git-init + README + entity write + two process runs. Seconds per fixture. Estimated ~25-30 fixtures (18 error + ~12 positive/branch).

## Decision 3 — Disjoint Go package boundary + helper-sharing seam (closes AC-3 disjointness)

`build` needs pure helpers that today live UNEXPORTED in `package status` (symbol survey done this cycle):
- `parseFrontmatter(path) map[string]string` — `internal/status/frontmatter.go:82`
- `parseStagesWithDefaults(path) ([]stage, map[string]string)` — `internal/status/stages.go:68`
- `findGitRoot(startDir) string` — `internal/status/path.go:76`
- the folder-slug rule — encoded in `internal/status/discover.go` (folder `{slug}/index.md` → folder name); needs a small exported `EntitySlug(path)` accessor.
- `extractStageSubsection` (README `### {stage}` parser) — does NOT yet exist in Go; it is net-new and lives in `internal/dispatch`.

Same-package collision is a REAL risk, not hypothetical: commit `356c7e7` ("rename test helper contains -> containsSlug (resolve package-status redeclare)") records an actual `package status` redeclaration collision during status work. Adding `build` directly into `package status` reopens that risk.

Decision: implement `build` + `show-stage-def` in a NEW package `internal/dispatch`, routed from a NEW `cli.go` switch case `dispatch` (disjoint from the existing `case "status":` arm — the two never share a hunk; the new arm calls a `runDispatch` sibling of `runStatus`). Command surface: `spacedock dispatch build --workflow-dir {wd}` (stdin JSON → stdout JSON) and `spacedock dispatch show-stage-def --workflow-dir {wd} --stage {stage}`. The command name `dispatch` (over `team`) names the reimplemented surface — the dispatch path — and the emitted fetch line `spacedock dispatch show-stage-def` is self-describing.

**Helper-sharing seam: export-in-place** (Cycle 1 directive, FO-resolved). `internal/dispatch` imports `internal/status` and uses exported `status.ParseFrontmatter`, `status.ParseStagesWithDefaults`, `status.FindGitRoot`, `status.EntitySlug`. This is the Cycle-1-mandated approach over the Cycle-0 leaf-lift (`internal/spec`), and it matches the Python precedent: the Python `claude-team` imports `parse_frontmatter`/`parse_stages_with_defaults`/`find_git_root` from its sibling `status` module rather than relocating them. Rationale for rejecting the leaf-lift: it would rewrite `handlers.go`/`stages.go`/`frontmatter.go`/`path.go` to move the definitions out, colliding with the concurrent agent-output-modes work touching `package status`. Export-in-place adds renames (`parseFrontmatter` → `ParseFrontmatter`, etc.) confined to the four definition sites + their `package status` call sites — a mechanical capitalization, no cross-package move, disjoint from `internal/dispatch`'s own files.

**One wrinkle the implementer must resolve:** `ParseStagesWithDefaults` returns `[]stage`, and `stage` (with fields `name`, `worktree`, `optional map[string]string` carrying `agent`/`model`) is unexported (`stages.go:17`). Export-in-place therefore requires EITHER exporting the `stage` type and the fields `build` reads (`Name`, `Worktree`, and an accessor for the `optional` map's `agent`/`model` entries), OR adding a thin exported accessor surface. Recommend exporting `Stage` with exported fields + a `Model()`/`Agent()` accessor over the optional map, so `internal/dispatch` reads stage metadata without reaching into an unexported map. This is the only non-trivial part of the seam; it stays within the rename-in-place discipline (no logic change) and is a small reviewable diff landed BEFORE the `build` implementation.

The renames touch `package status` files, so this is NOT a zero-status-edit approach — but per the Cycle 1 FO resolution, export-in-place is preferred precisely because it avoids the leaf-lift's larger `handlers.go` rewrite. If the concurrent agent-output-modes work and these renames still collide at merge, the renames are isolated to their own commit and can be rebased independently of the `internal/dispatch` implementation.

## Refined acceptance criteria

The `## Acceptance criteria` section above was rewritten this cycle (AC-1 semantic-equivalence, AC-1b str(e)-carve-out, AC-2 deterministic predicate, AC-3 deferral + behavioral guard). This section records the concrete code targets those ACs imply:
- AC-3's scope is decided: native = `build` + `show-stage-def`; deferred to sibling `claude-runtime-segregation` = `context-budget`, `spawn-standing`, `list-standing`, `show-standing`, and `build`'s `_mods`/standing fetch-line branch.
- AC-2's native command is named: `spacedock dispatch build` and `spacedock dispatch show-stage-def`. The vendored FO runtime adapter's MANDATORY dispatch block (`skills/first-officer/references/claude-first-officer-runtime.md`) changes ONE line — line 95 — from `echo '<json>' | {project_root}/skills/commission/bin/claude-team build --workflow-dir {workflow_dir}` to `echo '<json>' | spacedock dispatch build --workflow-dir {workflow_dir}`. The `build`-emitted fetch line (vendored `claude-team` lines 408-411) changes `claude-team show-stage-def` → `spacedock dispatch show-stage-def`. The `list-standing` (line 34), `spawn-standing` (line 44), and `context-budget` (line 160) references in the FO adapter are NOT dispatch-path and stay as-is — they invoke the deferred subcommands the sibling entity owns; AC-2's predicate scopes its "zero claude-team" assertion to the dispatch block + emitted fetch lines, which is satisfied.

## Test plan

1. **Golden parity (AC-1, AC-1b)** — `internal/dispatch` golden/parity tests, mechanism-first then cross-product, per Decision 2. Three-channel split compare (stdout JSON with fetch carved out / dispatch body with fetch line rewritten / exit+stderr). Gate: native `build` semantically equivalent to project-vendored `claude-team build` across flat/folder × split/single × worktree/non-worktree × team/bare, plus model-precedence, bare-mode, HTML-escape, shlex-quote, feedback+scope, and the 18 byte-identical + 2 structural-prefix error fixtures. Go tests, seconds each.
2. **Native `show-stage-def` parity** — diff `spacedock dispatch show-stage-def` vs project-vendored `claude-team show-stage-def` across well-formed, decorated-heading, malformed-heading, and missing-stage fixtures. The malformed-heading ValueError text is a hand-built f-string (NOT a Python `str(e)`), so the spike confirms it is byte-reproducible — assert it byte-for-byte. Go tests.
3. **Static skill predicate (AC-2)** — extend `skills/integration/skill_text_test.go` with a predicate that is true iff: the FO adapter's dispatch block pipes to `spacedock dispatch build` (not `{project_root}/skills/commission/bin/claude-team build`), AND no `claude-team` token remains in that dispatch block. Pair it with the AC-1 body-parity test that observes the emitted fetch line is `spacedock dispatch show-stage-def` (the emitted bytes, not prose). The `context-budget`/`list-standing`/`spawn-standing` lines may retain `claude-team` (sibling-owned). Update `dispatch_test.go` to drive the native binary as primary, keeping a Python-oracle parity arm.
4. **Ensign bootstrap (AC-2)** — confirm a dispatched ensign's first-action fetch command (`spacedock dispatch show-stage-def`) resolves against the native binary with the binary on PATH, closing the "claude-team not on PATH" gap. Verified by the AC-1 emitted-fetch-line parity plus the native `show-stage-def` parity test; a live workflow smoke is NOT required because the fetch-line content is the testable claim.
5. **Deferred-subcommand guard (AC-3)** — a behavioral test asserting `spacedock dispatch context-budget` (or any deferred/unknown subcommand) exits non-zero with a usage diagnostic, so the deferral is observable rather than a silent no-op.
6. **Baseline** — `go test ./...` and `go test ./... -race` stay green (246 tests baseline today).

## Stage Report: ideation

- DONE: Scope the native dispatch subcommand surface: which claude-team subcommands are reimplemented natively (build is load-bearing) vs kept as thin shims / scoped out (context-budget, list-standing, spawn-standing — coupled to Claude Code transcripts/Agent specs). Record and justify the decision.
  Decision 1 in body: native = `build` + `show-stage-def` (the dispatch round-trip); scoped out = `context-budget` (jsonl/transcript-coupled, advisory), `spawn-standing` (Agent-spec/team-coupled), `list-standing`/`show-standing` (standing-teammate feature, absent from the `docs/dev` self-hosted path). Asymmetry noted: native `build` reproduces the show-standing fetch-line branch for parity but the subcommand stays Python (inert — no `_mods/` in dev workflow).
- DONE: Define the byte-identical build parity contract + golden test plan vs the vendored Python claude-team build across flat/folder, split-root/single-root, worktree/non-worktree fixtures (slug-not-stem name/branch/dispatch-file derivation + split-root state-checkout entity path).
  Decision 2 + Test plan in body: byte-compare stdout JSON + dispatch-file body + stderr advisories + exit codes; mechanism-first fixture (split-root + folder + worktree) then cross-product; oracle is the PROJECT-VENDORED `claude-team` (not the plugin copy — they diverge on `entity_slug`/`workflow_is_split_root`); reuse the `internal/status` golden-harness pattern; deterministic output needs no normalization.
- DONE: Confirm a disjoint Go package boundary (e.g. internal/dispatch + a new subcommand) so implementation will not same-package merge-collide with status work; design the native command name (e.g. spacedock dispatch build).
  Decision 3 in body: new `internal/dispatch` package + new `dispatch` cli switch case (disjoint hunk from `status`); command names `spacedock dispatch build` and `spacedock dispatch show-stage-def`. Same-package collision risk validated by commit 356c7e7 (a real `package status` redeclare). Recommend lifting the 3 pure parsers (`parseFrontmatter`/`parseStagesWithDefaults`/`findGitRoot` + folder-slug rule) into a leaf `internal/spec` package both packages import (Go analog of the Python RUNTIME-NEUTRAL borrowed surface); export-in-place flagged as fallback for the gate.

### Summary

Fleshed out the native dispatch-helper task into problem statement, three scope/design decisions, refined ACs, and a test plan. Key findings the implementer must not miss: the parity oracle is the PROJECT-VENDORED `skills/commission/bin/claude-team` (carries the Stage-4 slug-not-stem + split-root amendments), NOT the plugin copy which still derives `index` for folder-form entities; Stage 7 is already merged so the sequencing caveat is satisfied; and `package status` has a documented prior same-package collision (356c7e7), making the disjoint-package boundary load-bearing rather than precautionary. Two open choices left for the gate: leaf-package lift (recommended) vs export-in-place for sharing the 3 pure parsers, and the `dispatch` vs `team` command-name framing (recommend `dispatch`). No frontmatter modified; ideation committed to main (non-worktree entity).

## Feedback Cycles

### Cycle 1 — ideation gate REJECT (staff audit, 2026-05-30)

Three-lens adversarial staff audit returned material-concerns; rejected to ideation. Captain- and FO-resolved directives the revision MUST honor:

- **Parity north-star reframed (captain).** NOT byte-identical to the Python oracle. Target = a semantically-equivalent dispatch spec: byte-identical on the non-fetch channels (the spec JSON minus fetch_commands, the dispatch-file body, exit codes), with the `show-stage-def` fetch line rewritten to `spacedock dispatch show-stage-def`. The Python `str(e)` error paths (invalid-JSON L206, dispatch_file_write_failed L528) are structurally-equivalent, not byte-equal (carve them out explicitly). **Scope split (captain): the `_mods`/standing surface — `build`'s show-standing fetch-line branch, the `show-standing`/`list-standing`/`spawn-standing` subcommands, AND `context-budget` — all MOVE to the new sibling entity `claude-runtime-segregation` (zse4a3ds0x19gpdcjh7anhgs).** native-dispatch-helper's `build` is therefore scoped to NON-`_mods` workflows (the self-hosted `docs/dev` path has no `_mods/`); it needs no `parse_mod_metadata` and emits no show-standing line. Parity (AC-1) is tested on non-`_mods` workflows; the sibling closes the `_mods` parity gap. This resolves the Decision-1-vs-AC-2 conflict by DEFERRAL rather than by pulling standing into native scope — keeping this entity lean.
- **Helper-sharing seam (FO).** export-in-place — `internal/dispatch` imports `internal/status`'s exported parsers — NOT the leaf-lift, which would rewrite `handlers.go` and collide with the concurrent agent-output-modes work. Matches the Python precedent (dispatch imports the sibling status module).
- **Sequencing (captain).** This entity lands FIRST, before spacedock-packaging.

Revision must run a **SPIKE**: execute the project-vendored Python oracle on a model-precedence fixture (stage-set / defaults-set / neither) + a malformed-stdin fixture + the riskiest-derivation fixture (split-root + folder-form + worktree); capture exact stdout/stderr/exit bytes; define the parity contract (byte-identical vs rewritten vs structurally-equivalent) from observed reality. Then: enumerate EVERY distinct error byte-string (~22, not "12"); add the model-precedence + bare-mode-no-team fixtures (the `[build] effective_model` and `WARN bare_mode` stderr channels are uncovered today); rewrite the test helper to split stdout/stderr (`CombinedOutput` cannot byte-compare both); recast AC-2 as a deterministic predicate (the FO dispatch block AND every emitted fetch line target `spacedock dispatch`; the `context-budget`/`list-standing`/`spawn-standing` FO references may retain `claude-team` — those subcommands are the sibling entity `claude-runtime-segregation`'s concern, not this entity's).

## Stage Report: ideation (cycle 2)

- DONE: Run the SPIKE first: execute the PROJECT-VENDORED Python oracle (skills/commission/bin/claude-team build) on a model-precedence fixture (stage-set/defaults-set/neither) + a malformed-stdin fixture + the riskiest-derivation fixture (split-root + folder-form + worktree); capture exact stdout/stderr/exit bytes; record what is byte-identical vs rewritten (show-stage-def fetch line -> spacedock dispatch) vs structurally-equivalent (Python str(e) errors).
  Ran the spike under Python 3.14 across model-precedence (stage→opus / defaults→haiku / null), malformed-stdin, riskiest split-root+folder+worktree, bare-mode, non-split-root-worktree, and feedback+scope fixtures, plus a full error enumeration; artifacts under /tmp/spike-build/. Findings drove the parity contract: stdout JSON needs trailing `\n` + `SetEscapeHTML(false)` + `null` literal + fixed key order; the `[build] effective_model … → …` arrow is U+2192; one fetch line is rewritten; invalid-JSON and dispatch_file_write_failed are the only str(e) paths (the malformed-heading text is a hand-built f-string, byte-reproducible).
- DONE: Revise the design + ACs per the entity's ## Feedback Cycles -> Cycle 1 directives: semantic-equiv parity contract grounded in the spike bytes; build SCOPED TO NON-_mods workflows (the _mods/standing surface + context-budget are the SIBLING entity claude-runtime-segregation's concern, not this one); export-in-place seam; enumerate EVERY distinct error byte-string; model-precedence + bare-mode fixtures; split-stdout/stderr test helper; AC-2 as a deterministic predicate.
  Rewrote AC-1 (semantic-equiv, three-channel split compare), added AC-1b (str(e) carve-out: 18 byte-identical + 2 prefix-only errors), recast AC-2 (deterministic predicate scoped to dispatch block + emitted fetch line; sibling-owned refs may retain claude-team), recast AC-3 (deferral to claude-runtime-segregation + a behavioral guard that the deferred subcommand exits non-zero). Decision 1 dropped the _mods/standing branch entirely (build is non-_mods, no parse_mod_metadata). Decision 2 grounded in spike bytes incl. the split-stdout/stderr requirement and the 18+2 error enumeration. Decision 3 switched to export-in-place and surfaced the unexported-`stage`-type wrinkle.
- DONE: Confirm each revised AC is testable by an EXERCISE-AND-OBSERVE oracle (not a grep over code or prose); name the behavioral oracle per AC.
  AC-1: run native+oracle build on identical input, observe three-channel byte compare. AC-1b: run both on error inputs, observe exit+stderr (byte-equal 18 / prefix 2). AC-2: static predicate over the FO adapter dispatch block (the artifact under change) PAIRED with the AC-1 body-parity test that observes the emitted fetch-line bytes from a real build run — not a standalone prose grep. AC-3: run `spacedock dispatch context-budget`, observe non-zero exit + usage diagnostic (converts prose-deferral into observed behavior).

### Summary

Revision after the Cycle-1 ideation-gate REJECT. Ran the spike first and rebuilt the parity contract from observed bytes rather than from reading the Python source: the contract is semantic-equivalence (three channels compared independently, one fetch line intentionally rewritten, two str(e) paths carved to prefix-only), not byte-identity. Honored the captain scope split — `build` is now scoped to NON-`_mods` workflows with the entire standing/context-budget surface deferred to sibling `claude-runtime-segregation`, so native `build` carries no `parse_mod_metadata` and emits no show-standing line. Adopted the FO-directed export-in-place seam and surfaced the one real wrinkle (the unexported `stage` return type needs exporting or accessors). Every AC now names an exercise-and-observe oracle; the AC-3 behavioral guard replaces prose-only deferral. Two byte-level parity hazards the implementer must not miss: Go's default JSON HTML-escaping (needs `SetEscapeHTML(false)`) and the missing trailing newline (`MarshalIndent` omits it). No frontmatter modified; committed to the state checkout (non-worktree ideation stage).
