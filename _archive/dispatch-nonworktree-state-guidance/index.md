---
id: k69e2gcvykjrc5354ty7kt3g
title: Dispatch — inject split-root state-commit guidance for non-worktree (ideation) stages
status: done
source: FO dogfooding friction #1 (2026-05-31); root-caused build.go:302 worktree-gating
started: 2026-05-31T18:40:08Z
completed: 2026-05-31T19:15:57Z
verdict: PASSED
score: "0.24"
worktree: 
issue:
archived: 2026-05-31T19:15:57Z
---

The native dispatch build injects the split-root state-commit guidance (`git -C {state_checkout} add/commit -- {entity}`) ONLY for worktree stages — `internal/dispatch/build.go:302` gates the whole block on `if worktreePath != ""`. So NON-worktree dispatches (ideation, backlog) get NO state-commit instruction. Result: ideation ensigns edit the entity in `.spacedock-state` (git-excluded from the main checkout), try a bare `git add`, hit the exclusion, and report "couldn't commit — gitignored." This recurred on every ideation-stage dispatch this session (worktree-stage ensigns committed cleanly).

## Problem statement (ideation)

The split-root state-commit guidance in the native dispatch builder (`internal/dispatch/build.go`) is broken on BOTH branches of the same emission surface. Both defects are faithful ports of the same bugs in the vendored Python oracle (`skills/commission/bin/claude-team:414-431`), so neither is a native regression — but both must now be fixed as a deliberate, documented divergence from the oracle (the same kind already established for the fetch line; see `oracleFetchPrefix`/`nativeFetchPrefix` in `parity_harness_test.go`).

- **Defect A — non-worktree split-root stages get NO guidance.** The whole state-commit block sits inside `if worktreePath != "" {` at `build.go:302`. A non-worktree split-root stage (ideation, backlog) therefore receives zero state-commit instruction, edits the entity in `.spacedock-state` (git-excluded from the main checkout's index), tries a bare `git add`, hits the exclusion, and reports "couldn't commit — gitignored."
- **Defect B — worktree split-root stages get LITERAL placeholders.** `build.go:314-315` emits `git -C {state_checkout} add {entity_path} && git -C {state_checkout} commit -m "..." -- {entity_path}` with the brace tokens as Go string literals — they are never substituted with the resolved absolute paths. A literal copy-paste of that command fails (cwd is the worktree, which has no `.spacedock-state`, and `{state_checkout}` is a brace string). Worktree ensigns currently succeed only by *inferring* the real paths from the absolute entity path that appears in the entity-read line.

Both branches share one resolved fact pair: under split root the state checkout is `workflowDir/<state>` — build is called with the README/definition dir (`docs/dev`), and `resolveRoots` sets `entityDir = definitionDir/<state>` (e.g. `docs/dev/.spacedock-state`). The state checkout is NOT `workflowDir` itself: `workflowDir` is the definition root, where the state checkout is a git-excluded child. The entity path is the absolute `entityPath` the FO passed. Both the resolved state checkout and the entity path must be derived/in scope at the emission point.

## Acceptance criteria

**AC-1 — Both split-root stages carry resolved, brace-free state-commit guidance.**
End state: for a split-root workflow, the dispatch body for BOTH a worktree-stage AND a non-worktree-stage dispatch contains the path-scoped state-commit instruction — `git -C <ABS_STATE_CHECKOUT> add <ABS_ENTITY_PATH> && git -C <ABS_STATE_CHECKOUT> commit -m "..." -- <ABS_ENTITY_PATH>` — with the resolved absolute state-checkout path and entity path substituted in, the "never a bare `git add -A`/`git commit`" concurrency note present, and NO literal `{state_checkout}` or `{entity_path}` brace token anywhere in the body.
Verified by: a dispatch-build test (alongside `internal/dispatch/build_parity_test.go`) that builds each split-root dispatch and asserts on the emitted body — exercise the build, observe the body (behavioral, not a prose-grep of the contract).

**AC-2 — Non-split-root and worktree-stage CODE behavior unchanged.**
End state: a non-split-root dispatch emits no state-commit block at all (single-root has no state checkout); a split-root worktree stage still emits its CODE-branch directory/branch instructions ahead of the state-commit guidance.
Verified by: the existing build cross-product parity test stays green AFTER the parity harness subtracts this intentionally-diverged block the same way it rewrites the fetch line (see Design below) — and the new behavioral test in AC-1 covers the diverged content the parity test no longer asserts.

## Proposed approach

**build.go change.** Lift the split-root state-commit guidance OUT of the `if worktreePath != "" {` block and substitute the resolved paths. Concretely:

1. Compute the two resolved values once near the `splitRoot` derivation (build.go:222): the absolute state-checkout path is `workflowDir/<state>` — resolve it from the README's `state:` value (`splitRootStateCheckout(workflowDir)` = `filepath.Join(workflowDir, state)`, the same join `resolveRoots` uses), NOT `workflowDir` itself (the definition dir, where the state checkout is git-excluded). The absolute entity path is `entityPath`.
2. In the worktree+split branch (build.go:304-319), KEEP the CODE-directory/branch lines but replace the trailing `{state_checkout}`/`{entity_path}` literals with `%s` format verbs bound to those two resolved values — so the worktree split-root body ships the real command.
3. Add a NON-worktree split-root branch: when `worktreePath == "" && splitRoot`, emit a standalone state-commit guidance paragraph (the same "This workflow is split-root … never a bare `git add -A` …" wording, minus the CODE-directory lines, since a non-worktree stage runs from the repo root) with the resolved paths substituted. The cleanest factoring is a single `stateCommitGuidance(stateCheckout, entityPath)` helper string both split-root branches append, so the wording lives in one place.

**Ensign-contract de-frame.** The vendored `skills/ensign/references/ensign-shared-core.md` section is titled `### Split-Root Worktree Contract` and opens "When the workflow is split-root … your worktree isolates CODE only" — a non-worktree ensign reading it concludes it does not apply to them. De-frame it (see checklist item 2 for exact before/after) so a non-worktree split-root ensign sees the concurrency-safe state-commit rule governs them too.

## Test plan

- **New behavioral test** `internal/dispatch/build_statecommit_test.go` (Go unit test, alongside `build_parity_test.go`, reusing its `readmeWorktree`/`entityFM`/`writeFile`/`gitInit`/`runNative`/`readDispatchBody` fixtures). Two sub-cases, both `splitRoot: true`: one worktree stage (`implementation`), one non-worktree stage (`backlog`). For each, build the native dispatch, read the emitted body, and assert:
  - POSITIVE: the body contains `git -C <abs-state-checkout> add <abs-entity-path>` and the `commit -m` half, with the test's actual resolved absolute paths (the fixture's `workflowDir` and `entityPath`) substituted in.
  - NEGATIVE (the encoded failure): `strings.Contains(body, "{state_checkout}")` and `strings.Contains(body, "{entity_path}")` are both false. A body that still carries a literal `{state_checkout}` or `{entity_path}` brace MUST fail the test — this is the regression that pins Defect B and prevents a future revert to the literal-brace port.
  - The non-worktree sub-case additionally asserts the body is non-empty for the state-commit guidance (pins Defect A — proves the block is no longer gated away).
- **Parity-harness divergence.** Because the oracle still emits the gated/literal-brace block, the build cross-product parity test would mismatch after the fix. Extend the harness's existing oracle-rewrite step (the fetch-line rewrite in `parity_harness_test.go`) to also normalize away the state-commit guidance block before byte-comparing — mirroring the established `oracleFetchPrefix`→`nativeFetchPrefix` precedent — so AC-2's existing parity cases stay green. The diverged content is now covered by the AC-1 behavioral test instead.
- **Cost/complexity:** low. Pure Go unit tests + fixture builds; no live workflow run needed (the claim is emitted-string behavior, provable at the build seam). Estimated one new ~80-line test file plus a small harness rewrite addition.

## Related finding — folded in 2026-05-31 (FO, behavior-coverage sprint)

The **worktree-stage** branch of the same guidance is *also* broken, in a different way: `internal/dispatch/build.go:314-315` emits the path-scoped commit command with **literal, unsubstituted `{state_checkout}` and `{entity_path}` placeholders** — they are hardcoded brace-literals in the Go string, never replaced with the resolved absolute paths. Confirmed by reading `build.go` source and the generated dispatch file `/tmp/spacedock-dispatch/spacedock-ensign-*-implementation.md` (line 16 ships the literal braces).

Worktree-stage ensigns succeed only by *inferring* the real paths from the absolute entity path that appears elsewhere in the prompt ("Read the entity file at /abs/.../index.md"). A literal copy-paste of the emitted `git -C {state_checkout} …` command fails: the cwd is the worktree, which (correctly, by split-root design) contains no `.spacedock-state`, and `{state_checkout}` is a literal brace string. The FO had to inject explicit absolute-path guidance into worktree-stage dispatches this sprint to be safe.

**Scope impact:** the fix is broader than "add a non-worktree branch." The worktree-stage branch must **substitute** `{state_checkout}` and `{entity_path}` with the resolved absolute paths too. AC-1's verification should be strengthened to assert the emitted body for BOTH worktree and non-worktree split-root stages contains the **resolved absolute state-checkout path** and **no literal `{state_checkout}`/`{entity_path}` brace tokens**.

## Notes
- Small `internal/dispatch/build.go` change (lift the state-commit guidance out of the `worktreePath != ""` block, or add a non-worktree split-root branch). Companion: de-frame the vendored ensign contract's "Split-Root **Worktree** Contract" section so a non-worktree ensign sees it applies (skills/ensign/references/ensign-shared-core.md).
- Worktree-stage branch (build.go:314-315) must additionally substitute the `{state_checkout}`/`{entity_path}` placeholders with resolved absolute paths (see "Related finding" above) — the two defects share the same build.go state-commit-guidance surface and should be fixed together.
- Sequencing: touches `internal/dispatch/build.go` — coordinate with the module-path migration (which rewrites imports across the repo). Do this fix before OR after the migration, not concurrently.
- Not on the launcher/install critical path; queued.

## Ensign-contract de-frame — exact before/after (AC item 2)

Target: `skills/ensign/references/ensign-shared-core.md`, the `### Split-Root Worktree Contract` section (lines 28-39).

**Heading — before:** `### Split-Root Worktree Contract`
**Heading — after:** `### Split-Root State Contract`

**Opening — before:**
> When the workflow is split-root — the workflow README declares a `state:` checkout (e.g. `state: .spacedock-state`) — your worktree isolates **CODE only**. The entity body and your stage report do NOT live in the worktree; they live in the separate state checkout that the dispatch hands you as the entity path.

**Opening — after:**
> When the workflow is split-root — the workflow README declares a `state:` checkout (e.g. `state: .spacedock-state`) — the entity body and your stage report live in the separate state checkout that the dispatch hands you as the entity path, NOT alongside the code. This applies to every split-root stage. **If your stage has a worktree**, the worktree isolates **CODE only** and the entity/report stay in the state checkout. **If your stage has no worktree** (ideation, backlog), you run from the repo root and still write/commit the entity and report to the state checkout the dispatch named — the concurrency-safe commit rule below governs you too.

Rationale: the original frames the whole section through "your worktree isolates CODE," so a non-worktree ensign reads the section as not addressed to it and skips the concurrency-safe commit rule. The after-wording leads with the state-checkout fact (applies to all split-root stages) and splits the worktree/no-worktree cases explicitly, so a non-worktree ensign sees the commit discipline applies. The `**Concurrency-safe state commits.**` paragraph that follows (the bare-`git add -A` prohibition + tool-managed/path-scoped fallback) is unchanged — it already reads generically once the framing no longer presupposes a worktree.

## Stage Report: ideation

- DONE: Pin the build.go fix for BOTH defects on the shared state-commit-guidance surface
  Problem statement + Proposed approach name the exact change: lift the block out of `if worktreePath != ""` (build.go:302), substitute `%s` verbs bound to resolved `workflowDir` (abs state checkout) + `entityPath` for the literal `{state_checkout}`/`{entity_path}` at build.go:314-315, add a non-worktree split-root branch, factor wording into one `stateCommitGuidance(stateCheckout, entityPath)` helper.
- DONE: Name the companion ensign-contract de-frame with specific before/after wording
  "Ensign-contract de-frame" section gives verbatim before/after for the heading (`Split-Root Worktree Contract`→`Split-Root State Contract`) and the opening paragraph of `skills/ensign/references/ensign-shared-core.md` §28-39, plus rationale; the Concurrency-safe paragraph stays unchanged.
- DONE: Harden AC-1/AC-2 behavior-first with the encoded negative named
  Test plan specifies `build_statecommit_test.go` asserting both split-root stages' bodies contain the RESOLVED absolute paths and that `{state_checkout}`/`{entity_path}` braces are absent; the encoded negative is "a body still carrying a literal `{state_checkout}`/`{entity_path}` brace MUST fail." AC-2 keeps cross-product parity green by extending the harness's existing oracle-rewrite (fetch-line precedent) to subtract the diverged block.

### Summary

Root-grounded both defects against the live `internal/dispatch/build.go` source and the vendored Python oracle (`skills/commission/bin/claude-team:414-431`): the oracle is *identically* buggy on both branches (whole block gated on `if worktree_path:`, literal `{{...}}` braces), so the fix is a deliberate documented divergence — the same shape as the existing fetch-line divergence in `parity_harness_test.go`. The key design consequence the seed did not yet capture: the build cross-product parity test will break unless the parity harness subtracts this block before byte-comparing, with the AC-1 behavioral test covering the diverged content instead. Scope held tight to the one build.go surface + the companion contract de-frame; no code committed (ideation).

## Stage Report: implementation

- DONE: Fix internal/dispatch/build.go per the design — factor stateCommitGuidance helper, lift out of worktree gate, substitute resolved abs paths on both branches
  build.go: new `stateCommitGuidance(stateCheckout, entityPath)` holds the wording once; worktree+split branch appends it with `workflowDir`+`entityPath` (`%s`); added `else if splitRoot` non-worktree branch emitting the standalone helper. Commit 00d3e5b on branch spacedock-ensign/dispatch-nonworktree-state-guidance.
- DONE: Add behavioral test build_statecommit_test.go — two split-root sub-cases, POSITIVE resolved paths + NEGATIVE brace-absent; keep parity green via oracle-rewrite subtraction
  `build_statecommit_test.go` (TDD: failed first — wt shipped braces, nonwt emitted nothing). Two sub-cases assert `git -C <abs-state> add <abs-entity>` + commit half with the fixture's real resolved paths, and `strings.Contains` false for `{state_checkout}`/`{entity_path}`. Parity kept green by `stripStateCommitGuidance` (regex strips the diverged line from BOTH sides + collapses the blank-line artifact), the established fetch-line divergence shape. Commit 00d3e5b.
- DONE: Companion de-frame skills/ensign/references/ensign-shared-core.md per exact before/after; gates green with real captured exit codes
  `### Split-Root Worktree Contract` → `### Split-Root State Contract` + generalized opening (leads with the state-checkout fact, splits worktree/no-worktree cases) verbatim per entity §72-79; Concurrency-safe paragraph unchanged. Gates from inside the worktree: `go test ./...` GOTEST_EXIT:1 (only pre-existing env-only TestCodexResolveManifestAgainstInstalledHost), `gofmt -l .` GOFMT_EXIT:0 (empty), `go vet ./...` VET_EXIT:0. dispatch pkg: 81 passed.

### Summary

Lifted the split-root state-commit guidance out of build.go's `if worktreePath != ""` gate and substituted the resolved absolute `workflowDir` (state checkout) + `entityPath` for the literal `{state_checkout}`/`{entity_path}` braces on BOTH the worktree and the new non-worktree branch, with the wording factored into one `stateCommitGuidance` helper. Followed TDD: the new `build_statecommit_test.go` failed first (worktree shipped braces, non-worktree emitted nothing), then passed; the cross-product parity stayed green by extending the harness's oracle-rewrite to strip the deliberately-diverged block from both sides — mirroring the existing fetch-line divergence, since the Python oracle is identically buggy. Companion de-frame of the vendored ensign contract retitles the section and generalizes its opening so a non-worktree split-root ensign sees the commit discipline applies. All CODE on branch spacedock-ensign/dispatch-nonworktree-state-guidance (commit 00d3e5b); only the pre-existing env-only codex test fails.

## Stage Report: validation

- DONE: Reproduce the build.go fix behaviorally and INDEPENDENTLY: build a split-root WORKTREE dispatch AND a split-root NON-worktree dispatch via native dispatch.Run; confirm each emitted body contains the RESOLVED ABSOLUTE state-checkout + entity paths and NO literal `{state_checkout}`/`{entity_path}` braces. Reproduce the TDD red→green.
  Wrote my own throwaway diag test driving native `dispatch.Run` for `implementation` (worktree) and `backlog` (non-worktree) split-root stages: both bodies emit `git -C <ABS_STATE> add <ABS_ENTITY> && git -C <ABS_STATE> commit ...` with resolved absolute paths, zero brace tokens. Re-injected a literal `{state_checkout}`/`{entity_path}` brace into `stateCommitGuidance` → `build_statecommit_test.go` went RED on BOTH sub-cases (missing-resolved-path AND brace-present assertions fired); restored → GREEN.
- DONE: CRUX — confirm the parity-harness oracle-rewrite (stripStateCommitGuidance) subtracts ONLY the deliberately-diverged state-commit block and does NOT mask an unintended divergence.
  Dumped raw native vs oracle bodies for both split-root parity cases: the ONLY byte difference is the single state-commit guidance line (worktree: resolved-paths-vs-literal-braces; non-worktree: native-line-vs-oracle-nothing) — every other byte (CODE-dir lines, entity-read, checklist, fetch, completion signal) is identical; POST-STRIP both bodies are byte-equal (parity green for the RIGHT reason). Strip is NOT too aggressive: regex `This workflow is split-root: [^\n]*? after a short wait\.\n` is line-anchored (`[^\n]` cannot cross newlines, so no multi-line over-match) and the anchor phrase is UNIQUE — one occurrence per body, verified in both build.go (1×) and the oracle (1×, gated on `if worktree_path:`). The `\n{3,}`→`\n\n` collapse is applied symmetrically to both sides, so it cannot hide an asymmetry.
- DONE: AC-1/AC-2 + de-frame + gates: confirm the ensign-shared-core.md de-frame matches the entity's exact before/after; full gates green with REAL captured exit codes from inside the worktree.
  De-frame matches §72-79 verbatim: heading `Split-Root Worktree Contract`→`Split-Root State Contract`, opening generalized (leads with state-checkout fact, "applies to every split-root stage", splits worktree/no-worktree), Concurrency-safe paragraph unchanged (its `{state_checkout}` braces are generic contract docs, not an emitted command — correctly left untouched). Gates from inside the worktree at HEAD 00d3e5b (clean tree): `gofmt -l .` GOFMT_EXIT:0 (empty), `go vet ./...` VET_EXIT:0, `go test ./...` GOTEST_EXIT:1 — 488 passed, sole failure the pre-existing env-only `TestCodexResolveManifestAgainstInstalledHost`. AC-1 covered by `build_statecommit_test.go` (2 sub-cases, resolved-paths + brace-absent); AC-2 covered by `TestBuildParityCrossProduct` staying green (8 cases) post-strip.

### Summary

PASSED. Independently reproduced both defects' fixes behaviorally via native `dispatch.Run` and confirmed the TDD red→green by re-injecting a literal brace (test went RED on both worktree and non-worktree sub-cases, restored to GREEN). The crux check holds: the parity strip subtracts EXACTLY the one intended state-commit divergence — raw body diff shows that line is the sole difference, the strip regex is line-anchored to a phrase unique in each emitter, and POST-STRIP the bodies are byte-identical, so the cross-product parity is green for the right reason and the strip is not too aggressive to mask a future regression. The contract de-frame matches the entity's exact before/after with the Concurrency-safe paragraph unchanged. Gates green from inside the worktree with real captured exit codes; the only `./...` failure is the pre-existing env-only codex test, not a regression. Worktree clean at 00d3e5b after all mutation/diag testing.

## Feedback Cycles

### Cycle 1 — REJECTED (FO override of validator PASSED; parallel adversarial staff audit + FO reproduction both confirmed a Material bug)

Validation recommended PASSED but its diag test used a fixture where `workflowDir` IS the state checkout (conflated), so it never exercised the real `docs/dev` vs `docs/dev/.spacedock-state` split. The parallel adversarial staff audit (detached checkout) and an independent FO reproduction against the REAL `docs/dev` workflow both confirmed a Material correctness bug. Routing to implementation.

**M1 (Material) — the emitted `git -C` binds `workflowDir`, not the state checkout; the state commit FAILS on both branches.**
`stateCommitGuidance(workflowDir, entityPath)` (build.go:318 worktree branch, build.go:328 non-worktree branch) binds `stateCheckout = workflowDir`. Under split-root, `workflowDir` is the README/definition root (`docs/dev`); the real state git checkout is `filepath.Join(workflowDir, <state: value>)` = `docs/dev/.spacedock-state`. `workflowIsSplitRoot` (helpers.go:124-128) reads `state:` only as a boolean and discards the value, so build.go never learns the real state root. Reproduced against the real `docs/dev` workflow with the fixed binary — it emits: `git -C <abs>/docs/dev add <abs>/docs/dev/.spacedock-state/.../index.md`. `git -C docs/dev` targets the MAIN repo, where `.spacedock-state` is gitignored, so the command fails (`exit 1`, "paths are ignored") — on BOTH branches. The change converted a literal-placeholder line into a confidently-wrong absolute command, and ensign-shared-core.md tells the worker to TRUST the emitted path. **Fix:** resolve the real state checkout (e.g. `resolveRoots().entityDir` per status/roots.go:55-67, or `filepath.Join(workflowDir, stateValue)`) and bind THAT at both call sites; correct the false comment at build.go:304-305.

**M2 (Material) — the new test conflates `workflowDir` with the state checkout, so it cannot catch M1.**
build_statecommit_test.go:35-46 declares `state: state-checkout` in its fixture README but places the entity directly under `workflowDir` (not under the declared state checkout) and asserts `git -C <workflowDir>`. The fixture is internally inconsistent with its own README; a faithful fixture (entity at `workflowDir/state-checkout/thing/index.md`, assert `git -C workflowDir/state-checkout`) FAILS against current code. **Fix:** make the fixture place the entity under the declared `state:` checkout and assert the resolved checkout path, so the test actually catches M1; correct the comment at :33-34.

**Cleared (do NOT touch):** the parity-harness `stripStateCommitGuidance` (sound, line-anchored, not over-aggressive) and the ensign-shared-core.md de-frame (correct; its literal `{state_checkout}` placeholders are reference prose, not an emitted command).

## Stage Report: implementation (cycle 1)

- DONE: M1 fix — bind the RESOLVED state checkout (NOT raw workflowDir) at BOTH call sites
  `workflowIsSplitRoot` (bool, discarded the value) → `splitRootStateCheckout(workflowDir)` returning `filepath.Join(workflowDir, <state>)`; build.go derives `stateCheckout`/`splitRoot` from it and binds `stateCheckout` at the worktree branch + the non-worktree branch (was `workflowDir`). Corrected the false comment (build.go:304-305) + entity-body premise (Problem statement §23, Proposed approach step 1 §39). Proven against the REAL docs/dev workflow: emitted `git -C .../docs/dev/.spacedock-state add ...` (the state repo) — `add --dry-run` exit 0; the cycle-1 binding emitted `git -C .../docs/dev ...` which failed exit 1 ("paths are ignored"). Commit 41e76a7.
- DONE: M2 fix — make build_statecommit_test.go's fixture FAITHFUL
  Fixture now places the entity under the declared `state:` checkout (`workflowDir/state-checkout/thing/index.md`) and asserts `git -C <workflowDir/state-checkout>`, matching the README's `state: state-checkout`. Confirmed real red→green: FAILED on both sub-cases against the current workflowDir-binding (missing-resolved-path), PASSES after M1. Corrected the conflation comment at :33-34. Commit 41e76a7.
- DONE: Leave CLEARED items UNTOUCHED; full gates green with REAL captured exit codes from inside the worktree
  Did not touch `stripStateCommitGuidance` (parity stays green: cross-product 8 cases pass) or the ensign-shared-core.md de-frame. Gates from inside the worktree at 41e76a7: `gofmt -l .` GOFMT_EXIT:0 (empty), `go vet ./...` VET_EXIT:0, `go test ./internal/dispatch/` DISPATCH_EXIT:0 (81 passed), `go test ./...` GOTEST_ALL_EXIT:1 — 488 passed, sole failure the pre-existing env-only `TestCodexResolveManifestAgainstInstalledHost`.

### Summary

Fixed the M1 state-checkout binding: the emitted `git -C` now targets the resolved `workflowDir/<state>` (the actual state git repo, e.g. `docs/dev/.spacedock-state`) instead of `workflowDir` (the definition root, where `.spacedock-state` is gitignored). `workflowIsSplitRoot` became `splitRootStateCheckout`, returning the resolved absolute checkout instead of discarding the `state:` value; build.go binds it at both the worktree and non-worktree call sites. Proven end-to-end against the REAL docs/dev workflow — the fixed binary's `git add --dry-run` exits 0, the cycle-1 binding exited 1 on a gitignored path. The M2 test fixture is now faithful (entity under the declared `state:` checkout, asserts the resolved checkout), confirmed red→green so it genuinely catches M1. CLEARED items (parity strip, de-frame) untouched; gates green with real captured exit codes, only the pre-existing env-only codex test fails. Code on branch spacedock-ensign/dispatch-nonworktree-state-guidance at 41e76a7.

## Stage Report: validation (cycle 2)

- DONE: Reproduce the M1 fix against the REAL split-root topology (workflowDir=docs/dev ≠ state checkout=docs/dev/.spacedock-state), NOT a conflated fixture. Build binary; run `dispatch build` for a worktree-stage AND a non-worktree-stage; confirm emitted `git -C` targets the state repo; `add --dry-run` exits 0 (old workflowDir binding exits 1 on the gitignored path). Reproduce M2 red→green.
  Built /tmp/sd-cycle2/spacedock from the worktree (exit 0). Confirmed real topology: `docs/dev/.spacedock-state` is its OWN git toplevel AND is gitignored in the main repo (`git check-ignore` exit 0). Ran real `dispatch build --workflow-dir <abs>/docs/dev` for `implementation` (worktree, entity dispatch-nonworktree-state-guidance) and `ideation` (non-worktree, entity cli-ergonomics): BOTH emit `git -C <abs>/docs/dev/.spacedock-state add <abs-entity> && ... commit ...` (the state repo), 0 brace tokens each. M1 proof: `git -C <abs>/docs/dev/.spacedock-state add --dry-run <entity>` exit 0; OLD `git -C <abs>/docs/dev add --dry-run <entity>` exit 1 ("paths are ignored by one of your .gitignore files: docs/dev/.spacedock-state"). M2 red→green: reverted build.go binding (`stateCheckout := workflowDir`) → build_statecommit_test.go RED on BOTH sub-cases ("body missing resolved add command" → `.../state-checkout`); restored verbatim (git diff empty) → fresh `go test -count=1` GREEN.
- DONE: Collateral-regression check on the `workflowIsSplitRoot`→`splitRootStateCheckout` refactor; confirm `splitRootStateCheckout` returns "" for non-split-root (no state-commit block) and resolves for split; CLEARED items intact.
  `grep workflowIsSplitRoot internal/` → ZERO matches (clean rename); the old name had exactly ONE caller (build.go:222) + its def, both updated. Non-split-root behavioral proof: a single-root README (no `state:`) non-worktree build emits 0 `This workflow is split-root` lines and 0 `git -C` lines — the block is suppressed. Split case resolves to `filepath.Join(workflowDir, state)` (helpers.go:127-135). CLEARED items intact: 41e76a7 (cycle-1 fix) touched NEITHER parity_harness_test.go NOR ensign-shared-core.md (`stripStateCommitGuidance` present at parity_harness_test.go:100; de-frame heading `### Split-Root State Contract` + generalized opening present verbatim per §72-79, Concurrency-safe paragraph unchanged).
- DONE: AC-1/AC-2 + gates: AC-1 (both split-root stages carry resolved brace-free guidance), AC-2 (non-split-root emits nothing; parity green); full gates green with REAL captured exit codes from inside the worktree.
  AC-1: `build_statecommit_test.go` GREEN (2 sub-cases, resolved abs paths + brace-absent) + real-topology reproduction above. AC-2: `TestBuildParityCrossProduct` GREEN (8 cases, split+single × worktree/non-worktree) + single-root build emits nothing (above). Gates from inside the worktree at 41e76a7 (clean tree): `gofmt -l .` GOFMT_EXIT:0 (empty), `go vet ./...` VET_EXIT:0, `go test ./...` GOTEST_ALL_EXIT:1 — sole failure `TestCodexResolveManifestAgainstInstalledHost` (env-only: `Failed to read config file ~/.codex/config.toml: Operation not permitted` — sandbox perms, NOT dispatch; branch 51ffb1e..HEAD touches no cli/codex file). Worktree clean after all mutation/diag testing (git diff empty).

### Summary

PASSED. Re-validated cycle-1's M1/M2 fix against the REAL `docs/dev` split-root topology that the cycle-1 validation missed by conflating workflowDir with the state checkout. With the fixed binary, both a worktree-stage and a non-worktree-stage `dispatch build` emit `git -C <abs>/docs/dev/.spacedock-state …` (the actual state git repo, which is gitignored in the main repo), and `git add --dry-run` against that path exits 0 — whereas the OLD workflowDir binding exits 1 on the gitignored path. Reproduced M2 red→green by reverting the binding (test RED on both sub-cases, restored to GREEN; tree byte-identical to HEAD). The `workflowIsSplitRoot`→`splitRootStateCheckout` rename is clean (zero stale callers; single-root resolves to "" so no block emits, proven behaviorally), and the CLEARED items (parity strip, contract de-frame) are intact and untouched by the cycle-1 fix. Full gates green from inside the worktree with real captured exit codes; the only `./...` failure is the named pre-existing env-only codex test, not a regression.
