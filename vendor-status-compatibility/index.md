---
id: jepnnnp1jr3zjzes2rz6ng6g
title: Vendor status compatibility
status: ideation
score: "0.90"
source: bootstrap roadmap
worktree:
started: 2026-05-30T04:24:41Z
---

# Vendor Status Compatibility

Make `spacedock status` preserve the current status script behavior through a vendored compatibility path with a narrow Go interface that Stage 5 (native-go-status) can swap for native code without changing callers.

## Problem Statement

`spacedock status` currently exits 2 with `not implemented yet` (`internal/cli/cli.go:23`). Every first-officer status operation today runs the plugin-shipped Python script at `skills/commission/bin/status` — a 2547-line stdlib-only program that is the single source of truth for workflow state. The FO depends on this script's exact behavior in load-bearing ways:

- It **parses `--boot` output by section** (MODS / ID_STYLE / NEXT_ID / ORPHANS / PR_STATE / DISPATCHABLE / TEAM_STATE) at startup (`first-officer-shared-core.md:14`).
- It **trusts `--set` stdout** (`field: old -> new`, one line per field) to narrate mutations without re-reading the entity file, specifically to avoid the Claude Code file-staleness echo (`first-officer-shared-core.md:335-336`).
- It **parses `--resolve` output** (`workflow= scope= slug= id= path=`) for deterministic lookup, and reads `--next` / default table rows to choose dispatchable entities.
- It relies on **mutation guards** in `--set`/`--archive` (mod-block, merge-hook, terminal-transition invariants) and on **exit codes** to gate the event loop. The exit domain is exactly `{0 success, 1 error}` — there is no exit-2 path; usage errors (missing flag argument, bad `--set` field) also exit 1, and an unrecognized top-level flag is silently ignored and falls through to the default table at exit 0.

The bootstrap is compatibility-first (`AGENTS.md`): the launcher must preserve every observable property of the current script before any new semantics. The roadmap explicitly forbids reimplementing in native Go here — that is Stage 5 (native-go-status, roadmap §5) — and forbids `state:` split-root resolution here — that is Stage 6 (native-state-dir, roadmap §6). The risk this stage protects against is the launcher silently changing status behavior the FO has already encoded its instructions around.

The design problem is therefore: **route `spacedock status` to the current script's behavior unchanged, behind an interface narrow enough that Stage 5 can replace the runner with native Go and keep every caller and test green.**

## Proposed Approach

**Chosen: vendor-copy-and-exec.** Copy `skills/commission/bin/status` verbatim into the launcher repo (proposed: `internal/status/vendor/status`), and have `spacedock status` exec the vendored copy under `python3`, passing all arguments through and streaming stdout/stderr/exit-code transparently. The seam is a single narrow interface in `internal/status/`:

```
type Request struct {
    Args   []string          // forwarded verbatim, incl. --workflow-dir
    Dir    string            // working directory (cwd); load-bearing for --discover
    Env    []string          // explicit environment; load-bearing vars listed below
    Stdin  io.Reader         // consumed by Stage 5 native --new < STDIN; pass-through today
    Stdout io.Writer
    Stderr io.Writer
}

type Runner interface {
    Run(ctx context.Context, req Request) (exitCode int, err error)
}
```

`spacedock status <args...>` builds a `Request`, forwards `<args...>` (including `--workflow-dir`), and returns the runner's exit code. The vendored-exec implementation backs this interface today; Stage 5 supplies a native-Go implementation behind the same interface with no caller change.

**Why `Dir` and `Env` are explicit on the interface (not ambient).** The script's output depends on more than argv: ambient environment and working directory are part of its observable inputs, so the seam must expose them or AC-4's decoupling/pinning test cannot reproduce output. Load-bearing variables, each confirmed against the oracle: `USER`/`USERNAME` and `SPACEDOCK_ID_ACTOR`/`SPACEDOCK_ID_CONTEXT`/`SPACEDOCK_TEST_SD_B32_TIMESTAMP` (feed the sd-b32 `--next-id`/`--boot` NEXT_ID digest — changing `USER` alone changes the id), `HOME` (the `--boot` TEAM_STATE probe scans `$HOME/.claude/teams/*/config.json`), and `PATH` (PR_STATE locates `gh`; the runner itself locates `python3`). `Dir` is load-bearing for `--discover` without `--root`, which derives its scan root from `git rev-parse --show-toplevel` in the cwd, falling back to `os.getcwd()`. `Stdin` is retained deliberately: Stage 5's native `--new` reads the entity body from stdin (`--new < STDIN`), so the seam must carry it forward now even though the vendored read/mutation flags do not consume it.

**Why vendor-copy-and-exec over the alternatives:**

- **vs. delegate-to-path** (exec the live `skills/commission/bin/status` in the plugin dir): rejected. Stage 4's AC is "no skill path references plugin-private `skills/commission/bin/status`" (roadmap §4). Delegating to that path keeps the launcher coupled to plugin internals and to whatever version happens to be installed, so golden fixtures could not pin a known script. Vendoring makes the launcher self-contained, version-pinned, and reproducible — which is exactly what makes byte-for-byte golden comparison meaningful. It also matches the stated migration goal in `AGENTS.md` ("migrate Spacedock from plugin-shipped scripts to a stable command surface").
- **vs. reimplement in native Go now**: forbidden by the roadmap and `AGENTS.md` (compatibility-first; native parity is Stage 5). Reimplementing now would duplicate Stage 5 and risk diverging from the oracle before golden fixtures exist to catch it.
- **vs. embed the script via `//go:embed` and write to a temp file at runtime**: deferred as an implementation detail of the vendored-exec Runner, not an interface decision. The contract surface (argv → stdout/stderr/exit/fs-mutation) is identical either way; the implementer may embed-and-exec or exec a vendored file on disk, whichever keeps the test gates green. The narrow interface above does not depend on the choice.

**Interface contract (the load-bearing design goal).** The full observable contract of the current script — and therefore of the `Runner` — is:

```
(args, dir, env, stdin) -> (stdout bytes, stderr bytes, exit code, filesystem mutations under workflow_dir)
```

The launcher adds nothing to and removes nothing from this contract. It does not parse, reformat, or reorder any output; it does not interpret or strip flags (it forwards argv verbatim — including unrecognized tokens — so the vendored script applies its own argument semantics); it does not translate exit codes. This is what lets Stage 5 drop in a native implementation: the native code must reproduce the same outputs for the same `(args, dir, env, stdin)`, verified by the same golden + mutation tests written in this stage. Path resolution, mutation guards, and validation stay owned by the runner (per `AGENTS.md`: "Let the binary own path resolution and mutation guards"), not the CLI layer.

**Vendoring landmine — `--workflow-dir` is mandatory.** The script resolves its target directory as `workflow_dir or $PIPELINE_DIR or dirname(__file__)` (`status` main(), ~line 2128). Once vendored, `dirname(__file__)` is `internal/status/vendor/` — an empty directory — not the plugin's commission dir. So a no-`--workflow-dir` invocation under the launcher would scan an empty dir and *appear* to match the oracle only by coincidence (both empty), masking a real divergence on any non-empty cwd. The launcher therefore must always pass `--workflow-dir`, and the test plan both mandates it and adds an explicit omitted-`--workflow-dir` case so the fallback difference is observed rather than hidden.

**Scope guards.** This stage does NOT add `state:` split-root resolution (Stage 6) — the vendored script already reads stages from `.spacedock-state/README.md` via the existing symlink, so no split-root logic is needed. It does NOT reimplement parsing in Go (Stage 5). It changes only `internal/cli/` routing for `status` and adds `internal/status/` plus the vendored script and tests; it does not touch other product code.

## Acceptance Criteria

**AC-1 - `spacedock status` reproduces the current script's stdout, stderr, and exit code for every FO-load-bearing flag.**
The launcher forwards arguments to the vendored runner verbatim and returns its exit code unmodified for the flag set the FO depends on: `--workflow-dir`, default table, `--next`, `--next-id`, `--validate`, `--resolve`, `--short-id`, `--boot`, `--where`, `--archived`, `--discover`, plus `--id-seed`/`--id-actor` (sd-b32 creation) and the `--fields`/`--all-fields`/`--root`/`--force` modifiers. The exit domain is `{0, 1}` — the launcher never injects an exit-2 path of its own for `status`. The CLI layer neither validates nor strips flags: argv is forwarded unchanged so the vendored script applies its own argument semantics. That semantics is not "ignore unknowns" — an unrecognized token mid-`--set` truncates the field list (`--set alpha --bogus status=done` drops `status=done` and exits 1), while an unrecognized top-level flag falls through to the default table at exit 0. The launcher reproduces whichever behavior the runner produces.
Verified by: Go fixture tests that run `spacedock status <flag...>` against a checked-in workflow fixture and assert stdout/exit-code equal the vendored script's output for the same args (`internal/cli`, `internal/status`); a routing test asserting argv reaches the runner unmodified, including the `--set alpha --bogus status=done` truncation case (exit 1, `status` unchanged) and the unknown-top-level-flag case (exit 0, default table).

**AC-2 - Read-path output is byte-for-byte identical to the current script for pinned fixtures.**
For a frozen workflow fixture, the launcher's stdout for the default table, `--next`, `--validate`, `--resolve`, and `--short-id` matches the current script byte-for-byte after a defined normalization for env-dependent fields (see test plan). The launcher itself injects no formatting differences; any normalization is applied identically to both oracle and launcher output in the test, not in the product.
Verified by: golden fixture tests in `internal/status` comparing launcher stdout to golden files captured from the current script (the oracle) for each of the five read subcommands.

**AC-3 - Mutation commands preserve current frontmatter semantics and narration.**
`--set` and `--archive` through `spacedock status` produce the same frontmatter edits and the same stdout narration (`--set`: `field: old -> new` / `field: old -> ` clear / `field:  -> {ts}` bare-timestamp fill; `--archive`: `archived: {dest}`, where `{dest}` tracks the `--workflow-dir` spelling — a relative `--workflow-dir .` yields `archived: ./_archive/{slug}.md`, an absolute one yields an absolute dest), enforce the same mutation guards (mod-block, merge-hook, terminal-transition rejection with exit 1), and stamp timestamps in the same ISO-8601 UTC format.
Verified by: temp-workflow mutation tests in `internal/status` that run `--set` (field set, clear, bare-timestamp fill) and `--archive` through the launcher, then assert the resulting frontmatter and the captured stdout (timestamps normalized); guard tests asserting a terminal `--set` under an active mod-block exits 1 with the current error text.

**AC-4 - The compatibility layer exposes a narrow `Runner` interface that Stage 5 can back with native Go without changing callers.**
A single interface in `internal/status/` defines the whole seam — `Run(ctx, req) -> (exitCode, err)` where `req` carries `Args`, `Dir`, `Env`, `Stdin`, `Stdout`, `Stderr`. `internal/cli/` depends only on this interface, not on the vendored script, `python3`, or any exec detail. Because `Dir` and `Env` are explicit, a test can inject a fake `Runner` with pinned env/cwd and reproduce id/timestamp/discovery output deterministically — the property Stage 5's native implementation must also satisfy.
Verified by: a Go test that injects a fake `Runner` into the `status` command path and asserts the CLI forwards `Args`/`Dir`/`Env`/`Stdin` and returns the runner's exit code unchanged; a test that pins `Env` (`USER`, `SPACEDOCK_TEST_SD_B32_TIMESTAMP`, `--id-seed`, `--id-actor`) and asserts the sd-b32 `--next-id` candidate is reproducible; a compile-time assertion that the vendored-exec type satisfies `Runner`.

## Test Plan

All proof is **Go unit/fixture tests** at the level of the claim (CLI routing + status command behavior), per the ideation guidance. Baseline gate: `go test ./...` and `go test ./... -race` (`AGENTS.md`). Estimated complexity: moderate — the runner and routing are thin; the work is in the parity harness and normalization. No live workflow tests needed; the oracle is a local script.

**Invocation invariant.** Every launcher-side and oracle-side invocation in the harness passes an explicit `--workflow-dir` pointing at the fixture/temp dir. This is mandatory because of the vendoring landmine above (the `dirname(__file__)` fallback differs between the live plugin script and the vendored copy). One dedicated negative test omits `--workflow-dir` and asserts the launcher does NOT silently fall back to `dirname(__file__)` of the vendored script — it must surface the empty-dir scan rather than masquerade as parity. Both oracle and launcher also run under a pinned env (see locale/version pin below) so id and timestamp material is reproducible.

**Parity harness shape.**
- **Golden read fixtures (AC-1, AC-2).** Check in a small, frozen workflow fixture under `internal/status/testdata/` (a README with a stages block and a handful of flat + folder-form entities, covering at least one sd-b32 and one sequential style in separate fixtures). For each read subcommand (default, `--next`, `--validate`, `--resolve`, `--short-id`), capture a golden file by running the current script (the oracle) once at fixture-creation time, and the test compares launcher stdout against the golden after normalization. A regenerate path (e.g. `go test -run TestGolden -update` or a documented script) re-captures goldens from the oracle so drift is visible in review, never silently absorbed.
- **Temp-workflow mutation tests (AC-3).** Each mutation test copies the fixture into `t.TempDir()`, `git init`s it, runs the mutation through the launcher with an explicit `--workflow-dir`, and asserts (a) the resulting entity frontmatter and (b) the captured stdout. Run the same mutation through the oracle into a second temp dir and diff the resulting files + stdout (normalized) to prove parity rather than hand-asserting expected strings. Include one `--archive` case with a relative `--workflow-dir .` (asserting `archived: ./_archive/{slug}.md`) and one with an absolute `--workflow-dir` so the dest-spelling behavior is locked. Include a guard case: a terminal `--set` under an active mod-block exits 1 with the current error text.
- **Interface decoupling (AC-4).** Fake-`Runner` injection test in `internal/cli`; compile-time `var _ status.Runner = (*vendorExecRunner)(nil)`.

**Byte-for-byte risk areas and the normalization/pinning strategy for each** (each confirmed by running the oracle during ideation):

- **Wall-clock timestamps** — `--set <bare-timestamp>` and `--archive` stamp `datetime.now(timezone.utc)` as second-precision `YYYY-MM-DDTHH:MM:SSZ`; this is NOT overridable by any env var. Note the sd-b32 path uses a *different* shape — microsecond `YYYY-MM-DDTHH:MM:SS.ffffffZ` from `sd_b32_timestamp()` (pinnable via `SPACEDOCK_TEST_SD_B32_TIMESTAMP`). *Strategy:* do not pin the mutation stamp; normalize. The test replaces any ISO-8601 UTC timestamp with a placeholder using a regex that matches BOTH shapes — `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z` → `<TS>` — in both oracle and launcher output before comparison, asserting shape and field placement, not the instant. The trailing `(\.\d+)?` is required or the microsecond sd-b32 timestamp would slip through un-normalized and break comparison.
- **Path spelling tracks the input** — `--resolve` emits `workflow=` (run through `os.path.realpath`, so on macOS `/var` → `/private/var`) and `path=` (NOT realpath'd, echoes the `--workflow-dir` spelling joined with the entity name). `--archive`'s `archived:` dest is NOT always absolute: it is `{--workflow-dir}/_archive/{slug}.md` verbatim, so `--workflow-dir .` yields `./_archive/{slug}.md` and an absolute `--workflow-dir` yields an absolute dest. *Strategy:* fix the `--workflow-dir` spelling per test (absolute for the golden-resolve cases), run both oracle and launcher against the same root, and normalize that root prefix to a placeholder; apply `realpath` to the expected root for the `workflow=` field specifically so the macOS symlink rewrite is accounted for, but NOT for `path=`/`archived:` which are not realpath'd. Never bake a machine-specific absolute path into a golden file.
- **sd-b32 `--next-id` / `--boot` NEXT_ID** — SHA-derived from workflow realpath + seed + actor (`$USER` fallback) + `SPACEDOCK_ID_CONTEXT` + timestamp + nonce; non-deterministic across environments. *Strategy:* pin all material — set `SPACEDOCK_TEST_SD_B32_TIMESTAMP` (the script's test hook), pass explicit `--id-seed` and `--id-actor`, and set/clear `SPACEDOCK_ID_CONTEXT` — so the candidate is reproducible; OR exclude raw `--next-id` value from byte-comparison and instead assert format (24 chars in the SD-B32 alphabet) plus equality against the oracle run with the identical pinned env. `--next-id` is in scope for AC-1 pass-through but is asserted at format+oracle-equality level, not against a static golden.
- **`--boot` env-dependent sections** — `TEAM_STATE` reads `~/.claude/teams/*/config.json` mtime within a 30-minute window (ambient, machine-specific); `PR_STATE` shells out to `gh`; `ORPHANS` calls `git worktree list`. *Strategy:* `--boot` is verified at the structural level (section headers present and in order, ID_STYLE/NEXT_ID/DISPATCHABLE bodies parity-checked) rather than whole-output byte parity. The volatile sections (`TEAM_STATE` hint line, live `PR_STATE`/`ORPHANS` rows) are normalized away; the fixture is constructed with no orphans and no PRs so those sections render their deterministic `none` forms. This keeps the FO-parsed sections covered without making the golden flaky on team/gh/worktree state.
- **Entity ordering** — `discover_entity_files` sorts by slug; tables sort by `(stage_order, -score)` with empty score sorting last. Deterministic given fixed input. *Strategy:* the fixture pins distinct scores and stages so ordering is unambiguous; one fixture entity intentionally has an empty score to lock the "empty sorts last" rule. No normalization needed — this is captured by the golden directly.
- **`python3` availability / interpreter** — the vendored-exec runner depends on a `python3` on PATH. *Strategy:* tests assume `python3` (the oracle requires it too); a runner-level test asserts a clear error path (non-zero exit, diagnostic on stderr) when the interpreter is missing, so the launcher fails loudly rather than silently mis-reporting status. The vendored script's `{spacedock_version}` etc. placeholders live only in comments and do not affect execution (confirmed: the script parses and runs as-is).
- **Locale and interpreter version are part of the oracle pin (AC-2 reproducibility)** — byte-for-byte output is only reproducible if the bytes are produced the same way across machines/CI. A C-locale CI could alter Unicode handling (the script emits a `…` ellipsis when truncating extra-field cells, and reads UTF-8 frontmatter). *Strategy:* run both oracle and launcher under a pinned locale — `PYTHONUTF8=1` and `LANG=C.UTF-8` in the harness `Env` — and treat the `python3` version as part of the oracle pin: record the interpreter version alongside the goldens and regenerate goldens under the same pinned locale + interpreter so AC-2 is reproducible rather than machine-dependent. This is exactly why `Env` is explicit on the seam.
- **Context cancellation reaches grandchildren** — the script spawns three subprocess sites (`git worktree list` in `--boot` ORPHANS, `gh pr view` in PR_STATE, `git rev-parse` in `--discover`). A naive `exec.CommandContext` cancel kills only the python child, leaving git/gh grandchildren orphaned. *Strategy:* either start the python child in its own process group and signal the group on cancel (so grandchildren die with it), or explicitly document cancellation as best-effort and out of parity scope. A runner test cancels mid-`--boot` and asserts no orphaned grandchild survives the chosen contract. This is a runner-implementation property, not a Stage-5 parity requirement, but it is named here so the implementer does not discover the orphan leak after wiring the seam.

## Notes

This stage does not add `state:` split-root behavior (that is Stage 6, native-state-dir) and does not reimplement status parsing in native Go (that is Stage 5, native-go-status). It protects the known contract first: route `spacedock status` to the current script's behavior unchanged, behind a narrow `Runner` interface that Stage 5 replaces. The current script at `skills/commission/bin/status` is the compatibility oracle for all golden fixtures.

## Stage Report: ideation

- DONE: Choose and justify the compatibility approach; honor the roadmap (vendor/delegate, not native Go); design a narrow Go interface Stage 5 can back natively.
  Chose vendor-copy-and-exec over delegate-to-path (rejected: violates Stage 4 "no plugin-private path" AC, can't version-pin goldens) and over native-Go-now (forbidden: that is Stage 5). Narrow seam is `Runner.Run(ctx, args, stdin, stdout, stderr) (exitCode, err)` in `internal/status/`; justification in Proposed Approach.
- DONE: Rewrite AC into `**AC-N - property.**` + `Verified by:` format; enumerate FO-depended status flags and byte-for-byte subcommands; specify golden capture from the oracle and `--set`/`--archive` mutation parity.
  AC-1..AC-4 added. Flags enumerated from grep of first-officer skill (`--set`×18, `--workflow-dir`×10, `--next-id`×8, `--next`×6, `--resolve`/`--boot`/`--archive`×4, `--where`/`--id-seed`×3, `--validate`/`--archived`/`--discover`/`--short-id`); ensign skill issues no status calls (FO owns them). Read parity scoped to default/`--next`/`--validate`/`--resolve`/`--short-id`; mutation parity to `--set`/`--archive`.
- DONE: Test plan names parity-harness shape (oracle goldens + temp-workflow mutation) AND flags byte-for-byte risk areas with a concrete strategy each.
  Six risk areas each confirmed by running the oracle during ideation: wall-clock timestamps (normalize, not pinnable), absolute paths incl. macOS `/var`→`/private/var` realpath, sd-b32 `--next-id` (pin `SPACEDOCK_TEST_SD_B32_TIMESTAMP`+seed+actor or format-assert), `--boot` env sections (TEAM_STATE/PR_STATE/ORPHANS structural-only), entity ordering (fixture pins scores incl. one empty), `python3` dependency (loud-failure test).

### Summary

Designed Stage 2 as vendor-copy-and-exec of the current Python `status` script behind a single narrow `Runner` interface in `internal/status/`, with `internal/cli/` depending only on that interface so Stage 5 can drop in native Go unchanged. The contract preserved is `(args, dir, env, stdin) -> (stdout, stderr, exit code, fs mutations)` with the launcher adding no formatting, flag interpretation, or exit-code translation. Key decision: vendor (version-pinned, self-contained) rather than delegate to the plugin path, because Stage 4 forbids referencing that path and goldens need a pinned oracle. All byte-for-byte flakiness risks were verified against the live oracle and given a concrete normalization/pinning strategy; no split-root (Stage 6) or native reimplementation (Stage 5) is introduced here.

## Stage Report: ideation (cycle 2 — staff-review fixes)

- DONE: M1 fact-corrections against the oracle.
  (a) Removed the false "exit 2 usage" claim from Problem Statement and AC-1; verified exit domain is `{0,1}` only (usage errors exit 1; unknown top-level flag falls through to default table at exit 0). (b) Corrected `--archive` dest: tracks the `--workflow-dir` spelling (`--workflow-dir .` → `archived: ./_archive/{slug}.md`), not always absolute. (c) Reworded AC-1 "unknown flags pass through": an unknown token mid-`--set` truncates the field list (`--set alpha --bogus status=done` drops `status=done`, exit 1) — added as a routing test.
- DONE: M2 vendoring landmine.
  Documented the `workflow_dir or $PIPELINE_DIR or dirname(__file__)` fallback (~line 2128) and that vendoring repoints `dirname(__file__)` to an empty dir, risking false parity. Test plan now mandates `--workflow-dir` on every invocation plus a dedicated omitted-`--workflow-dir` negative test.
- DONE: M3 seam vs contract.
  Made `Dir` and `Env` explicit on the interface (`Run(ctx, req)` with a `Request` struct) and listed load-bearing ambient vars (`USER`/`USERNAME`, `SPACEDOCK_ID_*`, `HOME`, `PATH`) and cwd-for-`--discover`, each oracle-confirmed (changing `USER` alone changes the sd-b32 id). Kept `Stdin` and stated its purpose: Stage 5 native `--new < STDIN`. AC-4 Verified-by updated to inject/pin env.
- DONE: M4/M5/M6 folded into the test plan.
  M4: ctx-cancel must signal the python process group so git/gh grandchildren die (3 subprocess sites) or be documented best-effort — added with a cancel test. M5: pin locale (`PYTHONUTF8=1`/`LANG=C.UTF-8`) and treat the `python3` version as part of the oracle pin for AC-2 reproducibility. M6: extended the timestamp-normalization regex to `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z` so the microsecond sd-b32 isoformat is also normalized.
- SKIPPED: P1 (drop "large/streamed output" from the risk list).
  No such risk bullet existed in the spec — nothing to remove. Column widths are fixed format strings (COLUMNS-independent), which the spec already implies; no change needed.

### Summary

Applied the staff-review fixes, each re-verified by running the live oracle on this box: corrected three factual errors (no exit-2 path; relative `--archive` dest; unknown-flag `--set` truncation), documented the `dirname(__file__)` vendoring landmine and mandated `--workflow-dir`, and made `Dir`/`Env` explicit on the `Runner` seam so the decoupling/pinning test can reproduce env-dependent output (sd-b32 ids, team probe, discovery). Folded process-group cancellation, locale+interpreter pinning, and the microsecond-timestamp regex into the test plan. The approved approach (vendor-copy-and-exec behind the `Runner` seam) is unchanged.
