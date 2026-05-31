---
id: mvmpr2vx79pj8w8j65g4vqyz
title: Retire the prose-grep integration assertions + repoint dispatch_test.go at native dispatch.Run
status: validation
source: coverage matrix rows 9/10/12/13/14 (prose-grep antipattern) + handoff-confirmed test debt (dispatch_test.go drives retired Python claude-team build)
started: 2026-05-31T18:12:42Z
completed:
verdict:
score: "0.28"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-harden-integration-prose-grep-tests
issue:
---

Two pieces of `skills/integration/` test debt the coverage matrix (archived entity
`behavior-test-skeleton-and-matrix`, id `8033qbqdrh4zba10w0d34m4j`) surfaced. Both are
"make an existing integration test genuinely behavioral instead of prose-grep / retired-helper."

**(a) Prose-grep antipattern (matrix rows 9/10/12/13/14).** Several assertions in
`skills/integration/skill_text_test.go` (and `contract_status_path_test.go`,
`contract_gate_test.go`) `os.ReadFile` a `.md` instruction file and `strings.Contains` the
contract TEXT for a clause. They pass even if the clause is **behaviorally dead** — the
contract says "use `--json`" but nothing proves a run consumes `--json`. Named in the matrix:
- row 9-prose: `contract_gate_test.go::TestStartupStepZeroIsContractGate` (asserts step-1 prose Contains, ordering via `strings.Index`)
- row 10-prose: `skill_text_test.go::TestLauncherStatusInvocations`, `contract_status_path_test.go::TestVendoredSkillsCallSpacedockStatus`
- row 11-prose: `skill_text_test.go::TestConcurrencySafeCommitClause`
- row 12: `skill_text_test.go::TestEventLoopReadsUseJSON`
- row 13-prose: `skill_text_test.go::TestDispatchBlockUsesNativeBuild`
- row 14: `skill_text_test.go::TestSplitRootContractClause`, `TestNoPRMergeOrModBehaviorIntroduced`, `contract_gate_test.go::TestStartupEmbeddedRangeBracketsContractVersion`

Most already have a behavioral counterpart (matrix `v1-implements` column). The matrix's
hx-reconciliation gives the decision rule: **port to behavioral where a seam exists; keep
genuine structural invariants (hx-AC-2 kind: an oracle-backed assertion over real on-disk
structure) with that label.** Do NOT blindly delete — distinguish prose-grep (no oracle, greps
prose) from legitimate-structural.

**(b) Retired-helper dependency (handoff-confirmed).** `skills/integration/dispatch_test.go`
still drives the **retired Python** `claude-team build` via `exec.Command("python3",
vendoredClaudeTeam(t), "build", ...)` (line ~45). The native seam is the in-process
`dispatch.Run`. Repoint the test at native `dispatch.Run` so it stops exercising the dead Python path.

## Acceptance criteria (provisional — ideation hardens)

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 — No `skills/integration` test asserts contract behavior via bare prose-grep where a
behavioral seam exists.** Verified by: each matrix-named prose-grep assertion is either
retargeted at its behavioral seam (binary/git/`dispatch.Run`) with the bare-Contains retired,
or explicitly retained+labeled as a genuine structural invariant with its oracle named; a
reproducible enumeration (grep/test-list) shows none of the ported ones remain as bare
prose-grep.

**AC-2 — `dispatch_test.go` exercises native `dispatch.Run`, not the retired Python helper.**
Verified by: `grep -n 'python3\|claude-team' skills/integration/dispatch_test.go` returns no
build-driving invocation; the test drives in-process `dispatch.Run` and asserts the same
observable dispatch outputs.

**AC-3 — The full suite stays green.** Verified by: `go test ./...` EXIT=0 (modulo the
known environment-only `TestCodexResolveManifestAgainstInstalledHost` failure), `gofmt -l`
clean, `go vet` clean — with real captured exit codes.

## Out of scope
- Rows 16/17 (team fail-early live name, codex packaged-agent) and the live-e2e CI net —
  deferred to a live-runtime harness.
- Row 15 (gate/feedback loop) — its own entity (`gate-feedback-loop-behavior-coverage`).

## Notes
Lives entirely in `skills/integration/`. Disjoint from the row-15 entity (which extends
`internal/ensigncycle`) — safe to run in a parallel worktree.

---

# Ideation design (hardened)

## Disposition rule (from the archived matrix's hx reconciliation)

The archived `behavior-test-skeleton-and-matrix` entity (id `8033qbqdrh4zba10w0d34m4j`)
reconciles hx P1-vs-P2 and gives the decision rule applied below:
- **hx-AC-2 → KEEP+label.** A static assertion that, even while it `os.ReadFile`s a `.md`,
  asserts a genuine *structural* fact the system mechanically depends on, validated against an
  **oracle** (a compiled constant, a parser, a real build's emitted bytes). Not prose-grep.
- **hx-AC-3 → PORT (retire bare-Contains).** A static assertion with **NO oracle** that only
  `strings.Contains` instruction prose for a clause. Passes even if the clause is behaviorally
  dead. This is the P2 antipattern. Port to its behavioral seam where one exists, retire the
  bare-Contains; only keep a residue if it is genuinely hx-AC-2 structural.

The load-bearing test: **does the assertion name an oracle?** If the only thing it compares
against is "the prose says X", it is hx-AC-3 (PORT). If it compares against a build's emitted
bytes / a compiled constant / a parser verdict / a real binary's observable behavior, it is
hx-AC-2 (KEEP).

## Per-assertion DISPOSITION table (rows 9/10/11/12/13/14)

Each row: the named test, what it actually asserts (read from `skills/integration/`), the `.md`
it greps, and the disposition with the named seam (PORT) or named oracle (KEEP).

| # | Test (file::name) | What it asserts today | Disposition | Seam (PORT) / Oracle (KEEP) |
|---|---|---|---|---|
| 9a | `contract_gate_test.go::TestStartupStepZeroIsContractGate` | FO `## Startup` step-1 prose `Contains` `spacedock --version` + `contract` + `ABORT`; ordering of gate-before-discover/boot via `strings.Index` over the prose | **PORT** (retire) | Seam: `internal/contract/gate_test.go::TestStartupGateAbortsBeforeDiscover` — drives a real `spacedock` stub `--version`, parses the `contract` token, runs `gateAndMaybeDiscover`, and OBSERVES discover is invoked 0× on out-of-range / exactly 1× on compatible. That is the gate behavior; the `.md` Contains/Index is zero-oracle prose-grep. |
| 9b | `contract_gate_test.go::TestStartupEmbeddedRangeBracketsContractVersion` | Parses the `>=N,<M` literal embedded in `## Startup` via `contract.ParseRange`, asserts it brackets `contract.CONTRACT_VERSION`, and that exactly 1 such literal exists | **KEEP+label** | Oracle: `contract.ParseRange` (parser) + `contract.CONTRACT_VERSION` (compiled constant). This closes the 4th-source-of-truth drift — the FO prose embeds its own expected range and this proves it brackets the binary's real contract version. hx-AC-2 structural invariant, NOT prose-grep. |
| 10a | `skill_text_test.go::TestLauncherStatusInvocations` | FO `.md` `Contains` 5 literal `spacedock status …` invocation strings (discover/boot/set/archive/overview), each with `{workflow_dir}`/`{slug}` placeholders | **PORT** (retire) | Seam: `skills/integration/launcher_smoke_test.go::TestLauncherListSetArchive` drives the real status binary for list/set/archive observably; `internal/status/*` covers the surface. The FO-text Contains has no oracle that a *run* consumes those flags — bare prose-grep. |
| 10b | `contract_status_path_test.go::TestVendoredSkillsCallSpacedockStatus` | FO `.md` `Contains "spacedock status"` (positive) **and** neither FO nor ensign `.md` `Contains` any of 3 plugin-private status paths (negative) | **SPLIT: positive PORT, negative KEEP+label** | Positive half (`Contains "spacedock status"`) is bare prose-grep → covered by the launcher-smoke behavioral seam, retire. Negative half (no `skills/commission/bin/status` / `{spacedock_plugin_dir}` / `commission/bin/status` ref) is a genuine hx-AC-2 **absence invariant** over the vendored on-disk surface — its oracle is "the vendored skill tree must not re-introduce the retired plugin-private path"; a real seam cannot prove an absence. KEEP the negative, label it. |
| 11 | `skill_text_test.go::TestConcurrencySafeCommitClause` | ensign+FO `.md` `Contains` 4 commit-discipline clauses (`path-scoped`, the `git -C {state_checkout} commit -m` form, `Never a bare git add -A`, `tool-managed atomic state commits`) | **PORT** (retire) | Seam: `skills/integration/concurrency_test.go::TestPathScopedCommitDoesNotSweepSibling` — runs real `git`, stages two siblings, does a path-scoped commit, and OBSERVES the sibling is NOT swept. That is the invariant the prose describes; the 4-clause Contains has no oracle that a commit is actually path-scoped. Bare prose-grep. |
| 12 | `skill_text_test.go::TestEventLoopReadsUseJSON` | FO runtime `## Event Loop` section: per-line walk asserts every `status --next` / `status --where` read line also `Contains "--json"`, both `--next` reads present, mod-block clear stays bare | **PORT** (retire) | Seam: there is NO v1 behavioral seam yet that proves the FO scheduling loop CONSUMES `--json` (the FO is a live LLM; the `--json` output shape itself is covered by `internal/status` JSON tests). Per the matrix this is a zero-oracle prose-grep with no current behavioral counterpart → retire the bare-Contains; the `--json` *output contract* is owned behaviorally by `internal/status` JSON tests, and the live "FO actually reads --json" half stays GAP (deferred live-runtime, matrix Out-of-scope). Do NOT keep as structural — it has no oracle. |
| 13 | `skill_text_test.go::TestDispatchBlockUsesNativeBuild` | FO runtime `## Dispatch Adapter` section `Contains` the `… \| spacedock dispatch build --workflow-dir {workflow_dir}` command, does NOT contain the vendored Python build path, and no fenced line `Contains "claude-team"` | **PORT** (retire) | Seam: `internal/dispatch/build_parity_test.go::TestBuildParityCrossProduct` + `cycle2_parity_test.go` — drive the real native `dispatch.Run` build and byte-compare the emitted dispatch body to the oracle. The behavior "the dispatch block runs native build" is proven by native build EXISTING and emitting parity bytes; the `.md` Contains has no oracle. Bare prose-grep. |
| 14a | `skill_text_test.go::TestSplitRootContractClause` | ensign+FO `.md` `Contains "Split-Root Worktree Contract"` + `Contains "CODE only"` | **PORT** (retire) | Seam: `skills/integration/dispatch_test.go::TestSplitRootFolderWorktreeDispatch` (repointed at native `dispatch.Run` per AC-2) — OBSERVES that a split-root worktree dispatch emits the worktree CODE working-dir while the entity-read + completion-signal point at the state path (no `.worktrees/` segment). That IS the CODE-only split-root behavior; the 2-clause Contains has no oracle. Bare prose-grep. |
| 14b | `skill_text_test.go::TestNoPRMergeOrModBehaviorIntroduced` | ensign `.md` has no `## Hook:`; FO split-root region has no `## Hook:`; no file `Contains` 3 PR-merge markers (`gh pr merge`, `git merge --no-ff`, `git merge --ff-only main`) | **KEEP+label** | Oracle: this is an hx-AC-2 **absence invariant** over the vendored on-disk surface — "the vendored skill amendments introduce no new lifecycle `## Hook:` and no PR-merge command". A real seam cannot prove an absence of behavior; the oracle is the structural scope-fence over the amendment regions (scoped via `sectionAfter`). KEEP, label as scope-guard structural invariant. |
| 14c | `contract_gate_test.go::TestStartupEmbeddedRangeBracketsContractVersion` | (same as 9b — the seed lists it under both row 9 and row 14) | **KEEP+label** | Same as 9b: oracle `contract.ParseRange` + `contract.CONTRACT_VERSION`. hx-AC-2 structural. |

### Disposition summary

- **PORT (retire bare-Contains):** 9a, 10a, 11, 12, 13, 14a, and the positive half of 10b. Each
  (except 12) has a named behavioral seam already in the tree that observes the real behavior; the
  bare `.md` Contains is redundant zero-oracle prose-grep. Row 12's `--json` *consumption* by the
  live FO has no v1 seam → its bare-Contains is retired and the residual live half stays GAP
  (matrix Out-of-scope), with the JSON output-shape contract owned by `internal/status` tests.
- **KEEP+label (genuine hx-AC-2 structural, oracle named):** 9b/14c (embedded-range brackets
  `CONTRACT_VERSION`, oracle = `contract.ParseRange` + constant), 14b (no-Hook/no-PR-merge absence
  invariant over the amendment regions), and the negative absence half of 10b (no plugin-private
  status path). Each names its oracle and is provably not bare prose-grep.

**Why NOT blindly delete.** 9b, 14b, and 10b-negative read `.md` files but each asserts a
structural fact with an oracle: a parsed range vs a compiled constant, and absence invariants the
vendored surface mechanically must not violate (a re-introduced plugin status path or a new
PR-merge hook would silently break the launcher contract, and no positive behavioral seam can
catch an absence). Deleting them would lose real coverage.

## AC-2 — `dispatch_test.go` repoint: feasibility CONFIRMED by spike

`skills/integration/dispatch_test.go::runBuild` drives the retired Python helper via
`exec.Command("python3", vendoredClaudeTeam(t), "build", "--workflow-dir", workflowDir)` (line
~45). The native seam is the in-process `internal/dispatch.Run([]string{"build","--workflow-dir",
wd}, stdin, &stdout, &stderr)` — already exercised by `build_parity_test.go::runNative`, which
byte-compares the native build's stdout JSON and dispatch body against the oracle across the full
slug/split/worktree cross-product. So native build is a proven byte-for-byte drop-in (modulo the
one fetch-prefix rewrite, which neither `dispatch_test.go` assertion touches).

**Spike (riskiest path, done FIRST — observed green before committing the design):** I added a
throwaway `spike_dispatch_native_test.go` in the `integration` package with `runBuildNative`
(same `buildResult` contract as `runBuild`, but calling `dispatch.Run` in-process) and mirrored
BOTH affected tests' assertions:
- `TestSpikeSplitRootNative` — asserts `res.Name == spacedock-ensign-skill-launcher-implementation`,
  the dispatch-file suffix, no `index` leak, the `spacedock-ensign/skill-launcher` branch line, the
  worktree working-dir present, and the entity-read + completion-signal lines pointing at the
  state-checkout path with no `.worktrees/` segment.
- `TestSpikeFlatNative` — asserts `res.Name == spacedock-ensign-vendor-script-backlog`.

Result: **2 passed**, EXIT=0. Every observable output the current Python path asserts is produced
identically by native `dispatch.Run`. Spike then removed; integration baseline re-run green (15
passed). This confirms AC-2 is mechanically feasible with no contract change.

**Repoint plan (implementation stage):** rewrite `runBuild` to drive `dispatch.Run` in-process —
import `internal/dispatch`, `t.Setenv("HOME", t.TempDir())` for the bare-mode probe hermeticity
(the Python path used `cmd.Env … HOME=t.TempDir()`), `json.Marshal` the input to stdin, call
`dispatch.Run`, parse stdout JSON into `buildResult`, read the dispatch body. Delete
`vendoredClaudeTeam` and the `exec`/`python3` dependency from this file. The two tests'
assertion bodies stay byte-identical (proven by the spike). After repoint:
`grep -n 'python3\|claude-team' skills/integration/dispatch_test.go` returns no build-driving
invocation (AC-2 verification).

## Hardened acceptance criteria (behavior-first)

**AC-1 — Each rows-9/10/11/12/13/14 prose-grep assertion is dispositioned per the table above:
PORTed (bare-Contains retired, behavior owned by the named seam) or KEPT+labeled as an
oracle-backed hx-AC-2 structural invariant.** Verified by: the seven PORT assertions
(9a, 10a, 10b-positive, 11, 12, 13, 14a) have their bare `strings.Contains` over `.md` prose
removed; a reproducible `grep`/`go test -list` over `skills/integration/` shows no PORTed test
remains as a bare-Contains-over-prose assertion; the named behavioral seam for each PORTed row
stays green (`go test ./internal/contract/ ./internal/dispatch/ ./internal/status/
./skills/integration/`). The four KEPT assertions (9b/14c, 14b, 10b-negative) each carry a
comment naming their oracle (`contract.ParseRange`+`CONTRACT_VERSION`; the amendment-region
scope-fence; the plugin-private-path absence invariant) so each is provably not bare prose-grep.

**AC-2 — `dispatch_test.go` exercises native `dispatch.Run`, not the retired Python helper.**
Verified by: `grep -n 'python3\|claude-team' skills/integration/dispatch_test.go` returns no
build-driving invocation; `runBuild` calls in-process `dispatch.Run`; both
`TestSplitRootFolderWorktreeDispatch` and `TestFlatEntitySlugUnchanged` assert the same observable
dispatch outputs (name, dispatch-file path, branch line, worktree working-dir, state-path
entity-read + completion-signal) and stay green. Feasibility already proven by the ideation spike
(2 passed in-process).

**AC-3 — The full suite stays green.** Verified by: `go test ./...` EXIT=0 (modulo the known
environment-only `TestCodexResolveManifestAgainstInstalledHost` failure), `gofmt -l` clean,
`go vet` clean — with real captured exit codes.

## Test plan

- **AC-1 dispositions:** Go unit tests at the seam abstraction (`internal/contract/gate_test.go`,
  `internal/dispatch/build_parity_test.go`, `internal/status/*`, `concurrency_test.go`,
  `launcher_smoke_test.go`) already exist and stay green — implementation only *removes* the
  redundant bare-Contains assertions, it adds no new behavioral test. The four KEPT structural
  assertions are re-labeled (comment naming the oracle), not changed in logic. Cost: low
  (deletions + comments). Reproducible enumeration: a `go test -list` / grep showing no PORTed
  test name survives as a prose-Contains.
- **AC-2 repoint:** rewrite one helper (`runBuild`) + drop the `python3`/`exec` dependency; the
  two test bodies are unchanged. Cost: low, proven feasible by the ideation spike. CLI/in-process
  test (no live workflow).
- **AC-3 gates:** `go test ./...`, `gofmt -l`, `go vet` with real captured exit codes.

## Out of scope (unchanged)
- Rows 16/17 (team fail-early live name, codex packaged-agent) and the live-e2e CI net.
- Row 15 (gate/feedback loop) — its own entity.
- Row 12's live "FO actually consumes `--json`" half — deferred to a live-runtime harness (the
  `--json` output-shape contract IS covered behaviorally by `internal/status` JSON tests).

## Stage Report: ideation

- DONE: Produce a per-assertion DISPOSITION table for every matrix-named prose-grep assertion (rows 9/10/11/12/13/14): for each, read the real test in skills/integration/ AND the .md file it greps, then classify port-to-behavioral-at-seam-X (name the seam) vs keep-as-legitimate-structural-with-oracle-Y (name the oracle), applying hx-AC-2 vs hx-AC-3
  Per-assertion table in "Per-assertion DISPOSITION table" — read all four test files (skill_text_test.go, contract_gate_test.go, contract_status_path_test.go, dispatch_test.go) and the seam tests (contract/gate_test.go, build_parity_test.go). 7 PORT (named seam each), 4 KEEP+label (named oracle each), with 10b split positive-PORT/negative-KEEP.
- DONE: Confirm the dispatch_test.go repoint is feasible: verify the native in-process dispatch.Run seam can produce the same observable outputs the current exec.Command("python3", claude-team, "build") path asserts; spike the minimal repoint and observe it green before committing the design
  Spiked `runBuildNative` (dispatch.Run in-process) + mirrored both affected tests' asserts: 2 passed EXIT=0; spike removed; integration baseline re-run 15 passed. See "AC-2 — dispatch_test.go repoint".
- DONE: Harden AC-1/AC-2/AC-3 to behavior-first with the concrete per-test disposition and the dispatch_test.go repoint plan; ensure each retained static assertion names its oracle (so it is provably not bare prose-grep)
  Hardened ACs in "Hardened acceptance criteria"; AC-1 enumerates the 7 PORT + 4 KEEP with each KEEP naming its oracle; AC-2 carries the proven-feasible repoint plan; AC-3 is the green-gate bar.

### Summary

Ideation dispositioned all six matrix-named prose-grep rows by reading each real test and the `.md` it greps, then applying the archived matrix's hx rule (oracle-backed structural = KEEP; zero-oracle prose-grep = PORT). The load-bearing distinction is whether the assertion names an oracle: seven assertions only `strings.Contains` instruction prose with no oracle and each already has a behavioral seam in the tree (gate stub, build parity, concurrency commit, launcher smoke, native build) — those PORT. Four are genuine hx-AC-2 structural invariants with named oracles — the embedded-range-brackets-`CONTRACT_VERSION` check (`contract.ParseRange` + compiled constant) and three absence invariants (no plugin-private status path, no new `## Hook:`/PR-merge) that no positive behavioral seam can catch — those KEEP+label. The AC-2 repoint was de-risked FIRST with a throwaway spike proving native `dispatch.Run` produces byte-identical observable outputs (2 passed), so the implementation is a low-risk one-helper rewrite + redundant-assertion deletions, not new test authorship.

## Stage Report: implementation

- DONE: PORT — remove the 7 redundant bare-Contains assertions per the disposition table (9a TestStartupStepZeroIsContractGate, 10a TestLauncherStatusInvocations, 10b-positive the Contains "spacedock status" half of TestVendoredSkillsCallSpacedockStatus, 11 TestConcurrencySafeCommitClause, 12 TestEventLoopReadsUseJSON, 13 TestDispatchBlockUsesNativeBuild, 14a TestSplitRootContractClause); reproducible enumeration shows none survive, every named seam stays green
  `go test -list` shows all 7 names GONE; named seams green: TestStartupGateAbortsBeforeDiscover, TestBuildParityCrossProduct, TestPathScopedCommitDoesNotSweepSibling, TestLauncherListSetArchive (commit 0f05d16). 10b-positive folded into the launcher-smoke seam; the file's `spacedock status` test renamed to the absence-only TestNoPluginPrivateStatusPathInContracts.
- DONE: KEEP+label — leave the 4 oracle-backed structural assertions in place, each carrying a comment naming its oracle
  9b/14c TestStartupEmbeddedRangeBracketsContractVersion labels oracle = contract.ParseRange + CONTRACT_VERSION; 14b TestNoPRMergeOrModBehaviorIntroduced labels oracle = amendment-region scope-fence; 10b-negative TestNoPluginPrivateStatusPathInContracts labels oracle = plugin-private-path absence invariant. All remaining `.md`-prose Contains are absence negatives inside these labeled tests, not presence prose-grep.
- DONE: Repoint dispatch_test.go runBuild to in-process internal/dispatch.Run (drop python3/exec/vendoredClaudeTeam); both TestSplitRootFolderWorktreeDispatch + TestFlatEntitySlugUnchanged stay green; grep returns no build-driving claude-team/python3; full gates green with real exit codes
  runBuild now calls dispatch.Run with t.Setenv HOME + bytes.Buffer capture; `grep -n 'python3\|claude-team' dispatch_test.go` = 0 matches; both tests PASS unchanged. gofmt -l clean, go vet ./... EXIT=0, go test ./... EXIT=1 only on pre-existing env-only TestCodexResolveManifestAgainstInstalledHost (cannot read ~/.codex/config.toml under sandbox — package internal/cli untouched).

### Summary

Repointed dispatch_test.go's runBuild at the in-process internal/dispatch.Run build (dropping the retired python3/claude-team exec path) — both split-root and flat dispatch tests assert byte-identical observable outputs and stay green, and the AC-2 grep returns no build-driving Python invocation. Retired the 7 zero-oracle bare-Contains-over-.md-prose assertions whose behavior is owned by an existing named seam (gate stub, build parity, concurrency commit, launcher smoke), confirmed each seam stays green and a `go test -list` enumeration shows no PORTed name survives. Kept and labeled the 4 genuine hx-AC-2 structural invariants (embedded-range-brackets-CONTRACT_VERSION and three absence invariants no positive seam can catch), each comment now naming its oracle so none is provable as bare prose-grep. Full gates: gofmt/vet clean, ./... green except the pre-existing environment-only codex test in an untouched package.

## Stage Report: validation

- DONE: COVERAGE-EQUIVALENCE — for EACH of the 7 PORTed assertions, read the named seam and confirm it asserts the SAME behavioral property; row 12 NO-seam confirmed documented intent
  9a→internal/contract/gate_test.go::TestStartupGateAbortsBeforeDiscover (real stub --version, parses `contract` token, OBSERVES discover 0×/1× — covers gate+ordering, stronger than the deleted prose Contains). 11→concurrency_test.go::TestPathScopedCommitDoesNotSweepSibling (real git, path-scoped commit B, OBSERVES A not swept + still staged — the exact commit-discipline behavior). 10a+10b-positive→launcher_smoke_test.go::TestLauncherListSetArchive (real status binary list/set/archive observable). 13→internal/dispatch/build_parity_test.go::TestBuildParityCrossProduct (drives native dispatch.Run build, byte-compares emitted dispatch body to oracle). 14a→dispatch_test.go::TestSplitRootFolderWorktreeDispatch (OBSERVES worktree CODE working-dir present while entity-read+completion point at state path, no .worktrees/ — the CODE-only split-root behavior; verified at test lines 226-252). Row 12 has NO seam by documented intent: the --json OUTPUT-shape contract is genuinely covered by internal/status (TestBootJSONStructure, TestJSONReadGolden, TestSetJSONEnvelope, TestJSONStatusRoundTripsTableColumns, TestValidateJSONBranches), and the live "FO consumes --json" half is deferred Out-of-scope — confirmed not an accidental gap. No PORT lost real coverage.
- DONE: AC-2 reproduce — grep returns 0 build-driving invocations; runBuild drives in-process dispatch.Run; both dispatch tests pass on the same observable outputs
  `grep -nE 'python3|claude-team' skills/integration/dispatch_test.go` exit=1 (0 matches). runBuild calls dispatch.Run in-process (dispatch_test.go:43) with t.Setenv HOME + bytes.Buffer capture; os/exec residue is fixture git-init only. TestSplitRootFolderWorktreeDispatch + TestFlatEntitySlugUnchanged: --- PASS (name, dispatch-file suffix, no index leak, branch line, worktree working-dir, state-path entity-read + completion-signal, no .worktrees/ — all asserted at test lines 208-252).
- DONE: AC-1 KEEP-integrity + AC-3 gates — 4 KEPT assertions present each naming its oracle; 7 PORTed names gone via go test -list; full gates green with real captured exit codes
  KEEP: TestStartupEmbeddedRangeBracketsContractVersion (9b/14c, oracle=contract.ParseRange+CONTRACT_VERSION, comment lines 35-41), TestNoPluginPrivateStatusPathInContracts (10b-neg, oracle=plugin-private-path absence invariant, lines 27-33), TestNoPRMergeOrModBehaviorIntroduced (14b, oracle=amendment-region scope-fence, lines 69-76) — 9b/14c are one test spanning two rows, so 3 distinct tests cover 4 KEEP rows. `go test -list` (via rtk proxy): all 7 PORTed names GONE (StartupStepZeroIsContractGate, LauncherStatusInvocations, VendoredSkillsCallSpacedockStatus, ConcurrencySafeCommitClause, EventLoopReadsUseJSON, DispatchBlockUsesNativeBuild, SplitRootContractClause). Gates with real captured exits: gofmt -l . exit=0 (no files), go vet ./... exit=0, go test ./... exit=1 with the SOLE failure TestCodexResolveManifestAgainstInstalledHost ("failed to load configuration" — env codex host probe, package internal/cli untouched by this 4-file skills/integration change). Named seam packages green: internal/contract, internal/dispatch, internal/status, skills/integration all ok.

### Summary

PASSED. Independently reproduced the crux risk (silent coverage loss): read every one of the 7 PORTed assertions' named seams and confirmed each seam OBSERVES the same behavioral property the deleted bare-Contains claimed — gate version-probe+ordering (gate_test stub), path-scoped commit non-sweep (concurrency_test real git), launcher status surface (launcher_smoke real binary), native build body parity (build_parity_test), and split-root CODE-only handoff (the repointed dispatch_test). Row 12's no-seam disposition is documented intent — the --json output-shape contract is genuinely owned by internal/status JSON tests and the live FO-consumes---json half is deferred Out-of-scope, not an accidental gap. The 4 KEPT structural invariants each carry an oracle-naming comment (ParseRange+CONTRACT_VERSION; two absence invariants), provably not bare prose-grep. AC-2 grep clean, both dispatch tests green on identical observable outputs; full gates green with real captured exit codes, the only ./... failure being the pre-existing env-only codex test in an untouched package.
