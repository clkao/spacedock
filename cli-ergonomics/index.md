---
id: xdt3cjnppc89amm5g23s86mm
title: CLI ergonomics — workflow auto-discovery and actionable errors
status: validation
source: session 1 debrief — ergonomics
score: "0.30"
worktree: .worktrees/spacedock-ensign-cli-ergonomics
started: 2026-05-30T21:28:40Z
mod-block: merge:pr-merge
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

## Stage Report: implementation

- DONE: Discovery walk-up + state-checkout detector land with the design's exact error strings and exit codes; explicit --workflow-dir / PIPELINE_DIR precedence preserved (existing TestSplitRoot* and empty-explicit-dir behavior stay green)
  `discoverWorkflowDir`/`stateCheckoutParent` in `internal/status/discover_walkup.go`; wired in `native_runner.go` dispatch (discovery before resolveRoots, state-checkout gate after). Exact strings pinned in `discovery_dispatch_test.go`; `TestExplicitWorkflowDirSkipsDiscovery` + `TestExplicitEmptyDirUnchanged` + all 263 status tests green. Commit 4dd31f1e.
- DONE: AC-2 regression fixture reproduces the real symptom: sd-b32 entities in a no-README state dir, asserted under --validate, with a guard that the 'non-numeric sequential id' line is gone (M-9 + M-4)
  `TestStateCheckoutPointedAtError`: sd-b32 entities, lstat-guarded no state README (M-9), run with `--workflow-dir <state> --validate` (M-4 pins command context), asserts named error ending in def dir + regression guard that `non-numeric sequential id` is absent. Confirmed against the real `docs/dev/.spacedock-state` layout via the built binary.
- DONE: new + completion verbs and --help lines exist; discovery walk-up is innermost-wins with a nested-workflow test (M-1); completion bad-shell exits 2
  `new`/`completion` cases in `cli.go` `run` switch; `new` aliases `status --new` (reuses runNew), `completion bash|zsh` emits static script (exit 0), bad/missing shell exit 2 (FO decision on M-5). `--help` lists both. M-1: `discoverWorkflowDir` first-match-wins documented as innermost-wins in code comment + `TestDiscoverWorkflowDirInnermostWins`. Tests in `verbs_test.go`.

### Summary

Implemented per the Design section with TDD (failing tests first, then minimal code). Two detectors (`discoverWorkflowDir`, `stateCheckoutParent`) reuse the existing `commissioned-by`/`state:` frontmatter rules; dispatch gains a discovery step (no-flag path only) and a state-checkout gate (detected case only), leaving every explicit-dir path unchanged. CLI gains `new` (alias) and `completion` (static script) verbs plus `--help` lines. All behavioral oracles are runNative/cli.Run exit/stderr/stdout — no greps-as-proof. `go vet` clean; 23 targeted + full suite green except the pre-existing env-dependent `TestCodexResolveManifestAgainstInstalledHost` (fails identically on the clean base branch — unrelated codex-host failure). Code commit 4dd31f1e on `spacedock-ensign/cli-ergonomics`.

## Stage Report: validation

- DONE: INDEPENDENTLY reproduce the two contract-critical behaviors with the built branch binary: no-flag discovery walk-up renders the enclosing workflow AND the state-checkout error names the definition dir, with the design's exact stderr strings + exit 1; and explicit --workflow-dir/PIPELINE_DIR precedence preserved
  Built `/tmp/spacedock-cli-ergo-bin` from `./cmd/spacedock`. No-flag run from 3-levels-deep cwd inside a real `.spacedock-state` checkout rendered the enclosing workflow (exit 0, both sd-b32 entities listed); no-flag from a non-enclosed empty dir printed the exact `no Spacedock workflow here …` string (exit 1). `--workflow-dir` AT the state checkout printed the exact `this is a state checkout; … : /tmp/cli-ergo-val/def` string (exit 1) under read/`--validate`/`--boot`. Explicit `--workflow-dir` and `PIPELINE_DIR` from an unrelated cwd both rendered the explicit workflow (discovery skipped); empty explicit dir fell through to an empty table (exit 0, no error). All 5 `TestSplitRoot*` + 10 new discovery/precedence tests green.
- DONE: Verify the AC-2 regression actually guards the real symptom: M-9 fixture uses sd-b32 entities in a no-README state dir, the assertion runs under --validate (M-4), and the test would FAIL if the 'non-numeric sequential id' symptom returned
  Inspected `TestStateCheckoutPointedAtError`: README declares `id-style: sd-b32`, entities carry sd-b32-shaped ids, lstat guard (lines 85-87) asserts no state README (M-9), run executes under `--validate` (M-4). Flip-test: disabled the state-checkout gate in `native_runner.go`; the test FAILED with stderr showing the exact `non-numeric sequential id` symptom, and the rebuilt binary reproduced the same symptom against the real on-disk fixture (exit 1). Gate restored, test green, working tree clean. Guard is load-bearing.
- DONE: Reproduce new/completion verbs + --help against the real binary; innermost-wins walk-up (M-1) and completion bad-shell exit 2 (M-5) hold; go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost
  `completion bash|zsh` emitted scripts containing `status`/`new`/`completion` (exit 0); `completion` (missing) and `completion fish` (unknown) both exit 2 with the named usage error (M-5). `--help` lists both `spacedock new` and `spacedock completion` lines. `new minted-task` from a deep cwd with no `--workflow-dir` auto-discovered the workflow, minted the entity (exit 0), file exists, `--validate` returned `VALID`. M-1: nested commissioned workflows — deep cwd inside the inner one rendered only `inner-task` (innermost-wins). `go test ./...` = 529 passed, 1 failed; the sole failure is `TestCodexResolveManifestAgainstInstalledHost` (local codex-host "failed to load configuration"; test source unchanged since e3868282, predates this branch). `go vet ./...` clean.

### Summary

PASSED. Independently rebuilt the branch binary and reproduced every AC against it rather than trusting the implementation report: AC-1 discovery walk-up + no-workflow error, AC-2 named state-checkout error (exact strings, exit 1, symptom gone), AC-3 new/completion/--help. The riskiest surface — the `--workflow-dir` resolution re-interpretation — holds: explicit `--workflow-dir`/`PIPELINE_DIR` skip discovery, empty-explicit-dir is unchanged, and a flip-test proved the AC-2 regression guard would fail if the `non-numeric sequential id` misdiagnosis returned. Full suite green except the pre-existing, env-gated codex-host test that is unrelated to this change.

## Stage Report: implementation (cycle 2)

- DONE: [A-1 BLOCKER] discovery gate regressed `--root --resolve 'wf::ref'` (and unqualified `--root --resolve <ref>`) from a non-enclosed cwd: exit 0 → exit 1 "no Spacedock workflow here"
  The `--root` cross-workflow resolve path (`resolveFromRootOrExit`, including the `wf::ref` qualifier) takes its workflows from the explicit root and never consumes `pipelineDir`/`roots`, so the `pipelineDir == ""` discovery gate must not fire for it. Fix in `native_runner.go`: gate discovery on `pipelineDir == "" && rootPath == ""`, mirroring the `--discover` early-return exemption; the state-checkout gate gets the same `rootPath == ""` guard (its cwd-derived `definitionDir` must not be reinterpreted as a misdirected `--workflow-dir`). Commit 0f47ca3a.
- DONE: TDD — failing regression test FIRST, the gap that kept the suite green
  Added `internal/status/root_resolve_discovery_test.go`: `TestRootResolveSkipsDiscovery` (qualified `wfa::shared-task`) and `TestRootResolveUnqualifiedSkipsDiscovery` both from a non-enclosed cwd assert exit 0 + correct resolution; both FAILED pre-fix (exit 1, no-workflow error), pass post-fix.
- DONE: polish-item message decision pinned
  Plain `--resolve <ref>` (no --root, no --workflow-dir) DOES require a workflow, so it stays on the discovery path: a non-workflow cwd now yields the named no-workflow error — more actionable than the prior "unknown reference", both exit 1. Decision is intentional and locked by `TestPlainResolveFromNonWorkflowEmitsNoWorkflow`.

### Summary

Fixed the A-1 BLOCKER the adversarial detached-checkout audit found: the new discovery gate hard-errored the shipped `--root --resolve` cross-workflow path from a non-enclosed cwd. Wrote the missing failing regression tests first (the exact gap that left the validation suite green), then added a one-condition `rootPath == ""` guard to both the discovery gate and the state-checkout gate, mirroring the existing `--discover` exemption. Kept the plain-`--resolve`→no-workflow message change (it's the more actionable error and that path genuinely needs a workflow), pinned by a test. `gofmt`/`go vet` clean; `go test ./...` = 532 passed, 1 failed — the sole failure is the pre-existing env-gated `TestCodexResolveManifestAgainstInstalledHost` (fails identically on the clean base). Code commit 0f47ca3a on `spacedock-ensign/cli-ergonomics`.

## Stage Report: validation (cycle 2)

- DONE: INDEPENDENTLY reproduce the A-1 fix with the built branch binary — qualified + unqualified `--root --resolve` from a NON-enclosed cwd both exit 0 with correct resolution; TestRootResolveSkipsDiscovery + TestRootResolveUnqualifiedSkipsDiscovery green
  Built `/tmp/spacedock-cli-ergo-reval` from `./cmd/spacedock` at HEAD 0f47ca3a. Real fixture (commissioned `wfa` README + `shared-task`, git-init'd) under a fresh root; cwd a separate non-enclosed empty tempdir. `--root <root> --resolve 'wfa::shared-task'` and unqualified `--resolve 'shared-task'` BOTH exit 0 resolving `slug=shared-task path=…/wfa/shared-task.md`. Flip-test: archived parent commit 4dd31f1e (guard absent, confirmed by grep) into a temp tree, built `/tmp/spacedock-cli-ergo-prefix`; SAME two invocations regressed to exit 1 `no Spacedock workflow here` — the fix is load-bearing. Both named regression tests PASS (verbose raw `go test -run`).
- DONE: ADVERSARIALLY probe the `rootPath == ""` guard's scoping both ways — no over-correction, no under-correction, no --root/discovery case mis-scoped
  Over-correction (guard didn't disable wanted behavior): (a) no-flag discovery from a 3-deep cwd INSIDE a real `.spacedock-state` checkout self-corrects to the def dir and renders both sd-b32 entities (exit 0); (b) `--workflow-dir` AT the state checkout STILL fires the exact state-checkout error ending in the def dir (exit 1) under read/`--validate`/`--boot`, and `non-numeric sequential id` is absent. Under-correction (no `--root` case hard-errors on cwd): (c) `--root` without `--resolve` from a non-workflow cwd reaches the clean `--root is only supported with --discover or --resolve` (exit 1) — `resolveRoots("",cwd)` does not crash; (c2) `--root --discover` exit 0; (d) `--root --resolve` for a missing ref reaches the resolver → `unknown reference` (exit 1), NOT the discovery gate; (e) bad root → resolver's `ambiguous or unknown` qualifier error. Combined-flag scoping: (f) `--root` + `--workflow-dir`-at-state-checkout + `--resolve` → rootPath wins, resolves (exit 0), state-checkout gate correctly exempt (the `--workflow-dir` is unconsumed on the `--root` path); (g) PIPELINE_DIR (rootPath=="") at def dir skips discovery and renders, at state checkout STILL fires the state-checkout error. Guard exempts ONLY the `--root` path, which genuinely never consumes the cwd-derived workflow — precisely scoped, no mis-scope found.
- DONE: go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; gofmt/vet clean; plain-`--resolve`→no-workflow decision holds; cli.Run signature + exit-code contract unchanged
  `gofmt -l .` empty; `go vet ./...` exit 0; `go test ./...` = sole failure `TestCodexResolveManifestAgainstInstalledHost` — env, not code: `codex` host returns "failed to load configuration" because `/Users/clkao/.codex/config.toml` is "Operation not permitted (os error 1)"; branch does not touch `codex_resolve_test.go` (only `cli.go`+`verbs_test.go` in `internal/cli`). `TestPlainResolveFromNonWorkflowEmitsNoWorkflow` green (plain `--resolve` from non-workflow → exact no-workflow error, exit 1). Verbs reconfirmed on the binary: `completion bash|zsh` exit 0, `completion`/`completion fish` exit 2, `--help` lists both `spacedock new` and `spacedock completion`. `cli.Run(args []string, stdout, stderr io.Writer) int` identical on branch and origin/next — signature + exit domain unchanged.

### Summary

PASSED (re-validation). Rebuilt the branch binary at 0f47ca3a and independently reproduced the A-1 fix: qualified and unqualified `--root --resolve` from a non-enclosed cwd both exit 0 with correct resolution, and a flip-test against the pre-fix parent commit proved both regress to exit 1 — the `rootPath == ""` guard is load-bearing. Adversarially probed the guard both directions across nine cases (discovery self-correction, state-checkout error under read/validate/boot, `--root` without/with `--resolve`, missing-ref routing, bad root, combined `--root`+`--workflow-dir`, PIPELINE_DIR at def-dir and at state-checkout): no over-correction, no under-correction, no mis-scope — the exemption fires only for `--root`, which never consumes the cwd workflow. Full suite green except the pre-existing env-gated codex-host test (untouched by this branch; fails on permission to read `~/.codex/config.toml`). gofmt/vet clean; `cli.Run` signature and exit-code contract unchanged.

## Stage Report: implementation (cycle 3)

- DONE: Rebase the worktree branch onto origin/next and resolve the internal/cli/cli.go conflict by re-expressing the `new` and `completion` verbs as cobra subcommands matching the just-merged cobra structure on next (z0 cli-cobra-redesign) — NOT the old hand-rolled run-switch cases; update verbs_test.go and the --help listing to the cobra structure
  Rebased onto origin/next (cobra migration #241). Conflict was ONLY in cli.go as predicted. Resolved by taking the cobra tree wholesale and adding `newNewCommand`/`newCompletionCommand` as `GroupID: "workflow"` cobra subcommands (DisableFlagParsing, RunE→exitCodeError), registered in `newRootCommand` alongside status/dispatch. `new` aliases `status --new` via runStatus (reuses runNew); `completion` calls the kept `runCompletion`+`bashCompletion`/`zshCompletion` constants (verb list updated init→install to match next's command set). Dropped the obsolete hand-rolled `printUsage`. Added `new`/`completion` rows to the grouped `topLevelHelp` Workflow block in help.go. Updated `TestHelpListsNewVerbs` to assert the cobra grouped-help one-liners instead of the old `spacedock new` usage lines. Code commits 9b812967 + f24ca53b on spacedock-ensign/cli-ergonomics.
- DONE: Preserve xd's already-validated core unchanged: the discovery walk-up + state-checkout detector + the `rootPath==""` guard in internal/status (discover_walkup.go, native_runner.go, discovery_dispatch_test.go) apply cleanly on rebase — do NOT alter their behavior; only the cli.go verb wiring + verbs_test.go change for cobra
  All three status-layer files applied clean on rebase and are byte-identical to the validated commit 0f47ca3a (`git diff 0f47ca3a:<f> HEAD:<f>` empty for discover_walkup.go, discovery_dispatch_test.go, native_runner.go). The A-1 `rootPath==""` guard (second commit) rebased clean. Only internal/cli (cli.go, help.go, verbs_test.go) changed for the cobra re-expression.
- DONE: go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; verbs behave identically to the validated pre-rebase behavior; gofmt/vet clean; cli.Run signature + exit-code contract unchanged
  `go test ./...` = 554 passed, 1 failed (sole failure `TestCodexResolveManifestAgainstInstalledHost` — codex host "failed to load configuration", env-only, branch does not touch codex_resolve_test.go). `gofmt -l .` empty; `go vet ./...` clean. Built `/tmp/spacedock-cli-ergo-c3` and reproduced every AC-3 behavior on the real binary: `new` from a 3-deep cwd with no --workflow-dir auto-discovered + minted `id=minted-task` + `--validate`→VALID; `completion bash|zsh` exit 0 with status/new/completion in the script; `completion`/`completion fish` exit 2 with the named usage error; `--version` carries `(contract 1)`; unknown-command exit 2; state-checkout error exact string + exit 1 + `non-numeric sequential id` absent; `--help` lists `new` and `completion` under Workflow. `cli.Run(args []string, stdout, stderr io.Writer) int` identical to origin/next.

### Summary

PASSED (cycle-3 integration rework). Rebased the validated branch onto the now-cobra'd origin/next; the conflict was confined to internal/cli/cli.go exactly as the review predicted. Re-expressed `new` and `completion` as cobra Workflow-group subcommands matching z0's structure (DisableFlagParsing + exitCodeError contract), reused the unchanged `runStatus`/`runCompletion` cores, moved the help listing into the grouped `topLevelHelp`, and retargeted `TestHelpListsNewVerbs` to the cobra help shape. The validated discovery core (discover_walkup.go, native_runner.go incl. the A-1 `rootPath==""` guard, discovery_dispatch_test.go) is byte-identical to the pre-rebase validated commit — untouched. Full suite green except the pre-existing env-gated codex-host test; gofmt/vet clean; `cli.Run` signature and exit-code contract unchanged; every AC-3 verb behavior reconfirmed on a freshly built binary.

## Stage Report: validation (cycle 3)

- DONE: Confirm the validated discovery CORE is genuinely unchanged by the rebase: `git diff 0f47ca3a:<f> HEAD:<f>` EMPTY for discover_walkup.go, discovery_dispatch_test.go, native_runner.go; then spot-reproduce on the rebased binary that the discovery walk-up, the state-checkout error (exact string, exit 1, no 'non-numeric sequential id'), and the A-1 --root --resolve guard still behave
  All three `git diff 0f47ca3a:internal/status/<f> HEAD:...` are EMPTY (byte-identical). Built `/tmp/spacedock-cli-ergo-c3reval` from HEAD f24ca53b. On the real `docs/dev` split-root layout: no-flag run from a 3-deep cwd inside `.spacedock-state` rendered the enclosing workflow (exit 0); `--workflow-dir` AT the state checkout printed the exact `this is a state checkout; … : …/docs/dev` string (exit 1) with `non-numeric sequential id` count 0; no-flag from a non-enclosed empty dir printed the exact `no Spacedock workflow here …` string (exit 1). A-1 guard: real git-init'd `wfa` fixture under a fresh root, qualified `--root <r> --resolve 'wfa::1'` AND unqualified `--resolve '1'` from a non-enclosed cwd BOTH exit 0 with correct `slug=1 path=…/wfa/1.md` resolution.
- DONE: Validate the cobra verb re-expression AND its coexistence with z0's cobra commands: new auto-discovers+mints; completion bash|zsh exit 0 with status/new/completion, completion/completion fish exit 2; --help lists new+completion under Workflow. Coexistence: AC-5 verbatim forwarding holds, no name collision, DisableFlagParsing scoped to these verbs only
  `new reval-probe-task` from a 3-deep cwd with no --workflow-dir auto-discovered the real workflow + minted `id=861wdbwhpeh9m4y6cynk3m90` (file existed, `--validate`→VALID; probe was untracked, removed after). `completion bash|zsh` exit 0 with status/new/completion present; `completion` (missing) + `completion fish` (unknown) exit 2 with `Error: completion requires a shell: bash or zsh`. `--help` Workflow block lists `new` and `completion`. Coexistence: `status --some-bogus-flag` and `status --where status=backlog` (with `--workflow-dir`) BOTH reached the runner and rendered the table (exit 0) — NOT cobra-rejected (AC-5 verbatim forwarding intact); each of the 8 command names appears exactly once in --help (no collision); root has NO DisableFlagParsing so `--bogus-root-flag` is cobra-rejected (exit 2) while all 8 subcommands carry DisableFlagParsing (z0 baseline for status/dispatch/claude/codex/install/doctor + the two new verbs — consistent, no leak); cobra's default `completion` command is replaced via DisableDefaultCmd (our usage error, not cobra's menu).
- DONE: go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; gofmt/vet clean; cli.Run(args, stdout, stderr) int signature + exit-code contract (unknown=2, version=0) identical to origin/next
  `go test ./...` = 554 passed, 1 failed — sole failure `TestCodexResolveManifestAgainstInstalledHost` (codex host "failed to load configuration"; `codex_resolve_test.go` is UNTOUCHED by this branch vs origin/next — env-only). `gofmt -l .` empty; `go vet ./...` clean. `func Run(args []string, stdout io.Writer, stderr io.Writer) int` identical on HEAD and origin/next; `exitCodeFor` and `printVersion` byte-identical to origin/next (diff empty); on the binary unknown-command exit=2 and `--version` exit=0 with the `(contract 1)` token. Worktree clean at f24ca53b.

### Summary

PASSED (cycle-3 re-validation). The validated discovery core (discover_walkup.go, native_runner.go incl. the A-1 `rootPath==""` guard, discovery_dispatch_test.go) is byte-identical to the pre-rebase validated commit 0f47ca3a — the rebase preserved it, and I reproduced the discovery walk-up, the exact state-checkout error (exit 1, `non-numeric sequential id` absent), and the qualified+unqualified `--root --resolve` guard on a freshly built rebased binary. The only NEW surface — the cobra `new`/`completion` Workflow-group verbs — is correct (auto-discover+mint→VALID, completion exit codes, --help listing) and coexists cleanly with z0's cobra tree: AC-5 verbatim forwarding still reaches the runner for status/dispatch, no command-name collision, and DisableFlagParsing is consistently scoped (root parses flags, all 8 subcommands disable — no leak). Full suite green except the pre-existing env-gated codex-host test (untouched source); gofmt/vet clean; `cli.Run` signature, exit-code contract, and version token unchanged from origin/next.
