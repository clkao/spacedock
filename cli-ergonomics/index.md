---
id: xdt3cjnppc89amm5g23s86mm
title: CLI ergonomics — workflow auto-discovery and actionable errors
status: ideation
source: session 1 debrief — ergonomics
score: "0.30"
worktree:
started: 2026-05-30T21:28:40Z
---

Make `spacedock status` forgiving and discoverable. Today `--workflow-dir` is mandatory and unforgiving: omitting it falls back to the cwd and fails with a misleading "README.md has no stages block"; and pointing at a state checkout (the `state:` dir) post-migration errors with `non-numeric sequential id` instead of naming the real problem.

## Problem statement

Three observed UX failures, all confirmed against the production binary on the real no-symlink split-root layout (`docs/dev` definition dir, `docs/dev/.spacedock-state` state checkout, `sd-b32` ids). See **SPIKE evidence** for the exact reproductions.

1. **No discovery.** `--workflow-dir` is required for any non-trivial run. A bare `spacedock status` from a deep cwd inside a workflow either renders an empty table (cwd happens to be an empty dir) or fails with a misleading error — it never finds the enclosing workflow the way `git` finds `.git`.
2. **State-checkout misdiagnosis.** Pointing `--workflow-dir` at the `state:` checkout (which has no `README.md`, so id-style silently defaults to `sequential`) prints one `non-numeric sequential id` line per entity (15 lines on the real workflow, exit 1), or `README.md has no stages block` for `--boot`/`--next`. The real problem — "you pointed at the state checkout, not the definition dir" — is never named.
3. **No discoverable verbs.** Creating an entity requires the non-obvious `status --new <slug>` with a body piped on stdin; there is no `completion` integration; neither surfaces in `--help`.

## Scope and non-goals

In scope: discovery walk-up, the two actionable error classes, and two top-level verbs (`new`, `completion`). Lean / YAGNI: no config file, no caching of the discovered dir, no multi-workflow ambiguity UX (that is `--discover`/`--root`, already shipped). Out of scope and owned elsewhere — do NOT duplicate: `spacedock doctor`, `spacedock claude`, `spacedock codex` (spacedock-packaging); `spacedock dispatch` (native-dispatch-helper).

## Design

### Discovery walk-up (backs AC-1)

A new `discoverWorkflowDir(startDir) (dir string, ok bool)` in `internal/status`, mirroring the existing `commissioned-by: spacedock@` predicate already used by `discoverWorkflows` (`handlers.go:372`). It walks **up** from `startDir`, and at each ancestor checks for a `README.md` whose frontmatter (via the existing `parseFrontmatter`) has `commissioned-by` with prefix `spacedock@`. First match wins; stop at the filesystem root.

Resolution precedence in `dispatch` (`native_runner.go`), evaluated before `resolveRoots`:

1. **Explicit `--workflow-dir`** (or `PIPELINE_DIR`) → used verbatim, discovery skipped. Preserves every existing test and the FO's always-explicit invocations.
2. **No explicit dir** → `discoverWorkflowDir(req.Dir)`. On `ok`, that becomes the workflow dir. On miss → the AC-2 *no-workflow-here* error (exit 1), replacing today's empty-table / "no stages block" fallback for the no-flag path only.

Split-root interaction: when the cwd is *inside* a state checkout (e.g. `…/.spacedock-state/cli-ergonomics`), walking up finds the definition dir's README (`docs/dev`) before anything else — because the state checkout itself carries no commissioned README. SPIKE E confirms this resolves to `docs/dev`, the correct definition dir, which `resolveRoots` then re-splits to the state dir. So discovery from inside a state checkout is self-correcting and needs no special case.

### State-checkout detection (backs AC-2, the riskiest mechanism — SPIKED)

A new `stateCheckoutParent(pointedDir) (defDir string, ok bool)` in `internal/status`. Given the dir `--workflow-dir` points at, walk **up**; at each ancestor `A`, read `A/README.md` frontmatter; if it has a non-empty `state:` field whose cleaned, non-escaping value resolves (via the same `realpathOf` already in the codebase) to the same realpath as `pointedDir`, then `A` is the definition dir and `ok` is true. This reuses the exact `state:` validation rules already in `resolveRoots` (reject absolute / `..`-escaping).

This predicate fires only when the pointed-at dir has **no own commissioned README** (else it is a normal workflow dir and proceeds unchanged). Gating order in `dispatch`, after `resolveRoots` succeeds but before the read/validate/boot handlers run:
- if `roots.definitionDir` has a `README.md` with `commissioned-by: spacedock@` → normal path (unchanged).
- else if `stateCheckoutParent(roots.definitionDir)` is `ok` → AC-2 *state-checkout* error (exit 1).
- else → fall through to existing behavior (an arbitrary non-workflow explicit dir still yields today's empty table / "no stages block"; **not** reclassified, to preserve `TestSplitRoot*`-adjacent and empty-dir behavior under explicit `--workflow-dir`).

SPIKE D confirms detection works against the real layout: from `docs/dev/.spacedock-state`, the parent `docs/dev/README.md` declares `state: .spacedock-state` and `realpath(docs/dev/.spacedock-state) == realpath(pointed-at)`. The state-checkout error is grounded, not assumed.

### Exact error strings + exit codes (backs AC-2)

Emitted via the existing `errExit` shape (`Error: <msg>\n` to stderr, exit 1):

- **no-workflow-here** (no discoverable workflow, no `--workflow-dir`):
  `Error: no Spacedock workflow here — pass --workflow-dir or run inside a workflow`
- **state-checkout-pointed-at** (`--workflow-dir` at a `state:` checkout):
  `Error: this is a state checkout; point --workflow-dir at the definition dir (the one whose README declares state:): <defDir>`
  where `<defDir>` is the discovered definition dir, so the message names the exact fix path.

Both exit 1 (the native runner's sole error code; usage errors are 1, never 2 — see `native_runner.go` `errExit`).

### Discoverable verbs (backs AC-3)

Routed in `internal/cli/cli.go`'s `run` switch, alongside `status`:

- **`spacedock new [--folder] <slug>`** — forwards to the runner as `status --new [--folder] <slug>`, threading stdin/env/cwd through unchanged. It is a pure alias: the body is still read from stdin and the existing `runNew` atomic-create path (mint id + temp-rename) is reused verbatim. With discovery, `spacedock new` run inside a workflow needs no `--workflow-dir`.
- **`spacedock completion <shell>`** — emits a static completion script for `bash`|`zsh` to stdout (exit 0); an unknown/missing shell prints `Error: completion requires a shell: bash or zsh` to stderr (exit 2, matching the CLI-layer usage-error code used by unknown-command). Stdlib-only: the script is a Go string constant per shell listing the top-level verbs and the common `status` flags. No dynamic completion of slugs (YAGNI).
- **`--help`** — `printUsage` gains `new` and `completion` lines.

## Acceptance criteria

Each AC is an end-state property of the finished change, with a behavioral oracle: run the binary (or the in-process runner), assert stdout/stderr/exit/discovered-path. No greps as proof.

**AC-1 — Run with no `--workflow-dir` from anywhere inside a commissioned workflow renders that workflow; run with no `--workflow-dir` and no enclosing workflow yields the AC-2 no-workflow error.**
Verified by: a fixture tree (built like `buildSplitRoot`) whose cwd is several levels below the workflow README; `runNative(dir=deepCwd)` with no `--workflow-dir` exits 0 and the rendered table lists that workflow's entities (assert specific slugs present, stage-ordered). A second case with `dir` set to an empty, non-enclosed tempdir and no `--workflow-dir` exits 1 with the exact no-workflow stderr string. Oracle: walk-up fixture test in `internal/status`, driven through `runNative`.

**AC-2 — The two failure classes produce their named, fix-oriented errors instead of downstream id/stage symptoms.**
Verified by two golden assertions on stderr + exit:
- no-workflow-here: `runNative` with no `--workflow-dir` and a non-enclosed cwd → stderr equals the exact no-workflow string, exit 1.
- state-checkout-pointed-at: `buildSplitRoot` materializes `<def>/README.md` (declaring `state:`) + `<def>/.spacedock-state` with sd-b32 entities and **no** state README; `runNative --workflow-dir <def>/.spacedock-state` → stderr equals the exact state-checkout string ending in the def dir, exit 1, and stderr does **not** contain `non-numeric sequential id`. A regression guard asserts the pre-change symptom is gone. Oracle: golden stderr+exit tests in `internal/status`.

**AC-3 — `spacedock new` and `spacedock completion` exist as top-level verbs and appear in `--help`.**
Verified by:
- `cli.Run(["new","minted-task"])` with a fenced body on stdin, run inside a discovered workflow, mints an entity and exits 0 (assert the minted-id narration on stdout; assert the entity file now exists and validates) — proving the alias reaches `runNew`.
- `cli.Run(["completion","bash"])` exits 0 and stdout contains the `status`, `new`, and `completion` verbs; `cli.Run(["completion","zsh"])` exits 0; `cli.Run(["completion"])` and `cli.Run(["completion","fish"])` exit 2 with the named usage error on stderr.
- `cli.Run(["--help"])` stdout contains both `new` and `completion` usage lines.
Oracle: `internal/cli` smoke tests through `cli.Run` (the real router, real stdin), plus an `internal/status` `--new`-via-alias create test reusing the `native_new_test` body fixture.

## Test plan

All proof is exercise-and-observe; no mocks, no greps-as-proof. Stdlib-only Go; cost is low (fixture + in-process runner, no live workflow).

| Test | Layer | Oracle | Cost |
|------|-------|--------|------|
| Walk-up discovery from deep cwd | `internal/status` | `runNative(dir=deepCwd)` no `--workflow-dir`; assert exit 0 + slugs rendered, stage-ordered | low |
| No-workflow-here error | `internal/status` | `runNative(dir=emptyTmp)` no flag; assert exact stderr + exit 1 | low |
| State-checkout error (real-shape fixture) | `internal/status` | `buildSplitRoot` no state README + sd-b32 entities; `runNative --workflow-dir <state>`; assert exact stderr + exit 1 + no `non-numeric sequential id` | low |
| Explicit `--workflow-dir` precedence preserved | `internal/status` | existing `TestSplitRoot*` + a case asserting an explicit dir skips discovery | low |
| `spacedock new` alias create | `internal/cli` + `internal/status` | `cli.Run(["new",slug])` with stdin body; assert minted-id stdout + file exists + `--validate` clean | low |
| `spacedock completion <shell>` | `internal/cli` | `cli.Run(["completion","bash"\|"zsh"])` exit 0 + verbs in stdout; bad/missing shell exit 2 | low |
| `--help` lists new verbs | `internal/cli` | `cli.Run(["--help"])`; assert `new` + `completion` present | low |

Test gates: `go test ./...` (and `-race` is available but not required — no new concurrency).

## Staff review

**Recommended.** This change adds two new on-disk-shape detectors (the discovery walk-up and the state-checkout-parent predicate) that re-interpret `--workflow-dir` resolution — the same risk class the README's staff-review note calls out (split-root behavior). The riskiest mechanism (state-checkout detection during walk-up) has been SPIKED against the real layout below, so review should focus on: (a) the precedence ordering in `dispatch` not regressing any existing explicit-`--workflow-dir` test, (b) whether the no-workflow-here error should also fire for an explicit empty dir (this design deliberately does not, to preserve current behavior — flag if reviewers disagree), and (c) the completion script's exit-code choice (2 at the CLI layer vs the runner's 1).

## SPIKE evidence

Run against `/tmp/spacedock-spike` (built from this tree) on the real `docs/dev` split-root layout:

- **State checkout, table/`--validate`** (`--workflow-dir docs/dev/.spacedock-state`): 15× `Error: non-numeric sequential id: workflow=…/.spacedock-state scope=… slug=… id=<sd-b32> path=…`, exit 1. Confirms the misdiagnosis symptom.
- **State checkout, `--boot`/`--next`**: `Error: README.md has no stages block. --boot requires stage metadata.`, exit 1. Misleading — the README exists at the parent.
- **State-checkout detector (SPIKE D)**: walking up from the state checkout, `docs/dev/README.md` declares `state: .spacedock-state` and `realpath(docs/dev/.spacedock-state) == realpath(pointed-at)` → detection succeeds. Grounds the AC-2 state-checkout error.
- **Walk-up discovery (SPIKE E)**: from `docs/dev/.spacedock-state/cli-ergonomics` (3 levels deep, inside the state checkout), the nearest ancestor README with `commissioned-by: spacedock@` is `docs/dev` — the definition dir. Grounds AC-1.
- **Behavior to preserve (SPIKE B/C)**: an explicit `--workflow-dir` at an empty non-workflow dir today exits 0 with an empty table (read) / exit 1 "no stages block" (`--boot`). The design fires the new errors only on the no-flag discovery path and the detected state-checkout path, leaving the arbitrary-explicit-dir behavior unchanged.

## Notes

From the session-1 debrief ergonomics list. Auto-discovery is the biggest single ergonomic win; actionable errors are pure UX. `spacedock doctor` lives in spacedock-packaging; `spacedock dispatch` in native-dispatch-helper; `spacedock claude`/`codex` in spacedock-packaging — not duplicated here. Lower-priority nice-to-have (score 0.30).

## Stage Report: ideation

- DONE: Design the auto-discovery walk-up (AC-1) with precedence and split-root interaction
  `discoverWorkflowDir(startDir)` walks up to the nearest `commissioned-by: spacedock@` README (reuses `handlers.go:372` predicate); precedence explicit `--workflow-dir`/`PIPELINE_DIR` > discovered > no-workflow error; split-root self-corrects (SPIKE E: deep state-checkout cwd resolves to `docs/dev`).
- DONE: Design actionable errors (AC-2) with exact stderr + exit codes and discoverable verbs (AC-3)
  Two `errExit` strings pinned (no-workflow-here, state-checkout-pointed-at-with-defDir), both exit 1; AC-3 verbs `new` (alias to `status --new`) + `completion <shell>` (static stdlib script) + `--help`. AC-1..AC-3 rewritten as end-state properties, each naming a behavioral oracle (runNative/cli.Run, not greps).
- DONE: State whether ideation warrants staff review and SPIKE the unverified mechanism
  Staff review recommended (re-interprets `--workflow-dir` resolution — split-root risk class). State-checkout-vs-definition-dir detection SPIKED against the real no-symlink split-root layout: SPIKE D confirms `realpath(state) == realpath(pointed-at)` via parent README `state:` field, so the AC-2 state-checkout error is grounded.

### Summary

Fleshed-out design for forgiving/discoverable `spacedock status` plus two top-level verbs. The two riskiest mechanisms — the discovery walk-up and the state-checkout detector — were SPIKED against the production binary on the real `docs/dev` split-root layout (no state README, sd-b32 ids), confirming both the misdiagnosis symptom and that detection works. Key decision: the new errors fire only on the no-flag discovery path and the detected state-checkout path, leaving arbitrary explicit-`--workflow-dir` behavior unchanged to preserve existing tests. Staff review recommended; flagged the empty-explicit-dir and completion-exit-code choices for reviewers.
