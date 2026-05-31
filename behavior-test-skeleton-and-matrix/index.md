---
id: 8033qbqdrh4zba10w0d34m4j
title: Behavior-test skeleton + coverage matrix (replace the prose-grep antipattern)
status: validation
source: sprint — captain (2026-05-31); prereq to cutting the ensign contract
started: 2026-05-31T02:36:17Z
completed:
verdict:
score: "0.36"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-behavior-test-skeleton-and-matrix
issue:
---

Establish a behavioral test footing for the ensign/FO/launcher contract so the contract can be cut/trimmed safely. Two deliverables: a **skeleton behavior test** and a **coverage matrix**.

## Why
v1's contract coverage today is `skills/integration/*_test.go` — **STATIC prose-grep** (asserts the contract TEXT contains clauses; passes even if a clause doesn't behaviorally work). That's the antipattern P1/P2 distrust. The Python side has the real behavioral net: a CI workflow `~/git/spacedock/.github/workflows/runtime-live-e2e.yml` plus live pytest (`test_team_fail_early_live.py`, `test_checklist_e2e.py`, `test_codex_packaged_agent_e2e.py`, `test_agent_captain_interaction.py`, …). Before cutting the ensign contract we need (a) a behavioral skeleton in v1 and (b) a map of what's covered where.

## Deliverables / Acceptance criteria (provisional — ideation hardens)

**AC-1 — Coverage MATRIX.** A matrix: rows = the ensign/FO/launcher contract behaviors that SHOULD be tested (dispatch→ensign→stage cycle, stage-report shape, completion signal, gate/feedback flow, checklist accounting, split-root commits, launcher argv, etc.); columns = **python-covers** (which pytest/CI, and whether behavioral or static) × **v1-implements** (which Go test — or "prose-grep antipattern" / "GAP"). Surfaces the prose-grep tests + the real gaps; becomes the port roadmap. Verified by: the matrix exists, every row cites concrete test names/paths on both sides (reproducible, not hand-waved).

**AC-2 — SKELETON behavior test.** One minimal BEHAVIORAL test in v1 that exercises a real dispatch→ensign→stage mechanical cycle (or ports the smallest meaningful Python live behavior) and asserts mechanical outputs (stage-report section shape, the state commit, the completion signal) of a scripted/fixture run — NOT a live LLM agent, NOT prose-grep. A scaffold others extend. Verified by: the test runs in `go test` (or a documented harness command) and FAILS if the asserted mechanical output is broken.

## Out of scope (this entity)
- Full port of all Python live tests (the matrix plans it; this ships the skeleton + map).
- CI setup (the `runtime-live-e2e.yml` analog) — deferred, "when we get there".
- Actually cutting the ensign contract (this is the prereq, not the cut).

## Notes / sequencing
Test-infra surface (new test package/harness) — disjoint from frontdoor.go, so parallel with launch-parity. The matrix is the high-value planning artifact; the skeleton proves the behavioral pattern.

---

# Ideation design (hardened)

## Framing: what "behavioral" means here, and what the antipattern actually is

The ensign IS a live LLM. There is no Go function that "is" the ensign, so there is no fully-mechanical, no-live-agent test of the *whole* dispatch→ensign→stage loop. The Python net handles that with `@pytest.mark.live_claude` / `live_codex` tests that actually run `claude -p` / codex and parse the run log. v1 cannot port those as `go test` without a live runtime (out of scope: "CI when we get there").

What v1 CAN test mechanically, and what the skeleton targets, is the **deterministic seam on both sides of the LLM**:
- **FO→ensign handoff:** `internal/dispatch.Run(...)` is a pure in-process function (`args, stdin, stdout, stderr`) that assembles the dispatch body. The dispatch body IS the mechanical contract — it dictates the stage-report shape, the path-scoped state-commit form, and the completion-signal emit line.
- **ensign→state output:** the stage-report section appended to the entity, and the path-scoped state commit. These are byte-/git-observable.

The skeleton stubs the LLM with a **scripted ensign** (a Go-driven shell stand-in that performs exactly the mechanical protocol the dispatch body prescribes), then asserts the observable outputs. That is behavioral (it exercises the real build seam and a real git commit and a real entity append), not prose-grep (it does not assert that instruction TEXT contains a clause).

**The prose-grep antipattern, precisely.** `skills/integration/skill_text_test.go` (and `contract_status_path_test.go`, plus the bracketing assertions in `contract_gate_test.go`) `os.ReadFile` a `.md` instruction file and assert `strings.Contains(content, "<clause>")`. They pass even if the clause is never behaviorally exercised — the contract TEXT says "use `--json`", but nothing proves a run consumes `--json`. NOT all of `skills/integration/*_test.go` is the antipattern: `dispatch_test.go`, `launcher_smoke_test.go`, and `concurrency_test.go` are genuinely behavioral (they drive a binary / git and assert observable outputs). The matrix classifies each individually.

## Spike result (riskiest unknown — done FIRST, see AC-3 below)

Built a throwaway `TestSpikeScriptedEnsignCycle` in `internal/dispatch/` and ran it:
- **Positive:** fixture workflow + `dispatch.Run` build → real dispatch body; scripted ensign appends a `## Stage Report:` section and does a path-scoped `git commit -- {entity}`; asserts (a) stage-report heading + `- DONE:` marker + `### Summary`, (b) the commit landed and named only the entity, (c) the completion-signal emit line. PASS in ~0.1s, no live agent.
- **Negative-1** (scripted ensign writes `- [x]` checkbox bullets instead of `- DONE:`): FAILS at the DONE-marker assertion. Good.
- **Negative-2** (regress `dispatch.Run`: rename the emitted `SendMessage(...)` call to `Notify(...)`): with a naive `strings.Contains(body, 'SendMessage(to="team-lead"')` the test WRONGLY PASSED — because the body also carries a *"Do NOT paraphrase `SendMessage(...)`"* warning line, so Contains matched the prose even though the real emit line broke. **This is the prose-grep trap reappearing inside the skeleton.** Fixing the assertion to an anchored, indented emit-line regex `(?m)^    SendMessage\(to="team-lead", message="Done: ` made Negative-2 FAIL correctly.

**Lesson the skeleton bakes in:** assert the EMIT FORM (the literal call the ensign must run, anchored at its indentation), not a bare substring that prose can satisfy. This is the discipline the matrix demands of every ported behavioral row. Spike removed after validation; baseline `go test ./internal/dispatch/ ./skills/integration/` green (92 tests).

## AC-2 — SKELETON behavior test design

**Home:** new package `internal/ensigncycle` (test-infra surface, disjoint from `frontdoor.go` — parallel with launch-parity). Reuses the existing in-process `dispatch.Run` seam; no new production code in the dispatch package.

**`TestEnsignCycleMechanicalOutputs`** (runnable as `go test ./internal/ensigncycle/`):
1. Stage a fixture: git-init'd root, `README.md` with a non-worktree stage, a flat `{slug}.md` entity in `backlog`.
2. Call `dispatch.Run([]string{"build","--workflow-dir",root}, stdinJSON, &stdout, &stderr)` with a 2-item checklist; parse `dispatch_file_path` from stdout JSON; read the dispatch body.
3. **Scripted ensign** (the LLM stand-in): parse the checklist items from the body's `### Completion checklist` block, append a protocol-shaped `## Stage Report: backlog` section (one `- DONE:` per item + `### Summary`) to the entity, then path-scoped `git -C {root} commit -- {entity}`.
4. **Assert mechanical outputs:**
   - (a) entity matches `(?m)^## Stage Report: backlog$`, has `(?m)^- DONE:`, contains `### Summary`, and has NO `(?m)^- \[[ xX]\]` checkbox bullet (the checklist-e2e protocol rule).
   - (b) `git show --name-only HEAD` names the entity and ONLY the entity (path-scoped, no sibling sweep — mirrors `concurrency_test.go`'s invariant at the cycle level).
   - (c) the dispatch body contains the anchored completion-signal EMIT line `(?m)^    SendMessage\(to="team-lead", message="Done: ` (NOT a bare Contains — per the spike lesson).
   - (d) the dispatch body's stage-report directive names the `## Stage Report: {stage_name}` shape, DONE/SKIPPED/FAILED markers, and the path-scoped commit form (assert via the fetch-resolved stage def OR the body's prescribed protocol block — anchored, not bare Contains).

**Why this FAILS when the contract breaks (the AC-2 verification):** demonstrated by the spike's two negative controls — a broken ensign output (neg-1) and a regressed `dispatch.Run` emit line (neg-2) each turn the test red. The test is a scaffold others extend: each new ported Python behavior adds a fixture + a scripted-ensign step + an anchored mechanical assertion.

**What it deliberately does NOT cover** (matrix plans the port): live LLM compliance (does a *real* ensign emit the signal), gate/feedback approval loops, codex-runtime dispatch, multi-entity concurrency under a live FO. Those stay live-pytest until v1 grows a runtime harness.

## AC-1 — COVERAGE MATRIX

Rows = ensign/FO/launcher contract behaviors that SHOULD be tested. Columns:
- **python-covers** = concrete pytest/CI path + `[behavioral-live]` (runs a real runtime) / `[static]` (no runtime).
- **v1-implements** = concrete Go test, or `prose-grep antipattern`, or `GAP`.

| # | Contract behavior | python-covers | v1-implements |
|---|---|---|---|
| 1 | Dispatch body carries a Completion checklist; ensign accounts for it in a Stage Report (DONE/SKIPPED/FAILED, `### Summary`, no checkboxes) | `tests/test_checklist_e2e.py::test_checklist_e2e_{helper,break_glass,codex}` **[behavioral-live]** | **GAP** (skeleton AC-2 closes the mechanical half; live half deferred) |
| 2 | Team-mode dispatch emits the `SendMessage(to="team-lead")` completion-signal emit line; entity advances to done | `tests/test_dispatch_completion_signal.py::test_dispatch_completion_signal` **[behavioral-live, teams_mode]** | skeleton AC-2 asserts the emit-line in the body (mechanical); live advancement is **GAP** |
| 3 | Dispatch-name derivation: slug-not-stem, folder-form, kebab-case, length cap | `tests/test_dispatch_names.py` **[static + 1 live]** | `internal/dispatch/build_parity_test.go` (`name` channel), `skills/integration/dispatch_test.go::TestSplitRootFolderWorktreeDispatch` / `TestFlatEntitySlugUnchanged` **[behavioral]** |
| 4 | Split-root worktree: CODE-only isolation, entity-read + completion-signal point at state path (no `.worktrees/` segment) | `tests/test_claude_team.py` (build paths) **[static]** | `skills/integration/dispatch_test.go::TestSplitRootFolderWorktreeDispatch` **[behavioral]** + `internal/dispatch/build_parity_test.go` split cases |
| 5 | Fetch-on-demand: dispatch body emits a `### Fetch commands` block with show-stage-def | `tests/test_fetch_on_demand_dispatch.py` **[1 static + 2 live]** | `internal/dispatch/build_parity_test.go` (body parity asserts the fetch line) **[behavioral]** |
| 6 | Break-glass manual-dispatch template carries the stage-report block verbatim | `tests/test_breakglass_dispatch_prompt.py` **[static]** | **GAP** (v1 native build has no break-glass template path yet) |
| 7 | Build input validation (missing fields, bad schema_version, empty checklist, unreadable entity, worktree-abs path) | `tests/test_claude_team.py` **[static]** | `internal/dispatch/build_errors_test.go`, `build_hazards_test.go` **[behavioral, in-process + oracle parity]** |
| 8 | Per-stage model precedence (stage > defaults > null) into Agent model field | `tests/test_claude_per_stage_model.py` **[static]** | `internal/dispatch/build_parity_test.go` (model channel) **[behavioral]** |
| 9 | FO startup contract gate: `spacedock --version` parses `contract`, aborts on mismatch, precedes discover/boot | (no direct python analog; FO contract is template prose) | `skills/integration/contract_gate_test.go::TestStartupStepZeroIsContractGate` — **prose-grep antipattern** (asserts step-1 prose `Contains "spacedock --version"`, ordering by `strings.Index`); the *behavioral* half (binary emits `contract` token, fixture brackets it) lives in `internal/contract/gate_test.go` **[behavioral]** |
| 10 | FO issues load-bearing reads/mutations via `spacedock status` (discover/boot/set/archive); no plugin-private status path | `tests/test_status_script.py` (status behavior, not FO wiring) **[static]** | **prose-grep antipattern**: `skills/integration/skill_text_test.go::TestLauncherStatusInvocations`, `contract_status_path_test.go::TestVendoredSkillsCallSpacedockStatus` (assert FO text Contains the invocation strings). Behavioral counterpart of the status surface itself: `internal/status/*` + `skills/integration/launcher_smoke_test.go::TestLauncherListSetArchive` **[behavioral]** |
| 11 | Concurrency-safe path-scoped state commit (no sibling sweep) | (guardrail prose in templates) **[static]** | `skills/integration/concurrency_test.go::TestPathScopedCommitDoesNotSweepSibling` **[behavioral]**; the *prose* requirement: `skill_text_test.go::TestConcurrencySafeCommitClause` — **prose-grep** |
| 12 | Event-loop scheduling reads consume `--json`; mod-block clear stays bare | (FO template prose) | `skills/integration/skill_text_test.go::TestEventLoopReadsUseJSON` — **prose-grep antipattern** |
| 13 | Dispatch block invokes native `spacedock dispatch build`, no `claude-team` in fenced cmd | (n/a — Python IS claude-team) | `skills/integration/skill_text_test.go::TestDispatchBlockUsesNativeBuild` — **prose-grep**; behavioral counterpart: `build_parity_test.go` body parity **[behavioral]** |
| 14 | Split-root / no-PR-merge / no-Hook contract clauses present | (template prose) | `skill_text_test.go::TestSplitRootContractClause`, `TestNoPRMergeOrModBehaviorIntroduced`, `contract_gate_test.go::TestStartupEmbeddedRangeBracketsContractVersion` — **prose-grep** (15 is a legitimate-structural exception, see below) |
| 15 | Gate/feedback approval loop + rejection reflow + feedback keep-alive | `tests/test_gate_guardrail.py`, `test_rejection_flow{,_codex}.py`, `test_feedback_keepalive.py`, `test_merge_hook_guardrail.py` **[behavioral-live]** | **GAP** (no v1 behavioral coverage of the gate loop at all) |
| 16 | Team fail-early: fresh-suffixed TeamCreate name, no pre-dispatch config probe | `tests/test_team_fail_early_live.py` **[behavioral-live, standalone script]** | partial: `internal/dispatch/guard_test.go` covers the bare-mode team-evidence WARN **[behavioral]**; the live TeamCreate-name check is **GAP** |
| 17 | Codex packaged-agent dispatch + completion-commit-hash wording | `tests/test_codex_packaged_agent_e2e.py` (6 static wording + 1 live) | **GAP** (v1 codex dispatch behavior untested) |
| CI | The whole live net (claude per-stage, bare-team, codex tiers) | `.github/workflows/runtime-live-e2e.yml` (4 jobs: `--runtime claude` / `--team-mode=bare` / model-override / `--runtime codex`) **[behavioral-live CI]** | **GAP — deferred** ("CI when we get there") |

### hx (deliverable-contract-hardening) reconciliation — P1 vs P2

hx's code guard catches **P1** (self-reference ACs — an AC that asserts about its own entity text). It does NOT catch **P2** (prose-grep behavioral tests). THIS entity is the P2 complement. Classifying hx's two planned static invariants:
- **hx AC-2 (legitimate structural invariant):** a static assertion over genuine on-disk *structure* (e.g. an AC that asserts a frontmatter field shape, a file's existence, a parser-level invariant) is NOT prose-grep — it asserts a structural fact the system mechanically depends on, with an oracle. Matrix rows **3, 4, 7, 8** are this kind: they exercise the real build and compare against the Python oracle byte-for-byte. **Keep.**
- **hx AC-3 (zero-oracle-token → low-stakes prose-grep):** a static assertion with NO oracle that only greps instruction prose for a clause (matrix rows **9-prose, 10-prose, 11-prose, 12, 13-prose, 14**) is the P2 antipattern. These pass while the asserted behavior may be dead. **Port to behavioral** where a seam exists (most have a behavioral counterpart already — see the v1-implements column), or accept as low-stakes-prose with that label.

**Port roadmap (priority order, from the matrix):**
1. Row 1 + 2 (checklist accounting + completion-signal emit) — the skeleton AC-2 closes the mechanical half NOW; this is the highest-value behavioral footing.
2. Row 15 (gate/feedback loop) — the largest pure GAP; needs a scripted-FO harness extension of the skeleton.
3. Rows 9/10/12/13/14 prose-grep — retarget each at its behavioral seam (binary/git/build), retire the bare-Contains assertion, keeping only genuine hx-AC-2 structural invariants.
4. Rows 16/17 (team fail-early, codex) and CI — deferred to a live-runtime harness.

## Stage Report: ideation

- DONE: The COVERAGE MATRIX design (rows = contract behaviors; columns = python-covers [behavioral/static] × v1-implements [Go test / prose-grep antipattern / GAP])
  17 rows + CI row, every row cites real names on both sides; includes the skills/integration prose-grep rows (9/10/11/12/13/14) and hx AC-2/AC-3 classification (legitimate-structural vs prose-grep-to-replace). See "AC-1 — COVERAGE MATRIX".
- DONE: The SKELETON behavior test design (one minimal BEHAVIORAL test exercising a real dispatch→ensign→stage mechanical cycle, asserting mechanical outputs of a scripted/fixture run — not live LLM, not prose-grep; pinned as runnable `go test` that FAILS when the asserted output breaks)
  `internal/ensigncycle/TestEnsignCycleMechanicalOutputs` design in "AC-2 — SKELETON behavior test design"; FAIL-on-break proven by the spike's two negative controls.
- DONE: Spike the riskiest unknown FIRST (confirm the chosen skeleton approach actually RUNS in v1 — build the smallest version and observe pass/fail before committing the design)
  Built+ran `TestSpikeScriptedEnsignCycle` in internal/dispatch: positive PASS (~0.1s, no live agent); negative-1 (checkbox bullets) FAIL; negative-2 (regressed emit line) FAIL only after fixing a discovered prose-grep trap. Spike removed; baseline green. See "Spike result".

### Summary

Ideation produced the coverage matrix (the port roadmap) and the skeleton design, both grounded in a run spike rather than on paper. The load-bearing finding: the ensign is a live LLM, so v1's behavioral footing must target the deterministic seams on both sides — `dispatch.Run` (in-process build) and the entity stage-report + git commit — stubbing the LLM with a scripted ensign. The spike surfaced a real trap: a bare `strings.Contains` for the completion signal is fooled by the body's own "do not paraphrase `SendMessage(...)`" warning prose; the skeleton therefore asserts the anchored, indented EMIT line. That trap-discovery is precisely the prose-grep failure mode this entity exists to replace, caught inside the skeleton itself. Matrix classifies all six skills/integration prose-grep assertions and reconciles with hx (P1 self-ref vs P2 prose-grep): this entity is the P2 complement.

## Stage Report: implementation

- DONE: Build the SKELETON behavioral test per the AC-2 design: fixture workflow + real `dispatch.Run` build → real dispatch body; a scripted-ensign stand-in appends a protocol-shaped `## Stage Report` (one `- DONE:` per checklist item + `### Summary`, no checkbox bullets) and does a path-scoped `git -C {root} commit -- {entity}`; assert the MECHANICAL outputs — stage-report heading/markers, the commit landed naming ONLY the entity, and the completion-signal EMIT LINE — using ANCHORED forms (e.g. `(?m)^    SendMessage\(to="team-lead", message="Done: `), NOT bare `strings.Contains`
  `internal/ensigncycle/cycle_test.go::TestEnsignCycleMechanicalOutputs` (commit 3c5dc60). Asserts (a) anchored stage-report heading/`- DONE:`/`### Summary` + NO checkbox bullet, (b) `git show --name-only HEAD` names only `make-it-work.md`, (c) anchored completion-signal emit regex on the real body, (d) anchored `Skill(skill="spacedock:ensign")` first-action + `show-stage-def` fetch emit lines (the protocol-loading wiring; the protocol text lives in ensign-shared-core.md, not the body — so asserting the loading mechanism is the behavioral link, not prose-grep).
- DONE: The test FAILS when the mechanical output breaks: include the spike's negative cases as guard assertions (rename the emit call → test must fail; drop the stage-report heading → test must fail). Runs in `go test`. The coverage MATRIX (AC-1) in the entity body is the planning deliverable — confirm every row cites concrete test names both sides
  `TestEnsignCycleGoesRedOnBrokenOutput` holds both spike negatives (checkbox-bullets, Notify-rename + bare-Contains trap demo). Mutation-verified: regressing the real `dispatch.Run` emit line to `Notify(...)` turned `TestEnsignCycleMechanicalOutputs` RED (the prose `SendMessage(...)` warning survived — exactly the prose-grep trap the anchored regex catches), green again on restore. Matrix v1-side names spot-checked: all 9 cited Go tests resolve to real files (dispatch_test.go, contract_gate_test.go, skill_text_test.go, concurrency_test.go, launcher_smoke_test.go, …).
- DONE: Gates green with REAL captured exit codes (go test ./..., -race, gofmt -l, go vet)
  `go test ./...` 430 passed EXIT=0; `go test -race ./internal/ensigncycle/` EXIT=0; `gofmt -l internal/ensigncycle/` empty EXIT=0; `go vet ./internal/ensigncycle/` no issues EXIT=0.

### Summary

Shipped the skeleton behavioral test in a new `internal/ensigncycle` package (disjoint from frontdoor.go, reusing the in-process `dispatch.Run` seam with no new production code). It exercises a real build → scripted-ensign stage-report append → real path-scoped git commit, and asserts the mechanical outputs via anchored EMIT-form regexes — baking in the spike's load-bearing lesson that a bare `strings.Contains` for the completion signal is fooled by the body's own "do not paraphrase SendMessage(...)" prose. The two negative guards encode the spike's controls; a live mutation of the real emit line confirmed the positive test goes red (and that the prose warning is the trap), proving the FAIL-on-break bar rather than asserting it. The AC-1 coverage matrix (the port roadmap) stays in the entity body; its v1-side test names were spot-checked against the tree and all resolve.

## Stage Report: validation

- DONE: Independently confirm the SKELETON test RUNS and is genuinely BEHAVIORAL (real dispatch.Run build + real git commit + real entity append via a Go-scripted ensign, not live LLM, not prose-grep); reproduce the run
  Reproduced: `go test ./internal/ensigncycle/` 4/4 pass (~0.4s). `TestEnsignCycleMechanicalOutputs` calls real `dispatch.Run(["build",...])` in-process, a Go scripted ensign appends a `## Stage Report` section, then real `git add -- / git commit -- {entity}`. No live agent.
- DONE: Confirm it is a real GUARD that FAILS when mechanical output breaks; reproduce both negatives; verify the emit assertion is ANCHORED not bare strings.Contains
  REAL mutations (beyond the in-test guard): (1) renamed production emit `SendMessage`->`Notify` at build.go:375 -> test RED at the anchored-emit assertion; failure dump showed the prose warning `Do NOT paraphrase SendMessage(...)` survived intact, proving a bare Contains would have wrongly PASSED and the anchored `(?m)^    SendMessage\(to="team-lead", message="Done: ` regex caught it. (2) dropped the `## Stage Report` heading in the scripted ensign -> RED at cycle_test.go:154. Both reverted; tree clean.
- DONE: Confirm the coverage MATRIX (AC-1) is complete + accurate; every row cites concrete test names both sides; spot-check 2-3 classifications; gates green with REAL captured exit codes
  All 13 v1-cited Go test names resolve to real files (dispatch_test.go, contract_gate_test.go, skill_text_test.go, concurrency_test.go, launcher_smoke_test.go, contract_status_path_test.go); cited internal/{contract,dispatch} test files exist; Python-side files exist in ~/git/spacedock/tests/. Spot-checked: row 9 prose-half (strings.Contains+Index over startup prose = prose-grep), row 11 (real exec.Command git, observable commit = behavioral), row 12 (Contains-grep of runtime .md for --json = prose-grep) — all match classifications. Gates: `go test ./...` EXIT=0; `go test -race ./internal/ensigncycle/` EXIT=0; `gofmt -l internal/ensigncycle/` empty EXIT=0; `go vet ./internal/ensigncycle/` EXIT=0.

### Summary

VERDICT: PASSED. The skeleton (`internal/ensigncycle/cycle_test.go`, the branch's sole net-new file vs main, disjoint from frontdoor.go, no production code touched) is a genuine behavioral guard, not prose-grep: it drives the real in-process `dispatch.Run` build seam, a real path-scoped git commit, and a real entity append via a Go-scripted ensign stand-in. The behavioral-proof bar holds under independent REAL mutation: regressing the production emit line to `Notify(...)` turns the positive test RED while the body's own `Do NOT paraphrase SendMessage(...)` prose warning survives — the exact prose-grep trap the spike found, now demonstrably caught by the anchored EMIT-form regex rather than a bare `strings.Contains`. Dropping the stage-report heading also goes red. The AC-1 coverage matrix is complete and accurate (every row cites resolvable test names on both sides; spot-checked classifications correct) and is the port roadmap that flags the six skills/integration prose-grep assertions. One immaterial nit: the implementation report says "430 passed" but the current count is 424 (suite is green EXIT=0 regardless).
