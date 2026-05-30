---
id: zrep3rt8a21y7k3szbs3ysyn
title: Retest without README symlink
status: done
score: "0.45"
source: bootstrap roadmap
worktree: 
started: 2026-05-30T04:30:16Z
completed: 2026-05-30T18:50:09Z
verdict: PASSED
---

# Retest Without README Symlink

Remove the compatibility symlink (`.spacedock-state/README.md -> ../README.md`) from the state checkout and prove that native split-root status is complete: the same workflow that worked through the symlink keeps working with the symlink gone, with no runtime state churn leaking into the main repo.

## Problem Statement

During the compatibility phase, this development workflow is read through a symlink. The current Python status oracle (`/Users/clkao/git/spacedock/skills/commission/bin/status`) is fundamentally **single-root**: it joins `--workflow-dir` with `README.md` (`status:587`, `status:713`) and scans entities in that same directory (`status:608`). It has no concept of the `state:` field (confirmed: zero matches for `state:`/`state_dir` in the script). Pointing it at `.spacedock-state` works **only** because `.spacedock-state/README.md` is a symlink resolving to the real README, so the stages block and the entities are reachable from one directory.

This is verifiable today:

- Oracle against the state dir (symlink present) renders all 7 entities: `python3 .../status --workflow-dir docs/dev/.spacedock-state` → full table.
- Oracle against the real workflow dir renders an **empty** table: `python3 .../status --workflow-dir docs/dev` → header rows only, because the README is there but the entities are not (they live in `.spacedock-state`, which the oracle does not compose in).
- Oracle against a state dir with the symlink removed cannot find any README there and falls back to wrong id-style detection (reproduced: `Error: non-numeric sequential id ...`).

So the symlink is load-bearing scaffolding for the *oracle*. The roadmap's split-root design (Stage 6, native-state-dir) moves this composition into the native binary: stages from the main `README.md`, entities from the `state:` directory, resolved as one workflow view (`docs/specs/state-behavior-extension.md:90-110`). Once that lands, the symlink is dead weight — but "should be removable" is not "is removable." Removing it can expose latent dependencies: a code path that still expects a README inside the state dir, a discovery routine that double-counts `.spacedock-state` once it stops looking like a symlinked twin of the parent, or mutation/archive paths that resolve against the wrong root.

**The design problem is therefore: define the acceptance test that proves the native split-root path is self-sufficient — that deleting the symlink changes nothing observable about status, mutation, and archive — and pin the exact end-state git properties of the two-repo layout (main repo carries no runtime churn; the state checkout carries the entity mutations) so a validator can rerun it deterministically.**

### Layout facts this test depends on (verified against the live checkout)

- The main repo (`/Users/clkao/git/spacedock-research/spacedock-v1`) and the state checkout (`docs/dev/.spacedock-state`) are **two independent git repositories**. The state checkout has its own `.git` and is on branch `spacedock-state/dev`.
- The main repo **already ignores the entire state directory**: its `.git/info/exclude` contains `**/.spacedock-state/`. So `git status --short` in the main repo excludes `.spacedock-state` regardless of the symlink. The "no runtime churn in main" property is structurally provided by this ignore rule; the test's job is to *prove it holds*, not to establish it.
- The symlink itself is a **tracked file in the state repo**: `git -C .spacedock-state ls-files --stage README.md` → `120000 ... README.md`. Removing it is a tracked deletion **in the state repo**, not the main repo. This deletion is the one intended permanent change of this stage; it must not appear as churn that contaminates the entity-mutation evidence.
- The main repo tracks the real workflow README at `docs/dev/README.md`, which declares `state: .spacedock-state` and the stages block. This is the definition root the native binary reads from.

## Proposed Approach

Run the retest as a **two-phase acceptance test against the native binary**, on the live development checkout (this is the dogfood workflow), not only a synthetic fixture. The native binary and its `state:` handling are produced by Stage 6 (native-state-dir); this stage consumes them and certifies them.

**Phase A — symlink-present baseline (capture the oracle for the native binary to match).**
With the symlink still in place, capture the native binary's read output for the workflow so there is a concrete before/after reference:

    spacedock status --workflow-dir docs/dev
    spacedock status --workflow-dir docs/dev --next
    spacedock status --workflow-dir docs/dev --validate

Record these as the baseline. They must already pass once Stage 6 is merged (that is Stage 6's own AC); Phase A simply freezes the reference so Phase B can prove *invariance*, not just *success*.

**Phase B — remove the symlink, prove invariance and the git-property end-state.**

1. Delete the symlink in the state checkout: `rm docs/dev/.spacedock-state/README.md` (equivalently `git -C docs/dev/.spacedock-state rm README.md`). Nothing else changes.
2. Rerun the full Go suite: `go test ./...` and `go test ./... -race`. The suite includes the Stage 6 split-root fixture tests (a fixture with **no** `.spacedock-state/README.md`), which is what makes the suite a meaningful gate for symlink-free operation.
3. Rerun the read commands from Phase A against the **native binary** and assert their output is **byte-for-byte identical** to the Phase A baseline (after the normalizations the sibling vendor-status-compatibility stage already defined for timestamps/abs-paths). Identical output with the symlink gone is the core proof that the native binary composes the two roots itself.
4. Run the mutation cycle through the native binary against the live state checkout and assert it touches **only** the state repo:
   - `spacedock status --workflow-dir docs/dev --set <slug> status=<next-stage>` then revert, or run it on a scratch entity, so the dev workflow is left in its prior state.
   - `spacedock status --workflow-dir docs/dev --archive <slug>` on a disposable entity, then restore.
5. Capture the git end-state in both repos and assert the two properties below.

**Why on the live checkout and not only a fixture.** This is the dogfood migration gate (the entity's own frontmatter calls it "the migration gate from compatibility profile to native profile"). The two-repo, nested-checkout, `info/exclude`-driven layout is a property of *this* checkout; a `t.TempDir()` fixture with `git init` reproduces the directory shape but not the exact ignore wiring unless the test sets it up explicitly. The Go suite (step 2) covers the fixture-level split-root behavior portably; the live rerun (steps 3-5) certifies that *this* workflow, the one the team actually runs, survives symlink removal. Both are required: fixtures for portable regression coverage, the live rerun for the migration certification.

**Scope guards.** This stage **adds no product code** — it is a retest/certification stage. It does not implement split-root (Stage 6) or native parsing (Stage 5); it consumes them. It does not introduce mods, PR, or lifecycle-hook behavior (out of roadmap scope). The only repository change it makes permanent is the deletion of the tracked symlink in the **state** repo; all entity mutations exercised during the test are reverted/restored so the dogfood workflow is left in a clean, working state.

**Dependency.** Hard dependency on **native-state-dir (Stage 6)**: this entity is the acceptance test *for* native split-root. If Stage 6 is not merged, the native binary cannot read the workflow with the symlink gone (the oracle structurally cannot, per Problem Statement), and every command below fails. This entity must not enter implementation until Stage 6 is PASSED. (Assumption stated for FO reconciliation at the gate: Stage 6 lands a native `spacedock status` that resolves `state:` relative to the README dir and composes stages+entities; if Stage 6's final command surface differs from `spacedock status --workflow-dir docs/dev`, the exact invocations below must be reconciled to match Stage 6's accepted interface.)

## Acceptance criteria

Each AC names a property of the finished retest, with the exact command a validator reruns to reproduce its evidence.

**AC-1 - With the symlink deleted, the full Go suite passes, including the split-root fixture that has no `.spacedock-state/README.md`.**
The symlink is scaffolding for the oracle, not for the native binary; the suite must be green with it gone. The Stage 6 fixture (no README in the state dir) is the portable regression guard.
Verified by: after `rm docs/dev/.spacedock-state/README.md`, run `go test ./...` and `go test ./... -race` from the main repo root; both exit 0. Re-run is deterministic — no symlink, network, or team-state dependency.

**AC-2 - Native status read output is invariant to symlink removal: identical bytes with the symlink present and absent.**
Removing the symlink changes nothing a reader of status sees, proving the native binary composes the definition root (`docs/dev/README.md`) and the state root (`docs/dev/.spacedock-state`) on its own rather than relying on the symlink.
Verified by: capture `spacedock status --workflow-dir docs/dev`, `... --next`, and `... --validate` with the symlink present (Phase A baseline); delete the symlink; rerun the same three commands; assert each output equals its baseline byte-for-byte after the timestamp/abs-path normalizations defined in vendor-status-compatibility's test plan. A diff of zero is the pass condition.

**AC-3 - The main repo shows no runtime state churn after a full status/mutation/archive cycle run with the symlink gone.**
`git status --short` in the main repo excludes `.spacedock-state` entirely (provided by `**/.spacedock-state/` in `.git/info/exclude`), so no entity creation, `--set`, or `--archive` performed through the native binary appears as a main-repo change. The deletion of the tracked symlink is a *state-repo* change and likewise does not surface in the main repo.
Verified by: after running the Phase B mutation cycle (`--set`, `--archive` on disposable entities), run `git -C /Users/clkao/git/spacedock-research/spacedock-v1 status --short`; assert no line begins with or contains `docs/dev/.spacedock-state`. (The only expected main-repo lines, if any, are unrelated product/docs edits the retest deliberately makes, e.g. updating the README's symlink-phase wording — see AC-5.)

**AC-4 - The state checkout shows exactly the workflow-state changes the cycle produced, and nothing else.**
Mutations performed through the native binary land in the state repo and only there; `--set` rewrites entity frontmatter under `.spacedock-state`, `--archive` moves entities under `.spacedock-state/_archive`, and the symlink deletion shows as a removed tracked file — with no spurious churn (no stray README regeneration in the state dir, no entity touched that the cycle did not target).
Verified by: run `git -C /Users/clkao/git/spacedock-research/spacedock-v1/docs/dev/.spacedock-state status --short`; assert the changed-paths set equals exactly { the symlink deletion `D README.md`, the entity file(s) the `--set` targeted, the archive move the `--archive` produced }. Each listed path is accounted for by a command the validator ran; any unlisted path is a failure.

**AC-5 - Reports and artifacts stay attached to folder-form entities through an archive with the symlink gone.**
Archiving a folder-form entity (e.g. `<slug>/index.md` with `reports/` and `artifacts/`) under split-root moves the whole folder to `.spacedock-state/_archive/<slug>/` with its `reports/` and `artifacts/` intact — proving folder-form composition does not depend on the symlink and that the native archive path resolves against the state root.
Verified by: create a disposable folder-form entity with a `reports/` file under `.spacedock-state`, `spacedock status --workflow-dir docs/dev --archive <slug>`, then assert `docs/dev/.spacedock-state/_archive/<slug>/index.md` and `.../_archive/<slug>/reports/<file>` exist and the active path is gone; restore afterward. Confirms folder-form detection (entities with `reports/`/`artifacts/` are not misdetected as separate entities) survives symlink removal.

## Test Plan

This is a **certification stage**: the proof is rerunning real commands and asserting git/file end-states, not writing new product code. Two layers, both required.

**Layer 1 — portable Go regression (covers AC-1, and AC-2/AC-4/AC-5 at fixture level).** The Stage 6 (native-state-dir) suite already builds a split-root fixture **with no `.spacedock-state/README.md`** and asserts: status reads stages from the main README and entities from the state dir; `--set` mutates only the state dir; `--archive` moves only state-dir files; discovery finds the main README and ignores `.spacedock-state` as a second workflow; folder-form `reports/`/`artifacts/` survive archive. This stage's Layer-1 contribution is to confirm that suite is green **after the symlink is physically removed from the live checkout** — i.e. the suite never depended on the symlink existing on disk. Cost: low (the suite already exists by this stage); commands `go test ./...`, `go test ./... -race`. If any Layer-1 test still reads a `.spacedock-state/README.md`, that is a Stage 6 gap this stage surfaces back to the FO rather than papers over.

**Layer 2 — live dogfood migration certification (covers AC-2, AC-3, AC-4, AC-5 on the real checkout).** Run against `/Users/clkao/git/spacedock-research/spacedock-v1/docs/dev`:

1. *Baseline capture (Phase A).* Save `spacedock status --workflow-dir docs/dev`, `--next`, `--validate` outputs while the symlink is present.
2. *Remove symlink.* `git -C docs/dev/.spacedock-state rm README.md` (tracked deletion; keeps the state repo's index honest).
3. *Invariance (AC-2).* Rerun the three read commands; `diff` against baseline after normalization → empty.
4. *Mutation cycle (AC-3, AC-4).* On a disposable scratch entity: `--set <scratch> status=ideation`, then `--archive <scratch>`. Capture both repos' `git status --short`.
5. *Folder-form archive (AC-5).* On a disposable folder-form scratch entity with a `reports/` file: `--archive`, assert the archived folder retains `reports/`.
6. *Assert git properties (AC-3, AC-4).* Main repo `git status --short` contains no `.spacedock-state` path; state repo `git status --short` equals the exact expected change set.
7. *Restore.* Revert the scratch mutations/archives so the dogfood workflow is left clean; the **only** intended residual change is the committed symlink deletion (plus any AC-5 README wording edit in the main repo).

**Normalization (reused, not reinvented).** AC-2's byte comparison uses the exact timestamp and absolute-path normalizations the sibling `vendor-status-compatibility` stage defined (ISO-8601 UTC → `<TS>`; temp/abs-root → placeholder; macOS `/var`→`/private/var` realpath accounting). No new normalization is introduced here.

**Cost / risk.** Layer 1 is cheap and deterministic. Layer 2 is a live run on the dogfood checkout — its risk is leaving the workflow dirty; the restore step (7) and using disposable scratch entities mitigate that. No network, `gh`, or team-state dependency is exercised (the retest avoids `--boot`'s volatile sections). The single irreversible action is the symlink deletion, which is the intended deliverable of the stage.

**Exact commands a validator reruns** (the deterministic core, assuming Stage 6's accepted command surface):

    # AC-1
    rm -f docs/dev/.spacedock-state/README.md   # or: git -C docs/dev/.spacedock-state rm README.md
    go test ./...
    go test ./... -race

    # AC-2 (capture baseline BEFORE the rm above, then diff after)
    spacedock status --workflow-dir docs/dev
    spacedock status --workflow-dir docs/dev --next
    spacedock status --workflow-dir docs/dev --validate

    # AC-3 / AC-4 (after a --set/--archive cycle on a scratch entity)
    git -C /Users/clkao/git/spacedock-research/spacedock-v1 status --short            # excludes .spacedock-state
    git -C /Users/clkao/git/spacedock-research/spacedock-v1/docs/dev/.spacedock-state status --short  # exact change set

## Notes

This is the migration gate from compatibility profile to native profile. It is the acceptance test **for** native split-root (Stage 6, native-state-dir) and must not enter implementation until Stage 6 is PASSED — the current Python oracle structurally cannot read the workflow with the symlink gone (single-root: README and entities must share a directory), so only the native binary can satisfy these ACs. The compatibility oracle for any baseline comparison remains `/Users/clkao/git/spacedock/skills/commission/bin/status`, used only while the symlink is still present (Phase A). The two-repo layout fact that the main repo's `.git/info/exclude` already carries `**/.spacedock-state/` is what makes the "no main-repo churn" property structural rather than aspirational; the retest proves it holds end-to-end.

## Stage Report: ideation

- DONE: Design the retest plan: remove .spacedock-state/README.md (the symlink), then prove split-root still works WITHOUT it — full Go suite + a pilot workflow run against split-root mode.
  Two-layer plan in Proposed Approach / Test Plan: Layer 1 = `go test ./...` + `-race` (incl. Stage 6 split-root fixture with no state README); Layer 2 = live dogfood rerun on `docs/dev` — Phase A baseline capture, `rm` symlink, prove read invariance, run `--set`/`--archive` cycle, restore.
- DONE: AC (**AC-N** + Verified by) framed as the post-symlink end-state properties: main repo git status shows NO runtime state churn; .spacedock-state git status shows ONLY workflow state changes.
  AC-1..AC-5 rewritten to end-state properties: suite-green-without-symlink (AC-1), read invariance (AC-2), `git -C <main> status --short` excludes `.spacedock-state` (AC-3), `git -C <state> status --short` equals exact change set (AC-4), folder-form reports/artifacts survive archive (AC-5). Each carries a reproducible `Verified by:` command.
- DONE: Note the hard dependency on native-state-dir (Stage 6); this entity is the acceptance test for native split-root. Name the exact commands a validator reruns.
  Hard dependency on Stage 6 stated in Proposed Approach (Dependency) and Notes, with an explicit FO-reconciliation assumption that command surfaces match Stage 6's accepted interface. Exact validator commands listed verbatim in Test Plan's "Exact commands a validator reruns" block.

### Summary

Designed Stage 7 as a certification stage (no product code) that consumes Stage 6's native split-root binary and proves the compatibility symlink is removable with no observable effect. Verified empirically against the live checkout that the Python oracle is single-root (`os.path.join(workflow_dir, 'README.md')`, no `state:` awareness), that the main repo and `.spacedock-state` are two independent git repos, and that the main repo's `.git/info/exclude` already ignores `**/.spacedock-state/` — making "no main-repo churn" a structural property the test certifies rather than establishes. The riskiest dependency (Stage 6 must land first, since the oracle cannot read the workflow symlink-free) is called out for FO reconciliation at the gate; the symlink lives as a tracked file in the state repo, so its removal is the one intended permanent change and is kept distinct from entity-mutation evidence.

## Stage Report: implementation

- DONE: THE FLIP — cli.go Run() default -> `&status.NativeRunner{}` so production `spacedock status` runs native Go and split-root works through the real binary. Added the carried cli.Run-default unit test exercising the PUBLIC cli.Run with NO runner injection.
  Commit 4e1a343. `TestRunDefaultsToNativeRunner` drives `cli.Run` against a split-root fixture (README with `state:`, entities in the state subdir, no state README) and asserts the state-dir entity renders — behavior only NativeRunner produces (the single-root vendor oracle renders an empty table). Test confirmed RED on the old VendorRunner default before the flip, GREEN after; locks the flip at the public entrypoint, not just the injectable `run()` core.
- DONE: RETEST no-symlink split-root — built the now-native-default binary and drove a TEMP split-root fixture (README in def dir, entities under `.spacedock-state`, NO state README symlink) proving status/--next/--validate/--set/--archive/--discover all work WITHOUT any symlink. Full Go suite green.
  Native @ def-dir reads `docs/dev/README.md` directly (resolve.go:55 `filepath.Join(abs, "README.md")`), so the symlink is unused for native. Live-binary drive on the no-symlink fixture: status renders both entities ordered by the def-README stages block; --next/--validate pass; --set rewrites the state entity (README intact); --archive moves flat + folder-form (reports/ carried) under `state/_archive`, none under the def dir; --discover single-counts (state subdir pruned). REAL exit codes: `go test ./...` 0, `go test ./... -race` 0, `gofmt -l` clean, `go vet` 0 (run via `rtk proxy`, not piped through tail).
- DONE: TWO-REPO cleanliness + migration runbook — temp nested-git layout (main repo + gitignored state-checkout sub-repo); --set/--archive produce ZERO churn in the main repo and only the entity change in the state checkout. Migration runbook below.
  Temp fixture mirrors live wiring (main repo `**/.spacedock-state/` in `.git/info/exclude`; state checkout a nested independent repo). After `--set add-login` + `--archive big-feature`: `git -C main status --short` EMPTY (AC-3; unrelated tracked `notes.md` and README untouched); `git -C state status --short` = exactly `M add-login.md`, `D big-feature/index.md`, `D big-feature/reports/ideation.md`, `?? _archive/` (AC-4) — no stray state README, no def-dir `_archive`.

### FO Migration Runbook (post-merge, live `docs/dev` workflow)

The FO executes these AFTER this branch merges, to migrate the live workflow off the symlink. Do NOT run mid-flight; quiesce in-flight FO ops first. The ensign did NOT touch the live `docs/dev/.spacedock-state` or its symlink.

1. Rebuild the now-native binary from the merged main: `go build -o <bin> ./cmd/spacedock` (the merged `cli.Run` default is NativeRunner).
2. Switch FO status ops from the state-dir spelling to the def-dir spelling: replace `--workflow-dir docs/dev/.spacedock-state` with `--workflow-dir docs/dev` everywhere FO drives status (boot/list/--set/--archive/--discover). Native reads `docs/dev/README.md` and composes `docs/dev/.spacedock-state` for entities.
3. Remove the now-unused live symlink as a tracked deletion in the STATE repo: `git -C docs/dev/.spacedock-state rm README.md && git -C docs/dev/.spacedock-state commit -m "remove compatibility README symlink" -- README.md`. This is a state-repo change; it does not surface in the main repo (ignored via `info/exclude`).
4. Verify two-repo cleanliness on the live checkout: run a scratch `--set`/`--archive` cycle, then `git -C <main> status --short` (must show no `.spacedock-state` path) and `git -C docs/dev/.spacedock-state status --short` (must equal exactly the scratch change set); revert the scratch mutations.
5. Re-confirm reads: `<bin> status --workflow-dir docs/dev`, `--next`, `--validate` render the live entities with the symlink gone.

### Summary

Executed the captain-approved deferred flip and certified split-root is symlink-free. Flipped `cli.Run`'s production default to NativeRunner (4e1a343) and pinned it with a public-entrypoint test using native-only split-root rendering. Drove the real native binary against TEMP no-symlink fixtures proving every status/mutation/discover path works without the symlink, and against a temp nested-git two-repo layout proving zero main-repo churn and an exact state-repo change set. Found and fixed one carried test gap the flip exposed: the launcher smoke test (e13cc19) encoded the retired symlink-compat operator model (point at the state dir, rely on the symlink); migrated it to the native model (point at the def dir, no symlink) — the same migration the FO runbook performs live. Full Go gate green with real exit codes (test/-race/gofmt/vet). Live `docs/dev/.spacedock-state` and its symlink were left untouched per scope; the FO runbook above carries out the live migration post-merge.

## Stage Report: validation

- DONE: Reproduce THE FLIP + lock — cli.Run default is NativeRunner; RED-GREEN check TestRunDefaultsToNativeRunner; build prod binary and confirm native split-root @ docs/dev renders live entities; confirm state-dir silent-empty is ONLY the still-present symlink and the runbook step 3 transient.
  cli.go:21 injects `&status.NativeRunner{}`. RED-GREEN reproduced independently: reverted line 21 to `VendorRunner{}` -> `TestRunDefaultsToNativeRunner` FAILED (empty table, "VendorRunner cannot compose split-root"); restored -> PASS. Built prod binary from the worktree; native `status --workflow-dir docs/dev` renders all live actives (7w native-dispatch-helper, p3 external-tracker-checkpoint, zr remove-symlink-retest). Native `--workflow-dir docs/dev/.spacedock-state` is silent-empty ONLY because the live symlink resolves to `../README.md` (which re-declares `state:`), so native composes the nonexistent `.spacedock-state/.spacedock-state` -> empty; pointing native at a no-README state-dir gives the honest `non-numeric sequential id` error (matches the oracle byte-for-byte), the transient runbook step 3 resolves. Embedded `vendor/status` is byte-identical to the commission oracle and has zero `state:` field awareness (only `optional_field in state`), confirming the oracle is genuinely single-root.
- DONE: Reproduce no-symlink split-root on a FRESH temp fixture — status/--next/--validate/--set/--archive/--discover all work WITHOUT a symlink; two-repo cleanliness (zero main churn, exact state change set); full suite green (test + -race + gofmt -l + go vet, real exit codes).
  Fresh fixture (README+state: in def dir, entities under .spacedock-state, no state README; Lstat guard confirmed absent): status renders both entities ordered by the def stages block; --next/--validate pass; --set rewrote the state entity (README untouched); --archive moved flat + folder-form (reports/ carried) under state/_archive, none under def, sources gone; --discover single-counts the def dir (state subdir pruned). Two-repo nested-git fixture mirroring live (main repo `**/.spacedock-state/` in info/exclude; state a nested independent repo; tracked symlink blob 32d46ee identical to live): after --set add-login + --archive big-feature, `git -C main status --short` EMPTY (AC-3; unrelated notes.md + README untouched), `git -C state status --short` = exactly `M add-login.md`, `D big-feature/index.md`, `D big-feature/reports/ideation.md`, `?? _archive/` (AC-4); a subsequent `git -C state rm README.md` (runbook step 3) shows `D README.md` in state with main STILL empty. AC-2 invariance on a live-shaped copy (entities + symlink preserved): native @ def status/--next/--validate byte-for-byte IDENTICAL with the symlink present vs removed (no normalization needed). Gate green, real exit codes: `go test ./...` 0 (246 tests), `-race` 0, `gofmt -l .` empty, `go vet ./...` 0.
- DONE: Review the FO MIGRATION RUNBOOK for correctness + safety (5 steps, esp. step 2 operator switch to --workflow-dir docs/dev and step 3 `git -C docs/dev/.spacedock-state rm README.md`); confirm executing it migrates the live workflow off the symlink without main-repo churn; PASSED/REJECTED with reproduced evidence.
  Runbook is correct and safe. Step 1 (rebuild) — merged cli.Run default is NativeRunner (verified). Step 2 (operator switch) is real and necessary: the live `docs/dev/README.md` (lines 128/173/174) currently documents ops with `--workflow-dir docs/dev/.spacedock-state` (symlink-compat); native @ def-dir reads `docs/dev/README.md` directly (roots.go:55 `Join(abs, "README.md")`), never the state symlink. Step 3 deletion is a STATE-repo-only change: the symlink is tracked there (live mode 120000); reproduced `git rm` -> `D README.md` in state, zero main churn. Live structural preconditions all hold: `**/.spacedock-state/` is in the live main `.git/info/exclude` (main shows 0 state churn), state checkout is an independent repo on `spacedock-state/dev`. Ordering (switch before remove) avoids a transient broken window, and even a wrong order fails LOUD (honest no-README error), never silent corruption; the quiesce-in-flight guard is correct. Minor non-blocking gap: step 2 covers operational invocations but does not explicitly call out updating the README's documented symlink-phase invocation lines (128/173/174) — that is the one expected main-repo doc edit noted in AC-5, harmless if it lags.

### Summary

PASSED. Independently reproduced every checklist item against the real native binary, never the live symlink: the flip is load-bearing (RED-GREEN), native split-root @ docs/dev renders the live entities, and the state-dir silent-empty is purely the still-present symlink re-declaring `state:` (the runbook step 3 transient, which resolves to an honest no-README error matching the oracle). On fresh and live-shaped no-symlink fixtures, status/--next/--validate/--set/--archive/--discover all work, AC-2 read output is byte-invariant to symlink removal, and a two-repo nested-git layout proves zero main-repo churn with an exact state-repo change set (AC-3/AC-4) plus folder-form reports/ surviving archive (AC-5). Full Go gate green with real exit codes (test 246, -race, gofmt clean, vet). The FO Migration Runbook is correct and safe — executing it would migrate the live workflow off the symlink as a state-repo-only deletion with no main-repo churn; the only nit is that step 2 should also update the README's three documented state-dir invocation lines (the AC-5 doc edit), which is harmless if deferred. Live `docs/dev/.spacedock-state`, its symlink, and the entity frontmatter were left untouched; cli.go was reverted/restored cleanly (worktree clean).
